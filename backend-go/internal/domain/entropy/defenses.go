package entropy

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TTLSweeper — 防线1：基于时间过期自动清理低价值规则。
type TTLSweeper struct {
	db  *gorm.DB
	log *zap.Logger
}

const defaultTTLDays = 90

func NewTTLSweeper(db *gorm.DB, log *zap.Logger) *TTLSweeper {
	return &TTLSweeper{db: db, log: log}
}

// ExpireStaleRules 将超过 TTL 且未使用的 active 规则标记为 retired。
func (s *TTLSweeper) ExpireStaleRules(userID int64, ttlDays ...int) ([]PersonalRule, error) {
	days := defaultTTLDays
	if len(ttlDays) > 0 && ttlDays[0] > 0 {
		days = ttlDays[0]
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	var rules []PersonalRule
	if err := s.db.Where(
		"user_id = ? AND status = ? AND last_applied_at IS NOT NULL AND last_applied_at < ?",
		userID, "active", cutoff,
	).Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("ttl: query stale rules: %w", err)
	}

	for _, r := range rules {
		oldVal := `"active"`
		newVal := `"retired"`
		if err := s.db.Model(&r).Update("status", "retired").Error; err != nil {
			s.log.Warn("ttl: failed to retire rule", zap.Int64("rule_id", r.ID), zap.Error(err))
			continue
		}
		if err := logChange(s.db, RuleMarkChange{
			TargetType:    "personal_rule",
			TargetID:      r.ID,
			FieldPath:     "$.status",
			OldValue:      &oldVal,
			NewValue:      newVal,
			SourceType:    "gds",
			SourceID:      strPtr("ttl_sweeper"),
			ChangeSummary: fmt.Sprintf("TTL过期自动退休: 超过%d天未使用", days),
			ContextJSON:   jsonStrPtr(map[string]interface{}{"ttl_days": days, "cutoff": cutoff}),
		}); err != nil {
			s.log.Warn("ttl: log change failed", zap.Error(err))
		}
	}

	return rules, nil
}

// BudgetEnforcer — 防线2：按类别限制规则数量。
type BudgetEnforcer struct {
	db  *gorm.DB
	log *zap.Logger
}

var defaultBudgets = map[string]int{
	"veto":      10,
	"threshold": 20,
	"strategy":  15,
	"style":     10,
}

func NewBudgetEnforcer(db *gorm.DB, log *zap.Logger) *BudgetEnforcer {
	return &BudgetEnforcer{db: db, log: log}
}

// EnforceBudgets 检查每类规则是否超限，超限部分降级为 shadow。
func (e *BudgetEnforcer) EnforceBudgets(userID int64, budgets map[string]int) ([]PersonalRule, error) {
	if budgets == nil {
		budgets = defaultBudgets
	}
	var allExceeded []PersonalRule

	for ruleType, limit := range budgets {
		var rules []PersonalRule
		if err := e.db.Where(
			"user_id = ? AND rule_type = ? AND status = ?",
			userID, ruleType, "active",
		).Order("priority DESC, COALESCE(last_applied_at, '1970-01-01') DESC").Find(&rules).Error; err != nil {
			return nil, fmt.Errorf("budget: query rules: %w", err)
		}

		if len(rules) > limit {
			excess := rules[limit:]
			for _, r := range excess {
				oldVal := `"active"`
				newVal := `"shadow"`
				if err := e.db.Model(&r).Update("status", "shadow").Error; err != nil {
					e.log.Warn("budget: failed to shadow rule", zap.Int64("rule_id", r.ID), zap.Error(err))
					continue
				}
				if err := logChange(e.db, RuleMarkChange{
					TargetType:    "personal_rule",
					TargetID:      r.ID,
					FieldPath:     "$.status",
					OldValue:      &oldVal,
					NewValue:      newVal,
					SourceType:    "gds",
					SourceID:      strPtr("budget_enforcer"),
					ChangeSummary: fmt.Sprintf("Budget超限: %s最多%d条, 降级为shadow", ruleType, limit),
					ContextJSON:   jsonStrPtr(map[string]interface{}{"rule_type": ruleType, "limit": limit, "total": len(rules)}),
				}); err != nil {
					e.log.Warn("budget: log change failed", zap.Error(err))
				}
				allExceeded = append(allExceeded, r)
			}
		}
	}

	return allExceeded, nil
}

// DecayScheduler — 防线3：未使用规则置信度衰减。
type DecayScheduler struct {
	db  *gorm.DB
	log *zap.Logger
}

const (
	defaultDecayRate    = 0.05
	minConfidence       = 0.1
)

