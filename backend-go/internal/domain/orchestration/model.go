package orchestration

import "time"

// LifecycleStep defines a step in the product lifecycle pipeline.
type LifecycleStep struct {
	ID          int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID   int64      `gorm:"column:product_id;not null;index" json:"product_id"`
	Step        string     `gorm:"column:step;not null" json:"step"`          // sourcing, enrichment, compliance, pricing, listing, monitoring, delisting
	AgentID     string     `gorm:"column:agent_id;not null" json:"agent_id"` // who handles this step
	Status      string     `gorm:"column:status;default:pending" json:"status"` // pending, running, completed, failed, skipped
	StartedAt   *time.Time `gorm:"column:started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	DurationMs  int        `gorm:"column:duration_ms" json:"duration_ms"`
	Result      string     `gorm:"column:result;type:text" json:"result,omitempty"`
	Error       string     `gorm:"column:error;type:text" json:"error,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (LifecycleStep) TableName() string { return "lifecycle_step" }

// OrchestrationConfig defines per-product or per-tenant pipeline rules.
type OrchestrationConfig struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name            string    `gorm:"column:name;not null" json:"name"`
	Steps           string    `gorm:"column:steps;type:jsonb" json:"steps"`                     // ordered array of step definitions
	FailureAction   string    `gorm:"column:failure_action;default:stop" json:"failure_action"` // stop, skip, retry
	AutoApprovePct  float64   `gorm:"column:auto_approve_pct;default:80" json:"auto_approve_pct"` // auto-approve if trust score > this
	AutoRetryCount  int       `gorm:"column:auto_retry_count;default:3" json:"auto_retry_count"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (OrchestrationConfig) TableName() string { return "orchestration_config" }

// stepAgentMapping maps pipleine step names to their responsible agents.
var stepAgentMapping = map[string]string{
	"sourcing":    "A8",
	"enrichment":  "content_ai",
	"compliance":  "G3",
	"pricing":     "A3",
	"listing":     "A2",
	"monitoring":  "A5",
	"delisting":   "scheduler",
}

// DefaultPipeline defines the standard step order for product lifecycle.
var DefaultPipeline = []string{
	"sourcing",    // A8  finds product candidate
	"enrichment",  // Content AI  generates title/description/images
	"compliance",  // G3  checks certificates, HS code, banned words
	"pricing",     // A3  calculates landed cost, sets price
	"listing",     // A2  publishes to chosen platforms
	"monitoring",  // A5/A6  watches sales, stock, margin
	"delisting",   // (scheduled)  auto-delist when lifecycle ends
}

// Pipeline step status constants
const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
	StepStatusFailed    = "failed"
	StepStatusSkipped   = "skipped"
)

// Failure action constants
const (
	FailureActionStop = "stop"
	FailureActionSkip = "skip"
	FailureActionRetry = "retry"
)
