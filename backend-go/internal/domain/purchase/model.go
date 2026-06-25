package purchase

import (
	"encoding/json"
	"time"
)

// PurchaseOrderStatus constants.
const (
	StatusDraft             = "draft"
	StatusPending           = "pending"
	StatusApproved          = "approved"
	StatusPartiallyReceived = "partially_received"
	StatusCompleted         = "completed"
	StatusCancelled         = "cancelled"
)

// validTransitions maps current status -> set of allowed next statuses.
var validTransitions = map[string]map[string]bool{
	StatusDraft:             {StatusPending: true, StatusApproved: true, StatusCancelled: true},
	StatusPending:           {StatusApproved: true, StatusCancelled: true},
	StatusApproved:          {StatusPartiallyReceived: true, StatusCompleted: true, StatusCancelled: true},
	StatusPartiallyReceived: {StatusCompleted: true},
	StatusCompleted:         {},
	StatusCancelled:         {},
}

// PurchaseOrder maps to "purchase_order".
type PurchaseOrder struct {
	ID               int64               `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderNo          string              `gorm:"column:order_no;uniqueIndex" json:"order_no"`
	SupplierID       int64               `gorm:"column:supplier_id;not null" json:"supplier_id"`
	Status           string              `gorm:"column:status;default:draft" json:"status"`
	TotalAmount      float64             `gorm:"column:total_amount;default:0" json:"total_amount"`
	ExpectedDelivery *string             `gorm:"column:expected_delivery" json:"expected_delivery,omitempty"`
	Remark           string              `gorm:"column:remark" json:"remark"`
	CreatedAt        time.Time           `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time           `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Items            []PurchaseOrderItem `gorm:"foreignKey:PurchaseOrderID" json:"items,omitempty"`
}

// TableName overrides the default table name.
func (PurchaseOrder) TableName() string { return "purchase_order" }

// PurchaseOrderItem maps to "purchase_order_item".
type PurchaseOrderItem struct {
	ID              int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PurchaseOrderID int64   `gorm:"column:purchase_order_id;not null;index" json:"purchase_order_id"`
	SkuID           int64   `gorm:"column:sku_id;not null" json:"sku_id"`
	Quantity        int     `gorm:"column:quantity;not null" json:"quantity"`
	ReceivedQty     int     `gorm:"column:received_qty;default:0" json:"received_qty"`
	UnitPrice       float64 `gorm:"column:unit_price;not null" json:"unit_price"`
	Subtotal        float64 `gorm:"column:subtotal;not null" json:"subtotal"`
}

// TableName overrides the default table name.
func (PurchaseOrderItem) TableName() string { return "purchase_order_item" }

// Supplier maps to "purchase_supplier".
type Supplier struct {
	ID            int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name          string          `gorm:"column:name;not null" json:"name"`
	ContactPerson string          `gorm:"column:contact_person" json:"contact_person"`
	Phone         string          `gorm:"column:phone" json:"phone"`
	Email         string          `gorm:"column:email" json:"email"`
	Address       string          `gorm:"column:address" json:"address"`
	KpiScore      float64         `gorm:"column:kpi_score;default:0" json:"kpi_score"`
	PriceHistory  json.RawMessage `gorm:"column:price_history;type:jsonb" json:"price_history,omitempty"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (Supplier) TableName() string { return "purchase_supplier" }

// PurchaseSuggestion maps to "purchase_suggestion".
type PurchaseSuggestion struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID        int64     `gorm:"column:sku_id;not null;index" json:"sku_id"`
	SuggestedQty int       `gorm:"column:suggested_qty;not null" json:"suggested_qty"`
	Reason       string    `gorm:"column:reason;not null" json:"reason"`
	Status       string    `gorm:"column:status;default:pending" json:"status"`
	GeneratedAt  time.Time `gorm:"column:generated_at;autoCreateTime" json:"generated_at"`
}

// TableName overrides the default table name.
func (PurchaseSuggestion) TableName() string { return "purchase_suggestion" }

// ---------- Request / Response structs ----------

// CreateOrderInput is the payload for POST /purchase/orders.
type CreateOrderInput struct {
	SupplierID       int64            `json:"supplier_id" binding:"required"`
	ExpectedDelivery *string          `json:"expected_delivery"`
	Remark           string           `json:"remark"`
	Items            []OrderItemInput `json:"items" binding:"required,min=1"`
}

// OrderItemInput is one line in CreateOrderInput.
type OrderItemInput struct {
	SkuID     int64   `json:"sku_id" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required,min=1"`
	UnitPrice float64 `json:"unit_price" binding:"required"`
}

// ReceiveItemInput is one item's received quantity.
type ReceiveItemInput struct {
	ItemID      int64 `json:"item_id" binding:"required"`
	ReceivedQty int   `json:"received_qty" binding:"required,min=0"`
}

// ReceiveOrderInput is the payload for POST /purchase/orders/:id/receive.
type ReceiveOrderInput struct {
	Items []ReceiveItemInput `json:"items" binding:"required,min=1"`
}

// PurchaseOrderListFilter captures query parameters for listing orders.
type PurchaseOrderListFilter struct {
	Status     string
	SupplierID *int64
	Search     string
}

// CreateSupplierInput is the payload for POST /purchase/suppliers.
type CreateSupplierInput struct {
	Name          string          `json:"name" binding:"required"`
	ContactPerson string          `json:"contact_person"`
	Phone         string          `json:"phone"`
	Email         string          `json:"email"`
	Address       string          `json:"address"`
	PriceHistory  json.RawMessage `json:"price_history"`
}

// UpdateSupplierInput is the payload for PUT /purchase/suppliers/:id.
type UpdateSupplierInput struct {
	Name          *string         `json:"name"`
	ContactPerson *string         `json:"contact_person"`
	Phone         *string         `json:"phone"`
	Email         *string         `json:"email"`
	Address       *string         `json:"address"`
	PriceHistory  json.RawMessage `json:"price_history"`
}

// SupplierKPIResponse is the KPI detail response.
type SupplierKPIResponse struct {
	SupplierID   int64   `json:"supplier_id"`
	SupplierName string  `json:"supplier_name"`
	KpiScore     float64 `json:"kpi_score"`
	OrderCount   int64   `json:"order_count"`
	OnTimeRate   float64 `json:"on_time_rate"`
}
