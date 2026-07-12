package demandcase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func accessService(t *testing.T) *Service {
	db := dbtest.NewDB(t, &DemandCase{}, &DemandEvidence{}, &DemandVerdict{}, &ResearchBatch{}, &ResearchSnapshot{}, &DataAccessRecord{}, &PermissionRequest{})
	return NewService(db, zap.NewNop())
}

func seedAccessSnapshot(t *testing.T, s *Service, c *DemandCase, run string) int64 {
	t.Helper()
	b := ResearchBatch{BatchKey: "batch-" + run, OwnerID: c.OwnerID}
	if err := s.db.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	snap := ResearchSnapshot{BatchID: b.ID, OwnerID: c.OwnerID, DemandCaseID: c.ID, RunID: run, RunType: RunDataReality, Collector: "agent:test-data-reality", SourceURI: "https://official.example", CollectedAt: time.Now(), RawPayload: "{}", RawSHA256: payloadHash([]byte("{}"))}
	if err := s.db.Create(&snap).Error; err != nil {
		t.Fatal(err)
	}
	return snap.ID
}

func TestDataRealityAcceptsOnlyFixedAccessStatuses(t *testing.T) {
	s := accessService(t)
	ctx := context.Background()
	c := DemandCase{OwnerID: 1, Region: "DE", Consumer: "可观察消费者", NeedScenario: "待验证需求", SalesChannel: "marketplace"}
	if err := s.Create(ctx, &c); err != nil {
		t.Fatal(err)
	}
	err := s.RecordDataAccess(ctx, 1, &DataAccessRecord{DemandCaseID: c.ID, FieldName: "paid_orders", Status: "probably_available", SourceURI: "https://official.example", RunID: "reality-1"})
	if err == nil {
		t.Fatal("non-canonical access status must fail")
	}
}

