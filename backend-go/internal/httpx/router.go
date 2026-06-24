package httpx

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/agent"
	"github.com/lingmirror/backend-go/internal/agentos"
	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/auth"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/domain/actionpolicy"
	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/domain/allocation"
	"github.com/lingmirror/backend-go/internal/domain/brand"
	"github.com/lingmirror/backend-go/internal/domain/category"
	"github.com/lingmirror/backend-go/internal/domain/dashboard"
	"github.com/lingmirror/backend-go/internal/domain/decision"
	"github.com/lingmirror/backend-go/internal/domain/exceptions"
	"github.com/lingmirror/backend-go/internal/domain/finance"
	"github.com/lingmirror/backend-go/internal/domain/imagegen"
	"github.com/lingmirror/backend-go/internal/domain/importbatch"
	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"github.com/lingmirror/backend-go/internal/domain/listing"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/domain/notification"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/orderimport"
	"github.com/lingmirror/backend-go/internal/domain/platform"
	"github.com/lingmirror/backend-go/internal/domain/platformfee"
	"github.com/lingmirror/backend-go/internal/domain/price"
	"github.com/lingmirror/backend-go/internal/domain/report"
	"github.com/lingmirror/backend-go/internal/domain/search"
	"github.com/lingmirror/backend-go/internal/domain/settlement"
	"github.com/lingmirror/backend-go/internal/domain/shipping"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"github.com/lingmirror/backend-go/internal/domain/supplier"
	"github.com/lingmirror/backend-go/internal/domain/exchangerate"
	"github.com/lingmirror/backend-go/internal/domain/agentrule"
	"github.com/lingmirror/backend-go/internal/domain/entropy"
	"github.com/lingmirror/backend-go/internal/domain/evolution"
	"time"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/platform/scheduler"
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
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())
	r.Use(middleware.RecoveryWithSentry(cfg, logger))
	r.Use(middleware.Audit(db, logger))

	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "0.1.0",
		})
	})

	// ==========================================================
	// Phase 1 Infrastructure: Event Bus + Command + Scheduler
	// ==========================================================

	// Create event bus (with optional outbox persistence).
	bus := eventbus.New(logger, eventbus.WithDB(db), eventbus.WithWorkers(4))
	busCtx, busCancel := context.WithCancel(context.Background())
	defer busCancel()
	bus.Start(busCtx)

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
	aiOrch := ai.NewOrchestrator(db, logger)

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
	// scheduler.tick.trustscore → recalculate trust scores
	bus.Subscribe("scheduler.tick.trustscore", func(ctx context.Context, evt eventbus.Event) error {
		svc := trustscore.NewService(db, logger)
		return svc.Recalculate()
	})

	// -------------------------------------------------------
	// Pipeline chain rules (via event bus)
	// -------------------------------------------------------

	// A5 stock_alert (red) → G3 discount_risk_check
	bus.Subscribe("agent.decided.A5.stock_alert", func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		if status, ok := payload["stock_status"].(string); ok && status == "red" {
			_, err := aiOrch.Run(&ai.RunAgentRequest{
				AgentID:       "G3",
				DecisionPoint: "discount_risk_check",
				Context:       payload,
			})
			return err
		}
		return nil
	})

	// G3 discount_risk_check (block) → A6 profit_watch
	bus.Subscribe("agent.decided.G3.discount_risk_check", func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		if action, ok := payload["action"].(string); ok && action == "block" {
			_, err := aiOrch.Run(&ai.RunAgentRequest{
				AgentID:       "A6",
				DecisionPoint: "profit_watch",
				Context:       payload,
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
			_, err := aiOrch.Run(&ai.RunAgentRequest{
				AgentID:       "A2",
				DecisionPoint: "listing_optimize",
				Context:       payload,
			})
			return err
		}
		return nil
	})

	// G0 system_health (anomaly_count > 3) → G1 dashboard_overview
	bus.Subscribe("agent.decided.G0.system_health", func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		if count, ok := payload["anomaly_count"].(int); ok && count > 3 {
			_, err := aiOrch.Run(&ai.RunAgentRequest{
				AgentID:       "G1",
				DecisionPoint: "dashboard_overview",
				Context:       payload,
			})
			return err
		}
		// Also check for float64 (JSON unmarshal from scheduler)
		if count, ok := payload["anomaly_count"].(float64); ok && count > 3 {
			_, err := aiOrch.Run(&ai.RunAgentRequest{
				AgentID:       "G1",
				DecisionPoint: "dashboard_overview",
				Context:       payload,
			})
			return err
		}
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
		Interval: time.Hour * 6, Description: "熵防御周期",
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

	// RBAC routes
	rbac.RegisterRoutes(protected, db, logger)

	// Agent routes (wired through the AI orchestrator)
	agent.RegisterRoutes(protected, db, logger, aiOrch)

	// AgentOS routes
	agentos.RegisterRoutes(protected, db, logger)

	// Domain routes (all require authentication)
	category.RegisterRoutes(protected, db, logger)
	brand.RegisterRoutes(protected, db, logger)
	sku.RegisterRoutes(protected, db, logger)
	price.RegisterRoutes(protected, db, logger)
	inventory.RegisterRoutes(protected, db, logger)
	supplier.RegisterRoutes(protected, db, logger)
	platform.RegisterRoutes(protected, db, logger)
	listing.RegisterRoutes(protected, db, logger)
	listingtask.RegisterRoutes(protected, db, logger)
	shipping.RegisterRoutes(protected, db, logger)
	platformfee.RegisterRoutes(protected, db, logger)
	order.RegisterRoutes(protected, db, logger)
	orderimport.RegisterRoutes(protected, db, logger)
	settlement.RegisterRoutes(protected, db, logger)
	finance.RegisterRoutes(protected, db, logger)
	decision.RegisterRoutes(protected, db, logger)
	allocation.RegisterRoutes(protected, db, logger)
	exceptions.RegisterRoutes(protected, db, logger)
	notification.RegisterRoutes(protected, db, logger)
	dashboard.RegisterRoutes(protected, db, logger)
	search.RegisterRoutes(protected, db, logger)
	imagegen.RegisterRoutes(protected, db, logger)
	importbatch.RegisterRoutes(protected, db, logger)
	operationlog.RegisterRoutes(protected, db, logger)
	integrations.RegisterRoutes(protected, db, logger)
	actionpolicy.RegisterRoutes(protected, db, logger)
	aftersales.RegisterRoutes(protected, db, logger)
	sourcing1688.RegisterRoutes(protected, db, logger)
	trustscore.RegisterRoutes(protected, db, logger)
	report.RegisterRoutes(protected, db, logger)
	exchangerate.RegisterRoutes(protected, db, logger)
	agentrule.RegisterRoutes(protected, db, logger)
	evolution.RegisterRoutes(protected, db, logger)
	entropy.RegisterRoutes(protected, db, logger)

	// WebSocket route
	hub := realtime.NewHub(logger)
	go hub.Run()
	wsHandler := realtime.NewHandler(hub, logger)
	r.GET("/ws", wsHandler.ServeWS)

	// AI routes need the hub for realtime broadcasts.
	ai.RegisterRoutes(protected, db, logger, hub)

	return r
}
