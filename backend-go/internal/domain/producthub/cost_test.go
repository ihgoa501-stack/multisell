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
