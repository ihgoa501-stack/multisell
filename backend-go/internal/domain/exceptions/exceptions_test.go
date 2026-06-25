package exceptions

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create
	e := &ExceptionItem{
		SourceModule: "order",
		Severity:     "high",
		Title:        "订单支付异常",
		Description:  "订单 #12345 支付超时",
		Status:       "open",
	}
	err := svc.Create(e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.ID == 0 {
		t.Fatal("ID should be set")
	}

	// GetByID
	got, err := svc.GetByID(e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "订单支付异常" {
		t.Fatalf("Title = %s", got.Title)
	}

	// List
	items, total, err := svc.List(ListFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// Filter by severity
	items, total, err = svc.List(ListFilter{Severity: "high"}, 1, 10)
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("filtered total = %d", total)
	}

	// Assign
	assigned, err := svc.Assign(e.ID, "user_zhang")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if assigned.AssignedTo != "user_zhang" {
		t.Fatalf("AssignedTo = %s", assigned.AssignedTo)
	}
	if assigned.Status != "assigned" {
		t.Fatalf("Status = %s", assigned.Status)
	}

	// Resolve
	resolved, err := svc.Resolve(e.ID, "user_li", "已处理")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("Status = %s", resolved.Status)
	}
	if resolved.ResolvedBy != "user_li" {
		t.Fatalf("ResolvedBy = %s", resolved.ResolvedBy)
	}

	// Delete
	if err := svc.Delete(e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(e.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
