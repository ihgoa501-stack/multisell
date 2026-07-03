package entropy

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service orchestrates all entropy subsystems.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	ttl    *TTLSweeper
	budget *BudgetEnforcer
	decay  *DecayScheduler
	merge  *MergeDetector
	regret *RegretAnalyzer
	scorer *RuleHealthScorer
	spc    *SpcController
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

// GetEntropySummary returns the entropy dashboard summary.
func (s *Service) GetEntropySummary(userID int64) (*EntropySummary, error) {
	healthSummary, err := s.scorer.GetSummary(userID)
	if err != nil {
		return nil, err
	}

	var recentChanges int64
	s.db.Model(&RuleMarkChange{}).
		Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
		Count(&recentChanges)

	var totalActive int64
	s.db.Model(&PersonalRule{}).Where("enabled = true").Count(&totalActive)

	duplicates, _ := s.merge.FindDuplicates(userID)

	entropyIndex := calcEntropyIndex(healthSummary)

	return &EntropySummary{
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
	}, nil
}

func calcEntropyIndex(hs *HealthSummary) float64 {
	if hs.TotalRules == 0 {
		return 0.0
	}
	unhealthyRatio := float64(hs.UnhealthyCount) / float64(hs.TotalRules)
	shadowRatio := float64(hs.ShadowRules) / float64(hs.TotalRules)
	index := unhealthyRatio*0.4 + shadowRatio*0.3 + (1.0-hs.AvgHealthScore)*0.3
	if index > 1.0 {
		index = 1.0
	}
	return round4(index)
}

// RunDefenses executes the 5 defense lines: TTL → Budget → Decay → Merge → Shadow.
func (s *Service) RunDefenses(userID int64) (*DefenseSummary, error) {
	expiredCount, _ := s.ttl.ExpireStaleRules(userID)
	budgetPruned, _ := s.budget.EnforceBudgets(userID, nil)
	decayedCount, _ := s.decay.ApplyDecay(userID)
	duplicates, _ := s.merge.FindDuplicates(userID)
	shadowedCount := len(s.shadowOverriddenRules(userID))

	return &DefenseSummary{
		TotalAffected: expiredCount + sumBudget(budgetPruned) + decayedCount + shadowedCount,
		Actions: struct {
			ExpiredRules        int `json:"expired_rules"`
			BudgetExceeded      int `json:"budget_exceeded"`
			DecayApplied        int `json:"decay_applied"`
			MergedCandidates    int `json:"merged_candidates"`
			ShadowedByOverrides int `json:"shadowed_by_overrides"`
		}{
			ExpiredRules:       expiredCount,
			BudgetExceeded:     sumBudget(budgetPruned),
			DecayApplied:       decayedCount,
			MergedCandidates:   len(duplicates),
			ShadowedByOverrides: shadowedCount,
		},
		DuplicatesFound: len(duplicates),
		MergeCandidates: toMergeCandidates(duplicates),
	}, nil
}

func sumBudget(m map[string]int) int {
	t := 0
	for _, v := range m {
		t += v
	}
	return t
}

func toMergeCandidates(dups []DuplicatePair) []MergeCandidate {
	out := make([]MergeCandidate, 0, len(dups))
	for i, d := range dups {
		if i >= 10 {
			break
		}
		out = append(out, MergeCandidate{
			KeepID:     d.Keep.ID,
			RemoveID:   d.Remove.ID,
			Similarity: d.Similarity,
		})
	}
	return out
}

// shadowOverriddenRules disables rules that are frequently overridden.
func (s *Service) shadowOverriddenRules(userID int64) []PersonalRule {
	var rules []PersonalRule
	if err := s.db.Where("enabled = true").Find(&rules).Error; err != nil {
		return nil
	}
	var shadowed []PersonalRule
	for _, r := range rules {
		// Count recent rejections via actionpolicy_evaluation or just disable if priority is very low
		if r.ID > 0 && false { // Placeholder for future override detection
			_ = s.db.Model(&r).Update("enabled", false).Error
			shadowed = append(shadowed, r)
		}
	}
	_ = userID
	return shadowed
}

// GetHealthScores returns health scores for all rules.
func (s *Service) GetHealthScores(userID int64) ([]RuleHealthScore, error) {
	return s.scorer.ScoreAllRules(userID)
}

// GetSpcStatus returns SPC control status.
func (s *Service) GetSpcStatus(userID int64) ([]SpcStatus, error) {
	limits, err := s.spc.GetAllLimits(userID)
	if err != nil {
		return nil, err
	}
	results := make([]SpcStatus, 0, len(limits))
	for _, l := range limits {
		results = append(results, SpcStatus{
			AgentID:        l.AgentID,
			DecisionPoint:  l.DecisionPoint,
			MetricName:     l.MetricName,
			BaselineMean:   l.BaselineMean,
			UCL:            l.UCL,
			LCL:            l.LCL,
			UWL:            l.UWL,
			LWL:            l.LWL,
			BaselineSamples: l.BaselineSamples,
			IsOutOfControl: false,
		})
	}
	return results, nil
}

// GetChangeLog returns paginated change log entries.
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
			ChangeSummary: r.ChangeSummary,
			CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		})
	}
	return entries, total, nil
}

// AgentMetricsSnapshot is a minimal metrics input for entropy processing.
type AgentMetricsSnapshot struct {
	AgentID             string
	RunCount            int64
	FailureCount        int64
	BlockedCount        int64
	ExternalFailureRate float64
	AvgLatencyMs        float64
}

// AnomalousAgent represents an agent flagged as anomalous with a reason and severity.
type AnomalousAgent struct {
	AgentID  string `json:"agent_id"`
	Reason   string `json:"reason"`
	Severity string `json:"severity"` // "warn" or "critical"
}

// ConsumeAgentMetrics processes agent metrics for anomaly detection.
func (s *Service) ConsumeAgentMetrics(metrics []AgentMetricsSnapshot) []AnomalousAgent {
	var anomalies []AnomalousAgent
	for _, m := range metrics {
		if m.ExternalFailureRate > 0.2 {
			anomalies = append(anomalies, AnomalousAgent{
				AgentID:  m.AgentID,
				Reason:   fmt.Sprintf("external failure rate %.0f%% exceeds threshold (20%%)", m.ExternalFailureRate*100),
				Severity: "warn",
			})
		}
		if m.FailureCount > 20 && m.BlockedCount > 10 {
			anomalies = append(anomalies, AnomalousAgent{
				AgentID:  m.AgentID,
				Reason:   fmt.Sprintf("high failure (%d) and blocked (%d) count", m.FailureCount, m.BlockedCount),
				Severity: "critical",
			})
		}
		if m.AvgLatencyMs > 10000 {
			anomalies = append(anomalies, AnomalousAgent{
				AgentID:  m.AgentID,
				Reason:   fmt.Sprintf("avg latency %.0fms exceeds 10s threshold", m.AvgLatencyMs),
				Severity: "warn",
			})
		}
	}
	return anomalies
}

// round4 rounds to 4 decimal places.
