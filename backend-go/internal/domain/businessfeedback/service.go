package businessfeedback

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"gorm.io/gorm"
)

var ErrInvalid = errors.New("invalid business feedback input")
var ErrNotAuthorized = errors.New("business action is not authorized by latest Owner decision")

type Service struct {
	db           *gorm.DB
	dispatcher   *command.Dispatcher
	policy       command.PolicyChecker
	capabilities map[string]struct{}
}

func NewService(db *gorm.DB, dispatcher *command.Dispatcher, policy command.PolicyChecker, capabilityIDs []string) *Service {
	c := make(map[string]struct{}, len(capabilityIDs))
	for _, id := range capabilityIDs {
		c[id] = struct{}{}
	}
	return &Service{db: db, dispatcher: dispatcher, policy: policy, capabilities: c}
}

type ownerDecisionRow struct {
	ID, OwnerID, DecisionCaseID                                            int64
	Decision, CapabilityID, CommandType, TargetType, TargetID, InputSHA256 string
	CreatedAt                                                              time.Time
}

func (s *Service) CreateAction(ctx context.Context, ownerID int64, in CreateActionInput) (*ControlledAction, error) {
	in.CapabilityID, in.CommandType, in.TargetType, in.TargetID, in.IdempotencyKey = strings.TrimSpace(in.CapabilityID), strings.TrimSpace(in.CommandType), strings.TrimSpace(in.TargetType), strings.TrimSpace(in.TargetID), strings.TrimSpace(in.IdempotencyKey)
	var inputValue any
	if json.Unmarshal(in.InputPayload, &inputValue) != nil {
		return nil, ErrInvalid
	}
	canonicalPayload, canonicalErr := json.Marshal(inputValue)
	if ownerID <= 0 || in.OwnerDecisionID <= 0 || in.ApprovalID <= 0 || in.CapabilityID == "" || in.CommandType == "" || in.TargetType == "" || in.TargetID == "" || in.IdempotencyKey == "" || len(in.IdempotencyKey) > 200 || canonicalErr != nil {
		return nil, ErrInvalid
	}
	if _, ok := s.capabilities[in.CapabilityID]; !ok {
		return nil, fmt.Errorf("%w: capability is not registered", ErrInvalid)
	}
	if in.CapabilityID != "command."+in.CommandType+".v1" {
		return nil, fmt.Errorf("%w: capability does not bind the command", ErrInvalid)
	}
	registered := false
	if s.dispatcher != nil {
		for _, t := range s.dispatcher.RegisteredTypes() {
			if t == in.CommandType {
				registered = true
				break
			}
		}
	}
	if !registered {
		return nil, fmt.Errorf("%w: command is not registered", ErrInvalid)
	}
	in.InputPayload = canonicalPayload
	h := sha256.Sum256(canonicalPayload)
	digest := hex.EncodeToString(h[:])
	var out ControlledAction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var d ownerDecisionRow
		if err := tx.Table("business_owner_decision").Where("id=? AND owner_id=?", in.OwnerDecisionID, ownerID).Take(&d).Error; err != nil {
			return ErrNotAuthorized
		}
		var latest ownerDecisionRow
		if err := tx.Table("business_owner_decision").Where("decision_case_id=? AND owner_id=?", d.DecisionCaseID, ownerID).Order("created_at DESC, id DESC").Take(&latest).Error; err != nil || latest.ID != d.ID || d.Decision != "selected" {
			return ErrNotAuthorized
		}
		if d.CapabilityID != in.CapabilityID || d.CommandType != in.CommandType || d.TargetType != in.TargetType || d.TargetID != in.TargetID || d.InputSHA256 != digest {
			return fmt.Errorf("%w: action differs from frozen Owner authorization", ErrNotAuthorized)
		}
		var existing ControlledAction
		if err := tx.Where("owner_id=? AND idempotency_key=?", ownerID, in.IdempotencyKey).Take(&existing).Error; err == nil {
			if existing.InputSHA256 != digest || existing.OwnerDecisionID != d.ID {
				return fmt.Errorf("idempotency conflict")
			}
			out = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		out = ControlledAction{OwnerID: ownerID, OwnerDecisionID: d.ID, ApprovalID: in.ApprovalID, CapabilityID: in.CapabilityID, CommandType: in.CommandType, TargetType: in.TargetType, TargetID: in.TargetID, IdempotencyKey: in.IdempotencyKey, InputPayload: append(json.RawMessage(nil), in.InputPayload...), InputSHA256: digest, Status: "approved_pending_execution"}
		return tx.Create(&out).Error
	})
	return &out, err
}

