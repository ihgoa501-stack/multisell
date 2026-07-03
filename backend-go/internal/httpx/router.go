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
	"github.com/lingmirror/backend-go/internal/agent/impl"
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
	"github.com/lingmirror/backend-go/internal/domain/agentrule"
	"github.com/lingmirror/backend-go/internal/domain/allocation"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/brand"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/category"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/compliance"
	"github.com/lingmirror/backend-go/internal/domain/consolidation"
	"github.com/lingmirror/backend-go/internal/domain/content"
	"github.com/lingmirror/backend-go/internal/domain/cost"
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
	"github.com/lingmirror/backend-go/internal/domain/landedcost"
	"github.com/lingmirror/backend-go/internal/domain/listing"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/logistics"
	"github.com/lingmirror/backend-go/internal/domain/loop"
	"github.com/lingmirror/backend-go/internal/domain/metabolism"
	"github.com/lingmirror/backend-go/internal/domain/mock"
	"github.com/lingmirror/backend-go/internal/domain/notification"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/orchestration"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/orderimport"
	"github.com/lingmirror/backend-go/internal/domain/owner"
	"github.com/lingmirror/backend-go/internal/domain/personalrule"
	"github.com/lingmirror/backend-go/internal/domain/platform"
	"github.com/lingmirror/backend-go/internal/domain/platformfee"
	"github.com/lingmirror/backend-go/internal/domain/price"
	"github.com/lingmirror/backend-go/internal/domain/productanalysis"
	"github.com/lingmirror/backend-go/internal/domain/producthub"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"github.com/lingmirror/backend-go/internal/domain/purchase"
	"github.com/lingmirror/backend-go/internal/domain/report"
	"github.com/lingmirror/backend-go/internal/domain/search"
	"github.com/lingmirror/backend-go/internal/domain/sentiment"
	"github.com/lingmirror/backend-go/internal/domain/settings"
	"github.com/lingmirror/backend-go/internal/domain/settlement"
	"github.com/lingmirror/backend-go/internal/domain/shipping"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"github.com/lingmirror/backend-go/internal/domain/sourcing"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"github.com/lingmirror/backend-go/internal/domain/supplier"
	"github.com/lingmirror/backend-go/internal/domain/supplychain"
	"github.com/lingmirror/backend-go/internal/domain/support"
	"github.com/lingmirror/backend-go/internal/domain/tariff"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"github.com/lingmirror/backend-go/internal/domain/workflow"
	"github.com/lingmirror/backend-go/internal/feedback"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/platform/actioncatalog"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/platform/scheduler"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge/drivers"
	"github.com/lingmirror/backend-go/internal/prismadapter"
	"github.com/lingmirror/backend-go/internal/rbac"
	"github.com/lingmirror/backend-go/internal/realtime"
	"go.uber.org/zap"
	"gorm.io/gorm"

	_ "github.com/lingmirror/backend-go/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// App holds the Gin engine and background services for graceful shutdown.
type App struct {
	Engine    *gin.Engine
	Bus       *eventbus.Bus
	Scheduler *scheduler.Scheduler
	Cancel    context.CancelFunc
}

