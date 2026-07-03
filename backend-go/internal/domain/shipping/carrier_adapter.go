package shipping

import (
	"context"
	"time"
)

// CarrierAPIMode represents execution mode for carrier API calls.
type CarrierAPIMode string

const (
	CarrierModeDryRun     CarrierAPIMode = "dry_run"
	CarrierModeSandbox    CarrierAPIMode = "sandbox"
	CarrierModeProduction CarrierAPIMode = "production"
)

// The quote request/response structs.
type CarrierQuoteRequest struct {
	OriginCountry      string  `json:"origin_country"`
	DestinationCountry string  `json:"destination_country"`
	WeightKg           float64 `json:"weight_kg"`
	LengthCm           float64 `json:"length_cm"`
	WidthCm            float64 `json:"width_cm"`
	HeightCm           float64 `json:"height_cm"`
	CargoType          string  `json:"cargo_type"`
}

type CarrierQuoteResponse struct {
	TotalFee         float64 `json:"total_fee"`
	Currency         string  `json:"currency"`
	EstimatedDaysMin int     `json:"estimated_days_min"`
	EstimatedDaysMax int     `json:"estimated_days_max"`
	ServiceName      string  `json:"service_name"`
}

type CarrierShipmentRequest struct {
	OrderID               int64   `json:"order_id"`
	TrackingNumber        string  `json:"tracking_number"`
	DestinationCountry    string  `json:"destination_country"`
	DestinationPostalCode string  `json:"destination_postal_code"`
	WeightKg              float64 `json:"weight_kg"`
	LengthCm              float64 `json:"length_cm"`
	WidthCm               float64 `json:"width_cm"`
	HeightCm              float64 `json:"height_cm"`
	Reference             string  `json:"reference"`
}

type CarrierShipmentResponse struct {
	ShipmentID     string  `json:"shipment_id"`
	LabelURL       string  `json:"label_url"`
	TrackingNumber string  `json:"tracking_number"`
	EstimatedFee   float64 `json:"estimated_fee"`
	Currency       string  `json:"currency"`
}

type CarrierTrackingInfo struct {
	Status            string          `json:"status"`
	EstimatedDelivery *time.Time      `json:"estimated_delivery,omitempty"`
	Events            []TrackingEvent `json:"events,omitempty"`
}

// CarrierAdapter is the interface for external carrier API integrations.
// In dry_run mode: validate only, no side effects.
// In sandbox mode: execute against test endpoints.
// In production mode: real API calls, requires approval_id for mutations.
// approvalID validation is centralized in the service layer — adapters
// must NOT re-implement it.
type CarrierAdapter interface {
	Name() string
	GetQuote(ctx context.Context, mode CarrierAPIMode, req *CarrierQuoteRequest) (*CarrierQuoteResponse, error)
	CreateShipment(ctx context.Context, mode CarrierAPIMode, req *CarrierShipmentRequest, approvalID int64) (*CarrierShipmentResponse, error)
	TrackShipment(ctx context.Context, mode CarrierAPIMode, trackingNumber string) (*CarrierTrackingInfo, error)
	CancelShipment(ctx context.Context, mode CarrierAPIMode, shipmentID string, approvalID int64) error
}

// MockCarrierAdapter is a no-op adapter for development and testing.
type MockCarrierAdapter struct{}

func (m *MockCarrierAdapter) Name() string { return "mock_carrier" }

func (m *MockCarrierAdapter) GetQuote(ctx context.Context, mode CarrierAPIMode, req *CarrierQuoteRequest) (*CarrierQuoteResponse, error) {
	return &CarrierQuoteResponse{
		TotalFee: 15.0, Currency: "CNY",
		EstimatedDaysMin: 7, EstimatedDaysMax: 15,
		ServiceName: "Mock Standard",
	}, nil
}

func (m *MockCarrierAdapter) CreateShipment(ctx context.Context, mode CarrierAPIMode, req *CarrierShipmentRequest, approvalID int64) (*CarrierShipmentResponse, error) {
	// validation centralized in service layer — adapters must NOT check approvalID
	return &CarrierShipmentResponse{
		ShipmentID:     "mock-" + req.TrackingNumber,
		LabelURL:       "https://mock.carrier/label/" + req.TrackingNumber,
		TrackingNumber: req.TrackingNumber,
		EstimatedFee:   20.0, Currency: "CNY",
	}, nil
}

func (m *MockCarrierAdapter) TrackShipment(ctx context.Context, mode CarrierAPIMode, trackingNumber string) (*CarrierTrackingInfo, error) {
	now := time.Now()
	return &CarrierTrackingInfo{
		Status: "in_transit",
		EstimatedDelivery: &now,
		Events: []TrackingEvent{
			{Timestamp: now.Format(time.RFC3339), Status: "picked_up", Message: "Package picked up"},
		},
	}, nil
}

func (m *MockCarrierAdapter) CancelShipment(ctx context.Context, mode CarrierAPIMode, shipmentID string, approvalID int64) error {
	// validation centralized in service layer — adapters must NOT check approvalID
	return nil
}
