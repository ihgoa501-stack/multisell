package sourcing1688

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestControlledWorkflowCaptureReviewConvertToDraft(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &demandCaseRow{}, &experimentRow{}, &gateRow{}, &objectLinkRow{}, &platformRow{}, &productRow{}, &skuRow{}, &mediaRow{}, &costRow{}, &listingRow{}, &draftRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	db.Create(&demandCaseRow{ID: 7, OwnerID: 42, SalesChannel: "Ozon", Status: "experiment_ready"})
	db.Create(&experimentRow{ExperimentID: "EXP-1", OwnerID: 42, Status: "active", Stage: "product"})
	db.Create(&gateRow{ExperimentID: "EXP-1", Stage: "opportunity", Result: "pass"})
	db.Create(&objectLinkRow{ExperimentID: "EXP-1", ObjectType: "demand_case", ObjectID: "7"})
	db.Create(&platformRow{ID: 3, Name: "Ozon", Code: "ozon", Status: 1})
	now := time.Now().UTC().Truncate(time.Second)
	capture := &CaptureInput{DemandCaseID: 7, ExperimentID: "EXP-1", SourceURL: "https://detail.1688.com/offer/123.html#ignored", CollectedAt: now, CollectedBy: 42, Driver: "owner-browser", ParserVersion: "1.0.0", RawPayload: json.RawMessage(`{"offerId":123}`)}
	p, err := svc.Capture(capture)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if p.Status != StatusPendingReview || p.SnapshotID == nil {
		t.Fatalf("capture = %#v", p)
	}
	firstSnapshot := *p.SnapshotID
	p2, err := svc.Capture(capture)
	if err != nil || p2.ID != p.ID || *p2.SnapshotID != firstSnapshot {
		t.Fatalf("idempotent Capture = %#v, %v", p2, err)
	}
	var snapshotCount int64
	db.Model(&Sourcing1688Snapshot{}).Count(&snapshotCount)
	if snapshotCount != 1 {
		t.Fatalf("snapshot count = %d", snapshotCount)
	}
	var snapshot Sourcing1688Snapshot
	db.First(&snapshot, firstSnapshot)
	if snapshot.ParserVersion != "1.0.0" || len(snapshot.RawSHA256) != 64 {
		t.Fatalf("snapshot evidence = %#v", snapshot)
	}
	if _, err := svc.Convert(p.ID, completeConvertInput(now)); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("convert before review err = %v", err)
	}
	if _, err := svc.Review(p.ID, &ReviewInput{ReviewedBy: 42, Notes: "来源、权限与字段已人工核验"}); err != nil {
		t.Fatalf("Review: %v", err)
	}
	result, err := svc.Convert(p.ID, completeConvertInput(now))
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if result.Status != "draft" || result.ProductID == 0 || result.ListingID == 0 || len(result.SKUIDs) != 1 {
		t.Fatalf("result = %#v", result)
	}
	var listing listingRow
	db.First(&listing, result.ListingID)
	if listing.Status != "draft" {
		t.Fatalf("listing status = %s", listing.Status)
	}
	result2, err := svc.Convert(p.ID, completeConvertInput(now))
	if err != nil || result2.DraftID != result.DraftID {
		t.Fatalf("idempotent Convert = %#v, %v", result2, err)
	}
	if snapshot, err := svc.GetSnapshot(p.ID); err != nil || snapshot.ID != firstSnapshot {
		t.Fatalf("GetSnapshot = %#v, %v", snapshot, err)
	}
	if draft, err := svc.GetDraft(p.ID); err != nil || draft.Listing.Status != "draft" || len(draft.SKUs) != 1 || len(draft.Costs) != 11 {
		t.Fatalf("GetDraft = %#v, %v", draft, err)
	}
}

func completeConvertInput(now time.Time) *ConvertInput {
	costTypes := []string{"purchase", "domestic_shipping", "packaging", "cross_border_shipping", "platform_fee", "payment_fee", "advertising", "tax", "duty", "return_loss", "exchange_rate"}
	costs := make([]CostInput, 0, len(costTypes))
	for _, typ := range costTypes {
		costs = append(costs, CostInput{CostType: typ, Amount: 1, Currency: "CNY", TruthStatus: "quoted", SourceURI: "evidence://" + typ, ObservedAt: now})
	}
	supplierChecks := checks(now, []string{"identity", "operating_history", "transaction_history", "mixed_batch", "lead_time", "sample", "returns"})
	complianceChecks := checks(now, []string{"brand_ip", "patent", "certification", "dangerous_goods", "material", "labeling_instructions"})
	return &ConvertInput{CreatedBy: 42, PlatformID: 3, Title: "真实商品", Description: "已复核", CategoryID: 1, Unit: "件", LocalizedTitle: "Test product", LocalizedDescription: "Reviewed", TargetLocale: "ru-RU", ShippingTemplateID: "ozon-fbo-1", CategorySchemaURI: "evidence://ozon/category/1", CategoryObservedAt: now, SupplierAssessment: supplierChecks, ComplianceChecks: complianceChecks, PlatformSKU: "OZ-1", SKUVariants: []DraftSKUInput{{SupplierSKU: "1688-red", ChannelSKU: "OZ-1-red", SpecDesc: "red", SpecValues: json.RawMessage(`{"color":"red"}`), CostPrice: 10, Price: 20}}, Media: []MediaInput{{SourceURL: "https://cbu01.alicdn.com/a.jpg", ProcessedURL: "https://assets.local/a-clean.jpg", MediaRole: "main", RightsStatus: "verified", RightsEvidenceURI: "evidence://rights/1", Operations: json.RawMessage(`["crop","background_remove"]`), Width: 1200, Height: 1200, ChannelRuleURI: "evidence://ozon/images/1"}}, Costs: costs, ListingPayload: json.RawMessage(`{"category":"approved"}`)}
}

