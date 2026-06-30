package landedcost

import (
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides landed cost business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new landed cost service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ---------- Lookup helpers ----------

// feeRuleRow is a scan target for the platform fee rule query.
type feeRuleRow struct {
	FeeRatePct float64
}

// exchangeRateRow is a scan target for the exchange rate query.
type exchangeRateRow struct {
	Rate float64
}

// lookupCommissionRate queries the best-matching commission rule for the given platform.
func (s *Service) lookupCommissionRate(platformID int64, categoryID *int64, countryCode string) float64 {
	q := s.db.Table("platform_fee_rule").
		Select("fee_rate_pct").
		Where("platform_id = ?", platformID).
		Where("status = ?", "active").
		Where("fee_type = ?", "commission").
		Where("(effective_from IS NULL OR effective_from <= NOW())").
		Where("(effective_to IS NULL OR effective_to >= NOW())")

	if categoryID != nil {
		q = q.Where("(category_id IS NULL OR category_id = ?)", *categoryID)
	}
	if countryCode != "" {
		q = q.Where("(country_code = '' OR country_code = ?)", countryCode)
	}

	var row feeRuleRow
	if err := q.Order("priority ASC, id ASC").Take(&row).Error; err != nil {
		return 0
	}
	return row.FeeRatePct
}

// lookupExchangeRate queries the latest exchange rate from CNY to the target currency.
// It first tries platform-specific currency, then default USD.
func (s *Service) lookupExchangeRate(toCurrency string) float64 {
	if toCurrency == "" || toCurrency == "CNY" {
		return 1.0
	}
	q := s.db.Table("exchange_rate").
		Select("rate").
		Where("UPPER(from_currency) = ?", "CNY").
		Where("UPPER(to_currency) = ?", toCurrency).
		Order("effective_date DESC, id DESC")

	var row exchangeRateRow
	if err := q.Take(&row).Error; err != nil {
		return 0
	}
	return row.Rate
}

// lookupPlatformCurrency returns the currency for a platform by its ID.
func (s *Service) lookupPlatformCurrency(platformID int64) string {
	var code string
	if err := s.db.Table("platform").Select("code").Where("id = ?", platformID).Take(&code).Error; err != nil {
		return "USD"
	}
	// Map common platform codes to currencies
	switch code {
	case "ozon":
		return "RUB"
	case "shopee":
		return "USD" // Shopee uses multiple local currencies; default USD
	case "aliexpress", "alibaba":
		return "USD"
	case "amazon":
		return "USD"
	case "ebay":
		return "USD"
	case "etsy":
		return "USD"
	case "mercadolibre":
		return "USD"
	default:
		return "USD"
	}
}

// ---------- Calculation ----------

// calculate performs the landed cost computation given all inputs.
// This is a pure function — no DB access.
func calculate(in *CalculateRequest, platformFeePct, exchangeRate float64) *CalculateResult {
	// Use input values or defaults
	unitCost := in.UnitCostCNY
	freight := in.FreightCNY
	insurance := in.InsuranceCNY

	dutyRate := 0.0
	if in.DutyRate != nil {
		dutyRate = *in.DutyRate
	}

	vatRate := 0.0
	if in.VatRate != nil {
		vatRate = *in.VatRate
	}

	feePct := platformFeePct
	if in.PlatformFeePct != nil {
		feePct = *in.PlatformFeePct
	}

	clearing := 0.0
	if in.ClearingFeeCNY != nil {
		clearing = *in.ClearingFeeCNY
	}

	margin := in.TargetMarginPct
	if margin <= 0 {
		margin = 15.0 // default 15%
	}

	rate := exchangeRate
	if rate <= 0 {
		rate = 7.0 // fallback CNY->USD
	}

	// Step 1: Calculate duty
	dutyCNY := unitCost * (dutyRate / 100.0)

	// Step 2: Calculate VAT: (cost + freight + insurance + duty) * vat_rate
	cifCNY := unitCost + freight + insurance + dutyCNY
	vatCNY := cifCNY * (vatRate / 100.0)

	// Step 3: Calculate total without platform fee
	totalBeforePlatformCNY := cifCNY + vatCNY + clearing

	// Step 4: Platform fee is a % of selling price (local). Solve iteratively.
	//   totalCNY = totalBeforePlatformCNY + platformFeeCNY
	//   platformFeeCNY = targetPrice * exchangeRate * (feePct / 100)
	//   totalLocal = totalCNY / exchangeRate
	//   targetPrice = totalLocal / (1 - margin)
	//  → targetPrice = (totalBeforePlatformCNY / exchangeRate) / (1 - margin - feePct/100)
	// When feePct is treated as cost % of selling price.
	denom := 1.0 - (margin / 100.0) - (feePct / 100.0)
	if denom <= 0 {
		denom = 0.01 // prevent division by zero
	}
	targetPrice := (totalBeforePlatformCNY / rate) / denom
	targetPrice = math.Round(targetPrice*100) / 100

	// Recalculate platform fee in CNY
	platformFeeCNY := targetPrice * rate * (feePct / 100.0)
	platformFeeCNY = math.Round(platformFeeCNY*100) / 100

	// Total costs in CNY
	totalCNY := totalBeforePlatformCNY + platformFeeCNY
	totalCNY = math.Round(totalCNY*100) / 100

	// Total costs in local currency
	totalLocal := totalCNY / rate
	totalLocal = math.Round(totalLocal*100) / 100

	// Actual profit margin
	profitMargin := 0.0
	if targetPrice > 0 {
		profitMargin = ((targetPrice - totalLocal) / targetPrice) * 100.0
		profitMargin = math.Round(profitMargin*100) / 100
	}

	now := time.Now()

	lc := LandedCost{
		ProductID:      in.ProductID,
		PlatformID:     in.PlatformID,
		UnitCostCNY:    math.Round(unitCost*100) / 100,
		FreightCNY:     math.Round(freight*100) / 100,
		InsuranceCNY:   math.Round(insurance*100) / 100,
		DutyRate:       dutyRate,
		DutyCNY:        math.Round(dutyCNY*100) / 100,
		VatRate:        vatRate,
		VatCNY:         math.Round(vatCNY*100) / 100,
		PlatformFeePct: feePct,
		PlatformFeeCNY: platformFeeCNY,
		ClearingFeeCNY: math.Round(clearing*100) / 100,
		TotalCostCNY:   totalCNY,
		ExchangeRate:   rate,
		TotalCostLocal: totalLocal,
		TargetPrice:    targetPrice,
		CalculatedAt:   now,
	}

	return &CalculateResult{
		LandedCost:       lc,
		ProfitMarginPct:  profitMargin,
		RecommendedPrice: targetPrice,
	}
}

// Calculate computes landed cost, auto-populating defaults from DB, saves to DB.
func (s *Service) Calculate(in *CalculateRequest) (*CalculateResult, error) {
	// 1. Look up platform fee rate
	feePct := s.lookupCommissionRate(in.PlatformID, in.CategoryID, in.CountryCode)

	// 2. Look up exchange rate
	currency := s.lookupPlatformCurrency(in.PlatformID)
	rate := s.lookupExchangeRate(currency)

	// 3. Compute
	res := calculate(in, feePct, rate)

	// 4. Save to DB
	if err := s.db.Create(&res.LandedCost).Error; err != nil {
		s.logger.Warn("landedcost: failed to save calculation", zap.Error(err))
		// Return the result even if we can't save — the user still wants the numbers
	}

	return res, nil
}

// GetByProductPlatform retrieves the latest landed cost for a product on a platform.
func (s *Service) GetByProductPlatform(productID, platformID int64) (*LandedCost, error) {
	var lc LandedCost
	err := s.db.Where("product_id = ? AND platform_id = ?", productID, platformID).
		Order("calculated_at DESC").
		First(&lc).Error
	if err != nil {
		return nil, err
	}
	return &lc, nil
}

// GetByID retrieves a single landed cost record by ID.
func (s *Service) GetByID(id int64) (*LandedCost, error) {
	var lc LandedCost
	if err := s.db.First(&lc, id).Error; err != nil {
		return nil, err
	}
	return &lc, nil
}

// ListByProduct returns all landed cost records for a product.
func (s *Service) ListByProduct(productID int64) ([]LandedCost, error) {
	var items []LandedCost
	if err := s.db.Where("product_id = ?", productID).Order("calculated_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CompareAcrossPlatforms returns the latest landed cost for a product across multiple platforms.
// It returns one result per platform — the most recent calculation for each.
func (s *Service) CompareAcrossPlatforms(productID int64) ([]CompareItem, error) {
	var items []LandedCost
	err := s.db.Where("product_id = ?", productID).
		Order("calculated_at DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	// Deduplicate by platform — keep latest
	seen := make(map[int64]bool)
	var compare []CompareItem
	for i := range items {
		lc := items[i]
		if seen[lc.PlatformID] {
			continue
		}
		seen[lc.PlatformID] = true

		margin := 0.0
		if lc.TargetPrice > 0 {
			margin = ((lc.TargetPrice - lc.TotalCostLocal) / lc.TargetPrice) * 100
		}
		compare = append(compare, CompareItem{
			PlatformID:      lc.PlatformID,
			TotalCostLocal:  lc.TotalCostLocal,
			TargetPrice:     lc.TargetPrice,
			ProfitMarginPct: math.Round(margin*100) / 100,
		})
	}
	return compare, nil
}

// ---------- Format helpers for display ----------

// CostBreakdown returns a human-readable map of cost components for display.
func CostBreakdown(lc *LandedCost) map[string]interface{} {
	return map[string]interface{}{
		"采购成本 (CNY)":        fmt.Sprintf("%.2f", lc.UnitCostCNY),
		"运费 (CNY)":          fmt.Sprintf("%.2f", lc.FreightCNY),
		"保险费 (CNY)":         fmt.Sprintf("%.2f", lc.InsuranceCNY),
		"关税 (CNY)":          fmt.Sprintf("%.2f (税率 %.2f%%)", lc.DutyCNY, lc.DutyRate),
		"VAT (CNY)":          fmt.Sprintf("%.2f (税率 %.2f%%)", lc.VatCNY, lc.VatRate),
		"平台佣金 (CNY)":        fmt.Sprintf("%.2f (费率 %.2f%%)", lc.PlatformFeeCNY, lc.PlatformFeePct),
		"清关费 (CNY)":         fmt.Sprintf("%.2f", lc.ClearingFeeCNY),
		"总成本 (CNY)":         fmt.Sprintf("%.2f", lc.TotalCostCNY),
		"汇率":                fmt.Sprintf("%.4f", lc.ExchangeRate),
		"总成本 (本地货币)":       fmt.Sprintf("%.2f", lc.TotalCostLocal),
		"建议售价 (本地货币)":      fmt.Sprintf("%.2f", lc.TargetPrice),
	}
}
