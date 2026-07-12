package sourcing1688

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestRequireWorkflowActorRejectsMissingJWTUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if _, ok := requireWorkflowActor(c); ok || w.Code != http.StatusUnauthorized {
		t.Fatalf("missing actor should return 401, code=%d", w.Code)
	}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("user_id", int64(42))
	if actor, ok := requireWorkflowActor(c); !ok || actor != 42 {
		t.Fatalf("valid actor = %d, %v", actor, ok)
	}
}

func TestOwnerScopedListAndSummaryDoNotLeakOtherOwners(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	db.Create(&demandCaseRow{ID: 1, OwnerID: 42, Status: "experiment_ready"})
	db.Create(&demandCaseRow{ID: 2, OwnerID: 99, Status: "experiment_ready"})
	one, two := int64(1), int64(2)
	db.Create(&Sourcing1688Product{SourceURL: "https://detail.1688.com/offer/1.html", DemandCaseID: &one, Status: StatusPendingReview})
	db.Create(&Sourcing1688Product{SourceURL: "https://detail.1688.com/offer/2.html", DemandCaseID: &two, Status: StatusPendingReview})

	items, total, err := svc.ListOwned(42, &common.Pagination{Page: 1, Size: 20}, &ListFilter{})
	if err != nil || total != 1 || len(items) != 1 || items[0].SourceURL != "https://detail.1688.com/offer/1.html" {
		t.Fatalf("ListOwned = %#v, total=%d, err=%v", items, total, err)
	}
	summary, err := svc.SummaryOwned(42)
	if err != nil || summary.Total != 1 || summary.ByStatus[StatusPendingReview] != 1 {
		t.Fatalf("SummaryOwned = %#v, err=%v", summary, err)
	}
}

