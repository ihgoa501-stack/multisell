package exceptions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ListFilter holds exception list query parameters.
type ListFilter struct {
	SourceModule string
	SourceType  string
	Severity    string
	Status      string
	AssignedTo  string
}

// Service provides exceptions business logic.
type Service struct {
	db          *gorm.DB
	logger      *zap.Logger
	llm         ai.LLMProvider            // optional, for AI suggestion generation
	oplogSvc    *operationlog.Service     // optional, for audit logging
	approvalSvc *approval.Service         // optional, for high-risk approval
}

// NewService creates a new exceptions service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// WithLLM attaches an LLM provider for AI suggestion generation.
func (s *Service) WithLLM(llm ai.LLMProvider) *Service {
	s.llm = llm
	return s
}

// WithOperationLog attaches an operation log service for audit logging.
func (s *Service) WithOperationLog(svc *operationlog.Service) *Service {
	s.oplogSvc = svc
	return s
}

// WithApproval attaches an approval service for high-risk resolution approval.
func (s *Service) WithApproval(svc *approval.Service) *Service {
	s.approvalSvc = svc
	return s
}

// List returns a paginated list of exceptions filtered by severity/status/type.
func (s *Service) List(f ListFilter, page, size int) ([]ExceptionItem, int64, error) {
	q := s.db.Model(&ExceptionItem{})
	if f.SourceModule != "" {
		q = q.Where("source_module = ?", f.SourceModule)
	}
	if f.SourceType != "" {
		q = q.Where("source_type = ?", f.SourceType)
	}
	if f.Severity != "" {
		q = q.Where("severity = ?", f.Severity)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.AssignedTo != "" {
		q = q.Where("assigned_to = ?", f.AssignedTo)
	}
	var total int64
	q.Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var items []ExceptionItem
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a single exception item.
func (s *Service) GetByID(id int64) (*ExceptionItem, error) {
	var e ExceptionItem
	if err := s.db.First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// Create inserts a new exception item.
func (s *Service) Create(e *ExceptionItem) error {
	return s.db.Create(e).Error
}

// Resolve marks an exception as resolved and records the resolver + note.
func (s *Service) Resolve(id int64, resolvedBy, note string) (*ExceptionItem, error) {
	var e ExceptionItem
	if err := s.db.First(&e, id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	e.Status = "resolved"
	e.ResolvedAt = &now
	e.ResolvedBy = resolvedBy
	if note != "" {
		e.Note = note
	}
	if err := s.db.Save(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// Assign assigns an exception to a user and sets status to assigned.
func (s *Service) Assign(id int64, assignedTo string) (*ExceptionItem, error) {
	var e ExceptionItem
	if err := s.db.First(&e, id).Error; err != nil {
		return nil, err
	}
	e.AssignedTo = assignedTo
	e.Status = "assigned"
	if err := s.db.Save(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// Update performs a general update on an exception item.
func (s *Service) Update(e *ExceptionItem) error {
	return s.db.Save(e).Error
}

// Delete removes an exception item.
func (s *Service) Delete(id int64) error {
	return s.db.Delete(&ExceptionItem{}, id).Error
}

// Suggest generates an AI resolution suggestion for an exception.
// Falls back gracefully to rule-based defaults when no LLM is configured.
func (s *Service) Suggest(ctx context.Context, exceptionID int64) (*ResolutionSuggestion, error) {
	e, err := s.GetByID(exceptionID)
	if err != nil {
		return nil, err
	}

	sug := &ResolutionSuggestion{
		ExceptionID: exceptionID,
		CreatedAt:   time.Now(),
	}

	if s.llm == nil {
		return s.fallbackSuggestion(e, sug)
	}

	prompt := fmt.Sprintf(`你是一个跨境电商异常处理专家。请分析以下异常并提供处理建议。

异常类型：%s
标题：%s
描述：%s
严重程度：%s

请以JSON格式回复，包含以下字段：
- suggestion_text: 处理建议的详细说明（中文）
- suggested_action: 建议采取的行动（restock/cancel_order/adjust_price/contact_carrier/investigate_fee/other）
- risk_level: 该操作的风险等级（low/medium/high）
- auto_executable: 是否可自动执行（true/false）`, e.SourceType, e.Title, e.Description, e.Severity)

	resp, err := s.llm.Chat(ctx, &ai.LLMRequest{
		Messages:    []ai.LLMMessage{{Role: "user", Content: prompt}},
		Temperature: 0.3,
		MaxTokens:   500,
	})
	if err != nil {
		s.logger.Warn("LLM suggestion failed, falling back to rule-based", zap.Error(err))
		return s.fallbackSuggestion(e, sug)
	}

	if err := sug.parseLLMResponse(resp.Answer, e); err != nil {
		s.logger.Warn("LLM response parse failed, falling back to rule-based", zap.Error(err))
		return s.fallbackSuggestion(e, sug)
	}
	return sug, nil
}

// ResolveWithApproval resolves an exception after checking risk level.
// High-risk exceptions require an approval request; an audit log is always written.
func (s *Service) ResolveWithApproval(ctx context.Context, id int64, resolvedBy, suggestion, action string) (*ExceptionItem, error) {
	e, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	riskLevel := e.Severity
	if riskLevel == "" {
		riskLevel = "medium"
	}

	// High risk: create an approval request for audit trail
	if riskLevel == "high" && s.approvalSvc != nil {
		reason := fmt.Sprintf("Exception #%d: %s\nAgent suggestion: %s\nAction: %s", id, e.Title, suggestion, action)
		_, err := s.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			ProductID:   0,
			RequestType: "exception_resolve",
			Requester:   "system",
			TargetType:  "exception",
			TargetID:    id,
			RiskLevel:   riskLevel,
			Reason:      reason,
			EntityType:  "exception",
			EntityID:    id,
		})
		if err != nil {
			return nil, fmt.Errorf("approval setup failed: %w", err)
		}
		s.logger.Info("approval request created for high-risk exception resolution",
			zap.Int64("exception_id", id),
			zap.String("action", action))
	}

	// Audit log
	if s.oplogSvc != nil {
		_ = s.oplogSvc.Log("exceptions", "resolve", fmt.Sprintf("%d", id), resolvedBy,
			fmt.Sprintf("Owner resolved exception #%d. Suggestion: %s. Action: %s", id, suggestion, action))
	}

	// Mark as resolved
	return s.Resolve(id, resolvedBy, fmt.Sprintf("Suggestion: %s | Action: %s", suggestion, action))
}

// fallbackSuggestion generates a rule-based suggestion when no LLM is available.
func (s *Service) fallbackSuggestion(e *ExceptionItem, sug *ResolutionSuggestion) (*ResolutionSuggestion, error) {
	sug.SuggestionText = suggestTextForType(e.SourceType)
	sug.SuggestedAction = suggestActionForType(e.SourceType)
	sug.RiskLevel = e.Severity
	if sug.RiskLevel == "" {
		sug.RiskLevel = "medium"
	}
	sug.AutoExecutable = sug.RiskLevel != "high"
	return sug, nil
}

// parseLLMResponse extracts fields from the LLM's JSON response.
func (s *ResolutionSuggestion) parseLLMResponse(answer string, e *ExceptionItem) error {
	body := extractJSONBlock(answer)
	if body == "" {
		return errors.New("no JSON block found in LLM response")
	}
	var parsed struct {
		SuggestionText  string `json:"suggestion_text"`
		SuggestedAction string `json:"suggested_action"`
		RiskLevel       string `json:"risk_level"`
		AutoExecutable  bool   `json:"auto_executable"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	s.SuggestionText = parsed.SuggestionText
	s.SuggestedAction = parsed.SuggestedAction
	s.RiskLevel = parsed.RiskLevel
	s.AutoExecutable = parsed.AutoExecutable

	// Validate required fields
	if s.SuggestionText == "" {
		return errors.New("suggestion_text is empty")
	}
	if s.RiskLevel == "" {
		s.RiskLevel = "medium"
	}
	if s.SuggestedAction == "" {
		s.SuggestedAction = "other"
	}
	return nil
}

// extractJSONBlock extracts a JSON object from text that may be wrapped in markdown code fences.
func extractJSONBlock(text string) string {
	text = strings.TrimSpace(text)
	// Try markdown code block first
	if idx := strings.Index(text, "```json"); idx >= 0 {
		rest := text[idx+7:]
		if end := strings.Index(rest, "```"); end >= 0 {
			return strings.TrimSpace(rest[:end])
		}
	}
	// Try raw JSON object
	idx := strings.Index(text, "{")
	if idx < 0 {
		return ""
	}
	depth := 0
	for i := idx; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[idx : i+1]
			}
		}
	}
	return ""
}

func suggestTextForType(t string) string {
	switch t {
	case TypeLossOrder:
		return "建议审查该订单的采购成本和售价，评估是否调价或取消订单止损"
	case TypeOutOfStock:
		return "建议立即补货并检查安全库存设置，避免再次缺货"
	case TypeLogisticsAbnormal:
		return "建议联系物流商确认包裹状态，必要时发起理赔或补发"
	case TypeFeeAbnormal:
		return "建议核对平台费用明细，确认是否有误收或新增收费项目"
	default:
		return "建议人工审查异常详情，根据具体情况处理"
	}
}

func suggestActionForType(t string) string {
	switch t {
	case TypeLossOrder:
		return "adjust_price"
	case TypeOutOfStock:
		return "restock"
	case TypeLogisticsAbnormal:
		return "contact_carrier"
	case TypeFeeAbnormal:
		return "investigate_fee"
	default:
		return "other"
	}
}

// AutoDetect scans all data sources for anomalies and creates exception records
// for any that don't already exist (duplicate avoidance per source_type+source_id).
func (s *Service) AutoDetect(ctx context.Context) ([]ExceptionItem, error) {
	s.logger.Info("starting auto anomaly detection")

	var all []ExceptionItem

	for _, detect := range []struct {
		name string
		fn   func(context.Context) ([]ExceptionItem, error)
	}{
		{"loss_orders", s.detectLossOrders},
		{"out_of_stock", s.detectOutOfStock},
		{"logistics_abnormal", s.detectLogisticsAbnormal},
		{"fee_abnormal", s.detectFeeAbnormal},
	} {
		items, err := detect.fn(ctx)
		if err != nil {
			s.logger.Warn("anomaly detection step failed", zap.String("step", detect.name), zap.Error(err))
			continue
		}
		all = append(all, items...)
	}

	s.logger.Info("auto anomaly detection complete", zap.Int("created", len(all)))
	return all, nil
}

func (s *Service) detectLossOrders(ctx context.Context) ([]ExceptionItem, error) {
	type row struct {
		OrderID int64
		Profit  float64
	}
	var rows []row
	if err := s.db.WithContext(ctx).Table("order_profit_record").
		Select("order_id, profit").
		Where("profit < 0").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	var result []ExceptionItem
	for _, r := range rows {
		if ok, _ := s.exceptionExists(ctx, TypeLossOrder, r.OrderID); ok {
			continue
		}
		e := ExceptionItem{
			SourceModule: "order",
			SourceType:   TypeLossOrder,
			SourceID:     &r.OrderID,
			Severity:     "high",
			Title:        "亏损订单",
			Description:  fmt.Sprintf("订单 #%d 亏损 %.2f", r.OrderID, r.Profit),
			Status:       "open",
		}
		if err := s.db.WithContext(ctx).Create(&e).Error; err != nil {
			s.logger.Warn("failed to create loss order exception", zap.Error(err))
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

func (s *Service) detectOutOfStock(ctx context.Context) ([]ExceptionItem, error) {
	type row struct {
		SkuID          int64
		Quantity       int
		SafetyStock    int
		Available      int
	}
	var rows []row
	if err := s.db.WithContext(ctx).Table("inventory").
		Select("sku_id, quantity, safety_stock, (quantity - locked_quantity) AS available").
		Where("(quantity - locked_quantity) <= safety_stock").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	var result []ExceptionItem
	for _, r := range rows {
		sid := r.SkuID
		if ok, _ := s.exceptionExists(ctx, TypeOutOfStock, sid); ok {
			continue
		}
		title := "库存不足"
		if r.Quantity == 0 {
			title = "缺货"
		}
		e := ExceptionItem{
			SourceModule: "inventory",
			SourceType:   TypeOutOfStock,
			SourceID:     &sid,
			Severity:     "high",
			Title:        title,
			Description:  fmt.Sprintf("SKU #%d 可用库存 %d，安全库存 %d", r.SkuID, r.Available, r.SafetyStock),
			Status:       "open",
		}
		if err := s.db.WithContext(ctx).Create(&e).Error; err != nil {
			s.logger.Warn("failed to create out-of-stock exception", zap.Error(err))
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

func (s *Service) detectLogisticsAbnormal(ctx context.Context) ([]ExceptionItem, error) {
	var result []ExceptionItem

	// 1. Lost/returned/damaged shipments from fulfillment_tracking
	type trackRow struct {
		OrderID    int64
		IsLost     bool
		IsReturned bool
		IsDamaged  bool
	}
	var tracks []trackRow
	if err := s.db.WithContext(ctx).Table("fulfillment_tracking").
		Select("order_id, is_lost, is_returned, is_damaged").
		Where("is_lost = ? OR is_returned = ? OR is_damaged = ?", true, true, true).
		Scan(&tracks).Error; err != nil {
		return nil, err
	}
	for _, t := range tracks {
		if ok, _ := s.exceptionExists(ctx, TypeLogisticsAbnormal, t.OrderID); ok {
			continue
		}
		desc := "物流异常"
		switch {
		case t.IsLost:
			desc = "包裹丢失"
		case t.IsReturned:
			desc = "包裹退回"
		case t.IsDamaged:
			desc = "包裹损坏"
		}
		e := ExceptionItem{
			SourceModule: "shipping",
			SourceType:   TypeLogisticsAbnormal,
			SourceID:     &t.OrderID,
			Severity:     "high",
			Title:        desc,
			Description:  fmt.Sprintf("订单 #%d 物流状态异常: %s", t.OrderID, desc),
			Status:       "open",
		}
		if err := s.db.WithContext(ctx).Create(&e).Error; err != nil {
			s.logger.Warn("failed to create logistics exception", zap.Error(err))
			continue
		}
		result = append(result, e)
	}

	// 2. Stale shipments: shipped >14 days without delivery
	type staleRow struct {
		ID        int64
		ShippedAt *time.Time
	}
	var stales []staleRow
	if err := s.db.WithContext(ctx).Table("sales_order").
		Select("id, shipped_at").
		Where("shipped_at IS NOT NULL AND delivered_at IS NULL").
		Scan(&stales).Error; err != nil {
		return nil, err
	}
	cutoff := time.Now().Add(-14 * 24 * time.Hour)
	for _, o := range stales {
		if o.ShippedAt == nil || o.ShippedAt.After(cutoff) {
			continue
		}
		if ok, _ := s.exceptionExists(ctx, TypeLogisticsAbnormal, o.ID); ok {
			continue
		}
		e := ExceptionItem{
			SourceModule: "shipping",
			SourceType:   TypeLogisticsAbnormal,
			SourceID:     &o.ID,
			Severity:     "medium",
			Title:        "配送延迟",
			Description:  fmt.Sprintf("订单 #%d 已发货超过14天仍未送达", o.ID),
			Status:       "open",
		}
		if err := s.db.WithContext(ctx).Create(&e).Error; err != nil {
			s.logger.Warn("failed to create delivery delay exception", zap.Error(err))
			continue
		}
		result = append(result, e)
	}

	return result, nil
}

func (s *Service) detectFeeAbnormal(ctx context.Context) ([]ExceptionItem, error) {
	type feeRow struct {
		ID           int64
		PlatformID   *int64
		PayAmount    float64
		PlatformFee  float64
		ExpectedRate float64
	}
	var rows []feeRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT o.id, o.platform_id, o.pay_amount, o.platform_fee,
		       COALESCE((
		           SELECT fee_rate_pct FROM platform_fee_rule
		           WHERE platform_id = o.platform_id
		             AND fee_type = 'commission'
		             AND status = 'active'
		           ORDER BY priority LIMIT 1
		       ), 15) AS expected_rate
		FROM sales_order o
		WHERE o.platform_fee > 0 AND o.pay_amount > 0`).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	var result []ExceptionItem
	for _, r := range rows {
		expected := r.PayAmount * r.ExpectedRate / 100
		if expected <= 0 {
			continue
		}
		deviation := math.Abs(r.PlatformFee-expected) / expected
		if deviation <= 0.2 {
			continue
		}
		if ok, _ := s.exceptionExists(ctx, TypeFeeAbnormal, r.ID); ok {
			continue
		}
		e := ExceptionItem{
			SourceModule: "order",
			SourceType:   TypeFeeAbnormal,
			SourceID:     &r.ID,
			Severity:     "medium",
			Title:        "平台费用异常",
			Description:  fmt.Sprintf("订单 #%d 实际平台费用 %.2f，预期 %.2f (偏差 %.1f%%)", r.ID, r.PlatformFee, expected, deviation*100),
			Status:       "open",
		}
		if err := s.db.WithContext(ctx).Create(&e).Error; err != nil {
			s.logger.Warn("failed to create fee abnormal exception", zap.Error(err))
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

// exceptionExists checks whether an exception already exists for the given source_type + source_id.
func (s *Service) exceptionExists(ctx context.Context, sourceType string, sourceID int64) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&ExceptionItem{}).
		Where("source_type = ? AND source_id = ?", sourceType, sourceID).
		Count(&count).Error
	return count > 0, err
}
