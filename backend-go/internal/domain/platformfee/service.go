package platformfee

import (
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides platformfee business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new platformfee service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated fee rules with optional filter.
func (s *Service) List(p *common.Pagination, f *RuleListFilter) ([]PlatformFeeRule, int64, error) {
	q := s.db.Model(&PlatformFeeRule{})
	if f != nil {
		if f.PlatformID != nil {
			q = q.Where("platform_id = ?", *f.PlatformID)
		}
		if f.FeeType != "" {
			q = q.Where("fee_type = ?", f.FeeType)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []PlatformFeeRule
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns a single fee rule by id.
func (s *Service) Get(id int64) (*PlatformFeeRule, error) {
	var r PlatformFeeRule
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// Create inserts a new fee rule.
func (s *Service) Create(in *CreateRuleInput) (*PlatformFeeRule, error) {
	r := PlatformFeeRule{
		PlatformID:    in.PlatformID,
		CountryCode:   in.CountryCode,
		CategoryID:    in.CategoryID,
		FeeType:       in.FeeType,
		EffectiveFrom: in.EffectiveFrom,
		EffectiveTo:   in.EffectiveTo,
		Remark:        in.Remark,
	}
	if in.FeeRatePct != nil {
		r.FeeRatePct = *in.FeeRatePct
	}
	if in.FixedAmount != nil {
		r.FixedAmount = *in.FixedAmount
	}
	if in.MinAmount != nil {
		r.MinAmount = *in.MinAmount
	}
	if in.MaxAmount != nil {
		r.MaxAmount = *in.MaxAmount
	}
	if in.Priority != nil {
		r.Priority = *in.Priority
	}
	if in.Currency != "" {
		r.Currency = in.Currency
	} else {
		r.Currency = "CNY"
	}
	if in.Status != "" {
		r.Status = in.Status
	} else {
		r.Status = "active"
	}
	if err := s.db.Create(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// Update applies partial updates to a fee rule.
func (s *Service) Update(id int64, in *UpdateRuleInput) (*PlatformFeeRule, error) {
	var r PlatformFeeRule
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.PlatformID != nil {
		updates["platform_id"] = *in.PlatformID
	}
	if in.CountryCode != nil {
		updates["country_code"] = *in.CountryCode
	}
	if in.CategoryID != nil {
		updates["category_id"] = *in.CategoryID
	}
	if in.FeeType != nil {
		updates["fee_type"] = *in.FeeType
	}
	if in.FeeRatePct != nil {
		updates["fee_rate_pct"] = *in.FeeRatePct
	}
	if in.FixedAmount != nil {
		updates["fixed_amount"] = *in.FixedAmount
	}
	if in.MinAmount != nil {
		updates["min_amount"] = *in.MinAmount
	}
	if in.MaxAmount != nil {
		updates["max_amount"] = *in.MaxAmount
	}
	if in.Currency != nil {
		updates["currency"] = *in.Currency
	}
	if in.EffectiveFrom != nil {
		updates["effective_from"] = *in.EffectiveFrom
	}
	if in.EffectiveTo != nil {
		updates["effective_to"] = *in.EffectiveTo
	}
	if in.Priority != nil {
		updates["priority"] = *in.Priority
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.Remark != nil {
		updates["remark"] = *in.Remark
	}
	if len(updates) == 0 {
		return &r, nil
	}
	if err := s.db.Model(&r).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// Delete removes a fee rule by id.
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&PlatformFeeRule{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Calculate computes the fee for the given request by matching the highest-priority rule.
// Priority convention follows shipping module: lower priority number = higher priority.
func (s *Service) Calculate(req *CalculateRequest) (*CalculateResult, error) {
	now := time.Now()
	q := s.db.Model(&PlatformFeeRule{}).
		Where("platform_id = ?", req.PlatformID).
		Where("status = ?", "active").
		Where("(effective_from IS NULL OR effective_from <= ?)", now).
		Where("(effective_to IS NULL OR effective_to >= ?)", now)
	// Prefer rules matching category and country, then fall back to general rules
	if req.CategoryID != nil {
		q = q.Where("(category_id IS NULL OR category_id = ?)", *req.CategoryID)
	}
	if req.CountryCode != "" {
		q = q.Where("(country_code = '' OR country_code = ?)", req.CountryCode)
	}
	// Highest priority = lowest priority number (consistent with shipping module)
	var r PlatformFeeRule
	err := q.Order("priority ASC, id ASC").First(&r).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &CalculateResult{Matched: false}, nil
		}
		return nil, err
	}

	fee := computeFee(&r, req.Amount)
	// Clamp to [min_amount, max_amount] when max_amount > 0
	if r.MinAmount > 0 && fee < r.MinAmount {
		fee = r.MinAmount
	}
	if r.MaxAmount > 0 && fee > r.MaxAmount {
		fee = r.MaxAmount
	}

	return &CalculateResult{
		RuleID:        r.ID,
		FeeType:       r.FeeType,
		FeeRatePct:    r.FeeRatePct,
		FixedAmount:   r.FixedAmount,
		CalculatedFee: fee,
		MinAmount:     r.MinAmount,
		MaxAmount:     r.MaxAmount,
		Currency:      r.Currency,
		Matched:       true,
	}, nil
}

// computeFee calculates the raw fee based on fee_type before min/max clamping.
func computeFee(r *PlatformFeeRule, amount float64) float64 {
	switch r.FeeType {
	case "commission":
		return amount * r.FeeRatePct / 100.0
	case "payment":
		return amount * r.FeeRatePct / 100.0
	case "fixed":
		return r.FixedAmount
	case "storage":
		return r.FixedAmount
	case "other":
		return r.FixedAmount + amount*r.FeeRatePct/100.0
	default:
		return r.FixedAmount
	}
}
