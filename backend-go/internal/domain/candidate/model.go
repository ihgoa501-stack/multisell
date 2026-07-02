package candidate

import (
	"encoding/json"
	"time"
)

// CandidateProduct maps to the "candidate_product" table.
type CandidateProduct struct {
	ID                int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title             string          `gorm:"column:title;not null" json:"title"`
	Description       string          `gorm:"column:description" json:"description"`
	MainImage         string          `gorm:"column:main_image" json:"main_image"`
	Images            json.RawMessage `gorm:"column:images;type:jsonb" json:"images,omitempty"`
	CategoryID        *int64          `gorm:"column:category_id" json:"category_id,omitempty"`
	BrandID           *int64          `gorm:"column:brand_id" json:"brand_id,omitempty"`
	SpecJSON          json.RawMessage `gorm:"column:spec_json;type:jsonb" json:"spec_json,omitempty"`
	SupplierID        *int64          `gorm:"column:supplier_id" json:"supplier_id,omitempty"`
	PurchasePrice     float64         `gorm:"column:purchase_price;default:0" json:"purchase_price"`
	PurchaseCurrency  string          `gorm:"column:purchase_currency;default:CNY" json:"purchase_currency"`
	PackageWeightKg   float64         `gorm:"column:package_weight_kg;default:0" json:"package_weight_kg"`
	PackageLengthCm   float64         `gorm:"column:package_length_cm;default:0" json:"package_length_cm"`
	PackageWidthCm    float64         `gorm:"column:package_width_cm;default:0" json:"package_width_cm"`
	PackageHeightCm   float64         `gorm:"column:package_height_cm;default:0" json:"package_height_cm"`
	HSCode            string          `gorm:"column:hs_code" json:"hs_code"`
	OriginCountry     string          `gorm:"column:origin_country;default:CN" json:"origin_country"`
	TargetSalePrice   float64         `gorm:"column:target_sale_price;default:0" json:"target_sale_price"`
	TargetCurrency    string          `gorm:"column:target_currency;default:USD" json:"target_currency"`
	TargetPlatformID  *int64          `gorm:"column:target_platform_id" json:"target_platform_id,omitempty"`
	DestinationCountry string        `gorm:"column:destination_country;default:US" json:"destination_country"`
	SourceURL          string           `gorm:"column:source_url;varchar(2048)" json:"source_url"`
	SourcePlatform     string           `gorm:"column:source_platform;varchar(64);default:''" json:"source_platform"`
	RawPayload         *json.RawMessage `gorm:"column:raw_payload;type:jsonb" json:"raw_payload,omitempty"`
	CompletenessStatus string           `gorm:"column:completeness_status;varchar(32);default:incomplete" json:"completeness_status"`
	CollectedAt        *time.Time       `gorm:"column:collected_at" json:"collected_at,omitempty"`
	Status            string          `gorm:"column:status" json:"status"` // draft, in_review, approved, rejected
	IsSeedData        bool            `gorm:"column:is_seed_data;default:false" json:"is_seed_data"`
	CreatedBy         string          `gorm:"column:created_by" json:"created_by"`
	UpdatedBy         string          `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (CandidateProduct) TableName() string { return "candidate_product" }

// CollectLead maps to the "collect_leads" table.
type CollectLead struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title         string     `gorm:"column:title" json:"title"`
	PriceRange    string     `gorm:"column:price_range" json:"price_range"`
	DetailURL     string     `gorm:"column:detail_url" json:"detail_url"`
	ImageURL      string     `gorm:"column:image_url" json:"image_url"`
	ShopHint      string     `gorm:"column:shop_hint" json:"shop_hint"`
	SourcePageURL string     `gorm:"column:source_page_url" json:"source_page_url"`
	Status        string     `gorm:"column:status;default:pending_detail_collect" json:"status"`
	CollectedAt   *time.Time `gorm:"column:collected_at" json:"collected_at,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CollectLead) TableName() string { return "collect_leads" }

// CreateCandidateInput is the payload for creating a candidate product.
type CreateCandidateInput struct {
	Title             string          `json:"title" binding:"required"`
	Description       string          `json:"description"`
	MainImage         string          `json:"main_image"`
	Images            json.RawMessage `json:"images"`
	CategoryID        *int64          `json:"category_id"`
	BrandID           *int64          `json:"brand_id"`
	SpecJSON          json.RawMessage `json:"spec_json"`
	SupplierID        *int64          `json:"supplier_id"`
	PurchasePrice     *float64        `json:"purchase_price"`
	PurchaseCurrency  string          `json:"purchase_currency"`
	PackageWeightKg   *float64        `json:"package_weight_kg"`
	PackageLengthCm   *float64        `json:"package_length_cm"`
	PackageWidthCm    *float64        `json:"package_width_cm"`
	PackageHeightCm   *float64        `json:"package_height_cm"`
	HSCode            string          `json:"hs_code"`
	OriginCountry     string          `json:"origin_country"`
	TargetSalePrice   *float64        `json:"target_sale_price"`
	TargetCurrency    string          `json:"target_currency"`
	TargetPlatformID  *int64          `json:"target_platform_id"`
	DestinationCountry string          `json:"destination_country"`
	SourceURL          string            `json:"source_url"`
	SourcePlatform     string            `json:"source_platform"`
	RawPayload         *json.RawMessage  `json:"raw_payload"`
	CompletenessStatus string            `json:"completeness_status"`
	CollectedAt        *time.Time        `json:"collected_at"`
	Status            string          `json:"status"`
	CreatedBy         string          `json:"created_by"`
}

// UpdateCandidateInput allows partial updates.
type UpdateCandidateInput struct {
	Title             *string          `json:"title"`
	Description       *string          `json:"description"`
	MainImage         *string          `json:"main_image"`
	Images            *json.RawMessage `json:"images"`
	CategoryID        *int64           `json:"category_id"`
	BrandID           *int64           `json:"brand_id"`
	SpecJSON          *json.RawMessage `json:"spec_json"`
	SupplierID        *int64           `json:"supplier_id"`
	PurchasePrice     *float64         `json:"purchase_price"`
	PurchaseCurrency  *string          `json:"purchase_currency"`
	PackageWeightKg   *float64         `json:"package_weight_kg"`
	PackageLengthCm   *float64         `json:"package_length_cm"`
	PackageWidthCm    *float64         `json:"package_width_cm"`
	PackageHeightCm   *float64         `json:"package_height_cm"`
	HSCode            *string          `json:"hs_code"`
	OriginCountry     *string          `json:"origin_country"`
	TargetSalePrice   *float64         `json:"target_sale_price"`
	TargetCurrency    *string          `json:"target_currency"`
	TargetPlatformID  *int64           `json:"target_platform_id"`
	DestinationCountry *string         `json:"destination_country"`
	SourceURL          *string           `json:"source_url"`
	SourcePlatform     *string           `json:"source_platform"`
	RawPayload         *json.RawMessage  `json:"raw_payload"`
	CompletenessStatus *string           `json:"completeness_status"`
	Status            *string          `json:"status"`
	UpdatedBy         *string          `json:"updated_by"`
}
