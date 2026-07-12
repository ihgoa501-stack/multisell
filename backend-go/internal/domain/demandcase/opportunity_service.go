package demandcase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrDecisionConflict    = errors.New("decision conflicts with current state")
	ErrMarketNotSelected   = errors.New("market is not selected by Owner")
	ErrOpportunityNotReady = errors.New("product opportunity is not ready for Owner decision")
)

func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (s *Service) DecideMarket(ctx context.Context, caseID, ownerID int64, in MarketDecisionInput) (*MarketOwnerDecision, error) {
	in.Decision, in.Reason, in.IdempotencyKey = strings.TrimSpace(in.Decision), strings.TrimSpace(in.Reason), strings.TrimSpace(in.IdempotencyKey)
	if ownerID <= 0 || caseID <= 0 || in.Reason == "" || in.IdempotencyKey == "" || (in.Decision != MarketDecisionSelected && in.Decision != MarketDecisionRejected && in.Decision != MarketDecisionPaused && in.Decision != MarketDecisionMoreEvidence) {
		return nil, errors.New("invalid Owner market decision")
	}
	var out MarketOwnerDecision
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing MarketOwnerDecision
		if err := tx.Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; err == nil {
			if existing.DemandCaseID != caseID || existing.Decision != in.Decision || existing.Reason != in.Reason {
				return ErrDecisionConflict
			}
			out = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var c DemandCase
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ?", caseID, ownerID).First(&c).Error; err != nil {
			return err
		}
		var verdict DemandVerdict
		if err := tx.Where("demand_case_id = ?", caseID).Order("id DESC").First(&verdict).Error; err != nil {
			return ErrDecisionConflict
		}
		if in.Decision == MarketDecisionSelected && verdict.Status != VerdictExperimentReady {
			return ErrDecisionConflict
		}
		var latestEvidenceID int64
		if err := tx.Model(&DemandEvidence{}).Where("demand_case_id = ?", caseID).Select("COALESCE(MAX(id), 0)").Scan(&latestEvidenceID).Error; err != nil {
			return err
		}
		if latestEvidenceID > verdict.EvidenceMaxID {
			return fmt.Errorf("%w: research evidence changed after the evaluated verdict", ErrDecisionConflict)
		}
		out = MarketOwnerDecision{DemandCaseID: caseID, OwnerID: ownerID, VerdictID: verdict.ID, Decision: in.Decision, Reason: in.Reason, EvidenceHash: hashJSON(struct {
			Case    DemandCase
			Verdict DemandVerdict
		}{c, verdict}), IdempotencyKey: in.IdempotencyKey}
		return tx.Create(&out).Error
	})
	return &out, err
}

func (s *Service) LatestMarketDecision(ctx context.Context, caseID, ownerID int64) (*MarketOwnerDecision, error) {
	if _, err := s.requireOwner(ctx, caseID, ownerID); err != nil {
		return nil, err
	}
	var row MarketOwnerDecision
	err := s.db.WithContext(ctx).Where("demand_case_id = ? AND owner_id = ?", caseID, ownerID).Order("id DESC").First(&row).Error
	return &row, err
}

func opportunityHash(in ProductOpportunityInput, ownerID int64) string {
	return hashJSON(struct {
		OwnerID int64
		Input   ProductOpportunityInput
	}{ownerID, in})
}

func (s *Service) CreateProductOpportunity(ctx context.Context, ownerID int64, in ProductOpportunityInput) (*ProductOpportunity, error) {
	if ownerID <= 0 || in.DemandCaseID <= 0 || in.MarketDecisionID <= 0 {
		return nil, errors.New("market decision is required")
	}
	var out ProductOpportunity
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var decision MarketOwnerDecision
		if err := tx.Where("id = ? AND demand_case_id = ? AND owner_id = ? AND decision = ?", in.MarketDecisionID, in.DemandCaseID, ownerID, MarketDecisionSelected).First(&decision).Error; err != nil {
			return ErrMarketNotSelected
		}
		var latest MarketOwnerDecision
		if err := tx.Where("demand_case_id = ? AND owner_id = ?", in.DemandCaseID, ownerID).Order("id DESC").First(&latest).Error; err != nil || latest.ID != decision.ID {
			return ErrMarketNotSelected
		}
		unknowns, _ := json.Marshal(in.Unknowns)
		out = ProductOpportunity{OwnerID: ownerID, DemandCaseID: in.DemandCaseID, MarketDecisionID: decision.ID, Title: strings.TrimSpace(in.Title), ConsumerProblem: strings.TrimSpace(in.ConsumerProblem), ProductThesis: strings.TrimSpace(in.ProductThesis), TargetChannel: strings.TrimSpace(in.TargetChannel), ValueHypothesis: strings.TrimSpace(in.ValueHypothesis), PriceHypothesis: strings.TrimSpace(in.PriceHypothesis), SourceURI: strings.TrimSpace(in.SourceURI), TruthStatus: strings.TrimSpace(in.TruthStatus), StrongestCounterevidence: strings.TrimSpace(in.StrongestCounterevidence), UnknownsJSON: string(unknowns), StopCondition: strings.TrimSpace(in.StopCondition), Status: OpportunityDraft, Version: 1, ContentHash: opportunityHash(in, ownerID), Unknowns: in.Unknowns}
		return tx.Create(&out).Error
	})
	return &out, err
}

