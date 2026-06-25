package personalrule

import (
	"encoding/json"
	"time"
)

// PersonalRule maps to the "personal_rule" table.
type PersonalRule struct {
	ID              int64            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID          int64            `gorm:"column:user_id;not null;index" json:"user_id"`
	AgentID         string           `gorm:"column:agent_id;not null" json:"agent_id"`
	DecisionPoint   string           `gorm:"column:decision_point;not null" json:"decision_point"`
	RuleType        string           `gorm:"column:rule_type;not null" json:"rule_type"`
	RuleName        string           `gorm:"column:rule_name;not null" json:"rule_name"`
	RuleCondition   json.RawMessage  `gorm:"column:rule_condition;type:jsonb;not null" json:"rule_condition"`
	RuleAction      json.RawMessage  `gorm:"column:rule_action;type:jsonb;not null" json:"rule_action"`
	Priority        int              `gorm:"column:priority;default:100" json:"priority"`
	Source          string           `gorm:"column:source;not null" json:"source"`
	SourceDecisions json.RawMessage  `gorm:"column:source_decisions;type:jsonb" json:"source_decisions,omitempty"`
	Status          string           `gorm:"column:status;default:active" json:"status"`
	Confidence      float64          `gorm:"column:confidence;type:numeric(4,3);default:0" json:"confidence"`
	TimesApplied    int              `gorm:"column:times_applied;default:0" json:"times_applied"`
	TimesOverridden int              `gorm:"column:times_overridden;default:0" json:"times_overridden"`
	LastAppliedAt   *time.Time       `gorm:"column:last_applied_at" json:"last_applied_at,omitempty"`
	CreatedAt       time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PersonalRule) TableName() string { return "personal_rule" }

// RuleMarkChange maps to the "rule_mark_change" table.
type RuleMarkChange struct {
	ID                 int64            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TargetType         string           `gorm:"column:target_type;not null" json:"target_type"`
	TargetID           int64            `gorm:"column:target_id;not null" json:"target_id"`
	FieldPath          string           `gorm:"column:field_path;not null" json:"field_path"`
	OldValue           json.RawMessage  `gorm:"column:old_value;type:jsonb" json:"old_value,omitempty"`
	NewValue           json.RawMessage  `gorm:"column:new_value;type:jsonb;not null" json:"new_value"`
	SourceType         string           `gorm:"column:source_type;not null" json:"source_type"`
	SourceID           string           `gorm:"column:source_id" json:"source_id,omitempty"`
	ChangeSummary      string           `gorm:"column:change_summary;not null" json:"change_summary"`
	ParentChangeID     *int64           `gorm:"column:parent_change_id" json:"parent_change_id,omitempty"`
	RelatedDecisionIDs json.RawMessage  `gorm:"column:related_decision_ids;type:jsonb" json:"related_decision_ids,omitempty"`
	ContextJSON        json.RawMessage  `gorm:"column:context_json;type:jsonb" json:"context_json,omitempty"`
	CreatedAt          time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (RuleMarkChange) TableName() string { return "rule_mark_change" }

// RuleResult is returned by ApplyRules describing what each rule did.
type RuleResult struct {
	RuleID        int64                    `json:"rule_id"`
	RuleName      string                   `json:"rule_name"`
	RuleType      string                   `json:"rule_type"`
	Matched       bool                     `json:"matched"`
	Blocked       bool                     `json:"blocked,omitempty"`
	Notifications []map[string]interface{} `json:"notifications,omitempty"`
}

// Condition is the parsed structure of rule_condition.
type Condition struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}
