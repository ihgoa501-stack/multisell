package aftersales

import (
	"context"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"gorm.io/gorm"
)

// inventoryRestockAdapter implements InventoryRestocker using direct GORM access.
type inventoryRestockAdapter struct {
	db *gorm.DB
}

// NewInventoryRestockAdapter creates a new InventoryRestocker backed by *gorm.DB.
func NewInventoryRestockAdapter(db *gorm.DB) InventoryRestocker {
	return &inventoryRestockAdapter{db: db}
}

// Restock runs its own transaction: reads inventory by skuID, updates Quantity
// and LockedQuantity, saves, and creates an InventoryLog entry.
func (a *inventoryRestockAdapter) Restock(ctx context.Context, skuID int64, quantity int, operator string, remark string) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv inventory.Inventory
		if err := tx.Where("sku_id = ?", skuID).First(&inv).Error; err != nil {
			return fmt.Errorf("inventory not found for sku_id %d: %w", skuID, err)
		}
		beforeQty := inv.Quantity
		inv.Quantity += quantity
		inv.LockedQuantity -= quantity
		if inv.LockedQuantity < 0 {
			inv.LockedQuantity = 0
		}
		if err := tx.Save(&inv).Error; err != nil {
			return err
		}
		return tx.Create(&inventory.InventoryLog{
			SkuID:      skuID,
			ChangeType: "in",
			ChangeQty:  quantity,
			BeforeQty:  beforeQty,
			AfterQty:   inv.Quantity,
			Remark:     remark,
			Operator:   operator,
			CreatedAt:  time.Now(),
		}).Error
	})
}
