package demandcase

import (
	"context"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func problemService(t *testing.T) *Service {
	db := dbtest.NewDB(t, &ProblemCase{}, &ProblemEvidence{}, &DemandCase{}, &DemandEvidence{}, &DemandVerdict{}, &ResearchBatch{}, &ResearchSnapshot{}, &DataAccessRecord{})
	return NewService(db, zap.NewNop())
}

func TestProblemFirstLeadHasNoChannelAndNeedsIndependentCounter(t *testing.T) {
	s := problemService(t)
	ctx := context.Background()
	p := ProblemCase{OwnerID: 1, ProblemKey: "test-independent-counter", Region: "GB", ObservablePopulation: "可观察人群", ProblemScenario: "具体问题", CurrentWorkaround: "现有办法", Responsibility: ResponsibilityConsumer, ProductSolvability: SolvabilityPlausible, HarmRisk: HarmLow, NextMinimumEvidence: "验证问题频率"}
	if err := s.CreateProblem(ctx, &p); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := s.AddProblemEvidence(ctx, 1, &ProblemEvidence{ProblemCaseID: p.ID, Kind: EvidenceSupport, Title: "official support", SourceURI: "https://official.example/support", ObservedAt: now, Collector: "agent:scout", RawPayload: "support", RawSHA256: hashPayload([]byte("support")), TrustedRun: true}); err != nil {
		t.Fatal(err)
	}
	v, err := s.EvaluateProblem(ctx, p.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if v != ProblemEvidenceMissing {
		t.Fatalf("without independent counter got %s", v)
	}
	if err := s.AddProblemEvidence(ctx, 1, &ProblemEvidence{ProblemCaseID: p.ID, Kind: EvidenceCounter, Title: "official counter", SourceURI: "https://official.example/counter", ObservedAt: now, Collector: "agent:falsifier", RawPayload: "counter", RawSHA256: hashPayload([]byte("counter")), TrustedRun: true}); err != nil {
		t.Fatal(err)
	}
	v, err = s.EvaluateProblem(ctx, p.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if v != ProblemSurvives {
		t.Fatalf("independently challenged problem got %s", v)
	}
}

func TestStructuralOrThirdPartyResponsibilityRejectsProductLead(t *testing.T) {
	s := problemService(t)
	ctx := context.Background()
	p := ProblemCase{OwnerID: 1, ProblemKey: "test-structural", Region: "GB", ObservablePopulation: "租客", ProblemScenario: "建筑结构潮湿", CurrentWorkaround: "联系房东", Responsibility: ResponsibilityLandlord, ProductSolvability: SolvabilityStructural, HarmRisk: HarmMedium, NextMinimumEvidence: "房东修缮记录"}
	if err := s.CreateProblem(ctx, &p); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, e := range []ProblemEvidence{{ProblemCaseID: p.ID, Kind: EvidenceSupport, Title: "housing survey", SourceURI: "https://official.example/support", ObservedAt: now, Collector: "agent:scout", RawPayload: "s", RawSHA256: hashPayload([]byte("s")), TrustedRun: true}, {ProblemCaseID: p.ID, Kind: EvidenceCounter, Title: "landlord duty", SourceURI: "https://official.example/counter", ObservedAt: now, Collector: "agent:falsifier", RawPayload: "c", RawSHA256: hashPayload([]byte("c")), TrustedRun: true}} {
		if err := s.AddProblemEvidence(ctx, 1, &e); err != nil {
			t.Fatal(err)
		}
	}
	v, err := s.EvaluateProblem(ctx, p.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if v != ProblemRejected {
		t.Fatalf("structural responsibility got %s", v)
	}
}

func TestClientSuppliedCollectorNamesCannotSatisfyIndependence(t *testing.T) {
	s := problemService(t)
	ctx := context.Background()
	p := ProblemCase{OwnerID: 1, ProblemKey: "test-untrusted", Region: "US", ObservablePopulation: "households", ProblemScenario: "scenario", CurrentWorkaround: "workaround", Responsibility: ResponsibilityConsumer, ProductSolvability: SolvabilityPlausible, HarmRisk: HarmLow, NextMinimumEvidence: "next"}
	if err := s.CreateProblem(ctx, &p); err != nil {
		t.Fatal(err)
	}
	for _, e := range []ProblemEvidence{{ProblemCaseID: p.ID, Kind: EvidenceSupport, Title: "s", SourceURI: "https://official.example/s", ObservedAt: time.Now(), Collector: "fake-scout", RawPayload: "s", RawSHA256: hashPayload([]byte("s"))}, {ProblemCaseID: p.ID, Kind: EvidenceCounter, Title: "c", SourceURI: "https://official.example/c", ObservedAt: time.Now(), Collector: "fake-falsifier", RawPayload: "c", RawSHA256: hashPayload([]byte("c"))}} {
		if err := s.AddProblemEvidence(ctx, 1, &e); err != nil {
			t.Fatal(err)
		}
	}
	status, err := s.EvaluateProblem(ctx, p.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if status != ProblemEvidenceMissing {
		t.Fatalf("untrusted identities produced %s", status)
	}
}

func TestOnlySurvivingProblemCanBecomeChannelCandidate(t *testing.T) {
	s := problemService(t)
	ctx := context.Background()
	p := ProblemCase{OwnerID: 1, ProblemKey: "test-promotion", Region: "US", ObservablePopulation: "可观察人群", ProblemScenario: "具体问题", CurrentWorkaround: "现有办法", Responsibility: ResponsibilityConsumer, ProductSolvability: SolvabilityPlausible, HarmRisk: HarmLow, NextMinimumEvidence: "渠道字段"}
	if err := s.CreateProblem(ctx, &p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PromoteProblem(ctx, p.ID, 1, "marketplace"); err == nil {
		t.Fatal("unevaluated problem must not become demand case")
	}
	p.Status = ProblemSurvives
	if err := s.db.Model(&p).Update("status", ProblemSurvives).Error; err != nil {
		t.Fatal(err)
	}
	d, err := s.PromoteProblem(ctx, p.ID, 1, "marketplace")
	if err != nil {
		t.Fatal(err)
	}
	if d.SalesChannel != "marketplace" || d.NeedScenario != p.ProblemScenario {
		t.Fatalf("bad promoted case %+v", d)
	}
}

func TestProblemEvidenceRejectsHashMismatch(t *testing.T) {
	s := problemService(t)
	p := ProblemCase{OwnerID: 1, ProblemKey: "test-hash", Region: "US", ObservablePopulation: "households", ProblemScenario: "smoke exposure", CurrentWorkaround: "stay indoors", Responsibility: ResponsibilityShared, ProductSolvability: SolvabilityPartial, HarmRisk: HarmLow, NextMinimumEvidence: "local observation"}
	if err := s.CreateProblem(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	e := ProblemEvidence{ProblemCaseID: p.ID, Kind: EvidenceSupport, Title: "source", SourceURI: "https://official.example", ObservedAt: time.Now(), Collector: "agent:scout", RawPayload: "actual", RawSHA256: hashPayload([]byte("different"))}
	if err := s.AddProblemEvidence(context.Background(), 1, &e); err == nil {
		t.Fatal("mismatched evidence hash must be rejected")
	}
}
