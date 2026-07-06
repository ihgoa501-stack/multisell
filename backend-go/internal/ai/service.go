package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/aios/guardrails"
	"github.com/lingmirror/backend-go/internal/platform/actioncatalog"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides AI trace/action business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	traces *TraceWriter
	cmd    *command.Dispatcher
	cat    *actioncatalog.Catalog
	guard  *guardrails.Chain
}

// NewService creates a new AI service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
		traces: NewTraceWriter(db, logger),
	}
}

// WithDispatcher attaches the command dispatcher so that ExecuteAction can
// dispatch actions through registered command handlers.
func (s *Service) WithDispatcher(cmd *command.Dispatcher) *Service {
	s.cmd = cmd
	return s
}

// WithCatalog attaches the action catalog for production validation.
func (s *Service) WithCatalog(cat *actioncatalog.Catalog) *Service {
	s.cat = cat
	return s
}

// WithGuard attaches the guardrails chain for execution-level checks.
func (s *Service) WithGuard(g *guardrails.Chain) *Service {
	s.guard = g
	return s
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
		ExecutionMode:      in.ExecutionMode,
		IdempotencyKey:     in.IdempotencyKey,
	}
	if err := s.db.Create(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// ApproveAction transitions an action to "approved",
// storing the user ID if provided.
func (s *Service) ApproveAction(id int64, operator, _ string, userID *int64) (*UnifiedAction, error) {
	updates := map[string]interface{}{
		"approved_by": operator,
		"approved_at": nowPtr(),
	}
	if userID != nil {
		updates["approved_by_user_id"] = *userID
	}
	return s.transitionAction(id, "approved", updates, "suggested", "pending")
}

// RejectAction transitions an action to "rejected". Only allowed from
// "suggested" or "pending" — rejecting an already-approved action is not
// permitted (revoke via a separate flow instead).
func (s *Service) RejectAction(id int64, operator, reason string, _ *int64) (*UnifiedAction, error) {
	return s.transitionAction(id, "rejected", map[string]interface{}{
		"rejected_by":      operator,
		"rejected_at":      nowPtr(),
		"rejection_reason": reason,
	}, "suggested", "pending")
}

// ExecuteAction transitions an action through "executing" → "executed"/"failed" and
// dispatches the underlying business command via the CommandDispatcher.
//
// The enhanced flow:
//  1. Idempotency check — if the action has an idempotency key and is already
//     executed/reviewed, return the existing result as a no-op.
//  2. Validate against action catalog.
//  3. Check status allows execution (suggested/approved).
//  4. Check approval requirements.
//  5. Run guardrails chain (L4 execution guard) if configured.
//  6. Dry-run mode — skip dispatch, mark as reviewed.
//  7. Execute normally through the command dispatcher.
//
// userID is stored in executed_by_user_id when provided (system calls pass nil).
func (s *Service) ExecuteAction(id int64, operator, _ string, userID *int64) (*UnifiedAction, error) {
	var a UnifiedAction
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}

	// 1. Idempotency check: if the action has an idempotency key and is already
	// in a terminal execution state, return it as-is.
	if a.IdempotencyKey != "" && (a.Status == ActionStatusExecuted || a.Status == ActionStatusReviewed) {
		s.logger.Debug("idempotent action — already executed",
			zap.Int64("action_id", a.ID),
			zap.String("idempotency_key", a.IdempotencyKey),
			zap.String("status", a.Status))
		return &a, nil
	}

	// 2. Validate against action catalog before execution.
	if s.cat != nil {
		if err := s.cat.ValidateProduction(a.ActionType, riskLevelToInt(a.RiskLevel), a.Status == "approved"); err != nil {
			if errors.Is(err, actioncatalog.ErrApprovalRequired) {
				return nil, ErrApprovalRequired
			}
			return nil, err
		}
	}

	// 3. Allowed source states: "suggested" (auto-approved) or "approved".
	if a.Status != "suggested" && a.Status != "approved" {
		return nil, &InvalidTransitionError{From: a.Status, To: "executing"}
	}

	// 4. If the action claims to require approval but hasn't been approved, refuse.
	if a.RequiresApproval && a.Status == "suggested" {
		return nil, ErrApprovalRequired
	}

	// 5. Run guardrails chain on execution (L4 execution guard).
	if s.guard != nil {
		payload := map[string]interface{}{}
		if len(a.Payload) > 0 {
			_ = json.Unmarshal(a.Payload, &payload)
		}
		// Build a GuardInput that the execution guard can evaluate.
		gIn := &guardrails.GuardInput{
			Level:  4,
			AgentID: a.AgentID,
			ToolName: a.ActionType,
			ToolInput: map[string]interface{}{
				"action_type": a.ActionType,
				"risk_level":  a.RiskLevel,
				"amount":      payload["amount"],
				"quantity":    payload["quantity"],
			},
		}
		if userID != nil {
			gIn.UserID = *userID
		}
		gRes, gErr := s.guard.Check(context.Background(), gIn)
		if gErr != nil {
			s.logger.Warn("guardrails execution check failed",
				zap.Int64("action_id", a.ID),
				zap.Error(gErr))
		} else if gRes.Blocked {
			return nil, &InvalidTransitionError{
				From: a.Status,
				To:   "blocked_by_guardrails",
			}
		}
	}

	now := nowPtr()
	executedByUserIDVal := interface{}(nil)
	if userID != nil {
		executedByUserIDVal = *userID
	}
	updates := map[string]interface{}{
		"status":            "executing",
		"executed_by":       operator,
		"executing_at":      now,
		"executed_by_user_id": executedByUserIDVal,
	}
	if err := s.db.Model(&a).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 6. Dry-run mode: skip command dispatch, mark as executed immediately.
	if a.ExecutionMode == "dry_run" {
		execNow := nowPtr()
		finalUpdates := map[string]interface{}{
			"status":     "executed",
			"executed_at": execNow,
			"updated_at": execNow,
		}
		if err := s.db.Model(&a).Updates(finalUpdates).Error; err != nil {
			return nil, err
		}
		return &a, nil
	}

	// 7. Dispatch through CommandDispatcher to execute real business logic.
	// The cmd field may be nil; in that case the command is not dispatched
	// but the action still transitions to "executed" (the caller has opted
	// out of command dispatch entirely, not just this command).
	var execErr error
	var cmdResult *command.Result
	if s.cmd != nil {
		payload := map[string]interface{}{}
		if len(a.Payload) > 0 {
			_ = json.Unmarshal(a.Payload, &payload)
		}
		cmdResult, execErr = s.cmd.Dispatch(context.Background(), a.ActionType, payload)
		if execErr != nil {
			s.logger.Warn("command dispatch failed during action execution",
				zap.Int64("action_id", a.ID),
				zap.String("action_type", a.ActionType),
				zap.Error(execErr))
		}
	}

	// Build final updates based on the command result.
	execNow := nowPtr()
	finalUpdates := map[string]interface{}{
		"executed_at": execNow,
		"updated_at":  execNow,
	}

	if execErr != nil || (cmdResult != nil && !cmdResult.Success) {
		errMsg := ""
		if execErr != nil {
			errMsg = execErr.Error()
		} else if cmdResult != nil {
			errMsg = cmdResult.ErrorMessage
		}
		finalUpdates["status"] = "failed"
		finalUpdates["failed_at"] = execNow
		finalUpdates["rejection_reason"] = errMsg
	} else {
		finalUpdates["status"] = "executed"
		// Record the after_snapshot from the command result so the UI can
		// see what actually changed.
		if cmdResult != nil && cmdResult.AfterSnapshot != nil {
			snapJSON, _ := json.Marshal(cmdResult.AfterSnapshot)
			finalUpdates["after_snapshot"] = snapJSON
		}
	}

	if err := s.db.Model(&a).Updates(finalUpdates).Error; err != nil {
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
// Uses aggregated GROUP BY queries instead of N+1 per-agent lookups.
func (s *Service) Roster() ([]AgentRosterSummary, error) {
	registry := DefaultRegistry()
	if len(registry.Agents) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(registry.Agents))
	for _, a := range registry.Agents {
		ids = append(ids, a.ID)
	}

	// Aggregate 1: trace count per agent.
	type rowCount struct {
		AgentID string
		Count   int64
	}
	var traceCounts []rowCount
	_ = s.db.Model(&AITrace{}).
		Select("agent_id, COUNT(*) AS count").
		Where("agent_id IN ?", ids).Group("agent_id").Scan(&traceCounts).Error
	tcMap := make(map[string]int64, len(traceCounts))
	for _, v := range traceCounts {
		tcMap[v.AgentID] = v.Count
	}

	// Aggregate 2: action count per agent.
	var actionCounts []rowCount
	_ = s.db.Model(&UnifiedAction{}).
		Select("agent_id, COUNT(*) AS count").
		Where("agent_id IN ?", ids).Group("agent_id").Scan(&actionCounts).Error
	acMap := make(map[string]int64, len(actionCounts))
	for _, v := range actionCounts {
		acMap[v.AgentID] = v.Count
	}

	// Aggregate 3: pending action count per agent.
	var pendingCounts []rowCount
	_ = s.db.Model(&UnifiedAction{}).
		Select("agent_id, COUNT(*) AS count").
		Where("agent_id IN ? AND status IN ?", ids, []string{"suggested", "pending"}).
		Group("agent_id").Scan(&pendingCounts).Error
	pcMap := make(map[string]int64, len(pendingCounts))
	for _, v := range pendingCounts {
		pcMap[v.AgentID] = v.Count
	}

	// Aggregate 4: average confidence per agent.
	type confAvg struct {
		AgentID string
		Avg     float64
	}
	var confAvgs []confAvg
	_ = s.db.Model(&AITrace{}).
		Select("agent_id, COALESCE(AVG(confidence),0) AS avg").
		Where("agent_id IN ? AND confidence IS NOT NULL", ids).
		Group("agent_id").Scan(&confAvgs).Error
	caMap := make(map[string]float64, len(confAvgs))
	for _, v := range confAvgs {
		caMap[v.AgentID] = v.Avg
	}

	out := make([]AgentRosterSummary, 0, len(registry.Agents))
	for _, a := range registry.Agents {
		out = append(out, AgentRosterSummary{
			AgentID:       a.ID,
			Name:          a.Name,
			Squad:         a.Squad,
			DecisionPoint: a.PrimaryDecisionPoint(),
			AutonomyLevel: a.Autonomy,
			TraceCount:    tcMap[a.ID],
			ActionCount:   acMap[a.ID],
			PendingCount:  pcMap[a.ID],
			AvgConfidence: caMap[a.ID],
		})
	}
	return out, nil
}

// transitionAction is the shared lifecycle helper.
// After Model(&a).Updates, GORM refreshes a in-place so no second First() is needed.
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

// riskLevelToInt converts a risk level string to an actioncatalog risk constant.
func riskLevelToInt(level string) int {
	switch level {
	case "low":
		return actioncatalog.RiskLow
	case "medium":
		return actioncatalog.RiskMedium
	case "high":
		return actioncatalog.RiskHigh
	default:
		return actioncatalog.RiskMedium
	}
}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}
