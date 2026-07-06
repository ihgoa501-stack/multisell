package reliability

import (
	"time"
)

// AgentStatus represents agent runtime status.
type AgentStatus struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	AgentID       string     `gorm:"uniqueIndex;size:100" json:"agent_id"`
	AgentName     string     `gorm:"size:255" json:"agent_name"`
	Squad         string     `gorm:"size:100" json:"squad"`
	Status        string     `gorm:"size:50" json:"status"` // running, paused, stopped, error
	LastHeartbeat time.Time  `json:"last_heartbeat"`
	FailedCount   int        `json:"failed_count"`
	ErrorReason   string     `json:"error_reason"`
	IsPaused      bool       `json:"is_paused"`
}

// TableName overrides the default table name.
func (AgentStatus) TableName() string { return "agent_status" }

// LLMCostRecord tracks token consumption per agent run.
type LLMCostRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AgentID      string    `gorm:"index;size:100" json:"agent_id"`
	ModelName    string    `gorm:"size:100" json:"model_name"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName overrides the default table name.
func (LLMCostRecord) TableName() string { return "llm_cost_records" }

// FailureRecord tracks agent action failures.
type FailureRecord struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	AgentID      string     `gorm:"index;size:100" json:"agent_id"`
	Action       string     `gorm:"size:255" json:"action"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	RetryCount   int        `json:"retry_count"`
	Status       string     `gorm:"size:50" json:"status"` // pending, resolved, ignored
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at"`
}

// TableName overrides the default table name.
func (FailureRecord) TableName() string { return "failure_records" }

// AgentStatusResponse wraps agent status list for API responses.
type AgentStatusResponse struct {
	Statuses []AgentStatus `json:"statuses"`
	Total    int64         `json:"total"`
}

// LLMCostResponse wraps LLM cost summary for API responses.
type LLMCostResponse struct {
	Period       string      `json:"period"`
	TotalTokens  int         `json:"total_tokens"`
	TotalCostUSD float64     `json:"total_cost_usd"`
	ByAgent      []AgentCost `json:"by_agent"`
}

// AgentCost summarizes cost per agent.
type AgentCost struct {
	AgentID      string  `json:"agent_id"`
	AgentName    string  `json:"agent_name"`
	TotalTokens  int     `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}
