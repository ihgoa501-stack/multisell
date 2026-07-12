package sourcing1688

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"gorm.io/gorm"
)

type fakePublishAdapter struct {
	validateCalls int
	publishCalls  int
	valid         bool
	publishErr    error
	lastInput     *integrations.PublishInput
}

func (f *fakePublishAdapter) ValidateCredentials(context.Context, int64) (bool, error) {
	f.validateCalls++
	return f.valid, nil
}

func (f *fakePublishAdapter) Publish(_ context.Context, in *integrations.PublishInput) (*integrations.PublishResult, error) {
	f.publishCalls++
	f.lastInput = in
	if f.publishErr != nil {
		return nil, f.publishErr
	}
	return &integrations.PublishResult{PlatformProductID: "external-123", PlatformSKU: "INT-1", PlatformURL: "https://platform.example/p/123", PublishedData: map[string]any{"external_status": "accepted"}, SyncMessage: "accepted"}, nil
}

func newPublishTestService(t *testing.T) (*Service, *gorm.DB, *fakePublishAdapter, int64, int64) {
	t.Helper()
	db := dbtest.NewDB(t,
		&Sourcing1688Product{}, &demandCaseRow{}, &experimentRow{}, &gateRow{}, &objectLinkRow{}, &platformRow{},
		&productRow{}, &skuRow{}, &listingRow{}, &draftRow{}, &platformAccountRow{},
		&approval.ApprovalRequest{}, &operationlog.OperationLog{}, &PublishAttempt{},
	)
	ownerID := int64(42)
	demandCaseID := int64(7)
	experimentID := "EXP-PUBLISH-1"
	db.Create(&demandCaseRow{ID: demandCaseID, OwnerID: ownerID, SalesChannel: "Ozon", Status: "experiment_ready"})
	db.Create(&experimentRow{ExperimentID: experimentID, OwnerID: ownerID, Status: "active", Stage: "channel"})
	db.Create(&gateRow{ExperimentID: experimentID, Stage: "opportunity", Result: "pass"})
	db.Create(&objectLinkRow{ExperimentID: experimentID, ObjectType: "demand_case", ObjectID: "7"})
	db.Create(&objectLinkRow{ExperimentID: experimentID, ObjectType: "sourcing_1688", ObjectID: "1"})
	db.Create(&platformRow{ID: 3, Name: "Ozon", Code: "ozon", Status: 1})
	product := productRow{Name: "本地标题", Description: "本地描述", CategoryID: 9, Unit: "件", Status: 1, MainImage: "https://asset.example/main.jpg"}
	db.Create(&product)
	sku := skuRow{ProductID: product.ID, Code: "INT-1", Price: 19.99, CostPrice: 8, Status: 1}
	db.Create(&sku)
	listingPayload := json.RawMessage(`{"localized_title":"Localized title","localized_description":"Localized description"}`)
	listing := listingRow{ProductID: product.ID, PlatformID: 3, PlatformSKU: "CHANNEL-1", Status: "draft", PublishedData: listingPayload}
	db.Create(&listing)
	draftApproval := approval.ApprovalRequest{ProductID: product.ID, RequestType: DraftApprovalRequestType, Requester: "42", Status: approval.StatusApproved, TargetType: DraftApprovalTargetType, RiskLevel: "medium"}
	db.Create(&draftApproval)
	draft := draftRow{SourcingProductID: 1, SnapshotID: 11, ProductID: product.ID, ListingID: listing.ID, DemandCaseID: demandCaseID, ExperimentID: experimentID, CreatedBy: ownerID, ApprovalID: &draftApproval.ID, ApprovalStatus: approval.StatusApproved}
	db.Create(&draft)
	db.Model(&draftApproval).Updates(map[string]any{"target_id": draft.ID, "entity_type": DraftApprovalTargetType, "entity_id": draft.ID})
	source := Sourcing1688Product{SourceURL: "https://detail.1688.com/offer/123.html", Status: StatusDraftCreated, DemandCaseID: &demandCaseID, ExperimentID: &experimentID, ProductID: &product.ID, LifecycleStatus: LifecycleApprovedDraft}
	// Force the ID to match the already-created draft link.
	source.ID = 1
	db.Create(&source)
	account := platformAccountRow{PlatformID: 3, Status: "active", ExecutionMode: int8(integrations.ExecutionModeApprovalRequired)}
	db.Create(&account)
	fake := &fakePublishAdapter{valid: true}
	svc := NewService(db, dbtest.NewLogger(t))
	svc.resolvePublisher = func(code string) (publishAdapter, bool) {
		return fake, code == "ozon"
	}
	return svc, db, fake, source.ID, account.ID
}

