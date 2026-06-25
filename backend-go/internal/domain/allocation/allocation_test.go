package allocation

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/shopspring/decimal"
)

func TestService_Warehouse_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Warehouse{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.CreateWarehouse(context.Background(), &Warehouse{
		Name: "上海仓",
		Code: "SH01",
	})
	if err != nil {
		t.Fatalf("CreateWarehouse: %v", err)
	}

	items, total, err := svc.ListWarehouses(context.Background(), 1, 10, "")
	if err != nil {
		t.Fatalf("ListWarehouses: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d", len(items))
	}
	if items[0].Name != "上海仓" {
		t.Fatalf("Name = %s", items[0].Name)
	}

	got, err := svc.GetWarehouseByID(context.Background(), items[0].ID)
	if err != nil {
		t.Fatalf("GetWarehouseByID: %v", err)
	}
	if got.Code != "SH01" {
		t.Fatalf("Code = %s", got.Code)
	}
}

func TestService_Warehouse_UpdateAndDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Warehouse{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreateWarehouse(context.Background(), &Warehouse{Name: "广州仓", Code: "GZ01"})
	items, _, _ := svc.ListWarehouses(context.Background(), 1, 10, "")
	w := items[0]
	w.Name = "广州总仓"
	err := svc.UpdateWarehouse(context.Background(), &w)
	if err != nil {
		t.Fatalf("UpdateWarehouse: %v", err)
	}

	got, _ := svc.GetWarehouseByID(context.Background(), w.ID)
	if got.Name != "广州总仓" {
		t.Fatalf("Name = %s", got.Name)
	}

	err = svc.DeleteWarehouse(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("DeleteWarehouse: %v", err)
	}
	_, err = svc.GetWarehouseByID(context.Background(), w.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Warehouse_ListSearch(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Warehouse{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreateWarehouse(context.Background(), &Warehouse{Name: "上海仓", Code: "SH01"})
	svc.CreateWarehouse(context.Background(), &Warehouse{Name: "广州仓", Code: "GZ01"})

	// List without search (ILIKE is PG-specific, so skip filtered search)
	items, total, err := svc.ListWarehouses(context.Background(), 1, 10, "")
	if err != nil {
		t.Fatalf("ListWarehouses: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d (expected 2)", total)
	}
	_ = items

	// Filter by name search is not testable in SQLite (ILIKE is PG-specific)
}

// ── AllocationRule ──

func TestService_Rule_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &AllocationRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.CreateRule(context.Background(), &AllocationRule{
		Name:        "主仓优先",
		RuleType:    "priority",
		WarehouseID: 1,
		Priority:    1,
	})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	items, total, err := svc.ListRules(context.Background(), 1, 10, 1)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d", len(items))
	}

	got, err := svc.GetRuleByID(context.Background(), items[0].ID)
	if err != nil {
		t.Fatalf("GetRuleByID: %v", err)
	}
	if got.Name != "主仓优先" {
		t.Fatalf("Name = %s", got.Name)
	}
}

func TestService_Rule_UpdateAndDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &AllocationRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreateRule(context.Background(), &AllocationRule{
		Name: "备份仓", RuleType: "backup", WarehouseID: 2, Priority: 5,
	})
	items, _, _ := svc.ListRules(context.Background(), 1, 10, 0)
	r := items[0]
	r.Priority = 1
	err := svc.UpdateRule(context.Background(), &r)
	if err != nil {
		t.Fatalf("UpdateRule: %v", err)
	}

	got, _ := svc.GetRuleByID(context.Background(), r.ID)
	if got.Priority != 1 {
		t.Fatalf("Priority = %d", got.Priority)
	}

	err = svc.DeleteRule(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}
	_, err = svc.GetRuleByID(context.Background(), r.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

// ── CostAllocationBatch ──

func TestService_Batch_CreateAndList(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.CreateBatch(context.Background(), &CostAllocationBatch{
		AllocationType:   "shipping",
		AllocationMethod: "weight",
		TotalAmount:      decimal.NewFromFloat(5000),
		Currency:         "CNY",
	})
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	items, total, err := svc.ListBatches(context.Background(), 1, 10, "")
	if err != nil {
		t.Fatalf("ListBatches: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	got, err := svc.GetBatchByID(context.Background(), items[0].ID)
	if err != nil {
		t.Fatalf("GetBatchByID: %v", err)
	}
	if got.AllocationType != "shipping" {
		t.Fatalf("AllocationType = %s", got.AllocationType)
	}
}

func TestService_Batch_ListItems(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreateBatch(context.Background(), &CostAllocationBatch{
		AllocationType: "shipping", AllocationMethod: "weight", TotalAmount: decimal.NewFromFloat(1000),
	})
	batches, _, _ := svc.ListBatches(context.Background(), 1, 10, "")
	batchID := batches[0].ID

	// Create items directly
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 1, SkuID: 100, Quantity: 2})
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 2, SkuID: 101, Quantity: 3})

	items, total, err := svc.ListBatchItems(context.Background(), batchID, 1, 10)
	if err != nil {
		t.Fatalf("ListBatchItems: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	_ = items
}
