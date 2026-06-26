package adapters

import (
	"time"

	"github.com/lingmirror/backend-go/internal/domain/metabolism"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// ExecutionLogsAdapter
// ---------------------------------------------------------------------------

// executionLogsRow is the raw DB row shape for the execution_logs table.
type executionLogsRow struct {
	ID        int64
	AgentID   string
	Action    string
	Status    string
	CreatedAt time.Time
}

// TableName tells GORM which table to query.
func (executionLogsRow) TableName() string {
	return "execution_logs"
}

// ExecutionLogsAdapter implements ScoringAdapter by querying the
// execution_logs table.
type ExecutionLogsAdapter struct {
	db *gorm.DB
}

// NewExecutionLogsAdapter creates a new adapter for the execution_logs table.
func NewExecutionLogsAdapter(db *gorm.DB) *ExecutionLogsAdapter {
	return &ExecutionLogsAdapter{db: db}
}

// ScorableEvents returns rows from execution_logs filtered by status.
func (a *ExecutionLogsAdapter) ScorableEvents(status string) ([]metabolism.ScorableEvent, error) {
	var rows []executionLogsRow
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
			Topic:     r.Action,
			CreatedAt: r.CreatedAt,
			Status:    r.Status,
		}
	}
	return results, nil
}

// MarkExcreted sets excreted_at and excretion_reason on the execution_logs row.
func (a *ExecutionLogsAdapter) MarkExcreted(eventID int64, reason string) error {
	return a.db.Exec(
		"UPDATE execution_logs SET excreted_at = NOW(), excretion_reason = ? WHERE id = ?",
		reason, eventID,
	).Error
}

func (a *ExecutionLogsAdapter) ClearExcreted(eventID int64) error {
	return a.db.Exec("UPDATE execution_logs SET excreted_at = NULL, excretion_reason = NULL WHERE id = ?", eventID).Error
}
