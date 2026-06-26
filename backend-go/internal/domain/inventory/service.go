package inventory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/allocation"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides inventory business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new inventory service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ── Inventory ─────────────────────────────────────────────────────

// List returns a paginated list of inventory records with optional filters.
func (s *Service) List(ctx context.Context, page, size int, skuID int64, warehouse string) ([]Inventory, int64, error) {
	var items []Inventory
	var total int64

	q := s.db.WithContext(ctx).Model(&Inventory{})
	if skuID > 0 {
		q = q.Where("sku_id = ?", skuID)
	}
	if warehouse != "" {
		q = q.Where("warehouse ILIKE ?", "%"+warehouse+"%")
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

// GetByID retrieves a single inventory record by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*Inventory, error) {
	var inv Inventory
	if err := s.db.WithContext(ctx).First(&inv, id).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

// GetBySkuID retrieves inventory by SKU ID.
func (s *Service) GetBySkuID(ctx context.Context, skuID int64) (*Inventory, error) {
	var inv Inventory
	if err := s.db.WithContext(ctx).Where("sku_id = ?", skuID).First(&inv).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

// UpdateStock updates the quantity of an inventory record and logs the change.
func (s *Service) UpdateStock(ctx context.Context, id int64, newQty int, operator, remark string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv Inventory
		if err := tx.First(&inv, id).Error; err != nil {
			return err
		}

		beforeQty := inv.Quantity
		changeQty := newQty - beforeQty

		changeType := "adjust"
		if changeQty > 0 {
			changeType = "in"
		} else if changeQty < 0 {
			changeType = "out"
		}

		inv.Quantity = newQty
		if err := tx.Save(&inv).Error; err != nil {
			return err
		}

		log := InventoryLog{
			SkuID:      inv.SkuID,
			ChangeType: changeType,
			ChangeQty:  changeQty,
			BeforeQty:  beforeQty,
			AfterQty:   newQty,
			Remark:     remark,
			Operator:   operator,
			CreatedAt:  time.Now(),
		}
		return tx.Create(&log).Error
	})
}

// LockStock increases the locked_quantity of an inventory record.
func (s *Service) LockStock(ctx context.Context, id int64, qty int, operator string) error {
	if qty <= 0 {
		return fmt.Errorf("lock quantity must be positive")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv Inventory
		if err := tx.First(&inv, id).Error; err != nil {
			return err
		}

		available := inv.Quantity - inv.LockedQuantity
		if qty > available {
			return fmt.Errorf("insufficient available stock: requested %d, available %d", qty, available)
		}

		inv.LockedQuantity += qty
		if err := tx.Save(&inv).Error; err != nil {
			return err
		}

		log := InventoryLog{
			SkuID:      inv.SkuID,
			ChangeType: "adjust",
			ChangeQty:  0,
			BeforeQty:  inv.LockedQuantity - qty,
			AfterQty:   inv.LockedQuantity,
			Remark:     fmt.Sprintf("locked %d units", qty),
			Operator:   operator,
			CreatedAt:  time.Now(),
		}
		return tx.Create(&log).Error
	})
}

// UnlockStock decreases the locked_quantity of an inventory record.
func (s *Service) UnlockStock(ctx context.Context, id int64, qty int, operator string) error {
	if qty <= 0 {
		return fmt.Errorf("unlock quantity must be positive")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv Inventory
		if err := tx.First(&inv, id).Error; err != nil {
			return err
		}

		if qty > inv.LockedQuantity {
			return fmt.Errorf("cannot unlock more than locked: requested %d, locked %d", qty, inv.LockedQuantity)
		}

		inv.LockedQuantity -= qty
		if err := tx.Save(&inv).Error; err != nil {
			return err
		}

		log := InventoryLog{
			SkuID:      inv.SkuID,
			ChangeType: "adjust",
			ChangeQty:  0,
			BeforeQty:  inv.LockedQuantity + qty,
			AfterQty:   inv.LockedQuantity,
			Remark:     fmt.Sprintf("unlocked %d units", qty),
			Operator:   operator,
			CreatedAt:  time.Now(),
		}
		return tx.Create(&log).Error
	})
}

// ListLogs returns inventory change logs.
func (s *Service) ListLogs(ctx context.Context, skuID int64, page, size int) ([]InventoryLog, int64, error) {
	var items []InventoryLog
	var total int64

	q := s.db.WithContext(ctx).Model(&InventoryLog{})
	if skuID > 0 {
		q = q.Where("sku_id = ?", skuID)
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

// ── Warehouse ─────────────────────────────────────────────────────

// ListWarehouses returns a paginated list of warehouses.
// Warehouse is defined in the allocation module.
func (s *Service) ListWarehouses(ctx context.Context, page, size int, search string) ([]allocation.Warehouse, int64, error) {
	var items []allocation.Warehouse
	var total int64

	q := s.db.WithContext(ctx).Model(&allocation.Warehouse{})
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
func (s *Service) GetWarehouseByID(ctx context.Context, id int64) (*allocation.Warehouse, error) {
	var w allocation.Warehouse
	if err := s.db.WithContext(ctx).First(&w, id).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

// CreateWarehouse inserts a new warehouse.
func (s *Service) CreateWarehouse(ctx context.Context, w *allocation.Warehouse) error {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(w).Error
}

// UpdateWarehouse saves changes to an existing warehouse.
func (s *Service) UpdateWarehouse(ctx context.Context, w *allocation.Warehouse) error {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(w).Error
}

// DeleteWarehouse removes a warehouse by ID (hard delete).
func (s *Service) DeleteWarehouse(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&allocation.Warehouse{}, id).Error
}

// ── InventoryWarehouse ───────────────────────────────────────────

// ListInventoryBySku returns inventory per warehouse for a SKU.
func (s *Service) ListInventoryBySku(ctx context.Context, skuID int64) ([]InventoryWarehouse, error) {
	var items []InventoryWarehouse
	if err := s.db.WithContext(ctx).Where("sku_id = ?", skuID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
