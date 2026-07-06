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
	db := dbtest.NewDB(t, &Price{}, &PriceChangeLog{}, &CompetitorPrice{}, &PricingStrategy{}, &PricingRecommendation{})
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
	h := NewHandler(svc, nil)
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

// ---------------------------------------------------------------------------
// New helpers for competitor pricing tests
// ---------------------------------------------------------------------------

func newTestDBWithEngine(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &Price{}, &PriceChangeLog{}, &CompetitorPrice{}, &PricingStrategy{}, &PricingRecommendation{})
	return NewService(db, zap.NewNop())
}

func setupPriceRouter(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(svc, nil)
	r := gin.New()
	rg := r.Group("/api/v1")

	// Price CRUD routes
	prices := rg.Group("/prices")
	{
		prices.GET("", h.ListPrices)
		prices.GET("/:id", h.GetPrice)
		prices.POST("", h.SetPrice)
		prices.PUT("/:id", h.UpdatePrice)
		prices.DELETE("/:id", h.DeletePrice)
	}

	// SKU-scoped price routes
	skus := rg.Group("/skus")
	{
		skus.GET("/:id/prices", h.ListPricesBySKU)
		skus.GET("/:id/current-price", h.GetCurrentPrice)
		skus.GET("/:id/price-history", h.PriceHistory)
	}

	// Competitor Price routes
	cp := rg.Group("/competitor-prices")
	{
		cp.GET("", h.ListCompetitorPrices)
		cp.GET("/:id", h.GetCompetitorPrice)
		cp.POST("", h.CreateCompetitorPrice)
		cp.DELETE("/:id", h.DeleteCompetitorPrice)
	}

	// Pricing Strategy routes
	ps := rg.Group("/pricing-strategies")
	{
		ps.GET("", h.ListPricingStrategies)
		ps.GET("/:id", h.GetPricingStrategy)
		ps.POST("", h.SavePricingStrategy)
		ps.PUT("/:id", h.UpdatePricingStrategy)
		ps.DELETE("/:id", h.DeletePricingStrategy)
	}

	// Pricing Recommendation routes
	pr := rg.Group("/pricing-recommendations")
	{
		pr.GET("", h.ListRecommendations)
		pr.POST("/generate", h.GenerateRecommendation)
		pr.POST("/:id/apply", h.ApplyRecommendation)
	}

	return r
}

// ---------------------------------------------------------------------------
// Pricing Engine -- unit tests
// ---------------------------------------------------------------------------

func TestPricingEngine_BuyBoxFirst(t *testing.T) {
	engine := NewPricingEngine()
	now := time.Now()

	result := engine.Generate(EngineInput{
		SkuID:        1,
		CurrentPrice: decimal.NewFromInt(100),
		CompetitorPrices: []CompetitorPrice{
			{CompetitorName: "comp_a", Price: decimal.NewFromInt(95), CapturedAt: now},
			{CompetitorName: "comp_b", Price: decimal.NewFromInt(90), CapturedAt: now},
		},
		StrategyType: StrategyBuyBoxFirst,
	})

	expected := decimal.NewFromInt(90).Mul(decimal.NewFromFloat(0.98)).Round(2)
	got := result.RecommendedPrice.Round(2)
	if !got.Equal(expected) {
		t.Fatalf("RecommendedPrice=%v, want %v", got, expected)
	}
	if result.StrategyUsed != StrategyBuyBoxFirst {
		t.Fatalf("StrategyUsed=%q, want %q", result.StrategyUsed, StrategyBuyBoxFirst)
	}
	if result.CompetitorCount != 2 {
		t.Fatalf("CompetitorCount=%d, want 2", result.CompetitorCount)
	}
	if result.RiskLevel != "medium" {
		t.Fatalf("RiskLevel=%q, want medium", result.RiskLevel)
	}
	if result.Reason == "" {
		t.Fatal("expected non-empty Reason")
	}
}

