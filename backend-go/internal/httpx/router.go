package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/agent"
	"github.com/lingmirror/backend-go/internal/agent/pipeline"
	"github.com/lingmirror/backend-go/internal/agentos"
	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/aios/costcontrol"
	"github.com/lingmirror/backend-go/internal/aios/setup"
	"github.com/lingmirror/backend-go/internal/aios/toolregistry/tools"
	"github.com/lingmirror/backend-go/internal/auth"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/domain/actionpolicy"
	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/domain/agentlearning"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/agentrule"
	"github.com/lingmirror/backend-go/internal/domain/allocation"
	"github.com/lingmirror/backend-go/internal/domain/brand"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/category"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/consolidation"
	"github.com/lingmirror/backend-go/internal/domain/cost"
	"github.com/lingmirror/backend-go/internal/domain/landedcost"
	"github.com/lingmirror/backend-go/internal/domain/orchestration"
	"github.com/lingmirror/backend-go/internal/domain/personalrule"
	"github.com/lingmirror/backend-go/internal/domain/producthub"
	"github.com/lingmirror/backend-go/internal/domain/sentiment"
	"github.com/lingmirror/backend-go/internal/domain/dashboard"
	"github.com/lingmirror/backend-go/internal/domain/decision"
	"github.com/lingmirror/backend-go/internal/domain/entropy"
	"github.com/lingmirror/backend-go/internal/domain/evolution"
	"github.com/lingmirror/backend-go/internal/domain/exceptions"
	"github.com/lingmirror/backend-go/internal/domain/exchangerate"
	"github.com/lingmirror/backend-go/internal/domain/finance"
	"github.com/lingmirror/backend-go/internal/domain/imagegen"
	"github.com/lingmirror/backend-go/internal/domain/importbatch"
	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"github.com/lingmirror/backend-go/internal/domain/listing"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/logistics"
	"github.com/lingmirror/backend-go/internal/domain/loop"
	"github.com/lingmirror/backend-go/internal/domain/metabolism"
	"github.com/lingmirror/backend-go/internal/domain/mock"
	"github.com/lingmirror/backend-go/internal/domain/notification"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/orderimport"
	"github.com/lingmirror/backend-go/internal/domain/owner"
	"github.com/lingmirror/backend-go/internal/domain/platform"
	"github.com/lingmirror/backend-go/internal/domain/platformfee"
	"github.com/lingmirror/backend-go/internal/domain/price"
	"github.com/lingmirror/backend-go/internal/domain/productanalysis"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"github.com/lingmirror/backend-go/internal/domain/purchase"
	"github.com/lingmirror/backend-go/internal/domain/report"
	"github.com/lingmirror/backend-go/internal/domain/search"
	"github.com/lingmirror/backend-go/internal/domain/settings"
	"github.com/lingmirror/backend-go/internal/domain/settlement"
	"github.com/lingmirror/backend-go/internal/domain/shipping"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"github.com/lingmirror/backend-go/internal/domain/sourcing"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"github.com/lingmirror/backend-go/internal/domain/supplier"
	"github.com/lingmirror/backend-go/internal/domain/supplychain"
	"github.com/lingmirror/backend-go/internal/domain/tariff"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/platform/scheduler"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"github.com/lingmirror/backend-go/internal/prismadapter"
	"github.com/lingmirror/backend-go/internal/rbac"
	"github.com/lingmirror/backend-go/internal/realtime"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewRouter creates and configures the Gin engine with all routes.
