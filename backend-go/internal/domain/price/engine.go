package price

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// PricingEngine generates pricing recommendations based on competitor prices
// and strategy configuration. It is stateless and safe for concurrent use.
type PricingEngine struct{}

// NewPricingEngine creates a new PricingEngine.
func NewPricingEngine() *PricingEngine {
	return &PricingEngine{}
}

// EngineInput contains all data the engine needs to generate a recommendation.
type EngineInput struct {
	SkuID            int64
	CurrentPrice     decimal.Decimal
	CompetitorPrices []CompetitorPrice
	StrategyType     string
	Parameters       StrategyParameters
	Cost             decimal.Decimal
	PlatformFeeRate  float64
}

// Generate produces a pricing recommendation from the given input.
func (e *PricingEngine) Generate(input EngineInput) *PricingRecommendation {
	params := input.Parameters
	if params.BuyBoxDiscount == 0 && params.MinProfitMargin == 0 &&
		params.MinPrice == 0 && params.MaxPrice == 0 {
		params = DefaultStrategyParams(input.StrategyType)
	}

	// Deduplicate to latest entry per competitor (input must be sorted by captured_at DESC).
	latest := deduplicateCompetitorPrices(input.CompetitorPrices)
	count := len(latest)

	var target decimal.Decimal
	var reason string

	if count == 0 {
		// No competitor data — recommend keeping current price with high risk.
		target = input.CurrentPrice
		reason = "no competitor prices available for this SKU"

		return &PricingRecommendation{
			SkuID:            input.SkuID,
			CurrentPrice:     input.CurrentPrice,
			RecommendedPrice: target,
			StrategyUsed:     input.StrategyType,
			Reason:           reason,
			RiskLevel:        "high",
			CompetitorCount:  0,
		}
	} else {
		lowest := getLowestPrice(latest)

		switch input.StrategyType {
		case StrategyBuyBoxFirst:
			discount := decimal.NewFromFloat(params.BuyBoxDiscount)
			target = lowest.Mul(decimal.NewFromFloat(1.0).Sub(discount))
			if target.LessThan(decimal.Zero) {
				target = decimal.Zero
			}
			reason = fmt.Sprintf("buy_box_first: %.1f%% below lowest competitor %.2f",
				params.BuyBoxDiscount*100, lowest.InexactFloat64())

		case StrategyProfitPriority:
			if input.Cost.IsPositive() {
				one := decimal.NewFromFloat(1.0)
				minMargin := decimal.NewFromFloat(params.MinProfitMargin)
				feeRate := decimal.NewFromFloat(input.PlatformFeeRate)
				minPrice := input.Cost.Mul(one.Add(minMargin)).Div(one.Sub(feeRate))
				target = decimal.Max(input.CurrentPrice, minPrice)
				reason = fmt.Sprintf("profit_priority: maintain at least %.0f%% margin (cost=%.2f, fee=%.1f%%)",
					params.MinProfitMargin*100, input.Cost.InexactFloat64(), input.PlatformFeeRate*100)
			} else {
				target = input.CurrentPrice
				reason = "profit_priority: no cost data, keeping current price"
			}

		case StrategyMatch:
			target = lowest
			reason = fmt.Sprintf("match: match lowest competitor price %.2f", lowest.InexactFloat64())

		default:
			target = input.CurrentPrice
			reason = fmt.Sprintf("unknown strategy type: %s", input.StrategyType)
		}
	}

	// Apply floor / ceiling bounds.
	if params.MinPrice > 0 {
		minBound := decimal.NewFromFloat(params.MinPrice)
		if target.LessThan(minBound) {
			target = minBound
			reason += " (adjusted to minimum price)"
		}
	}
	if params.MaxPrice > 0 {
		maxBound := decimal.NewFromFloat(params.MaxPrice)
		if target.GreaterThan(maxBound) {
			target = maxBound
			reason += " (adjusted to maximum price)"
		}
	}

	riskLevel := calculateRiskLevel(input.CurrentPrice, target)

	return &PricingRecommendation{
		SkuID:            input.SkuID,
		CurrentPrice:     input.CurrentPrice,
		RecommendedPrice: target,
		StrategyUsed:     input.StrategyType,
		Reason:           reason,
		RiskLevel:        riskLevel,
		CompetitorCount:  count,
	}
}

// deduplicateCompetitorPrices keeps only the latest entry per competitor name.
// Expects input sorted by captured_at DESC.
func deduplicateCompetitorPrices(prices []CompetitorPrice) []CompetitorPrice {
	seen := make(map[string]bool)
	var result []CompetitorPrice
	for _, p := range prices {
		if !seen[p.CompetitorName] {
			seen[p.CompetitorName] = true
			result = append(result, p)
		}
	}
	return result
}

// getLowestPrice finds the minimum price among competitor prices.
func getLowestPrice(prices []CompetitorPrice) decimal.Decimal {
	if len(prices) == 0 {
		return decimal.Zero
	}
	lowest := prices[0].Price
	for _, p := range prices[1:] {
		if p.Price.LessThan(lowest) {
			lowest = p.Price
		}
	}
	return lowest
}

// calculateRiskLevel returns the risk level based on price difference.
func calculateRiskLevel(current, recommended decimal.Decimal) string {
	if current.IsZero() {
		if recommended.IsZero() {
			return "medium"
		}
		return "high"
	}
	diff := recommended.Sub(current).Abs().Div(current)
	if diff.LessThan(decimal.NewFromFloat(0.05)) {
		return "low"
	}
	if diff.LessThan(decimal.NewFromFloat(0.15)) {
		return "medium"
	}
	return "high"
}

// DefaultStrategyParams returns sensible defaults for the given strategy type.
func DefaultStrategyParams(strategyType string) StrategyParameters {
	switch strategyType {
	case StrategyBuyBoxFirst:
		return StrategyParameters{BuyBoxDiscount: 0.02}
	case StrategyProfitPriority:
		return StrategyParameters{MinProfitMargin: 0.15}
	default:
		return StrategyParameters{}
	}
}
