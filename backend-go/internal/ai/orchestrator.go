package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/agent/impl"
	"github.com/lingmirror/backend-go/internal/domain/actionpolicy"
	"github.com/lingmirror/backend-go/internal/domain/agentrule"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"github.com/lingmirror/backend-go/internal/realtime"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Orchestrator coordinates AI agent workflows.
type Orchestrator struct {
	db         *gorm.DB
	logger     *zap.Logger
	registry   *AgentRegistry
	traces     *TraceWriter
	provider   LLMProvider
	agentImpls map[string]impl.Agent
	hub        *realtime.Hub

	// trustScoreSync forces synchronous trust score recalculation after each Run().
	// In production this runs in a goroutine; tests with SQLite can set this to
	// true to avoid table-lock races during cleanup.
	trustScoreSync bool
}

// NewOrchestrator creates a new AI orchestrator.
func NewOrchestrator(db *gorm.DB, logger *zap.Logger) *Orchestrator {
	return &Orchestrator{
		db:         db,
		logger:     logger,
		registry:   DefaultRegistry(),
		traces:     NewTraceWriter(db, logger),
		provider:   NewLLMProvider(logger),
		agentImpls: impl.All(db, logger),
	}
}

// WithProvider overrides the LLM provider (useful for tests).
func (o *Orchestrator) WithProvider(p LLMProvider) *Orchestrator {
	o.provider = p
	return o
}

// WithSyncTrustScore forces synchronous trust score recalculation after Run().
// In production the recalculation runs in a goroutine; tests using shared SQLite
// should set this to true to avoid table-lock races during cleanup.
func (o *Orchestrator) WithSyncTrustScore() *Orchestrator {
	o.trustScoreSync = true
	return o
}

// WithHub sets the WebSocket hub for real-time notifications.
func (o *Orchestrator) WithHub(hub *realtime.Hub) *Orchestrator {
	o.hub = hub
	return o
}

// Provider exposes the underlying LLM provider.
func (o *Orchestrator) Provider() LLMProvider { return o.provider }

// Registry exposes the agent registry for handlers.
func (o *Orchestrator) Registry() *AgentRegistry { return o.registry }

// TraceWriter exposes the trace writer.
func (o *Orchestrator) TraceWriter() *TraceWriter { return o.traces }

// RunAgentRequest is the input for executing an agent decision.
type RunAgentRequest struct {
	AgentID       string                 `json:"agent_id" binding:"required"`
	DecisionPoint string                 `json:"decision_point" binding:"required"`
	UserID        *int64                 `json:"user_id"`
	Context       map[string]interface{} `json:"context"`
	Stream        bool                   `json:"stream"`
}

// RunAgentResult is the output of an agent run.
type RunAgentResult struct {
	TraceID       string                 `json:"trace_id"`
	AgentID       string                 `json:"agent_id"`
	DecisionPoint string                 `json:"decision_point"`
	Output        map[string]interface{} `json:"output"`
	Confidence    float64                `json:"confidence"`
	RiskLevel     string                 `json:"risk_level"`
	Action        *UnifiedAction         `json:"action,omitempty"`
}

// Run executes an agent decision end-to-end:
// 1. Resolve agent spec
// 2. Start a trace
// 3. Emit prompt_start + tool_call events (stubbed)
// 4. Produce a synthesized final output
// 5. Apply personal rules
// 6. Check approval gate (approval policy + trust score)
// 7. Optionally create a unified action
// 8. Complete the trace
func (o *Orchestrator) Run(req *RunAgentRequest) (*RunAgentResult, error) {
	return o.runWithTimeout(req, 0)
}

