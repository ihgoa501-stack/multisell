// Package impl provides concrete agent implementations.
//
// CoordinatorAgent implements G0 Coordinator — the supervisor agent that
// monitors system health, escalates anomalies, coordinates cross-squad
// collaboration, and audits other agents' decisions.
package impl

import (
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

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
func (a *CoordinatorAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "system_health":
		return a.systemHealth(ctx)
	case "anomaly_escalation":
		return a.anomalyEscalation(ctx)
	case "cross_squad_coordinate":
		return a.crossSquadCoordinate(ctx)
	case "agent_audit":
		return a.agentAudit(ctx)
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
func (a *CoordinatorAgent) systemHealth(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	health := "healthy"
	var warnings []string
	var criticals []string

	// Aggregate agent stats from DB if available.
	if a.db != nil {
		// Count pending actions older than 1 hour (SLA breach).
		var stalePending int64
		a.db.Raw(
			`SELECT COUNT(*) FROM unified_action
			 WHERE status IN ('suggested','pending')
			 AND proposed_at < ?`,
			time.Now().Add(-1*time.Hour),
		).Scan(&stalePending)
		if stalePending > 0 {
			warnings = append(warnings, fmt.Sprintf("%d pending actions exceed 1h SLA", stalePending))
		}

		// Count recent failures in the last 24h.
		var recentFails int64
		a.db.Raw(
			`SELECT COUNT(*) FROM ai_trace
			 WHERE status = 'failed'
			 AND started_at > ?`,
			time.Now().Add(-24*time.Hour),
		).Scan(&recentFails)
		if recentFails > 5 {
			warnings = append(warnings, fmt.Sprintf("%d trace failures in 24h", recentFails))
		}
		if recentFails > 20 {
			criticals = append(criticals, fmt.Sprintf("%d trace failures in 24h — critical", recentFails))
		}

		// Count actions with critical risk pending.
		var criticalPending int64
		a.db.Raw(
			`SELECT COUNT(*) FROM unified_action
			 WHERE status IN ('suggested','pending')
			 AND risk_level = 'critical'`,
		).Scan(&criticalPending)
		if criticalPending > 0 {
			criticals = append(criticals, fmt.Sprintf("%d critical-risk actions pending", criticalPending))
		}
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
func (a *CoordinatorAgent) anomalyEscalation(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	anomalyType := stringFromCtx(ctx, "anomaly_type", "unknown")
	severity := stringFromCtx(ctx, "severity", "low")
	details := stringFromCtx(ctx, "details", "")

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
		"status":           "completed",
		"decision_point":   "anomaly_escalation",
		"anomaly_type":     anomalyType,
		"severity":         severity,
		"details":          details,
		"escalate_to":      escalateTo,
		"suggested_actions": suggestedActions,
		"confidence":       confidence,
	}
	return output, confidence, riskLevel, nil
}

// ----- cross_squad_coordinate -----

// crossSquadCoordinate handles resource contention and priority conflicts
// across squads. Expected context: conflicts (list of conflict objects).
func (a *CoordinatorAgent) crossSquadCoordinate(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	conflicts := ctx["conflicts"]
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
			"conflict":   "general_coordination",
			"priority":   "balanced",
			"action":     "maintain_current_allocation",
			"reasoning":  "无明确冲突，保持当前资源分配",
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
func (a *CoordinatorAgent) agentAudit(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	var auditResults []map[string]interface{}

	if a.db != nil {
		// Query trust scores for all agents.
		type agentTrust struct {
			AgentID    string
			TrustScore float64
			Autonomy   string
			TotalAct   int
			RejectedCt int
		}
		var scores []agentTrust
		a.db.Raw(
			`SELECT agent_id, trust_score, autonomy_level as autonomy,
			        total_actions as total_act, rejected_actions as rejected_ct
			 FROM agent_trust_score
			 ORDER BY trust_score ASC`,
		).Scan(&scores)

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
	}

	// If no DB, return a placeholder audit.
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
