package sourcing1688

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
		&productRow{}, &skuRow{}, &mediaRow{}, &costRow{}, &listingRow{}, &draftRow{}, &platformAccountRow{},
		&approval.ApprovalRequest{}, &operationlog.OperationLog{}, &PublishAttempt{}, &PublishTerminalEvidence{}, &Sourcing1688TaskLink{}, &SourcingComplianceEvidence{},
		&sourcingOpportunityRow{}, &sourcingOpportunityDecisionRow{}, &sourcingMarketDecisionRow{},
	)
	ownerID := int64(42)
	demandCaseID := int64(7)
	experimentID := "EXP-PUBLISH-1"
	db.Create(&demandCaseRow{ID: demandCaseID, OwnerID: ownerID, SalesChannel: "Ozon", TargetLocale: "ru-RU", Status: "experiment_ready"})
	marketDecision := sourcingMarketDecisionRow{DemandCaseID: demandCaseID, OwnerID: ownerID, Decision: "selected"}
	db.Create(&marketDecision)
	opportunity := sourcingOpportunityRow{OwnerID: ownerID, DemandCaseID: demandCaseID, MarketDecisionID: marketDecision.ID, Version: 1, Title: "approved opportunity", TargetChannel: "Ozon", Status: "approved", ContentHash: "opportunity-hash"}
	db.Create(&opportunity)
	opportunityDecision := sourcingOpportunityDecisionRow{OpportunityID: opportunity.ID, OwnerID: ownerID, Version: 1, Decision: "approved", ContentHash: opportunity.ContentHash}
	db.Create(&opportunityDecision)
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
	draft := draftRow{SourcingProductID: 1, SnapshotID: 11, ProductID: product.ID, ListingID: listing.ID, DemandCaseID: demandCaseID, ExperimentID: experimentID, CreatedBy: ownerID}
	db.Create(&draft)
	contentHash, err := calculateDraftContentSHA256(db, &draft)
	if err != nil {
		t.Fatal(err)
	}
	newValue, err := marshalDraftApprovalNewValue(contentHash)
	if err != nil {
		t.Fatal(err)
	}
	draftApproval := approval.ApprovalRequest{ProductID: product.ID, RequestType: DraftApprovalRequestType, Requester: "42", RequesterUserID: &ownerID, Reviewer: "42", ReviewerUserID: &ownerID, Status: approval.StatusApproved, TargetType: DraftApprovalTargetType, TargetID: draft.ID, RiskLevel: "medium", EntityType: DraftApprovalTargetType, EntityID: draft.ID, NewValue: newValue}
	db.Create(&draftApproval)
	db.Model(&draft).Updates(map[string]any{"approval_id": draftApproval.ID, "approval_status": approval.StatusApproved, "approval_content_sha256": contentHash})
	source := Sourcing1688Product{SourceURL: "https://detail.1688.com/offer/123.html", Status: StatusDraftCreated, DemandCaseID: &demandCaseID, ExperimentID: &experimentID, ProductID: &product.ID, LifecycleStatus: LifecycleApprovedDraft}
	// Force the ID to match the already-created draft link.
	source.ID = 1
	db.Create(&source)
	link := Sourcing1688TaskLink{SourcingProductID: source.ID, DemandCaseID: demandCaseID, ExperimentID: experimentID, OwnerID: ownerID, ProductOpportunityID: &opportunity.ID, OpportunityDecisionID: &opportunityDecision.ID, AuthorityKind: "product_opportunity", Status: "linked", IsPrimary: true}
	db.Create(&link)
	observed := time.Now().UTC().Add(-time.Hour)
	for _, code := range StandardPublishComplianceRequirementCodes {
		db.Create(&SourcingComplianceEvidence{OwnerID: ownerID, SourcingProductID: source.ID, TaskLinkID: link.ID, ProductOpportunityID: opportunity.ID, SourceSnapshotID: 11, ProductID: product.ID, CountryCode: "RU", ChannelCode: "ozon", RequirementCode: code, RequirementText: code, EvidenceSource: "test://compliance/" + code, TruthStatus: "actual", Scope: "product", ObservedAt: observed, ReviewStatus: ComplianceReviewApproved, ReviewedBy: &ownerID, ReviewedAt: &observed, CreatedBy: ownerID})
	}
	account := platformAccountRow{PlatformID: 3, Status: "active", ExecutionMode: int8(integrations.ExecutionModeApprovalRequired)}
	db.Create(&account)
	fake := &fakePublishAdapter{valid: true}
	svc := NewService(db, dbtest.NewLogger(t))
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
	// A mutation after draft approval invalidates the approval and cannot reuse
	// either the draft approval or the later publish approval.
	db.Model(&skuRow{}).Where("product_id = ?", attempt.ProductID).Update("price", 99.99)
	if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("tampered approved draft execute = %v", err)
	}
	if fake.publishCalls != 0 {
		t.Fatal("adapter called for tampered approved draft")
	}
	// Restore the exact approved content. The legacy request is still a URL-based
	// adapter payload, so even complete sourcing/compliance/Owner approval must
	// fail closed before any adapter or state mutation.
	db.Model(&skuRow{}).Where("product_id = ?", attempt.ProductID).Update("price", 19.99)
	blocked, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42)
	if !errors.Is(err, ErrImageReleaseAttestationRequired) || blocked != nil {
		t.Fatalf("legacy publish = %+v, %v", blocked, err)
	}
	if fake.validateCalls != 0 || fake.publishCalls != 0 || fake.lastInput != nil {
		t.Fatalf("legacy adapter reached: validation=%d publish=%d input=%+v", fake.validateCalls, fake.publishCalls, fake.lastInput)
	}
	var stored PublishAttempt
	if err := db.First(&stored, attempt.ID).Error; err != nil || stored.Status != PublishStatusApproved || stored.ExecutedAt != nil || stored.CompletedAt != nil {
		t.Fatalf("blocked attempt mutated: %+v err=%v", stored, err)
	}
	replayedRequest, err := svc.RequestPublish(sourceID, in)
	if err != nil || replayedRequest.ID != attempt.ID || fake.publishCalls != 0 {
		t.Fatalf("request replay=%+v err=%v calls=%d", replayedRequest, err, fake.publishCalls)
	}
	var listing listingRow
	db.First(&listing, attempt.ListingID)
	if listing.Status != "draft" {
		t.Fatalf("listing status = %q", listing.Status)
	}
	var auditCount int64
	db.Model(&operationlog.OperationLog{}).Where("action = ? AND entity_id = ?", "publish.execute", attempt.ID).Count(&auditCount)
	if auditCount != 0 {
		t.Fatalf("publish audit count = %d", auditCount)
	}
	var claimedAuditCount int64
	db.Model(&operationlog.OperationLog{}).Where("action = ? AND entity_id = ?", "publish.execute.claimed", attempt.ID).Count(&claimedAuditCount)
	if claimedAuditCount != 0 {
		t.Fatalf("publish claimed audit count = %d", claimedAuditCount)
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

func TestPublishRequestRejectsChangedApprovedDraftContent(t *testing.T) {
	svc, db, fake, sourceID, accountID := newPublishTestService(t)
	var source Sourcing1688Product
	if err := db.First(&source, sourceID).Error; err != nil {
		t.Fatal(err)
	}
	if source.ProductID == nil {
		t.Fatal("missing product")
	}
	if err := db.Model(&productRow{}).Where("id = ?", *source.ProductID).Update("name", "changed after Owner approval").Error; err != nil {
		t.Fatal(err)
	}
	_, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "tampered-draft-request", Reason: "must not reuse stale approval", Inventories: map[string]int{"INT-1": 1}})
	if !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("request with stale draft approval = %v", err)
	}
	if fake.publishCalls != 0 {
		t.Fatal("request validation called external adapter")
	}
}

