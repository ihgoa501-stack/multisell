package settlement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	SettlementTruthExternalObserved = "external_observed"
	SettlementTruthMock             = "mock"
)

var settlementLineKinds = map[string]bool{
	"sale": true, "fee": true, "refund": true, "commission": true,
}
var settlementFeeCodes = map[string]bool{
	"platform_fee": true, "payment_fee": true, "tax_fee": true,
	"fulfillment_fee": true, "advertising_fee": true, "other_fee": true,
}

// PlatformSettlementIngest is the immutable receipt for one platform settlement event.
// RawPayload is deliberately bytes rather than JSONB: the server digest describes the
// exact bytes presented by the authenticated connector, not a database-normalized value.
type PlatformSettlementIngest struct {
	ID                   int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID              int64     `gorm:"column:owner_id;not null" json:"owner_id"`
	AccountID            int64     `gorm:"column:account_id;not null" json:"account_id"`
	PlatformCode         string    `gorm:"column:platform_code;not null" json:"platform_code"`
	ExternalEventID      string    `gorm:"column:external_event_id;not null" json:"external_event_id"`
	ExternalSettlementID string    `gorm:"column:external_settlement_id;not null" json:"external_settlement_id"`
	TruthStatus          string    `gorm:"column:truth_status;not null" json:"truth_status"`
	Currency             string    `gorm:"column:currency;size:3;not null" json:"currency"`
	RawPayload           []byte    `gorm:"column:raw_payload;not null" json:"raw_payload"`
	PayloadSHA256        string    `gorm:"column:payload_sha256;size:64;not null" json:"payload_sha256"`
	ContentSHA256        string    `gorm:"column:content_sha256;size:64;not null" json:"content_sha256"`
	ObservedAt           time.Time `gorm:"column:observed_at;not null" json:"observed_at"`
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PlatformSettlementIngest) TableName() string { return "platform_settlement_ingest" }

