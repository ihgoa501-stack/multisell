package order

import (
	"context"
	"errors"
	"strings"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides order business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new order service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated orders with optional filter.
func (s *Service) List(p *common.Pagination, f *OrderListFilter) ([]Order, int64, error) {
	q := s.db.Model(&Order{})
	if f != nil {
		if f.Search != "" {
			like := "%" + strings.ToLower(f.Search) + "%"
			q = q.Where("LOWER(order_no) LIKE ? OR LOWER(COALESCE(recipient_name,'')) LIKE ? OR LOWER(COALESCE(tracking_number,'')) LIKE ?", like, like, like)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.PlatformID != nil {
			q = q.Where("platform_id = ?", *f.PlatformID)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Order
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns a single order with items and status logs.
func (s *Service) Get(id int64) (*OrderDetail, error) {
	var o Order
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	var items []OrderItem
	if err := s.db.Where("order_id = ?", id).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	var logs []OrderStatusLog
	if err := s.db.Where("order_id = ?", id).Order("id ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return &OrderDetail{Order: o, Items: items, StatusLogs: logs}, nil
}

// Create inserts an order with its items in a transaction.
func (s *Service) Create(in *CreateOrderInput) (*Order, error) {
	if strings.TrimSpace(in.OrderNo) == "" {
		return nil, errors.New("order_no is required")
	}
	status := in.Status
	if status == "" {
		status = "pending"
	}
	o := Order{
		OrderNo:        in.OrderNo,
		PlatformID:     in.PlatformID,
		Status:         status,
		TrackingNumber: in.TrackingNumber,
		RecipientName:  in.RecipientName,
		RecipientPhone: in.RecipientPhone,
		ShippingAddress: in.ShippingAddress,
		PaymentMethod:  in.PaymentMethod,
		Remark:         in.Remark,
	}
	if in.TotalAmount != nil {
		o.TotalAmount = *in.TotalAmount
	}
	if in.ShippingFee != nil {
		o.ShippingFee = *in.ShippingFee
	}
	if in.PayAmount != nil {
		o.PayAmount = *in.PayAmount
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&o).Error; err != nil {
			return err
		}
		if len(in.Items) == 0 {
			return nil
		}
		items := make([]OrderItem, 0, len(in.Items))
		for _, it := range in.Items {
			items = append(items, OrderItem{
				OrderID:     o.ID,
				SkuID:       it.SkuID,
				ProductID:   it.ProductID,
				ProductName: it.ProductName,
				SkuCode:     it.SkuCode,
				SpecDesc:    it.SpecDesc,
				UnitPrice:   it.UnitPrice,
				Quantity:    it.Quantity,
				Subtotal:    it.UnitPrice * float64(it.Quantity),
			})
		}
		return tx.Create(&items).Error
	})
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// Update applies partial updates to an order.
func (s *Service) Update(id int64, in *UpdateOrderInput) (*Order, error) {
	var o Order
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Status != nil {
		sm := NewOrderStateMachine()
		if err := sm.MustTransition(context.Background(), o.Status, *in.Status, nil); err != nil {
			return nil, err
		}
		updates["status"] = *in.Status
	}
	if in.TrackingNumber != nil {
		updates["tracking_number"] = *in.TrackingNumber
	}
	if in.RecipientName != nil {
		updates["recipient_name"] = *in.RecipientName
	}
	if in.RecipientPhone != nil {
		updates["recipient_phone"] = *in.RecipientPhone
	}
	if in.ShippingAddress != nil {
		updates["shipping_address"] = *in.ShippingAddress
	}
	if in.PaymentMethod != nil {
		updates["payment_method"] = *in.PaymentMethod
	}
	if in.Remark != nil {
		updates["remark"] = *in.Remark
	}
	if len(updates) == 0 {
		return &o, nil
	}
	if err := s.db.Model(&o).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Delete soft-removes an order (hard delete for now, cascade items).
func (s *Service) Delete(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_id = ?", id).Delete(&OrderItem{}).Error; err != nil {
			return err
		}
		if err := tx.Where("order_id = ?", id).Delete(&OrderStatusLog{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Order{}, id).Error
	})
}

// UpdateStatus transitions an order's status and logs the change.
func (s *Service) UpdateStatus(id int64, from, to, operator, remark string) error {
	sm := NewOrderStateMachine()
	if err := sm.MustTransition(context.Background(), from, to, nil); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var o Order
		if err := tx.First(&o, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&o).Update("status", to).Error; err != nil {
			return err
		}
		return tx.Create(&OrderStatusLog{
			OrderID:    id,
			FromStatus: from,
			ToStatus:   to,
			Operator:   operator,
			Remark:     remark,
		}).Error
	})
}

// Summary returns dashboard aggregation.
func (s *Service) Summary() (*OrderSummary, error) {
	var total int64
	if err := s.db.Model(&Order{}).Count(&total).Error; err != nil {
		return nil, err
	}
	type statusCount struct {
		Status string
		Cnt    int64
	}
	var scs []statusCount
	if err := s.db.Model(&Order{}).Select("status, COUNT(*) AS cnt").Group("status").Scan(&scs).Error; err != nil {
		return nil, err
	}
	byStatus := make(map[string]int64, len(scs))
	for _, sc := range scs {
		byStatus[sc.Status] = sc.Cnt
	}
	var rev struct {
		Revenue float64
		Profit  float64
	}
	if err := s.db.Model(&Order{}).
		Select("COALESCE(SUM(pay_amount),0) AS revenue, COALESCE(SUM(profit_amount),0) AS profit").
		Scan(&rev).Error; err != nil {
		return nil, err
	}
	return &OrderSummary{Total: total, ByStatus: byStatus, TotalRevenue: rev.Revenue, TotalProfit: rev.Profit}, nil
}
