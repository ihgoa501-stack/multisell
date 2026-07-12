package productimage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func governanceService(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &RightsGrant{}, &Review{}, &CostEntry{})
	return NewService(db, zap.NewNop(), nil)
}
func readyGovernanceTask(t *testing.T, s *Service, owner int64, processor string) *Task {
	t.Helper()
	task := &Task{OwnerID: owner, AssetID: 1, ImageServiceJobID: "job", OutputBlobID: strings.Repeat("b", 64), IdempotencyKey: "task-" + processor, ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Processor: processor, Version: 1, Width: 100, Height: 100, Format: "png", Status: "READY"}
	if err := s.db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	return task
}
func validRightsInput() RightsGrantInput {
	now := time.Now().UTC().Add(-time.Hour)
	until := now.Add(24 * time.Hour)
	return RightsGrantInput{AssetSHA: strings.Repeat("b", 64), CanCopy: true, CanModify: true, CanThirdPartyAI: true, CanCrossBorder: true, CanCommercialPublish: true, CanPlatformSublicense: true, TrademarkCleared: true, LikenessCleared: true, Purpose: "listing_main", Jurisdiction: "ru-RU", Channel: "ozon", Provider: "deterministic", Region: "local", Grantor: "Owner", RightsChain: "Owner-created product photograph", EvidenceSHA: strings.Repeat("e", 64), OwnerVerified: true, ValidFrom: now, ValidUntil: &until, IdempotencyKey: "rights-1", ExpectedVersion: 1}
}

func TestRightsGrantBindsExactOwnerAssetHashAndIsIdempotent(t *testing.T) {
	s := governanceService(t)
	in := validRightsInput()
	first, err := s.CreateRightsGrant(context.Background(), 42, in)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.CreateRightsGrant(context.Background(), 42, in)
	if err != nil || again.ID != first.ID {
		t.Fatalf("replay=%+v err=%v", again, err)
	}
	changed := in
	changed.Channel = "amazon"
	if _, err = s.CreateRightsGrant(context.Background(), 42, changed); err == nil {
		t.Fatal("same key with changed scope must conflict")
	}
	items, total, err := s.ListRights(context.Background(), 99, in.AssetSHA, 1, 20)
	if err != nil || total != 0 || len(items) != 0 {
		t.Fatalf("cross-owner leakage items=%v total=%d err=%v", items, total, err)
	}
}

