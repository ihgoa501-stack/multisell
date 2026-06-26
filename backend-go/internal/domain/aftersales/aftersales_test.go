package aftersales

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/allocation"
	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/platform"
	"github.com/lingmirror/backend-go/internal/domain/sku"
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
	if err := db.AutoMigrate(&AfterSalesOrder{}, &order.Order{}, &order.OrderItem{}, &order.OrderStatusLog{}, &inventory.Inventory{}, &inventory.InventoryLog{}, &allocation.Warehouse{}, &platform.Platform{}, &sku.Sku{}); err != nil {
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
	return NewService(db, testLogger(), NewInventoryRestockAdapter(db), NewOrderWriterAdapter(db))
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

func TestService_Approve_InvalidStatus_FromApproved(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)
	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin"})
	svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})
	_, err := svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})
	if err == nil {
		t.Fatal("expected error when approving already approved aftersales order")
	}
}

func TestService_Approve_InvalidStatus_FromReceived(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)
	o := setupOrder(t, db)
	skuID := int64(2001)
	setupInventory(t, db, skuID, 10)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, SkuID: &skuID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin"})
	svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})
	svc.Receive(as.ID, &ReceiveInput{ReceivedBy: "warehouse"})
	_, err := svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})
	if err == nil {
		t.Fatal("expected error when approving received aftersales order")
	}
}

func TestService_Approve_InvalidStatus_FromRejected(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)
	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin"})
	svc.Reject(as.ID, &RejectInput{RejectedBy: "manager", RejectionReason: "超过期限"})
	_, err := svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})
	if err == nil {
		t.Fatal("expected error when approving rejected aftersales order")
	}
}

func TestService_Reject_InvalidStatus_FromReceived(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)
	o := setupOrder(t, db)
	skuID := int64(2002)
	setupInventory(t, db, skuID, 10)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, SkuID: &skuID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin"})
	svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})
	svc.Receive(as.ID, &ReceiveInput{ReceivedBy: "warehouse"})
	_, err := svc.Reject(as.ID, &RejectInput{RejectedBy: "manager", RejectionReason: "已收货不可拒绝"})
	if err == nil {
		t.Fatal("expected error when rejecting received aftersales order")
	}
}

func TestService_Reject_InvalidStatus_FromRefunded(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)
	o := setupOrder(t, db)
	skuID := int64(2003)
	setupInventory(t, db, skuID, 10)
	qty := 1
	amt := 500.0
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, SkuID: &skuID, ReturnQuantity: &qty, Reason: "退货", RefundAmount: &amt, CreatedBy: "admin"})
	svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})
	svc.Receive(as.ID, &ReceiveInput{ReceivedBy: "warehouse"})
	svc.Refund(as.ID, &RefundInput{RefundedBy: "finance", RefundAmount: 500})
	_, err := svc.Reject(as.ID, &RejectInput{RejectedBy: "manager", RejectionReason: "不可拒绝"})
	if err == nil {
		t.Fatal("expected error when rejecting refunded aftersales order")
	}
}

func TestService_Reject_FromApproved(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)
	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin"})
	svc.Approve(as.ID, &ApproveInput{ApprovedBy: "manager", InspectionResult: "OK"})
	rejected, err := svc.Reject(as.ID, &RejectInput{RejectedBy: "manager", RejectionReason: "审批后发现问题"})
	if err != nil {
		t.Fatalf("Reject from approved failed: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Errorf("expected status rejected, got %s", rejected.Status)
	}
}

func TestService_Reject_FromRejected(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db)
	o := setupOrder(t, db)
	qty := 1
	as, _ := svc.Create(&CreateInput{OrderID: o.ID, ReturnQuantity: &qty, Reason: "退货", CreatedBy: "admin"})
	svc.Reject(as.ID, &RejectInput{RejectedBy: "manager", RejectionReason: "首次拒绝"})
	_, err := svc.Reject(as.ID, &RejectInput{RejectedBy: "manager", RejectionReason: "再次拒绝"})
	if err == nil {
		t.Fatal("expected error when rejecting an already rejected aftersales order")
	}
}

// ── ReturnSyncWorker tests ─────────────────────────────────────────