func TestPublishWorkflowRequiresIndependentApprovalAndIsIdempotent(t *testing.T) {
	svc, db, fake, sourceID, accountID := newPublishTestService(t)
	in := &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-real-001", Reason: "Owner确认真实上架", Inventories: map[string]int{"INT-1": 3}}
	attempt, err := svc.RequestPublish(sourceID, in)
	if err != nil {
		t.Fatalf("RequestPublish: %v", err)
	}
	if attempt.Status != PublishStatusPending || attempt.ApprovalID == nil {
		t.Fatalf("attempt = %+v", attempt)
	}
	if len(attempt.AdapterRequestPayload) == 0 {
		t.Fatal("exact adapter request was not frozen before approval")
	}
	var publishApproval approval.ApprovalRequest
	if err := db.First(&publishApproval, *attempt.ApprovalID).Error; err != nil {
		t.Fatal(err)
	}
	if publishApproval.RequestType != PublishApprovalRequestType || publishApproval.RiskLevel != "high" {
		t.Fatalf("publish approval = %+v", publishApproval)
	}
	var draft draftRow
	db.Where("sourcing_product_id = ?", sourceID).First(&draft)
	if draft.ApprovalID == nil || *draft.ApprovalID == *attempt.ApprovalID {
		t.Fatal("draft approval and publish approval must be independent records")
	}
	if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("execute before publish approval = %v", err)
	}
	if fake.publishCalls != 0 {
		t.Fatal("adapter called before approval")
	}
	approved, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "批准这一次账号和库存快照"})
	if err != nil || approved.Status != PublishStatusApproved {
		t.Fatalf("DecidePublish = %+v, %v", approved, err)
	}
	// Account binding is checked again immediately before the external call.
	db.Model(&platformAccountRow{}).Where("id = ?", accountID).Update("platform_id", 99)
	if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("account mismatch execute = %v", err)
	}
	if fake.publishCalls != 0 {
		t.Fatal("adapter called with mismatched account")
	}
	db.Model(&platformAccountRow{}).Where("id = ?", accountID).Update("platform_id", 3)
	// The approved request is immutable: a later catalog mutation must not alter
	// the exact price sent under this approval.
	db.Model(&skuRow{}).Where("product_id = ?", attempt.ProductID).Update("price", 99.99)
	completed, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42)
	if err != nil {
		t.Fatalf("ExecutePublish: %v", err)
	}
	if completed.Status != PublishStatusSubmitted || fake.validateCalls != 1 || fake.publishCalls != 1 {
		t.Fatalf("completed=%+v validation=%d publish=%d", completed, fake.validateCalls, fake.publishCalls)
	}
	if fake.lastInput == nil || fake.lastInput.AccountID != accountID || fake.lastInput.IdempotencyKey != in.IdempotencyKey || fake.lastInput.Inventories[fake.lastInput.SKUs[0].SkuID] != 3 {
		t.Fatalf("frozen publish input = %+v", fake.lastInput)
	}
	if fake.lastInput.Prices[fake.lastInput.SKUs[0].SkuID] != "19.99" {
		t.Fatalf("approved price snapshot was not frozen: %+v", fake.lastInput.Prices)
	}
	// Exact replay returns the ledger result and never calls the adapter again.
	replayed, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42)
	if err != nil || replayed.Status != PublishStatusSubmitted || fake.publishCalls != 1 {
		t.Fatalf("replay=%+v err=%v calls=%d", replayed, err, fake.publishCalls)
	}
	replayedRequest, err := svc.RequestPublish(sourceID, in)
	if err != nil || replayedRequest.ID != attempt.ID || fake.publishCalls != 1 {
		t.Fatalf("request replay=%+v err=%v calls=%d", replayedRequest, err, fake.publishCalls)
	}
	var listing listingRow
	db.First(&listing, attempt.ListingID)
	if listing.Status != "submitted" {
		t.Fatalf("listing status = %q", listing.Status)
	}
	var auditCount int64
	db.Model(&operationlog.OperationLog{}).Where("action = ? AND entity_id = ?", "publish.execute", attempt.ID).Count(&auditCount)
	if auditCount != 1 {
		t.Fatalf("publish audit count = %d", auditCount)
	}
	var requestAuditCount int64
	db.Model(&operationlog.OperationLog{}).Where("action = ? AND entity_id = ?", "publish.request", attempt.ID).Count(&requestAuditCount)
	if requestAuditCount != 1 {
		t.Fatalf("publish request audit count = %d", requestAuditCount)
	}
	var sameKey PublishRequestInput = *in
	sameKey.Inventories = map[string]int{"INT-1": 4}
	if _, err := svc.RequestPublish(sourceID, &sameKey); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("rebound idempotency key = %v", err)
	}
}

func TestPublishFailureStoresOnlySafeClassificationAndNeverRetries(t *testing.T) {
	svc, db, fake, sourceID, accountID := newPublishTestService(t)
	fake.publishErr = errors.New("https://api.example/publish?access_token=SECRET: provider rejected")
	attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-fail-001", Reason: "Owner确认", Inventories: map[string]int{"INT-1": 0}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "批准"}); err != nil {
		t.Fatal(err)
	}
	failed, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42)
	if err == nil || failed == nil || failed.Status != PublishStatusFailed || failed.ErrorMessage != "platform_publish_failed" {
		t.Fatalf("failed=%+v err=%v", failed, err)
	}
	if string(failed.ResponsePayload) != "null" && len(failed.ResponsePayload) != 0 {
		t.Fatalf("unexpected failed response payload %s", failed.ResponsePayload)
	}
	if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); err != nil {
		t.Fatalf("terminal failure replay: %v", err)
	}
	if fake.publishCalls != 1 {
		t.Fatalf("failed publish was retried, calls=%d", fake.publishCalls)
	}
	var stored PublishAttempt
	db.First(&stored, attempt.ID)
	if stored.ErrorMessage != "platform_publish_failed" {
		t.Fatalf("unsafe error persisted: %q", stored.ErrorMessage)
	}
}

