package demandcase

import (
	"context"
	"errors"
	"sort"

	"gorm.io/gorm"
)

type ComparisonCandidate struct {
	Case                     DemandCase                  `json:"case"`
	Verdict                  *DemandVerdict              `json:"verdict"`
	OwnerDecision            *MarketOwnerDecision        `json:"owner_decision,omitempty"`
	EvidenceByDimension      map[string][]DemandEvidence `json:"evidence_by_dimension"`
	StrongestCounterevidence string                      `json:"strongest_counterevidence"`
	Unknowns                 []string                    `json:"unknowns"`
}

type MarketComparison struct {
	Dimensions []string              `json:"dimensions"`
	Candidates []ComparisonCandidate `json:"candidates"`
}

func (s *Service) Compare(ctx context.Context, ownerID int64, ids []int64) (*MarketComparison, error) {
	if ownerID <= 0 || len(ids) < 2 || len(ids) > 4 {
		return nil, errors.New("comparison requires 2 to 4 candidate markets")
	}
	seen := map[int64]bool{}
	result := &MarketComparison{Dimensions: append([]string(nil), RequiredDimensions...), Candidates: make([]ComparisonCandidate, 0, len(ids))}
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return nil, errors.New("comparison candidate ids must be unique")
		}
		seen[id] = true
		detail, err := s.Get(ctx, id, ownerID)
		if err != nil {
			return nil, err
		}
		candidate := ComparisonCandidate{Case: detail.Case, Verdict: detail.Verdict, EvidenceByDimension: map[string][]DemandEvidence{}, StrongestCounterevidence: "尚无独立反证", Unknowns: []string{}}
		for _, evidence := range detail.Evidence {
			candidate.EvidenceByDimension[evidence.Dimension] = append(candidate.EvidenceByDimension[evidence.Dimension], evidence)
			if evidence.Kind == EvidenceCounter && usableEvidence(evidence) {
				candidate.StrongestCounterevidence = evidence.Title
			}
			if evidence.TruthStatus == TruthUnknown || evidence.TruthStatus == TruthMock || evidence.TruthStatus == TruthInferred || evidence.Kind == EvidenceConflict {
				candidate.Unknowns = append(candidate.Unknowns, evidence.Dimension+":"+evidence.Title)
			}
		}
		if detail.Verdict != nil {
			candidate.Unknowns = append(candidate.Unknowns, detail.Verdict.Blockers...)
		}
		var decision MarketOwnerDecision
		if err := s.db.WithContext(ctx).Where("demand_case_id = ? AND owner_id = ?", id, ownerID).Order("id DESC").First(&decision).Error; err == nil {
			candidate.OwnerDecision = &decision
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		sort.Strings(candidate.Unknowns)
		candidate.Unknowns = uniqueStrings(candidate.Unknowns)
		result.Candidates = append(result.Candidates, candidate)
	}
	return result, nil
}

func uniqueStrings(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}
