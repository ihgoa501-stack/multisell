package producthub

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newCostVersionService(t *testing.T) *CostVersionService {
	t.Helper()
	db := dbtest.NewDB(t, &ProductMaster{}, &CostVersion{})
	return NewCostVersionService(db, zap.NewNop())
}

func TestCostLandedCost(t *testing.T) {
	cv := &CostVersion{
		BaseCost:      100.0,
		MaterialCost:  50.0,
		PackagingCost: 10.0,
		FreightCost:   25.0,
		DutyCost:      15.0,
	}
	got := cv.CostLandedCost()
	want := 200.0 // 100 + 50 + 10 + 25 + 15
	if got != want {
		t.Fatalf("CostLandedCost() = %.2f, want %.2f", got, want)
	}
}

func TestCostLandedCost_Zero(t *testing.T) {
	cv := &CostVersion{}
	got := cv.CostLandedCost()
	if got != 0 {
		t.Fatalf("CostLandedCost() = %.2f, want 0", got)
	}
}

func TestCostGrossMargin(t *testing.T) {
	cv := &CostVersion{
		BaseCost:        10.0,
		MaterialCost:    5.0,
		PackagingCost:   1.0,
		FreightCost:     3.0,
		DutyCost:        2.0,
		PlatformFeeRate: 15.0,
		AdCostEstimate:  2.0,
	}
	// Landed = 21, price = 50
	// GM = (50 - 21 - 50*0.15 - 2) / 50 * 100 = (50 - 21 - 7.5 - 2) / 50 * 100 = 19.5/50*100 = 39
	got := cv.CostGrossMargin(50.0)
	want := 39.0
	if got != want {
		t.Fatalf("CostGrossMargin(50) = %.2f, want %.2f", got, want)
	}
}

func TestCostGrossMargin_ZeroPrice(t *testing.T) {
	cv := &CostVersion{
		BaseCost:     10.0,
		MaterialCost: 5.0,
	}
	got := cv.CostGrossMargin(0)
	if got != 0 {
		t.Fatalf("CostGrossMargin(0) = %.2f, want 0", got)
	}
}

func TestCostVersionCreateAndCalculate(t *testing.T) {
	db := dbtest.NewDB(t, &ProductMaster{}, &CostVersion{})
	ms := NewMasterService(db, zap.NewNop())
	svc := NewCostVersionService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Cost Test", OwnerID: 1}
	ms.Create(ctx, master)

	cv := &CostVersion{
		ProductMasterID:  master.ID,
		BaseCost:         10.0,
		MaterialCost:     5.0,
		PackagingCost:    1.0,
		FreightCost:      3.0,
		DutyCost:         2.0,
		RecommendedPrice: 30.0,
	}
	if err := svc.Create(ctx, cv); err != nil {
		t.Fatal(err)
	}

	if cv.LandedCost != 21.0 {
		t.Fatalf("expected landed cost 21.0, got %.2f", cv.LandedCost)
	}
	if cv.GrossMargin <= 0 {
		t.Fatalf("expected positive gross margin, got %.2f", cv.GrossMargin)
	}

	if err := svc.Confirm(ctx, cv.ID); err != nil {
		t.Fatal(err)
	}
	confirmed, _ := svc.GetLatestByMaster(ctx, master.ID)
	if confirmed.Status != "confirmed" {
		t.Fatalf("expected 'confirmed', got '%s'", confirmed.Status)
	}
}

func TestCostService_CRUD(t *testing.T) {
	db := dbtest.NewDB(t, &ProductMaster{}, &CostVersion{})
	ms := NewMasterService(db, zap.NewNop())
	svc := NewCostVersionService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Cost Full CRUD", OwnerID: 1}
	if err := ms.Create(ctx, master); err != nil {
		t.Fatal(err)
	}

	cv := &CostVersion{
		ProductMasterID:  master.ID,
		BaseCost:         50.0,
		MaterialCost:     20.0,
		RecommendedPrice: 100.0,
	}
	if err := svc.Create(ctx, cv); err != nil {
		t.Fatal(err)
	}
	if cv.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}

	got, err := svc.GetLatestByMaster(ctx, master.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseCost != 50.0 {
		t.Fatalf("expected BaseCost 50, got %.2f", got.BaseCost)
	}

	if err := svc.Confirm(ctx, cv.ID); err != nil {
		t.Fatal(err)
	}
	confirmed, _ := svc.GetLatestByMaster(ctx, master.ID)
	if confirmed.Status != "confirmed" {
		t.Fatalf("expected confirmed, got %s", confirmed.Status)
	}
}

func TestCostService_ListByMaster(t *testing.T) {
	db := dbtest.NewDB(t, &ProductMaster{}, &CostVersion{})
	ms := NewMasterService(db, zap.NewNop())
	svc := NewCostVersionService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Cost List Test", OwnerID: 1}
	if err := ms.Create(ctx, master); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		cv := &CostVersion{
			ProductMasterID: master.ID,
			BaseCost:        float64((i + 1) * 10),
		}
		if err := svc.Create(ctx, cv); err != nil {
			t.Fatal(err)
		}
	}

	items, err := svc.ListByMaster(ctx, master.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(items))
	}
	// Ordered by ID DESC
	if items[0].BaseCost != 30.0 || items[2].BaseCost != 10.0 {
		t.Fatalf("expected descending order: 30, 20, 10; got %.0f, %.0f, %.0f",
			items[0].BaseCost, items[1].BaseCost, items[2].BaseCost)
	}
}
