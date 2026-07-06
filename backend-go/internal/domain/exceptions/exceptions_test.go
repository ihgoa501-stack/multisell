package exceptions

import (
	"fmt"
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

func TestCreate_DifferentSeverities(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	levels := []string{"low", "medium", "high", "critical"}
	for _, lvl := range levels {
		e := &ExceptionItem{
			SourceModule: "test",
			Severity:     lvl,
			Title:        "severity-" + lvl,
		}
		if err := svc.Create(e); err != nil {
			t.Fatalf("Create(%s): %v", lvl, err)
		}
		if e.ID == 0 {
			t.Fatalf("Create(%s): ID not set", lvl)
		}
	}

	items, total, err := svc.List(ListFilter{}, 1, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != int64(len(levels)) {
		t.Fatalf("total = %d, want %d", total, len(levels))
	}
	// Verify each severity was stored
	seen := map[string]bool{}
	for _, it := range items {
		seen[it.Severity] = true
	}
	for _, lvl := range levels {
		if !seen[lvl] {
			t.Fatalf("missing severity %s in results", lvl)
		}
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.GetByID(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestList_FilterBySeverity(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	for _, s := range []string{"low", "high"} {
		e := &ExceptionItem{
			SourceModule: "order",
			Severity:     s,
			Title:        "sev-" + s,
		}
		if err := svc.Create(e); err != nil {
			t.Fatalf("Create(%s): %v", s, err)
		}
	}

	items, total, err := svc.List(ListFilter{Severity: "high"}, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(items) != 1 || items[0].Severity != "high" {
		t.Fatalf("expected 1 high item, got %d", len(items))
	}
}

func TestList_FilterByStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	for _, s := range []string{"open", "assigned"} {
		e := &ExceptionItem{
			SourceModule: "order",
			Status:       s,
			Title:        "status-" + s,
		}
		if err := svc.Create(e); err != nil {
			t.Fatalf("Create(%s): %v", s, err)
		}
	}

	items, total, err := svc.List(ListFilter{Status: "open"}, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(items) != 1 || items[0].Status != "open" {
		t.Fatalf("expected 1 open item, got %d", len(items))
	}
}

func TestList_FilterBySourceModule(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	for _, src := range []string{"order", "fulfillment"} {
		e := &ExceptionItem{
			SourceModule: src,
			Title:        "src-" + src,
		}
		if err := svc.Create(e); err != nil {
			t.Fatalf("Create(%s): %v", src, err)
		}
	}

	items, total, err := svc.List(ListFilter{SourceModule: "order"}, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(items) != 1 || items[0].SourceModule != "order" {
		t.Fatalf("expected 1 order item, got %d", len(items))
	}
}

func TestList_Pagination(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	for i := 0; i < 5; i++ {
		e := &ExceptionItem{
			SourceModule: "order",
			Title:        fmt.Sprintf("item-%d", i),
		}
		if err := svc.Create(e); err != nil {
			t.Fatalf("Create(%d): %v", i, err)
		}
	}

	// page 1, size 2 → 2 items
	p1, total, err := svc.List(ListFilter{}, 1, 2)
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(p1) != 2 {
		t.Fatalf("page 1 got %d items, want 2", len(p1))
	}

	// page 4 (past last page) → 0 items
	p4, _, err := svc.List(ListFilter{}, 4, 2)
	if err != nil {
		t.Fatalf("List past-last page: %v", err)
	}
	if len(p4) != 0 {
		t.Fatalf("past-last page got %d items, want 0", len(p4))
	}

	// default page/size when 0 is passed
	def, _, err := svc.List(ListFilter{}, 0, 0)
	if err != nil {
		t.Fatalf("List defaults: %v", err)
	}
	if len(def) != 5 {
		t.Fatalf("default page got %d items, want 5", len(def))
	}
}

func TestResolve_Complete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	e := &ExceptionItem{
		SourceModule: "order",
		Title:        "test resolve",
	}
	if err := svc.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	resolved, err := svc.Resolve(e.ID, "user_li", "已修复")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("Status = %q, want resolved", resolved.Status)
	}
	if resolved.ResolvedBy != "user_li" {
		t.Fatalf("ResolvedBy = %q, want user_li", resolved.ResolvedBy)
	}
	if resolved.ResolvedAt == nil {
		t.Fatal("ResolvedAt is nil, expected a timestamp")
	}
	if resolved.Note != "已修复" {
		t.Fatalf("Note = %q, want 已修复", resolved.Note)
	}
}

func TestResolve_AlreadyResolved(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	e := &ExceptionItem{
		SourceModule: "order",
		Title:        "test re-resolve",
	}
	if err := svc.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First resolve
	first, err := svc.Resolve(e.ID, "user_a", "first")
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if first.Status != "resolved" {
		t.Fatalf("Status = %q after first resolve", first.Status)
	}

	// Re-resolve — current implementation allows it
	second, err := svc.Resolve(e.ID, "user_b", "re-resolved")
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if second.Status != "resolved" {
		t.Fatalf("Status = %q after second resolve", second.Status)
	}
	if second.ResolvedBy != "user_b" {
		t.Fatalf("ResolvedBy = %q, want user_b", second.ResolvedBy)
	}
	if second.Note != "re-resolved" {
		t.Fatalf("Note = %q, want re-resolved", second.Note)
	}
}
