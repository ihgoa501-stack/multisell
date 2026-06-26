package logistics

import (
	"context"
	"sort"
)

// ---------- Interfaces ----------

// RateQuoter defines the interface for obtaining shipping rate quotes.
type RateQuoter interface {
	// Quote returns shipping rate quotes for the given request.
	Quote(ctx context.Context, req *RateQuoteRequest) (*RateQuoteResponse, error)
}

// CarrierAdapter defines the interface for third-party carrier API integration.
// Each carrier provider implements this interface to connect to its API.
type CarrierAdapter interface {
	// Name returns a unique carrier code (e.g. "yanwen", "yuntu").
	Name() string

	// Quote returns rates from this carrier for the given request.
	Quote(ctx context.Context, req *RateQuoteRequest) (*RateQuoteResponse, error)

	// ValidateCredentials checks whether the configured API credentials are valid.
	ValidateCredentials(ctx context.Context) error
}

// ---------- LocalRateEngine ----------

// LocalRateEngine implements RateQuoter using the in-process RateEngine with
// static rate tables. It provides a RateQuoter-compatible wrapper around the
// core calculation engine.
type LocalRateEngine struct {
	svc *Service
}

// NewLocalRateEngine creates a LocalRateEngine backed by the given Service.
func NewLocalRateEngine(svc *Service) *LocalRateEngine {
	return &LocalRateEngine{svc: svc}
}

// Quote implements RateQuoter using the local rate engine.
func (e *LocalRateEngine) Quote(ctx context.Context, req *RateQuoteRequest) (*RateQuoteResponse, error) {
	return e.svc.GetQuotes(req)
}

// ---------- CarrierRegistry ----------

// CarrierRegistry manages a set of configured carrier adapters.
type CarrierRegistry struct {
	adapters []CarrierAdapter
}

// NewCarrierRegistry creates a carrier registry.
func NewCarrierRegistry() *CarrierRegistry {
	return &CarrierRegistry{}
}

// Register adds a carrier adapter to the registry.
func (r *CarrierRegistry) Register(a CarrierAdapter) {
	r.adapters = append(r.adapters, a)
}

// List returns all registered carriers.
func (r *CarrierRegistry) List() []CarrierAdapter {
	sorted := make([]CarrierAdapter, len(r.adapters))
	copy(sorted, r.adapters)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name() < sorted[j].Name()
	})
	return sorted
}

// Get returns a carrier by name.
func (r *CarrierRegistry) Get(name string) CarrierAdapter {
	for _, a := range r.adapters {
		if a.Name() == name {
			return a
		}
	}
	return nil
}

// ---------- MockCarrier ----------

// MockCarrier is a test/development carrier that returns fixed mock rates.
type MockCarrier struct{}

// Name returns the carrier code.
func (m *MockCarrier) Name() string { return "mock" }

// Quote returns fixed mock rates based on weight and destination.
func (m *MockCarrier) Quote(ctx context.Context, req *RateQuoteRequest) (*RateQuoteResponse, error) {
	quotes := []RateQuote{
		{
			Carrier:     "MockCarrier",
			ServiceName: "Mock Standard",
			TotalCost:   25.00 + req.WeightKG*30.00,
			Currency:    "CNY",
			EstDaysMin:  10,
			EstDaysMax:  20,
			Confidence:  "high",
		},
		{
			Carrier:     "MockCarrier",
			ServiceName: "Mock Express",
			TotalCost:   50.00 + req.WeightKG*50.00,
			Currency:    "CNY",
			EstDaysMin:  5,
			EstDaysMax:  10,
			Confidence:  "high",
		},
	}

	// Add destination-specific premium.
	switch req.DestinationCountry {
	case "RU":
		quotes[0].TotalCost *= 1.0
		quotes[1].TotalCost *= 1.0
	case "KZ":
		quotes[0].TotalCost *= 0.9
		quotes[1].TotalCost *= 0.9
	case "US":
		quotes[0].TotalCost *= 1.5
		quotes[1].TotalCost *= 1.5
	}

	return &RateQuoteResponse{Quotes: quotes}, nil
}

// ValidateCredentials always succeeds for the mock carrier.
func (m *MockCarrier) ValidateCredentials(ctx context.Context) error { return nil }
