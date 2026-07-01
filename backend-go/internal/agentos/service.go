package agentos

import (
	"fmt"
	"strconv"
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

// ---------------------------------------------------------------------------
// WorkItemDetail types and method
// ---------------------------------------------------------------------------

// WorkItemDetailResponse is the full context payload for a single work item.
type WorkItemDetailResponse struct {
	ID            int64    `json:"id"`
	Title         string   `json:"title"`
	AgentID       string   `json:"agent_id"`
	SquadID       string   `json:"squad_id"`
	RiskLevel     string   `json:"risk_level"`
	Status        string   `json:"status"`
	Confidence    *float64 `json:"confidence,omitempty"`
	ProposedAt    string   `json:"proposed_at"`
	DecisionPoint string   `json:"decision_point"`
	Reason        string   `json:"reason"`
	InputSummary  string   `json:"input_summary"`
	OutputSummary string   `json:"output_summary"`

	// Linked business entities
	EntityType   string `json:"entity_type"`
	EntityID     *int64 `json:"entity_id"`
	EntityStatus string `json:"entity_status"`

	// Linked approval
	Approval *LinkedApproval `json:"approval,omitempty"`

	// Trace
	TraceID *string `json:"trace_id"`

	// Pipeline chain links
	UpstreamItems   []LinkedItem `json:"upstream_items"`
	DownstreamItems []LinkedItem `json:"downstream_items"`

	// Recent audit trail
	AuditLogs []AuditEntry `json:"audit_logs"`
}

// LinkedApproval shows linked approval request info.
type LinkedApproval struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	RiskLevel string `json:"risk_level"`
}

// LinkedItem is a referenced entity in the pipeline chain.
type LinkedItem struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"` // "unified_action", "listing_task", "approval_request"
	Title  string `json:"title"`
	Status string `json:"status"`
}

// AuditEntry is a single audit log row in the detail view.
type AuditEntry struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	Content   string `json:"content"`
	Operator  string `json:"operator"`
	CreatedAt string `json:"created_at"`
}