func (s *Service) List(ctx context.Context, ownerID, decisionID int64) ([]ControlledAction, error) {
	var rows []ControlledAction
	q := s.db.WithContext(ctx).Where("owner_id=?", ownerID)
	if decisionID > 0 {
		q = q.Where("owner_decision_id=?", decisionID)
	}
	return rows, q.Order("id DESC").Find(&rows).Error
}

func (s *Service) Get(ctx context.Context, ownerID, id int64) (*ActionDetail, error) {
	var d ActionDetail
	if err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", id, ownerID).Take(&d.Action).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("controlled_action_id=? AND owner_id=?", id, ownerID).Order("id").Find(&d.Observations).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("controlled_action_id=? AND owner_id=?", id, ownerID).Order("id").Find(&d.NextRecommendations).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Service) Execute(ctx context.Context, ownerID, id int64) (*ControlledAction, error) {
	var a ControlledAction
	if err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", id, ownerID).Take(&a).Error; err != nil {
		return nil, err
	}
	if a.Status == "executing" {
		now := time.Now().UTC()
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&ControlledAction{}).Where("id=? AND owner_id=? AND status=?", id, ownerID, "executing").Updates(map[string]interface{}{"status": "reconcile_required", "failure_message": "execution interrupted before a durable success receipt", "executed_at": now})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected != 1 {
				return fmt.Errorf("action execution state changed concurrently")
			}
			return tx.Create(&operationlog.OperationLog{Module: "businessfeedback", Action: "business_action.execute", ResourceID: fmt.Sprint(id), Operator: fmt.Sprint(ownerID), Result: "reconcile_required", TriggerType: "recovery", ApprovalID: &a.ApprovalID, EntityType: "business_controlled_action", EntityID: id, CorrelationID: fmt.Sprintf("business-action:%d", id)}).Error
		})
		if err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", id, ownerID).Take(&a).Error; err != nil {
			return nil, err
		}
		return &a, nil
	}
	if a.Status != "approved_pending_execution" {
		return &a, nil
	}
	var d ownerDecisionRow
	if err := s.db.WithContext(ctx).Table("business_owner_decision").Where("id=? AND owner_id=?", a.OwnerDecisionID, ownerID).Take(&d).Error; err != nil {
		return nil, ErrNotAuthorized
	}
	var latest ownerDecisionRow
	if err := s.db.WithContext(ctx).Table("business_owner_decision").Where("decision_case_id=? AND owner_id=?", d.DecisionCaseID, ownerID).Order("created_at DESC, id DESC").Take(&latest).Error; err != nil || latest.ID != d.ID || d.Decision != "selected" {
		return nil, ErrNotAuthorized
	}
	if d.CapabilityID != a.CapabilityID || d.CommandType != a.CommandType || d.TargetType != a.TargetType || d.TargetID != a.TargetID || d.InputSHA256 != a.InputSHA256 {
		return nil, fmt.Errorf("%w: stored action differs from frozen Owner authorization", ErrNotAuthorized)
	}
	var input map[string]interface{}
	if err := json.Unmarshal(a.InputPayload, &input); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&ControlledAction{}).Where("id=? AND owner_id=? AND status=?", id, ownerID, "approved_pending_execution").Update("status", "executing")
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return fmt.Errorf("action already claimed")
		}
		return tx.Create(&operationlog.OperationLog{Module: "businessfeedback", Action: "business_action.execute", ResourceID: fmt.Sprint(id), Operator: fmt.Sprint(ownerID), Result: "pending", TriggerType: "owner_approval", ApprovalID: &a.ApprovalID, EntityType: "business_controlled_action", EntityID: id, CorrelationID: fmt.Sprintf("business-action:%d", id)}).Error
	}); err != nil {
		return nil, err
	}
	action := command.AgentAction{ActionType: a.CommandType, Version: "v1", AgentID: "xiao_q", Actor: fmt.Sprint(ownerID), TargetType: a.TargetType, TargetID: a.TargetID, RiskLevel: command.RiskHigh, ApprovalRequired: true, ApprovalID: &a.ApprovalID, Mode: command.ModeProduction, IdempotencyKey: a.IdempotencyKey, CorrelationID: fmt.Sprintf("business-action:%d", a.ID), Input: input}
	result, err := s.dispatcher.DispatchSafe(ctx, action, s.policy)
	now := time.Now().UTC()
	updates := map[string]interface{}{"executed_at": now}
	if err != nil || result == nil || !result.Success {
		// Once dispatch starts, a missing success receipt is not proof that the
		// external side effect failed. Keep the idempotency key blocked until the
		// Owner reconciles the provider outcome.
		updates["status"] = "reconcile_required"
		if err != nil && result == nil && !errors.Is(err, command.ErrOutcomeUnknown) {
			updates["status"] = "failed"
		}
		if err != nil {
			updates["failure_message"] = err.Error()
		} else if result != nil {
			updates["failure_message"] = result.ErrorMessage
		}
	} else {
		updates["status"] = "succeeded"
		updates["command_business_id"] = result.BusinessID
	}
	if saveErr := s.db.WithContext(context.WithoutCancel(ctx)).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ControlledAction{}).Where("id=? AND owner_id=?", id, ownerID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Create(&operationlog.OperationLog{Module: "businessfeedback", Action: "business_action.execute", ResourceID: fmt.Sprint(id), Operator: fmt.Sprint(ownerID), Result: updates["status"].(string), TriggerType: "owner_approval", ApprovalID: &a.ApprovalID, EntityType: "business_controlled_action", EntityID: id, CorrelationID: fmt.Sprintf("business-action:%d", id)}).Error
	}); saveErr != nil {
		return nil, saveErr
	}
	if err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", id, ownerID).Take(&a).Error; err != nil {
		return nil, err
	}
	return &a, err
}

