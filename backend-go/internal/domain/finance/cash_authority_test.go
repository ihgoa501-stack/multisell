package finance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

type cashTestSettlement struct {
	ID          int64 `gorm:"primaryKey"`
	OwnerID     int64
	Currency    string
	TruthStatus string
}

func (cashTestSettlement) TableName() string { return "platform_settlement_ingest" }

type cashTestSettlementLine struct {
	ID          int64 `gorm:"primaryKey"`
	IngestID    int64
	Kind        string
	AmountMinor int64
	Currency    string
}

func (cashTestSettlementLine) TableName() string { return "platform_settlement_fact_line" }

func cashTestService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	db := dbtest.NewDB(t, &FinanceAccount{}, &CashReceipt{}, &CashReconciliation{}, &cashTestSettlement{}, &cashTestSettlementLine{})
	return NewService(db, zap.NewNop(), nil), context.Background()
}

func seedCashAccount(t *testing.T, svc *Service, owner int64, accountType, currency string) FinanceAccount {
	t.Helper()
	a := FinanceAccount{OwnerID: owner, Name: "actual account", AccountType: accountType, Currency: currency, Status: "active"}
	if err := svc.db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	return a
}

func receiptInput(account int64, key string, amount int64, currency string, payload json.RawMessage) CreateCashReceiptInput {
	return CreateCashReceiptInput{FinanceAccountID: account, SourceType: "bank", ExternalReceiptID: "bank-" + key, IdempotencyKey: key, AmountMinor: amount, Currency: currency, ObservedAt: time.Date(2026, 7, 12, 8, 30, 0, 0, time.UTC), RawPayload: payload}
}

