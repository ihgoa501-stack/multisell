package trustscore

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	ts := rg.Group("/trust-scores")
	{
		ts.GET("", h.List)
		ts.GET("/:agent_id", h.GetByAgent)
		ts.POST("/recalculate", h.Recalculate)
		ts.POST("/eligible", h.Eligible)
		ts.PUT("/:agent_id/level", h.UpdateLevel)
		ts.POST("/auto-upgrade", h.AutoUpgrade)
		ts.GET("/summary", h.Summary)
	}
}
