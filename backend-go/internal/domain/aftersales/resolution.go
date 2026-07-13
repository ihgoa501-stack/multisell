package aftersales

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ResolutionRequested = "requested"
	ResolutionApproved  = "approved"
	ResolutionRejected  = "rejected"
	ResolutionSubmitted = "execution_submitted"
	ResolutionSucceeded = "succeeded"
	ResolutionFailed    = "failed"
)

// ResolutionCase separates a buyer/platform request, the Owner's decision and
// external execution evidence. It deliberately does not mutate order,
// inventory or financial ledgers; those consequences require their own facts.
type ResolutionCase struct {
	ID                int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID           int64      `gorm:"column:owner_id;not null;index" json:"owner_id"`
	AfterSalesID      *int64     `gorm:"column:after_sales_id" json:"after_sales_id,omitempty"`
	OrderID           int64      `gorm:"column:order_id;not null;index" json:"order_id"`
	PlatformAccountID int64      `gorm:"column:platform_account_id;not null" json:"platform_account_id"`
	Kind              string     `gorm:"column:kind;not null" json:"kind"`
	RequestedMinor    int64      `gorm:"column:requested_minor;not null" json:"requested_minor"`
	Currency          string     `gorm:"column:currency;not null" json:"currency"`
	Reason            string     `gorm:"column:reason;not null" json:"reason"`
	RequestSource     string     `gorm:"column:request_source;not null" json:"request_source"`
	RequestEvidenceID string     `gorm:"column:request_evidence_id;not null" json:"request_evidence_id"`
	RequestObservedAt time.Time  `gorm:"column:request_observed_at;not null" json:"request_observed_at"`
	RequestKey        string     `gorm:"column:request_key;not null" json:"-"`
	Status            string     `gorm:"column:status;not null" json:"status"`
	DecisionReason    string     `gorm:"column:decision_reason" json:"decision_reason,omitempty"`
	DecisionKey       string     `gorm:"column:decision_key" json:"-"`
	DecidedBy         *int64     `gorm:"column:decided_by" json:"decided_by,omitempty"`
	DecidedAt         *time.Time `gorm:"column:decided_at" json:"decided_at,omitempty"`
	ExecutionKey      string     `gorm:"column:execution_key" json:"-"`
	ExternalRequestID string     `gorm:"column:external_request_id" json:"external_request_id,omitempty"`
	SubmittedAt       *time.Time `gorm:"column:submitted_at" json:"submitted_at,omitempty"`
	ConsequenceStatus string     `gorm:"column:consequence_status;not null" json:"consequence_status"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ResolutionCase) TableName() string { return "aftersales_resolution_case" }

type ResolutionReceipt struct {
	ID                int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID           int64           `gorm:"column:owner_id;not null;index" json:"owner_id"`
	ResolutionID      int64           `gorm:"column:resolution_id;not null;uniqueIndex" json:"resolution_id"`
	Outcome           string          `gorm:"column:outcome;not null" json:"outcome"`
	SourceType        string          `gorm:"column:source_type;not null" json:"source_type"`
	EvidenceID        string          `gorm:"column:evidence_id;not null" json:"evidence_id"`
	ExternalReceiptID string          `gorm:"column:external_receipt_id;not null" json:"external_receipt_id"`
	ObservedAt        time.Time       `gorm:"column:observed_at;not null" json:"observed_at"`
	ActualMinor       int64           `gorm:"column:actual_minor;not null" json:"actual_minor"`
	Currency          string          `gorm:"column:currency;not null" json:"currency"`
	FailureCode       string          `gorm:"column:failure_code" json:"failure_code,omitempty"`
	Payload           json.RawMessage `gorm:"column:receipt_payload;type:json" json:"receipt_payload"`
	PayloadSHA256     string          `gorm:"column:receipt_sha256;not null" json:"receipt_sha256"`
	RecordedAt        time.Time       `gorm:"column:recorded_at;autoCreateTime" json:"recorded_at"`
}

func (ResolutionReceipt) TableName() string { return "aftersales_resolution_receipt" }

type CreateResolutionInput struct {
	AfterSalesID      *int64    `json:"after_sales_id"`
	OrderID           int64     `json:"order_id" binding:"required"`
	PlatformAccountID int64     `json:"platform_account_id" binding:"required"`
	Kind              string    `json:"kind" binding:"required"`
	RequestedMinor    int64     `json:"requested_minor" binding:"required"`
	Currency          string    `json:"currency" binding:"required"`
	Reason            string    `json:"reason" binding:"required"`
	RequestSource     string    `json:"request_source" binding:"required"`
	RequestEvidenceID string    `json:"request_evidence_id" binding:"required"`
	ObservedAt        time.Time `json:"observed_at" binding:"required"`
	IdempotencyKey    string    `json:"idempotency_key" binding:"required"`
}

type ResolutionDecisionInput struct{ Decision, Reason, IdempotencyKey string }
type ResolutionExecutionInput struct{ ExternalRequestID, IdempotencyKey string }
type ResolutionReceiptInput struct {
	Outcome, SourceType, EvidenceID, ExternalReceiptID, Currency, FailureCode string
	ObservedAt                                                                time.Time
	ActualMinor                                                               int64
	Payload                                                                   json.RawMessage
	PayloadSHA256                                                             string
}

type ResolutionDetail struct {
	Case    ResolutionCase     `json:"case"`
	Receipt *ResolutionReceipt `json:"receipt,omitempty"`
}

type resolutionOrderAuthority struct {
	ID, OwnerID, AccountID, NormalizedOrderID int64
	TruthStatus, ProcessingStatus             string
}

func (resolutionOrderAuthority) TableName() string { return "platform_order_ingest" }

func (s *Service) CreateResolution(ownerID int64, in *CreateResolutionInput) (*ResolutionCase, error) {
	if ownerID <= 0 || in.OrderID <= 0 || in.PlatformAccountID <= 0 || in.RequestedMinor <= 0 || in.ObservedAt.IsZero() || strings.TrimSpace(in.IdempotencyKey) == "" {
		return nil, fmt.Errorf("invalid resolution request")
	}
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Kind != "refund" && in.Kind != "return" && in.Kind != "dispute" {
		return nil, fmt.Errorf("invalid resolution kind")
	}
	if len(in.Currency) != 3 || (in.RequestSource != "platform_request" && in.RequestSource != "buyer_request") {
		return nil, fmt.Errorf("invalid request evidence")
	}
	var authorityCount int64
	if err := s.db.Model(&resolutionOrderAuthority{}).Where("owner_id=? AND account_id=? AND normalized_order_id=? AND truth_status=? AND processing_status=?", ownerID, in.PlatformAccountID, in.OrderID, "external_observed", "applied").Count(&authorityCount).Error; err != nil {
		return nil, err
	}
	if authorityCount != 1 {
		return nil, fmt.Errorf("order and platform account are not backed by one applied external Owner fact")
	}
	var existing ResolutionCase
	if err := s.db.Where("owner_id = ? AND request_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; err == nil {
		if existing.OrderID != in.OrderID || existing.PlatformAccountID != in.PlatformAccountID || existing.Kind != in.Kind || existing.RequestedMinor != in.RequestedMinor || existing.Currency != in.Currency || existing.RequestEvidenceID != in.RequestEvidenceID {
			return nil, fmt.Errorf("idempotency key payload conflict")
		}
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var evidenceCount int64
	if err := s.db.Model(&ResolutionCase{}).Where("owner_id=? AND request_source=? AND request_evidence_id=?", ownerID, in.RequestSource, in.RequestEvidenceID).Count(&evidenceCount).Error; err != nil {
		return nil, err
	}
	if evidenceCount > 0 {
		return nil, fmt.Errorf("request evidence identity already used")
	}
	c := ResolutionCase{OwnerID: ownerID, AfterSalesID: in.AfterSalesID, OrderID: in.OrderID, PlatformAccountID: in.PlatformAccountID, Kind: in.Kind, RequestedMinor: in.RequestedMinor, Currency: in.Currency, Reason: in.Reason, RequestSource: in.RequestSource, RequestEvidenceID: in.RequestEvidenceID, RequestObservedAt: in.ObservedAt.UTC(), RequestKey: in.IdempotencyKey, Status: ResolutionRequested, ConsequenceStatus: "deferred"}
	if err := s.db.Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) DecideResolution(ownerID, actorID, id int64, in ResolutionDecisionInput) (*ResolutionCase, error) {
	if actorID != ownerID || (in.Decision != ResolutionApproved && in.Decision != ResolutionRejected) || strings.TrimSpace(in.Reason) == "" {
		return nil, fmt.Errorf("invalid Owner decision")
	}
	var out ResolutionCase
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var c ResolutionCase
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ?", id, ownerID).First(&c).Error; err != nil {
			return err
		}
		if c.Status == in.Decision && c.DecisionReason == in.Reason && c.DecisionKey == in.IdempotencyKey {
			out = c
			return nil
		}
		if strings.TrimSpace(in.IdempotencyKey) == "" || c.Status != ResolutionRequested {
			return fmt.Errorf("resolution is not awaiting Owner decision")
		}
		now := time.Now().UTC()
		if err := tx.Model(&c).Updates(map[string]any{"status": in.Decision, "decision_reason": in.Reason, "decision_key": in.IdempotencyKey, "decided_by": actorID, "decided_at": now}).Error; err != nil {
			return err
		}
		return tx.First(&out, c.ID).Error
	})
	return &out, err
}

func (s *Service) SubmitResolution(ownerID, id int64, in ResolutionExecutionInput) (*ResolutionCase, error) {
	if strings.TrimSpace(in.IdempotencyKey) == "" || strings.TrimSpace(in.ExternalRequestID) == "" {
		return nil, fmt.Errorf("external request evidence required")
	}
	var c ResolutionCase
	if err := s.db.Where("id = ? AND owner_id = ?", id, ownerID).First(&c).Error; err != nil {
		return nil, err
	}
	if c.Status == ResolutionSubmitted && c.ExecutionKey == in.IdempotencyKey && c.ExternalRequestID == in.ExternalRequestID {
		return &c, nil
	}
	if c.Status != ResolutionApproved {
		return nil, fmt.Errorf("resolution is not Owner-approved")
	}
	now := time.Now().UTC()
	res := s.db.Model(&ResolutionCase{}).Where("id=? AND owner_id=? AND status=?", id, ownerID, ResolutionApproved).Updates(map[string]any{"status": ResolutionSubmitted, "execution_key": in.IdempotencyKey, "external_request_id": in.ExternalRequestID, "submitted_at": now})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected != 1 {
		return nil, fmt.Errorf("concurrent resolution transition")
	}
	s.db.First(&c, id)
	return &c, nil
}

func (s *Service) RecordResolutionReceipt(ownerID, id int64, in ResolutionReceiptInput) (*ResolutionCase, error) {
	if in.Outcome != "succeeded" && in.Outcome != "failed" {
		return nil, fmt.Errorf("invalid receipt outcome")
	}
	if in.SourceType != "platform_receipt" && in.SourceType != "controlled_reconciliation" {
		return nil, fmt.Errorf("untrusted receipt source")
	}
	if in.ObservedAt.IsZero() || strings.TrimSpace(in.EvidenceID) == "" || strings.TrimSpace(in.ExternalReceiptID) == "" {
		return nil, fmt.Errorf("receipt evidence required")
	}
	sum := sha256.Sum256(in.Payload)
	if hex.EncodeToString(sum[:]) != strings.ToLower(in.PayloadSHA256) {
		return nil, fmt.Errorf("receipt SHA-256 mismatch")
	}
	in.Currency = strings.ToUpper(in.Currency)
	var out ResolutionCase
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var c ResolutionCase
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", id, ownerID).First(&c).Error; err != nil {
			return err
		}
		var old ResolutionReceipt
		if err := tx.Where("resolution_id=? AND owner_id=?", id, ownerID).First(&old).Error; err == nil {
			if old.EvidenceID == in.EvidenceID && old.PayloadSHA256 == strings.ToLower(in.PayloadSHA256) {
				out = c
				return nil
			}
			return fmt.Errorf("terminal receipt is immutable")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if c.Status != ResolutionSubmitted {
			return fmt.Errorf("execution was not submitted")
		}
		if in.Currency != c.Currency || in.ActualMinor < 0 || (in.Outcome == "succeeded" && in.ActualMinor != c.RequestedMinor) || (in.Outcome == "failed" && strings.TrimSpace(in.FailureCode) == "") {
			return fmt.Errorf("receipt amount, currency or failure evidence invalid")
		}
		r := ResolutionReceipt{OwnerID: ownerID, ResolutionID: id, Outcome: in.Outcome, SourceType: in.SourceType, EvidenceID: in.EvidenceID, ExternalReceiptID: in.ExternalReceiptID, ObservedAt: in.ObservedAt.UTC(), ActualMinor: in.ActualMinor, Currency: in.Currency, FailureCode: in.FailureCode, Payload: in.Payload, PayloadSHA256: strings.ToLower(in.PayloadSHA256)}
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
		if err := tx.Model(&c).Update("status", in.Outcome).Error; err != nil {
			return err
		}
		return tx.First(&out, id).Error
	})
	return &out, err
}

func (s *Service) GetResolution(ownerID, id int64) (*ResolutionCase, error) {
	var c ResolutionCase
	err := s.db.Where("id=? AND owner_id=?", id, ownerID).First(&c).Error
	return &c, err
}

func (s *Service) GetResolutionDetail(ownerID, id int64) (*ResolutionDetail, error) {
	c, err := s.GetResolution(ownerID, id)
	if err != nil {
		return nil, err
	}
	detail := &ResolutionDetail{Case: *c}
	var receipt ResolutionReceipt
	if err := s.db.Where("resolution_id=? AND owner_id=?", id, ownerID).First(&receipt).Error; err == nil {
		detail.Receipt = &receipt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return detail, nil
}
