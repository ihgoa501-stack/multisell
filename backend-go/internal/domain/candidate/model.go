package candidate

import (
	"encoding/json"
	"time"
)

// completenessFieldNames is the set of fields checked by computeCompleteness.
// Used for validation in field fill/skip operations.
var completenessFieldNames = map[string]bool{
	"title": true, "purchase_price": true, "main_image": true,
	"supplier_id":       true,
	"package_weight_kg": true, "package_length_cm": true, "package_width_cm": true, "package_height_cm": true,
	"hs_code": true, "target_sale_price": true, "origin_country": true,
}

// CandidateProduct maps to the "candidate_product" table.
type CandidateProduct struct {
	ID                 int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title              string          `gorm:"column:title;not null" json:"title"`
	Description        string          `gorm:"column:description" json:"description"`
	MainImage          string          `gorm:"column:main_image" json:"main_image"`
	Images             json.RawMessage `gorm:"column:images;type:jsonb" json:"images,omitempty"`
	CategoryID         *int64          `gorm:"column:category_id" json:"category_id,omitempty"`
	BrandID            *int64          `gorm:"column:brand_id" json:"brand_id,omitempty"`
	SpecJSON           json.RawMessage `gorm:"column:spec_json;type:jsonb" json:"spec_json,omitempty"`
	SupplierID         *int64          `gorm:"column:supplier_id" json:"supplier_id,omitempty"`
	PurchasePrice      float64         `gorm:"column:purchase_price;default:0" json:"purchase_price"`
	PurchaseCurrency   string          `gorm:"column:purchase_currency;default:CNY" json:"purchase_currency"`
	PackageWeightKg    float64         `gorm:"column:package_weight_kg;default:0" json:"package_weight_kg"`
	PackageLengthCm    float64         `gorm:"column:package_length_cm;default:0" json:"package_length_cm"`
	PackageWidthCm     float64         `gorm:"column:package_width_cm;default:0" json:"package_width_cm"`
	PackageHeightCm    float64         `gorm:"column:package_height_cm;default:0" json:"package_height_cm"`
	HSCode             string          `gorm:"column:hs_code" json:"hs_code"`
	OriginCountry      string          `gorm:"column:origin_country;default:CN" json:"origin_country"`
	TargetSalePrice    float64         `gorm:"column:target_sale_price;default:0" json:"target_sale_price"`
	TargetCurrency     string          `gorm:"column:target_currency;default:USD" json:"target_currency"`
	TargetPlatformID   *int64          `gorm:"column:target_platform_id" json:"target_platform_id,omitempty"`
	DestinationCountry string          `gorm:"column:destination_country;default:US" json:"destination_country"`
	Status             string          `gorm:"column:status" json:"status"`
	IsSeedData         bool            `gorm:"column:is_seed_data;default:false" json:"is_seed_data"`
	SourceURL          string          `gorm:"column:source_url;size:2048;default:''" json:"source_url"`
	SourcePlatform     string          `gorm:"column:source_platform;size:64;default:''" json:"source_platform"`
	RawPayload         json.RawMessage `gorm:"column:raw_payload;type:jsonb" json:"raw_payload,omitempty"`
	CompletenessStatus string          `gorm:"column:completeness_status;size:32;default:incomplete" json:"completeness_status"`
	SkippedFields      json.RawMessage `gorm:"column:skipped_fields;type:jsonb" json:"skipped_fields,omitempty"`
	CollectedAt        *time.Time      `gorm:"column:collected_at" json:"collected_at,omitempty"`
	CreatedBy          string          `gorm:"column:created_by" json:"created_by"`
	UpdatedBy          string          `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt          time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (CandidateProduct) TableName() string { return "candidate_product" }

// CollectLead maps to the "collect_leads" table.
// List page results (search, shop pages) are stored as CollectLead entries,
// NOT as CandidateProduct. CandidateProduct is only for detail page results.
type CollectLead struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title            string     `gorm:"column:title;not null;default:''" json:"title"`
	PriceRange       string     `gorm:"column:price_range;size:128;default:''" json:"price_range"`
	DetailURL        string     `gorm:"column:detail_url;size:2048;default:''" json:"detail_url"`
	CanonicalKey     *string    `gorm:"column:canonical_key;size:64;uniqueIndex" json:"canonical_key,omitempty"`
	ImageURL         string     `gorm:"column:image_url;size:2048;default:''" json:"image_url"`
	ShopHint         string     `gorm:"column:shop_hint;size:256;default:''" json:"shop_hint"`
	SourcePageURL    string     `gorm:"column:source_page_url;size:2048;default:''" json:"source_page_url"`
	CollectionDriver string     `gorm:"column:collection_driver;size:64;default:''" json:"collection_driver"`
	EvidenceID       *int64     `gorm:"column:evidence_id;index" json:"evidence_id,omitempty"`
	ConfidenceState  string     `gorm:"column:confidence_state;size:32;default:unverified" json:"confidence_state"`
	Status           string     `gorm:"column:status;size:64;default:pending_detail_collect" json:"status"`
	CollectedAt      *time.Time `gorm:"column:collected_at" json:"collected_at,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CollectLead) TableName() string { return "collect_leads" }

// CollectionEvidence stores one immutable raw snapshot per collected page.
// Individual leads reference it instead of duplicating the page payload.
type CollectionEvidence struct {
	ID             int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourceURL      string          `gorm:"column:source_url;size:2048;not null" json:"source_url"`
	Driver         string          `gorm:"column:driver;size:64;not null" json:"driver"`
	RawPayload     json.RawMessage `gorm:"column:raw_payload;type:jsonb;not null" json:"raw_payload"`
	ParserVersion  string          `gorm:"column:parser_version;size:64;not null" json:"parser_version"`
	EvidenceSHA256 string          `gorm:"column:evidence_sha256;size:64;not null;index" json:"evidence_sha256"`
	CorrelationID  string          `gorm:"column:correlation_id;size:128;not null;index" json:"correlation_id"`
	CollectedAt    time.Time       `gorm:"column:collected_at;not null" json:"collected_at"`
	CreatedAt      time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CollectionEvidence) TableName() string { return "collection_evidence" }

// CreateCandidateInput is the payload for creating a candidate product.
type CreateCandidateInput struct {
	Title              string          `json:"title" binding:"required"`
	Description        string          `json:"description"`
	MainImage          string          `json:"main_image"`
	Images             json.RawMessage `json:"images"`
	CategoryID         *int64          `json:"category_id"`
	BrandID            *int64          `json:"brand_id"`
	SpecJSON           json.RawMessage `json:"spec_json"`
	SupplierID         *int64          `json:"supplier_id"`
	PurchasePrice      *float64        `json:"purchase_price"`
	PurchaseCurrency   string          `json:"purchase_currency"`
	PackageWeightKg    *float64        `json:"package_weight_kg"`
	PackageLengthCm    *float64        `json:"package_length_cm"`
	PackageWidthCm     *float64        `json:"package_width_cm"`
	PackageHeightCm    *float64        `json:"package_height_cm"`
	HSCode             string          `json:"hs_code"`
	OriginCountry      string          `json:"origin_country"`
	TargetSalePrice    *float64        `json:"target_sale_price"`
	TargetCurrency     string          `json:"target_currency"`
	TargetPlatformID   *int64          `json:"target_platform_id"`
	DestinationCountry string          `json:"destination_country"`
	Status             string          `json:"status"`
	CreatedBy          string          `json:"created_by"`
	SourceURL          string          `json:"source_url"`
	SourcePlatform     string          `json:"source_platform"`
	RawPayload         json.RawMessage `json:"raw_payload"`
	CompletenessStatus string          `json:"completeness_status"`
	CollectedAt        *time.Time      `json:"collected_at"`
}

// UpdateCandidateInput allows partial updates.
type UpdateCandidateInput struct {
	Title              *string          `json:"title"`
	Description        *string          `json:"description"`
	MainImage          *string          `json:"main_image"`
	Images             *json.RawMessage `json:"images"`
	CategoryID         *int64           `json:"category_id"`
	BrandID            *int64           `json:"brand_id"`
	SpecJSON           *json.RawMessage `json:"spec_json"`
	SupplierID         *int64           `json:"supplier_id"`
	PurchasePrice      *float64         `json:"purchase_price"`
	PurchaseCurrency   *string          `json:"purchase_currency"`
	PackageWeightKg    *float64         `json:"package_weight_kg"`
	PackageLengthCm    *float64         `json:"package_length_cm"`
	PackageWidthCm     *float64         `json:"package_width_cm"`
	PackageHeightCm    *float64         `json:"package_height_cm"`
	HSCode             *string          `json:"hs_code"`
	OriginCountry      *string          `json:"origin_country"`
	TargetSalePrice    *float64         `json:"target_sale_price"`
	TargetCurrency     *string          `json:"target_currency"`
	TargetPlatformID   *int64           `json:"target_platform_id"`
	DestinationCountry *string          `json:"destination_country"`
	Status             *string          `json:"status"`
	UpdatedBy          *string          `json:"updated_by"`
}

// CandidateDetail enriches CandidateProduct with computed completeness info.
type CandidateDetail struct {
	CandidateProduct
	MissingFields []string `json:"missing_fields"`
}

// ListCandidateFilter holds optional filters for listing candidate products.
type ListCandidateFilter struct {
	Status             string
	Search             string
	CompletenessStatus string
}

// FillField describes one field to fill manually.
type FillField struct {
	Field string      `json:"field" binding:"required"`
	Value interface{} `json:"value" binding:"required"`
}

// FillFieldsInput is the payload for manual field fill.
type FillFieldsInput struct {
	Fields []FillField `json:"fields" binding:"required"`
}

// SkipFieldInput is the payload for marking a field as skipped.
type SkipFieldInput struct {
	Field string `json:"field" binding:"required"`
}
