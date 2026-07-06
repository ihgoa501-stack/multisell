package impl

import (
	"context"
	"testing"
)

// TestProductScoutAgent_ProductResearch verifies that product_research returns
// recommended_directions with at least one direction, and that each direction
// has required metadata fields.
func TestProductScoutAgent_ProductResearch(t *testing.T) {
	a := NewProductScoutAgent()
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
		t.Errorf("status = %v, want research_ready", out["status"])
	}
	// Verify recommended_directions is non-empty.
	dirs, ok := out["recommended_directions"].([]map[string]interface{})
	if !ok {
		raw, ok2 := out["recommended_directions"].([]interface{})
		if !ok2 || len(raw) == 0 {
			t.Fatal("expected non-empty recommended_directions")
		}
		dirs = make([]map[string]interface{}, len(raw))
		for i, r := range raw {
			dirs[i] = r.(map[string]interface{})
		}
	}
	if len(dirs) == 0 {
		t.Fatal("expected at least one recommended direction")
	}
	// Each direction should have name and why.
	for _, d := range dirs {
		if _, ok := d["name"]; !ok || d["name"] == "" {
			t.Error("direction missing non-empty name")
		}
		if _, ok := d["why"]; !ok || d["why"] == "" {
			t.Error("direction missing non-empty why")
		}
	}
	_ = conf
	_ = risk
}