func TestPublishRequestRejectsSecondActiveAttemptWithDifferentKey(t *testing.T) {
	svc, _, fake, sourceID, accountID := newPublishTestService(t)
	first := &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-active-001", Reason: "first request", Inventories: map[string]int{"INT-1": 1}}
	if _, err := svc.RequestPublish(sourceID, first); err != nil {
		t.Fatalf("first request: %v", err)
	}
	second := *first
	second.IdempotencyKey = "publish-active-002"
	if _, err := svc.RequestPublish(sourceID, &second); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("second active request = %v, want workflow gate", err)
	}
	if fake.publishCalls != 0 {
		t.Fatal("parallel request must not invoke platform adapter")
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
	if !errors.Is(err, ErrImageReleaseAttestationRequired) || failed != nil {
		t.Fatalf("legacy adapter failure path must be unreachable: failed=%+v err=%v", failed, err)
	}
	if fake.validateCalls != 0 || fake.publishCalls != 0 {
		t.Fatalf("legacy adapter was called: validate=%d publish=%d", fake.validateCalls, fake.publishCalls)
	}
	var stored PublishAttempt
	db.First(&stored, attempt.ID)
	if stored.Status != PublishStatusApproved || stored.ErrorMessage != "" || stored.ExecutedAt != nil || stored.CompletedAt != nil {
		t.Fatalf("blocked attempt mutated: %+v", stored)
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
	if !errors.Is(err, ErrImageReleaseAttestationRequired) || failed != nil || fake.validateCalls != 0 || fake.publishCalls != 0 {
		t.Fatalf("legacy mock-code path failed=%+v err=%v validate=%d calls=%d", failed, err, fake.validateCalls, fake.publishCalls)
	}
}

func TestPublishDryRunAndSandboxAreLocalMockOnly(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode integrations.ExecutionMode
	}{
		{name: "dry_run", mode: integrations.ExecutionModeDryRun},
		{name: "sandbox", mode: integrations.ExecutionModeSandbox},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, db, fake, sourceID, accountID := newPublishTestService(t)
			if err := db.Model(&platformAccountRow{}).Where("id = ?", accountID).Update("execution_mode", int8(tc.mode)).Error; err != nil {
				t.Fatal(err)
			}
			// The fixture's approved draft deliberately contains an arbitrary HTTPS
			// image URL. Mock execution must never forward it to an adapter or
			// reinterpret it as controlled image bytes.
			attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "mock-" + tc.name, Reason: "Owner mock verification", Inventories: map[string]int{"INT-1": 0}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "mock only"}); err != nil {
				t.Fatal(err)
			}
			got, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42)
			if err != nil || got == nil || got.Status != PublishStatusMocked {
				t.Fatalf("mock result=%+v err=%v", got, err)
			}
			if fake.validateCalls != 0 || fake.publishCalls != 0 || fake.lastInput != nil {
				t.Fatalf("adapter reached: validate=%d publish=%d input=%+v", fake.validateCalls, fake.publishCalls, fake.lastInput)
			}
			var payload map[string]any
			if err := json.Unmarshal(got.ResponsePayload, &payload); err != nil || payload["truth_status"] != "mock" || payload["external_write"] != false || payload["execution_mode"] != tc.mode.String() {
				t.Fatalf("mock payload=%s parsed=%+v err=%v", got.ResponsePayload, payload, err)
			}
			if strings.Contains(string(got.ResponsePayload), "asset.example") {
				t.Fatalf("mock response leaked input URL: %s", got.ResponsePayload)
			}
			var listing listingRow
			if err := db.First(&listing, attempt.ListingID).Error; err != nil || listing.Status != "draft" {
				t.Fatalf("mock mutated listing=%+v err=%v", listing, err)
			}
			var link Sourcing1688TaskLink
			if err := db.First(&link, attempt.TaskLinkID).Error; err != nil || link.WorkflowStatus != "publish_mocked" {
				t.Fatalf("mock task=%+v err=%v", link, err)
			}
			terminal := &PublishTerminalObservationInput{OwnerID: 42, Outcome: PublishStatusSucceeded, SourceType: PublishTerminalSourcePlatformReceipt, EvidenceID: "must-not-exist", ExternalReceiptID: "must-not-exist", ObservedAt: time.Now().UTC(), PlatformProductID: "fake", ReceiptPayload: json.RawMessage(`{"status":"fake"}`)}
			if _, err := svc.ObserveTaskPublishTerminal(context.Background(), sourceID, link.ID, attempt.ID, terminal); !errors.Is(err, ErrWorkflowGate) {
				t.Fatalf("mock upgraded to external observation: %v", err)
			}
		})
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

