package productimage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ReleaseDecisionApproved = "approved"
	ReleaseDecisionRejected = "rejected"
	ReleaseStatusIssued     = "issued"
	ReleaseStatusClaimed    = "claimed"
	ReleaseStatusReconcile  = "reconcile_required"
	ReleaseStatusConsumed   = "consumed"
	ReleaseStatusRevoked    = "revoked"
)

var (
	ErrReleaseGateBlocked  = errors.New("image release attestation gate blocked")
	ErrAttestationExpired  = errors.New("image release attestation expired")
	ErrAttestationConsumed = errors.New("image release attestation already consumed")
)

// ImageRuleSnapshot is immutable. A changed rule creates a new version and
// makes a previously issued attestation unverifiable for a new release.
type ImageRuleSnapshot struct {
	ID             int64           `json:"id" gorm:"primaryKey"`
	OwnerID        int64           `json:"owner_id" gorm:"not null;uniqueIndex:ux_image_rule_scope_version,priority:1;uniqueIndex:ux_image_rule_owner_idem,priority:1"`
	Channel        string          `json:"channel" gorm:"size:64;not null;uniqueIndex:ux_image_rule_scope_version,priority:2"`
	Site           string          `json:"site" gorm:"size:64;not null;uniqueIndex:ux_image_rule_scope_version,priority:3"`
	Locale         string          `json:"locale" gorm:"size:32;not null;uniqueIndex:ux_image_rule_scope_version,priority:4"`
	CategoryID     int64           `json:"category_id" gorm:"not null;uniqueIndex:ux_image_rule_scope_version,priority:5"`
	Version        int64           `json:"version" gorm:"not null;uniqueIndex:ux_image_rule_scope_version,priority:6"`
	Rules          json.RawMessage `json:"rules" gorm:"type:jsonb;not null"`
	RulesSHA256    string          `json:"rules_sha256" gorm:"size:64;not null"`
	EffectiveAt    time.Time       `json:"effective_at" gorm:"not null"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
	IdempotencyKey string          `json:"idempotency_key" gorm:"size:100;not null;uniqueIndex:ux_image_rule_owner_idem,priority:2"`
	RequestSHA256  string          `json:"request_sha256" gorm:"size:64;not null"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ImageRuleDocument is the smallest enforceable channel-image contract. A
// snapshot is rejected unless every field is present and meaningful; the same
// document is evaluated again at issuance and consumption.
type ImageRuleDocument struct {
	SchemaVersion  string   `json:"schema_version"`
	MaxImages      int      `json:"max_images"`
	AllowedRoles   []string `json:"allowed_roles"`
	AllowedFormats []string `json:"allowed_formats"`
	MinWidthPX     int      `json:"min_width_px"`
	MinHeightPX    int      `json:"min_height_px"`
	MaxFileBytes   int64    `json:"max_file_bytes"`
}

func decodeImageRuleDocument(raw []byte) (ImageRuleDocument, error) {
	var rule ImageRuleDocument
	if err := decodeStrict(bytes.NewReader(raw), &rule); err != nil || rule.SchemaVersion != "1" || rule.MaxImages <= 0 || rule.MaxImages > 100 || rule.MinWidthPX <= 0 || rule.MinHeightPX <= 0 || rule.MaxFileBytes <= 0 || len(rule.AllowedRoles) == 0 || len(rule.AllowedFormats) == 0 {
		return ImageRuleDocument{}, ErrInvalidInput
	}
	roles := make(map[string]bool, len(rule.AllowedRoles))
	for i, role := range rule.AllowedRoles {
		rule.AllowedRoles[i] = cleanScope(role)
		if rule.AllowedRoles[i] == "" || roles[rule.AllowedRoles[i]] {
			return ImageRuleDocument{}, ErrInvalidInput
		}
		roles[rule.AllowedRoles[i]] = true
	}
	formats := make(map[string]bool, len(rule.AllowedFormats))
	for i, format := range rule.AllowedFormats {
		format = strings.ToLower(strings.TrimSpace(format))
		if (format != "png" && format != "jpeg") || formats[format] {
			return ImageRuleDocument{}, ErrInvalidInput
		}
		rule.AllowedFormats[i], formats[format] = format, true
	}
	return rule, nil
}

func imageRuleAllows(rule ImageRuleDocument, role, mime string, width, height int, size int64) bool {
	roleOK := false
	for _, allowed := range rule.AllowedRoles {
		roleOK = roleOK || allowed == cleanScope(role)
	}
	format := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(mime)), "image/")
	if format == "jpg" {
		format = "jpeg"
	}
	formatOK := false
	for _, allowed := range rule.AllowedFormats {
		formatOK = formatOK || allowed == format
	}
	return roleOK && formatOK && width >= rule.MinWidthPX && height >= rule.MinHeightPX && size > 0 && size <= rule.MaxFileBytes
}

func (ImageRuleSnapshot) TableName() string { return "product_image_rule_snapshots" }

