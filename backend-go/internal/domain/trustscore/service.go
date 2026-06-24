package trustscore

import (
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service calculates and manages agent trust scores.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a trust score service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns all trust scores.
func (s *Service) List() ([]TrustScore, error) {
	var scores []TrustScore
	if err := s.db.Order("trust_score DESC").Find(&scores).Error; err != nil {
		return nil, err
	}
	return scores, nil
}

// GetByAgent returns trust score for a specific agent.
func (s *Service) GetByAgent(agentID string) (*TrustScore, error) {
	var score TrustScore
	if err := s.db.Where("agent_id = ?", agentID).First(&score).Error; err != nil {
		// If not found, return nil without error
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &score, nil
}

// Recalculate recomputes trust scores for all registered agents.
func (s *Service) Recalculate() error {
	for _, agent := range defaultAgentList() {
		if err := s.RecalculateForAgent(agent.ID, agent.Name, agent.Squad); err != nil {
			s.logger.Warn("recalculate failed", zap.String("agent", agent.ID), zap.Error(err))
		}
	}
	return nil
}

// RecalculateForAgent recomputes trust score for one agent.
func (s *Service) RecalculateForAgent(agentID, agentName, squadID string) error {
	// Get existing score or create new
	score, err := s.GetByAgent(agentID)
	if err != nil {
		return err
	}
	if score == nil {
		score = NewTrustScore(agentID, agentName, squadID)
	}

	// Query action stats
	var actionStats struct {
		Total   int
		Adopted int   // approved + executed
		Rejected int
		Failed  int
		Auto    int
	}
	s.db.Raw(`
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status IN ('approved','executing','executed','reviewed')) AS adopted,
			COUNT(*) FILTER (WHERE status = 'rejected' AND rejected_by != 'policy') AS rejected,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE status IN ('approved','executing','executed') AND approved_by = 'policy') AS auto
		FROM unified_action WHERE agent_id = ?
	`, agentID).Scan(&actionStats)

	// Query avg confidence from traces
	var confStats struct {
		AvgConf float64
	}
	s.db.Raw(`SELECT COALESCE(AVG(confidence),0) AS avg_conf FROM ai_trace WHERE agent_id = ? AND confidence IS NOT NULL`, agentID).Scan(&confStats)

	// Query last action time
	var lastAction struct {
		MaxTime *time.Time
	}
	s.db.Raw(`SELECT MAX(proposed_at) AS max_time FROM unified_action WHERE agent_id = ?`, agentID).Scan(&lastAction)

	// Calculate metrics
	total := float64(maxInt(actionStats.Total, 1))
	adoptionRate := float64(actionStats.Adopted) / total
	execSuccess := 1.0
	if actionStats.Failed > 0 {
		execSuccess = 1.0 - (float64(actionStats.Failed) / total)
	}
	avgConf := clamp01(confStats.AvgConf)

	// Composite trust score
	// adoptionRate (40%) + execSuccess (30%) + avgConf (30%)
	trustScore := adoptionRate*0.40 + execSuccess*0.30 + avgConf*0.30
	trustScore = clamp01(trustScore)

	// Determine current and target autonomy level
	currentLevel := score.AutonomyLevel
	targetLevel := determineTargetLevel(trustScore, currentLevel)

	// Update score record
	updates := map[string]interface{}{
		"agent_name":       agentName,
		"squad_id":         squadID,
		"total_actions":    actionStats.Total,
		"adopted_actions":  actionStats.Adopted,
		"rejected_actions": actionStats.Rejected,
		"failed_actions":   actionStats.Failed,
		"auto_approved":    actionStats.Auto,
		"adoption_rate":    math.Round(adoptionRate*10000) / 10000,
		"execution_success": math.Round(execSuccess*10000) / 10000,
		"avg_confidence":   math.Round(avgConf*10000) / 10000,
		"trust_score":      math.Round(trustScore*10000) / 10000,
		"autonomy_level":   currentLevel,
		"target_level":     targetLevel,
		"last_action_at":   lastAction.MaxTime,
	}

	if score.ID == 0 {
		// Insert new record
		score.AgentID = agentID
		score.AgentName = agentName
		score.SquadID = squadID
		for k, v := range updates {
			switch k {
			case "autonomy_level":
				score.AutonomyLevel = v.(string)
			}
		}
		return s.db.Create(score).Error
	}

	return s.db.Model(score).Updates(updates).Error
}

// GetEligibleForUpgrade returns agents whose trust score qualifies them
// for the next autonomy level.
func (s *Service) GetEligibleForUpgrade() ([]TrustScore, error) {
	var scores []TrustScore
	if err := s.db.Where("trust_score >= ? AND autonomy_level != 'autonomous'", AutonomyThresholds["supervised"]).Find(&scores).Error; err != nil {
		return nil, err
	}
	return scores, nil
}

// UpdateAutonomyLevel manually sets an agent's autonomy level.
func (s *Service) UpdateAutonomyLevel(agentID, level string) error {
	return s.db.Model(&TrustScore{}).Where("agent_id = ?", agentID).Update("autonomy_level", level).Error
}

// determineTargetLevel finds the highest autonomy level the agent qualifies for.
func determineTargetLevel(trustScore float64, currentLevel string) string {
	levels := []string{"autonomous", "supervised", "guided", "advisory"}
	levelOrder := map[string]int{
		"advisory":   0,
		"guided":     1,
		"supervised": 2,
		"autonomous": 3,
	}
	currentOrd := levelOrder[currentLevel]

	best := currentLevel
	for _, level := range levels {
		threshold := AutonomyThresholds[level]
		if trustScore >= threshold && levelOrder[level] >= currentOrd {
			best = level
		}
	}
	// Only upgrade by one level at a time
	if levelOrder[best] > currentOrd+1 {
		for _, l := range levels {
			if levelOrder[l] == currentOrd+1 {
				return l
			}
		}
	}
	return best
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}


// defaultAgentList returns the canonical list of agents for trust score calculation.
// Avoids importing the ai package to prevent circular dependencies.
func defaultAgentList() []agentInfo {
	return []agentInfo{
		{ID: "A1", Name: "Product Scout", Squad: "autonomous"},
		{ID: "A2", Name: "Listing Optimizer", Squad: "autonomous"},
		{ID: "A3", Name: "Ad Advice", Squad: "autonomous"},
		{ID: "A4", Name: "Customer Service", Squad: "autonomous"},
		{ID: "A5", Name: "Inventory Alert", Squad: "autonomous"},
		{ID: "A6", Name: "Profit Watch", Squad: "autonomous"},
		{ID: "A7", Name: "Compliance Guard", Squad: "autonomous"},
		{ID: "G1", Name: "Dashboard", Squad: "governance"},
		{ID: "G2", Name: "Warehouse Customs", Squad: "governance"},
		{ID: "G3", Name: "Discount Risk", Squad: "governance"},
		{ID: "G0", Name: "Coordinator", Squad: "governance"},
		{ID: "A8", Name: "Settlement Recon", Squad: "settle"},
		{ID: "A10", Name: "Logistics Ops", Squad: "fulfillment"},
		{ID: "A11", Name: "Aftersales Mgmt", Squad: "settle"},
	}
}

type agentInfo struct {
	ID    string
	Name  string
	Squad string
}
