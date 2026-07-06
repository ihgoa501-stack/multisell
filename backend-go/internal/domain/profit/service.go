package profit

import (
	"fmt"
	"math"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/exchangerate"
	"github.com/lingmirror/backend-go/internal/domain/platformfee"
	"github.com/lingmirror/backend-go/internal/domain/supplier"
	"github.com/lingmirror/backend-go/internal/domain/tariff"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides profit summary business logic.
type Service struct {
	db      *gorm.DB
	logger  *zap.Logger
	rateSvc *exchangerate.Service
}

// NewService creates a new profit summary service.
func NewService(db *gorm.DB, logger *zap.Logger, rateSvc *exchangerate.Service) *Service {
	return &Service{db: db, logger: logger, rateSvc: rateSvc}
}

// getCNYRate returns the current CNY->USD exchange rate from the exchangerate module.
// Falls back to 7.2 if the rate service is unavailable or the lookup fails.
func (s *Service) getCNYRate() float64 {
	if s.rateSvc == nil {
		return 7.2
	}
	rate, err := s.rateSvc.GetLatest("CNY", "USD")
	if err != nil {
		s.logger.Warn("failed to get CNY/USD exchange rate, using default 7.2", zap.Error(err))
		return 7.2
	}
	if rate.Rate <= 0 {
		return 7.2
	}
	return rate.Rate
}

// Calculate computes a full profit summary for a candidate product.
// It pulls purchase cost from candidate_product, shipping estimate from logistics
// (or uses a default fallback), platform fee from platformfee domain, and tariff
// from the tariff engine.
func (s *Service) Calculate(productID int64, calculatedBy string) (*ProfitResult, error) {
	var prod candidate.CandidateProduct
	if err := s.db.First(&prod, productID).Error; err != nil {
		return nil, err
	}

	if calculatedBy == "" {
		calculatedBy = "system"
	}

	// 1. Purchase cost (in USD, converts CNY->USD at current exchange rate)
	purchaseCost := prod.PurchasePrice
	if prod.PurchaseCurrency == "CNY" && purchaseCost > 0 {
		purchaseCost = purchaseCost / s.getCNYRate()
	}

	// 2. Shipping cost -- try logistics estimate, fallback to default $15
	shippingCost := s.estimateShipping(&prod)

	// 3. Platform fee -- use platformfee module or default 15%
	var platformFee float64
	if prod.TargetPlatformID != nil && *prod.TargetPlatformID > 0 {
		platformFee = s.calculatePlatformFee(*prod.TargetPlatformID, prod.TargetSalePrice)
	} else {
		platformFee = prod.TargetSalePrice * 0.15 // default 15%
	}

	// 4. Tariff -- use tariff module or default 5%
	tariffCost := s.calculateTariff(&prod)

	// 5. Other cost (packaging, misc) -- 2% of target price
	otherCost := prod.TargetSalePrice * 0.02

	// 6. Totals
	totalCost := purchaseCost + shippingCost + platformFee + tariffCost + otherCost
	estimatedProfit := prod.TargetSalePrice - totalCost
	profitMargin := 0.0
	if prod.TargetSalePrice > 0 {
		profitMargin = (estimatedProfit / prod.TargetSalePrice) * 100
		profitMargin = math.Round(profitMargin*100) / 100
	}

	// 7. Status classification
	status := classifyProfit(profitMargin)

	// Store the result
	summary := ProfitSummary{
		ProductID:       productID,
		PurchaseCost:    math.Round(purchaseCost*100) / 100,
		ShippingCost:    math.Round(shippingCost*100) / 100,
		PlatformFee:     math.Round(platformFee*100) / 100,
		TariffCost:      math.Round(tariffCost*100) / 100,
		OtherCost:       math.Round(otherCost*100) / 100,
		TotalCost:       math.Round(totalCost*100) / 100,
		TargetRevenue:   prod.TargetSalePrice,
		EstimatedProfit: math.Round(estimatedProfit*100) / 100,
		ProfitMargin:    profitMargin,
		Status:          status,
		Currency:        "USD",
		CalculatedBy:    calculatedBy,
	}
	s.db.Create(&summary)

	return &ProfitResult{
		ProductID:       productID,
		Title:           prod.Title,
		PurchaseCost:    summary.PurchaseCost,
		ShippingCost:    summary.ShippingCost,
		PlatformFee:     summary.PlatformFee,
		TariffCost:      summary.TariffCost,
		OtherCost:       summary.OtherCost,
		TotalCost:       summary.TotalCost,
		TargetRevenue:   prod.TargetSalePrice,
		EstimatedProfit: summary.EstimatedProfit,
		ProfitMargin:    profitMargin,
		Status:          status,
		Currency:        "USD",
	}, nil
}

// estimateShipping estimates shipping cost based on package dimensions.
// Uses a simplified volumetric weight calculation.
func (s *Service) estimateShipping(prod *candidate.CandidateProduct) float64 {
	if prod.PackageWeightKg <= 0 {
		return 15.0 // default flat rate
	}

	// Volumetric weight (DIM factor 5000 for cross-border)
	volWeight := (prod.PackageLengthCm * prod.PackageWidthCm * prod.PackageHeightCm) / 5000.0
	billableWeight := math.Max(prod.PackageWeightKg, volWeight)

	// Simplified tiered shipping rates
	if billableWeight <= 0.5 {
		return 8.0
	} else if billableWeight <= 1.0 {
		return 12.0
	} else if billableWeight <= 2.0 {
		return 18.0
	} else if billableWeight <= 5.0 {
		return 25.0
	}
	return 35.0
}

// calculatePlatformFee computes platform fees.
func (s *Service) calculatePlatformFee(platformID int64, salePrice float64) float64 {
	// Try to use platformfee service to get actual rate
	var feeRules []platformfee.PlatformFeeRule
	if err := s.db.Where("platform_id = ?", platformID).Find(&feeRules).Error; err == nil && len(feeRules) > 0 {
		totalFee := 0.0
		for _, rule := range feeRules {
			if rule.FeeType == "commission" {
				totalFee += salePrice * rule.FeeRatePct / 100.0
			} else if rule.FeeType == "fixed" {
				totalFee += rule.FixedAmount
			}
		}
		if totalFee > 0 {
			return totalFee
		}
	}
	// Default: 15% commission
	return salePrice * 0.15
}

// calculateTariff computes tariff cost.
func (s *Service) calculateTariff(prod *candidate.CandidateProduct) float64 {
	// Try to find a matching tariff rule
	if prod.HSCode != "" {
		var rule tariff.TariffRule
		dest := prod.DestinationCountry
		if dest == "" {
			dest = "US"
		}
		if err := s.db.Where("country_code = ? AND hs_code = ? AND status = ?", dest, prod.HSCode, "active").
			Or("country_code = ? AND hs_code_prefix = ? AND status = ?", dest, prod.HSCode[:minInt(4, len(prod.HSCode))], "active").
			First(&rule).Error; err == nil {
			totalRate := rule.DutyRatePct + rule.VatRatePct + rule.OtherTaxRatePct
			return prod.PurchasePrice / s.getCNYRate() * totalRate / 100.0
		}
	}
	// Default: 5% duty on purchase cost (converted to USD)
	return (prod.PurchasePrice / s.getCNYRate()) * 0.05
}

// ListSummaries returns paginated profit summaries with optional status and date range filters.
func (s *Service) ListSummaries(page, size int, status string, startDate, endDate string) ([]ProfitSummary, int64, error) {
	var items []ProfitSummary
	var total int64
	q := s.db.Model(&ProfitSummary{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			q = q.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByProductID returns the latest profit summary for a product.
func (s *Service) GetByProductID(productID int64) (*ProfitSummary, error) {
	var ps ProfitSummary
	if err := s.db.Where("product_id = ?", productID).Order("id DESC").First(&ps).Error; err != nil {
		return nil, err
	}
	return &ps, nil
}

func classifyProfit(margin float64) string {
	if margin >= 15 {
		return "profitable"
	} else if margin >= 0 {
		return "marginal"
	}
	return "unprofitable"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Ensure imports compile
var _ = fmt.Sprintf
var _ = supplier.Supplier{}
