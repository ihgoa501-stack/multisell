package productanalysis

import "math"

// CalculateProfitMargin computes estimated profit margin and a normalized
// profit score (0–100) from target sale price and total estimated cost.
//
// Returns:
//   - marginPct: estimated profit margin as a percentage (nil if price <= 0)
//   - score: normalized profit score 0–100 (nil if price <= 0)
//
// Edge cases handled:
//   - price <= 0  → both returns are nil (invalid input)
//   - cost >= price → margin is 0 or negative, score is 0
//   - cost == 0   → margin is 100%, score is 100
func CalculateProfitMargin(price, cost float64) (marginPct *float64, score *float64) {
	if price <= 0 {
		return nil, nil
	}

	// Clamp cost at 0
	if cost < 0 {
		cost = 0
	}

	var m float64
	if cost >= price {
		m = 0
	} else {
		m = (price - cost) / price * 100
	}

	// Round to 2 decimal places
	m = math.Round(m*100) / 100
	marginPct = &m

	// Normalize to 0–100 score: margin capped at 50% → score 100,
	// 0% margin → score 0, linear in between
	raw := m / 50.0 * 100
	if raw > 100 {
		raw = 100
	}
	if raw < 0 {
		raw = 0
	}
	s := math.Round(raw)
	score = &s

	return marginPct, score
}

// CalculateDemandScore is a stub until Phase 0 provides a data source.
// Always returns nil, "no_data".
func CalculateDemandScore() (*float64, string) {
	return nil, "no_data"
}

// CalculateCompetitionScore is a stub until Phase 0 provides a data source.
// Always returns nil, "no_data".
func CalculateCompetitionScore() (*float64, string) {
	return nil, "no_data"
}
