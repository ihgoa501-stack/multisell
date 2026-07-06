package reliability

import (
	"context"
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

// GetAgentStatus returns all agent heartbeat statuses.
func (s *Service) GetAgentStatus(ctx context.Context) ([]AgentStatus, error) {
	var statuses []AgentStatus
	if err := s.db.WithContext(ctx).Model(&AgentStatus{}).Order("agent_id ASC").Find(&statuses).Error; err != nil {
		return nil, err
	}
	return statuses, nil
}

// UpsertAgentHeartbeat creates or updates an agent's heartbeat status.
func (s *Service) UpsertAgentHeartbeat(ctx context.Context, agentID, agentName, squad, status, errorReason string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Where("agent_id = ?", agentID).Assign(&AgentStatus{
		AgentName:     agentName,
		Squad:         squad,
		Status:        status,
		LastHeartbeat: now,
		ErrorReason:   errorReason,
	}).FirstOrCreate(&AgentStatus{}, &AgentStatus{
		AgentID: agentID,
	}).Error
}

// RecordFailure creates a new failure record.
func (s *Service) RecordFailure(ctx context.Context, agentID, action, errorMsg string, retryCount int) error {
	return s.db.WithContext(ctx).Create(&FailureRecord{
		AgentID:      agentID,
		Action:       action,
		ErrorMessage: errorMsg,
		RetryCount:   retryCount,
		Status:       "pending",
	}).Error
}

// GetFailures returns all failure records ordered by most recent first.
func (s *Service) GetFailures(ctx context.Context) ([]FailureRecord, error) {
	var records []FailureRecord
	if err := s.db.WithContext(ctx).Model(&FailureRecord{}).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// ResolveFailure marks a failure record as resolved.
func (s *Service) ResolveFailure(ctx context.Context, id uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&FailureRecord{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      "resolved",
		"resolved_at": &now,
	}).Error
}

// GetLLMCost returns LLM cost summary for a given period.
func (s *Service) GetLLMCost(ctx context.Context, period string) (*LLMCostResponse, error) {
	var since time.Time
	now := time.Now()
	switch period {
	case "today":
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		since = now.AddDate(0, 0, -int(weekday-time.Monday))
	case "month":
		since = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	var totalTokens int
	var totalCost float64
	type agentRow struct {
		AgentID      string
		TotalTokens  int
		TotalCostUSD float64
	}
	var byAgent []agentRow

	if err := s.db.WithContext(ctx).Model(&LLMCostRecord{}).
		Select("COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens, COALESCE(SUM(cost_usd), 0) as total_cost").
		Where("created_at >= ?", since).
		Row().Scan(&totalTokens, &totalCost); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(&LLMCostRecord{}).
		Select("agent_id, COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("created_at >= ?", since).
		Group("agent_id").
		Order("total_cost_usd DESC").
		Find(&byAgent).Error; err != nil {
		return nil, err
	}

	ac := make([]AgentCost, len(byAgent))
	for i, a := range byAgent {
		ac[i] = AgentCost{
			AgentID:      a.AgentID,
			TotalTokens:  a.TotalTokens,
			TotalCostUSD: a.TotalCostUSD,
		}
	}

	return &LLMCostResponse{
		Period:       period,
		TotalTokens:  totalTokens,
		TotalCostUSD: totalCost,
		ByAgent:      ac,
	}, nil
}

// RecordLLMCost inserts an LLM cost record.
func (s *Service) RecordLLMCost(ctx context.Context, r LLMCostRecord) error {
	return s.db.WithContext(ctx).Create(&r).Error
}
