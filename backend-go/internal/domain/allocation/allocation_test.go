package allocation

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Warehouse{}, &AllocationRule{}, &CostAllocationBatch{}, &CostAllocationItem{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func TestWarehouse_Create(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "Main Warehouse", Code: "WH-001"}
	if err := svc.CreateWarehouse(context.Background(), w); err != nil {
		t.Fatalf("CreateWarehouse failed: %v", err)
	}
	if w.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestWarehouse_Create_EmptyName(t *testing.T) {
	svc := newService(t)

	if err := svc.CreateWarehouse(context.Background(), &Warehouse{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestWarehouse_GetByID(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "Findable", Code: "WH-002"}
	_ = svc.CreateWarehouse(context.Background(), w)

	got, err := svc.GetWarehouseByID(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("GetWarehouseByID failed: %v", err)
	}
	if got.Name != "Findable" {
		t.Fatalf("Name=%q, want Findable", got.Name)
	}
}

func TestWarehouse_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetWarehouseByID(context.Background(), 999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestWarehouse_Update(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "Old Name", Code: "WH-003"}
	_ = svc.CreateWarehouse(context.Background(), w)

	w.Name = "Updated Name"
	if err := svc.UpdateWarehouse(context.Background(), w); err != nil {
		t.Fatalf("UpdateWarehouse failed: %v", err)
	}

	got, _ := svc.GetWarehouseByID(context.Background(), w.ID)
	if got.Name != "Updated Name" {
		t.Fatalf("Name=%q, want Updated Name", got.Name)
	}
}

func TestWarehouse_Delete(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "To Delete", Code: "WH-004"}
	_ = svc.CreateWarehouse(context.Background(), w)

	if err := svc.DeleteWarehouse(context.Background(), w.ID); err != nil {
		t.Fatalf("DeleteWarehouse failed: %v", err)
	}

	if _, err := svc.GetWarehouseByID(context.Background(), w.ID); err == nil {
		t.Fatal("expected error after DeleteWarehouse")
	}
}

func TestWarehouse_ListWarehouses(t *testing.T) {
	svc := newService(t)

	_ = svc.CreateWarehouse(context.Background(), &Warehouse{Name: "Alpha", Code: "WH-A"})
	_ = svc.CreateWarehouse(context.Background(), &Warehouse{Name: "Beta", Code: "WH-B"})

	items, total, err := svc.ListWarehouses(context.Background(), 1, 10, "")
	if err != nil {
		t.Fatalf("ListWarehouses failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
}

func TestWarehouse_ListWarehouses_Defaults(t *testing.T) {
	svc := newService(t)

	_ = svc.CreateWarehouse(context.Background(), &Warehouse{Name: "WH", Code: "WH-D"})

	// page=0 should default to page 1
	items, total, err := svc.ListWarehouses(context.Background(), 0, 0, "")
	if err != nil {
		t.Fatalf("ListWarehouses with defaults failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items)=%d, want 1", len(items))
	}
}

func TestRule_Create(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "WH", Code: "WH-R1"}
	_ = svc.CreateWarehouse(context.Background(), w)

	r := &AllocationRule{Name: "Rule 1", RuleType: "percentage", WarehouseID: w.ID}
	if err := svc.CreateRule(context.Background(), r); err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestRule_Create_EmptyName(t *testing.T) {
	svc := newService(t)

	if err := svc.CreateRule(context.Background(), &AllocationRule{Name: "", RuleType: "percentage"}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestRule_Create_EmptyRuleType(t *testing.T) {
	svc := newService(t)

	if err := svc.CreateRule(context.Background(), &AllocationRule{Name: "Rule", RuleType: ""}); err == nil {
		t.Fatal("expected error for empty rule type")
	}
}

func TestRule_GetByID(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "WH", Code: "WH-R2"}
	_ = svc.CreateWarehouse(context.Background(), w)

	r := &AllocationRule{Name: "Findable", RuleType: "fixed", WarehouseID: w.ID}
	_ = svc.CreateRule(context.Background(), r)

	got, err := svc.GetRuleByID(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetRuleByID failed: %v", err)
	}
	if got.Name != "Findable" {
		t.Fatalf("Name=%q, want Findable", got.Name)
	}
}

func TestRule_Update(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "WH", Code: "WH-R3"}
	_ = svc.CreateWarehouse(context.Background(), w)

	r := &AllocationRule{Name: "Old", RuleType: "percentage", WarehouseID: w.ID}
	_ = svc.CreateRule(context.Background(), r)

	r.Name = "Updated"
	if err := svc.UpdateRule(context.Background(), r); err != nil {
		t.Fatalf("UpdateRule failed: %v", err)
	}

	got, _ := svc.GetRuleByID(context.Background(), r.ID)
	if got.Name != "Updated" {
		t.Fatalf("Name=%q, want Updated", got.Name)
	}
}

func TestRule_Delete(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "WH", Code: "WH-R4"}
	_ = svc.CreateWarehouse(context.Background(), w)

	r := &AllocationRule{Name: "To Delete", RuleType: "fixed", WarehouseID: w.ID}
	_ = svc.CreateRule(context.Background(), r)

	if err := svc.DeleteRule(context.Background(), r.ID); err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}

	if _, err := svc.GetRuleByID(context.Background(), r.ID); err == nil {
		t.Fatal("expected error after DeleteRule")
	}
}

func TestRule_ListRules_FilterByWarehouseID(t *testing.T) {
	svc := newService(t)

	w1 := &Warehouse{Name: "WH1", Code: "WH-F1"}
	w2 := &Warehouse{Name: "WH2", Code: "WH-F2"}
	_ = svc.CreateWarehouse(context.Background(), w1)
	_ = svc.CreateWarehouse(context.Background(), w2)

	_ = svc.CreateRule(context.Background(), &AllocationRule{Name: "R1", RuleType: "pct", WarehouseID: w1.ID})
	_ = svc.CreateRule(context.Background(), &AllocationRule{Name: "R2", RuleType: "pct", WarehouseID: w2.ID})
	_ = svc.CreateRule(context.Background(), &AllocationRule{Name: "R3", RuleType: "fixed", WarehouseID: w1.ID})

	items, total, err := svc.ListRules(context.Background(), 1, 10, w1.ID)
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
}

func TestRule_ListRules_NoFilter(t *testing.T) {
	svc := newService(t)

	w := &Warehouse{Name: "WH", Code: "WH-NF"}
	_ = svc.CreateWarehouse(context.Background(), w)

	_ = svc.CreateRule(context.Background(), &AllocationRule{Name: "R1", RuleType: "pct", WarehouseID: w.ID})
	_ = svc.CreateRule(context.Background(), &AllocationRule{Name: "R2", RuleType: "fixed", WarehouseID: w.ID})

	items, total, err := svc.ListRules(context.Background(), 1, 10, 0)
	if err != nil {
		t.Fatalf("ListRules no filter failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
}

func TestBatch_Create(t *testing.T) {
	svc := newService(t)

	b := &CostAllocationBatch{AllocationType: "shipping", AllocationMethod: "weight"}
	if err := svc.CreateBatch(context.Background(), b); err != nil {
		t.Fatalf("CreateBatch failed: %v", err)
	}
	if b.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestBatch_Create_EmptyFields(t *testing.T) {
	svc := newService(t)

	if err := svc.CreateBatch(context.Background(), &CostAllocationBatch{AllocationType: "", AllocationMethod: "weight"}); err == nil {
		t.Fatal("expected error for empty allocation type")
	}
	if err := svc.CreateBatch(context.Background(), &CostAllocationBatch{AllocationType: "shipping", AllocationMethod: ""}); err == nil {
		t.Fatal("expected error for empty allocation method")
	}
}

func TestBatch_GetByID(t *testing.T) {
	svc := newService(t)

	b := &CostAllocationBatch{AllocationType: "shipping", AllocationMethod: "volume"}
	_ = svc.CreateBatch(context.Background(), b)

	got, err := svc.GetBatchByID(context.Background(), b.ID)
	if err != nil {
		t.Fatalf("GetBatchByID failed: %v", err)
	}
	if got.AllocationType != "shipping" {
		t.Fatalf("AllocationType=%q, want shipping", got.AllocationType)
	}
}

func TestBatch_ListBatches_FilterByType(t *testing.T) {
	svc := newService(t)

	_ = svc.CreateBatch(context.Background(), &CostAllocationBatch{AllocationType: "shipping", AllocationMethod: "weight"})
	_ = svc.CreateBatch(context.Background(), &CostAllocationBatch{AllocationType: "platform_fee", AllocationMethod: "amount"})
	_ = svc.CreateBatch(context.Background(), &CostAllocationBatch{AllocationType: "shipping", AllocationMethod: "volume"})

	items, total, err := svc.ListBatches(context.Background(), 1, 10, "shipping")
	if err != nil {
		t.Fatalf("ListBatches failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
}
