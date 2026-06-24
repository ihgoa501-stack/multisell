package ai

import (
	"encoding/json"
	"time"
)

// AITrace maps to "ai_trace".
type AITrace struct {
	ID            int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TraceID       string          `gorm:"column:trace_id;uniqueIndex" json:"trace_id"`
	UserID        *int64          `gorm:"column:user_id" json:"user_id,omitempty"`
	AgentID       string          `gorm:"column:agent_id;not null;index" json:"agent_id"`
	DecisionPoint string          `gorm:"column:decision_point;not null" json:"decision_point"`
	Status        string          `gorm:"column:status;default:running;index" json:"status"`
	ModelProvider string          `gorm:"column:model_provider" json:"model_provider"`
	ModelName     string          `gorm:"column:model_name" json:"model_name"`
	PromptVersion string          `gorm:"column:prompt_version" json:"prompt_version"`
	InputContext  json.RawMessage `gorm:"column:input_context;type:jsonb" json:"input_context"`
	FinalOutput   json.RawMessage `gorm:"column:final_output;type:jsonb" json:"final_output,omitempty"`
	Confidence    *float64        `gorm:"column:confidence" json:"confidence,omitempty"`
	RiskLevel     string          `gorm:"column:risk_level" json:"risk_level"`
	TokenCount    int             `gorm:"column:token_count" json:"token_count"`
	LatencyMs     int             `gorm:"column:latency_ms" json:"latency_ms"`
	StartedAt     time.Time       `gorm:"column:started_at" json:"started_at"`
	CompletedAt   *time.Time      `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AITrace) TableName() string { return "ai_trace" }

// AITraceEvent maps to "ai_trace_event".
type AITraceEvent struct {
	ID         int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TraceID    string          `gorm:"column:trace_id;not null;uniqueIndex:uq_trace_seq" json:"trace_id"`
	EventType  string          `gorm:"column:event_type;not null" json:"event_type"`
	Seq        int             `gorm:"column:seq;not null;uniqueIndex:uq_trace_seq" json:"seq"`
	Content    string          `gorm:"column:content" json:"content"`
	Payload    json.RawMessage `gorm:"column:payload;type:jsonb" json:"payload"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AITraceEvent) TableName() string { return "ai_trace_event" }

