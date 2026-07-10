package integrations

import (
	"context"
	"net/http"
)

// WebhookVerifier is an optional interface that adapters can implement to
// verify incoming webhook payload signatures before any processing occurs.
// The webhook handler checks for this interface and rejects unauthenticated
// payloads before they reach the event bus.
type WebhookVerifier interface {
	// VerifyWebhookSignature checks the HMAC or other signature on an incoming
	// webhook request. body is the raw request body, and headers contains all
	// HTTP headers from the request (canonicalized per Go's http.Header).
	// Returns true if and only if the signature is valid.
	VerifyWebhookSignature(ctx context.Context, body []byte, headers http.Header) bool
}

// PlatformAdapter defines the interface for interacting with a third-party
// e-commerce platform (Shopify, Lazada, Shopee, etc.).
// Each method maps to one capability domain: listing, inventory, logistics,
// orders, settlements, and returns.
type PlatformAdapter interface {
	// ---------- 发布域 ----------

	// Publish publishes a product (with its SKUs, prices, and inventories) to
	// the external platform and returns the platform-side result.
	Publish(ctx context.Context, input *PublishInput) (*PublishResult, error)

	// SyncStatus checks the current sync / publish status on the platform.
	SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error)

	// ValidateCredentials verifies that the stored API credentials for the
	// given platform account are still valid.
	ValidateCredentials(ctx context.Context, accountID int64) (bool, error)

	// ---------- 库存域 ----------

	// SyncInventory pushes a local inventory level to the platform.
	SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error)

	// ---------- 物流域 ----------

	// PushTracking sends shipping tracking information to the platform.
	PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error)

	// ---------- 订单域 ----------

	// FetchOrders pulls new / updated orders from the platform since the
	// given timestamp.
	FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error)

	// ---------- 结算域 ----------

	// FetchSettlements pulls settlement / transaction records from the
	// platform since the given timestamp.
	FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error)

	// ---------- 售后域 ----------

	// FetchReturns pulls return / refund requests from the platform since the
	// given timestamp.
	FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error)

	// ---------- 原始 API 域 ----------

	// FetchRaw makes a raw API call to the platform and returns the response body
	// as bytes. This is the base method for the AI mapper — it separates HTTP
	// communication from data transformation.
	// endpoint: the platform-specific API path (e.g. "/v3/posting/fbs/list")
	// payload: the request body, nil for GET-like calls
	// Returns the raw response body as bytes, or an error.
	FetchRaw(ctx context.Context, platformID int64, endpoint string, payload interface{}) ([]byte, error)
}