func TestPricingEngine_Match(t *testing.T) {
	engine := NewPricingEngine()
	now := time.Now()

	result := engine.Generate(EngineInput{
		SkuID:        2,
		CurrentPrice: decimal.NewFromInt(120),
		CompetitorPrices: []CompetitorPrice{
			{CompetitorName: "comp_x", Price: decimal.NewFromInt(110), CapturedAt: now},
			{CompetitorName: "comp_y", Price: decimal.NewFromInt(105), CapturedAt: now},
			{CompetitorName: "comp_z", Price: decimal.NewFromInt(108), CapturedAt: now},
		},
		StrategyType: StrategyMatch,
	})

	expected := decimal.NewFromInt(105)
	if !result.RecommendedPrice.Equal(expected) {
		t.Fatalf("RecommendedPrice=%v, want %v", result.RecommendedPrice, expected)
	}
	if result.CompetitorCount != 3 {
		t.Fatalf("CompetitorCount=%d, want 3", result.CompetitorCount)
	}
	if result.RiskLevel != "medium" {
		t.Fatalf("RiskLevel=%q, want medium", result.RiskLevel)
	}
}

func TestPricingEngine_ProfitPriority(t *testing.T) {
	engine := NewPricingEngine()

	result := engine.Generate(EngineInput{
		SkuID:           3,
		CurrentPrice:    decimal.NewFromInt(100),
		CompetitorPrices: []CompetitorPrice{
			{CompetitorName: "comp_a", Price: decimal.NewFromInt(80), CapturedAt: time.Now()},
		},
		StrategyType:    StrategyProfitPriority,
		Cost:            decimal.NewFromInt(50),
		PlatformFeeRate: 0.10,
	})

	expected := decimal.NewFromInt(100)
	if !result.RecommendedPrice.Equal(expected) {
		t.Fatalf("RecommendedPrice=%v, want %v", result.RecommendedPrice, expected)
	}
	if result.RiskLevel != "low" {
		t.Fatalf("RiskLevel=%q, want low", result.RiskLevel)
	}
}

func TestPricingEngine_ProfitPriorityBelowMinMargin(t *testing.T) {
	engine := NewPricingEngine()

	result := engine.Generate(EngineInput{
		SkuID:           4,
		CurrentPrice:    decimal.NewFromInt(40),
		CompetitorPrices: []CompetitorPrice{
			{CompetitorName: "comp_a", Price: decimal.NewFromInt(38), CapturedAt: time.Now()},
		},
		StrategyType:    StrategyProfitPriority,
		Cost:            decimal.NewFromInt(50),
		PlatformFeeRate: 0.10,
	})

	expected := decimal.NewFromInt(50).Mul(decimal.NewFromFloat(1.15)).Div(decimal.NewFromFloat(0.9)).Round(2)
	got := result.RecommendedPrice.Round(2)
	if !got.Equal(expected) {
		t.Fatalf("RecommendedPrice=%v, want %v", got, expected)
	}
}

func TestPricingEngine_ProfitPriorityNoCost(t *testing.T) {
	engine := NewPricingEngine()

	result := engine.Generate(EngineInput{
		SkuID:           5,
		CurrentPrice:    decimal.NewFromInt(100),
		CompetitorPrices: []CompetitorPrice{
			{CompetitorName: "comp_a", Price: decimal.NewFromInt(80), CapturedAt: time.Now()},
		},
		StrategyType: StrategyProfitPriority,
	})

	if !result.RecommendedPrice.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("RecommendedPrice=%v, want 100 (no cost fallback)", result.RecommendedPrice)
	}
	if result.RiskLevel != "low" {
		t.Fatalf("RiskLevel=%q, want low", result.RiskLevel)
	}
}

func TestPricingEngine_NoCompetitors(t *testing.T) {
	engine := NewPricingEngine()

	result := engine.Generate(EngineInput{
		SkuID:        6,
		CurrentPrice: decimal.NewFromInt(100),
		StrategyType: StrategyBuyBoxFirst,
	})

	if !result.RecommendedPrice.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("RecommendedPrice=%v, want 100 (no competitor fallback)", result.RecommendedPrice)
	}
	if result.RiskLevel != "high" {
		t.Fatalf("RiskLevel=%q, want high for missing data", result.RiskLevel)
	}
	if result.CompetitorCount != 0 {
		t.Fatalf("CompetitorCount=%d, want 0", result.CompetitorCount)
	}
	if !strings.Contains(result.Reason, "no competitor prices") {
		t.Fatalf("Reason=%q, want 'no competitor prices'", result.Reason)
	}
}

