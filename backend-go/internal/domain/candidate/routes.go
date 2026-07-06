package candidate

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers candidate routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	r := rg.Group("/candidates")
	{
		r.GET("", h.List)
		r.GET("/collect-leads", h.ListCollectLeads)
		r.GET("/collect-leads/:id", h.GetCollectLead)
		r.GET("/:id", h.Get)
		r.POST("", h.Create)
		r.PUT("/:id", h.Update)
		r.DELETE("/:id", h.Delete)
		r.GET("/count", h.Count)
		r.GET("/dedup", h.Dedup)
		r.POST("/seed", h.Seed)

		// Field completion actions
		r.PUT("/:id/fields", h.FillFields)
		r.POST("/:id/skip-field", h.SkipField)
		r.POST("/:id/rescrape", h.Rescrape)
	}
}
