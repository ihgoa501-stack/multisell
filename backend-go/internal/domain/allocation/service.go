package allocation

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
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

// ── CostAllocation Computation ─────────────────────────────────────

// ComputeAllocation computes cost allocation for all items in a batch
// based on the batch's allocation_method ("weight", "volume", "value", "equal").
// It saves allocation_factor and allocated_amount back to each item, then
// updates the batch status to "computed".
func (s *Service) ComputeAllocation(ctx context.Context, batchID int64) error {
	// Fetch batch
	var batch CostAllocationBatch
	if err := s.db.WithContext(ctx).First(&batch, batchID).Error; err != nil {
		return fmt.Errorf("batch not found: %w", err)
	}

	// Fetch all items for this batch
	var items []CostAllocationItem
	if err := s.db.WithContext(ctx).
		Where("batch_id = ?", batchID).
		Order("row_number ASC, id ASC").
		Find(&items).Error; err != nil {
		return fmt.Errorf("failed to fetch items: %w", err)
	}
	if len(items) == 0 {
		return fmt.Errorf("no items found for batch %d", batchID)
	}

	totalAmount := batch.TotalAmount

	// Compute basis total from items
	var basisTotal decimal.Decimal
	switch batch.AllocationMethod {
	case "weight":
		for i := range items {
			if items[i].WeightKg != nil {
				basisTotal = basisTotal.Add(*items[i].WeightKg)
			}
		}
	case "volume":
		for i := range items {
			if items[i].VolumeM3 != nil {
				basisTotal = basisTotal.Add(*items[i].VolumeM3)
			}
		}
	case "value":
		for i := range items {
			if items[i].ItemValue != nil {
				basisTotal = basisTotal.Add(*items[i].ItemValue)
			}
		}
	case "equal":
		basisTotal = decimal.NewFromInt(int64(len(items)))
	default:
		return fmt.Errorf("unsupported allocation method: %s", batch.AllocationMethod)
	}

	if basisTotal.IsZero() {
		return fmt.Errorf("allocation basis total is zero for method '%s'", batch.AllocationMethod)
	}

	for i := range items {
		var factor decimal.Decimal
		switch batch.AllocationMethod {
		case "weight":
			if items[i].WeightKg != nil {
				factor = items[i].WeightKg.Div(basisTotal)
			}
		case "volume":
			if items[i].VolumeM3 != nil {
				factor = items[i].VolumeM3.Div(basisTotal)
			}
		case "value":
			if items[i].ItemValue != nil {
				factor = items[i].ItemValue.Div(basisTotal)
			}
		case "equal":
			factor = decimal.NewFromFloat(1).Div(basisTotal)
		}

		allocatedAmount := totalAmount.Mul(factor)

		items[i].AllocationFactor = &factor
		items[i].AllocatedAmount = allocatedAmount

		if err := s.db.WithContext(ctx).Save(&items[i]).Error; err != nil {
			return fmt.Errorf("failed to save item %d: %w", items[i].ID, err)
		}
	}

	// Update batch status to "computed"
	if err := s.db.WithContext(ctx).
		Model(&CostAllocationBatch{}).
		Where("id = ?", batchID).
		Update("status", "computed").Error; err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	return nil
}

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

// allocationTarget represents how much to allocate to a single warehouse.
type allocationTarget struct {
	WarehouseID int64
	Quantity    int
}

// inventoryWarehouseRow mirrors inventory.InventoryWarehouse to avoid an import cycle.
type inventoryWarehouseRow struct {
	ID              int64
	SkuID           int64
	WarehouseID     int64
	Quantity        int
	LockedQuantity  int
}

func (inventoryWarehouseRow) TableName() string { return "inventory_warehouse" }

// AutoAllocate distributes available inventory for a SKU across warehouses
// according to active AllocationRule records sorted by priority.
//
// Percentage rules allocate total_available * pct / 100 to the target warehouse.
// Fixed-quantity rules allocate allocation_qty units to the target warehouse.
// Rules are applied in priority order; once the available inventory is exhausted,
// remaining rules are skipped.
func (s *Service) AutoAllocate(ctx context.Context, skuID int64) error {
	// 1. Fetch active rules for this SKU, sorted by priority ASC.
	var rules []AllocationRule
	if err := s.db.WithContext(ctx).
		Where("sku_id = ? AND status = 1", skuID).
		Order("priority ASC, id ASC").
		Find(&rules).Error; err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	// 2. Compute total available quantity across all warehouses for this SKU.
	//    Available = quantity - locked_quantity (items already committed to orders).
	type TotalRow struct {
		Total int64
	}
	var row TotalRow
	if err := s.db.WithContext(ctx).Table("inventory_warehouse").
		Select("COALESCE(SUM(quantity - locked_quantity), 0) AS total").
		Where("sku_id = ?", skuID).
		Scan(&row).Error; err != nil {
		return err
	}
	totalAvailable := int(row.Total)

	// 3. Work through rules in priority order, building allocation targets.
	var targets []allocationTarget
	remaining := totalAvailable

	for _, rule := range rules {
		if remaining <= 0 {
			break
		}

		var allocQty int
		switch rule.RuleType {
		case "percentage":
			// pct of the ORIGINAL total (consistent allocation regardless of order)
			if rule.AllocationPct.IsZero() {
				continue
			}
			pct := rule.AllocationPct.Div(decimal.NewFromInt(100))
			allocQty = int(pct.Mul(decimal.NewFromInt(int64(totalAvailable))).IntPart())
		case "fixed":
			// fixed units, capped at remaining
			if rule.AllocationQty <= 0 {
				continue
			}
			allocQty = rule.AllocationQty
		default:
			s.logger.Warn("unknown allocation rule type, skipping",
				zap.Int64("rule_id", rule.ID),
				zap.String("rule_type", rule.RuleType),
			)
			continue
		}

		if allocQty <= 0 {
			continue
		}
		if allocQty > remaining {
			allocQty = remaining
		}

		targets = append(targets, allocationTarget{
			WarehouseID: rule.WarehouseID,
			Quantity:    allocQty,
		})
		remaining -= allocQty
	}

	// 4. Persist the allocations inside a transaction.
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Fetch existing InventoryWarehouse records for this SKU.
		var existing []inventoryWarehouseRow
		if err := tx.Where("sku_id = ?", skuID).Find(&existing).Error; err != nil {
			return err
		}

		existingByWarehouse := make(map[int64]*inventoryWarehouseRow)
		for i := range existing {
			existingByWarehouse[existing[i].WarehouseID] = &existing[i]
		}

		// Apply each allocation target.
		for _, t := range targets {
			if rec, ok := existingByWarehouse[t.WarehouseID]; ok {
				rec.Quantity = t.Quantity
				if err := tx.Save(rec).Error; err != nil {
					return err
				}
			} else {
				rec := &inventoryWarehouseRow{
					SkuID:       skuID,
					WarehouseID: t.WarehouseID,
					Quantity:    t.Quantity,
				}
				if err := tx.Create(rec).Error; err != nil {
					return err
				}
			}
		}

		// Zero out warehouses that were previously holding stock but are not
		// targeted by any rule.
		for wid, rec := range existingByWarehouse {
			hasTarget := false
			for _, t := range targets {
				if t.WarehouseID == wid {
					hasTarget = true
					break
				}
			}
			if !hasTarget {
				rec.Quantity = 0
				if err := tx.Save(rec).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}
