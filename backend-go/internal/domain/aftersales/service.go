package aftersales

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/supplyevent"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// validTransitions defines allowed status transitions.
// Map key = current status, values = set of allowed next statuses.
var validTransitions = map[string]map[string]bool{
	"pending":  {"approved": true, "rejected": true},
	"approved": {"received": true, "rejected": true},
	"received": {"refunded": true},
}

// checkTransition returns an error if moving from `current` to `target` is not allowed.
func checkTransition(current, target string) error {
	allowed, ok := validTransitions[current]
	if !ok {
		return fmt.Errorf("cannot transition aftersales order from terminal status %s", current)
	}
	if !allowed[target] {
		return fmt.Errorf("cannot transition aftersales order from %s to %s", current, target)
	}
	return nil
}

// Service provides aftersales business logic.
type Service struct {
	db          *gorm.DB
	logger      *zap.Logger
	orderWriter OrderWriter
	events      EventPublisher
}

// NewService creates a new aftersales service.
func NewService(db *gorm.DB, logger *zap.Logger, orderWriter OrderWriter, events EventPublisher) *Service {
	return &Service{
		db:          db,
		logger:      logger,
		orderWriter: orderWriter,
		events:      events,
	}
}

// List returns paginated aftersales orders with optional filter.
func (s *Service) List(p *common.Pagination, f *ListFilter) ([]AfterSalesOrder, int64, error) {
	q := s.db.Model(&AfterSalesOrder{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where(
				"LOWER(reason) LIKE LOWER(?) OR order_id IN (SELECT id FROM sales_order WHERE LOWER(order_no) LIKE LOWER(?))",
				like, like,
			)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.OrderID != nil {
			q = q.Where("order_id = ?", *f.OrderID)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []AfterSalesOrder
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns a single aftersales order.
func (s *Service) Get(id int64) (*AfterSalesOrder, error) {
	var o AfterSalesOrder
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Create inserts a new aftersales order.
//
// On success it publishes a "supplychain.aftersale.returned" event so the
// supply-chain orchestrator can spin up a reverse-logistics flow (return →
// inspect → refund/resend) and create the corresponding approval-gated
// UnifiedAction. The event is published best-effort: a publish failure is
// logged but does not roll back the create, because the aftersales order
// itself is the system of record — downstream reconciliation can recover.
func (s *Service) Create(in *CreateInput) (*AfterSalesOrder, error) {
	status := in.Status
	if status == "" {
		status = "pending"
	}
	if status != "pending" {
		return nil, fmt.Errorf("aftersales order must be created in pending status")
	}
	o := AfterSalesOrder{
		OrderID:          in.OrderID,
		ItemID:           in.ItemID,
		SkuID:            in.SkuID,
		Reason:           in.Reason,
		Status:           status,
		InspectionResult: "",
		CreatedBy:        in.CreatedBy,
	}
	if in.ReturnQuantity != nil {
		o.ReturnQuantity = *in.ReturnQuantity
	}
	if in.RefundAmount != nil {
		o.RefundAmount = *in.RefundAmount
	}
	if err := s.db.Create(&o).Error; err != nil {
		return nil, err
	}
	s.publishAftersaleReturned(&o)
	return &o, nil
}

// publishAftersaleReturned publishes a supplyevent.AftersaleReturned event
// so the supply-chain orchestrator can create a reverse-logistics flow.
func (s *Service) publishAftersaleReturned(o *AfterSalesOrder) {
	if s.events == nil {
		return
	}
	// Only publish when there's a SKU to track — without a SKU the
	// orchestrator cannot correlate the return to inventory or sourcing.
	if o.SkuID == nil {
		return
	}
	evt := supplyevent.AftersaleReturned{
		AftersaleID: o.ID,
		OrderID:     o.OrderID,
		SkuID:       *o.SkuID,
		Quantity:    o.ReturnQuantity,
		Reason:      o.Reason,
		ReturnedAt:  time.Now(),
	}
	payload, err := supplyevent.ToPayload(evt)
	if err != nil {
		s.logger.Warn("failed to serialize AftersaleReturned event",
			zap.Int64("aftersale_id", o.ID), zap.Error(err))
		return
	}
	if _, err := s.events.Publish(context.Background(),
		"supplychain.aftersale.returned", "aftersales", payload); err != nil {
		s.logger.Warn("failed to publish AftersaleReturned event",
			zap.Int64("aftersale_id", o.ID), zap.Error(err))
	}
}

// Update applies partial updates to an aftersales order.
func (s *Service) Update(id int64, in *UpdateInput) (*AfterSalesOrder, error) {
	var o AfterSalesOrder
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.ReturnQuantity != nil {
		updates["return_quantity"] = *in.ReturnQuantity
	}
	if in.Reason != nil {
		updates["reason"] = *in.Reason
	}
	if in.Status != nil {
		if err := checkTransition(o.Status, *in.Status); err != nil {
			return nil, err
		}
		updates["status"] = *in.Status
	}
	if in.RefundAmount != nil {
		updates["refund_amount"] = *in.RefundAmount
	}
	if in.InspectionResult != nil {
		updates["inspection_result"] = *in.InspectionResult
	}
	if in.RejectionReason != nil {
		updates["rejection_reason"] = *in.RejectionReason
	}
	if len(updates) == 0 {
		return &o, nil
	}
	if err := s.db.Model(&o).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Delete removes an aftersales order by id.
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&AfterSalesOrder{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Approve marks an aftersales order as approved.
func (s *Service) Approve(id int64, in *ApproveInput) (*AfterSalesOrder, error) {
	var o AfterSalesOrder
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	if err := checkTransition(o.Status, "approved"); err != nil {
		return nil, err
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":            "approved",
		"approved_by":       in.ApprovedBy,
		"approved_at":       &now,
		"inspection_result": in.InspectionResult,
	}
	if err := s.db.Model(&o).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Reject marks an aftersales order as rejected.
func (s *Service) Reject(id int64, in *RejectInput) (*AfterSalesOrder, error) {
	var o AfterSalesOrder
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	if err := checkTransition(o.Status, "rejected"); err != nil {
		return nil, err
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":           "rejected",
		"rejected_by":      in.RejectedBy,
		"rejected_at":      &now,
		"rejection_reason": in.RejectionReason,
	}
	if err := s.db.Model(&o).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Receive marks an aftersales order as received (goods returned received)
// and publishes an event so inventory auto-restocks.
func (s *Service) Receive(id int64, in *ReceiveInput) (*AfterSalesOrder, error) {
	var o AfterSalesOrder
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	if o.Status != "approved" {
		return nil, fmt.Errorf("cannot receive aftersales order in status %s", o.Status)
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":      "received",
		"received_by": in.ReceivedBy,
		"received_at": &now,
	}
	if err := s.db.Model(&AfterSalesOrder{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	if o.SkuID != nil && o.ReturnQuantity > 0 {
		s.publishAfterSaleProcessed(&o)
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Refund marks an aftersales order as refunded and records the refund amount.
// It also transitions the original order to "cancelled" status.
func (s *Service) Refund(id int64, in *RefundInput) (*AfterSalesOrder, error) {
	var o AfterSalesOrder
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	if o.Status != "received" && o.Status != "approved" {
		return nil, fmt.Errorf("cannot refund aftersales order in status %s", o.Status)
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":        "refunded",
		"refunded_by":   in.RefundedBy,
		"refunded_at":   &now,
		"refund_amount": in.RefundAmount,
	}
	if err := s.db.Model(&AfterSalesOrder{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	// Cancel the original order in its own transaction (eventual consistency)
	remark := fmt.Sprintf("aftersales refund, aftersales_id=%d", id)
	if err := s.orderWriter.CancelOrder(context.Background(), o.OrderID, in.RefundedBy, remark); err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Summary returns aggregation by status and total refund amount.
func (s *Service) Summary() (*Summary, error) {
	var total int64
	if err := s.db.Model(&AfterSalesOrder{}).Count(&total).Error; err != nil {
		return nil, err
	}
	type statusCount struct {
		Status string
		Cnt    int64
	}
	var scs []statusCount
	if err := s.db.Model(&AfterSalesOrder{}).
		Select("status, COUNT(*) AS cnt").Group("status").Scan(&scs).Error; err != nil {
		return nil, err
	}
	byStatus := make(map[string]int64, len(scs))
	for _, sc := range scs {
		byStatus[sc.Status] = sc.Cnt
	}
	var rev struct {
		Total float64
	}
	if err := s.db.Model(&AfterSalesOrder{}).
		Select("COALESCE(SUM(refund_amount),0) AS total").
		Where("status = ?", "refunded").
		Scan(&rev).Error; err != nil {
		return nil, err
	}
	return &Summary{Total: total, ByStatus: byStatus, TotalRefunded: rev.Total}, nil
}

// publishAfterSaleProcessed publishes a supplyevent.AfterSaleProcessed event.
func (s *Service) publishAfterSaleProcessed(o *AfterSalesOrder) {
	if s.events == nil || o.SkuID == nil {
		return
	}
	evt := supplyevent.AfterSaleProcessed{
		AftersaleID: o.ID,
		OrderID:     o.OrderID,
		SkuID:       *o.SkuID,
		Quantity:    o.ReturnQuantity,
		Type:        "return",
		ProcessedAt: time.Now(),
	}
	payload, err := supplyevent.ToPayload(evt)
	if err != nil {
		s.logger.Warn("failed to serialize AfterSaleProcessed event", zap.Error(err))
		return
	}
	if _, err := s.events.Publish(
		eventbus.WithIdempotencyKey(context.Background(), fmt.Sprintf("aftersale_processed:%d", o.ID)),
		"supplychain.aftersale.completed", "aftersales", payload); err != nil {
		s.logger.Warn("failed to publish AfterSaleProcessed event", zap.Error(err))
	}
}

// ---------------------------------------------------------------------------
// Dispute Resolution Engine
// ---------------------------------------------------------------------------

// Dispute thresholds for rule-based scoring.
const (
	DisputeAmountLowThreshold  = 50.0  // below this is low-risk
	DisputeAmountHighThreshold = 500.0 // above this is high-risk
	DisputeAutoApproveScore    = 75.0  // score >= this => auto-approve
	DisputeAutoRejectScore     = 25.0  // score <= this => auto-reject
)

// DisputeService provides dispute case evaluation and auto-decision.
type DisputeService struct {
	db              *gorm.DB
	logger          *zap.Logger
	deliveryChecker DeliveryChecker
}

// NewDisputeService creates a new DisputeService.
func NewDisputeService(db *gorm.DB, logger *zap.Logger, deliveryChecker DeliveryChecker) *DisputeService {
	if deliveryChecker == nil {
		deliveryChecker = &noopDeliveryChecker{}
	}
	return &DisputeService{
		db:              db,
		logger:          logger,
		deliveryChecker: deliveryChecker,
	}
}

// noopDeliveryChecker always returns false (not delivered) when no real
// delivery checker is wired in. This avoids nil checks in production code.
type noopDeliveryChecker struct{}

func (n *noopDeliveryChecker) IsDelivered(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

// CreateCase creates a new dispute case.
func (ds *DisputeService) CreateCase(ctx context.Context, in *CreateDisputeInput) (*DisputeCase, error) {
	dc := DisputeCase{
		TransactionID: in.TransactionID,
		Platform:      in.Platform,
		ClaimType:     in.ClaimType,
		Amount:        in.Amount,
		Status:        "pending",
		Evidence:      in.Evidence,
		DecisionScore: 0,
	}
	if err := ds.db.WithContext(ctx).Create(&dc).Error; err != nil {
		return nil, fmt.Errorf("create dispute case: %w", err)
	}
	return &dc, nil
}

// GetCase returns a single dispute case by ID.
func (ds *DisputeService) GetCase(ctx context.Context, id int64) (*DisputeCase, error) {
	var dc DisputeCase
	if err := ds.db.WithContext(ctx).First(&dc, id).Error; err != nil {
		return nil, err
	}
	return &dc, nil
}

// ListCases returns paginated dispute cases with optional filters.
func (ds *DisputeService) ListCases(ctx context.Context, p *common.Pagination, f *DisputeListFilter) ([]DisputeCase, int64, error) {
	q := ds.db.Model(&DisputeCase{})
	if f != nil {
		if f.Platform != "" {
			q = q.Where("platform = ?", f.Platform)
		}
		if f.ClaimType != "" {
			q = q.Where("claim_type = ?", f.ClaimType)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []DisputeCase
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// EvaluateCase runs rule-based scoring on a dispute case. It does NOT call
// any external AI API. The score is computed from:
//   - delivery status (is the shipment delivered?)
//   - dispute amount relative to thresholds
//   - claim type characteristics
//
// Returns the updated dispute case with DecisionScore, AiReason, and
// DecisionSource populated.
func (ds *DisputeService) EvaluateCase(ctx context.Context, id int64) (*EvaluateDisputeResult, error) {
	dc, err := ds.GetCase(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get dispute case: %w", err)
	}

	breakdown := make([]RuleBreakdownItem, 0)
	score := 50.0 // neutral baseline

	// 1. Delivery check for "not_received" claims
	if dc.ClaimType == "not_received" {
		delivered, checkErr := ds.deliveryChecker.IsDelivered(ctx, dc.TransactionID, dc.Platform)
		if checkErr != nil {
			ds.logger.Warn("delivery check failed", zap.Error(checkErr))
		} else if delivered {
			score -= 30
			breakdown = append(breakdown, RuleBreakdownItem{
				Rule:   "delivery_status",
				Score:  -30,
				Reason: "Buyer claims not received but tracking shows delivered",
			})
		} else {
			breakdown = append(breakdown, RuleBreakdownItem{
				Rule:   "delivery_status",
				Score:  0,
				Reason: "Buyer claims not received and tracking does not show delivered",
			})
		}
	}

	// 2. Amount check
	switch {
	case dc.Amount <= 0:
		// No amount specified — neutral
	case dc.Amount < DisputeAmountLowThreshold:
		score += 20
		breakdown = append(breakdown, RuleBreakdownItem{
			Rule:   "amount_threshold",
			Score:  20,
			Reason: fmt.Sprintf("Dispute amount $%.2f is below low threshold $%.2f", dc.Amount, DisputeAmountLowThreshold),
		})
	case dc.Amount > DisputeAmountHighThreshold:
		score -= 10
		breakdown = append(breakdown, RuleBreakdownItem{
			Rule:   "amount_threshold",
			Score:  -10,
			Reason: fmt.Sprintf("Dispute amount $%.2f exceeds high threshold $%.2f", dc.Amount, DisputeAmountHighThreshold),
		})
	default:
		breakdown = append(breakdown, RuleBreakdownItem{
			Rule:   "amount_threshold",
			Score:  0,
			Reason: fmt.Sprintf("Dispute amount $%.2f is within normal range", dc.Amount),
		})
	}

	// 3. Claim type scoring
	switch dc.ClaimType {
	case "damaged", "defective":
		score += 10
		breakdown = append(breakdown, RuleBreakdownItem{
			Rule:   "claim_type",
			Score:  10,
			Reason: fmt.Sprintf("Claim type '%s' — supports buyer (quality issue)", dc.ClaimType),
		})
	case "wrong_item":
		score += 15
		breakdown = append(breakdown, RuleBreakdownItem{
			Rule:   "claim_type",
			Score:  15,
			Reason: fmt.Sprintf("Claim type '%s' — supports buyer (seller error)", dc.ClaimType),
		})
	case "not_received":
		score += 5
		breakdown = append(breakdown, RuleBreakdownItem{
			Rule:   "claim_type",
			Score:  5,
			Reason: fmt.Sprintf("Claim type '%s' — neutral (needs delivery check)", dc.ClaimType),
		})
	case "change_of_mind":
		score -= 10
		breakdown = append(breakdown, RuleBreakdownItem{
			Rule:   "claim_type",
			Score:  -10,
			Reason: fmt.Sprintf("Claim type '%s' — does not support buyer (change of mind)", dc.ClaimType),
		})
	default:
		breakdown = append(breakdown, RuleBreakdownItem{
			Rule:   "claim_type",
			Score:  0,
			Reason: fmt.Sprintf("Claim type '%s' — no specific rule", dc.ClaimType),
		})
	}

	// Clamp score to [0, 100]
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// Build AI reason summary
	reasonParts := make([]string, 0, len(breakdown))
	for _, item := range breakdown {
		if item.Score != 0 {
			reasonParts = append(reasonParts, fmt.Sprintf("%s (%+.0f): %s", item.Rule, item.Score, item.Reason))
		}
	}
	aiReason := fmt.Sprintf("Score: %.0f/100.", score)
	if len(reasonParts) > 0 {
		aiReason += " Rules triggered: " + strings.Join(reasonParts, "; ")
	} else {
		aiReason += " No rules triggered."
	}

	// Persist evaluation result
	updates := map[string]interface{}{
		"decision_score":  score,
		"ai_reason":       aiReason,
		"decision_source": "rule",
	}
	if err := ds.db.Model(&DisputeCase{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update dispute case score: %w", err)
	}

	// Reload
	dc.DecisionScore = score
	dc.AiReason = aiReason
	dc.DecisionSource = "rule"

	decision := evaluateDecision(score)
	return &EvaluateDisputeResult{
		Dispute:       dc,
		Score:         score,
		Decision:      decision,
		RuleBreakdown: breakdown,
	}, nil
}

// AutoDecide evaluates the case and automatically decides it based on the
// rule-based score:
//   - Score >= DisputeAutoApproveScore (75): auto-approve
//   - Score <= DisputeAutoRejectScore (25): auto-reject
//   - Otherwise: mark as manual_review
func (ds *DisputeService) AutoDecide(ctx context.Context, id int64) (*EvaluateDisputeResult, error) {
	result, err := ds.EvaluateCase(ctx, id)
	if err != nil {
		return nil, err
	}

	newStatus := evaluateDecision(result.Score)
	updates := map[string]interface{}{
		"status": newStatus,
	}
	if err := ds.db.Model(&DisputeCase{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("auto decide dispute case: %w", err)
	}

	result.Dispute.Status = newStatus
	result.Decision = newStatus
	return result, nil
}

// evaluateDecision maps a score to a status decision.
func evaluateDecision(score float64) string {
	switch {
	case score >= DisputeAutoApproveScore:
		return "approved"
	case score <= DisputeAutoRejectScore:
		return "rejected"
	default:
		return "manual_review"
	}
}

// UpdateDisputeStatus manually updates a dispute case's status and reason.
func (ds *DisputeService) UpdateDisputeStatus(ctx context.Context, id int64, status, reason string) (*DisputeCase, error) {
	updates := map[string]interface{}{
		"status":    status,
		"ai_reason": reason,
	}
	if err := ds.db.Model(&DisputeCase{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update dispute case status: %w", err)
	}
	return ds.GetCase(ctx, id)
}