func NewRouter(db *gorm.DB, cfg *config.Config, logger *zap.Logger) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// Global middleware
	r.Use(middleware.CORS(cfg))
	r.Use(middleware.RequestID())

	// Prometheus metrics (before recovery to capture all requests)
	if cfg.Metrics.Enabled {
		r.Use(middleware.Metrics())
	}
	r.Use(middleware.RecoveryWithSentry(cfg, logger))
	r.Use(middleware.Audit(db, logger))

	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "0.1.0",
		})
	})

	// Prometheus metrics endpoint
	if cfg.Metrics.Enabled {
		r.GET("/metrics", middleware.MetricsHandler())
	}

	// ==========================================================
	// Phase 1 Infrastructure: Event Bus + Command + Scheduler
	// ==========================================================

	// Create event bus (with optional outbox persistence).
	bus := eventbus.New(logger, eventbus.WithDB(db), eventbus.WithWorkers(4))
	busCtx, busCancel := context.WithCancel(context.Background())
	defer busCancel()
	bus.Start(busCtx)

	// Initialize platform adapters (Ozon, Shopee, etc.).
	integrations.InitAdapters(db, logger)

	// Create command dispatcher and register Phase 1 handlers.
	cmd := command.NewDispatcher(logger)
	cmd.Register("stock_alert", command.StockAlertHandler(db, logger))
	cmd.Register("replenish", command.InventoryReplenishHandler(db, logger))
	cmd.Register("price_review", command.PriceAdjustHandler(db, logger))
	cmd.Register("listing_optimize", command.ListingDraftHandler(db, logger))
	cmd.Register("compliance_check", command.FlagNonCompliantHandler(db, logger))

	// Create scheduler with all registered tasks.
	sched := scheduler.New(bus, logger)

	// AI orchestrator (shared by /ai and /agents routes).
	aiosCfg := setup.Initialize(db, bus, logger)
	budget := costcontrol.NewController(db, logger, cfg.LLM.DailyBudgetUSD, 5*time.Minute, 3.0)
	aiOrch := ai.NewOrchestrator(db, logger).WithGuardrails(aiosCfg.Guardrails).WithBudget(budget).WithBus(bus)

	// Pipeline engine for declarative decision DAG (replaces inline chain handlers).
	pipelineEng := pipeline.NewEngine(func(ctx context.Context, agentID, dp string, ctxMap map[string]interface{}) error {
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := aiOrch.RunWithContext(timeoutCtx, &ai.RunAgentRequest{
			AgentID:       agentID,
			DecisionPoint: dp,
			Context:       ctxMap,
		})
		return err
	}, pipeline.DefaultEdges, logger)

	// ToolBridge for sourcing data collection
	toolBridge := toolbridge.NewToolBridge(nil, 0, logger.Named("toolbridge")) // drivers registered later

	// Reverse logistics return rate tracker (DB-backed).
	returnRateTracker := aftersales.NewReturnRateTracker(db, logger)

	// Supply chain orchestrator (bridges A8 sourcing with A10 logistics quoting,
	// and handles reverse logistics via HandleAftersaleReturn).
	supplyChainOrch := supplychain.NewOrchestrator(bus, db, logger, returnRateTracker)

	// ==========================================================
	// Event Bus Subscriptions: agent triggers + pipeline chains
	// ==========================================================

	// scheduler.tick.A5 → orchestrator runs A5 stock_alert
	bus.Subscribe("scheduler.tick.A5", func(ctx context.Context, evt eventbus.Event) error {
		_, err := aiOrch.Run(&ai.RunAgentRequest{
			AgentID:       "A5",
			DecisionPoint: "stock_alert",
			Context:       evt.Payload,
		})
		return err
	})
	// scheduler.tick.G0 → orchestrator runs G0 system_health
	bus.Subscribe("scheduler.tick.G0", func(ctx context.Context, evt eventbus.Event) error {
		_, err := aiOrch.Run(&ai.RunAgentRequest{
			AgentID:       "G0",
			DecisionPoint: "system_health",
			Context:       evt.Payload,
		})
		return err
	})
	// scheduler.tick.A6 → A6 profit_watch
	bus.Subscribe("scheduler.tick.A6", func(ctx context.Context, evt eventbus.Event) error {
		_, err := aiOrch.Run(&ai.RunAgentRequest{
			AgentID:       "A6",
			DecisionPoint: "profit_watch",
			Context:       evt.Payload,
		})
		return err
	})
	// scheduler.tick.G3 → G3 discount_risk_check
	bus.Subscribe("scheduler.tick.G3", func(ctx context.Context, evt eventbus.Event) error {
		_, err := aiOrch.Run(&ai.RunAgentRequest{
			AgentID:       "G3",
			DecisionPoint: "discount_risk_check",
			Context:       evt.Payload,
		})
		return err
	})
	// scheduler.tick.A7 → A7 compliance_check
	bus.Subscribe("scheduler.tick.A7", func(ctx context.Context, evt eventbus.Event) error {
		_, err := aiOrch.Run(&ai.RunAgentRequest{
			AgentID:       "A7",
			DecisionPoint: "compliance_check",
			Context:       evt.Payload,
		})
		return err
	})
	// scheduler.tick.A4 → A4 auto_reply
	bus.Subscribe("scheduler.tick.A4", func(ctx context.Context, evt eventbus.Event) error {
		_, err := aiOrch.Run(&ai.RunAgentRequest{
			AgentID:       "A4",
			DecisionPoint: "auto_reply",
			Context:       evt.Payload,
		})
		return err
	})
	// scheduler.tick.G2 → G2 warehouse_routing
	bus.Subscribe("scheduler.tick.G2", func(ctx context.Context, evt eventbus.Event) error {
		_, err := aiOrch.Run(&ai.RunAgentRequest{
			AgentID:       "G2",
			DecisionPoint: "warehouse_routing",
			Context:       evt.Payload,
		})
		return err
	})
	// scheduler.tick.A3 → A3 acos_analysis
	bus.Subscribe("scheduler.tick.A3", func(ctx context.Context, evt eventbus.Event) error {
		_, err := aiOrch.Run(&ai.RunAgentRequest{
			AgentID:       "A3",
			DecisionPoint: "acos_analysis",
			Context:       evt.Payload,
		})
		return err
	})
	// scheduler.tick.trustscore → recalculate trust scores + auto-upgrade eligible agents
	bus.Subscribe("scheduler.tick.trustscore", func(ctx context.Context, evt eventbus.Event) error {
		ug := trustscore.NewUpgrader(db, logger)
		_, err := ug.AutoUpgrade()
		return err
	})
	// scheduler.tick.entropy → run entropy defenses + agent health check
	bus.Subscribe("scheduler.tick.entropy", func(ctx context.Context, evt eventbus.Event) error {
		svc := entropy.NewService(db, logger)
		_, err := svc.RunDefenses(0)

		// Check agent health and publish unhealthy agents to the bus.
		unhealthy, healthErr := svc.CheckAgentHealth()
		if healthErr != nil {
			logger.Warn("entropy: agent health check failed", zap.Error(healthErr))
		} else {
			for _, ua := range unhealthy {
				logger.Warn("entropy: agent health below threshold",
					zap.String("agent_id", ua.AgentID),
					zap.Float64("health_score", ua.HealthScore),
				)
				bus.Publish(ctx, "entropy.agent.unhealthy."+ua.AgentID, "entropy", map[string]interface{}{
					"agent_id":     ua.AgentID,
					"health_score": ua.HealthScore,
				})
			}
		}

		return err
	})

	// scheduler.tick.A8 → sourcing batch scan
	bus.Subscribe("scheduler.tick.A8", func(ctx context.Context, evt eventbus.Event) error {
		return sourcing.HandleSourcingTick(db, logger)(ctx, evt)
	})

	// -------------------------------------------------------
	// Declarative Pipeline DAG (replaces inline chain handlers).
	// All pipeline edges are defined in pipeline.DefaultEdges and evaluated
	// by pipelineEng.Dispatch. This wildcard subscription catches all
	// agent.decided.* events and routes them through the DAG engine.
	// -------------------------------------------------------

	// Wildcard DAG subscription — routes all agent decisions through the pipeline engine.
	bus.Subscribe("agent.decided.*", func(ctx context.Context, evt eventbus.Event) error {
		return pipelineEng.Dispatch(ctx, evt)
	})

	// A5 stock_alert (red) → G3 discount_risk_check
	bus.Subscribe("agent.decided.A5.stock_alert", func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		if status, ok := payload["stock_status"].(string); ok && status == "red" {
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer timeoutCancel()
			traceID, _ := payload["trace_id"].(string)
			_, err := aiOrch.RunWithContext(timeoutCtx, &ai.RunAgentRequest{
				AgentID:       "G3",
				DecisionPoint: "discount_risk_check",
				Context:       payload,
				ParentTraceID: traceID,
			})
			return err
		}
		return nil
	})

	// A6 profit_watch (loss) → A2 listing_optimize
	bus.Subscribe("agent.decided.A6.profit_watch", func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		isLoss, _ := payload["is_loss"].(bool)
		belowThreshold, _ := payload["below_threshold"].(bool)
		if isLoss || belowThreshold {
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer timeoutCancel()
			traceID, _ := payload["trace_id"].(string)
			_, err := aiOrch.RunWithContext(timeoutCtx, &ai.RunAgentRequest{
				AgentID:       "A2",
				DecisionPoint: "listing_optimize",
				Context:       payload,
				ParentTraceID: traceID,
			})
			return err
		}
		return nil
	})

	// G0 system_health (anomaly_count > 3) → G1 dashboard_overview
	bus.Subscribe("agent.decided.G0.system_health", func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		if count, ok := payload["anomaly_count"].(int); ok && count > 3 {
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer timeoutCancel()
			traceID, _ := payload["trace_id"].(string)
			_, err := aiOrch.RunWithContext(timeoutCtx, &ai.RunAgentRequest{
				AgentID:       "G1",
				DecisionPoint: "dashboard_overview",
				Context:       payload,
				ParentTraceID: traceID,
			})
			return err
		}
		// Also check for float64 (JSON unmarshal from scheduler)
		if count, ok := payload["anomaly_count"].(float64); ok && count > 3 {
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer timeoutCancel()
			traceID, _ := payload["trace_id"].(string)
			_, err := aiOrch.RunWithContext(timeoutCtx, &ai.RunAgentRequest{
				AgentID:       "G1",
				DecisionPoint: "dashboard_overview",
				Context:       payload,
				ParentTraceID: traceID,
			})
			return err
		}
		return nil
	})

	// -------------------------------------------------------
	// Agent Decision Audit Chain: log every agent.decided.*
	// event to the operation_log for the full audit trail.
	// -------------------------------------------------------
	auditSvc := operationlog.NewService(db, logger)
	bus.Subscribe("agent.decided.**", func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload

		if len(payload) == 0 {
			logger.Warn("agent.decided event with empty payload, skipping audit",
				zap.String("topic", evt.Topic))
			return nil
		}

		// Chain: if action is "block", trigger A6 profit_watch proactively.
		if action, ok := payload["action"].(string); ok && action == "block" {
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer timeoutCancel()
			traceID, _ := payload["trace_id"].(string)
			_, err := aiOrch.RunWithContext(timeoutCtx, &ai.RunAgentRequest{
				AgentID:       "A6",
				DecisionPoint: "profit_watch",
				Context:       payload,
				ParentTraceID: traceID,
			})
			if err != nil {
				logger.Warn("agent decision chain: A6 profit_watch triggered via block action",
					zap.Error(err))
			}
		}

		// Extract agent_id and decision_point from payload or topic.
		agentID, _ := payload["agent_id"].(string)
		decisionPoint, _ := payload["decision_point"].(string)
		if agentID == "" || decisionPoint == "" {
			// Parse from topic: agent.decided.{agent_id}.{decision_point}
			parts := strings.SplitN(evt.Topic, ".", 4)
			if len(parts) >= 4 {
				if agentID == "" {
					agentID = parts[2]
				}
				if decisionPoint == "" {
					decisionPoint = parts[3]
				}
			}
		}

		// Extract audit-relevant fields.
		confidence, _ := payload["confidence"].(float64)
		riskLevel, _ := payload["risk_level"].(string)
		status, _ := payload["status"].(string)
		if riskLevel == "" {
			// Some agents use risk instead of risk_level.
			if r, ok := payload["risk"].(string); ok {
				riskLevel = r
			}
		}

		// Build a compact content JSON summarizing the decision.
		contentParts := []string{}
		if status != "" {
			contentParts = append(contentParts, "status:"+status)
		}
		if riskLevel != "" {
			contentParts = append(contentParts, "risk:"+riskLevel)
		}
		if action, ok := payload["action"].(string); ok {
			contentParts = append(contentParts, "action:"+action)
		}
		contentSummary := strings.Join(contentParts, " ")
		if contentSummary == "" {
			contentSummary = agentID + "/" + decisionPoint
		}

		// Marshal the full payload as content JSON for audit records.
		payloadBytes, _ := json.Marshal(payload)
		content := string(payloadBytes)
		if len(content) > 2048 {
			content = content[:2048]
		}

		entry := &operationlog.OperationLog{
			Module:     fmt.Sprintf("agent:%s", agentID),
			Action:     "agent_decision",
			ResourceID: decisionPoint,
			Content:    content,
			Operator:   agentID,
			Result:     "success",
			Duration:   0,
		}
		if err := auditSvc.Create(entry); err != nil {
			logger.Warn("agent decision audit log write failed",
				zap.String("topic", evt.Topic),
				zap.Error(err))
		}
		logger.Debug("agent decision audit logged",
			zap.String("agent_id", agentID),
			zap.String("decision_point", decisionPoint),
			zap.Float64("confidence", confidence),
			zap.String("risk_level", riskLevel),
			zap.String("summary", contentSummary))
		return nil
	})

	// ==========================================================
	// Schedule all agent periodic tasks
	// ==========================================================

	sched.Register(scheduler.Task{
		ID: "tick-g0", AgentID: "G0", DecisionPoint: "system_health",
		Interval: time.Minute * 5, Description: "协调仲裁健康检查",
	})
	sched.Register(scheduler.Task{
		ID: "tick-a4", AgentID: "A4", DecisionPoint: "auto_reply",
		Interval: time.Minute * 5, Description: "客服待处理消息检查",
	})
	sched.Register(scheduler.Task{
		ID: "tick-g1", AgentID: "G1", DecisionPoint: "dashboard_overview",
		Interval: time.Minute * 5, Description: "驾驶舱聚合",
	})
	sched.Register(scheduler.Task{
		ID: "tick-a5", AgentID: "A5", DecisionPoint: "stock_alert",
		Interval: time.Minute * 15, Description: "库存检查",
	})
	sched.Register(scheduler.Task{
		ID: "tick-g3", AgentID: "G3", DecisionPoint: "discount_risk_check",
		Interval: time.Minute * 30, Description: "折扣风控扫描",
	})
	sched.Register(scheduler.Task{
		ID: "tick-a6", AgentID: "A6", DecisionPoint: "profit_watch",
		Interval: time.Hour * 1, Description: "利润看护",
	})
	sched.Register(scheduler.Task{
		ID: "tick-a3", AgentID: "A3", DecisionPoint: "acos_analysis",
		Interval: time.Hour * 1, Description: "广告分析",
	})
	sched.Register(scheduler.Task{
		ID: "tick-g2", AgentID: "G2", DecisionPoint: "warehouse_routing",
		Interval: time.Hour * 1, Description: "仓储报关",
	})
	sched.Register(scheduler.Task{
		ID: "tick-a7", AgentID: "A7", DecisionPoint: "compliance_check",
		Interval: time.Hour * 2, Description: "合规检测",
	})
	sched.Register(scheduler.Task{
		ID: "tick-trustscore", AgentID: "trustscore", DecisionPoint: "recalculate",
		Interval: time.Hour * 1, Description: "信任分重算",
	})
	sched.Register(scheduler.Task{
		ID: "tick-entropy", AgentID: "entropy", DecisionPoint: "defend",
		Interval: time.Hour * 6, Description: "熵防御周期",
	})
	sched.Register(scheduler.Task{
		ID: "tick-ozon-sync", AgentID: "ozon_sync", DecisionPoint: "sync_orders",
		Interval: time.Minute * 15, Description: "Ozon 订单同步",
	})
	sched.Register(scheduler.Task{
		ID: "tick-a8", AgentID: "A8", DecisionPoint: "sourcing_scan",
		Interval: time.Hour * 1, Description: "选品扫描",
	})
	sched.Register(scheduler.Task{
		ID: "tick-m1", AgentID: "M1", DecisionPoint: "excretion_scoring",
		Interval: time.Hour * 1, Description: "代谢排泄评分",
	})
	sched.Register(scheduler.Task{
		ID: "tick-sla-escalation", AgentID: "agentos", DecisionPoint: "sla_escalation",
		Interval: time.Minute * 15, Description: "SLA过期升级待审批动作",
	})

	// Ozon sync handler
	bus.Subscribe("scheduler.tick.ozon_sync", func(ctx context.Context, evt eventbus.Event) error {
		integrations.InitAdapters(db, logger)
		svc := integrations.NewService(db, logger)
		return svc.SyncOzonOrders(ctx)
	})

	// Start scheduler in background goroutine.
	go sched.Start(busCtx)

	// ==========================================================
	// HTTP routes
	// ==========================================================

	// API v1 routes
	api := r.Group("/api/v1")

	// API v1 Health check (public)
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "0.1.0",
		})
	})

	// Auth routes (public — login, register, refresh)
	auth.RegisterRoutes(api, db, cfg, logger)

	// Protected routes (require JWT authentication)
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg))

	// RBAC routes — require rbac.manage permission
	rbacRoutes := protected.Group("", middleware.RequirePermission(db, "rbac.manage"))
	rbac.RegisterRoutes(rbacRoutes, db, logger)

	// Agent routes (wired through the AI orchestrator)
	agent.RegisterRoutes(protected, db, logger, aiOrch)

	// AgentOS routes
	agentos.RegisterRoutes(protected, db, logger)

	// Domain routes (all require authentication)
	category.RegisterRoutes(protected, db, logger)
	brand.RegisterRoutes(protected, db, logger)
	sku.RegisterRoutes(protected, db, logger)
	inventory.RegisterRoutes(protected, db, logger)
	supplier.RegisterRoutes(protected, db, logger)
	purchase.RegisterRoutes(protected, db, logger, bus)

	// Supply chain event: purchase order received → auto-increment inventory
	bus.Subscribe("supplychain.order.received", func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		invSvc := inventory.NewService(db, logger)
		items, ok := payload["items"].([]interface{})
		if !ok {
			return nil
		}
		orderNo, _ := payload["order_no"].(string)
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			skuID := int64(m["sku_id"].(float64))
			qty := int(m["qty"].(float64))
			inv, err := invSvc.GetBySkuID(ctx, skuID)
			if err != nil {
				logger.Warn("supplychain: inventory not found for sku", zap.Int64("sku_id", skuID), zap.Error(err))
				continue
			}
			_ = invSvc.UpdateStock(ctx, inv.ID, inv.Quantity+qty, "system", "采购入库: "+orderNo)
		}
		return nil
	})
	// sourcing.recommend (A8) -> A2 listing_optimize for high-score products
	bus.Subscribe("sourcing.recommend", func(ctx context.Context, evt eventbus.Event) error {
		// Process recommendation through the handler (logging, etc.)
		_ = sourcing.HandleSourcingRecommend(db, logger)(ctx, evt)
		score, _ := evt.Payload["score"].(int)
		scoreFloat, _ := evt.Payload["score"].(float64)
		if score >= 7 || scoreFloat >= 7 {
			return runAgentWithTimeout(aiOrch, "A2", "listing_optimize", evt.Payload)
		}
		return nil
	})

	// sourcing.recommend → orchestrator triggers A10 shipping quoting
	bus.Subscribe("sourcing.recommend", func(ctx context.Context, evt eventbus.Event) error {
		return supplyChainOrch.HandleRecommendEvent(ctx, evt)
	})
	// scheduler.tick.orch → supply chain orchestrator heartbeat (no-op)
	bus.Subscribe("scheduler.tick.orch", func(ctx context.Context, evt eventbus.Event) error {
		return supplyChainOrch.HandleTick(ctx, evt)
	})

	// Supply chain event: after-sale completed → auto-adjust inventory
	bus.Subscribe("supplychain.aftersale.completed", func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		invSvc := inventory.NewService(db, logger)
		skuID := int64(payload["sku_id"].(float64))
		qty := int(payload["quantity"].(float64))
		inv, err := invSvc.GetBySkuID(ctx, skuID)
		if err != nil {
			logger.Warn("supplychain: inventory not found for sku in aftersale", zap.Int64("sku_id", skuID), zap.Error(err))
			return nil
		}
		// Aftersales restock adds stock back to inventory
		_ = invSvc.UpdateStock(ctx, inv.ID, inv.Quantity+qty, "system", "售后入库")
		return nil
	})

	// Supply chain event: after-sale return initiated -> create reverse logistics flow
	bus.Subscribe("supplychain.aftersale.returned", func(ctx context.Context, evt eventbus.Event) error {
		return supplyChainOrch.HandleAftersaleReturn(ctx, evt)
	})

	// -------------------------------------------------------
	// Data flywheel: fulfillment data backflow (Issue #5)
	// -------------------------------------------------------

	// Shared logistics service for in-memory performance stats.
	flywheelLogSvc := logistics.NewService(nil)

	// supplychain.flywheel → A10: update carrier_performance
	bus.Subscribe("supplychain.flywheel", flywheelLogSvc.HandleFlywheelEvent())

	// supplychain.flywheel → A8: update category_performance
	bus.Subscribe("supplychain.flywheel", flywheelLogSvc.HandleCategoryFlywheelEvent())

	// Supply chain event: stock critical (A5) → orchestrator triggers A8 sourcing rescan
	bus.Subscribe("supplychain.stock.critical", func(ctx context.Context, evt eventbus.Event) error {
		return supplyChainOrch.HandleStockCritical(ctx, evt)
	})

	platform.RegisterRoutes(protected, db, logger)
	listing.RegisterRoutes(protected, db, logger)

	// Initialize Prism client (config-driven; nil if disabled).
	var prismSvc prismadapter.PrismService
	prismStrict := cfg.Prism.Strict
	if cfg.Prism.Enabled && cfg.Prism.BaseURL != "" {
		timeout := time.Duration(cfg.Prism.Timeout) * time.Second
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		prismSvc = prismadapter.NewClient(cfg.Prism.BaseURL, cfg.Prism.APIKey, timeout)
		logger.Info("Prism client initialized", zap.String("base_url", cfg.Prism.BaseURL), zap.Bool("strict", prismStrict))
	} else {
		logger.Info("Prism client disabled")
	}
	listingtask.RegisterRoutes(protected, db, logger, prismSvc, prismStrict)

	candidate.RegisterRoutes(protected, db, logger)
	completeness.RegisterRoutes(protected, db, logger)
	profit.RegisterRoutes(protected, db, logger)
	loop.RegisterRoutes(protected, db, logger, prismSvc, prismStrict)
	mock.RegisterRoutes(protected, db, logger)
	// Auto-seed mock demo data on startup
	func() {
		ms := mock.NewService(db, logger.Named("mock"))
		if err := ms.SeedMockData(); err != nil {
			logger.Warn("mock data seed failed", zap.Error(err))
		}
	}()

	owner.RegisterRoutes(protected, db, logger)
	agentlearning.RegisterRoutes(protected, db, logger)
	approval.RegisterRoutes(protected, db, logger)
	landedcost.RegisterRoutes(protected, db, logger)
	orchestration.RegisterRoutes(protected, db, bus, aiOrch, logger)
	personalrule.RegisterRoutes(protected, db, logger)
	producthub.RegisterRoutes(protected, db, logger)
	sentiment.RegisterRoutes(protected, db, logger)

	shipping.RegisterRoutes(protected, db, logger)
	platformfee.RegisterRoutes(protected, db, logger)
	order.RegisterRoutes(protected, db, logger)
	orderimport.RegisterRoutes(protected, db, logger)
	settlement.RegisterRoutes(protected, db, logger)
	financeRoutes := protected.Group("", middleware.RequirePermission(db, "finance.read"))
	finance.RegisterRoutes(financeRoutes, db, logger)
	price.RegisterRoutes(financeRoutes, db, logger)
	decision.RegisterRoutes(protected, db, logger)
	allocation.RegisterRoutes(protected, db, logger)
	exceptions.RegisterRoutes(protected, db, logger)
	notification.RegisterRoutes(protected, db, logger)
	dashboard.RegisterRoutes(protected, db, logger)
	search.RegisterRoutes(protected, db, logger)
	settings.RegisterRoutes(protected, db, logger)
	imagegen.RegisterRoutes(protected, db, logger)
	importbatch.RegisterRoutes(protected, db, logger)
	operationlog.RegisterRoutes(protected, db, logger)
	integrations.RegisterRoutes(protected, db, logger)
	actionpolicy.RegisterRoutes(protected, db, logger)
	aftersales.RegisterRoutes(protected, db, logger, bus)
	sourcing.RegisterRoutes(protected, db, logger, toolBridge, sourcing.NewAgentEventPublisher(bus))
	sourcing1688.RegisterRoutes(protected, db, logger)
	tariff.RegisterRoutes(protected, db, logger)
	logistics.RegisterRoutes(protected, db, logger)
	consolidation.RegisterRoutes(protected, db, logger)
	// Wire the loaded carrier-rate engine into the sourcing profit-calc tool
	// so sourcing.recommend uses real shipping rates instead of the static
	// fallback map. No-op if DefaultEngine is nil (no YAML files loaded).
	tools.SetSourcingEngine(logistics.DefaultEngine)
	supplychain.RegisterRoutes(protected, db, logger, bus)
	productanalysis.RegisterRoutes(protected, db, logger)
	trustscore.RegisterRoutes(protected, db, logger)
	reportRoutes := protected.Group("", middleware.RequirePermission(db, "report.read"))
	report.RegisterRoutes(reportRoutes, db, logger)
	exchangerate.RegisterRoutes(protected, db, logger)
	agentrule.RegisterRoutes(protected, db, logger)
	evolution.RegisterRoutes(protected, db, logger)
	entropy.RegisterRoutes(protected, db, logger)
	cost.RegisterRoutes(protected, db, logger, cfg.LLM.DailyBudgetUSD)

	// Metabolism M1 -- scheduled excretion scoring
	m1Svc := metabolism.NewService(db, logger.Named("metabolism"), nil, nil)
	// scheduler.tick.agentos -> SLA escalation for overdue pending actions
	bus.Subscribe("scheduler.tick.agentos", func(ctx context.Context, evt eventbus.Event) error {
		agentosSvc := agentos.NewService(db, logger)
		return agentosSvc.SLAEscalation()
	})

	bus.Subscribe("scheduler.tick.M1", func(ctx context.Context, evt eventbus.Event) error {
		logger.Info("metabolism: M1 tick received")
		_, err := m1Svc.ScoreAndExcreteEntities(false)
		return err
	})
	metabolism.RegisterRoutes(protected, db, logger, nil, nil)

	// WebSocket route
	hub := realtime.NewHub(logger)
	go hub.Run()

	// Wire the 4-level EscalationManager (Issue #35) into the supply chain
	// orchestrator state machine now that the WebSocket hub is available —
	// Level 2 (manual review) and Level 3 (global alert) broadcast via the hub.
	supplyChainOrch.SetEscalationManager(supplychain.NewEscalationManager(logger, hub))

	wsHandler := realtime.NewHandler(hub, logger, cfg.JWT.Secret).WithAIChat(ai.NewAIChatHandler(aiOrch))
	r.GET("/ws", wsHandler.ServeWS)

	// AI routes need the hub for realtime broadcasts.
	ai.RegisterRoutes(protected, db, logger, hub)

	return r
}

// runAgentWithTimeout runs an agent with a 30-second timeout for pipeline chains.
func runAgentWithTimeout(orch *ai.Orchestrator, agentID, decisionPoint string, ctx map[string]interface{}) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := orch.RunWithContext(timeoutCtx, &ai.RunAgentRequest{
		AgentID:       agentID,
		DecisionPoint: decisionPoint,
		Context:       ctx,
	})
	return err
}