// RunWithContext runs an agent decision with a context deadline.
// The context governs the entire execution — cancelled/deadline-exceeded
// contexts abort the run and return an error.
func (o *Orchestrator) RunWithContext(ctx context.Context, req *RunAgentRequest) (*RunAgentResult, error) {
	done := make(chan struct {
		res *RunAgentResult
		err error
	}, 1)
	go func() {
		res, err := o.runWithTimeout(req, 0)
		done <- struct {
			res *RunAgentResult
			err error
		}{res, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-done:
		return r.res, r.err
	}
}

// runWithTimeout is the internal implementation used by both Run and RunWithContext.
// A timeoutSeconds value of 0 means no timeout.
func (o *Orchestrator) runWithTimeout(req *RunAgentRequest, timeoutSeconds int) (*RunAgentResult, error) {
	agent, ok := o.registry.Get(req.AgentID)
	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", req.AgentID)
	}
	if !contains(agent.DecisionPoints, req.DecisionPoint) {
		return nil, fmt.Errorf("agent %s does not serve decision point %s", agent.ID, req.DecisionPoint)
	}

	// Start trace.
	ctxBytes, _ := json.Marshal(req.Context)
	traceID, err := o.traces.Start(&CreateTraceInput{
		AgentID:       agent.ID,
		DecisionPoint: req.DecisionPoint,
		UserID:        req.UserID,
		ModelProvider: o.provider.Name(),
		ModelName:     agent.ModelHint,
		PromptVersion: "v1",
		InputContext:  ctxBytes,
	})
	if err != nil {
		return nil, err
	}

	// On any early return, mark the trace as failed so it does not
	// remain in 'running' status forever.
	var runErr error
	defer func() {
		if runErr != nil {
			finalBytes, _ := json.Marshal(map[string]string{"error": runErr.Error()})
			_, _ = o.traces.Complete(traceID, &CompleteTraceInput{
				FinalOutput: finalBytes,
				Confidence:  nil,
				RiskLevel:   "high",
				Status:      "failed",
			})
		}
	}()

	// Emit prompt_start.
	_, _ = o.traces.AppendEvent(traceID, &AppendEventInput{
		EventType: "prompt_start",
		Content:   "prompt v1 dispatched to " + agent.ModelHint,
		Payload:   mustJSON(map[string]interface{}{"agent": agent.ID, "decision_point": req.DecisionPoint}),
	})

	// Emit tool_call stub events.
	toolCalls := stubToolCalls(agent, req.DecisionPoint)
	for _, tc := range toolCalls {
		_, _ = o.traces.AppendEvent(traceID, &AppendEventInput{
			EventType: "tool_call",
			Content:   tc,
			Payload:   mustJSON(map[string]interface{}{"tool": tc, "status": "ok"}),
		})
		// Attach evidence for known lookups.
		if ev := stubEvidenceForTool(tc); ev != nil {
			_, _ = o.traces.AddEvidence(traceID, ev)
		}
	}

	// Synthesize final output. If a real LLM provider is configured, call it;
	// otherwise fall back to the deterministic stub.
	output, confidence, riskLevel, err := o.synthesizeOutput(agent, req.DecisionPoint, req.Context)
	if err != nil {
		runErr = err
		return nil, err
	}

	// Apply personal rules to modify/block the output.
	if req.UserID != nil {
		ruleSvc := agentrule.NewService(o.db, o.logger)
		ruleResult, ruleErr := ruleSvc.Evaluate(*req.UserID, agent.ID, req.DecisionPoint, output)
		if ruleErr != nil {
			o.logger.Warn("personal rules evaluation failed", zap.String("agent_id", agent.ID), zap.Error(ruleErr))
		} else if ruleResult.Blocked {
			o.logger.Info("action blocked by personal rule", zap.String("agent_id", agent.ID), zap.String("reason", ruleResult.BlockReason))
			finalBytes, _ := json.Marshal(ruleResult.Output)
			_, _ = o.traces.Complete(traceID, &CompleteTraceInput{
				FinalOutput: finalBytes,
				Confidence:  &confidence,
				RiskLevel:   riskLevel,
				TokenCount:  380 + len(toolCalls)*120,
				Status:      "blocked",
			})
			return &RunAgentResult{
				TraceID:       traceID,
				AgentID:       agent.ID,
				DecisionPoint: req.DecisionPoint,
				Output:        ruleResult.Output,
				Confidence:    confidence,
				RiskLevel:     riskLevel,
			}, nil
		}
		output = ruleResult.Output
	}

	// ──────────────────────────────────────────────────────────────
	// Approval Gate: check ApprovalPolicy for this agent+decision
	// ──────────────────────────────────────────────────────────────
	policySvc := actionpolicy.NewService(o.db, o.logger)
	approvalPolicy, polErr := policySvc.GetMatchingPolicy(agent.ID, req.DecisionPoint)
	if polErr != nil {
		o.logger.Warn("approval policy lookup failed", zap.String("agent_id", agent.ID), zap.Error(polErr))
	} else if approvalPolicy != nil && approvalPolicy.RequiresApproval {
		// Check the agent's current trust score.
		trustSvc := trustscore.NewService(o.db, o.logger)
		ts, tsErr := trustSvc.GetByAgent(agent.ID)
		if tsErr != nil {
			o.logger.Warn("trust score lookup failed for approval gate", zap.String("agent_id", agent.ID), zap.Error(tsErr))
		} else {
			tsVal := 0.0
			if ts != nil {
				tsVal = ts.TrustScore
			}
			if tsVal < approvalPolicy.MinTrustScore {
				// Trust score too low — gate this decision.
				o.logger.Info("approval gate triggered",
					zap.String("agent_id", agent.ID),
					zap.String("decision_point", req.DecisionPoint),
					zap.Float64("trust_score", tsVal),
					zap.Float64("min_trust_score", approvalPolicy.MinTrustScore),
				)

				payloadBytes, _ := json.Marshal(output)
				req, createErr := policySvc.SubmitApproval(
					approvalPolicy.ID,
					agent.ID,
					req.DecisionPoint,
					payloadBytes,
					"agent:"+agent.ID,
				)
				if createErr != nil {
					o.logger.Warn("failed to create approval request", zap.Error(createErr))
				} else {
					// Broadcast WebSocket notification.
					if o.hub != nil && req != nil {
						notif, _ := json.Marshal(map[string]interface{}{
							"type": "approval:new_request",
							"data": req,
						})
						o.hub.Broadcast(notif)
					}
				}

				// Complete trace with gated status.
				finalBytes, _ := json.Marshal(output)
				_, _ = o.traces.Complete(traceID, &CompleteTraceInput{
					FinalOutput: finalBytes,
					Confidence:  &confidence,
					RiskLevel:   riskLevel,
					TokenCount:  380 + len(toolCalls)*120,
					Status:      "gated",
				})
				return &RunAgentResult{
					TraceID:       traceID,
					AgentID:       agent.ID,
					DecisionPoint: req.DecisionPoint,
					Output:        output,
					Confidence:    confidence,
					RiskLevel:     riskLevel,
				}, nil
			}
			// Trust score sufficient — auto-approve and continue.
			// (Proceed to normal action creation below.)
		}
	}

	// Emit reasoning event.
	_, _ = o.traces.AppendEvent(traceID, &AppendEventInput{
		EventType: "reasoning",
		Content:   stubReasoning(agent, req.DecisionPoint),
		Payload:   mustJSON(output),
	})

	// Optionally create a unified action if the agent's autonomy is non-advisory.
	var action *UnifiedAction
	if agent.Autonomy != "advisory" {
		actionInput := &CreateActionInput{
			SourceTable:        "ai_trace",
			SourceID:           traceID,
			SourceType:         "agent_run",
			TraceID:            traceID,
			AgentID:            agent.ID,
			SquadID:            agent.Squad,
			UserID:             req.UserID,
			ActionType:         req.DecisionPoint,
			BusinessObjectType: stubBusinessObject(req.DecisionPoint),
			Title:              stubActionTitle(agent, req.DecisionPoint),
			Description:        stubReasoning(agent, req.DecisionPoint),
			Payload:            mustJSON(output),
			RiskLevel:          maxRisk(agent.RiskFloor, riskLevel),
			Confidence:         &confidence,
			ProposedBy:         "agent:" + agent.ID,
		}
		// Advisory = no approval needed; supervised = needs approval.
		requires := agent.Autonomy == "supervised" || agent.Autonomy == "guided"
		actionInput.RequiresApproval = &requires
		action, _ = o.persistAction(actionInput)
		// Evaluate against approval policy for auto-approve/block decisions.
		if action != nil {
			policySvc := actionpolicy.NewService(o.db, o.logger)
			amount, quantity := actionpolicy.UnmarshalPayload(action.Payload)
			polCtx := &actionpolicy.ActionContext{
				AgentID:            action.AgentID,
				SquadID:            action.SquadID,
				ActionType:         action.ActionType,
				RiskLevel:          action.RiskLevel,
				BusinessObjectType: action.BusinessObjectType,
				BusinessObjectID:   action.BusinessObjectID,
				Amount:             amount,
				Quantity:           quantity,
				Confidence:         action.Confidence,
			}
			result, policyErr := policySvc.Evaluate(polCtx)
			if policyErr != nil {
				o.logger.Warn("policy evaluation failed", zap.Int64("action_id", action.ID), zap.Error(policyErr))
			} else if result.FinalOutcome == "auto_approve" {
				aiSvc := NewService(o.db, o.logger)
				if _, err := aiSvc.ApproveAction(action.ID, "policy", "auto-approved: "+result.Verdicts[0].Reason); err != nil {
					o.logger.Warn("auto-approve failed", zap.Error(err))
				} else if _, err := aiSvc.ExecuteAction(action.ID, "policy", "auto-executed"); err != nil {
					o.logger.Warn("auto-execute failed", zap.Error(err))
				} else {
					o.logger.Info("policy auto-approved and executed action", zap.Int64("action_id", action.ID))
					action, _ = aiSvc.GetAction(action.ID)
				}
			} else if result.FinalOutcome == "block" {
				o.logger.Warn("policy blocked action", zap.Int64("action_id", action.ID))
				aiSvc := NewService(o.db, o.logger)
				if _, err := aiSvc.RejectAction(action.ID, "policy", "blocked: "+result.Verdicts[0].Reason); err != nil {
					o.logger.Warn("reject failed", zap.Error(err))
				} else {
					action, _ = aiSvc.GetAction(action.ID)
				}
			}
		}
	}

	// Complete trace.
	finalBytes, _ := json.Marshal(output)
	_, _ = o.traces.Complete(traceID, &CompleteTraceInput{
		FinalOutput: finalBytes,
		Confidence:  &confidence,
		RiskLevel:   riskLevel,
		TokenCount:  380 + len(toolCalls)*120,
		Status:      "completed",
	})

	// Trigger trust score recalculation asynchronously — must not block the
	// agent run response. Recalculation iterates all agents and can be expensive
	// when the trace or action tables are large.
	recalcTrustScores := func() {
		tsSvc := trustscore.NewService(o.db, o.logger)
		if err := tsSvc.Recalculate(); err != nil {
			o.logger.Warn("trust score recalculation failed", zap.Error(err))
			return
		}
		ug := trustscore.NewUpgrader(o.db, o.logger)
		if upgraded, err := ug.UpgradeEligible(); err != nil {
			o.logger.Warn("autonomy upgrade failed", zap.Error(err))
		} else if len(upgraded) > 0 {
			for _, u := range upgraded {
				o.logger.Info("agent autonomy upgraded via trust score",
					zap.String("agent", u.AgentID),
					zap.String("from", u.FromLevel),
					zap.String("to", u.ToLevel),
				)
			}
		}
	}
	if o.trustScoreSync {
		recalcTrustScores()
	} else {
		go recalcTrustScores()
	}

	return &RunAgentResult{
		TraceID:       traceID,
		AgentID:       agent.ID,
		DecisionPoint: req.DecisionPoint,
		Output:        output,
		Confidence:    confidence,
		RiskLevel:     riskLevel,
		Action:        action,
	}, nil
}

