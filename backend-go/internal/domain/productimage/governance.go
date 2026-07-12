package productimage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	ReviewPassed  = "passed"
	ReviewBlocked = "blocked"
	ReviewUnknown = "unknown"
)

var (
	decimalPattern         = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$`)
	positiveDecimalPattern = regexp.MustCompile(`^([1-9][0-9]{0,9})(\.[0-9]{1,4})?$|^0\.[0-9]{0,3}[1-9]$`)
	sha256Pattern          = regexp.MustCompile(`^[0-9a-f]{64}$`)
	ErrGateBlocked         = errors.New("image rights or five-axis review gate blocked")
)

type RightsGrantInput struct {
	AssetID               *int64     `json:"asset_id"`
	AssetSHA              string     `json:"asset_sha256"`
	CanCopy               bool       `json:"can_copy"`
	CanModify             bool       `json:"can_modify"`
	CanThirdPartyAI       bool       `json:"can_third_party_ai"`
	CanCrossBorder        bool       `json:"can_cross_border"`
	CanCommercialPublish  bool       `json:"can_commercial_publish"`
	CanPlatformSublicense bool       `json:"can_platform_sublicense"`
	TrademarkCleared      bool       `json:"trademark_cleared"`
	LikenessCleared       bool       `json:"likeness_cleared"`
	Purpose               string     `json:"purpose"`
	Jurisdiction          string     `json:"jurisdiction"`
	Channel               string     `json:"channel"`
	Provider              string     `json:"provider"`
	Region                string     `json:"region"`
	Grantor               string     `json:"grantor"`
	RightsChain           string     `json:"rights_chain"`
	EvidenceSHA           string     `json:"evidence_sha256"`
	OwnerVerified         bool       `json:"owner_verified"`
	ValidFrom             time.Time  `json:"valid_from"`
	ValidUntil            *time.Time `json:"valid_until"`
	IdempotencyKey        string     `json:"idempotency_key"`
	ExpectedVersion       int64      `json:"expected_version"`
}

type RevokeRightsInput struct {
	ExpectedVersion int64  `json:"expected_version"`
	IdempotencyKey  string `json:"idempotency_key"`
	Reason          string `json:"reason"`
}

type FiveAxisReviewInput struct {
	AssetSHA            string `json:"asset_sha256"`
	Purpose             string `json:"purpose"`
	Channel             string `json:"channel"`
	ProductAuthenticity string `json:"product_authenticity"`
	RightsStatus        string `json:"rights"`
	ChannelRules        string `json:"channel_rules"`
	ClaimsScene         string `json:"claims_scene"`
	TechnicalVisual     string `json:"technical_visual"`
	EvidenceSHA         string `json:"evidence_sha256"`
	EvidenceTruth       string `json:"evidence_truth"`
	Notes               string `json:"notes"`
	IdempotencyKey      string `json:"idempotency_key"`
	ExpectedVersion     int64  `json:"expected_version"`
}

type CostEntryInput struct {
	Kind               string    `json:"kind"`
	Category           string    `json:"category"`
	Provider           string    `json:"provider"`
	Amount             string    `json:"amount"`
	Currency           string    `json:"currency"`
	ExchangeRate       string    `json:"exchange_rate"`
	ExchangeRateSource string    `json:"exchange_rate_source"`
	ObservedAt         time.Time `json:"observed_at"`
	BillingStatus      string    `json:"billing_status"`
	EvidenceSHA        string    `json:"evidence_sha256"`
	IdempotencyKey     string    `json:"idempotency_key"`
	ExpectedVersion    int64     `json:"expected_version"`
}

func requestHash(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func cleanScope(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
func validScope(s string) bool   { return s != "" && len(s) <= 64 }

func (s *Service) CreateRightsGrant(ctx context.Context, ownerID int64, in RightsGrantInput) (*RightsGrant, error) {
	in.AssetSHA, in.Purpose, in.Jurisdiction, in.Channel, in.Provider, in.Region = cleanScope(in.AssetSHA), cleanScope(in.Purpose), cleanScope(in.Jurisdiction), cleanScope(in.Channel), cleanScope(in.Provider), cleanScope(in.Region)
	in.Grantor, in.RightsChain, in.IdempotencyKey = strings.TrimSpace(in.Grantor), strings.TrimSpace(in.RightsChain), strings.TrimSpace(in.IdempotencyKey)
	in.EvidenceSHA = cleanScope(in.EvidenceSHA)
	if ownerID <= 0 || !sha256Pattern.MatchString(in.AssetSHA) || !sha256Pattern.MatchString(in.EvidenceSHA) || !validScope(in.Purpose) || !validScope(in.Jurisdiction) || !validScope(in.Channel) || !validScope(in.Provider) || !validScope(in.Region) || in.Grantor == "" || in.RightsChain == "" || !in.OwnerVerified || in.ValidFrom.IsZero() || in.IdempotencyKey == "" || in.ExpectedVersion != 1 || (in.ValidUntil != nil && !in.ValidUntil.After(in.ValidFrom)) {
		return nil, ErrInvalidInput
	}
	if in.AssetID != nil {
		var asset Asset
		if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ? AND sha256 = ?", *in.AssetID, ownerID, in.AssetSHA).First(&asset).Error; err != nil {
			return nil, err
		}
	}
	hash := requestHash(in)
	var existing RightsGrant
	if err := s.db.WithContext(ctx).Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; err == nil {
		if existing.RequestHash != hash {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	grant := &RightsGrant{OwnerID: ownerID, AssetID: in.AssetID, AssetSHA: in.AssetSHA, CanCopy: in.CanCopy, CanModify: in.CanModify, CanThirdPartyAI: in.CanThirdPartyAI, CanCrossBorder: in.CanCrossBorder, CanCommercialPublish: in.CanCommercialPublish, CanPlatformSublicense: in.CanPlatformSublicense, TrademarkCleared: in.TrademarkCleared, LikenessCleared: in.LikenessCleared, Purpose: in.Purpose, Jurisdiction: in.Jurisdiction, Channel: in.Channel, Provider: in.Provider, Region: in.Region, Grantor: in.Grantor, RightsChain: in.RightsChain, EvidenceSHA: in.EvidenceSHA, OwnerVerified: true, ValidFrom: in.ValidFrom.UTC(), ValidUntil: in.ValidUntil, IdempotencyKey: in.IdempotencyKey, RequestHash: hash, Version: 1}
	if err := s.db.WithContext(ctx).Create(grant).Error; err != nil {
		return nil, err
	}
	return grant, nil
}

func (s *Service) RevokeRightsGrant(ctx context.Context, ownerID, id int64, in RevokeRightsInput) (*RightsGrant, error) {
	in.IdempotencyKey, in.Reason = strings.TrimSpace(in.IdempotencyKey), strings.TrimSpace(in.Reason)
	if ownerID <= 0 || id <= 0 || in.ExpectedVersion <= 0 || in.IdempotencyKey == "" || in.Reason == "" {
		return nil, ErrInvalidInput
	}
	var grant RightsGrant
	hash := requestHash(in)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND owner_id = ?", id, ownerID).First(&grant).Error; err != nil {
			return err
		}
		if grant.RevokedAt != nil {
			if grant.Version == in.ExpectedVersion+1 && grant.RevocationIdempotencyKey == in.IdempotencyKey && grant.RevocationRequestHash == hash {
				return nil
			}
			if grant.RevocationIdempotencyKey == in.IdempotencyKey {
				return &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
			}
			return &ConflictError{Code: "VERSION_CONFLICT"}
		}
		now := time.Now().UTC()
		res := tx.Model(&RightsGrant{}).Where("id = ? AND owner_id = ? AND version = ? AND revoked_at IS NULL", id, ownerID, in.ExpectedVersion).Updates(map[string]any{"revoked_at": now, "revocation_reason": in.Reason, "revocation_idempotency_key": in.IdempotencyKey, "revocation_request_hash": hash, "version": gorm.Expr("version + 1")})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return &ConflictError{Code: "VERSION_CONFLICT"}
		}
		grant.RevokedAt = &now
		grant.RevocationReason = in.Reason
		grant.RevocationIdempotencyKey = in.IdempotencyKey
		grant.RevocationRequestHash = hash
		grant.Version++
		return nil
	})
	return &grant, err
}

func (s *Service) ListRights(ctx context.Context, ownerID int64, assetSHA string, page, size int) ([]RightsGrant, int64, error) {
	page, size = normalizePage(page, size)
	q := s.db.WithContext(ctx).Model(&RightsGrant{}).Where("owner_id = ?", ownerID)
	if assetSHA = cleanScope(assetSHA); assetSHA != "" {
		q = q.Where("asset_sha = ?", assetSHA)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []RightsGrant
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&out).Error
	return out, total, err
}

func (s *Service) CreateFiveAxisReview(ctx context.Context, ownerID, taskID int64, in FiveAxisReviewInput) (*Review, error) {
	in.AssetSHA, in.Purpose, in.Channel, in.EvidenceSHA, in.EvidenceTruth, in.IdempotencyKey = cleanScope(in.AssetSHA), cleanScope(in.Purpose), cleanScope(in.Channel), cleanScope(in.EvidenceSHA), cleanScope(in.EvidenceTruth), strings.TrimSpace(in.IdempotencyKey)
	statuses := []string{in.ProductAuthenticity, in.RightsStatus, in.ChannelRules, in.ClaimsScene, in.TechnicalVisual}
	for i := range statuses {
		statuses[i] = cleanScope(statuses[i])
		if !validReviewStatus(statuses[i]) {
			return nil, ErrInvalidInput
		}
	}
	if ownerID <= 0 || taskID <= 0 || !sha256Pattern.MatchString(in.AssetSHA) || !sha256Pattern.MatchString(in.EvidenceSHA) || !validScope(in.Purpose) || !validScope(in.Channel) || in.EvidenceTruth == TruthActual || !validEvidenceTruth(in.EvidenceTruth) || in.IdempotencyKey == "" || in.ExpectedVersion <= 0 {
		return nil, ErrInvalidInput
	}
	var task Task
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", taskID, ownerID).First(&task).Error; err != nil {
		return nil, err
	}
	if task.Version != in.ExpectedVersion || task.OutputBlobID != in.AssetSHA {
		return nil, &ConflictError{Code: "VERSION_CONFLICT"}
	}
	hash := requestHash(in)
	var existing Review
	if err := s.db.WithContext(ctx).Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; err == nil {
		if existing.RequestHash != hash {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := time.Now().UTC()
	review := &Review{OwnerID: ownerID, TaskID: taskID, Decision: "five_axis_review", Truth: TruthUnknown, Notes: strings.TrimSpace(in.Notes), AssetSHA: in.AssetSHA, Purpose: in.Purpose, Channel: in.Channel, ProductAuthenticity: statuses[0], RightsStatus: statuses[1], ChannelRules: statuses[2], ClaimsScene: statuses[3], TechnicalVisual: statuses[4], EvidenceSHA: in.EvidenceSHA, EvidenceTruth: in.EvidenceTruth, IdempotencyKey: in.IdempotencyKey, RequestHash: hash, ExpectedTaskVersion: in.ExpectedVersion, VerifiedAt: &now}
	if err := s.db.WithContext(ctx).Create(review).Error; err != nil {
		return nil, err
	}
	return review, nil
}

func (s *Service) ListReviews(ctx context.Context, ownerID, taskID int64, page, size int) ([]Review, int64, error) {
	page, size = normalizePage(page, size)
	q := s.db.WithContext(ctx).Model(&Review{}).Where("owner_id = ? AND task_id = ?", ownerID, taskID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Review
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&out).Error
	return out, total, err
}

func (s *Service) CreateCostEntry(ctx context.Context, ownerID, taskID int64, in CostEntryInput) (*CostEntry, error) {
	in.Kind, in.Category, in.Provider, in.Currency, in.ExchangeRateSource, in.BillingStatus, in.IdempotencyKey = cleanScope(in.Kind), cleanScope(in.Category), cleanScope(in.Provider), strings.ToUpper(strings.TrimSpace(in.Currency)), strings.TrimSpace(in.ExchangeRateSource), cleanScope(in.BillingStatus), strings.TrimSpace(in.IdempotencyKey)
	in.EvidenceSHA = cleanScope(in.EvidenceSHA)
	if ownerID <= 0 || taskID <= 0 || (in.Kind != "estimated" && in.Kind != "actual") || !validScope(in.Category) || !validScope(in.Provider) || !decimalPattern.MatchString(in.Amount) || !positiveDecimalPattern.MatchString(in.ExchangeRate) || !allowedExecutionCurrency(in.Currency) || in.ExchangeRateSource == "" || in.ObservedAt.IsZero() || !validBillingStatus(in.BillingStatus) || in.IdempotencyKey == "" || in.ExpectedVersion <= 0 || (in.EvidenceSHA != "" && !sha256Pattern.MatchString(in.EvidenceSHA)) {
		return nil, ErrInvalidInput
	}
	var task Task
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", taskID, ownerID).First(&task).Error; err != nil {
		return nil, err
	}
	if task.Version != in.ExpectedVersion {
		return nil, &ConflictError{Code: "VERSION_CONFLICT"}
	}
	hash := requestHash(in)
	var existing CostEntry
	if err := s.db.WithContext(ctx).Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; err == nil {
		if existing.RequestHash != hash {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	entry := &CostEntry{OwnerID: ownerID, TaskID: taskID, Kind: in.Kind, Category: in.Category, Provider: in.Provider, Amount: in.Amount, Currency: in.Currency, ExchangeRate: in.ExchangeRate, ExchangeRateSource: in.ExchangeRateSource, ObservedAt: in.ObservedAt.UTC(), BillingStatus: in.BillingStatus, EvidenceSHA: in.EvidenceSHA, IdempotencyKey: in.IdempotencyKey, RequestHash: hash, ExpectedTaskVersion: in.ExpectedVersion}
	if err := s.db.WithContext(ctx).Create(entry).Error; err != nil {
		return nil, err
	}
	return entry, nil
}
func (s *Service) ListCosts(ctx context.Context, ownerID, taskID int64, page, size int) ([]CostEntry, int64, error) {
	page, size = normalizePage(page, size)
	q := s.db.WithContext(ctx).Model(&CostEntry{}).Where("owner_id = ? AND task_id = ?", ownerID, taskID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []CostEntry
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&out).Error
	return out, total, err
}

func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}
func validReviewStatus(v string) bool {
	return v == ReviewPassed || v == ReviewBlocked || v == ReviewUnknown
}
func validEvidenceTruth(v string) bool {
	return v == TruthUnknown || v == TruthQuoted || v == "inferred"
}
func validBillingStatus(v string) bool {
	switch v {
	case "estimated", "pending", "invoiced", "paid", "reconciled", "unknown":
		return true
	}
	return false
}

// VerifyPublicationGate is fail-closed and binds both rights and review to the
// exact output, purpose and channel used by an image-set item.
func (s *Service) VerifyPublicationGate(ctx context.Context, ownerID, taskID int64, assetSHA, purpose, channel, jurisdiction string) error {
	now := time.Now().UTC()
	assetSHA, purpose, channel, jurisdiction = cleanScope(assetSHA), cleanScope(purpose), cleanScope(channel), cleanScope(jurisdiction)
	var task Task
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", taskID, ownerID).First(&task).Error; err != nil {
		return err
	}
	provider := cleanScope(task.Processor)
	if provider == "" {
		provider = "deterministic"
	}
	region := "*"
	if provider == "deterministic" {
		region = "local"
	}
	var rights int64
	q := s.db.WithContext(ctx).Model(&RightsGrant{}).Where("owner_id = ? AND asset_sha = ? AND purpose IN ? AND channel IN ? AND jurisdiction IN ? AND provider IN ? AND region IN ? AND owner_verified = ? AND can_copy = ? AND can_cross_border = ? AND can_commercial_publish = ? AND can_platform_sublicense = ? AND trademark_cleared = ? AND likeness_cleared = ? AND revoked_at IS NULL AND valid_from <= ? AND (valid_until IS NULL OR valid_until > ?)", ownerID, assetSHA, []string{purpose, "*"}, []string{channel, "*"}, []string{jurisdiction, "*"}, []string{provider, "*"}, []string{region, "*"}, true, true, true, true, true, true, true, now, now)
	if err := q.Count(&rights).Error; err != nil {
		return err
	}
	if rights == 0 {
		return ErrGateBlocked
	}
	var latest Review
	if err := s.db.WithContext(ctx).Where("owner_id = ? AND task_id = ? AND asset_sha = ? AND purpose = ? AND channel = ?", ownerID, taskID, assetSHA, purpose, channel).Order("id DESC").First(&latest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGateBlocked
		}
		return err
	}
	if latest.ExpectedTaskVersion != task.Version || latest.ProductAuthenticity != ReviewPassed || latest.RightsStatus != ReviewPassed || latest.ChannelRules != ReviewPassed || latest.ClaimsScene != ReviewPassed || latest.TechnicalVisual != ReviewPassed {
		return ErrGateBlocked
	}
	if task.Processor != "" && task.Processor != "deterministic" {
		var costs int64
		if err := s.db.WithContext(ctx).Model(&CostEntry{}).Where("owner_id = ? AND task_id = ? AND billing_status <> ?", ownerID, taskID, "unknown").Count(&costs).Error; err != nil {
			return err
		}
		if costs == 0 {
			return ErrGateBlocked
		}
	}
	return nil
}
