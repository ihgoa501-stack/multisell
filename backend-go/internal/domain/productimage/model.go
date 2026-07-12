package productimage

import "time"

const (
	TruthUnknown = "unknown"
	TruthQuoted  = "quoted"
	TruthActual  = "actual"
)

// Asset is LingMirror's Owner-scoped reference to bytes held by Image Service.
// It is an input fact, not proof that the Owner has editing or publishing rights.
type Asset struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	OwnerID     int64     `json:"owner_id" gorm:"not null;index:idx_product_image_assets_owner_created"`
	BlobID      string    `json:"blob_id" gorm:"size:64;not null"`
	Filename    string    `json:"filename" gorm:"size:255;not null"`
	ContentType string    `json:"content_type" gorm:"size:100;not null"`
	SizeBytes   int64     `json:"size_bytes" gorm:"not null"`
	SHA256      string    `json:"sha256" gorm:"size:64;not null"`
	Truth       string    `json:"truth" gorm:"size:20;not null;default:unknown"`
	CreatedAt   time.Time `json:"created_at" gorm:"index:idx_product_image_assets_owner_created"`
}

func (Asset) TableName() string { return "product_image_assets" }

// Task is the Owner-facing business record for one technical Image Service job.
type Task struct {
	ID                int64     `json:"id" gorm:"primaryKey"`
	OwnerID           int64     `json:"owner_id" gorm:"not null;uniqueIndex:uidx_product_image_task_owner_idem;index:idx_product_image_tasks_owner_created"`
	AssetID           int64     `json:"asset_id" gorm:"not null;index"`
	ImageServiceJobID string    `json:"image_service_job_id,omitempty" gorm:"size:100;index"`
	OutputBlobID      string    `json:"output_blob_id,omitempty" gorm:"size:64;index"`
	OutputURL         string    `json:"output_url,omitempty" gorm:"-"`
	IdempotencyKey    string    `json:"idempotency_key" gorm:"size:100;not null;uniqueIndex:uidx_product_image_task_owner_idem"`
	ManifestHash      string    `json:"manifest_hash" gorm:"size:64;not null"`
	Operation         string    `json:"operation" gorm:"size:64;not null"`
	Processor         string    `json:"processor" gorm:"size:64;not null;default:deterministic"`
	Version           int64     `json:"version" gorm:"not null;default:1"`
	Width             int       `json:"width" gorm:"not null"`
	Height            int       `json:"height" gorm:"not null"`
	Format            string    `json:"format" gorm:"size:20;not null"`
	Status            string    `json:"status" gorm:"size:32;not null"`
	ErrorCode         string    `json:"error_code,omitempty" gorm:"size:100"`
	CreatedAt         time.Time `json:"created_at" gorm:"index:idx_product_image_tasks_owner_created"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (Task) TableName() string { return "product_image_tasks" }

// Review records an Owner decision. Ordinary input may not assert actual truth.
type Review struct {
	ID                  int64      `json:"id" gorm:"primaryKey"`
	OwnerID             int64      `json:"owner_id" gorm:"not null;index"`
	TaskID              int64      `json:"task_id" gorm:"not null;index"`
	Decision            string     `json:"decision" gorm:"size:32;not null"`
	Truth               string     `json:"truth" gorm:"size:20;not null;default:unknown"`
	Notes               string     `json:"notes,omitempty" gorm:"type:text"`
	AssetSHA            string     `json:"asset_sha256,omitempty" gorm:"size:64;index"`
	Purpose             string     `json:"purpose,omitempty" gorm:"size:64"`
	Channel             string     `json:"channel,omitempty" gorm:"size:64"`
	ProductAuthenticity string     `json:"product_authenticity,omitempty" gorm:"size:16"`
	RightsStatus        string     `json:"rights_status,omitempty" gorm:"size:16"`
	ChannelRules        string     `json:"channel_rules,omitempty" gorm:"size:16"`
	ClaimsScene         string     `json:"claims_scene,omitempty" gorm:"size:16"`
	TechnicalVisual     string     `json:"technical_visual,omitempty" gorm:"size:16"`
	EvidenceSHA         string     `json:"evidence_sha256,omitempty" gorm:"size:64"`
	EvidenceTruth       string     `json:"evidence_truth,omitempty" gorm:"size:20"`
	IdempotencyKey      string     `json:"idempotency_key,omitempty" gorm:"size:100;uniqueIndex:uidx_product_image_review_owner_idem"`
	RequestHash         string     `json:"request_hash,omitempty" gorm:"size:64"`
	ExpectedTaskVersion int64      `json:"expected_task_version,omitempty"`
	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

func (Review) TableName() string { return "product_image_reviews" }

// RightsGrant is an Owner-verified, revocable authorization for exact image
// bytes. Each permission is explicit so a broad-sounding note cannot silently
// authorize AI processing or commercial distribution.
type RightsGrant struct {
	ID                       int64      `json:"id" gorm:"primaryKey"`
	OwnerID                  int64      `json:"owner_id" gorm:"not null;uniqueIndex:uidx_product_image_rights_owner_idem;index:idx_product_image_rights_scope"`
	AssetID                  *int64     `json:"asset_id,omitempty" gorm:"index"`
	AssetSHA                 string     `json:"asset_sha256" gorm:"size:64;not null;index:idx_product_image_rights_scope"`
	CanCopy                  bool       `json:"can_copy"`
	CanModify                bool       `json:"can_modify"`
	CanThirdPartyAI          bool       `json:"can_third_party_ai"`
	CanCrossBorder           bool       `json:"can_cross_border"`
	CanCommercialPublish     bool       `json:"can_commercial_publish"`
	CanPlatformSublicense    bool       `json:"can_platform_sublicense"`
	TrademarkCleared         bool       `json:"trademark_cleared"`
	LikenessCleared          bool       `json:"likeness_cleared"`
	Purpose                  string     `json:"purpose" gorm:"size:64;not null;index:idx_product_image_rights_scope"`
	Jurisdiction             string     `json:"jurisdiction" gorm:"size:64;not null"`
	Channel                  string     `json:"channel" gorm:"size:64;not null;index:idx_product_image_rights_scope"`
	Provider                 string     `json:"provider" gorm:"size:64;not null"`
	Region                   string     `json:"region" gorm:"size:64;not null"`
	Grantor                  string     `json:"grantor" gorm:"size:255;not null"`
	RightsChain              string     `json:"rights_chain" gorm:"type:text;not null"`
	EvidenceSHA              string     `json:"evidence_sha256" gorm:"size:64;not null"`
	OwnerVerified            bool       `json:"owner_verified" gorm:"not null"`
	ValidFrom                time.Time  `json:"valid_from"`
	ValidUntil               *time.Time `json:"valid_until,omitempty"`
	RevokedAt                *time.Time `json:"revoked_at,omitempty"`
	RevocationReason         string     `json:"revocation_reason,omitempty" gorm:"type:text"`
	RevocationIdempotencyKey string     `json:"revocation_idempotency_key,omitempty" gorm:"size:100"`
	RevocationRequestHash    string     `json:"revocation_request_hash,omitempty" gorm:"size:64"`
	IdempotencyKey           string     `json:"idempotency_key" gorm:"size:100;not null;uniqueIndex:uidx_product_image_rights_owner_idem"`
	RequestHash              string     `json:"request_hash" gorm:"size:64;not null"`
	Version                  int64      `json:"version" gorm:"not null;default:1"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

func (RightsGrant) TableName() string { return "product_image_rights_grants" }

type CostEntry struct {
	ID                  int64     `json:"id" gorm:"primaryKey"`
	OwnerID             int64     `json:"owner_id" gorm:"not null;uniqueIndex:uidx_product_image_cost_owner_idem;index"`
	TaskID              int64     `json:"task_id" gorm:"not null;index"`
	Kind                string    `json:"kind" gorm:"size:16;not null"` // estimated or actual
	Category            string    `json:"category" gorm:"size:32;not null"`
	Provider            string    `json:"provider" gorm:"size:64;not null"`
	Amount              string    `json:"amount" gorm:"size:32;not null"`
	Currency            string    `json:"currency" gorm:"size:3;not null"`
	ExchangeRate        string    `json:"exchange_rate" gorm:"size:32;not null"`
	ExchangeRateSource  string    `json:"exchange_rate_source" gorm:"size:255;not null"`
	ObservedAt          time.Time `json:"observed_at"`
	BillingStatus       string    `json:"billing_status" gorm:"size:24;not null"`
	EvidenceSHA         string    `json:"evidence_sha256,omitempty" gorm:"size:64"`
	IdempotencyKey      string    `json:"idempotency_key" gorm:"size:100;not null;uniqueIndex:uidx_product_image_cost_owner_idem"`
	RequestHash         string    `json:"request_hash" gorm:"size:64;not null"`
	ExpectedTaskVersion int64     `json:"expected_task_version"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (CostEntry) TableName() string { return "product_image_cost_entries" }

type CreateTaskInput struct {
	AssetID        int64  `json:"asset_id" binding:"required"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Operation      string `json:"operation" binding:"required"`
	Width          int    `json:"width" binding:"required"`
	Height         int    `json:"height" binding:"required"`
	Format         string `json:"format" binding:"required"`
}

type ExecutionInput struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

type ExecutionApproval struct {
	ID           int64      `json:"id" gorm:"primaryKey"`
	ExecutionID  string     `json:"execution_id" gorm:"size:64;not null;uniqueIndex"`
	OwnerID      int64      `json:"owner_id" gorm:"not null;index"`
	TaskID       int64      `json:"task_id" gorm:"not null;index"`
	TaskVersion  int64      `json:"task_version" gorm:"not null"`
	ManifestHash string     `json:"manifest_hash" gorm:"size:64;not null"`
	Operation    string     `json:"operation" gorm:"size:64;not null"`
	Processor    string     `json:"processor" gorm:"size:64;not null"`
	MaxCost      string     `json:"max_cost" gorm:"size:32;not null"`
	Currency     string     `json:"currency" gorm:"size:3;not null"`
	Nonce        string     `json:"-" gorm:"size:64;not null;uniqueIndex"`
	ApprovedAt   time.Time  `json:"approved_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	ConsumedAt   *time.Time `json:"consumed_at,omitempty"`
}

func (ExecutionApproval) TableName() string { return "product_image_execution_approvals" }

type ApprovalInput struct {
	Processor       string `json:"processor" binding:"required"`
	MaxCost         string `json:"max_cost" binding:"required"`
	Currency        string `json:"currency" binding:"required"`
	ExpectedVersion int64  `json:"expected_version" binding:"required"`
}
