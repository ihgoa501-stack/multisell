package ai

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterLegacyReadRoutes keeps historical traces and actions available for
// audit without exposing the superseded A/G runtime or its mutation endpoints.
func RegisterLegacyReadRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	h := NewHandler(NewService(db, logger), nil, nil)

	ai := rg.Group("/ai", middleware.RequirePermission(db, "audit.read"))
	{
		ai.GET("/traces", h.ListTraces)
		ai.GET("/actions", h.ListActions)
		ai.GET("/traces/:trace_id", h.GetTrace)
		ai.GET("/actions/:id", h.GetAction)
	}
}
