package settlement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type factAccount struct {
	ID         int64 `gorm:"primaryKey"`
	PlatformID int64
}

func (factAccount) TableName() string { return "platform_integration_account" }

type factAuthority struct {
	OwnerID      int64 `gorm:"primaryKey"`
	AccountID    int64 `gorm:"primaryKey"`
	PlatformCode string
}

func (factAuthority) TableName() string { return "owner_platform_account_authority" }

type factOrder struct {
	ID      int64 `gorm:"primaryKey"`
	OrderNo string
}

func (factOrder) TableName() string { return "sales_order" }

type factOrderIngest struct {
	ID                                                          int64 `gorm:"primaryKey"`
	OwnerID, AccountID                                          int64
	ExternalOrderID, EventAction, TruthStatus, ProcessingStatus string
	NormalizedOrderID                                           *int64
}

func (factOrderIngest) TableName() string { return "platform_order_ingest" }

func newPlatformFactService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&factAccount{}, &factAuthority{}, &factOrder{}, &factOrderIngest{}, &PlatformSettlementIngest{}, &PlatformSettlementFactLine{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&factAccount{ID: 10, PlatformID: 1})
	db.Create(&factAuthority{OwnerID: 42, AccountID: 10, PlatformCode: "ozon"})
	db.Create(&factOrder{ID: 20, OrderNo: "ozon:10:order-1"})
	orderID := int64(20)
	db.Create(&factOrderIngest{ID: 30, OwnerID: 42, AccountID: 10, ExternalOrderID: "order-1", EventAction: "reserve", TruthStatus: "external_observed", ProcessingStatus: "applied", NormalizedOrderID: &orderID})
	db.Create(&factOrder{ID: 21, OrderNo: "ozon:10:mock-order"})
	mockOrderID := int64(21)
	db.Create(&factOrderIngest{ID: 31, OwnerID: 42, AccountID: 10, ExternalOrderID: "mock-order", EventAction: "reserve", TruthStatus: "mock", ProcessingStatus: "applied", NormalizedOrderID: &mockOrderID})
	return NewService(db, zap.NewNop()), db
}

func validPlatformSettlementInput() IngestPlatformSettlementInput {
	now := time.Date(2026, 7, 12, 10, 30, 0, 0, time.UTC)
	return IngestPlatformSettlementInput{OwnerID: 42, AccountID: 10, PlatformCode: "ozon", ExternalEventID: "event-1", ExternalSettlementID: "settlement-1", TruthStatus: "external_observed", Currency: "rub", ObservedAt: now, RawPayload: json.RawMessage("{\n  \"event\": \"event-1\"\n}"), Lines: []PlatformSettlementLineInput{
		{ExternalLineID: "line-sale", ExternalOrderID: "order-1", Kind: "sale", AmountMinor: 10000, Currency: "RUB", ExternalTransaction: "txn-sale", OccurredAt: now},
		{ExternalLineID: "line-commission", ExternalOrderID: "order-1", Kind: "commission", FeeCode: "platform_fee", AmountMinor: 1500, Currency: "RUB", ExternalTransaction: "txn-commission", OccurredAt: now},
		{ExternalLineID: "line-refund", ExternalOrderID: "order-1", Kind: "refund", AmountMinor: 500, Currency: "RUB", ExternalTransaction: "txn-refund", OccurredAt: now},
	}}
}