func (s *Service) CreateObservation(ctx context.Context, ownerID, actionID int64, in CreateObservationInput) (*ActionObservation, error) {
	in.EvidenceKind, in.SourceObjectType, in.TargetMetric = strings.TrimSpace(in.EvidenceKind), strings.TrimSpace(in.SourceObjectType), strings.TrimSpace(in.TargetMetric)
	if in.EvidenceKind != "support" && in.EvidenceKind != "counter" && in.EvidenceKind != "conflict" {
		return nil, ErrInvalid
	}
	if in.TargetMetric == "" || in.SourceObjectID <= 0 {
		return nil, ErrInvalid
	}
	var out ActionObservation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var action ControlledAction
		if err := tx.Where("id=? AND owner_id=? AND status=?", actionID, ownerID, "succeeded").Take(&action).Error; err != nil {
			return err
		}
		truth, manifest, observed, err := authoritativeSource(tx, ownerID, &action, in.SourceObjectType, in.SourceObjectID)
		if err != nil {
			return err
		}
		out = ActionObservation{OwnerID: ownerID, ControlledActionID: actionID, EvidenceKind: in.EvidenceKind, TruthStatus: truth, SourceObjectType: in.SourceObjectType, SourceObjectID: in.SourceObjectID, SourceManifestSHA256: manifest, ObservedAt: observed, TargetMetric: in.TargetMetric, TargetValue: strings.TrimSpace(in.TargetValue), ActualValue: strings.TrimSpace(in.ActualValue), ComparisonNote: strings.TrimSpace(in.ComparisonNote)}
		return tx.Create(&out).Error
	})
	return &out, err
}

