package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"crypto/sha256"
	"github.com/lingmirror/backend-go/internal/agent/impl"
	"github.com/lingmirror/backend-go/internal/aios/costcontrol"
	"github.com/lingmirror/backend-go/internal/aios/guardrails"
	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/domain/actionpolicy"
	"github.com/lingmirror/backend-go/internal/domain/agentrule"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"github.com/lingmirror/backend-go/internal/platform/actioncatalog"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"github.com/lingmirror/backend-go/internal/realtime"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"sort"
	"sync"
)

// EventPublisher publishes events to the event bus.
// This interface avoids direct dependency on the eventbus package.
type EventPublisher interface {
	Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error)
}

// Orchestrator coordinates AI agent workflows.
type Orchestrator struct {
	db            *gorm.DB
	logger        *zap.Logger
	registry      *AgentRegistry
	traces        *TraceWriter
	provider      LLMProvider
	agentImpls    map[string]impl.Agent
	hub           *realtime.Hub
	bus           EventPublisher
	decisionCache *decisionCache

	// guardrails is the AIOS guardrails chain for L1-L5 defensive checks.
	// When nil, all checks pass through (no-op). Set via WithGuardrails().
	guardrails *guardrails.Chain

	// budget controls LLM spend. When nil, no cost capping applied.
	budget *costcontrol.Controller

	// trustScoreSync forces synchronous trust score recalculation after each Run().
	// In production this runs in a goroutine; tests with SQLite can set this to
	// true to avoid table-lock races during cleanup.
	trustScoreSync bool

	// cmd is the command dispatcher for executing actions through registered handlers.
	cmd *command.Dispatcher

	// cat is the action catalog for production validation before execution.
	cat *actioncatalog.Catalog
}

// NewOrchestrator creates a new AI orchestrator.
func NewOrchestrator(db *gorm.DB, logger *zap.Logger) *Orchestrator {
	return &Orchestrator{
		decisionCache: newDecisionCache(5 * time.Minute),
		db:            db,
		logger:        logger,
		registry:      DefaultRegistry(),
		traces:        NewTraceWriter(db, logger),
		provider:      NewLLMProvider(logger),
		agentImpls:    impl.All(db, logger),
	}
}

// WithProvider overrides the LLM provider (useful for tests).
func (o *Orchestrator) WithProvider(p LLMProvider) *Orchestrator {
	o.provider = p
	return o
}

// Provider exposes the underlying LLM provider.
func (o *Orchestrator) Provider() LLMProvider { return o.provider }

// Registry exposes the agent registry for handlers.
func (o *Orchestrator) Registry() *AgentRegistry { return o.registry }

// TraceWriter exposes the trace writer.
func (o *Orchestrator) TraceWriter() *TraceWriter { return o.traces }

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

// WithBus sets the event bus for publishing agent decision events.
func (o *Orchestrator) WithBus(bus EventPublisher) *Orchestrator {
	o.bus = bus
	return o
}

// WithDispatcher attaches a command dispatcher so that auto-execution
// path dispatches through registered command handlers.
func (o *Orchestrator) WithDispatcher(cmd *command.Dispatcher) *Orchestrator {
	o.cmd = cmd
	return o
}

// WithCatalog attaches an action catalog for production validation
// before action execution.
func (o *Orchestrator) WithCatalog(cat *actioncatalog.Catalog) *Orchestrator {
	o.cat = cat
	return o
}

// RegisterAgent adds or replaces an agent implementation after construction.
func (o *Orchestrator) RegisterAgent(id string, agent impl.Agent) {
	o.agentImpls[id] = agent
}

// WithDecisionCache configures the decision cache TTL. A zero TTL disables caching.
func (o *Orchestrator) WithDecisionCache(ttl time.Duration) *Orchestrator {
	if ttl <= 0 {
		o.decisionCache = nil
	} else {
		o.decisionCache = newDecisionCache(ttl)
	}
	return o
}

// ClearDecisionCache clears all cached decision results.
func (o *Orchestrator) ClearDecisionCache() {
	if o.decisionCache != nil {
		o.decisionCache.clear()
	}
}

