package candidate

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func TestComputeCompleteness_AllFieldsPresent(t *testing.T) {
	t.Parallel()
	sid := int64(1)
	p := &CandidateProduct{
		Title:           "Complete Product",
		PurchasePrice:   100.0,
		MainImage:       "https://example.com/img.jpg",
		PackageWeightKg: 0.5,
		PackageLengthCm: 10,
		PackageWidthCm:  8,
		PackageHeightCm: 6,
		SupplierID:      &sid,
	}
	status, missing := computeCompleteness(p)
	if status != "ready_for_profit_check" {
		t.Fatalf("expected ready_for_profit_check, got %s", status)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing fields, got %v", missing)
	}
}

func TestComputeCompleteness_MissingTitle(t *testing.T) {
	t.Parallel()
	sid := int64(1)
	p := &CandidateProduct{
		PurchasePrice:   100.0,
		MainImage:       "https://example.com/img.jpg",
		PackageWeightKg: 0.5,
		PackageLengthCm: 10,
		PackageWidthCm:  8,
		PackageHeightCm: 6,
		SupplierID:      &sid,
	}
	status, missing := computeCompleteness(p)
	if status != "incomplete" {
		t.Fatalf("expected incomplete, got %s", status)
	}
	found := false
	for _, f := range missing {
		if f == "title" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected title in missing fields")
	}
}

func TestComputeCompleteness_ZeroPurchasePrice(t *testing.T) {
	t.Parallel()
	sid := int64(1)
	p := &CandidateProduct{
		Title:           "Zero Price",
		PurchasePrice:   0,
		MainImage:       "https://example.com/img.jpg",
		PackageWeightKg: 0.5,
		PackageLengthCm: 10,
		PackageWidthCm:  8,
		PackageHeightCm: 6,
		SupplierID:      &sid,
	}
	status, missing := computeCompleteness(p)
	if status != "incomplete" {
		t.Fatalf("expected incomplete, got %s", status)
	}
	found := false
	for _, f := range missing {
		if f == "purchase_price" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected purchase_price in missing fields")
	}
}

func TestComputeCompleteness_MissingMainImage(t *testing.T) {
	t.Parallel()
	sid := int64(1)
	p := &CandidateProduct{
		Title:           "No Image",
		PurchasePrice:   50.0,
		MainImage:       "",
		PackageWeightKg: 0.5,
		PackageLengthCm: 10,
		PackageWidthCm:  8,
		PackageHeightCm: 6,
		SupplierID:      &sid,
	}
	status, missing := computeCompleteness(p)
	if status != "incomplete" {
		t.Fatalf("expected incomplete, got %s", status)
	}
	found := false
	for _, f := range missing {
		if f == "main_image" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected main_image in missing fields")
	}
}

func TestComputeCompleteness_MissingDimensions(t *testing.T) {
	t.Parallel()
	sid := int64(1)
	p := &CandidateProduct{
		Title:           "No Dimensions",
		PurchasePrice:   50.0,
		MainImage:       "https://example.com/img.jpg",
		PackageWeightKg: 0.5,
		PackageLengthCm: 10,
		PackageWidthCm:  0,
		PackageHeightCm: 0,
		SupplierID:      &sid,
	}
	status, missing := computeCompleteness(p)
	if status != "incomplete" {
		t.Fatalf("expected incomplete, got %s", status)
	}
	dimCount := 0
	for _, f := range missing {
		if f == "package_width_cm" || f == "package_height_cm" {
			dimCount++
		}
	}
	if dimCount != 2 {
		t.Fatalf("expected 2 dimension missing fields, got %v", missing)
	}
}

func TestComputeCompleteness_MissingSupplier(t *testing.T) {
	t.Parallel()
	p := &CandidateProduct{
		Title:           "No Supplier",
		PurchasePrice:   50.0,
		MainImage:       "https://example.com/img.jpg",
		PackageWeightKg: 0.5,
		PackageLengthCm: 10,
		PackageWidthCm:  8,
		PackageHeightCm: 6,
		SupplierID:      nil,
	}
	status, missing := computeCompleteness(p)
	if status != "incomplete" {
		t.Fatalf("expected incomplete, got %s", status)
	}
	found := false
	for _, f := range missing {
		if f == "supplier_id" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected supplier_id in missing fields")
	}
}

