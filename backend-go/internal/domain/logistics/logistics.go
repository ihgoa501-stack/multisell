package logistics

import (
	"errors"
	"sort"
)

// Service is the A10 Logistics Agent's core service.
// It wraps RateEngine and provides high-level APIs for other agents (e.g. A8).
type Service struct {
	engine *RateEngine
}

// NewService creates a logistics service over the given rate table entries.
func NewService(tables []RateTableEntry) *Service {
	return &Service{engine: NewRateEngine(tables)}
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
// Results are sorted by TotalShippingFee ascending; the first entry is returned.
func (s *Service) GetCheapestQuote(cargo Cargo, destination, cargoType string) (*QuoteResult, error) {
	resp, err := s.GetQuote(cargo, destination, cargoType)
	if err != nil {
		return nil, err
	}

	// Sort results by total fee ascending so the cheapest is first.
	sorted := make([]QuoteResult, len(resp.Results))
	copy(sorted, resp.Results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TotalShippingFee < sorted[j].TotalShippingFee
	})

	return &sorted[0], nil
}
