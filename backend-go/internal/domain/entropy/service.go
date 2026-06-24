package entropy

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service orchestrates all entropy subsystems.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger

	ttl     *TTLSweeper
	budget  *BudgetEnforcer
	decay   *DecayScheduler
	merge   *MergeDetector
	regret  *RegretAnalyzer
	scorer  *RuleHealthScorer
	spc     *SpcController
}

// NewService creates the entropy service with all subsystems.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
		ttl:    NewTTLSweeper(db, logger),
		budget: NewBudgetEnforcer(db, logger),
		decay:  NewDecayScheduler(db, logger),
		merge:  NewMergeDetector(db, logger),
		regret: NewRegretAnalyzer(db, logger),
		scorer: NewRuleHealthScorer(db),
		spc:    NewSpcController(db),
	}
}

// GetEntropySummary 返回熵仪表盘摘要。
func (s *Service) GetEntropySummary(userID int64) (*EntropySummary, error) {
	healthSummary, err := s.scorer.GetSummary(userID)
	if err != nil {
		return nil, err
	}

	// 最近24小时变更数
	recent24h := time.Now().Add(-24 * time.Hour)
	var recentChanges int64
	s.db.Model(&RuleMarkChange{}).
		Where("created_at >= ?", recent24h).
		Count(&recentChanges)

	// 活跃规则总数
	var totalActive int64
	s.db.Model(&PersonalRule{}).
		Where("user_id = ? AND status = ?", userID, "active").
		Count(&totalActive)

	// 待合并数
	duplicates, err := s.merge.FindDuplicates(userID)
	if err != nil {
		s.logger.Warn("entropy: find duplicates failed", zap.Error(err))
	}

	entropyIndex := calcEntropyIndex(healthSummary)

	summary := &EntropySummary{
		TotalRules:         int(totalActive),
		ActiveRules:        healthSummary.ActiveRules,
		ShadowRules:        healthSummary.ShadowRules,
		AvgHealthScore:     healthSummary.AvgHealthScore,
		UnhealthyRuleCount: healthSummary.UnhealthyCount,
		WarningRuleCount:   healthSummary.WarningCount,
		PendingMergeCount:  len(duplicates),
		RecentChangesCount: int(recentChanges),
		ConflictsCount:     len(duplicates),
		SystemEntropyIndex: entropyIndex,
	}

	return summary, nil
}

func calcEntropyIndex(hs *HealthSummary) float64 {
	total := hs.TotalRules
	if total == 0 {
		return 0.0
	}
	unhealthyRatio := float64(hs.UnhealthyCount) / float64(total)
	shadowRatio := float64(hs.ShadowRules) / float64(total)
	avgScore := hs.AvgHealthScore

	index := unhealthyRatio*0.4 + shadowRatio*0.3 + (1.0-avgScore)*0.3
	if index > 1.0 {
		index = 1.0
	}
	return round4(index)
}

// RunDefenses 执行5道防线：TTL → Budget → Decay → Merge(仅检测) → 降级。
func (s *Service) RunDefenses(userID int64) (*DefenseSummary, error) {
	expired, err := s.ttl.ExpireStaleRules(userID)
	if err != nil {
		s.logger.Warn("entropy: ttl failed", zap.Error(err))
	}

	budgetExceeded, err := s.budget.EnforceBudgets(userID, nil)
	if err != nil {
		s.logger.Warn("entropy: budget failed", zap.Error(err))
	}

	decayed, err := s.decay.ApplyDecay(userID)
	if err != nil {
		s.logger.Warn("entropy: decay failed", zap.Error(err))
	}

	duplicates, err := s.merge.FindDuplicates(userID)
	if err != nil {
		s.logger.Warn("entropy: merge detect failed", zap.Error(err))
	}

	shadowed, err := s.shadowOverriddenRules(userID)
	if err != nil {
		s.logger.Warn("entropy: shadow overridden failed", zap.Error(err))
	}

	var markChanges []ChangeLogEntry
	for _, r := range expired {
		markChanges = append(markChanges, ChangeLogEntry{
			TargetType:    "personal_rule",
			TargetID:      r.ID,
			ChangeSummary: "防守动作: " + r.RuleName + " → retired",
		})
	}
	for _, r := range budgetExceeded {
		markChanges = append(markChanges, ChangeLogEntry{
			TargetType:    "personal_rule",
			TargetID:      r.ID,
			ChangeSummary: "防守动作: " + r.RuleName + " → shadow",
		})
	}
	// Limit mark changes to avoid huge payloads
	if len(markChanges) > 50 {
		markChanges = markChanges[:50]
	}

	var mc []MergeCandidate
	for _, dup := range duplicates {
		if len(mc) >= 10 {
			break
		}
		mc = append(mc, MergeCandidate{
			KeepID:     dup.Keep.ID,
			KeepName:   dup.Keep.RuleName,
			RemoveID:   dup.Remove.ID,
			RemoveName: dup.Remove.RuleName,
			Similarity: dup.Similarity,
		})
	}

	total := len(expired) + len(budgetExceeded) + len(decayed) + len(shadowed)

	summary := &DefenseSummary{}
	summary.Actions.ExpiredRules = len(expired)
	summary.Actions.BudgetExceeded = len(budgetExceeded)
	summary.Actions.DecayApplied = len(decayed)
	summary.Actions.MergedCandidates = len(duplicates)
	summary.Actions.ShadowedByOverrides = len(shadowed)
	summary.TotalAffected = total
	summary.MarkChanges = markChanges
	summary.DuplicatesFound = len(duplicates)
	summary.MergeCandidates = mc

	return summary, nil
}

