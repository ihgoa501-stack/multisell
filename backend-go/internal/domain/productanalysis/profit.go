package productanalysis

import (
	"math"
	"time"

	"gorm.io/gorm"
)

// CalculateProfitMargin computes estimated profit margin and a normalized
// profit score (0–100) from target sale price and total estimated cost.
//
// Returns:
//   - marginPct: estimated profit margin as a percentage (nil if price <= 0)
//   - score: normalized profit score 0–100 (nil if price <= 0)
//
// Edge cases handled:
//   - price <= 0  → both returns are nil (invalid input)
//   - cost >= price → margin is 0 or negative, score is 0
//   - cost == 0   → margin is 100%, score is 100
func CalculateProfitMargin(price, cost float64) (marginPct *float64, score *float64) {
	if price <= 0 {
		return nil, nil
	}

	// Clamp cost at 0
	if cost < 0 {
		cost = 0
	}

	var m float64
	if cost >= price {
		m = 0
	} else {
		m = (price - cost) / price * 100
	}

	// Round to 2 decimal places
	m = math.Round(m*100) / 100
	marginPct = &m

	// Normalize to 0–100 score: margin capped at 50% → score 100,
	// 0% margin → score 0, linear in between
	raw := m / 50.0 * 100
	if raw > 100 {
		raw = 100
	}
	if raw < 0 {
		raw = 0
	}
	s := math.Round(raw)
	score = &s

	return marginPct, score
}

// CalculateDemandScore computes a demand score (0–100) for a product
// based on its sales volume relative to the category average over the
// last 30 days.
//
// Parameters:
//   - db: database connection
//   - productID: the product table ID (not sourcing product ID)
//
// Returns:
//   - score: 0–100 (nil if insufficient data)
//   - status: "calculated" or "no_data"
//
// Formula:
//   score = min(product_sales / category_avg_sales * 50, 100)
//
// A product selling at the category average gets score 50;
// double the average gets capped at 100.
func CalculateDemandScore(db *gorm.DB, productID int64) (*float64, string) {
	// Get the product's category
	var catID int64
	if err := db.Table("product").
		Select("category_id").
		Where("id = ?", productID).
		Take(&catID).Error; err != nil {
		return nil, "no_data"
	}

	cutoff := time.Now().Add(-30 * 24 * time.Hour)

	// This product's total quantity sold in the last 30 days
	var productQty float64
	db.Table("sales_order_item").
		Select("COALESCE(SUM(quantity), 0)").
		Where("product_id = ?", productID).
		Where("created_at >= ?", cutoff).
		Scan(&productQty)

	if productQty == 0 {
		return nil, "no_data"
	}

	// Total quantity sold for all products in the same category (last 30 days)
	var catQty float64
	db.Table("sales_order_item").
		Select("COALESCE(SUM(si.quantity), 0)").
		Joins("JOIN product p ON p.id = si.product_id").
		Where("p.category_id = ?", catID).
		Where("si.created_at >= ?", cutoff).
		Scan(&catQty)

	// Count distinct products that have sales in this category
	var productCount int64
	db.Table("sales_order_item").
		Select("COUNT(DISTINCT si.product_id)").
		Joins("JOIN product p ON p.id = si.product_id").
		Where("p.category_id = ?", catID).
		Where("si.created_at >= ?", cutoff).
		Scan(&productCount)

	if productCount == 0 {
		return nil, "no_data"
	}

	avgSales := catQty / float64(productCount)
	if avgSales == 0 {
		return nil, "no_data"
	}

	// Score: product sales relative to category average
	score := productQty / avgSales * 50
	if score > 100 {
		score = 100
	}
	score = math.Round(score)

	return &score, "calculated"
}

// CalculateCompetitionScore computes a competition score (0–100) for a
// product category. Higher score = more competition = harder to enter.
//
// Parameters:
//   - db: database connection
//   - categoryID: the category table ID
//
// Returns:
//   - score: 0–100 (nil if insufficient data)
//   - status: "calculated" or "no_data"
//
// Formula:
//   score = min(listing_density * 10, 100)
//
// Where listing_density = active_listings / unique_sellers in the category.
// Density of 1 listing/seller → score 10 (easy).
// Density of 10 listings/seller → score 100 (very competitive).
func CalculateCompetitionScore(db *gorm.DB, categoryID int64) (*float64, string) {
	// Count active listings in the same category
	var listingCount int64
	db.Table("product_listing").
		Select("COUNT(DISTINCT pl.id)").
		Joins("JOIN product p ON p.id = pl.product_id").
		Where("pl.status = 'active'").
		Where("p.category_id = ?", categoryID).
		Scan(&listingCount)

	if listingCount == 0 {
		return nil, "no_data"
	}

	// Count unique stores (sellers) with active listings in this category
	var sellerCount int64
	db.Table("product_listing").
		Select("COUNT(DISTINCT s.id)").
		Joins("JOIN product p ON p.id = pl.product_id").
		Joins("JOIN stores s ON s.platform_id = pl.platform_id").
		Where("pl.status = 'active'").
		Where("p.category_id = ?", categoryID).
		Scan(&sellerCount)

	if sellerCount == 0 {
		// Avoid division by zero — treat as a single seller
		sellerCount = 1
	}

	// Listing density = total active listings / unique sellers
	density := float64(listingCount) / float64(sellerCount)

	// Normalize: density 1 → score 10, density 10+ → score 100
	score := math.Min(density*10, 100)
	score = math.Round(score)

	return &score, "calculated"
}
