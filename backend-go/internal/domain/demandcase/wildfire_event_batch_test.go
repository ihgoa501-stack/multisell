package demandcase

import (
	"context"
	"strings"
	"testing"
)

func TestReviewedWildfireEventBatchIsIdempotentAndRejectsHypothesis(t *testing.T) {
	s := problemService(t)
	for i := 0; i < 2; i++ {
		out, err := s.ImportReviewedWildfireEventBatch(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Problems) != 1 || out.StatusCounts[ProblemRejected] != 1 || out.StatusCounts[ProblemSurvives] != 0 {
			t.Fatalf("unexpected outcome %+v", out)
		}
		p := out.Problems[0]
		if p.ResidualBarrierStatus != ResidualBarrierNotConfirmed || p.Region != "US-CA-HOOPA" {
			t.Fatalf("wrong event case %+v", p)
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
	if problems != 1 || evidence != 2 {
		t.Fatalf("duplicate import created rows: problems=%d evidence=%d", problems, evidence)
	}
	var counter ProblemEvidence
	if err := s.db.Where("kind = ?", EvidenceCounter).First(&counter).Error; err != nil {
		t.Fatal(err)
	}
	if counter.Title != "Monument Fire timing and existing substitution do not confirm the residual barrier" {
		t.Fatalf("counter title overstates evidence: %s", counter.Title)
	}
	if len(counter.RawPayload) < 500 || !strings.Contains(counter.RawPayload, "artifact_path") || !strings.Contains(counter.RawPayload, "public_space_audit") {
		t.Fatalf("counter payload is not the complete reviewed audit: %s", counter.RawPayload)
	}
}