func TestPublishBlocksMultipleActiveAccountsAndExperimentGateIsTraceOnly(t *testing.T) {
	t.Run("multiple active accounts", func(t *testing.T) {
		svc, db, fake, sourceID, accountID := newPublishTestService(t)
		db.Create(&platformAccountRow{PlatformID: 3, Status: "active", ExecutionMode: int8(integrations.ExecutionModeProduction)})
		_, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-multi-account", Reason: "Owner确认", Inventories: map[string]int{"INT-1": 0}})
		if !errors.Is(err, ErrWorkflowGate) || fake.publishCalls != 0 {
			t.Fatalf("multiple account request err=%v calls=%d", err, fake.publishCalls)
		}
	})
	t.Run("historical experiment gate does not revoke current authority", func(t *testing.T) {
		svc, db, fake, sourceID, accountID := newPublishTestService(t)
		attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-gate-revoked", Reason: "Owner确认", Inventories: map[string]int{"INT-1": 0}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "批准"}); err != nil {
			t.Fatal(err)
		}
		db.Model(&gateRow{}).Where("experiment_id = ? AND stage = ?", "EXP-PUBLISH-1", "opportunity").Update("result", "return")
		got, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42)
		if !errors.Is(err, ErrImageReleaseAttestationRequired) || got != nil {
			t.Fatalf("legacy publish did not reach image release gate: got=%+v err=%v", got, err)
		}
		if fake.validateCalls != 0 || fake.publishCalls != 0 {
			t.Fatalf("adapter reached: validate=%d publish=%d", fake.validateCalls, fake.publishCalls)
		}
	})
}