func TestOwnerPermissionCardRequestsReadOnlyScopeAndExplainsRefusal(t *testing.T) {
	s := accessService(t)
	ctx := context.Background()
	c := DemandCase{OwnerID: 1, Region: "DE", Consumer: "可观察消费者", NeedScenario: "待验证需求", SalesChannel: "marketplace"}
	if err := s.Create(ctx, &c); err != nil {
		t.Fatal(err)
	}
	snapshotID := seedAccessSnapshot(t, s, &c, "reality-1")
	if err := s.RecordDataAccess(ctx, 1, &DataAccessRecord{DemandCaseID: c.ID, FieldName: "paid_orders", Status: AccessRequiresOwner, RequiredScope: "read_orders", DecisionPurpose: "判断是否存在非关联真实付款", RefusalImpact: "保持证据不足并停止该渠道", SourceURI: "https://official.example", RunID: "reality-1", SnapshotID: snapshotID, AccessMode: "read_only", PreflightRequired: true}); err != nil {
		t.Fatal(err)
	}
	cards, err := s.PermissionRequests(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 {
		t.Fatalf("cards=%d", len(cards))
	}
	if cards[0].AccessMode != "read_only" || cards[0].RequiredScope != "read_orders" {
		t.Fatalf("unsafe card %+v", cards[0])
	}
}

func TestListingOrTransactionRequirementCannotBecomeReadOnlyAuthorization(t *testing.T) {
	s := accessService(t)
	ctx := context.Background()
	c := DemandCase{OwnerID: 1, Region: "DE", Consumer: "可观察消费者", NeedScenario: "待验证需求", SalesChannel: "marketplace"}
	if err := s.Create(ctx, &c); err != nil {
		t.Fatal(err)
	}
	snapshotID := seedAccessSnapshot(t, s, &c, "reality-2")
	if err := s.RecordDataAccess(ctx, 1, &DataAccessRecord{DemandCaseID: c.ID, FieldName: "returns", Status: AccessRequiresTransaction, DecisionPurpose: "观察真实售后", RefusalImpact: "桌面研究结束", SourceURI: "https://official.example", RunID: "reality-2", SnapshotID: snapshotID, AccessMode: "read_only"}); err != nil {
		t.Fatal(err)
	}
	cards, err := s.PermissionRequests(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 0 {
		t.Fatal("transaction requirement must not be presented as a read-only permission request")
	}
}

func TestLiveMarketBatchSeparatesPermissionCandidatesFromHeldCases(t *testing.T) {
	s := accessService(t)
	out, err := s.ImportReviewedMarketPermissionBatch(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Cards) != 3 || len(out.PermissionCandidateIDs) != 0 || len(out.HeldCaseIDs) != 2 || len(out.RejectedCaseIDs) != 1 {
		t.Fatalf("unexpected outcome %+v", out)
	}
	if out.Cards[0].Verdict != VerdictEvidenceMissing || out.Cards[1].Verdict != VerdictEvidenceMissing || out.Cards[2].Verdict != VerdictRejected {
		t.Fatalf("reviewed research crossed boundary: %+v", out.Cards)
	}
	again, err := s.ImportReviewedMarketPermissionBatch(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Cards) != 3 {
		t.Fatal("live batch must be idempotent")
	}
}

func TestRequiredAccessBlocksThenLatestAvailableUnblocks(t *testing.T) {
	s := accessService(t)
	ctx := context.Background()
	c := DemandCase{OwnerID: 1, Region: "GB", Consumer: "可观察消费者", NeedScenario: "场景", SalesChannel: "marketplace"}
	if err := s.Create(ctx, &c); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	scoutSID := seedEvidenceSnapshot(t, s, &c, "access-scout", RunScout, "agent:access-scout")
	for _, d := range RequiredDimensions {
		if err := s.AddEvidence(ctx, 1, &DemandEvidence{DemandCaseID: c.ID, Dimension: d, Kind: EvidenceSupport, TruthStatus: TruthQuoted, Title: d, SourceURI: "https://official.example/" + d, ObservedAt: &now, RunID: "access-scout", SnapshotID: scoutSID}); err != nil {
			t.Fatal(err)
		}
	}
	counterSID := seedEvidenceSnapshot(t, s, &c, "access-falsifier", RunFalsifier, "agent:access-falsifier")
	if err := s.AddEvidence(ctx, 1, &DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: "counter", SourceURI: "https://counter.example", ObservedAt: &now, RunID: "access-falsifier", SnapshotID: counterSID}); err != nil {
		t.Fatal(err)
	}
	sid := seedAccessSnapshot(t, s, &c, "reality-block")
	if err := s.RecordDataAccess(ctx, 1, &DataAccessRecord{DemandCaseID: c.ID, FieldName: "fees", Status: AccessRequiresOwner, RequiredScope: "read_fees", DecisionPurpose: "核验费用", RefusalImpact: "停止", SourceURI: "https://official.example", RunID: "reality-block", SnapshotID: sid, AccessMode: "read_only", PreflightRequired: true}); err != nil {
		t.Fatal(err)
	}
	v, err := s.Evaluate(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != VerdictEvidenceMissing {
		t.Fatalf("required access crossed gate: %+v", v)
	}
	sid2 := seedAccessSnapshot(t, s, &c, "reality-available")
	if err := s.RecordDataAccess(ctx, 1, &DataAccessRecord{DemandCaseID: c.ID, FieldName: "fees", Status: AccessAvailable, DecisionPurpose: "费用已可读取", RefusalImpact: "无", SourceURI: "https://official.example", RunID: "reality-available", SnapshotID: sid2, AccessMode: "read_only", PreflightRequired: true}); err != nil {
		t.Fatal(err)
	}
	v, err = s.Evaluate(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != VerdictExperimentReady {
		t.Fatalf("latest available status did not supersede unknown: %+v", v)
	}
	cards, err := s.PermissionRequests(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 0 {
		t.Fatalf("stale permission request survived available status: %+v", cards)
	}
	card, err := s.DecisionCard(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(card.NextAuthorityOrCost, "read_fees") {
		t.Fatalf("decision card retained stale permission: %+v", card)
	}
}

func TestWriteCapableScopeIsRejected(t *testing.T) {
	s := accessService(t)
	ctx := context.Background()
	c := DemandCase{OwnerID: 1, Region: "US", Consumer: "可观察消费者", NeedScenario: "场景", SalesChannel: "marketplace"}
	if err := s.Create(ctx, &c); err != nil {
		t.Fatal(err)
	}
	sid := seedAccessSnapshot(t, s, &c, "reality-write")
	err := s.RecordDataAccess(ctx, 1, &DataAccessRecord{DemandCaseID: c.ID, FieldName: "catalog", Status: AccessRequiresOwner, RequiredScope: "Listings write", DecisionPurpose: "读取目录", RefusalImpact: "停止", SourceURI: "https://official.example", RunID: "reality-write", SnapshotID: sid, AccessMode: "read_only"})
	if err == nil {
		t.Fatal("write-capable scope must be rejected")
	}
}