// persistAction stores a unified action via the Service to avoid duplicates.
func (o *Orchestrator) persistAction(in *CreateActionInput) (*UnifiedAction, error) {
	svc := NewService(o.db, o.logger)
	return svc.CreateAction(in)
}

// synthesizeOutput produces the agent's final output. It checks for a concrete
// agent implementation first, and falls back to the LLM provider or deterministic
// stub when no implementation is registered.
func (o *Orchestrator) synthesizeOutput(agent AgentSpec, dp string, ctx map[string]interface{}) (map[string]interface{}, float64, string, error) {
	// Check if there is a concrete implementation for this agent.
	if implAgent, ok := o.agentImpls[agent.ID]; ok {
		o.logger.Debug("using concrete agent implementation",
			zap.String("agent_id", agent.ID),
			zap.String("decision_point", dp),
		)
		return implAgent.Decide(dp, ctx)
	}

	// Build the prompt.
	system := fmt.Sprintf("You are %s (%s), a LingMirror agent in the %s squad. Decision point: %s. Description: %s. Respond in Chinese, be concise (<=120 chars).",
		agent.ID, agent.Name, agent.Squad, dp, agent.Description)
	userMsg := fmt.Sprintf("Agent %s @ %s — please decide.", agent.ID, dp)
	if ctx != nil {
		if m, ok := ctx["message"].(string); ok && m != "" {
			userMsg = m
		}
	}
	req := &LLMRequest{
		Model:    agent.ModelHint,
		System:   system,
		Messages: []LLMMessage{{Role: "user", Content: userMsg}},
		Metadata: map[string]interface{}{"agent_id": agent.ID, "decision_point": dp},
	}

	// Only call the real provider when it isn't the stub, to avoid silently
	// depending on an API key in dev.
	if o.provider != nil && o.provider.Name() != "stub" {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := o.provider.Chat(ctxTimeout, req)
		if err != nil {
			o.logger.Warn("LLM call failed",
				zap.String("provider", o.provider.Name()),
				zap.String("agent", agent.ID),
				zap.Error(err))
			// In production, do NOT silently fall back to stub output.
			envVal := strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
			ginVal := strings.ToLower(strings.TrimSpace(os.Getenv("GIN_MODE")))
			if envVal == "production" || ginVal == "release" {
				return nil, 0, "", fmt.Errorf("LLM provider call failed: %w", err)
			}
		} else if resp != nil {
			out := map[string]interface{}{
				"agent_id":       agent.ID,
				"decision_point": dp,
				"recommendation": resp.Answer,
				"reasoning":      stubReasoning(agent, dp),
				"model":          resp.Model,
				"tokens_in":      resp.TokensIn,
				"tokens_out":     resp.TokensOut,
				"latency_ms":     resp.LatencyMs,
				"timestamp":      time.Now().Format(time.RFC3339),
			}
			if ctx != nil {
				out["echoed_context"] = ctx
			}
			confidence := 0.82
			if resp.TokensOut > 0 {
				confidence = clampConfidence(0.65 + float64(resp.TokensOut)/600.0)
			}
			return out, confidence, agent.RiskFloor, nil
		}
	}
	out, conf, risk := stubFinalOutput(agent, dp, ctx)
	return out, conf, risk, nil
}

