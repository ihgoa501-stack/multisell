package compliance

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers compliance routes on the given group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	g := rg.Group("/compliance")
	{
		g.POST("/check", h.Check)
		g.POST("/scan", h.Scan)
		g.GET("/results", h.ListResults)
		g.GET("/results/:id", h.GetResult)
		g.PUT("/results/:id/suppress", h.SuppressResult)
	}
}
