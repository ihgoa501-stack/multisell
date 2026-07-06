package completeness

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
)

func TestService_Check_CreateErrorIsReturned(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{}, &candidate.CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Insert a candidate so First() succeeds
	platformID := int64(1)
	prod := candidate.CandidateProduct{
		Title:            "Test",
		Description:      "Enough chars for completeness to pass title/desc checks",
		MainImage:        "https://example.test/img.jpg",
		Images:           []byte(`["img1.jpg"]`),
		CategoryID:       nil,
		BrandID:          nil,
		SpecJSON:         []byte(`{}`),
		PurchasePrice:    10,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  0.5,
		PackageLengthCm:  10,
		PackageWidthCm:   8,
		PackageHeightCm:  6,
		HSCode:           "1234.56",
		TargetSalePrice:  30,
		TargetPlatformID: &platformID,
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	// Drop the target table so Create fails
	db.Exec("DROP TABLE completeness_check")

	_, err := svc.Check(prod.ID, "tester")
	if err == nil {
		t.Fatal("expected error from Create (table dropped)")
	}
}

func TestService_Check_CompleteProduct(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	// The check reads candidate_product which won't exist in this test DB
	svc := NewService(db, dbtest.NewLogger(t))

	// Without a candidate product, it should error
	_, err := svc.Check(999, "tester")
	if err == nil {
		t.Fatal("expected error for non-existent product")
	}
}

func TestService_ListChecks(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Insert some checks directly
	db.Create(&CompletenessCheck{ProductID: 1, Score: 90, Status: "complete", TriggeredBy: "system"})
	db.Create(&CompletenessCheck{ProductID: 2, Score: 45, Status: "incomplete", TriggeredBy: "system"})

	items, total, err := svc.ListChecks(1, 10, "")
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
}

func TestService_ListChecks_Filtered(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&CompletenessCheck{ProductID: 1, Score: 90, Status: "complete"})
	db.Create(&CompletenessCheck{ProductID: 2, Score: 45, Status: "incomplete"})

	items, total, err := svc.ListChecks(1, 10, "complete")
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	if items[0].Status != "complete" {
		t.Fatalf("status = %s", items[0].Status)
	}
}

// ---------- unit tests for makeDim ----------

func TestMakeDim_Complete(t *testing.T) {
	t.Parallel()
	svc := NewService(nil, dbtest.NewLogger(t))

	dim := svc.makeDim("商品标题", true, "any value", "")
	if !dim.Complete {
		t.Fatal("expected Complete=true")
	}
	if dim.Score != 100 {
		t.Fatalf("expected Score=100, got %f", dim.Score)
	}
	if dim.Reason != "" {
		t.Fatalf("expected empty Reason, got %q", dim.Reason)
	}
	if dim.Label != "商品标题" {
		t.Fatalf("expected Label=商品标题, got %q", dim.Label)
	}
}

func TestMakeDim_Incomplete(t *testing.T) {
	t.Parallel()
	svc := NewService(nil, dbtest.NewLogger(t))

	dim := svc.makeDim("商品标题", false, "", "标题为空或过短（至少10个字符）")
	if dim.Complete {
		t.Fatal("expected Complete=false")
	}
	if dim.Score != 0 {
		t.Fatalf("expected Score=0, got %f", dim.Score)
	}
	if dim.Reason != "标题为空或过短（至少10个字符）" {
		t.Fatalf("expected Reason=%q, got %q", "标题为空或过短（至少10个字符）", dim.Reason)
	}
}

// ---------- unit tests for checkDimension ----------

func TestCheckDimension_ValidTitle(t *testing.T) {
	t.Parallel()
	svc := NewService(nil, dbtest.NewLogger(t))

	prod := &candidate.CandidateProduct{Title: "This is a valid title"}
	dim := svc.checkDimension(prod, "title", "商品标题")
	if !dim.Complete {
		t.Fatal("expected Complete=true for valid title")
	}
	if dim.Score != 100 {
		t.Fatalf("expected Score=100, got %f", dim.Score)
	}
}

func TestCheckDimension_MissingTitle(t *testing.T) {
	t.Parallel()
	svc := NewService(nil, dbtest.NewLogger(t))

	prod := &candidate.CandidateProduct{Title: "short"} // < 10 chars → incomplete
	dim := svc.checkDimension(prod, "title", "商品标题")
	if dim.Complete {
		t.Fatal("expected Complete=false for short title")
	}
	if dim.Score != 0 {
		t.Fatalf("expected Score=0, got %f", dim.Score)
	}
	if dim.Reason == "" {
		t.Fatal("expected non-empty Reason")
	}
}

func TestCheckDimension_EmptyImage(t *testing.T) {
	t.Parallel()
	svc := NewService(nil, dbtest.NewLogger(t))

	prod := &candidate.CandidateProduct{MainImage: ""}
	dim := svc.checkDimension(prod, "main_image", "主图")
	if dim.Complete {
		t.Fatal("expected Complete=false for empty main image")
	}
}

func TestCheckDimension_InsufficientImages(t *testing.T) {
	t.Parallel()
	svc := NewService(nil, dbtest.NewLogger(t))

	prod := &candidate.CandidateProduct{Images: []byte(`[]`)} // len ≤ 2
	dim := svc.checkDimension(prod, "images", "多图")
	if dim.Complete {
		t.Fatal("expected Complete=false for insufficient images")
	}
}

func TestCheckDimension_ZeroPurchasePrice(t *testing.T) {
	t.Parallel()
	svc := NewService(nil, dbtest.NewLogger(t))

	prod := &candidate.CandidateProduct{PurchasePrice: 0}
	dim := svc.checkDimension(prod, "purchase_price", "采购成本")
	if dim.Complete {
		t.Fatal("expected Complete=false for zero purchase price")
	}
}