// TestProductScoutAgent_FirstKMScout verifies that first_km_scout returns
// confidence > 0.5 (combined research + supplier confidence).
func TestProductScoutAgent_FirstKMScout(t *testing.T) {
	a := NewProductScoutAgent()
	out, conf, risk, err := a.Decide(context.Background(), "first_km_scout", map[string]interface{}{
		"category":        "宠物用品",
		"target_market":   "RU",
		"target_platform": "Ozon",
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if conf < 0.5 {
		t.Errorf("confidence = %f, want > 0.5 (combined research + supplier)", conf)
	}
	if conf > 1.0 {
		t.Errorf("confidence = %f, want <= 1.0", conf)
	}
	if out["status"] != "collection_guidance_ready" {
		t.Errorf("status = %v, want collection_guidance_ready", out["status"])
	}
	_ = risk
}

// TestProductScoutAgent_SupplierDiscovery verifies that supplier_discovery
// returns suggested_pages (1688 search URLs) when keywords are provided.
func TestProductScoutAgent_SupplierDiscovery(t *testing.T) {
	a := NewProductScoutAgent()
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
		t.Errorf("status = %v, want collection_plan_ready", out["status"])
	}
	// Verify suggested_pages using the existing getPages helper.
	pages := getPages(out)
	if len(pages) == 0 {
		t.Error("expected at least one suggested page from keywords")
	}
	for _, p := range pages {
		page, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := page["url"]; !ok || page["url"] == "" {
			t.Error("page missing url")
		}
		if _, ok := page["reason"]; !ok {
			t.Error("page missing reason")
		}
	}
	_ = conf
	_ = risk
}

// TestProductScoutAgent_UnknownDP verifies that an unrecognized decision point
// returns status "unknown" with an error message and zero confidence.
func TestProductScoutAgent_UnknownDP(t *testing.T) {
	a := NewProductScoutAgent()
	out, conf, risk, err := a.Decide(context.Background(), "nonexistent_dp", nil)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if out == nil {
		t.Fatal("nil output")
	}
	if out["status"] != "unknown" {
		t.Errorf("status = %v, want unknown", out["status"])
	}
	if conf != 0 {
		t.Errorf("confidence = %f, want 0", conf)
	}
	if risk != "low" {
		t.Errorf("risk = %v, want low", risk)
	}
	errMsg, ok := out["error"].(string)
	if !ok || errMsg == "" {
		t.Error("expected non-empty error message in output")
	}
	if _, ok := out["decision_point"].(string); !ok {
		t.Error("output missing decision_point")
	}
}

// TestProductScoutAgent_OutputSchema verifies that each decision point's
// output map has the correct field types.
func TestProductScoutAgent_OutputSchema(t *testing.T) {
	a := NewProductScoutAgent()

	t.Run("product_research field types", func(t *testing.T) {
		out, _, _, _ := a.Decide(context.Background(), "product_research", map[string]interface{}{
			"category": "家居", "target_market": "US", "target_platform": "Amazon",
		})
		if out == nil {
			t.Fatal("nil output")
		}
		if _, ok := out["status"].(string); !ok {
			t.Error("status not string")
		}
		if _, ok := out["category"].(string); !ok {
			t.Error("category not string")
		}
		// recommended_directions
		if _, ok := out["recommended_directions"].([]map[string]interface{}); !ok {
			if _, ok2 := out["recommended_directions"].([]interface{}); !ok2 {
				t.Error("recommended_directions not a slice")
			}
		}
		// warnings is a string slice.
		if _, ok := out["warnings"].([]string); !ok {
			if _, ok2 := out["warnings"].([]interface{}); !ok2 {
				t.Error("warnings not a slice")
			}
		}
	})

	t.Run("supplier_discovery field types", func(t *testing.T) {
		out, _, _, _ := a.Decide(context.Background(), "supplier_discovery", map[string]interface{}{
			"keywords": []string{"test"},
		})
		if out == nil {
			t.Fatal("nil output")
		}
		if _, ok := out["source_platform"].(string); !ok {
			t.Error("source_platform not string")
		}
		// suggested_pages is []map[string]interface{}.
		if _, ok := out["suggested_pages"].([]map[string]interface{}); !ok {
			if _, ok2 := out["suggested_pages"].([]interface{}); !ok2 {
				t.Error("suggested_pages not a slice")
			}
		}
		// supplier_filter_rules is a string slice.
		if _, ok := out["supplier_filter_rules"].([]string); !ok {
			if _, ok2 := out["supplier_filter_rules"].([]interface{}); !ok2 {
				t.Error("supplier_filter_rules not string slice")
			}
		}
		if _, ok := out["warnings"].([]string); !ok {
			if _, ok2 := out["warnings"].([]interface{}); !ok2 {
				t.Error("warnings not a slice")
			}
		}
	})

	t.Run("first_km_scout field types", func(t *testing.T) {
		out, _, _, _ := a.Decide(context.Background(), "first_km_scout", map[string]interface{}{
			"category": "宠物用品", "target_market": "RU", "target_platform": "Ozon",
		})
		if out == nil {
			t.Fatal("nil output")
		}
		if _, ok := out["research"].(map[string]interface{}); !ok {
			t.Error("research not map")
		}
		if _, ok := out["supplier"].(map[string]interface{}); !ok {
			t.Error("supplier not map")
		}
		if _, ok := out["next_actions"].([]string); !ok {
			t.Error("next_actions not string slice")
		}
		if _, ok := out["result_entrypoints"].(map[string]interface{}); !ok {
			t.Error("result_entrypoints not map")
		}
		if _, ok := out["safety_warnings"].([]string); !ok {
			t.Error("safety_warnings not string slice")
		}
	})
}

// TestProductScoutAgent_Evidence verifies that decision points include
// uncertainty / warning data (the closest analogue to "evidence").
func TestProductScoutAgent_Evidence(t *testing.T) {
	a := NewProductScoutAgent()

	t.Run("product_research has warnings", func(t *testing.T) {
		out, _, _, _ := a.Decide(context.Background(), "product_research", map[string]interface{}{
			"category": "家居", "target_market": "RU", "target_platform": "Ozon",
		})
		if out == nil {
			t.Fatal("nil output")
		}
		warnings, ok := out["warnings"].([]string)
		if !ok {
			raw, ok2 := out["warnings"].([]interface{})
			if !ok2 || len(raw) == 0 {
				t.Fatal("expected non-empty warnings")
			}
			warnings = make([]string, len(raw))
			for i, r := range raw {
				warnings[i] = r.(string)
			}
		}
		if len(warnings) == 0 {
			t.Fatal("expected at least one warning (uncertainty note)")
		}
	})

	t.Run("supplier_discovery has warnings", func(t *testing.T) {
		out, _, _, _ := a.Decide(context.Background(), "supplier_discovery", map[string]interface{}{
			"keywords": []string{"test"},
		})
		if out == nil {
			t.Fatal("nil output")
		}
		warnings, ok := out["warnings"].([]string)
		if !ok {
			raw, ok2 := out["warnings"].([]interface{})
			if !ok2 || len(raw) == 0 {
				t.Fatal("expected non-empty supplier_discovery warnings")
			}
			warnings = make([]string, len(raw))
			for i, r := range raw {
				warnings[i] = r.(string)
			}
		}
		if len(warnings) == 0 {
			t.Fatal("expected at least one warning in supplier_discovery")
		}
	})

	t.Run("first_km_scout has safety_warnings", func(t *testing.T) {
		out, _, _, _ := a.Decide(context.Background(), "first_km_scout", map[string]interface{}{
			"category": "宠物用品", "target_market": "RU", "target_platform": "Ozon",
		})
		if out == nil {
			t.Fatal("nil output")
		}
		warnings, ok := out["safety_warnings"].([]string)
		if !ok {
			raw, ok2 := out["safety_warnings"].([]interface{})
			if !ok2 || len(raw) == 0 {
				t.Fatal("expected non-empty safety_warnings")
			}
			warnings = make([]string, len(raw))
			for i, r := range raw {
				warnings[i] = r.(string)
			}
		}
		if len(warnings) == 0 {
			t.Fatal("expected at least one safety warning")
		}
	})
}
