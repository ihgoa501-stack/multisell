package decision

import (
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides decision business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new decision service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated decisions with optional filter.
func (s *Service) List(p *common.Pagination, f *ListFilter) ([]PreListingDecision, int64, error) {
	q := s.db.Model(&PreListingDecision{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("reasoning ILIKE ? OR trace_id ILIKE ?", like, like)
		}
		if f.SkuID != nil {
			q = q.Where("sku_id = ?", *f.SkuID)
		}
		if f.PlatformID != nil {
			q = q.Where("platform_id = ?", *f.PlatformID)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.RiskLevel != "" {
			q = q.Where("risk_level = ?", f.RiskLevel)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []PreListingDecision
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns a single decision.
func (s *Service) Get(id int64) (*PreListingDecision, error) {
	var d PreListingDecision
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// Create inserts a new decision.
func (s *Service) Create(in *CreateInput) (*PreListingDecision, error) {
	status := in.Status
	if status == "" {
		status = "pending"
	}
	decisionPoint := in.DecisionPoint
	if decisionPoint == "" {
		decisionPoint = "pre_listing"
	}
	riskLevel := in.RiskLevel
	if riskLevel == "" {
		riskLevel = "medium"
	}
	d := PreListingDecision{
		SkuID:         in.SkuID,
		PlatformID:    in.PlatformID,
		CountryCode:   in.CountryCode,
		DecisionPoint: decisionPoint,
		RiskLevel:     riskLevel,
		Recommendation: in.Recommendation,
		Reasoning:     in.Reasoning,
		Status:        status,
		TraceID:       in.TraceID,
	}
	if in.EstimatedRevenue != nil {
		d.EstimatedRevenue = *in.EstimatedRevenue
	}
	if in.EstimatedProductCost != nil {
		d.EstimatedProductCost = *in.EstimatedProductCost
	}
	if in.EstimatedShippingCost != nil {
		d.EstimatedShippingCost = *in.EstimatedShippingCost
	}
	if in.EstimatedPlatformFee != nil {
		d.EstimatedPlatformFee = *in.EstimatedPlatformFee
	}
	if in.EstimatedPaymentFee != nil {
		d.EstimatedPaymentFee = *in.EstimatedPaymentFee
	}
	if in.EstimatedOtherFee != nil {
		d.EstimatedOtherFee = *in.EstimatedOtherFee
	}
	if in.EstimatedProfit != nil {
		d.EstimatedProfit = *in.EstimatedProfit
	}
	if in.ProfitMargin != nil {
		d.ProfitMargin = *in.ProfitMargin
	}
	if in.ConfidenceScore != nil {
		d.ConfidenceScore = *in.ConfidenceScore
	}
	if err := s.db.Create(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// Update applies partial updates to a decision.
func (s *Service) Update(id int64, in *UpdateInput) (*PreListingDecision, error) {
	var d PreListingDecision
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.PlatformID != nil {
		updates["platform_id"] = *in.PlatformID
	}
	if in.CountryCode != nil {
		updates["country_code"] = *in.CountryCode
	}
	if in.EstimatedRevenue != nil {
		updates["estimated_revenue"] = *in.EstimatedRevenue
	}
	if in.EstimatedProductCost != nil {
		updates["estimated_product_cost"] = *in.EstimatedProductCost
	}
	if in.EstimatedShippingCost != nil {
		updates["estimated_shipping_cost"] = *in.EstimatedShippingCost
	}
	if in.EstimatedPlatformFee != nil {
		updates["estimated_platform_fee"] = *in.EstimatedPlatformFee
	}
	if in.EstimatedPaymentFee != nil {
		updates["estimated_payment_fee"] = *in.EstimatedPaymentFee
	}
	if in.EstimatedOtherFee != nil {
		updates["estimated_other_fee"] = *in.EstimatedOtherFee
	}
	if in.EstimatedProfit != nil {
		updates["estimated_profit"] = *in.EstimatedProfit
	}
	if in.ProfitMargin != nil {
		updates["profit_margin"] = *in.ProfitMargin
	}
	if in.ConfidenceScore != nil {
		updates["confidence_score"] = *in.ConfidenceScore
	}
	if in.RiskLevel != nil {
		updates["risk_level"] = *in.RiskLevel
	}
	if in.Recommendation != nil {
		updates["recommendation"] = *in.Recommendation
	}
	if in.Reasoning != nil {
		updates["reasoning"] = *in.Reasoning
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.TraceID != nil {
		updates["trace_id"] = *in.TraceID
	}
	if len(updates) == 0 {
		return &d, nil
	}
	if err := s.db.Model(&d).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// Delete removes a decision by id.
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&PreListingDecision{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Approve marks a decision as approved.
func (s *Service) Approve(id int64, in *ApproveInput) (*PreListingDecision, error) {
	var d PreListingDecision
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":     "approved",
		"decided_by": in.DecidedBy,
		"decided_at": &now,
	}
	if err := s.db.Model(&d).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// Reject marks a decision as rejected and records the reason into reasoning.
func (s *Service) Reject(id int64, in *RejectInput) (*PreListingDecision, error) {
	var d PreListingDecision
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	reasoning := in.Reason
	updates := map[string]interface{}{
		"status":     "rejected",
		"decided_by": in.DecidedBy,
		"decided_at": &now,
		"reasoning":  reasoning,
	}
	if err := s.db.Model(&d).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// Summary returns aggregation by recommendation, risk_level, and status.
func (s *Service) Summary() (*Summary, error) {
	var total int64
	if err := s.db.Model(&PreListingDecision{}).Count(&total).Error; err != nil {
		return nil, err
	}
	byRecommendation, err := s.groupCount("recommendation")
	if err != nil {
		return nil, err
	}
	byRiskLevel, err := s.groupCount("risk_level")
	if err != nil {
		return nil, err
	}
	byStatus, err := s.groupCount("status")
	if err != nil {
		return nil, err
	}
	return &Summary{
		Total:            total,
		ByRecommendation: byRecommendation,
		ByRiskLevel:      byRiskLevel,
		ByStatus:         byStatus,
	}, nil
}

func (s *Service) groupCount(col string) (map[string]int64, error) {
	type row struct {
		Key string
		Cnt int64
	}
	var rows []row
	if err := s.db.Model(&PreListingDecision{}).
		Select(col + " AS key, COUNT(*) AS cnt").Group(col).Scan(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]int64, len(rows))
	for _, r := range rows {
		m[r.Key] = r.Cnt
	}
	return m, nil
}
