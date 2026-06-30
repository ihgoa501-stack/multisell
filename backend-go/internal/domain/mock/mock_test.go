package mock

import (
	"strings"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func newTestDB(t *testing.T) *MockOrder {
	t.Helper()
	return &MockOrder{}
}

func TestService_SeedMockData(t *testing.T) {
	db := dbtest.NewDB(t, &MockOrder{}, &MockSettlement{}, &MockSyncStatus{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.SeedMockData()
	if err != nil {
		t.Fatalf("SeedMockData failed: %v", err)
	}

	// Verify orders were seeded.
	var orderCount int64
	db.Model(&MockOrder{}).Count(&orderCount)
	if orderCount != 30 {
		t.Errorf("expected 30 mock orders, got %d", orderCount)
	}

	// Verify settlements were seeded.
	var settlementCount int64
	db.Model(&MockSettlement{}).Count(&settlementCount)
	if settlementCount != 9 {
		t.Errorf("expected 9 mock settlements (3 periods * 3 platforms), got %d", settlementCount)
	}

	// Verify sync statuses were seeded (3 platforms * 3 types + 1 failed = 10)
	var syncCount int64
	db.Model(&MockSyncStatus{}).Count(&syncCount)
	if syncCount < 9 {
		t.Errorf("expected at least 9 sync statuses, got %d", syncCount)
	}
}

func TestService_SeedMockData_Idempotent(t *testing.T) {
	db := dbtest.NewDB(t, &MockOrder{}, &MockSettlement{}, &MockSyncStatus{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Seed twice.
	svc.SeedMockData()
	svc.SeedMockData()

	// Second seed should not create duplicates.
	var count int64
	db.Model(&MockOrder{}).Count(&count)
	if count > 30 {
		t.Errorf("expected at most 30 orders (idempotent), got %d", count)
	}
}

func TestService_ListOrders_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &MockOrder{})
	svc := NewService(db, dbtest.NewLogger(t))

	items, total, err := svc.ListOrders(1, 10)
	if err != nil {
		t.Fatalf("ListOrders failed: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if items == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestService_ListOrders_Pagination(t *testing.T) {
	db := dbtest.NewDB(t, &MockOrder{})
	svc := NewService(db, dbtest.NewLogger(t))

	for i := 0; i < 5; i++ {
		db.Create(&MockOrder{
			PlatformID:  int64(i%3 + 1),
			OrderNo:    "ORD-" + dbtest.IToA(int64(i)),
			ProductName: "Product",
			Quantity:   1,
			TotalAmount: 10.0,
			Status:     "pending",
		})
	}

	items, total, err := svc.ListOrders(1, 2)
	if err != nil {
		t.Fatalf("ListOrders failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestService_ListSettlements_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &MockSettlement{})
	svc := NewService(db, dbtest.NewLogger(t))

	items, total, err := svc.ListSettlements(1, 10)
	if err != nil {
		t.Fatalf("ListSettlements failed: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if items == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestService_ListSettlements_Pagination(t *testing.T) {
	db := dbtest.NewDB(t, &MockSettlement{})
	svc := NewService(db, dbtest.NewLogger(t))

	for i := 0; i < 3; i++ {
		db.Create(&MockSettlement{
			PlatformID:   1,
			Period:       "2026-06",
			TotalRevenue: 1000.0,
			TotalFee:     100.0,
			NetAmount:    900.0,
			OrderCount:   50,
		})
	}

	items, total, err := svc.ListSettlements(1, 10)
	if err != nil {
		t.Fatalf("ListSettlements failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestService_GetSyncStatuses_Empty(t *testing.T) {
	db := dbtest.NewDB(t, &MockSyncStatus{})
	svc := NewService(db, dbtest.NewLogger(t))

	items, err := svc.GetSyncStatuses()
	if err != nil {
		// GetSyncStatuses uses PG-specific DISTINCT ON which SQLite doesn't support.
		if strings.Contains(err.Error(), "syntax error") || strings.Contains(err.Error(), "near") {
			t.Skip("GetSyncStatuses requires PG-specific DISTINCT ON:", err)
		}
		t.Fatalf("GetSyncStatuses failed: %v", err)
	}
	if items == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestService_GetSyncStatuses_WithData(t *testing.T) {
	db := dbtest.NewDB(t, &MockSyncStatus{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create multiple sync status entries for same platform+type.
	db.Create(&MockSyncStatus{PlatformID: 1, PlatformName: "Ozon", SyncType: "orders", Status: "success", RecordsSynced: 100, LastSyncAt: nil})
	db.Create(&MockSyncStatus{PlatformID: 1, PlatformName: "Ozon", SyncType: "orders", Status: "failed", RecordsSynced: 0, LastSyncAt: nil})
	db.Create(&MockSyncStatus{PlatformID: 1, PlatformName: "Ozon", SyncType: "products", Status: "success", RecordsSynced: 50, LastSyncAt: nil})

	items, err := svc.GetSyncStatuses()
	if err != nil {
		if strings.Contains(err.Error(), "syntax error") || strings.Contains(err.Error(), "near") {
			t.Skip("GetSyncStatuses requires PG-specific DISTINCT ON:", err)
		}
		t.Fatalf("GetSyncStatuses failed: %v", err)
	}
	// Should deduplicate by (platform_id, sync_type) keeping the latest (highest ID).
	if len(items) != 2 {
		t.Errorf("expected 2 unique statuses, got %d", len(items))
	}
}

func TestRandString(t *testing.T) {
	s1 := randString(8)
	s2 := randString(8)
	if len(s1) != 8 {
		t.Errorf("expected length 8, got %d", len(s1))
	}
	// Very unlikely to collide
	if s1 == s2 {
		t.Errorf("unlikely random collision: %s", s1)
	}
}