func TestPricingEngine_Bounds(t *testing.T) {
	engine := NewPricingEngine()
	now := time.Now()

	result := engine.Generate(EngineInput{
		SkuID:        7,
		CurrentPrice: decimal.NewFromInt(100),
		CompetitorPrices: []CompetitorPrice{
			{CompetitorName: "comp_a", Price: decimal.NewFromInt(50), CapturedAt: now},
		},
		StrategyType: StrategyMatch,
		Parameters: StrategyParameters{
			MinPrice: 60,
		},
	})

	expected := decimal.NewFromInt(60)
	if !result.RecommendedPrice.Equal(expected) {
		t.Fatalf("RecommendedPrice=%v, want %v (clamped by min price)", result.RecommendedPrice, expected)
	}
	if !strings.Contains(result.Reason, "minimum price") {
		t.Fatalf("Reason=%q, want 'adjusted to minimum price'", result.Reason)
	}
}

func TestPricingEngine_DuplicateNames(t *testing.T) {
	engine := NewPricingEngine()
	now := time.Now()
	earlier := now.Add(-time.Hour)

	result := engine.Generate(EngineInput{
		SkuID:        8,
		CurrentPrice: decimal.NewFromInt(100),
		CompetitorPrices: []CompetitorPrice{
			{CompetitorName: "comp_a", Price: decimal.NewFromInt(90), CapturedAt: earlier},
			{CompetitorName: "comp_a", Price: decimal.NewFromInt(95), CapturedAt: now},
			{CompetitorName: "comp_b", Price: decimal.NewFromInt(85), CapturedAt: now},
		},
		StrategyType: StrategyMatch,
	})

	expected := decimal.NewFromInt(85)
	if !result.RecommendedPrice.Equal(expected) {
		t.Fatalf("RecommendedPrice=%v, want %v (dedup kept latest)", result.RecommendedPrice, expected)
	}
	if result.CompetitorCount != 2 {
		t.Fatalf("CompetitorCount=%d, want 2 (dedup)", result.CompetitorCount)
	}
}

func TestPricingEngine_CustomParams(t *testing.T) {
	engine := NewPricingEngine()
	now := time.Now()

	result := engine.Generate(EngineInput{
		SkuID:        9,
		CurrentPrice: decimal.NewFromInt(200),
		CompetitorPrices: []CompetitorPrice{
			{CompetitorName: "comp_a", Price: decimal.NewFromInt(180), CapturedAt: now},
		},
		StrategyType: StrategyBuyBoxFirst,
		Parameters: StrategyParameters{
			BuyBoxDiscount: 0.05,
		},
	})

	expected := decimal.NewFromInt(180).Mul(decimal.NewFromFloat(0.95)).Round(2)
	got := result.RecommendedPrice.Round(2)
	if !got.Equal(expected) {
		t.Fatalf("RecommendedPrice=%v, want %v (5%% discount)", got, expected)
	}
}

// ---------------------------------------------------------------------------
// Service tests -- CompetitorPrice CRUD
// ---------------------------------------------------------------------------

func TestCreateCompetitorPrice(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	cp := &CompetitorPrice{
		SkuID:          1,
		Platform:       "ozon",
		CompetitorName: "seller_x",
		Price:          decimal.NewFromInt(99),
	}
	if err := svc.CreateCompetitorPrice(ctx, cp); err != nil {
		t.Fatalf("CreateCompetitorPrice: %v", err)
	}
	if cp.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}
	if cp.Currency != "USD" {
		t.Fatalf("Currency=%q, want USD (default)", cp.Currency)
	}
	if cp.CapturedAt.IsZero() {
		t.Fatal("expected CapturedAt to be set")
	}
}

