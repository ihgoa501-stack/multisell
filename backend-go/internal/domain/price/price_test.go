package price

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestDB(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &Price{}, &PriceChangeLog{})
	return NewService(db, zap.NewNop())
}

func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

// ---------------------------------------------------------------------------
// Service tests — ListPrices
// ---------------------------------------------------------------------------

func TestListPrices_Empty(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	items, total, err := svc.ListPrices(ctx, 1, 20, 0, "")
	if err != nil {
		t.Fatalf("ListPrices on empty DB: %v", err)
	}
	if total != 0 {
		t.Fatalf("total=%d, want 0", total)
	}
	if len(items) != 0 {
		t.Fatalf("len(items)=%d, want 0", len(items))
	}
}

func TestListPrices_FilterBySkuID(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	svc.db.Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})
	svc.db.Create(&Price{SkuID: 1, PriceType: "compare_price", Price: decimal.NewFromInt(200)})
	svc.db.Create(&Price{SkuID: 2, PriceType: "sale_price", Price: decimal.NewFromInt(300)})

	items, total, err := svc.ListPrices(ctx, 1, 20, 1, "")
	if err != nil {
		t.Fatalf("ListPrices: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
	for _, it := range items {
		if it.SkuID != 1 {
			t.Fatalf("item SkuID=%d, want 1", it.SkuID)
		}
	}
}

func TestListPrices_FilterByPriceType(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	svc.db.Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})
	svc.db.Create(&Price{SkuID: 2, PriceType: "sale_price", Price: decimal.NewFromInt(200)})
	svc.db.Create(&Price{SkuID: 3, PriceType: "compare_price", Price: decimal.NewFromInt(300)})

	items, total, err := svc.ListPrices(ctx, 1, 20, 0, "sale_price")
	if err != nil {
		t.Fatalf("ListPrices: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
	for _, it := range items {
		if it.PriceType != "sale_price" {
			t.Fatalf("item PriceType=%q, want sale_price", it.PriceType)
		}
	}
}

func TestListPrices_Pagination(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		svc.db.Create(&Price{SkuID: int64(i), PriceType: "sale_price", Price: decimal.NewFromInt(int64(i * 100))})
	}

	// Page 1, size 2
	items, total, err := svc.ListPrices(ctx, 1, 2, 0, "")
	if err != nil {
		t.Fatalf("ListPrices page 1: %v", err)
	}
	if total != 5 {
		t.Fatalf("total=%d, want 5", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}

	// Page 3, size 2 — should get last item
	items, total, err = svc.ListPrices(ctx, 3, 2, 0, "")
	if err != nil {
		t.Fatalf("ListPrices page 3: %v", err)
	}
	if total != 5 {
		t.Fatalf("total=%d, want 5", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items)=%d, want 1", len(items))
	}
}

func TestListPrices_DefaultPagination(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	// Zero page/size should default to page=1 size=20
	items, total, err := svc.ListPrices(ctx, 0, 0, 0, "")
	if err != nil {
		t.Fatalf("ListPrices with zero pagination: %v", err)
	}
	if total != 0 {
		t.Fatalf("total=%d, want 0", total)
	}
	if len(items) != 0 {
		t.Fatalf("len(items)=%d, want 0", len(items))
	}
}

// ---------------------------------------------------------------------------
// Service tests — GetPriceByID
// ---------------------------------------------------------------------------

func TestGetPriceByID_Found(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	svc.db.Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})

	p, err := svc.GetPriceByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetPriceByID: %v", err)
	}
	if p.ID != 1 {
		t.Fatalf("ID=%d, want 1", p.ID)
	}
	if p.PriceType != "sale_price" {
		t.Fatalf("PriceType=%q, want sale_price", p.PriceType)
	}
}

