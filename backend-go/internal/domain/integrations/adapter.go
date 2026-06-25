package integrations

import "context"

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
}
