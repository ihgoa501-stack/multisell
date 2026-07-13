package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/aios/costcontrol"
	"github.com/lingmirror/backend-go/internal/aios/setup"
	"github.com/lingmirror/backend-go/internal/aios/toolregistry/tools"
	"github.com/lingmirror/backend-go/internal/auth"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/domain/actionpolicy"
	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/domain/allocation"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/brand"
	"github.com/lingmirror/backend-go/internal/domain/businessdecision"
	"github.com/lingmirror/backend-go/internal/domain/businessfeedback"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/category"
	"github.com/lingmirror/backend-go/internal/domain/competitor"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/compliance"
	"github.com/lingmirror/backend-go/internal/domain/consolidation"
	"github.com/lingmirror/backend-go/internal/domain/cost"
	"github.com/lingmirror/backend-go/internal/domain/dashboard"
	"github.com/lingmirror/backend-go/internal/domain/decision"
	"github.com/lingmirror/backend-go/internal/domain/demandcase"
	"github.com/lingmirror/backend-go/internal/domain/exceptions"
	"github.com/lingmirror/backend-go/internal/domain/exchangerate"
	"github.com/lingmirror/backend-go/internal/domain/experiment"
	"github.com/lingmirror/backend-go/internal/domain/finance"
	"github.com/lingmirror/backend-go/internal/domain/imagegen"
	"github.com/lingmirror/backend-go/internal/domain/importbatch"
	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/integrations/aimapper"
	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"github.com/lingmirror/backend-go/internal/domain/landedcost"
	"github.com/lingmirror/backend-go/internal/domain/listing"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/logistics"
	"github.com/lingmirror/backend-go/internal/domain/loop"
	"github.com/lingmirror/backend-go/internal/domain/mock"
	"github.com/lingmirror/backend-go/internal/domain/notification"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/orderimport"
	"github.com/lingmirror/backend-go/internal/domain/owner"
	"github.com/lingmirror/backend-go/internal/domain/platform"
	"github.com/lingmirror/backend-go/internal/domain/platformfee"
	"github.com/lingmirror/backend-go/internal/domain/platformtruth"
	"github.com/lingmirror/backend-go/internal/domain/price"
	"github.com/lingmirror/backend-go/internal/domain/productanalysis"
	"github.com/lingmirror/backend-go/internal/domain/producthub"
	"github.com/lingmirror/backend-go/internal/domain/productimage"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"github.com/lingmirror/backend-go/internal/domain/purchase"
	"github.com/lingmirror/backend-go/internal/domain/reliability"
	"github.com/lingmirror/backend-go/internal/domain/report"
	"github.com/lingmirror/backend-go/internal/domain/search"
	"github.com/lingmirror/backend-go/internal/domain/sentiment"
	"github.com/lingmirror/backend-go/internal/domain/settlement"
	"github.com/lingmirror/backend-go/internal/domain/shipping"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"github.com/lingmirror/backend-go/internal/domain/supplier"
	"github.com/lingmirror/backend-go/internal/domain/supplychain"
	"github.com/lingmirror/backend-go/internal/domain/support"
	"github.com/lingmirror/backend-go/internal/domain/tariff"
	"github.com/lingmirror/backend-go/internal/domain/workflow"
	"github.com/lingmirror/backend-go/internal/domain/xiaoq"
	"github.com/lingmirror/backend-go/internal/feedback"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"github.com/lingmirror/backend-go/internal/platform/actioncatalog"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/platform/killswitch"
	"github.com/lingmirror/backend-go/internal/platform/routecatalog"
	"github.com/lingmirror/backend-go/internal/platform/scheduler"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge/drivers"
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
	Engine           *gin.Engine
	Bus              *eventbus.Bus
	Scheduler        *scheduler.Scheduler
	Cancel           context.CancelFunc
	acceptingTraffic atomic.Bool
}

// BeginDrain removes the process from readiness before HTTP shutdown starts.
func (a *App) BeginDrain() { a.acceptingTraffic.Store(false) }

