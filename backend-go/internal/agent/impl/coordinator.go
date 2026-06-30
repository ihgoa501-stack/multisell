// Package impl provides concrete agent implementations.
//
// CoordinatorAgent implements G0 Coordinator — the supervisor agent that
// monitors system health, escalates anomalies, coordinates cross-squad
// collaboration, and audits other agents' decisions.
package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// agentTrust holds trust score data for a single agent.
type agentTrust struct {
	AgentID    string
	TrustScore float64
	Autonomy   string
	TotalAct   int
	RejectedCt int
}

// CoordinatorAgent implements G0 Coordinator / Supervisor logic.
type CoordinatorAgent struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewCoordinatorAgent creates a new G0 CoordinatorAgent.
func NewCoordinatorAgent(db *gorm.DB, logger *zap.Logger) *CoordinatorAgent {
	return &CoordinatorAgent{db: db, logger: logger}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
func (a *CoordinatorAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "system_health":
		return a.systemHealth(ctx, params)
	case "anomaly_escalation":
		return a.anomalyEscalation(ctx, params)
	case "cross_squad_coordinate":
		return a.crossSquadCoordinate(ctx, params)
	case "agent_audit":
		return a.agentAudit(ctx, params)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "high", nil
	}
}

// ----- system_health -----

// systemHealth assesses the overall health of the AgentOS by aggregating
// trust scores, pending actions, recent traces, and SLA compliance.
func (a *CoordinatorAgent) systemHealth(callerCtx context.Context, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	health := "healthy"
	var warnings []string
	var criticals []string

	ctxGo, cancel := context.WithTimeout(callerCtx, 30*time.Second)
	defer cancel()

	// Aggregate agent stats via tool registry or DB fallback.
	var stalePending int64
	var recentFails int64
	var criticalPending int64

	// 1. Count stale pending actions (SLA breach > 1h).
	if ok, _ := a.invokeToolInt64(ctxGo, "coordinator.action.stale_pending_count",
		map[string]interface{}{"since_hours": 1}, &stalePending); !ok && a.db != nil {
		a.db.Raw(
			`SELECT COUNT(*) FROM unified_action
			 WHERE status IN ('suggested','pending')
			 AND proposed_at < ?`,
			time.Now().Add(-1*time.Hour),
		).Scan(&stalePending)
	}
	if stalePending > 0 {
		warnings = append(warnings, fmt.Sprintf("%d pending actions exceed 1h SLA", stalePending))
	}

	// 2. Count recent failures in the last 24h.
	if ok, _ := a.invokeToolInt64(ctxGo, "coordinator.trace.failure_count_24h",
		map[string]interface{}{"since_hours": 24}, &recentFails); !ok && a.db != nil {
		a.db.Raw(
			`SELECT COUNT(*) FROM ai_trace
			 WHERE status = 'failed'
			 AND started_at > ?`,
			time.Now().Add(-24*time.Hour),
		).Scan(&recentFails)
	}
	if recentFails > 5 {
		warnings = append(warnings, fmt.Sprintf("%d trace failures in 24h", recentFails))
	}
	if recentFails > 20 {
		criticals = append(criticals, fmt.Sprintf("%d trace failures in 24h — critical", recentFails))
	}

	// 3. Count critical-risk pending actions.
	if ok, _ := a.invokeToolInt64(ctxGo, "coordinator.action.critical_pending_count",
		map[string]interface{}{"risk_level": "critical"}, &criticalPending); !ok && a.db != nil {
		a.db.Raw(
			`SELECT COUNT(*) FROM unified_action
			 WHERE status IN ('suggested','pending')
			 AND risk_level = 'critical'`,
		).Scan(&criticalPending)
	}
	if criticalPending > 0 {
		criticals = append(criticals, fmt.Sprintf("%d critical-risk actions pending", criticalPending))
	}

	if len(criticals) > 0 {
		health = "critical"
		confidence = 0.95
		riskLevel = "high"
	} else if len(warnings) > 0 {
		health = "warning"
		confidence = 0.88
		riskLevel = "medium"
	} else {
		confidence = 0.85
		riskLevel = "low"
	}

	output = map[string]interface{}{
		"status":         "completed",
		"decision_point": "system_health",
		"health":         health,
		"warnings":       warnings,
		"criticals":      criticals,
		"anomaly_count":  len(criticals) + len(warnings),
		"confidence":     confidence,
	}
	return output, confidence, riskLevel, nil
}

