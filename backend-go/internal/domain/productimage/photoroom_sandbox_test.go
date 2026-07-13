package productimage

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/imageservice"
)

type photoroomCapabilityImageService struct{ *fakeImageService }

func (f *photoroomCapabilityImageService) ListProcessors(context.Context) ([]imageservice.ProcessorCapability, error) {
	return []imageservice.ProcessorCapability{
		{Code: "deterministic", Available: true, Operations: []string{"DETERMINISTIC_RESIZE"}, SafetyLevel: "local"},
		{Code: photoroomProcessor, Available: true, Operations: []string{"PHOTOROOM_REMOVE_BACKGROUND_SANDBOX", "PHOTOROOM_WHITE_BACKGROUND_SANDBOX", "PHOTOROOM_AI_SHADOW_SANDBOX"}, SafetyLevel: "sandbox_only", ProviderEnvironment: photoroomEnvironment, Region: photoroomRegion, Watermarked: true, NonPublishable: true, QuotaAvailable: true, QuotaRemaining: 1},
	}, nil
}

func seedPhotoroomInput(t *testing.T, svc *Service, owner int64) (*Asset, CreateTaskInput, *RightsGrant) {
	t.Helper()
	if err := svc.db.AutoMigrate(&canonicalSKU{}); err != nil {
		t.Fatal(err)
	}
	if err := svc.db.FirstOrCreate(&canonicalSKU{ID: 1}, 1).Error; err != nil {
		t.Fatal(err)
	}
	asset, err := svc.CreateAsset(t.Context(), owner, "canary.png", "image/png", []byte("sandbox-canary"))
	if err != nil {
		t.Fatal(err)
	}
	assetID := asset.ID
	now := time.Now().UTC()
	grant := &RightsGrant{OwnerID: owner, AssetID: &assetID, AssetSHA: asset.SHA256, CanCopy: true, CanModify: true, CanThirdPartyAI: true, CanCrossBorder: true, Purpose: "listing_main", Channel: "ozon", Jurisdiction: "us", Provider: photoroomProcessor, Region: photoroomRegion, Grantor: "owner", RightsChain: "test", EvidenceSHA: strings.Repeat("e", 64), OwnerVerified: true, ValidFrom: now.Add(-time.Minute), IdempotencyKey: "photoroom-rights-" + strconv.FormatInt(owner, 10), RequestHash: strings.Repeat("f", 64), Version: 1}
	if err := svc.db.Create(grant).Error; err != nil {
		t.Fatal(err)
	}
	in := CreateTaskInput{AssetID: asset.ID, SKUID: 1, RecipeKey: "photoroom-recipe", RecipeVersion: 1, CandidateRound: 1, Recipe: RecipeManifest{SceneStructure: "clean studio", Model: "photoroom", ModelVersion: "sandbox", Parameters: json.RawMessage(`{}`), MustNotChange: []string{"product shape", "color"}}, IdempotencyKey: "photoroom-task", Operation: "PHOTOROOM_REMOVE_BACKGROUND_SANDBOX", Processor: photoroomProcessor, Purpose: "listing_main", Channel: "ozon", Region: photoroomRegion, Width: 100, Height: 100, Format: "png", MaxCost: "0", Currency: "USD"}
	return asset, in, grant
}

