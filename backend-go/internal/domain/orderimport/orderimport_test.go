package orderimport

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, err := svc.Create(&CreateInput{
		FileName:   "orders_20260601.csv",
		SourceType: "manual",
		CreatedBy:  "test_user",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if o.ID == 0 {
		t.Fatal("ID should be set")
	}
	if o.FileName != "orders_20260601.csv" {
		t.Fatalf("FileName = %s", o.FileName)
	}
	if o.Status != "pending" {
		t.Fatalf("Status = %s", o.Status)
	}

	got, err := svc.Get(o.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.FileName != "orders_20260601.csv" {
		t.Fatalf("FileName = %s", got.FileName)
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "test.csv"})
	totalRows := 100
	updated, err := svc.Update(o.ID, &UpdateInput{
		TotalRows: &totalRows,
		Status:    dbtest.StringPtr("processing"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.TotalRows != 100 {
		t.Fatalf("TotalRows = %d", updated.TotalRows)
	}
	if updated.Status != "processing" {
		t.Fatalf("Status = %s", updated.Status)
	}
}

func TestService_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{FileName: "a.csv"})
	svc.Create(&CreateInput{FileName: "b.csv", Status: "completed"})
	svc.Create(&CreateInput{FileName: "c.csv", Status: "completed"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	items, total, err = svc.List(&p, &ListFilter{Status: "completed"})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 completed, got %d", total)
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "del.csv"})
	if err := svc.Delete(o.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.Get(o.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_ProcessAndComplete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	o, _ := svc.Create(&CreateInput{FileName: "proc.csv"})

	proc, err := svc.Process(o.ID)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if proc.Status != "processing" {
		t.Fatalf("Status = %s (expected processing)", proc.Status)
	}

	comp, err := svc.Complete(o.ID, &CompleteInput{
		SuccessCount: 80,
		ErrorCount:   20,
		Status:       "completed",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if comp.SuccessCount != 80 {
		t.Fatalf("SuccessCount = %d", comp.SuccessCount)
	}
	if comp.ErrorCount != 20 {
		t.Fatalf("ErrorCount = %d", comp.ErrorCount)
	}
	if comp.Status != "completed" {
		t.Fatalf("Status = %s", comp.Status)
	}
}

func TestService_Summary(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImport{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{FileName: "a.csv", Status: "completed", TotalRows: dbtest.IntPtr(50)})
	svc.Create(&CreateInput{FileName: "b.csv", Status: "processing", TotalRows: dbtest.IntPtr(30)})

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Total != 2 {
		t.Fatalf("Total = %d", summary.Total)
	}
	if summary.ByStatus["completed"] != 1 {
		t.Fatalf("completed count = %d", summary.ByStatus["completed"])
	}
}

func TestService_ImportSyncStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportSyncStatus{}, &OrderImportBatch{}, &OrderImportItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create a batch first
	batch := OrderImportBatch{
		AdapterCode:    "test_adapter",
		Platform:       "test_platform",
		StoreName:      "test_store",
		SourceFilename: "sync_orders.csv",
		ImportedBy:     "test_user",
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatalf("create batch: %v", err)
	}

	// Initial status: empty
	statuses, err := svc.GetSyncStatus()
	if err != nil {
		t.Fatalf("GetSyncStatus (initial): %v", err)
	}
	if len(statuses) != 0 {
		t.Fatalf("expected 0 statuses, got %d", len(statuses))
	}

	// Sync orders
	orders := []SyncOrderInput{
		{Platform: "test_platform", PlatformOrderNo: "PO-001", SkuCode: "SKU-001", Quantity: 2},
		{Platform: "test_platform", PlatformOrderNo: "PO-002", SkuCode: "SKU-002", Quantity: 1},
	}
	result, err := svc.SyncOrders(batch.ID, orders)
	if err != nil {
		t.Fatalf("SyncOrders: %v", err)
	}
	if result.CreatedCount != 2 {
		t.Fatalf("CreatedCount = %d (expected 2)", result.CreatedCount)
	}
	if result.SkippedCount != 0 {
		t.Fatalf("SkippedCount = %d (expected 0)", result.SkippedCount)
	}
	if result.LastSyncResult != "success" {
		t.Fatalf("LastSyncResult = %s (expected success)", result.LastSyncResult)
	}

	// SyncOrders already called UpsertSyncStatus internally — verify the recorded status
	statuses, err = svc.GetSyncStatus()
	if err != nil {
		t.Fatalf("GetSyncStatus (after sync): %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Platform != "test_platform" {
		t.Fatalf("Platform = %s", statuses[0].Platform)
	}
	if statuses[0].LastSyncResult != "success" {
		t.Fatalf("LastSyncResult = %s", statuses[0].LastSyncResult)
	}
	if statuses[0].OrderCount != 2 {
		t.Fatalf("OrderCount = %d (expected 2)", statuses[0].OrderCount)
	}
}

func TestService_IdempotentSync(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportSyncStatus{}, &OrderImportBatch{}, &OrderImportItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := OrderImportBatch{
		AdapterCode:    "test_adapter",
		Platform:       "idempotent_test",
		StoreName:      "test_store",
		SourceFilename: "dup_orders.csv",
		ImportedBy:     "test_user",
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatalf("create batch: %v", err)
	}

	// First sync: insert 3 orders
	orders := []SyncOrderInput{
		{Platform: "idempotent_test", PlatformOrderNo: "PO-010", SkuCode: "SKU-010", Quantity: 1},
		{Platform: "idempotent_test", PlatformOrderNo: "PO-011", SkuCode: "SKU-011", Quantity: 2},
		{Platform: "idempotent_test", PlatformOrderNo: "PO-012", SkuCode: "SKU-012", Quantity: 3},
	}
	result, err := svc.SyncOrders(batch.ID, orders)
	if err != nil {
		t.Fatalf("SyncOrders (first): %v", err)
	}
	if result.CreatedCount != 3 {
		t.Fatalf("first sync created %d (expected 3)", result.CreatedCount)
	}

	// Second sync: mix of duplicates and new orders
	orders2 := []SyncOrderInput{
		{Platform: "idempotent_test", PlatformOrderNo: "PO-010", SkuCode: "SKU-010", Quantity: 1}, // duplicate
		{Platform: "idempotent_test", PlatformOrderNo: "PO-013", SkuCode: "SKU-013", Quantity: 4}, // new
		{Platform: "idempotent_test", PlatformOrderNo: "PO-011", SkuCode: "SKU-011", Quantity: 2}, // duplicate
	}
	result2, err := svc.SyncOrders(batch.ID, orders2)
	if err != nil {
		t.Fatalf("SyncOrders (second): %v", err)
	}
	if result2.CreatedCount != 1 {
		t.Fatalf("second sync created %d (expected 1)", result2.CreatedCount)
	}
	if result2.SkippedCount != 2 {
		t.Fatalf("second sync skipped %d (expected 2)", result2.SkippedCount)
	}

	// Verify item count: only 4 unique orders
	var totalItems int64
	db.Model(&OrderImportItem{}).Count(&totalItems)
	if totalItems != 4 {
		t.Fatalf("total items = %d (expected 4)", totalItems)
	}
}

func TestService_SyncPartialFailure(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportSyncStatus{}, &OrderImportBatch{}, &OrderImportItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := OrderImportBatch{
		AdapterCode:    "test_adapter",
		Platform:       "partial_test",
		StoreName:      "test_store",
		SourceFilename: "partial_orders.csv",
		ImportedBy:     "test_user",
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatalf("create batch: %v", err)
	}

	// Sync a valid order first
	orders := []SyncOrderInput{
		{Platform: "partial_test", PlatformOrderNo: "PO-020", SkuCode: "SKU-020", Quantity: 1},
	}
	result, err := svc.SyncOrders(batch.ID, orders)
	if err != nil {
		t.Fatalf("SyncOrders: %v", err)
	}
	if result.CreatedCount != 1 {
		t.Fatalf("CreatedCount = %d", result.CreatedCount)
	}
	if result.LastSyncResult != "success" {
		t.Fatalf("LastSyncResult = %s (expected success)", result.LastSyncResult)
	}
}

func TestService_RetryImport(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportSyncStatus{}, &OrderImportBatch{}, &OrderImportItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := OrderImportBatch{
		AdapterCode:      "test_adapter",
		Platform:         "retry_test",
		StoreName:        "test_store",
		SourceFilename:   "retry_orders.csv",
		ImportedBy:       "test_user",
		ChainStatus:      "chain_failed",
		CreatedOrderCount: 0,
		FailedCount:      5,
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatalf("create batch: %v", err)
	}

	// Add some failed items
	for i := 0; i < 3; i++ {
		item := OrderImportItem{
			BatchID:   batch.ID,
			RowNumber: i + 1,
			SkuCode:   "FAIL-SKU",
			Status:    "import_failed",
		}
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("create item: %v", err)
		}
	}

	// Retry
	retried, err := svc.RetryImport(batch.ID)
	if err != nil {
		t.Fatalf("RetryImport: %v", err)
	}
	if retried.ChainStatus != "chain_pending" {
		t.Fatalf("ChainStatus = %s (expected chain_pending)", retried.ChainStatus)
	}
	if retried.FailedCount != 0 {
		t.Fatalf("FailedCount = %d (expected 0)", retried.FailedCount)
	}

	// Verify items were marked for retry
	var pendingItems int64
	db.Model(&OrderImportItem{}).Where("batch_id = ? AND status = ?", batch.ID, "retry_pending").Count(&pendingItems)
	if pendingItems != 3 {
		t.Fatalf("pending retry items = %d (expected 3)", pendingItems)
	}

	// Verify sync status was updated
	statuses, err := svc.GetSyncStatus()
	if err != nil {
		t.Fatalf("GetSyncStatus: %v", err)
	}
	if len(statuses) == 0 {
		t.Fatal("expected sync status after retry")
	}
	if statuses[0].LastSyncResult != "retrying" {
		t.Fatalf("LastSyncResult = %s (expected retrying)", statuses[0].LastSyncResult)
	}
}

func TestService_GetBatch(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OrderImportBatch{})
	svc := NewService(db, dbtest.NewLogger(t))

	batch := OrderImportBatch{
		AdapterCode:    "get_batch_test",
		Platform:       "test_platform",
		StoreName:      "test_store",
		SourceFilename: "batch.csv",
		ImportedBy:     "test_user",
	}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatalf("create batch: %v", err)
	}

	got, err := svc.GetBatch(batch.ID)
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}
	if got.AdapterCode != "get_batch_test" {
		t.Fatalf("AdapterCode = %s", got.AdapterCode)
	}
}

func TestService_UpsertSyncStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ImportSyncStatus{})
	svc := NewService(db, dbtest.NewLogger(t))

	now := time.Now()

	// First upsert creates
	if err := svc.UpsertSyncStatus("ozon", &now, "success", 10, ""); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second upsert updates (cumulative order count)
	if err := svc.UpsertSyncStatus("ozon", &now, "partial", 5, "some errors"); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	statuses, err := svc.GetSyncStatus()
	if err != nil {
		t.Fatalf("GetSyncStatus: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].OrderCount != 15 {
		t.Fatalf("OrderCount = %d (expected 15 = 10 + 5)", statuses[0].OrderCount)
	}
	if statuses[0].LastSyncResult != "partial" {
		t.Fatalf("LastSyncResult = %s", statuses[0].LastSyncResult)
	}
	if statuses[0].ErrorMessage != "some errors" {
		t.Fatalf("ErrorMessage = %s", statuses[0].ErrorMessage)
	}
}