func checks(now time.Time, types []string) []EvidenceCheck {
	result := make([]EvidenceCheck, 0, len(types))
	for _, typ := range types {
		result = append(result, EvidenceCheck{CheckType: typ, Result: "pass", TruthStatus: "quoted", SourceURI: "evidence://" + typ, ObservedAt: now})
	}
	return result
}

func TestService_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, err := svc.Create(&CreateInput{
		SourceURL:    "https://detail.1688.com/offer/123.html",
		SupplierName: "某供应商",
		Price:        dbtest.FloatPtr(50.0),
		MOQ:          dbtest.IntPtr(10),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("ID should be set")
	}
	if p.SourceURL != "https://detail.1688.com/offer/123.html" {
		t.Fatalf("SourceURL = %s", p.SourceURL)
	}
	if p.Status != "collected" {
		t.Fatalf("Status = %s, expected collected", p.Status)
	}

	got, err := svc.Get(p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SupplierName != "某供应商" {
		t.Fatalf("SupplierName = %s", got.SupplierName)
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "https://detail.1688.com/offer/456.html"})
	updated, err := svc.Update(p.ID, &UpdateInput{
		SupplierName: dbtest.StringPtr("新供应商"),
		Price:        dbtest.FloatPtr(60.0),
		Status:       dbtest.StringPtr("reviewed"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.SupplierName != "新供应商" {
		t.Fatalf("SupplierName = %s", updated.SupplierName)
	}
	if updated.Price == nil || *updated.Price != 60.0 {
		t.Fatalf("Price = %v", updated.Price)
	}
}

func TestService_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{SourceURL: "url1", SupplierName: "A供应商"})
	svc.Create(&CreateInput{SourceURL: "url2", SupplierName: "B供应商", Status: "imported"})
	svc.Create(&CreateInput{SourceURL: "url3", SupplierName: "C供应商", Status: "rejected"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	items, total, err = svc.List(&p, &ListFilter{Status: "imported"})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 imported, got %d", total)
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "url_del"})
	if err := svc.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.Get(p.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Import(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "url_import"})
	imported, err := svc.Import(p.ID, &ImportInput{ImportedBy: "admin"})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if imported.Status != "imported" {
		t.Fatalf("Status = %s", imported.Status)
	}
}

func TestService_Reject(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "url_reject"})
	rejected, err := svc.Reject(p.ID, &RejectInput{RejectedBy: "admin", Reason: "价格过高"})
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Fatalf("Status = %s", rejected.Status)
	}
}

func TestService_Summary(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{SourceURL: "u1", Status: "collected"})
	svc.Create(&CreateInput{SourceURL: "u2", Status: "imported"})
	svc.Create(&CreateInput{SourceURL: "u3", Status: "collected"})

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Total != 3 {
		t.Fatalf("Total = %d", summary.Total)
	}
	if summary.ByStatus["collected"] != 2 {
		t.Fatalf("collected = %d", summary.ByStatus["collected"])
	}
}

