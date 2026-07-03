package agentos

import (
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
)

// Service provides AgentOS cockpit aggregation.
type Service struct {
	db       *gorm.DB
	logger   *zap.Logger
	registry *ai.AgentRegistry
	tracker  *toolbridge.ExternalCallTracker
}

// NewService creates a new AgentOS service.
func NewService(db *gorm.DB, logger *zap.Logger, tracker *toolbridge.ExternalCallTracker) *Service {
	return &Service{
		db:       db,
		logger:   logger,
		registry: ai.DefaultRegistry(),
		tracker:  tracker,
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

// ---------------------------------------------------------------------------
// TrafficSummary types and method
// ---------------------------------------------------------------------------

// TrafficSummary is the funnel overview of all action statuses.
type TrafficSummary struct {
	StatusDistribution map[string]int64            `json:"status_distribution"`
	InterceptedTotal   int64                       `json:"intercepted_total"`
	Funnel             FunnelStats                 `json:"funnel"`
	ByRisk             map[string]map[string]int64 `json:"by_risk"`
}

// FunnelStats is the derived funnel from status distribution.
type FunnelStats struct {
	Produced        int64 `json:"produced"`
	Approved        int64 `json:"approved"`
	Executed        int64 `json:"executed"`
	BlockedByPolicy int64 `json:"blocked_by_policy"`
	RejectedByOwner int64 `json:"rejected_by_owner"`
}

// TrafficSummary returns the distribution of all action statuses.
func (s *Service) TrafficSummary() (*TrafficSummary, error) {
	summary := &TrafficSummary{
		StatusDistribution: map[string]int64{},
		ByRisk:             map[string]map[string]int64{},
	}

	// Status distribution: single query with GROUP BY
	type statusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []statusCount
	if err := s.db.Table("unified_action").
		Select("status, COUNT(*) AS count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, err
	}
	for _, sc := range statusCounts {
		summary.StatusDistribution[sc.Status] = sc.Count
	}

	// Intercepted total: blocked actions
	var ic struct{ Count int64 }
	s.db.Table("unified_action").Select("COUNT(*) AS count").Where("status = ?", "blocked").Scan(&ic)
	summary.InterceptedTotal = ic.Count

	// Funnel
	var funnel struct {
		Produced int64
		Approved int64
		Executed int64
		Rejected int64
		Blocked  int64
	}
	s.db.Raw(`
		SELECT
			COUNT(*) AS produced,
			COUNT(*) FILTER (WHERE status = 'approved') AS approved,
			COUNT(*) FILTER (WHERE status = 'executed') AS executed,
			COUNT(*) FILTER (WHERE status = 'rejected') AS rejected,
			COUNT(*) FILTER (WHERE status = 'blocked') AS blocked
		FROM unified_action`,
	).Scan(&funnel)
	summary.Funnel = FunnelStats{
		Produced:        funnel.Produced,
		Approved:        funnel.Approved,
		Executed:        funnel.Executed,
		BlockedByPolicy: funnel.Blocked,
		RejectedByOwner: funnel.Rejected,
	}

	// By risk x status cross-tab
	type riskStatusCount struct {
		RiskLevel string
		Status    string
		Count     int64
	}
	var cross []riskStatusCount
	s.db.Table("unified_action").
		Select("risk_level, status, COUNT(*) AS count").
		Group("risk_level, status").
		Scan(&cross)
	for _, c := range cross {
		if _, ok := summary.ByRisk[c.RiskLevel]; !ok {
			summary.ByRisk[c.RiskLevel] = map[string]int64{}
		}
		summary.ByRisk[c.RiskLevel][c.Status] = c.Count
	}

	return summary, nil
}

// ---------------------------------------------------------------------------
// InterceptedActions types and method
// ---------------------------------------------------------------------------

// InterceptedAction is a single blocked/rejected action for the dashboard.
type InterceptedAction struct {
	ID            int64  `json:"id"`
	ActionType    string `json:"action_type"`
	AgentID       string `json:"agent_id"`
	RiskLevel     string `json:"risk_level"`
	BlockReason   string `json:"block_reason"`
	BlockedAt     string `json:"blocked_at"`
	TargetSummary string `json:"target_summary"`
}

// InterceptedActions returns recently blocked/rejected actions.
func (s *Service) InterceptedActions(limit int) ([]InterceptedAction, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	type row struct {
		ID            int64
		ActionType    string
		AgentID       string
		RiskLevel     string
		BlockReason   string
		CreatedAt     time.Time
		TargetSummary string
	}
	var rows []row
	q := s.db.Table("unified_action").
		Select("id, action_type, agent_id, risk_level, COALESCE(block_reason,'') AS block_reason, created_at, COALESCE(description,'') AS target_summary").
		Where("status IN ?", []string{"blocked", "rejected"})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]InterceptedAction, 0, len(rows))
	for _, r := range rows {
		items = append(items, InterceptedAction{
			ID:            r.ID,
			ActionType:    r.ActionType,
			AgentID:       r.AgentID,
			RiskLevel:     r.RiskLevel,
			BlockReason:   r.BlockReason,
			BlockedAt:     r.CreatedAt.Format("2006-01-02 15:04:05"),
			TargetSummary: r.TargetSummary,
		})
	}
	return items, total, nil
}

