package logistics

import (
	"fmt"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Default weights
// ---------------------------------------------------------------------------

// DefaultRouterWeights are the default scoring weights for route recommendations.
// Cost is weighted highest (0.5), followed by speed (0.3), then reliability (0.2).
var DefaultRouterWeights = RouterWeights{
	CostWeight:       0.5,
	SpeedWeight:      0.3,
	ReliabilityWeight: 0.2,
}

// ---------------------------------------------------------------------------
// RouterWeights — configurable scoring strategy
// ---------------------------------------------------------------------------

// RouterWeights configures the trade-off between cost, speed, and reliability
// when scoring and ranking shipping channel recommendations.
type RouterWeights struct {
	CostWeight       float64 `json:"cost_weight"`
	SpeedWeight      float64 `json:"speed_weight"`
	ReliabilityWeight float64 `json:"reliability_weight"`
}

// Validate returns an error if any weight is negative or all are zero.
func (w RouterWeights) Validate() error {
	if w.CostWeight < 0 || w.SpeedWeight < 0 || w.ReliabilityWeight < 0 {
		return fmt.Errorf("router weights: negative weights not allowed (cost=%.2f speed=%.2f rel=%.2f)",
			w.CostWeight, w.SpeedWeight, w.ReliabilityWeight)
	}
	if w.CostWeight == 0 && w.SpeedWeight == 0 && w.ReliabilityWeight == 0 {
		return fmt.Errorf("router weights: at least one weight must be positive")
	}
	return nil
}

// ---------------------------------------------------------------------------
// RouteRequest
// ---------------------------------------------------------------------------

// RouteRequest represents a shipping route recommendation query.
type RouteRequest struct {
	WeightKg    float64 `json:"weight_kg"`
	LengthCm    float64 `json:"length_cm,omitempty"`
	WidthCm     float64 `json:"width_cm,omitempty"`
	HeightCm    float64 `json:"height_cm,omitempty"`
	Destination string  `json:"destination"`
	CargoType   string  `json:"cargo_type,omitempty"`
	MaxDays     int     `json:"max_days,omitempty"`     // optional constraint: max delivery days (0 = no constraint)
	MaxCost     float64 `json:"max_cost,omitempty"`     // optional constraint: max total cost (0 = no constraint)
}

// ---------------------------------------------------------------------------
// RouteRecommendation
// ---------------------------------------------------------------------------

// RouteRecommendation is a single recommended shipping channel with scoring.
type RouteRecommendation struct {
	ChannelName          string  `json:"channel_name"`
	ProviderName         string  `json:"provider_name"`
	EstimatedCost        float64 `json:"estimated_cost"`
	EstimatedDaysMin     int     `json:"estimated_days_min"`
	EstimatedDaysMax     int     `json:"estimated_days_max"`
	ReliabilityScore     float64 `json:"reliability_score"`
	CompositeScore       float64 `json:"composite_score"`
	RecommendationReason string  `json:"recommendation_reason"`
}

// ---------------------------------------------------------------------------
// Router — shipment channel recommender
// ---------------------------------------------------------------------------

// Router recommends and ranks shipping channels for a given RouteRequest.
// It wraps the logistics Service (RateEngine + carrier performance stats)
// to produce scored, sorted RouteRecommendations.
type Router struct {
	svc     *Service
	weights RouterWeights
}

// NewRouter creates a Router with the given Service and optional configuration.
//
// Example:
//
//	r := NewRouter(svc)
//	r := NewRouter(svc, WithWeights(RouterWeights{CostWeight: 0.6, SpeedWeight: 0.4}))
func NewRouter(svc *Service, opts ...RouterOption) *Router {
	r := &Router{
		svc:     svc,
		weights: DefaultRouterWeights,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RouterOption configures the Router.
type RouterOption func(*Router)

// WithWeights sets custom scoring weights. A zero-value RouterWeights falls
// back to DefaultRouterWeights.
func WithWeights(w RouterWeights) RouterOption {
	return func(r *Router) {
		if err := w.Validate(); err != nil {
			// Invalid weights silently fall back to defaults.
			return
		}
		r.weights = w
	}
}

// GetRecommendations returns all matching route recommendations sorted by
// composite score descending.  Returns nil if no channels match the request
// or constraints.
//
// Scoring uses three normalized dimensions:
//   - Cost score (weighted by RouterWeights.CostWeight):
//     lower total cost = higher score
//   - Speed score (weighted by RouterWeights.SpeedWeight):
//     faster delivery = higher score
//   - Reliability score (weighted by RouterWeights.ReliabilityWeight):
//     derived from CarrierPerformance.LossRate when available;
//     0.5 (neutral) when no performance data exists.
func (r *Router) GetRecommendations(req RouteRequest) []RouteRecommendation {
	cargo := Cargo{
		ActualWeightKg: req.WeightKg,
		LengthCm:       req.LengthCm,
		WidthCm:        req.WidthCm,
		HeightCm:       req.HeightCm,
	}

	cargoType := req.CargoType
	if cargoType == "" {
		cargoType = "normal"
	}

	resp, err := r.svc.GetQuote(cargo, req.Destination, cargoType)
	if err != nil || len(resp.Results) == 0 {
		return nil
	}

	// Apply MaxDays and MaxCost constraints.
	candidates := filterByConstraints(resp.Results, req.MaxDays, req.MaxCost)
	if len(candidates) == 0 {
		return nil
	}

	// Build recommendation objects with reliability scores.
	recs := make([]RouteRecommendation, len(candidates))
	for i, q := range candidates {
		recs[i] = RouteRecommendation{
			ChannelName:      q.ChannelName,
			ProviderName:     q.ProviderName,
			EstimatedCost:    q.TotalShippingFee,
			EstimatedDaysMin: q.EstimatedDeliveryMin,
			EstimatedDaysMax: q.EstimatedDeliveryMax,
			ReliabilityScore: r.reliabilityScore(q.ChannelName, q.ProviderName),
		}
	}

	// Compute normalized composite scores.
	scoreRecommendations(recs, r.weights)

	// Sort by composite score descending (highest first).
	sort.SliceStable(recs, func(i, j int) bool {
		return recs[i].CompositeScore > recs[j].CompositeScore
	})

	// Generate human-readable recommendation reasons.
	addReasons(recs)

	return recs
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// reliabilityScore returns a reliability score in [0.0, 1.0] derived from
// carrier performance data.  Returns 0.5 (neutral) when no data is available.
func (r *Router) reliabilityScore(channel, provider string) float64 {
	cp := r.svc.GetCarrierPerformanceByChannel(channel, provider)
	if cp == nil || cp.TotalOrders == 0 {
		return 0.5
	}
	// Loss-rate-based: 1.0 - lossRate%
	score := 1.0 - cp.LossRate/100.0
	if score < 0 {
		return 0
	}
	return score
}

// filterByConstraints removes quote results that violate delivery time or cost
// bounds.  A bound of zero means "no constraint".
func filterByConstraints(results []QuoteResult, maxDays int, maxCost float64) []QuoteResult {
	if maxDays <= 0 && maxCost <= 0 {
		return results
	}
	filtered := make([]QuoteResult, 0, len(results))
	for _, q := range results {
		if maxDays > 0 && q.EstimatedDeliveryMin > maxDays {
			continue
		}
		if maxCost > 0 && q.TotalShippingFee > maxCost {
			continue
		}
		filtered = append(filtered, q)
	}
	return filtered
}

// scoreRecommendations normalizes cost and speed across all candidates, then
// computes the composite score using the provided weights.
func scoreRecommendations(recs []RouteRecommendation, w RouterWeights) {
	n := len(recs)
	if n == 0 {
		return
	}
	if n == 1 {
		// Single candidate always scores 1.0.
		recs[0].CompositeScore = 1.0
		return
	}

	// Find max values for min-max normalization.
	maxCost := recs[0].EstimatedCost
	maxDays := float64(recs[0].EstimatedDaysMax)
	for i := 1; i < n; i++ {
		if recs[i].EstimatedCost > maxCost {
			maxCost = recs[i].EstimatedCost
		}
		if float64(recs[i].EstimatedDaysMax) > maxDays {
			maxDays = float64(recs[i].EstimatedDaysMax)
		}
	}

	for i := range recs {
		// Cost score: lower cost → higher score (inverse of normalized cost).
		costScore := 0.0
		if maxCost > 0 {
			costScore = 1.0 - (recs[i].EstimatedCost / maxCost)
		}

		// Speed score: faster delivery → higher score (inverse of normalized days).
		speedScore := 0.0
		if maxDays > 0 {
			speedScore = 1.0 - (float64(recs[i].EstimatedDaysMax) / maxDays)
		}

		recs[i].CompositeScore = w.CostWeight*costScore +
			w.SpeedWeight*speedScore +
			w.ReliabilityWeight*recs[i].ReliabilityScore
	}
}

// addReasons generates human-readable recommendation reasons for each result.
// The top-ranked option is labeled "Recommended choice"; others are "Alternative".
func addReasons(recs []RouteRecommendation) {
	if len(recs) == 0 {
		return
	}

	// Find the minimum cost across all recommendations.
	minCost := recs[0].EstimatedCost
	for _, rec := range recs {
		if rec.EstimatedCost < minCost {
			minCost = rec.EstimatedCost
		}
	}

	for i := range recs {
		parts := make([]string, 0, 3)

		if i == 0 {
			parts = append(parts, "Recommended choice")
		} else {
			parts = append(parts, "Alternative")
		}

		// Cost comparison.
		if recs[i].EstimatedCost == minCost {
			parts = append(parts, "Lowest price")
		} else {
			diff := recs[i].EstimatedCost - minCost
			parts = append(parts, fmt.Sprintf("¥%.2f more than cheapest", diff))
		}

		// Reliability note.
		if recs[i].ReliabilityScore >= 0.8 {
			parts = append(parts, "High reliability")
		} else if recs[i].ReliabilityScore < 0.3 {
			parts = append(parts, "Low reliability")
		}

		recs[i].RecommendationReason = strings.Join(parts, " · ")
	}
}