func TestCreate_DefaultStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, err := svc.Create(&CreateInput{
		SourceURL: "https://detail.1688.com/offer/default-status.html",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Status != "collected" {
		t.Fatalf("Status = %q, want %q", p.Status, "collected")
	}
}

func TestCreate_WithAllFields(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	title := "测试商品标题"
	price := 99.50
	moq := 5
	supplierName := "优质供应商"
	shopURL := "https://shop.1688.com/abc"
	shopLocation := "浙江杭州"
	description := "优质商品描述"
	collectedBy := "collector01"

	p, err := svc.Create(&CreateInput{
		SourceURL:    "https://detail.1688.com/offer/all-fields.html",
		Title:        &title,
		Price:        &price,
		MOQ:          &moq,
		SupplierName: supplierName,
		ShopURL:      &shopURL,
		ShopLocation: &shopLocation,
		Description:  &description,
		CollectedBy:  &collectedBy,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("ID should be set")
	}
	if p.Title == nil || *p.Title != title {
		t.Fatalf("Title = %v", p.Title)
	}
	if p.Price == nil || *p.Price != price {
		t.Fatalf("Price = %v", p.Price)
	}
	if p.MOQ != moq {
		t.Fatalf("MOQ = %d", p.MOQ)
	}
	if p.SupplierName != supplierName {
		t.Fatalf("SupplierName = %s", p.SupplierName)
	}
	if p.ShopURL == nil || *p.ShopURL != shopURL {
		t.Fatalf("ShopURL = %v", p.ShopURL)
	}
	if p.ShopLocation == nil || *p.ShopLocation != shopLocation {
		t.Fatalf("ShopLocation = %v", p.ShopLocation)
	}
	if p.Description == nil || *p.Description != description {
		t.Fatalf("Description = %v", p.Description)
	}
	if p.CollectedBy == nil || *p.CollectedBy != collectedBy {
		t.Fatalf("CollectedBy = %v", p.CollectedBy)
	}
}

func TestList_Search(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{SourceURL: "u1", SupplierName: "Alpha Supplier"})
	svc.Create(&CreateInput{SourceURL: "u2", SupplierName: "Beta Supplier"})
	svc.Create(&CreateInput{SourceURL: "u3", SupplierName: "Gamma Corp"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, &ListFilter{Search: "Supplier"})
	if err != nil {
		t.Fatalf("List search 'Supplier': %v", err)
	}
	if total != 2 {
		t.Fatalf("search 'Supplier' total = %d, want 2", total)
	}
	_ = items

	items, total, err = svc.List(&p, &ListFilter{Search: "Alpha"})
	if err != nil {
		t.Fatalf("List search 'Alpha': %v", err)
	}
	if total != 1 {
		t.Fatalf("search 'Alpha' total = %d, want 1", total)
	}
	_ = items

	items, total, err = svc.List(&p, &ListFilter{Search: "NonExistent"})
	if err != nil {
		t.Fatalf("List search 'NonExistent': %v", err)
	}
	if total != 0 {
		t.Fatalf("search 'NonExistent' total = %d, want 0", total)
	}
	_ = items
}

func TestList_FilterByStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{SourceURL: "u1", Status: "collected"})
	svc.Create(&CreateInput{SourceURL: "u2", Status: "imported"})
	svc.Create(&CreateInput{SourceURL: "u3", Status: "rejected"})
	svc.Create(&CreateInput{SourceURL: "u4", Status: "collected"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, &ListFilter{Status: "collected"})
	if err != nil {
		t.Fatalf("List filter collected: %v", err)
	}
	if total != 2 {
		t.Fatalf("collected count = %d, want 2", total)
	}
	_ = items

	items, total, err = svc.List(&p, &ListFilter{Status: "imported"})
	if err != nil {
		t.Fatalf("List filter imported: %v", err)
	}
	if total != 1 {
		t.Fatalf("imported count = %d, want 1", total)
	}
	_ = items
}

func TestList_Pagination(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{SourceURL: "pag1", SupplierName: "A"})
	svc.Create(&CreateInput{SourceURL: "pag2", SupplierName: "B"})
	svc.Create(&CreateInput{SourceURL: "pag3", SupplierName: "C"})
	svc.Create(&CreateInput{SourceURL: "pag4", SupplierName: "D"})
	svc.Create(&CreateInput{SourceURL: "pag5", SupplierName: "E"})

	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 2}, nil)
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(items) != 2 {
		t.Fatalf("page 1 len = %d, want 2", len(items))
	}

	items, total, err = svc.List(&common.Pagination{Page: 3, Size: 2}, nil)
	if err != nil {
		t.Fatalf("List page 3: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(items) != 1 {
		t.Fatalf("page 3 len = %d, want 1", len(items))
	}
}

func TestImport_StatusTransition(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "url_import_transition"})
	imported, err := svc.Import(p.ID, &ImportInput{ImportedBy: "admin"})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if imported.Status != "imported" {
		t.Fatalf("Status = %s, want imported", imported.Status)
	}
}

func TestReject_StatusTransition(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "url_reject_transition"})
	rejected, err := svc.Reject(p.ID, &RejectInput{RejectedBy: "admin", Reason: "价格过高"})
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Fatalf("Status = %s, want rejected", rejected.Status)
	}
}

func TestReject_WithReason(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "url_reject_reason"})
	reason := "品质不过关"
	rejected, err := svc.Reject(p.ID, &RejectInput{RejectedBy: "reviewer", Reason: reason})
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Fatalf("Status = %s, want rejected", rejected.Status)
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.Get(99999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestSummary_Counts(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{SourceURL: "s1", Status: "collected"})
	svc.Create(&CreateInput{SourceURL: "s2", Status: "imported"})
	svc.Create(&CreateInput{SourceURL: "s3", Status: "collected"})
	svc.Create(&CreateInput{SourceURL: "s4", Status: "rejected"})

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Total != 4 {
		t.Fatalf("Total = %d, want 4", summary.Total)
	}
	if summary.ByStatus["collected"] != 2 {
		t.Fatalf("collected = %d, want 2", summary.ByStatus["collected"])
	}
	if summary.ByStatus["imported"] != 1 {
		t.Fatalf("imported = %d, want 1", summary.ByStatus["imported"])
	}
	if summary.ByStatus["rejected"] != 1 {
		t.Fatalf("rejected = %d, want 1", summary.ByStatus["rejected"])
	}
}
