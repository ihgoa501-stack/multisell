package order

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDBCounter int64

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	testDBCounter++
	dsn := "file:order_test_" + itoa(testDBCounter) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Order{}, &OrderItem{}, &OrderStatusLog{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestService_Create_WithItems(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	o, err := svc.Create(&CreateOrderInput{
		OrderNo:   "ORD-001",
		Status:    "pending",
		RecipientName: "张三",
		Items: []OrderItemInput{
			{SkuID: 1, ProductID: 1, ProductName: "商品A", UnitPrice: 10.5, Quantity: 2},
			{SkuID: 2, ProductID: 1, ProductName: "商品A-规格2", UnitPrice: 20, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if o.ID == 0 {
		t.Fatal("ID should be set")
	}
	if o.OrderNo != "ORD-001" {
		t.Fatalf("OrderNo = %s", o.OrderNo)
	}

	// Verify items persisted with subtotal computed.
	detail, err := svc.Get(o.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(detail.Items) != 2 {
		t.Fatalf("items = %d", len(detail.Items))
	}
	if detail.Items[0].Subtotal != 21.0 {
		t.Fatalf("subtotal[0] = %v", detail.Items[0].Subtotal)
	}
}

func TestService_Update(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	o, _ := svc.Create(&CreateOrderInput{OrderNo: "ORD-002"})
	newStatus := "shipped"
	newTracking := "TRACK-123"
	updated, err := svc.Update(o.ID, &UpdateOrderInput{Status: &newStatus, TrackingNumber: &newTracking})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "shipped" {
		t.Fatalf("status = %s", updated.Status)
	}
	if updated.TrackingNumber != "TRACK-123" {
		t.Fatalf("tracking = %s", updated.TrackingNumber)
	}
}

func TestService_UpdateStatus_LogsTransition(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	o, _ := svc.Create(&CreateOrderInput{OrderNo: "ORD-003", Status: "pending"})
	if err := svc.UpdateStatus(o.ID, "pending", "paid", "alice", "user paid"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	detail, _ := svc.Get(o.ID)
	if len(detail.StatusLogs) != 1 {
		t.Fatalf("status logs = %d", len(detail.StatusLogs))
	}
	log := detail.StatusLogs[0]
	if log.FromStatus != "pending" || log.ToStatus != "paid" {
		t.Fatalf("transition = %s → %s", log.FromStatus, log.ToStatus)
	}
	if log.Operator != "alice" {
		t.Fatalf("operator = %s", log.Operator)
	}
}

func TestService_List_Filtering(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	_, _ = svc.Create(&CreateOrderInput{OrderNo: "ORD-A", Status: "pending"})
	_, _ = svc.Create(&CreateOrderInput{OrderNo: "ORD-B", Status: "shipped"})
	_, _ = svc.Create(&CreateOrderInput{OrderNo: "ORD-C", Status: "pending"})

	p := common.Pagination{Page: 1, Size: 10}

	// All
	all, total, _ := svc.List(&p, &OrderListFilter{})
	if total != 3 || len(all) != 3 {
		t.Fatalf("all: total=%d len=%d", total, len(all))
	}

	// Filter by status
	pending, total, _ := svc.List(&p, &OrderListFilter{Status: "pending"})
	if total != 2 {
		t.Fatalf("pending total = %d", total)
	}
	for _, o := range pending {
		if o.Status != "pending" {
			t.Fatalf("status = %s", o.Status)
		}
	}

	// Search by order_no
	found, total, _ := svc.List(&p, &OrderListFilter{Search: "ORD-B"})
	if total != 1 || len(found) != 1 {
		t.Fatalf("search: total=%d len=%d", total, len(found))
	}
	if found[0].OrderNo != "ORD-B" {
		t.Fatalf("orderNo = %s", found[0].OrderNo)
	}
}

func TestService_Delete_Cascades(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	o, _ := svc.Create(&CreateOrderInput{
		OrderNo: "ORD-DEL",
		Items:   []OrderItemInput{{SkuID: 1, ProductID: 1, ProductName: "x", UnitPrice: 1, Quantity: 1}},
	})
	// Add a status log
	_ = svc.UpdateStatus(o.ID, "pending", "paid", "op", "")

	if err := svc.Delete(o.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Order + items + logs should all be gone.
	_, err := svc.Get(o.ID)
	if err == nil {
		t.Fatal("expected Get to fail after delete")
	}
	var itemCount int64
	db.Model(&OrderItem{}).Where("order_id = ?", o.ID).Count(&itemCount)
	if itemCount != 0 {
		t.Fatalf("orphan items = %d", itemCount)
	}
}

func TestService_Create_DuplicateOrderNo(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	_, _ = svc.Create(&CreateOrderInput{OrderNo: "DUP-1"})
	_, err := svc.Create(&CreateOrderInput{OrderNo: "DUP-1"})
	if err == nil {
		t.Fatal("expected duplicate order_no to fail")
	}
}

func TestService_Summary(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	_, _ = svc.Create(&CreateOrderInput{OrderNo: "S-1", Status: "pending"})
	_, _ = svc.Create(&CreateOrderInput{OrderNo: "S-2", Status: "shipped"})
	_, _ = svc.Create(&CreateOrderInput{OrderNo: "S-3", Status: "pending"})

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Total != 3 {
		t.Fatalf("total = %d", summary.Total)
	}
	if summary.ByStatus["pending"] != 2 {
		t.Fatalf("pending = %d", summary.ByStatus["pending"])
	}
	if summary.ByStatus["shipped"] != 1 {
		t.Fatalf("shipped = %d", summary.ByStatus["shipped"])
	}
}
