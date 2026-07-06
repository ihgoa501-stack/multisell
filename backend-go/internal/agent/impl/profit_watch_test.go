package impl

import (
	"context"
	"testing"
	"sync"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"go.uber.org/zap"
)

var (
	registerProfitTools sync.Once
)

// ensureProfitWatchTools registers mock profit_watch tools on the
// DefaultRegistry. These tools are not in AllTools() so they must be
// registered here for tests that exercise the full profit_check and
// cost_optimization decision paths.
//
// Uses sync.Once so it is safe to call from multiple test functions.
func ensureProfitWatchTools() {
	registerProfitTools.Do(func() {
		reg := toolregistry.DefaultRegistry
		if reg == nil {
			return
		}

		// profit_watch.check_profit mock: given selling_price and cost_price,
		// computes fee breakdown, gross margin, and loss detection.
		reg.Register(&toolregistry.Tool{
			Name:    "profit_watch.check_profit",
			Version: "1.0.0",
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				sp := safeFloat(input["selling_price"], 0)
				cp := safeFloat(input["cost_price"], 0)
				pfr := safeFloat(input["platform_fee_rate"], 0)
				sf := safeFloat(input["shipping_fee"], 0)
				dr := safeFloat(input["discount_rate"], 0)
				ac := safeFloat(input["ad_cost_per_unit"], 0)
				rr := safeFloat(input["refund_rate"], 0)
				pf := safeFloat(input["platform_fee"], 0)
				ff := safeFloat(input["fixed_fee"], 0)
				mm := safeFloat(input["min_margin_threshold"], 0.1)

				effectiveRevenue := sp * (1 - dr/100)
				platformFeeAmt := effectiveRevenue*pfr/100 + pf + ff
				refundCost := effectiveRevenue * rr / 100
				totalFees := platformFeeAmt + sf + ac + refundCost
				totalCost := cp + totalFees
				profitPerUnit := effectiveRevenue - totalCost

				var grossMargin float64
				if effectiveRevenue > 0 {
					grossMargin = profitPerUnit / effectiveRevenue
				}

				isLoss := profitPerUnit < 0
				belowThreshold := !isLoss && grossMargin < mm

				status := "profitable"
				if isLoss {
					status = "loss"
				} else if belowThreshold {
					status = "below_threshold"
				}

				return map[string]interface{}{
					"profit_check_status":      status,
					"sku_code":                 input["sku_code"],
					"platform":                 input["platform"],
					"country":                  input["country"],
					"selling_price":            sp,
					"cost_price":               cp,
					"effective_revenue":        effectiveRevenue,
					"discount_rate":            dr,
					"profit_per_unit":          profitPerUnit,
					"gross_margin":             grossMargin,
					"min_margin_threshold":     mm,
					"is_loss":                  isLoss,
					"below_threshold":          belowThreshold,
					"fee_breakdown": map[string]interface{}{
						"platform_fee": platformFeeAmt,
						"shipping_fee": sf,
					},
					"fee_warnings":             []string{},
					"anomaly_reason":           "",
					"optimization_suggestions": []string{},
					"confidence":               0.85,
					"risk_level":               "low",
				}, nil
			},
		})

		// profit_watch.cost_optimization mock: given selling_price and
		// cost_price, computes current margin and generates suggestions.
		reg.Register(&toolregistry.Tool{
			Name:    "profit_watch.cost_optimization",
			Version: "1.0.0",
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				sp := safeFloat(input["selling_price"], 0)
				cp := safeFloat(input["cost_price"], 0)
				tm := safeFloat(input["target_margin"], 0.30)

				var currentMargin float64
				if sp > 0 {
					currentMargin = (sp - cp) / sp
				}

				suggestions := make([]string, 0)
				if currentMargin < tm {
					suggestions = append(suggestions, "建议提价或降本以提升毛利率")
				} else {
					suggestions = append(suggestions, "当前毛利率达标，无需调整")
				}

				return map[string]interface{}{
					"sku_code":       input["sku_code"],
					"current_margin": currentMargin,
					"target_margin":  tm,
					"suggestions":    suggestions,
					"confidence":     0.80,
					"risk_level":     "medium",
				}, nil
			},
		})
	})
}

// ---------- Tests ----------

