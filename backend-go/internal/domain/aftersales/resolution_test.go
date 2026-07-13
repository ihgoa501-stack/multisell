package aftersales

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func resolutionDB(t *testing.T) *Service {
	db := newTestDB(t)
	if err := db.AutoMigrate(&ResolutionCase{}, &ResolutionReceipt{}, &resolutionOrderAuthority{}); err != nil {
		t.Fatal(err)
	}
	for _, a := range []resolutionOrderAuthority{{ID: 1, OwnerID: 7, AccountID: 20, NormalizedOrderID: 10, TruthStatus: "external_observed", ProcessingStatus: "applied"}, {ID: 2, OwnerID: 1, AccountID: 1, NormalizedOrderID: 1, TruthStatus: "external_observed", ProcessingStatus: "applied"}} {
		if err := db.Create(&a).Error; err != nil {
			t.Fatal(err)
		}
	}
	return newSvc(db)
}

func TestResolutionRequiresOwnerDecisionAndTrustedTerminalReceipt(t *testing.T) {
	s := resolutionDB(t)
	now := time.Now().UTC()
	c, err := s.CreateResolution(7, &CreateResolutionInput{OrderID: 10, PlatformAccountID: 20, Kind: "refund", RequestedMinor: 1234, Currency: "usd", Reason: "damaged", RequestSource: "buyer_request", RequestEvidenceID: "buyer-msg-1", ObservedAt: now, IdempotencyKey: "request-1"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != ResolutionRequested || c.ConsequenceStatus != "deferred" {
		t.Fatalf("unexpected request: %#v", c)
	}
	replay, err := s.CreateResolution(7, &CreateResolutionInput{OrderID: 10, PlatformAccountID: 20, Kind: "refund", RequestedMinor: 1234, Currency: "usd", Reason: "damaged", RequestSource: "buyer_request", RequestEvidenceID: "buyer-msg-1", ObservedAt: now, IdempotencyKey: "request-1"})
	if err != nil || replay.ID != c.ID {
		t.Fatalf("same request was not idempotent: %#v %v", replay, err)
	}
	if _, err = s.CreateResolution(7, &CreateResolutionInput{OrderID: 999, PlatformAccountID: 20, Kind: "refund", RequestedMinor: 1, Currency: "USD", Reason: "other", RequestSource: "buyer_request", RequestEvidenceID: "other", ObservedAt: now, IdempotencyKey: "request-1"}); err == nil {
		t.Fatal("idempotency payload conflict accepted")
	}
	if _, err = s.SubmitResolution(7, c.ID, ResolutionExecutionInput{ExternalRequestID: "ext-1", IdempotencyKey: "exec-1"}); err == nil {
		t.Fatal("execution bypassed Owner decision")
	}
	if _, err = s.DecideResolution(7, 8, c.ID, ResolutionDecisionInput{Decision: "approved", Reason: "valid", IdempotencyKey: "decision-1"}); err == nil {
		t.Fatal("non-Owner decided case")
	}
	if _, err = s.DecideResolution(7, 7, c.ID, ResolutionDecisionInput{Decision: "approved", Reason: "evidence accepted", IdempotencyKey: "decision-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SubmitResolution(7, c.ID, ResolutionExecutionInput{ExternalRequestID: "ext-1", IdempotencyKey: "exec-1"}); err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"provider":"platform","state":"refunded"}`)
	sum := sha256.Sum256(payload)
	if _, err = s.RecordResolutionReceipt(7, c.ID, ResolutionReceiptInput{Outcome: "succeeded", SourceType: "manual_note", EvidenceID: "ev-1", ExternalReceiptID: "receipt-1", ObservedAt: now, ActualMinor: 1234, Currency: "USD", Payload: payload, PayloadSHA256: hex.EncodeToString(sum[:])}); err == nil {
		t.Fatal("untrusted receipt closed case")
	}
	done, err := s.RecordResolutionReceipt(7, c.ID, ResolutionReceiptInput{Outcome: "succeeded", SourceType: "platform_receipt", EvidenceID: "ev-1", ExternalReceiptID: "receipt-1", ObservedAt: now, ActualMinor: 1234, Currency: "USD", Payload: payload, PayloadSHA256: hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != ResolutionSucceeded || done.ConsequenceStatus != "deferred" {
		t.Fatalf("receipt did not close fact only: %#v", done)
	}
	detail, err := s.GetResolutionDetail(7, c.ID)
	if err != nil || detail.Receipt == nil || detail.Receipt.PayloadSHA256 != hex.EncodeToString(sum[:]) || detail.Case.Status != ResolutionSucceeded {
		t.Fatalf("immutable receipt is not visible in Owner detail: %#v %v", detail, err)
	}
	if _, err = s.GetResolution(8, c.ID); err == nil {
		t.Fatal("cross-Owner read leaked case")
	}
	if _, err = s.CreateResolution(8, &CreateResolutionInput{OrderID: 10, PlatformAccountID: 20, Kind: "refund", RequestedMinor: 1234, Currency: "USD", Reason: "x", RequestSource: "buyer_request", RequestEvidenceID: "cross-owner", ObservedAt: now, IdempotencyKey: "cross-owner"}); err == nil {
		t.Fatal("cross-Owner order/account authority accepted")
	}
}

func TestResolutionReceiptRejectsAmountHashAndMutation(t *testing.T) {
	s := resolutionDB(t)
	now := time.Now().UTC()
	c, _ := s.CreateResolution(1, &CreateResolutionInput{OrderID: 1, PlatformAccountID: 1, Kind: "refund", RequestedMinor: 500, Currency: "CNY", Reason: "r", RequestSource: "platform_request", RequestEvidenceID: "e", ObservedAt: now, IdempotencyKey: "r"})
	s.DecideResolution(1, 1, c.ID, ResolutionDecisionInput{Decision: "approved", Reason: "ok", IdempotencyKey: "d"})
	s.SubmitResolution(1, c.ID, ResolutionExecutionInput{ExternalRequestID: "x", IdempotencyKey: "x"})
	p := []byte(`{"ok":true}`)
	sum := sha256.Sum256(p)
	base := ResolutionReceiptInput{Outcome: "succeeded", SourceType: "controlled_reconciliation", EvidenceID: "e1", ExternalReceiptID: "x1", ObservedAt: now, ActualMinor: 499, Currency: "CNY", Payload: p, PayloadSHA256: hex.EncodeToString(sum[:])}
	if _, err := s.RecordResolutionReceipt(1, c.ID, base); err == nil {
		t.Fatal("wrong amount accepted")
	}
	base.ActualMinor = 500
	base.PayloadSHA256 = "bad"
	if _, err := s.RecordResolutionReceipt(1, c.ID, base); err == nil {
		t.Fatal("wrong hash accepted")
	}
	base.PayloadSHA256 = hex.EncodeToString(sum[:])
	if _, err := s.RecordResolutionReceipt(1, c.ID, base); err != nil {
		t.Fatal(err)
	}
	base.EvidenceID = "changed"
	if _, err := s.RecordResolutionReceipt(1, c.ID, base); err == nil {
		t.Fatal("terminal receipt mutated")
	}
}
