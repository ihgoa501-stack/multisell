package integrations

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MockListing represents a stateful product listing in the mock storefront.
type MockListing struct {
	ID         uint      `gorm:"primaryKey"`
	Platform   string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_platform_sku"` // "ozon" or "shopee"
	ProductID  int64     `gorm:"index"`                                                   // maps to local product ID
	SKU        string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_platform_sku"`
	PriceCents int64     `gorm:"not null"`
	Stock      int       `gorm:"not null;default:0"`
	Status     string    `gorm:"type:varchar(50);not null;default:'suggested'"` // e.g. "suggested", "live"
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TableName overrides the table name for GORM.
func (MockListing) TableName() string {
	return "mock_listings"
}

// MockService handles stateful mock storefront operations.
type MockService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewMockService creates a new MockService.
func NewMockService(db *gorm.DB, logger *zap.Logger) *MockService {
	return &MockService{
		db:     db,
		logger: logger,
	}
}

// SeedStorefront seeds the mock storefront with initial mock listings.
// It uses PG advisory locks to prevent race conditions during seeding.
func (s *MockService) SeedStorefront(db *gorm.DB) error {
	targetDB := db
	if targetDB == nil {
		targetDB = s.db
	}

	return targetDB.Transaction(func(tx *gorm.DB) error {
		dialect := tx.Dialector.Name()
		if dialect == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(777888)").Error; err != nil {
				return err
			}
		}

		if err := tx.AutoMigrate(&MockListing{}); err != nil {
			return err
		}

		// Seeding records
		var listing MockListing
		return tx.Where("platform = ? AND sku = ?", "ozon", "SKU-TEST-123").FirstOrCreate(&listing, MockListing{
			Platform:   "ozon",
			SKU:        "SKU-TEST-123",
			PriceCents: 1999,
			Stock:      100,
			Status:     "suggested",
		}).Error
	})
}

// MockPlatformAdapter implements PlatformAdapter using the stateful mock database.
type MockPlatformAdapter struct {
	platform string
	db       *gorm.DB
	logger   *zap.Logger
}

// NewMockPlatformAdapter creates a new MockPlatformAdapter.
func NewMockPlatformAdapter(platform string, db *gorm.DB, logger *zap.Logger) *MockPlatformAdapter {
	return &MockPlatformAdapter{
		platform: platform,
		db:       db,
		logger:   logger,
	}
}

// Publish implements PlatformAdapter.
func (a *MockPlatformAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	for _, sku := range input.SKUs {
		priceStr := input.Prices[sku.SkuID]
		priceVal, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			priceVal = 0.0
		}
		priceCents := int64(math.Round(priceVal * 100))
		stock := input.Inventories[sku.SkuID]

		err = a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&MockListing{}); err != nil {
				return err
			}

			var listing MockListing
			err := tx.Where("sku = ? AND platform = ?", sku.SkuCode, a.platform).First(&listing).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					newListing := MockListing{
						Platform:   a.platform,
						ProductID:  input.ProductID,
						SKU:        sku.SkuCode,
						PriceCents: priceCents,
						Stock:      stock,
						Status:     "live",
					}
					return tx.Create(&newListing).Error
				}
				return err
			}

			listing.Status = "live"
			listing.Stock = stock
			listing.PriceCents = priceCents
			listing.ProductID = input.ProductID
			return tx.Save(&listing).Error
		})
		if err != nil {
			return nil, fmt.Errorf("mock publish update error: %w", err)
		}
	}

	firstSKUCode := ""
	if len(input.SKUs) > 0 {
		firstSKUCode = input.SKUs[0].SkuCode
	}

	simulatedShippingRate := "5.00"

	return &PublishResult{
		PlatformProductID: fmt.Sprintf("mock-%s-prod-%d", a.platform, input.ProductID),
		PlatformSKU:       firstSKUCode,
		PlatformURL:       fmt.Sprintf("https://mock-%s.com/products/%s", a.platform, firstSKUCode),
		SyncMessage:       "live",
		PublishedData: map[string]interface{}{
			"status":        "live",
			"shipping_rate": simulatedShippingRate,
			"explanation":   "Successfully published to stateful mock platform",
		},
	}, nil
}

// SyncStatus implements PlatformAdapter.
func (a *MockPlatformAdapter) SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error) {
	var listing MockListing
	prodID := input.ListingID
	if prodID == 0 && input.PlatformProductID != "" {
		parts := strings.Split(input.PlatformProductID, "-prod-")
		if len(parts) == 2 {
			if parsed, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				prodID = parsed
			}
		}
	}

	err := a.db.WithContext(ctx).
		Where("platform = ? AND (product_id = ? OR sku = ?)", a.platform, prodID, input.PlatformProductID).
		First(&listing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "suggested", nil
		}
		return "", err
	}
	return listing.Status, nil
}

// ValidateCredentials implements PlatformAdapter.
func (a *MockPlatformAdapter) ValidateCredentials(ctx context.Context, accountID int64) (bool, error) {
	return true, nil
}

