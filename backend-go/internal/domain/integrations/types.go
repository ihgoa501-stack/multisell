package integrations

import (
	"context"
	"encoding/json"
	"time"
)

// ---------- 发布域 (Listing) ----------

// PublishInput carries the data needed to publish a product to a platform.
type PublishInput struct {
	ProductID   int64
	PlatformID  int64
	AccountID   int64
	SKUs        []PublishSKU
	Prices      map[int64]string // sku_id -> price
	Inventories map[int64]int    // sku_id -> quantity

	// IdempotencyKey prevents duplicate publishes when network failures cause retries.
	// Generated from listing_task.id + execution_mode + created_at by NewPublishHook.
	// ponytail: simple string key. Upgrade to EventBus + outbox when cross-service idempotency matters.
	IdempotencyKey string

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

// ---------- 执行模式 (Execution Mode) ----------

// ExecutionMode controls whether platform write operations are actually executed.
// ponytail: int8 is sufficient — 4 modes, no extra JSON/DB serialization needed.
type ExecutionMode int8

const (
	// ExecutionModeDryRun simulates the write and returns a mock result.
	// This is the DEFAULT mode — real writes require explicit opt-in.
	ExecutionModeDryRun ExecutionMode = 0
	// ExecutionModeSandbox routes writes through the platform's sandbox API.
	ExecutionModeSandbox ExecutionMode = 1
	// ExecutionModeApprovalRequired marks the operation as requiring explicit
	// user confirmation before execution. checkWriteMode passes through so the
	// handler layer can gate on it.
	ExecutionModeApprovalRequired ExecutionMode = 2
	// ExecutionModeProduction executes writes against the real platform API.
	ExecutionModeProduction ExecutionMode = 3
)

// modeCtxKey is the context key for execution mode.
type modeCtxKey struct{}

// WithExecutionMode stores the execution mode in the context.
func WithExecutionMode(ctx context.Context, mode ExecutionMode) context.Context {
	return context.WithValue(ctx, modeCtxKey{}, mode)
}

// ExecutionModeFromCtx retrieves the execution mode from the context.
// Returns ExecutionModeDryRun if not set.
func ExecutionModeFromCtx(ctx context.Context) ExecutionMode {
	if v, ok := ctx.Value(modeCtxKey{}).(ExecutionMode); ok {
		return v
	}
	return ExecutionModeDryRun
}

// String returns a human-readable name for the execution mode.
func (m ExecutionMode) String() string {
	switch m {
	case ExecutionModeDryRun:
		return "dry_run"
	case ExecutionModeSandbox:
		return "sandbox"
	case ExecutionModeApprovalRequired:
		return "approval_required"
	case ExecutionModeProduction:
		return "production"
	default:
		return "unknown"
	}
}

// IsWriteAllowed returns true if the mode permits actual writes to the
// external platform. Dry-run mode does not perform real writes.
func (m ExecutionMode) IsWriteAllowed() bool {
	return m == ExecutionModeProduction || m == ExecutionModeApprovalRequired
}

// ---------- 环境配置 (Environment Mode) ----------

// ModeRequest is the payload for PUT /platform-integrations/:id/mode.
type ModeRequest struct {
	Mode ExecutionMode `json:"mode" binding:"required"`
}

// ModeResponse is the response for GET /platform-integrations/:id/mode.
type ModeResponse struct {
	Mode        ExecutionMode `json:"mode"`
	AccountID   int64         `json:"account_id"`
	AccountName string        `json:"account_name"`
}

// ---------- 写回门禁域 (Write-Back) ----------

// WriteBackRequest is the payload for the generic write-back endpoint.
type WriteBackRequest struct {
	Action      string          `json:"action" binding:"required"` // sync_inventory, push_tracking, sync_status, validate_credentials
	AccountID   int64           `json:"account_id" binding:"required"`
	Payload     json.RawMessage `json:"payload"`
	ReferenceID string          `json:"reference_id,omitempty"`

	// Operator is set server-side from JWT — never accepted from the client.
	Operator string `json:"-"`
}

// WriteBackResult is the response for the write-back endpoint.
type WriteBackResult struct {
	ReferenceID string      `json:"reference_id"`
	Action      string      `json:"action"`
	Success     bool        `json:"success"`
	Message     string      `json:"message"`
	Retryable   bool        `json:"retryable"`
	Result      interface{} `json:"result,omitempty"`
}

// ---------- 平台事件域 (Platform Events) ----------

// PlatformEvent represents an event received from an external e-commerce platform.
type PlatformEvent struct {
	PlatformID int64                  `json:"platform_id"`
	EventType  string                 `json:"event_type"` // listing_blocked, listing_live, price_changed, inventory_changed
	ProductID  string                 `json:"product_id"` // platform's product ID
	SKUCode    string                 `json:"sku_code"`
	Data       map[string]interface{} `json:"data,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
}

// PlatformEventReceiver defines the interface for receiving events from
// external platforms. Implementations should poll or listen via webhook
// and push events to the provided channel.
type PlatformEventReceiver interface {
	Start(ctx context.Context, events chan<- PlatformEvent) error
}
