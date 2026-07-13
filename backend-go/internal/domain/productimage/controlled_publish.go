package productimage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PublishAttemptPending           = "pending"
	PublishAttemptCalling           = "calling"
	PublishAttemptReconcileRequired = "reconcile_required"
	PublishAttemptSucceeded         = "succeeded"
	PublishAttemptFailedTerminal    = "failed_terminal"
	PublishAttemptUnsupported       = "unsupported"
)

var (
	ErrUnsupportedPublisher = errors.New("controlled-byte publisher is unsupported")
	ErrReconcileRequired    = errors.New("publication outcome requires reconciliation")
	ErrPublishInProgress    = errors.New("publication attempt is already claimed")
)

// ControlledMedia is the only media shape a publisher may receive. Bytes are
// fetched from Image Service by the backend and re-hashed before this value is
// constructed. There is intentionally no URL field.
type ControlledMedia struct {
	Ordinal uint
	Role    string
	Bytes   []byte
	SHA256  string
	MIME    string
	Width   int
	Height  int
}

type ControlledPublishRequest struct {
	AttemptID         int64
	OwnerID           int64
	ListingID         int64
	PlatformID        int64
	PlatformAccountID int64
	AttestationID     int64
	IdempotencyKey    string
	MediaManifestSHA  string
	Media             []ControlledMedia
}

type ControlledPublishReceipt struct {
	RemoteReference string
	ReceiptEvidence json.RawMessage
}

type ReconcileResult struct {
	Resolved bool
	Success  bool
	Receipt  ControlledPublishReceipt
	Evidence json.RawMessage
}

// ControlledPublisher must upload the supplied bytes itself and must use the
// attempt idempotency key at the remote boundary. Legacy URL adapters do not
// implement this contract and therefore cannot be registered.
type ControlledPublisher interface {
	PublishControlled(context.Context, ControlledPublishRequest) (ControlledPublishReceipt, error)
	ReconcileControlled(context.Context, ControlledPublishRequest) (ReconcileResult, error)
}

type PublisherRegistry struct {
	mu         sync.RWMutex
	publishers map[string]ControlledPublisher
}

func NewPublisherRegistry() *PublisherRegistry {
	return &PublisherRegistry{publishers: make(map[string]ControlledPublisher)}
}