// mockReturnFetcher implements integrations.PlatformAdapter with
// controlled return data for testing.
type mockReturnFetcher struct {
	returns []*integrations.PlatformReturn
}

func (m *mockReturnFetcher) Publish(_ context.Context, _ *integrations.PublishInput) (*integrations.PublishResult, error) {
	return nil, nil
}
func (m *mockReturnFetcher) SyncStatus(_ context.Context, _ *integrations.SyncStatusInput) (string, error) {
	return "", nil
}
func (m *mockReturnFetcher) ValidateCredentials(_ context.Context, _ int64) (bool, error) {
	return false, nil
}
func (m *mockReturnFetcher) SyncInventory(_ context.Context, _ *integrations.SyncInventoryInput) (bool, error) {
	return false, nil
}
func (m *mockReturnFetcher) PushTracking(_ context.Context, _ *integrations.PushTrackingInput) (bool, error) {
	return false, nil
}
func (m *mockReturnFetcher) FetchOrders(_ context.Context, _ *integrations.FetchOrdersInput) ([]*integrations.PlatformOrder, error) {
	return nil, nil
}
func (m *mockReturnFetcher) FetchSettlements(_ context.Context, _ *integrations.FetchSettlementsInput) ([]*integrations.PlatformSettlement, error) {
	return nil, nil
}
func (m *mockReturnFetcher) FetchReturns(_ context.Context, _ *integrations.FetchReturnsInput) ([]*integrations.PlatformReturn, error) {
	return m.returns, nil
}

