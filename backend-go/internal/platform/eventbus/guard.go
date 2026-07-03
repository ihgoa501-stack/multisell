package eventbus

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// MutationAuditLogger is the interface the guard needs to record audit entries.
type MutationAuditLogger interface {
	LogStructured(input *MutationAuditInput) error
}

// MutationAuditInput carries structured audit fields for mutation guard logging.
type MutationAuditInput struct {
	Module      string
	Action      string
	ResourceID  string
	Operator    string
	Content     string
	Result      string
	TriggerType string
}

// MutationInfo describes a documented system mutation triggered by an event handler.
type MutationInfo struct {
	SystemAction string // ActionCatalog action type
	Domain       string // business domain
	Description  string // human-readable description
}

// MutationGuard wraps event handlers that mutate business state with audit logging.
type MutationGuard struct {
	logger *zap.Logger
	audit  MutationAuditLogger
}

// NewMutationGuard creates a MutationGuard.
func NewMutationGuard(logger *zap.Logger, audit MutationAuditLogger) *MutationGuard {
	return &MutationGuard{logger: logger, audit: audit}
}

// Guard wraps a mutating event handler with audit logging.
func (g *MutationGuard) Guard(info MutationInfo, next Handler) Handler {
	if g == nil || g.audit == nil {
		return next
	}
	return func(ctx context.Context, evt Event) error {
		resourceID := evt.EntityID
		if resourceID == "" {
			resourceID = evt.ID
		}
		operator := evt.Actor
		if operator == "" {
			operator = "system:" + info.Domain
		}

		_ = g.audit.LogStructured(&MutationAuditInput{
			Module:      info.Domain,
			Action:      info.SystemAction,
			ResourceID:  resourceID,
			Operator:    operator,
			Content:     fmt.Sprintf("eventbus mutation start: topic=%s system_action=%s correlation=%s", evt.Topic, info.SystemAction, CorrelationIDFromContext(ctx)),
			Result:      "pending",
			TriggerType: "eventbus",
		})

		err := next(ctx, evt)

		result := "executed"
		if err != nil {
			result = "failed"
		}
		_ = g.audit.LogStructured(&MutationAuditInput{
			Module:      info.Domain,
			Action:      info.SystemAction,
			ResourceID:  resourceID,
			Operator:    operator,
			Content:     fmt.Sprintf("eventbus mutation end: topic=%s system_action=%s correlation=%s", evt.Topic, info.SystemAction, CorrelationIDFromContext(ctx)),
			Result:      result,
			TriggerType: "eventbus",
		})

		if err != nil {
			g.logger.Warn("eventbus mutation guard: handler failed",
				zap.String("system_action", info.SystemAction),
				zap.String("topic", evt.Topic),
				zap.String("event_id", evt.ID),
				zap.Error(err))
		}
		return err
	}
}
