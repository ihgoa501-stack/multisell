package finance

import (
	"bytes"
	"context"
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

var (
	ErrCashValidation          = errors.New("cash authority validation failed")
	ErrCashNotFound            = errors.New("cash authority object not found")
	ErrCashIdempotencyConflict = errors.New("cash authority idempotency conflict")
	ErrCashObjectConflict      = errors.New("cash authority object conflict")
)

// CashReceipt is an immutable bank/payment-provider observation. It is not a
// settlement receivable and does not become reconciled merely because it was ingested.
type CashReceipt struct {
	ID                   int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID              int64           `gorm:"column:owner_id;not null;index" json:"owner_id"`
	FinanceAccountID     int64           `gorm:"column:finance_account_id;not null;index" json:"finance_account_id"`
	SourceType           string          `gorm:"column:source_type;not null" json:"source_type"`
	ExternalReceiptID    string          `gorm:"column:external_receipt_id;not null" json:"external_receipt_id"`
	IdempotencyKey       string          `gorm:"column:idempotency_key;not null" json:"idempotency_key"`
	RequestSHA256        string          `gorm:"column:request_sha256;not null" json:"request_sha256"`
	AmountMinor          int64           `gorm:"column:amount_minor;not null" json:"amount_minor"`
	Currency             string          `gorm:"column:currency;not null" json:"currency"`
	ObservedAt           time.Time       `gorm:"column:observed_at;not null" json:"observed_at"`
	ValueDate            *time.Time      `gorm:"column:value_date" json:"value_date,omitempty"`
	RawPayload           json.RawMessage `gorm:"column:raw_payload;type:blob;not null" json:"raw_payload"`
	RawPayloadSHA256     string          `gorm:"column:raw_payload_sha256;not null" json:"raw_payload_sha256"`
	TruthStatus          string          `gorm:"column:truth_status;not null" json:"truth_status"`
	ReconciliationStatus string          `gorm:"column:reconciliation_status;not null" json:"reconciliation_status"`
	CreatedAt            time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CashReceipt) TableName() string { return "cash_receipt" }

// CashReconciliation allocates one actual receipt to exactly one authoritative
// platform settlement ingest. Status is computed by the server from all allocations.
type CashReconciliation struct {
	ID                         int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID                    int64      `gorm:"column:owner_id;not null;index" json:"owner_id"`
	CashReceiptID              int64      `gorm:"column:cash_receipt_id;not null;index" json:"cash_receipt_id"`
	PlatformSettlementIngestID int64      `gorm:"column:platform_settlement_ingest_id;not null;index" json:"platform_settlement_ingest_id"`
	IdempotencyKey             string     `gorm:"column:idempotency_key;not null" json:"idempotency_key"`
	RequestSHA256              string     `gorm:"column:request_sha256;not null" json:"request_sha256"`
	AmountMinor                int64      `gorm:"column:amount_minor;not null" json:"amount_minor"`
	Currency                   string     `gorm:"column:currency;not null" json:"currency"`
	ExpectedReceivableMinor    int64      `gorm:"column:expected_receivable_minor;not null" json:"expected_receivable_minor"`
	Status                     string     `gorm:"column:status;not null" json:"status"`
	ConflictReason             string     `gorm:"column:conflict_reason;not null" json:"conflict_reason,omitempty"`
	ReconciledAt               *time.Time `gorm:"column:reconciled_at" json:"reconciled_at,omitempty"`
	CreatedAt                  time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CashReconciliation) TableName() string { return "cash_reconciliation" }

type CreateCashReceiptInput struct {
	FinanceAccountID  int64           `json:"finance_account_id"`
	SourceType        string          `json:"source_type"`
	ExternalReceiptID string          `json:"external_receipt_id"`
	IdempotencyKey    string          `json:"idempotency_key"`
	AmountMinor       int64           `json:"amount_minor"`
	Currency          string          `json:"currency"`
	ObservedAt        time.Time       `json:"observed_at"`
	ValueDate         *time.Time      `json:"value_date"`
	RawPayload        json.RawMessage `json:"raw_payload"`
}

type CreateCashReconciliationInput struct {
	CashReceiptID              int64  `json:"cash_receipt_id"`
	PlatformSettlementIngestID int64  `json:"platform_settlement_ingest_id"`
	IdempotencyKey             string `json:"idempotency_key"`
	AmountMinor                int64  `json:"amount_minor"`
}

type platformSettlementCashView struct {
	ID          int64
	OwnerID     int64
	Currency    string
	TruthStatus string
}

func normalizeCurrency(value string) string { return strings.ToUpper(strings.TrimSpace(value)) }

func validCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func rawJSONHash(raw json.RawMessage) (string, error) {
	if len(bytes.TrimSpace(raw)) == 0 || !json.Valid(raw) {
		return "", fmt.Errorf("%w: raw_payload must be valid JSON", ErrCashValidation)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func requestHash(value any) string {
	b, _ := json.Marshal(value)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Service) CreateCashReceipt(ctx context.Context, ownerID int64, in CreateCashReceiptInput) (*CashReceipt, bool, error) {
	in.SourceType = strings.TrimSpace(in.SourceType)
	in.ExternalReceiptID = strings.TrimSpace(in.ExternalReceiptID)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Currency = normalizeCurrency(in.Currency)
	if ownerID <= 0 || in.FinanceAccountID <= 0 || (in.SourceType != "bank" && in.SourceType != "payment") || in.ExternalReceiptID == "" || in.IdempotencyKey == "" || len(in.IdempotencyKey) > 200 || in.AmountMinor <= 0 || !validCurrency(in.Currency) || in.ObservedAt.IsZero() {
		return nil, false, ErrCashValidation
	}
	payloadHash, err := rawJSONHash(in.RawPayload)
	if err != nil {
		return nil, false, err
	}
	if in.ValueDate != nil {
		d := time.Date(in.ValueDate.UTC().Year(), in.ValueDate.UTC().Month(), in.ValueDate.UTC().Day(), 0, 0, 0, 0, time.UTC)
		in.ValueDate = &d
	}
	sig := requestHash(struct {
		Account                                int64
		Source, External                       string
		Amount                                 int64
		Currency, Observed, ValueDate, Payload string
	}{
		in.FinanceAccountID, in.SourceType, in.ExternalReceiptID, in.AmountMinor, in.Currency, in.ObservedAt.UTC().Format(time.RFC3339Nano), formatDate(in.ValueDate), payloadHash,
	})
	var out CashReceipt
	replayed := false
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing CashReceipt
		if e := tx.Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; e == nil {
			if existing.RequestSHA256 != sig {
				return ErrCashIdempotencyConflict
			}
			out = existing
			replayed = true
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var account FinanceAccount
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ? AND status = 'active'", in.FinanceAccountID, ownerID).First(&account).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return ErrCashNotFound
			}
			return e
		}
		if account.AccountType != "bank" && account.AccountType != "payment" {
			return fmt.Errorf("%w: receipt account must be bank or payment", ErrCashValidation)
		}
		if normalizeCurrency(account.Currency) != in.Currency {
			return fmt.Errorf("%w: account currency mismatch", ErrCashObjectConflict)
		}
		var sameExternal CashReceipt
		if e := tx.Where("owner_id = ? AND source_type = ? AND finance_account_id = ? AND external_receipt_id = ?", ownerID, in.SourceType, in.FinanceAccountID, in.ExternalReceiptID).First(&sameExternal).Error; e == nil {
			if sameExternal.RequestSHA256 != sig {
				return ErrCashIdempotencyConflict
			}
			out, replayed = sameExternal, true
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		out = CashReceipt{OwnerID: ownerID, FinanceAccountID: in.FinanceAccountID, SourceType: in.SourceType, ExternalReceiptID: in.ExternalReceiptID, IdempotencyKey: in.IdempotencyKey, RequestSHA256: sig, AmountMinor: in.AmountMinor, Currency: in.Currency, ObservedAt: in.ObservedAt.UTC(), ValueDate: in.ValueDate, RawPayload: append(json.RawMessage(nil), in.RawPayload...), RawPayloadSHA256: payloadHash, TruthStatus: "external_observed", ReconciliationStatus: "unmatched"}
		created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&out)
		if created.Error != nil {
			return created.Error
		}
		if created.RowsAffected == 0 {
			var duplicate CashReceipt
			q := tx.Where("owner_id = ? AND (idempotency_key = ? OR (source_type = ? AND finance_account_id = ? AND external_receipt_id = ?))", ownerID, in.IdempotencyKey, in.SourceType, in.FinanceAccountID, in.ExternalReceiptID).First(&duplicate).Error
			if q != nil {
				return q
			}
			if duplicate.RequestSHA256 != sig {
				return ErrCashIdempotencyConflict
			}
			out, replayed = duplicate, true
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return &out, replayed, nil
}

func formatDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

func (s *Service) ListCashReceipts(ctx context.Context, ownerID int64) ([]CashReceipt, error) {
	var rows []CashReceipt
	return rows, s.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("id DESC").Find(&rows).Error
}

func (s *Service) CreateCashReconciliation(ctx context.Context, ownerID int64, in CreateCashReconciliationInput) (*CashReconciliation, bool, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if ownerID <= 0 || in.CashReceiptID <= 0 || in.PlatformSettlementIngestID <= 0 || in.AmountMinor <= 0 || in.IdempotencyKey == "" || len(in.IdempotencyKey) > 200 {
		return nil, false, ErrCashValidation
	}
	sig := requestHash(in)
	var out CashReconciliation
	replayed := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing CashReconciliation
		if e := tx.Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; e == nil {
			if existing.RequestSHA256 != sig {
				return ErrCashIdempotencyConflict
			}
			out, replayed = existing, true
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var receipt CashReceipt
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ?", in.CashReceiptID, ownerID).First(&receipt).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return ErrCashNotFound
			}
			return e
		}
		var settlement platformSettlementCashView
		if e := tx.Table("platform_settlement_ingest").Select("id, owner_id, currency, truth_status").Where("id = ? AND owner_id = ?", in.PlatformSettlementIngestID, ownerID).Take(&settlement).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return ErrCashNotFound
			}
			return e
		}
		if settlement.TruthStatus != "external_observed" {
			return fmt.Errorf("%w: settlement is not externally observed", ErrCashObjectConflict)
		}
		settlement.Currency = normalizeCurrency(settlement.Currency)
		if receipt.Currency != settlement.Currency {
			return fmt.Errorf("%w: receipt and settlement currency differ", ErrCashObjectConflict)
		}
		var foreignCurrencyLines int64
		if e := tx.Table("platform_settlement_fact_line").Where("ingest_id = ? AND currency <> ?", settlement.ID, settlement.Currency).Count(&foreignCurrencyLines).Error; e != nil {
			return e
		}
		if foreignCurrencyLines > 0 {
			return fmt.Errorf("%w: settlement contains mixed currency lines", ErrCashObjectConflict)
		}
		var expected struct{ Total int64 }
		if e := tx.Table("platform_settlement_fact_line").Select("COALESCE(SUM(CASE WHEN kind = 'sale' THEN amount_minor ELSE -amount_minor END),0) AS total").Where("ingest_id = ? AND currency = ?", settlement.ID, settlement.Currency).Scan(&expected).Error; e != nil {
			return e
		}
		if expected.Total <= 0 {
			return fmt.Errorf("%w: settlement has no positive receivable", ErrCashObjectConflict)
		}
		var receiptAllocated struct{ Total int64 }
		if e := tx.Model(&CashReconciliation{}).Select("COALESCE(SUM(amount_minor),0) AS total").Where("owner_id = ? AND cash_receipt_id = ? AND status <> 'conflict'", ownerID, receipt.ID).Scan(&receiptAllocated).Error; e != nil {
			return e
		}
		if receiptAllocated.Total+in.AmountMinor > receipt.AmountMinor {
			return fmt.Errorf("%w: receipt allocation exceeds observed amount", ErrCashObjectConflict)
		}
		var settledAllocated struct{ Total int64 }
		if e := tx.Model(&CashReconciliation{}).Select("COALESCE(SUM(amount_minor),0) AS total").Where("owner_id = ? AND platform_settlement_ingest_id = ? AND status <> 'conflict'", ownerID, settlement.ID).Scan(&settledAllocated).Error; e != nil {
			return e
		}
		newTotal := settledAllocated.Total + in.AmountMinor
		var otherObjectCount int64
		if e := tx.Model(&CashReconciliation{}).Where("owner_id = ? AND cash_receipt_id = ? AND platform_settlement_ingest_id <> ? AND status <> 'conflict'", ownerID, receipt.ID, settlement.ID).Count(&otherObjectCount).Error; e != nil {
			return e
		}
		if otherObjectCount > 0 {
			return fmt.Errorf("%w: one receipt cannot be allocated across settlement objects", ErrCashObjectConflict)
		}
		status, reason := "partial", ""
		if newTotal == expected.Total {
			status = "reconciled"
		} else if newTotal > expected.Total {
			status, reason = "conflict", "allocation exceeds settlement receivable"
		}
		out = CashReconciliation{OwnerID: ownerID, CashReceiptID: receipt.ID, PlatformSettlementIngestID: settlement.ID, IdempotencyKey: in.IdempotencyKey, RequestSHA256: sig, AmountMinor: in.AmountMinor, Currency: receipt.Currency, ExpectedReceivableMinor: expected.Total, Status: status, ConflictReason: reason}
		if status == "reconciled" {
			now := time.Now().UTC()
			out.ReconciledAt = &now
		}
		if e := tx.Create(&out).Error; e != nil {
			return e
		}
		var receiptUsed struct{ Total int64 }
		if e := tx.Model(&CashReconciliation{}).Select("COALESCE(SUM(amount_minor),0) AS total").Where("owner_id = ? AND cash_receipt_id = ? AND status <> 'conflict'", ownerID, receipt.ID).Scan(&receiptUsed).Error; e != nil {
			return e
		}
		receiptStatus := "partial"
		if status == "conflict" {
			receiptStatus = "conflict"
		} else if receiptUsed.Total == receipt.AmountMinor && status == "reconciled" {
			receiptStatus = "reconciled"
		}
		return tx.Model(&CashReceipt{}).Where("id = ? AND owner_id = ?", receipt.ID, ownerID).Update("reconciliation_status", receiptStatus).Error
	})
	if err != nil {
		return nil, false, err
	}
	return &out, replayed, nil
}

func (s *Service) ListCashReconciliations(ctx context.Context, ownerID int64) ([]CashReconciliation, error) {
	var rows []CashReconciliation
	return rows, s.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("id DESC").Find(&rows).Error
}
