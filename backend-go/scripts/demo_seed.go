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
	"github.com/lingmirror/backend-go/internal/domain/brand"
	"github.com/lingmirror/backend-go/internal/domain/category"
	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	fmt.Println("Demo seed complete. Data ready for A5 stock_alert demo.")
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
		Name:           "Demo Product — Low Stock Alert Target",
		Subtitle:       "用于演示 A5 库存预警闭环",
		BrandID:        b.ID,
		CategoryID:     cat.ID,
		Unit:           "件",
		Status:         1,
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
	s := sku.Sku{
		ProductID:   product.ID,
		Code:        skuCode,
		Barcode:     "6900000000001",
		Stock:       5,
		LockStock:   0,
		WarningStock: 20,
		Weight:      decimal.NewFromFloat(1.5),
		Price:       decimal.NewFromFloat(88.00),
		CostPrice:   decimal.NewFromFloat(45.00),
		Status:      1,
	}

	// Upsert on code so re-running is safe.
	result = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		UpdateAll: true,
	}).Create(&s)
	if result.Error != nil {
		return fmt.Errorf("sku: %w", result.Error)
	}
	fmt.Printf("  sku: id=%d code=%s stock=%d\n", s.ID, s.Code, s.Stock)

	// --- Inventory ---
	inv := inventory.Inventory{
		SkuID:     s.ID,
		Warehouse: "默认仓库",
		Quantity:  5,
		LockedQuantity: 0,
		SafetyStock: 14,
		CreatedAt: now,
		UpdatedAt: now,
	}
	result = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "sku_id"}},
		UpdateAll: true,
	}).Create(&inv)
	if result.Error != nil {
		return fmt.Errorf("inventory: %w", result.Error)
	}
	fmt.Printf("  inventory: sku_id=%d quantity=%d safety_stock=%d\n", inv.SkuID, inv.Quantity, inv.SafetyStock)

	return nil
}