func TestControlledWorkflowCaptureReviewConvertToDraft(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &SourcingChangeEvent{}, &DuplicateCandidate{}, &ImageProcessingRecord{}, &CaptureAttempt{}, &demandCaseRow{}, &experimentRow{}, &gateRow{}, &objectLinkRow{}, &platformRow{}, &productRow{}, &skuRow{}, &mediaRow{}, &costRow{}, &listingRow{}, &draftRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	db.Create(&demandCaseRow{ID: 7, OwnerID: 42, SalesChannel: "Ozon", Status: "experiment_ready"})
	db.Create(&experimentRow{ExperimentID: "EXP-1", OwnerID: 42, Status: "active", Stage: "product"})
	db.Create(&gateRow{ExperimentID: "EXP-1", Stage: "opportunity", Result: "pass"})
	db.Create(&objectLinkRow{ExperimentID: "EXP-1", ObjectType: "demand_case", ObjectID: "7"})
	db.Create(&platformRow{ID: 3, Name: "Ozon", Code: "ozon", Status: 1})
	now := time.Now().UTC().Truncate(time.Second)
	if failed, err := svc.RecordCaptureFailure(&CaptureFailureRecordInput{DemandCaseID: 7, ExperimentID: "EXP-1", SourceURL: "https://detail.1688.com/offer/999.html", AttemptedAt: now, Driver: "owner-browser", ParserVersion: "1.0.0", ErrorCode: "login_required", ErrorMessage: "1688 login required", AttemptedBy: 42}); err != nil || failed.Status != LifecycleCaptureFailed {
		t.Fatalf("RecordCaptureFailure = %#v, %v", failed, err)
	}
	capture := &CaptureInput{DemandCaseID: 7, ExperimentID: "EXP-1", SourceURL: "https://detail.1688.com/offer/123.html#ignored", CollectedAt: now, CollectedBy: 42, Driver: "owner-browser", ParserVersion: "1.0.0", SupplierBusinessID: "supplier-1688-42", RawPayload: json.RawMessage(`{"offerId":123}`)}
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
	changed := *capture
	changed.RawPayload = json.RawMessage(`{"offerId":123,"price":11,"moq":2,"images":["https://cbu01.alicdn.com/a.png"]}`)
	changed.Price = dbtest.FloatPtr(11)
	changed.MOQ = dbtest.IntPtr(2)
	p, err = svc.Capture(&changed)
	if err != nil || p.SnapshotID == nil || *p.SnapshotID == firstSnapshot {
		t.Fatalf("changed Capture = %#v, %v", p, err)
	}
	currentSnapshot := *p.SnapshotID
	var changes []SourcingChangeEvent
	db.Where("sourcing_product_id = ?", p.ID).Find(&changes)
	if len(changes) != 2 {
		t.Fatalf("change events = %#v", changes)
	}
	if _, err := svc.Convert(p.ID, completeConvertInput(now)); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("convert before review err = %v", err)
	}
	if _, err := svc.Review(p.ID, &ReviewInput{ReviewedBy: 42, Notes: "来源、权限与字段已人工核验"}); err != nil {
		t.Fatalf("Review: %v", err)
	}
	processed, err := svc.ProcessImage(&ProcessImageInput{SourcingProductID: p.ID, SourceURL: "https://cbu01.alicdn.com/a.png", SourceBase64: testSourceImageBase64(), Width: 1200, Height: 1200, Format: "jpeg", Quality: 90, RightsEvidenceURI: "evidence://rights/1", RightsTruthStatus: "actual", RightsObservedAt: now, ChannelRuleURI: "evidence://ozon/images/1", ProcessedBy: 42})
	if err != nil {
		t.Fatalf("ProcessImage: %v", err)
	}
	convertInput := completeConvertInput(now)
	convertInput.Media[0].ProcessingRecordID = processed.RecordID
	convertInput.Media[0].ProcessedURL = processed.ContentURL
	convertInput.Media[0].ContentSHA256 = processed.ProcessedSHA256
	convertInput.Media[0].Operations = processed.Operations
	convertInput.Validation.Images[0].TruthStatus = "actual"
	convertInput.Validation.Images[0].SourceURI = "sha256:" + processed.ProcessedSHA256
	result, err := svc.Convert(p.ID, convertInput)
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
	result2, err := svc.Convert(p.ID, convertInput)
	if err != nil || result2.DraftID != result.DraftID {
		t.Fatalf("idempotent Convert = %#v, %v", result2, err)
	}
	updatedInput := completeConvertInput(now)
	updatedInput.Title = "真实商品（已编辑）"
	updatedInput.Media[0] = convertInput.Media[0]
	updatedInput.Validation.Images[0] = convertInput.Validation.Images[0]
	updated, err := svc.UpdateDraft(p.ID, updatedInput)
	if err != nil || updated.Product.Name != "真实商品（已编辑）" || updated.Listing.Status != "draft" {
		t.Fatalf("UpdateDraft = %#v, %v", updated, err)
	}
	if snapshot, err := svc.GetSnapshot(p.ID); err != nil || snapshot.ID != currentSnapshot {
		t.Fatalf("GetSnapshot = %#v, %v", snapshot, err)
	}
	if draft, err := svc.GetDraft(p.ID); err != nil || draft.Listing.Status != "draft" || len(draft.SKUs) != 1 || len(draft.Costs) != 10 {
		t.Fatalf("GetDraft = %#v, %v", draft, err)
	}
}

