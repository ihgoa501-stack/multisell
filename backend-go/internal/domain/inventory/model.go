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

// ── Cross-Platform Sync ────────────────────────────────────────────────────

// CrossPlatformSyncResult holds the result of a cross-platform inventory sync.
type CrossPlatformSyncResult struct {
	ProductID          int64                `json:"product_id"`
	AvailableInventory int                  `json:"available_inventory"`
	TotalCommitted     int                  `json:"total_committed"`
	OversellDetected   bool                 `json:"oversell_detected"`
	OversellBy         int                  `json:"oversell_by,omitempty"`
	PlatformBreakdown  []PlatformCommitment `json:"platform_breakdown"`
	AlertGenerated     bool                 `json:"alert_generated"`
}

// PlatformCommitment describes how much stock is committed to a single platform.
type PlatformCommitment struct {
	PlatformID int64  `json:"platform_id"`
	Status     string `json:"status"`
	Committed  int    `json:"committed"`
	MaxAllowed int    `json:"max_allowed"`
}

// InventoryOversellLog maps to the "inventory_oversell_log" table.
type InventoryOversellLog struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID       int64      `gorm:"column:product_id;not null" json:"product_id"`
	AvailableStock  int        `gorm:"column:available_stock;not null" json:"available_stock"`
	TotalCommitted  int        `gorm:"column:total_committed;not null" json:"total_committed"`
	OversellBy      int        `gorm:"column:oversell_by;default:0" json:"oversell_by"`
	DetectedAt      time.Time  `gorm:"column:detected_at;autoCreateTime" json:"detected_at"`
	ResolvedAt      *time.Time `gorm:"column:resolved_at" json:"resolved_at,omitempty"`
	Status          string     `gorm:"column:status;default:open" json:"status"`
}

// TableName overrides the default table name.
func (InventoryOversellLog) TableName() string { return "inventory_oversell_log" }

// ── Safe Stock Config ───────────────────────────────────────────────────────

// InventorySafetyConfig stores per-SKU safety stock configuration.
type InventorySafetyConfig struct {
	ID            int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID         int64   `gorm:"column:sku_id;not null;uniqueIndex" json:"sku_id"`
	MinStockLevel int     `gorm:"column:min_stock_level;default:0" json:"min_stock_level"`
	MaxStockLevel int     `gorm:"column:max_stock_level;default:0" json:"max_stock_level"`
	LeadTimeDays  int     `gorm:"column:lead_time_days;default:7" json:"lead_time_days"`
	SafetyDays    int     `gorm:"column:safety_days;default:7" json:"safety_days"`
	DailyAvgSales float64 `gorm:"column:daily_avg_sales;default:0" json:"daily_avg_sales"`
	AutoReorder   bool    `gorm:"column:auto_reorder;default:false" json:"auto_reorder"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (InventorySafetyConfig) TableName() string { return "inventory_safety_config" }

// ── Multi-Platform Allocation ────────────────────────────────────────────────

// AllocationRecommendation suggests how to distribute stock across platforms.
type AllocationRecommendation struct {
	SkuID           int64                        `json:"sku_id"`
	TotalAvailable  int                          `json:"total_available"`
	ReservedTotal   int                          `json:"reserved_total"`
	Recommendations []PlatformAllocation         `json:"recommendations"`
	Unallocated     int                          `json:"unallocated"`
}

// PlatformAllocation is a single platform's recommended allocation.
type PlatformAllocation struct {
	PlatformID   int64   `json:"platform_id"`
	PlatformName string  `json:"platform_name"`
	SalesShare   float64 `json:"sales_share"`   // % of total sales across platforms
	CurrentStock int     `json:"current_stock"` // current committed stock
	Recommended  int     `json:"recommended"`   // recommended stock to allocate
	Priority     int     `json:"priority"`      // 1=highest
}

// ── Dead Stock ───────────────────────────────────────────────────────────────

// DeadStockRecord identifies a SKU with stagnant inventory.
type DeadStockRecord struct {
	SkuID         int64     `json:"sku_id"`
	SkuCode       string    `json:"sku_code"`
	ProductName   string    `json:"product_name"`
	Warehouse     string    `json:"warehouse"`
	CurrentQty    int       `json:"current_qty"`
	LastMovedAt   *time.Time `json:"last_moved_at,omitempty"`
	DaysSinceMove int       `json:"days_since_move"`
	Status        string    `json:"status"` // "dead", "slow", "normal"
	Suggestion    string    `json:"suggestion"`
}

// DeadStockLog maps to the "dead_stock_log" table for tracking dead stock.
type DeadStockLog struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID         int64      `gorm:"column:sku_id;not null;index" json:"sku_id"`
	Quantity      int        `gorm:"column:quantity;default:0" json:"quantity"`
	DaysSinceMove int        `gorm:"column:days_since_move;default:0" json:"days_since_move"`
	Status        string     `gorm:"column:status;default:normal" json:"status"`
	Notes         string     `gorm:"column:notes" json:"notes"`
	DetectedAt    time.Time  `gorm:"column:detected_at;autoCreateTime" json:"detected_at"`
	ResolvedAt    *time.Time `gorm:"column:resolved_at" json:"resolved_at,omitempty"`
}

func (DeadStockLog) TableName() string { return "dead_stock_log" }
