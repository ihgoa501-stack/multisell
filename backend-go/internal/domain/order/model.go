package order

import (
	"time"
)

// Order maps to "sales_order".
type Order struct {
	ID             int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderNo        string          `gorm:"column:order_no;uniqueIndex" json:"order_no"`
	PlatformID     *int64          `gorm:"column:platform_id" json:"platform_id,omitempty"`
	Status         string          `gorm:"column:status;default:pending" json:"status"`
	TrackingNumber string          `gorm:"column:tracking_number" json:"tracking_number"`
	RecipientName  string          `gorm:"column:recipient_name" json:"recipient_name"`
	RecipientPhone string          `gorm:"column:recipient_phone" json:"recipient_phone"`
	ShippingAddress string         `gorm:"column:shipping_address" json:"shipping_address"`
	TotalAmount    float64         `gorm:"column:total_amount;default:0" json:"total_amount"`
	ShippingFee    float64         `gorm:"column:shipping_fee;default:0" json:"shipping_fee"`
	PayAmount      float64         `gorm:"column:pay_amount;default:0" json:"pay_amount"`
	PlatformFee    float64         `gorm:"column:platform_fee;default:0" json:"platform_fee"`
	PaymentFee     float64         `gorm:"column:payment_fee;default:0" json:"payment_fee"`
	OtherFee       float64         `gorm:"column:other_fee;default:0" json:"other_fee"`
	ProductCost    float64         `gorm:"column:product_cost;default:0" json:"product_cost"`
	ProfitAmount   float64         `gorm:"column:profit_amount;default:0" json:"profit_amount"`
	ProfitMargin   float64         `gorm:"column:profit_margin;default:0" json:"profit_margin"`
	PaymentMethod  string          `gorm:"column:payment_method" json:"payment_method"`
	Remark         string          `gorm:"column:remark" json:"remark"`
	PaidAt         *time.Time      `gorm:"column:paid_at" json:"paid_at,omitempty"`
	ShippedAt      *time.Time      `gorm:"column:shipped_at" json:"shipped_at,omitempty"`
	DeliveredAt    *time.Time      `gorm:"column:delivered_at" json:"delivered_at,omitempty"`
	CancelledAt    *time.Time      `gorm:"column:cancelled_at" json:"cancelled_at,omitempty"`
	CreatedAt      time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Order) TableName() string { return "sales_order" }

// OrderItem maps to "sales_order_item".
type OrderItem struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID     int64     `gorm:"column:order_id;not null;index" json:"order_id"`
	SkuID       int64     `gorm:"column:sku_id;not null" json:"sku_id"`
	ProductID   int64     `gorm:"column:product_id;not null" json:"product_id"`
	ProductName string    `gorm:"column:product_name;not null" json:"product_name"`
	SkuCode     string    `gorm:"column:sku_code" json:"sku_code"`
	SpecDesc    string    `gorm:"column:spec_desc" json:"spec_desc"`
	UnitPrice   float64   `gorm:"column:unit_price;not null" json:"unit_price"`
	Quantity    int       `gorm:"column:quantity;not null" json:"quantity"`
	Subtotal    float64   `gorm:"column:subtotal;not null" json:"subtotal"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (OrderItem) TableName() string { return "sales_order_item" }

// OrderStatusLog maps to "sales_order_status_log".
type OrderStatusLog struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID    int64     `gorm:"column:order_id;not null;index" json:"order_id"`
	FromStatus string    `gorm:"column:from_status" json:"from_status"`
	ToStatus   string    `gorm:"column:to_status;not null" json:"to_status"`
	Operator   string    `gorm:"column:operator" json:"operator"`
	Remark     string    `gorm:"column:remark" json:"remark"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (OrderStatusLog) TableName() string { return "sales_order_status_log" }

// OrderDetail is the composite detail payload.
type OrderDetail struct {
	Order     Order        `json:"order"`
	Items     []OrderItem  `json:"items"`
	StatusLogs []OrderStatusLog `json:"status_logs"`
}

// CreateOrderInput is the payload for POST /order.
type CreateOrderInput struct {
	OrderNo        string  `json:"order_no" binding:"required"`
	PlatformID     *int64  `json:"platform_id"`
	Status         string  `json:"status"`
	TrackingNumber string  `json:"tracking_number"`
	RecipientName  string  `json:"recipient_name"`
	RecipientPhone string  `json:"recipient_phone"`
	ShippingAddress string `json:"shipping_address"`
	TotalAmount    *float64 `json:"total_amount"`
	ShippingFee    *float64 `json:"shipping_fee"`
	PayAmount      *float64 `json:"pay_amount"`
	PaymentMethod  string  `json:"payment_method"`
	Remark         string  `json:"remark"`
	Items          []OrderItemInput `json:"items"`
}

// OrderItemInput is one line in CreateOrderInput.
type OrderItemInput struct {
	SkuID       int64   `json:"sku_id" binding:"required"`
	ProductID   int64   `json:"product_id" binding:"required"`
	ProductName string  `json:"product_name" binding:"required"`
	SkuCode     string  `json:"sku_code"`
	SpecDesc    string  `json:"spec_desc"`
	UnitPrice   float64 `json:"unit_price" binding:"required"`
	Quantity    int     `json:"quantity" binding:"required"`
}

// UpdateOrderInput allows partial updates.
type UpdateOrderInput struct {
	Status         *string  `json:"status"`
	TrackingNumber *string  `json:"tracking_number"`
	RecipientName  *string  `json:"recipient_name"`
	RecipientPhone *string  `json:"recipient_phone"`
	ShippingAddress *string `json:"shipping_address"`
	PaymentMethod  *string  `json:"payment_method"`
	Remark         *string  `json:"remark"`
}

// OrderListFilter captures query parameters.
type OrderListFilter struct {
	Search     string
	Status     string
	PlatformID *int64
}

// OrderSummary is a lightweight aggregation for dashboard.
type OrderSummary struct {
	Total      int64              `json:"total"`
	ByStatus   map[string]int64   `json:"by_status"`
	TotalRevenue float64          `json:"total_revenue"`
	TotalProfit float64           `json:"total_profit"`
}
