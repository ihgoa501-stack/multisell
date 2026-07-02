package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
)

// ── MOA Types ────────────────────────────────────────────────────────

// MOARequest describes a multi-agent orchestration request.
type MOARequest struct {
	Goal          string                 `json:"goal" binding:"required"`
	Context       map[string]interface{} `json:"context"`
	Participating []string               `json:"participating"`          // agent IDs, empty = all applicable
	Mode          string                 `json:"mode"`                   // dry_run | suggestion | production
}

// MOAResult is the output of a multi-agent orchestration.
type MOAResult struct {
	Goal          string            `json:"goal"`
	TraceID       string            `json:"trace_id"`
	Status        string            `json:"status"`         // completed | blocked | needs_approval
	AgentResults  []MOAAgentResult  `json:"agent_results"`
	Conflicts     []MOAConflict     `json:"conflicts,omitempty"`
	Suggestion    string            `json:"suggestion,omitempty"`
	RiskLevel     string            `json:"risk_level"`
	NeedsApproval bool              `json:"needs_approval"`
	ApprovalID    *int64            `json:"approval_id,omitempty"`
	ProposedAt    time.Time         `json:"proposed_at"`
}

// MOAAgentResult is one agent's contribution to an MOA.
type MOAAgentResult struct {
	AgentID    string                 `json:"agent_id"`
	Decision   string                 `json:"decision"`
	Confidence float64                `json:"confidence"`
	RiskLevel  string                 `json:"risk_level"`
	Output     map[string]interface{} `json:"output"`
}

// MOAConflict records where two agents disagree.
type MOAConflict struct {
	Between []string `json:"between"`
	On      string   `json:"on"`
	AOption string   `json:"a_option"`
	BOption string   `json:"b_option"`
}

// ── COORDINATOR ──────────────────────────────────────────────────────

// SchemaCatalog provides agent → decision point schema for coordinator.
// ponytail: flat map, expand to registry lookup if schema count grows.
type SchemaCatalog map[string][]string

// MOACoordinator runs a multi-agent orchestration: given a business goal,
// it spawns participating agents, collects results, detects conflicts,
// and produces a synthesized suggestion with risk assessment.
type MOACoordinator struct {
	orch     *Orchestrator
	bus      *eventbus.Bus
	approval *approval.Service
	catalog  SchemaCatalog
	logger   *zap.Logger
}

// NewMOACoordinator creates a coordinator. approval can be nil for dry-run only.
func NewMOACoordinator(orch *Orchestrator, bus *eventbus.Bus, approvalSvc *approval.Service, catalog SchemaCatalog, logger *zap.Logger) *MOACoordinator {
	return &MOACoordinator{
		orch:     orch,
		bus:      bus,
		approval: approvalSvc,
		catalog:  catalog,
		logger:   logger.Named("moa"),
	}
}

// Run executes a multi-agent orchestration.
func (c *MOACoordinator) Run(ctx context.Context, req *MOARequest) (*MOAResult, error) {
	participating, decisionPoints := c.resolveAgents(req.Goal, req.Participating)
	if len(participating) == 0 {
		return nil, fmt.Errorf("moa: no participating agents for goal %q", req.Goal)
	}

	c.logger.Info("moa run started",
		zap.String("goal", req.Goal),
		zap.String("mode", req.Mode),
		zap.Strings("agents", participating),
	)

	result := &MOAResult{
		Goal:       req.Goal,
		Status:     "running",
		ProposedAt: time.Now(),
	}

	// Phase 1: run each agent, collect results.
	for i, agentID := range participating {
		agentResult, err := c.runAgent(ctx, agentID, decisionPoints[i], req.Context)
		if err != nil {
			c.logger.Warn("moa agent failed",
				zap.String("agent", agentID),
				zap.Error(err),
			)
			result.AgentResults = append(result.AgentResults, MOAAgentResult{
				AgentID:  agentID,
				Decision: fmt.Sprintf("error: %s", err.Error()),
			})
			continue
		}
		result.AgentResults = append(result.AgentResults, *agentResult)
	}

	// Phase 2: detect conflicts.
	result.Conflicts = c.detectConflicts(result.AgentResults)

	// Phase 3: synthesize suggestion + risk level.
	riskLevel := c.computeRiskLevel(result.AgentResults, result.Conflicts)
	result.RiskLevel = riskLevel

	suggestion := c.synthesize(result.AgentResults, result.Conflicts)
	result.Suggestion = suggestion

	// Phase 4: approval gating for production mode.
	if req.Mode == "production" && riskLevel == "high" {
		result.Status = "needs_approval"
		result.NeedsApproval = true
		if c.approval != nil {
			// Create an approval request for the proposed action.
			// ponytail: MOA coordinator creates one approval per run,
			// per-agent approvals if finer granularity needed.
			body := map[string]interface{}{
				"goal":         req.Goal,
				"mode":         req.Mode,
				"agents":       participating,
				"conflicts":    result.Conflicts,
				"suggestion":   suggestion,
				"risk_level":   riskLevel,
			}
			payload, _ := json.Marshal(body)
			approvalReq, err := c.approval.Create(&approval.CreateApprovalInput{
				ProductID:   0,
				RequestType: "moa_proposal",
				Requester:   "moa_coordinator",
				NewValue:    string(payload),
				Reason:      fmt.Sprintf("MOA: %s (risk: %s, agents: %v)", req.Goal, riskLevel, participating),
				TargetType:  "moa_run",
				RiskLevel:   riskLevel,
			})
			if err != nil {
				c.logger.Warn("moa approval creation failed", zap.Error(err))
			} else {
				result.ApprovalID = &approvalReq.ID
				c.logger.Info("moa approval created",
					zap.Int64("approval_id", approvalReq.ID),
				)
			}
		}
	} else {
		result.Status = "completed"
	}

	// Publish MOA result event.
	if c.bus != nil {
		evtPayload := map[string]interface{}{
			"goal":           req.Goal,
			"mode":           req.Mode,
			"status":         result.Status,
			"risk_level":     riskLevel,
			"needs_approval": result.NeedsApproval,
			"agent_count":    len(participating),
			"conflict_count": len(result.Conflicts),
		}
		if result.ApprovalID != nil {
			evtPayload["approval_id"] = *result.ApprovalID
		}
		c.bus.Publish(ctx, "moa.completed", "moa", evtPayload)
	}

	c.logger.Info("moa run completed",
		zap.String("status", result.Status),
		zap.String("risk_level", riskLevel),
		zap.Int("agents", len(participating)),
		zap.Int("conflicts", len(result.Conflicts)),
	)

	return result, nil
}

