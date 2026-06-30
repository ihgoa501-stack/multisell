package tariff

import (
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides tariff business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new tariff service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated tariff rules with optional filter.
func (s *Service) List(p *common.Pagination, f *RuleListFilter) ([]TariffRule, int64, error) {
	q := s.db.Model(&TariffRule{})
	if f != nil {
		if f.CountryCode != "" {
			q = q.Where("country_code = ?", f.CountryCode)
		}
		if f.HSCode != "" {
			q = q.Where("(hs_code = ? OR hs_code_prefix = ?)", f.HSCode, f.HSCode)
		}
		if f.Incoterm != "" {
			q = q.Where("incoterm = ?", f.Incoterm)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []TariffRule
	if err := q.Order("priority ASC, id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns a single tariff rule by id.
func (s *Service) Get(id int64) (*TariffRule, error) {
	var r TariffRule
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// Create inserts a new tariff rule.
func (s *Service) Create(in *CreateRuleInput) (*TariffRule, error) {
	r := TariffRule{
		CountryCode:   in.CountryCode,
		HSCode:        in.HSCode,
		HSCodePrefix:  in.HSCodePrefix,
		EffectiveFrom: in.EffectiveFrom,
		EffectiveTo:   in.EffectiveTo,
		Remark:        in.Remark,
	}
	if in.DutyRatePct != nil {
		r.DutyRatePct = *in.DutyRatePct
	}
	if in.VatRatePct != nil {
		r.VatRatePct = *in.VatRatePct
	}
	if in.OtherTaxRatePct != nil {
		r.OtherTaxRatePct = *in.OtherTaxRatePct
	}
	if in.MinThresholdUSD != nil {
		r.MinThresholdUSD = *in.MinThresholdUSD
	}
	if in.MaxThresholdUSD != nil {
		r.MaxThresholdUSD = *in.MaxThresholdUSD
	}
	if in.Priority != nil {
		r.Priority = *in.Priority
	}
	if in.Incoterm != "" {
		r.Incoterm = in.Incoterm
	} else {
		r.Incoterm = "DDU"
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

// Update applies partial updates to a tariff rule.
func (s *Service) Update(id int64, in *UpdateRuleInput) (*TariffRule, error) {
	var r TariffRule
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.CountryCode != nil {
		updates["country_code"] = *in.CountryCode
	}
	if in.HSCode != nil {
		updates["hs_code"] = *in.HSCode
	}
	if in.HSCodePrefix != nil {
		updates["hs_code_prefix"] = *in.HSCodePrefix
	}
	if in.DutyRatePct != nil {
		updates["duty_rate_pct"] = *in.DutyRatePct
	}
	if in.VatRatePct != nil {
		updates["vat_rate_pct"] = *in.VatRatePct
	}
	if in.OtherTaxRatePct != nil {
		updates["other_tax_rate_pct"] = *in.OtherTaxRatePct
	}
	if in.MinThresholdUSD != nil {
		updates["min_threshold_usd"] = *in.MinThresholdUSD
	}
	if in.MaxThresholdUSD != nil {
		updates["max_threshold_usd"] = *in.MaxThresholdUSD
	}
	if in.Incoterm != nil {
		updates["incoterm"] = *in.Incoterm
	}
	if in.Priority != nil {
		updates["priority"] = *in.Priority
	}
	if in.EffectiveFrom != nil {
		updates["effective_from"] = *in.EffectiveFrom
	}
	if in.EffectiveTo != nil {
		updates["effective_to"] = *in.EffectiveTo
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
	return &r, nil
}

// Delete removes a tariff rule by id.
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&TariffRule{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Decide computes the DDP/DDU decision for the given request.
// It finds the best-matching tariff rule for the destination country,
// calculates all applicable duties and taxes, and recommends DDP or DDU.
//
// DDP (Delivered Duty Paid) = seller pays duties/taxes.
// DDU (Delivered Duty Unpaid) = buyer pays duties/taxes at destination.
//
// The decision algorithm:
//  1. Find the highest-priority active rule matching country (+ optional HS code).
//  2. Calculate duty + VAT + other tax amounts.
//  3. If total duty/tax is below min_threshold, recommend DDU (negligible).
//  4. If total duty/tax is below a cost-of-capital threshold (~5% of product value),
//     recommend DDP (seller can absorb).
//  5. Otherwise, respect the rule's incoterm preference.
func (s *Service) Decide(req *DecisionRequest) (*DecisionResult, error) {
	now := time.Now()
	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}
	totalValue := req.ProductValueUSD * float64(qty)

	// Find the best-matching active rule for the destination country.
	q := s.db.Model(&TariffRule{}).
		Where("country_code = ?", req.DestinationCountry).
		Where("status = ?", "active").
		Where("(effective_from IS NULL OR effective_from <= ?)", now).
		Where("(effective_to IS NULL OR effective_to >= ?)", now)

	// Prefer rules with HS code or HS code prefix matching.
	// First pass: find a rule matching HS code.
	var rules []TariffRule
	if err := q.Order("priority ASC, id DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		// No tariff rules for this country — default to DDU with no duty.
		totalValue := req.ProductValueUSD * float64(qty)
		return &DecisionResult{
			Incoterm:        "DDU",
			TotalDutyTaxUSD: 0,
			DutyAmountUSD:   0,
			VatAmountUSD:    0,
			OtherTaxAmountUSD: 0,
			RulesMatched:    nil,
			IncotermReason:  "No applicable tariff rule found; defaulting to DDU",
			TotalValueUSD:   totalValue,
		}, nil
	}

	// Try to find the best-matched rule considering HS code specificity.
	bestRule := s.pickBestRule(rules, req.HSCode)
	if bestRule == nil {
		bestRule = &rules[0]
	}

	// Apply de minimis threshold: if total value is below min_threshold, no duty/tax.
	baseValue := totalValue
	if bestRule.MinThresholdUSD > 0 && totalValue <= bestRule.MinThresholdUSD {
		return &DecisionResult{
			Incoterm:         "DDU",
			TotalDutyTaxUSD:  0,
			DutyAmountUSD:    0,
			VatAmountUSD:     0,
			OtherTaxAmountUSD: 0,
			RulesMatched:     []RuleMatchItem{s.buildMatchItem(bestRule, 0, 0, 0, baseValue)},
			IncotermReason:   "Value below de minimis threshold — no duty or tax applicable",
			TotalValueUSD:    totalValue,
		}, nil
	}

	// Calculate duty on the full value (or value capped by max_threshold if applicable).
	assessableValue := baseValue
	if bestRule.MaxThresholdUSD > 0 && baseValue > bestRule.MaxThresholdUSD {
		assessableValue = bestRule.MaxThresholdUSD
	}

	dutyAmount := assessableValue * bestRule.DutyRatePct / 100.0
	// VAT is applied after duty (duty + value as taxable base for VAT).
	taxableForVAT := assessableValue + dutyAmount
	vatAmount := taxableForVAT * bestRule.VatRatePct / 100.0
	// Other taxes applied on the original assessable value.
	otherTax := assessableValue * bestRule.OtherTaxRatePct / 100.0
	totalDutyTax := dutyAmount + vatAmount + otherTax

	// Build the result item.
	matchItem := s.buildMatchItem(bestRule, dutyAmount, vatAmount, otherTax, assessableValue)

	// Determine DDP vs DDU.
	incoterm, reason := s.decideIncoterm(bestRule, totalDutyTax, totalValue)

	return &DecisionResult{
		Incoterm:          incoterm,
		TotalDutyTaxUSD:   totalDutyTax,
		DutyAmountUSD:     dutyAmount,
		VatAmountUSD:      vatAmount,
		OtherTaxAmountUSD: otherTax,
		RulesMatched:      []RuleMatchItem{matchItem},
		IncotermReason:    reason,
		TotalValueUSD:     totalValue,
	}, nil
}

// pickBestRule selects the most specific rule from the candidate list.
// Preference: exact HS code match > HS code prefix match > country-only rule.
func (s *Service) pickBestRule(rules []TariffRule, hsCode string) *TariffRule {
	if len(rules) == 0 {
		return nil
	}
	if hsCode == "" {
		// No HS code provided; use the highest-priority country-only rule
		for i := range rules {
			if rules[i].HSCode == "" && rules[i].HSCodePrefix == "" {
				return &rules[i]
			}
		}
		return &rules[0]
	}

	// First pass: exact HS code match (prefer prefix match since HS codes vary in length)
	var exactMatch, prefixMatch *TariffRule
	for i := range rules {
		if rules[i].HSCode != "" && rules[i].HSCode == hsCode {
			exactMatch = &rules[i]
			break // priority-ordered; first match is best
		}
		if rules[i].HSCodePrefix != "" && len(hsCode) >= len(rules[i].HSCodePrefix) &&
			hsCode[:len(rules[i].HSCodePrefix)] == rules[i].HSCodePrefix {
			if prefixMatch == nil || rules[i].Priority < prefixMatch.Priority {
				prefixMatch = &rules[i]
			}
		}
	}
	if exactMatch != nil {
		return exactMatch
	}
	if prefixMatch != nil {
		return prefixMatch
	}
	// Fall back to the first country-only rule
	for i := range rules {
		if rules[i].HSCode == "" && rules[i].HSCodePrefix == "" {
			return &rules[i]
		}
	}
	return &rules[0]
}

// decideIncoterm determines whether DDP or DDU is appropriate.
func (s *Service) decideIncoterm(rule *TariffRule, totalDutyTax, totalValue float64) (string, string) {
	if totalDutyTax <= 0 {
		return "DDU", "No duty or tax applicable — defaulting to DDU"
	}

	// If total duty is negligible relative to product value (< 1%), recommend DDP.
	dutyRatio := totalDutyTax / totalValue
	if dutyRatio < 0.01 {
		return "DDP", "Duty/tax is negligible (under 1% of value); seller can absorb"
	}

	// If total duty is high (> 20%), recommend DDU since it's a significant cost.
	if dutyRatio > 0.20 {
		return "DDU", "Duty/tax exceeds 20% of product value; buyer should bear cost"
	}

	// Otherwise, respect the rule's configured incoterm.
	if rule.Incoterm == "DDP" {
		return "DDP", "Configured incoterm DDP based on tariff rule preference"
	}
	return "DDU", "Configured incoterm DDU based on tariff rule preference"
}

// buildMatchItem creates a rule match item with calculated amounts.
func (s *Service) buildMatchItem(rule *TariffRule, dutyAmt, vatAmt, otherAmt, taxableValue float64) RuleMatchItem {
	return RuleMatchItem{
		RuleID:          rule.ID,
		CountryCode:     rule.CountryCode,
		HSCode:          rule.HSCode,
		HSCodePrefix:    rule.HSCodePrefix,
		DutyRatePct:     rule.DutyRatePct,
		VatRatePct:      rule.VatRatePct,
		OtherTaxRatePct: rule.OtherTaxRatePct,
		Incoterm:        rule.Incoterm,
		Priority:        rule.Priority,
		DutyAmountUSD:   dutyAmt,
		VatAmountUSD:    vatAmt,
		OtherTaxAmount:  otherAmt,
		}
	}