// NewRouter creates and configures the Gin engine with all routes.
func NewRouter(db *gorm.DB, cfg *config.Config, logger *zap.Logger) *App {
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// Global middleware
	r.Use(middleware.CORS(cfg))
	r.Use(middleware.RequestID())

	// Prometheus metrics (before recovery to capture all requests)
	if cfg.Metrics.Enabled {
		r.Use(middleware.Metrics())
		middleware.RegisterDBMetrics(db)
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

	// Swagger API documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ==========================================================
	// Phase 1 Infrastructure: Event Bus + Command + Scheduler
	// ==========================================================

	// Initialize approval service early for command dispatcher security gates.
	// auditSvc is set later when it becomes available.
	approvalSvc := approval.NewService(db, logger, nil)

	// Create event bus (with optional outbox persistence).
	sr := eventbus.NewSchemaRegistry()
	// ponytail: register known payload schemas; add more as event contracts stabilize
	sr.Register("stock.alert", eventbus.StockAlertPayload{})
	sr.Register("profit.*", eventbus.ProfitWatchPayload{})
	sr.Register("compliance.*", eventbus.ComplianceCheckPayload{})
	bus := eventbus.New(logger, eventbus.WithDB(db), eventbus.WithWorkers(4), eventbus.WithSchema(sr))
	busCtx, busCancel := context.WithCancel(context.Background())
	bus.Start(busCtx)

	// Initialize platform adapters (Ozon, Shopee, etc.).
	integrations.InitAdapters(db, logger)

	// Create command dispatcher with action catalog and register Phase 1 handlers.
	cat := actioncatalog.Default()
	cmd := command.NewDispatcher(logger, command.WithCatalog(cat))
	cmd.Register("stock_alert", command.StockAlertHandler(db, logger))
	cmd.Register("replenish", command.InventoryReplenishHandler(db, logger))
	cmd.Register("price_review", command.PriceAdjustHandler(db, logger, approvalSvc))
	cmd.Register("price_update", command.PriceAdjustHandler(db, logger, approvalSvc))
	cmd.Register("listing_optimize", command.ListingDraftHandler(db, logger))
	cmd.Register("compliance_check", command.FlagNonCompliantHandler(db, logger))

	// Create scheduler with all registered tasks.
	sched := scheduler.New(bus, logger)

	// AI orchestrator (shared by /ai and /agents routes).
	aiosCfg := setup.Initialize(db, bus, logger)
	budget := costcontrol.NewController(db, logger, cfg.LLM.DailyBudgetUSD, 5*time.Minute, 3.0)
	aiOrch := ai.NewOrchestrator(db, logger).WithGuardrails(aiosCfg.Guardrails).WithBudget(budget).WithBus(bus).WithCatalog(cat).WithDispatcher(cmd)

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

	auditSvc := operationlog.NewService(db, logger)
	mutationGuard := eventbus.NewMutationGuard(logger, &auditAdapter{svc: auditSvc})

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
	bus.Subscribe("scheduler.tick.trustscore",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.trustscore.auto_upgrade",
			Domain:       "trustscore",
			Description:  "定时信任分重算 → 自动升级Agent自主化等级",
		}, func(ctx context.Context, evt eventbus.Event) error {
			ug := trustscore.NewUpgrader(db, logger)
			_, err := ug.AutoUpgrade()
			return err
		}))
	// scheduler.tick.entropy → run entropy defenses + agent health check
	bus.Subscribe("scheduler.tick.entropy",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.entropy.run_defenses",
			Domain:       "entropy",
			Description:  "定时熵防御运行: 异常检测和Agent健康检查",
		}, func(ctx context.Context, evt eventbus.Event) error {
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
		}))

	// scheduler.tick.A8 → sourcing batch scan
	bus.Subscribe("scheduler.tick.A8", func(ctx context.Context, evt eventbus.Event) error {
		return sourcing.HandleSourcingTick(db, logger)(ctx, evt)
	})
	// scheduler.tick.A9 → batch ops
	bus.Subscribe("scheduler.tick.A9", func(ctx context.Context, evt eventbus.Event) error {
		return nil // A9 is API-driven; auto-tick triggers no-op for now
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

	// -------------------------------------------------------
	// Agent Decision Audit Chain: log every agent.decided.*
	// event to the operation_log for the full audit trail.
	// -------------------------------------------------------
	bus.Subscribe("agent.decided.*",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.approval.agent_decision_auto_create",
			Domain:       "approval",
			Description:  "Agent决策自动创建审批: 低置信度Agent决策自动生成审批请求",
		}, approval.NewAgentDecisionSubscriber(db, logger, auditSvc)))
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
		ID: "tick-a9", AgentID: "A9", DecisionPoint: "batch_operations",
		Interval: time.Hour * 2, Description: "批量运维扫描",
	})
	sched.Register(scheduler.Task{
		ID: "tick-m1", AgentID: "M1", DecisionPoint: "excretion_scoring",
		Interval: time.Hour * 1, Description: "代谢排泄评分",
	})
	sched.Register(scheduler.Task{
		ID: "tick-sla-escalation", AgentID: "agentos", DecisionPoint: "sla_escalation",
		Interval: time.Minute * 15, Description: "SLA过期升级待审批动作",
	})
	sched.Register(scheduler.Task{
		ID: "tick-orch", AgentID: "orch", DecisionPoint: "supply_chain_heartbeat",
		Interval: time.Minute * 15, Description: "供应链编排心跳",
	})

	// Ozon sync handler
	bus.Subscribe("scheduler.tick.ozon_sync",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.integrations.ozon_order_sync",
			Domain:       "integrations",
			Description:  "定时从 Ozon 平台同步订单数据到本地数据库",
		}, func(ctx context.Context, evt eventbus.Event) error {
			integrations.InitAdapters(db, logger)
			svc := integrations.NewService(db, logger)
			return svc.SyncOzonOrders(ctx)
		}))

	// Start scheduler in background goroutine.
	go sched.Start(busCtx)

	// ==========================================================
	// HTTP routes
	// ==========================================================

	// API v1 routes
	api := r.Group("/api/v1")

	// API v1 Health check (public)
	integrations.RegisterWebhookRoutes(api, bus, logger)

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

	// RBAC public routes — accessible to any authenticated user
	rbac.RegisterPublicRoutes(protected, db, logger)

	// Agent routes (wired through the AI orchestrator)
	agent.RegisterRoutes(protected, db, logger, aiOrch)

	// AIOS routes (tool registry, runtime, guardrails health)
	setup.RegisterAIOSRoutes(protected, aiosCfg)
		// Scheduler health endpoint -- exposes task run state for AIOS dashboard.
		protected.GET("/aios/scheduler/tasks", func(c *gin.Context) {
		c.JSON(200, gin.H{"tasks": sched.TaskRunState()})
		})

	// AgentOS routes
	agentos.RegisterRoutes(protected, db, logger)

	// Domain routes (all require authentication)
	category.RegisterRoutes(protected, db, logger)
	brand.RegisterRoutes(protected, db, logger)
	productRoutes := protected.Group("", middleware.RequirePermission(db, "product.read"))
	sku.RegisterRoutes(productRoutes, db, logger)
	inventoryRoutes := protected.Group("", middleware.RequirePermission(db, "inventory.read"))
	inventory.RegisterRoutes(inventoryRoutes, db, logger)
	supplier.RegisterRoutes(protected, db, logger)
	purchase.RegisterRoutes(protected, db, logger, bus)

	// Supply chain event: purchase order received → auto-increment inventory.
	// GUARDRAIL (mutation guard): audited via MutationGuard, registered as system.inventory.receive.
	// Idempotency delegated to supply chain orchestrator via order_no.
	bus.Subscribe("supplychain.order.received",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.inventory.receive",
			Domain:       "inventory",
			Description:  "采购入库确认后自动增加对应 SKU 的库存数量",
		}, func(ctx context.Context, evt eventbus.Event) error {
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
		}))
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
	bus.Subscribe("sourcing.recommend",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.supplychain.create_recommend_flow",
			Domain:       "supplychain",
			Description:  "选品推荐事件: 创建供应链流水记录并触发物流报价",
		}, func(ctx context.Context, evt eventbus.Event) error {
			return supplyChainOrch.HandleRecommendEvent(ctx, evt)
		}))
	// scheduler.tick.orch → supply chain orchestrator heartbeat (no-op)
	bus.Subscribe("scheduler.tick.orch", func(ctx context.Context, evt eventbus.Event) error {
		return nil
	})

	// Supply chain event: after-sale completed → auto-adjust inventory.
	// GUARDRAIL (mutation guard): audited via MutationGuard, registered as system.inventory.aftersale_restock.
	// Idempotency delegated to aftersale orchestrator via return order dedup.
	bus.Subscribe("supplychain.aftersale.completed",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.inventory.aftersale_restock",
			Domain:       "inventory",
			Description:  "售后入库: 退货完成后自动增加对应 SKU 的库存数量",
		}, func(ctx context.Context, evt eventbus.Event) error {
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
		}))

	// Supply chain event: after-sale return initiated -> create reverse logistics flow
	bus.Subscribe("supplychain.aftersale.returned",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.supplychain.aftersale_return",
			Domain:       "supplychain",
			Description:  "售后退货: 创建逆向物流流水并记录退货跟踪数据",
		}, func(ctx context.Context, evt eventbus.Event) error {
			return supplyChainOrch.HandleAftersaleReturn(ctx, evt)
		}))

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
	bus.Subscribe("supplychain.stock.critical",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.supplychain.stock_critical",
			Domain:       "supplychain",
			Description:  "库存红色预警: 创建供应链流水并触发补货审批",
		}, func(ctx context.Context, evt eventbus.Event) error {
			return supplyChainOrch.HandleStockCritical(ctx, evt)
		}))

	platform.RegisterRoutes(protected, db, logger)
	listingRoutes := protected.Group("", middleware.RequirePermission(db, "listing.read"))
	listing.RegisterRoutes(listingRoutes, db, logger, bus)

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
	approvalSvc = approval.NewService(db, logger, auditSvc).WithBus(bus)
	rbacSvc := rbac.NewService(db, logger)
	loopSvc := loop.NewService(db, logger, prismSvc, prismStrict)
	// Build platform publish hook for listing task execution.
	// After ExecuteTask completes, this pushes the product to the platform API.
	publishHook := func(taskID int64) error {
		var t listingtask.ListingTask
		if err := db.First(&t, taskID).Error; err != nil {
			return fmt.Errorf("load listing task %d: %w", taskID, err)
		}
		// Resolve platform code → adapter.
		var plat struct{ Code string }
		if err := db.Table("platform").Select("code").Where("id = ?", t.PlatformID).Scan(&plat).Error; err != nil {
			return fmt.Errorf("load platform %d: %w", t.PlatformID, err)
		}
		adapter, ok := integrations.GetAdapter(plat.Code)
		if !ok {
			return fmt.Errorf("no adapter for platform %q", plat.Code)
		}
		// Load product data.
		var prod sku.Product
		if err := db.First(&prod, t.ProductID).Error; err != nil {
			return fmt.Errorf("load product %d: %w", t.ProductID, err)
		}
		type skuRow struct {
			ID   int64
			Code string
		}
		var skus []skuRow
		if err := db.Table("sku").Where("product_id = ?", t.ProductID).Find(&skus).Error; err != nil {
			return fmt.Errorf("load SKUs for product %d: %w", t.ProductID, err)
		}
		// Find first active account for this platform.
		// ponytail: picks the first active account by insertion order.
		// Add account_id to ListingTask if multi-store routing becomes needed.
		var acct integrations.PlatformIntegrationAccount
		if err := db.Where("platform_id = ? AND status = ?", t.PlatformID, "active").First(&acct).Error; err != nil {
			return fmt.Errorf("no active account for platform %d: %w", t.PlatformID, err)
		}
		prices := make(map[int64]string)
		inventories := make(map[int64]int)
		publishSKUs := make([]integrations.PublishSKU, 0, len(skus))
		for _, sk := range skus {
			publishSKUs = append(publishSKUs, integrations.PublishSKU{SkuID: sk.ID, SkuCode: sk.Code})
			if t.TargetSalePrice != nil {
				prices[sk.ID] = fmt.Sprintf("%.2f", *t.TargetSalePrice)
			}
			var s sku.Sku
			if err := db.First(&s, sk.ID).Error; err == nil {
				inventories[sk.ID] = s.Stock
			}
		}
		pkgH, _ := prod.PackageHeightCm.Float64()
		pkgW, _ := prod.PackageWidthCm.Float64()
		pkgL, _ := prod.PackageLengthCm.Float64()
		pkgWt, _ := prod.PackageWeightKg.Float64()

		result, err := adapter.Publish(context.Background(), &integrations.PublishInput{
			ProductID:      t.ProductID,
			PlatformID:     t.PlatformID,
			AccountID:      acct.ID,
			SKUs:           publishSKUs,
			Prices:         prices,
			Inventories:    inventories,
			ProductName:    prod.Name,
			Description:    prod.Description,
			CategoryID:     prod.CategoryID,
			MainImage:      prod.MainImage,
			PackageHeight:  pkgH,
			PackageWidth:   pkgW,
			PackageLength:  pkgL,
			PackageWeight:  pkgWt,
		})
		if err != nil {
			return fmt.Errorf("platform %s publish failed: %w", plat.Code, err)
		}
		// Merge platform publish result into existing item result
		// (which already contains Prism compliance data from ExecuteTask).
		resultJSON, _ := json.Marshal(result)
		db.Exec(`UPDATE listing_task_item SET result = COALESCE(result, '{}')::jsonb || ?::jsonb WHERE task_id = ?`,
			string(resultJSON), taskID)

		logger.Info("listing task platform publish succeeded",
			zap.Int64("task_id", taskID),
			zap.String("platform", plat.Code),
			zap.String("platform_product_id", result.PlatformProductID),
			zap.String("platform_url", result.PlatformURL),
		)
		return nil
	}
	listingtask.RegisterRoutes(listingRoutes, db, logger, prismSvc, prismStrict, approvalSvc, auditSvc, rbacSvc, loopSvc, publishHook)

	// Closed-loop: approval approved for listing_task → execute the task.
	bus.Subscribe("approval.approved.listing_task",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.listingtask.execute",
			Domain:       "listingtask",
			Description:  "审批通过: 执行上架任务",
		}, func(ctx context.Context, evt eventbus.Event) error {
			entityID, ok := evt.Payload["entity_id"].(float64)
			if !ok || int64(entityID) == 0 {
				return nil
			}
			ltSvc := listingtask.NewService(db, logger, prismSvc, prismStrict, approvalSvc, auditSvc, loopSvc)
			_, err := ltSvc.ExecuteTask(int64(entityID), "system")
			return err
		}))

	candidate.RegisterRoutes(protected, db, logger)
	completeness.RegisterRoutes(protected, db, logger)
	compliance.RegisterRoutes(protected, db, logger)
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
	approval.RegisterRoutes(protected, db, logger, auditSvc)
	landedcost.RegisterRoutes(protected, db, logger)
	orchestration.RegisterRoutes(protected, db, bus, aiOrch, logger)
	workflow.RegisterRoutes(protected, db, bus, aiOrch, cmd, logger)
	personalrule.RegisterRoutes(protected, db, logger)
	producthub.RegisterRoutes(productRoutes, db, logger)
	sentiment.RegisterRoutes(protected, db, logger)

	shippingRoutes := protected.Group("", middleware.RequirePermission(db, "shipping.read"))
	shipping.RegisterRoutes(shippingRoutes, db, logger)
	platformfee.RegisterRoutes(protected, db, logger)
	orderRoutes := protected.Group("", middleware.RequirePermission(db, "order.read"))
	order.RegisterRoutes(orderRoutes, db, logger)
	orderimport.RegisterRoutes(orderRoutes, db, logger)
	settlementRoutes := protected.Group("", middleware.RequirePermission(db, "settlement.read"))
	settlement.RegisterRoutes(settlementRoutes, db, logger)
	financeRoutes := protected.Group("", middleware.RequirePermission(db, "finance.read"))
	finance.RegisterRoutes(financeRoutes, db, logger)
	price.RegisterRoutes(financeRoutes, db, logger)
	decision.RegisterRoutes(protected, db, logger)
	allocation.RegisterRoutes(protected, db, logger)
	feedback.RegisterRoutes(protected, cfg, db, logger, nil, nil, nil)
	exceptions.RegisterRoutes(protected, db, logger)
	notification.RegisterRoutes(protected, db, logger)
	dashboard.RegisterRoutes(protected, db, logger)
	support.RegisterRoutes(protected, db, logger)
	search.RegisterRoutes(protected, db, logger)
	settings.RegisterRoutes(protected, db, logger)
	imagegen.RegisterRoutes(protected, db, logger)
	importbatch.RegisterRoutes(protected, db, logger)
	content.RegisterRoutes(protected, db, logger, aiOrch)
	operationlog.RegisterRoutes(protected, db, logger)
	integrations.RegisterRoutes(protected, db, logger)
	integrations.RegisterWebhookAdminRoutes(protected, db, logger)
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
	bus.Subscribe("scheduler.tick.agentos",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.agentos.sla_escalation",
			Domain:       "agentos",
			Description:  "定时SLA过期升级: 将超时的待审批动作升级处理",
		}, func(ctx context.Context, evt eventbus.Event) error {
			agentosSvc := agentos.NewService(db, logger)
			return agentosSvc.SLAEscalation()
		}))

	bus.Subscribe("scheduler.tick.M1",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.metabolism.excrete",
			Domain:       "metabolism",
			Description:  "定时代谢排泄评分: 评分并清除不合格实体",
		}, func(ctx context.Context, evt eventbus.Event) error {
			logger.Info("metabolism: M1 tick received")
			_, err := m1Svc.ScoreAndExcreteEntities(false)
			return err
		}))
	metabolism.RegisterRoutes(protected, db, logger, nil, nil)

	// WebSocket route
	hub := realtime.NewHub(logger)
	go hub.Run()

	// Wire the 4-level EscalationManager (Issue #35) into the supply chain
	// orchestrator state machine now that the WebSocket hub is available —
	// Level 2 (manual review) and Level 3 (global alert) broadcast via the hub.
	supplyChainOrch.SetEscalationManager(supplychain.NewEscalationManager(logger, hub))

	wsHandler := realtime.NewHandler(hub, logger, cfg.JWT.Secret, cfg.CORS.AllowedOrigins).WithAIChat(ai.NewAIChatHandler(aiOrch))
	r.GET("/ws", wsHandler.ServeWS)

	// AI routes need the hub for realtime broadcasts.
	moaCatalog := ai.SchemaCatalog{
		"A6": {"profit_health"},
		"A5": {"stock_alert"},
		"G3": {"compliance_check"},
	}
	moaCoord := ai.NewMOACoordinator(aiOrch, bus, approvalSvc, moaCatalog, logger)
	ai.RegisterRoutes(protected, db, logger, hub, moaCoord, cmd, aiosCfg.Guardrails)

	// Browser Extension WebSocket + A12 Collection Agent
	extSvc := &hubExtensionService{hub: hub}
	pluginDrv := drivers.NewPluginDriver(extSvc, 120*time.Second)
	toolBridge.AddDriver(toolbridge.DriverEntry{
		Name:   "plugin",
		Driver: pluginDrv,
		Weight: 10,
	})
	candSvc := candidate.NewService(db, logger)
	extHandler := realtime.NewExtensionHandler(hub, logger, cfg.JWT.Secret, cfg.CORS.AllowedOrigins).
		WithPluginDriver(&extPluginBridge{driver: pluginDrv}).
		OnAutoCollect(func(userID int64, payload json.RawMessage) error {
			var result struct {
				ID      string `json:"id"`
				Payload struct {
					Status string               `json:"status"`
					Data   *toolbridge.PageData `json:"data"`
				} `json:"payload"`
			}
			if err := json.Unmarshal(payload, &result); err != nil {
				return fmt.Errorf("parse auto-collect payload: %w", err)
			}
			if result.Payload.Status != "ok" || result.Payload.Data == nil {
				return nil
			}

			input := impl.PageDataToCandidate(result.Payload.Data, map[string]interface{}{
				"collected_by": fmt.Sprintf("extension:%d", userID),
				"url":          result.Payload.Data.SourceURL,
			})

			created, err := candSvc.Create(input)
			if err != nil {
				return fmt.Errorf("save auto-collected candidate: %w", err)
			}
			logger.Info("auto-collect saved candidate",
				zap.Int64("candidate_id", created.ID),
				zap.String("source_url", result.Payload.Data.SourceURL))
			return nil
		}).
		OnListCollect(func(userID int64, payload json.RawMessage) error {
			var result struct {
				Status string `json:"status"`
				Data   struct {
					PageURL     string `json:"page_url"`
					CollectedAt string `json:"collected_at"`
					Items       []struct {
						Title      string `json:"title"`
						PriceRange string `json:"price_range"`
						DetailURL  string `json:"detail_url"`
						ImageURL   string `json:"image_url"`
					} `json:"items"`
				} `json:"data"`
			}
			if err := json.Unmarshal(payload, &result); err != nil {
				return fmt.Errorf("parse list result: %w", err)
			}
			if result.Status != "ok" {
				return nil
			}
			for _, item := range result.Data.Items {
				lead := candidate.CollectLead{
					Title:         item.Title,
					PriceRange:    item.PriceRange,
					DetailURL:     item.DetailURL,
					ImageURL:      item.ImageURL,
					SourcePageURL: result.Data.PageURL,
					Status:        "pending_detail_collect",
				}
				if err := candSvc.CreateCollectLead(&lead); err != nil {
					logger.Warn("create collect lead failed", zap.String("detail_url", item.DetailURL), zap.Error(err))
				}
			}
			logger.Info("list collect saved leads", zap.Int("count", len(result.Data.Items)))
			return nil
		})
	r.GET("/ws/extension", extHandler.ServeWS)
	a12 := impl.NewCollectionAgent(toolBridge, candSvc, logger)
	aiOrch.RegisterAgent("A12", a12)

	return &App{Engine: r, Bus: bus, Scheduler: sched, Cancel: busCancel}
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