func TestListCompetitorPrices(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "a", Price: decimal.NewFromInt(100), CapturedAt: time.Now()})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "b", Price: decimal.NewFromInt(200), CapturedAt: time.Now()})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 2, CompetitorName: "c", Price: decimal.NewFromInt(300), CapturedAt: time.Now()})

	items, total, err := svc.ListCompetitorPrices(ctx, 1, 20, 1)
	if err != nil {
		t.Fatalf("ListCompetitorPrices: %v", err)
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

func TestGetCompetitorPriceByID(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "a", Price: decimal.NewFromInt(100), CapturedAt: time.Now()})

	cp, err := svc.GetCompetitorPriceByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetCompetitorPriceByID: %v", err)
	}
	if cp.SkuID != 1 {
		t.Fatalf("SkuID=%d, want 1", cp.SkuID)
	}
}

func TestGetCompetitorPriceByID_NotFound(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	_, err := svc.GetCompetitorPriceByID(ctx, 999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestDeleteCompetitorPrice(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "a", Price: decimal.NewFromInt(100), CapturedAt: time.Now()})

	if err := svc.DeleteCompetitorPrice(ctx, 1); err != nil {
		t.Fatalf("DeleteCompetitorPrice: %v", err)
	}

	var count int64
	svc.db.Model(&CompetitorPrice{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 records after delete, got %d", count)
	}
}

func TestGetLatestCompetitorPrices_Dedup(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	now := time.Now()
	earlier := now.Add(-2 * time.Hour)

	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "comp_a", Price: decimal.NewFromInt(80), CapturedAt: earlier})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "comp_b", Price: decimal.NewFromInt(90), CapturedAt: now})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "comp_a", Price: decimal.NewFromInt(85), CapturedAt: now})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 2, CompetitorName: "comp_c", Price: decimal.NewFromInt(100), CapturedAt: now})

	prices, err := svc.GetLatestCompetitorPrices(ctx, 1)
	if err != nil {
		t.Fatalf("GetLatestCompetitorPrices: %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("len(prices)=%d, want 2 (dedup)", len(prices))
	}

	for _, p := range prices {
		if p.CompetitorName == "comp_a" && !p.Price.Equal(decimal.NewFromInt(85)) {
			t.Fatalf("comp_a price=%v, want 85 (latest)", p.Price)
		}
	}
}

// ---------------------------------------------------------------------------
// Service tests -- PricingStrategy CRUD
// ---------------------------------------------------------------------------

func TestSaveAndGetPricingStrategy(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	ps := &PricingStrategy{
		SkuID:        int64Ptr(1),
		StrategyType: StrategyBuyBoxFirst,
		Parameters:   `{"buy_box_discount":0.03}`,
		Active:       true,
	}
	if err := svc.SavePricingStrategy(ctx, ps); err != nil {
		t.Fatalf("SavePricingStrategy: %v", err)
	}
	if ps.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := svc.GetPricingStrategyByID(ctx, ps.ID)
	if err != nil {
		t.Fatalf("GetPricingStrategyByID: %v", err)
	}
	if got.StrategyType != StrategyBuyBoxFirst {
		t.Fatalf("StrategyType=%q, want %q", got.StrategyType, StrategyBuyBoxFirst)
	}
}

func TestGetEffectiveStrategy_SkuFirst(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingStrategy{
		SkuID: nil, StrategyType: StrategyMatch, Active: true,
	})
	svc.db.WithContext(ctx).Create(&PricingStrategy{
		SkuID: int64Ptr(1), StrategyType: StrategyBuyBoxFirst, Active: true,
	})

	strategy, err := svc.GetEffectiveStrategy(ctx, 1)
	if err != nil {
		t.Fatalf("GetEffectiveStrategy: %v", err)
	}
	if strategy == nil {
		t.Fatal("expected non-nil strategy")
	}
	if strategy.StrategyType != StrategyBuyBoxFirst {
		t.Fatalf("StrategyType=%q, want buy_box_first (SKU-specific)", strategy.StrategyType)
	}
}

