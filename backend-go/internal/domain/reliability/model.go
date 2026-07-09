package reliability

import "time"

// LLMBudget is the monthly LLM spend budget with a hard cap.
// There should be exactly one row; BudgetMonth is "YYYY-MM" to scope the current
// month's spend. When CurrentMonthUSD >= MonthlyLimitUSD (or IsPaused is true),
// the AI orchestrator blocks further LLM calls.
type LLMBudget struct {
	ID              uint      `gorm:"primaryKey"`
	MonthlyLimitUSD float64   `gorm:"not null"`
	CurrentMonthUSD float64   `gorm:"default:0"`
	BudgetMonth     string    `gorm:"size:7"`
	IsPaused        bool      `gorm:"default:false"`
	UpdatedAt       time.Time
}