type PlatformSettlementFactLine struct {
	ID                  int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	IngestID            int64     `gorm:"column:ingest_id;not null" json:"ingest_id"`
	LineNumber          int       `gorm:"column:line_number;not null" json:"line_number"`
	ExternalLineID      string    `gorm:"column:external_line_id;not null" json:"external_line_id"`
	ExternalOrderID     string    `gorm:"column:external_order_id;not null" json:"external_order_id"`
	OrderID             int64     `gorm:"column:order_id;not null" json:"order_id"`
	Kind                string    `gorm:"column:kind;not null" json:"kind"`
	FeeCode             string    `gorm:"column:fee_code;not null" json:"fee_code"`
	AmountMinor         int64     `gorm:"column:amount_minor;not null" json:"amount_minor"`
	Currency            string    `gorm:"column:currency;size:3;not null" json:"currency"`
	ExternalTransaction string    `gorm:"column:external_transaction_id;not null" json:"external_transaction_id"`
	OccurredAt          time.Time `gorm:"column:occurred_at;not null" json:"occurred_at"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PlatformSettlementFactLine) TableName() string { return "platform_settlement_fact_line" }

type PlatformSettlementLineInput struct {
	ExternalLineID      string    `json:"external_line_id" binding:"required"`
	ExternalOrderID     string    `json:"external_order_id" binding:"required"`
	Kind                string    `json:"kind" binding:"required"`
	FeeCode             string    `json:"fee_code"`
	AmountMinor         int64     `json:"amount_minor"`
	Currency            string    `json:"currency" binding:"required"`
	ExternalTransaction string    `json:"external_transaction_id" binding:"required"`
	OccurredAt          time.Time `json:"occurred_at" binding:"required"`
}

type IngestPlatformSettlementInput struct {
	OwnerID              int64                         `json:"-"`
	AccountID            int64                         `json:"-"`
	PlatformCode         string                        `json:"platform_code" binding:"required"`
	ExternalEventID      string                        `json:"external_event_id" binding:"required"`
	ExternalSettlementID string                        `json:"external_settlement_id" binding:"required"`
	TruthStatus          string                        `json:"truth_status" binding:"required"`
	Currency             string                        `json:"currency" binding:"required"`
	ObservedAt           time.Time                     `json:"observed_at" binding:"required"`
	RawPayload           json.RawMessage               `json:"raw_payload" binding:"required"`
	Lines                []PlatformSettlementLineInput `json:"lines" binding:"required,min=1"`
}

type IngestPlatformSettlementResult struct {
	IngestID int64 `json:"ingest_id"`
	Replay   bool  `json:"replay"`
}

type PlatformSettlementFactDetail struct {
	Ingest PlatformSettlementIngest     `json:"ingest"`
	Lines  []PlatformSettlementFactLine `json:"lines"`
}

func settlementContentDigest(in IngestPlatformSettlementInput) (string, error) {
	canonical := struct {
		PlatformCode, ExternalEventID, ExternalSettlementID, TruthStatus, Currency string
		ObservedAt                                                                 time.Time
		Lines                                                                      []PlatformSettlementLineInput
	}{in.PlatformCode, in.ExternalEventID, in.ExternalSettlementID, in.TruthStatus, in.Currency, in.ObservedAt.UTC(), in.Lines}
	b, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// IngestPlatformSettlement validates Owner/account/order authority and appends
// the receipt plus all normalized lines atomically. Reusing an event identity is
// a replay only when both the raw bytes and normalized content are identical.
func (s *Service) IngestPlatformSettlement(ctx context.Context, in IngestPlatformSettlementInput) (*IngestPlatformSettlementResult, error) {
	if in.OwnerID <= 0 || in.AccountID <= 0 {
		return nil, errors.New("owner and platform account are required")
	}
	in.PlatformCode = strings.TrimSpace(in.PlatformCode)
	in.ExternalEventID = strings.TrimSpace(in.ExternalEventID)
	in.ExternalSettlementID = strings.TrimSpace(in.ExternalSettlementID)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.PlatformCode == "" || in.ExternalEventID == "" || in.ExternalSettlementID == "" || len(in.Currency) != 3 || in.ObservedAt.IsZero() || len(in.RawPayload) == 0 || len(in.Lines) == 0 {
		return nil, errors.New("platform/event/settlement identity, currency, observation, raw payload and lines are required")
	}
	if in.TruthStatus != SettlementTruthExternalObserved && in.TruthStatus != SettlementTruthMock {
		return nil, errors.New("truth_status must be external_observed or mock")
	}
	seenLines, seenTransactions := map[string]bool{}, map[string]bool{}
	for i := range in.Lines {
		line := &in.Lines[i]
		line.ExternalLineID = strings.TrimSpace(line.ExternalLineID)
		line.ExternalOrderID = strings.TrimSpace(line.ExternalOrderID)
		line.Kind = strings.ToLower(strings.TrimSpace(line.Kind))
		line.FeeCode = strings.ToLower(strings.TrimSpace(line.FeeCode))
		line.Currency = strings.ToUpper(strings.TrimSpace(line.Currency))
		line.ExternalTransaction = strings.TrimSpace(line.ExternalTransaction)
		if line.ExternalLineID == "" || line.ExternalOrderID == "" || !settlementLineKinds[line.Kind] || line.AmountMinor < 0 || line.Currency != in.Currency || line.ExternalTransaction == "" || line.OccurredAt.IsZero() {
			return nil, fmt.Errorf("invalid settlement line %d", i+1)
		}
		if (line.Kind == "fee" || line.Kind == "commission") != settlementFeeCodes[line.FeeCode] {
			return nil, fmt.Errorf("line %d fee_code must be an allowed value exactly for fee/commission", i+1)
		}
		if seenLines[line.ExternalLineID] || seenTransactions[line.ExternalTransaction] {
			return nil, fmt.Errorf("duplicate line or transaction identity at line %d", i+1)
		}
		seenLines[line.ExternalLineID], seenTransactions[line.ExternalTransaction] = true, true
	}
	payloadSum := sha256.Sum256(in.RawPayload)
	payloadDigest := hex.EncodeToString(payloadSum[:])
	contentDigest, err := settlementContentDigest(in)
	if err != nil {
		return nil, err
	}
	result := &IngestPlatformSettlementResult{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var authority int64
		if err := tx.Table("owner_platform_account_authority").Select("account_id").Where("owner_id=? AND account_id=? AND platform_code=?", in.OwnerID, in.AccountID, in.PlatformCode).Take(&authority).Error; err != nil {
			return fmt.Errorf("platform account is not verified for owner: %w", err)
		}
		var existing PlatformSettlementIngest
		err := tx.Where("owner_id=? AND account_id=? AND external_event_id=?", in.OwnerID, in.AccountID, in.ExternalEventID).First(&existing).Error
		if err == nil {
			if existing.PayloadSHA256 != payloadDigest || existing.ContentSHA256 != contentDigest {
				return errors.New("external settlement event id was reused with different content")
			}
			result.IngestID, result.Replay = existing.ID, true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		receipt := PlatformSettlementIngest{OwnerID: in.OwnerID, AccountID: in.AccountID, PlatformCode: in.PlatformCode, ExternalEventID: in.ExternalEventID, ExternalSettlementID: in.ExternalSettlementID, TruthStatus: in.TruthStatus, Currency: in.Currency, RawPayload: append([]byte(nil), in.RawPayload...), PayloadSHA256: payloadDigest, ContentSHA256: contentDigest, ObservedAt: in.ObservedAt.UTC()}
		if err := tx.Create(&receipt).Error; err != nil {
			return err
		}
		for i, line := range in.Lines {
			var orderFact struct{ NormalizedOrderID *int64 }
			if err := tx.Table("platform_order_ingest").Select("normalized_order_id").Where("owner_id=? AND account_id=? AND external_order_id=? AND event_action='reserve' AND processing_status='applied' AND truth_status=?", in.OwnerID, in.AccountID, line.ExternalOrderID, in.TruthStatus).Take(&orderFact).Error; err != nil {
				return fmt.Errorf("line %d does not bind to an authoritative order of the same truth class: %w", i+1, err)
			}
			if orderFact.NormalizedOrderID == nil {
				return fmt.Errorf("line %d authoritative order has no normalized identity", i+1)
			}
			fact := PlatformSettlementFactLine{IngestID: receipt.ID, LineNumber: i + 1, ExternalLineID: line.ExternalLineID, ExternalOrderID: line.ExternalOrderID, OrderID: *orderFact.NormalizedOrderID, Kind: line.Kind, FeeCode: line.FeeCode, AmountMinor: line.AmountMinor, Currency: line.Currency, ExternalTransaction: line.ExternalTransaction, OccurredAt: line.OccurredAt.UTC()}
			if err := tx.Create(&fact).Error; err != nil {
				return err
			}
		}
		result.IngestID = receipt.ID
		return nil
	})
	// A concurrent identical request can lose the unique-key race after both
	// transactions observed no row. Resolve it as a replay only after comparing
	// both server digests; never mask a same-key/different-content conflict.
	if err != nil {
		var existing PlatformSettlementIngest
		if lookupErr := s.db.WithContext(ctx).Where("owner_id=? AND account_id=? AND external_event_id=?", in.OwnerID, in.AccountID, in.ExternalEventID).First(&existing).Error; lookupErr == nil {
			if existing.PayloadSHA256 == payloadDigest && existing.ContentSHA256 == contentDigest {
				result.IngestID, result.Replay = existing.ID, true
				return result, nil
			}
			return nil, errors.New("external settlement event id was reused with different content")
		}
	}
	return result, err
}

func (s *Service) GetPlatformSettlementFact(ctx context.Context, ownerID, id int64) (*PlatformSettlementFactDetail, error) {
	var out PlatformSettlementFactDetail
	if err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", id, ownerID).First(&out.Ingest).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("ingest_id=?", id).Order("line_number").Find(&out.Lines).Error; err != nil {
		return nil, err
	}
	return &out, nil
}
