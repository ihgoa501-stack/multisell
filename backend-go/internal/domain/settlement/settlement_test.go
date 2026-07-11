package settlement

import (
	"testing"
	"time"

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
	dsn := "file:settlement_test_" + itoa(testDBCounter) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Settlement{}, &SettlementItem{}); err != nil {
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

func TestService_Create_Full(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	now := time.Now()
	pid := int64(42)
	rev := 1000.0
	fee := 50.0
	ref := 10.0
	net := 940.0
	qty := 2

	st, err := svc.Create(&CreateSettlementInput{
		PlatformID:   &pid,
		SettlementNo: "SETTLE-001",
		PeriodStart:  &now,
		PeriodEnd:    &now,
		Currency:     "USD",
		TotalRevenue: &rev,
		TotalFee:     &fee,
		TotalRefund:  &ref,
		TotalNet:     &net,
		Status:       "pending",
		Items: []SettlementItemInput{
			{TransactionType: "sale", OrderNo: "ORD-001", Amount: &rev, Fee: &fee, Net: &net, Quantity: &qty, OccurredAt: &now},
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if st.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if st.SettlementNo != "SETTLE-001" {
		t.Errorf("expected SETTLE-001, got %s", st.SettlementNo)
	}
	if st.TotalRevenue != rev {
		t.Errorf("expected revenue %.0f, got %.0f", rev, st.TotalRevenue)
	}
	if st.TotalFee != fee {
		t.Errorf("expected fee %.0f, got %.0f", fee, st.TotalFee)
	}
	if st.TotalNet != net {
		t.Errorf("expected net %.0f, got %.0f", net, st.TotalNet)
	}

	// Verify items persisted
	detail, err := svc.Get(st.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(detail.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(detail.Items))
	}
	if detail.Items[0].OrderNo != "ORD-001" {
		t.Errorf("expected ORD-001, got %s", detail.Items[0].OrderNo)
	}
	if detail.Items[0].Amount != rev {
		t.Errorf("expected amount %.0f, got %.0f", rev, detail.Items[0].Amount)
	}
}

func TestService_Create_Defaults(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	st, err := svc.Create(&CreateSettlementInput{
		PlatformID:   &pid,
		SettlementNo: "SETTLE-DEFAULT",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if st.Currency != "CNY" {
		t.Errorf("expected default currency CNY, got %s", st.Currency)
	}
	if st.Status != "pending" {
		t.Errorf("expected default status pending, got %s", st.Status)
	}
}

func TestService_Create_NoItems(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	st, err := svc.Create(&CreateSettlementInput{
		PlatformID:   &pid,
		SettlementNo: "SETTLE-NO-ITEMS",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	detail, err := svc.Get(st.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(detail.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(detail.Items))
	}
}

func TestService_Create_Validation(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	// Missing platform_id
	_, err := svc.Create(&CreateSettlementInput{
		SettlementNo: "SETTLE-NO-PLATFORM",
	})
	if err == nil {
		t.Fatal("expected error for missing platform_id")
	}

	// Missing settlement_no
	pid := int64(1)
	_, err = svc.Create(&CreateSettlementInput{
		PlatformID: &pid,
	})
	if err == nil {
		t.Fatal("expected error for missing settlement_no")
	}

	_, err = svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-BYPASS", Status: "reconciled",
	})
	if err == nil {
		t.Fatal("expected create to reject a reconciliation status bypass")
	}
}

func TestService_Get_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	_, err := svc.Get(99999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestService_List_Pagination(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	for i := 0; i < 5; i++ {
		no := "SETTLE-LIST-" + itoa(int64(i+1))
		_, err := svc.Create(&CreateSettlementInput{
			PlatformID: &pid, SettlementNo: no,
		})
		if err != nil {
			t.Fatalf("create %d failed: %v", i, err)
		}
	}

	// Page 1: size=2
	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 2}, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	// Page 3: size=2 (last page)
	items, total, err = svc.List(&common.Pagination{Page: 3, Size: 2}, nil)
	if err != nil {
		t.Fatalf("List page 3 failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item on last page, got %d", len(items))
	}
}

func TestService_List_FilterByStatus(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	st1, _ := svc.Create(&CreateSettlementInput{PlatformID: &pid, SettlementNo: "S-PENDING", Status: "pending"})
	svc.Create(&CreateSettlementInput{PlatformID: &pid, SettlementNo: "S-RECONCILED", Status: "reconciled"})

	_ = st1
	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 10}, &SettlementListFilter{Status: "pending"})
	if err != nil {
		t.Fatalf("List filter failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 pending settlement, got %d", total)
	}
	if len(items) != 1 || items[0].SettlementNo != "S-PENDING" {
		t.Errorf("expected S-PENDING, got %s", items[0].SettlementNo)
	}
}

func TestService_List_FilterBySearch(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	svc.Create(&CreateSettlementInput{PlatformID: &pid, SettlementNo: "OZON-JUNE"})
	svc.Create(&CreateSettlementInput{PlatformID: &pid, SettlementNo: "SHOPEE-MAY"})

	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 10}, &SettlementListFilter{Search: "OZON"})
	if err != nil {
		t.Fatalf("List search failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 result for search OZON, got %d", total)
	}
	if len(items) != 1 || items[0].SettlementNo != "OZON-JUNE" {
		t.Errorf("expected OZON-JUNE, got %s", items[0].SettlementNo)
	}
}

func TestService_List_FilterByPlatform(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid1 := int64(10)
	pid2 := int64(20)
	svc.Create(&CreateSettlementInput{PlatformID: &pid1, SettlementNo: "P01"})
	svc.Create(&CreateSettlementInput{PlatformID: &pid2, SettlementNo: "P02"})

	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 10}, &SettlementListFilter{PlatformID: &pid1})
	if err != nil {
		t.Fatalf("List platform filter failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 result for platform 10, got %d", total)
	}
	if len(items) != 1 || items[0].SettlementNo != "P01" {
		t.Errorf("expected P01, got %s", items[0].SettlementNo)
	}
}

