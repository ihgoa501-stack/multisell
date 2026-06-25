package price

import (
	"time"

	"github.com/shopspring/decimal"
)

// Price maps to the PostgreSQL "price" table.
type Price struct {
	ID        int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID     int64           `gorm:"column:sku_id;not null" json:"sku_id"`
	PriceType string          `gorm:"column:price_type;not null" json:"price_type"`
	Price     decimal.Decimal `gorm:"column:price;type:numeric(10,2);not null" json:"price"`
	StartTime *time.Time      `gorm:"column:start_time" json:"start_time,omitempty"`
	EndTime   *time.Time      `gorm:"column:end_time" json:"end_time,omitempty"`
	Status    int16           `gorm:"column:status;default:1" json:"status"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Price) TableName() string { return "price" }

// PriceChangeLog maps to the PostgreSQL "price_change_log" table.
type PriceChangeLog struct {
	ID         int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID      int64           `gorm:"column:sku_id;not null" json:"sku_id"`
	OldPrice   *decimal.Decimal `gorm:"column:old_price;type:numeric(10,2)" json:"old_price,omitempty"`
	NewPrice   *decimal.Decimal `gorm:"column:new_price;type:numeric(10,2)" json:"new_price,omitempty"`
	PriceType  string          `gorm:"column:price_type" json:"price_type"`
	ChangeType string          `gorm:"column:change_type" json:"change_type"`
	Operator   string          `gorm:"column:operator" json:"operator"`
	Remark     string          `gorm:"column:remark" json:"remark"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (PriceChangeLog) TableName() string { return "price_change_log" }