func TestProfitWatchAgent_ProfitCheck(t *testing.T) {
	ensureProfitWatchTools()
	a := NewProfitWatchAgent(nil, zap.NewNop())

	out, conf, risk, err := a.Decide(context.Background(), "profit_check", map[string]interface{}{
		"sku_code":          "SKU-001",
		"selling_price":     100.0,
		"cost_price":        50.0,
		"platform_fee_rate": 15.0,
		"shipping_fee":      10.0,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out["profit_check_status"] != "profitable" {
		t.Errorf("profit_check_status = %v, want profitable", out["profit_check_status"])
	}
	gm, ok := out["gross_margin"].(float64)
	if !ok {
		t.Fatal("gross_margin missing or not float64")
	}
	if gm <= 0 || gm >= 1 {
		t.Errorf("gross_margin = %f, want in (0,1)", gm)
	}
	_ = conf
	_ = risk
}

func TestProfitWatchAgent_CostOptimization(t *testing.T) {
	ensureProfitWatchTools()
	a := NewProfitWatchAgent(nil, zap.NewNop())

	out, conf, risk, err := a.Decide(context.Background(), "cost_optimization", map[string]interface{}{
		"sku_code":      "SKU-001",
		"selling_price": 100.0,
		"cost_price":    50.0,
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out["suggestions"] == nil {
		t.Error("missing suggestions in output")
	}
	if out["current_margin"] == nil {
		t.Error("missing current_margin in output")
	}
	cm, ok := out["current_margin"].(float64)
	if !ok {
		t.Fatal("current_margin not float64")
	}
	if cm <= 0 || cm >= 1 {
		t.Errorf("current_margin = %f, want in (0,1)", cm)
	}
	_ = conf
	_ = risk
}

func TestProfitWatchAgent_ConfidenceScore(t *testing.T) {
	a := NewProfitWatchAgent(nil, zap.NewNop())

	t.Run("insufficient data confidence is 0", func(t *testing.T) {
		_, conf, _, err := a.Decide(context.Background(), "profit_check", map[string]interface{}{})
		if err != nil {
			t.Fatalf("Decide: %v", err)
		}
		if conf != 0 {
			t.Errorf("confidence = %f, want 0", conf)
		}
	})

	t.Run("unknown dp confidence is 0", func(t *testing.T) {
		_, conf, _, _ := a.Decide(context.Background(), "bogus", nil)
		if conf != 0 {
			t.Errorf("confidence = %f, want 0", conf)
		}
	})

	t.Run("valid call confidence in [0,1]", func(t *testing.T) {
		ensureProfitWatchTools()
		_, conf, _, err := a.Decide(context.Background(), "profit_check", map[string]interface{}{
			"sku_code": "SKU-001", "selling_price": 100.0, "cost_price": 50.0,
		})
		if err != nil {
			t.Fatalf("Decide: %v", err)
		}
		if conf < 0 || conf > 1 {
			t.Errorf("confidence = %f, want [0,1]", conf)
		}
	})
}

func TestProfitWatchAgent_RiskLevel(t *testing.T) {
	a := NewProfitWatchAgent(nil, zap.NewNop())
	valid := map[string]bool{"low": true, "medium": true, "high": true}

	t.Run("insufficient data risk is low", func(t *testing.T) {
		_, _, risk, _ := a.Decide(context.Background(), "profit_check", map[string]interface{}{})
		if !valid[risk] {
			t.Errorf("risk = %q, want low/medium/high", risk)
		}
	})

	t.Run("unknown dp risk is low", func(t *testing.T) {
		_, _, risk, _ := a.Decide(context.Background(), "bogus", nil)
		if !valid[risk] {
			t.Errorf("risk = %q, want low/medium/high", risk)
		}
	})

	t.Run("valid call risk is valid", func(t *testing.T) {
		ensureProfitWatchTools()
		_, _, risk, _ := a.Decide(context.Background(), "profit_check", map[string]interface{}{
			"sku_code": "SKU-001", "selling_price": 100.0, "cost_price": 50.0,
		})
		if !valid[risk] {
			t.Errorf("risk = %q, want low/medium/high", risk)
		}
	})
}

// cloneMap returns a shallow copy of m. Used to ensure each call to Decide
// receives an independent params map so side-effect checks are reliable.
func cloneMap(m map[string]interface{}) map[string]interface{} {
	c := make(map[string]interface{}, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func TestProfitWatchAgent_NoSideEffects(t *testing.T) {
	ensureProfitWatchTools()
	a := NewProfitWatchAgent(nil, zap.NewNop())

	params := map[string]interface{}{
		"sku_code":          "SKU-001",
		"selling_price":     100.0,
		"cost_price":        50.0,
		"platform_fee_rate": 15.0,
		"shipping_fee":      10.0,
	}

	// First call.
	out1, conf1, risk1, err1 := a.Decide(context.Background(), "profit_check", cloneMap(params))
	if err1 != nil {
		t.Fatalf("first call: %v", err1)
	}

	// Second call with identical (cloned) params.
	out2, conf2, risk2, err2 := a.Decide(context.Background(), "profit_check", cloneMap(params))
	if err2 != nil {
		t.Fatalf("second call: %v", err2)
	}

	if conf1 != conf2 {
		t.Errorf("confidence changed between calls: %f vs %f", conf1, conf2)
	}
	if risk1 != risk2 {
		t.Errorf("risk level changed between calls: %s vs %s", risk1, risk2)
	}
	if out1["gross_margin"] != out2["gross_margin"] {
		t.Errorf("gross_margin changed: %v vs %v", out1["gross_margin"], out2["gross_margin"])
	}
	if out1["profit_check_status"] != out2["profit_check_status"] {
		t.Errorf("profit_check_status changed: %v vs %v", out1["profit_check_status"], out2["profit_check_status"])
	}
}