func TestService_Update(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-UPDATE",
	})

	newStatus := "closed"
	if _, err := svc.Update(st.ID, &UpdateSettlementInput{Status: &newStatus}); err == nil {
		t.Fatal("pending settlement must not bypass reconciliation and close directly")
	}
}

func TestService_Update_NoChanges(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-NOOP",
	})

	updated, err := svc.Update(st.ID, &UpdateSettlementInput{})
	if err != nil {
		t.Fatalf("Update with no changes failed: %v", err)
	}
	if updated.SettlementNo != "SETTLE-NOOP" {
		t.Errorf("expected unchanged settlement_no")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	status := "closed"
	_, err := svc.Update(99999, &UpdateSettlementInput{Status: &status})
	if err == nil {
		t.Fatal("expected error for not found update")
	}
}

func TestService_Delete(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	rev := 500.0
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-DEL",
		Items: []SettlementItemInput{{TransactionType: "sale", Amount: &rev}},
	})

	if err := svc.Delete(st.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err := svc.Get(st.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}

	// Items should be cascade deleted
	var count int64
	db.Model(&SettlementItem{}).Where("settlement_id = ?", st.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 items after cascade delete, got %d", count)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	err := svc.Delete(99999)
	if err == nil {
		t.Fatal("expected error for not found delete")
	}
}

func TestService_Reconcile_SingleItem(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	rev := 200.0
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-REC",
		Items: []SettlementItemInput{{TransactionType: "sale", Amount: &rev}},
	})

	detail, _ := svc.Get(st.ID)
	itemID := detail.Items[0].ID

	err := svc.Reconcile(st.ID, &ReconcileInput{
		ItemID:               &itemID,
		ReconciliationStatus: "matched",
		ReconciliationNote:   "一切正常",
		ReconciledBy:         "tester",
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify item status
	detail, _ = svc.Get(st.ID)
	if detail.Items[0].ReconciliationStatus != "matched" {
		t.Errorf("expected item status matched, got %s", detail.Items[0].ReconciliationStatus)
	}
	if detail.Items[0].ReconciliationNote != "一切正常" {
		t.Errorf("expected note, got %s", detail.Items[0].ReconciliationNote)
	}
	if detail.Items[0].ReconciledBy != "tester" {
		t.Errorf("expected reconciled_by tester, got %s", detail.Items[0].ReconciledBy)
	}

	// Settlement should be reconciled when all items done
	updatedDetail, _ := svc.Get(st.ID)
	if updatedDetail.Settlement.Status != "reconciled" {
		t.Errorf("expected settlement status reconciled after all items matched, got %s", updatedDetail.Settlement.Status)
	}
}

func TestService_Reconcile_BatchAllPending(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	rev := 100.0
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-BATCH",
		Items: []SettlementItemInput{
			{TransactionType: "sale", Amount: &rev},
			{TransactionType: "refund", Amount: &rev},
		},
	})

	// Reconcile all pending items (omit ItemID)
	err := svc.Reconcile(st.ID, &ReconcileInput{
		ReconciliationStatus: "matched",
		ReconciledBy:         "tester",
	})
	if err != nil {
		t.Fatalf("Batch reconcile failed: %v", err)
	}

	detail, _ := svc.Get(st.ID)
	for _, item := range detail.Items {
		if item.ReconciliationStatus != "matched" {
			t.Errorf("expected all items matched, got item %d: %s", item.ID, item.ReconciliationStatus)
		}
	}
	if detail.Settlement.Status != "reconciled" {
		t.Errorf("expected settlement reconciled, got %s", detail.Settlement.Status)
	}
}

func TestService_Reconcile_Partial(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	rev := 100.0
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-PARTIAL",
		Items: []SettlementItemInput{
			{TransactionType: "sale", Amount: &rev},
			{TransactionType: "refund", Amount: &rev},
		},
	})

	detail, _ := svc.Get(st.ID)
	itemID := detail.Items[0].ID

	// Reconcile only first item
	svc.Reconcile(st.ID, &ReconcileInput{
		ItemID: &itemID, ReconciliationStatus: "matched", ReconciledBy: "tester",
	})

	// Settlement should now be "reconciling" (one item still pending)
	updated, _ := svc.Get(st.ID)
	if updated.Settlement.Status != "reconciling" {
		t.Errorf("expected reconciling after partial match, got %s", updated.Settlement.Status)
	}
}