func TestPlatformSettlementFactIngestBindsAuthorityAndPreservesExactPayload(t *testing.T) {
	svc, _ := newPlatformFactService(t)
	in := validPlatformSettlementInput()
	got, err := svc.IngestPlatformSettlement(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := svc.GetPlatformSettlementFact(context.Background(), 42, got.IngestID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Ingest.OwnerID != 42 || detail.Ingest.AccountID != 10 || detail.Ingest.TruthStatus != "external_observed" || len(detail.Lines) != 3 {
		t.Fatalf("wrong authority detail: %+v", detail)
	}
	if string(detail.Ingest.RawPayload) != string(in.RawPayload) {
		t.Fatalf("raw bytes changed: %q", detail.Ingest.RawPayload)
	}
	sum := sha256.Sum256(in.RawPayload)
	if detail.Ingest.PayloadSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("server payload digest mismatch")
	}
	for _, line := range detail.Lines {
		if line.OrderID != 20 || line.Currency != "RUB" {
			t.Fatalf("line not bound to exact order/currency: %+v", line)
		}
	}
}

func TestPlatformSettlementFactIdempotentReplayAndConflict(t *testing.T) {
	svc, db := newPlatformFactService(t)
	in := validPlatformSettlementInput()
	first, err := svc.IngestPlatformSettlement(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := svc.IngestPlatformSettlement(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replay || replay.IngestID != first.IngestID {
		t.Fatalf("not an exact replay: %+v", replay)
	}
	changed := in
	changed.Lines = append([]PlatformSettlementLineInput(nil), in.Lines...)
	changed.Lines[0].AmountMinor++
	if _, err = svc.IngestPlatformSettlement(context.Background(), changed); err == nil || !strings.Contains(err.Error(), "different content") {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
	var receipts, lines int64
	db.Model(&PlatformSettlementIngest{}).Count(&receipts)
	db.Model(&PlatformSettlementFactLine{}).Count(&lines)
	if receipts != 1 || lines != 3 {
		t.Fatalf("conflict caused partial writes: receipts=%d lines=%d", receipts, lines)
	}
}

func TestPlatformSettlementFactSeparatesTruthAndFailsAtomically(t *testing.T) {
	svc, db := newPlatformFactService(t)
	in := validPlatformSettlementInput()
	in.TruthStatus = "mock"
	if _, err := svc.IngestPlatformSettlement(context.Background(), in); err == nil || !strings.Contains(err.Error(), "same truth class") {
		t.Fatalf("mock settlement bound external order: %v", err)
	}
	in = validPlatformSettlementInput()
	in.Lines[1].ExternalOrderID = "missing"
	if _, err := svc.IngestPlatformSettlement(context.Background(), in); err == nil {
		t.Fatal("missing order should fail")
	}
	var count int64
	db.Model(&PlatformSettlementIngest{}).Count(&count)
	if count != 0 {
		t.Fatalf("failed batch left receipt: %d", count)
	}
}

func TestPlatformSettlementFactRejectsCrossOwnerAndInvalidMoneyClassification(t *testing.T) {
	svc, _ := newPlatformFactService(t)
	in := validPlatformSettlementInput()
	in.OwnerID = 99
	if _, err := svc.IngestPlatformSettlement(context.Background(), in); err == nil {
		t.Fatal("cross-owner account accepted")
	}
	in = validPlatformSettlementInput()
	in.Lines[0].AmountMinor = -1
	if _, err := svc.IngestPlatformSettlement(context.Background(), in); err == nil {
		t.Fatal("negative unsigned amount accepted")
	}
	in = validPlatformSettlementInput()
	in.Lines[1].FeeCode = ""
	if _, err := svc.IngestPlatformSettlement(context.Background(), in); err == nil {
		t.Fatal("unclassified commission accepted")
	}
	in = validPlatformSettlementInput()
	in.Lines[0].FeeCode = "platform_fee"
	if _, err := svc.IngestPlatformSettlement(context.Background(), in); err == nil {
		t.Fatal("sale with fee_code accepted")
	}
}

func TestPlatformSettlementFactOwnerReadIsolation(t *testing.T) {
	svc, _ := newPlatformFactService(t)
	got, err := svc.IngestPlatformSettlement(context.Background(), validPlatformSettlementInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetPlatformSettlementFact(context.Background(), 99, got.IngestID); err == nil {
		t.Fatal("cross-owner read succeeded")
	}
}
