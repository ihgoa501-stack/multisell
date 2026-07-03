package shipping

import (
	"context"
)

// CarrierAPIMode represents execution mode for carrier API calls.
type CarrierAPIMode string

const (
	CarrierModeDryRun     CarrierAPIMode = "dry_run"
	CarrierModeSandbox    CarrierAPIMode = "sandbox"
	CarrierModeProduction CarrierAPIMode = "production"
)

// CarrierQuoteRequest is the request for a carrier rate quote.
type CarrierQuoteRequest struct {
	OriginCountry      string  `json:"origin_country"`
	DestinationCountry string  `json:"destination_country"`
	WeightKg           float64 `json:"weight_kg"`
	LengthCm           float64 `json:"length_cm"`
	WidthCm            float64 `json:"width_cm"`
	HeightCm           float64 `json:"height_cm"`
	CargoType          string  `json:"cargo_type"`
}

// CarrierQuoteResponse is the response from a carrier rate quote.
type CarrierQuoteResponse struct {
	TotalFee         float64 `json:"total_fee"`
	Currency         string  `json:"currency"`
	EstimatedDaysMin int     `json:"estimated_days_min"`
	EstimatedDaysMax int     `json:"estimated_days_max"`
	ServiceName      string  `json:"service_name"`
}

// MockCarrierAdapter is a no-op carrier adapter for development and testing.
type MockCarrierAdapter struct{}

func (m *MockCarrierAdapter) GetQuote(_ context.Context, mode CarrierAPIMode, req *CarrierQuoteRequest) (*CarrierQuoteResponse, error) {
	return &CarrierQuoteResponse{
		TotalFee: 15.0, Currency: "CNY",
		EstimatedDaysMin: 7, EstimatedDaysMax: 15,
		ServiceName: "Mock Standard",
	}, nil
}
