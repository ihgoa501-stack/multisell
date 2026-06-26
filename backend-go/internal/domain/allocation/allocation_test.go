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

// ── ComputeAllocation ──

func TestService_ComputeAllocation_weight(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := &CostAllocationBatch{
		AllocationType:   "freight",
		AllocationMethod: "weight",
		TotalAmount:      decimal.NewFromFloat(1000),
	}
	if err := svc.CreateBatch(context.Background(), batch); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	batches, _, _ := svc.ListBatches(context.Background(), 1, 10, "")
	batchID := batches[0].ID

	w1 := decimal.NewFromFloat(10)
	w2 := decimal.NewFromFloat(30)
	w3 := decimal.NewFromFloat(60) // total 100 kg
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 1, WeightKg: &w1})
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 2, WeightKg: &w2})
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 3, WeightKg: &w3})

	if err := svc.ComputeAllocation(context.Background(), batchID); err != nil {
		t.Fatalf("ComputeAllocation: %v", err)
	}

	var items []CostAllocationItem
	db.Where("batch_id = ?", batchID).Order("row_number ASC").Find(&items)
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}

	// 10/100=0.1 → 100; 30/100=0.3 → 300; 60/100=0.6 → 600
	cases := []struct {
		factor float64
		amount float64
	}{
		{0.1, 100},
		{0.3, 300},
		{0.6, 600},
	}
	for i, c := range cases {
		f, _ := items[i].AllocatedAmount.Float64()
		if f != c.amount {
			t.Errorf("item %d allocated_amount = %.2f, want %.2f", i, f, c.amount)
		}
		if items[i].AllocationFactor == nil {
			t.Errorf("item %d allocation_factor is nil", i)
		} else {
			gf, _ := (*items[i].AllocationFactor).Float64()
			if gf != c.factor {
				t.Errorf("item %d allocation_factor = %.4f, want %.4f", i, gf, c.factor)
			}
		}
	}
}

func TestService_ComputeAllocation_volume(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := &CostAllocationBatch{
		AllocationType:   "freight",
		AllocationMethod: "volume",
		TotalAmount:      decimal.NewFromFloat(500),
	}
	svc.CreateBatch(context.Background(), batch)
	batches, _, _ := svc.ListBatches(context.Background(), 1, 10, "")
	batchID := batches[0].ID

	v1 := decimal.NewFromFloat(2)   // 2 m³
	v2 := decimal.NewFromFloat(8)   // 8 m³ → total 10 m³
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 1, VolumeM3: &v1})
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 2, VolumeM3: &v2})

	if err := svc.ComputeAllocation(context.Background(), batchID); err != nil {
		t.Fatalf("ComputeAllocation: %v", err)
	}

	var items []CostAllocationItem
	db.Where("batch_id = ?", batchID).Order("row_number ASC").Find(&items)
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	// 2/10=0.2 → 100; 8/10=0.8 → 400
	cases := []struct {
		factor float64
		amount float64
	}{
		{0.2, 100},
		{0.8, 400},
	}
	for i, c := range cases {
		a, _ := items[i].AllocatedAmount.Float64()
		if a != c.amount {
			t.Errorf("item %d allocated_amount = %.2f, want %.2f", i, a, c.amount)
		}
	}
}

func TestService_ComputeAllocation_value(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := &CostAllocationBatch{
		AllocationType:   "customs",
		AllocationMethod: "value",
		TotalAmount:      decimal.NewFromFloat(200),
	}
	svc.CreateBatch(context.Background(), batch)
	batches, _, _ := svc.ListBatches(context.Background(), 1, 10, "")
	batchID := batches[0].ID

	iv1 := decimal.NewFromFloat(100)
	iv2 := decimal.NewFromFloat(300) // total item value 400
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 1, ItemValue: &iv1})
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 2, ItemValue: &iv2})

	if err := svc.ComputeAllocation(context.Background(), batchID); err != nil {
		t.Fatalf("ComputeAllocation: %v", err)
	}

	var items []CostAllocationItem
	db.Where("batch_id = ?", batchID).Order("row_number ASC").Find(&items)

	// 100/400=0.25 → 50; 300/400=0.75 → 150
	expected := []float64{50, 150}
	for i, exp := range expected {
		a, _ := items[i].AllocatedAmount.Float64()
		if a != exp {
			t.Errorf("item %d allocated_amount = %.2f, want %.2f", i, a, exp)
		}
	}
}