func TestPhotoroomTaskFreezesSandboxContractAndRemoteIdentity(t *testing.T) {
	db := dbtest.NewDB(t, &canonicalSKU{}, &Asset{}, &Task{}, &RightsGrant{})
	if err := db.Create(&canonicalSKU{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	remote := &photoroomCapabilityImageService{newFakeImageService()}
	svc := NewService(db, dbtest.NewLogger(t), remote)
	_, in, _ := seedPhotoroomInput(t, svc, 81)
	task, err := svc.CreateTask(t.Context(), 81, in)
	if err != nil {
		t.Fatal(err)
	}
	if task.ID <= 0 || task.Processor != photoroomProcessor || task.Region != photoroomRegion || task.ProviderEnvironment != photoroomEnvironment || task.MaxCost != "0" || task.Currency != "USD" || !task.Sandbox || !task.Watermarked || !task.NonPublishable {
		t.Fatalf("sandbox task contract not frozen: %+v", task)
	}
	job := remote.jobs[task.ImageServiceJobID]
	if job.LingMirrorTaskID != strconv.FormatInt(task.ID, 10) || job.LingMirrorTaskVersion != 1 || !verifyRemoteTaskIdentity(task, &job, 81) {
		t.Fatalf("remote identity/restrictions mismatch: %+v", job)
	}
	replay, err := svc.CreateTask(t.Context(), 81, in)
	if err != nil || replay.ID != task.ID || remote.createCalls != 1 {
		t.Fatalf("replay=%+v err=%v calls=%d", replay, err, remote.createCalls)
	}
}

func TestPhotoroomTaskRequiresExactFourRightsAndFixedContract(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mutateInput func(*CreateTaskInput)
		mutateGrant func(*RightsGrant)
	}{
		{name: "third-party AI", mutateGrant: func(g *RightsGrant) { g.CanThirdPartyAI = false }},
		{name: "cross-border", mutateGrant: func(g *RightsGrant) { g.CanCrossBorder = false }},
		{name: "provider", mutateGrant: func(g *RightsGrant) { g.Provider = "*" }},
		{name: "region", mutateGrant: func(g *RightsGrant) { g.Region = "*" }},
		{name: "production environment disguised as region", mutateInput: func(in *CreateTaskInput) { in.Region = "sandbox" }},
		{name: "nonzero cost", mutateInput: func(in *CreateTaskInput) { in.MaxCost = "0.01" }},
		{name: "wrong currency", mutateInput: func(in *CreateTaskInput) { in.Currency = "EUR" }},
		{name: "unsupported operation", mutateInput: func(in *CreateTaskInput) { in.Operation = "PHOTOROOM_RELIGHT" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := dbtest.NewDB(t, &Asset{}, &Task{}, &RightsGrant{})
			remote := &photoroomCapabilityImageService{newFakeImageService()}
			svc := NewService(db, dbtest.NewLogger(t), remote)
			_, in, grant := seedPhotoroomInput(t, svc, 82)
			if tc.mutateGrant != nil {
				tc.mutateGrant(grant)
				if err := db.Save(grant).Error; err != nil {
					t.Fatal(err)
				}
			}
			if tc.mutateInput != nil {
				tc.mutateInput(&in)
			}
			if _, err := svc.CreateTask(t.Context(), 82, in); err == nil {
				t.Fatal("invalid sandbox task accepted")
			}
			if remote.createCalls != 0 {
				t.Fatalf("invalid request reached Image Service: %d", remote.createCalls)
			}
		})
	}
}

func TestPhotoroomExecutionRechecksRevokedRightsBeforeApprovalConsumption(t *testing.T) {
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &RightsGrant{}, &ExecutionApproval{}, &ExecutionRightsSnapshot{}, &BudgetReservation{}, &CostEntry{})
	remote := &photoroomCapabilityImageService{newFakeImageService()}
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("k", 32))
	_, in, grant := seedPhotoroomInput(t, svc, 83)
	task, err := svc.CreateTask(t.Context(), 83, in)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	approval := ExecutionApproval{ExecutionID: "sandbox-approval", OwnerID: 83, TaskID: task.ID, TaskVersion: task.Version, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, MaxCost: "0", Currency: "USD", Nonce: strings.Repeat("n", 64), ApprovedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := db.Create(&approval).Error; err != nil {
		t.Fatal(err)
	}
	reservation := BudgetReservation{OwnerID: 83, PolicyID: 1, ApprovalID: approval.ID, TaskID: task.ID, TaskVersion: task.Version, ManifestHash: task.ManifestHash, Provider: task.Processor, Currency: "USD", ReservedAmount: "0", State: "reserved"}
	if err := db.Create(&reservation).Error; err != nil {
		t.Fatal(err)
	}
	cost := CostEntry{OwnerID: 83, TaskID: task.ID, Kind: "estimated", Category: "provider", Provider: task.Processor, Amount: "0", Currency: "USD", ExchangeRate: "1", ExchangeRateSource: "same_currency", ObservedAt: now, BillingStatus: "estimated", IdempotencyKey: "sandbox-cost", RequestHash: strings.Repeat("c", 64), ExpectedTaskVersion: task.Version}
	if err := db.Create(&cost).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(grant).Updates(map[string]any{"revoked_at": now, "revocation_reason": "withdrawn"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Execute(t.Context(), 83, task.ID, "sandbox-execute"); conflictCode(err) != "INPUT_RIGHTS_REQUIRED" {
		t.Fatalf("revoked right execution err=%v", err)
	}
	var stored ExecutionApproval
	if err := db.First(&stored, approval.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ConsumedAt != nil || remote.authorizedExecuteCalls != 0 {
		t.Fatalf("approval consumed=%v calls=%d", stored.ConsumedAt, remote.authorizedExecuteCalls)
	}
}

func TestPhotoroomApprovalReservesZeroBudgetAndExecutionUsesOneTimeToken(t *testing.T) {
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &RightsGrant{}, &ExecutionApproval{}, &ExecutionRightsSnapshot{}, &BudgetPolicy{}, &BudgetReservation{}, &BudgetCharge{}, &CostEntry{})
	remote := &photoroomCapabilityImageService{newFakeImageService()}
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("k", 32))
	_, in, grant := seedPhotoroomInput(t, svc, 84)
	task, err := svc.CreateTask(t.Context(), 84, in)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err = svc.CreateBudgetPolicy(t.Context(), 84, BudgetPolicyInput{Currency: "USD", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "1", IdempotencyKey: "sandbox-budget"}); err != nil {
		t.Fatal(err)
	}
	approval, err := svc.ApproveExecution(t.Context(), 84, task.ID, ApprovalInput{Processor: photoroomProcessor, MaxCost: "0", Currency: "USD", ExpectedVersion: task.Version})
	if err != nil {
		t.Fatal(err)
	}
	var reservation BudgetReservation
	if err := db.Where("approval_id=?", approval.ID).First(&reservation).Error; err != nil {
		t.Fatal(err)
	}
	if reservation.ReservedAmount != "0" || reservation.State != "reserved" {
		t.Fatalf("reservation=%+v", reservation)
	}
	if _, err = svc.Execute(t.Context(), 84, task.ID, "sandbox-exec"); err != nil {
		t.Fatal(err)
	}
	if remote.authorizedExecuteCalls != 1 || remote.lastExecutionToken == "" {
		t.Fatalf("calls=%d token=%q", remote.authorizedExecuteCalls, remote.lastExecutionToken)
	}
	var snapshot ExecutionRightsSnapshot
	if err := db.Where("approval_id=?", approval.ID).First(&snapshot).Error; err != nil {
		t.Fatal(err)
	}
	if snapshot.GrantID <= 0 || snapshot.GrantVersion != 1 || snapshot.EvidenceSHA != strings.Repeat("e", 64) || snapshot.ApprovalExecutionID != approval.ExecutionID || !snapshot.CanCopy || !snapshot.CanModify || !snapshot.CanThirdPartyAI || !snapshot.CanCrossBorder {
		t.Fatalf("execution rights snapshot did not freeze exact authorization: %+v", snapshot)
	}
	if _, err = svc.Execute(t.Context(), 84, task.ID, "sandbox-exec-2"); conflictCode(err) != "APPROVAL_REQUIRED" {
		t.Fatalf("one-time approval replay err=%v", err)
	}
	if remote.authorizedExecuteCalls != 1 {
		t.Fatalf("duplicate external execution: %d", remote.authorizedExecuteCalls)
	}
	if _, err := svc.RevokeRightsGrant(t.Context(), 84, grant.ID, RevokeRightsInput{ExpectedVersion: 1, IdempotencyKey: "post-claim-revoke", Reason: "withdrawn for future executions"}); err != nil {
		t.Fatal(err)
	}
	var afterRevoke ExecutionRightsSnapshot
	if err := db.First(&afterRevoke, snapshot.ID).Error; err != nil || afterRevoke.GrantVersion != 1 || afterRevoke.EvidenceSHA != snapshot.EvidenceSHA {
		t.Fatalf("later revocation changed frozen authorization: snapshot=%+v err=%v", afterRevoke, err)
	}
}

func TestPhotoroomOutputCannotEnterImageSetOrReleasePath(t *testing.T) {
	remote := &photoroomCapabilityImageService{newFakeImageService()}
	task := &Task{ID: 9, OwnerID: 7, ImageServiceJobID: "sandbox-job", ManifestHash: strings.Repeat("a", 64), Operation: "PHOTOROOM_WHITE_BACKGROUND_SANDBOX", Processor: photoroomProcessor, Region: photoroomRegion, ProviderEnvironment: photoroomEnvironment, MaxCost: "0", Currency: "USD", Version: 1, Status: "READY", OutputBlobID: strings.Repeat("b", 64), Sandbox: true, Watermarked: true, NonPublishable: true}
	remote.jobs[task.ImageServiceJobID] = imageservice.Job{ID: task.ImageServiceJobID, OwnerID: 7, LingMirrorTaskID: "9", LingMirrorTaskVersion: 1, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, Region: task.Region, ProviderEnvironment: task.ProviderEnvironment, MaxCost: "0", Currency: "USD", Status: "READY", OutputBlobID: task.OutputBlobID, Sandbox: true, Watermarked: true, NonPublishable: true}
	h := &ImageSetHandler{image: remote}
	if _, err := h.verifyTaskLineage(t.Context(), 7, task); !errors.Is(err, ErrImageSetInvalid) {
		t.Fatalf("sandbox output entered image set: %v", err)
	}
	job := remote.jobs[task.ImageServiceJobID]
	if !isNonPublishableOutput(task, &job) {
		t.Fatal("release/publish restriction was not permanent")
	}
}