func TestCheckDimension_MissingHSCode(t *testing.T) {
	t.Parallel()
	svc := NewService(nil, dbtest.NewLogger(t))

	prod := &candidate.CandidateProduct{HSCode: ""}
	dim := svc.checkDimension(prod, "hs_code", "HS编码")
	if dim.Complete {
		t.Fatal("expected Complete=false for empty HS code")
	}
}

func TestCheckDimension_UnknownKey(t *testing.T) {
	t.Parallel()
	svc := NewService(nil, dbtest.NewLogger(t))

	prod := &candidate.CandidateProduct{}
	dim := svc.checkDimension(prod, "nonexistent", "未知")
	if dim.Complete {
		t.Fatal("expected Complete=false for unknown key")
	}
	if dim.Reason != "未知检查项" {
		t.Fatalf("expected Reason=%q, got %q", "未知检查项", dim.Reason)
	}
}

// ---------- integration tests for Check ----------

func TestCheck_AllComplete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{}, &candidate.CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	catID := int64(1)
	brandID := int64(1)
	platformID := int64(1)

	prod := candidate.CandidateProduct{
		Title:            "优质蓝牙耳机",
		Description:      "高音质蓝牙耳机，支持主动降噪功能，续航时间长",
		MainImage:        "https://example.com/main.jpg",
		Images:           []byte(`["img1.jpg","img2.jpg","img3.jpg"]`),
		CategoryID:       &catID,
		BrandID:          &brandID,
		SpecJSON:         []byte(`{"color":"black","weight":"200g"}`),
		PurchasePrice:    50.0,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  0.5,
		PackageLengthCm:  10,
		PackageWidthCm:   8,
		PackageHeightCm:  6,
		HSCode:           "8518.30",
		TargetSalePrice:  120.0,
		TargetPlatformID: &platformID,
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	result, err := svc.Check(prod.ID, "tester")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Score != 100.0 {
		t.Fatalf("expected Score=100, got %f", result.Score)
	}
	if result.Status != "complete" {
		t.Fatalf("expected Status=complete, got %s", result.Status)
	}
	if len(result.MissingItems) != 0 {
		t.Fatalf("expected 0 missing items, got %v", result.MissingItems)
	}
	if result.ProductID != prod.ID {
		t.Fatalf("expected ProductID=%d, got %d", prod.ID, result.ProductID)
	}
	if len(result.Dimensions) != 12 {
		t.Fatalf("expected 12 dimensions, got %d", len(result.Dimensions))
	}
}

func TestCheck_Partial(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{}, &candidate.CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	platformID := int64(1)

	// Missing: CategoryID, BrandID, HSCode, TargetSalePrice
	prod := candidate.CandidateProduct{
		Title:            "优质蓝牙耳机 Pro",
		Description:      "高音质蓝牙耳机，支持主动降噪功能，续航时间长。",
		MainImage:        "https://example.com/main.jpg",
		Images:           []byte(`["img1.jpg","img2.jpg","img3.jpg"]`),
		SpecJSON:         []byte(`{"color":"black"}`),
		PurchasePrice:    50.0,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  0.5,
		PackageLengthCm:  10,
		PackageWidthCm:   8,
		PackageHeightCm:  6,
		HSCode:           "", // empty → incomplete
		TargetSalePrice:  0,  // zero → incomplete
		TargetPlatformID: &platformID,
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	result, err := svc.Check(prod.ID, "tester")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Score >= 100 {
		t.Fatalf("expected Score < 100, got %f", result.Score)
	}
	if result.Status != "incomplete" {
		t.Fatalf("expected Status=incomplete, got %s", result.Status)
	}
	if len(result.MissingItems) == 0 {
		t.Fatal("expected missing items")
	}
	if result.ProductID != prod.ID {
		t.Fatalf("expected ProductID=%d, got %d", prod.ID, result.ProductID)
	}
}

func TestCheck_StoreResult(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{}, &candidate.CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	catID := int64(1)
	brandID := int64(1)
	platformID := int64(1)

	prod := candidate.CandidateProduct{
		Title:            "优质蓝牙耳机 Pro Max",
		Description:      "高音质蓝牙耳机，支持主动降噪功能，续航时间长。",
		MainImage:        "https://example.com/main.jpg",
		Images:           []byte(`["img1.jpg","img2.jpg","img3.jpg"]`),
		CategoryID:       &catID,
		BrandID:          &brandID,
		SpecJSON:         []byte(`{"color":"black"}`),
		PurchasePrice:    50.0,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  0.5,
		PackageLengthCm:  10,
		PackageWidthCm:   8,
		PackageHeightCm:  6,
		HSCode:           "8518.30",
		TargetSalePrice:  120.0,
		TargetPlatformID: &platformID,
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	result, err := svc.Check(prod.ID, "tester")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	var stored CompletenessCheck
	if err := db.Where("product_id = ?", prod.ID).First(&stored).Error; err != nil {
		t.Fatalf("find stored completeness_check: %v", err)
	}
	if stored.Score != result.Score {
		t.Fatalf("stored Score=%f, result Score=%f", stored.Score, result.Score)
	}
	if stored.Status != result.Status {
		t.Fatalf("stored Status=%s, result Status=%s", stored.Status, result.Status)
	}
	if stored.TriggeredBy != "tester" {
		t.Fatalf("expected TriggeredBy=tester, got %s", stored.TriggeredBy)
	}
	if stored.ProductID != prod.ID {
		t.Fatalf("expected ProductID=%d, got %d", prod.ID, stored.ProductID)
	}
	if stored.ScoreBreakdown == "" {
		t.Fatal("expected non-empty ScoreBreakdown")
	}
}