func NewDecayScheduler(db *gorm.DB, log *zap.Logger) *DecayScheduler {
	return &DecayScheduler{db: db, log: log}
}

// ApplyDecay 对所有 active/shadow 规则降低置信度。
func (s *DecayScheduler) ApplyDecay(userID int64, decayRate ...float64) ([]PersonalRule, error) {
	rate := defaultDecayRate
	if len(decayRate) > 0 && decayRate[0] > 0 {
		rate = decayRate[0]
	}

	var rules []PersonalRule
	if err := s.db.Where(
		"user_id = ? AND status IN (?) AND confidence > ?",
		userID, []string{"active", "shadow"}, minConfidence,
	).Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("decay: query rules: %w", err)
	}

	var affected []PersonalRule
	for _, r := range rules {
		oldConf := r.Confidence
		newConf := oldConf - rate
		if newConf < minConfidence {
			newConf = minConfidence
		}
		if err := s.db.Model(&r).Update("confidence", newConf).Error; err != nil {
			s.log.Warn("decay: failed to update confidence", zap.Int64("rule_id", r.ID), zap.Error(err))
			continue
		}
		oldVal := fmt.Sprintf("%.4f", oldConf)
		newVal := fmt.Sprintf("%.4f", newConf)
		if err := logChange(s.db, RuleMarkChange{
			TargetType:    "personal_rule",
			TargetID:      r.ID,
			FieldPath:     "$.confidence",
			OldValue:      &oldVal,
			NewValue:      newVal,
			SourceType:    "gds",
			SourceID:      strPtr("decay_scheduler"),
			ChangeSummary: fmt.Sprintf("置信度衰减: %.4f → %.4f", oldConf, newConf),
			ContextJSON:   jsonStrPtr(map[string]interface{}{"decay_rate": rate}),
		}); err != nil {
			s.log.Warn("decay: log change failed", zap.Error(err))
		}
		affected = append(affected, r)
	}

	return affected, nil
}

// MergeDetector — 防线4：检测相似/冲突规则。
type MergeDetector struct {
	db  *gorm.DB
	log *zap.Logger
}

const defaultSimilarityThreshold = 0.85

func NewMergeDetector(db *gorm.DB, log *zap.Logger) *MergeDetector {
	return &MergeDetector{db: db, log: log}
}

// FindDuplicates 按 agent_id:decision_point:rule_type 分组，检测重复规则。
func (d *MergeDetector) FindDuplicates(userID int64) ([]DuplicatePair, error) {
	var rules []PersonalRule
	if err := d.db.Where(
		"user_id = ? AND status IN (?)",
		userID, []string{"active", "shadow"},
	).Order("agent_id, decision_point, created_at").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("merge: query rules: %w", err)
	}

	groups := make(map[string][]PersonalRule)
	for _, r := range rules {
		key := fmt.Sprintf("%s:%s:%s", r.AgentID, r.DecisionPoint, r.RuleType)
		groups[key] = append(groups[key], r)
	}

	var duplicates []DuplicatePair
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				if areSimilar(&group[i], &group[j]) {
					duplicates = append(duplicates, DuplicatePair{
						Keep:       &group[i],
						Remove:     &group[j],
						Similarity: defaultSimilarityThreshold,
					})
				}
			}
		}
	}

	return duplicates, nil
}

// areSimilar 比较两条规则的 rule_condition 和 rule_action。
func areSimilar(a, b *PersonalRule) bool {
	if a.RuleCondition == b.RuleCondition && a.RuleAction == b.RuleAction {
		return true
	}
	if a.RuleCondition != "" && b.RuleCondition != "" {
		var aCond, bCond map[string]interface{}
		if json.Unmarshal([]byte(a.RuleCondition), &aCond) == nil &&
			json.Unmarshal([]byte(b.RuleCondition), &bCond) == nil {
			aField, _ := aCond["field"].(string)
			bField, _ := bCond["field"].(string)
			if aField != "" && aField == bField {
				return true
			}
		}
	}
	return false
}

