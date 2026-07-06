package workflow

import (
	"encoding/json"
	"time"
)

// ── Step type constants ──────────────────────────────────────────────

const (
	StepTypeAgent     = "agent"     // run an AI agent decision
	StepTypeCommand   = "command"   // dispatch a command
	StepTypeFork      = "fork"      // fan-out to parallel sub-steps
	StepTypeJoin      = "join"      // barrier: wait for fork results
	StepTypeDelay     = "delay"     // timer wait
	StepTypeEvent     = "event"     // wait for event bus message
	StepTypeCondition = "condition" // evaluate a condition expression
	StepTypeApproval  = "approval"  // wait for human approval/rejection
	StepTypeAction    = "action"    // dispatch a predefined action
)

// Run status constants.
const (
	RunStatusPending   = "pending"
	RunStatusRunning   = "running"
	RunStatusPaused    = "paused"
	RunStatusCompleted = "completed"
	RunStatusFailed    = "failed"
)

// Step status constants.
const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
	StepStatusFailed    = "failed"
	StepStatusSkipped   = "skipped"
	StepStatusApproved  = "approved"
	StepStatusRejected  = "rejected"
)

// ── DB models ────────────────────────────────────────────────────────

// WorkflowDef is a workflow template — steps defined as JSON.
type WorkflowDef struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;not null" json:"name"`
	Description string    `gorm:"column:description;type:text" json:"description,omitempty"`
	Steps       string    `gorm:"column:steps;type:jsonb;not null;default:'[]'" json:"steps"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (WorkflowDef) TableName() string { return "workflow_def" }

// WorkflowRun is a running instance of a workflow.
type WorkflowRun struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WorkflowDefID int64      `gorm:"column:workflow_def_id" json:"workflow_def_id"`
	Name          string     `gorm:"column:name;not null" json:"name"`
	Status        string     `gorm:"column:status;default:pending" json:"status"`
	Context       string     `gorm:"column:context;type:jsonb;default:'{}'" json:"context"`
	StartedAt     *time.Time `gorm:"column:started_at" json:"started_at,omitempty"`
	CompletedAt   *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	Error         string     `gorm:"column:error;type:text" json:"error,omitempty"`
	CurrentNodeID int64      `gorm:"column:current_node_id;default:0" json:"current_node_id"`
	RetryCount    int        `gorm:"column:retry_count;default:0" json:"retry_count"`
	MaxRetries    int        `gorm:"column:max_retries;default:3" json:"max_retries"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	Steps []WorkflowStepRun `gorm:"-" json:"steps,omitempty"` // populated by API
}

func (WorkflowRun) TableName() string { return "workflow_run" }

// WorkflowStepRun is one step execution within a workflow run.
type WorkflowStepRun struct {
	ID             int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WorkflowRunID  int64      `gorm:"column:workflow_run_id;not null;index" json:"workflow_run_id"`
	StepName       string     `gorm:"column:step_name;not null" json:"step_name"`
	StepType       string     `gorm:"column:step_type;not null" json:"step_type"`
	ParentID       *int64     `gorm:"column:parent_id" json:"parent_id,omitempty"`
	Status         string     `gorm:"column:status;default:pending" json:"status"`
	Input          string     `gorm:"column:input;type:jsonb;default:'{}'" json:"input"`
	Output         string     `gorm:"column:output;type:jsonb;default:'{}'" json:"output"`
	Error          string     `gorm:"column:error;type:text" json:"error,omitempty"`
	Attempt        int        `gorm:"column:attempt;default:1" json:"attempt"`
	MaxAttempts    int        `gorm:"column:max_attempts;default:1" json:"max_attempts"`
	TimeoutSeconds int        `gorm:"column:timeout_seconds;default:300" json:"timeout_seconds"`
	StartedAt      *time.Time `gorm:"column:started_at" json:"started_at,omitempty"`
	CompletedAt    *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (WorkflowStepRun) TableName() string { return "workflow_step_run" }

// ── JSON-friendly step definition (stored in WorkflowDef.Steps) ──────

// StepDef defines one step in a workflow definition.
type StepDef struct {
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	AgentID        string                 `json:"agent_id,omitempty"`
	DecisionPoint  string                 `json:"decision_point,omitempty"`
	Command        string                 `json:"command,omitempty"`
	Condition      string                 `json:"condition,omitempty"`
	Inputs         map[string]interface{} `json:"inputs,omitempty"`
	Forks          []StepDef              `json:"forks,omitempty"`
	JoinSteps      []string               `json:"join_steps,omitempty"`
	WaitForEvent   string                 `json:"wait_for_event,omitempty"`
	DelaySeconds   int                    `json:"delay_seconds,omitempty"`
	TimeoutSeconds int                    `json:"timeout_seconds,omitempty"`
	RetryCount     int                    `json:"retry_count,omitempty"`
	RetryBackoffMs int                    `json:"retry_backoff_ms,omitempty"`
}

// ── WorkflowNode (structured node table) ─────────────────────────────

// WorkflowNode represents a single node in a workflow definition.
type WorkflowNode struct {
	ID         uint            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WorkflowID uint            `gorm:"column:workflow_def_id;not null;index" json:"workflow_def_id"`
	Type       string          `gorm:"column:type;not null" json:"type"` // condition, approval, action, event
	Config     json.RawMessage `gorm:"column:config;type:jsonb;default:'{}'" json:"config"`
	OrderIndex int             `gorm:"column:order_index;default:0" json:"order_index"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (WorkflowNode) TableName() string { return "workflow_node" }

// NodeConfig holds typed configuration for each node type.
type NodeConfig struct {
	// Common
	Condition      string `json:"condition,omitempty"`       // for condition nodes
	ApprovalRoles  string `json:"approval_roles,omitempty"`  // comma-separated roles for approval
	Command        string `json:"command,omitempty"`         // for action nodes
	AgentID        string `json:"agent_id,omitempty"`        // for action nodes
	DecisionPoint  string `json:"decision_point,omitempty"`  // for action nodes
	EventTopic     string `json:"event_topic,omitempty"`     // for event nodes
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"` // all types
}

// ── Approval step input/output ───────────────────────────────────────

// ApprovalResult is the payload for an approval step decision.
type ApprovalResult struct {
	Approved bool   `json:"approved"`
	Comment  string `json:"comment,omitempty"`
	Reviewer string `json:"reviewer,omitempty"`
}
