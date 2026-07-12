package demandcase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func opportunityService(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &DemandCase{}, &DemandEvidence{}, &DemandVerdict{}, &ResearchSnapshot{}, &ResearchBatch{}, &MarketOwnerDecision{}, &ProductOpportunity{}, &ProductOpportunityDecision{})
	return NewService(db, zap.NewNop())
}

func readyMarket(t *testing.T, s *Service, ownerID int64) *DemandCase {
	t.Helper()
	ctx, now := context.Background(), time.Now().UTC()
	base := ResearchResult{BatchKey: "market-ready", Region: "DE", Consumer: "城市养猫家庭", NeedScenario: "短途出行饮水", SalesChannel: "独立站", TargetLocale: "de-DE", StopCondition: "无法核验收款或完整费用时停止", CollectedAt: now}
	importRun := func(id, typ, source string, findings []ResearchFinding) *DemandCase {
		raw := []byte(`{"run":"` + id + `"}`)
		in := base
		in.RunID, in.RunType, in.SourceURI, in.RawPayload, in.RawSHA256, in.Findings = id, typ, source, raw, hashPayload(raw), findings
		row, err := s.ImportResearchResult(ctx, ownerID, in)
		if err != nil {
			t.Fatal(err)
		}
		return row
	}
	c := importRun("scout-owner", RunScout, "https://research.example/scout", completeFindings(EvidenceSupport))
	importRun("counter-owner", RunFalsifier, "https://research.example/counter", []ResearchFinding{{Dimension: DimensionDemand, TruthStatus: TruthQuoted, Title: "低价替代品可能足够"}})
	importRun("reality-owner", RunDataReality, "https://research.example/reality", nil)
	if v, err := s.Evaluate(ctx, c.ID, ownerID); err != nil || v.Status != VerdictExperimentReady {
		t.Fatalf("evaluate=%+v err=%v", v, err)
	}
	return c
}

func TestOwnerSelectedMarketCreatesAndApprovesProductOpportunity(t *testing.T) {
	s, ctx := opportunityService(t), context.Background()
	c := readyMarket(t, s, 7)
	marketDecision, err := s.DecideMarket(ctx, c.ID, 7, MarketDecisionInput{Decision: MarketDecisionSelected, Reason: "八维研究已达到审议门槛，先形成商品机会", IdempotencyKey: "market-select-0001"})
	if err != nil {
		t.Fatal(err)
	}
	in := ProductOpportunityInput{DemandCaseID: c.ID, MarketDecisionID: marketDecision.ID, Title: "便携猫咪饮水方案", ConsumerProblem: "短途出行时饮水不便", ProductThesis: "轻量防漏饮水器", TargetChannel: "独立站", ValueHypothesis: "减少携带和漏水麻烦", PriceHypothesis: "先验证可接受价格区间", SourceURI: "https://research.example/scout", TruthStatus: TruthQuoted, StrongestCounterevidence: "普通水碗价格更低", Unknowns: []string{"获客成本", "退货率"}, StopCondition: "无法取得完整费用时停止"}
	o, err := s.CreateProductOpportunity(ctx, 7, in)
	if err != nil {
		t.Fatal(err)
	}
	o, blockers, err := s.EvaluateProductOpportunity(ctx, o.ID, 7)
	if err != nil || len(blockers) != 0 || o.Status != OpportunityReady {
		t.Fatalf("opportunity=%+v blockers=%v err=%v", o, blockers, err)
	}
	d, err := s.DecideProductOpportunity(ctx, o.ID, 7, OpportunityDecisionInput{Decision: OpportunityApproved, Reason: "批准进入货源研究，不授权采购或发布", IdempotencyKey: "opportunity-approve-0001", Version: o.Version})
	if err != nil || d.ContentHash != o.ContentHash {
		t.Fatalf("decision=%+v err=%v", d, err)
	}
	stored, err := s.GetProductOpportunity(ctx, o.ID, 7)
	if err != nil || stored.Status != OpportunityApproved {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
}

func TestProductOpportunityRequiresLatestSelectedMarket(t *testing.T) {
	s, ctx := opportunityService(t), context.Background()
	c := readyMarket(t, s, 9)
	_, err := s.CreateProductOpportunity(ctx, 9, ProductOpportunityInput{DemandCaseID: c.ID, MarketDecisionID: 999})
	if !errors.Is(err, ErrMarketNotSelected) {
		t.Fatalf("err=%v", err)
	}
	selected, err := s.DecideMarket(ctx, c.ID, 9, MarketDecisionInput{Decision: MarketDecisionSelected, Reason: "select", IdempotencyKey: "market-select-0002"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DecideMarket(ctx, c.ID, 9, MarketDecisionInput{Decision: MarketDecisionRejected, Reason: "different", IdempotencyKey: "market-select-0002"}); !errors.Is(err, ErrDecisionConflict) {
		t.Fatalf("idempotency conflict err=%v", err)
	}
	if _, err := s.DecideMarket(ctx, c.ID, 9, MarketDecisionInput{Decision: MarketDecisionPaused, Reason: "pause", IdempotencyKey: "market-pause-0001"}); err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateProductOpportunity(ctx, 9, ProductOpportunityInput{DemandCaseID: c.ID, MarketDecisionID: selected.ID})
	if !errors.Is(err, ErrMarketNotSelected) {
		t.Fatalf("superseded selection err=%v", err)
	}
}

func TestOpportunityApprovalRequiresReadyStateAndCurrentVersion(t *testing.T) {
	s, ctx := opportunityService(t), context.Background()
	c := readyMarket(t, s, 11)
	selected, _ := s.DecideMarket(ctx, c.ID, 11, MarketDecisionInput{Decision: MarketDecisionSelected, Reason: "select", IdempotencyKey: "market-select-0011"})
	o, err := s.CreateProductOpportunity(ctx, 11, ProductOpportunityInput{DemandCaseID: c.ID, MarketDecisionID: selected.ID, Title: "incomplete"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.DecideProductOpportunity(ctx, o.ID, 11, OpportunityDecisionInput{Decision: OpportunityApproved, Reason: "approve", IdempotencyKey: "approve-incomplete", Version: o.Version})
	if !errors.Is(err, ErrOpportunityNotReady) {
		t.Fatalf("err=%v", err)
	}
}

func TestOwnerCannotSelectMarketFromStaleVerdict(t *testing.T) {
	s, ctx := opportunityService(t), context.Background()
	c := readyMarket(t, s, 13)
	now := time.Now().UTC().Add(time.Second)
	if err := s.db.Create(&DemandEvidence{DemandCaseID: c.ID, Dimension: DimensionPayment, Kind: EvidenceConflict, TruthStatus: TruthUnknown, Title: "new payment conflict", RunID: "new-reality", SnapshotID: 999, CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	_, err := s.DecideMarket(ctx, c.ID, 13, MarketDecisionInput{Decision: MarketDecisionSelected, Reason: "stale select", IdempotencyKey: "market-select-stale"})
	if !errors.Is(err, ErrDecisionConflict) {
		t.Fatalf("stale verdict decision err=%v", err)
	}
}
