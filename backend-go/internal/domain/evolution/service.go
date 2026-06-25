package evolution

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service manages agent evolution: trust-score-based nudges and upgrades.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates an evolution service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// EvaluateNudges checks all agents and creates Nudge records for any that
// are eligible for an autonomy level upgrade.
func (s *Service) EvaluateNudges() ([]Nudge, error) {
	var nudges []Nudge
	registry := ai.DefaultRegistry()

	for _, agent := range registry.Agents {
		// Skip agents already at max level.
		if agent.Autonomy == "autonomous" {
			continue
		}

		// Query trust score for this agent.
		var ts trustscore.TrustScore
		if err := s.db.Where("agent_id = ?", agent.ID).First(&ts).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			s.logger.Warn("trust score query failed", zap.String("agent_id", agent.ID), zap.Error(err))
			continue
		}

		// Determine next level threshold.
		nextLevel := nextAutonomyLevel(ts.AutonomyLevel)
		if nextLevel == "" {
			continue
		}
		threshold, ok := trustscore.AutonomyThresholds[nextLevel]
		if !ok {
			continue
		}

		// Agent's trust score must be at 80% of the next threshold to trigger a nudge.
		triggerThreshold := threshold * 0.80
		if ts.TrustScore < triggerThreshold {
			continue
		}

		// Check if there's already a pending nudge for this (agent, user, target).
		var existing int64
		s.db.Model(&Nudge{}).Where(
			"agent_id = ? AND target_level = ? AND status = 'pending'",
			agent.ID, nextLevel,
		).Count(&existing)
		if existing > 0 {
			continue // Don't create duplicate nudges.
		}

		// Build metrics snapshot.
		metrics := map[string]interface{}{
			"trust_score":       ts.TrustScore,
			"threshold":         threshold,
			"trigger_at":        triggerThreshold,
			"total_actions":     ts.TotalActions,
			"adoption_rate":     ts.AdoptionRate,
			"execution_success": ts.ExecutionSuccess,
			"avg_confidence":    ts.AvgConfidence,
		}
		metricsJSON, _ := json.Marshal(metrics)

		msg := upgradeMessage(agent.ID, agent.Name, ts.AutonomyLevel, nextLevel, ts.TrustScore)

		nudge := Nudge{
			UserID:       0, // System-level nudge; for multi-user, iterate users.
			AgentID:      agent.ID,
			CurrentLevel: ts.AutonomyLevel,
			TargetLevel:  nextLevel,
			TrustScore:   ts.TrustScore,
			Status:       "pending",
			Message:      msg,
			Metrics:      metricsJSON,
		}
		if err := s.db.Create(&nudge).Error; err != nil {
			s.logger.Warn("nudge creation failed", zap.String("agent_id", agent.ID), zap.Error(err))
			continue
		}
		nudges = append(nudges, nudge)

		s.logger.Info("nudge created",
			zap.String("agent_id", agent.ID),
			zap.String("from", ts.AutonomyLevel),
			zap.String("to", nextLevel),
			zap.Float64("trust_score", ts.TrustScore))
	}

	return nudges, nil
}

// ListNudges returns pending nudges, optionally filtered by agent and user.
func (s *Service) ListNudges(userID *int64, agentID string, status string) ([]Nudge, error) {
	q := s.db.Model(&Nudge{}).Order("created_at DESC")
	if userID != nil {
		q = q.Where("user_id = ?", *userID)
	}
	if agentID != "" {
		q = q.Where("agent_id = ?", agentID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var nudges []Nudge
	if err := q.Find(&nudges).Error; err != nil {
		return nil, err
	}
	return nudges, nil
}

// AcceptNudge marks a nudge as accepted and triggers the upgrade.
func (s *Service) AcceptNudge(nudgeID int64) error {
	var n Nudge
	if err := s.db.First(&n, nudgeID).Error; err != nil {
		return err
	}
	if n.Status != "pending" {
		return fmt.Errorf("nudge %d is not pending (status: %s)", nudgeID, n.Status)
	}

	now := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Update nudge status.
		if err := tx.Model(&n).Updates(map[string]interface{}{
			"status":     "accepted",
			"decided_at": now,
		}).Error; err != nil {
			return err
		}
		// Update trust score autonomy level.
		if err := tx.Model(&trustscore.TrustScore{}).
			Where("agent_id = ?", n.AgentID).
			Update("autonomy_level", n.TargetLevel).Error; err != nil {
			return err
		}
		return nil
	})
}

// DismissNudge marks a nudge as dismissed.
func (s *Service) DismissNudge(nudgeID int64) error {
	now := time.Now()
	return s.db.Model(&Nudge{}).Where("id = ?", nudgeID).Updates(map[string]interface{}{
		"status":     "dismissed",
		"decided_at": now,
	}).Error
}

// nextAutonomyLevel returns the next level in the progression.
func nextAutonomyLevel(current string) string {
	levels := []string{"advisory", "guided", "supervised", "autonomous"}
	for i, l := range levels {
		if l == current && i < len(levels)-1 {
			return levels[i+1]
		}
	}
	return ""
}

// upgradeMessage generates a Chinese-language nudge message.
func upgradeMessage(agentID, name, from, to string, score float64) string {
	templates := map[string]string{
		"guided":     "%s (%s) 的信任分已达 %.1f%%，建议从「观察」升级为「引导」—— 可创建待审批的动作",
		"supervised": "%s (%s) 的信任分已达 %.1f%%，建议从「引导」升级为「监督」—— 动作自动创建但仍需审批",
		"autonomous": "%s (%s) 的信任分已达 %.1f%%，建议从「监督」升级为「自主」—— 低风险动作可自动执行",
	}
	msg, ok := templates[to]
	if !ok {
		msg = "%s (%s) 的信任分已达 %.1f%%，建议升级自主度"
	}
	return fmt.Sprintf(msg, name, agentID, score*100)
}