// setupSyncDB creates an in-memory SQLite database with all models needed
// for return sync worker tests.
func setupSyncDB(t *testing.T) *gorm.DB {
	t.Helper()
	testDBCounter++
	dsn := "file:sync_test_" + itoa(testDBCounter) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AfterSalesOrder{}, &order.Order{}, &order.OrderItem{}, &platform.Platform{}, &sku.Sku{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func TestReturnSyncWorker_SyncOnce_CreatesReturn(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	// Register mock adapter and clean up after test.
	platformCode := "test_create_return"
	mock := &mockReturnFetcher{
		returns: []*integrations.PlatformReturn{
			{
				ReturnID:     "ret-001",
				OrderSN:      "ORD-TEST-001",
				SkuCode:      "SKU-TEST-001",
				Quantity:     2,
				Reason:       "Defective",
				Status:       "pending",
				RefundAmount: "29.99",
			},
		},
	}
	integrations.RegisterAdapter(platformCode, mock)

	// Create platform (status=1 = active).
	if err := db.Create(&platform.Platform{Code: platformCode, Name: "Test Platform", Status: 1}).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}

	// Create order.
	if err := db.Create(&order.Order{OrderNo: "ORD-TEST-001", Status: "paid", RecipientName: "Test"}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Create SKU.
	if err := db.Create(&sku.Sku{Code: "SKU-TEST-001", Status: 1}).Error; err != nil {
		t.Fatalf("create sku: %v", err)
	}

	w := NewReturnSyncWorker(db, log)
	w.syncOnce()

	var asos []AfterSalesOrder
	if err := db.Find(&asos).Error; err != nil {
		t.Fatalf("query aftersales orders: %v", err)
	}
	if len(asos) != 1 {
		t.Fatalf("expected 1 aftersales order, got %d", len(asos))
	}

	if asos[0].ReturnQuantity != 2 {
		t.Errorf("expected return_quantity 2, got %d", asos[0].ReturnQuantity)
	}
	if asos[0].Reason != "Defective" {
		t.Errorf("expected reason 'Defective', got %q", asos[0].Reason)
	}
	if asos[0].Status != "pending" {
		t.Errorf("expected status 'pending', got %q", asos[0].Status)
	}
	if asos[0].CreatedBy != "system" {
		t.Errorf("expected created_by 'system', got %q", asos[0].CreatedBy)
	}
	if asos[0].RefundAmount != 29.99 {
		t.Errorf("expected refund_amount 29.99, got %f", asos[0].RefundAmount)
	}
}

func TestReturnSyncWorker_DedupSameReturn(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	platformCode := "test_dedup"
	mock := &mockReturnFetcher{
		returns: []*integrations.PlatformReturn{
			{ReturnID: "ret-002", OrderSN: "ORD-DEDUP", SkuCode: "SKU-DEDUP", Quantity: 1},
		},
	}
	integrations.RegisterAdapter(platformCode, mock)

	if err := db.Create(&platform.Platform{Code: platformCode, Name: "Dedup", Status: 1}).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}
	if err := db.Create(&order.Order{OrderNo: "ORD-DEDUP", Status: "paid", RecipientName: "Dedup"}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := db.Create(&sku.Sku{Code: "SKU-DEDUP", Status: 1}).Error; err != nil {
		t.Fatalf("create sku: %v", err)
	}

	w := NewReturnSyncWorker(db, log)

	// First sync → creates the record.
	w.syncOnce()

	var firstCount int64
	db.Model(&AfterSalesOrder{}).Count(&firstCount)
	if firstCount != 1 {
		t.Fatalf("expected 1 record after first sync, got %d", firstCount)
	}

	// Second sync → should NOT create a duplicate (dedup).
	w.syncOnce()

	var secondCount int64
	db.Model(&AfterSalesOrder{}).Count(&secondCount)
	if secondCount != 1 {
		t.Errorf("expected 1 record after second sync (dedup), got %d", secondCount)
	}
}

func TestReturnSyncWorker_MissingOrder(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	platformCode := "test_missing_order"
	mock := &mockReturnFetcher{
		returns: []*integrations.PlatformReturn{
			{ReturnID: "ret-003", OrderSN: "ORD-DOES-NOT-EXIST", SkuCode: "SKU-TEST", Quantity: 1},
		},
	}
	integrations.RegisterAdapter(platformCode, mock)

	if err := db.Create(&platform.Platform{Code: platformCode, Name: "Missing Order", Status: 1}).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}
	// NOTE: no order is created — the return should be skipped.

	// Create SKU that exists.
	if err := db.Create(&sku.Sku{Code: "SKU-TEST", Status: 1}).Error; err != nil {
		t.Fatalf("create sku: %v", err)
	}

	w := NewReturnSyncWorker(db, log)
	w.syncOnce()

	var count int64
	db.Model(&AfterSalesOrder{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 aftersales orders (order not found), got %d", count)
	}
}

func TestReturnSyncWorker_MissingSku(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	platformCode := "test_missing_sku"
	mock := &mockReturnFetcher{
		returns: []*integrations.PlatformReturn{
			{ReturnID: "ret-004", OrderSN: "ORD-TEST-004", SkuCode: "SKU-DOES-NOT-EXIST", Quantity: 1},
		},
	}
	integrations.RegisterAdapter(platformCode, mock)

	if err := db.Create(&platform.Platform{Code: platformCode, Name: "Missing SKU", Status: 1}).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}
	if err := db.Create(&order.Order{OrderNo: "ORD-TEST-004", Status: "paid", RecipientName: "Test"}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}
	// NOTE: no SKU is created — the return should be skipped.

	w := NewReturnSyncWorker(db, log)
	w.syncOnce()

	var count int64
	db.Model(&AfterSalesOrder{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 aftersales orders (sku not found), got %d", count)
	}
}

func TestReturnSyncWorker_NoActivePlatforms(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	// Create a platform with status=0 (inactive).
	if err := db.Create(&platform.Platform{Code: "inactive", Name: "Inactive", Status: 0}).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}

	w := NewReturnSyncWorker(db, log)
	w.syncOnce()
	// Should not panic — clean exit because no active platforms.
}

func TestReturnSyncWorker_NoAdapterRegistered(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	// Create an active platform but DO NOT register an adapter for it.
	if err := db.Create(&platform.Platform{Code: "no-adapter", Name: "No Adapter", Status: 1}).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}

	w := NewReturnSyncWorker(db, log)
	w.syncOnce()
	// Should not panic — clean skip because no adapter registered.
}

