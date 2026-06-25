package supplier

import (
	"time"

	"github.com/shopspring/decimal"
)

// Supplier maps to the PostgreSQL "supplier" table.
type Supplier struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"column:name;not null" json:"name"`
	ContactPerson string    `gorm:"column:contact_person" json:"contact_person"`
	ContactPhone  string    `gorm:"column:contact_phone" json:"contact_phone"`
	Email         string    `gorm:"column:email" json:"email"`
	Address       string    `gorm:"column:address" json:"address"`
	Status        int16     `gorm:"column:status;default:1" json:"status"`
	Remark        string    `gorm:"column:remark" json:"remark"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Supplier) TableName() string { return "supplier" }

// ProductSupplier maps to the PostgreSQL "product_supplier" table.
type ProductSupplier struct {
	ID           int64            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID    int64            `gorm:"column:product_id;not null" json:"product_id"`
	SupplierID   int64            `gorm:"column:supplier_id;not null" json:"supplier_id"`
	SupplyPrice  *decimal.Decimal `gorm:"column:supply_price;type:numeric(10,2)" json:"supply_price,omitempty"`
	MinOrderQty  int              `gorm:"column:min_order_qty;default:1" json:"min_order_qty"`
	CreatedAt    time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (ProductSupplier) TableName() string { return "product_supplier" }
