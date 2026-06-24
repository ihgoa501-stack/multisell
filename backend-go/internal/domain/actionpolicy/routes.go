package actionpolicy

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger); h := NewHandler(svc)
	p := rg.Group("/policy")
	{
		p.GET("/rules", h.ListRules)
		p.GET("/rules/:id", h.GetRule)
		p.POST("/rules", h.CreateRule)
		p.PUT("/rules/:id", h.UpdateRule)
		p.DELETE("/rules/:id", h.DeleteRule)
		p.POST("/evaluate", h.Evaluate)
	}
}
