package entropy

import (
	"sort"

	"go.uber.org/zap"
)

// GetAgentHealthScore returns the median health score for an agent across all
// its rules. If decisionPoint is not empty, scores are further filtered to
// only rules matching that decision point.
func (s *Service) GetAgentHealthScore(agentID string, decisionPoint string) (float64, error) {
	var rules []PersonalRule
	q := s.db.Where("agent_id = ?", agentID)
	if decisionPoint != "" {
		q = q.Where("decision_point = ?", decisionPoint)
	}
	if err := q.Find(&rules).Error; err != nil {
		return 0, err
	}
	if len(rules) == 0 {
		return 0, nil
	}

	scores := make([]float64, 0, len(rules))
	for i := range rules {
		hs := s.scorer.ScoreRule(&rules[i])
		scores = append(scores, hs.Score)
	}

	return median(scores), nil
}

// UnhealthyAgent represents an agent whose health score has dropped below the
// unhealthy threshold.
type UnhealthyAgent struct {
	AgentID     string  `json:"agent_id"`
	HealthScore float64 `json:"health_score"`
}

// CheckAgentHealth iterates all unique agents with registered rules and returns
// those whose aggregate health score falls below the unhealthy threshold (0.4).
func (s *Service) CheckAgentHealth() ([]UnhealthyAgent, error) {
	var agentIDs []string
	if err := s.db.Model(&PersonalRule{}).
		Select("DISTINCT agent_id").
		Pluck("agent_id", &agentIDs).Error; err != nil {
		return nil, err
	}

	var unhealthy []UnhealthyAgent
	for _, agentID := range agentIDs {
		score, err := s.GetAgentHealthScore(agentID, "")
		if err != nil {
			s.logger.Warn("failed to score agent health",
				zap.String("agent_id", agentID),
				zap.Error(err),
			)
			continue
		}
		if score < unhealthyThreshold {
			unhealthy = append(unhealthy, UnhealthyAgent{
				AgentID:     agentID,
				HealthScore: score,
			})
		}
	}

	return unhealthy, nil
}

// median returns the median of a float64 slice. Returns 0 for an empty slice.
func median(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := make([]float64, n)
	copy(sorted, values)
	sort.Float64s(sorted)

	if n%2 == 0 {
		return round4((sorted[n/2-1] + sorted[n/2]) / 2.0)
	}
	return round4(sorted[n/2])
}
