package reliability

import "time"

// LLMBudget tracks monthly LLM spending with a hard cap.
type LLMBudget struct {
	ID              uint      `gorm:"primaryKey"`
	MonthlyLimitUSD float64   `gorm:"column:monthly_limit_usd;not null;default:200"`
	CurrentMonthUSD float64   `gorm:"column:current_month_usd;default:0"`
	BudgetMonth     string    `gorm:"column:budget_month;size:7"`
	IsPaused        bool      `gorm:"column:is_paused;default:false"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (LLMBudget) TableName() string { return "llm_budgets" }

// BudgetConfig is the request/response for the budget API.
type BudgetConfig struct {
	MonthlyLimitUSD float64 `json:"monthly_limit_usd"`
}
