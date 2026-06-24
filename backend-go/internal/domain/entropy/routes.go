package entropy

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers entropy routes under the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)
	e := rg.Group("/entropy")
	{
		e.GET("/dashboard", h.Dashboard)
		e.GET("/health", h.Health)
		e.GET("/defenses", h.Defenses)
		e.GET("/changes", h.Changes)
		spc := e.Group("/spc")
		{
			spc.GET("", h.Spc)
			spc.POST("/check", h.CheckPoint)
		}
	}
}
