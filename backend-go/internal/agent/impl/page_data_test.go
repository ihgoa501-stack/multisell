package impl

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
)

func TestPageDataToCandidate_ContentScriptPayload(t *testing.T) {
	db := dbtest.NewDB(t, &candidate.CandidateProduct{})
	svc := candidate.NewService(db, dbtest.NewLogger(t))

	// Simulates the JSON payload the Chrome content script sends via WebSocket:
	// price_1688, min_order_qty, source_url (content-script.ts field names)
	rawCS := `{
		"source_url": "https://detail.1688.com/offer/12345.html",
		"title": "智能蓝牙耳机 TWS Pro 2025新款",
		"price_1688": 45.0,
		"price_min": 35.0,
		"price_max": 55.0,
		"currency": "CNY",
		"min_order_qty": 2,
		"images": ["https://img.1688.com/01.jpg", "https://img.1688.com/02.jpg"],
		"supplier_name": "深圳某科技公司",
		"supplier_id_1688": "cn123456",
		"description": "高品质蓝牙5.3无线耳机",
		"package_weight_kg": 0.15,
		"package_length_cm": 8.0,
		"package_width_cm": 6.0,
		"package_height_cm": 4.0
	}`

	var pd toolbridge.PageData
	if err := json.Unmarshal([]byte(rawCS), &pd); err != nil {
		t.Fatalf("Unmarshal content-script payload: %v", err)
	}

	// Verify JSON tag aliases work (price_1688 → PriceCNY, min_order_qty → MOQ)
	if pd.PriceCNY != 45.0 {
		t.Fatalf("PriceCNY = %f (from price_1688), want 45.0", pd.PriceCNY)
	}
	if pd.MOQ != 2 {
		t.Fatalf("MOQ = %d (from min_order_qty), want 2", pd.MOQ)
	}
	if pd.Title != "智能蓝牙耳机 TWS Pro 2025新款" {
		t.Fatalf("Title = %q", pd.Title)
	}
	if pd.SourceURL != "https://detail.1688.com/offer/12345.html" {
		t.Fatalf("SourceURL = %q", pd.SourceURL)
	}

	// Convert PageData → CreateCandidateInput via pageDataToCandidate
	params := map[string]interface{}{
		"collected_by": "extension:1",
		"url":          pd.SourceURL,
	}
	input := pageDataToCandidate(&pd, params)

	// Verify core fields
	if input.Title != pd.Title {
		t.Fatalf("Title = %q", input.Title)
	}
	if input.SourceURL != pd.SourceURL {
		t.Fatalf("SourceURL = %q", input.SourceURL)
	}
	if input.SourcePlatform != "1688" {
		t.Fatalf("SourcePlatform = %q", input.SourcePlatform)
	}
	if input.PurchasePrice == nil || *input.PurchasePrice != 45.0 {
		t.Fatalf("PurchasePrice = %v, want 45.0", input.PurchasePrice)
	}
	if input.PackageWeightKg == nil || !approxEq(*input.PackageWeightKg, 0.15) {
		t.Fatalf("PackageWeightKg = %v, want 0.15", *input.PackageWeightKg)
	}
	if input.CollectedAt == nil {
		t.Fatal("CollectedAt is nil")
	}
	if input.CreatedBy != "extension:1" {
		t.Fatalf("CreatedBy = %q, want extension:1", input.CreatedBy)
	}

	// Verify raw_payload is set (marshaled PageData)
	if input.RawPayload == nil {
		t.Fatal("RawPayload is nil")
	}
	var back toolbridge.PageData
	if err := json.Unmarshal(*input.RawPayload, &back); err != nil {
		t.Fatalf("RawPayload unmarshal: %v", err)
	}
	if back.PriceCNY != 45.0 {
		t.Fatalf("RawPayload PriceCNY = %f", back.PriceCNY)
	}

	// Verify main_image is first image
	if input.MainImage != "https://img.1688.com/01.jpg" {
		t.Fatalf("MainImage = %q", input.MainImage)
	}

	// Verify completeness_status computed by Create
	c, err := svc.Create(input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.CompletenessStatus != "ready_for_profit_check" {
		t.Fatalf("CompletenessStatus = %q, want ready_for_profit_check", c.CompletenessStatus)
	}
	if c.SourceURL != "https://detail.1688.com/offer/12345.html" {
		t.Fatalf("SourceURL = %q", c.SourceURL)
	}
}

func TestPageDataToCandidate_CustomUnmarshalPriceCNY(t *testing.T) {
	// Verify canonical price_cny field name also works
	raw := `{"title":"test","price_cny":30.0,"source_url":"https://example.com/p"}`
	var pd toolbridge.PageData
	if err := json.Unmarshal([]byte(raw), &pd); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if pd.PriceCNY != 30.0 {
		t.Fatalf("PriceCNY = %f, want 30.0", pd.PriceCNY)
	}
}

func approxEq(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}