func TestGetPriceByID_NotFound(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	_, err := svc.GetPriceByID(ctx, 999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Service tests — SetPrice
// ---------------------------------------------------------------------------

func TestSetPrice_CreateNew(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	p := &Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)}
	err := svc.SetPrice(ctx, p, "admin")
	if err != nil {
		t.Fatalf("SetPrice create: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}

	// Verify a change log was created
	var logs []PriceChangeLog
	svc.db.Where("sku_id = ?", 1).Find(&logs)
	if len(logs) != 1 {
		t.Fatalf("expected 1 change log, got %d", len(logs))
	}
	if logs[0].ChangeType != "manual" {
		t.Fatalf("ChangeType=%q, want manual", logs[0].ChangeType)
	}
	if logs[0].Operator != "admin" {
		t.Fatalf("Operator=%q, want admin", logs[0].Operator)
	}
	if logs[0].OldPrice != nil {
		t.Fatalf("OldPrice should be nil for new price, got %v", *logs[0].OldPrice)
	}
	if logs[0].NewPrice == nil || !logs[0].NewPrice.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("NewPrice should be 100, got %v", logs[0].NewPrice)
	}
}

func TestSetPrice_UpdateExistingBySkuType(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	// First create a price
	p := &Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)}
	if err := svc.SetPrice(ctx, p, "admin"); err != nil {
		t.Fatalf("first SetPrice: %v", err)
	}
	originalID := p.ID

	// Call SetPrice again with same SkuID/PriceType but different price
	// SetPrice should find the existing active price and update it
	p2 := &Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(150)}
	if err := svc.SetPrice(ctx, p2, "admin"); err != nil {
		t.Fatalf("second SetPrice: %v", err)
	}
	if p2.ID != originalID {
		t.Fatalf("after update ID=%d, want original ID=%d", p2.ID, originalID)
	}

	// Verify only 1 price record exists (was updated, not duplicated)
	var count int64
	svc.db.Model(&Price{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 price record, got %d", count)
	}

	// Verify the price was updated
	var updated Price
	svc.db.First(&updated, originalID)
	if !updated.Price.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("updated Price=%v, want 150", updated.Price)
	}

	// Verify 2 change logs exist
	var logs []PriceChangeLog
	svc.db.Where("sku_id = ?", 1).Order("id ASC").Find(&logs)
	if len(logs) != 2 {
		t.Fatalf("expected 2 change logs, got %d", len(logs))
	}
	// First log: create
	if logs[0].OldPrice != nil {
		t.Fatalf("log[0] OldPrice should be nil, got %v", *logs[0].OldPrice)
	}
	if logs[0].NewPrice == nil || !logs[0].NewPrice.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("log[0] NewPrice should be 100, got %v", logs[0].NewPrice)
	}
	// Second log: update
	if logs[1].OldPrice == nil || !logs[1].OldPrice.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("log[1] OldPrice should be 100, got %v", logs[1].OldPrice)
	}
	if logs[1].NewPrice == nil || !logs[1].NewPrice.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("log[1] NewPrice should be 150, got %v", logs[1].NewPrice)
	}
	if logs[1].ChangeType != "manual" {
		t.Fatalf("log[1] ChangeType=%q, want manual", logs[1].ChangeType)
	}
}

func TestSetPrice_UpdateExistingByID(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	// Create a price directly
	svc.db.Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})

	// Now update it by passing ID
	p := &Price{ID: 1, SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(200)}
	if err := svc.SetPrice(ctx, p, "admin"); err != nil {
		t.Fatalf("SetPrice update by ID: %v", err)
	}

	var price Price
	svc.db.First(&price, 1)
	if !price.Price.Equal(decimal.NewFromInt(200)) {
		t.Fatalf("Price=%v, want 200", price.Price)
	}

	// Verify only 1 price record
	var count int64
	svc.db.Model(&Price{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 price record, got %d", count)
	}
}

func TestSetPrice_EmptyPriceType(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	p := &Price{SkuID: 1, PriceType: "  ", Price: decimal.NewFromInt(100)}
	err := svc.SetPrice(ctx, p, "admin")
	if err == nil {
		t.Fatal("expected error for empty PriceType")
	}
}

