package integrations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestMockStorefront_Seeding(t *testing.T) {
	db := dbtest.NewDB(t, &MockListing{})
	logger := dbtest.NewLogger(t)
	svc := NewMockService(db, logger)

	// Seed first time
	err := svc.SeedStorefront(db)
	require.NoError(t, err)

	// Verify mock listing was seeded
	var listing MockListing
	err = db.Where("platform = ? AND sku = ?", "ozon", "SKU-TEST-123").First(&listing).Error
	require.NoError(t, err)
	assert.Equal(t, "suggested", listing.Status)
	assert.Equal(t, int64(1999), listing.PriceCents)
	assert.Equal(t, 100, listing.Stock)

	// Seed again (idempotent / lock check)
	err = svc.SeedStorefront(db)
	require.NoError(t, err)
}

func TestMockStorefront_Publish(t *testing.T) {
	db := dbtest.NewDB(t, &MockListing{})
	logger := dbtest.NewLogger(t)
	svc := NewMockService(db, logger)

	// Seed first to get the 'suggested' listing
	err := svc.SeedStorefront(db)
	require.NoError(t, err)

	// Create adapter
	adapter := NewMockOzonAdapter(db, logger)

	ctx := context.Background()
	input := &PublishInput{
		ProductID:   456,
		PlatformID:  1,
		AccountID:   2,
		ProductName: "Test Product",
		SKUs: []PublishSKU{
			{SkuID: 1, SkuCode: "SKU-TEST-123"},
		},
		Prices: map[int64]string{
			1: "29.99",
		},
		Inventories: map[int64]int{
			1: 150,
		},
	}

	// Publish product
	res, err := adapter.Publish(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "live", res.SyncMessage)
	assert.Equal(t, "SKU-TEST-123", res.PlatformSKU)

	// Verify MockListing status changed from 'suggested' to 'live' and stock updated
	var listing MockListing
	err = db.Where("platform = ? AND sku = ?", "ozon", "SKU-TEST-123").First(&listing).Error
	require.NoError(t, err)
	assert.Equal(t, "live", listing.Status)
	assert.Equal(t, int64(2999), listing.PriceCents)
	assert.Equal(t, 150, listing.Stock)
	assert.Equal(t, int64(456), listing.ProductID)
}

func TestMockStorefront_SyncInventory(t *testing.T) {
	db := dbtest.NewDB(t, &MockListing{})
	logger := dbtest.NewLogger(t)

	// Create adapter
	adapter := NewMockShopeeAdapter(db, logger)

	ctx := context.Background()
	input := &SyncInventoryInput{
		PlatformID:  1,
		SkuCode:     "SKU-SH-888",
		PlatformSKU: "SKU-SH-888",
		Quantity:    88,
	}

	// Sync inventory (non-existent SKU should be created with status live)
	ok, err := adapter.SyncInventory(ctx, input)
	require.NoError(t, err)
	assert.True(t, ok)

	var listing MockListing
	err = db.Where("platform = ? AND sku = ?", "shopee", "SKU-SH-888").First(&listing).Error
	require.NoError(t, err)
	assert.Equal(t, "live", listing.Status)
	assert.Equal(t, 88, listing.Stock)

	// Sync again to update inventory
	input.Quantity = 99
	ok, err = adapter.SyncInventory(ctx, input)
	require.NoError(t, err)
	assert.True(t, ok)

	err = db.Where("platform = ? AND sku = ?", "shopee", "SKU-SH-888").First(&listing).Error
	require.NoError(t, err)
	assert.Equal(t, 99, listing.Stock)
}

func TestMockStorefront_FetchOrders(t *testing.T) {
	db := dbtest.NewDB(t, &MockListing{})
	logger := dbtest.NewLogger(t)

	// Seed and publish to create a live listing
	svc := NewMockService(db, logger)
	require.NoError(t, svc.SeedStorefront(db))

	adapter := NewMockOzonAdapter(db, logger)
	_, err := adapter.Publish(context.Background(), &PublishInput{
		ProductID: 456,
		SKUs:      []PublishSKU{{SkuID: 1, SkuCode: "SKU-TEST-123"}},
		Prices:    map[int64]string{1: "19.99"},
		Inventories: map[int64]int{1: 10},
	})
	require.NoError(t, err)

	// Fetch orders
	orders, err := adapter.FetchOrders(context.Background(), &FetchOrdersInput{})
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, "PAID", orders[0].Status)
	assert.Equal(t, "19.99", orders[0].TotalAmount)
	require.Len(t, orders[0].Items, 1)
	assert.Equal(t, "SKU-TEST-123", orders[0].Items[0].SkuCode)
}
