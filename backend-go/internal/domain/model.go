package sku

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Product maps to the PostgreSQL "product" table.
type Product struct {
	ID               int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name             string          `gorm:"column:name;not null" json:"name"`
	Subtitle         string          `gorm:"column:subtitle" json:"subtitle"`
	Description      string          `gorm:"column:description" json:"description"`
	BrandID          int64           `gorm:"column:brand_id;default:0" json:"brand_id"`
	CategoryID       int64           `gorm:"column:category_id" json:"category_id"`
	Unit             string          `gorm:"column:unit;default:件" json:"unit"`
	Status           int16           `gorm:"column:status;default:0" json:"status"`
	MainImage        string          `gorm:"column:main_image" json:"main_image"`
	Images           json.RawMessage `gorm:"column:images;type:jsonb" json:"images"`
	ProductLengthCm  decimal.Decimal `gorm:"column:product_length_cm;type:numeric(10,2)" json:"product_length_cm"`
	ProductWidthCm   decimal.Decimal `gorm:"column:product_width_cm;type:numeric(10,2)" json:"product_width_cm"`
	ProductHeightCm  decimal.Decimal `gorm:"column:product_height_cm;type:numeric(10,2)" json:"product_height_cm"`
	ProductWeightKg  decimal.Decimal `gorm:"column:product_weight_kg;type:numeric(10,2)" json:"product_weight_kg"`
	PackageLengthCm  decimal.Decimal `gorm:"column:package_length_cm;type:numeric(10,2)" json:"package_length_cm"`
	PackageWidthCm   decimal.Decimal `gorm:"column:package_width_cm;type:numeric(10,2)" json:"package_width_cm"`
	PackageHeightCm  decimal.Decimal `gorm:"column:package_height_cm;type:numeric(10,2)" json:"package_height_cm"`
	PackageWeightKg  decimal.Decimal `gorm:"column:package_weight_kg;type:numeric(10,2)" json:"package_weight_kg"`
	CargoType        string          `gorm:"column:cargo_type;default:normal" json:"cargo_type"`
	AiTitle          string          `gorm:"column:ai_title" json:"ai_title"`
	AiDescription    string          `gorm:"column:ai_description" json:"ai_description"`
	SeoKeywords      json.RawMessage `gorm:"column:seo_keywords;type:jsonb" json:"seo_keywords"`
	AiStatus         string          `gorm:"column:ai_status;default:pending" json:"ai_status"`
	PlatformStatuses json.RawMessage `gorm:"column:platform_statuses;type:jsonb" json:"platform_statuses"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Product) TableName() string { return "product" }

// SpecName maps to the PostgreSQL "spec_name" table.
type SpecName struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID int64     `gorm:"column:product_id;not null" json:"product_id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	SortOrder int       `gorm:"column:sort_order;default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (SpecName) TableName() string { return "spec_name" }

// SpecValue maps to the PostgreSQL "spec_value" table.
type SpecValue struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SpecNameID int64     `gorm:"column:spec_name_id;not null" json:"spec_name_id"`
	ProductID  int64     `gorm:"column:product_id;not null" json:"product_id"`
	Value      string    `gorm:"column:value;not null" json:"value"`
	SortOrder  int       `gorm:"column:sort_order;default:0" json:"sort_order"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (SpecValue) TableName() string { return "spec_value" }

// Sku maps to the PostgreSQL "sku" table.
type Sku struct {
	ID          int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID   int64           `gorm:"column:product_id;not null" json:"product_id"`
	Code        string          `gorm:"column:code" json:"code"`
	Barcode     string          `gorm:"column:barcode" json:"barcode"`
	SpecDesc    string          `gorm:"column:spec_desc" json:"spec_desc"`
	SpecValues  json.RawMessage `gorm:"column:spec_values;type:jsonb" json:"spec_values"`
	Price       decimal.Decimal `gorm:"column:price;type:numeric(10,2);default:0" json:"price"`
	CostPrice   decimal.Decimal `gorm:"column:cost_price;type:numeric(10,2);default:0" json:"cost_price"`
	MarketPrice decimal.Decimal `gorm:"column:market_price;type:numeric(10,2);default:0" json:"market_price"`
	Stock       int             `gorm:"column:stock;default:0" json:"stock"`
	LockStock   int             `gorm:"column:lock_stock;default:0" json:"lock_stock"`
	WarningStock int            `gorm:"column:warning_stock;default:0" json:"warning_stock"`
	Weight      decimal.Decimal `gorm:"column:weight;type:numeric(10,2);default:0" json:"weight"`
	SkuLengthCm decimal.Decimal `gorm:"column:sku_length_cm;type:numeric(10,2)" json:"sku_length_cm"`
	SkuWidthCm  decimal.Decimal `gorm:"column:sku_width_cm;type:numeric(10,2)" json:"sku_width_cm"`
	SkuHeightCm decimal.Decimal `gorm:"column:sku_height_cm;type:numeric(10,2)" json:"sku_height_cm"`
	SkuWeightKg decimal.Decimal `gorm:"column:sku_weight_kg;type:numeric(10,2)" json:"sku_weight_kg"`
	Image       string          `gorm:"column:image" json:"image"`
	Status      int16           `gorm:"column:status;default:1" json:"status"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Sku) TableName() string { return "sku" }
