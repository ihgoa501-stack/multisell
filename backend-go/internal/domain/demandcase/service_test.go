package demandcase

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &DemandCase{}, &DemandEvidence{}, &DemandVerdict{}, &ResearchBatch{}, &ResearchSnapshot{}, &DataAccessRecord{})
	return NewService(db, zap.NewNop())
}

func seedEvidenceSnapshot(t *testing.T, s *Service, c *DemandCase, run, runType, collector string) int64 {
	t.Helper()
	b := ResearchBatch{BatchKey: "evidence-" + run, OwnerID: c.OwnerID}
	if err := s.db.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	snap := ResearchSnapshot{BatchID: b.ID, OwnerID: c.OwnerID, DemandCaseID: c.ID, RunID: run, RunType: runType, Collector: collector, SourceURI: "https://official.example/" + run, CollectedAt: time.Now(), RawPayload: "{}", RawSHA256: payloadHash([]byte("{}"))}
	if err := s.db.Create(&snap).Error; err != nil {
		t.Fatal(err)
	}
	return snap.ID
}

func TestEvaluateRequiresEveryMarketDimensionAndIndependentCounterevidence(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	c := &DemandCase{OwnerID: 7, Region: "DE", Consumer: "城市养猫家庭", NeedScenario: "短途出行饮水", SalesChannel: "独立站"}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	scoutSnapshot := seedEvidenceSnapshot(t, svc, c, "scout-1", RunScout, "agent:scout")
	for _, dimension := range RequiredDimensions {
		e := DemandEvidence{DemandCaseID: c.ID, Dimension: dimension, Kind: EvidenceSupport, TruthStatus: TruthQuoted, Title: dimension + " evidence", SourceURI: "https://example.com/" + dimension, ObservedAt: &now, RunID: "scout-1", SnapshotID: scoutSnapshot}
		if err := svc.AddEvidence(ctx, 7, &e); err != nil {
			t.Fatal(err)
		}
	}
	v, err := svc.Evaluate(ctx, c.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != VerdictEvidenceMissing {
		t.Fatalf("without independent counterevidence got %q", v.Status)
	}

	counterSnapshot := seedEvidenceSnapshot(t, svc, c, "falsifier-2", RunFalsifier, "agent:falsifier")
	counter := DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: "替代方案已经很便宜", SourceURI: "https://counter.example/source", ObservedAt: &now, RunID: "falsifier-2", SnapshotID: counterSnapshot}
	if err := svc.AddEvidence(ctx, 7, &counter); err != nil {
		t.Fatal(err)
	}
	v, err = svc.Evaluate(ctx, c.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != VerdictExperimentReady {
		t.Fatalf("complete evidence got %q blockers=%v", v.Status, v.Blockers)
	}
}

func TestEvaluateKeepsUnknownCriticalCostOrPermissionBlocked(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	c := &DemandCase{OwnerID: 7, Region: "US", Consumer: "新手犬主", NeedScenario: "航空箱训练", SalesChannel: "marketplace"}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, dimension := range RequiredDimensions {
		truth := TruthQuoted
		if dimension == DimensionProfit {
			truth = TruthUnknown
		}
		e := DemandEvidence{DemandCaseID: c.ID, Dimension: dimension, Kind: EvidenceSupport, TruthStatus: truth, Title: dimension, SourceURI: "https://example.com/" + dimension, ObservedAt: &now, RunID: "scout"}
		if err := svc.AddEvidence(ctx, 7, &e); err != nil {
			t.Fatal(err)
		}
	}
	counter := DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: "counter", SourceURI: "https://counter.example", ObservedAt: &now, RunID: "falsifier"}
	if err := svc.AddEvidence(ctx, 7, &counter); err != nil {
		t.Fatal(err)
	}
	v, err := svc.Evaluate(ctx, c.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != VerdictEvidenceMissing {
		t.Fatalf("unknown profit must block, got %q", v.Status)
	}
}

func TestDecisionCardUsesFactsAndNeverClaimsDemandIsProven(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	c := &DemandCase{OwnerID: 7, Region: "JP", Consumer: "老年猫主人", NeedScenario: "喂药", SalesChannel: "marketplace", StopCondition: "无法取得完整费用时停止"}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	card, err := svc.DecisionCard(ctx, c.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if card.NotProven == "" || card.StopCondition != c.StopCondition {
		t.Fatalf("incomplete owner card: %+v", card)
	}
	if card.Verdict != VerdictEvidenceMissing {
		t.Fatalf("new case verdict=%q", card.Verdict)
	}
}

func TestOrdinaryEvidenceCannotClaimActual(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	c := &DemandCase{OwnerID: 7, Region: "FR", Consumer: "犬主", NeedScenario: "出行", SalesChannel: "marketplace"}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	err := svc.AddEvidence(ctx, 7, &DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionDemand, Kind: EvidenceSupport, TruthStatus: TruthActual, Title: "claimed actual", SourceURI: "https://example.com", ObservedAt: &now, RunID: "scout"})
	if err == nil {
		t.Fatal("ordinary evidence input must not declare actual")
	}
}

func TestConflictBlocksExperimentReady(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	c := &DemandCase{OwnerID: 7, Region: "CA", Consumer: "猫主人", NeedScenario: "旅行", SalesChannel: "marketplace"}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, dimension := range RequiredDimensions {
		if err := svc.AddEvidence(ctx, 7, &DemandEvidence{DemandCaseID: c.ID, Dimension: dimension, Kind: EvidenceSupport, TruthStatus: TruthQuoted, Title: dimension, SourceURI: "https://example.com/" + dimension, ObservedAt: &now, RunID: "scout"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.AddEvidence(ctx, 7, &DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: "counter", SourceURI: "https://counter.example", ObservedAt: &now, RunID: "falsifier"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddEvidence(ctx, 7, &DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionPayment, Kind: EvidenceConflict, TruthStatus: TruthQuoted, Title: "payment scope conflicts", SourceURI: "https://conflict.example", ObservedAt: &now, RunID: "reality"}); err != nil {
		t.Fatal(err)
	}
	v, err := svc.Evaluate(ctx, c.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != VerdictEvidenceMissing {
		t.Fatalf("unresolved conflict must block, got %q", v.Status)
	}
}

func TestDecisionCardDoesNotPromoteMockOrInferredEvidence(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	c := &DemandCase{OwnerID: 7, Region: "GB", Consumer: "犬主", NeedScenario: "训练", SalesChannel: "marketplace"}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	for _, truth := range []string{TruthMock, TruthInferred} {
		if err := svc.AddEvidence(ctx, 7, &DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionDemand, Kind: EvidenceSupport, TruthStatus: truth, Title: truth, RunID: truth}); err != nil {
			t.Fatal(err)
		}
	}
	card, err := svc.DecisionCard(ctx, c.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if card.Proven != "尚无足够的可核验市场证据" {
		t.Fatalf("mock/inferred promoted: %q", card.Proven)
	}
}

func TestManualFatalCounterCannotRejectWithoutIndependentSnapshot(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	c := DemandCase{OwnerID: 7, Region: "DE", Consumer: "消费者", NeedScenario: "场景", SalesChannel: "channel"}
	if err := svc.Create(ctx, &c); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := svc.AddEvidence(ctx, 7, &DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: "manual fatal", SourceURI: "https://counter.example", ObservedAt: &now, RunID: "typed-string", Fatal: true}); err != nil {
		t.Fatal(err)
	}
	v, err := svc.Evaluate(ctx, c.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status == VerdictRejected {
		t.Fatalf("manual fatal counter bypassed collector gate: %+v", v)
	}
}

func TestEmptyFalsifierSnapshotCannotCombineWithManualCounter(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	c := DemandCase{OwnerID: 7, Region: "US", Consumer: "消费者", NeedScenario: "场景", SalesChannel: "channel"}
	if err := svc.Create(ctx, &c); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	scout := seedEvidenceSnapshot(t, svc, &c, "combo-scout", RunScout, "agent:scout")
	for _, d := range RequiredDimensions {
		if err := svc.AddEvidence(ctx, 7, &DemandEvidence{DemandCaseID: c.ID, Dimension: d, Kind: EvidenceSupport, TruthStatus: TruthQuoted, Title: d, SourceURI: "https://support.example/" + d, ObservedAt: &now, RunID: "combo-scout", SnapshotID: scout}); err != nil {
			t.Fatal(err)
		}
	}
	_ = seedEvidenceSnapshot(t, svc, &c, "empty-falsifier", RunFalsifier, "agent:falsifier")
	if err := svc.AddEvidence(ctx, 7, &DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: "manual counter", SourceURI: "https://counter.example", ObservedAt: &now, RunID: "manual"}); err != nil {
		t.Fatal(err)
	}
	v, err := svc.Evaluate(ctx, c.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status == VerdictExperimentReady {
		t.Fatalf("empty falsifier snapshot combined with manual counter: %+v", v)
	}
}
