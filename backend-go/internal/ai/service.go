package ai

import (
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides AI trace/action business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	traces *TraceWriter
}

// NewService creates a new AI service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
		traces: NewTraceWriter(db, logger),
	}
}

// TraceWriter exposes the underlying trace writer for the orchestrator.
func (s *Service) TraceWriter() *TraceWriter { return s.traces }

// ListTraces returns paginated AI traces.
func (s *Service) ListTraces(p *common.Pagination, f *TraceListFilter) ([]AITrace, int64, error) {
	q := s.db.Model(&AITrace{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("trace_id ILIKE ? OR agent_id ILIKE ? OR decision_point ILIKE ?", like, like, like)
		}
		if f.AgentID != "" {
			q = q.Where("agent_id = ?", f.AgentID)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.DecisionPoint != "" {
			q = q.Where("decision_point = ?", f.DecisionPoint)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []AITrace
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetTrace returns full trace detail.
func (s *Service) GetTrace(traceID string) (*TraceDetail, error) {
	return s.traces.GetDetail(traceID)
}

// ListActions returns paginated unified actions.
func (s *Service) ListActions(p *common.Pagination, f *ActionListFilter) ([]UnifiedAction, int64, error) {
	q := s.db.Model(&UnifiedAction{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("title ILIKE ? OR description ILIKE ? OR action_type ILIKE ?", like, like, like)
		}
		if f.AgentID != "" {
			q = q.Where("agent_id = ?", f.AgentID)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.RiskLevel != "" {
			q = q.Where("risk_level = ?", f.RiskLevel)
		}
		if f.SquadID != "" {
			q = q.Where("squad_id = ?", f.SquadID)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []UnifiedAction
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetAction returns a single unified action.
func (s *Service) GetAction(id int64) (*UnifiedAction, error) {
	var a UnifiedAction
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateAction inserts a unified action in the "suggested" state.
func (s *Service) CreateAction(in *CreateActionInput) (*UnifiedAction, error) {
	risk := in.RiskLevel
	if risk == "" {
		risk = "medium"
	}
	requires := true
	if in.RequiresApproval != nil {
		requires = *in.RequiresApproval
	}
	payload := in.Payload
	if len(payload) == 0 {
		payload = nil
	}
	a := UnifiedAction{
		SourceTable:        in.SourceTable,
		SourceID:           in.SourceID,
		SourceType:         in.SourceType,
		TraceID:            in.TraceID,
		AgentID:            in.AgentID,
		SquadID:            in.SquadID,
		UserID:             in.UserID,
		ActionType:         in.ActionType,
		BusinessObjectType: in.BusinessObjectType,
		BusinessObjectID:   in.BusinessObjectID,
		Title:              in.Title,
		Description:        in.Description,
		Payload:            payload,
		BeforeSnapshot:     in.BeforeSnapshot,
		AfterSnapshot:      in.AfterSnapshot,
		RiskLevel:          risk,
		RequiresApproval:   requires,
		Status:             "suggested",
		Confidence:         in.Confidence,
		ProposedBy:         in.ProposedBy,
	}
	if err := s.db.Create(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// ApproveAction transitions an action to "approved".
func (s *Service) ApproveAction(id int64, operator, _ string) (*UnifiedAction, error) {
	return s.transitionAction(id, "approved", map[string]interface{}{
		"approved_by": operator,
		"approved_at": nowPtr(),
	}, "suggested", "pending")
}

// RejectAction transitions an action to "rejected". Only allowed from
// "suggested" or "pending" — rejecting an already-approved action is not
// permitted (revoke via a separate flow instead).
func (s *Service) RejectAction(id int64, operator, reason string) (*UnifiedAction, error) {
	return s.transitionAction(id, "rejected", map[string]interface{}{
		"rejected_by":      operator,
		"rejected_at":      nowPtr(),
		"rejection_reason": reason,
	}, "suggested", "pending")
}

// ExecuteAction transitions an action through "executing" → "executed".
// For low-risk auto-approved actions (RequiresApproval=false) this can be
// called directly from "suggested"; otherwise the action must first be
// approved.
//
// Note: we rely on the action's status (not the RequiresApproval boolean)
// for the primary gate, because SQLite stores booleans as integers and the
// boolean round-trip can be unreliable across drivers. If an action is in
// "suggested" and was created with RequiresApproval=false (advisory agent),
// it executes directly; if it required approval it would be in "approved".
func (s *Service) ExecuteAction(id int64, operator, _ string) (*UnifiedAction, error) {
	var a UnifiedAction
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	// Allowed source states: "suggested" (auto-approved) or "approved".
	if a.Status != "suggested" && a.Status != "approved" {
		return nil, &InvalidTransitionError{From: a.Status, To: "executing"}
	}
	// If the action claims to require approval but hasn't been approved, refuse.
	// (Defensive — the status check above should already cover this.)
	if a.RequiresApproval && a.Status == "suggested" {
		return nil, ErrApprovalRequired
	}
	now := nowPtr()
	updates := map[string]interface{}{
		"status":       "executing",
		"executed_by":  operator,
		"executing_at": now,
	}
	if err := s.db.Model(&a).Updates(updates).Error; err != nil {
		return nil, err
	}
	// Simulate synchronous execution success.
	execNow := nowPtr()
	if err := s.db.Model(&a).Updates(map[string]interface{}{
		"status":      "executed",
		"executed_at": execNow,
		"updated_at":  execNow,
	}).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// FailAction marks an executing action as failed.
func (s *Service) FailAction(id int64, reason string) (*UnifiedAction, error) {
	return s.transitionAction(id, "failed", map[string]interface{}{
		"failed_at":        nowPtr(),
		"rejection_reason": reason,
	}, "executing")
}

// ReviewAction marks an executed action as reviewed.
func (s *Service) ReviewAction(id int64) (*UnifiedAction, error) {
	return s.transitionAction(id, "reviewed", map[string]interface{}{
		"reviewed_at": nowPtr(),
	}, "executed", "failed")
}

// Roster returns per-agent summaries for the cockpit.
func (s *Service) Roster() ([]AgentRosterSummary, error) {
	registry := DefaultRegistry()
	out := make([]AgentRosterSummary, 0, len(registry.Agents))
	for _, a := range registry.Agents {
		summary := AgentRosterSummary{
			AgentID:       a.ID,
			Name:          a.Name,
			Squad:         a.Squad,
			DecisionPoint: a.PrimaryDecisionPoint(),
			AutonomyLevel: a.Autonomy,
		}
		_ = s.db.Model(&AITrace{}).Where("agent_id = ?", a.ID).Count(&summary.TraceCount).Error
		_ = s.db.Model(&UnifiedAction{}).Where("agent_id = ?", a.ID).Count(&summary.ActionCount).Error
		_ = s.db.Model(&UnifiedAction{}).Where("agent_id = ? AND status IN ?", a.ID, []string{"suggested", "pending"}).Count(&summary.PendingCount).Error
		var conf struct{ Avg float64 }
		_ = s.db.Model(&AITrace{}).Where("agent_id = ? AND confidence IS NOT NULL", a.ID).
			Select("COALESCE(AVG(confidence),0) AS avg").Scan(&conf).Error
		summary.AvgConfidence = conf.Avg
		out = append(out, summary)
	}
	return out, nil
}

// transitionAction is the shared lifecycle helper.
func (s *Service) transitionAction(id int64, toStatus string, updates map[string]interface{}, allowedFrom ...string) (*UnifiedAction, error) {
	var a UnifiedAction
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	if len(allowedFrom) > 0 {
		ok := false
		for _, from := range allowedFrom {
			if strings.EqualFold(a.Status, from) {
				ok = true
				break
			}
		}
		if !ok {
			return nil, &InvalidTransitionError{From: a.Status, To: toStatus}
		}
	}
	updates["status"] = toStatus
	if err := s.db.Model(&a).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// InvalidTransitionError indicates an illegal action status transition.
type InvalidTransitionError struct {
	From string
	To   string
}

func (e *InvalidTransitionError) Error() string {
	return "invalid action transition: " + e.From + " → " + e.To
}

// ErrApprovalRequired signals that an action needs approval first.
var ErrApprovalRequired = &InvalidTransitionError{From: "suggested", To: "executing"}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}
