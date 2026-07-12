package demandcase

import (
	"context"
	"testing"
	"time"
)

func TestReviewedProblemBatchIsIdempotentAndSelectsNothing(t *testing.T) {
	s := problemService(t)
	for i := 0; i < 2; i++ {
		out, err := s.ImportReviewedProblemBatch(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Problems) != 4 || out.StatusCounts[ProblemEvidenceMissing] != 2 || out.StatusCounts[ProblemRejected] != 2 || out.StatusCounts[ProblemSurvives] != 0 {
			t.Fatalf("unexpected outcome %+v", out)
		}
		if out.PaidDemand != "unknown" || out.SelectedItems != 0 || out.SelectedChannels != 0 {
			t.Fatalf("batch crossed business boundary %+v", out)
		}
	}
	var problems, evidence int64
	if err := s.db.Model(&ProblemCase{}).Count(&problems).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.db.Model(&ProblemEvidence{}).Count(&evidence).Error; err != nil {
		t.Fatal(err)
	}
	if problems != 4 || evidence != 8 {
		t.Fatalf("duplicate import created rows: problems=%d evidence=%d", problems, evidence)
	}
}

func TestReviewedProblemBatchRejectsKeyCollision(t *testing.T) {
	s := problemService(t)
	p := reviewedProblems()[0].caseData
	p.OwnerID = 1
	p.ProblemScenario = "tampered scenario"
	if err := s.CreateProblem(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportReviewedProblemBatch(context.Background(), 1); err == nil {
		t.Fatal("reviewed batch must reject a conflicting existing key")
	}
	var count int64
	if err := s.db.Model(&ProblemCase{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("failed transaction left partial rows: %d", count)
	}
}

func TestReviewedProblemBatchRejectsUntrustedEvidenceCollision(t *testing.T) {
	s := problemService(t)
	ctx := context.Background()
	item := reviewedProblems()[0]
	p := item.caseData
	p.OwnerID = 1
	if err := s.CreateProblem(ctx, &p); err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, 7, 11, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	e := ProblemEvidence{ProblemCaseID: p.ID, Kind: EvidenceSupport, Title: item.supportTitle, SourceURI: item.supportURI, ObservedAt: observedAt, Collector: "agent:problem_first_scout", RawPayload: item.supportPayload, RawSHA256: payloadHash([]byte(item.supportPayload))}
	if err := s.AddProblemEvidence(ctx, 1, &e); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportReviewedProblemBatch(ctx, 1); err == nil {
		t.Fatal("untrusted evidence collision must fail")
	}
	var problems, evidence int64
	s.db.Model(&ProblemCase{}).Count(&problems)
	s.db.Model(&ProblemEvidence{}).Count(&evidence)
	if problems != 1 || evidence != 1 {
		t.Fatalf("failed transaction left partial rows: problems=%d evidence=%d", problems, evidence)
	}
}