func TestService_ComputeAllocation_equal(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := &CostAllocationBatch{
		AllocationType:   "overhead",
		AllocationMethod: "equal",
		TotalAmount:      decimal.NewFromFloat(600),
	}
	svc.CreateBatch(context.Background(), batch)
	batches, _, _ := svc.ListBatches(context.Background(), 1, 10, "")
	batchID := batches[0].ID

	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 1})
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 2})

	if err := svc.ComputeAllocation(context.Background(), batchID); err != nil {
		t.Fatalf("ComputeAllocation: %v", err)
	}

	var items []CostAllocationItem
	db.Where("batch_id = ?", batchID).Order("row_number ASC").Find(&items)
	if len(items) != 2 {
		t.Fatalf("got %d items", len(items))
	}

	// 600 / 2 = 300 each (exact in decimal arithmetic)
	for i := range items {
		a, _ := items[i].AllocatedAmount.Float64()
		if a != 300.0 {
			t.Errorf("item %d allocated_amount = %.2f, want 300", i, a)
		}
	}
}

func TestService_ComputeAllocation_nilField(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Items with nil WeightKg — should get zero allocation
	batch := &CostAllocationBatch{
		AllocationType:   "freight",
		AllocationMethod: "weight",
		TotalAmount:      decimal.NewFromFloat(500),
	}
	svc.CreateBatch(context.Background(), batch)
	batches, _, _ := svc.ListBatches(context.Background(), 1, 10, "")
	batchID := batches[0].ID

	w1 := decimal.NewFromFloat(10)
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 1, WeightKg: &w1})
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 2}) // nil WeightKg

	if err := svc.ComputeAllocation(context.Background(), batchID); err != nil {
		t.Fatalf("ComputeAllocation: %v", err)
	}

	var items []CostAllocationItem
	db.Where("batch_id = ?", batchID).Order("row_number ASC").Find(&items)

	// Item 1: 10/10=1 → 500
	// Item 2: 0/10=0 → 0
	a0, _ := items[0].AllocatedAmount.Float64()
	a1, _ := items[1].AllocatedAmount.Float64()
	if a0 != 500.0 {
		t.Errorf("item 0 allocated = %.2f, want 500", a0)
	}
	if a1 != 0.0 {
		t.Errorf("item 1 (nil weight) allocated = %.2f, want 0", a1)
	}
}

func TestService_ComputeAllocation_zeroBasis(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := &CostAllocationBatch{
		AllocationType:   "freight",
		AllocationMethod: "weight",
		TotalAmount:      decimal.NewFromFloat(500),
	}
	svc.CreateBatch(context.Background(), batch)
	batches, _, _ := svc.ListBatches(context.Background(), 1, 10, "")
	batchID := batches[0].ID

	// All items have nil WeightKg → basis total is zero
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 1})
	db.Create(&CostAllocationItem{BatchID: batchID, RowNumber: 2})

	err := svc.ComputeAllocation(context.Background(), batchID)
	if err == nil {
		t.Fatal("expected error for zero basis total, got nil")
	}
}

func TestService_ComputeAllocation_noItems(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := &CostAllocationBatch{
		AllocationType:   "freight",
		AllocationMethod: "weight",
		TotalAmount:      decimal.NewFromFloat(500),
	}
	svc.CreateBatch(context.Background(), batch)
	batches, _, _ := svc.ListBatches(context.Background(), 1, 10, "")
	batchID := batches[0].ID

	// No items created
	err := svc.ComputeAllocation(context.Background(), batchID)
	if err == nil {
		t.Fatal("expected error for no items, got nil")
	}
}

