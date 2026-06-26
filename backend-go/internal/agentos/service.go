package agentos

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lingmirror/backend-go/internal/ai"
)

// Service provides AgentOS cockpit aggregation.
type Service struct {
	db       *gorm.DB
	logger   *zap.Logger
	registry *ai.AgentRegistry
}

// NewService creates a new AgentOS service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{
		db:       db,
		logger:   logger,
		registry: ai.DefaultRegistry(),
	}
}

// CockpitOverview is the top-level AgentOS cockpit payload.
type CockpitOverview struct {
	Squads        []SquadHealth          `json:"squads"`
	Agents        []ai.AgentRosterSummary `json:"agents"`
	PendingByRisk map[string]int64       `json:"pending_by_risk"`
	PendingTotal  int64                  `json:"pending_total"`
	SlaBreached   int64                  `json:"sla_breached"`
	WorkQueueLen  int64                  `json:"work_queue_len"`
}

// SquadHealth summarizes one squad's health.
type SquadHealth struct {
	Squad         string   `json:"squad"`
	AgentCount    int      `json:"agent_count"`
	TraceCount    int64    `json:"trace_count"`
	ActionCount   int64    `json:"action_count"`
	PendingCount  int64    `json:"pending_count"`
	AvgConfidence float64  `json:"avg_confidence"`
	Health        string   `json:"health"` // ok | warn | critical
	Warnings      []string `json:"warnings"`
}

// Overview builds the cockpit overview with best-effort querying.
// DB errors are logged and result in zero defaults — partial data is
// returned rather than failing the entire request.
func (s *Service) Overview() (*CockpitOverview, error) {
	overview := &CockpitOverview{
		PendingByRisk: map[string]int64{"low": 0, "medium": 0, "high": 0, "critical": 0},
	}

	// Build per-agent summaries.
	aiSvc := ai.NewService(s.db, s.logger)
	roster, err := aiSvc.Roster()
	if err != nil {
		return nil, err
	}
	overview.Agents = roster

	// Group by squad — use a single batched query.
	bySquad := s.registry.BySquad()
	squads := make([]SquadHealth, 0, len(bySquad))
	for squadName, agents := range bySquad {
		sh := SquadHealth{Squad: squadName, AgentCount: len(agents), Warnings: []string{}}
		var agentIDs []string
		for _, a := range agents {
			agentIDs = append(agentIDs, a.ID)
		}
		if len(agentIDs) > 0 {
			// Single batched query for all squad metrics.
			type squadMetrics struct {
				TraceCount    int64
				ActionCount   int64
				PendingCount  int64
				AvgConfidence float64
			}
			var m squadMetrics
			err := s.db.Raw(`
				SELECT
					COALESCE(t.trace_count, 0) AS trace_count,
					COALESCE(a.action_count, 0) AS action_count,
					COALESCE(a.pending_count, 0) AS pending_count,
					COALESCE(t.avg_conf, 0) AS avg_confidence
				FROM (
					SELECT
						COUNT(*) AS trace_count,
						COALESCE(AVG(confidence), 0) AS avg_conf
					FROM ai_trace
					WHERE agent_id IN ? AND confidence IS NOT NULL
				) t,
				(
					SELECT
						COUNT(*) AS action_count,
						COUNT(*) FILTER (WHERE status IN ('suggested','pending')) AS pending_count
					FROM unified_action
					WHERE agent_id IN ?
				) a`,
				agentIDs, agentIDs,
			).Scan(&m).Error
			if err != nil {
				s.logger.Warn("overview squad query failed",
					zap.String("squad", squadName),
					zap.Error(err))
				// Continue with zero defaults.
			} else {
				sh.TraceCount = m.TraceCount
				sh.ActionCount = m.ActionCount
				sh.PendingCount = m.PendingCount
				sh.AvgConfidence = m.AvgConfidence
			}
		}
		sh.Health = classifyHealth(sh.PendingCount, sh.AvgConfidence)
		if sh.Health == "warn" {
			sh.Warnings = append(sh.Warnings, "待审批积压")
		}
		if sh.Health == "critical" {
			sh.Warnings = append(sh.Warnings, "平均置信度过低")
		}
		squads = append(squads, sh)
	}
	overview.Squads = squads

	// Pending by risk — single query with GROUP BY.
	type riskCount struct {
		RiskLevel string
		Count     int64
	}
	var riskCounts []riskCount
	if err := s.db.Table("unified_action").
		Select("risk_level, COUNT(*) AS count").
		Where("status IN ?", []string{"suggested", "pending"}).
		Group("risk_level").
		Scan(&riskCounts).Error; err != nil {
		s.logger.Warn("overview risk count query failed", zap.Error(err))
	} else {
		for _, rc := range riskCounts {
			if _, ok := overview.PendingByRisk[rc.RiskLevel]; ok {
				overview.PendingByRisk[rc.RiskLevel] = rc.Count
			}
			overview.PendingTotal += rc.Count
		}
	}

	// Work queue and SLA — single query with conditional aggregation.
	type queueMetrics struct {
		QueueLen    int64
		SlaBreached int64
	}
	var qm queueMetrics
	if err := s.db.Raw(`
		SELECT
			COUNT(*) AS queue_len,
			COUNT(*) FILTER (WHERE proposed_at < NOW() - INTERVAL '1 hour') AS sla_breached
		FROM unified_action
		WHERE status IN ('suggested', 'pending')`,
	).Scan(&qm).Error; err != nil {
		s.logger.Warn("overview queue query failed", zap.Error(err))
	} else {
		overview.WorkQueueLen = qm.QueueLen
		overview.SlaBreached = qm.SlaBreached
	}

	return overview, nil
}