// ----- anomaly_escalation -----

// anomalyEscalation receives an anomaly event and determines the appropriate
// escalation action. Expected context: anomaly_type, severity, details.
func (a *CoordinatorAgent) anomalyEscalation(ctx context.Context, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	anomalyType := stringFromCtx(params, "anomaly_type", "unknown")
	severity := stringFromCtx(params, "severity", "low")
	details := stringFromCtx(params, "details", "")

	var suggestedActions []string
	var escalateTo string

	switch severity {
	case "critical":
		escalateTo = "admin"
		suggestedActions = []string{
			"立即通知管理员",
			"暂停受影响Agent的自动执行",
			"启动回滚预案",
		}
		confidence = 0.95
		riskLevel = "critical"
	case "high":
		escalateTo = "manager"
		suggestedActions = []string{
			"通知相关业务负责人",
			"限制高风险操作",
			"加强监控频率",
		}
		confidence = 0.90
		riskLevel = "high"
	default:
		escalateTo = "auto"
		suggestedActions = []string{
			"记录异常日志",
			"继续监控",
		}
		confidence = 0.82
		riskLevel = "medium"
	}

	output = map[string]interface{}{
		"status":            "completed",
		"decision_point":    "anomaly_escalation",
		"anomaly_type":      anomalyType,
		"severity":          severity,
		"details":           details,
		"escalate_to":       escalateTo,
		"suggested_actions": suggestedActions,
		"confidence":        confidence,
	}
	return output, confidence, riskLevel, nil
}

// ----- cross_squad_coordinate -----

// crossSquadCoordinate handles resource contention and priority conflicts
// across squads. Expected context: conflicts (list of conflict objects).
func (a *CoordinatorAgent) crossSquadCoordinate(ctx context.Context, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	conflicts := params["conflicts"]
	var resolutions []map[string]interface{}

	// If we have specific conflicts, resolve them.
	if cList, ok := conflicts.([]interface{}); ok {
		for i, raw := range cList {
			if c, ok := raw.(map[string]interface{}); ok {
				resolution := resolveConflict(c, i)
				resolutions = append(resolutions, resolution)
			}
		}
	}

	// Default resolution if no conflicts specified.
	if len(resolutions) == 0 {
		resolutions = append(resolutions, map[string]interface{}{
			"conflict":  "general_coordination",
			"priority":  "balanced",
			"action":    "maintain_current_allocation",
			"reasoning": "无明确冲突，保持当前资源分配",
		})
	}

	confidence = 0.88
	riskLevel = "medium"

	output = map[string]interface{}{
		"status":         "completed",
		"decision_point": "cross_squad_coordinate",
		"resolutions":    resolutions,
		"conflict_count": len(resolutions),
		"confidence":     confidence,
	}
	return output, confidence, riskLevel, nil
}

// resolveConflict determines the priority and action for a single conflict.
func resolveConflict(conflict map[string]interface{}, idx int) map[string]interface{} {
	source := fmt.Sprintf("%v", conflict["source"])
	target := fmt.Sprintf("%v", conflict["target"])
	resource := fmt.Sprintf("%v", conflict["resource"])
	urgency := strings.ToLower(fmt.Sprintf("%v", conflict["urgency"]))

	priority := "normal"
	if urgency == "critical" || urgency == "high" {
		priority = "high"
	}

	return map[string]interface{}{
		"conflict_id": fmt.Sprintf("conflict-%d", idx+1),
		"source":      source,
		"target":      target,
		"resource":    resource,
		"priority":    priority,
		"action":      "escalate_to_human",
		"reasoning":   fmt.Sprintf("资源 %s 在 %s 和 %s 之间冲突，建议人工仲裁", resource, source, target),
	}
}

// ----- agent_audit -----