// resolveAgents maps a goal to participating agents. If explicitly provided,
// use those; otherwise use the catalog.
func (c *MOACoordinator) resolveAgents(goal string, preferred []string) ([]string, []string) {
	if len(preferred) > 0 {
		dps := make([]string, len(preferred))
		for i, id := range preferred {
			if dp, ok := c.catalog[id]; ok && len(dp) > 0 {
				dps[i] = dp[0] // first decision point
			} else {
				dps[i] = "resolve"
			}
		}
		return preferred, dps
	}

	// By default, run known agents.
	ids := make([]string, 0, len(c.catalog))
	dps := make([]string, 0, len(c.catalog))
	for id, points := range c.catalog {
		ids = append(ids, id)
		dp := "resolve"
		if len(points) > 0 {
			dp = points[0]
		}
		dps = append(dps, dp)
	}
	return ids, dps
}

func (c *MOACoordinator) runAgent(ctx context.Context, agentID, decisionPoint string, contextMap map[string]interface{}) (*MOAAgentResult, error) {
	req := &RunAgentRequest{
		AgentID:       agentID,
		DecisionPoint: decisionPoint,
		Context:       contextMap,
	}

	result, err := c.orch.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &MOAAgentResult{
		AgentID:    result.AgentID,
		Decision:   result.DecisionPoint,
		Confidence: result.Confidence,
		RiskLevel:  result.RiskLevel,
		Output:     result.Output,
	}, nil
}

// detectConflicts finds disagreements between agents.
// ponytail: simple heuristic — agents whose confidence < 0.5 disagree with
// the majority; expand to semantic analysis if needed.
func (c *MOACoordinator) detectConflicts(results []MOAAgentResult) []MOAConflict {
	var conflicts []MOAConflict
	if len(results) < 2 {
		return nil
	}
	highCount := 0
	for _, r := range results {
		if r.Confidence >= 0.5 {
			highCount++
		}
	}
	if highCount < len(results) && highCount > 0 {
		var low []string
		for _, r := range results {
			if r.Confidence < 0.5 {
				low = append(low, r.AgentID)
			}
		}
		conflicts = append(conflicts, MOAConflict{
			Between: low,
			On:      "confidence_threshold",
			AOption: "recommend",
			BOption: "caution (low confidence)",
		})
	}
	return conflicts
}

// computeRiskLevel returns the highest risk across all agents.
func (c *MOACoordinator) computeRiskLevel(results []MOAAgentResult, _ []MOAConflict) string {
	levels := map[string]int{"low": 0, "medium": 1, "high": 2}
	maxLevel := 0
	for _, r := range results {
		if l, ok := levels[r.RiskLevel]; ok && l > maxLevel {
			maxLevel = l
		}
	}
	if maxLevel >= 2 {
		return "high"
	}
	if maxLevel >= 1 {
		return "medium"
	}
	return "low"
}

// synthesize produces a human-readable suggestion from agent results.
// ponytail: naive concatenation, replace with LLM summary when quality matters.
func (c *MOACoordinator) synthesize(results []MOAAgentResult, conflicts []MOAConflict) string {
	if len(results) == 0 {
		return "No agent results available."
	}
	text := fmt.Sprintf("MOA analysis for %d agent(s): ", len(results))
	for _, r := range results {
		text += fmt.Sprintf("[%s] confidence=%.2f risk=%s; ", r.AgentID, r.Confidence, r.RiskLevel)
	}
	if len(conflicts) > 0 {
		text += fmt.Sprintf("Conflicts: %d area(s) of disagreement. ", len(conflicts))
	}
	text += "Review recommended before action."
	return text
}