func clampConfidence(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 0.99 {
		return 0.99
	}
	return v
}

// Chat is a convenience entry point for /ai/chat. It maps a natural-language
// message to an agent + decision point using simple keyword matching.
func (o *Orchestrator) Chat(message string, userID *int64) (*RunAgentResult, error) {
	agentID, decisionPoint := routeChat(message)
	if agentID == "" {
		agentID = "G1"
		decisionPoint = "dashboard_overview"
	}
	return o.Run(&RunAgentRequest{
		AgentID:       agentID,
		DecisionPoint: decisionPoint,
		UserID:        userID,
		Context:       map[string]interface{}{"message": message},
	})
}

// routeEntry associates keywords with a target agent + decision point.
type routeEntry struct {
	Keywords      []string
	AgentID       string
	DecisionPoint string
}

// routeTable is the ordered routing table. Earlier entries win on ties.
var routeTable = []routeEntry{
	{Keywords: []string{"库存", "缺货", "补货", "stock", "inventory", "replenishment", "backorder", "存货"}, AgentID: "A5", DecisionPoint: "stock_alert"},
	{Keywords: []string{"利润", "成本", "亏损", "profit", "cost", "margin"}, AgentID: "A6", DecisionPoint: "profit_check"},
	{Keywords: []string{"listing", "标题", "描述", "优化", "关键词", "keyword"}, AgentID: "A2", DecisionPoint: "listing_optimize"},
	{Keywords: []string{"广告", "acos", "投放", "ad", "ads", "spend"}, AgentID: "A3", DecisionPoint: "acos_analysis"},
	{Keywords: []string{"选品", "新品", "市场", "scout", "product"}, AgentID: "A1", DecisionPoint: "product_scout"},
	{Keywords: []string{"客服", "回复", "意图", "customer", "service", "退款", "退货"}, AgentID: "A4", DecisionPoint: "auto_reply"},
	{Keywords: []string{"合规", "认证", "禁售", "compliance", "regulation", "cert"}, AgentID: "A7", DecisionPoint: "compliance_check"},
	{Keywords: []string{"折扣", "促销", "价格底线", "discount", "promotion", "coupon"}, AgentID: "G3", DecisionPoint: "discount_check"},
	{Keywords: []string{"仓库", "报关", "warehouse", "customs", "仓储"}, AgentID: "G2", DecisionPoint: "warehouse_routing"},
	{Keywords: []string{"dashboard", "概览", "总览", "overview", "驾驶舱"}, AgentID: "G1", DecisionPoint: "dashboard_overview"},
}

