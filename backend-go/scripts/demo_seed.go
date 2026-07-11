// Command demo_seed populates the database with deterministic demo data for
// the AgentOS closed-loop demo (A5 stock_alert pipeline).
//
// Usage:
//
//	DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=multisell \
//	  go run scripts/demo_seed.go
//
// Idempotent: safe to re-run. Uses upsert/natural-key patterns.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/database"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/brand"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/category"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/loop"
	"github.com/lingmirror/backend-go/internal/domain/platformfee"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("create logger: %v", err)
	}

	db, err := database.Connect(cfg, logger)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	if err := seed(db); err != nil {
		log.Fatalf("seed: %v", err)
	}

	fmt.Println("Demo seed complete. Data ready for A5 stock_alert demo & Product Loop E2E.")
	os.Exit(0)
}

func seed(db *gorm.DB) error {
	now := time.Now().UTC()

	// --- Brand ---
	b := brand.Brand{
		Name:   "DemoBrand",
		Status: 1,
	}
	result := db.Where("name = ?", b.Name).FirstOrCreate(&b)
	if result.Error != nil {
		return fmt.Errorf("brand: %w", result.Error)
	}
	fmt.Printf("  brand: id=%d name=%s\n", b.ID, b.Name)

	// --- Category ---
	cat := category.Category{
		Name:   "DemoCategory",
		Status: 1,
	}
	result = db.Where("name = ?", cat.Name).FirstOrCreate(&cat)
	if result.Error != nil {
		return fmt.Errorf("category: %w", result.Error)
	}
	fmt.Printf("  category: id=%d name=%s\n", cat.ID, cat.Name)

	// --- Product ---
	product := sku.Product{
		Name:            "Demo Product — Low Stock Alert Target",
		Subtitle:        "用于演示 A5 库存预警闭环",
		BrandID:         b.ID,
		CategoryID:      cat.ID,
		Unit:            "件",
		Status:          1,
		ProductWeightKg: decimal.NewFromFloat(1.5),
		PackageWeightKg: decimal.NewFromFloat(2.0),
	}
	result = db.Where("name = ?", product.Name).FirstOrCreate(&product)
	if result.Error != nil {
		return fmt.Errorf("product: %w", result.Error)
	}
	fmt.Printf("  product: id=%d name=%s\n", product.ID, product.Name)

	// --- SKU ---
	skuCode := "DEMO-SKU-A5-001"
	var s sku.Sku
	if err := db.Where("code = ?", skuCode).First(&s).Error; err != nil {
		s = sku.Sku{
			ProductID:    product.ID,
			Code:         skuCode,
			Barcode:      "6900000000001",
			Stock:        5,
			LockStock:    0,
			WarningStock: 20,
			Weight:       decimal.NewFromFloat(1.5),
			Price:        decimal.NewFromFloat(88.00),
			CostPrice:    decimal.NewFromFloat(45.00),
			Status:       1,
		}
		if err := db.Create(&s).Error; err != nil {
			return fmt.Errorf("sku: %w", err)
		}
	} else {
		s.Stock = 5
		s.Price = decimal.NewFromFloat(88.00)
		s.CostPrice = decimal.NewFromFloat(45.00)
		db.Save(&s)
	}
	fmt.Printf("  sku: id=%d code=%s stock=%d\n", s.ID, s.Code, s.Stock)

	// --- Inventory ---
	var inv inventory.Inventory
	if err := db.Where("sku_id = ?", s.ID).First(&inv).Error; err != nil {
		inv = inventory.Inventory{
			SkuID:          s.ID,
			Warehouse:      "默认仓库",
			Quantity:       5,
			LockedQuantity: 0,
			SafetyStock:    14,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := db.Create(&inv).Error; err != nil {
			return fmt.Errorf("inventory: %w", err)
		}
	} else {
		inv.Quantity = 5
		inv.SafetyStock = 14
		db.Save(&inv)
	}
	fmt.Printf("  inventory: sku_id=%d quantity=%d safety_stock=%d\n", inv.SkuID, inv.Quantity, inv.SafetyStock)

	// --- Product Loop E2E seed data ---
	if err := seedProductLoopData(db); err != nil {
		return fmt.Errorf("product loop seed: %w", err)
	}

	return nil
}

// seedProductLoopData inserts 5 deterministic scenarios for the Product Loop E2E.
// Idempotent: uses title-based FirstOrCreate for candidate products.
func seedProductLoopData(db *gorm.DB) error {

	// ========================================================================
	// Scenario 1: Profitable Listing ("Premium Wireless Earbuds")
	//   → complete data, positive margin, "list" recommendation
	// ========================================================================
	fmt.Println("  [product-loop-scenario-1] Profitable Listing")
	{
		title := "Premium Wireless Earbuds"
		prod := candidate.CandidateProduct{
			Title:              title,
			Description:        "High-quality wireless Bluetooth earbuds with noise cancellation",
			MainImage:          "https://picsum.photos/seed/earbuds1/400",
			SupplierID:         int64Ptr(1),
			PurchasePrice:      45.00,
			PurchaseCurrency:   "CNY",
			PackageWeightKg:    0.15,
			PackageLengthCm:    8.0,
			PackageWidthCm:     4.0,
			PackageHeightCm:    3.0,
			HSCode:             "8518.30",
			OriginCountry:      "CN",
			TargetSalePrice:    29.99,
			TargetCurrency:     "USD",
			DestinationCountry: "US",
			IsSeedData:         true,
			CompletenessStatus: "complete",
			CreatedBy:          "seed",
		}
		if err := db.Where("title = ?", title).FirstOrCreate(&prod).Error; err != nil {
			return fmt.Errorf("scenario1 candidate: %w", err)
		}
		fmt.Printf("    candidate_product: id=%d title=%s\n", prod.ID, prod.Title)

		_ = db.Where("product_id = ?", prod.ID).Delete(&completeness.CompletenessCheck{})
		comp := completeness.CompletenessCheck{
			ProductID:    prod.ID,
			Score:        85,
			MissingItems: "[]",
			Status:       "complete",
			TriggeredBy:  "seed",
		}
		if err := db.Create(&comp).Error; err != nil {
			return fmt.Errorf("scenario1 complete check: %w", err)
		}
		fmt.Printf("    completeness_check: id=%d score=%.0f\n", comp.ID, comp.Score)

		_ = db.Where("product_id = ?", prod.ID).Delete(&profit.ProfitSummary{})
		ps := profit.ProfitSummary{
			ProductID:       prod.ID,
			PurchaseCost:    6.25,
			ShippingCost:    8.50,
			PlatformFee:     4.50,
			TotalCost:       19.25,
			TargetRevenue:   29.99,
			EstimatedProfit: 10.74,
			ProfitMargin:    35.80,
			Status:          "profitable",
			Currency:        "USD",
			CalculatedBy:    "seed",
		}
		if err := db.Create(&ps).Error; err != nil {
			return fmt.Errorf("scenario1 profit: %w", err)
		}
		fmt.Printf("    profit_summary: id=%d margin=%.2f%%\n", ps.ID, ps.ProfitMargin)

		_ = db.Where("product_id = ? AND triggered_by = 'seed'", prod.ID).Delete(&loop.ListingRecommendation{})
		rec := loop.ListingRecommendation{
			ProductID:         prod.ID,
			CompletenessScore: 85,
			ProfitMargin:      35.80,
			EstimatedProfit:   10.74,
			Decision:          "list",
			Confidence:        0.95,
			Reason:            "资料完整（85%），利润良好（35.80%），建议上架。",
			RiskFlags:         "[]",
			TriggeredBy:       "seed",
			FeedbackStatus:    "pending",
		}
		if err := db.Create(&rec).Error; err != nil {
			return fmt.Errorf("scenario1 recommendation: %w", err)
		}
		fmt.Printf("    listing_recommendation: id=%d decision=%s\n", rec.ID, rec.Decision)
	}

	// ========================================================================
	// Scenario 2: Loss-Making Listing ("Unknown Brand Charger")
	//   → negative margin, "skip" recommendation
	// ========================================================================
	fmt.Println("  [product-loop-scenario-2] Loss-Making Listing")
	{
		title := "Unknown Brand Charger"
		prod := candidate.CandidateProduct{
			Title:              title,
			Description:        "Generic phone charger, unknown brand quality",
			MainImage:          "",
			SupplierID:         nil,
			PurchasePrice:      85.00,
			PurchaseCurrency:   "CNY",
			PackageWeightKg:    0.20,
			PackageLengthCm:    10.0,
			PackageWidthCm:     6.0,
			PackageHeightCm:    3.0,
			HSCode:             "8504.40",
			OriginCountry:      "CN",
			TargetSalePrice:    9.99,
			TargetCurrency:     "USD",
			DestinationCountry: "US",
			IsSeedData:         true,
			CompletenessStatus: "needs_review",
			CreatedBy:          "seed",
		}
		if err := db.Where("title = ?", title).FirstOrCreate(&prod).Error; err != nil {
			return fmt.Errorf("scenario2 candidate: %w", err)
		}
		fmt.Printf("    candidate_product: id=%d title=%s\n", prod.ID, prod.Title)

		_ = db.Where("product_id = ?", prod.ID).Delete(&completeness.CompletenessCheck{})
		comp := completeness.CompletenessCheck{
			ProductID:    prod.ID,
			Score:        75,
			MissingItems: `["main_image","supplier_id"]`,
			Status:       "incomplete",
			TriggeredBy:  "seed",
		}
		if err := db.Create(&comp).Error; err != nil {
			return fmt.Errorf("scenario2 complete check: %w", err)
		}

		_ = db.Where("product_id = ?", prod.ID).Delete(&profit.ProfitSummary{})
		ps := profit.ProfitSummary{
			ProductID:       prod.ID,
			PurchaseCost:    11.81,
			ShippingCost:    3.50,
			PlatformFee:     1.50,
			TotalCost:       16.81,
			TargetRevenue:   9.99,
			EstimatedProfit: -6.82,
			ProfitMargin:    -68.27,
			Status:          "unprofitable",
			Currency:        "USD",
			CalculatedBy:    "seed",
		}
		if err := db.Create(&ps).Error; err != nil {
			return fmt.Errorf("scenario2 profit: %w", err)
		}

		_ = db.Where("product_id = ? AND triggered_by = 'seed'", prod.ID).Delete(&loop.ListingRecommendation{})
		rec := loop.ListingRecommendation{
			ProductID:         prod.ID,
			CompletenessScore: 75,
			ProfitMargin:      -68.27,
			EstimatedProfit:   -6.82,
			Decision:          "skip",
			Confidence:        0.30,
			Reason:            "利润为负（unprofitable: -68.27%），成本（$16.81）高于目标售价（$9.99）。",
			RiskFlags:         `["负利润"]`,
			TriggeredBy:       "seed",
			FeedbackStatus:    "pending",
		}
		if err := db.Create(&rec).Error; err != nil {
			return fmt.Errorf("scenario2 recommendation: %w", err)
		}
		fmt.Printf("    listing_recommendation: id=%d decision=%s\n", rec.ID, rec.Decision)
	}

	// ========================================================================
	// Scenario 3: Missing Logistics Fee ("Leather Phone Case")
	//   → no package_weight/shipping data, completeness < 50, "skip"
	// ========================================================================
	fmt.Println("  [product-loop-scenario-3] Missing Logistics Fee")
	{
		title := "Leather Phone Case"
		prod := candidate.CandidateProduct{
			Title:              title,
			Description:        "Premium leather phone case for iPhone 15",
			MainImage:          "https://picsum.photos/seed/case1/400",
			SupplierID:         int64Ptr(1),
			PurchasePrice:      20.00,
			PurchaseCurrency:   "CNY",
			PackageWeightKg:    0, // missing — no logistics cost data
			PackageLengthCm:    0, // missing
			PackageWidthCm:     0, // missing
			PackageHeightCm:    0, // missing
			HSCode:             "4202.32",
			OriginCountry:      "CN",
			TargetSalePrice:    15.99,
			TargetCurrency:     "USD",
			DestinationCountry: "US",
			IsSeedData:         true,
			CompletenessStatus: "incomplete",
			CreatedBy:          "seed",
		}
		if err := db.Where("title = ?", title).FirstOrCreate(&prod).Error; err != nil {
			return fmt.Errorf("scenario3 candidate: %w", err)
		}
		fmt.Printf("    candidate_product: id=%d title=%s\n", prod.ID, prod.Title)

		_ = db.Where("product_id = ?", prod.ID).Delete(&completeness.CompletenessCheck{})
		comp := completeness.CompletenessCheck{
			ProductID:    prod.ID,
			Score:        40,
			MissingItems: `["package_weight_kg","package_length_cm","package_width_cm","package_height_cm"]`,
			Status:       "incomplete",
			TriggeredBy:  "seed",
		}
		if err := db.Create(&comp).Error; err != nil {
			return fmt.Errorf("scenario3 complete check: %w", err)
		}

		_ = db.Where("product_id = ?", prod.ID).Delete(&profit.ProfitSummary{})
		ps := profit.ProfitSummary{
			ProductID:       prod.ID,
			PurchaseCost:    2.78,
			ShippingCost:    0, // missing — no logistics
			PlatformFee:     2.40,
			TotalCost:       5.18,
			TargetRevenue:   15.99,
			EstimatedProfit: 10.81,
			ProfitMargin:    67.60, // misleading — shipping not included
			Status:          "profitable",
			Currency:        "USD",
			CalculatedBy:    "seed",
		}
		if err := db.Create(&ps).Error; err != nil {
			return fmt.Errorf("scenario3 profit: %w", err)
		}

		_ = db.Where("product_id = ? AND triggered_by = 'seed'", prod.ID).Delete(&loop.ListingRecommendation{})
		rec := loop.ListingRecommendation{
			ProductID:         prod.ID,
			CompletenessScore: 40,
			ProfitMargin:      67.60,
			EstimatedProfit:   10.81,
			Decision:          "skip",
			Confidence:        0.25,
			Reason:            "资料完整度过低（评分 <50），不建议上架。请先补充：package_weight_kg、package_length_cm、package_width_cm、package_height_cm",
			RiskFlags:         `["资料严重缺失"]`,
			TriggeredBy:       "seed",
			FeedbackStatus:    "pending",
		}
		if err := db.Create(&rec).Error; err != nil {
			return fmt.Errorf("scenario3 recommendation: %w", err)
		}
		fmt.Printf("    listing_recommendation: id=%d decision=%s\n", rec.ID, rec.Decision)
	}

	// ========================================================================
	// Scenario 4: Missing Platform/Category Fee ("Portable Bluetooth Speaker")
	//   → moderate completeness, marginal profit, "cautious" recommendation
	// ========================================================================
	fmt.Println("  [product-loop-scenario-4] Missing Platform Fee")
	{
		title := "Portable Bluetooth Speaker"
		prod := candidate.CandidateProduct{
			Title:              title,
			Description:        "Portable waterproof Bluetooth speaker, 10W output",
			MainImage:          "https://picsum.photos/seed/speaker1/400",
			SupplierID:         nil, // missing
			PurchasePrice:      110.00,
			PurchaseCurrency:   "CNY",
			PackageWeightKg:    0.50,
			PackageLengthCm:    16.0,
			PackageWidthCm:     8.0,
			PackageHeightCm:    8.0,
			HSCode:             "", // missing
			OriginCountry:      "CN",
			TargetSalePrice:    22.00,
			TargetCurrency:     "USD",
			DestinationCountry: "US",
			IsSeedData:         true,
			CompletenessStatus: "needs_review",
			CreatedBy:          "seed",
		}
		if err := db.Where("title = ?", title).FirstOrCreate(&prod).Error; err != nil {
			return fmt.Errorf("scenario4 candidate: %w", err)
		}
		fmt.Printf("    candidate_product: id=%d title=%s\n", prod.ID, prod.Title)

		_ = db.Where("product_id = ?", prod.ID).Delete(&completeness.CompletenessCheck{})
		comp := completeness.CompletenessCheck{
			ProductID:    prod.ID,
			Score:        75,
			MissingItems: `["supplier_id","hs_code"]`,
			Status:       "incomplete",
			TriggeredBy:  "seed",
		}
		if err := db.Create(&comp).Error; err != nil {
			return fmt.Errorf("scenario4 complete check: %w", err)
		}

		_ = db.Where("product_id = ?", prod.ID).Delete(&profit.ProfitSummary{})
		ps := profit.ProfitSummary{
			ProductID:       prod.ID,
			PurchaseCost:    15.28,
			ShippingCost:    4.00,
			PlatformFee:     0, // missing — no fee rule matched for this category
			TotalCost:       19.28,
			TargetRevenue:   22.00,
			EstimatedProfit: 2.72,
			ProfitMargin:    12.36,
			Status:          "marginal",
			Currency:        "USD",
			CalculatedBy:    "seed",
		}
		if err := db.Create(&ps).Error; err != nil {
			return fmt.Errorf("scenario4 profit: %w", err)
		}

		_ = db.Where("product_id = ? AND triggered_by = 'seed'", prod.ID).Delete(&loop.ListingRecommendation{})
		rec := loop.ListingRecommendation{
			ProductID:         prod.ID,
			CompletenessScore: 75,
			ProfitMargin:      12.36,
			EstimatedProfit:   2.72,
			Decision:          "cautious",
			Confidence:        0.60,
			Reason:            "条件适中：完整度 75%，利润率 12.36%。建议在补充资料或确认成本后决定。",
			RiskFlags:         `["利润偏低"]`,
			TriggeredBy:       "seed",
			FeedbackStatus:    "pending",
		}
		if err := db.Create(&rec).Error; err != nil {
			return fmt.Errorf("scenario4 recommendation: %w", err)
		}
		fmt.Printf("    listing_recommendation: id=%d decision=%s\n", rec.ID, rec.Decision)
	}

	// ========================================================================
	// Scenario 5: Approval-to-Sandbox Pipeline ("Eco-Friendly Water Bottle")
	//   → full pipeline: candidate → completeness → profit → recommendation
	//     → listing_task (blocked) → approval_request (pending)
	// ========================================================================
	fmt.Println("  [product-loop-scenario-5] Approval Pipeline")
	{
		title := "Eco-Friendly Water Bottle"
		prod := candidate.CandidateProduct{
			Title:              title,
			Description:        "BPA-free reusable stainless steel water bottle, 500ml",
			MainImage:          "https://picsum.photos/seed/bottle1/400",
			SupplierID:         int64Ptr(1),
			PurchasePrice:      25.00,
			PurchaseCurrency:   "CNY",
			PackageWeightKg:    0.30,
			PackageLengthCm:    25.0,
			PackageWidthCm:     8.0,
			PackageHeightCm:    8.0,
			HSCode:             "9617.00",
			OriginCountry:      "CN",
			TargetSalePrice:    19.99,
			TargetCurrency:     "USD",
			TargetPlatformID:   int64Ptr(1),
			DestinationCountry: "US",
			IsSeedData:         true,
			CompletenessStatus: "complete",
			CreatedBy:          "seed",
		}
		if err := db.Where("title = ?", title).FirstOrCreate(&prod).Error; err != nil {
			return fmt.Errorf("scenario5 candidate: %w", err)
		}
		fmt.Printf("    candidate_product: id=%d title=%s\n", prod.ID, prod.Title)

		_ = db.Where("product_id = ?", prod.ID).Delete(&completeness.CompletenessCheck{})
		comp := completeness.CompletenessCheck{
			ProductID:    prod.ID,
			Score:        95,
			MissingItems: "[]",
			Status:       "complete",
			TriggeredBy:  "seed",
		}
		if err := db.Create(&comp).Error; err != nil {
			return fmt.Errorf("scenario5 complete check: %w", err)
		}

		_ = db.Where("product_id = ?", prod.ID).Delete(&profit.ProfitSummary{})
		ps := profit.ProfitSummary{
			ProductID:       prod.ID,
			PurchaseCost:    3.47,
			ShippingCost:    4.00,
			PlatformFee:     3.00,
			TotalCost:       10.47,
			TargetRevenue:   19.99,
			EstimatedProfit: 9.52,
			ProfitMargin:    47.62,
			Status:          "profitable",
			Currency:        "USD",
			CalculatedBy:    "seed",
		}
		if err := db.Create(&ps).Error; err != nil {
			return fmt.Errorf("scenario5 profit: %w", err)
		}

		// Platform fee rule for scenario 5 (commission for platform_id=1)
		_ = db.Where("platform_id = 1 AND fee_type = 'commission' AND remark = 'seed-e2e'").Delete(&platformfee.PlatformFeeRule{})
		feeRule := platformfee.PlatformFeeRule{
			PlatformID: int64Ptr(1),
			FeeType:    "commission",
			FeeRatePct: 15.00,
			Status:     "active",
			Remark:     "seed-e2e",
		}
		if err := db.Create(&feeRule).Error; err != nil {
			return fmt.Errorf("scenario5 fee rule: %w", err)
		}
		fmt.Printf("    platform_fee_rule: id=%d rate=%.2f%%\n", feeRule.ID, feeRule.FeeRatePct)

		// Create listing_task in blocked state (pre-approval)
		_ = db.Where("source_item_key = ?", fmt.Sprintf("candidate:%d", prod.ID)).Delete(&listingtask.ListingTask{})
		lTask := listingtask.ListingTask{
			ProductID:          prod.ID,
			PlatformID:         1,
			SourceType:         "decision",
			SourceItemKey:      fmt.Sprintf("candidate:%d", prod.ID),
			Status:             "blocked",
			DestinationCountry: "US",
			TargetSalePrice:    float64Ptr(19.99),
			TargetProfitMargin: float64Ptr(47.62),
			CreatedBy:          "seed",
		}
		if err := db.Create(&lTask).Error; err != nil {
			return fmt.Errorf("scenario5 listing task: %w", err)
		}
		fmt.Printf("    listing_task: id=%d status=%s\n", lTask.ID, lTask.Status)

		// Create approval_request linked to listing_task
		_ = db.Where("entity_type = 'listing_task' AND entity_id = ?", lTask.ID).Delete(&approval.ApprovalRequest{})
		appReq := approval.ApprovalRequest{
			ProductID:   prod.ID,
			RequestType: "listing_task",
			Requester:   "seed",
			Status:      "pending",
			TargetType:  "listing_task",
			TargetID:    lTask.ID,
			RiskLevel:   "high",
			EntityType:  "listing_task",
			EntityID:    lTask.ID,
		}
		if err := db.Create(&appReq).Error; err != nil {
			return fmt.Errorf("scenario5 approval: %w", err)
		}
		fmt.Printf("    approval_request: id=%d status=%s\n", appReq.ID, appReq.Status)

		// Create listing_recommendation linking to the listing_task
		_ = db.Where("product_id = ? AND triggered_by = 'seed'", prod.ID).Delete(&loop.ListingRecommendation{})
		rec := loop.ListingRecommendation{
			ProductID:            prod.ID,
			CompletenessScore:    95,
			ProfitMargin:         47.62,
			EstimatedProfit:      9.52,
			Decision:             "list",
			Confidence:           0.95,
			Reason:               "资料完整（95%），利润良好（47.62%），建议上架。",
			RiskFlags:            "[]",
			CreatedListingTaskID: &lTask.ID,
			TriggeredBy:          "seed",
			FeedbackStatus:       "pending",
		}
		if err := db.Create(&rec).Error; err != nil {
			return fmt.Errorf("scenario5 recommendation: %w", err)
		}
		fmt.Printf("    listing_recommendation: id=%d decision=%s listing_task_id=%d\n", rec.ID, rec.Decision, *rec.CreatedListingTaskID)
	}

	return nil
}

// Helpers for pointer values.
func int64Ptr(v int64) *int64       { return &v }
func float64Ptr(v float64) *float64 { return &v }
