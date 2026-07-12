package demandcase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func researchService(t *testing.T) *Service {
	db := dbtest.NewDB(t, &DemandCase{}, &DemandEvidence{}, &DemandVerdict{}, &ResearchSnapshot{}, &ResearchBatch{})
	return NewService(db, zap.NewNop())
}

func TestResearchContractRejectsMissingSourceAndHashMismatch(t *testing.T) {
	s := researchService(t)
	in := ResearchResult{BatchKey: "batch-1", RunID: "scout-1", RunType: RunScout, Region: "DE", Consumer: "城市养猫家庭", NeedScenario: "短途出行", SalesChannel: "独立站", TargetLocale: "de-DE", CollectedAt: time.Now(), RawPayload: []byte(`{"x":1}`), RawSHA256: "wrong"}
	if _, err := s.ImportResearchResult(context.Background(), 1, in); err == nil {
		t.Fatal("hash mismatch must fail")
	}
	in.RawSHA256 = hashPayload(in.RawPayload)
	in.SourceURI = ""
	if _, err := s.ImportResearchResult(context.Background(), 1, in); err == nil {
		t.Fatal("missing source must fail")
	}
}

func TestResearchBatchIsIdempotentAndIndependentRunsAreRequired(t *testing.T) {
	s := researchService(t)
	ctx := context.Background()
	now := time.Now()
	raw := []byte(`{"claim":"lead"}`)
	scout := ResearchResult{BatchKey: "real-1", RunID: "scout-1", RunType: RunScout, Region: "RU", Consumer: "Ozon可观察消费者", NeedScenario: "跨境商品需求待验证", SalesChannel: "Ozon", TargetLocale: "ru-RU", CollectedAt: now, SourceURI: "https://docs.ozon.com/global/en/analytics/analytics-and-metrics/analytics-tools/", RawPayload: raw, RawSHA256: hashPayload(raw), Findings: completeFindings(EvidenceSupport)}
	c1, err := s.ImportResearchResult(ctx, 1, scout)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := s.ImportResearchResult(ctx, 1, scout)
	if err != nil {
		t.Fatal(err)
	}
	if c1.ID != c2.ID {
		t.Fatalf("duplicate import created cases %d %d", c1.ID, c2.ID)
	}
	falsifier := scout
	falsifier.RunType = RunFalsifier
	falsifier.Findings = []ResearchFinding{{Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: "公开分析不能证明陌生买家最终付款"}}
	if _, err := s.ImportResearchResult(ctx, 1, falsifier); err == nil {
		t.Fatal("same run id must not count as independent falsification")
	}
}

func TestFirstPublicBatchProducesOneHonestPermissionCandidate(t *testing.T) {
	s := researchService(t)
	cards, err := s.RunFirstPublicResearchBatch(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 {
		t.Fatalf("cards=%d", len(cards))
	}
	if cards[0].Verdict != VerdictEvidenceMissing {
		t.Fatalf("public research must stop at evidence_missing, got %s", cards[0].Verdict)
	}
	again, err := s.RunFirstPublicResearchBatch(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 1 {
		t.Fatal("batch must be idempotent")
	}
}

func TestControlledThreeRunResearchCanReachReviewReady(t *testing.T) {
	s := researchService(t)
	ctx := context.Background()
	now := time.Now().UTC()
	base := ResearchResult{BatchKey: "controlled-1", Region: "DE", Consumer: "城市养猫家庭", NeedScenario: "短途出行饮水", SalesChannel: "独立站", TargetLocale: "de-DE", StopCondition: "无法核验收款或完整费用时停止", CollectedAt: now}
	importRun := func(runID, runType, source string, findings []ResearchFinding) *DemandCase {
		raw := []byte(`{"run":"` + runID + `"}`)
		in := base
		in.RunID, in.RunType, in.SourceURI, in.RawPayload, in.RawSHA256, in.Findings = runID, runType, source, raw, hashPayload(raw), findings
		item, err := s.ImportResearchResult(ctx, 1, in)
		if err != nil {
			t.Fatal(err)
		}
		return item
	}
	c := importRun("scout-controlled", RunScout, "https://source.example/scout", completeFindings(EvidenceSupport))
	importRun("falsifier-controlled", RunFalsifier, "https://source.example/falsifier", []ResearchFinding{{Dimension: DimensionDemand, TruthStatus: TruthQuoted, Title: "替代方案可能更便宜"}})
	importRun("reality-controlled", RunDataReality, "https://source.example/reality", nil)
	v, err := s.Evaluate(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != VerdictExperimentReady {
		t.Fatalf("controlled research got %q blockers=%v", v.Status, v.Blockers)
	}
}

func completeFindings(kind string) []ResearchFinding {
	out := make([]ResearchFinding, 0, len(RequiredDimensions))
	for _, d := range RequiredDimensions {
		out = append(out, ResearchFinding{Dimension: d, Kind: kind, TruthStatus: TruthQuoted, Title: d + " research"})
	}
	return out
}
func hashPayload(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