// shadowOverriddenRules 将用户频繁覆写的规则降级为 shadow。
func (s *Service) shadowOverriddenRules(userID int64, overrides ...struct {
	maxOverrideRate float64
	minApplied      int
}) ([]PersonalRule, error) {
	maxRate := 0.5
	minApp := 5
	if len(overrides) > 0 {
		if overrides[0].maxOverrideRate > 0 {
			maxRate = overrides[0].maxOverrideRate
		}
		if overrides[0].minApplied > 0 {
			minApp = overrides[0].minApplied
		}
	}

	var rules []PersonalRule
	if err := s.db.Where("user_id = ? AND status = ?", userID, "active").Find(&rules).Error; err != nil {
		return nil, err
	}

	var shadowed []PersonalRule
	for _, r := range rules {
		applied := r.TimesApplied
		overridden := r.TimesOverridden
		if applied >= minApp && overridden > 0 {
			rate := float64(overridden) / float64(applied)
			if rate > maxRate {
				oldVal := `"active"`
				newVal := `"shadow"`
				if err := s.db.Model(&r).Update("status", "shadow").Error; err != nil {
					s.logger.Warn("shadow: update failed", zap.Int64("rule_id", r.ID), zap.Error(err))
					continue
				}
				_ = logChange(s.db, RuleMarkChange{
					TargetType:    "personal_rule",
					TargetID:      r.ID,
					FieldPath:     "$.status",
					OldValue:      &oldVal,
					NewValue:      newVal,
					SourceType:    "gds",
					SourceID:      strPtr("override_shadow"),
					ChangeSummary: "覆盖率过高自动降级为shadow",
				})
				shadowed = append(shadowed, r)
			}
		}
	}

	return shadowed, nil
}

// GetHealthScores 返回所有规则的健康评分。
func (s *Service) GetHealthScores(userID int64) ([]RuleHealthScore, error) {
	return s.scorer.ScoreAllRules(userID)
}

// GetSpcStatus 返回所有 SPC 控制状态。
func (s *Service) GetSpcStatus(userID int64) ([]SpcStatus, error) {
	limits, err := s.spc.GetAllLimits(userID)
	if err != nil {
		return nil, err
	}

	results := make([]SpcStatus, 0, len(limits))
	for _, l := range limits {
		nextRecalc := l.NextRecalcAt
		results = append(results, SpcStatus{
			AgentID:             l.AgentID,
			DecisionPoint:       l.DecisionPoint,
			MetricName:          l.MetricName,
			BaselineMean:        l.BaselineMean,
			UCL:                 l.UCL,
			LCL:                 l.LCL,
			UWL:                 l.UWL,
			LWL:                 l.LWL,
			BaselineSamples:     l.BaselineSamples,
			ConsecutiveSameSide: l.ConsecutiveSameSide,
			IsOutOfControl:      false,
			IsWarning:           false,
			LastBreachAt:        l.LastBreachAt,
			NextRecalcAt:        &nextRecalc,
		})
	}
	return results, nil
}

// GetChangeLog 查询变更日志（分页）。
func (s *Service) GetChangeLog(userID int64, sourceType string, page, pageSize int) ([]ChangeLogEntry, int64, error) {
	q := s.db.Model(&RuleMarkChange{})
	if sourceType != "" {
		q = q.Where("source_type = ?", sourceType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var records []RuleMarkChange
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	entries := make([]ChangeLogEntry, 0, len(records))
	for _, r := range records {
		entries = append(entries, ChangeLogEntry{
			ID:            r.ID,
			TargetType:    r.TargetType,
			TargetID:      r.TargetID,
			FieldPath:     r.FieldPath,
			OldValue:      r.OldValue,
			NewValue:      r.NewValue,
			SourceType:    r.SourceType,
			SourceID:      r.SourceID,
			ChangeSummary: r.ChangeSummary,
			CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		})
	}

	return entries, total, nil
}
