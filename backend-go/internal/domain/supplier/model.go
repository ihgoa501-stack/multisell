package supplier

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Supplier maps to the PostgreSQL "supplier" table.
type Supplier struct {
	ID            int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name          string          `gorm:"column:name;not null" json:"name"`
	ContactPerson string          `gorm:"column:contact_person" json:"contact_person"`
	ContactPhone  string          `gorm:"column:contact_phone" json:"contact_phone"`
	Email         string          `gorm:"column:email" json:"email"`
	Address       string          `gorm:"column:address" json:"address"`
	Status        int16           `gorm:"column:status;default:1" json:"status"`
	Remark        string          `gorm:"column:remark" json:"remark"`
	KpiScore      float64         `gorm:"column:kpi_score;default:0" json:"kpi_score"`
	PriceHistory  json.RawMessage `gorm:"column:price_history;type:jsonb" json:"price_history,omitempty"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
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

// SupplierScore stores computed credibility scores for a supplier.
type SupplierScore struct {
	ID                  int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SupplierID          int64     `gorm:"column:supplier_id;not null;uniqueIndex" json:"supplier_id"`
	OnTimeDeliveryRate  float64   `gorm:"column:on_time_delivery_rate;default:0" json:"on_time_delivery_rate"`   // 0-100
	QualityPassRate     float64   `gorm:"column:quality_pass_rate;default:0" json:"quality_pass_rate"`           // 0-100
	CommunicationScore  float64   `gorm:"column:communication_score;default:0" json:"communication_score"`       // 0-100
	OrderFulfillmentPct float64   `gorm:"column:order_fulfillment_pct;default:0" json:"order_fulfillment_pct"`   // % of orders fulfilled
	AvgLeadTimeDays     float64   `gorm:"column:avg_lead_time_days;default:0" json:"avg_lead_time_days"`
	ReliabilityScore    float64   `gorm:"column:reliability_score;default:0" json:"reliability_score"`           // composite 0-100
	DataFreshness       int       `gorm:"column:data_freshness;default:0" json:"data_freshness"`                 // days since last update
	LastOrderDate       *time.Time `gorm:"column:last_order_date" json:"last_order_date,omitempty"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (SupplierScore) TableName() string { return "supplier_score" }

// SupplierComparisonResponse is the response for product-vs-supplier comparison.
type SupplierComparisonResponse struct {
	ProductID   int64             `json:"product_id"`
	ProductName string            `json:"product_name"`
	Suppliers   []SupplierRow     `json:"suppliers"`
	SpecNames   map[string]string `json:"spec_names,omitempty"`
}

// SupplierRow is one supplier's row in a comparison.
type SupplierRow struct {
	SupplierID     int64            `json:"supplier_id"`
	SupplierName   string           `json:"supplier_name"`
	SupplyPrice    *decimal.Decimal `json:"supply_price,omitempty"`
	MinOrderQty    int              `json:"min_order_qty"`
	SpecSummary    string           `json:"spec_summary"`     // from sourcing_1688_product
	PackageLength  *decimal.Decimal `json:"package_length_cm,omitempty"`
	PackageWidth   *decimal.Decimal `json:"package_width_cm,omitempty"`
	PackageHeight  *decimal.Decimal `json:"package_height_cm,omitempty"`
	PackageWeight  *decimal.Decimal `json:"package_weight_kg,omitempty"`
}
