package inventory

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ── Cross-Platform Sync ────────────────────────────────────────────────────

// SyncAcrossPlatforms prevents overselling by checking inventory across all
// platforms. If total committed stock (sum of active listings across platforms)
// exceeds available stock, it flags the result.
func (s *Service) SyncAcrossPlatforms(ctx context.Context, productID int64) (*CrossPlatformSyncResult, error) {
	// Get total available inventory for all SKUs belonging to this product.
	var availableInventory int
	if err := s.db.WithContext(ctx).
		Raw(`SELECT COALESCE(SUM(i.quantity), 0)
			 FROM inventory i
			 JOIN sku sk ON sk.id = i.sku_id
			 WHERE sk.product_id = ?`, productID).
		Scan(&availableInventory).Error; err != nil {
		return nil, fmt.Errorf("sync: failed to sum available inventory: %w", err)
	}

	// Get all active listings for this product with per-platform committed
	// quantity from published_data->>'stock'. If the field is missing or
	// null we treat it as 1 (assumes each active listing reserves at least one unit).
	type listingRow struct {
		PlatformID int64
		Status     string
		Quantity   int
	}
	var listings []listingRow
	if err := s.db.WithContext(ctx).
		Raw(`SELECT platform_id, status,
			        COALESCE((published_data->>'stock')::int, 1) AS quantity
			 FROM product_listing
			 WHERE product_id = ?
			   AND status IN ('active','live','published')`, productID).
		Scan(&listings).Error; err != nil {
		return nil, fmt.Errorf("sync: failed to query active listings: %w", err)
	}

	totalCommitted := 0
	platformBreakdown := make([]PlatformCommitment, 0, len(listings))
	for _, l := range listings {
		totalCommitted += l.Quantity
		platformBreakdown = append(platformBreakdown, PlatformCommitment{
			PlatformID: l.PlatformID,
			Status:     l.Status,
			Committed:  l.Quantity,
			MaxAllowed: availableInventory,
		})
	}

	oversellDetected := totalCommitted > availableInventory
	oversellBy := 0
	if oversellDetected {
		oversellBy = totalCommitted - availableInventory
	}

	// Log the oversell detection to the oversell log table.
	alertGenerated := false
	if oversellDetected {
		alertGenerated = true
		_ = s.db.WithContext(ctx).Create(&InventoryOversellLog{
			ProductID:      productID,
			AvailableStock: availableInventory,
			TotalCommitted: totalCommitted,
			OversellBy:     oversellBy,
			DetectedAt:     time.Now(),
			Status:         "open",
		}).Error
	}

	return &CrossPlatformSyncResult{
		ProductID:          productID,
		AvailableInventory: availableInventory,
		TotalCommitted:     totalCommitted,
		OversellDetected:   oversellDetected,
		OversellBy:         oversellBy,
		PlatformBreakdown:  platformBreakdown,
		AlertGenerated:     alertGenerated,
	}, nil
}

// ListOversellReport returns all oversell detections with pagination.
func (s *Service) ListOversellReport(ctx context.Context, page, size int) ([]InventoryOversellLog, int64, error) {
	var items []InventoryOversellLog
	var total int64

	q := s.db.WithContext(ctx).Model(&InventoryOversellLog{})
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
	if err := q.Order("detected_at DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

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

// ── InventoryWarehouse ───────────────────────────────────────────

// ListInventoryBySku returns inventory per warehouse for a SKU.
func (s *Service) ListInventoryBySku(ctx context.Context, skuID int64) ([]InventoryWarehouse, error) {
	var items []InventoryWarehouse
	if err := s.db.WithContext(ctx).Where("sku_id = ?", skuID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