func TestRightsGrantRequiresOwnerVerificationEvidenceAndValidWindow(t *testing.T) {
	for name, mutate := range map[string]func(*RightsGrantInput){"not verified": func(i *RightsGrantInput) { i.OwnerVerified = false }, "bad evidence": func(i *RightsGrantInput) { i.EvidenceSHA = "bad" }, "expired window": func(i *RightsGrantInput) { past := i.ValidFrom.Add(-time.Hour); i.ValidUntil = &past }, "missing jurisdiction": func(i *RightsGrantInput) { i.Jurisdiction = "" }} {
		t.Run(name, func(t *testing.T) {
			s := governanceService(t)
			in := validRightsInput()
			mutate(&in)
			if _, err := s.CreateRightsGrant(context.Background(), 42, in); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestFiveAxisReviewRejectsActualAndRequiresEveryExplicitStatus(t *testing.T) {
	s := governanceService(t)
	task := readyGovernanceTask(t, s, 42, "deterministic")
	in := FiveAxisReviewInput{AssetSHA: task.OutputBlobID, Purpose: "listing_main", Channel: "ozon", ProductAuthenticity: "passed", RightsStatus: "passed", ChannelRules: "passed", ClaimsScene: "passed", TechnicalVisual: "passed", EvidenceSHA: strings.Repeat("e", 64), EvidenceTruth: "quoted", IdempotencyKey: "review-1", ExpectedVersion: 1}
	review, err := s.CreateFiveAxisReview(context.Background(), 42, task.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if review.ProductAuthenticity != ReviewPassed {
		t.Fatalf("review=%+v", review)
	}
	in.IdempotencyKey = "review-2"
	in.EvidenceTruth = "actual"
	if _, err := s.CreateFiveAxisReview(context.Background(), 42, task.ID, in); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ordinary actual accepted: %v", err)
	}
	in.EvidenceTruth = "quoted"
	in.TechnicalVisual = "maybe"
	if _, err := s.CreateFiveAxisReview(context.Background(), 42, task.ID, in); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid status accepted: %v", err)
	}
}

func TestCostEntryStrictDecimalVersionAndOwnerScope(t *testing.T) {
	s := governanceService(t)
	task := readyGovernanceTask(t, s, 42, "openai")
	base := CostEntryInput{Kind: "estimated", Category: "provider", Provider: "openai", Amount: "1.2500", Currency: "USD", ExchangeRate: "7.2000", ExchangeRateSource: "Owner supplied rate", ObservedAt: time.Now().UTC(), BillingStatus: "estimated", IdempotencyKey: "cost-1", ExpectedVersion: 1}
	entry, err := s.CreateCostEntry(context.Background(), 42, task.ID, base)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.CreateCostEntry(context.Background(), 42, task.ID, base)
	if err != nil || again.ID != entry.ID {
		t.Fatalf("replay=%+v err=%v", again, err)
	}
	for _, bad := range []string{"1e2", "01.2", "-1", "1.00000", " 1.2 "} {
		in := base
		in.IdempotencyKey = "bad-" + bad
		in.Amount = bad
		if _, err := s.CreateCostEntry(context.Background(), 42, task.ID, in); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("amount %q accepted err=%v", bad, err)
		}
	}
	if _, err := s.CreateCostEntry(context.Background(), 99, task.ID, base); err == nil {
		t.Fatal("another owner created cost")
	}
}

func TestPublicationGateRequiresCurrentRightsAndFivePassedReviewsButNotDeterministicCost(t *testing.T) {
	s := governanceService(t)
	task := readyGovernanceTask(t, s, 42, "deterministic")
	if err := s.VerifyPublicationGate(context.Background(), 42, task.ID, task.OutputBlobID, "listing_main", "ozon", "ru-RU"); !errors.Is(err, ErrGateBlocked) {
		t.Fatalf("empty gate err=%v", err)
	}
	rights := validRightsInput()
	if _, err := s.CreateRightsGrant(context.Background(), 42, rights); err != nil {
		t.Fatal(err)
	}
	review := FiveAxisReviewInput{AssetSHA: task.OutputBlobID, Purpose: "listing_main", Channel: "ozon", ProductAuthenticity: "passed", RightsStatus: "passed", ChannelRules: "passed", ClaimsScene: "passed", TechnicalVisual: "passed", EvidenceSHA: strings.Repeat("e", 64), EvidenceTruth: "quoted", IdempotencyKey: "review", ExpectedVersion: 1}
	if _, err := s.CreateFiveAxisReview(context.Background(), 42, task.ID, review); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyPublicationGate(context.Background(), 42, task.ID, task.OutputBlobID, "listing_main", "ozon", "ru-RU"); err != nil {
		t.Fatalf("deterministic cost must not block: %v", err)
	}
}

func TestPaidPublicationGateRequiresKnownCost(t *testing.T) {
	s := governanceService(t)
	task := readyGovernanceTask(t, s, 42, "openai")
	rights := validRightsInput()
	rights.Provider = "openai"
	rights.Region = "*"
	if _, err := s.CreateRightsGrant(context.Background(), 42, rights); err != nil {
		t.Fatal(err)
	}
	review := FiveAxisReviewInput{AssetSHA: task.OutputBlobID, Purpose: "listing_main", Channel: "ozon", ProductAuthenticity: "passed", RightsStatus: "passed", ChannelRules: "passed", ClaimsScene: "passed", TechnicalVisual: "passed", EvidenceSHA: strings.Repeat("e", 64), EvidenceTruth: "quoted", IdempotencyKey: "review", ExpectedVersion: 1}
	if _, err := s.CreateFiveAxisReview(context.Background(), 42, task.ID, review); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyPublicationGate(context.Background(), 42, task.ID, task.OutputBlobID, "listing_main", "ozon", "ru-RU"); !errors.Is(err, ErrGateBlocked) {
		t.Fatalf("missing cost err=%v", err)
	}
	cost := CostEntryInput{Kind: "estimated", Category: "provider", Provider: "openai", Amount: "2.00", Currency: "USD", ExchangeRate: "7.2", ExchangeRateSource: "Owner", ObservedAt: time.Now().UTC(), BillingStatus: "estimated", IdempotencyKey: "cost", ExpectedVersion: 1}
	if _, err := s.CreateCostEntry(context.Background(), 42, task.ID, cost); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyPublicationGate(context.Background(), 42, task.ID, task.OutputBlobID, "listing_main", "ozon", "ru-RU"); err != nil {
		t.Fatal(err)
	}
}

func TestRevokedOrExpiredRightsFailClosed(t *testing.T) {
	s := governanceService(t)
	task := readyGovernanceTask(t, s, 42, "deterministic")
	rights := validRightsInput()
	grant, err := s.CreateRightsGrant(context.Background(), 42, rights)
	if err != nil {
		t.Fatal(err)
	}
	review := FiveAxisReviewInput{AssetSHA: task.OutputBlobID, Purpose: "listing_main", Channel: "ozon", ProductAuthenticity: "passed", RightsStatus: "passed", ChannelRules: "passed", ClaimsScene: "passed", TechnicalVisual: "passed", EvidenceSHA: strings.Repeat("e", 64), EvidenceTruth: "quoted", IdempotencyKey: "review", ExpectedVersion: 1}
	if _, err := s.CreateFiveAxisReview(context.Background(), 42, task.ID, review); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RevokeRightsGrant(context.Background(), 42, grant.ID, RevokeRightsInput{ExpectedVersion: 1, IdempotencyKey: "revoke", Reason: "permission withdrawn"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RevokeRightsGrant(context.Background(), 42, grant.ID, RevokeRightsInput{ExpectedVersion: 1, IdempotencyKey: "revoke", Reason: "permission withdrawn"}); err != nil {
		t.Fatalf("revocation replay: %v", err)
	}
	if _, err := s.RevokeRightsGrant(context.Background(), 42, grant.ID, RevokeRightsInput{ExpectedVersion: 1, IdempotencyKey: "revoke", Reason: "changed reason"}); err == nil {
		t.Fatal("changed revocation reused idempotency key")
	}
	if err := s.VerifyPublicationGate(context.Background(), 42, task.ID, task.OutputBlobID, "listing_main", "ozon", "ru-RU"); !errors.Is(err, ErrGateBlocked) {
		t.Fatalf("revoked rights passed: %v", err)
	}
}

func TestLatestBlockedReviewOverridesEarlierPassedReview(t *testing.T) {
	s := governanceService(t)
	task := readyGovernanceTask(t, s, 42, "deterministic")
	rights := validRightsInput()
	if _, err := s.CreateRightsGrant(context.Background(), 42, rights); err != nil {
		t.Fatal(err)
	}
	base := FiveAxisReviewInput{AssetSHA: task.OutputBlobID, Purpose: "listing_main", Channel: "ozon", ProductAuthenticity: "passed", RightsStatus: "passed", ChannelRules: "passed", ClaimsScene: "passed", TechnicalVisual: "passed", EvidenceSHA: strings.Repeat("e", 64), EvidenceTruth: "quoted", IdempotencyKey: "review-pass", ExpectedVersion: 1}
	if _, err := s.CreateFiveAxisReview(context.Background(), 42, task.ID, base); err != nil {
		t.Fatal(err)
	}
	base.IdempotencyKey = "review-block"
	base.ChannelRules = "blocked"
	if _, err := s.CreateFiveAxisReview(context.Background(), 42, task.ID, base); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyPublicationGate(context.Background(), 42, task.ID, task.OutputBlobID, "listing_main", "ozon", "ru-RU"); !errors.Is(err, ErrGateBlocked) {
		t.Fatalf("latest blocked review was bypassed: %v", err)
	}
}
