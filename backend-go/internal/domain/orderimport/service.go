package orderimport

import (
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides orderimport business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new orderimport service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated order imports with optional filter.
func (s *Service) List(p *common.Pagination, f *ListFilter) ([]OrderImport, int64, error) {
	q := s.db.Model(&OrderImport{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("file_name ILIKE ?", like)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.PlatformID != nil {
			q = q.Where("platform_id = ?", *f.PlatformID)
		}
		if f.SourceType != "" {
			q = q.Where("source_type = ?", f.SourceType)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []OrderImport
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns a single order import.
func (s *Service) Get(id int64) (*OrderImport, error) {
	var o OrderImport
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Create inserts a new order import.
func (s *Service) Create(in *CreateInput) (*OrderImport, error) {
	status := in.Status
	if status == "" {
		status = "pending"
	}
	sourceType := in.SourceType
	if sourceType == "" {
		sourceType = "manual"
	}
	o := OrderImport{
		PlatformID: in.PlatformID,
		SourceType: sourceType,
		FileName:   in.FileName,
		Status:     status,
		CreatedBy:  in.CreatedBy,
	}
	if in.TotalRows != nil {
		o.TotalRows = *in.TotalRows
	}
	if err := s.db.Create(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Update applies partial updates to an order import.
func (s *Service) Update(id int64, in *UpdateInput) (*OrderImport, error) {
	var o OrderImport
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.PlatformID != nil {
		updates["platform_id"] = *in.PlatformID
	}
	if in.SourceType != nil {
		updates["source_type"] = *in.SourceType
	}
	if in.FileName != nil {
		updates["file_name"] = *in.FileName
	}
	if in.TotalRows != nil {
		updates["total_rows"] = *in.TotalRows
	}
	if in.SuccessCount != nil {
		updates["success_count"] = *in.SuccessCount
	}
	if in.ErrorCount != nil {
		updates["error_count"] = *in.ErrorCount
	}
	if in.ErrorDetail != nil {
		updates["error_detail"] = *in.ErrorDetail
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if len(updates) == 0 {
		return &o, nil
	}
	if err := s.db.Model(&o).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Delete removes an order import by id.
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&OrderImport{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Process marks an order import as processing.
func (s *Service) Process(id int64) (*OrderImport, error) {
	var o OrderImport
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&o).Update("status", "processing").Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Complete finalizes an order import with success/error counts and status.
func (s *Service) Complete(id int64, in *CompleteInput) (*OrderImport, error) {
	var o OrderImport
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = "completed"
	}
	updates := map[string]interface{}{
		"success_count": in.SuccessCount,
		"error_count":   in.ErrorCount,
		"status":        status,
	}
	if len(in.ErrorDetail) > 0 {
		updates["error_detail"] = in.ErrorDetail
	}
	if err := s.db.Model(&o).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Summary returns aggregation by status and total/success/error row counts.
func (s *Service) Summary() (*Summary, error) {
	var total int64
	if err := s.db.Model(&OrderImport{}).Count(&total).Error; err != nil {
		return nil, err
	}
	type statusCount struct {
		Status string
		Cnt    int64
	}
	var scs []statusCount
	if err := s.db.Model(&OrderImport{}).
		Select("status, COUNT(*) AS cnt").Group("status").Scan(&scs).Error; err != nil {
		return nil, err
	}
	byStatus := make(map[string]int64, len(scs))
	for _, sc := range scs {
		byStatus[sc.Status] = sc.Cnt
	}
	var agg struct {
		TotalRows    int64
		SuccessCount int64
		ErrorCount   int64
	}
	if err := s.db.Model(&OrderImport{}).
		Select("COALESCE(SUM(total_rows),0) AS total_rows, COALESCE(SUM(success_count),0) AS success_count, COALESCE(SUM(error_count),0) AS error_count").
		Scan(&agg).Error; err != nil {
		return nil, err
	}
	return &Summary{
		Total:        total,
		ByStatus:     byStatus,
		TotalRows:    agg.TotalRows,
		SuccessCount: agg.SuccessCount,
		ErrorCount:   agg.ErrorCount,
	}, nil
}

// SyncOrders idempotently imports orders: skips rows whose platform_order_no
// already exists in order_import_item for the same platform.
func (s *Service) SyncOrders(batchID int64, orders []SyncOrderInput) (*SyncResult, error) {
	var createdCount, skippedCount, failedCount int
	var firstErr string
	platform := ""
	if len(orders) > 0 {
		platform = orders[0].Platform
	}

	for _, o := range orders {
		// Check for existing platform_order_no
		var existing int64
		s.db.Model(&OrderImportItem{}).
			Where("platform_order_no = ?", o.PlatformOrderNo).
			Count(&existing)
		if existing > 0 {
			skippedCount++
			s.db.Model(&OrderImportBatch{}).Where("id = ?", batchID).
				UpdateColumn("skipped_duplicate_count", gorm.Expr("skipped_duplicate_count + 1"))
			continue
		}

		item := OrderImportItem{
			BatchID:         batchID,
			RowNumber:       skippedCount + createdCount + failedCount + 1,
			Platform:        o.Platform,
			StoreName:       o.StoreName,
			PlatformOrderNo: o.PlatformOrderNo,
			SkuCode:         o.SkuCode,
			Quantity:        o.Quantity,
			UnitPrice:       o.UnitPrice,
			Currency:        o.Currency,
			RecipientName:   o.RecipientName,
			RecipientPhone:  o.RecipientPhone,
			CountryCode:     o.CountryCode,
			ShippingAddress: o.ShippingAddress,
			ShippingFee:     o.ShippingFee,
			TrackingNumber:  o.TrackingNumber,
			PaidAt:          o.PaidAt,
			Status:          "imported",
		}
		if err := s.db.Create(&item).Error; err != nil {
			failedCount++
			if firstErr == "" {
				firstErr = err.Error()
			}
			// ponytail: per-batch failure counter, no per-item rollback
			s.db.Model(&OrderImportBatch{}).Where("id = ?", batchID).
				UpdateColumn("failed_count", gorm.Expr("failed_count + 1"))
			continue
		}
		createdCount++
		s.db.Model(&OrderImportBatch{}).Where("id = ?", batchID).
			UpdateColumn("created_order_count", gorm.Expr("created_order_count + 1"))
	}

	result := "success"
	if failedCount > 0 && createdCount > 0 {
		result = "partial"
	} else if failedCount > 0 {
		result = "failed"
	}
	now := time.Now()
	_ = s.UpsertSyncStatus(platform, &now, result, createdCount, firstErr)
	// ponytail: only tracks one platform; for multi-platform batches, iterate per platform

	return &SyncResult{
		Platform:       platform,
		CreatedCount:   createdCount,
		SkippedCount:   skippedCount,
		FailedCount:    failedCount,
		LastSyncResult: result,
		ErrorMessage:   firstErr,
	}, nil
}

// GetSyncStatus returns import sync status for all platforms.
func (s *Service) GetSyncStatus() ([]ImportSyncStatus, error) {
	var statuses []ImportSyncStatus
	if err := s.db.Order("platform ASC").Find(&statuses).Error; err != nil {
		return nil, err
	}
	return statuses, nil
}

// UpsertSyncStatus creates or updates the sync status for a platform.
func (s *Service) UpsertSyncStatus(platform string, lastSyncAt *time.Time, result string, orderCount int, errMsg string) error {
	var existing ImportSyncStatus
	err := s.db.Where("platform = ?", platform).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(&ImportSyncStatus{
			Platform:       platform,
			LastSyncAt:     lastSyncAt,
			LastSyncResult: result,
			OrderCount:     orderCount,
			ErrorMessage:   errMsg,
		}).Error
	}
	if err != nil {
		return err
	}
	existing.LastSyncAt = lastSyncAt
	existing.LastSyncResult = result
	existing.OrderCount += orderCount
	existing.ErrorMessage = errMsg
	return s.db.Save(&existing).Error
}

// RetryImport resets a failed batch for retry and returns the updated batch.
func (s *Service) RetryImport(batchID int64) (*OrderImportBatch, error) {
	var batch OrderImportBatch
	if err := s.db.First(&batch, batchID).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{
		"chain_status":          "chain_pending",
		"failed_count":          0,
		"created_order_count":   0,
		"skipped_duplicate_count": 0,
	}

	// Also reset associated items: mark both imported (chain-pending) and failed items for retry
	s.db.Model(&OrderImportItem{}).Where("batch_id = ? AND status IN ?", batchID, []string{"imported", "import_failed"}).
		UpdateColumn("status", "retry_pending")

	// Reset sync status for the batch's platform
	platform := batch.Platform
	if platform != "" {
		_ = s.UpsertSyncStatus(platform, nil, "retrying", 0, "")
	}

	if err := s.db.Model(&batch).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&batch, batchID).Error; err != nil {
		return nil, err
	}
	return &batch, nil
}

// GetBatch returns a single batch.
func (s *Service) GetBatch(id int64) (*OrderImportBatch, error) {
	var b OrderImportBatch
	if err := s.db.First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}