func TestPublishRejectsMockPlatformCode(t *testing.T) {
	svc, db, fake, sourceID, accountID := newPublishTestService(t)
	db.Model(&platformRow{}).Where("id = ?", 3).Update("code", "mock-ozon")
	attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-mock-001", Reason: "Owner确认", Inventories: map[string]int{"INT-1": 0}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "批准"}); err != nil {
		t.Fatal(err)
	}
	failed, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42)
	if err == nil || failed.ErrorMessage != "adapter_unavailable" || fake.publishCalls != 0 {
		t.Fatalf("mock execution failed=%+v err=%v calls=%d", failed, err, fake.publishCalls)
	}
}

func TestPublishApprovalExpiryBlocksExecution(t *testing.T) {
	svc, db, fake, sourceID, accountID := newPublishTestService(t)
	attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-expired-001", Reason: "Owner确认", Inventories: map[string]int{"INT-1": 0}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "批准"}); err != nil {
		t.Fatal(err)
	}
	expired := time.Now().UTC().Add(-time.Minute)
	db.Model(&approval.ApprovalRequest{}).Where("id = ?", *attempt.ApprovalID).Update("expires_at", expired)
	if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("expired approval execute = %v", err)
	}
	if fake.publishCalls != 0 {
		t.Fatal("adapter called with expired approval")
	}
}

func TestPublishBlocksMultipleActiveAccountsAndRevokedOpportunityGate(t *testing.T) {
	t.Run("multiple active accounts", func(t *testing.T) {
		svc, db, fake, sourceID, accountID := newPublishTestService(t)
		db.Create(&platformAccountRow{PlatformID: 3, Status: "active", ExecutionMode: int8(integrations.ExecutionModeProduction)})
		_, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-multi-account", Reason: "Owner确认", Inventories: map[string]int{"INT-1": 0}})
		if !errors.Is(err, ErrWorkflowGate) || fake.publishCalls != 0 {
			t.Fatalf("multiple account request err=%v calls=%d", err, fake.publishCalls)
		}
	})
	t.Run("opportunity gate revoked after approval", func(t *testing.T) {
		svc, db, fake, sourceID, accountID := newPublishTestService(t)
		attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-gate-revoked", Reason: "Owner确认", Inventories: map[string]int{"INT-1": 0}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "批准"}); err != nil {
			t.Fatal(err)
		}
		db.Model(&gateRow{}).Where("experiment_id = ? AND stage = ?", "EXP-PUBLISH-1", "opportunity").Update("result", "return")
		if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("revoked gate execute = %v", err)
		}
		if fake.publishCalls != 0 {
			t.Fatal("adapter called after gate revocation")
		}
	})
}

func TestPublishTimeoutRequiresReconciliation(t *testing.T) {
	svc, _, fake, sourceID, accountID := newPublishTestService(t)
	fake.publishErr = context.DeadlineExceeded
	attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-timeout", Reason: "Owner确认", Inventories: map[string]int{"INT-1": 0}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "批准"}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42)
	if err == nil || got.Status != PublishStatusReconcile || got.ErrorMessage != "timeout" {
		t.Fatalf("timeout attempt=%+v err=%v", got, err)
	}
	if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); err != nil || fake.publishCalls != 1 {
		t.Fatalf("timeout must not auto-retry: err=%v calls=%d", err, fake.publishCalls)
	}
	reconciled, err := svc.ReconcilePublish(context.Background(), sourceID, attempt.ID, &PublishReconcileInput{OwnerID: 42, Outcome: PublishStatusSubmitted, EvidenceURI: "evidence://platform/query/123", ObservedAt: time.Now().UTC(), TruthStatus: "actual", PlatformResult: integrations.PublishResult{PlatformProductID: "observed-123", PlatformURL: "https://platform.example/p/observed-123", PublishedData: map[string]any{"status": "observed"}, SyncMessage: "observed by Owner"}})
	if err != nil || reconciled.Status != PublishStatusSubmitted || reconciled.ResponseSHA256 == "" {
		t.Fatalf("reconciled=%+v err=%v", reconciled, err)
	}
	var auditCount int64
	svc.db.Model(&operationlog.OperationLog{}).Where("action = ? AND entity_id = ?", "publish.reconcile", attempt.ID).Count(&auditCount)
	if auditCount != 1 {
		t.Fatalf("reconcile audit count = %d", auditCount)
	}
}