// WorkItemDetail returns full context for a single work item.
func (s *Service) WorkItemDetail(id int64) (*WorkItemDetailResponse, error) {
	// 1. Query the unified_action row using raw SQL for full control.
	type rawAction struct {
		ID                 int64
		Title              string
		AgentID            string
		SquadID            string
		RiskLevel          string
		Status             string
		Confidence         *float64
		ProposedAt         time.Time
		Description        string
		TraceID            *string
		BusinessObjectType string
		BusinessObjectID   string
	}
	var ra rawAction
	err := s.db.Table("unified_action").
		Select("id, title, agent_id, COALESCE(squad_id,'') AS squad_id, risk_level, status, confidence, proposed_at, COALESCE(description,'') AS description, trace_id, COALESCE(business_object_type,'') AS business_object_type, COALESCE(business_object_id,'') AS business_object_id").
		Where("id = ?", id).
		Scan(&ra).Error
	if err != nil {
		return nil, fmt.Errorf("work item query error: %w", err)
	}
	if ra.ID == 0 {
		return nil, fmt.Errorf("work item %d not found", id)
	}

	result := &WorkItemDetailResponse{
		ID:       ra.ID,
		Title:    ra.Title,
		AgentID:  ra.AgentID,
		SquadID:  ra.SquadID,
		RiskLevel: ra.RiskLevel,
		Status:   ra.Status,
		Confidence: ra.Confidence,
		ProposedAt: ra.ProposedAt.Format("2006-01-02 15:04:05"),
		DecisionPoint: "",
		Reason:  ra.Description,
		InputSummary:  "",
		OutputSummary: "",
		EntityType:    ra.BusinessObjectType,
		AuditLogs:     []AuditEntry{},
		UpstreamItems:  []LinkedItem{},
		DownstreamItems: []LinkedItem{},
		TraceID: ra.TraceID,
	}

	// Parse business_object_id as int64 for entity_id.
	if ra.BusinessObjectID != "" {
		if eid, err := strconv.ParseInt(ra.BusinessObjectID, 10, 64); err == nil {
			result.EntityID = &eid
		}
	}

	// 2. If trace_id exists, get decision_point, input_context, final_output from ai_trace.
	if ra.TraceID != nil && *ra.TraceID != "" {
		type traceExtra struct {
			DecisionPoint string
			InputContext  string
			FinalOutput   string
		}
		var te traceExtra
		if err := s.db.Table("ai_trace").
			Select("COALESCE(decision_point,'') AS decision_point, COALESCE(CAST(input_context AS TEXT),'') AS input_context, COALESCE(CAST(final_output AS TEXT),'') AS final_output").
			Where("trace_id = ?", *ra.TraceID).
			Scan(&te).Error; err == nil {
			if te.DecisionPoint != "" {
				result.DecisionPoint = te.DecisionPoint
			}
			if len(te.InputContext) > 200 {
				result.InputSummary = te.InputContext[:200] + "..."
			} else {
				result.InputSummary = te.InputContext
			}
			if len(te.FinalOutput) > 200 {
				result.OutputSummary = te.FinalOutput[:200] + "..."
			} else {
				result.OutputSummary = te.FinalOutput
			}
		}

		// 2a. Find pipeline chain siblings sharing the same trace_id.
		type sibling struct {
			ID         int64
			Title      string
			Status     string
			ProposedAt time.Time
		}
		var siblings []sibling
		s.db.Table("unified_action").
			Select("id, title, status, proposed_at").
			Where("trace_id = ? AND id != ?", *ra.TraceID, id).
			Order("proposed_at ASC").
			Scan(&siblings)
		for _, sib := range siblings {
			item := LinkedItem{
				ID:     sib.ID,
				Type:   "unified_action",
				Title:  sib.Title,
				Status: sib.Status,
			}
			if sib.ProposedAt.Before(ra.ProposedAt) {
				result.UpstreamItems = append(result.UpstreamItems, item)
			} else {
				result.DownstreamItems = append(result.DownstreamItems, item)
			}
		}
	}

	// 3. Query linked entity status if entity_type and entity_id are available.
	entityID := result.EntityID
	if entityID != nil && result.EntityType != "" {
		switch result.EntityType {
		case "listing_task":
			var status string
			if err := s.db.Table("listing_task").
				Select("COALESCE(status,'')").
				Where("id = ?", *entityID).
				Scan(&status).Error; err == nil {
				result.EntityStatus = status
			}

			// 3a. For listing_task, also find linked approval request.
			type appRow struct {
				ID        int64
				Status    string
				RiskLevel string
			}
			var app appRow
			if err := s.db.Table("approval_request").
				Select("id, COALESCE(status,'') AS status, COALESCE(risk_level,'medium') AS risk_level").
				Where("entity_type = ? AND entity_id = ?", "listing_task", *entityID).
				Order("id DESC").Limit(1).
				Scan(&app).Error; err == nil && app.ID > 0 {
				result.Approval = &LinkedApproval{
					ID:        app.ID,
					Status:    app.Status,
					RiskLevel: app.RiskLevel,
				}
			}
		default:
			// Try generic table lookup for known entity types.
			var status string
			_ = s.db.Table(result.EntityType).
				Select("COALESCE(status,'')").
				Where("id = ?", *entityID).
				Scan(&status).Error
			if status != "" {
				result.EntityStatus = status
			}
		}
	}

	// 4. Query recent operation_log entries related to this action.
	type auditRow struct {
		ID        int64
		Action    string
		Content   string
		Operator  string
		CreatedAt time.Time
	}
	var auditRows []auditRow
	s.db.Table("operation_log").
		Select("id, COALESCE(action,'') AS action, COALESCE(content,'') AS content, COALESCE(operator,'') AS operator, created_at").
		Where("(entity_type = 'unified_action' AND entity_id = ?) OR resource_id = ?", id, fmt.Sprintf("%d", id)).
		Order("created_at DESC").Limit(20).
		Scan(&auditRows)
	for _, a := range auditRows {
		result.AuditLogs = append(result.AuditLogs, AuditEntry{
			ID:        a.ID,
			Action:    a.Action,
			Content:   a.Content,
			Operator:  a.Operator,
			CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// AgentTimeline types and method
// ---------------------------------------------------------------------------

// AgentTimelineEntry groups recent actions per agent.
type AgentTimelineEntry struct {
	AgentID       string         `json:"agent_id"`
	AgentName     string         `json:"agent_name"`
	RecentActions []ActionEvent  `json:"recent_actions"`
	StatusSummary map[string]int `json:"status_summary"` // "suggested": 3, "approved": 1, etc.
}

// ActionEvent is a single action row in the timeline.
type ActionEvent struct {
	ID         int64    `json:"id"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	RiskLevel  string   `json:"risk_level"`
	Confidence *float64 `json:"confidence,omitempty"`
	EntityType string   `json:"entity_type"`
	EntityID   *int64   `json:"entity_id"`
	CreatedAt  string   `json:"created_at"`
}

// AgentTimeline returns recent actions grouped by agent, limited to the last N actions.
func (s *Service) AgentTimeline(limit int) ([]AgentTimelineEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// 1. Query latest actions from unified_action, ordered by created_at DESC.
	type rawAction struct {
		ID                int64
		Title             string
		AgentID           string
		Status            string
		RiskLevel         string
		Confidence        *float64
		BusinessObjectType string
		BusinessObjectID  string
		CreatedAt         time.Time
	}
	var actions []rawAction
	if err := s.db.Table("unified_action").
		Select("id, title, agent_id, status, risk_level, confidence, COALESCE(business_object_type,'') AS business_object_type, COALESCE(business_object_id,'') AS business_object_id, created_at").
		Order("created_at DESC").
		Limit(limit * 2). // fetch extra to avoid losing agents due to limit grouping; ponytail: if single agent exceeds limit*2, other agents are still crowded out
		Scan(&actions).Error; err != nil {
		return nil, err
	}

	// 2. Group by agent_id, preserving input order.
	agentMap := make(map[string]*AgentTimelineEntry)
	var agentOrder []string
	for _, a := range actions {
		entry, ok := agentMap[a.AgentID]
		if !ok {
			entry = &AgentTimelineEntry{
				AgentID:       a.AgentID,
				AgentName:     a.AgentID, // default to ID, override below
				RecentActions: []ActionEvent{},
				StatusSummary: make(map[string]int),
			}
			agentMap[a.AgentID] = entry
			agentOrder = append(agentOrder, a.AgentID)
		}

		var entityID *int64
		if a.BusinessObjectID != "" {
			if eid, err := strconv.ParseInt(a.BusinessObjectID, 10, 64); err == nil {
				entityID = &eid
			}
		}

		entry.RecentActions = append(entry.RecentActions, ActionEvent{
			ID:         a.ID,
			Title:      a.Title,
			Status:     a.Status,
			RiskLevel:  a.RiskLevel,
			Confidence: a.Confidence,
			EntityType: a.BusinessObjectType,
			EntityID:   entityID,
			CreatedAt:  a.CreatedAt.Format("2006-01-02 15:04:05"),
		})
		entry.StatusSummary[a.Status]++
	}

	// 3. Map agent names from registry.
	result := make([]AgentTimelineEntry, 0, len(agentOrder))
	for _, agentID := range agentOrder {
		entry := agentMap[agentID]
		if spec, ok := s.registry.Get(agentID); ok && spec.Name != "" {
			entry.AgentName = spec.Name
		}
		result = append(result, *entry)
	}

	return result, nil
}