// agentAudit performs a periodic audit of agent decision quality.
// It reviews trust score trends, rejection patterns, and user feedback.
func (a *CoordinatorAgent) agentAudit(callerCtx context.Context, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	var auditResults []map[string]interface{}

	ctxGo, cancel := context.WithTimeout(callerCtx, 30*time.Second)
	defer cancel()

	// Try to fetch trust scores via tool registry first.
	var scores []agentTrust
	if ok, _ := a.invokeToolJSON(ctxGo, "coordinator.trust_score.list",
		map[string]interface{}{}, &scores); !ok || len(scores) == 0 {
		// Fall back to direct DB query.
		if a.db != nil {
			a.db.Raw(
				`SELECT agent_id, trust_score, autonomy_level as autonomy,
				        total_actions as total_act, rejected_actions as rejected_ct
				 FROM agent_trust_score
				 ORDER BY trust_score ASC`,
			).Scan(&scores)
		}
	}

	for _, s := range scores {
		status := "healthy"
		var issues []string

		if s.TrustScore < 0.3 {
			status = "at_risk"
			issues = append(issues, "信任分低于0.3，建议降级或停用")
		} else if s.TrustScore < 0.5 {
			status = "warning"
			issues = append(issues, "信任分偏低，需要关注")
		}

		if s.TotalAct > 0 {
			rejectRate := float64(s.RejectedCt) / float64(s.TotalAct)
			if rejectRate > 0.3 {
				issues = append(issues, fmt.Sprintf("拒绝率%.0f%%偏高", rejectRate*100))
				if status == "healthy" {
					status = "warning"
				}
			}
		}

		auditResults = append(auditResults, map[string]interface{}{
			"agent_id":    s.AgentID,
			"trust_score": math.Round(s.TrustScore*100) / 100,
			"autonomy":    s.Autonomy,
			"status":      status,
			"issues":      issues,
		})
	}

	// If no results, return a placeholder audit.
	if len(auditResults) == 0 {
		auditResults = append(auditResults, map[string]interface{}{
			"agent_id":    "all",
			"trust_score": "unknown",
			"status":      "no_data",
			"issues":      []string{"无法连接数据库进行审计"},
		})
		confidence = 0.5
	} else {
		confidence = 0.90
	}

	riskLevel = "medium"
	output = map[string]interface{}{
		"status":         "completed",
		"decision_point": "agent_audit",
		"audit_results":  auditResults,
		"audited_agents": len(auditResults),
		"confidence":     confidence,
	}
	return output, confidence, riskLevel, nil
}

// ----- tool registry helpers -----

// invokeToolInt64 calls a tool via DefaultRegistry and unmarshals the result
// into an int64 pointer. Returns (true, nil) on success, false on error/miss.
func (a *CoordinatorAgent) invokeToolInt64(ctx context.Context, name string, input map[string]interface{}, out *int64) (bool, error) {
	if toolregistry.DefaultRegistry == nil {
		return false, nil
	}
	result, err := toolregistry.DefaultRegistry.Invoke(name, input, ctx)
	if err != nil {
		return false, err
	}
	switch v := result.(type) {
	case float64:
		*out = int64(v)
		return true, nil
	case int64:
		*out = v
		return true, nil
	case int:
		*out = int64(v)
		return true, nil
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return false, err
		}
		*out = n
		return true, nil
	}
	return false, fmt.Errorf("invokeToolInt64: unexpected type %T for tool %s", result, name)
}

// invokeToolJSON calls a tool via DefaultRegistry and unmarshals the result
// into the provided target using json.Unmarshal. Returns (true, nil) on
// success, false on error/miss.
func (a *CoordinatorAgent) invokeToolJSON(ctx context.Context, name string, input map[string]interface{}, target interface{}) (bool, error) {
	if toolregistry.DefaultRegistry == nil {
		return false, nil
	}
	result, err := toolregistry.DefaultRegistry.Invoke(name, input, ctx)
	if err != nil {
		return false, err
	}
	// If the result is already the right type, assign directly.
	if t, ok := result.([]byte); ok {
		if err := json.Unmarshal(t, target); err != nil {
			return false, err
		}
		return true, nil
	}
	// Try marshalling and unmarshalling through JSON for safe conversion.
	data, err := json.Marshal(result)
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return false, err
	}
	return true, nil
}

// stringFromCtx safely extracts a string value from context map.
func stringFromCtx(ctx map[string]interface{}, key string, defaults ...string) string {
	if v, ok := ctx[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return ""
}