// WithGuardrails sets the AIOS guardrails chain for L1-L5 defensive checks.
func (o *Orchestrator) WithGuardrails(c *guardrails.Chain) *Orchestrator {
	o.guardrails = c
	return o
}

// WithBudget sets the LLM cost budget controller.
func (o *Orchestrator) WithBudget(b *costcontrol.Controller) *Orchestrator {
	o.budget = b
	return o
}

// RunAgentRequest is the input for executing an agent decision.
type RunAgentRequest struct {
	AgentID       string                 `json:"agent_id" binding:"required"`
	DecisionPoint string                 `json:"decision_point" binding:"required"`
	UserID        *int64                 `json:"user_id"`
	Context       map[string]interface{} `json:"context"`
	Stream        bool                   `json:"stream"`
	ParentTraceID string                 `json:"parent_trace_id,omitempty"`
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
// 5. Optionally create a unified action
// 6. Complete the trace
//
// In production this would proxy to an LLM provider and real tool calls.
// For now it produces deterministic stub output so the full UI flow works.
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
	// Build a context with agent identity so tool calls carry caller info.
	agentCtx := toolregistry.WithAgentID(context.Background(), agent.ID)
	agentCtx = toolregistry.WithAgentWorkspace(agentCtx)
	output, confidence, riskLevel, err := o.synthesizeOutput(agentCtx, agent, req.DecisionPoint, req.Context)
	if err != nil {
		// Complete the trace with failed status so the failure is recorded, not silent.
		errMsg := err.Error()
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
		_, _ = o.traces.Complete(traceID, &CompleteTraceInput{
			FinalOutput: []byte(`{"error":"` + strings.ReplaceAll(errMsg, `"`, `\"`) + `"}`),
			Confidence:  nil,
			RiskLevel:   "medium",
			TokenCount:  0,
			Status:      "failed",
		})
		// Publish agent.decided event with error context for audit/pipeline.
		o.publishDecisionEvent(traceID, agent, req, map[string]interface{}{"error": errMsg}, 0, "medium")
		return &RunAgentResult{
			TraceID:       traceID,
			AgentID:       agent.ID,
			DecisionPoint: req.DecisionPoint,
			Output:        map[string]interface{}{"error": errMsg, "failed": true},
			Confidence:    0,
			RiskLevel:     "medium",
		}, nil
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

	// Emit reasoning event.
	_, _ = o.traces.AppendEvent(traceID, &AppendEventInput{
		EventType: "reasoning",
		Content:   stubReasoning(agent, req.DecisionPoint),
		Payload:   mustJSON(output),
	})

	// Optionally create a unified action if the agent's autonomy is non-advisory.
	// The autonomy level is looked up from the trust score table (dynamic, with
	// fallback to the in-memory registry spec) so that agent upgrades from the
	// autonomy pipeline are reflected at runtime.
	var action *UnifiedAction
	dynamicAutonomy := agent.Autonomy
	if ts, tsErr := trustscore.NewService(o.db, o.logger).GetByAgent(agent.ID); tsErr == nil && ts != nil {
		dynamicAutonomy = ts.AutonomyLevel
	}
	if dynamicAutonomy != "advisory" {
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
		// advisory = no action; guided/supervised = human approval required; autonomous = no approval needed
		requires := dynamicAutonomy == "supervised" || dynamicAutonomy == "guided"
		actionInput.RequiresApproval = &requires
		action, _ = o.persistAction(actionInput)
		// Forbidden action check: block actions matching forbidden_action table.
		if action != nil {
			if fbErr := actionpolicy.CheckForbidden(o.db, action.AgentID, action.ActionType, action.RiskLevel); fbErr != nil {
				o.logger.Warn("action blocked by forbidden rules", zap.Int64("action_id", action.ID), zap.Error(fbErr))
				aiSvc := NewService(o.db, o.logger).WithDispatcher(o.cmd).WithCatalog(o.cat)
				if _, rejErr := aiSvc.RejectAction(action.ID, "governance", nil, "blocked: "+fbErr.Error()); rejErr != nil {
					o.logger.Warn("failed to reject forbidden action", zap.Error(rejErr))
				}
				action, _ = aiSvc.GetAction(action.ID)
			}
		}
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
				aiSvc := NewService(o.db, o.logger).WithDispatcher(o.cmd).WithCatalog(o.cat)
				if _, err := aiSvc.ApproveAction(action.ID, "policy", nil, "auto-approved: "+result.Verdicts[0].Reason); err != nil {
					o.logger.Warn("auto-approve failed", zap.Error(err))
				} else if _, err := aiSvc.ExecuteAction(action.ID, nil, "policy", "auto-executed"); err != nil {
					o.logger.Warn("auto-execute failed", zap.Error(err))
				} else {
					o.logger.Info("policy auto-approved and executed action", zap.Int64("action_id", action.ID))
					action, _ = aiSvc.GetAction(action.ID)
				}
			} else if result.FinalOutcome == "block" {
				o.logger.Warn("policy blocked action", zap.Int64("action_id", action.ID))
				aiSvc := NewService(o.db, o.logger).WithDispatcher(o.cmd).WithCatalog(o.cat)
				if _, err := aiSvc.RejectAction(action.ID, "policy", nil, "blocked: "+result.Verdicts[0].Reason); err != nil {
					o.logger.Warn("reject failed", zap.Error(err))
				} else {
					action, _ = aiSvc.GetAction(action.ID)
				}
			}
			// If action requires approval and wasn't auto-processed, create approval request.
			if action != nil && requires {
				o.ensureApprovalCreated(action)
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
	// Cache the result if caching is enabled.
	if o.decisionCache != nil {
		o.decisionCache.set(req.AgentID, req.DecisionPoint, req.Context, &RunAgentResult{
			TraceID:       traceID,
			AgentID:       agent.ID,
			DecisionPoint: req.DecisionPoint,
			Output:        output,
			Confidence:    confidence,
			RiskLevel:     riskLevel,
			Action:        action,
		})
	}

	// Publish agent.decided.* event to the event bus for pipeline chaining.
	o.publishDecisionEvent(traceID, agent, req, output, confidence, riskLevel)

	// Trigger trust score recalculation asynchronously — must not block the
	// agent run response. Recalculation iterates all agents and can be expensive
	// when the trace or action tables are large.
	go func(db *gorm.DB, logger *zap.Logger) {
		tsSvc := trustscore.NewService(db, logger)
		if err := tsSvc.Recalculate(); err != nil {
			logger.Warn("trust score recalculation failed", zap.Error(err))
			return
		}
		ug := trustscore.NewUpgrader(db, logger)
		if upgraded, err := ug.UpgradeEligible(); err != nil {
			logger.Warn("autonomy upgrade failed", zap.Error(err))
		} else if len(upgraded) > 0 {
			for _, u := range upgraded {
				logger.Info("agent autonomy upgraded via trust score",
					zap.String("agent", u.AgentID),
					zap.String("from", u.FromLevel),
					zap.String("to", u.ToLevel),
				)
			}
		}
	}(o.db, o.logger)

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

// publishDecisionEvent publishes an agent.decided.* event to the event bus
// so that pipeline chain subscribers can react to this agent's decision.
func (o *Orchestrator) publishDecisionEvent(traceID string, agent AgentSpec, req *RunAgentRequest, output map[string]interface{}, confidence float64, riskLevel string) {
	if o.bus == nil {
		return
	}
	payload := make(map[string]interface{}, len(output)+5)
	for k, v := range output {
		payload[k] = v
	}
	payload["trace_id"] = traceID
	payload["agent_id"] = agent.ID
	payload["decision_point"] = req.DecisionPoint
	payload["confidence"] = confidence
	payload["risk_level"] = riskLevel
	if req.ParentTraceID != "" {
		payload["parent_trace_id"] = req.ParentTraceID
	}
	topic := fmt.Sprintf("agent.decided.%s.%s", agent.ID, req.DecisionPoint)
	go func() {
		_, err := o.bus.Publish(context.Background(), topic, "orchestrator", payload)
		if err != nil {
			o.logger.Warn("failed to publish agent.decided event",
				zap.String("topic", topic),
				zap.Error(err))
		}
	}()
}

// persistAction stores a unified action via the Service to avoid duplicates.
func (o *Orchestrator) persistAction(in *CreateActionInput) (*UnifiedAction, error) {
	svc := NewService(o.db, o.logger).WithDispatcher(o.cmd).WithCatalog(o.cat)
	return svc.CreateAction(in)
}

// ensureApprovalCreated creates an approval_request for a UnifiedAction that
// requires human review. The caller already has the action with a fresh status
// (post-policy-evaluation), so we use it directly to avoid a DB re-read.
func (o *Orchestrator) ensureApprovalCreated(a *UnifiedAction) {
	if a.Status != ActionStatusSuggested && a.Status != ActionStatusEscalated {
		return // already processed by policy (auto-approved, blocked, etc.)
	}
	approvalSvc := approval.NewService(o.db, o.logger, nil)
	_, appErr := approvalSvc.Create(&approval.CreateApprovalInput{
		ProductID:   0,
		RequestType: a.ActionType,
		Requester:   "agent:" + a.AgentID,
		Reason:      a.Title,
		RiskLevel:   a.RiskLevel,
		EntityType:  "unified_action",
		EntityID:    a.ID,
	})
	if appErr != nil {
		o.logger.Warn("failed to create approval for action",
			zap.Int64("action_id", a.ID),
			zap.Error(appErr))
		return
	}
	o.logger.Info("created approval for action",
		zap.Int64("action_id", a.ID),
		zap.String("action_type", a.ActionType))
}

// synthesizeOutput produces the agent's final output. It checks for a concrete
// agent implementation first, and falls back to the LLM provider or deterministic
// stub when no implementation is registered.
func (o *Orchestrator) synthesizeOutput(ctx context.Context, agent AgentSpec, dp string, params map[string]interface{}) (map[string]interface{}, float64, string, error) {
	// Check if there is a concrete implementation for this agent.
	if implAgent, ok := o.agentImpls[agent.ID]; ok {
		o.logger.Debug("using concrete agent implementation",
			zap.String("agent_id", agent.ID),
			zap.String("decision_point", dp),
		)
		return implAgent.Decide(ctx, dp, params)
	}

	// Build the prompt.
	system := fmt.Sprintf("You are %s (%s), a LingMirror agent in the %s squad. Decision point: %s. Description: %s. Respond in Chinese, be concise (<=120 chars).",
		agent.ID, agent.Name, agent.Squad, dp, agent.Description)
	userMsg := fmt.Sprintf("Agent %s @ %s — please decide.", agent.ID, dp)
	if params != nil {
		if m, ok := params["message"].(string); ok && m != "" {
			userMsg = m
		}
	}
	// Run guardrails on input before sending to LLM.
	if o.guardrails != nil {
		inp := &guardrails.GuardInput{RawInput: userMsg}
		res, err := o.guardrails.Check(context.Background(), inp)
		if err != nil {
			o.logger.Warn("guardrails input check failed", zap.Error(err))
		} else if res.Blocked {
			o.logger.Warn("guardrails blocked LLM input", zap.String("reason", res.Reason))
			userMsg = fmt.Sprintf("[input blocked: %s] %s", res.Reason, userMsg)
		}
	}
	req := &LLMRequest{
		Model:    agent.ModelHint,
		System:   system,
		Messages: []LLMMessage{{Role: "user", Content: userMsg}},
		Metadata: map[string]interface{}{"agent_id": agent.ID, "decision_point": dp},
	}

	// Budget check — can block or force downgrade to cheapest model.
	if o.budget != nil {
		// Monthly budget hard limit — check llm_budgets table first.
		var mb struct {
			MonthlyLimitUSD float64
			CurrentMonthUSD float64
			IsPaused        bool
			BudgetMonth     string
		}
		if err := o.db.Table("llm_budgets").Select("monthly_limit_usd, current_month_usd, is_paused, budget_month").First(&mb).Error; err == nil {
			currentMonth := time.Now().Format("2006-01")
			if mb.MonthlyLimitUSD > 0 {
				observedSpend := 0.0
				if mb.BudgetMonth == currentMonth {
					observedSpend = mb.CurrentMonthUSD
				}
				if spend, err := o.monthlyLLMSpendUSD(currentMonth); err == nil && spend > observedSpend {
					observedSpend = spend
				}
				if mb.IsPaused || observedSpend >= mb.MonthlyLimitUSD {
					o.logger.Warn("monthly LLM budget exceeded",
						zap.String("agent", agent.ID),
						zap.Float64("current", observedSpend),
						zap.Float64("limit", mb.MonthlyLimitUSD),
						zap.Bool("paused", mb.IsPaused),
					)
					out, conf, risk := stubFinalOutput(agent, dp, params)
					return out, conf, risk, nil
				}
			}
		}

		budgetIn := costcontrol.AllowInput{AgentID: agent.ID, Model: req.Model, Tokens: len(userMsg) / 4}
		budgetRes, budgetErr := o.budget.Allow(context.Background(), budgetIn)
		if budgetErr == nil && budgetRes.Action == costcontrol.ActionBlock {
			o.logger.Warn("budget blocked LLM call",
				zap.String("agent", agent.ID),
				zap.String("reason", budgetRes.Reason),
				zap.Float64("daily_spent", budgetRes.DailySpent),
			)
			out, conf, risk := stubFinalOutput(agent, dp, params)
			return out, conf, risk, nil
		}
		if budgetErr == nil && budgetRes.Action == costcontrol.ActionDowngrade {
			o.logger.Info("budget downgraded LLM model",
				zap.String("agent", agent.ID),
				zap.String("from", req.Model),
				zap.String("to", budgetRes.Cheapest),
			)
			req.Model = budgetRes.Cheapest
		}
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
			o.recordLLMCost(agent, req, resp, 0, false)
			// In production, do NOT silently fall back to stub output.
			envVal := strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
			ginVal := strings.ToLower(strings.TrimSpace(os.Getenv("GIN_MODE")))
			if envVal == "production" || ginVal == "release" {
				return nil, 0, "", fmt.Errorf("LLM provider call failed: %w", err)
			}
		} else if resp != nil {
			o.recordLLMCost(agent, req, resp, 0, false)

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
			if params != nil {
				out["echoed_context"] = params
			}
			confidence := 0.82
			if resp.TokensOut > 0 {
				confidence = clampConfidence(0.65 + float64(resp.TokensOut)/600.0)
			}

			// Validate LLM output through guardrails chain.
			if o.guardrails != nil {
				inp := &guardrails.GuardInput{RawOutput: resp.Answer}
				res, err := o.guardrails.Check(context.Background(), inp)
				if err != nil {
					o.logger.Warn("guardrails output check failed", zap.Error(err))
				} else if res.Blocked {
					o.logger.Warn("guardrails blocked LLM output",
						zap.String("reason", res.Reason),
						zap.String("risk", res.Risk),
					)
					envVal := strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
					ginVal := strings.ToLower(strings.TrimSpace(os.Getenv("GIN_MODE")))
					if envVal == "production" || ginVal == "release" {
						return nil, 0, "", fmt.Errorf("LLM output blocked by guardrails: %s", res.Reason)
					}
					// In non-production, fall back to stub output.
					out, conf, risk := stubFinalOutput(agent, dp, params)
					return out, conf, risk, nil
				}
				if !res.Pass {
					o.logger.Warn("guardrails warning on LLM output",
						zap.String("reason", res.Reason),
						zap.String("risk", res.Risk),
					)
				}
			}
			return out, confidence, agent.RiskFloor, nil
		}
	}
	out, conf, risk := stubFinalOutput(agent, dp, params)
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

func (o *Orchestrator) monthlyLLMSpendUSD(month string) (float64, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return 0, err
	}
	end := start.AddDate(0, 1, 0)
	var total float64
	err = o.db.Model(&costcontrol.CostLog{}).
		Select("COALESCE(SUM(cost_usd),0)").
		Where("window_date >= ? AND window_date < ?", start, end).
		Scan(&total).Error
	return total, err
}

// recordLLMCost writes the cost of an LLM call to the budget controller.
// resp may be nil (failed calls); cached indicates whether the response was a cache hit.
func (o *Orchestrator) recordLLMCost(agent AgentSpec, req *LLMRequest, resp *LLMResponse, userID int64, cached bool) {
	if o.budget == nil {
		return
	}
	if cached || resp == nil {
		return
	}
	// Estimate cost from token counts (Haiku ≈ $0.25/M, Sonnet ≈ $3/M, Opus ≈ $15/M input).
	var rateIn, rateOut float64
	m := req.Model
	switch {
	case strings.Contains(m, "haiku"):
		rateIn, rateOut = 0.25, 1.25
	case strings.Contains(m, "sonnet"):
		rateIn, rateOut = 3.0, 15.0
	case strings.Contains(m, "opus"):
		rateIn, rateOut = 15.0, 75.0
	default:
		rateIn, rateOut = 3.0, 15.0 // sonnet default
	}
	cost := (float64(resp.TokensIn)/1_000_000)*rateIn + (float64(resp.TokensOut)/1_000_000)*rateOut
	rec := costcontrol.RecordInput{
		UserID:    userID,
		AgentID:   agent.ID,
		Model:     resp.Model,
		TokensIn:  resp.TokensIn,
		TokensOut: resp.TokensOut,
		CostUSD:   cost,
		Cached:    cached,
	}
	_ = o.budget.Record(context.Background(), rec)
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
	// first_km_scout: "调研" + target market → full conversational workflow.
	// Placed before product_research so it wins when market context is present.
	// Keywords are market-specific only — "调研" is NOT here so plain "调研家居"
	// falls through to product_research. When market keywords are present,
	// first_km_scout wins by having more matches (research + market).
	{Keywords: []string{"俄罗斯", "russia", "ozon", "wildberries", "目标市场", "target market", "美国", "usa", "amazon", "日本", "japan", "shopee", "lazada"}, AgentID: "A1", DecisionPoint: "first_km_scout"},
	{Keywords: []string{"调研", "research", "品类调研", "研究方向", "research category"}, AgentID: "A1", DecisionPoint: "product_research"},
	{Keywords: []string{"选品", "新品", "市场", "scout", "product"}, AgentID: "A1", DecisionPoint: "product_scout"},
	{Keywords: []string{"供应商", "货源", "1688", "supplier", "供应链", "source product", "采集页面"}, AgentID: "A1", DecisionPoint: "supplier_discovery"},
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

// decisionCache caches agent decision results with a TTL.
type decisionCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	result    *RunAgentResult
	expiresAt time.Time
}

func newDecisionCache(ttl time.Duration) *decisionCache {
	return &decisionCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

func (c *decisionCache) cacheKey(agentID, decisionPoint string, ctx map[string]interface{}) string {
	h := sha256.New()
	h.Write([]byte(agentID))
	h.Write([]byte{0})
	h.Write([]byte(decisionPoint))
	if ctx != nil {
		keys := make([]string, 0, len(ctx))
		for k := range ctx {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.Write([]byte(k))
			h.Write([]byte{'='})
			switch v := ctx[k].(type) {
			case string:
				h.Write([]byte(v))
			case float64:
				fmt.Fprintf(h, "%f", v)
			default:
				b, _ := json.Marshal(v)
				h.Write(b)
			}
			h.Write([]byte{'&'})
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (c *decisionCache) set(agentID, decisionPoint string, ctx map[string]interface{}, result *RunAgentResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.cacheKey(agentID, decisionPoint, ctx)
	c.entries[key] = &cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *decisionCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}