// MergeRules 合并两条规则的统计数据，将 remove 标记为 retired。
func (d *MergeDetector) MergeRules(keepID, removeID int64) (*PersonalRule, error) {
	var keep, remove PersonalRule
	if err := d.db.First(&keep, keepID).Error; err != nil {
		return nil, fmt.Errorf("merge: keep rule %d not found: %w", keepID, err)
	}
	if err := d.db.First(&remove, removeID).Error; err != nil {
		return nil, fmt.Errorf("merge: remove rule %d not found: %w", removeID, err)
	}

	keep.TimesApplied = keep.TimesApplied + remove.TimesApplied
	keep.TimesOverridden = keep.TimesOverridden + remove.TimesOverridden
	if remove.Confidence > keep.Confidence {
		keep.Confidence = remove.Confidence
	}

	tx := d.db.Begin()
	if err := tx.Model(&keep).Updates(map[string]interface{}{
		"times_applied":    keep.TimesApplied,
		"times_overridden": keep.TimesOverridden,
		"confidence":       keep.Confidence,
	}).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("merge: update keep: %w", err)
	}
	if err := tx.Model(&remove).Update("status", "retired").Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("merge: retire remove: %w", err)
	}

	oldStatus := `"active"`
	newStatus := `"retired"`
	_ = logChange(tx, RuleMarkChange{
		TargetType:    "personal_rule",
		TargetID:      keep.ID,
		FieldPath:     "$.merged_from",
		OldValue:      nil,
		NewValue:      fmt.Sprintf(`%d`, removeID),
		SourceType:    "gds",
		SourceID:      strPtr("merge_detector"),
		ChangeSummary: fmt.Sprintf("合并规则: %s(#%d) → %s(#%d)", remove.RuleName, removeID, keep.RuleName, keepID),
	})
	_ = logChange(tx, RuleMarkChange{
		TargetType:    "personal_rule",
		TargetID:      remove.ID,
		FieldPath:     "$.status",
		OldValue:      &oldStatus,
		NewValue:      newStatus,
		SourceType:    "gds",
		SourceID:      strPtr("merge_detector"),
		ChangeSummary: fmt.Sprintf("因合并到#%d而退休", keepID),
	})

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("merge: commit: %w", err)
	}
	return &keep, nil
}

// RegretAnalyzer — 防线5：分析被用户否决的决策规律。
type RegretAnalyzer struct {
	db  *gorm.DB
	log *zap.Logger
}

const (
	regretWindowHours = 48
	regretThreshold   = 0.15
)

func NewRegretAnalyzer(db *gorm.DB, log *zap.Logger) *RegretAnalyzer {
	return &RegretAnalyzer{db: db, log: log}
}

// FindRegrettableChanges 检测变更后指标下降超过阈值的规则。
type RegretChange struct {
	Change    RuleMarkChange
	Rule      PersonalRule
	BeforeAvg float64
	AfterAvg  float64
	Drop      float64
}

func (a *RegretAnalyzer) FindRegrettableChanges(userID int64) ([]RegretChange, error) {
	windowStart := time.Now().Add(-time.Hour * regretWindowHours)

	var changes []RuleMarkChange
	if err := a.db.Where(
		"source_type IN (?) AND created_at >= ?",
		[]string{"gds", "gds_proxy", "nudge"}, windowStart,
	).Order("created_at DESC").Find(&changes).Error; err != nil {
		return nil, fmt.Errorf("regret: query changes: %w", err)
	}

	var result []RegretChange
	for _, ch := range changes {
		if ch.TargetType != "personal_rule" {
			continue
		}
		var rule PersonalRule
		if err := a.db.First(&rule, ch.TargetID).Error; err != nil {
			continue
		}

		beforeStart := windowStart.AddDate(0, 0, -30)

		type AvgResult struct {
			Avg *float64
		}
		var beforeAvg AvgResult
		a.db.Raw(
			`SELECT AVG(confidence) as avg FROM agent_decision WHERE user_id = ? AND agent_id = ? AND decision_point = ? AND created_at >= ? AND created_at < ?`,
			userID, rule.AgentID, rule.DecisionPoint, beforeStart, windowStart,
		).Scan(&beforeAvg)

		var afterAvg AvgResult
		a.db.Raw(
			`SELECT AVG(confidence) as avg FROM agent_decision WHERE user_id = ? AND agent_id = ? AND decision_point = ? AND created_at >= ?`,
			userID, rule.AgentID, rule.DecisionPoint, windowStart,
		).Scan(&afterAvg)

		bAvg := 0.0
		if beforeAvg.Avg != nil {
			bAvg = *beforeAvg.Avg
		}
		aAvg := 0.0
		if afterAvg.Avg != nil {
			aAvg = *afterAvg.Avg
		}

		if bAvg > 0 && (bAvg-aAvg) >= regretThreshold {
			result = append(result, RegretChange{
				Change:    ch,
				Rule:      rule,
				BeforeAvg: bAvg,
				AfterAvg:  aAvg,
				Drop:      bAvg - aAvg,
			})
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

func logChange(db *gorm.DB, ch RuleMarkChange) error {
	return db.Create(&ch).Error
}

func strPtr(s string) *string { return &s }

func jsonStrPtr(v interface{}) *string {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}
