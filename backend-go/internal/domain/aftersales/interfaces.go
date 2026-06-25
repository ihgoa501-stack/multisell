package aftersales

import "context"

// InventoryItem is the aftersales view of inventory (only fields needed by this module).
type InventoryItem struct {
	SkuID          int64
	Quantity       int
	LockedQuantity int
}

// InventoryRestocker provides inventory restock operations for aftersales.
type InventoryRestocker interface {
	Restock(ctx context.Context, skuID int64, quantity int, operator string, remark string) error
}

// OrderWriter provides order mutation operations for aftersales.
type OrderWriter interface {
	CancelOrder(ctx context.Context, orderID int64, operator string, remark string) error
}
