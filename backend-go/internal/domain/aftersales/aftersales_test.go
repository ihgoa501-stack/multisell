package aftersales

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDBCounter int64

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	testDBCounter++
	dsn := "file:aftersales_test_" + itoa(testDBCounter) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AfterSalesOrder{}, &order.Order{}, &order.OrderItem{}, &order.OrderStatusLog{}, &inventory.Inventory{}, &inventory.InventoryLog{}, &inventory.Warehouse{}); err != nil {
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

func newSvc(db *gorm.DB) *Service {
	return NewService(db, testLogger(), NewInventoryRestockAdapter(db), NewOrderWriterAdapter(db), nil)
}

func setupOrder(t *testing.T, db *gorm.DB) *order.Order {
	t.Helper()
	o := order.Order{
		OrderNo:       "ORD-" + itoa(time.Now().UnixNano()),
		Status:        "paid",
		RecipientName: "张三",
		TotalAmount:   1000,
		PayAmount:     980,
	}
	if err := db.Create(&o).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}
	return &o
}

func setupInventory(t *testing.T, db *gorm.DB, skuID int64, qty int) *inventory.Inventory {
	t.Helper()
	inv := inventory.Inventory{
		SkuID:          skuID,
		Quantity:       qty,
		LockedQuantity: 0,
	}
	if err := db.Create(&inv).Error; err != nil {
		t.Fatalf("create inventory: %v", err)
	}
	return &inv
}

func TestService_Create_Full(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 2
	amt := 500.0

	as, err := svc.Create(&CreateInput{
		OrderID:        o.ID,
		ReturnQuantity: &qty,
		Reason:         "质量问题",
		RefundAmount:   &amt,
		CreatedBy:      "admin",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if as.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if as.Status != "pending" {
		t.Errorf("expected default status pending, got %s", as.Status)
	}
	if as.ReturnQuantity != 2 {
		t.Errorf("expected return_quantity 2, got %d", as.ReturnQuantity)
	}
	if as.Reason != "质量问题" {
		t.Errorf("expected reason 质量问题, got %s", as.Reason)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	_, err := svc.Get(99999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestService_List_Pagination(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 1
	for i := 0; i < 3; i++ {
		svc.Create(&CreateInput{
			OrderID: o.ID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin",
		})
	}

	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 2}, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestService_List_FilterByStatus(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 1
	svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "待处理", Status: "pending", CreatedBy: "admin"})
	svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "已处理", Status: "approved", CreatedBy: "admin"})

	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 10}, &ListFilter{Status: "pending"})
	if err != nil {
		t.Fatalf("List filter failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 pending, got %d", total)
	}
	if len(items) != 1 || items[0].Reason != "待处理" {
		t.Errorf("expected 待处理, got %s", items[0].Reason)
	}
}

func TestService_Update(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "更新前", CreatedBy: "admin"})

	newReason := "更新后"
	updated, err := svc.Update(as.ID, &UpdateInput{Reason: &newReason})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Reason != "更新后" {
		t.Errorf("expected reason 更新后, got %s", updated.Reason)
	}
}

func TestService_Delete(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "删除测试", CreatedBy: "admin"})

	if err := svc.Delete(as.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err := svc.Get(as.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	err := svc.Delete(99999)
	if err == nil {
		t.Fatal("expected error for not found delete")
	}
}

func TestService_Approve(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin"})

	approved, err := svc.Approve(as.ID, &ApproveInput{
		ApprovedBy:       "manager",
		InspectionResult: "商品完好，同意退货",
	})
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}
	if approved.Status != "approved" {
		t.Errorf("expected status approved, got %s", approved.Status)
	}
	if approved.InspectionResult != "商品完好，同意退货" {
		t.Errorf("expected inspection result, got %s", approved.InspectionResult)
	}
	if approved.ApprovedBy != "manager" {
		t.Errorf("expected approved_by manager, got %s", approved.ApprovedBy)
	}
}