// ---------------------------------------------------------------------------
// AuditReplay types and method
// ---------------------------------------------------------------------------

// AuditReplayEvent is one step in an action's full trace.
type AuditReplayEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	ActionID  *int64 `json:"action_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Detail    string `json:"detail,omitempty"`
	Timestamp string `json:"timestamp"`
}

// AuditReplayResponse is the full timeline for one correlation ID.
type AuditReplayResponse struct {
	CorrelationID string             `json:"correlation_id"`
	Events        []AuditReplayEvent `json:"events"`
}

// AuditReplay returns the full event timeline for a correlation ID.
func (s *Service) AuditReplay(correlationID string) (*AuditReplayResponse, error) {
	resp := &AuditReplayResponse{
		CorrelationID: correlationID,
		Events:        []AuditReplayEvent{},
	}

	// 1. Find all unified_actions with this correlation_id
	type actionRow struct {
		ID         int64
		ActionType string
		AgentID    string
		Status     string
		CreatedAt  time.Time
	}
	var actions []actionRow
	if err := s.db.Table("unified_action").
		Select("id, action_type, agent_id, status, created_at").
		Where("correlation_id = ?", correlationID).
		Order("created_at ASC").
		Scan(&actions).Error; err != nil {
		return nil, err
	}
	for _, a := range actions {
		aid := a.ID
		resp.Events = append(resp.Events, AuditReplayEvent{
			Type:      "action",
			AgentID:   a.AgentID,
			Subtype:   a.ActionType,
			ActionID:  &aid,
			Status:    a.Status,
			Timestamp: a.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 2. Find approval requests linked to these actions
	if len(actions) > 0 {
		var ids []int64
		for _, a := range actions {
			ids = append(ids, a.ID)
		}
		type appRow struct {
			ID        int64
			Status    string
			Reviewer  string
			UpdatedAt time.Time
		}
		var approvals []appRow
		if err := s.db.Table("approval_request").
			Select("id, status, COALESCE(reviewer,'') AS reviewer, updated_at").
			Where("entity_type = 'unified_action' AND entity_id IN ?", ids).
			Order("updated_at ASC").
			Scan(&approvals).Error; err != nil {
			s.logger.Warn("audit replay: approval_request query failed", zap.Error(err))
		}
		for _, ap := range approvals {
			resp.Events = append(resp.Events, AuditReplayEvent{
				Type:      "approval",
				Subtype:   "approval_request",
				Status:    ap.Status,
				Detail:    "reviewer: " + ap.Reviewer,
				Timestamp: ap.UpdatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// 3. Find operation_log entries
	type logRow struct {
		Action    string
		Content   string
		CreatedAt time.Time
	}
	var logs []logRow
	if err := s.db.Table("operation_log").
		Select("action, COALESCE(content,'') AS content, created_at").
		Where("correlation_id = ?", correlationID).
		Order("created_at ASC").
		Scan(&logs).Error; err != nil {
		s.logger.Warn("audit replay: operation_log query failed", zap.Error(err))
	}
	for _, l := range logs {
		resp.Events = append(resp.Events, AuditReplayEvent{
			Type:      "audit",
			Subtype:   l.Action,
			Detail:    l.Content,
			Timestamp: l.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return resp, nil
}

// ---------------------------------------------------------------------------
// AgentMetrics types and method
// ---------------------------------------------------------------------------

// AgentMetrics is the per-agent health and performance snapshot.
type AgentMetrics struct {
	AgentID             string  `json:"agent_id"`
	RunCount            int64   `json:"run_count"`
	SuccessCount        int64   `json:"success_count"`
	FailureCount        int64   `json:"failure_count"`
	BlockedCount        int64   `json:"blocked_count"`
	ApprovalRate        float64 `json:"approval_rate"`
	OwnerAcceptanceRate float64 `json:"owner_acceptance_rate"`
	AvgLatencyMs        float64 `json:"avg_latency_ms"`
	ExternalFailureRate float64 `json:"external_failure_rate"`
	Health              string  `json:"health"`
}

// AgentMetrics returns per-agent metrics aggregated from unified_action and ai_trace.
func (s *Service) AgentMetrics() ([]AgentMetrics, error) {
	type rawMetrics struct {
		AgentID   string
		RunCount  int64
		Succeeded int64
		Failed    int64
		Blocked   int64
	}

	var actions []rawMetrics
	s.db.Table("unified_action").
		Select(`
			agent_id,
			COUNT(*) AS run_count,
			COUNT(*) FILTER (WHERE status = 'completed') AS succeeded,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE status = 'blocked') AS blocked
		`).
		Group("agent_id").
		Scan(&actions)

	type latRow struct {
		AgentID string
		AvgLat  float64
	}
	var latencies []latRow
	s.db.Table("ai_trace").
		Select("agent_id, COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(completed_at, started_at) - started_at))), 0) AS avg_lat").
		Where("completed_at IS NOT NULL").
		Group("agent_id").
		Scan(&latencies)
	latMap := make(map[string]float64, len(latencies))
	for _, l := range latencies {
		latMap[l.AgentID] = l.AvgLat * 1000
	}

	type accRow struct {
		AgentID  string
		Approved int64
		Rejected int64
	}
	var accRates []accRow
	s.db.Table("unified_action").
		Select(`
			agent_id,
			COUNT(*) FILTER (WHERE status = 'approved') AS approved,
			COUNT(*) FILTER (WHERE status = 'rejected') AS rejected
		`).
		Group("agent_id").
		Scan(&accRates)
	accMap := make(map[string]float64, len(accRates))
	for _, a := range accRates {
		total := a.Approved + a.Rejected
		if total > 0 {
			accMap[a.AgentID] = float64(a.Approved) / float64(total)
		}
	}

	result := make([]AgentMetrics, 0, len(actions))
	for _, a := range actions {
		total := a.Succeeded + a.Failed + a.Blocked
		extFailRate := 0.0
		if a.Failed > 0 && total > 0 {
			extFailRate = float64(a.Failed) / float64(total)
		}
		lat := latMap[a.AgentID]

		health := "ok"
		if extFailRate > 0.2 || a.Failed > 10 {
			health = "warn"
		}
		if extFailRate > 0.5 || a.Failed > 50 {
			health = "critical"
		}

		result = append(result, AgentMetrics{
			AgentID:             a.AgentID,
			RunCount:            a.RunCount,
			SuccessCount:        a.Succeeded,
			FailureCount:        a.Failed,
			BlockedCount:        a.Blocked,
			ApprovalRate:        accMap[a.AgentID],
			OwnerAcceptanceRate: accMap[a.AgentID],
			AvgLatencyMs:        lat,
			ExternalFailureRate: extFailRate,
			Health:              health,
		})
	}
	return result, nil
}

// ExternalHealth returns a generic external service health check for the cockpit.
func (s *Service) ExternalHealth() []toolbridge.PlatformStatsSnapshot {
	if s.tracker != nil {
		return s.tracker.Stats()
	}
	return []toolbridge.PlatformStatsSnapshot{}
}

// FailedRun represents a failed agent run with error context.
type FailedRun struct {
	ID            int64  `json:"id"`
	TraceID       string `json:"trace_id"`
	AgentID       string `json:"agent_id"`
	DecisionPoint string `json:"decision_point"`
	Status        string `json:"status"`
	ErrorMessage  string `json:"error_message"`
	StartedAt     string `json:"started_at"`
	CompletedAt   string `json:"completed_at"`
}

// FailedRuns returns recent failed agent traces.
func (s *Service) FailedRuns(limit int) ([]FailedRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	type row struct {
		ID            int64
		TraceID       string
		AgentID       string
		DecisionPoint string
		Status        string
		FinalOutput   *string // jsonb, might contain error message
		StartedAt     time.Time
		CompletedAt   *time.Time
	}
	var rows []row
	if err := s.db.Table("ai_trace").
		Select("id, trace_id, agent_id, decision_point, status, CAST(final_output AS TEXT) AS final_output, started_at, completed_at").
		Where("status = ?", "failed").
		Order("started_at DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]FailedRun, 0, len(rows))
	for _, r := range rows {
		// Try to extract error message from final_output JSON.
		errMsg := ""
		if r.FinalOutput != nil && *r.FinalOutput != "" {
			// The final_output might be JSON like {"error": "..."}
			// Fall back: just show the raw output as the error message.
			if len(*r.FinalOutput) > 500 {
				errMsg = (*r.FinalOutput)[:500]
			} else {
				errMsg = *r.FinalOutput
			}
		}
		completedAt := ""
		if r.CompletedAt != nil {
			completedAt = r.CompletedAt.Format("2006-01-02 15:04:05")
		}
		result = append(result, FailedRun{
			ID:            r.ID,
			TraceID:       r.TraceID,
			AgentID:       r.AgentID,
			DecisionPoint: r.DecisionPoint,
			Status:        r.Status,
			ErrorMessage:  errMsg,
			StartedAt:     r.StartedAt.Format("2006-01-02 15:04:05"),
			CompletedAt:   completedAt,
		})
	}
	return result, nil
}