func TestGetEffectiveStrategy_GlobalFallback(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingStrategy{
		SkuID: nil, StrategyType: StrategyMatch, Active: true,
	})

	strategy, err := svc.GetEffectiveStrategy(ctx, 999)
	if err != nil {
		t.Fatalf("GetEffectiveStrategy: %v", err)
	}
	if strategy == nil {
		t.Fatal("expected non-nil strategy from global fallback")
	}
	if strategy.StrategyType != StrategyMatch {
		t.Fatalf("StrategyType=%q, want match (global default)", strategy.StrategyType)
	}
}

func TestGetEffectiveStrategy_None(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	strategy, err := svc.GetEffectiveStrategy(ctx, 1)
	if err != nil {
		t.Fatalf("GetEffectiveStrategy: %v", err)
	}
	if strategy != nil {
		t.Fatal("expected nil when no strategy configured")
	}
}

func TestListPricingStrategies(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingStrategy{StrategyType: StrategyBuyBoxFirst, Active: true})
	svc.db.WithContext(ctx).Create(&PricingStrategy{StrategyType: StrategyMatch, Active: true})
	svc.db.WithContext(ctx).Create(&PricingStrategy{SkuID: int64Ptr(1), StrategyType: StrategyProfitPriority, Active: true})

	items, total, err := svc.ListPricingStrategies(ctx, 1, 10, 0)
	if err != nil {
		t.Fatalf("ListPricingStrategies: %v", err)
	}
	if total != 3 {
		t.Fatalf("total=%d, want 3", total)
	}
	if len(items) != 3 {
		t.Fatalf("len(items)=%d, want 3", len(items))
	}
}

// ---------------------------------------------------------------------------
// Service tests -- Recommendation generation (integration)
// ---------------------------------------------------------------------------

func TestGenerateRecommendation_WithCompetitors(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&Price{
		SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1,
	})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{
		SkuID: 1, CompetitorName: "comp_a", Price: decimal.NewFromInt(90),
		CapturedAt: time.Now(),
	})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{
		SkuID: 1, CompetitorName: "comp_b", Price: decimal.NewFromInt(85),
		CapturedAt: time.Now(),
	})

	rec, err := svc.GenerateRecommendation(ctx, GenerateRecommendationInput{
		SkuID:        1,
		StrategyType: StrategyBuyBoxFirst,
	})
	if err != nil {
		t.Fatalf("GenerateRecommendation: %v", err)
	}
	if rec.ID == 0 {
		t.Fatal("expected non-zero ID (persisted)")
	}
	if rec.SkuID != 1 {
		t.Fatalf("SkuID=%d, want 1", rec.SkuID)
	}
	if !rec.CurrentPrice.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("CurrentPrice=%v, want 100", rec.CurrentPrice)
	}
	if rec.CompetitorCount != 2 {
		t.Fatalf("CompetitorCount=%d, want 2", rec.CompetitorCount)
	}
}

func TestGenerateRecommendation_NoCompetitors(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	rec, err := svc.GenerateRecommendation(ctx, GenerateRecommendationInput{
		SkuID:        1,
		StrategyType: StrategyMatch,
	})
	if err != nil {
		t.Fatalf("GenerateRecommendation: %v", err)
	}
	if rec.RiskLevel != "high" {
		t.Fatalf("RiskLevel=%q, want high for no competitors", rec.RiskLevel)
	}
	if rec.CompetitorCount != 0 {
		t.Fatalf("CompetitorCount=%d, want 0", rec.CompetitorCount)
	}
}

func TestGenerateRecommendation_EmptyStrategyUsesDefault(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&CompetitorPrice{
		SkuID: 1, CompetitorName: "comp_a", Price: decimal.NewFromInt(90),
		CapturedAt: time.Now(),
	})

	rec, err := svc.GenerateRecommendation(ctx, GenerateRecommendationInput{
		SkuID: 1,
	})
	if err != nil {
		t.Fatalf("GenerateRecommendation: %v", err)
	}
	if rec.RecommendedPrice.IsZero() {
		t.Fatal("expected non-zero RecommendedPrice")
	}
	if rec.StrategyUsed != StrategyBuyBoxFirst {
		t.Fatalf("StrategyUsed=%q, want buy_box_first (default)", rec.StrategyUsed)
	}
}