// auditAdapter bridges eventbus.MutationAuditInput -> operationlog.StructuredLogInput.
type auditAdapter struct {
	svc *operationlog.Service
}

func (a *auditAdapter) LogStructured(input *eventbus.MutationAuditInput) error {
	if a == nil || a.svc == nil {
		return nil
	}
	return a.svc.LogStructured(&operationlog.StructuredLogInput{
		Module:      input.Module,
		Action:      input.Action,
		ResourceID:  input.ResourceID,
		Operator:    input.Operator,
		Content:     input.Content,
		Result:      input.Result,
		TriggerType: input.TriggerType,
	})
}

// ── Extension WebSocket adapter types ────────────────────────────────

// hubExtensionService adapts the realtime Hub to the drivers.ExtensionService
// interface, allowing the PluginDriver to send messages to the browser extension.
type hubExtensionService struct {
	hub *realtime.Hub
}

func (s *hubExtensionService) SendToUser(userID int64, msg []byte) error {
	return s.hub.SendToUser(userID, msg)
}

func (s *hubExtensionService) RegisterCallback(userID int64, _ func([]byte)) {
	// Callback registration is handled via the ExtensionHandler's
	// OnFetchProductResult routing. No additional registration needed.
}

// extPluginBridge adapts drivers.PluginDriver to the realtime.PluginDriver
// interface, so fetch_product_result messages from the extension WS are
// routed to the PluginDriver's pending-request channels.
type extPluginBridge struct {
	driver *drivers.PluginDriver
}

func (b *extPluginBridge) OnFetchProductResult(_ int64, data json.RawMessage) error {
	b.driver.HandleResponse(data)
	return nil
}

func (b *extPluginBridge) HasPending(requestID string) bool {
	return b.driver.HasPending(requestID)
}
