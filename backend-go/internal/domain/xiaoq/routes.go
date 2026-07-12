package xiaoq

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/domain/demandcase"
	"github.com/lingmirror/backend-go/internal/domain/experiment"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	provider := ai.NewLLMProvider(logger)
	experimentService := experiment.NewService(db, logger)
	service := NewService(db, logger, demandcase.NewService(db, logger), experimentService, provider, ai.NewTraceWriter(db, logger)).WithSourcingReader(sourcing1688.NewService(db, logger)).WithBusinessClosureReader(experimentService)
	h := NewHandler(service)
	g := rg.Group("/xiao-q", middleware.RequirePermission(db, "agent.read"))
	g.GET("/identity", h.Identity)
	g.GET("/capabilities", middleware.RequirePermission(db, "agent.write"), h.Capabilities)
	g.GET("/traces/:trace_id", h.Trace)
	g.POST("/messages", middleware.RequirePermission(db, "agent.write"), h.Message)
}

func RegisterHandlerRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("/xiao-q")
	g.GET("/identity", h.Identity)
	g.GET("/capabilities", h.Capabilities)
	g.POST("/messages", h.Message)
	g.GET("/traces/:trace_id", h.Trace)
}
