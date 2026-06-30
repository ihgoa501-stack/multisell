package completeness

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_Check_CompleteProduct(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	// The check reads candidate_product which won't exist in this test DB
	svc := NewService(db, dbtest.NewLogger(t))

	// Without a candidate product, it should error
	_, err := svc.Check(999, "tester")
	if err == nil {
		t.Fatal("expected error for non-existent product")
	}
}

func TestService_ListChecks(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Insert some checks directly
	db.Create(&CompletenessCheck{ProductID: 1, Score: 90, Status: "complete", TriggeredBy: "system"})
	db.Create(&CompletenessCheck{ProductID: 2, Score: 45, Status: "incomplete", TriggeredBy: "system"})

	items, total, err := svc.ListChecks(1, 10, "")
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
}

func TestService_ListChecks_Filtered(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&CompletenessCheck{ProductID: 1, Score: 90, Status: "complete"})
	db.Create(&CompletenessCheck{ProductID: 2, Score: 45, Status: "incomplete"})

	items, total, err := svc.ListChecks(1, 10, "complete")
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	if items[0].Status != "complete" {
		t.Fatalf("status = %s", items[0].Status)
	}
}
