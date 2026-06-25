package entropy

import (
	"time"
)

// SpcControlLimit maps to the spc_control_limit table.
type SpcControlLimit struct {
	ID                  int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID              int64      `gorm:"column:user_id;not null;uniqueIndex:uq_spc_metric" json:"user_id"`
	AgentID             string     `gorm:"column:agent_id;not null;uniqueIndex:uq_spc_metric" json:"agent_id"`
	DecisionPoint       string     `gorm:"column:decision_point;not null;uniqueIndex:uq_spc_metric" json:"decision_point"`
	MetricName          string     `gorm:"column:metric_name;not null;uniqueIndex:uq_spc_metric" json:"metric_name"`
	BaselineMean        float64    `gorm:"column:baseline_mean;type:numeric(10,4);not null" json:"baseline_mean"`
	BaselineStddev      float64    `gorm:"column:baseline_stddev;type:numeric(10,4);not null" json:"baseline_stddev"`
	BaselineSamples     int        `gorm:"column:baseline_samples;not null" json:"baseline_samples"`
	UCL                 float64    `gorm:"column:ucl;type:numeric(10,4);not null" json:"ucl"`
	LCL                 float64    `gorm:"column:lcl;type:numeric(10,4);not null" json:"lcl"`
	UWL                 float64    `gorm:"column:uwl;type:numeric(10,4);not null" json:"uwl"`
	LWL                 float64    `gorm:"column:lwl;type:numeric(10,4);not null" json:"lwl"`
	ConsecutiveSameSide int        `gorm:"column:consecutive_same_side;default:0" json:"consecutive_same_side"`
	LastBreachAt        *time.Time `gorm:"column:last_breach_at" json:"last_breach_at,omitempty"`
	BaselineRecalcAt    time.Time  `gorm:"column:baseline_recalc_at;not null" json:"baseline_recalc_at"`
	NextRecalcAt        time.Time  `gorm:"column:next_recalc_at;not null" json:"next_recalc_at"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SpcControlLimit) TableName() string { return "spc_control_limit" }

// PersonalRule maps to the personal_rule table (used by defenses/health).
type PersonalRule struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID          int64      `gorm:"column:user_id;not null;index" json:"user_id"`
	AgentID         string     `gorm:"column:agent_id;not null" json:"agent_id"`
	DecisionPoint   string     `gorm:"column:decision_point;not null" json:"decision_point"`
	RuleType        string     `gorm:"column:rule_type;not null" json:"rule_type"`
	RuleName        string     `gorm:"column:rule_name;not null" json:"rule_name"`
	RuleCondition   string     `gorm:"column:rule_condition;type:jsonb;not null" json:"rule_condition"`
	RuleAction      string     `gorm:"column:rule_action;type:jsonb;not null" json:"rule_action"`
	Priority        int        `gorm:"column:priority;default:100" json:"priority"`
	Status          string     `gorm:"column:status;default:active" json:"status"`
	Confidence      float64    `gorm:"column:confidence;type:numeric(4,3);default:0" json:"confidence"`
	TimesApplied    int        `gorm:"column:times_applied;default:0" json:"times_applied"`
	TimesOverridden int        `gorm:"column:times_overridden;default:0" json:"times_overridden"`
	LastAppliedAt   *time.Time `gorm:"column:last_applied_at" json:"last_applied_at,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PersonalRule) TableName() string { return "personal_rule" }

// AgentDecision maps to the agent_decision table.
type AgentDecision struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID          int64      `gorm:"column:user_id;not null;index" json:"user_id"`
	AgentID         string     `gorm:"column:agent_id;not null" json:"agent_id"`
	DecisionPoint   string     `gorm:"column:decision_point;not null" json:"decision_point"`
	ContextJSON     string     `gorm:"column:context_json;type:jsonb;not null" json:"context_json"`
	AgentOutput     string     `gorm:"column:agent_output;type:jsonb;not null" json:"agent_output"`
	FinalDecision   string     `gorm:"column:final_decision;type:jsonb;not null" json:"final_decision"`
	UserAction      string     `gorm:"column:user_action;not null" json:"user_action"`
	UserOverrides   *string    `gorm:"column:user_overrides;type:jsonb" json:"user_overrides,omitempty"`
	UserFeedback    *string    `gorm:"column:user_feedback" json:"user_feedback,omitempty"`
	RulesApplied    *string    `gorm:"column:rules_applied;type:jsonb" json:"rules_applied,omitempty"`
	RuleOverrides   int        `gorm:"column:rule_overrides;default:0" json:"rule_overrides"`
	EvolutionStage  string     `gorm:"column:evolution_stage;not null" json:"evolution_stage"`
	Confidence      *float64   `gorm:"column:confidence;type:numeric(4,3)" json:"confidence,omitempty"`
	ResponseTimeMs  *int       `gorm:"column:response_time_ms" json:"response_time_ms,omitempty"`
	TokenCount      *int       `gorm:"column:token_count" json:"token_count,omitempty"`
	SessionID       string     `gorm:"column:session_id;not null" json:"session_id"`
	EpisodeID       *int64     `gorm:"column:episode_id" json:"episode_id,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AgentDecision) TableName() string { return "agent_decision" }

// RuleMarkChange maps to the rule_mark_change audit table.
type RuleMarkChange struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TargetType      string     `gorm:"column:target_type;not null" json:"target_type"`
	TargetID        int64      `gorm:"column:target_id;not null" json:"target_id"`
	FieldPath       string     `gorm:"column:field_path;not null" json:"field_path"`
	OldValue        *string    `gorm:"column:old_value;type:jsonb" json:"old_value,omitempty"`
	NewValue        string     `gorm:"column:new_value;type:jsonb;not null" json:"new_value"`
	SourceType      string     `gorm:"column:source_type;not null" json:"source_type"`
	SourceID        *string    `gorm:"column:source_id" json:"source_id,omitempty"`
	ChangeSummary   string     `gorm:"column:change_summary;not null" json:"change_summary"`
	ParentChangeID  *int64     `gorm:"column:parent_change_id" json:"parent_change_id,omitempty"`
	RelatedDecisionIDs *string `gorm:"column:related_decision_ids;type:jsonb" json:"related_decision_ids,omitempty"`
	ContextJSON     *string    `gorm:"column:context_json;type:jsonb" json:"context_json,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (RuleMarkChange) TableName() string { return "rule_mark_change" }

// ---------------------------------------------------------------------------
// View-object / response types
// ---------------------------------------------------------------------------

// DefenseResult is the result of a single defense layer execution.
type DefenseResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`  // ok / warn / critical
	Action  string `json:"action"`  // cleaned / blocked / decayed / merged / reported
	Count   int    `json:"count"`
	Message string `json:"message,omitempty"`
}

// EntropySummary is the dashboard summary response.
type EntropySummary struct {
	TotalRules         int     `json:"total_rules"`
	ActiveRules        int     `json:"active_rules"`
	ShadowRules        int     `json:"shadow_rules"`
	AvgHealthScore     float64 `json:"avg_health_score"`
	UnhealthyRuleCount int     `json:"unhealthy_rule_count"`
	WarningRuleCount   int     `json:"warning_rule_count"`
	PendingMergeCount  int     `json:"pending_merge_count"`
	RecentChangesCount int     `json:"recent_changes_count"`
	ConflictsCount     int     `json:"conflicts_count"`
	SystemEntropyIndex float64 `json:"system_entropy_index"`
}

// RuleHealthScore is a single rule's health assessment.
type RuleHealthScore struct {
	RuleID               int64            `json:"rule_id"`
	RuleName             string           `json:"rule_name"`
	RuleType             string           `json:"rule_type"`
	AgentID              string           `json:"agent_id"`
	DecisionPoint        string           `json:"decision_point"`
	Status               string           `json:"status"`
	Score                float64          `json:"score"`
	Dimensions           HealthDimensions `json:"dimensions"`
	TimesApplied         int              `json:"times_applied"`
	TimesOverridden      int              `json:"times_overridden"`
	OverrideRate         float64          `json:"override_rate"`
	DaysSinceLastApplied *int             `json:"days_since_last_applied,omitempty"`
	Confidence           float64          `json:"confidence"`
	RiskLevel            string           `json:"risk_level"`
}

// HealthDimensions holds the five dimension scores.
type HealthDimensions struct {
	Acceptance float64 `json:"acceptance"`
	Confidence float64 `json:"confidence"`
	Freshness  float64 `json:"freshness"`
	Frequency  float64 `json:"frequency"`
	TypeWeight float64 `json:"type_weight"`
}

// HealthSummary is the aggregate health overview.
type HealthSummary struct {
	TotalRules     int     `json:"total_rules"`
	ActiveRules    int     `json:"active_rules"`
	ShadowRules    int     `json:"shadow_rules"`
	AvgHealthScore float64 `json:"avg_health_score"`
	UnhealthyCount int     `json:"unhealthy_count"`
	HealthyCount   int     `json:"healthy_count"`
	WarningCount   int     `json:"warning_count"`
}

// SpcStatus is the SPC control chart status for one metric.
type SpcStatus struct {
	AgentID              string     `json:"agent_id"`
	DecisionPoint        string     `json:"decision_point"`
	MetricName           string     `json:"metric_name"`
	CurrentValue         float64    `json:"current_value,omitempty"`
	BaselineMean         float64    `json:"baseline_mean"`
	UCL                  float64    `json:"ucl"`
	LCL                  float64    `json:"lcl"`
	UWL                  float64    `json:"uwl"`
	LWL                  float64    `json:"lwl"`
	BaselineSamples      int        `json:"baseline_samples"`
	ConsecutiveSameSide  int        `json:"consecutive_same_side"`
	IsOutOfControl       bool       `json:"is_out_of_control"`
	IsWarning            bool       `json:"is_warning"`
	Alerts               []SpcAlert `json:"alerts,omitempty"`
	LastBreachAt         *time.Time `json:"last_breach_at,omitempty"`
	NextRecalcAt         *time.Time `json:"next_recalc_at,omitempty"`
}

// SpcAlert is a single SPC alert entry.
type SpcAlert struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// DefenseSummary aggregates all defense layer results.
type DefenseSummary struct {
	Actions struct {
		ExpiredRules        int `json:"expired_rules"`
		BudgetExceeded      int `json:"budget_exceeded"`
		DecayApplied        int `json:"decay_applied"`
		MergedCandidates    int `json:"merged_candidates"`
		ShadowedByOverrides int `json:"shadowed_by_overrides"`
	} `json:"actions"`
	TotalAffected    int              `json:"total_affected"`
	MarkChanges      []ChangeLogEntry `json:"mark_changes,omitempty"`
	DuplicatesFound  int              `json:"duplicates_found"`
	MergeCandidates  []MergeCandidate `json:"merge_candidates,omitempty"`
}

// ChangeLogEntry is a single rule mark change record.
type ChangeLogEntry struct {
	ID            int64   `json:"id"`
	TargetType    string  `json:"target_type"`
	TargetID      int64   `json:"target_id"`
	FieldPath     string  `json:"field_path"`
	OldValue      *string `json:"old_value,omitempty"`
	NewValue      string  `json:"new_value"`
	SourceType    string  `json:"source_type"`
	SourceID      *string `json:"source_id,omitempty"`
	ChangeSummary string  `json:"change_summary"`
	CreatedAt     string  `json:"created_at,omitempty"`
}

// MergeCandidate is a pair of rules that may be merged.
type MergeCandidate struct {
	KeepID     int64   `json:"keep_id"`
	KeepName   string  `json:"keep_name"`
	RemoveID   int64   `json:"remove_id"`
	RemoveName string  `json:"remove_name"`
	Similarity float64 `json:"similarity"`
}

// DuplicatePair is an internal struct for merge detection.
type DuplicatePair struct {
	Keep       *PersonalRule
	Remove     *PersonalRule
	Similarity float64
}
