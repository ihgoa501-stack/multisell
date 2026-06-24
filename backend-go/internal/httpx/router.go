package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/agent"
	"github.com/lingmirror/backend-go/internal/agentos"
	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/auth"
	"github.com/lingmirror/backend-go/internal/config"
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
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
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
	r.Use(middleware.Recovery(logger))
	// Audit logs mutations (POST/PUT/PATCH/DELETE) to operation_log.
	// Safe no-ops for GET/HEAD/OPTIONS and /api/health.
	r.Use(middleware.Audit(db, logger))

	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "0.1.0",
		})
	})

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

	// AI orchestrator is shared by /ai and /agents routes.
	aiOrch := ai.NewOrchestrator(db, logger)

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
	aftersales.RegisterRoutes(protected, db, logger)
	sourcing1688.RegisterRoutes(protected, db, logger)
	report.RegisterRoutes(protected, db, logger)
	exchangerate.RegisterRoutes(protected, db, logger)

	// WebSocket route
	hub := realtime.NewHub(logger)
	go hub.Run()
	wsHandler := realtime.NewHandler(hub, logger)
	r.GET("/ws", wsHandler.ServeWS)

	// AI routes need the hub for realtime broadcasts.
	ai.RegisterRoutes(protected, db, logger, hub)

	return r
}
