package cost

import (
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/costcontrol"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides LLM cost dashboard queries.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new cost service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// rowDaily is the raw scan target for daily cost aggregation.
type rowDaily struct {
	WindowDate string
	TotalCost  float64
	TotalCalls int64
}

// GetDailySummary returns per-day cost aggregates for the last N days.
func (s *Service) GetDailySummary(days int) ([]DailyCostVO, error) {
	if days <= 0 {
		days = 7
	}
	var rows []rowDaily
	if err := s.db.Model(&costcontrol.CostLog{}).
		Select("window_date, SUM(cost_usd) AS total_cost, COUNT(*) AS total_calls").
		Where("window_date >= CURRENT_DATE - INTERVAL ? DAY", fmt.Sprintf("%d", days)).
		Group("window_date").
		Order("window_date ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]DailyCostVO, 0, len(rows))
	for _, r := range rows {
		out = append(out, DailyCostVO{
			Date:    r.WindowDate,
			CostUSD: r.TotalCost,
			Calls:   int(r.TotalCalls),
		})
	}
	return out, nil
}

// rowAgent is the raw scan target for per-agent cost aggregation.
type rowAgent struct {
	AgentID    string
	TotalCost  float64
	TotalCalls int64
}

// GetAgentSummary returns per-agent cost aggregates for the last N days.
func (s *Service) GetAgentSummary(days int) ([]AgentCostVO, error) {
	if days <= 0 {
		days = 7
	}
	var rows []rowAgent
	if err := s.db.Model(&costcontrol.CostLog{}).
		Select("agent_id, SUM(cost_usd) AS total_cost, COUNT(*) AS total_calls").
		Where("window_date >= CURRENT_DATE - INTERVAL ? DAY", fmt.Sprintf("%d", days)).
		Group("agent_id").
		Order("total_cost DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AgentCostVO, 0, len(rows))
	for _, r := range rows {
		out = append(out, AgentCostVO{
			AgentID: r.AgentID,
			CostUSD: r.TotalCost,
			Calls:   int(r.TotalCalls),
		})
	}
	return out, nil
}

// todaySummary returns the single-day cost for today (or zero if no rows).
func (s *Service) todaySummary() (DailyCostVO, error) {
	today := time.Now().UTC().Format("2006-01-02")
	var r rowDaily
	err := s.db.Model(&costcontrol.CostLog{}).
		Select("window_date, COALESCE(SUM(cost_usd),0) AS total_cost, COUNT(*) AS total_calls").
		Where("window_date = CURRENT_DATE").
		Group("window_date").
		Take(&r).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return DailyCostVO{Date: today, CostUSD: 0, Calls: 0}, nil
		}
		return DailyCostVO{}, err
	}
	return DailyCostVO{
		Date:    r.WindowDate,
		CostUSD: r.TotalCost,
		Calls:   int(r.TotalCalls),
	}, nil
}

// GetDashboard returns the full cost dashboard with budget calculations.
func (s *Service) GetDashboard(budgetUSD float64) (*CostDashboardVO, error) {
	today, err := s.todaySummary()
	if err != nil {
		return nil, fmt.Errorf("today summary: %w", err)
	}

	last7, err := s.GetDailySummary(7)
	if err != nil {
		return nil, fmt.Errorf("last 7 days: %w", err)
	}

	byAgent, err := s.GetAgentSummary(7)
	if err != nil {
		return nil, fmt.Errorf("by agent: %w", err)
	}

	pct := 0.0
	if budgetUSD > 0 {
		pct = (today.CostUSD / budgetUSD) * 100
	}

	return &CostDashboardVO{
		Today:       today,
		Last7Days:   last7,
		ByAgent:     byAgent,
		DailyBudget: budgetUSD,
		BudgetUsed:  today.CostUSD,
		BudgetPct:   pct,
	}, nil
}
