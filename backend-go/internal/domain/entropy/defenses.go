package entropy

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TTLSweeper 第一层防线：过期规则自动退休。
type TTLSweeper struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewTTLSweeper(db *gorm.DB, logger *zap.Logger) *TTLSweeper {
	return &TTLSweeper{db: db, logger: logger}
}

func (s *TTLSweeper) ExpireStaleRules(userID int64) (int, error) {
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	res := s.db.Model(&PersonalRule{}).
		Where("user_id = ? AND status = 'active' AND (last_applied_at IS NULL OR last_applied_at < ?)", userID, cutoff).
		Update("status", "expired")
	return int(res.RowsAffected), res.Error
}

// BudgetEnforcer 第二层防线：规则数量封顶。
type BudgetEnforcer struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBudgetEnforcer(db *gorm.DB, logger *zap.Logger) *BudgetEnforcer {
	return &BudgetEnforcer{db: db, logger: logger}
}

func (s *BudgetEnforcer) EnforceBudgets(userID int64, _ map[string]interface{}) (map[string]int, error) {
	var count int64
	s.db.Model(&PersonalRule{}).Where("user_id = ? AND status = 'active'", userID).Count(&count)
	result := map[string]int{"active": int(count)}
	const maxRules = 50
	if count > maxRules {
		var overCount int64 = count - maxRules
		var lowPriority []PersonalRule
		s.db.Where("user_id = ? AND status = 'active'", userID).
			Order("priority ASC, confidence ASC").
			Limit(int(overCount)).
			Find(&lowPriority)
		for _, r := range lowPriority {
			s.db.Model(&r).Update("status", "shadow")
		}
		result["disabled"] = int(overCount)
	}
	return result, nil
}

// DecayScheduler 第三层防线：置信度衰减。
type DecayScheduler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewDecayScheduler(db *gorm.DB, logger *zap.Logger) *DecayScheduler {
	return &DecayScheduler{db: db, logger: logger}
}

func (s *DecayScheduler) ApplyDecay(userID int64) (int, error) {
	cutoff := time.Now().Add(-14 * 24 * time.Hour)
	res := s.db.Model(&PersonalRule{}).
		Where("user_id = ? AND status = 'active' AND updated_at < ?", userID, cutoff).
		Update("confidence", gorm.Expr("CASE WHEN confidence * 0.95 < 0.1 THEN 0.1 ELSE confidence * 0.95 END"))
	return int(res.RowsAffected), res.Error
}

// MergeDetector 第四层防线：检测相似规则。
type MergeDetector struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewMergeDetector(db *gorm.DB, logger *zap.Logger) *MergeDetector {
	return &MergeDetector{db: db, logger: logger}
}

func (s *MergeDetector) FindDuplicates(userID int64) ([]DuplicatePair, error) {
	var rules []PersonalRule
	if err := s.db.Where("user_id = ? AND status = 'active'", userID).
		Order("agent_id, decision_point, priority DESC").
		Find(&rules).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]*PersonalRule)
	var pairs []DuplicatePair
	for i := range rules {
		r := &rules[i]
		key := r.AgentID + ":" + r.DecisionPoint
		if existing, ok := seen[key]; ok {
			pairs = append(pairs, DuplicatePair{Keep: existing, Remove: r, Similarity: 0.85})
		} else {
			seen[key] = r
		}
	}
	if pairs == nil {
		pairs = []DuplicatePair{}
	}
	return pairs, nil
}

// RegretAnalyzer 第五层防线：分析用户否决模式。
type RegretAnalyzer struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewRegretAnalyzer(db *gorm.DB, logger *zap.Logger) *RegretAnalyzer {
	return &RegretAnalyzer{db: db, logger: logger}
}

func (s *RegretAnalyzer) AnalyzeOverrides(userID int64) (map[string]int, error) {
	type result struct {
		AgentID       string
		DecisionPoint string
		Count         int
	}
	var rows []result
	s.db.Model(&AgentDecision{}).
		Select("agent_id, decision_point, COUNT(*) as count").
		Where("user_id = ? AND user_action = 'override'", userID).
		Group("agent_id, decision_point").
		Order("count DESC").
		Limit(10).
		Scan(&rows)
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.AgentID+":"+r.DecisionPoint] = r.Count
	}
	return out, nil
}

func (s *RegretAnalyzer) GetInsights() ([]string, error) {
	type row struct {
		AgentID string
		Count   int
	}
	var rows []row
	s.db.Model(&AgentDecision{}).
		Select("agent_id, COUNT(*) as count").
		Where("user_action = 'override'").
		Group("agent_id").
		Order("count DESC").
		Limit(5).
		Scan(&rows)
	insights := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Count >= 10 {
			insights = append(insights, fmt.Sprintf("Agent %s has %d overrides", r.AgentID, r.Count))
		}
	}
	return insights, nil
}
