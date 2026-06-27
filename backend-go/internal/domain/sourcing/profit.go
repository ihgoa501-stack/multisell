// Package sourcing provides the sourcing intelligence domain module.
// It includes profit calculation, AI-powered sourcing recommendations,
// and the REST API for the sourcing panel.
package sourcing

import (
	"math"
)

// ---------- Profit Calculation ----------

// ProfitInput is the input for profit estimation.
type ProfitInput struct {
	SourcePriceCNY float64 `json:"source_price_cny"` // 1688 price in CNY
	WeightKg       float64 `json:"weight_kg"`         // estimated package weight
	Destination    string  `json:"destination"`       // target marketplace (US/EU/JP/RU)
	MarkupPct      float64 `json:"markup_pct"`        // desired markup percentage (e.g. 250.0 = 2.5x)
}

// ProfitBreakdown is the detailed profit calculation result.
type ProfitBreakdown struct {
	SourcePriceCNY  float64 `json:"source_price_cny"`
	WeightKg        float64 `json:"weight_kg"`
	Destination     string  `json:"destination"`
	MarkupPct       float64 `json:"markup_pct"`

	TargetPriceCNY  float64 `json:"target_price_cny"`   // selling price in CNY
	ShippingCostCNY float64 `json:"shipping_cost_cny"`  // estimated shipping cost
	PlatformFeeCNY  float64 `json:"platform_fee_cny"`   // estimated platform commission
	PlatformFeePct  float64 `json:"platform_fee_pct"`   // platform fee rate applied
	TotalCostCNY    float64 `json:"total_cost_cny"`     // source + shipping + fees
	ProfitCNY       float64 `json:"profit_cny"`         // target - total cost
	MarginPct       float64 `json:"margin_pct"`         // profit / target * 100
	IsViable        bool    `json:"is_viable"`          // margin >= 15%
}

// shippingEstimates maps destination → cost per kg (CNY).
var shippingEstimates = map[string]float64{
	"US": 45.0,
	"EU": 50.0,
	"JP": 35.0,
	"RU": 55.0,
	"BR": 70.0,
	"AU": 50.0,
}

// platformFeeEstimates maps destination → estimated platform fee rate (percent).
var platformFeeEstimates = map[string]float64{
	"US": 15.0,
	"EU": 15.0,
	"JP": 10.0,
	"RU": 12.0,
	"BR": 18.0,
	"AU": 15.0,
}

const (
	defaultShippingCost = 50.0
	defaultPlatformFee  = 15.0
	minViableMargin     = 15.0
)

// CalculateProfit computes a detailed profit estimate for sourcing a product.
func CalculateProfit(in *ProfitInput) *ProfitBreakdown {
	shipCost := defaultShippingCost
	if c, ok := shippingEstimates[in.Destination]; ok {
		shipCost = c
	}

	platFeePct := defaultPlatformFee
	if f, ok := platformFeeEstimates[in.Destination]; ok {
		platFeePct = f
	}

	markup := in.MarkupPct
	if markup <= 0 {
		markup = 250.0
	}
	targetPrice := in.SourcePriceCNY * markup / 100.0

	shippingCost := shipCost * in.WeightKg
	platformFee := targetPrice * platFeePct / 100.0
	totalCost := in.SourcePriceCNY + shippingCost + platformFee
	profit := targetPrice - totalCost

	margin := 0.0
	if targetPrice > 0 {
		margin = profit / targetPrice * 100.0
	}

	return &ProfitBreakdown{
		SourcePriceCNY:  round2(in.SourcePriceCNY),
		WeightKg:        round2(in.WeightKg),
		Destination:     in.Destination,
		MarkupPct:       markup,
		TargetPriceCNY:  round2(targetPrice),
		ShippingCostCNY: round2(shippingCost),
		PlatformFeeCNY:  round2(platformFee),
		PlatformFeePct:  round2(platFeePct),
		TotalCostCNY:    round2(totalCost),
		ProfitCNY:       round2(profit),
		MarginPct:       round2(margin),
		IsViable:        margin >= minViableMargin,
	}
}

// round2 rounds to 2 decimal places.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