func (r *PublisherRegistry) Register(channel string, p ControlledPublisher) error {
	channel = cleanScope(channel)
	if !validScope(channel) || p == nil {
		return ErrInvalidInput
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.publishers[channel]; exists {
		return &ConflictError{Code: "PUBLISHER_ALREADY_REGISTERED"}
	}
	r.publishers[channel] = p
	return nil
}

func (r *PublisherRegistry) Get(channel string) (ControlledPublisher, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.publishers[cleanScope(channel)]
	return p, ok
}

type ImagePublishAttempt struct {
	ID                  int64           `json:"id" gorm:"primaryKey"`
	OwnerID             int64           `json:"owner_id" gorm:"not null;uniqueIndex:ux_image_publish_attempt_owner_idem,priority:1;index"`
	AttestationID       int64           `json:"attestation_id" gorm:"not null;index"`
	ListingID           int64           `json:"listing_id" gorm:"not null"`
	PlatformID          int64           `json:"platform_id" gorm:"not null"`
	PlatformAccountID   int64           `json:"platform_account_id" gorm:"not null"`
	Channel             string          `json:"channel" gorm:"size:64;not null"`
	MediaManifestSHA    string          `json:"media_manifest_sha256" gorm:"size:64;not null"`
	Status              string          `json:"status" gorm:"size:32;not null;index"`
	IdempotencyKey      string          `json:"idempotency_key" gorm:"size:100;not null;uniqueIndex:ux_image_publish_attempt_owner_idem,priority:2"`
	RequestSHA256       string          `json:"request_sha256" gorm:"size:64;not null"`
	RemoteReference     string          `json:"remote_reference,omitempty" gorm:"type:text"`
	ReceiptEvidence     json.RawMessage `json:"receipt_evidence,omitempty" gorm:"type:jsonb"`
	ReceiptSHA256       string          `json:"receipt_sha256,omitempty" gorm:"size:64"`
	ReconcileEvidence   json.RawMessage `json:"reconcile_evidence,omitempty" gorm:"type:jsonb"`
	FailureCode         string          `json:"failure_code,omitempty" gorm:"size:64"`
	CreatedAt           time.Time       `json:"created_at"`
	ClaimedAt           *time.Time      `json:"claimed_at,omitempty"`
	ExternalCalledAt    *time.Time      `json:"external_called_at,omitempty"`
	ReconcileRequiredAt *time.Time      `json:"reconcile_required_at,omitempty"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty"`
}

func (ImagePublishAttempt) TableName() string { return "product_image_publish_attempts" }

type PublishService struct {
	db       *gorm.DB
	image    ImageService
	release  *ReleaseService
	registry *PublisherRegistry
}

func NewPublishService(db *gorm.DB, image ImageService, release *ReleaseService, registry *PublisherRegistry) *PublishService {
	return &PublishService{db: db, image: image, release: release, registry: registry}
}

func (s *PublishService) Get(ctx context.Context, ownerID, attemptID int64) (*ImagePublishAttempt, error) {
	var out ImagePublishAttempt
	err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", attemptID, ownerID).First(&out).Error
	return &out, err
}

func publishRequestHash(a *ImageReleaseAttestation) string {
	_, hash, _ := canonicalJSON(struct {
		AttestationID, ListingID, PlatformID, PlatformAccountID int64
		Channel, Manifest                                       string
	}{a.ID, a.ListingID, a.PlatformID, a.PlatformAccountID, a.Channel, a.MediaManifestSHA})
	return hash
}

func (s *PublishService) Execute(ctx context.Context, ownerID, attestationID int64, idempotencyKey string) (*ImagePublishAttempt, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if ownerID <= 0 || attestationID <= 0 || idempotencyKey == "" || len(idempotencyKey) > 100 || s.image == nil || s.release == nil {
		return nil, ErrInvalidInput
	}
	a, err := s.release.Get(ctx, ownerID, attestationID)
	if err != nil {
		return nil, err
	}
	requestHash := publishRequestHash(a)
	publisher, supported := s.registry.Get(a.Channel)
	var attempt ImagePublishAttempt
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("owner_id=? AND idempotency_key=?", ownerID, idempotencyKey).First(&attempt).Error; e == nil {
			if attempt.RequestSHA256 != requestHash {
				return &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
			}
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		status := PublishAttemptPending
		failure := ""
		if !supported {
			status, failure = PublishAttemptUnsupported, "CONTROLLED_PUBLISHER_UNSUPPORTED"
		}
		attempt = ImagePublishAttempt{OwnerID: ownerID, AttestationID: a.ID, ListingID: a.ListingID, PlatformID: a.PlatformID, PlatformAccountID: a.PlatformAccountID, Channel: a.Channel, MediaManifestSHA: a.MediaManifestSHA, Status: status, FailureCode: failure, IdempotencyKey: idempotencyKey, RequestSHA256: requestHash}
		return tx.Create(&attempt).Error
	})
	if err != nil {
		// A concurrent request may have committed the same unique idempotency
		// key after our initial lookup. Reloading turns only the exact same
		// request into a replay; changed intent remains a conflict.
		if reloadErr := s.db.WithContext(ctx).Where("owner_id=? AND idempotency_key=?", ownerID, idempotencyKey).First(&attempt).Error; reloadErr != nil {
			return nil, err
		}
		if attempt.RequestSHA256 != requestHash {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
	}
	if attempt.Status == PublishAttemptUnsupported {
		return &attempt, ErrUnsupportedPublisher
	}
	if attempt.Status != PublishAttemptPending {
		if attempt.Status == PublishAttemptReconcileRequired {
			return &attempt, ErrReconcileRequired
		}
		return &attempt, nil
	}

	request, err := s.controlledRequest(ctx, &attempt, a)
	if err != nil {
		return &attempt, err
	}
	now := time.Now().UTC()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&ImagePublishAttempt{}).Where("id=? AND owner_id=? AND status=?", attempt.ID, ownerID, PublishAttemptPending).Updates(map[string]any{"status": PublishAttemptCalling, "claimed_at": now, "external_called_at": now})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrPublishInProgress
		}
		return s.release.claimOnTx(ctx, tx, ownerID, a.ID, "controlled_media_publish_attempt", attempt.ID, ReleaseStatusClaimed)
	})
	if err != nil {
		current, _ := s.Get(ctx, ownerID, attempt.ID)
		return current, err
	}

	receipt, callErr := publisher.PublishControlled(ctx, request)
	if callErr != nil || strings.TrimSpace(receipt.RemoteReference) == "" {
		reason := "PUBLISH_OUTCOME_UNKNOWN"
		if callErr == nil {
			reason = "INVALID_PUBLISH_RECEIPT"
		}
		_ = s.markReconcileRequired(ctx, attempt.ID, reason)
		current, _ := s.Get(ctx, ownerID, attempt.ID)
		return current, ErrReconcileRequired
	}
	if err := s.completeSuccess(ctx, ownerID, attempt.ID, receipt, nil); err != nil {
		_ = s.markReconcileRequired(ctx, attempt.ID, "RECEIPT_PERSISTENCE_UNCERTAIN")
		current, _ := s.Get(ctx, ownerID, attempt.ID)
		return current, ErrReconcileRequired
	}
	return s.Get(ctx, ownerID, attempt.ID)
}

