package logistics

import (
	"errors"
	"sort"
	"sync"
)

// Service is the A10 Logistics Agent's core service.
// It wraps RateEngine for quote calculations and maintains in-memory
// fulfillment statistics (carrier_performance, category_performance)
// fed by the supplychain.flywheel data-flywheel events.
type Service struct {
	engine       *RateEngine
	mu           sync.RWMutex
	carrierStats map[string]*CarrierPerformance   // key: "channel|provider"
	categoryStats map[string]*CategoryPerformance // key: "category|channel"
}

// NewService creates a logistics service over the given rate table entries.
func NewService(tables []RateTableEntry) *Service {
	return &Service{
		engine:        NewRateEngine(tables),
		carrierStats:  make(map[string]*CarrierPerformance),
		categoryStats: make(map[string]*CategoryPerformance),
	}
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
