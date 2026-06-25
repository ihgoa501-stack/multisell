package aftersales

import "context"

// Order is the aftersales view of a sales order (only fields needed by this module).
type Order struct {
	ID     int64
	Status string
}

// OrderReader provides read access to sales orders without importing the order package.
type OrderReader interface {
	GetByID(ctx context.Context, orderID int64) (*Order, error)
	// TODO: wire via dependency injection
}

// InventoryChecker provides inventory checks and reservations for aftersales operations.
type InventoryChecker interface {
	GetStockLevel(ctx context.Context, skuID int64) (int, error)
	ReserveStock(ctx context.Context, skuID int64, quantity int) error
	// TODO: wire via dependency injection
}
