package personalrule

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers personal rule routes under the given group.
// Routes are mounted at /agents/rules.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	g := rg.Group("/agents/rules")
	{
		g.GET("", h.ListRules)
		g.GET("/:id", h.GetRule)
		g.POST("", h.CreateRule)
		g.PUT("/:id", h.UpdateRule)
		g.DELETE("/:id", h.DeleteRule)
		g.POST("/apply", h.ApplyRules)
	}
}