func TestPublishFailsClosedWhenProductOpportunityAuthorityIsNoLongerCurrent(t *testing.T) {
	request := func(accountID int64, key string) *PublishRequestInput {
		return &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: key, Reason: "Owner confirms", Inventories: map[string]int{"INT-1": 0}}
	}

	t.Run("request rejects paused market", func(t *testing.T) {
		svc, db, fake, sourceID, accountID := newPublishTestService(t)
		db.Create(&sourcingMarketDecisionRow{DemandCaseID: 7, OwnerID: 42, Decision: "paused"})
		if _, err := svc.RequestPublish(sourceID, request(accountID, "paused-before-request")); !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("RequestPublish with paused market = %v", err)
		}
		if fake.validateCalls != 0 || fake.publishCalls != 0 {
			t.Fatalf("adapter called during rejected request: validate=%d publish=%d", fake.validateCalls, fake.publishCalls)
		}
	})

	t.Run("decision rejects rejected market", func(t *testing.T) {
		svc, db, fake, sourceID, accountID := newPublishTestService(t)
		attempt, err := svc.RequestPublish(sourceID, request(accountID, "rejected-before-decision"))
		if err != nil {
			t.Fatal(err)
		}
		db.Create(&sourcingMarketDecisionRow{DemandCaseID: 7, OwnerID: 42, Decision: "rejected"})
		if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "approve"}); !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("DecidePublish with rejected market = %v", err)
		}
		if fake.validateCalls != 0 || fake.publishCalls != 0 {
			t.Fatalf("adapter called during rejected decision: validate=%d publish=%d", fake.validateCalls, fake.publishCalls)
		}
	})

	t.Run("execute rejects changed opportunity", func(t *testing.T) {
		svc, db, fake, sourceID, accountID := newPublishTestService(t)
		attempt, err := svc.RequestPublish(sourceID, request(accountID, "opportunity-revoked-before-execute"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "approve"}); err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&sourcingOpportunityRow{}).Where("status = ?", "approved").Update("status", "rejected").Error; err != nil {
			t.Fatal(err)
		}
		if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("ExecutePublish with changed opportunity = %v", err)
		}
		if fake.validateCalls != 0 || fake.publishCalls != 0 {
			t.Fatalf("adapter called after authority revocation: validate=%d publish=%d", fake.validateCalls, fake.publishCalls)
		}
	})

	t.Run("execute rejects authority link for another source", func(t *testing.T) {
		svc, db, fake, sourceID, accountID := newPublishTestService(t)
		attempt, err := svc.RequestPublish(sourceID, request(accountID, "source-link-mismatch"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "approve"}); err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&Sourcing1688TaskLink{}).Where("sourcing_product_id = ?", sourceID).Update("sourcing_product_id", sourceID+100).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("ExecutePublish with mismatched source authority = %v", err)
		}
		if fake.validateCalls != 0 || fake.publishCalls != 0 {
			t.Fatalf("adapter called for another source authority: validate=%d publish=%d", fake.validateCalls, fake.publishCalls)
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
	if !errors.Is(err, ErrImageReleaseAttestationRequired) || got != nil || fake.publishCalls != 0 {
		t.Fatalf("legacy timeout seam must be unreachable: attempt=%+v err=%v calls=%d", got, err, fake.publishCalls)
	}
	// Preserve coverage of reconciliation for a historical attempt that had
	// already reached an ambiguous external outcome before this seam was frozen.
	if err := svc.db.Model(&PublishAttempt{}).Where("id = ?", attempt.ID).Updates(map[string]any{"status": PublishStatusReconcile, "error_message": "timeout"}).Error; err != nil {
		t.Fatal(err)
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

func TestPublishReconciliationFailsClosedAfterOpportunityAuthorityRevocation(t *testing.T) {
	svc, db, fake, sourceID, accountID := newPublishTestService(t)
	fake.publishErr = context.DeadlineExceeded
	attempt, err := svc.RequestPublish(sourceID, &PublishRequestInput{RequesterID: 42, PlatformAccountID: accountID, IdempotencyKey: "publish-timeout-revoked", Reason: "Owner确认", Inventories: map[string]int{"INT-1": 0}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DecidePublish(sourceID, attempt.ID, &PublishDecisionInput{OwnerID: 42, Action: "approve", Note: "批准"}); err != nil {
		t.Fatal(err)
	}
	if got, err := svc.ExecutePublish(context.Background(), sourceID, attempt.ID, 42); !errors.Is(err, ErrImageReleaseAttestationRequired) || got != nil || fake.publishCalls != 0 {
		t.Fatalf("legacy timeout seam must be unreachable: attempt=%+v err=%v calls=%d", got, err, fake.publishCalls)
	}
	if err := db.Model(&PublishAttempt{}).Where("id = ?", attempt.ID).Updates(map[string]any{"status": PublishStatusReconcile, "error_message": "timeout"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&sourcingMarketDecisionRow{DemandCaseID: 7, OwnerID: 42, Decision: "paused"}).Error; err != nil {
		t.Fatal(err)
	}
	_, err = svc.ReconcilePublish(context.Background(), sourceID, attempt.ID, &PublishReconcileInput{OwnerID: 42, Outcome: PublishStatusSubmitted, EvidenceURI: "evidence://platform/query/revoked", ObservedAt: time.Now().UTC(), TruthStatus: "actual", PlatformResult: integrations.PublishResult{PlatformProductID: "observed-revoked"}})
	if !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("reconciliation after market pause err=%v", err)
	}
	var stored PublishAttempt
	if err := db.First(&stored, attempt.ID).Error; err != nil || stored.Status != PublishStatusReconcile {
		t.Fatalf("revoked reconciliation mutated attempt=%+v err=%v", stored, err)
	}
	var listing listingRow
	if err := db.First(&listing, attempt.ListingID).Error; err != nil || listing.Status != "draft" {
		t.Fatalf("revoked reconciliation mutated listing=%+v err=%v", listing, err)
	}
}