func TestCreate_SetsCompleteness(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create a product with minimal fields - should be "incomplete"
	price := 50.0
	c, err := svc.Create(&CreateCandidateInput{
		Title:         "Incomplete Product",
		PurchasePrice: &price,
		CreatedBy:     "tester",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.CompletenessStatus != "incomplete" {
		t.Fatalf("expected incomplete, got %s", c.CompletenessStatus)
	}
}

func TestCreate_CompletenessOverride(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	c, err := svc.Create(&CreateCandidateInput{
		Title:              "Overridden",
		PurchasePrice:      &price,
		CreatedBy:          "tester",
		CompletenessStatus: "ready_for_profit_check",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.CompletenessStatus != "ready_for_profit_check" {
		t.Fatalf("expected ready_for_profit_check, got %s", c.CompletenessStatus)
	}
}

func TestCreate_DedupRejectsDuplicateSourceURL(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	url := "https://example.com/product/123"
	_, err := svc.Create(&CreateCandidateInput{
		Title:         "First",
		PurchasePrice: &price,
		SourceURL:     url,
		CreatedBy:     "tester",
	})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err = svc.Create(&CreateCandidateInput{
		Title:         "Duplicate",
		PurchasePrice: &price,
		SourceURL:     url,
		CreatedBy:     "tester",
	})
	if err == nil {
		t.Fatal("expected error for duplicate source_url")
	}
	if err != ErrDuplicateSourceURL {
		t.Fatalf("expected ErrDuplicateSourceURL, got %v", err)
	}
}

func TestCreate_AllowsSameSourceURLAfterDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	url := "https://example.com/product/recycled"
	c, err := svc.Create(&CreateCandidateInput{
		Title:         "First",
		PurchasePrice: &price,
		SourceURL:     url,
		CreatedBy:     "tester",
	})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	if err := svc.Delete(c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// After soft/hard delete, same URL should be allowed
	_, err = svc.Create(&CreateCandidateInput{
		Title:         "Recycled",
		PurchasePrice: &price,
		SourceURL:     url,
		CreatedBy:     "tester",
	})
	if err != nil {
		t.Fatalf("Create after delete: %v", err)
	}
}

func TestDedup_EmptyResults(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	results, err := svc.Dedup(2)
	if err != nil {
		t.Fatalf("Dedup: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results, got %d", len(results))
	}
}

func TestDedup_FindsDuplicates(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Insert directly via DB to bypass the duplicate check in Create
	now := time.Now()
	records := []CandidateProduct{
		{Title: "A", PurchasePrice: 50, SourceURL: "https://example.com/dup1", Status: "draft", CreatedBy: "tester", CreatedAt: now, UpdatedAt: now},
		{Title: "B", PurchasePrice: 50, SourceURL: "https://example.com/dup1", Status: "draft", CreatedBy: "tester", CreatedAt: now, UpdatedAt: now},
		{Title: "C", PurchasePrice: 50, SourceURL: "https://example.com/unique", Status: "draft", CreatedBy: "tester", CreatedAt: now, UpdatedAt: now},
	}
	for _, r := range records {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	results, err := svc.Dedup(2)
	if err != nil {
		t.Fatalf("Dedup: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 dup group, got %d", len(results))
	}
	if results[0].SourceURL != "https://example.com/dup1" {
		t.Fatalf("expected dup1 URL, got %s", results[0].SourceURL)
	}
	if results[0].Count != 2 {
		t.Fatalf("expected count 2, got %d", results[0].Count)
	}
}

func TestDedup_IgnoresEmptySourceURL(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Insert directly via DB to bypass duplicate check
	now := time.Now()
	records := []CandidateProduct{
		{Title: "A", PurchasePrice: 50, SourceURL: "", Status: "draft", CreatedBy: "tester", CreatedAt: now, UpdatedAt: now},
		{Title: "B", PurchasePrice: 50, SourceURL: "", Status: "draft", CreatedBy: "tester", CreatedAt: now, UpdatedAt: now},
	}
	for _, r := range records {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	results, err := svc.Dedup(2)
	if err != nil {
		t.Fatalf("Dedup: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty source_urls, got %d", len(results))
	}
}

func TestService_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	weight := 0.5
	salePrice := 120.0

	c, err := svc.Create(&CreateCandidateInput{
		Title:              "Test Product",
		Description:        "A test product description",
		PurchasePrice:      &price,
		PackageWeightKg:    &weight,
		TargetSalePrice:    &salePrice,
		DestinationCountry: "US",
		CreatedBy:          "tester",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("ID should be set")
	}
	if c.Title != "Test Product" {
		t.Fatalf("Title = %s", c.Title)
	}
	if c.Status != "draft" {
		t.Fatalf("Status = %s", c.Status)
	}

	got, err := svc.GetByID(c.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != c.ID {
		t.Fatal("ID mismatch")
	}
}

func TestService_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	svc.Create(&CreateCandidateInput{Title: "Prod A", PurchasePrice: &price, CreatedBy: "tester"})
	svc.Create(&CreateCandidateInput{Title: "Prod B", PurchasePrice: &price, CreatedBy: "tester"})
	svc.Create(&CreateCandidateInput{Title: "Prod C", PurchasePrice: &price, CreatedBy: "tester"})

	p := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.List(&p, "", "", "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d", len(items))
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	c, _ := svc.Create(&CreateCandidateInput{Title: "Original", PurchasePrice: &price, CreatedBy: "tester"})

	newTitle := "Updated"
	updated, err := svc.Update(c.ID, &UpdateCandidateInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("Title = %s", updated.Title)
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	c, _ := svc.Create(&CreateCandidateInput{Title: "ToDelete", PurchasePrice: &price, CreatedBy: "tester"})

	if err := svc.Delete(c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.GetByID(c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Count(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	svc.Create(&CreateCandidateInput{Title: "A", PurchasePrice: &price, CreatedBy: "tester"})
	svc.Create(&CreateCandidateInput{Title: "B", PurchasePrice: &price, CreatedBy: "tester"})

	total, err := svc.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
}

func TestService_Create_CompletenessStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	t.Run("ready_for_profit_check", func(t *testing.T) {
		price := 100.0
		weight := 0.5
		lengthCm := 10.0
		widthCm := 8.0
		heightCm := 6.0
		supplierID := int64(1)
		now := time.Now()
		c, err := svc.Create(&CreateCandidateInput{
			Title:           "测试商品",
			MainImage:       "https://example.com/img.jpg",
			PurchasePrice:   &price,
			PackageWeightKg: &weight,
			PackageLengthCm: &lengthCm,
			PackageWidthCm:  &widthCm,
			PackageHeightCm: &heightCm,
			SupplierID:      &supplierID,
			SourceURL:       "https://detail.1688.com/offer/123.html",
			SourcePlatform:  "1688",
			CollectedAt:     &now,
			CreatedBy:       "extension:1",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if c.CompletenessStatus != "ready_for_profit_check" {
			t.Fatalf("CompletenessStatus = %q, want ready_for_profit_check", c.CompletenessStatus)
		}
		if c.SourceURL != "https://detail.1688.com/offer/123.html" {
			t.Fatalf("SourceURL = %q", c.SourceURL)
		}
		if c.PurchasePrice != 100.0 {
			t.Fatalf("PurchasePrice = %f", c.PurchasePrice)
		}
	})

	t.Run("incomplete_missing_title", func(t *testing.T) {
		price := 100.0
		c, err := svc.Create(&CreateCandidateInput{
			PurchasePrice:  &price,
			SourceURL:      "https://detail.1688.com/offer/456.html",
			SourcePlatform: "1688",
			CreatedBy:      "extension:1",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if c.CompletenessStatus != "incomplete" {
			t.Fatalf("CompletenessStatus = %q, want incomplete", c.CompletenessStatus)
		}
	})

	t.Run("incomplete_missing_price", func(t *testing.T) {
		c, err := svc.Create(&CreateCandidateInput{
			Title:          "无价格商品",
			SourceURL:      "https://detail.1688.com/offer/789.html",
			SourcePlatform: "1688",
			CreatedBy:      "extension:1",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if c.CompletenessStatus != "incomplete" {
			t.Fatalf("CompletenessStatus = %q, want incomplete", c.CompletenessStatus)
		}
	})
}

func TestListCollectLeads(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CollectLead{})
	svc := NewService(db, newTestLogger(t))
	// Empty list.
	p := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListCollectLeads(&p, "")
	if err != nil {
		t.Fatalf("ListCollectLeads: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}

	// Create some leads.
	for i := 0; i < 3; i++ {
		err := svc.CreateCollectLead(&CollectLead{
			Title:   "测试商品",
			ShopHint: "店铺A",
		})
		if err != nil {
			t.Fatalf("CreateCollectLead: %v", err)
		}
	}

	// List with limit.
	p2 := common.Pagination{Page: 1, Size: 2}
	items, total, err = svc.ListCollectLeads(&p2, "")
	if err != nil {
		t.Fatalf("ListCollectLeads: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Total from pagination.
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}

}
func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zap.NewNop()
}
