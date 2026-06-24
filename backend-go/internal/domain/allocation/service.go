package allocation

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides allocation business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new allocation service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ── Warehouse ─────────────────────────────────────────────────────

// ListWarehouses returns a paginated list of warehouses.
func (s *Service) ListWarehouses(ctx context.Context, page, size int, search string) ([]Warehouse, int64, error) {
	var items []Warehouse
	var total int64

	q := s.db.WithContext(ctx).Model(&Warehouse{})
	if search != "" {
		q = q.Where("name ILIKE ? OR code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetWarehouseByID retrieves a single warehouse by ID.
func (s *Service) GetWarehouseByID(ctx context.Context, id int64) (*Warehouse, error) {
	var w Warehouse
	if err := s.db.WithContext(ctx).First(&w, id).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

// CreateWarehouse inserts a new warehouse.
func (s *Service) CreateWarehouse(ctx context.Context, w *Warehouse) error {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(w).Error
}

// UpdateWarehouse saves changes to an existing warehouse.
func (s *Service) UpdateWarehouse(ctx context.Context, w *Warehouse) error {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(w).Error
}

// DeleteWarehouse removes a warehouse by ID (hard delete).
func (s *Service) DeleteWarehouse(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&Warehouse{}, id).Error
}

// ── AllocationRule ───────────────────────────────────────────────

// ListRules returns a paginated list of allocation rules.
func (s *Service) ListRules(ctx context.Context, page, size int, warehouseID int64) ([]AllocationRule, int64, error) {
	var items []AllocationRule
	var total int64

	q := s.db.WithContext(ctx).Model(&AllocationRule{})
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size
	if err := q.Order("priority ASC, id DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetRuleByID retrieves a single allocation rule by ID.
func (s *Service) GetRuleByID(ctx context.Context, id int64) (*AllocationRule, error) {
	var r AllocationRule
	if err := s.db.WithContext(ctx).First(&r, id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRule inserts a new allocation rule.
func (s *Service) CreateRule(ctx context.Context, r *AllocationRule) error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return gorm.ErrInvalidData
	}
	r.RuleType = strings.TrimSpace(r.RuleType)
	if r.RuleType == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(r).Error
}

// UpdateRule saves changes to an existing allocation rule.
func (s *Service) UpdateRule(ctx context.Context, r *AllocationRule) error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return gorm.ErrInvalidData
	}
	r.RuleType = strings.TrimSpace(r.RuleType)
	if r.RuleType == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(r).Error
}

// DeleteRule removes an allocation rule by ID (hard delete).
func (s *Service) DeleteRule(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&AllocationRule{}, id).Error
}

// ── CostAllocationBatch ──────────────────────────────────────────

// ListBatches returns a paginated list of cost allocation batches.
func (s *Service) ListBatches(ctx context.Context, page, size int, allocationType string) ([]CostAllocationBatch, int64, error) {
	var items []CostAllocationBatch
	var total int64

	q := s.db.WithContext(ctx).Model(&CostAllocationBatch{})
	if allocationType != "" {
		q = q.Where("allocation_type = ?", allocationType)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetBatchByID retrieves a single cost allocation batch by ID, including its items.
func (s *Service) GetBatchByID(ctx context.Context, id int64) (*CostAllocationBatch, error) {
	var b CostAllocationBatch
	if err := s.db.WithContext(ctx).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateBatch inserts a new cost allocation batch.
func (s *Service) CreateBatch(ctx context.Context, b *CostAllocationBatch) error {
	b.AllocationType = strings.TrimSpace(b.AllocationType)
	b.AllocationMethod = strings.TrimSpace(b.AllocationMethod)
	if b.AllocationType == "" || b.AllocationMethod == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(b).Error
}

// ListBatchItems returns the items for a cost allocation batch.
func (s *Service) ListBatchItems(ctx context.Context, batchID int64, page, size int) ([]CostAllocationItem, int64, error) {
	var items []CostAllocationItem
	var total int64

	q := s.db.WithContext(ctx).Model(&CostAllocationItem{}).Where("batch_id = ?", batchID)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size
	if err := q.Order("row_number ASC, id ASC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