func (s *PublishService) controlledRequest(ctx context.Context, attempt *ImagePublishAttempt, a *ImageReleaseAttestation) (ControlledPublishRequest, error) {
	media := make([]ControlledMedia, 0, len(a.Items))
	for _, item := range a.Items {
		body, mime, err := s.image.GetBlob(ctx, item.BlobID)
		if err != nil || shaHex(body) != item.SHA256 || mime != item.MIME {
			return ControlledPublishRequest{}, ErrReleaseGateBlocked
		}
		media = append(media, ControlledMedia{Ordinal: item.Ordinal, Role: item.Role, Bytes: body, SHA256: item.SHA256, MIME: mime, Width: item.Width, Height: item.Height})
	}
	if len(media) == 0 {
		return ControlledPublishRequest{}, ErrReleaseGateBlocked
	}
	return ControlledPublishRequest{AttemptID: attempt.ID, OwnerID: a.OwnerID, ListingID: a.ListingID, PlatformID: a.PlatformID, PlatformAccountID: a.PlatformAccountID, AttestationID: a.ID, IdempotencyKey: attempt.IdempotencyKey, MediaManifestSHA: a.MediaManifestSHA, Media: media}, nil
}

func (s *PublishService) markReconcileRequired(ctx context.Context, id int64, code string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		attemptResult := tx.Model(&ImagePublishAttempt{}).Where("id=? AND status=?", id, PublishAttemptCalling).Updates(map[string]any{"status": PublishAttemptReconcileRequired, "failure_code": code, "reconcile_required_at": now})
		if attemptResult.Error != nil {
			return attemptResult.Error
		}
		if attemptResult.RowsAffected != 1 {
			return ErrPublishInProgress
		}
		attestationResult := tx.Model(&ImageReleaseAttestation{}).Where("consumed_by_type=? AND consumed_by_id=? AND status=?", "controlled_media_publish_attempt", id, ReleaseStatusClaimed).Update("status", ReleaseStatusReconcile)
		if attestationResult.Error != nil {
			return attestationResult.Error
		}
		if attestationResult.RowsAffected != 1 {
			return ErrReleaseGateBlocked
		}
		return nil
	})
}

