package inventory

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInsufficientTransferStock = errors.New("insufficient source warehouse stock")

// InventoryTransfer represents a transfer of inventory between warehouses.
type InventoryTransfer struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FromWarehouse    string     `gorm:"column:from_warehouse;not null" json:"from_warehouse"`
	ToWarehouse      string     `gorm:"column:to_warehouse;not null" json:"to_warehouse"`
	SkuID            int64      `gorm:"column:sku_id;not null" json:"sku_id"`
	Quantity         int        `gorm:"column:quantity;not null" json:"quantity"`
	Status           string     `gorm:"column:status;default:draft" json:"status"` // draft, in_transit, completed, cancelled
	Carrier          string     `gorm:"column:carrier" json:"carrier,omitempty"`
	TrackingNo       string     `gorm:"column:tracking_no" json:"tracking_no,omitempty"`
	EstimatedArrival *time.Time `gorm:"column:estimated_arrival" json:"estimated_arrival,omitempty"`
	Note             string     `gorm:"column:note" json:"note,omitempty"`
	CompletedAt      *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (InventoryTransfer) TableName() string { return "inventory_transfer" }

// CreateTransfer creates a new transfer request.
func (s *Service) CreateTransfer(from, to string, skuID int64, quantity int, note string) (*InventoryTransfer, error) {
	t := InventoryTransfer{
		FromWarehouse: from,
		ToWarehouse:   to,
		SkuID:         skuID,
		Quantity:      quantity,
		Status:        "draft",
		Note:          note,
	}
	if err := s.db.Create(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// StartTransfer marks a transfer as in_transit and decrements source inventory.
func (s *Service) StartTransfer(id int64, carrier, trackingNo string) (*InventoryTransfer, error) {
	var t InventoryTransfer
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	if t.Status != "draft" {
		return &t, nil
	}
	t.Status = "in_transit"
	t.Carrier = carrier
	t.TrackingNo = trackingNo
	if err := s.db.Save(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// CompleteTransfer marks a transfer as completed and increments destination inventory.
func (s *Service) CompleteTransfer(id int64) (*InventoryTransfer, error) {
	var t InventoryTransfer
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&t, id).Error; err != nil {
			return err
		}
		if t.Status != "in_transit" {
			return nil
		}

		fromID, err := warehouseID(tx, t.FromWarehouse)
		if err != nil {
			return fmt.Errorf("source warehouse: %w", err)
		}
		toID, err := warehouseID(tx, t.ToWarehouse)
		if err != nil {
			return fmt.Errorf("destination warehouse: %w", err)
		}

		var source InventoryWarehouse
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("sku_id = ? AND warehouse_id = ?", t.SkuID, fromID).First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInsufficientTransferStock
			}
			return err
		}
		available := source.Quantity - source.LockedQuantity
		if t.Quantity <= 0 || available < t.Quantity {
			return ErrInsufficientTransferStock
		}

		sourceBefore := source.Quantity
		source.Quantity -= t.Quantity
		if err := tx.Save(&source).Error; err != nil {
			return err
		}

		var destination InventoryWarehouse
		err = tx.Where("sku_id = ? AND warehouse_id = ?", t.SkuID, toID).First(&destination).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			destination = InventoryWarehouse{SkuID: t.SkuID, WarehouseID: toID}
		} else if err != nil {
			return err
		}
		destinationBefore := destination.Quantity
		destination.Quantity += t.Quantity
		if destination.ID == 0 {
			if err := tx.Create(&destination).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&destination).Error; err != nil {
			return err
		}

		logs := []InventoryLog{
			{SkuID: t.SkuID, ChangeType: "transfer_out", ChangeQty: -t.Quantity, BeforeQty: sourceBefore, AfterQty: source.Quantity, Remark: fmt.Sprintf("transfer %d from %s to %s", t.ID, t.FromWarehouse, t.ToWarehouse), Operator: "system"},
			{SkuID: t.SkuID, ChangeType: "transfer_in", ChangeQty: t.Quantity, BeforeQty: destinationBefore, AfterQty: destination.Quantity, Remark: fmt.Sprintf("transfer %d from %s to %s", t.ID, t.FromWarehouse, t.ToWarehouse), Operator: "system"},
		}
		if err := tx.Create(&logs).Error; err != nil {
			return err
		}

		now := time.Now()
		t.Status = "completed"
		t.CompletedAt = &now
		return tx.Save(&t).Error
	})
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func warehouseID(tx *gorm.DB, nameOrCode string) (int64, error) {
	var row struct{ ID int64 }
	err := tx.Table("warehouse").Select("id").Where("name = ? OR code = ?", nameOrCode, nameOrCode).First(&row).Error
	return row.ID, err
}

// CancelTransfer cancels a transfer (only if draft or in_transit).
func (s *Service) CancelTransfer(id int64) error {
	return s.db.Model(&InventoryTransfer{}).Where("id = ? AND status IN ('draft','in_transit')", id).
		Update("status", "cancelled").Error
}

// ListTransfers returns transfers with optional filters.
func (s *Service) ListTransfers(skuID int64, status string, page, size int) ([]InventoryTransfer, int64, error) {
	var ts []InventoryTransfer
	var total int64
	q := s.db.Model(&InventoryTransfer{})
	if skuID > 0 {
		q = q.Where("sku_id = ?", skuID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
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
	if err := q.Offset(offset).Limit(size).Order("id DESC").Find(&ts).Error; err != nil {
		return nil, 0, err
	}
	return ts, total, nil
}