func TestGenerateRecommendation_UsesConfiguredStrategy(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingStrategy{
		SkuID: nil, StrategyType: StrategyMatch, Active: true,
	})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{
		SkuID: 1, CompetitorName: "comp_a", Price: decimal.NewFromInt(90),
		CapturedAt: time.Now(),
	})

	rec, err := svc.GenerateRecommendation(ctx, GenerateRecommendationInput{
		SkuID: 1,
	})
	if err != nil {
		t.Fatalf("GenerateRecommendation: %v", err)
	}
	if rec.StrategyUsed != StrategyMatch {
		t.Fatalf("StrategyUsed=%q, want match (from configured strategy)", rec.StrategyUsed)
	}
	if !rec.RecommendedPrice.Equal(decimal.NewFromInt(90)) {
		t.Fatalf("RecommendedPrice=%v, want 90 (matched)", rec.RecommendedPrice)
	}
}

func TestListRecommendations(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingRecommendation{
		SkuID: 1, CurrentPrice: decimal.NewFromInt(100), RecommendedPrice: decimal.NewFromInt(95),
		StrategyUsed: StrategyBuyBoxFirst, RiskLevel: "medium", CompetitorCount: 2,
	})
	svc.db.WithContext(ctx).Create(&PricingRecommendation{
		SkuID: 1, CurrentPrice: decimal.NewFromInt(100), RecommendedPrice: decimal.NewFromInt(85),
		StrategyUsed: StrategyMatch, RiskLevel: "medium", CompetitorCount: 3,
	})
	svc.db.WithContext(ctx).Create(&PricingRecommendation{
		SkuID: 2, CurrentPrice: decimal.NewFromInt(200), RecommendedPrice: decimal.NewFromInt(200),
		StrategyUsed: StrategyProfitPriority, RiskLevel: "low", CompetitorCount: 1,
	})

	items, total, err := svc.ListRecommendations(ctx, 1, 10, 1)
	if err != nil {
		t.Fatalf("ListRecommendations: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
}

func TestApplyRecommendation(t *testing.T) {
	svc := newTestDBWithEngine(t)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingRecommendation{
		SkuID: 1, CurrentPrice: decimal.NewFromInt(100), RecommendedPrice: decimal.NewFromInt(95),
		StrategyUsed: StrategyBuyBoxFirst, RiskLevel: "low",
	})

	if err := svc.ApplyRecommendation(ctx, 1); err != nil {
		t.Fatalf("ApplyRecommendation: %v", err)
	}

	var rec PricingRecommendation
	svc.db.WithContext(ctx).First(&rec, 1)
	if !rec.Applied {
		t.Fatal("expected Applied=true after ApplyRecommendation")
	}
	if rec.AppliedAt == nil {
		t.Fatal("expected non-nil AppliedAt")
	}
}

// ---------------------------------------------------------------------------
// Handler tests -- Competitor Prices
// ---------------------------------------------------------------------------

func TestHandler_CreateCompetitorPrice(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)

	body := `{"sku_id":1,"competitor_name":"seller_x","price":99.99,"platform":"ozon"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/competitor-prices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	verifyCode(t, w.Body.Bytes(), 0)

	var count int64
	svc.db.Model(&CompetitorPrice{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 competitor price, got %d", count)
	}
}

func TestHandler_CreateCompetitorPrice_Invalid(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)

	body := `{"sku_id":1}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/competitor-prices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
}

func TestHandler_ListCompetitorPrices(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "a", Price: decimal.NewFromInt(100), CapturedAt: time.Now()})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "b", Price: decimal.NewFromInt(200), CapturedAt: time.Now()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/competitor-prices?page=1&size=10&sku_id=1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)

	var pageResp struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pageResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pageResp.Total != 2 {
		t.Fatalf("Total=%d, want 2", pageResp.Total)
	}
}