func TestService_ComputeAllocation_batchNotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CostAllocationBatch{}, &CostAllocationItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.ComputeAllocation(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for missing batch, got nil")
	}
}

func TestService_AutoAllocate_Percentage(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Warehouse{}, &AllocationRule{}, &inventoryWarehouseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	// Create warehouses
	db.Create(&Warehouse{ID: 1, Name: "上海仓", Code: "SH01"})
	db.Create(&Warehouse{ID: 2, Name: "广州仓", Code: "GZ02"})

	// Create percentage rules: WH1 gets 60%, WH2 gets 40%
	svc.CreateRule(ctx, &AllocationRule{
		Name: "主仓60%", RuleType: "percentage", SkuID: 100, WarehouseID: 1,
		AllocationPct: decimal.NewFromInt(60), Priority: 1,
	})
	svc.CreateRule(ctx, &AllocationRule{
		Name: "备仓40%", RuleType: "percentage", SkuID: 100, WarehouseID: 2,
		AllocationPct: decimal.NewFromInt(40), Priority: 2,
	})

	// Create initial inventory: 50 units in WH1, 50 units in WH2 = 100 total
	db.Create(&inventoryWarehouseRow{
		SkuID: 100, WarehouseID: 1, Quantity: 50, LockedQuantity: 0,
	})
	db.Create(&inventoryWarehouseRow{
		SkuID: 100, WarehouseID: 2, Quantity: 50, LockedQuantity: 0,
	})

	// Run auto-allocate
	if err := svc.AutoAllocate(ctx, 100); err != nil {
		t.Fatalf("AutoAllocate: %v", err)
	}

	// Verify: WH1 gets 60 (60% of 100), WH2 gets 40 (40% of 100)
	var records []inventoryWarehouseRow
	db.Where("sku_id = ?", 100).Order("warehouse_id ASC").Find(&records)

	if len(records) != 2 {
		t.Fatalf("expected 2 inventory records, got %d", len(records))
	}

	for _, r := range records {
		switch r.WarehouseID {
		case 1:
			if r.Quantity != 60 {
				t.Errorf("WH1 expected qty 60, got %d", r.Quantity)
			}
		case 2:
			if r.Quantity != 40 {
				t.Errorf("WH2 expected qty 40, got %d", r.Quantity)
			}
		}
	}
}

func TestService_AutoAllocate_Fixed(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Warehouse{}, &AllocationRule{}, &inventoryWarehouseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	// Create warehouses
	db.Create(&Warehouse{ID: 1, Name: "上海仓", Code: "SH01"})
	db.Create(&Warehouse{ID: 2, Name: "广州仓", Code: "GZ02"})

	// Create fixed rules: WH1 gets 30 units, WH2 gets 20 units
	svc.CreateRule(ctx, &AllocationRule{
		Name: "主仓30", RuleType: "fixed", SkuID: 100, WarehouseID: 1,
		AllocationQty: 30, Priority: 1,
	})
	svc.CreateRule(ctx, &AllocationRule{
		Name: "备仓20", RuleType: "fixed", SkuID: 100, WarehouseID: 2,
		AllocationQty: 20, Priority: 2,
	})

	// Create initial inventory: 100 units in WH1, 0 in WH2
	db.Create(&inventoryWarehouseRow{
		SkuID: 100, WarehouseID: 1, Quantity: 100, LockedQuantity: 0,
	})

	// Run auto-allocate
	if err := svc.AutoAllocate(ctx, 100); err != nil {
		t.Fatalf("AutoAllocate: %v", err)
	}

	// Verify: WH1 gets 30, WH2 gets 20 (new record created)
	var records []inventoryWarehouseRow
	db.Where("sku_id = ?", 100).Order("warehouse_id ASC").Find(&records)

	if len(records) != 2 {
		t.Fatalf("expected 2 inventory records, got %d", len(records))
	}

	for _, r := range records {
		switch r.WarehouseID {
		case 1:
			if r.Quantity != 30 {
				t.Errorf("WH1 expected qty 30, got %d", r.Quantity)
			}
		case 2:
			if r.Quantity != 20 {
				t.Errorf("WH2 expected qty 20, got %d", r.Quantity)
			}
		}
	}
}

