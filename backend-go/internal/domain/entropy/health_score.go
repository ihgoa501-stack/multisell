package entropy

import (
	"math"
	"sort"
	"time"

	"gorm.io/gorm"
)

// RuleHealthScorer 计算5维加权健康评分。
type RuleHealthScorer struct {
	db *gorm.DB
}

var (
	healthWeights = map[string]float64{
		"acceptance": 0.30,
		"confidence": 0.25,
		"freshness":  0.20,
		"frequency":  0.15,
		"type_weight": 0.10,
	}

	typeScores = map[string]float64{
		"veto":      1.0,
		"threshold": 0.8,
		"strategy":  0.6,
		"style":     0.4,
	}

	unhealthyThreshold = 0.40
	warningThreshold   = 0.60
)

func NewRuleHealthScorer(db *gorm.DB) *RuleHealthScorer {
	return &RuleHealthScorer{db: db}
}

// ScoreRule 计算单条规则的健康评分。
func (s *RuleHealthScorer) ScoreRule(rule *PersonalRule) RuleHealthScore {
	now := time.Now()

	applied := rule.TimesApplied
	if applied < 0 {
		applied = 0
	}
	overridden := rule.TimesOverridden
	if overridden < 0 {
		overridden = 0
	}

	// 1. 采纳率
	maxApplied := applied
	if maxApplied < 1 {
		maxApplied = 1
	}
	acceptance := 1.0 - float64(overridden)/float64(maxApplied)
	if acceptance < 0 {
		acceptance = 0
	}

	// 2. 置信度
	confidence := rule.Confidence
	if confidence < 0 {
		confidence = 0
	}

	// 3. 新鲜度
	var freshness float64
	if rule.LastAppliedAt != nil {
		daysSince := now.Sub(*rule.LastAppliedAt).Hours() / 24.0
		freshness = math.Max(0, 1.0-(daysSince/180.0))
	} else {
		freshness = 0.2
	}

	// 4. 应用频次
	frequency := 1.0 / (1.0 + math.Exp(-0.1*float64(applied)))

	// 5. 规则类型权重
	typeScore := 0.5
	if ts, ok := typeScores[rule.RuleType]; ok {
		typeScore = ts
	}

	total := acceptance*healthWeights["acceptance"] +
		confidence*healthWeights["confidence"] +
		freshness*healthWeights["freshness"] +
		frequency*healthWeights["frequency"] +
		typeScore*healthWeights["type_weight"]

	var riskLevel string
	switch {
	case total < unhealthyThreshold:
		riskLevel = "unhealthy"
	case total < warningThreshold:
		riskLevel = "warning"
	default:
		riskLevel = "healthy"
	}

	var daysSince *int
	if rule.LastAppliedAt != nil {
		d := int(now.Sub(*rule.LastAppliedAt).Hours() / 24.0)
		daysSince = &d
	}

	overrideRate := 0.0
	if applied > 0 {
		overrideRate = float64(overridden) / float64(applied)
	}

	return RuleHealthScore{
		RuleID:        rule.ID,
		RuleName:      rule.RuleName,
		RuleType:      rule.RuleType,
		AgentID:       rule.AgentID,
		DecisionPoint: rule.DecisionPoint,
		Status:        rule.Status,
		Score:         round4(total),
		Dimensions: HealthDimensions{
			Acceptance: round4(acceptance),
			Confidence: round4(confidence),
			Freshness:  round4(freshness),
			Frequency:  round4(frequency),
			TypeWeight: round4(typeScore),
		},
		TimesApplied:         applied,
		TimesOverridden:      overridden,
		OverrideRate:         round4(overrideRate),
		DaysSinceLastApplied: daysSince,
		Confidence:           rule.Confidence,
		RiskLevel:            riskLevel,
	}
}

// ScoreAllRules 计算 user 所有 active/shadow 规则的健康评分。
func (s *RuleHealthScorer) ScoreAllRules(userID int64) ([]RuleHealthScore, error) {
	var rules []PersonalRule
	if err := s.db.Where(
		"user_id = ? AND status IN (?)",
		userID, []string{"active", "shadow"},
	).Find(&rules).Error; err != nil {
		return nil, err
	}

	scores := make([]RuleHealthScore, 0, len(rules))
	for i := range rules {
		scores = append(scores, s.ScoreRule(&rules[i]))
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score < scores[j].Score
	})

	return scores, nil
}

// GetSummary 返回聚合健康摘要。
func (s *RuleHealthScorer) GetSummary(userID int64) (*HealthSummary, error) {
	scores, err := s.ScoreAllRules(userID)
	if err != nil {
		return nil, err
	}

	total := len(scores)
	summary := &HealthSummary{TotalRules: total}

	if total == 0 {
		return summary, nil
	}

	var sum float64
	for _, sc := range scores {
		sum += sc.Score
		switch sc.RiskLevel {
		case "unhealthy":
			summary.UnhealthyCount++
		case "warning":
			summary.WarningCount++
		case "healthy":
			summary.HealthyCount++
		}
	}
	summary.AvgHealthScore = round4(sum / float64(total))

	// Count active/shadow directly from DB
	var activeCount, shadowCount int64
	s.db.Model(&PersonalRule{}).Where("user_id = ? AND status = ?", userID, "active").Count(&activeCount)
	s.db.Model(&PersonalRule{}).Where("user_id = ? AND status = ?", userID, "shadow").Count(&shadowCount)
	summary.ActiveRules = int(activeCount)
	summary.ShadowRules = int(shadowCount)

	return summary, nil
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
