package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/guardrails"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
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
	oplogSvc *operationlog.Service // optional audit logging sink
}

// NewService creates a new AI service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
		traces: NewTraceWriter(db, logger),
	}
}

// WithGuard sets the guardrails chain for execution checks (L1-L5).
func (s *Service) WithGuard(g *guardrails.Chain) *Service {
	s.guard = g
	return s
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

// WithOperationLog attaches an optional operation-log service for audit logging.
// If nil, audit logging is silently skipped (graceful degradation).
func (s *Service) WithOperationLog(svc *operationlog.Service) *Service {
	s.oplogSvc = svc
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
	execMode := in.ExecutionMode
	if execMode == "" {
		execMode = "production"
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
		IdempotencyKey:     in.IdempotencyKey,
		ExecutionMode:      execMode,
		Confidence:         in.Confidence,
		ProposedBy:         in.ProposedBy,
	}
	if err := s.db.Create(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// ApproveAction transitions an action to "approved",
// storing the user ID if provided.
func (s *Service) ApproveAction(id int64, operator string, userID *int64, _ string) (*UnifiedAction, error) {
	return s.transitionAction(id, "approved", map[string]interface{}{
		"approved_by":        operator,
		"approved_by_user_id": userID,
		"approved_at":        nowPtr(),
	}, "suggested", "pending")
}

// RejectAction transitions an action to "rejected". Only allowed from
// "suggested" or "pending" — rejecting an already-approved action is not
// permitted (revoke via a separate flow instead).
func (s *Service) RejectAction(id int64, operator string, userID *int64, reason string) (*UnifiedAction, error) {
	return s.transitionAction(id, "rejected", map[string]interface{}{
		"rejected_by":         operator,
		"rejected_by_user_id": userID,
		"rejected_at":         nowPtr(),
		"rejection_reason":    reason,
	}, "suggested", "pending")
}

// ExecuteAction transitions an action through "executing" → "executed"/"failed" and
// dispatches the underlying business command via the CommandDispatcher.
//
// Security gates (all checked before any mutation):
//   - Idempotency: if IdempotencyKey is set and the action is already executed,
//     returns the existing result without re-executing.
//   - Execution mode: dry_run validates and reports, never mutates.
//   - Guardrails: runs the L4 ExecutionGuard on the action payload.
//   - Approval: requires approved status for high-risk actions.
//   - Actor identity: operator is set server-side from JWT (not client-supplied).
//
// For low-risk auto-approved actions (RequiresApproval=false) this can be
// called directly from "suggested"; otherwise the action must first be
// approved.
func (s *Service) ExecuteAction(id int64, userID *int64, operator, _ string) (*UnifiedAction, error) {
	var a UnifiedAction
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}

	// ── Gate 1: Idempotency key ────────────────────────────────────
	if a.IdempotencyKey != "" && (a.Status == "executed" || a.Status == "failed") {
		s.logger.Debug("idempotent action — returning existing result",
			zap.Int64("action_id", a.ID),
			zap.String("idempotency_key", a.IdempotencyKey))
		return &a, nil
	}

	// ── Gate 2: Validate against action catalog before execution ───
	isDryRun := a.ExecutionMode == "dry_run"

	if s.cat != nil {
		if err := s.cat.ValidateProduction(a.ActionType, riskLevelToInt(a.RiskLevel), isDryRun || a.Status == "approved"); err != nil {
			if errors.Is(err, actioncatalog.ErrApprovalRequired) {
				return nil, ErrApprovalRequired
			}
			return nil, err
		}
	}

	// ── Gate 3: Status transition check ────────────────────────────
	if a.Status != "suggested" && a.Status != "approved" {
		return nil, &InvalidTransitionError{From: a.Status, To: "executing"}
	}

	// ── Gate 4: Execution mode check ──────────────────────────
	switch a.ExecutionMode {
	case "", "production":
		// production execution continues below
	case "dry_run":
		// isDryRun already set at Gate 2
		// dry_run validates through all gates but does not dispatch or mutate.
	case "sandbox":
		return nil, fmt.Errorf("sandbox execution requires a sandbox executor; no sandbox configured")
	default:
		return nil, fmt.Errorf("unknown execution mode: %s", a.ExecutionMode)
	}

	// ── Gate 5: Approval check ─────────────────────────────────────
	if !isDryRun && a.RequiresApproval && a.Status == "suggested" {
		return nil, ErrApprovalRequired
	}

	// ── Gate 6: Guardrails check (L4 ExecutionGuard) ──────────────
	var guardPayload map[string]interface{}
	if s.guard != nil {
		payload := map[string]interface{}{}
		if len(a.Payload) > 0 {
			_ = json.Unmarshal(a.Payload, &payload)
		}
		// ponytail: use a generic GuardInput for the action-level check;
		// per-tool checks happen inside the tool registry hooks.
		// Seed action_type into ToolInput so ExecutionGuard rules can match by action type.
		if payload == nil {
			payload = map[string]interface{}{}
		}
		payload["action_type"] = a.ActionType
		payload["risk_level"] = a.RiskLevel
		gr, err := s.guard.Check(context.Background(), &guardrails.GuardInput{
			AgentID:   a.AgentID,
			ToolName:  a.ActionType,
			ToolInput: payload,
		})
		guardPayload = payload
		if err != nil {
			s.logger.Warn("guardrails check error during execution",
				zap.Int64("action_id", a.ID),
				zap.Error(err))
			return nil, fmt.Errorf("execution guard check failed: %w", err)
		}
		if gr.Blocked {
			s.logger.Warn("action blocked by execution guard",
				zap.Int64("action_id", a.ID),
				zap.String("action_type", a.ActionType),
				zap.String("reason", gr.Reason),
			)
			return nil, fmt.Errorf("%w: %s", ErrBlockedByGuardrails, gr.Reason)
		}
	}

	// Dry-run: all validation passed, return without mutation.
	if isDryRun {
		s.logger.Info("dry-run action — all gates passed, execution skipped",
			zap.Int64("action_id", a.ID),
			zap.String("action_type", a.ActionType))
		return &a, nil
	}

	// ── Execute: transition to executing (atomic claim) ────────
	now := nowPtr()
	res := s.db.Model(&UnifiedAction{}).
		Where("id = ? AND status IN ?", a.ID, []string{"suggested", "approved"}).
		Updates(map[string]interface{}{
			"status":              "executing",
			"executed_by":         operator,
			"executed_by_user_id": userID,
			"executing_at":        now,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	// RowsAffected == 0 means another request already claimed this action.
	if res.RowsAffected == 0 {
		var current UnifiedAction
		if err := s.db.First(&current, a.ID).Error; err != nil {
			return nil, err
		}
		// Idempotent: if already completed, return existing result.
		if current.IdempotencyKey != "" && (current.Status == "executed" || current.Status == "failed") {
			return &current, nil
		}
		return nil, fmt.Errorf("action %d is already %s (conflict)", a.ID, current.Status)
	}

	// Re-read the action to get fresh values after the atomic UPDATE.
	if err := s.db.First(&a, a.ID).Error; err != nil {
		return nil, err
	}

	// Dispatch through CommandDispatcher to execute real business logic.
	var execErr error
	var cmdResult *command.Result
	if s.cmd != nil {
		cmdPayload := guardPayload
		if cmdPayload == nil {
			cmdPayload = map[string]interface{}{}
			if len(a.Payload) > 0 {
				_ = json.Unmarshal(a.Payload, &cmdPayload)
			}
		}
		cmdResult, execErr = s.cmd.Dispatch(context.Background(), a.ActionType, cmdPayload)
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
		if cmdResult != nil && cmdResult.AfterSnapshot != nil {
			finalUpdates["after_snapshot"] = cmdResult.AfterSnapshot
		}
	}

	if err := s.db.Model(&a).Updates(finalUpdates).Error; err != nil {
		return nil, err
	}

	// Audit log: execution complete (success or failure).
	status, _ := finalUpdates["status"].(string)
	s.logExecuteAction(a, operator, status)

	return &a, nil
}

// logExecuteAction writes an audit log entry for an action execution event.
// Silently skips if no operation-log service is configured (graceful degradation).
func (s *Service) logExecuteAction(a UnifiedAction, operator, status string) {
	if s.oplogSvc == nil {
		return
	}
	content, _ := json.Marshal(a)
	_ = s.oplogSvc.LogStructured(&operationlog.StructuredLogInput{
		Action:      "ai.action." + status,
		Module:      "ai.action",
		ResourceID:  strconv.FormatInt(a.ID, 10),
		Operator:    operator,
		Content:     string(content),
		Result:      status,
		TriggerType: "agent",
	})
}

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

// ErrBlockedByGuardrails is returned when the execution guardrail chain blocks an action.
var ErrBlockedByGuardrails = errors.New("action blocked by guardrails")

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