func TestService_AutoAllocate_FixedExhaustsRemaining(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Warehouse{}, &AllocationRule{}, &inventoryWarehouseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	db.Create(&Warehouse{ID: 1, Name: "上海仓", Code: "SH01"})
	db.Create(&Warehouse{ID: 2, Name: "广州仓", Code: "GZ02"})

	// Create rule that asks for more than available
	svc.CreateRule(ctx, &AllocationRule{
		Name: "主仓200", RuleType: "fixed", SkuID: 100, WarehouseID: 1,
		AllocationQty: 200, Priority: 1,
	})

	// Only 50 units total available
	db.Create(&inventoryWarehouseRow{
		SkuID: 100, WarehouseID: 1, Quantity: 50, LockedQuantity: 0,
	})

	if err := svc.AutoAllocate(ctx, 100); err != nil {
		t.Fatalf("AutoAllocate: %v", err)
	}

	var rec inventoryWarehouseRow
	db.Where("sku_id = ? AND warehouse_id = ?", 100, 1).First(&rec)
	if rec.Quantity != 50 {
		t.Errorf("WH1 expected qty 50 (capped at available), got %d", rec.Quantity)
	}
}

func TestService_AutoAllocate_NoRules(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Warehouse{}, &AllocationRule{}, &inventoryWarehouseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	// No rules for SKU 999
	db.Create(&inventoryWarehouseRow{
		SkuID: 999, WarehouseID: 1, Quantity: 100, LockedQuantity: 0,
	})

	if err := svc.AutoAllocate(ctx, 999); err != nil {
		t.Fatalf("AutoAllocate: %v", err)
	}
	// No rules means no-op; verify the inventory is unchanged
	var rec inventoryWarehouseRow
	db.Where("sku_id = ? AND warehouse_id = ?", 999, 1).First(&rec)
	if rec.Quantity != 100 {
		t.Errorf("expected unchanged qty 100, got %d", rec.Quantity)
	}
}

func TestService_AutoAllocate_ZeroesUntargetedWarehouses(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Warehouse{}, &AllocationRule{}, &inventoryWarehouseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx := context.Background()

	db.Create(&Warehouse{ID: 1, Name: "上海仓", Code: "SH01"})
	db.Create(&Warehouse{ID: 2, Name: "广州仓", Code: "GZ02"})

	// Only WH1 has a rule
	svc.CreateRule(ctx, &AllocationRule{
		Name: "主仓100%", RuleType: "percentage", SkuID: 100, WarehouseID: 1,
		AllocationPct: decimal.NewFromInt(100), Priority: 1,
	})

	// WH2 has existing inventory that should be zeroed
	db.Create(&inventoryWarehouseRow{
		SkuID: 100, WarehouseID: 1, Quantity: 30, LockedQuantity: 0,
	})
	db.Create(&inventoryWarehouseRow{
		SkuID: 100, WarehouseID: 2, Quantity: 70, LockedQuantity: 0,
	})

	if err := svc.AutoAllocate(ctx, 100); err != nil {
		t.Fatalf("AutoAllocate: %v", err)
	}

	var records []inventoryWarehouseRow
	db.Where("sku_id = ?", 100).Order("warehouse_id ASC").Find(&records)

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	for _, r := range records {
		switch r.WarehouseID {
		case 1:
			if r.Quantity != 100 {
				t.Errorf("WH1 expected qty 100, got %d", r.Quantity)
			}
		case 2:
			if r.Quantity != 0 {
				t.Errorf("WH2 (no rule) expected qty 0, got %d", r.Quantity)
			}
		}
	}
}