// NewRouter creates and configures the Gin engine with all routes.
func NewRouter(db *gorm.DB, cfg *config.Config, logger *zap.Logger) *App {
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// Global middleware
	r.Use(middleware.CORS(cfg))
	r.Use(middleware.RequestID())
	r.Use(middleware.MaxRequestBody(cfg.Server.MaxRequestBodyBytes))

	// Prometheus metrics (before recovery to capture all requests)
	if cfg.Metrics.Enabled {
		r.Use(middleware.Metrics())
		middleware.RegisterDBMetrics(db)
	}
	r.Use(middleware.RecoveryWithSentry(cfg, logger))
	r.Use(middleware.Audit(db, logger))

	// Prometheus metrics endpoint
	if cfg.Metrics.Enabled {
		r.GET("/metrics", middleware.MetricsHandler())
	}

	// Swagger is development-only; release config rejects public exposure.
	if cfg.Server.SwaggerEnabled {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

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

	// Initialize platform adapters (Ozon, Shopee, etc.).
	integrations.InitAdapters(db, logger)

	// Create command dispatcher with action catalog and register Phase 1 handlers.
	cat := actioncatalog.Default()
	cmd := command.NewDispatcher(logger,
		command.WithCatalog(cat),
		command.WithRateLimiter(command.NewRateLimiter(20, time.Hour)),
		command.WithIdempotencyStore(command.NewGormIdempotencyStore(db, 5*time.Minute)),
	)
	cmd.Register("stock_alert", command.StockAlertHandler(db, logger))
	cmd.Register("replenish", command.InventoryReplenishHandler(db, logger))
	cmd.Register("price_review", command.PriceAdjustHandler(db, logger, approvalSvc))
	cmd.Register("price_update", command.PriceAdjustHandler(db, logger, approvalSvc))
	cmd.Register("listing_optimize", command.ListingDraftHandler(db, logger))
	cmd.Register("compliance_check", command.FlagNonCompliantHandler(db, logger))

	// Create scheduler with all registered tasks.
	sched := scheduler.New(bus, logger).WithRetryStore(scheduler.NewGormRetryStore(db))
	if db.Dialector.Name() == "postgres" {
		sched.WithLeaderLease(scheduler.NewPostgresLeaderLease(db))
	}

	// The historical A/G multi-Agent runtime is frozen. Domain services remain
	// available to xiao_q through registered capabilities.
	aiosCfg := setup.Initialize(db, bus, logger)
	budget := costcontrol.NewController(db, logger, cfg.LLM.DailyBudgetUSD, 5*time.Minute, 3.0)
	aiOrch := ai.NewOrchestrator(db, logger).Disable("superseded by xiao_q").WithGuardrails(aiosCfg.Guardrails).WithBudget(budget).WithBus(bus).WithCatalog(cat).WithDispatcher(cmd)

	// ToolBridge for sourcing data collection
	extTracker := toolbridge.NewExternalCallTracker(3)
	toolBridge := toolbridge.NewToolBridge(nil, 0, logger.Named("toolbridge"),
		toolbridge.WithTracker(extTracker),
		toolbridge.WithApprovalVerifier(approval.NewApprovalPolicyChecker(approvalSvc)),
		toolbridge.WithIdempotencyStore(toolbridge.NewGormToolIdempotencyStore(db, 5*time.Minute)),
	) // drivers registered later

	// Reverse logistics return rate tracker (DB-backed).
	returnRateTracker := aftersales.NewReturnRateTracker(db, logger)

	// Supply chain orchestrator (bridges A8 sourcing with A10 logistics quoting,
	// and handles reverse logistics via HandleAftersaleReturn).
	supplyChainOrch := supplychain.NewOrchestrator(bus, db, logger, returnRateTracker)

	auditSvc := operationlog.NewService(db, logger)
	mutationGuard := eventbus.NewMutationGuard(logger, &auditAdapter{svc: auditSvc})
	bus.Subscribe("scheduler.tick.audit_integrity", func(ctx context.Context, _ eventbus.Event) error {
		return operationlog.NewService(db, logger).VerifyIntegrity(ctx)
	})
	sched.Register(scheduler.Task{
		ID: "tick-audit-integrity", AgentID: "audit_integrity", DecisionPoint: "verify_hash_chain",
		Interval: time.Hour, Description: "审计日志哈希链完整性检查",
	})

	app := &App{Engine: r, Bus: bus, Scheduler: sched, Cancel: busCancel}
	app.acceptingTraffic.Store(true)
	registerHealthRoutes(r, db, bus, sched, app.acceptingTraffic.Load, cfg.Server.Version)

	// ==========================================================
	// HTTP routes
	// ==========================================================

	// API v1 routes
	api := r.Group("/api/v1")

	// API v1 Health check (public)
	integrations.RegisterWebhookRoutesWithPipeline(api, bus, logger,
		aimapper.NewPipeline(aimapper.NewMapper(), db, logger), db)

	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "0.1.0",
		})
	})

	// Auth routes (public — login, register, refresh)
	auth.RegisterRoutes(api, db, cfg, logger)
	// Device-bound browser extension seam. This is intentionally outside the
	// normal web JWT group and accepts only sourcing1688.collect credentials.
	sourcing1688.RegisterExtensionRoutes(api, db, cfg, logger)

	// Protected routes (require JWT authentication)
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg))

	// Mutation policy and high-risk approval gate. Every write route must be
	// explicitly classified; unclassified routes fail closed. High routes
	// require a bound one-time approval and idempotency key.
	protected.Use(middleware.ApprovalRequired(db, logger))

	// RBAC routes — require rbac.manage permission
	rbacRoutes := protected.Group("", middleware.RequirePermission(db, "rbac.manage"))
	rbac.RegisterRoutes(rbacRoutes, db, logger)

	// RBAC public routes — accessible to any authenticated user
	rbac.RegisterPublicRoutes(protected, db, logger)

	// AIOS routes (tool registry, runtime, guardrails health)
	setup.RegisterAIOSRoutes(protected, aiosCfg)
	// Scheduler health endpoint -- exposes task run state for AIOS dashboard.
	protected.GET("/aios/scheduler/tasks", func(c *gin.Context) {
		c.JSON(200, gin.H{"tasks": sched.TaskRunState()})
	})
	// Scheduler retry queue -- failed ticks awaiting retry.
	protected.GET("/aios/scheduler/retry-queue", func(c *gin.Context) {
		c.JSON(200, gin.H{"retry_queue": sched.RetryQueue()})
	})

	// Domain routes (all require authentication)
	category.RegisterRoutes(protected, db, logger)
	brand.RegisterRoutes(protected, db, logger)
	competitor.RegisterRoutes(protected, db, logger)
	productRoutes := protected.Group("", middleware.RequirePermission(db, "product.read"))
	sku.RegisterRoutes(productRoutes, db, logger)
	inventoryRoutes := protected.Group("", middleware.RequirePermission(db, "inventory.read"))
	inventory.RegisterRoutes(inventoryRoutes, db, logger, approvalSvc)
	supplier.RegisterRoutes(protected, db, logger)
	purchase.RegisterRoutes(protected, db, logger, bus)

	// Legacy supplychain.order.received events are intentionally not consumed.
	// They were derived from an internal status transition and could therefore
	// manufacture stock. Purchase inventory now changes only in the transaction
	// that persists an immutable external_observed receiving fact and ledger row.
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

	platform.RegisterRoutes(protected, db, logger, approvalSvc)
	listingRoutes := protected.Group("", middleware.RequirePermission(db, "listing.read"))
	listing.RegisterRoutes(listingRoutes, db, logger, bus, approvalSvc)

	approvalSvc = approval.NewService(db, logger, auditSvc).WithBus(bus)
	approval.RegisterRoutes(protected, approvalSvc)
	rbacSvc := rbac.NewService(db, logger)
	loopSvc := loop.NewService(db, logger)
	// Build platform publish hook for listing task execution.
	// After ExecuteTask completes, this pushes the product to the platform API.
	// The shared factory resolves adapters, propagates execution mode, writes audit,
	// and records external platform references.
	publishHook := listingtask.NewPublishHook(db, auditSvc, logger)
	listingTaskSvc := listingtask.RegisterRoutes(listingRoutes, db, logger, approvalSvc, auditSvc, rbacSvc, loopSvc, publishHook)

	// Closed-loop: approval approved for listing_task → approve the listing task.
	// The subscriber writes back approval_id, transitions listing_task to "approved",
	// and records recommendation feedback. It does NOT auto-execute the task —
	// the Owner or system calls ExecuteTask separately after verifying the approved state.
	bus.Subscribe("approval.approved.listing_task",
		mutationGuard.Guard(eventbus.MutationInfo{
			SystemAction: "system.listingtask.approval_approved",
			Domain:       "listingtask",
			Description:  "审批通过: 推进上架任务到已批准",
		}, func(ctx context.Context, evt eventbus.Event) error {
			entityType, _ := evt.Payload["entity_type"].(string)
			if entityType != "listing_task" {
				return nil
			}
			entityID, ok := evt.Payload["entity_id"].(float64)
			if !ok || int64(entityID) == 0 {
				return fmt.Errorf("listing approval event missing entity_id")
			}
			approvalID, ok := evt.Payload["approval_id"].(float64)
			if !ok || int64(approvalID) == 0 {
				return fmt.Errorf("listing approval event missing approval_id")
			}
			reviewer, _ := evt.Payload["reviewer"].(string)
			taskID := int64(entityID)
			aid := int64(approvalID)

			return listingTaskSvc.ApplyOwnerApproval(taskID, aid, reviewer)
		}))

	candidate.RegisterRoutes(protected, db, logger)
	completeness.RegisterRoutes(protected, db, logger)
	compliance.RegisterRoutes(protected, db, logger)
	profit.RegisterRoutes(protected, db, logger)
	experiment.RegisterRoutes(protected, db, logger)
	demandcase.RegisterRoutes(protected, db, logger)
	businessdecision.RegisterRoutes(protected, db)
	registeredCommands := cmd.RegisteredTypes()
	capabilityIDs := make([]string, 0, len(registeredCommands))
	for _, commandType := range registeredCommands {
		capabilityIDs = append(capabilityIDs, "command."+commandType+".v1")
	}
	businessfeedback.RegisterRoutes(protected, db, logger, cmd, approval.NewApprovalPolicyChecker(approvalSvc), capabilityIDs)
	platformtruth.RegisterRoutes(protected)
	xiaoq.RegisterRoutes(protected, db, logger)
	evidenceHandler := profit.NewEvidenceHandler(db)
	protected.GET("/profit/evidence-card/:productId", evidenceHandler.GetEvidenceCard)
	loop.RegisterRoutes(protected, db, logger)
	developmentFixtures := allowDevelopmentFixtures(cfg)
	registerDevelopmentFixtureRoutes(protected, db, logger, developmentFixtures)
	// Mock data must never appear automatically in a production environment.
	if developmentFixtures && cfg.Server.Mode != gin.ReleaseMode {
		ms := mock.NewService(db, logger.Named("mock"))
		if err := ms.SeedMockData(); err != nil {
			logger.Warn("mock data seed failed", zap.Error(err))
		}
	}
	landedcost.RegisterRoutes(protected, db, logger)
	workflow.RegisterRoutes(protected, db, bus, aiOrch, cmd, logger)
	producthub.RegisterRoutes(productRoutes, db, logger)
	sentiment.RegisterRoutes(protected, db, logger)

	shipping.RegisterRoutes(protected, db, logger, developmentFixtures)
	platformfee.RegisterRoutes(protected, db, logger)
	orderRoutes := protected.Group("", middleware.RequirePermission(db, "order.read"))
	order.RegisterRoutes(orderRoutes, db, logger, approvalSvc)
	orderimport.RegisterRoutes(orderRoutes, db, logger)
	settlementRoutes := protected.Group("", middleware.RequirePermission(db, "settlement.read"))
	settlement.RegisterRoutes(settlementRoutes, db, logger)
	financeRoutes := protected.Group("", middleware.RequirePermission(db, "finance.read"))
	finance.RegisterRoutes(financeRoutes, db, logger)
	price.RegisterRoutes(financeRoutes, db, logger, approvalSvc)
	decision.RegisterRoutes(protected, db, logger)
	allocation.RegisterRoutes(protected, db, logger)
	feedback.RegisterRoutes(protected, cfg, db, logger, nil, nil, nil)
	exceptions.RegisterRoutes(protected, db, logger)
	dashboard.RegisterRoutes(protected, db, logger)
	support.RegisterRoutes(protected, db, logger)
	search.RegisterRoutes(protected, db, logger)
	imagegen.RegisterRoutes(protected, db, logger)
	var imageClient *imageservice.Client
	if baseURL, secret := os.Getenv("IMAGE_SERVICE_BASE_URL"), os.Getenv("IMAGE_SERVICE_SHARED_SECRET"); baseURL != "" && secret != "" {
		client, err := imageservice.New(imageservice.Config{BaseURL: baseURL, SharedSecret: secret, Timeout: 20 * time.Second})
		if err != nil {
			logger.Error("Image Service client configuration rejected", zap.Error(err))
		} else {
			imageClient = client
			logger.Info("Image Service client initialized", zap.String("base_url", baseURL))
		}
	} else {
		logger.Info("Image Service client disabled")
	}
	executionTokenSecret := os.Getenv("IMAGE_SERVICE_EXECUTION_TOKEN_SECRET")
	if executionTokenSecret == os.Getenv("IMAGE_SERVICE_SHARED_SECRET") {
		logger.Error("Image Service execution token signing disabled: secret must be independent")
		executionTokenSecret = ""
	} else if executionTokenSecret != "" && len(executionTokenSecret) < 32 {
		logger.Error("Image Service execution token signing disabled: secret must be at least 32 bytes")
		executionTokenSecret = ""
	}
	releaseAttestationSecret := os.Getenv("IMAGE_RELEASE_ATTESTATION_SECRET")
	releaseAttestationKeyID := os.Getenv("IMAGE_RELEASE_ATTESTATION_KEY_ID")
	if releaseAttestationKeyID == "" {
		releaseAttestationKeyID = "image-release-v1"
	}
	productimage.RegisterRoutes(protected, db, logger, imageClient, executionTokenSecret, releaseAttestationSecret, releaseAttestationKeyID)
	importbatch.RegisterRoutes(protected, db, logger)
	operationlog.RegisterRoutes(protected, db, logger)
	integrations.RegisterRoutes(protected, db, logger, approvalSvc, developmentFixtures)
	integrations.RegisterWebhookAdminRoutes(protected, db, logger)
	actionpolicy.RegisterRoutes(protected, db, logger)
	aftersales.RegisterRoutes(protected, db, logger, bus)
	sourcing1688.RegisterRoutes(protected, db, logger, toolBridge)
	tariff.RegisterRoutes(protected, db, logger)
	logistics.RegisterRoutes(protected, db, logger)
	consolidation.RegisterRoutes(protected, db, logger)
	// Wire the loaded carrier-rate engine into the sourcing profit-calc tool
	// so sourcing.recommend uses real shipping rates instead of the static
	// fallback map. No-op if DefaultEngine is nil (no YAML files loaded).
	tools.SetSourcingEngine(logistics.DefaultEngine)
	supplychain.RegisterRoutes(protected, db, logger, bus)
	productanalysis.RegisterRoutes(protected, db, logger)
	reportRoutes := protected.Group("", middleware.RequirePermission(db, "report.read"))
	report.RegisterRoutes(reportRoutes, db, logger)
	exchangerate.RegisterRoutes(protected, db, logger)
	cost.RegisterRoutes(protected, db, logger, cfg.LLM.DailyBudgetUSD)

	reliability.RegisterRoutes(protected, db, logger)

	// Production write kill switch management (admin only).
	// Activate/deactivate require admin.killswitch permission.
	killswitchAdmin := protected.Group("", middleware.RequirePermission(db, "admin.killswitch"))
	killswitch.RegisterRoutes(killswitchAdmin)

	// WebSocket route
	hub := realtime.NewHub(logger)
	go hub.Run()

	// Wire the 4-level EscalationManager (Issue #35) into the supply chain
	// orchestrator state machine now that the WebSocket hub is available —
	// Level 2 (manual review) and Level 3 (global alert) broadcast via the hub.
	supplyChainOrch.SetEscalationManager(supplychain.NewEscalationManager(logger, hub))

	wsHandler := realtime.NewHandler(hub, logger, cfg.JWT.Secret, cfg.CORS.AllowedOrigins)
	r.GET("/ws", wsHandler.ServeWS)

	// AI routes need the hub for realtime broadcasts.
	ai.RegisterLegacyReadRoutes(protected, db, logger)
	notification.RegisterRoutes(protected, db, logger, hub)

	// Browser Extension WebSocket + A12 Collection Agent
	extSvc := &hubExtensionService{hub: hub}
	pluginDrv := drivers.NewPluginDriver(extSvc, 120*time.Second)
	toolBridge.AddDriver(toolbridge.DriverEntry{
		Name:   "plugin",
		Driver: pluginDrv,
		Weight: 10,
	})
	extHandler := realtime.NewExtensionHandler(hub, logger, cfg.JWT.Secret, cfg.CORS.AllowedOrigins, db, cfg.Server.EffectiveDeploymentEnvironment()).
		WithPluginDriver(&extPluginBridge{driver: pluginDrv})
	r.GET("/ws/extension", extHandler.ServeWS)

	runtimeRoutes := make([]routecatalog.RuntimeRoute, 0, len(r.Routes()))
	for _, route := range r.Routes() {
		runtimeRoutes = append(runtimeRoutes, routecatalog.RuntimeRoute{Method: route.Method, Path: route.Path})
	}
	if err := routecatalog.ValidateRuntimeRoutes(runtimeRoutes); err != nil {
		panic("HTTP mutation security policy invalid: " + err.Error())
	}

	// Start background infrastructure only after every subscription and task is
	// registered, so recovered events can never run against a partial topology.
	bus.Start(busCtx)
	go sched.Start(busCtx)
	return app
}

func allowDevelopmentFixtures(cfg *config.Config) bool {
	return cfg.Server.EffectiveDeploymentEnvironment() == "development"
}

func registerDevelopmentFixtureRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, allow bool) {
	if !allow {
		return
	}
	mock.RegisterRoutes(rg, db, logger)
	owner.RegisterRoutes(rg, db, logger)
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

func (b *extPluginBridge) OnListPageResult(_ int64, data json.RawMessage) error {
	b.driver.HandleResponse(data)
	return nil
}

func (b *extPluginBridge) HasPendingList(requestID string) bool {
	return b.driver.HasPendingList(requestID)
}