// AIEvidenceRef maps to "ai_evidence_ref".
type AIEvidenceRef struct {
	ID         int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TraceID    string          `gorm:"column:trace_id;not null;index" json:"trace_id"`
	SourceType string          `gorm:"column:source_type;not null" json:"source_type"`
	SourceID   string          `gorm:"column:source_id;not null" json:"source_id"`
	Title      string          `gorm:"column:title;not null" json:"title"`
	Summary    string          `gorm:"column:summary" json:"summary"`
	Payload    json.RawMessage `gorm:"column:payload;type:jsonb" json:"payload"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AIEvidenceRef) TableName() string { return "ai_evidence_ref" }

// UnifiedAction maps to "unified_action".
type UnifiedAction struct {
	ID                int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourceTable       string          `gorm:"column:source_table;not null;uniqueIndex:uq_source" json:"source_table"`
	SourceID          string          `gorm:"column:source_id;not null;uniqueIndex:uq_source" json:"source_id"`
	SourceType        string          `gorm:"column:source_type;not null" json:"source_type"`
	TraceID           string          `gorm:"column:trace_id" json:"trace_id,omitempty"`
	AgentID           string          `gorm:"column:agent_id;index" json:"agent_id,omitempty"`
	SquadID           string          `gorm:"column:squad_id" json:"squad_id,omitempty"`
	UserID            *int64          `gorm:"column:user_id;index" json:"user_id,omitempty"`
	ActionType        string          `gorm:"column:action_type;not null" json:"action_type"`
	BusinessObjectType string         `gorm:"column:business_object_type" json:"business_object_type,omitempty"`
	BusinessObjectID  string          `gorm:"column:business_object_id" json:"business_object_id,omitempty"`
	Title             string          `gorm:"column:title;not null" json:"title"`
	Description       string          `gorm:"column:description" json:"description,omitempty"`
	Payload           json.RawMessage `gorm:"column:payload;type:jsonb" json:"payload"`
	BeforeSnapshot    json.RawMessage `gorm:"column:before_snapshot;type:jsonb" json:"before_snapshot,omitempty"`
	AfterSnapshot     json.RawMessage `gorm:"column:after_snapshot;type:jsonb" json:"after_snapshot,omitempty"`
	RiskLevel         string          `gorm:"column:risk_level;default:medium" json:"risk_level"`
	RequiresApproval  bool            `gorm:"column:requires_approval" json:"requires_approval"`
	Status            string          `gorm:"column:status;default:suggested;index" json:"status"`
	Confidence        *float64        `gorm:"column:confidence" json:"confidence,omitempty"`
	ProposedBy        string          `gorm:"column:proposed_by" json:"proposed_by"`
	ApprovedBy        string          `gorm:"column:approved_by" json:"approved_by,omitempty"`
	RejectedBy        string          `gorm:"column:rejected_by" json:"rejected_by,omitempty"`
	ExecutedBy        string          `gorm:"column:executed_by" json:"executed_by,omitempty"`
	RejectionReason   string          `gorm:"column:rejection_reason" json:"rejection_reason,omitempty"`
	ProposedAt        time.Time       `gorm:"column:proposed_at;autoCreateTime;index" json:"proposed_at"`
	ApprovedAt        *time.Time      `gorm:"column:approved_at" json:"approved_at,omitempty"`
	RejectedAt        *time.Time      `gorm:"column:rejected_at" json:"rejected_at,omitempty"`
	ExecutingAt       *time.Time      `gorm:"column:executing_at" json:"executing_at,omitempty"`
	ExecutedAt        *time.Time      `gorm:"column:executed_at" json:"executed_at,omitempty"`
	ReviewedAt        *time.Time      `gorm:"column:reviewed_at" json:"reviewed_at,omitempty"`
	FailedAt          *time.Time      `gorm:"column:failed_at" json:"failed_at,omitempty"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (UnifiedAction) TableName() string { return "unified_action" }

// TraceDetail is the composite trace payload for trace replay.
type TraceDetail struct {
	Trace    AITrace         `json:"trace"`
	Events   []AITraceEvent  `json:"events"`
	Evidence []AIEvidenceRef `json:"evidence"`
	Actions  []UnifiedAction `json:"actions"`
}

// CreateTraceInput starts a new trace.
type CreateTraceInput struct {
	AgentID       string          `json:"agent_id" binding:"required"`
	DecisionPoint string          `json:"decision_point" binding:"required"`
	UserID        *int64          `json:"user_id"`
	ModelProvider string          `json:"model_provider"`
	ModelName     string          `json:"model_name"`
	PromptVersion string          `json:"prompt_version"`
	InputContext  json.RawMessage `json:"input_context"`
}

// AppendEventInput adds an event to a trace.
type AppendEventInput struct {
	EventType string          `json:"event_type" binding:"required"`
	Content   string          `json:"content"`
	Payload   json.RawMessage `json:"payload"`
}

// AddEvidenceInput attaches evidence to a trace.
type AddEvidenceInput struct {
	SourceType string          `json:"source_type" binding:"required"`
	SourceID   string          `json:"source_id" binding:"required"`
	Title      string          `json:"title" binding:"required"`
	Summary    string          `json:"summary"`
	Payload    json.RawMessage `json:"payload"`
}

// CompleteTraceInput closes a running trace.
type CompleteTraceInput struct {
	FinalOutput json.RawMessage `json:"final_output"`
	Confidence  *float64        `json:"confidence"`
	RiskLevel   string          `json:"risk_level"`
	TokenCount  int             `json:"token_count"`
	Status      string          `json:"status"` // completed | failed
}

// CreateActionInput creates a unified action.
type CreateActionInput struct {
	SourceTable        string          `json:"source_table" binding:"required"`
	SourceID           string          `json:"source_id" binding:"required"`
	SourceType         string          `json:"source_type" binding:"required"`
	TraceID            string          `json:"trace_id"`
	AgentID            string          `json:"agent_id"`
	SquadID            string          `json:"squad_id"`
	UserID             *int64          `json:"user_id"`
	ActionType         string          `json:"action_type" binding:"required"`
	BusinessObjectType string          `json:"business_object_type"`
	BusinessObjectID   string          `json:"business_object_id"`
	Title              string          `json:"title" binding:"required"`
	Description        string          `json:"description"`
	Payload            json.RawMessage `json:"payload"`
	BeforeSnapshot     json.RawMessage `json:"before_snapshot"`
	AfterSnapshot      json.RawMessage `json:"after_snapshot"`
	RiskLevel          string          `json:"risk_level"`
	RequiresApproval   *bool           `json:"requires_approval"`
	Confidence         *float64        `json:"confidence"`
	ProposedBy         string          `json:"proposed_by"`
}

// ActionDecisionInput is used for approve/reject/execute.
type ActionDecisionInput struct {
	Operator string `json:"operator" binding:"required"`
	Reason   string `json:"reason"`
}

// ChatInput is the payload for POST /ai/chat.
type ChatInput struct {
	Message    string                 `json:"message" binding:"required"`
	AgentID    string                 `json:"agent_id"`
	Context    map[string]interface{} `json:"context"`
	Stream     bool                   `json:"stream"`
}

// ChatResponse is the non-streaming chat response.
type ChatResponse struct {
	TraceID    string                 `json:"trace_id"`
	AgentID    string                 `json:"agent_id"`
	Answer     string                 `json:"answer"`
	Confidence float64                `json:"confidence"`
	RiskLevel  string                 `json:"risk_level"`
	Actions    []UnifiedAction        `json:"actions"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// TraceListFilter captures query parameters.
type TraceListFilter struct {
	Search     string
	AgentID    string
	Status     string
	DecisionPoint string
}

// ActionListFilter captures query parameters.
type ActionListFilter struct {
	Search   string
	AgentID  string
	Status   string
	RiskLevel string
	SquadID  string
}

// AgentRosterSummary is a per-agent summary for the cockpit.
type AgentRosterSummary struct {
	AgentID       string  `json:"agent_id"`
	Name          string  `json:"name"`
	Squad         string  `json:"squad"`
	DecisionPoint string  `json:"decision_point"`
	AutonomyLevel string  `json:"autonomy_level"`
	TraceCount    int64   `json:"trace_count"`
	ActionCount   int64   `json:"action_count"`
	PendingCount  int64   `json:"pending_count"`
	AvgConfidence float64 `json:"avg_confidence"`
}