// routeChat scores keywords against each route entry and returns the best match.
func routeChat(msg string) (string, string) {
	m := strings.ToLower(msg)
	bestScore := 0
	bestAgent := ""
	bestPoint := ""
	for _, entry := range routeTable {
		score := 0
		for _, kw := range entry.Keywords {
			if strings.Contains(m, kw) {
				score++
			}
		}
		if score > 0 && score > bestScore {
			bestScore = score
			bestAgent = entry.AgentID
			bestPoint = entry.DecisionPoint
		}
	}
	return bestAgent, bestPoint
}

// ---- stub helpers ----

func stubToolCalls(agent AgentSpec, dp string) []string {
	switch agent.ID {
	case "A1":
		return []string{"search_market_trend", "fetch_competitor_listings"}
	case "A2":
		return []string{"fetch_listing", "keyword_suggest"}
	case "A3":
		return []string{"fetch_ad_metrics", "acos_compute"}
	case "A4":
		return []string{"intent_classify", "draft_reply"}
	case "A5":
		return []string{"fetch_inventory", "compute_replenishment"}
	case "A6":
		return []string{"fetch_sku_profit", "cost_breakdown"}
	case "A7":
		return []string{"fetch_compliance_rules", "scan_keywords"}
	case "G1":
		return []string{"fetch_dashboard_summary"}
	case "G2":
		return []string{"fetch_warehouse_capacity", "customs_check"}
	case "G3":
		return []string{"fetch_discount_rule", "margin_check"}
	}
	return []string{"noop"}
}