func TestCashReceiptPreservesRawPayloadAndEnforcesIdempotency(t *testing.T) {
	svc, ctx := cashTestService(t)
	a := seedCashAccount(t, svc, 7, "bank", "USD")
	raw := json.RawMessage(`{ "external_id":"r-1", "amount":"12.34" }`)
	in := receiptInput(a.ID, "receive-1", 1234, "usd", raw)
	r, replay, err := svc.CreateCashReceipt(ctx, 7, in)
	if err != nil || replay {
		t.Fatalf("create err=%v replay=%v", err, replay)
	}
	sum := sha256.Sum256(raw)
	if r.RawPayloadSHA256 != hex.EncodeToString(sum[:]) || string(r.RawPayload) != string(raw) {
		t.Fatalf("raw payload/hash not preserved: %#v", r)
	}
	if r.ReconciliationStatus != "unmatched" || r.TruthStatus != "external_observed" || r.Currency != "USD" {
		t.Fatalf("unexpected receipt: %#v", r)
	}
	r2, replay, err := svc.CreateCashReceipt(ctx, 7, in)
	if err != nil || !replay || r2.ID != r.ID {
		t.Fatalf("exact replay err=%v replay=%v row=%#v", err, replay, r2)
	}
	in.AmountMinor++
	if _, _, err := svc.CreateCashReceipt(ctx, 7, in); !errors.Is(err, ErrCashIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestCashReceiptIsOwnerAccountAndCurrencyIsolated(t *testing.T) {
	svc, ctx := cashTestService(t)
	a := seedCashAccount(t, svc, 1, "bank", "EUR")
	in := receiptInput(a.ID, "x", 100, "EUR", json.RawMessage(`{"id":"x"}`))
	if _, _, err := svc.CreateCashReceipt(ctx, 2, in); !errors.Is(err, ErrCashNotFound) {
		t.Fatalf("cross-owner account accepted: %v", err)
	}
	in.Currency = "USD"
	if _, _, err := svc.CreateCashReceipt(ctx, 1, in); !errors.Is(err, ErrCashObjectConflict) {
		t.Fatalf("currency mismatch accepted: %v", err)
	}
	cash := seedCashAccount(t, svc, 1, "cash", "EUR")
	in.FinanceAccountID, in.Currency = cash.ID, "EUR"
	if _, _, err := svc.CreateCashReceipt(ctx, 1, in); !errors.Is(err, ErrCashValidation) {
		t.Fatalf("non-bank/payment account accepted: %v", err)
	}
}

func TestCashReconciliationRequiresSameOwnerCurrencyActualSettlementAndFullMatch(t *testing.T) {
	svc, ctx := cashTestService(t)
	a := seedCashAccount(t, svc, 9, "payment", "USD")
	r1, _, _ := svc.CreateCashReceipt(ctx, 9, receiptInput(a.ID, "r1", 600, "USD", json.RawMessage(`{"id":"r1"}`)))
	r2in := receiptInput(a.ID, "r2", 400, "USD", json.RawMessage(`{"id":"r2"}`))
	r2in.ExternalReceiptID = "bank-r2"
	r2, _, _ := svc.CreateCashReceipt(ctx, 9, r2in)
	st := cashTestSettlement{OwnerID: 9, Currency: "USD", TruthStatus: "external_observed"}
	if err := svc.db.Create(&st).Error; err != nil {
		t.Fatal(err)
	}
	lines := []cashTestSettlementLine{{IngestID: st.ID, Kind: "sale", AmountMinor: 1200, Currency: "USD"}, {IngestID: st.ID, Kind: "fee", AmountMinor: 100, Currency: "USD"}, {IngestID: st.ID, Kind: "refund", AmountMinor: 100, Currency: "USD"}}
	if err := svc.db.Create(&lines).Error; err != nil {
		t.Fatal(err)
	}
	one, replay, err := svc.CreateCashReconciliation(ctx, 9, CreateCashReconciliationInput{CashReceiptID: r1.ID, PlatformSettlementIngestID: st.ID, IdempotencyKey: "m1", AmountMinor: 600})
	if err != nil || replay || one.Status != "partial" || one.ExpectedReceivableMinor != 1000 {
		t.Fatalf("partial: row=%#v replay=%v err=%v", one, replay, err)
	}
	two, _, err := svc.CreateCashReconciliation(ctx, 9, CreateCashReconciliationInput{CashReceiptID: r2.ID, PlatformSettlementIngestID: st.ID, IdempotencyKey: "m2", AmountMinor: 400})
	if err != nil || two.Status != "reconciled" || two.ReconciledAt == nil {
		t.Fatalf("full: row=%#v err=%v", two, err)
	}
	var got CashReceipt
	if err := svc.db.First(&got, r2.ID).Error; err != nil || got.ReconciliationStatus != "reconciled" {
		t.Fatalf("receipt state: %#v err=%v", got, err)
	}
}

func TestCashReconciliationRejectsWrongObjectTruthAndCurrency(t *testing.T) {
	svc, ctx := cashTestService(t)
	a := seedCashAccount(t, svc, 5, "bank", "USD")
	r, _, _ := svc.CreateCashReceipt(ctx, 5, receiptInput(a.ID, "r", 500, "USD", json.RawMessage(`{"id":"r"}`)))
	mock := cashTestSettlement{OwnerID: 5, Currency: "USD", TruthStatus: "mock"}
	svc.db.Create(&mock)
	svc.db.Create(&cashTestSettlementLine{IngestID: mock.ID, Kind: "sale", AmountMinor: 500, Currency: "USD"})
	_, _, err := svc.CreateCashReconciliation(ctx, 5, CreateCashReconciliationInput{CashReceiptID: r.ID, PlatformSettlementIngestID: mock.ID, IdempotencyKey: "mock", AmountMinor: 500})
	if !errors.Is(err, ErrCashObjectConflict) {
		t.Fatalf("mock settlement accepted: %v", err)
	}
	eur := cashTestSettlement{OwnerID: 5, Currency: "EUR", TruthStatus: "external_observed"}
	svc.db.Create(&eur)
	svc.db.Create(&cashTestSettlementLine{IngestID: eur.ID, Kind: "sale", AmountMinor: 500, Currency: "EUR"})
	_, _, err = svc.CreateCashReconciliation(ctx, 5, CreateCashReconciliationInput{CashReceiptID: r.ID, PlatformSettlementIngestID: eur.ID, IdempotencyKey: "eur", AmountMinor: 500})
	if !errors.Is(err, ErrCashObjectConflict) {
		t.Fatalf("cross-currency accepted: %v", err)
	}
	actual := cashTestSettlement{OwnerID: 5, Currency: "USD", TruthStatus: "external_observed"}
	svc.db.Create(&actual)
	svc.db.Create(&cashTestSettlementLine{IngestID: actual.ID, Kind: "sale", AmountMinor: 500, Currency: "USD"})
	_, _, err = svc.CreateCashReconciliation(ctx, 6, CreateCashReconciliationInput{CashReceiptID: r.ID, PlatformSettlementIngestID: actual.ID, IdempotencyKey: "owner", AmountMinor: 500})
	if !errors.Is(err, ErrCashNotFound) {
		t.Fatalf("cross-owner receipt accepted: %v", err)
	}
}

func TestCashReconciliationPersistsAmountConflictWithoutCallingItReconciled(t *testing.T) {
	svc, ctx := cashTestService(t)
	a := seedCashAccount(t, svc, 3, "bank", "USD")
	r, _, _ := svc.CreateCashReceipt(ctx, 3, receiptInput(a.ID, "large", 1200, "USD", json.RawMessage(`{"id":"large"}`)))
	st := cashTestSettlement{OwnerID: 3, Currency: "USD", TruthStatus: "external_observed"}
	svc.db.Create(&st)
	svc.db.Create(&cashTestSettlementLine{IngestID: st.ID, Kind: "sale", AmountMinor: 1000, Currency: "USD"})
	row, _, err := svc.CreateCashReconciliation(ctx, 3, CreateCashReconciliationInput{CashReceiptID: r.ID, PlatformSettlementIngestID: st.ID, IdempotencyKey: "too-large", AmountMinor: 1200})
	if err != nil || row.Status != "conflict" || row.ReconciledAt != nil || row.ConflictReason == "" {
		t.Fatalf("expected persisted conflict, row=%#v err=%v", row, err)
	}
	var got CashReceipt
	svc.db.First(&got, r.ID)
	if got.ReconciliationStatus != "conflict" {
		t.Fatalf("receipt status=%s", got.ReconciliationStatus)
	}
}
