package entropy

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	ent := rg.Group("/entropy")
	{
		ent.GET("", h.GetSummary)
		ent.POST("/defense", h.RunDefenses)
		ent.GET("/health", h.GetHealthScores)
		ent.GET("/spc", h.GetSpcStatus)
		ent.GET("/changelog", h.GetChangeLog)
	}
}
