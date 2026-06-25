package operationlog

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create via Log convenience helper
	err := svc.Log("order", "create", "123", "admin", "创建订单 #123")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	// Create directly
	err = svc.Create(&OperationLog{
		Module:     "product",
		Action:     "update",
		ResourceID: "456",
		Operator:   "admin",
		Content:    "更新商品价格",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// List
	items, total, err := svc.List(ListFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// GetByID
	got, err := svc.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Module != "order" {
		t.Fatalf("Module = %s", got.Module)
	}

	// Filter by module
	items, total, err = svc.List(ListFilter{Module: "product"}, 1, 10)
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("filtered total = %d", total)
	}

	// Filter by time range
	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)
	items, total, err = svc.List(ListFilter{From: from, To: to}, 1, 10)
	if err != nil {
		t.Fatalf("List time filtered: %v", err)
	}
	if total != 2 {
		t.Fatalf("time filtered total = %d (expected 2)", total)
	}
}