// ImageSetDecision records the Owner's explicit decision over exact frozen
// image-set bytes. It never mutates the set itself.
type ImageSetDecision struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	OwnerID         int64     `json:"owner_id" gorm:"not null;uniqueIndex:ux_image_set_decision_owner_idem,priority:1"`
	ImageSetID      uint64    `json:"image_set_id" gorm:"not null;index"`
	ImageSetVersion uint      `json:"image_set_version" gorm:"not null"`
	SetManifestSHA  string    `json:"set_manifest_sha256" gorm:"size:64;not null"`
	Decision        string    `json:"decision" gorm:"size:16;not null"`
	Reason          string    `json:"reason" gorm:"type:text;not null"`
	IdempotencyKey  string    `json:"idempotency_key" gorm:"size:100;not null;uniqueIndex:ux_image_set_decision_owner_idem,priority:2"`
	RequestSHA256   string    `json:"request_sha256" gorm:"size:64;not null"`
	DecidedAt       time.Time `json:"decided_at"`
}

func (ImageSetDecision) TableName() string { return "product_image_set_decisions" }

type ImageReleaseAttestation struct {
	ID                 int64                         `json:"id" gorm:"primaryKey"`
	OwnerID            int64                         `json:"owner_id" gorm:"not null;uniqueIndex:ux_image_attestation_owner_idem,priority:1;index"`
	ListingID          int64                         `json:"listing_id" gorm:"not null;index"`
	ProductID          int64                         `json:"product_id" gorm:"not null"`
	PlatformID         int64                         `json:"platform_id" gorm:"not null"`
	PlatformAccountID  int64                         `json:"platform_account_id" gorm:"not null"`
	Channel            string                        `json:"channel" gorm:"size:64;not null"`
	Site               string                        `json:"site" gorm:"size:64;not null"`
	Locale             string                        `json:"locale" gorm:"size:32;not null"`
	CategoryID         int64                         `json:"category_id" gorm:"not null"`
	ListingSnapshotSHA string                        `json:"listing_snapshot_sha256" gorm:"size:64;not null"`
	ImageSetID         uint64                        `json:"image_set_id" gorm:"not null"`
	ImageSetVersion    uint                          `json:"image_set_version" gorm:"not null"`
	SetManifestSHA     string                        `json:"set_manifest_sha256" gorm:"size:64;not null"`
	MediaManifest      json.RawMessage               `json:"media_manifest" gorm:"type:jsonb;not null"`
	MediaManifestSHA   string                        `json:"media_manifest_sha256" gorm:"size:64;not null"`
	RuleSnapshotID     int64                         `json:"rule_snapshot_id" gorm:"not null"`
	RuleSnapshotSHA    string                        `json:"rule_snapshot_sha256" gorm:"size:64;not null"`
	SetDecisionID      int64                         `json:"set_decision_id" gorm:"not null"`
	RightsManifestSHA  string                        `json:"rights_manifest_sha256" gorm:"size:64;not null"`
	ReviewManifestSHA  string                        `json:"review_manifest_sha256" gorm:"size:64;not null"`
	Claims             json.RawMessage               `json:"claims" gorm:"type:jsonb;not null"`
	ClaimsSHA256       string                        `json:"claims_sha256" gorm:"size:64;not null"`
	Signature          string                        `json:"signature" gorm:"size:64;not null"`
	KeyID              string                        `json:"key_id" gorm:"size:64;not null"`
	Nonce              string                        `json:"nonce" gorm:"size:64;not null;uniqueIndex"`
	Status             string                        `json:"status" gorm:"size:16;not null;index"`
	IdempotencyKey     string                        `json:"idempotency_key" gorm:"size:100;not null;uniqueIndex:ux_image_attestation_owner_idem,priority:2"`
	RequestSHA256      string                        `json:"request_sha256" gorm:"size:64;not null"`
	IssuedAt           time.Time                     `json:"issued_at"`
	ExpiresAt          time.Time                     `json:"expires_at"`
	ConsumedAt         *time.Time                    `json:"consumed_at,omitempty"`
	ConsumedByType     string                        `json:"consumed_by_type,omitempty" gorm:"size:64"`
	ConsumedByID       int64                         `json:"consumed_by_id,omitempty"`
	RevokedAt          *time.Time                    `json:"revoked_at,omitempty"`
	CreatedAt          time.Time                     `json:"created_at"`
	Items              []ImageReleaseAttestationItem `json:"items" gorm:"foreignKey:AttestationID;constraint:OnDelete:RESTRICT"`
}

func (ImageReleaseAttestation) TableName() string { return "product_image_release_attestations" }

type ImageReleaseAttestationItem struct {
	ID            int64  `json:"id" gorm:"primaryKey"`
	AttestationID int64  `json:"attestation_id" gorm:"not null;uniqueIndex:ux_attestation_item_ordinal,priority:1"`
	Ordinal       uint   `json:"ordinal" gorm:"not null;uniqueIndex:ux_attestation_item_ordinal,priority:2"`
	Role          string `json:"role" gorm:"size:32;not null"`
	TaskID        int64  `json:"task_id" gorm:"not null"`
	BlobID        string `json:"blob_id" gorm:"size:64;not null"`
	SHA256        string `json:"sha256" gorm:"size:64;not null"`
	MIME          string `json:"mime" gorm:"size:64;not null"`
	Width         int    `json:"width" gorm:"not null"`
	Height        int    `json:"height" gorm:"not null"`
}

