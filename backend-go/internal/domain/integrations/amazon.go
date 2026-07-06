package integrations

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AmazonAdapter implements PlatformAdapter for Amazon SP-API / Selling Partner API.
type AmazonAdapter struct {
	httpClient *http.Client
	db         *gorm.DB
	logger     *zap.Logger
}

// NewAmazonAdapter creates a new Amazon adapter.
func NewAmazonAdapter(db *gorm.DB, logger *zap.Logger) *AmazonAdapter {
	return &AmazonAdapter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		db:         db,
		logger:     logger,
	}
}

func (a *AmazonAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("amazon publish: no SKUs")
	}
	return &PublishResult{
		PlatformProductID: fmt.Sprintf("amazon-pending-%d", input.ProductID),
		PlatformSKU:       input.SKUs[0].SkuCode,
		PlatformURL:       "",
		PublishedData:     map[string]interface{}{"status": "stub"},
		SyncMessage:       "amazon adapter stub: publish not yet implemented",
	}, nil
}

func (a *AmazonAdapter) SyncStatus(_ context.Context, input *SyncStatusInput) (string, error) {
	if input.PlatformProductID == "" {
		return "unknown", fmt.Errorf("amazon sync_status: empty platform product id")
	}
	return "synced", nil
}

func (a *AmazonAdapter) ValidateCredentials(_ context.Context, accountID int64) (bool, error) {
	return false, fmt.Errorf("amazon ValidateCredentials: not yet implemented for account %d", accountID)
}

func (a *AmazonAdapter) SyncInventory(_ context.Context, input *SyncInventoryInput) (bool, error) {
	return false, fmt.Errorf("amazon sync_inventory: not yet implemented for sku %s", input.SkuCode)
}

func (a *AmazonAdapter) PushTracking(_ context.Context, input *PushTrackingInput) (bool, error) {
	return false, fmt.Errorf("amazon push_tracking: not yet implemented for order %s", input.OrderSN)
}

func (a *AmazonAdapter) FetchOrders(_ context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	return []*PlatformOrder{}, nil
}

func (a *AmazonAdapter) FetchSettlements(_ context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	return []*PlatformSettlement{}, nil
}

func (a *AmazonAdapter) FetchReturns(_ context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	return []*PlatformReturn{}, nil
}
