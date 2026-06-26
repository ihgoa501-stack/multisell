package adapters

import (
	"time"

	"github.com/lingmirror/backend-go/internal/domain/metabolism"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// AgentDecisionsAdapter
// ---------------------------------------------------------------------------

// agentDecisionRow is the raw DB row shape for the agent_decision table.
type agentDecisionRow struct {
	ID            int64
	AgentID       string
	DecisionPoint string
	Status        string
	CreatedAt     time.Time
}

// TableName tells GORM which table to query.
func (agentDecisionRow) TableName() string {
	return "agent_decision"
}

// AgentDecisionsAdapter implements ScoringAdapter by querying the
// agent_decision table.
type AgentDecisionsAdapter struct {
	db *gorm.DB
}

// NewAgentDecisionsAdapter creates a new adapter for the agent_decision table.
func NewAgentDecisionsAdapter(db *gorm.DB) *AgentDecisionsAdapter {
	return &AgentDecisionsAdapter{db: db}
}

// ScorableEvents returns rows from agent_decision filtered by status.
func (a *AgentDecisionsAdapter) ScorableEvents(status string) ([]metabolism.ScorableEvent, error) {
	var rows []agentDecisionRow
	tx := a.db.Order("created_at ASC")
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if err := tx.Find(&rows).Error; err != nil {
		return nil, err
	}

	results := make([]metabolism.ScorableEvent, len(rows))
	for i, r := range rows {
		results[i] = metabolism.ScorableEvent{
			ID:        r.ID,
			Source:    r.AgentID,
			Topic:     r.DecisionPoint,
			CreatedAt: r.CreatedAt,
			Status:    r.Status,
		}
	}
	return results, nil
}

// MarkExcreted sets excreted_at and excretion_reason on the agent_decision row.
func (a *AgentDecisionsAdapter) MarkExcreted(eventID int64, reason string) error {
	return a.db.Exec(
		"UPDATE agent_decision SET excreted_at = NOW(), excretion_reason = ? WHERE id = ?",
		reason, eventID,
	).Error
}

func (a *AgentDecisionsAdapter) ClearExcreted(eventID int64) error {
	return a.db.Exec("UPDATE agent_decisions SET excreted_at = NULL, excretion_reason = NULL WHERE id = ?", eventID).Error
}