func (ImageReleaseAttestationItem) TableName() string {
	return "product_image_release_attestation_items"
}

type CreateRuleSnapshotInput struct {
	Channel, Site, Locale string
	CategoryID            int64
	Rules                 json.RawMessage
	EffectiveAt           time.Time
	ExpiresAt             *time.Time
	IdempotencyKey        string
}

type DecideImageSetInput struct {
	Decision, Reason, IdempotencyKey string
	ExpectedVersion                  uint
}
type IssueAttestationInput struct {
	ImageSetID                        uint64
	RuleSnapshotID, PlatformAccountID int64
	Site, IdempotencyKey              string
	TTL                               time.Duration
}

type ReleaseService struct {
	db    *gorm.DB
	image ImageService
	key   []byte
	keyID string
}

func NewReleaseService(db *gorm.DB, image ImageService, key, keyID string) *ReleaseService {
	return &ReleaseService{db: db, image: image, key: []byte(key), keyID: strings.TrimSpace(keyID)}
}

func ReleaseModels() []any {
	return []any{&ImageRuleSnapshot{}, &ImageSetDecision{}, &ImageReleaseAttestation{}, &ImageReleaseAttestationItem{}}
}

func canonicalJSON(v any) ([]byte, string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, "", err
	}
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, "", err
	}
	raw, err = json.Marshal(normalized)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(raw)
	return raw, hex.EncodeToString(sum[:]), nil
}

func (s *ReleaseService) CreateRuleSnapshot(ctx context.Context, ownerID int64, in CreateRuleSnapshotInput) (*ImageRuleSnapshot, error) {
	in.Channel, in.Site, in.Locale, in.IdempotencyKey = cleanScope(in.Channel), cleanScope(in.Site), strings.TrimSpace(in.Locale), strings.TrimSpace(in.IdempotencyKey)
	if ownerID <= 0 || !validScope(in.Channel) || !validScope(in.Site) || in.Locale == "" || in.CategoryID <= 0 || len(in.Rules) == 0 || in.EffectiveAt.IsZero() || in.IdempotencyKey == "" || (in.ExpiresAt != nil && !in.ExpiresAt.After(in.EffectiveAt)) {
		return nil, ErrInvalidInput
	}
	rules, err := decodeImageRuleDocument(in.Rules)
	if err != nil {
		return nil, ErrInvalidInput
	}
	canonicalRules, rulesHash, err := canonicalJSON(rules)
	if err != nil {
		return nil, err
	}
	req := struct {
		Channel, Site, Locale string
		CategoryID            int64
		RulesSHA              string
		EffectiveAt           time.Time
		ExpiresAt             *time.Time
	}{in.Channel, in.Site, in.Locale, in.CategoryID, rulesHash, in.EffectiveAt.UTC(), in.ExpiresAt}
	_, reqHash, _ := canonicalJSON(req)
	var out ImageRuleSnapshot
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&out).Error; e == nil {
			if out.RequestSHA256 != reqHash {
				return &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
			}
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var version int64
		if e := tx.Model(&ImageRuleSnapshot{}).Where("owner_id=? AND channel=? AND site=? AND locale=? AND category_id=?", ownerID, in.Channel, in.Site, in.Locale, in.CategoryID).Select("COALESCE(MAX(version),0)").Scan(&version).Error; e != nil {
			return e
		}
		out = ImageRuleSnapshot{OwnerID: ownerID, Channel: in.Channel, Site: in.Site, Locale: in.Locale, CategoryID: in.CategoryID, Version: version + 1, Rules: canonicalRules, RulesSHA256: rulesHash, EffectiveAt: in.EffectiveAt.UTC(), ExpiresAt: in.ExpiresAt, IdempotencyKey: in.IdempotencyKey, RequestSHA256: reqHash}
		return tx.Create(&out).Error
	})
	return &out, err
}

