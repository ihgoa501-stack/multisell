package logistics

import (
	"errors"
	"sort"
)

// EventPublisher defines the interface for publishing events.
type EventPublisher interface {
	Publish(topic string, payload map[string]interface{}) error
}

// Service is the A10 Logistics Agent's core service.
// It wraps RateEngine and provides high-level APIs for other agents (e.g. A8).
type Service struct {
	engine *RateEngine
	events EventPublisher
}

// NewService creates a logistics service over the given rate table entries.
func NewService(tables []RateTableEntry) *Service {
	return &Service{engine: NewRateEngine(tables)}
}

// NewServiceWithEvents creates a logistics service with event publishing.
func NewServiceWithEvents(tables []RateTableEntry, events EventPublisher) *Service {
	return &Service{engine: NewRateEngine(tables), events: events}
}

// GetQuote returns all shipping quotes for the given cargo, destination, and cargo type.
func (s *Service) GetQuote(cargo Cargo, destination, cargoType string) (*QuoteResponse, error) {
	if s.engine == nil {
		return nil, errors.New("logistics service: no rate engine configured")
	}
	resp, err := s.engine.CalculateRate(cargo, destination, cargoType)
	if err != nil {
		return nil, err
	}
	if len(resp.Results) == 0 {
		return nil, errors.New("logistics service: no matching shipping rates for the given parameters")
	}
	return resp, nil
}

// GetCheapestQuote returns the cheapest matching shipping quote.
func (s *Service) GetCheapestQuote(cargo Cargo, destination, cargoType string) (*QuoteResult, error) {
	resp, err := s.GetQuote(cargo, destination, cargoType)
	if err != nil {
		return nil, err
	}

	sorted := make([]QuoteResult, len(resp.Results))
	copy(sorted, resp.Results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TotalShippingFee < sorted[j].TotalShippingFee
	})

	return &sorted[0], nil
}

// GetQuotes returns on-demand freight quotes for the given request.
// This is a higher-level method suitable for direct API or agent calls.
func (s *Service) GetQuotes(req *RateQuoteRequest) (*RateQuoteResponse, error) {
	cargo := Cargo{
		ActualWeightKg: req.WeightKG,
		LengthCm:       req.LengthCM,
		WidthCm:        req.WidthCM,
		HeightCm:       req.HeightCM,
	}

	origin := req.OriginCountry
	if origin == "" {
		origin = "CN"
	}

	cargoType := req.CargoType
	if cargoType == "" {
		cargoType = "normal"
	}

	resp, err := s.GetQuote(cargo, req.DestinationCountry, cargoType)
	if err != nil {
		return nil, err
	}

	rateResp := resp.ToRateQuoteResponse()

	// Publish quote event if event publisher is configured.
	if s.events != nil {
		_ = s.events.Publish("logistics.quote", map[string]interface{}{
			"origin":             origin,
			"destination":        req.DestinationCountry,
			"weight_kg":          req.WeightKG,
			"quote_count":        len(rateResp.Quotes),
			"declared_value":     req.DeclaredValue,
		})
	}

	return rateResp, nil
}

// ListCarriers returns information about configured carriers.
func (s *Service) ListCarriers() []CarrierInfo {
	if s.engine == nil {
		return nil
	}

	seen := make(map[string]*CarrierInfo)
	for _, entry := range s.engine.tables {
		key := entry.ProviderName + "|" + entry.DestinationCountry
		if _, ok := seen[key]; !ok {
			seen[key] = &CarrierInfo{
				Code:        entry.ProviderName,
				Name:        entry.ProviderName,
				CountryCode: entry.DestinationCountry,
				Channels:    0,
			}
		}
		seen[key].Channels++
	}

	result := make([]CarrierInfo, 0, len(seen))
	for _, info := range seen {
		result = append(result, *info)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
