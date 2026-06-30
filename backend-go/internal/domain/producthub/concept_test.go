package producthub

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newConceptService(t *testing.T) *ConceptService {
	t.Helper()
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductConcept{})
	return NewConceptService(db, zap.NewNop())
}

func TestConceptUpsert(t *testing.T) {
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductConcept{})
	ms := NewMasterService(db, zap.NewNop())
	svc := NewConceptService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Concept Test", OwnerID: 1}
	if err := ms.Create(ctx, master); err != nil {
		t.Fatal(err)
	}

	c := &ProductConcept{ProductMasterID: master.ID, Brief: "A great idea", DesignSource: "internal"}
	if err := svc.Upsert(ctx, c); err != nil {
		t.Fatal(err)
	}

	got, err := svc.GetByMasterID(ctx, master.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Brief != "A great idea" {
		t.Fatalf("expected 'A great idea', got '%s'", got.Brief)
	}

	c.Brief = "Revised idea"
	if err := svc.Upsert(ctx, c); err != nil {
		t.Fatal(err)
	}
	got2, _ := svc.GetByMasterID(ctx, master.ID)
	if got2.Brief != "Revised idea" {
		t.Fatalf("expected 'Revised idea', got '%s'", got2.Brief)
	}
}
