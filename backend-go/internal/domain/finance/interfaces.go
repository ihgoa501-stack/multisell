package finance

import (
	"context"
	"time"
)

// Order is the finance view of a sales order (only fields finance reads).
type Order struct {
	ID          int64
	PayAmount   float64
	ProductCost float64
	ShippingFee float64
	PlatformFee float64
	PaymentFee  float64
	OtherFee    float64
}

// OrderItem is the finance view of an order item.
type OrderItem struct {
	Subtotal float64
	SkuID    int64
}

// OrderFinanceReader provides read access to orders and their items without importing the order package.
type OrderFinanceReader interface {
	GetByID(ctx context.Context, orderID int64) (*Order, error)
	GetItemsByOrderID(ctx context.Context, orderID int64) ([]OrderItem, error)
	ListByTimeRange(ctx context.Context, since, until time.Time) ([]Order, error)
	// TODO: wire via dependency injection
}
