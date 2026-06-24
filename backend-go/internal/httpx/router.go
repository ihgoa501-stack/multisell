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
	"github.com/lingmirror/backend-go/internal/feedback"
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

	// Auth routes (public + protected)
	auth.RegisterRoutes(api, db, cfg, logger)

	// RBAC routes
	rbac.RegisterRoutes(api, db, logger)

	// AI orchestrator is shared by /ai and /agents routes.
	aiOrch := ai.NewOrchestrator(db, logger)

	// Agent routes (wired through the AI orchestrator)
	agent.RegisterRoutes(api, db, logger, aiOrch)

	// AgentOS routes
	agentos.RegisterRoutes(api, db, logger)

	// Domain routes
	category.RegisterRoutes(api, db, logger)
	brand.RegisterRoutes(api, db, logger)
	sku.RegisterRoutes(api, db, logger)
	price.RegisterRoutes(api, db, logger)
	inventory.RegisterRoutes(api, db, logger)
	supplier.RegisterRoutes(api, db, logger)
	platform.RegisterRoutes(api, db, logger)
	listing.RegisterRoutes(api, db, logger)
	listingtask.RegisterRoutes(api, db, logger)
	shipping.RegisterRoutes(api, db, logger)
	platformfee.RegisterRoutes(api, db, logger)
	order.RegisterRoutes(api, db, logger)
	orderimport.RegisterRoutes(api, db, logger)
	settlement.RegisterRoutes(api, db, logger)
	finance.RegisterRoutes(api, db, logger)
	decision.RegisterRoutes(api, db, logger)
	allocation.RegisterRoutes(api, db, logger)
	exceptions.RegisterRoutes(api, db, logger)
	notification.RegisterRoutes(api, db, logger)
	dashboard.RegisterRoutes(api, db, logger)
	search.RegisterRoutes(api, db, logger)
	imagegen.RegisterRoutes(api, db, logger)
	importbatch.RegisterRoutes(api, db, logger)
	operationlog.RegisterRoutes(api, db, logger)
	integrations.RegisterRoutes(api, db, logger)
	aftersales.RegisterRoutes(api, db, logger)
	sourcing1688.RegisterRoutes(api, db, logger)
	report.RegisterRoutes(api, db, logger)
	exchangerate.RegisterRoutes(api, db, logger)

	// WebSocket route
	hub := realtime.NewHub(logger)
	go hub.Run()
	wsHandler := realtime.NewHandler(hub, logger)
	r.GET("/ws", wsHandler.ServeWS)

	// AI routes need the hub for realtime broadcasts.
	ai.RegisterRoutes(api, db, logger, hub)

	// Feedback routes with full AgentOS integration
	feedback.RegisterRoutes(api, cfg, db, logger,
		// AI classification function
		func(ctx context.Context, system, user string) (string, error) {
			resp, err := aiOrch.Provider().Chat(ctx, &ai.LLMRequest{
				System:      system,
				Messages:    []ai.LLMMessage{{Role: "user", Content: user}},
				Temperature: 0.1,
				MaxTokens:   300,
			})
			if err != nil {
				return "", err
			}
			return resp.Answer, nil
		},
		// WebSocket hub for real-time notifications
		hub,
		// UnifiedAction creator for AgentOS integration
		func(table, sourceID, title, payload string) error {
			aiSvc := ai.NewService(db, logger)
			_, err := aiSvc.CreateAction(&ai.CreateActionInput{
				SourceTable:  table,
				SourceID:     sourceID,
				SourceType:   "feedback",
				ActionType:   "feedback_triage",
				Title:        title,
				Description:  payload,
				AgentID:      "A8",
				SquadID:      "governance",
				RiskLevel:    "medium",
				RequiresApproval: boolPtr(true),
			})
			return err
		},
	)
	return r
}

func boolPtr(b bool) *bool { return &b }
