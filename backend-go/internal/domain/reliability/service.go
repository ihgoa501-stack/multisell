package reliability

import (
	"errors"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/costcontrol"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ErrBudgetExceeded is returned when the monthly LLM budget is exhausted.
var ErrBudgetExceeded = errors.New("monthly LLM budget exceeded")

// Service provides LLM budget management logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new reliability service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// GetBudget returns the current LLMBudget row, initialising it if none exists.
func (s *Service) GetBudget() (*LLMBudget, error) {
	var b LLMBudget
	if err := s.db.First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			b = LLMBudget{MonthlyLimitUSD: 0, BudgetMonth: time.Now().Format("2006-01")}
			if err := s.db.Create(&b).Error; err != nil {
				return nil, err
			}
			return &b, nil
		}
		return nil, err
	}
	s.applyObservedMonthlySpend(&b)
	return &b, nil
}

// SetBudget updates the monthly limit. If the budget month has rolled over,
// it also resets CurrentMonthUSD to 0.
func (s *Service) SetBudget(limitUSD float64) (*LLMBudget, error) {
	b, err := s.GetBudget()
	if err != nil {
		return nil, err
	}
	currentMonth := time.Now().Format("2006-01")
	if b.BudgetMonth != currentMonth {
		b.CurrentMonthUSD = 0
		b.BudgetMonth = currentMonth
	}
	b.MonthlyLimitUSD = limitUSD
	b.IsPaused = false
	if err := s.db.Save(b).Error; err != nil {
		return nil, err
	}
	s.applyObservedMonthlySpend(b)
	return b, nil
}

// CheckBudget returns true if LLM calls are allowed (budget not exceeded and not paused).
func (s *Service) CheckBudget() (bool, error) {
	b, err := s.GetBudget()
	if err != nil {
		return true, err // on db error, allow to avoid blocking
	}
	if b.MonthlyLimitUSD <= 0 {
		return true, nil // no limit = unlimited
	}
	s.applyObservedMonthlySpend(b)
	if b.IsPaused {
		s.logger.Warn("LLM calls paused by admin")
		return false, ErrBudgetExceeded
	}
	if b.CurrentMonthUSD >= b.MonthlyLimitUSD {
		s.logger.Warn("monthly LLM budget exceeded",
			zap.Float64("current", b.CurrentMonthUSD),
			zap.Float64("limit", b.MonthlyLimitUSD),
		)
		return false, ErrBudgetExceeded
	}
	return true, nil
}

// RecordSpend adds cost to the current month's total.
func (s *Service) RecordSpend(costUSD float64) error {
	b, err := s.GetBudget()
	if err != nil {
		return err
	}
	currentMonth := time.Now().Format("2006-01")
	if b.BudgetMonth != currentMonth {
		b.CurrentMonthUSD = 0
		b.BudgetMonth = currentMonth
	}
	b.CurrentMonthUSD += costUSD
	b.IsPaused = b.CurrentMonthUSD >= b.MonthlyLimitUSD && b.MonthlyLimitUSD > 0
	if b.IsPaused {
		s.logger.Warn("LLM calls auto-paused: monthly budget reached",
			zap.Float64("current", b.CurrentMonthUSD),
			zap.Float64("limit", b.MonthlyLimitUSD),
		)
	}
	now := time.Now()
	b.UpdatedAt = now
	// Use Updates to avoid zero-value overwrite issues
	return s.db.Model(&LLMBudget{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
		"current_month_usd": b.CurrentMonthUSD,
		"budget_month":      b.BudgetMonth,
		"is_paused":         b.IsPaused,
		"updated_at":        now,
	}).Error
}

// BudgetResponse is the JSON payload for the budget endpoint.
type BudgetResponse struct {
	MonthlyLimitUSD float64 `json:"monthly_limit_usd"`
	CurrentMonthUSD float64 `json:"current_month_usd"`
	BudgetMonth     string  `json:"budget_month"`
	IsPaused        bool    `json:"is_paused"`
	RemainingUSD    float64 `json:"remaining_usd"`
}

// InBudgetResponse converts the internal model to an API response.
func InBudgetResponse(b *LLMBudget) BudgetResponse {
	r := BudgetResponse{
		MonthlyLimitUSD: b.MonthlyLimitUSD,
		CurrentMonthUSD: b.CurrentMonthUSD,
		BudgetMonth:     b.BudgetMonth,
		IsPaused:        b.IsPaused,
	}
	if b.MonthlyLimitUSD > 0 {
		r.RemainingUSD = b.MonthlyLimitUSD - b.CurrentMonthUSD
		if r.RemainingUSD < 0 {
			r.RemainingUSD = 0
		}
	}
	return r
}

// BudgetInput is the JSON body for setting the budget.
type BudgetInput struct {
	MonthlyLimitUSD float64 `json:"monthly_limit_usd" binding:"min=0"`
}

// Ensure BudgetInput satisfies a common validation interface.
func (i *BudgetInput) Validate() error {
	if i.MonthlyLimitUSD < 0 {
		return fmt.Errorf("monthly_limit_usd must be >= 0")
	}
	return nil
}

func (s *Service) applyObservedMonthlySpend(b *LLMBudget) {
	currentMonth := time.Now().Format("2006-01")
	observed, err := s.monthlySpendUSD(currentMonth)
	if err != nil {
		s.logger.Warn("failed to aggregate monthly LLM spend", zap.Error(err))
		return
	}
	if b.BudgetMonth != currentMonth {
		b.CurrentMonthUSD = observed
		b.BudgetMonth = currentMonth
	} else if observed > b.CurrentMonthUSD {
		b.CurrentMonthUSD = observed
	}
	if b.MonthlyLimitUSD > 0 && b.CurrentMonthUSD >= b.MonthlyLimitUSD {
		b.IsPaused = true
	}
}

func (s *Service) monthlySpendUSD(month string) (float64, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return 0, err
	}
	end := start.AddDate(0, 1, 0)
	var total float64
	err = s.db.Model(&costcontrol.CostLog{}).
		Select("COALESCE(SUM(cost_usd),0)").
		Where("window_date >= ? AND window_date < ?", start, end).
		Scan(&total).Error
	return total, err
}