func validateOpportunity(o ProductOpportunity) []string {
	missing := []string{}
	fields := map[string]string{"title": o.Title, "consumer_problem": o.ConsumerProblem, "product_thesis": o.ProductThesis, "target_channel": o.TargetChannel, "value_hypothesis": o.ValueHypothesis, "price_hypothesis": o.PriceHypothesis, "source_uri": o.SourceURI, "strongest_counterevidence": o.StrongestCounterevidence, "stop_condition": o.StopCondition}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, "missing:"+name)
		}
	}
	if o.TruthStatus != TruthQuoted && o.TruthStatus != TruthEstimated {
		missing = append(missing, "invalid:truth_status")
	}
	return missing
}

func (s *Service) EvaluateProductOpportunity(ctx context.Context, id, ownerID int64) (*ProductOpportunity, []string, error) {
	var out ProductOpportunity
	var blockers []string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ?", id, ownerID).First(&out).Error; err != nil {
			return err
		}
		if out.Status == OpportunityApproved || out.Status == OpportunityRejected {
			return ErrDecisionConflict
		}
		blockers = validateOpportunity(out)
		status := OpportunityReady
		if len(blockers) > 0 {
			status = OpportunityEvidenceMissing
		}
		if err := tx.Model(&out).Updates(map[string]any{"status": status}).Error; err != nil {
			return err
		}
		out.Status = status
		return nil
	})
	return &out, blockers, err
}

func (s *Service) DecideProductOpportunity(ctx context.Context, id, ownerID int64, in OpportunityDecisionInput) (*ProductOpportunityDecision, error) {
	in.Decision, in.Reason, in.IdempotencyKey = strings.TrimSpace(in.Decision), strings.TrimSpace(in.Reason), strings.TrimSpace(in.IdempotencyKey)
	if in.Reason == "" || in.IdempotencyKey == "" || (in.Decision != OpportunityApproved && in.Decision != OpportunityRejected && in.Decision != OpportunityPaused) {
		return nil, errors.New("invalid opportunity decision")
	}
	var out ProductOpportunityDecision
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing ProductOpportunityDecision
		if err := tx.Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; err == nil {
			if existing.OpportunityID != id || existing.Decision != in.Decision || existing.Reason != in.Reason {
				return ErrDecisionConflict
			}
			out = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var o ProductOpportunity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ?", id, ownerID).First(&o).Error; err != nil {
			return err
		}
		if o.Version != in.Version || (in.Decision == OpportunityApproved && o.Status != OpportunityReady) {
			return ErrOpportunityNotReady
		}
		out = ProductOpportunityDecision{OpportunityID: id, OwnerID: ownerID, Version: o.Version, ContentHash: o.ContentHash, Decision: in.Decision, Reason: in.Reason, IdempotencyKey: in.IdempotencyKey}
		if err := tx.Create(&out).Error; err != nil {
			return err
		}
		return tx.Model(&o).Updates(map[string]any{"status": in.Decision}).Error
	})
	return &out, err
}

func (s *Service) ListProductOpportunities(ctx context.Context, ownerID int64) ([]ProductOpportunity, error) {
	var rows []ProductOpportunity
	if err := s.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		_ = json.Unmarshal([]byte(rows[i].UnknownsJSON), &rows[i].Unknowns)
	}
	return rows, nil
}

func (s *Service) GetProductOpportunity(ctx context.Context, id, ownerID int64) (*ProductOpportunity, error) {
	var row ProductOpportunity
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).First(&row).Error; err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(row.UnknownsJSON), &row.Unknowns)
	return &row, nil
}

func (o ProductOpportunity) String() string {
	return fmt.Sprintf("opportunity:%d@v%d", o.ID, o.Version)
}
