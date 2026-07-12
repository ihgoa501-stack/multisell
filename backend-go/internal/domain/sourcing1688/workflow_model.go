package sourcing1688

import (
	"encoding/json"
	"time"
)

const (
	StatusPendingReview = "pending_review"
	StatusReviewed      = "reviewed"
	StatusDraftCreated  = "draft_created"
)

type ConvertInput struct {
	CreatedBy            int64           `json:"created_by"`
	PlatformID           int64           `json:"platform_id" binding:"required"`
	Title                string          `json:"title" binding:"required"`
	Description          string          `json:"description" binding:"required"`
	CategoryID           int64           `json:"category_id" binding:"required"`
	Unit                 string          `json:"unit" binding:"required"`
	LocalizedTitle       string          `json:"localized_title" binding:"required"`
	LocalizedDescription string          `json:"localized_description" binding:"required"`
	PlatformSKU          string          `json:"platform_sku" binding:"required"`
	SKUVariants          []DraftSKUInput `json:"sku_variants" binding:"required,min=1,dive"`
	Media                []MediaInput    `json:"media" binding:"required,min=1,dive"`
	Costs                []CostInput     `json:"costs" binding:"required,min=1,dive"`
	ListingPayload       json.RawMessage `json:"listing_payload" binding:"required"`
	TargetLocale         string          `json:"target_locale" binding:"required"`
	ShippingTemplateID   string          `json:"shipping_template_id" binding:"required"`
	CategorySchemaURI    string          `json:"category_schema_uri" binding:"required"`
	CategoryObservedAt   time.Time       `json:"category_observed_at" binding:"required"`
	SupplierAssessment   []EvidenceCheck `json:"supplier_assessment" binding:"required,min=1,dive"`
	ComplianceChecks     []EvidenceCheck `json:"compliance_checks" binding:"required,min=1,dive"`
}

type EvidenceCheck struct {
	CheckType   string    `json:"check_type" binding:"required"`
	Result      string    `json:"result" binding:"required"`
	TruthStatus string    `json:"truth_status" binding:"required"`
	SourceURI   string    `json:"source_uri" binding:"required"`
	ObservedAt  time.Time `json:"observed_at" binding:"required"`
	Notes       string    `json:"notes"`
}

type DraftSKUInput struct {
	SupplierSKU string          `json:"supplier_sku" binding:"required"`
	ChannelSKU  string          `json:"channel_sku" binding:"required"`
	SpecDesc    string          `json:"spec_desc" binding:"required"`
	SpecValues  json.RawMessage `json:"spec_values" binding:"required"`
	CostPrice   float64         `json:"cost_price" binding:"required,gt=0"`
	Price       float64         `json:"price" binding:"required,gt=0"`
	Weight      float64         `json:"weight"`
	Image       string          `json:"image"`
}

type MediaInput struct {
	SourceURL         string          `json:"source_url" binding:"required"`
	ProcessedURL      string          `json:"processed_url" binding:"required"`
	MediaRole         string          `json:"media_role" binding:"required"`
	RightsStatus      string          `json:"rights_status" binding:"required"`
	RightsEvidenceURI string          `json:"rights_evidence_uri" binding:"required"`
	Operations        json.RawMessage `json:"operations" binding:"required"`
	ContentSHA256     string          `json:"content_sha256"`
	Width             int             `json:"width" binding:"required,gt=0"`
	Height            int             `json:"height" binding:"required,gt=0"`
	HasWatermark      bool            `json:"has_watermark"`
	HasChineseText    bool            `json:"has_chinese_text"`
	HasBrandMark      bool            `json:"has_brand_mark"`
	ChannelRuleURI    string          `json:"channel_rule_uri" binding:"required"`
}

type CostInput struct {
	CostType    string    `json:"cost_type" binding:"required"`
	Amount      float64   `json:"amount" binding:"required,gte=0"`
	Currency    string    `json:"currency" binding:"required"`
	TruthStatus string    `json:"truth_status" binding:"required"`
	SourceURI   string    `json:"source_uri" binding:"required"`
	ObservedAt  time.Time `json:"observed_at" binding:"required"`
}

type ConvertResult struct {
	SourcingProductID int64   `json:"sourcing_product_id"`
	SnapshotID        int64   `json:"snapshot_id"`
	ProductID         int64   `json:"product_id"`
	SKUIDs            []int64 `json:"sku_ids"`
	ListingID         int64   `json:"listing_id"`
	DraftID           int64   `json:"draft_id"`
	Status            string  `json:"status"`
}

type DraftDetail struct {
	Draft   draftRow   `json:"draft"`
	Listing listingRow `json:"listing"`
	Product productRow `json:"product"`
	SKUs    []skuRow   `json:"skus"`
	Media   []mediaRow `json:"media"`
	Costs   []costRow  `json:"costs"`
}

type productRow struct {
	ID                                 int64 `gorm:"primaryKey"`
	Name, Description, Unit, MainImage string
	CategoryID                         int64
	Status                             int16
	Images                             json.RawMessage
}

func (productRow) TableName() string { return "product" }

type skuRow struct {
	ID, ProductID            int64
	Code, SpecDesc, Image    string
	SpecValues               json.RawMessage
	Price, CostPrice, Weight float64
	Status                   int16
}

func (skuRow) TableName() string { return "sku" }

type mediaRow struct {
	ID, ProductID, SourceSnapshotID                                     int64
	SourceURL, ProcessedURL, MediaRole, RightsStatus, RightsEvidenceURI string
	Operations                                                          json.RawMessage
	ContentSHA256                                                       string
	Width, Height                                                       int
	HasWatermark, HasChineseText, HasBrandMark                          bool
	ChannelRuleURI                                                      string
	CreatedAt                                                           time.Time
}

func (mediaRow) TableName() string { return "product_media_asset" }

type costRow struct {
	ID, ProductID                    int64
	ExperimentID, CostType           string
	Amount                           float64
	Currency, TruthStatus, SourceURI string
	ObservedAt, CreatedAt            time.Time
}

func (costRow) TableName() string { return "product_cost_input" }

type listingRow struct {
	ID, ProductID, PlatformID int64
	PlatformSKU, Status       string
	PublishedData             json.RawMessage
}

func (listingRow) TableName() string { return "product_listing" }

type draftRow struct {
	ID, SourcingProductID, SnapshotID, ProductID, ListingID, DemandCaseID int64
	ExperimentID                                                          string
	CreatedBy                                                             int64
	CreatedAt                                                             time.Time
}

func (draftRow) TableName() string { return "sourcing_listing_draft" }

type demandCaseRow struct {
	ID, OwnerID          int64
	SalesChannel, Status string
}

func (demandCaseRow) TableName() string { return "demand_case" }

type experimentRow struct {
	ExperimentID, Status, Stage string
	OwnerID                     int64
}

func (experimentRow) TableName() string { return "experiment_case" }

type gateRow struct{ ExperimentID, Stage, Result string }

func (gateRow) TableName() string { return "experiment_gate_decision" }

type objectLinkRow struct{ ExperimentID, ObjectType, ObjectID string }

func (objectLinkRow) TableName() string { return "experiment_object_link" }

type platformRow struct {
	ID         int64
	Name, Code string
	Status     int16
}

func (platformRow) TableName() string { return "platform" }
