package actionpolicy

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ForbiddenAction maps to forbidden_action table.
// Defines a permanently blocked action pattern — no approval can override it.
type ForbiddenAction struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ActionType string    `gorm:"column:action_type;not null;index:idx_forbidden_action" json:"action_type"`
	AgentID    string    `gorm:"column:agent_id;index:idx_forbidden_action" json:"agent_id,omitempty"`
	RiskLevel  string    `gorm:"column:risk_level" json:"risk_level"`
	Reason     string    `gorm:"column:reason;not null" json:"reason"`
	Enabled    bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (ForbiddenAction) TableName() string { return "forbidden_action" }

// CheckForbidden checks if an action is forbidden. Returns nil if allowed.
// Returns nil (allowed) if the forbidden_action table does not exist
// (graceful degradation for environments without the migration).
func CheckForbidden(db *gorm.DB, agentID, actionType, riskLevel string) error {
	// First check if the table exists — skip forbidden check if not migrated.
	if !db.Migrator().HasTable(&ForbiddenAction{}) {
		return nil
	}
	var count int64
	q := db.Model(&ForbiddenAction{}).Where("enabled = ?", true)
	q = q.Where("(action_type = ? OR action_type = '*') AND (agent_id = ? OR agent_id = '' OR agent_id = '*')", actionType, agentID)
	if riskLevel != "" {
		q = q.Where("(risk_level = ? OR risk_level = '' OR risk_level = '*')", riskLevel)
	}
	if err := q.Count(&count).Error; err != nil {
		return fmt.Errorf("forbidden check failed: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("action %s is forbidden for agent %s", actionType, agentID)
	}
	return nil
}
