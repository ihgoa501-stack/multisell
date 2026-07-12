package demandcase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	RunScout       = "scout_result"
	RunFalsifier   = "falsifier_result"
	RunDataReality = "data_reality_result"
)

type ResearchFinding struct {
	Dimension         string `json:"dimension"`
	Kind              string `json:"kind"`
	TruthStatus       string `json:"truth_status"`
	Title             string `json:"title"`
	Fatal             bool   `json:"fatal"`
	FieldName         string `json:"field_name,omitempty"`
	AccessStatus      string `json:"access_status,omitempty"`
	RequiredScope     string `json:"required_scope,omitempty"`
	DecisionPurpose   string `json:"decision_purpose,omitempty"`
	RefusalImpact     string `json:"refusal_impact,omitempty"`
	PreflightRequired bool   `json:"preflight_required,omitempty"`
}
type ResearchResult struct {
	BatchKey      string            `json:"batch_key"`
	RunID         string            `json:"run_id"`
	RunType       string            `json:"run_type"`
	Collector     string            `json:"collector"`
	Region        string            `json:"region"`
	Consumer      string            `json:"consumer"`
	NeedScenario  string            `json:"need_scenario"`
	SalesChannel  string            `json:"sales_channel"`
	StopCondition string            `json:"stop_condition"`
	SourceURI     string            `json:"source_uri"`
	CollectedAt   time.Time         `json:"collected_at"`
	RawPayload    []byte            `json:"raw_payload"`
	RawSHA256     string            `json:"raw_sha256"`
	Findings      []ResearchFinding `json:"findings"`
}

func payloadHash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }

func (s *Service) ImportResearchResult(ctx context.Context, ownerID int64, in ResearchResult) (*DemandCase, error) {
	if ownerID <= 0 || strings.TrimSpace(in.BatchKey) == "" || strings.TrimSpace(in.RunID) == "" || strings.TrimSpace(in.Collector) == "" || strings.TrimSpace(in.SourceURI) == "" || in.CollectedAt.IsZero() || len(in.RawPayload) == 0 || payloadHash(in.RawPayload) != in.RawSHA256 {
		return nil, errors.New("invalid research provenance or payload hash")
	}
	if in.RunType != RunScout && in.RunType != RunFalsifier && in.RunType != RunDataReality {
		return nil, errors.New("invalid research run type")
	}
	var result DemandCase
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var batch ResearchBatch
		if err := tx.Where("batch_key = ? AND owner_id = ?", in.BatchKey, ownerID).First(&batch).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			batch = ResearchBatch{BatchKey: in.BatchKey, OwnerID: ownerID}
			if err := tx.Create(&batch).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		var existing ResearchSnapshot
		if err := tx.Where("owner_id = ? AND run_id = ? AND run_type = ?", ownerID, in.RunID, in.RunType).First(&existing).Error; err == nil {
			if existing.BatchID != batch.ID || existing.RawSHA256 != in.RawSHA256 || existing.SourceURI != in.SourceURI || existing.Collector != in.Collector {
				return errors.New("duplicate run id has different provenance")
			}
			return tx.Where("id = ? AND owner_id = ?", existing.DemandCaseID, ownerID).First(&result).Error
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var other int64
		if err := tx.Model(&ResearchSnapshot{}).Where("owner_id = ? AND run_id = ? AND run_type <> ?", ownerID, in.RunID, in.RunType).Count(&other).Error; err != nil {
			return err
		}
		if other > 0 {
			return errors.New("research roles require independent run ids")
		}
		if in.RunType == RunScout {
			result = DemandCase{OwnerID: ownerID, Region: in.Region, Consumer: in.Consumer, NeedScenario: in.NeedScenario, SalesChannel: in.SalesChannel, StopCondition: in.StopCondition}
			if err := NewService(tx, s.logger).Create(ctx, &result); err != nil {
				return err
			}
		} else {
			if err := tx.Where("owner_id = ? AND region = ? AND consumer = ? AND need_scenario = ? AND sales_channel = ?", ownerID, in.Region, in.Consumer, in.NeedScenario, in.SalesChannel).First(&result).Error; err != nil {
				return errors.New("scout result must be imported before counter or data reality")
			}
		}
		if in.RunType == RunFalsifier {
			var scout ResearchSnapshot
			if err := tx.Where("demand_case_id = ? AND run_type = ?", result.ID, RunScout).Order("id DESC").First(&scout).Error; err != nil {
				return errors.New("falsifier requires a scout snapshot")
			}
			if scout.Collector == in.Collector {
				return errors.New("falsifier must use an independent collector")
			}
		}
		snap := ResearchSnapshot{BatchID: batch.ID, OwnerID: ownerID, DemandCaseID: result.ID, RunID: in.RunID, RunType: in.RunType, Collector: in.Collector, SourceURI: in.SourceURI, CollectedAt: in.CollectedAt, RawPayload: string(in.RawPayload), RawSHA256: in.RawSHA256}
		if err := tx.Create(&snap).Error; err != nil {
			return err
		}
		for _, f := range in.Findings {
			if in.RunType == RunDataReality {
				r := DataAccessRecord{DemandCaseID: result.ID, FieldName: f.FieldName, Status: f.AccessStatus, RequiredScope: f.RequiredScope, DecisionPurpose: f.DecisionPurpose, RefusalImpact: f.RefusalImpact, SourceURI: in.SourceURI, RunID: in.RunID, SnapshotID: snap.ID, AccessMode: "read_only", PreflightRequired: f.PreflightRequired}
				if err := NewService(tx, s.logger).RecordDataAccess(ctx, ownerID, &r); err != nil {
					return err
				}
				continue
			}
			kind := f.Kind
			if in.RunType == RunFalsifier {
				kind = EvidenceCounter
			}
			if in.RunType == RunScout && kind != EvidenceSupport {
				return errors.New("scout may only submit support leads")
			}
			e := DemandEvidence{DemandCaseID: result.ID, Dimension: f.Dimension, Kind: kind, TruthStatus: f.TruthStatus, Title: f.Title, SourceURI: in.SourceURI, ObservedAt: &in.CollectedAt, RunID: in.RunID, SnapshotID: snap.ID, Fatal: f.Fatal}
			if err := NewService(tx, s.logger).AddEvidence(ctx, ownerID, &e); err != nil {
				return err
			}
		}
		return nil
	})
	return &result, err
}
