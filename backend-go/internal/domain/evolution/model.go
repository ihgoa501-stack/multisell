package evolution

import (
	"encoding/json"
	"time"
)

// Nudge represents a prompt to upgrade an agent's autonomy level.
type Nudge struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID       int64           `gorm:"column:user_id;not null;index" json:"user_id"`
	AgentID      string          `gorm:"column:agent_id;not null" json:"agent_id"`
	CurrentLevel string          `gorm:"column:current_level;not null" json:"current_level"`
	TargetLevel  string          `gorm:"column:target_level;not null" json:"target_level"`
	TrustScore   float64         `gorm:"column:trust_score;type:numeric(5,4)" json:"trust_score"`
	Status       string          `gorm:"column:status;default:pending" json:"status"` // pending | accepted | dismissed
	Message      string          `gorm:"column:message" json:"message"`
	Metrics      json.RawMessage `gorm:"column:metrics;type:jsonb" json:"metrics,omitempty"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	DecidedAt    *time.Time      `gorm:"column:decided_at" json:"decided_at,omitempty"`
}

func (Nudge) TableName() string { return "agent_nudge" }
