package exceptions

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &ExceptionItem{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func TestException_Create(t *testing.T) {
	svc := newService(t)

	e := &ExceptionItem{
		SourceModule: "order",
		SourceType:   "sales_order",
		Severity:     "high",
		Status:       "open",
		Title:        "Order payment failed",
	}
	if err := svc.Create(e); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if e.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestException_GetByID(t *testing.T) {
	svc := newService(t)

	e := &ExceptionItem{
		SourceModule: "inventory",
		Severity:     "medium",
		Status:       "open",
		Title:        "Low stock alert",
	}
	_ = svc.Create(e)

	got, err := svc.GetByID(e.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Title != "Low stock alert" {
		t.Fatalf("Title=%q, want Low stock alert", got.Title)
	}
}

func TestException_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetByID(999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestException_Update(t *testing.T) {
	svc := newService(t)

	e := &ExceptionItem{
		SourceModule: "order",
		Severity:     "low",
		Status:       "open",
		Title:        "Original title",
	}
	_ = svc.Create(e)

	e.Title = "Updated title"
	e.Severity = "high"
	if err := svc.Update(e); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := svc.GetByID(e.ID)
	if got.Title != "Updated title" {
		t.Fatalf("Title=%q, want Updated title", got.Title)
	}
	if got.Severity != "high" {
		t.Fatalf("Severity=%q, want high", got.Severity)
	}
}

func TestException_Delete(t *testing.T) {
	svc := newService(t)

	e := &ExceptionItem{
		SourceModule: "sku",
		Severity:     "low",
		Status:       "open",
		Title:        "To delete",
	}
	_ = svc.Create(e)

	if err := svc.Delete(e.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := svc.GetByID(e.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

func TestException_Resolve(t *testing.T) {
	svc := newService(t)

	e := &ExceptionItem{
		SourceModule: "order",
		Severity:     "high",
		Status:       "open",
		Title:        "Payment issue",
	}
	_ = svc.Create(e)

	resolved, err := svc.Resolve(e.ID, "admin", "Fixed via manual payment")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("Status=%q, want resolved", resolved.Status)
	}
	if resolved.ResolvedAt == nil {
		t.Fatal("expected non-nil ResolvedAt")
	}
	if resolved.ResolvedBy != "admin" {
		t.Fatalf("ResolvedBy=%q, want admin", resolved.ResolvedBy)
	}
	if resolved.Note != "Fixed via manual payment" {
		t.Fatalf("Note=%q, want Fixed via manual payment", resolved.Note)
	}
}

func TestException_Resolve_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.Resolve(999, "admin", ""); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestException_Assign(t *testing.T) {
	svc := newService(t)

	e := &ExceptionItem{
		SourceModule: "inventory",
		Severity:     "medium",
		Status:       "open",
		Title:        "Stock mismatch",
	}
	_ = svc.Create(e)

	assigned, err := svc.Assign(e.ID, "user1")
	if err != nil {
		t.Fatalf("Assign failed: %v", err)
	}
	if assigned.AssignedTo != "user1" {
		t.Fatalf("AssignedTo=%q, want user1", assigned.AssignedTo)
	}
	if assigned.Status != "assigned" {
		t.Fatalf("Status=%q, want assigned", assigned.Status)
	}
}

func TestException_Assign_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.Assign(999, "user1"); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestException_List_FilterBySeverity(t *testing.T) {
	svc := newService(t)

	_ = svc.Create(&ExceptionItem{SourceModule: "order", Severity: "high", Status: "open", Title: "H1"})
	_ = svc.Create(&ExceptionItem{SourceModule: "order", Severity: "high", Status: "open", Title: "H2"})
	_ = svc.Create(&ExceptionItem{SourceModule: "order", Severity: "low", Status: "open", Title: "L1"})

	items, total, err := svc.List(ListFilter{Severity: "high"}, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want 2", len(items))
	}
}

func TestException_List_FilterByStatus(t *testing.T) {
	svc := newService(t)

	_ = svc.Create(&ExceptionItem{SourceModule: "sku", Severity: "medium", Status: "open", Title: "O1"})
	_ = svc.Create(&ExceptionItem{SourceModule: "sku", Severity: "medium", Status: "resolved", Title: "R1"})

	items, total, err := svc.List(ListFilter{Status: "resolved"}, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(items) != 1 || items[0].Title != "R1" {
		t.Fatalf("expected R1, got %v", items)
	}
}

func TestException_List_FilterByAssignedTo(t *testing.T) {
	svc := newService(t)

	_ = svc.Create(&ExceptionItem{SourceModule: "order", Severity: "low", Status: "assigned", Title: "A1", AssignedTo: "alice"})
	_ = svc.Create(&ExceptionItem{SourceModule: "order", Severity: "low", Status: "assigned", Title: "A2", AssignedTo: "bob"})

	items, total, err := svc.List(ListFilter{AssignedTo: "alice"}, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(items) != 1 || items[0].Title != "A1" {
		t.Fatalf("expected A1, got %v", items)
	}
}

func TestException_List_FilterBySourceModule(t *testing.T) {
	svc := newService(t)

	_ = svc.Create(&ExceptionItem{SourceModule: "order", Severity: "low", Status: "open", Title: "O1"})
	_ = svc.Create(&ExceptionItem{SourceModule: "inventory", Severity: "low", Status: "open", Title: "I1"})

	items, total, err := svc.List(ListFilter{SourceModule: "inventory"}, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(items) != 1 || items[0].Title != "I1" {
		t.Fatalf("expected I1, got %v", items)
	}
}

func TestException_List_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 5; i++ {
		_ = svc.Create(&ExceptionItem{SourceModule: "order", Severity: "low", Status: "open", Title: "Item"})
	}

	items, total, err := svc.List(ListFilter{}, 1, 2)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 5 {
		t.Fatalf("total=%d, want 5", total)
	}
	if len(items) != 2 {
		t.Fatalf("page1 len=%d, want 2", len(items))
	}

	items2, _, err := svc.List(ListFilter{}, 2, 2)
	if err != nil {
		t.Fatalf("List page2 failed: %v", err)
	}
	if len(items2) != 2 {
		t.Fatalf("page2 len=%d, want 2", len(items2))
	}
}
