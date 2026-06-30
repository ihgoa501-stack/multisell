package impl

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:impl_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestAdAdviceAgent_Decide(t *testing.T) {
	a := NewAdAdviceAgent()
	points := []string{"acos_analysis", "ad_optimization", "bogus"}
	for _, dp := range points {
		t.Run(dp, func(t *testing.T) {
			out, conf, risk, err := a.Decide(context.Background(), dp, map[string]interface{}{"campaign_id": "c1", "spend": 100.0, "sales": 500.0})
			if err != nil {
				t.Fatalf("Decide(%q): %v", dp, err)
			}
			if out == nil {
				t.Fatalf("Decide(%q): nil output", dp)
			}
			if conf < 0 || conf > 1 {
				t.Fatalf("Decide(%q): confidence out of range: %f", dp, conf)
			}
			if risk == "" {
				t.Fatalf("Decide(%q): empty risk level", dp)
			}
		})
	}
}

func TestCustomerServiceAgent_Classify(t *testing.T) {
	a := NewCustomerServiceAgent()
	tests := []struct{ msg, want string }{
		{"I want a refund", "high_risk"},
		{"where is my order", "shipping"},
		{"退货", "return"},
		{"hello world", "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.msg, func(t *testing.T) {
			out, _, _, _ := a.Decide(context.Background(), "auto_reply", map[string]interface{}{"message": tc.msg, "language": "en"})
			if out == nil {
				t.Fatalf("nil output for %q", tc.msg)
			}
			if got, ok := out["intent"].(string); ok && got != tc.want {
				t.Fatalf("intent = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBatchOpsAgent_Decide(t *testing.T) {
	a := NewBatchOpsAgent()
	out, conf, risk, err := a.Decide(context.Background(), "batch_prelisting", map[string]interface{}{"skus": []string{"s1", "s2"}})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if conf < 0 || conf > 1 {
		t.Fatalf("confidence out of range: %f", conf)
	}
	_ = risk
}

func TestComplianceGuardAgent_Decide(t *testing.T) {
	a := NewComplianceGuardAgent()
	out, conf, risk, err := a.Decide(context.Background(), "compliance_check", map[string]interface{}{"sku_code": "s1", "title": "test"})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestCoordinatorAgent_Decide(t *testing.T) {
	a := NewCoordinatorAgent(nil, zap.NewNop())
	out, conf, risk, err := a.Decide(context.Background(), "system_health", nil)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestDiscountRiskAgent_Decide(t *testing.T) {
	a := NewDiscountRiskAgent(nil, zap.NewNop())
	out, conf, risk, err := a.Decide(context.Background(), "discount_risk_check", map[string]interface{}{"sku_code": "s1", "price": 100.0, "discount": 0.2})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestInventoryAlertAgent_Decide(t *testing.T) {
	a := NewInventoryAlertAgent(nil, zap.NewNop())
	out, conf, risk, err := a.Decide(context.Background(), "stock_alert", map[string]interface{}{"sku_code": "s1", "current_stock": 10, "safety_stock": 50})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestListingOptimizerAgent_Decide(t *testing.T) {
	a := NewListingOptimizerAgent()
	out, conf, risk, err := a.Decide(context.Background(), "listing_optimize", map[string]interface{}{"sku_code": "s1", "title": "test product", "marketplace": "US"})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestProfitWatchAgent_Decide(t *testing.T) {
	a := NewProfitWatchAgent(nil, zap.NewNop())
	out, conf, risk, err := a.Decide(context.Background(), "profit_watch", map[string]interface{}{"sku_code": "s1", "cost": 50.0, "price": 100.0, "sales_volume": 100})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestSettlementReconAgent_Decide(t *testing.T) {
	a := NewSettlementReconAgent(nil, zap.NewNop())
	out, conf, risk, err := a.Decide(context.Background(), "settlement_import", map[string]interface{}{"platform_id": 1, "period": "2026-06"})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestLogisticsOpsAgent_Decide(t *testing.T) {
	a := NewLogisticsOpsAgent(nil, zap.NewNop())
	out, conf, risk, err := a.Decide(context.Background(), "carrier_compare", map[string]interface{}{"weight_kg": 5.0, "destination": "US"})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestProductScoutAgent_Decide(t *testing.T) {
	a := NewProductScoutAgent()
	out, conf, risk, err := a.Decide(context.Background(), "product_scout", map[string]interface{}{"category": "electronics", "marketplace": "US"})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestDashboardAgent_Decide(t *testing.T) {
	a := NewDashboardAgent()
	out, conf, risk, err := a.Decide(context.Background(), "dashboard_overview", nil)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	_ = conf
	_ = risk
}

func TestSourcingAgent_Decide(t *testing.T) {
	a := NewSourcingAgent()
	// Viable product.
	out, conf, risk, err := a.Decide(context.Background(), "sourcing_recommend", map[string]interface{}{
		"source_url": "https://detail.1688.com/offer/xxx.html",
		"price_1688": 50.0,
		"weight_kg":  0.5,
		"destination": "US",
		"product_name": "测试商品",
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out["status"] != "viable" {
		t.Errorf("expected viable, got %v", out["status"])
	}
	if conf < 0 || conf > 1 {
		t.Fatalf("confidence out of range: %f", conf)
	}
	_ = risk

	// Missing required fields.
	out2, conf2, _, _ := a.Decide(context.Background(), "sourcing_recommend", map[string]interface{}{})
	if out2 == nil {
		t.Fatal("nil output for missing fields")
	}
	if out2["status"] != "insufficient_data" {
		t.Errorf("expected insufficient_data, got %v", out2["status"])
	}
	if conf2 != 0 {
		t.Errorf("expected confidence 0 for insufficient data, got %f", conf2)
	}

	// Unknown decision point.
	out3, conf3, _, _ := a.Decide(context.Background(), "unknown_dp", nil)
	if out3 == nil {
		t.Fatal("nil output for unknown dp")
	}
	if conf3 != 0 {
		t.Errorf("expected confidence 0 for unknown dp, got %f", conf3)
	}
}

func TestAftersalesMgmtAgent_Decide(t *testing.T) {
	a := NewAftersalesMgmtAgent(nil, zap.NewNop())

	tests := []struct {
		name string
		dp   string
		ctx  map[string]interface{}
	}{
		{"return analysis default period", "return_analysis", map[string]interface{}{}},
		{"return analysis 7d", "return_analysis", map[string]interface{}{"period": "7d"}},
		{"refund decision - single", "refund_decision", map[string]interface{}{"after_sales_id": float64(0)}},
		{"refund decision - scan", "refund_decision", map[string]interface{}{}},
		{"dispute manage", "dispute_manage", map[string]interface{}{}},
		{"aftersales report default", "aftersales_report", map[string]interface{}{}},
		{"aftersales report 90d", "aftersales_report", map[string]interface{}{"period": "90d"}},
		{"unknown decision point", "bogus", map[string]interface{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, conf, risk, err := a.Decide(context.Background(), tt.dp, tt.ctx)
			if err != nil {
				t.Fatalf("Decide(%q): %v", tt.dp, err)
			}
			if out == nil {
				t.Fatalf("Decide(%q): nil output", tt.dp)
			}
			if conf < 0 || conf > 1 {
				t.Fatalf("Decide(%q): confidence out of range: %f", tt.dp, conf)
			}
			if risk == "" {
				t.Fatalf("Decide(%q): empty risk level", tt.dp)
			}
		})
	}
}

func TestSafeFloat(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		def  []float64
		want float64
	}{
		{"nil", nil, nil, 0.0},
		{"nil with default", nil, []float64{42.0}, 42.0},
		{"int", 5, nil, 5.0},
		{"float", 3.14, nil, 3.14},
		{"string number", "7.5", nil, 0.0},
		{"string text", "abc", nil, 0.0},
		{"string text with default", "abc", []float64{1.0}, 1.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := safeFloat(tc.v, tc.def...)
			if got != tc.want {
				t.Errorf("safeFloat(%v, %v) = %f, want %f", tc.v, tc.def, got, tc.want)
			}
		})
	}
}