// ---------------------------------------------------------------------------
// Service tests — UpdatePrice
// ---------------------------------------------------------------------------

func TestUpdatePrice(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	svc.db.Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1})

	p, _ := svc.GetPriceByID(ctx, 1)
	p.Price = decimal.NewFromInt(250)
	p.Status = 0

	if err := svc.UpdatePrice(ctx, p); err != nil {
		t.Fatalf("UpdatePrice: %v", err)
	}

	updated, _ := svc.GetPriceByID(ctx, 1)
	if !updated.Price.Equal(decimal.NewFromInt(250)) {
		t.Fatalf("Price=%v, want 250", updated.Price)
	}
	if updated.Status != 0 {
		t.Fatalf("Status=%d, want 0", updated.Status)
	}
}

// ---------------------------------------------------------------------------
// Service tests — DeletePrice
// ---------------------------------------------------------------------------

func TestDeletePrice(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	svc.db.Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})

	if err := svc.DeletePrice(ctx, 1); err != nil {
		t.Fatalf("DeletePrice: %v", err)
	}

	// Hard delete — record should be gone
	var count int64
	svc.db.Model(&Price{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 records after hard delete, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Service tests — ListPricesBySKU
// ---------------------------------------------------------------------------

func TestListPricesBySKU(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	svc.db.Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1})
	svc.db.Create(&Price{SkuID: 1, PriceType: "compare_price", Price: decimal.NewFromInt(200), Status: 1})
	svc.db.Create(&Price{SkuID: 2, PriceType: "sale_price", Price: decimal.NewFromInt(400), Status: 1})

	items, err := svc.ListPricesBySKU(ctx, 1)
	if err != nil {
		t.Fatalf("ListPricesBySKU: %v", err)
	}
	// Only returns status=1 records for SKU 1
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
	for _, it := range items {
		if it.SkuID != 1 {
			t.Fatalf("unexpected SkuID=%d, want 1", it.SkuID)
		}
		if it.Status != 1 {
			t.Fatalf("unexpected Status=%d, want 1", it.Status)
		}
	}
}

func TestListPricesBySKU_NotFound(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	items, err := svc.ListPricesBySKU(ctx, 999)
	if err != nil {
		t.Fatalf("ListPricesBySKU: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items)=%d, want 0", len(items))
	}
}

// ---------------------------------------------------------------------------
// Service tests — GetCurrentPrice
// ---------------------------------------------------------------------------

func TestGetCurrentPrice_NoTimeConstraint(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	svc.db.Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1})
	svc.db.Create(&Price{SkuID: 1, PriceType: "compare_price", Price: decimal.NewFromInt(200), Status: 1})

	p, err := svc.GetCurrentPrice(ctx, 1)
	if err != nil {
		t.Fatalf("GetCurrentPrice: %v", err)
	}
	if !p.Price.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("Price=%v, want 100", p.Price)
	}
	if p.PriceType != "sale_price" {
		t.Fatalf("PriceType=%q, want sale_price", p.PriceType)
	}
}

func TestGetCurrentPrice_WithValidTimeRange(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	svc.db.Create(&Price{
		SkuID:     1,
		PriceType: "sale_price",
		Price:     decimal.NewFromInt(99),
		Status:    1,
		StartTime: &past,
		EndTime:   &future,
	})

	p, err := svc.GetCurrentPrice(ctx, 1)
	if err != nil {
		t.Fatalf("GetCurrentPrice with valid time range: %v", err)
	}
	if !p.Price.Equal(decimal.NewFromInt(99)) {
		t.Fatalf("Price=%v, want 99", p.Price)
	}
}

