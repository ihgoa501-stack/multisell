package sourcing1688

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/integrations"
)

func submittedPublishForTerminalTest(t *testing.T) (*Service, int64, int64, int64) {
	t.Helper()
	svc, _, _, sourceID, accountID := newPublishTestService(t)
	attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "terminal-test", Reason: "terminal test", Inventories: map[string]int{"INT-1": 1}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "approved"}); err != nil {
		t.Fatal(err)
	}
	var link Sourcing1688TaskLink
	if err := svc.db.Where("sourcing_product_id = ?", sourceID).First(&link).Error; err != nil {
		t.Fatal(err)
	}
	// Historical fixture only: terminal-observation behavior still has to read
	// records that were submitted before the URL adapter seam was frozen. New
	// ExecutePublish calls cannot create this state.
	response, err := json.Marshal(integrations.PublishResult{PlatformProductID: "external-123", PlatformURL: "https://platform.example/p/external-123"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&PublishAttempt{}).Where("id = ?", attempt.ID).Updates(map[string]any{"status": PublishStatusSubmitted, "response_payload": response}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&Sourcing1688TaskLink{}).Where("id = ?", link.ID).Update("workflow_status", PublishStatusSubmitted).Error; err != nil {
		t.Fatal(err)
	}
	return svc, sourceID, link.ID, attempt.ID
}

func TestObserveTaskPublishTerminalSucceededIsImmutableAndIdempotent(t *testing.T) {
	svc, sourceID, linkID, attemptID := submittedPublishForTerminalTest(t)
	in := &PublishTerminalObservationInput{OwnerID: 42, Outcome: PublishStatusSucceeded, SourceType: PublishTerminalSourcePlatformReceipt, EvidenceID: "ozon-event-1", ExternalReceiptID: "receipt-1", ObservedAt: time.Now().UTC().Add(-time.Minute), PlatformProductID: "external-123", ReceiptPayload: json.RawMessage(`{"event_id":"ozon-event-1","status":"published"}`)}
	got, err := svc.ObserveTaskPublishTerminal(context.Background(), sourceID, linkID, attemptID, in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != PublishStatusSucceeded || got.ReceiptSHA256 == "" {
		t.Fatalf("evidence=%+v", got)
	}
	replay, err := svc.ObserveTaskPublishTerminal(context.Background(), sourceID, linkID, attemptID, in)
	if err != nil || replay.ID != got.ID {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	var attempt PublishAttempt
	svc.db.First(&attempt, attemptID)
	if attempt.Status != PublishStatusSucceeded {
		t.Fatalf("attempt status=%s", attempt.Status)
	}
	var link Sourcing1688TaskLink
	svc.db.First(&link, linkID)
	if link.WorkflowStatus != PublishStatusSucceeded {
		t.Fatalf("task status=%s", link.WorkflowStatus)
	}

	changed := *in
	changed.ReceiptPayload = json.RawMessage(`{"event_id":"ozon-event-1","status":"different"}`)
	if _, err := svc.ObserveTaskPublishTerminal(context.Background(), sourceID, linkID, attemptID, &changed); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("changed replay err=%v", err)
	}
}

func TestObserveTaskPublishTerminalRejectsCrossTaskAndMismatchedReceipt(t *testing.T) {
	svc, sourceID, linkID, attemptID := submittedPublishForTerminalTest(t)
	base := &PublishTerminalObservationInput{OwnerID: 42, Outcome: PublishStatusSucceeded, SourceType: PublishTerminalSourceControlledReconciliation, EvidenceID: "reconcile-1", ExternalReceiptID: "case-1", ObservedAt: time.Now().UTC(), PlatformProductID: "wrong-product", ReceiptPayload: json.RawMessage(`{"status":"published"}`)}
	if _, err := svc.ObserveTaskPublishTerminal(context.Background(), sourceID, linkID+999, attemptID, base); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("cross task err=%v", err)
	}
	if _, err := svc.ObserveTaskPublishTerminal(context.Background(), sourceID, linkID, attemptID, base); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("mismatch err=%v", err)
	}
}

func TestObserveTaskPublishTerminalFailedRequiresExplanation(t *testing.T) {
	svc, sourceID, linkID, attemptID := submittedPublishForTerminalTest(t)
	in := &PublishTerminalObservationInput{OwnerID: 42, Outcome: PublishStatusFailed, SourceType: PublishTerminalSourcePlatformReceipt, EvidenceID: "event-failed", ExternalReceiptID: "receipt-failed", ObservedAt: time.Now().UTC(), ReceiptPayload: json.RawMessage(`{"status":"rejected"}`)}
	if _, err := svc.ObserveTaskPublishTerminal(context.Background(), sourceID, linkID, attemptID, in); !errors.Is(err, ErrInvalidWorkflow) {
		t.Fatalf("missing explanation err=%v", err)
	}
	in.FailureCode, in.FailureMessage = "platform_rejected", "platform rejected listing content"
	got, err := svc.ObserveTaskPublishTerminal(context.Background(), sourceID, linkID, attemptID, in)
	if err != nil || got.Outcome != PublishStatusFailed {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestPublishAndTerminalObservationFailClosedForExpiredOrRevokedCompliance(t *testing.T) {
	svc, db, fake, sourceID, accountID := newPublishTestService(t)
	attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "expired-compliance", Reason: "test expiry", Inventories: map[string]int{"INT-1": 1}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "approved"}); err != nil {
		t.Fatal(err)
	}
	expired := time.Now().UTC().Add(-time.Minute)
	if err := db.Model(&SourcingComplianceEvidence{}).Where("requirement_code = ?", StandardPublishComplianceRequirementCodes[0]).Update("expires_at", expired).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("expired compliance execute err=%v", err)
	}
	if fake.publishCalls != 0 {
		t.Fatal("adapter called despite expired compliance")
	}

	// A terminal observation also rechecks compliance instead of upgrading stale
	// authority. Put the attempt into submitted state without an external call to
	// isolate this gate, then revoke a different requirement.
	if err := db.Model(&PublishAttempt{}).Where("id = ?", attempt.ID).Update("status", PublishStatusSubmitted).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&SourcingComplianceEvidence{}).Where("requirement_code = ?", StandardPublishComplianceRequirementCodes[0]).Update("expires_at", nil).Error; err != nil {
		t.Fatal(err)
	}
	revoked := time.Now().UTC()
	if err := db.Model(&SourcingComplianceEvidence{}).Where("requirement_code = ?", StandardPublishComplianceRequirementCodes[1]).Update("revoked_at", revoked).Error; err != nil {
		t.Fatal(err)
	}
	var link Sourcing1688TaskLink
	db.Where("sourcing_product_id = ?", sourceID).First(&link)
	in := &PublishTerminalObservationInput{OwnerID: 42, Outcome: PublishStatusSucceeded, SourceType: PublishTerminalSourcePlatformReceipt, EvidenceID: "revoked-event", ExternalReceiptID: "revoked-receipt", ObservedAt: time.Now().UTC(), PlatformProductID: "external-123", ReceiptPayload: json.RawMessage(`{"status":"published"}`)}
	if _, err := svc.ObserveTaskPublishTerminal(context.Background(), sourceID, link.ID, attempt.ID, in); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("revoked compliance terminal err=%v", err)
	}
}
