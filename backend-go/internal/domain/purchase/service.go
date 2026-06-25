package purchase

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides purchase business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new purchase service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
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
	// Verify supplier exists
	var sup Supplier
	if err := s.db.First(&sup, in.SupplierID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("supplier not found")
		}
		return nil, err
	}

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
	return s.GetOrder(id)
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

// ---------- Supplier operations ----------

// CreateSupplier creates a new supplier.
func (s *Service) CreateSupplier(in *CreateSupplierInput) (*Supplier, error) {
	sup := Supplier{
		Name:          in.Name,
		ContactPerson: in.ContactPerson,
		Phone:         in.Phone,
		Email:         in.Email,
		Address:       in.Address,
		PriceHistory:  in.PriceHistory,
	}
	if err := s.db.Create(&sup).Error; err != nil {
		return nil, err
	}
	return &sup, nil
}

// UpdateSupplier updates a supplier.
func (s *Service) UpdateSupplier(id int64, in *UpdateSupplierInput) (*Supplier, error) {
	var sup Supplier
	if err := s.db.First(&sup, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.ContactPerson != nil {
		updates["contact_person"] = *in.ContactPerson
	}
	if in.Phone != nil {
		updates["phone"] = *in.Phone
	}
	if in.Email != nil {
		updates["email"] = *in.Email
	}
	if in.Address != nil {
		updates["address"] = *in.Address
	}
	if in.PriceHistory != nil {
		updates["price_history"] = in.PriceHistory
	}
	if len(updates) > 0 {
		if err := s.db.Model(&sup).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return &sup, nil
}

// DeleteSupplier deletes a supplier.
func (s *Service) DeleteSupplier(id int64) error {
	return s.db.Delete(&Supplier{}, id).Error
}

// ListSuppliers returns paginated suppliers.
func (s *Service) ListSuppliers(p *common.Pagination, search string) ([]Supplier, int64, error) {
	q := s.db.Model(&Supplier{})
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		q = q.Where("LOWER(name) LIKE ? OR LOWER(COALESCE(contact_person,'')) LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Supplier
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetSupplier returns a single supplier.
func (s *Service) GetSupplier(id int64) (*Supplier, error) {
	var sup Supplier
	if err := s.db.First(&sup, id).Error; err != nil {
		return nil, err
	}
	return &sup, nil
}

// GetSupplierKPI calculates and returns a supplier's KPI score.
func (s *Service) GetSupplierKPI(id int64) (*SupplierKPIResponse, error) {
	var sup Supplier
	if err := s.db.First(&sup, id).Error; err != nil {
		return nil, err
	}

	// Count orders for this supplier.
	var orderCount int64
	s.db.Model(&PurchaseOrder{}).Where("supplier_id = ?", id).Count(&orderCount)

	// Calculate on-time delivery rate based on completed orders with
	// expected_delivery set. If expected_delivery is before or on updated_at,
	// consider it on-time. This is simplified; real logic would compare
	// actual delivery date.
	var onTimeCount int64
	s.db.Model(&PurchaseOrder{}).
		Where("supplier_id = ? AND status = ? AND expected_delivery IS NOT NULL", id, StatusCompleted).
		Count(&onTimeCount)

	var onTimeRate float64
	if orderCount > 0 {
		onTimeRate = float64(onTimeCount) / float64(orderCount) * 100
	}

	return &SupplierKPIResponse{
		SupplierID:   sup.ID,
		SupplierName: sup.Name,
		KpiScore:     sup.KpiScore,
		OrderCount:   orderCount,
		OnTimeRate:   onTimeRate,
	}, nil
}

// canTransition checks if a status transition is allowed.
func canTransition(current, target string) bool {
	if allowed, ok := validTransitions[current]; ok {
		return allowed[target]
	}
	return false
}