func TestGetCurrentPrice_FutureStartTime(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	tomorrow := time.Now().Add(24 * time.Hour)
	nextWeek := time.Now().Add(7 * 24 * time.Hour)

	svc.db.Create(&Price{
		SkuID:     1,
		PriceType: "sale_price",
		Price:     decimal.NewFromInt(99),
		Status:    1,
		StartTime: &tomorrow,
		EndTime:   &nextWeek,
	})

	_, err := svc.GetCurrentPrice(ctx, 1)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound for future start time, got %v", err)
	}
}

func TestGetCurrentPrice_ExpiredEndTime(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	past := time.Now().Add(-2 * time.Hour)
	before := time.Now().Add(-1 * time.Hour)

	svc.db.Create(&Price{
		SkuID:     1,
		PriceType: "sale_price",
		Price:     decimal.NewFromInt(99),
		Status:    1,
		StartTime: &past,
		EndTime:   &before,
	})

	_, err := svc.GetCurrentPrice(ctx, 1)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound for expired end time, got %v", err)
	}
}

func TestGetCurrentPrice_NotFound(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	// Only compare_price exists, no sale_price
	svc.db.Create(&Price{SkuID: 1, PriceType: "compare_price", Price: decimal.NewFromInt(200), Status: 1})

	_, err := svc.GetCurrentPrice(ctx, 1)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestGetCurrentPrice_SkipsInactive(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	// Use raw SQL because GORM skips zero-value int16 fields in Create,
	// so Status=0 would use the database default (1).
	svc.db.WithContext(ctx).Exec(
		"INSERT INTO price (sku_id, price_type, price, status) VALUES (?, ?, ?, ?)",
		1, "sale_price", 100, 0,
	)

	_, err := svc.GetCurrentPrice(ctx, 1)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound for inactive price, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Service tests — ListChangeLogs
// ---------------------------------------------------------------------------

func TestListChangeLogs(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	svc.db.Create(&PriceChangeLog{
		SkuID: 1, PriceType: "sale_price",
		OldPrice: nil, NewPrice: decimalPtr(decimal.NewFromInt(100)),
		ChangeType: "manual", Operator: "admin",
	})
	svc.db.Create(&PriceChangeLog{
		SkuID: 1, PriceType: "sale_price",
		OldPrice: decimalPtr(decimal.NewFromInt(100)), NewPrice: decimalPtr(decimal.NewFromInt(120)),
		ChangeType: "manual", Operator: "admin",
	})
	svc.db.Create(&PriceChangeLog{
		SkuID: 2, PriceType: "sale_price",
		OldPrice: nil, NewPrice: decimalPtr(decimal.NewFromInt(500)),
		ChangeType: "auto", Operator: "system",
	})

	items, total, err := svc.ListChangeLogs(ctx, 1, 1, 10)
	if err != nil {
		t.Fatalf("ListChangeLogs: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
	for _, it := range items {
		if it.SkuID != 1 {
			t.Fatalf("item SkuID=%d, want 1", it.SkuID)
		}
	}
}

func TestListChangeLogs_Empty(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	items, total, err := svc.ListChangeLogs(ctx, 1, 1, 20)
	if err != nil {
		t.Fatalf("ListChangeLogs: %v", err)
	}
	if total != 0 {
		t.Fatalf("total=%d, want 0", total)
	}
	if len(items) != 0 {
		t.Fatalf("len(items)=%d, want 0", len(items))
	}
}

func TestListChangeLogs_Pagination(t *testing.T) {
	svc := newTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.db.Create(&PriceChangeLog{
			SkuID:      1,
			PriceType:  "sale_price",
			NewPrice:   decimalPtr(decimal.NewFromInt(int64(100 + i*10))),
			ChangeType: "manual",
			Operator:   "admin",
		})
	}

	items, total, err := svc.ListChangeLogs(ctx, 1, 1, 3)
	if err != nil {
		t.Fatalf("ListChangeLogs page 1: %v", err)
	}
	if total != 5 {
		t.Fatalf("total=%d, want 5", total)
	}
	if len(items) != 3 {
		t.Fatalf("len(items)=%d, want 3", len(items))
	}
}

// ---------------------------------------------------------------------------
// Handler test helpers
// ---------------------------------------------------------------------------

func setupRouter(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(svc)
	r := gin.New()
	rg := r.Group("/api/v1")
	prices := rg.Group("/prices")
	{
		prices.GET("", h.ListPrices)
		prices.GET("/:id", h.GetPrice)
		prices.POST("", h.SetPrice)
		prices.PUT("/:id", h.UpdatePrice)
		prices.DELETE("/:id", h.DeletePrice)
	}
	skus := rg.Group("/skus")
	{
		skus.GET("/:id/prices", h.ListPricesBySKU)
		skus.GET("/:id/current-price", h.GetCurrentPrice)
		skus.GET("/:id/price-history", h.PriceHistory)
	}
	return r
}

// verifyCode unmarshals the JSON response body and checks that the "code" field
// matches expectedCode. This avoids dealing with decimal.Decimal unmarshaling.
func verifyCode(t *testing.T, body []byte, expectedCode int) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v\nbody: %s", err, string(body))
	}
	if resp.Code != expectedCode {
		t.Fatalf("response code=%d, want %d\nbody: %s", resp.Code, expectedCode, string(body))
	}
}

// verifyMessage checks that the response message contains the given substring.
func verifyMessage(t *testing.T, body []byte, substr string) {
	t.Helper()
	var resp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if !strings.Contains(resp.Message, substr) {
		t.Fatalf("response message=%q, want substring %q", resp.Message, substr)
	}
}

// ---------------------------------------------------------------------------
// Handler tests — ListPrices
// ---------------------------------------------------------------------------

func TestHandler_ListPrices(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})
	svc.db.WithContext(ctx).Create(&Price{SkuID: 2, PriceType: "compare_price", Price: decimal.NewFromInt(200)})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/prices?page=1&size=10", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)

	// Verify pagination structure
	var pageResp struct {
		Total int64 `json:"total"`
		Page  int   `json:"page"`
		Size  int   `json:"size"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pageResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pageResp.Total != 2 {
		t.Fatalf("Total=%d, want 2", pageResp.Total)
	}
}

func TestHandler_ListPrices_Filtered(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})
	svc.db.WithContext(ctx).Create(&Price{SkuID: 2, PriceType: "sale_price", Price: decimal.NewFromInt(200)})
	svc.db.WithContext(ctx).Create(&Price{SkuID: 3, PriceType: "compare_price", Price: decimal.NewFromInt(300)})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/prices?page=1&size=10&price_type=sale_price", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)

	var pageResp struct {
		Total int64 `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &pageResp)
	if pageResp.Total != 2 {
		t.Fatalf("Total=%d, want 2", pageResp.Total)
	}
}

