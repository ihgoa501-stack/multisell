package purchase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/supplyevent"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EventPublisher is the minimal interface for publishing events.
// Matches the eventbus.Bus.Publish signature.
type EventPublisher interface {
	Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error)
}

// Service provides purchase business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	events EventPublisher
}

// NewService creates a new purchase service.
func NewService(db *gorm.DB, logger *zap.Logger, events EventPublisher) *Service {
	return &Service{db: db, logger: logger, events: events}
}

// GenerateOrderNo generates a unique purchase order number.
func (s *Service) GenerateOrderNo() (string, error) {
	var count int64
	if err := s.db.Model(&PurchaseOrder{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("PO-%s-%04d", time.Now().Format("20060102"), count+1), nil
}

// CreateOrder creates a purchase order with items in a transaction.
func (s *Service) CreateOrder(in *CreateOrderInput) (*PurchaseOrder, error) {
	orderNo, err := s.GenerateOrderNo()
	if err != nil {
		return nil, err
	}

	o := PurchaseOrder{
		OrderNo:          orderNo,
		SupplierID:       in.SupplierID,
		Status:           StatusDraft,
		ExpectedDelivery: in.ExpectedDelivery,
		Remark:           in.Remark,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&o).Error; err != nil {
			return err
		}
		var total float64
		items := make([]PurchaseOrderItem, 0, len(in.Items))
		for _, it := range in.Items {
			subtotal := it.UnitPrice * float64(it.Quantity)
			total += subtotal
			items = append(items, PurchaseOrderItem{
				PurchaseOrderID: o.ID,
				SkuID:           it.SkuID,
				Quantity:        it.Quantity,
				UnitPrice:       it.UnitPrice,
				Subtotal:        subtotal,
			})
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		return tx.Model(&o).Update("total_amount", total).Error
	})
	if err != nil {
		return nil, err
	}

	// Reload with items
	return s.GetOrder(o.ID)
}

// GetOrder returns a single purchase order with items.
func (s *Service) GetOrder(id int64) (*PurchaseOrder, error) {
	var o PurchaseOrder
	if err := s.db.Preload("Items").First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// ApproveOrder transitions a draft/pending order to approved.
func (s *Service) ApproveOrder(id int64) (*PurchaseOrder, error) {
	o, err := s.GetOrder(id)
	if err != nil {
		return nil, err
	}
	if !canTransition(o.Status, StatusApproved) {
		return nil, fmt.Errorf("cannot approve order in status %s", o.Status)
	}
	if err := s.db.Model(&o).Update("status", StatusApproved).Error; err != nil {
		return nil, err
	}
	o.Status = StatusApproved
	return o, nil
}

// ReceiveOrder handles partial or full receipt of ordered items.
func (s *Service) ReceiveOrder(id int64, in *ReceiveOrderInput) (*PurchaseOrder, error) {
	o, err := s.GetOrder(id)
	if err != nil {
		return nil, err
	}
	if !canTransition(o.Status, StatusPartiallyReceived) && o.Status != StatusPartiallyReceived {
		return nil, fmt.Errorf("cannot receive order in status %s", o.Status)
	}

	// Build lookup for incoming receipt.
	receiptMap := make(map[int64]int)
	for _, r := range in.Items {
		receiptMap[r.ItemID] = r.ReceivedQty
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		allCompleted := true
		for i := range o.Items {
			item := &o.Items[i]
			rqty, exists := receiptMap[item.ID]
			if !exists {
				if item.ReceivedQty < item.Quantity {
					allCompleted = false
				}
				continue
			}
			newReceived := item.ReceivedQty + rqty
			if newReceived > item.Quantity {
				return fmt.Errorf("received qty %d exceeds ordered qty %d for item %d", newReceived, item.Quantity, item.ID)
			}
			if err := tx.Model(&PurchaseOrderItem{}).Where("id = ?", item.ID).Update("received_qty", newReceived).Error; err != nil {
				return err
			}
			item.ReceivedQty = newReceived
			if newReceived < item.Quantity {
				allCompleted = false
			}
		}
		// Also check items not in receiptMap are already fully received.
		for _, item := range o.Items {
			if _, exists := receiptMap[item.ID]; !exists && item.ReceivedQty < item.Quantity {
				allCompleted = false
			}
		}
		newStatus := StatusCompleted
		if !allCompleted {
			newStatus = StatusPartiallyReceived
		}
		return tx.Model(&PurchaseOrder{}).Where("id = ?", id).Update("status", newStatus).Error
	})
	if err != nil {
		return nil, err
	}

	// Publish supply chain event so inventory can auto-update.
	o, _ = s.GetOrder(id)
	s.publishOrderReceived(o)

	return o, nil
}

// CancelOrder cancels a non-completed purchase order.
func (s *Service) CancelOrder(id int64) (*PurchaseOrder, error) {
	o, err := s.GetOrder(id)
	if err != nil {
		return nil, err
	}
	if !canTransition(o.Status, StatusCancelled) {
		return nil, fmt.Errorf("cannot cancel order in status %s", o.Status)
	}
	if err := s.db.Model(&o).Update("status", StatusCancelled).Error; err != nil {
		return nil, err
	}
	o.Status = StatusCancelled
	return o, nil
}

// ListOrders returns paginated purchase orders with optional filter.
func (s *Service) ListOrders(p *common.Pagination, f *PurchaseOrderListFilter) ([]PurchaseOrder, int64, error) {
	q := s.db.Model(&PurchaseOrder{}).Preload("Items")
	if f != nil {
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.SupplierID != nil {
			q = q.Where("supplier_id = ?", *f.SupplierID)
		}
		if f.Search != "" {
			like := "%" + strings.ToLower(f.Search) + "%"
			q = q.Where("LOWER(order_no) LIKE ? OR LOWER(COALESCE(remark,'')) LIKE ?", like, like)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []PurchaseOrder
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListSuggestions returns all purchase suggestions.
func (s *Service) ListSuggestions(p *common.Pagination, status string) ([]PurchaseSuggestion, int64, error) {
	q := s.db.Model(&PurchaseSuggestion{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []PurchaseSuggestion
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GenerateSuggestions automatically generates purchase suggestions based on
// inventory stock levels vs safety stock thresholds.
// It queries the inventory table (from the inventory domain) for each SKU
// and creates a PurchaseSuggestion when quantity <= safety_stock.
func (s *Service) GenerateSuggestions() ([]PurchaseSuggestion, error) {
	type InventoryRow struct {
		SkuID       int64 `gorm:"column:sku_id"`
		Quantity    int   `gorm:"column:quantity"`
		SafetyStock int   `gorm:"column:safety_stock"`
	}
	var rows []InventoryRow
	if err := s.db.Table("inventory").Find(&rows).Error; err != nil {
		return nil, err
	}

	var suggestions []PurchaseSuggestion
	now := time.Now()

	for _, row := range rows {
		available := row.Quantity
		if available >= row.SafetyStock {
			continue
		}
		shortage := row.SafetyStock - available
		suggestedQty := shortage + shortage/2 // order 1.5x the shortage
		if suggestedQty < 1 {
			suggestedQty = 1
		}

		reason := "安全库存"
		if available == 0 {
			reason = "缺货预警"
		}

		ps := PurchaseSuggestion{
			SkuID:        row.SkuID,
			SuggestedQty: suggestedQty,
			Reason:       reason,
			Status:       "pending",
			GeneratedAt:  now,
		}
		suggestions = append(suggestions, ps)
	}

	if len(suggestions) == 0 {
		return suggestions, nil
	}

	if err := s.db.Create(&suggestions).Error; err != nil {
		return nil, err
	}
	return suggestions, nil
}

// publishOrderReceived publishes a supplyevent.OrderReceived event after
// a purchase order is successfully received.
func (s *Service) publishOrderReceived(o *PurchaseOrder) {
	if s.events == nil {
		return
	}
	items := make([]supplyevent.ReceivedItem, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, supplyevent.ReceivedItem{
			SkuID: it.SkuID,
			Qty:   it.ReceivedQty,
		})
	}
	evt := supplyevent.OrderReceived{
		OrderNo:    o.OrderNo,
		SupplierID: o.SupplierID,
		Items:      items,
		ReceivedAt: time.Now(),
	}
	payload, err := supplyevent.ToPayload(evt)
	if err != nil {
		s.logger.Warn("failed to serialize OrderReceived event", zap.Error(err))
		return
	}
	if _, err := s.events.Publish(context.Background(), "supplychain.order.received", "purchase", payload); err != nil {
		s.logger.Warn("failed to publish OrderReceived event", zap.Error(err))
	}
}

// canTransition checks if a status transition is allowed.
func canTransition(current, target string) bool {
	if allowed, ok := validTransitions[current]; ok {
		return allowed[target]
	}
	return false
}
