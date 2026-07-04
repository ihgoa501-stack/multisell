package allocation

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Warehouse maps to the PostgreSQL "warehouse" table.
type Warehouse struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	Code      string    `gorm:"column:code;unique" json:"code"`
	Address   string    `gorm:"column:address" json:"address"`
	Contact   string    `gorm:"column:contact" json:"contact"`
	Phone     string    `gorm:"column:phone" json:"phone"`
	IsDefault int16     `gorm:"column:is_default;default:0" json:"is_default"`
	Status    int16     `gorm:"column:status;default:1" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Warehouse) TableName() string { return "warehouse" }

// AllocationRule maps to the PostgreSQL "allocation_rule" table.
type AllocationRule struct {
	ID            int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name          string          `gorm:"column:name;not null" json:"name"`
	SkuID         int64           `gorm:"column:sku_id;default:0;index" json:"sku_id"`
	Priority      int             `gorm:"column:priority;default:0" json:"priority"`
	RuleType      string          `gorm:"column:rule_type;not null" json:"rule_type"`
	WarehouseID   int64           `gorm:"column:warehouse_id;not null" json:"warehouse_id"`
	AllocationPct decimal.Decimal `gorm:"column:allocation_pct;type:numeric(5,2);default:100" json:"allocation_pct"`
	AllocationQty int             `gorm:"column:allocation_qty;default:0" json:"allocation_qty"`
	Status        int16           `gorm:"column:status;default:1" json:"status"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (AllocationRule) TableName() string { return "allocation_rule" }

// CostAllocationBatch maps to the PostgreSQL "cost_allocation_batch" table.
type CostAllocationBatch struct {
	ID               int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AllocationType   string          `gorm:"column:allocation_type;not null" json:"allocation_type"`
	AllocationMethod string          `gorm:"column:allocation_method;not null" json:"allocation_method"`
	TotalAmount      decimal.Decimal `gorm:"column:total_amount;type:numeric(14,2);not null" json:"total_amount"`
	Currency         string          `gorm:"column:currency;default:CNY" json:"currency"`
	SourceFilename   string          `gorm:"column:source_filename" json:"source_filename"`
	RowCount         int             `gorm:"column:row_count;default:0" json:"row_count"`
	Status           string          `gorm:"column:status;default:imported" json:"status"`
	PostedCount      int             `gorm:"column:posted_count;default:0" json:"posted_count"`
	CreatedBy        string          `gorm:"column:created_by" json:"created_by"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (CostAllocationBatch) TableName() string { return "cost_allocation_batch" }

// CostAllocationItem maps to the PostgreSQL "cost_allocation_item" table.
type CostAllocationItem struct {
	ID              int64            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BatchID         int64            `gorm:"column:batch_id;not null" json:"batch_id"`
	RowNumber       int              `gorm:"column:row_number;not null" json:"row_number"`
	SkuID           int64            `gorm:"column:sku_id" json:"sku_id"`
	SkuCode         string           `gorm:"column:sku_code" json:"sku_code"`
	OrderID         int64            `gorm:"column:order_id" json:"order_id"`
	Quantity        int              `gorm:"column:quantity;default:0" json:"quantity"`
	WeightKg        *decimal.Decimal `gorm:"column:weight_kg;type:numeric(10,3)" json:"weight_kg,omitempty"`
	VolumeM3        *decimal.Decimal `gorm:"column:volume_m3;type:numeric(10,4)" json:"volume_m3,omitempty"`
	ItemValue       *decimal.Decimal `gorm:"column:item_value;type:numeric(14,2)" json:"item_value,omitempty"`
	AllocationFactor *decimal.Decimal `gorm:"column:allocation_factor;type:numeric(14,4)" json:"allocation_factor,omitempty"`
	AllocatedAmount  decimal.Decimal  `gorm:"column:allocated_amount;type:numeric(14,2);default:0" json:"allocated_amount"`
	CostLayer       string           `gorm:"column:cost_layer;default:allocated" json:"cost_layer"`
	PostedToLedger  int             `gorm:"column:posted_to_ledger;default:0" json:"posted_to_ledger"`
	RawPayload      json.RawMessage  `gorm:"column:raw_payload;type:jsonb" json:"raw_payload,omitempty"`
	CreatedAt       time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (CostAllocationItem) TableName() string { return "cost_allocation_item" }