func (s *ReleaseService) DecideImageSet(ctx context.Context, ownerID int64, setID uint64, in DecideImageSetInput) (*ImageSetDecision, error) {
	in.Decision, in.Reason, in.IdempotencyKey = cleanScope(in.Decision), strings.TrimSpace(in.Reason), strings.TrimSpace(in.IdempotencyKey)
	if ownerID <= 0 || setID == 0 || (in.Decision != ReleaseDecisionApproved && in.Decision != ReleaseDecisionRejected) || in.Reason == "" || in.IdempotencyKey == "" || in.ExpectedVersion == 0 {
		return nil, ErrInvalidInput
	}
	var set ImageSet
	if err := s.db.WithContext(ctx).Preload("Items").Where("id=? AND owner_id=?", setID, ownerID).First(&set).Error; err != nil {
		return nil, err
	}
	if set.Status != ImageSetFrozen || set.Version != in.ExpectedVersion || set.ManifestSHA == "" {
		return nil, ErrReleaseGateBlocked
	}
	req := struct {
		SetID                      uint64
		Version                    uint
		Manifest, Decision, Reason string
	}{set.ID, set.Version, set.ManifestSHA, in.Decision, in.Reason}
	_, reqHash, _ := canonicalJSON(req)
	var out ImageSetDecision
	if err := s.db.WithContext(ctx).Where("owner_id=? AND idempotency_key=?", ownerID, in.IdempotencyKey).First(&out).Error; err == nil {
		if out.RequestSHA256 != reqHash {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		return &out, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	out = ImageSetDecision{OwnerID: ownerID, ImageSetID: set.ID, ImageSetVersion: set.Version, SetManifestSHA: set.ManifestSHA, Decision: in.Decision, Reason: in.Reason, IdempotencyKey: in.IdempotencyKey, RequestSHA256: reqHash, DecidedAt: time.Now().UTC()}
	if err := s.db.WithContext(ctx).Create(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

type releaseListingRow struct {
	ID, ProductID, PlatformID int64
	Status                    string
	PublishedData             json.RawMessage
}

func (releaseListingRow) TableName() string { return "product_listing" }

type releaseProductRow struct{ ID, CategoryID int64 }

func (releaseProductRow) TableName() string { return "product" }

type releasePlatformRow struct {
	ID   int64
	Code string
}

func (releasePlatformRow) TableName() string { return "platform" }

type releaseAccountRow struct {
	ID, PlatformID int64
	Status         string
}

func (releaseAccountRow) TableName() string { return "platform_integration_account" }

func (s *ReleaseService) Issue(ctx context.Context, ownerID int64, in IssueAttestationInput) (*ImageReleaseAttestation, error) {
	if ownerID <= 0 || in.ImageSetID == 0 || in.RuleSnapshotID <= 0 || in.PlatformAccountID <= 0 || strings.TrimSpace(in.Site) == "" || strings.TrimSpace(in.IdempotencyKey) == "" || in.TTL <= 0 || in.TTL > 24*time.Hour || len(s.key) < 32 || s.keyID == "" || s.image == nil {
		return nil, ErrInvalidInput
	}
	site := cleanScope(in.Site)
	requestIntent := struct {
		ImageSetID        uint64 `json:"image_set_id"`
		RuleSnapshotID    int64  `json:"rule_snapshot_id"`
		PlatformAccountID int64  `json:"platform_account_id"`
		Site              string `json:"site"`
		TTLSeconds        int64  `json:"ttl_seconds"`
	}{in.ImageSetID, in.RuleSnapshotID, in.PlatformAccountID, site, int64(in.TTL / time.Second)}
	_, intentHash, _ := canonicalJSON(requestIntent)
	var replay ImageReleaseAttestation
	if err := s.db.WithContext(ctx).Preload("Items").Where("owner_id=? AND idempotency_key=?", ownerID, strings.TrimSpace(in.IdempotencyKey)).First(&replay).Error; err == nil {
		if replay.RequestSHA256 != intentHash {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		return &replay, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var set ImageSet
	if err := s.db.WithContext(ctx).Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("ordinal") }).Where("id=? AND owner_id=?", in.ImageSetID, ownerID).First(&set).Error; err != nil {
		return nil, err
	}
	manifest, err := CanonicalImageSetManifest(set)
	if err != nil || set.Status != ImageSetFrozen || set.ManifestSHA == "" || manifest != set.ManifestSHA {
		return nil, ErrReleaseGateBlocked
	}
	var decision ImageSetDecision
	if err := s.db.WithContext(ctx).Where("owner_id=? AND image_set_id=? AND image_set_version=? AND set_manifest_sha=?", ownerID, set.ID, set.Version, set.ManifestSHA).Order("id DESC").First(&decision).Error; err != nil || decision.Decision != ReleaseDecisionApproved {
		return nil, ErrReleaseGateBlocked
	}
	var rule ImageRuleSnapshot
	if err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", in.RuleSnapshotID, ownerID).First(&rule).Error; err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if rule.Channel != set.Channel || rule.Site != site || rule.Locale != set.Locale || rule.EffectiveAt.After(now) || (rule.ExpiresAt != nil && !rule.ExpiresAt.After(now)) {
		return nil, ErrReleaseGateBlocked
	}
	ruleDoc, err := decodeImageRuleDocument(rule.Rules)
	if err != nil || len(set.Items) > ruleDoc.MaxImages || canonicalJSONHash(rule.Rules) != rule.RulesSHA256 {
		return nil, ErrReleaseGateBlocked
	}
	var latestRuleVersion int64
	if err := s.db.WithContext(ctx).Model(&ImageRuleSnapshot{}).Where("owner_id=? AND channel=? AND site=? AND locale=? AND category_id=?", ownerID, rule.Channel, rule.Site, rule.Locale, rule.CategoryID).Select("COALESCE(MAX(version),0)").Scan(&latestRuleVersion).Error; err != nil || latestRuleVersion != rule.Version {
		return nil, ErrReleaseGateBlocked
	}
	var listing releaseListingRow
	if err := s.db.WithContext(ctx).First(&listing, set.ListingID).Error; err != nil {
		return nil, err
	}
	var product releaseProductRow
	if err := s.db.WithContext(ctx).First(&product, listing.ProductID).Error; err != nil {
		return nil, err
	}
	var platform releasePlatformRow
	if err := s.db.WithContext(ctx).First(&platform, listing.PlatformID).Error; err != nil {
		return nil, err
	}
	var account releaseAccountRow
	if err := s.db.WithContext(ctx).First(&account, in.PlatformAccountID).Error; err != nil {
		return nil, err
	}
	if listing.Status != "draft" || product.CategoryID != rule.CategoryID || cleanScope(platform.Code) != set.Channel || account.PlatformID != listing.PlatformID || account.Status != "active" {
		return nil, ErrReleaseGateBlocked
	}
	var ownerListingCount int64
	if err := s.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM product_listing pl
JOIN product_variant pv ON pv.sku_product_id=pl.product_id
JOIN product_master pm ON pm.id=pv.product_master_id
WHERE pl.id=? AND pm.owner_id=?`, listing.ID, ownerID).Scan(&ownerListingCount).Error; err != nil || ownerListingCount != 1 {
		return nil, ErrReleaseGateBlocked
	}
	listingSnap := struct {
		ID, ProductID, PlatformID, AccountID, CategoryID int64
		Status, Channel, Site, Locale, PublishedSHA      string
	}{listing.ID, listing.ProductID, listing.PlatformID, account.ID, product.CategoryID, listing.Status, set.Channel, site, set.Locale, canonicalJSONHash(listing.PublishedData)}
	_, listingHash, _ := canonicalJSON(listingSnap)
	items := make([]ImageReleaseAttestationItem, 0, len(set.Items))
	rightsFacts := make([]any, 0, len(set.Items))
	reviewFacts := make([]any, 0, len(set.Items))
	for _, item := range set.Items {
		var task Task
		if err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", item.TaskID, ownerID).First(&task).Error; err != nil {
			return nil, err
		}
		remote, err := s.image.GetJob(ctx, task.ImageServiceJobID)
		if err != nil || remote.Status != "READY" || !verifyRemoteTaskIdentity(&task, remote, ownerID) || isNonPublishableOutput(&task, remote) || remote.OutputBlobID != item.OutputBlobID || remote.ManifestHash != item.TaskManifestHash || task.OutputBlobID != item.AssetSHA || task.Operation != item.Operation || task.Processor != item.Processor || verifyTaskChannelLineage(ctx, s.db, ownerID, &task, set.Channel, imagePurposeForRole(item.Role)) != nil {
			return nil, ErrReleaseGateBlocked
		}
		body, mime, err := s.image.GetBlob(ctx, item.OutputBlobID)
		if err != nil || shaHex(body) != item.AssetSHA {
			return nil, ErrReleaseGateBlocked
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
		if err != nil || !imageRuleAllows(ruleDoc, item.Role, mime, cfg.Width, cfg.Height, int64(len(body))) {
			return nil, ErrReleaseGateBlocked
		}
		purpose := imagePurposeForRole(item.Role)
		if err := (&Service{db: s.db, image: s.image}).VerifyPublicationGate(ctx, ownerID, item.TaskID, item.AssetSHA, purpose, set.Channel, set.Locale); err != nil {
			return nil, ErrReleaseGateBlocked
		}
		provider := cleanScope(task.Processor)
		if provider == "" {
			provider = "deterministic"
		}
		region := "*"
		if provider == "deterministic" {
			region = "local"
		}
		var grant RightsGrant
		if err := s.db.WithContext(ctx).Where("owner_id=? AND asset_sha=? AND purpose IN ? AND channel IN ? AND jurisdiction IN ? AND provider IN ? AND region IN ? AND owner_verified=? AND can_copy=? AND can_cross_border=? AND can_commercial_publish=? AND can_platform_sublicense=? AND trademark_cleared=? AND likeness_cleared=? AND revoked_at IS NULL AND valid_from<=? AND (valid_until IS NULL OR valid_until>?)", ownerID, item.AssetSHA, []string{purpose, "*"}, []string{set.Channel, "*"}, []string{cleanScope(set.Locale), "*"}, []string{provider, "*"}, []string{region, "*"}, true, true, true, true, true, true, true, now, now).Order("id DESC").First(&grant).Error; err != nil {
			return nil, ErrReleaseGateBlocked
		}
		var review Review
		if err := s.db.WithContext(ctx).Where("owner_id=? AND task_id=? AND asset_sha=? AND purpose=? AND channel=?", ownerID, item.TaskID, item.AssetSHA, purpose, set.Channel).Order("id DESC").First(&review).Error; err != nil {
			return nil, ErrReleaseGateBlocked
		}
		items = append(items, ImageReleaseAttestationItem{Ordinal: item.Ordinal, Role: item.Role, TaskID: item.TaskID, BlobID: item.OutputBlobID, SHA256: item.AssetSHA, MIME: mime, Width: cfg.Width, Height: cfg.Height})
		rightsFacts = append(rightsFacts, struct {
			ID, Version int64
			Evidence    string
		}{grant.ID, grant.Version, grant.EvidenceSHA})
		reviewFacts = append(reviewFacts, struct {
			ID, TaskVersion int64
			Evidence        string
		}{review.ID, review.ExpectedTaskVersion, review.EvidenceSHA})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Ordinal < items[j].Ordinal })
	mediaRaw, mediaHash, _ := canonicalJSON(items)
	_, rightsHash, _ := canonicalJSON(rightsFacts)
	_, reviewHash, _ := canonicalJSON(reviewFacts)
	nonce := shaHex([]byte(fmt.Sprintf("%d:%d:%d:%s", ownerID, set.ID, now.UnixNano(), in.IdempotencyKey)))
	claimsObj := struct {
		OwnerID, ListingID, ProductID, PlatformID, PlatformAccountID int64
		Channel, Site, Locale                                        string
		CategoryID                                                   int64
		ListingSHA                                                   string
		SetID                                                        uint64
		SetVersion                                                   uint
		SetSHA, MediaSHA                                             string
		RuleID                                                       int64
		RuleSHA                                                      string
		DecisionID                                                   int64
		RightsSHA, ReviewSHA, KeyID, Nonce                           string
		IssuedAt, ExpiresAt                                          time.Time
	}{ownerID, listing.ID, listing.ProductID, listing.PlatformID, account.ID, set.Channel, site, set.Locale, product.CategoryID, listingHash, set.ID, set.Version, set.ManifestSHA, mediaHash, rule.ID, rule.RulesSHA256, decision.ID, rightsHash, reviewHash, s.keyID, nonce, now, now.Add(in.TTL)}
	claims, claimsHash, _ := canonicalJSON(claimsObj)
	mac := hmac.New(sha256.New, s.key)
	mac.Write(claims)
	signature := hex.EncodeToString(mac.Sum(nil))
	out := ImageReleaseAttestation{OwnerID: ownerID, ListingID: listing.ID, ProductID: listing.ProductID, PlatformID: listing.PlatformID, PlatformAccountID: account.ID, Channel: set.Channel, Site: site, Locale: set.Locale, CategoryID: product.CategoryID, ListingSnapshotSHA: listingHash, ImageSetID: set.ID, ImageSetVersion: set.Version, SetManifestSHA: set.ManifestSHA, MediaManifest: mediaRaw, MediaManifestSHA: mediaHash, RuleSnapshotID: rule.ID, RuleSnapshotSHA: rule.RulesSHA256, SetDecisionID: decision.ID, RightsManifestSHA: rightsHash, ReviewManifestSHA: reviewHash, Claims: claims, ClaimsSHA256: claimsHash, Signature: signature, KeyID: s.keyID, Nonce: nonce, Status: ReleaseStatusIssued, IdempotencyKey: strings.TrimSpace(in.IdempotencyKey), RequestSHA256: intentHash, IssuedAt: now, ExpiresAt: now.Add(in.TTL), Items: items}
	if err := s.db.WithContext(ctx).Create(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func shaHex(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func canonicalJSONHash(raw []byte) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return shaHex(raw)
	}
	_, hash, err := canonicalJSON(v)
	if err != nil {
		return shaHex(raw)
	}
	return hash
}

func (s *ReleaseService) Get(ctx context.Context, ownerID, id int64) (*ImageReleaseAttestation, error) {
	var out ImageReleaseAttestation
	err := s.db.WithContext(ctx).Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("ordinal") }).Where("id=? AND owner_id=?", id, ownerID).First(&out).Error
	return &out, err
}

// Consume atomically proves the stored canonical claims/signature and current
// release gates before granting exactly one external-write attempt.
func (s *ReleaseService) Consume(ctx context.Context, ownerID, id int64, consumerType string, consumerID int64) error {
	if ownerID <= 0 || id <= 0 || consumerID <= 0 || strings.TrimSpace(consumerType) == "" {
		return ErrInvalidInput
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.claimOnTx(ctx, tx, ownerID, id, consumerType, consumerID, ReleaseStatusConsumed)
	})
}

// claimOnTx revalidates the complete attestation and performs its one-way,
// single-use transition in the caller's transaction. Controlled publication
// uses this so attempt creation and claim are one database commit before any
// external write. A claim is deliberately never rolled back to issued.
func (s *ReleaseService) claimOnTx(ctx context.Context, tx *gorm.DB, ownerID, id int64, consumerType string, consumerID int64, targetStatus string) error {
	if targetStatus != ReleaseStatusConsumed && targetStatus != ReleaseStatusClaimed {
		return ErrInvalidInput
	}
	var a ImageReleaseAttestation
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Items").Where("id=? AND owner_id=?", id, ownerID).First(&a).Error; err != nil {
		return err
	}
	if a.Status == ReleaseStatusConsumed {
		return ErrAttestationConsumed
	}
	if a.Status != ReleaseStatusIssued || !a.ExpiresAt.After(time.Now().UTC()) {
		return ErrAttestationExpired
	}
	var claimsValue any
	if json.Unmarshal(a.Claims, &claimsValue) != nil {
		return ErrReleaseGateBlocked
	}
	canonicalClaims, claimsHash, err := canonicalJSON(claimsValue)
	if err != nil || claimsHash != a.ClaimsSHA256 {
		return ErrReleaseGateBlocked
	}
	mac := hmac.New(sha256.New, s.key)
	mac.Write(canonicalClaims)
	if a.KeyID != s.keyID || !hmac.Equal([]byte(a.Signature), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		return ErrReleaseGateBlocked
	}
	var set ImageSet
	if err := tx.Preload("Items").Where("id=? AND owner_id=?", a.ImageSetID, ownerID).First(&set).Error; err != nil {
		return err
	}
	setHash, err := CanonicalImageSetManifest(set)
	if err != nil || set.Status != ImageSetFrozen || setHash != a.SetManifestSHA {
		return ErrReleaseGateBlocked
	}
	var decision ImageSetDecision
	if err := tx.Where("id=? AND owner_id=? AND image_set_id=? AND decision=?", a.SetDecisionID, ownerID, set.ID, ReleaseDecisionApproved).First(&decision).Error; err != nil || decision.SetManifestSHA != a.SetManifestSHA || decision.ImageSetVersion != a.ImageSetVersion {
		return ErrReleaseGateBlocked
	}
	var rule ImageRuleSnapshot
	if err := tx.Where("id=? AND owner_id=?", a.RuleSnapshotID, ownerID).First(&rule).Error; err != nil {
		return err
	}
	ruleDoc, ruleErr := decodeImageRuleDocument(rule.Rules)
	if ruleErr != nil || canonicalJSONHash(rule.Rules) != rule.RulesSHA256 || rule.RulesSHA256 != a.RuleSnapshotSHA || rule.Channel != a.Channel || rule.Site != a.Site || rule.Locale != a.Locale || rule.CategoryID != a.CategoryID || rule.EffectiveAt.After(time.Now().UTC()) || rule.ExpiresAt != nil && !rule.ExpiresAt.After(time.Now().UTC()) {
		return ErrReleaseGateBlocked
	}
	var latestRuleVersion int64
	if err := tx.Model(&ImageRuleSnapshot{}).Where("owner_id=? AND channel=? AND site=? AND locale=? AND category_id=?", ownerID, rule.Channel, rule.Site, rule.Locale, rule.CategoryID).Select("COALESCE(MAX(version),0)").Scan(&latestRuleVersion).Error; err != nil || latestRuleVersion != rule.Version {
		return ErrReleaseGateBlocked
	}
	var listing releaseListingRow
	if err := tx.First(&listing, a.ListingID).Error; err != nil {
		return err
	}
	var product releaseProductRow
	if err := tx.First(&product, a.ProductID).Error; err != nil {
		return err
	}
	var platform releasePlatformRow
	if err := tx.First(&platform, a.PlatformID).Error; err != nil {
		return err
	}
	var account releaseAccountRow
	if err := tx.First(&account, a.PlatformAccountID).Error; err != nil {
		return err
	}
	if listing.ProductID != a.ProductID || listing.PlatformID != a.PlatformID || listing.Status != "draft" || product.CategoryID != a.CategoryID || account.PlatformID != a.PlatformID || account.Status != "active" || cleanScope(platform.Code) != a.Channel {
		return ErrReleaseGateBlocked
	}
	var ownerListingCount int64
	if err := tx.Raw(`SELECT COUNT(*) FROM product_listing pl JOIN product_variant pv ON pv.sku_product_id=pl.product_id JOIN product_master pm ON pm.id=pv.product_master_id WHERE pl.id=? AND pm.owner_id=?`, listing.ID, ownerID).Scan(&ownerListingCount).Error; err != nil || ownerListingCount != 1 {
		return ErrReleaseGateBlocked
	}
	listingSnap := struct {
		ID, ProductID, PlatformID, AccountID, CategoryID int64
		Status, Channel, Site, Locale, PublishedSHA      string
	}{listing.ID, listing.ProductID, listing.PlatformID, account.ID, product.CategoryID, listing.Status, a.Channel, a.Site, a.Locale, canonicalJSONHash(listing.PublishedData)}
	_, listingHash, _ := canonicalJSON(listingSnap)
	if listingHash != a.ListingSnapshotSHA {
		return ErrReleaseGateBlocked
	}
	attestedByOrdinal := make(map[uint]ImageReleaseAttestationItem, len(a.Items))
	for _, item := range a.Items {
		attestedByOrdinal[item.Ordinal] = item
	}
	mediaNow := make([]ImageReleaseAttestationItem, 0, len(set.Items))
	if len(set.Items) > ruleDoc.MaxImages {
		return ErrReleaseGateBlocked
	}
	rightsFacts := make([]any, 0, len(set.Items))
	reviewFacts := make([]any, 0, len(set.Items))
	now := time.Now().UTC()
	for _, item := range set.Items {
		if err := (&Service{db: tx, image: s.image}).VerifyPublicationGate(ctx, ownerID, item.TaskID, item.AssetSHA, imagePurposeForRole(item.Role), set.Channel, set.Locale); err != nil {
			return ErrReleaseGateBlocked
		}
		var task Task
		if err := tx.Where("id=? AND owner_id=?", item.TaskID, ownerID).First(&task).Error; err != nil {
			return err
		}
		remote, err := s.image.GetJob(ctx, task.ImageServiceJobID)
		if err != nil || remote.Status != "READY" || !verifyRemoteTaskIdentity(&task, remote, ownerID) || isNonPublishableOutput(&task, remote) || remote.OutputBlobID != item.OutputBlobID || remote.ManifestHash != item.TaskManifestHash || task.Operation != item.Operation || task.Processor != item.Processor || verifyTaskChannelLineage(ctx, tx, ownerID, &task, set.Channel, imagePurposeForRole(item.Role)) != nil {
			return ErrReleaseGateBlocked
		}
		body, mime, err := s.image.GetBlob(ctx, item.OutputBlobID)
		if err != nil || shaHex(body) != item.AssetSHA {
			return ErrReleaseGateBlocked
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
		if err != nil || !imageRuleAllows(ruleDoc, item.Role, mime, cfg.Width, cfg.Height, int64(len(body))) {
			return ErrReleaseGateBlocked
		}
		attested, ok := attestedByOrdinal[item.Ordinal]
		if !ok || attested.Role != item.Role || attested.TaskID != item.TaskID || attested.BlobID != item.OutputBlobID || attested.SHA256 != item.AssetSHA || attested.MIME != mime || attested.Width != cfg.Width || attested.Height != cfg.Height {
			return ErrReleaseGateBlocked
		}
		mediaNow = append(mediaNow, ImageReleaseAttestationItem{Ordinal: item.Ordinal, Role: item.Role, TaskID: item.TaskID, BlobID: item.OutputBlobID, SHA256: item.AssetSHA, MIME: mime, Width: cfg.Width, Height: cfg.Height})
		purpose := imagePurposeForRole(item.Role)
		provider := cleanScope(task.Processor)
		if provider == "" {
			provider = "deterministic"
		}
		region := "*"
		if provider == "deterministic" {
			region = "local"
		}
		var grant RightsGrant
		if err := tx.Where("owner_id=? AND asset_sha=? AND purpose IN ? AND channel IN ? AND jurisdiction IN ? AND provider IN ? AND region IN ? AND owner_verified=? AND can_copy=? AND can_cross_border=? AND can_commercial_publish=? AND can_platform_sublicense=? AND trademark_cleared=? AND likeness_cleared=? AND revoked_at IS NULL AND valid_from<=? AND (valid_until IS NULL OR valid_until>?)", ownerID, item.AssetSHA, []string{purpose, "*"}, []string{set.Channel, "*"}, []string{cleanScope(set.Locale), "*"}, []string{provider, "*"}, []string{region, "*"}, true, true, true, true, true, true, true, now, now).Order("id DESC").First(&grant).Error; err != nil {
			return ErrReleaseGateBlocked
		}
		var review Review
		if err := tx.Where("owner_id=? AND task_id=? AND asset_sha=? AND purpose=? AND channel=?", ownerID, item.TaskID, item.AssetSHA, purpose, set.Channel).Order("id DESC").First(&review).Error; err != nil {
			return ErrReleaseGateBlocked
		}
		rightsFacts = append(rightsFacts, struct {
			ID, Version int64
			Evidence    string
		}{grant.ID, grant.Version, grant.EvidenceSHA})
		reviewFacts = append(reviewFacts, struct {
			ID, TaskVersion int64
			Evidence        string
		}{review.ID, review.ExpectedTaskVersion, review.EvidenceSHA})
	}
	_, mediaHash, _ := canonicalJSON(mediaNow)
	_, rightsHash, _ := canonicalJSON(rightsFacts)
	_, reviewHash, _ := canonicalJSON(reviewFacts)
	if mediaHash != a.MediaManifestSHA || rightsHash != a.RightsManifestSHA || reviewHash != a.ReviewManifestSHA {
		return ErrReleaseGateBlocked
	}
	now = time.Now().UTC()
	updates := map[string]any{"status": targetStatus, "consumed_by_type": consumerType, "consumed_by_id": consumerID}
	if targetStatus == ReleaseStatusConsumed {
		updates["consumed_at"] = now
	}
	res := tx.Model(&ImageReleaseAttestation{}).Where("id=? AND status=?", a.ID, ReleaseStatusIssued).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrAttestationConsumed
	}
	return nil
}
