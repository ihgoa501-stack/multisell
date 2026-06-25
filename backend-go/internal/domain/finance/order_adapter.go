package finance

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// orderFinanceReaderAdapter implements OrderFinanceReader using GORM.
// It queries the sales_order and sales_order_item tables directly,
// without importing the order package.
type orderFinanceReaderAdapter struct {
	db *gorm.DB
}

// NewOrderFinanceReaderAdapter creates a new OrderFinanceReader backed by GORM.
func NewOrderFinanceReaderAdapter(db *gorm.DB) OrderFinanceReader {
	return &orderFinanceReaderAdapter{db: db}
}

// GetByID retrieves a single order by its primary key.
func (a *orderFinanceReaderAdapter) GetByID(ctx context.Context, orderID int64) (*Order, error) {
	var o Order
	err := a.db.WithContext(ctx).Table("sales_order").
		Select("id, pay_amount, product_cost, shipping_fee, platform_fee, payment_fee, other_fee").
		Where("id = ?", orderID).
		Take(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// GetItemsByOrderID retrieves all items belonging to an order.
func (a *orderFinanceReaderAdapter) GetItemsByOrderID(ctx context.Context, orderID int64) ([]OrderItem, error) {
	var items []OrderItem
	err := a.db.WithContext(ctx).Table("sales_order_item").
		Select("subtotal, sku_id").
		Where("order_id = ?", orderID).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ListByTimeRange retrieves all orders with created_at in [since, until].
func (a *orderFinanceReaderAdapter) ListByTimeRange(ctx context.Context, since, until time.Time) ([]Order, error) {
	var orders []Order
	err := a.db.WithContext(ctx).Table("sales_order").
		Where("created_at >= ? AND created_at <= ?", since, until).
		Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}
