package integrations

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LazadaAdapter implements PlatformAdapter for Lazada Open Platform.
type LazadaAdapter struct {
	httpClient *http.Client
	db         *gorm.DB
	logger     *zap.Logger
}

// NewLazadaAdapter creates a new Lazada adapter.
func NewLazadaAdapter(db *gorm.DB, logger *zap.Logger) *LazadaAdapter {
	return &LazadaAdapter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		db:         db,
		logger:     logger,
	}
}

func (a *LazadaAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("lazada publish: no SKUs")
	}
	return &PublishResult{
		PlatformProductID: fmt.Sprintf("lazada-pending-%d", input.ProductID),
		PlatformSKU:       input.SKUs[0].SkuCode,
		PlatformURL:       "",
		PublishedData:     map[string]interface{}{"status": "stub"},
		SyncMessage:       "lazada adapter stub: publish not yet implemented",
	}, nil
}

func (a *LazadaAdapter) SyncStatus(_ context.Context, input *SyncStatusInput) (string, error) {
	if input.PlatformProductID == "" {
		return "unknown", fmt.Errorf("lazada sync_status: empty platform product id")
	}
	return "synced", nil
}

func (a *LazadaAdapter) ValidateCredentials(_ context.Context, accountID int64) (bool, error) {
	return false, fmt.Errorf("lazada ValidateCredentials: not yet implemented for account %d", accountID)
}

func (a *LazadaAdapter) SyncInventory(_ context.Context, input *SyncInventoryInput) (bool, error) {
	return false, fmt.Errorf("lazada sync_inventory: not yet implemented for sku %s", input.SkuCode)
}

func (a *LazadaAdapter) PushTracking(_ context.Context, input *PushTrackingInput) (bool, error) {
	return false, fmt.Errorf("lazada push_tracking: not yet implemented for order %s", input.OrderSN)
}

func (a *LazadaAdapter) FetchOrders(_ context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	return []*PlatformOrder{}, nil
}

func (a *LazadaAdapter) FetchSettlements(_ context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	return []*PlatformSettlement{}, nil
}

func (a *LazadaAdapter) FetchReturns(_ context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	return []*PlatformReturn{}, nil
}
