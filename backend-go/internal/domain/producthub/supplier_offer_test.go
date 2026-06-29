package producthub

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newSupplierOfferService(t *testing.T) *SupplierOfferService {
	t.Helper()
	db := dbtest.NewDB(t, &ProductMaster{}, &SupplierOffer{})
	return NewSupplierOfferService(db, zap.NewNop())
}

func TestSupplierOfferCreateAndList(t *testing.T) {
	db := dbtest.NewDB(t, &ProductMaster{}, &SupplierOffer{})
	ms := NewMasterService(db, zap.NewNop())
	svc := NewSupplierOfferService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Offer Test", OwnerID: 1}
	ms.Create(ctx, master)

	o := &SupplierOffer{ProductMasterID: master.ID, SupplierID: 100, UnitCost: 15.50, Currency: "CNY", MOQ: 1000}
	if err := svc.Create(ctx, o); err != nil {
		t.Fatal(err)
	}
	items, _ := svc.ListByMaster(ctx, master.ID)
	if len(items) != 1 || items[0].UnitCost != 15.50 {
		t.Fatalf("expected 1 offer with cost 15.50, got %d offers", len(items))
	}
}

func TestSupplierOfferUpdate(t *testing.T) {
	svc := newSupplierOfferService(t)
	db := dbtest.NewDB(t, &ProductMaster{}, &SupplierOffer{})
	ms := NewMasterService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Offer Update", OwnerID: 1}
	ms.Create(ctx, master)

	o := &SupplierOffer{ProductMasterID: master.ID, SupplierID: 200, UnitCost: 10.0}
	svc.Create(ctx, o)

	o.UnitCost = 12.0
	svc.Update(ctx, o)

	got, _ := svc.GetByID(ctx, o.ID)
	if got.UnitCost != 12.0 {
		t.Fatalf("expected 12.0, got %.2f", got.UnitCost)
	}
}
