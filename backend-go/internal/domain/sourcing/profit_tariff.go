package sourcing

import (
	"github.com/lingmirror/backend-go/internal/domain/tariff"
)

// usdToCNYFallback is the fallback USD→CNY exchange rate used when no
// exchange-rate service is wired in. It keeps ApplyTariff self-contained
// for tests and avoids a hard dependency on the exchangerate module.
const usdToCNYFallback = 7.2

// TariffDecider is the minimal subset of tariff.Service needed to compute
// DDP/DDU decisions. Defined as an interface so tests can substitute a
// fake; production code passes *tariff.Service directly.
type TariffDecider interface {
	Decide(req *tariff.DecisionRequest) (*tariff.DecisionResult, error)
}

// ProfitBreakdownWithTariff wraps a base ProfitBreakdown and adds
// tariff-cost adjustments for cross-border sourcing decisions.
//
// This is a wrapper struct (rather than a field on ProfitBreakdown) so the
// existing CalculateProfit function and ProfitBreakdown struct stay
// untouched — keeping the change isolated from concurrent refactors of
// profit.go.
type ProfitBreakdownWithTariff struct {
	*ProfitBreakdown

	// TariffCostCNY is the duty+VAT+other tax converted to CNY.
	TariffCostCNY float64 `json:"tariff_cost_cny"`
	// TariffRuleID is the matched tariff rule ID (0 if no rule matched).
	TariffRuleID int64 `json:"tariff_rule_id"`
	// DDP is true when the tariff decision recommends Delivered Duty Paid
	// (seller absorbs tariff), false for DDU (buyer pays at destination).
	DDP bool `json:"ddp"`
	// IncotermReason explains why DDP or DDU was chosen.
	IncotermReason string `json:"incoterm_reason"`

	// AfterTariffProfitCNY is profit after subtracting tariff cost (only
	// meaningful when DDP — under DDU the buyer pays, so seller profit
	// equals ProfitCNY, but we still compute the DDP scenario for
	// comparison).
	AfterTariffProfitCNY float64 `json:"after_tariff_profit_cny"`
	// AfterTariffMarginPct is the margin after tariff (DDP scenario).
	AfterTariffMarginPct float64 `json:"after_tariff_margin_pct"`
}

// ApplyTariffInput is the input for ApplyTariff. It extends ProfitInput
// with the fields needed to look up the matching tariff rule.
type ApplyTariffInput struct {
	*ProfitInput
	// HSCode is the Harmonized System code for the product (optional but
	// recommended for accurate duty calculation).
	HSCode string `json:"hs_code"`
	// Quantity is the number of units (defaults to 1).
	Quantity int `json:"quantity"`
	// USDCNYRate overrides the fallback exchange rate. Pass 0 to use the
	// fallback.
	USDCNYRate float64 `json:"usd_cny_rate,omitempty"`
}

// ApplyTariff takes a base ProfitBreakdown and a tariff decision service
// and returns a ProfitBreakdownWithTariff that includes the tariff cost
// (in CNY) and the DDP/DDU recommendation.
//
// If tariffSvc is nil or the destination has no matching rule, the
// wrapper returns the base breakdown with zero tariff and DDU default —
// ApplyTariff never fails on tariff lookup errors, it degrades to "no
// tariff" so sourcing decisions are not blocked by missing tariff data.
//
// The tariff decision operates on USD (the tariff_rule table stores
// thresholds and rates in USD), so we convert the source price CNY → USD
// at the given exchange rate before calling Decide, then convert the
// resulting duty/tax USD → CNY for the profit breakdown.
func ApplyTariff(base *ProfitBreakdown, tariffSvc TariffDecider, in *ApplyTariffInput) *ProfitBreakdownWithTariff {
	out := &ProfitBreakdownWithTariff{ProfitBreakdown: base}

	if base == nil || tariffSvc == nil {
		return out
	}

	rate := in.USDCNYRate
	if rate <= 0 {
		rate = usdToCNYFallback
	}

	// Convert source price CNY → USD for the tariff decision.
	// The tariff Decide function multiplies ProductValueUSD by Quantity
	// internally, so we pass the per-unit source price in USD.
	productValueUSD := base.SourcePriceCNY / rate

	qty := in.Quantity
	if qty <= 0 {
		qty = 1
	}

	decision, err := tariffSvc.Decide(&tariff.DecisionRequest{
		DestinationCountry: base.Destination,
		ProductValueUSD:    productValueUSD,
		HSCode:             in.HSCode,
		Quantity:           qty,
	})
	if err != nil || decision == nil {
		// Degrade to DDU with no tariff — sourcing must not block on
		// tariff lookup failures.
		out.DDP = false
		out.IncotermReason = "tariff lookup failed; defaulting to DDU"
		return out
	}

	out.TariffCostCNY = round2(decision.TotalDutyTaxUSD * rate)
	out.DDP = decision.Incoterm == "DDP"
	out.IncotermReason = decision.IncotermReason
	if len(decision.RulesMatched) > 0 {
		out.TariffRuleID = decision.RulesMatched[0].RuleID
	}

	// Under DDP, the seller bears the tariff cost, so it reduces profit.
	// Under DDU, the buyer pays at destination, so seller profit is
	// unchanged — but we still compute the DDP-scenario profit for
	// comparison (useful when the seller is choosing between DDP/DDU).
	if out.DDP {
		out.AfterTariffProfitCNY = round2(base.ProfitCNY - out.TariffCostCNY)
		if base.TargetPriceCNY > 0 {
			out.AfterTariffMarginPct = round2(out.AfterTariffProfitCNY / base.TargetPriceCNY * 100.0)
		}
	} else {
		// DDU: seller profit unaffected by tariff; mirror base values.
		out.AfterTariffProfitCNY = base.ProfitCNY
		out.AfterTariffMarginPct = base.MarginPct
	}

	return out
}
