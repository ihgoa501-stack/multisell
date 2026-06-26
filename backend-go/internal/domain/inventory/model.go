package inventory

import "time"

// Inventory maps to the PostgreSQL "inventory" table.
type Inventory struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID           int64     `gorm:"column:sku_id;not null;unique" json:"sku_id"`
	Warehouse       string    `gorm:"column:warehouse;default:默认仓库" json:"warehouse"`
	Location        string    `gorm:"column:location" json:"location"`
	Quantity        int       `gorm:"column:quantity;default:0" json:"quantity"`
	LockedQuantity  int       `gorm:"column:locked_quantity;default:0" json:"locked_quantity"`
	SafetyStock     int       `gorm:"column:safety_stock;default:0" json:"safety_stock"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Inventory) TableName() string { return "inventory" }

// InventoryLog maps to the PostgreSQL "inventory_log" table.
type InventoryLog struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID      int64     `gorm:"column:sku_id;not null" json:"sku_id"`
	ChangeType string    `gorm:"column:change_type;not null" json:"change_type"`
	ChangeQty  int       `gorm:"column:change_qty;not null" json:"change_qty"`
	BeforeQty  int       `gorm:"column:before_qty" json:"before_qty"`
	AfterQty   int       `gorm:"column:after_qty" json:"after_qty"`
	Remark     string    `gorm:"column:remark" json:"remark"`
	Operator   string    `gorm:"column:operator" json:"operator"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (InventoryLog) TableName() string { return "inventory_log" }

// Warehouse is managed by the allocation module.
// Import allocation.Warehouse for the canonical definition.
// See: internal/domain/allocation/model.go
// InventoryWarehouse maps to the PostgreSQL "inventory_warehouse" table.
type InventoryWarehouse struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID           int64     `gorm:"column:sku_id;not null;uniqueIndex:uq_sku_warehouse" json:"sku_id"`
	WarehouseID     int64     `gorm:"column:warehouse_id;not null;uniqueIndex:uq_sku_warehouse" json:"warehouse_id"`
	Quantity        int       `gorm:"column:quantity;default:0" json:"quantity"`
	LockedQuantity  int       `gorm:"column:locked_quantity;default:0" json:"locked_quantity"`
	SafetyStock     int       `gorm:"column:safety_stock;default:0" json:"safety_stock"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (InventoryWarehouse) TableName() string { return "inventory_warehouse" }