func TestService_AddItem(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-ADD-ITEM",
	})

	amount := 300.0
	item, err := svc.AddItem(st.ID, &AddItemInput{
		TransactionType: "adjustment",
		OrderNo:         "ORD-ADJ",
		Amount:          amount,
		Net:             amount,
	})
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}
	if item.SettlementID != st.ID {
		t.Errorf("expected settlement_id %d, got %d", st.ID, item.SettlementID)
	}
	if item.ReconciliationStatus != "pending" {
		t.Errorf("expected default status pending, got %s", item.ReconciliationStatus)
	}

	// Verify item count
	items, err := svc.ListItems(st.ID, "")
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestService_AddItem_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	_, err := svc.AddItem(99999, &AddItemInput{TransactionType: "sale"})
	if err == nil {
		t.Fatal("expected error for non-existent settlement")
	}
}

func TestService_ListItems_FilterByStatus(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	rev := 100.0
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-FILTER-ITEMS",
		Items: []SettlementItemInput{
			{TransactionType: "sale", Amount: &rev},
			{TransactionType: "refund", Amount: &rev},
		},
	})

	// By default both are pending
	items, err := svc.ListItems(st.ID, "pending")
	if err != nil {
		t.Fatalf("ListItems filter failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 pending items, got %d", len(items))
	}

	items, err = svc.ListItems(st.ID, "matched")
	if err != nil {
		t.Fatalf("ListItems filter failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 matched items, got %d", len(items))
	}
}

func TestService_UpdateItemReconciliation(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	rev := 100.0
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-UPD-ITEM",
		Items: []SettlementItemInput{{TransactionType: "sale", Amount: &rev}},
	})

	detail, _ := svc.Get(st.ID)
	itemID := detail.Items[0].ID

	item, err := svc.UpdateItemReconciliation(itemID, &UpdateReconciliationInput{
		ReconciliationStatus: "discrepancy",
		ReconciliationNote:   "金额不符",
		ReconciledBy:         "auditor",
	})
	if err != nil {
		t.Fatalf("UpdateItemReconciliation failed: %v", err)
	}
	if item.ReconciliationStatus != "discrepancy" {
		t.Errorf("expected discrepancy, got %s", item.ReconciliationStatus)
	}
	if item.ReconciliationNote != "金额不符" {
		t.Errorf("expected note '金额不符', got %s", item.ReconciliationNote)
	}
}

func TestService_UpdateItemReconciliation_InvalidStatus(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid := int64(1)
	rev := 100.0
	st, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid, SettlementNo: "SETTLE-INVALID",
		Items: []SettlementItemInput{{TransactionType: "sale", Amount: &rev}},
	})

	detail, _ := svc.Get(st.ID)
	itemID := detail.Items[0].ID

	_, err := svc.UpdateItemReconciliation(itemID, &UpdateReconciliationInput{
		ReconciliationStatus: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for invalid reconciliation status")
	}
}

func TestService_Summary(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	pid1 := int64(10)
	pid2 := int64(20)
	rev1 := 1000.0
	rev2 := 2000.0
	net1 := 800.0
	net2 := 1500.0

	s1, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid1, SettlementNo: "S1", TotalRevenue: &rev1, TotalFee: floatPtr(100), TotalRefund: floatPtr(50), TotalNet: &net1,
	})
	svc.Create(&CreateSettlementInput{
		PlatformID: &pid1, SettlementNo: "S2", TotalRevenue: &rev1, TotalFee: floatPtr(100), TotalRefund: floatPtr(50), TotalNet: &net1, Status: "pending",
	})
	s3, _ := svc.Create(&CreateSettlementInput{
		PlatformID: &pid2, SettlementNo: "S3", TotalRevenue: &rev2, TotalFee: floatPtr(200), TotalRefund: floatPtr(100), TotalNet: &net2,
	})
	db.Model(&Settlement{}).Where("id IN ?", []int64{s1.ID, s3.ID}).Update("status", "reconciled")

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary failed: %v", err)
	}
	if summary.Total != 3 {
		t.Errorf("expected total 3, got %d", summary.Total)
	}
	if summary.ByStatus["reconciled"] != 2 {
		t.Errorf("expected 2 reconciled, got %d", summary.ByStatus["reconciled"])
	}
	if summary.ByStatus["pending"] != 1 {
		t.Errorf("expected 1 pending, got %d", summary.ByStatus["pending"])
	}
	// net_by_platform should have 2 entries
	if len(summary.NetByPlatform) != 2 {
		t.Errorf("expected 2 platform net totals, got %d", len(summary.NetByPlatform))
	}
}

func TestService_Summary_Empty(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary on empty db failed: %v", err)
	}
	if summary.Total != 0 {
		t.Errorf("expected total 0, got %d", summary.Total)
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