func (s *PublishService) completeSuccess(ctx context.Context, ownerID, id int64, receipt ControlledPublishReceipt, reconcileEvidence json.RawMessage) error {
	remote := strings.TrimSpace(receipt.RemoteReference)
	if remote == "" {
		return ErrInvalidInput
	}
	receiptRaw, receiptHash, err := canonicalJSON(struct {
		Remote   string
		Manifest string
		Evidence json.RawMessage
	}{remote, "", receipt.ReceiptEvidence})
	if err != nil {
		return err
	}
	_ = receiptRaw
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var attempt ImagePublishAttempt
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", id, ownerID).First(&attempt).Error; err != nil {
			return err
		}
		if attempt.Status != PublishAttemptCalling && attempt.Status != PublishAttemptReconcileRequired {
			if attempt.Status == PublishAttemptSucceeded {
				return nil
			}
			return ErrReleaseGateBlocked
		}
		_, receiptHash, err = canonicalJSON(struct {
			Remote, Manifest string
			Evidence         json.RawMessage
		}{remote, attempt.MediaManifestSHA, receipt.ReceiptEvidence})
		if err != nil {
			return err
		}
		if err := tx.Model(&attempt).Updates(map[string]any{"status": PublishAttemptSucceeded, "remote_reference": remote, "receipt_evidence": receipt.ReceiptEvidence, "receipt_sha256": receiptHash, "reconcile_evidence": reconcileEvidence, "completed_at": now, "failure_code": ""}).Error; err != nil {
			return err
		}
		res := tx.Model(&ImageReleaseAttestation{}).Where("id=? AND owner_id=? AND consumed_by_type=? AND consumed_by_id=? AND status IN ?", attempt.AttestationID, ownerID, "controlled_media_publish_attempt", attempt.ID, []string{ReleaseStatusClaimed, ReleaseStatusReconcile}).Updates(map[string]any{"status": ReleaseStatusConsumed, "consumed_at": now})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrAttestationConsumed
		}
		return nil
	})
}

func (s *PublishService) Reconcile(ctx context.Context, ownerID, attemptID int64) (*ImagePublishAttempt, error) {
	attempt, err := s.Get(ctx, ownerID, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.Status != PublishAttemptReconcileRequired {
		return attempt, ErrInvalidInput
	}
	publisher, ok := s.registry.Get(attempt.Channel)
	if !ok {
		return attempt, ErrUnsupportedPublisher
	}
	a, err := s.release.Get(ctx, ownerID, attempt.AttestationID)
	if err != nil {
		return nil, err
	}
	req, err := s.controlledRequest(ctx, attempt, a)
	if err != nil {
		return attempt, err
	}
	result, err := publisher.ReconcileControlled(ctx, req)
	if err != nil || !result.Resolved {
		return attempt, ErrReconcileRequired
	}
	if result.Success {
		if err := s.completeSuccess(ctx, ownerID, attempt.ID, result.Receipt, result.Evidence); err != nil {
			return attempt, err
		}
		return s.Get(ctx, ownerID, attempt.ID)
	}
	now := time.Now().UTC()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&ImagePublishAttempt{}).Where("id=? AND owner_id=? AND status=?", attempt.ID, ownerID, PublishAttemptReconcileRequired).Updates(map[string]any{"status": PublishAttemptFailedTerminal, "reconcile_evidence": result.Evidence, "failure_code": "REMOTE_NOT_PUBLISHED", "completed_at": now})
		if res.Error != nil {
			return fmt.Errorf("reconcile transition failed: %w", res.Error)
		}
		if res.RowsAffected != 1 {
			return ErrPublishInProgress
		}
		attestationResult := tx.Model(&ImageReleaseAttestation{}).Where("id=? AND owner_id=? AND status=?", attempt.AttestationID, ownerID, ReleaseStatusReconcile).Updates(map[string]any{"status": ReleaseStatusRevoked, "revoked_at": now})
		if attestationResult.Error != nil {
			return attestationResult.Error
		}
		if attestationResult.RowsAffected != 1 {
			return ErrReleaseGateBlocked
		}
		return nil
	})
	if err != nil {
		return attempt, err
	}
	return s.Get(ctx, ownerID, attempt.ID)
}
