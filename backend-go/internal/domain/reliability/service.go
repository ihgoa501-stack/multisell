package reliability

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides reliability business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new reliability service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// GetBudget returns the current budget state. Creates a default row if none exists.
func (s *Service) GetBudget() (*LLMBudget, error) {
	var b LLMBudget
	if err := s.db.First(&b).Error; err != nil {
		now := time.Now()
		b = LLMBudget{
			MonthlyLimitUSD: 200,
			BudgetMonth:     now.Format("2006-01"),
		}
		if err := s.db.Create(&b).Error; err != nil {
			return nil, fmt.Errorf("create default budget: %w", err)
		}
	}
	return &b, nil
}

// SetBudget updates the monthly limit.
func (s *Service) SetBudget(limit float64) error {
	b, err := s.GetBudget()
	if err != nil {
		return err
	}
	return s.db.Model(&LLMBudget{}).Where("id = ?", b.ID).Update("monthly_limit_usd", limit).Error
}

// CheckBudget returns true if the monthly budget is not exceeded.
func (s *Service) CheckBudget() (bool, error) {
	b, err := s.GetBudget()
	if err != nil {
		return false, err
	}
	return b.CurrentMonthUSD < b.MonthlyLimitUSD && !b.IsPaused, nil
}

// RecordSpend adds to the current month's spend. Auto-pauses if limit exceeded.
// UpsertAgentHeartbeat records or updates an agent's heartbeat timestamp.
func (s *Service) UpsertAgentHeartbeat(ctx context.Context, agentID, agentName, squad, status, errReason string) error {
	return nil // ponytail: heartbeat tracking stubbed
}

func (s *Service) RecordSpend(amountUSD float64) error {
	b, err := s.GetBudget()
	if err != nil {
		return err
	}
	newTotal := b.CurrentMonthUSD + amountUSD
	updates := map[string]interface{}{"current_month_usd": newTotal}
	if newTotal >= b.MonthlyLimitUSD {
		updates["is_paused"] = true
	}
	return s.db.Model(&LLMBudget{}).Where("id = ?", b.ID).Updates(updates).Error
}
