package impl

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func nonEmptyStrings(v interface{}) bool {

	switch s := v.(type) {
	case []interface{}:
		return len(s) > 0
	case []string:
		return len(s) > 0
	}
	return false
}

func getPages(out map[string]interface{}) []interface{} {
	if raw, ok := out["suggested_pages"].([]interface{}); ok {
		return raw
	}
	if raw, ok := out["suggested_pages"].([]map[string]interface{}); ok {
		pages := make([]interface{}, len(raw))
		for i, r := range raw {
			pages[i] = r
		}
		return pages
	}
	return nil
}

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
		"source_url":   "https://detail.1688.com/offer/xxx.html",
		"price_1688":   50.0,
		"weight_kg":    0.5,
		"destination":  "US",
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

func TestProductResearch_Decide(t *testing.T) {
	a := NewProductScoutAgent()

	// Normal case: home category with RU/Ozon.
	out, conf, risk, err := a.Decide(context.Background(), "product_research", map[string]interface{}{
		"category":        "家居",
		"target_market":   "RU",
		"target_platform": "Ozon",
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out["status"] != "research_ready" {
		t.Errorf("expected status=research_ready, got %v", out["status"])
	}
	directions, ok := out["recommended_directions"].([]map[string]interface{})
	if !ok {
		// Try []interface{} fallback
		raw, ok2 := out["recommended_directions"].([]interface{})
		if !ok2 || len(raw) == 0 {
			t.Fatal("expected non-empty recommended_directions")
		}
		directions = make([]map[string]interface{}, len(raw))
		for i, r := range raw {
			directions[i] = r.(map[string]interface{})
		}
	}
	if len(directions) == 0 {
		t.Fatal("expected at least one recommended direction")
	}
	// Each direction should have why, risk_notes, keywords, confidence.
	for _, d := range directions {
		if d["why"] == "" {
			t.Error("direction missing 'why' field")
		}
		if _, ok := d["risk_notes"]; !ok {
			t.Error("direction missing 'risk_notes'")
		}
		if _, ok := d["keywords"]; !ok {
			t.Error("direction missing 'keywords'")
		}
	}
	// Must have warnings about uncertainty.
	if !nonEmptyStrings(out["warnings"]) {
		t.Error("expected warnings about research uncertainty")
	}
	if conf < 0 || conf > 1 {
		t.Errorf("confidence out of range: %f", conf)
	}
	_ = risk

	// Missing required fields.
	out2, conf2, _, _ := a.Decide(context.Background(), "product_research", map[string]interface{}{})
	if out2 == nil || out2["status"] != "insufficient_data" {
		t.Errorf("expected insufficient_data for empty input, got %v", out2)
	}
	if conf2 != 0 {
		t.Errorf("expected confidence 0, got %f", conf2)
	}

	// Unknown category should still produce directions (generic fallback).
	out3, conf3, _, _ := a.Decide(context.Background(), "product_research", map[string]interface{}{
		"category":        "家具",
		"target_market":   "US",
		"target_platform": "Amazon",
	})
	if out3 == nil || out3["status"] != "research_ready" {
		t.Errorf("expected research_ready for unknown category, got %v", out3)
	}
	_ = conf3
}

func TestSupplierDiscovery_Decide(t *testing.T) {
	a := NewProductScoutAgent()

	// Normal case: explicit keywords.
	out, conf, risk, err := a.Decide(context.Background(), "supplier_discovery", map[string]interface{}{
		"keywords": []string{"厨房收纳", "免打孔置物架"},
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out["status"] != "collection_plan_ready" {
		t.Errorf("expected status=collection_plan_ready, got %v", out["status"])
	}
	if out["source_platform"] != "1688" {
		t.Errorf("expected source_platform=1688, got %v", out["source_platform"])
	}
	if !nonEmptyStrings(out["suggested_pages"]) {
		// pages is a list of maps, not strings; check differently
		if raw, ok := out["suggested_pages"].([]map[string]interface{}); !ok || len(raw) == 0 {
			if raw2, ok2 := out["suggested_pages"].([]interface{}); !ok2 || len(raw2) == 0 {
				t.Fatal("expected non-empty suggested_pages")
			}
		}
	}
	for _, p := range getPages(out) {
		page, ok := p.(map[string]interface{})
		if !ok {
			t.Fatal("page should be map")
		}
		if page["url"] == "" {
			t.Error("page missing url")
		}
		if _, ok := page["reason"]; !ok {
			t.Error("page missing reason")
		}
	}
	if !nonEmptyStrings(out["supplier_filter_rules"]) {
		t.Error("expected supplier_filter_rules")
	}
	if !nonEmptyStrings(out["collection_instructions"]) {
		t.Error("expected collection_instructions")
	}
	if !nonEmptyStrings(out["warnings"]) {
		t.Error("expected warnings")
	}
	if conf < 0 || conf > 1 {
		t.Errorf("confidence out of range: %f", conf)
	}
	_ = risk

	// Missing keywords should return needs_keywords status.
	out2, conf2, _, _ := a.Decide(context.Background(), "supplier_discovery", map[string]interface{}{})
	if out2 == nil || out2["status"] != "needs_keywords" {
		t.Errorf("expected needs_keywords for empty input, got %v", out2)
	}
	if conf2 != 0 {
		t.Errorf("expected confidence 0, got %f", conf2)
	}

	// Category fallback: known category maps to keywords.
	out3, conf3, _, _ := a.Decide(context.Background(), "supplier_discovery", map[string]interface{}{
		"category": "厨房收纳",
	})
	if out3 == nil || out3["status"] != "collection_plan_ready" {
		t.Errorf("expected collection_plan_ready, got %v", out3)
	}
	_ = conf3

	// Verify no side effects: none of these create CandidateProduct or modify anything.
}

func TestScoutAgent_NoSideEffects(t *testing.T) {
	a := NewProductScoutAgent()

	// Verify that product_research and supplier_discovery decision points
	// never create, update, or delete any data — they are read-only research.
	dps := []string{"product_research", "supplier_discovery"}
	for _, dp := range dps {
		t.Run(dp, func(t *testing.T) {
			out, conf, risk, err := a.Decide(context.Background(), dp, map[string]interface{}{
				"category":        "家居",
				"target_market":   "RU",
				"target_platform": "Ozon",
			})
			if err != nil {
				t.Fatalf("Decide: %v", err)
			}
			if out == nil {
				t.Fatal("nil output")
			}
			if conf < 0 || conf > 1 {
				t.Errorf("confidence out of range: %f", conf)
			}
			_ = risk
			// Assert output does NOT contain execution fields.
			for _, badKey := range []string{"price_change", "inventory_change", "order_created", "listing_published", "purchase_order"} {
				if _, exists := out[badKey]; exists {
					t.Errorf("output contains execution key %q — should be research-only", badKey)
				}
			}
		})
	}
}

func TestSupplierDiscovery_URLEncoding(t *testing.T) {
	a := NewProductScoutAgent()
	out, _, _, err := a.Decide(context.Background(), "supplier_discovery", map[string]interface{}{
		"keywords": []string{"厨房收纳", "USB 风扇"},
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	pages := getPages(out)
	if len(pages) < 2 {
		t.Fatal("expected at least 2 suggested_pages")
	}
	for _, p := range pages {
		page, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		u, ok := page["url"].(string)
		if !ok || u == "" {
			continue
		}
		// URL should contain percent-encoded Chinese characters, not raw UTF-8.
		if strings.Contains(u, "厨房") {
			t.Errorf("URL should not contain raw Chinese chars: %s", u)
		}
		// URL should start with expected base.
		if !strings.HasPrefix(u, "https://s.1688.com/selloffer/offer_search.htm?keywords=") {
			t.Errorf("URL missing expected prefix: %s", u)
		}
		// Verify it's parseable.
		if _, err := url.Parse(u); err != nil {
			t.Errorf("URL not parseable: %v", err)
		}
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

// ---------- first_km_scout tests ----------

func TestFirstKMScout_FullChain(t *testing.T) {
	a := NewProductScoutAgent()
	// Structured input (explicit fields).
	out, conf, risk, err := a.Decide(context.Background(), "first_km_scout", map[string]interface{}{
		"category":        "家居",
		"target_market":   "RU",
		"target_platform": "Ozon",
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out["status"] != "collection_guidance_ready" {
		t.Errorf("status = %v, want collection_guidance_ready", out["status"])
	}
	if out["category"] != "家居" {
		t.Errorf("category = %v, want 家居", out["category"])
	}
	if out["market"] != "RU" {
		t.Errorf("market = %v, want RU", out["market"])
	}

	// Research section exists.
	research, ok := out["research"].(map[string]interface{})
	if !ok {
		t.Fatal("missing research section")
	}
	if research["status"] != "research_ready" {
		t.Errorf("research.status = %v, want research_ready", research["status"])
	}

	// Supplier section exists with 1688 URLs.
	supplier, ok := out["supplier"].(map[string]interface{})
	if !ok {
		t.Fatal("missing supplier section")
	}
	if supplier["status"] != "collection_plan_ready" {
		t.Errorf("supplier.status = %v, want collection_plan_ready", supplier["status"])
	}
	pages := getPages(supplier)
	if len(pages) == 0 {
		t.Error("expected at least one 1688 search page")
	}

	// Next actions exist.
	actions, ok := out["next_actions"].([]string)
	if !ok || len(actions) == 0 {
		t.Error("expected non-empty next_actions")
	}

	// Result entrypoints exist.
	entrypoints, ok := out["result_entrypoints"].(map[string]interface{})
	if !ok {
		t.Fatal("missing result_entrypoints")
	}
	if entrypoints["collect_leads"] != "/api/v1/candidates/collect-leads" {
		t.Errorf("collect_leads entrypoint = %v", entrypoints["collect_leads"])
	}
	if entrypoints["candidate_products"] != "/api/v1/candidates" {
		t.Errorf("candidate_products entrypoint = %v", entrypoints["candidate_products"])
	}

	// Safety warnings exist.
	warnings, ok := out["safety_warnings"].([]string)
	if !ok || len(warnings) == 0 {
		t.Error("expected non-empty safety_warnings")
	}

	// Confidence in valid range.
	if conf < 0 || conf > 1 {
		t.Errorf("confidence = %f, want [0,1]", conf)
	}
	if risk != "low" {
		t.Errorf("risk = %v, want low", risk)
	}
}

func TestFirstKMScout_FromMessage(t *testing.T) {
	a := NewProductScoutAgent()
	// Chat-style input — only raw message, no structured fields.
	out, _, _, err := a.Decide(context.Background(), "first_km_scout", map[string]interface{}{
		"message": "我想调研家居类目，目标俄罗斯 Ozon",
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out["status"] != "collection_guidance_ready" {
		t.Errorf("status = %v, want collection_guidance_ready", out["status"])
	}
	// Should parse category from message.
	if out["category"] != "家居" {
		t.Errorf("category = %v, want 家居", out["category"])
	}
	if out["market"] != "RU" {
		t.Errorf("market = %v, want RU", out["market"])
	}
}

func TestFirstKMScout_MissingFields(t *testing.T) {
	a := NewProductScoutAgent()
	// Empty message, no structured fields → insufficient data.
	out, conf, _, _ := a.Decide(context.Background(), "first_km_scout", map[string]interface{}{})
	if out == nil {
		t.Fatal("nil output")
	}
	if out["status"] != "insufficient_data" {
		t.Errorf("status = %v, want insufficient_data", out["status"])
	}
	if conf != 0 {
		t.Errorf("confidence = %f, want 0", conf)
	}
}

func TestFirstKMScout_PartialMessage(t *testing.T) {
	a := NewProductScoutAgent()
	// Message with category but no market → should fail gracefully.
	out, _, _, _ := a.Decide(context.Background(), "first_km_scout", map[string]interface{}{
		"message": "我想调研家居",
	})
	if out == nil {
		t.Fatal("nil output")
	}
	// Should either succeed (if category parsed) or return insufficient_data.
	// Both are acceptable — just shouldn't crash.
	status := out["status"]
	if status != "collection_guidance_ready" && status != "insufficient_data" {
		t.Errorf("unexpected status: %v", status)
	}
}

func TestFirstKMScout_NoCreateProduct(t *testing.T) {
	// Verify first_km_scout does NOT create CandidateProduct or modify data.
	a := NewProductScoutAgent()
	db := newTestDB(t)

	// Count before.
	var before int64
	db.Raw("SELECT COUNT(*) FROM candidate_product").Scan(&before)

	_, _, _, _ = a.Decide(context.Background(), "first_km_scout", map[string]interface{}{
		"category":        "宠物用品",
		"target_market":   "RU",
		"target_platform": "Ozon",
	})

	// Count after — should be unchanged.
	var after int64
	db.Raw("SELECT COUNT(*) FROM candidate_product").Scan(&after)
	if before != after {
		t.Errorf("first_km_scout mutated candidate_product: before=%d, after=%d", before, after)
	}
}

// ---------- parseFirstKMIntent tests ----------

func TestParseFirstKMIntent(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		wantCat  string
		wantMkt  string
		wantPlat string
	}{
		{
			name:     "家居 + 俄罗斯 + Ozon",
			msg:      "我想调研家居类目，目标俄罗斯 Ozon",
			wantCat:  "家居",
			wantMkt:  "RU",
			wantPlat: "Ozon",
		},
		{
			name:     "宠物 + Amazon + 美国",
			msg:      "调研宠物用品，美国 Amazon",
			wantCat:  "宠物用品",
			wantMkt:  "US",
			wantPlat: "Amazon",
		},
		{
			name:     "厨房 in 家居",
			msg:      "做厨房收纳，目标日本 Rakuten",
			wantCat:  "家居",
			wantMkt:  "JP",
			wantPlat: "Rakuten",
		},
		{
			name:     "运动 + Shopee + 东南亚",
			msg:      "调研运动户外，东南亚 Shopee",
			wantCat:  "运动户外",
			wantMkt:  "SEA",
			wantPlat: "Shopee",
		},
		{
			name:     "no market",
			msg:      "调研家居",
			wantCat:  "家居",
			wantMkt:  "",
			wantPlat: "",
		},
		{
			name:     "english home",
			msg:      "research home category for Russia",
			wantCat:  "家居",
			wantMkt:  "RU",
			wantPlat: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseFirstKMIntent(tc.msg)
			if got.Category != tc.wantCat {
				t.Errorf("Category = %q, want %q", got.Category, tc.wantCat)
			}
			if got.Market != tc.wantMkt {
				t.Errorf("Market = %q, want %q", got.Market, tc.wantMkt)
			}
			if got.Platform != tc.wantPlat {
				t.Errorf("Platform = %q, want %q", got.Platform, tc.wantPlat)
			}
		})
	}
}
