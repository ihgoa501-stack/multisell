// Package sourcing provides the A8 Sourcing Agent domain logic.
//
// The primary component is the profit formula engine (profit.go) — a pure-code
// calculator that aggregates cost inputs from six modules (1688 price,
// exchange rate, logistics rate engine, platform fee, weight estimate, target
// sale price) into a single ProfitResult with margin and recommendation.
//
// All money arithmetic is hard-coded Go; the LLM only handles qualitative
// analysis (product quality, competition, listing copy) at a higher layer.
//
// Related modules:
//   - sourcing1688: CRUD for raw 1688 product data
//   - logistics: rate tables and the A10 Rate Engine
//   - exchangerate: currency conversion
//   - platformfee: platform commission rules
package sourcing