// SyncInventory implements PlatformAdapter.
func (a *MockPlatformAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	var listing MockListing
	err := a.db.WithContext(ctx).
		Where("platform = ? AND sku = ?", a.platform, input.SkuCode).
		First(&listing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newListing := MockListing{
				Platform: a.platform,
				SKU:      input.SkuCode,
				Stock:    input.Quantity,
				Status:   "live",
			}
			if err := a.db.WithContext(ctx).Create(&newListing).Error; err != nil {
				return false, err
			}
			return true, nil
		}
		return false, err
	}

	listing.Stock = input.Quantity
	if err := a.db.WithContext(ctx).Save(&listing).Error; err != nil {
		return false, err
	}
	return true, nil
}

// PushTracking implements PlatformAdapter.
func (a *MockPlatformAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	return true, nil
}

// FetchOrders implements PlatformAdapter.
func (a *MockPlatformAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	var listings []MockListing
	if err := a.db.WithContext(ctx).
		Where("platform = ? AND status = ?", a.platform, "live").
		Find(&listings).Error; err != nil {
		return nil, err
	}

	if len(listings) == 0 {
		return []*PlatformOrder{}, nil
	}

	var items []PlatformOrderItem
	var totalCents int64
	for _, l := range listings {
		items = append(items, PlatformOrderItem{
			SkuCode:   l.SKU,
			Quantity:  1,
			UnitPrice: fmt.Sprintf("%.2f", float64(l.PriceCents)/100.0),
		})
		totalCents += l.PriceCents
	}

	order := &PlatformOrder{
		OrderSN:         fmt.Sprintf("MOCK-%s-ORD-%d", strings.ToUpper(a.platform), time.Now().Unix()),
		Status:          "PAID",
		TotalAmount:     fmt.Sprintf("%.2f", float64(totalCents)/100.0),
		ShippingFee:     "5.00",
		PaidAt:          time.Now().Format(time.RFC3339),
		RecipientName:   "Mock Customer",
		RecipientPhone:  "+15550100",
		ShippingAddress: "123 Mock Lane, Mock City",
		Items:           items,
	}

	return []*PlatformOrder{order}, nil
}

// FetchSettlements implements PlatformAdapter.
func (a *MockPlatformAdapter) FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	var listings []MockListing
	if err := a.db.WithContext(ctx).
		Where("platform = ? AND status = ?", a.platform, "live").
		Find(&listings).Error; err != nil {
		return nil, err
	}

	if len(listings) == 0 {
		return []*PlatformSettlement{}, nil
	}

	var settlements []*PlatformSettlement
	for idx, l := range listings {
		settlements = append(settlements, &PlatformSettlement{
			TransactionID:   fmt.Sprintf("MOCK-%s-TX-%d-%d", strings.ToUpper(a.platform), time.Now().Unix(), idx),
			TransactionType: "order_sale",
			OrderSN:         fmt.Sprintf("MOCK-%s-ORD-%d", strings.ToUpper(a.platform), time.Now().Unix()),
			Amount:          fmt.Sprintf("%.2f", float64(l.PriceCents)/100.0),
			Fee:             fmt.Sprintf("%.2f", float64(l.PriceCents)*0.15/100.0),
			Currency:        "USD",
			OccurredAt:      time.Now().Format(time.RFC3339),
			Description:     fmt.Sprintf("Mock settlement for SKU %s", l.SKU),
		})
	}
	return settlements, nil
}

// FetchReturns implements PlatformAdapter.
func (a *MockPlatformAdapter) FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	var listings []MockListing
	if err := a.db.WithContext(ctx).
		Where("platform = ? AND status = ?", a.platform, "live").
		Find(&listings).Error; err != nil {
		return nil, err
	}

	if len(listings) == 0 {
		return []*PlatformReturn{}, nil
	}

	l := listings[0]
	ret := &PlatformReturn{
		ReturnID:     fmt.Sprintf("MOCK-%s-RET-%d", strings.ToUpper(a.platform), time.Now().Unix()),
		OrderSN:      fmt.Sprintf("MOCK-%s-ORD-%d", strings.ToUpper(a.platform), time.Now().Unix()),
		SkuCode:      l.SKU,
		Quantity:     1,
		Reason:       "Mock customer changed mind",
		Status:       "APPROVED",
		CreatedAt:    time.Now().Format(time.RFC3339),
		RefundAmount: fmt.Sprintf("%.2f", float64(l.PriceCents)/100.0),
	}
	return []*PlatformReturn{ret}, nil
}

// MockOzonAdapter is a MockPlatformAdapter specifically for Ozon.
type MockOzonAdapter struct {
	*MockPlatformAdapter
}

// NewMockOzonAdapter creates a new MockOzonAdapter.
func NewMockOzonAdapter(db *gorm.DB, logger *zap.Logger) *MockOzonAdapter {
	return &MockOzonAdapter{NewMockPlatformAdapter("ozon", db, logger)}
}

// MockShopeeAdapter is a MockPlatformAdapter specifically for Shopee.
type MockShopeeAdapter struct {
	*MockPlatformAdapter
}

// NewMockShopeeAdapter creates a new MockShopeeAdapter.
func NewMockShopeeAdapter(db *gorm.DB, logger *zap.Logger) *MockShopeeAdapter {
	return &MockShopeeAdapter{NewMockPlatformAdapter("shopee", db, logger)}
}
