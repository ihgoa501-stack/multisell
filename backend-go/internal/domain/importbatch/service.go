package importbatch

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides importbatch business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new importbatch service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ListBatches returns import batches, optionally filtered by source_type/status.
func (s *Service) ListBatches(sourceType, status string, page, size int) ([]ImportBatch, int64, error) {
	q := s.db.Model(&ImportBatch{})
	if sourceType != "" {
		q = q.Where("source_type = ?", sourceType)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var items []ImportBatch
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetBatch returns a single import batch.
func (s *Service) GetBatch(id int64) (*ImportBatch, error) {
	var b ImportBatch
	if err := s.db.First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateBatch inserts a new import batch.
func (s *Service) CreateBatch(b *ImportBatch) error {
	if b.Status == "" {
		b.Status = "pending"
	}
	return s.db.Create(b).Error
}

// UpdateBatch updates an import batch (e.g. commit/preview status).
func (s *Service) UpdateBatch(b *ImportBatch) error {
	return s.db.Save(b).Error
}

// DeleteBatch removes an import batch and its rows.
func (s *Service) DeleteBatch(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("batch_id = ?", id).Delete(&ImportBatchRow{}).Error; err != nil {
			return err
		}
		return tx.Delete(&ImportBatch{}, id).Error
	})
}

// ===================== ImportBatchRow =====================

// ListRows returns the rows for a batch, optionally filtered by status.
func (s *Service) ListRows(batchID int64, status string, page, size int) ([]ImportBatchRow, int64, error) {
	q := s.db.Model(&ImportBatchRow{}).Where("batch_id = ?", batchID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 50
	}
	var items []ImportBatchRow
	if err := q.Order("row_index asc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CreateRow inserts a single import batch row.
func (s *Service) CreateRow(r *ImportBatchRow) error {
	return s.db.Create(r).Error
}
