package price

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Price{}, &PriceChangeLog{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	db := newTestDB(t)
	return NewService(db, dbtest.NewLogger(t))
}

// ── Price CRUD ──────────────────────────────────────────────────────

func TestPrice_GetPriceByID(t *testing.T) {
	svc := newService(t)

	p := &Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1}
	if err := svc.SetPrice(context.Background(), p, "admin"); err != nil {
		t.Fatalf("SetPrice failed: %v", err)
	}

	got, err := svc.GetPriceByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetPriceByID failed: %v", err)
	}
	if got.SkuID != 1 {
		t.Fatalf("GetPriceByID SkuID = %d, want 1", got.SkuID)
	}
	if got.PriceType != "sale_price" {
		t.Fatalf("GetPriceByID PriceType = %q, want %q", got.PriceType, "sale_price")
	}
}

func TestPrice_GetPriceByID_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetPriceByID(context.Background(), 999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestPrice_UpdatePrice(t *testing.T) {
	svc := newService(t)

	p := &Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1}
	if err := svc.SetPrice(context.Background(), p, "admin"); err != nil {
		t.Fatalf("SetPrice failed: %v", err)
	}

	p.Price = decimal.NewFromInt(200)
	if err := svc.UpdatePrice(context.Background(), p); err != nil {
		t.Fatalf("UpdatePrice failed: %v", err)
	}

	got, _ := svc.GetPriceByID(context.Background(), p.ID)
	if !got.Price.Equal(decimal.NewFromInt(200)) {
		t.Fatalf("after UpdatePrice Price = %s, want 200", got.Price)
	}
}

func TestPrice_DeletePrice(t *testing.T) {
	svc := newService(t)

	p := &Price{SkuID: 1, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1}
	if err := svc.SetPrice(context.Background(), p, "admin"); err != nil {
		t.Fatalf("SetPrice failed: %v", err)
	}

	if err := svc.DeletePrice(context.Background(), p.ID); err != nil {
		t.Fatalf("DeletePrice failed: %v", err)
	}

	if _, err := svc.GetPriceByID(context.Background(), p.ID); err == nil {
		t.Fatal("expected error after DeletePrice")
	}
}

// ── SetPrice ────────────────────────────────────────────────────────

func TestPrice_SetPrice_CreatesChangeLog(t *testing.T) {
	svc := newService(t)

	p := &Price{SkuID: 10, PriceType: "sale_price", Price: decimal.NewFromInt(99)}
	if err := svc.SetPrice(context.Background(), p, "alice"); err != nil {
		t.Fatalf("SetPrice failed: %v", err)
	}

	logs, total, err := svc.ListChangeLogs(context.Background(), 10, 1, 20)
	if err != nil {
		t.Fatalf("ListChangeLogs failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("ChangeLog total = %d, want 1", total)
	}
	if logs[0].Operator != "alice" {
		t.Fatalf("ChangeLog Operator = %q, want %q", logs[0].Operator, "alice")
	}
	if logs[0].ChangeType != "manual" {
		t.Fatalf("ChangeLog ChangeType = %q, want %q", logs[0].ChangeType, "manual")
	}
}

func TestPrice_SetPrice_EmptyPriceType(t *testing.T) {
	svc := newService(t)

	p := &Price{SkuID: 1, PriceType: "", Price: decimal.NewFromInt(100)}
	if err := svc.SetPrice(context.Background(), p, "admin"); err == nil {
		t.Fatal("expected error for empty PriceType")
	}
}

func TestPrice_SetPrice_WhitespacePriceType(t *testing.T) {
	svc := newService(t)

	p := &Price{SkuID: 1, PriceType: "   ", Price: decimal.NewFromInt(100)}
	if err := svc.SetPrice(context.Background(), p, "admin"); err == nil {
		t.Fatal("expected error for whitespace-only PriceType")
	}
}

func TestPrice_SetPrice_UpdatesExistingBySkuAndType(t *testing.T) {
	svc := newService(t)

	p1 := &Price{SkuID: 20, PriceType: "sale_price", Price: decimal.NewFromInt(100)}
	if err := svc.SetPrice(context.Background(), p1, "admin"); err != nil {
		t.Fatalf("first SetPrice failed: %v", err)
	}
	firstID := p1.ID

	p2 := &Price{SkuID: 20, PriceType: "sale_price", Price: decimal.NewFromInt(150)}
	if err := svc.SetPrice(context.Background(), p2, "admin"); err != nil {
		t.Fatalf("second SetPrice failed: %v", err)
	}

	if p2.ID != firstID {
		t.Fatalf("expected same ID on update, got %d want %d", p2.ID, firstID)
	}

	got, _ := svc.GetPriceByID(context.Background(), p2.ID)
	if !got.Price.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("Price = %s, want 150", got.Price)
	}

	logs, total, _ := svc.ListChangeLogs(context.Background(), 20, 1, 20)
	if total != 2 {
		t.Fatalf("expected 2 change logs, got %d", total)
	}
	if logs[0].OldPrice == nil {
		t.Fatal("expected non-nil OldPrice in second log")
	}
}

