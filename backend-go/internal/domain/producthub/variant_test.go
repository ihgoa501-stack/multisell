package producthub

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newVariantService(t *testing.T) *VariantService {
	t.Helper()
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductVariant{})
	return NewVariantService(db, zap.NewNop())
}

func TestVariantCreateAndList(t *testing.T) {
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductVariant{})
	ms := NewMasterService(db, zap.NewNop())
	svc := NewVariantService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Variant Parent", OwnerID: 1}
	if err := ms.Create(ctx, master); err != nil {
		t.Fatal(err)
	}

	v := &ProductVariant{ProductMasterID: master.ID, SKUCode: "TEST-001", VariantLabel: "Black-Large"}
	if err := svc.Create(ctx, v); err != nil {
		t.Fatal(err)
	}

	items, err := svc.ListByMaster(ctx, master.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(items))
	}
	if items[0].SKUCode != "TEST-001" {
		t.Fatalf("expected 'TEST-001', got '%s'", items[0].SKUCode)
	}
}

func TestVariantUpdateAndDelete(t *testing.T) {
	svc := newVariantService(t)
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductVariant{})
	ms := NewMasterService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Variant Update Test", OwnerID: 1}
	ms.Create(ctx, master)

	v := &ProductVariant{ProductMasterID: master.ID, SKUCode: "V1"}
	if err := svc.Create(ctx, v); err != nil {
		t.Fatal(err)
	}

	v.SKUCode = "V1-UPDATED"
	if err := svc.Update(ctx, v); err != nil {
		t.Fatal(err)
	}

	got, _ := svc.GetByID(ctx, v.ID)
	if got.SKUCode != "V1-UPDATED" {
		t.Fatalf("expected 'V1-UPDATED', got '%s'", got.SKUCode)
	}

	if err := svc.Delete(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetByID(ctx, v.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}
