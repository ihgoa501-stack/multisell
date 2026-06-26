package logistics

import (
	"context"
	"math"
)

// CarrierYuntuAdapter is a stub adapter for Yuntu (云途) carrier API.
// Phase 1: returns realistic mock rates based on weight + destination.
// Phase 2+: replace with real API integration.
type CarrierYuntuAdapter struct{}

// Name returns the carrier code.
func (a *CarrierYuntuAdapter) Name() string { return "yuntu" }

// Quote returns simulated Yuntu rates.
// Yuntu pricing is typically per-kg with volume-based tiers.
func (a *CarrierYuntuAdapter) Quote(ctx context.Context, req *RateQuoteRequest) (*RateQuoteResponse, error) {
	w := math.Max(req.WeightKG, 0.1)
	dest := req.DestinationCountry

	var economyRate, standardRate float64
	var econDaysMin, econDaysMax, stdDaysMin, stdDaysMax int

	switch dest {
	case "RU":
		economyRate = 48.0
		standardRate = 68.0
		econDaysMin, econDaysMax = 12, 20
		stdDaysMin, stdDaysMax = 7, 12
	case "KZ":
		economyRate = 45.0
		standardRate = 60.0
		econDaysMin, econDaysMax = 10, 18
		stdDaysMin, stdDaysMax = 6, 11
	case "US":
		economyRate = 58.0
		standardRate = 85.0
		econDaysMin, econDaysMax = 10, 18
		stdDaysMin, stdDaysMax = 5, 10
	default:
		economyRate = 52.0
		standardRate = 75.0
		econDaysMin, econDaysMax = 11, 19
		stdDaysMin, stdDaysMax = 6, 12
	}

	// Volume tier discount for heavy packages (>10kg)
	factor := 1.0
	if w > 10 {
		factor = 0.9 // 10% bulk discount
	} else if w > 5 {
		factor = 0.95 // 5% discount for 5-10kg
	}

	economyTotal := w * economyRate * factor
	standardTotal := w * standardRate * factor

	quotes := []RateQuote{
		{
			Carrier: "Yuntu", ServiceName: "云途经济线",
			TotalCost: math.Round(economyTotal*100) / 100,
			Currency:  "CNY",
			EstDaysMin: econDaysMin, EstDaysMax: econDaysMax,
			Confidence: "medium",
		},
		{
			Carrier: "Yuntu", ServiceName: "云途标准线",
			TotalCost: math.Round(standardTotal*100) / 100,
			Currency:  "CNY",
			EstDaysMin: stdDaysMin, EstDaysMax: stdDaysMax,
			Confidence: "high",
		},
	}

	return &RateQuoteResponse{Quotes: quotes}, nil
}

// ValidateCredentials always succeeds for the stub adapter.
func (a *CarrierYuntuAdapter) ValidateCredentials(ctx context.Context) error { return nil }
