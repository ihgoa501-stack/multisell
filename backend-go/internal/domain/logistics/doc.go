// Package logistics provides a clean-slate rate engine for cross-border shipping
// rate calculation from static rate tables. It is independent of the shipping
// domain package and does not require a database.
//
// Core components:
//   - RateEngine: evaluates rate table entries against cargo and destination
//     to produce shipping quotes.
//   - RateTableEntry: a single pricing rule with four pricing modes
//     (first_additional, tiered, fixed, per_kg).
//   - LoadRateTableFromYAML: parses rate tables from YAML configuration files.
//
// Pricing modes:
//   - first_additional: first kg price plus incremental additional kg charges
//   - tiered: weight bracket-based fixed prices
//   - fixed: flat fee regardless of weight
//   - per_kg: weight multiplied by per-kilogram price
//
// Each quote includes base fee, surcharges, fuel surcharge, and total.
// Minimum charge enforcement and fuel surcharge percentages are supported.
package logistics