// ── ListPrices ──────────────────────────────────────────────────────

func TestPrice_ListPrices_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 25; i++ {
		p := &Price{SkuID: int64(i), PriceType: "sale_price", Price: decimal.NewFromInt(int64(i * 10))}
		if err := svc.SetPrice(context.Background(), p, "admin"); err != nil {
			t.Fatalf("SetPrice %d failed: %v", i, err)
		}
	}

	items, total, err := svc.ListPrices(context.Background(), 1, 10, 0, "")
	if err != nil {
		t.Fatalf("ListPrices failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("ListPrices total = %d, want 25", total)
	}
	if len(items) != 10 {
		t.Fatalf("ListPrices returned %d items, want 10", len(items))
	}
}

func TestPrice_ListPrices_FilterBySkuID(t *testing.T) {
	svc := newService(t)

	_ = svc.SetPrice(context.Background(), &Price{SkuID: 50, PriceType: "sale_price", Price: decimal.NewFromInt(10)}, "admin")
	_ = svc.SetPrice(context.Background(), &Price{SkuID: 50, PriceType: "cost_price", Price: decimal.NewFromInt(5)}, "admin")
	_ = svc.SetPrice(context.Background(), &Price{SkuID: 51, PriceType: "sale_price", Price: decimal.NewFromInt(20)}, "admin")

	items, total, err := svc.ListPrices(context.Background(), 1, 20, 50, "")
	if err != nil {
		t.Fatalf("ListPrices with skuID filter failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

func TestPrice_ListPrices_FilterByPriceType(t *testing.T) {
	svc := newService(t)

	_ = svc.SetPrice(context.Background(), &Price{SkuID: 60, PriceType: "sale_price", Price: decimal.NewFromInt(100)}, "admin")
	_ = svc.SetPrice(context.Background(), &Price{SkuID: 61, PriceType: "cost_price", Price: decimal.NewFromInt(50)}, "admin")

	items, total, err := svc.ListPrices(context.Background(), 1, 20, 0, "cost_price")
	if err != nil {
		t.Fatalf("ListPrices with priceType filter failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if items[0].PriceType != "cost_price" {
		t.Fatalf("PriceType = %q, want %q", items[0].PriceType, "cost_price")
	}
}

// ── ListPricesBySKU ─────────────────────────────────────────────────

func TestPrice_ListPricesBySKU_OnlyActive(t *testing.T) {
	svc := newService(t)

	_ = svc.SetPrice(context.Background(), &Price{SkuID: 70, PriceType: "sale_price", Price: decimal.NewFromInt(100), Status: 1}, "admin")
	_ = svc.SetPrice(context.Background(), &Price{SkuID: 70, PriceType: "cost_price", Price: decimal.NewFromInt(50), Status: 1}, "admin")

	// create inactive price
	inactive := &Price{SkuID: 70, PriceType: "wholesale", Price: decimal.NewFromInt(30)}
	_ = svc.db.Create(inactive).Error
	// SQLite + GORM zero-value handling: explicit UPDATE to set inactive status
	svc.db.Model(&Price{}).Where("id = ?", inactive.ID).Update("status", 0)

	items, err := svc.ListPricesBySKU(context.Background(), 70)
	if err != nil {
		t.Fatalf("ListPricesBySKU failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2 (only active)", len(items))
	}
	for _, item := range items {
		if item.Status != 1 {
			t.Fatalf("expected only active prices, got status=%d", item.Status)
		}
	}
}

func TestPrice_ListPricesBySKU_Empty(t *testing.T) {
	svc := newService(t)

	items, err := svc.ListPricesBySKU(context.Background(), 999)
	if err != nil {
		t.Fatalf("ListPricesBySKU failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

// ── GetCurrentPrice ─────────────────────────────────────────────────

func TestPrice_GetCurrentPrice_ActiveWindow(t *testing.T) {
	svc := newService(t)

	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	p := &Price{
		SkuID:     80,
		PriceType: "sale_price",
		Price:     decimal.NewFromInt(99),
		StartTime: &past,
		EndTime:   &future,
		Status:    1,
	}
	_ = svc.db.Save(p).Error

	got, err := svc.GetCurrentPrice(context.Background(), 80)
	if err != nil {
		t.Fatalf("GetCurrentPrice failed: %v", err)
	}
	if !got.Price.Equal(decimal.NewFromInt(99)) {
		t.Fatalf("Price = %s, want 99", got.Price)
	}
}

func TestPrice_GetCurrentPrice_ExpiredWindow(t *testing.T) {
	svc := newService(t)

	past1 := time.Now().Add(-2 * time.Hour)
	past2 := time.Now().Add(-1 * time.Hour)

	p := &Price{
		SkuID:     81,
		PriceType: "sale_price",
		Price:     decimal.NewFromInt(50),
		StartTime: &past1,
		EndTime:   &past2,
		Status:    1,
	}
	_ = svc.db.Save(p).Error

	_, err := svc.GetCurrentPrice(context.Background(), 81)
	if err == nil {
		t.Fatal("expected error for expired price window")
	}
}

func TestPrice_GetCurrentPrice_FutureWindow(t *testing.T) {
	svc := newService(t)

	future1 := time.Now().Add(1 * time.Hour)
	future2 := time.Now().Add(2 * time.Hour)

	p := &Price{
		SkuID:     82,
		PriceType: "sale_price",
		Price:     decimal.NewFromInt(75),
		StartTime: &future1,
		EndTime:   &future2,
		Status:    1,
	}
	_ = svc.db.Save(p).Error

	_, err := svc.GetCurrentPrice(context.Background(), 82)
	if err == nil {
		t.Fatal("expected error for future price window")
	}
}

func TestPrice_GetCurrentPrice_NilTimeRange(t *testing.T) {
	svc := newService(t)

	p := &Price{
		SkuID:     83,
		PriceType: "sale_price",
		Price:     decimal.NewFromInt(110),
		Status:    1,
	}
	_ = svc.db.Save(p).Error

	got, err := svc.GetCurrentPrice(context.Background(), 83)
	if err != nil {
		t.Fatalf("GetCurrentPrice with nil times failed: %v", err)
	}
	if !got.Price.Equal(decimal.NewFromInt(110)) {
		t.Fatalf("Price = %s, want 110", got.Price)
	}
}

func TestPrice_GetCurrentPrice_WrongType(t *testing.T) {
	svc := newService(t)

	p := &Price{
		SkuID:     84,
		PriceType: "cost_price",
		Price:     decimal.NewFromInt(30),
		Status:    1,
	}
	_ = svc.db.Save(p).Error

	_, err := svc.GetCurrentPrice(context.Background(), 84)
	if err == nil {
		t.Fatal("expected error for wrong price type")
	}
}

// ── ListChangeLogs ──────────────────────────────────────────────────

func TestPrice_ListChangeLogs_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 15; i++ {
		p := &Price{SkuID: 90, PriceType: "sale_price", Price: decimal.NewFromInt(int64(i * 10))}
		if err := svc.SetPrice(context.Background(), p, "admin"); err != nil {
			t.Fatalf("SetPrice %d failed: %v", i, err)
		}
	}

	items, total, err := svc.ListChangeLogs(context.Background(), 90, 1, 10)
	if err != nil {
		t.Fatalf("ListChangeLogs failed: %v", err)
	}
	if total != 15 {
		t.Fatalf("total = %d, want 15", total)
	}
	if len(items) != 10 {
		t.Fatalf("got %d items, want 10", len(items))
	}
}

func TestPrice_ListChangeLogs_Empty(t *testing.T) {
	svc := newService(t)

	items, total, err := svc.ListChangeLogs(context.Background(), 999, 1, 20)
	if err != nil {
		t.Fatalf("ListChangeLogs failed: %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
}