func testSourceImageBase64() string {
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.RGBA{R: 240, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func completeConvertInput(now time.Time) *ConvertInput {
	costTypes := []string{"purchase", "domestic_shipping", "packaging", "cross_border_shipping", "platform_fee", "payment_fee", "advertising", "tax", "duty", "return_loss"}
	costs := make([]CostInput, 0, len(costTypes))
	for _, typ := range costTypes {
		costs = append(costs, CostInput{CostType: typ, Amount: 1, Currency: "CNY", TruthStatus: "quoted", SourceURI: "evidence://" + typ, ObservedAt: now})
	}
	validatedCosts := make([]CostLine, 0, len(costs))
	for _, cost := range costs {
		validatedCosts = append(validatedCosts, CostLine{Type: cost.CostType, Amount: cost.Amount, Currency: cost.Currency, TruthStatus: cost.TruthStatus, SourceURI: cost.SourceURI, ObservedAt: cost.ObservedAt})
	}
	supplierChecks := checks(now, []string{"identity", "operating_history", "transaction_history", "moq", "mixed_batch", "lead_time", "sample", "returns"})
	complianceChecks := checks(now, []string{"brand_ip", "patent", "certification", "dangerous_goods", "material", "labeling_instructions"})
	for i := range complianceChecks {
		complianceChecks[i].TruthStatus = "actual"
	}
	evidence := RuleEvidence{TruthStatus: "quoted", SourceURI: "evidence://rules", ObservedAt: now}
	channelEvidence := RuleEvidence{TruthStatus: "quoted", SourceURI: "evidence://ozon/category/1", ObservedAt: now}
	validation := DraftValidationInput{
		Localization:      LocalizationInput{Locale: "ru-RU", Title: "Тестовый товар", Description: "Проверенное описание", BulletPoints: []string{"Проверенное описание"}, Keywords: []string{"товар"}, Attributes: map[string]string{"material": "сталь"}, Unit: "件"},
		LocalizationRules: LocalizationRuleSnapshot{Evidence: evidence, Locale: "ru-RU", AllowedScripts: []string{"cyrillic"}, MinTitleLength: 3, MaxTitleLength: 100, MinBulletPoints: 1, MaxBulletLength: 200, MinKeywords: 1, AllowedUnits: []string{"件"}, ProhibitedWords: []string{"запрещено"}},
		Channel:           ChannelListingInput{PlatformID: 3, CategoryID: "1", CategorySchemaURI: "evidence://ozon/category/1", CategoryObservedAt: now, Attributes: map[string]string{"material": "steel"}, VariantDimensions: []string{"color"}, ImageCount: 1, ImageWidths: []int{1200}, ImageHeights: []int{1200}, ShippingTemplateID: "ozon-fbo-1"},
		ChannelRules:      ChannelRuleSnapshot{Evidence: channelEvidence, PlatformID: 3, CategoryID: "1", RequiredAttributes: []string{"material"}, RequiredVariantDimensions: []string{"color"}, AllowedVariantDimensions: []string{"color"}, MinImages: 1, MaxImages: 10, MinImageWidth: 1000, MinImageHeight: 1000, AllowedShippingTemplateIDs: []string{"ozon-fbo-1"}},
		Costs:             CostValidationInput{TargetCurrency: "CNY", Costs: validatedCosts, Revenue: RevenueInput{Amount: 30, Currency: "CNY", TruthStatus: "estimated", SourceURI: "evidence://revenue", ObservedAt: now}},
		Images:            []ImageValidationInput{{Role: "main", Width: 1200, Height: 1200, Background: "white", Cropped: true, ClarityScore: 0.95, TruthStatus: "actual", SourceURI: "sha256:", ObservedAt: now}},
		ImageRules:        ImageRuleSnapshot{Evidence: evidence, MinMainWidth: 1000, MinMainHeight: 1000, AllowedBackgrounds: []string{"white"}, RequireCrop: true, MinClarityScore: 0.8, MinImages: 1, MaxImages: 10},
		SKUs:              []SKUValidationInput{{SupplierSKU: "1688-red", InternalSKU: "INT-1-red", ChannelSKU: "OZ-1-red", Color: "red", Size: "standard", Material: "steel", Packaging: "box", TruthStatus: "quoted", SourceURI: "evidence://sku", ObservedAt: now}},
		SKURules:          SKUValidationRules{Evidence: evidence, RequireColor: true, RequireSize: true, RequireMaterial: true, RequirePackaging: true},
	}
	return &ConvertInput{CreatedBy: 42, PlatformID: 3, Title: "真实商品", Description: "已复核", CategoryID: 1, Unit: "件", LocalizedTitle: "Тестовый товар", LocalizedDescription: "Проверенное описание", TargetLocale: "ru-RU", ShippingTemplateID: "ozon-fbo-1", CategorySchemaURI: "evidence://ozon/category/1", CategoryObservedAt: now, SupplierAssessment: supplierChecks, ComplianceChecks: complianceChecks, PlatformSKU: "OZ-1", SKUVariants: []DraftSKUInput{{SupplierSKU: "1688-red", InternalSKU: "INT-1-red", ChannelSKU: "OZ-1-red", Color: "red", Size: "standard", Material: "steel", Packaging: "box", SpecDesc: "red", SpecValues: json.RawMessage(`{"color":"red","size":"standard","material":"steel","packaging":"box"}`), CostPrice: 10, Price: 20}}, Media: []MediaInput{{SourceURL: "https://cbu01.alicdn.com/a.png", ProcessedURL: "https://assets.local/a-clean.jpg", MediaRole: "main", RightsStatus: "verified", RightsEvidenceURI: "evidence://rights/1", RightsObservedAt: now, Operations: json.RawMessage(`["crop","background_remove"]`), Width: 1200, Height: 1200, ChannelRuleURI: "evidence://ozon/images/1"}}, Costs: costs, ListingPayload: json.RawMessage(`{"category":"approved"}`), Validation: validation}
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