func TestHandler_GetCompetitorPrice(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "a", Price: decimal.NewFromInt(100), CapturedAt: time.Now()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/competitor-prices/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

func TestHandler_DeleteCompetitorPrice(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&CompetitorPrice{SkuID: 1, CompetitorName: "a", Price: decimal.NewFromInt(100), CapturedAt: time.Now()})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/competitor-prices/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

// ---------------------------------------------------------------------------
// Handler tests -- Pricing Strategies
// ---------------------------------------------------------------------------

func TestHandler_SavePricingStrategy(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)

	body := `{"strategy_type":"buy_box_first","parameters":{"buy_box_discount":0.03}}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/pricing-strategies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

func TestHandler_UpdatePricingStrategy(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingStrategy{
		SkuID: nil, StrategyType: StrategyMatch, Active: true,
	})

	body := `{"strategy_type":"buy_box_first","active":false}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/pricing-strategies/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	verifyCode(t, w.Body.Bytes(), 0)

	var updated PricingStrategy
	svc.db.First(&updated, 1)
	if updated.StrategyType != StrategyBuyBoxFirst {
		t.Fatalf("StrategyType=%q, want buy_box_first", updated.StrategyType)
	}
	if updated.Active != false {
		t.Fatal("expected Active=false")
	}
}

func TestHandler_ListPricingStrategies(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingStrategy{StrategyType: StrategyBuyBoxFirst, Active: true})
	svc.db.WithContext(ctx).Create(&PricingStrategy{StrategyType: StrategyMatch, Active: true})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/pricing-strategies?page=1&size=10", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

func TestHandler_DeletePricingStrategy(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingStrategy{StrategyType: StrategyBuyBoxFirst, Active: true})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/pricing-strategies/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

// ---------------------------------------------------------------------------
// Handler tests -- Pricing Recommendations
// ---------------------------------------------------------------------------

func TestHandler_GenerateRecommendation(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&Price{
		SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1,
	})
	svc.db.WithContext(ctx).Create(&CompetitorPrice{
		SkuID: 1, CompetitorName: "comp_a", Price: decimal.NewFromInt(90),
		CapturedAt: time.Now(),
	})

	body := `{"sku_id":1,"strategy_type":"match"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/pricing-recommendations/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

func TestHandler_GenerateRecommendation_InvalidBody(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)

	body := `{"strategy_type":"match"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/pricing-recommendations/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
}

func TestHandler_ListRecommendations(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingRecommendation{
		SkuID: 1, CurrentPrice: decimal.NewFromInt(100), RecommendedPrice: decimal.NewFromInt(95),
		StrategyUsed: StrategyBuyBoxFirst, RiskLevel: "medium",
	})
	svc.db.WithContext(ctx).Create(&PricingRecommendation{
		SkuID: 1, CurrentPrice: decimal.NewFromInt(100), RecommendedPrice: decimal.NewFromInt(85),
		StrategyUsed: StrategyMatch, RiskLevel: "medium",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/pricing-recommendations?page=1&size=10&sku_id=1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), 0)
}

func TestHandler_ApplyRecommendation(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)
	ctx := context.Background()

	svc.db.WithContext(ctx).Create(&PricingRecommendation{
		SkuID: 1, CurrentPrice: decimal.NewFromInt(100), RecommendedPrice: decimal.NewFromInt(95),
		StrategyUsed: StrategyBuyBoxFirst, RiskLevel: "low",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/pricing-recommendations/1/apply", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	verifyCode(t, w.Body.Bytes(), 0)

	var rec PricingRecommendation
	svc.db.First(&rec, 1)
	if !rec.Applied {
		t.Fatal("expected recommendation marked as applied")
	}
}

func TestHandler_ApplyRecommendation_InvalidID(t *testing.T) {
	svc := newTestDBWithEngine(t)
	router := setupPriceRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/pricing-recommendations/abc/apply", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP status=%d, want 400", w.Code)
	}
	verifyCode(t, w.Body.Bytes(), http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

func int64Ptr(n int64) *int64 {
	return &n
}