// ---------------------------------------------------------------------------
// Handler tests — GetPrice
// ---------------------------------------------------------------------------

func TestHandler_GetPrice_Found(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/prices/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

func TestHandler_GetPrice_NotFound(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/prices/999", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("HTTP status=%d, want 404", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusNotFound)
	verifyMessage(t, w.Body.Bytes(), "price not found")
}

func TestHandler_GetPrice_InvalidID(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/prices/abc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
	verifyMessage(t, w.Body.Bytes(), "invalid id")
}

// ---------------------------------------------------------------------------
// Handler tests — SetPrice
// ---------------------------------------------------------------------------

func TestHandler_SetPrice(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	body := `{"sku_id":1,"price_type":"sale_price","price":199.99}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/prices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200\nbody: %s", w.Code, w.Body.String())
	}
	verifyCode(t, w.Body.Bytes(), 0)

	// Verify the price was actually created
	var prices []Price
	svc.db.Find(&prices)
	if len(prices) != 1 {
		t.Fatalf("expected 1 price in DB, got %d", len(prices))
	}
	if prices[0].SkuID != 1 {
		t.Fatalf("SkuID=%d, want 1", prices[0].SkuID)
	}
}

func TestHandler_SetPrice_InvalidBody(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/prices", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
	verifyMessage(t, w.Body.Bytes(), "invalid request body")
}

func TestHandler_SetPrice_EmptyPriceType(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	body := `{"sku_id":1,"price_type":"","price":199.99}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/prices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("HTTP status=%d, want 500", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusInternalServerError)
}

// ---------------------------------------------------------------------------
// Handler tests — UpdatePrice
// ---------------------------------------------------------------------------

func TestHandler_UpdatePrice(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})

	body := `{"sku_id":1,"price_type":"sale_price","price":250}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/prices/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)

	// Verify the price was updated
	var price Price
	svc.db.First(&price, 1)
	if !price.Price.Equal(decimal.NewFromInt(250)) {
		t.Fatalf("Price=%v, want 250", price.Price)
	}
}

func TestHandler_UpdatePrice_InvalidID(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/prices/abc", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
	verifyMessage(t, w.Body.Bytes(), "invalid id")
}

// ---------------------------------------------------------------------------
// Handler tests — DeletePrice
// ---------------------------------------------------------------------------

func TestHandler_DeletePrice(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100)})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/prices/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)

	// Verify hard delete
	var count int64
	svc.db.Model(&Price{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 records after delete, got %d", count)
	}
}

func TestHandler_DeletePrice_InvalidID(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/prices/abc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
	verifyMessage(t, w.Body.Bytes(), "invalid id")
}

// ---------------------------------------------------------------------------
// Handler tests — ListPricesBySKU
// ---------------------------------------------------------------------------

func TestHandler_ListPricesBySKU(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1})
	svc.db.WithContext(ctx).Create(&Price{SkuID: 1, PriceType: "compare_price", Price: decimal.NewFromInt(200), Status: 1})
	svc.db.WithContext(ctx).Create(&Price{SkuID: 2, PriceType: "sale_price", Price: decimal.NewFromInt(300), Status: 1})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/skus/1/prices", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

func TestHandler_ListPricesBySKU_InvalidID(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/skus/abc/prices", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
	verifyMessage(t, w.Body.Bytes(), "invalid id")
}

// ---------------------------------------------------------------------------
// Handler tests — GetCurrentPrice
// ---------------------------------------------------------------------------

func TestHandler_GetCurrentPrice(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(99), Status: 1})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/skus/1/current-price", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

func TestHandler_GetCurrentPrice_NotFound(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/skus/1/current-price", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("HTTP status=%d, want 404", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusNotFound)
	verifyMessage(t, w.Body.Bytes(), "no active price found")
}

func TestHandler_GetCurrentPrice_InvalidID(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/skus/abc/current-price", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
	verifyMessage(t, w.Body.Bytes(), "invalid id")
}

// ---------------------------------------------------------------------------
// Handler tests — PriceHistory
// ---------------------------------------------------------------------------

func TestHandler_PriceHistory(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PriceChangeLog{
		SkuID: 1, PriceType: "sale_price",
		OldPrice: nil, NewPrice: decimalPtr(decimal.NewFromInt(100)),
		ChangeType: "manual", Operator: "admin",
	})
	svc.db.WithContext(ctx).Create(&PriceChangeLog{
		SkuID: 1, PriceType: "sale_price",
		OldPrice: decimalPtr(decimal.NewFromInt(100)), NewPrice: decimalPtr(decimal.NewFromInt(150)),
		ChangeType: "manual", Operator: "admin",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/skus/1/price-history?page=1&size=20", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)

	var pageResp struct {
		Total int64 `json:"total"`
		Page  int   `json:"page"`
		Size  int   `json:"size"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pageResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pageResp.Total != 2 {
		t.Fatalf("Total=%d, want 2", pageResp.Total)
	}
}

func TestHandler_PriceHistory_InvalidID(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/skus/abc/price-history", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
	verifyMessage(t, w.Body.Bytes(), "invalid id")
}
