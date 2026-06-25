package agentos

import (
	"github.com/lingmirror/backend-go/internal/ai"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	Squads       []SquadHealth    `json:"squads"`
	Agents       []ai.AgentRosterSummary `json:"agents"`
	PendingByRisk map[string]int64 `json:"pending_by_risk"`
	PendingTotal int64            `json:"pending_total"`
	SlaBreached  int64            `json:"sla_breached"`
	WorkQueueLen int64            `json:"work_queue_len"`
}

// SquadHealth summarizes one squad's health.
type SquadHealth struct {
	Squad        string  `json:"squad"`
	AgentCount   int     `json:"agent_count"`
	TraceCount   int64   `json:"trace_count"`
	ActionCount  int64   `json:"action_count"`
	PendingCount int64   `json:"pending_count"`
	AvgConfidence float64 `json:"avg_confidence"`
	Health       string  `json:"health"` // ok | warn | critical
	Warnings     []string `json:"warnings"`
}

// Overview builds the cockpit overview.
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

	// Group by squad.
	bySquad := s.registry.BySquad()
	squads := make([]SquadHealth, 0, len(bySquad))
	for squadName, agents := range bySquad {
		sh := SquadHealth{Squad: squadName, AgentCount: len(agents), Warnings: []string{}}
		var agentIDs []string
		for _, a := range agents {
			agentIDs = append(agentIDs, a.ID)
		}
		if len(agentIDs) > 0 {
			_ = s.db.Table("ai_trace").Where("agent_id IN ?", agentIDs).Count(&sh.TraceCount).Error
			_ = s.db.Table("unified_action").Where("agent_id IN ?", agentIDs).Count(&sh.ActionCount).Error
			_ = s.db.Table("unified_action").Where("agent_id IN ? AND status IN ?", agentIDs, []string{"suggested", "pending"}).Count(&sh.PendingCount).Error
			var conf struct{ Avg float64 }
			_ = s.db.Table("ai_trace").Where("agent_id IN ? AND confidence IS NOT NULL", agentIDs).
				Select("COALESCE(AVG(confidence),0) AS avg").Scan(&conf).Error
			sh.AvgConfidence = conf.Avg
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

	// Pending by risk.
	var pendingLow, pendingMedium, pendingHigh, pendingCritical int64
	_ = s.db.Table("unified_action").Where("status IN ? AND risk_level = ?", []string{"suggested", "pending"}, "low").Count(&pendingLow).Error
	_ = s.db.Table("unified_action").Where("status IN ? AND risk_level = ?", []string{"suggested", "pending"}, "medium").Count(&pendingMedium).Error
	_ = s.db.Table("unified_action").Where("status IN ? AND risk_level = ?", []string{"suggested", "pending"}, "high").Count(&pendingHigh).Error
	_ = s.db.Table("unified_action").Where("status IN ? AND risk_level = ?", []string{"suggested", "pending"}, "critical").Count(&pendingCritical).Error
	overview.PendingByRisk["low"] = pendingLow
	overview.PendingByRisk["medium"] = pendingMedium
	overview.PendingByRisk["high"] = pendingHigh
	overview.PendingByRisk["critical"] = pendingCritical
	overview.PendingTotal = pendingLow + pendingMedium + pendingHigh + pendingCritical

	// Work queue = suggested actions older than 1h count as SLA breached.
	_ = s.db.Table("unified_action").Where("status IN ?", []string{"suggested", "pending"}).Count(&overview.WorkQueueLen).Error
	_ = s.db.Table("unified_action").Where("status IN ? AND proposed_at < NOW() - INTERVAL '1 hour'", []string{"suggested", "pending"}).Count(&overview.SlaBreached).Error

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
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	AgentID     string  `json:"agent_id"`
	SquadID     string  `json:"squad_id"`
	RiskLevel   string  `json:"risk_level"`
	Status      string  `json:"status"`
	Confidence  *float64 `json:"confidence,omitempty"`
	ProposedAt  string  `json:"proposed_at"`
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
	AgentID       string `json:"agent_id"`
	AutonomyLevel string `json:"autonomy_level"`
	RequiresApproval bool `json:"requires_approval"`
	MaxActionsPerHour int `json:"max_actions_per_hour"`
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