func authoritativeSource(tx *gorm.DB, ownerID int64, action *ControlledAction, typ string, id int64) (string, string, time.Time, error) {
	var scope struct {
		IngestID          int64
		NormalizedOrderID *int64
	}
	if err := tx.Table("business_owner_decision AS d").
		Select("c.object_id AS ingest_id, i.normalized_order_id").
		Joins("JOIN business_decision_case AS c ON c.id=d.decision_case_id AND c.owner_id=d.owner_id").
		Joins("JOIN platform_order_ingest AS i ON i.id=c.object_id AND i.owner_id=c.owner_id").
		Where("d.id=? AND d.owner_id=? AND c.object_type='platform_order_ingest' AND i.truth_status='external_observed' AND i.processing_status='applied'", action.OwnerDecisionID, ownerID).
		Take(&scope).Error; err != nil || scope.NormalizedOrderID == nil {
		return "", "", time.Time{}, fmt.Errorf("feedback action has no authoritative order scope")
	}
	var row struct {
		OwnerID           int64
		TruthStatus, Hash string
		ObservedAt        time.Time
	}
	switch typ {
	case "platform_order_ingest":
		if id != scope.IngestID {
			return "", "", time.Time{}, fmt.Errorf("observation order is outside decision scope")
		}
		err := tx.Table("platform_order_ingest").Select("owner_id, truth_status, payload_sha256 AS hash, observed_at").Where("id=? AND owner_id=? AND truth_status='external_observed' AND processing_status='applied'", id, ownerID).Take(&row).Error
		return "external_observed", row.Hash, row.ObservedAt, err
	case "order_final_profit_version":
		err := tx.Table("order_final_profit_version").Select("owner_id, source_manifest_sha256 AS hash, finalized_at AS observed_at").Where("id=? AND owner_id=? AND order_id=?", id, ownerID, *scope.NormalizedOrderID).Take(&row).Error
		return "actual", row.Hash, row.ObservedAt, err
	case "cash_reconciliation":
		var reconciliation struct {
			OwnerID, PlatformSettlementIngestID int64
			Hash                                string
			ObservedAt                          time.Time
		}
		err := tx.Table("cash_reconciliation").Select("owner_id, platform_settlement_ingest_id, request_sha256 AS hash, reconciled_at AS observed_at").Where("id=? AND owner_id=? AND status='reconciled' AND reconciled_at IS NOT NULL", id, ownerID).Take(&reconciliation).Error
		if err != nil {
			return "actual", "", time.Time{}, err
		}
		var orderScope struct {
			Count                  int64
			MinOrderID, MaxOrderID int64
		}
		err = tx.Table("platform_settlement_fact_line").Select("COUNT(DISTINCT order_id) AS count, MIN(order_id) AS min_order_id, MAX(order_id) AS max_order_id").Where("ingest_id=?", reconciliation.PlatformSettlementIngestID).Take(&orderScope).Error
		if err != nil || orderScope.Count != 1 || orderScope.MinOrderID != *scope.NormalizedOrderID || orderScope.MaxOrderID != *scope.NormalizedOrderID {
			return "", "", time.Time{}, fmt.Errorf("cash reconciliation scope conflicts with decision order")
		}
		row.Hash, row.ObservedAt = reconciliation.Hash, reconciliation.ObservedAt
		return "actual", row.Hash, row.ObservedAt, err
	default:
		return "", "", time.Time{}, ErrInvalid
	}
}

func (s *Service) CreateRecommendation(ctx context.Context, ownerID, actionID int64, in CreateRecommendationInput) (*NextActionRecommendation, error) {
	in.RecommendationText, in.Rationale = strings.TrimSpace(in.RecommendationText), strings.TrimSpace(in.Rationale)
	if in.RecommendationText == "" || in.Rationale == "" {
		return nil, ErrInvalid
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&ActionObservation{}).Where("owner_id=? AND controlled_action_id=?", ownerID, actionID).Count(&count).Error; err != nil || count == 0 {
		return nil, fmt.Errorf("observed fact required before recommendation")
	}
	out := NextActionRecommendation{OwnerID: ownerID, ControlledActionID: actionID, RecommendationText: in.RecommendationText, Rationale: in.Rationale, TruthStatus: "inferred", Status: "proposed"}
	return &out, s.db.WithContext(ctx).Create(&out).Error
}
