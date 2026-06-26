package logistics

import (
	"context"
	"math"
)

// CarrierYanwenAdapter is a stub adapter for Yanwen (燕文) carrier API.
// Phase 1: returns realistic mock rates based on weight + destination.
// Phase 2+: replace with real API integration.
type CarrierYanwenAdapter struct{}

// Name returns the carrier code.
func (a *CarrierYanwenAdapter) Name() string { return "yanwen" }

// Quote returns simulated Yanwen rates.
// Real pricing is based on first-weight + additional-weight model.
func (a *CarrierYanwenAdapter) Quote(ctx context.Context, req *RateQuoteRequest) (*RateQuoteResponse, error) {
	w := math.Max(req.WeightKG, 0.1)
	dest := req.DestinationCountry

	// Yanwen pricing zones
	var economyBase, economyPerKg, econDaysMin, econDaysMax float64
	var standardBase, standardPerKg, stdDaysMin, stdDaysMax float64

	switch dest {
	case "RU":
		economyBase, economyPerKg = 55.0, 25.0
		econDaysMin, econDaysMax = 15, 25
		standardBase, standardPerKg = 80.0, 35.0
		stdDaysMin, stdDaysMax = 10, 18
	case "KZ":
		economyBase, economyPerKg = 50.0, 22.0
		econDaysMin, econDaysMax = 12, 20
		standardBase, standardPerKg = 70.0, 30.0
		stdDaysMin, stdDaysMax = 8, 15
	case "US":
		economyBase, economyPerKg = 70.0, 30.0
		econDaysMin, econDaysMax = 12, 20
		standardBase, standardPerKg = 100.0, 45.0
		stdDaysMin, stdDaysMax = 7, 14
	default:
		economyBase, economyPerKg = 60.0, 28.0
		econDaysMin, econDaysMax = 14, 22
		standardBase, standardPerKg = 90.0, 38.0
		stdDaysMin, stdDaysMax = 8, 16
	}

	// First kg pricing model: first kg price + additional per-kg
	economyTotal := economyBase + math.Max(0, w-1)*economyPerKg
	standardTotal := standardBase + math.Max(0, w-1)*standardPerKg

	quotes := []RateQuote{
		{
			Carrier: "Yanwen", ServiceName: "燕文经济线",
			TotalCost: math.Round(economyTotal*100) / 100,
			Currency:  "CNY",
			EstDaysMin: int(econDaysMin), EstDaysMax: int(econDaysMax),
			Confidence: "medium",
		},
		{
			Carrier: "Yanwen", ServiceName: "燕文标准线",
			TotalCost: math.Round(standardTotal*100) / 100,
			Currency:  "CNY",
			EstDaysMin: int(stdDaysMin), EstDaysMax: int(stdDaysMax),
			Confidence: "medium",
		},
	}

	return &RateQuoteResponse{Quotes: quotes}, nil
}

// ValidateCredentials always succeeds for the stub adapter.
func (a *CarrierYanwenAdapter) ValidateCredentials(ctx context.Context) error { return nil }
