package operationlog

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &OperationLog{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func TestOperationLog_Create(t *testing.T) {
	svc := newService(t)

	l := &OperationLog{
		Module:     "product",
		Action:     "create",
		ResourceID: "42",
		Content:    "Created product Widget",
		Operator:   "admin",
		IP:         "127.0.0.1",
	}
	if err := svc.Create(l); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if l.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}
}

func TestOperationLog_GetByID(t *testing.T) {
	svc := newService(t)

	l := &OperationLog{Module: "sku", Action: "update", ResourceID: "10", Operator: "user1"}
	if err := svc.Create(l); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.GetByID(l.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Module != "sku" {
		t.Fatalf("Module = %q, want %q", got.Module, "sku")
	}
	if got.Action != "update" {
		t.Fatalf("Action = %q, want %q", got.Action, "update")
	}
}

func TestOperationLog_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetByID(999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestOperationLog_List_NoFilter(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 5; i++ {
		_ = svc.Create(&OperationLog{Module: "product", Action: "create", Operator: "admin"})
	}

	items, total, err := svc.List(ListFilter{}, 1, 20)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(items) != 5 {
		t.Fatalf("returned %d items, want 5", len(items))
	}
}

func TestOperationLog_List_FilterByModule(t *testing.T) {
	svc := newService(t)

	_ = svc.Create(&OperationLog{Module: "product", Action: "create", Operator: "admin"})
	_ = svc.Create(&OperationLog{Module: "sku", Action: "update", Operator: "admin"})
	_ = svc.Create(&OperationLog{Module: "product", Action: "delete", Operator: "user1"})

	items, total, err := svc.List(ListFilter{Module: "product"}, 1, 20)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	for _, item := range items {
		if item.Module != "product" {
			t.Fatalf("Module = %q, want %q", item.Module, "product")
		}
	}
	_ = items
}

func TestOperationLog_List_FilterByAction(t *testing.T) {
	svc := newService(t)

	_ = svc.Create(&OperationLog{Module: "product", Action: "create", Operator: "admin"})
	_ = svc.Create(&OperationLog{Module: "product", Action: "update", Operator: "admin"})
	_ = svc.Create(&OperationLog{Module: "product", Action: "create", Operator: "user1"})

	items, total, err := svc.List(ListFilter{Action: "create"}, 1, 20)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	for _, item := range items {
		if item.Action != "create" {
			t.Fatalf("Action = %q, want %q", item.Action, "create")
		}
	}
}

func TestOperationLog_List_FilterByOperator(t *testing.T) {
	svc := newService(t)

	_ = svc.Create(&OperationLog{Module: "product", Action: "create", Operator: "admin"})
	_ = svc.Create(&OperationLog{Module: "sku", Action: "create", Operator: "user1"})
	_ = svc.Create(&OperationLog{Module: "category", Action: "create", Operator: "admin"})

	items, total, err := svc.List(ListFilter{Operator: "user1"}, 1, 20)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if items[0].Operator != "user1" {
		t.Fatalf("Operator = %q, want %q", items[0].Operator, "user1")
	}
}

func TestOperationLog_List_CombinedFilter(t *testing.T) {
	svc := newService(t)

	_ = svc.Create(&OperationLog{Module: "product", Action: "create", Operator: "admin"})
	_ = svc.Create(&OperationLog{Module: "product", Action: "update", Operator: "admin"})
	_ = svc.Create(&OperationLog{Module: "product", Action: "create", Operator: "user1"})

	items, total, err := svc.List(ListFilter{Module: "product", Action: "create"}, 1, 20)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	for _, item := range items {
		if item.Module != "product" || item.Action != "create" {
			t.Fatalf("unexpected item: module=%q action=%q", item.Module, item.Action)
		}
	}
}

func TestOperationLog_List_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 25; i++ {
		_ = svc.Create(&OperationLog{Module: "product", Action: "create", Operator: "admin"})
	}

	items, total, err := svc.List(ListFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("total = %d, want 25", total)
	}
	if len(items) != 10 {
		t.Fatalf("returned %d items, want 10", len(items))
	}

	items2, _, err := svc.List(ListFilter{}, 2, 10)
	if err != nil {
		t.Fatalf("List page 2 failed: %v", err)
	}
	if len(items2) != 10 {
		t.Fatalf("page 2 returned %d items, want 10", len(items2))
	}
	if items[0].ID == items2[0].ID {
		t.Fatal("page 1 and page 2 should return different items")
	}
}

func TestOperationLog_Log_Convenience(t *testing.T) {
	svc := newService(t)

	if err := svc.Log("listing", "publish", "7", "agent", "Published listing 7"); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	items, total, err := svc.List(ListFilter{Module: "listing"}, 1, 20)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if items[0].Action != "publish" {
		t.Fatalf("Action = %q, want %q", items[0].Action, "publish")
	}
	if items[0].ResourceID != "7" {
		t.Fatalf("ResourceID = %q, want %q", items[0].ResourceID, "7")
	}
	if items[0].Operator != "agent" {
		t.Fatalf("Operator = %q, want %q", items[0].Operator, "agent")
	}
}

func TestOperationLog_List_OrderDesc(t *testing.T) {
	svc := newService(t)

	_ = svc.Create(&OperationLog{Module: "a", Action: "x", Operator: "u"})
	_ = svc.Create(&OperationLog{Module: "b", Action: "y", Operator: "u"})
	_ = svc.Create(&OperationLog{Module: "c", Action: "z", Operator: "u"})

	items, _, err := svc.List(ListFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("returned %d items, want 3", len(items))
	}
	if items[0].Module != "c" {
		t.Fatalf("first item Module = %q, want %q (should be desc order)", items[0].Module, "c")
	}
	if items[2].Module != "a" {
		t.Fatalf("last item Module = %q, want %q", items[2].Module, "a")
	}
}

func TestOperationLog_List_DefaultPagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 3; i++ {
		_ = svc.Create(&OperationLog{Module: "x", Action: "y", Operator: "z"})
	}

	items, total, err := svc.List(ListFilter{}, 0, 0)
	if err != nil {
		t.Fatalf("List with zero page/size failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(items) != 3 {
		t.Fatalf("returned %d items, want 3 (default size should be >= 3)", len(items))
	}
}