// WorkItems returns the pending action work queue, oldest first.
type WorkItemsFilter struct {
	Status    string
	RiskLevel string
	AgentID   string
	SquadID   string
}

// WorkItem is a lightweight action view for the queue.
type WorkItem struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	AgentID     string   `json:"agent_id"`
	SquadID     string   `json:"squad_id"`
	RiskLevel   string   `json:"risk_level"`
	Status      string   `json:"status"`
	Confidence  *float64 `json:"confidence,omitempty"`
	ProposedAt  string   `json:"proposed_at"`
}

// WorkItems lists the work queue.
func (s *Service) WorkItems(limit int, f *WorkItemsFilter) ([]WorkItem, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.Table("unified_action")
	if f != nil {
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		} else {
			q = q.Where("status IN ?", []string{"suggested", "pending"})
		}
		if f.RiskLevel != "" {
			q = q.Where("risk_level = ?", f.RiskLevel)
		}
		if f.AgentID != "" {
			q = q.Where("agent_id = ?", f.AgentID)
		}
		if f.SquadID != "" {
			q = q.Where("squad_id = ?", f.SquadID)
		}
	} else {
		q = q.Where("status IN ?", []string{"suggested", "pending"})
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []WorkItem
	if err := q.Order("proposed_at ASC").Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// AutonomyConfig holds trust/autonomy controls.
type AutonomyConfig struct {
	AgentID            string `json:"agent_id"`
	AutonomyLevel      string `json:"autonomy_level"`
	RequiresApproval   bool   `json:"requires_approval"`
	MaxActionsPerHour  int    `json:"max_actions_per_hour"`
}

// Autonomy returns the current autonomy config per agent.
func (s *Service) Autonomy() []AutonomyConfig {
	out := make([]AutonomyConfig, 0, len(s.registry.Agents))
	for _, a := range s.registry.Agents {
		out = append(out, AutonomyConfig{
			AgentID:           a.ID,
			AutonomyLevel:     a.Autonomy,
			RequiresApproval:  a.Autonomy == "supervised" || a.Autonomy == "guided",
			MaxActionsPerHour: 20,
		})
	}
	return out
}

func classifyHealth(pending int64, avgConf float64) string {
	if avgConf > 0 && avgConf < 0.5 {
		return "critical"
	}
	if pending > 50 {
		return "warn"
	}
	return "ok"
}

// SLAEscalation escalates actions that have been suggested for over 1 hour
// without human review. It updates their status to "escalated" and logs each
// escalation for observability.
func (s *Service) SLAEscalation() error {
	cutoff := time.Now().Add(-1 * time.Hour)
	var overdue []struct {
		ID     int64
		Title  string
		UserID *int64
	}
	if err := s.db.Table("unified_action").
		Select("id, title, user_id").
		Where("status = ? AND created_at < ?", ai.ActionStatusSuggested, cutoff).
		Find(&overdue).Error; err != nil {
		return err
	}

	for _, o := range overdue {
		s.logger.Warn("sla escalation",
			zap.Int64("action_id", o.ID),
			zap.String("title", o.Title),
		)
	}

	if len(overdue) > 0 {
		var ids []int64
		for _, o := range overdue {
			ids = append(ids, o.ID)
		}
		return s.db.Table("unified_action").
			Where("id IN ?", ids).
			Update("status", ai.ActionStatusEscalated).Error
	}
	return nil
}