func TestReturnSyncWorker_DefaultReasonAndQuantity(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	platformCode := "test_defaults"
	mock := &mockReturnFetcher{
		returns: []*integrations.PlatformReturn{
			{
				ReturnID: "ret-005",
				OrderSN:  "ORD-DEF-005",
				SkuCode:  "SKU-DEF-005",
				// Quantity defaults to 0 — worker should set to 1.
				// Reason is empty — worker should use default.
				RefundAmount: "invalid-amount", // should be parsed safely.
			},
		},
	}
	integrations.RegisterAdapter(platformCode, mock)

	if err := db.Create(&platform.Platform{Code: platformCode, Name: "Defaults", Status: 1}).Error; err != nil {
		t.Fatalf("create platform: %v", err)
	}
	if err := db.Create(&order.Order{OrderNo: "ORD-DEF-005", Status: "paid", RecipientName: "Defaults"}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := db.Create(&sku.Sku{Code: "SKU-DEF-005", Status: 1}).Error; err != nil {
		t.Fatalf("create sku: %v", err)
	}

	w := NewReturnSyncWorker(db, log)
	w.syncOnce()

	var asos []AfterSalesOrder
	db.Find(&asos)
	if len(asos) != 1 {
		t.Fatalf("expected 1 aftersales order, got %d", len(asos))
	}
	if asos[0].ReturnQuantity != 1 {
		t.Errorf("expected default return_quantity 1, got %d", asos[0].ReturnQuantity)
	}
	if asos[0].Reason != "平台发起退货" {
		t.Errorf("expected default reason '平台发起退货', got %q", asos[0].Reason)
	}
	if asos[0].RefundAmount != 0 {
		t.Errorf("expected refund_amount 0 for invalid string, got %f", asos[0].RefundAmount)
	}
}

func TestReturnSyncWorker_MultiplePlatforms(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	platformCode1 := "test_multi_1"
	platformCode2 := "test_multi_2"
	mock1 := &mockReturnFetcher{
		returns: []*integrations.PlatformReturn{
			{ReturnID: "ret-m1", OrderSN: "ORD-M1", SkuCode: "SKU-M1", Quantity: 1, Reason: "Platform 1"},
		},
	}
	mock2 := &mockReturnFetcher{
		returns: []*integrations.PlatformReturn{
			{ReturnID: "ret-m2", OrderSN: "ORD-M2", SkuCode: "SKU-M2", Quantity: 1, Reason: "Platform 2"},
		},
	}
	integrations.RegisterAdapter(platformCode1, mock1)
	integrations.RegisterAdapter(platformCode2, mock2)

	db.Create(&platform.Platform{Code: platformCode1, Name: "P1", Status: 1})
	db.Create(&platform.Platform{Code: platformCode2, Name: "P2", Status: 1})
	db.Create(&order.Order{OrderNo: "ORD-M1", Status: "paid", RecipientName: "M1"})
	db.Create(&order.Order{OrderNo: "ORD-M2", Status: "paid", RecipientName: "M2"})
	db.Create(&sku.Sku{Code: "SKU-M1", Status: 1})
	db.Create(&sku.Sku{Code: "SKU-M2", Status: 1})

	w := NewReturnSyncWorker(db, log)
	w.syncOnce()

	var total int64
	db.Model(&AfterSalesOrder{}).Count(&total)
	if total != 2 {
		t.Errorf("expected 2 aftersales orders (1 per platform), got %d", total)
	}

	var asos []AfterSalesOrder
	db.Find(&asos)

	reasons := make(map[string]bool)
	for _, a := range asos {
		reasons[a.Reason] = true
	}
	if !reasons["Platform 1"] {
		t.Error("missing reason 'Platform 1'")
	}
	if !reasons["Platform 2"] {
		t.Error("missing reason 'Platform 2'")
	}
}

func TestReturnSyncWorker_WithItemIDLookup(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	platformCode := "test_item_lookup"
	mock := &mockReturnFetcher{
		returns: []*integrations.PlatformReturn{
			{ReturnID: "ret-item", OrderSN: "ORD-ITEM", SkuCode: "SKU-ITEM", Quantity: 1},
		},
	}
	integrations.RegisterAdapter(platformCode, mock)

	db.Create(&platform.Platform{Code: platformCode, Name: "ItemLookup", Status: 1})

	ord := order.Order{OrderNo: "ORD-ITEM", Status: "paid", RecipientName: "Item"}
	db.Create(&ord)

	s := sku.Sku{Code: "SKU-ITEM", Status: 1}
	db.Create(&s)

	// Create an OrderItem so the worker can look up ItemID.
	item := order.OrderItem{OrderID: ord.ID, SkuID: s.ID, SkuCode: "SKU-ITEM", ProductID: 1, ProductName: "Item", UnitPrice: 100, Quantity: 1, Subtotal: 100}
	db.Create(&item)

	w := NewReturnSyncWorker(db, log)
	w.syncOnce()

	var asos []AfterSalesOrder
	db.Find(&asos)
	if len(asos) != 1 {
		t.Fatalf("expected 1 aftersales order, got %d", len(asos))
	}
	if asos[0].ItemID == nil {
		t.Fatal("expected ItemID to be set from OrderItem lookup")
	}
	if *asos[0].ItemID != item.ID {
		t.Errorf("expected ItemID %d, got %d", item.ID, *asos[0].ItemID)
	}
	if asos[0].SkuID == nil {
		t.Fatal("expected SkuID to be set")
	}
	if *asos[0].SkuID != s.ID {
		t.Errorf("expected SkuID %d, got %d", s.ID, *asos[0].SkuID)
	}
}

func TestReturnSyncWorker_StartStop(t *testing.T) {
	db := setupSyncDB(t)
	log := testLogger()

	platformCode := "test_start_stop"
	mock := &mockReturnFetcher{
		returns: []*integrations.PlatformReturn{
			{ReturnID: "ret-ss", OrderSN: "ORD-SS", SkuCode: "SKU-SS", Quantity: 1},
		},
	}
	integrations.RegisterAdapter(platformCode, mock)

	db.Create(&platform.Platform{Code: platformCode, Name: "SS", Status: 1})
	db.Create(&order.Order{OrderNo: "ORD-SS", Status: "paid", RecipientName: "SS"})
	db.Create(&sku.Sku{Code: "SKU-SS", Status: 1})

	w := NewReturnSyncWorker(db, log).WithInterval(100 * time.Millisecond)
	w.Start()
	time.Sleep(300 * time.Millisecond)
	w.Stop()

	var count int64
	db.Model(&AfterSalesOrder{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 aftersales order after Start/Stop, got %d", count)
	}
}

func TestReturnSyncWorker_FetchReturnsError(t *testing.T) {
	db := setupSyncDB(t)
	log := zap.NewNop()

	platformCode := "test_fetch_err"
	mock := &fetchErrAdapter{}
	mock.err = strconv.ErrSyntax // any non-nil error will do

	integrations.RegisterAdapter(platformCode, mock)

	db.Create(&platform.Platform{Code: platformCode, Name: "FetchError", Status: 1})

	w := NewReturnSyncWorker(db, log)
	w.syncOnce()
	// Should not panic — error is logged and the platform is skipped.

	var count int64
	db.Model(&AfterSalesOrder{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 aftersales orders after FetchReturns error, got %d", count)
	}
}

// fetchErrAdapter returns an error from FetchReturns.
type fetchErrAdapter struct {
	err error
}

func (m *fetchErrAdapter) Publish(_ context.Context, _ *integrations.PublishInput) (*integrations.PublishResult, error) { return nil, nil }
func (m *fetchErrAdapter) SyncStatus(_ context.Context, _ *integrations.SyncStatusInput) (string, error) { return "", nil }
func (m *fetchErrAdapter) ValidateCredentials(_ context.Context, _ int64) (bool, error) { return false, nil }
func (m *fetchErrAdapter) SyncInventory(_ context.Context, _ *integrations.SyncInventoryInput) (bool, error) { return false, nil }
func (m *fetchErrAdapter) PushTracking(_ context.Context, _ *integrations.PushTrackingInput) (bool, error) { return false, nil }
func (m *fetchErrAdapter) FetchOrders(_ context.Context, _ *integrations.FetchOrdersInput) ([]*integrations.PlatformOrder, error) { return nil, nil }
func (m *fetchErrAdapter) FetchSettlements(_ context.Context, _ *integrations.FetchSettlementsInput) ([]*integrations.PlatformSettlement, error) { return nil, nil }
func (m *fetchErrAdapter) FetchReturns(_ context.Context, _ *integrations.FetchReturnsInput) ([]*integrations.PlatformReturn, error) { return nil, m.err }