func TestService_Reject(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin"})

	rejected, err := svc.Reject(as.ID, &RejectInput{
		RejectedBy:      "manager",
		RejectionReason: "超过退货期限",
	})
	if err != nil {
		t.Fatalf("Reject failed: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Errorf("expected status rejected, got %s", rejected.Status)
	}
	if rejected.RejectionReason != "超过退货期限" {
		t.Errorf("expected rejection reason, got %s", rejected.RejectionReason)
	}
	if rejected.RejectedBy != "manager" {
		t.Errorf("expected rejected_by manager, got %s", rejected.RejectedBy)
	}
}

func TestService_ApproveThenReceive(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	skuID := int64(1001)
	inv := setupInventory(t, db, skuID, 50)
	qty := 3
	as, _ := svc.Create(&CreateInput{
		OrderID: o.ID, ItemID: &o.ID, SkuID: &skuID,
		ReturnQuantity: &qty, Reason: "退货退款", CreatedBy: "admin",
	})

	// Approve
	svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})

	// Receive (triggers restock)
	received, err := svc.Receive(as.ID, &ReceiveInput{ReceivedBy: "warehouse"})
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if received.Status != "received" {
		t.Errorf("expected status received, got %s", received.Status)
	}
	if received.ReceivedBy != "warehouse" {
		t.Errorf("expected received_by warehouse, got %s", received.ReceivedBy)
	}

	// Verify inventory was restocked
	var updatedInv inventory.Inventory
	if err := db.First(&updatedInv, inv.ID).Error; err != nil {
		t.Fatalf("fetch inventory: %v", err)
	}
	if updatedInv.Quantity != 53 {
		t.Errorf("expected inventory qty 53 (50+3), got %d", updatedInv.Quantity)
	}
}

func TestService_Receive_NotApproved(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "退货", Status: "pending", CreatedBy: "admin"})

	// Try to receive without approving
	_, err := svc.Receive(as.ID, &ReceiveInput{ReceivedBy: "warehouse"})
	if err == nil {
		t.Fatal("expected error when receiving unapproved aftersales order")
	}
}

func TestService_ReceiveThenRefund(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	skuID := int64(1002)
	setupInventory(t, db, skuID, 30)
	qty := 1
	amt := 500.0
	as, _ := svc.Create(&CreateInput{
		OrderID: o.ID, SkuID: &skuID,
		ReturnQuantity: &qty, Reason: "退货",
		RefundAmount: &amt, CreatedBy: "admin",
	})

	// Approve -> Receive -> Refund (full lifecycle)
	svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})
	svc.Receive(as.ID, &ReceiveInput{ReceivedBy: "warehouse"})
	refunded, err := svc.Refund(as.ID, &RefundInput{RefundedBy: "finance", RefundAmount: 500})
	if err != nil {
		t.Fatalf("Refund failed: %v", err)
	}
	if refunded.Status != "refunded" {
		t.Errorf("expected status refunded, got %s", refunded.Status)
	}
	if refunded.RefundAmount != 500 {
		t.Errorf("expected refund amount 500, got %.0f", refunded.RefundAmount)
	}

	// Verify order was cancelled
	var dbOrder order.Order
	if err := db.First(&dbOrder, o.ID).Error; err != nil {
		t.Fatalf("fetch order: %v", err)
	}
	if dbOrder.Status != "cancelled" {
		t.Errorf("expected order status cancelled after refund, got %s", dbOrder.Status)
	}

	// Verify status log was created
	var logs []order.OrderStatusLog
	db.Where("order_id = ?", o.ID).Find(&logs)
	if len(logs) == 0 {
		t.Errorf("expected at least 1 status log")
	}
}

func TestService_Refund_WrongStatus(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{
		OrderID: o.ID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin",
	})

	// Try to refund directly without approve/receive
	_, err := svc.Refund(as.ID, &RefundInput{RefundedBy: "finance", RefundAmount: 100})
	if err == nil {
		t.Fatal("expected error when refunding unprocessed aftersales order")
	}
}

func TestService_Summary(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	o := setupOrder(t, db)
	qty := 1
	svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "A", Status: "pending", CreatedBy: "admin"})
	svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "B", Status: "approved", CreatedBy: "admin"})

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary failed: %v", err)
	}
	if summary.Total != 2 {
		t.Errorf("expected total 2, got %d", summary.Total)
	}
	if summary.ByStatus["pending"] != 1 {
		t.Errorf("expected 1 pending, got %d", summary.ByStatus["pending"])
	}
	if summary.ByStatus["approved"] != 1 {
		t.Errorf("expected 1 approved, got %d", summary.ByStatus["approved"])
	}
}

func TestService_Summary_Empty(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary on empty db failed: %v", err)
	}
	if summary.Total != 0 {
		t.Errorf("expected total 0, got %d", summary.Total)
	}
}
