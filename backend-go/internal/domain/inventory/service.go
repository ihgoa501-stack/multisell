package inventory

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ── Cross-Platform Sync ────────────────────────────────────────────────────

// EventPublisher publishes events to the event bus.
type EventPublisher interface {
	Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error)
}

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
			        COALESCE(CAST(published_data->>'stock' AS INTEGER), 1) AS quantity
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
	events EventPublisher
}

// NewService creates a new inventory service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// WithEventBus wires an EventPublisher for publishing stock critical events.
// If the bus is nil, stock critical detection is a no-op.
func (s *Service) WithEventBus(ep EventPublisher) *Service {
	s.events = ep
	return s
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
// When the stock drops to or below the safety stock threshold, it publishes a
// supplychain.stock.critical event so the supply chain orchestrator can trigger
// A8 sourcing rescan.
func (s *Service) UpdateStock(ctx context.Context, id int64, newQty int, operator, remark string) error {
	var skuID int64
	var safetyStock int
	var beforeQty int

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv Inventory
		if err := tx.First(&inv, id).Error; err != nil {
			return err
		}

		skuID = inv.SkuID
		safetyStock = inv.SafetyStock
		beforeQty = inv.Quantity
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

	if err != nil {
		return err
	}

	// After a successful stock reduction that crosses the safety stock threshold,
	// publish a supplychain.stock.critical event so the supply chain orchestrator
	// can trigger A8 sourcing rescan and create an approval-gated replenishment action.
	if safetyStock > 0 && beforeQty > safetyStock && newQty <= safetyStock && s.events != nil {
		s.logger.Info("stock dropped below safety threshold, publishing critical event",
			zap.Int64("sku_id", skuID),
			zap.Int("before", beforeQty),
			zap.Int("after", newQty),
			zap.Int("safety_stock", safetyStock),
		)
		if _, pubErr := s.events.Publish(ctx, "supplychain.stock.critical", "inventory", map[string]interface{}{
			"sku_id":        skuID,
			"current_stock": newQty,
			"safety_stock":  safetyStock,
		}); pubErr != nil {
			s.logger.Warn("publish supplychain.stock.critical failed", zap.Error(pubErr))
		}
	}

	return nil
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

// ── Safe Stock Config (#201) ─────────────────────────────────────────

// GetSafetyConfig returns the safety stock config for a SKU.
func (s *Service) GetSafetyConfig(ctx context.Context, skuID int64) (*InventorySafetyConfig, error) {
	var cfg InventorySafetyConfig
	if err := s.db.WithContext(ctx).Where("sku_id = ?", skuID).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpsertSafetyConfig creates or updates a safety stock config.
func (s *Service) UpsertSafetyConfig(ctx context.Context, cfg *InventorySafetyConfig) error {
	var existing InventorySafetyConfig
	result := s.db.WithContext(ctx).Where("sku_id = ?", cfg.SkuID).First(&existing)
	if result.Error != nil {
		return s.db.WithContext(ctx).Create(cfg).Error
	}
	cfg.ID = existing.ID
	cfg.CreatedAt = existing.CreatedAt
	return s.db.WithContext(ctx).Save(cfg).Error
}

// ListSafetyConfigs returns all safety stock configs.
func (s *Service) ListSafetyConfigs(ctx context.Context) ([]InventorySafetyConfig, error) {
	var items []InventorySafetyConfig
	if err := s.db.WithContext(ctx).Order("sku_id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ── Multi-Platform Allocation (#201) ────────────────────────────────

// AllocateStock recommends how to distribute a SKU's stock across platforms.
// It uses each platform's historical sales share to proportionally allocate.
func (s *Service) AllocateStock(ctx context.Context, skuID int64) (*AllocationRecommendation, error) {
	// Get total available inventory.
	inv, err := s.GetBySkuID(ctx, skuID)
	if err != nil {
		return nil, fmt.Errorf("allocate: get inventory: %w", err)
	}

	totalAvailable := inv.Quantity - inv.LockedQuantity
	if totalAvailable < 0 {
		totalAvailable = 0
	}

	// Get live platform listings for this SKU.
	type platRow struct {
		PlatformID int64
		Status     string
	}
	var plats []platRow
	if err := s.db.WithContext(ctx).
		Raw(`SELECT DISTINCT pl.platform_id, pl.status
			 FROM product_listing pl
			 JOIN sku sk ON sk.product_id = pl.product_id
			 WHERE sk.id = ? AND pl.status IN ('active','live','published')`, skuID).
		Scan(&plats).Error; err != nil {
		return nil, fmt.Errorf("allocate: query platforms: %w", err)
	}

	totalCommitted := 0
	recs := make([]PlatformAllocation, 0, len(plats))

	// Equal distribution as default.
	equalShare := 1.0
	if len(plats) > 0 {
		equalShare = 1.0 / float64(len(plats))
	}

	for i, p := range plats {
		// ponytail: equal distribution; upgrade to sales-weighted when
		// multi-platform order data is aggregated.
		recommended := int(math.Floor(float64(totalAvailable) * equalShare))

		var platName string
		s.db.WithContext(ctx).Table("platform").Select("code").
			Where("id = ?", p.PlatformID).Scan(&platName)

		// Count how much is already committed.
		type commitRow struct{ Qty int }
		var cr commitRow
		s.db.WithContext(ctx).
			Raw(`SELECT COALESCE(SUM(COALESCE(CAST(published_data->>'stock' AS INTEGER), 1)), 0)
				 FROM product_listing
				 WHERE platform_id = ? AND product_id IN (
				   SELECT product_id FROM sku WHERE id = ?
				 )`, p.PlatformID, skuID).
			Scan(&cr)
		totalCommitted += cr.Qty

		recs = append(recs, PlatformAllocation{
			PlatformID:   p.PlatformID,
			PlatformName: platName,
			SalesShare:   equalShare * 100,
			CurrentStock: cr.Qty,
			Recommended:  recommended,
			Priority:     i + 1,
		})
	}

	return &AllocationRecommendation{
		SkuID:          skuID,
		TotalAvailable: totalAvailable,
		ReservedTotal:  totalCommitted,
		Recommendations: recs,
		Unallocated:    totalAvailable - totalCommitted,
	}, nil
}

// ── Dead Stock (#201) ─────────────────────────────────────────────

// IdentifyDeadStock scans inventory for SKUs with no recent movement.
// Thresholds: > 90 days no movement = "dead", > 60 days = "slow".
func (s *Service) IdentifyDeadStock(ctx context.Context, thresholdDays int) ([]DeadStockRecord, error) {
	if thresholdDays <= 0 {
		thresholdDays = 90
	}

	type row struct {
		SkuID       int64
		SkuCode     string
		ProductName string
		Warehouse   string
		Quantity    int
		LastMoveStr string
	}
	var rows []row
	err := s.db.WithContext(ctx).
		Raw(`SELECT i.sku_id,
		            COALESCE(sk.code, '') AS sku_code,
		            COALESCE(p.name, '') AS product_name,
		            i.warehouse,
		            i.quantity,
		            (SELECT MAX(created_at) FROM inventory_log il WHERE il.sku_id = i.sku_id) AS last_move
			 FROM inventory i
			 LEFT JOIN sku sk ON sk.id = i.sku_id
			 LEFT JOIN product p ON p.id = sk.product_id
			 WHERE i.quantity > 0
			 ORDER BY i.quantity DESC`).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("dead stock query: %w", err)
	}

	results := make([]DeadStockRecord, 0, len(rows))
	now := time.Now()
	for _, r := range rows {
		daysSince := -1
		if r.LastMoveStr != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", r.LastMoveStr); err == nil {
				daysSince = int(now.Sub(t).Hours() / 24)
			} else if t, err := time.Parse(time.RFC3339Nano, r.LastMoveStr); err == nil {
				daysSince = int(now.Sub(t).Hours() / 24)
			}
		}

		status := "normal"
		suggestion := ""
		if daysSince >= thresholdDays || (r.LastMoveStr == "" && r.Quantity > 0) {
			status = "dead"
			suggestion = "建议降价促销、捆绑销售或捐赠清除库存"
		} else if daysSince >= thresholdDays/2 {
			status = "slow"
			suggestion = "建议检查定价和曝光，考虑小幅降价或广告促销"
		}

		if status != "normal" {
			results = append(results, DeadStockRecord{
				SkuID:         r.SkuID,
				SkuCode:       r.SkuCode,
				ProductName:   r.ProductName,
				Warehouse:     r.Warehouse,
				CurrentQty:    r.Quantity,
				DaysSinceMove: daysSince,
				Status:        status,
				Suggestion:    suggestion,
			})

			if daysSince >= thresholdDays {
				log := DeadStockLog{
					SkuID:         r.SkuID,
					Quantity:      r.Quantity,
					DaysSinceMove: daysSince,
					Status:        status,
					Notes:         suggestion,
				}
				s.db.WithContext(ctx).Create(&log)
			}
		}
	}

	if results == nil {
		results = []DeadStockRecord{}
	}
	return results, nil
}

// ListDeadStockLogs returns dead stock detection history.
func (s *Service) ListDeadStockLogs(ctx context.Context, page, size int) ([]DeadStockLog, int64, error) {
	var items []DeadStockLog
	var total int64
	q := s.db.WithContext(ctx).Model(&DeadStockLog{})
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