func stubEvidenceForTool(tool string) *AddEvidenceInput {
	switch tool {
	case "fetch_inventory":
		return &AddEvidenceInput{SourceType: "inventory", SourceID: "demo-sku-1", Title: "SKU 库存快照", Summary: "当前可用库存 12 件，安全库存 50 件"}
	case "fetch_sku_profit":
		return &AddEvidenceInput{SourceType: "sku", SourceID: "demo-sku-1", Title: "SKU 利润快照", Summary: "利润率 8.2%，低于目标 15%"}
	case "fetch_ad_metrics":
		return &AddEvidenceInput{SourceType: "ad", SourceID: "camp-42", Title: "广告活动 7 日指标", Summary: "ACOS 32%，高于目标 20%"}
	case "fetch_compliance_rules":
		return &AddEvidenceInput{SourceType: "compliance", SourceID: "rule-v3", Title: "合规规则集 v3", Summary: "命中 2 条禁售词"}
	}
	return nil
}

func stubFinalOutput(agent AgentSpec, dp string, ctx map[string]interface{}) (map[string]interface{}, float64, string) {
	confidence := 0.78
	risk := agent.RiskFloor
	if risk == "" {
		risk = "low"
	}
	out := map[string]interface{}{
		"agent_id":       agent.ID,
		"decision_point": dp,
		"recommendation": stubRecommendation(agent, dp),
		"reasoning":      stubReasoning(agent, dp),
		"timestamp":      time.Now().Format(time.RFC3339),
	}
	if ctx != nil {
		out["echoed_context"] = ctx
	}
	return out, confidence, risk
}

func stubRecommendation(agent AgentSpec, dp string) string {
	switch agent.ID {
	case "A5":
		return "建议立即补货 200 件，并切换至渠道 B（运费低 18%）"
	case "A6":
		return "建议将该 SKU 售价上调 6%，否则利润率将持续低于目标"
	case "A2":
		return "建议重写标题：前置核心关键词，描述补充规格卖点"
	case "A7":
		return "建议移除标题中的敏感词，提交 CE 认证材料"
	case "G3":
		return "折扣 25% 触发价格底线，建议下调至 18%"
	}
	return "已生成建议，详见 trace"
}

func stubReasoning(agent AgentSpec, dp string) string {
	return fmt.Sprintf("Agent %s (%s) 在 %s 决策点完成推理：综合工具调用结果与历史规则，给出上述建议。", agent.ID, agent.Name, dp)
}

func stubActionTitle(agent AgentSpec, dp string) string {
	return fmt.Sprintf("[%s] %s — %s", agent.ID, agent.Name, dp)
}

func stubBusinessObject(dp string) string {
	switch {
	case strings.Contains(dp, "stock"), strings.Contains(dp, "replenishment"):
		return "inventory"
	case strings.Contains(dp, "listing"), strings.Contains(dp, "keyword"):
		return "listing"
	case strings.Contains(dp, "profit"), strings.Contains(dp, "cost"):
		return "sku"
	case strings.Contains(dp, "discount"), strings.Contains(dp, "promotion"):
		return "price_rule"
	case strings.Contains(dp, "compliance"):
		return "product"
	}
	return "general"
}

func maxRisk(a, b string) string {
	order := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	if order[a] >= order[b] {
		return a
	}
	return b
}

func contains(slice []string, s string) bool {
	for _, x := range slice {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

func mustJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

// ErrUnknownAgent is returned when an agent ID cannot be resolved.
var ErrUnknownAgent = errors.New("unknown agent")
