package integrations

import "time"

// ---------- 发布域 (Listing) ----------

// PublishInput carries the data needed to publish a product to a platform.
type PublishInput struct {
	ProductID   int64
	PlatformID  int64
	AccountID   int64
	SKUs        []PublishSKU
	Prices      map[int64]string // sku_id -> price
	Inventories map[int64]int    // sku_id -> quantity

	// Product metadata for platform publication
	ProductName   string
	Description   string
	CategoryID    int64
	BrandID       int64
	MainImage     string
	Images        []string
	PackageHeight float64
	PackageWidth  float64
	PackageLength float64
	PackageWeight float64
}

// PublishSKU describes one SKU to publish.
type PublishSKU struct {
	SkuID   int64
	SkuCode string
}

// PublishResult holds the response from a platform after a publish attempt.
type PublishResult struct {
	PlatformProductID string                 `json:"platform_product_id"`
	PlatformSKU       string                 `json:"platform_sku"`
	PlatformURL       string                 `json:"platform_url"`
	PublishedData     map[string]interface{} `json:"published_data"`
	SyncMessage       string                 `json:"sync_message,omitempty"`
}

// SyncStatusInput carries the data needed to check listing sync status.
type SyncStatusInput struct {
	ListingID         int64
	PlatformID        int64
	PlatformProductID string
}

// ---------- 库存域 (Inventory) ----------

// SyncInventoryInput carries the data needed to sync inventory to a platform.
type SyncInventoryInput struct {
	PlatformID  int64
	SkuCode     string
	PlatformSKU string
	Quantity    int
}

// ---------- 物流域 (Logistics) ----------

// PushTrackingInput carries the data needed to push tracking info.
type PushTrackingInput struct {
	PlatformID     int64
	OrderSN        string
	TrackingNumber string
	CarrierCode    string
}

// ---------- 订单域 (Orders) ----------

// FetchOrdersInput carries the data needed to pull orders from a platform.
type FetchOrdersInput struct {
	PlatformID int64
	Since      time.Time
}

// PlatformOrder represents an order fetched from a platform.
type PlatformOrder struct {
	OrderSN        string              `json:"order_sn"`
	Status         string              `json:"status"`
	TotalAmount    string              `json:"total_amount"`
	ShippingFee    string              `json:"shipping_fee"`
	PaidAt         string              `json:"paid_at"`
	RecipientName  string              `json:"recipient_name"`
	RecipientPhone string              `json:"recipient_phone"`
	ShippingAddress string             `json:"shipping_address"`
	Items          []PlatformOrderItem `json:"items"`
}

// PlatformOrderItem is a single line item within a PlatformOrder.
type PlatformOrderItem struct {
	SkuCode   string `json:"sku_code"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unit_price"`
}

// ---------- 结算域 (Settlements) ----------

// FetchSettlementsInput carries the data needed to pull settlements from a platform.
type FetchSettlementsInput struct {
	PlatformID int64
	Since      time.Time
}

// PlatformSettlement represents a settlement / transaction record from a platform.
type PlatformSettlement struct {
	TransactionID   string `json:"transaction_id"`
	TransactionType string `json:"transaction_type"` // order_sale / refund / shipping_fee / platform_fee / payment_fee
	OrderSN         string `json:"order_sn,omitempty"`
	Amount          string `json:"amount"`
	Fee             string `json:"fee,omitempty"`
	Currency        string `json:"currency"`
	OccurredAt      string `json:"occurred_at"`
	Description     string `json:"description,omitempty"`
}

// ---------- 售后域 (Returns) ----------

// FetchReturnsInput carries the data needed to pull returns from a platform.
type FetchReturnsInput struct {
	PlatformID int64
	Since      time.Time
}

// PlatformReturn represents a return / refund request from a platform.
type PlatformReturn struct {
	ReturnID     string `json:"return_id"`
	OrderSN      string `json:"order_sn"`
	SkuCode      string `json:"sku_code"`
	Quantity     int    `json:"quantity"`
	Reason       string `json:"reason"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	RefundAmount string `json:"refund_amount,omitempty"`
}
