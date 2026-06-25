package agentrule

import (
	"encoding/json"
	"time"
)

// PersonalRule maps to the "personal_rule" table.
// Users can create per-agent, per-decision-point rules that override or
// modify agent decisions.
type PersonalRule struct {
	ID            int64            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID        int64            `gorm:"column:user_id;not null;index:idx_agentrule_user" json:"user_id"`
	AgentID       string           `gorm:"column:agent_id;not null;index:idx_agentrule_user" json:"agent_id"`
	DecisionPoint string           `gorm:"column:decision_point" json:"decision_point"`
	RuleType      string           `gorm:"column:rule_type;not null" json:"rule_type"` // veto | threshold | strategy | style
	Name          string           `gorm:"column:name;not null" json:"name"`
	Conditions    json.RawMessage  `gorm:"column:conditions;type:jsonb" json:"conditions"`
	Effect        json.RawMessage  `gorm:"column:effect;type:jsonb" json:"effect"`
	Priority      int              `gorm:"column:priority;default:0" json:"priority"`
	Enabled       bool             `gorm:"column:enabled;default:true" json:"enabled"`
	Description   string           `gorm:"column:description" json:"description"`
	CreatedAt     time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PersonalRule) TableName() string { return "personal_rule" }

// Condition is a single expression evaluated against the agent output.
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // gt | gte | lt | lte | eq | neq | in | contains
	Value    interface{} `json:"value"`
}

// EvaluateResult is returned by Service.Evaluate.
type EvaluateResult struct {
	Output       map[string]interface{} `json:"output"`
	AppliedRules []AppliedRule          `json:"applied_rules"`
	Blocked      bool                   `json:"blocked"`
	BlockReason  string                 `json:"block_reason,omitempty"`
}

// AppliedRule records that a rule was matched and what action it took.
type AppliedRule struct {
	RuleID      int64  `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	RuleType    string `json:"rule_type"`
	Matched     bool   `json:"matched"`
	Blocked     bool   `json:"blocked,omitempty"`
	BlockReason string `json:"block_reason,omitempty"`
	Action      string `json:"action,omitempty"`
}

// CreateRuleInput is the JSON body for creating a personal rule.
type CreateRuleInput struct {
	UserID        int64            `json:"user_id" binding:"required"`
	AgentID       string           `json:"agent_id" binding:"required"`
	DecisionPoint string           `json:"decision_point"`
	RuleType      string           `json:"rule_type" binding:"required"`
	Name          string           `json:"name" binding:"required"`
	Conditions    json.RawMessage  `json:"conditions"`
	Effect        json.RawMessage  `json:"effect"`
	Priority      int              `json:"priority"`
	Enabled       *bool            `json:"enabled"`
	Description   string           `json:"description"`
}

// UpdateRuleInput is the JSON body for updating a personal rule.
type UpdateRuleInput struct {
	DecisionPoint *string          `json:"decision_point"`
	RuleType      *string          `json:"rule_type"`
	Name          *string          `json:"name"`
	Conditions    *json.RawMessage `json:"conditions"`
	Effect        *json.RawMessage `json:"effect"`
	Priority      *int             `json:"priority"`
	Enabled       *bool            `json:"enabled"`
	Description   *string          `json:"description"`
}
