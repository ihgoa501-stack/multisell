package aftersales

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/order"
	"gorm.io/gorm"
)

// orderWriterAdapter implements OrderWriter using direct GORM access.
type orderWriterAdapter struct {
	db *gorm.DB
}

// NewOrderWriterAdapter creates a new OrderWriter backed by *gorm.DB.
func NewOrderWriterAdapter(db *gorm.DB) OrderWriter {
	return &orderWriterAdapter{db: db}
}

// CancelOrder runs its own transaction: updates the order status to "cancelled"
// and creates an OrderStatusLog entry.
func (a *orderWriterAdapter) CancelOrder(ctx context.Context, orderID int64, operator string, remark string) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&order.Order{}).Where("id = ?", orderID).Update("status", "cancelled").Error; err != nil {
			return err
		}
		return tx.Create(&order.OrderStatusLog{
			OrderID:    orderID,
			FromStatus: "",
			ToStatus:   "cancelled",
			Operator:   operator,
			Remark:     remark,
			CreatedAt:  time.Now(),
		}).Error
	})
}
