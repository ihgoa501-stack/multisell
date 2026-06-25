package imagegen

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers imagegen routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/imagegen")
	{
		// Product image generation records
		group.GET("", h.ListImageGens)
		group.GET("/:id", h.GetImageGen)
		group.POST("", h.CreateImageGen)
		group.PUT("/:id/status", h.UpdateImageGenStatus)
		group.DELETE("/:id", h.DeleteImageGen)

		// Product canvases
		group.GET("/canvas", h.ListCanvases)
		group.POST("/canvas", h.CreateCanvas)
		group.GET("/canvas/:id", h.GetCanvas)
		group.PUT("/canvas/:id", h.UpdateCanvas)
		group.DELETE("/canvas/:id", h.DeleteCanvas)

		// Prompt templates
		group.GET("/templates", h.ListTemplates)
		group.POST("/templates", h.CreateTemplate)
		group.GET("/templates/:id", h.GetTemplate)
		group.PUT("/templates/:id", h.UpdateTemplate)
		group.POST("/templates/:id/use", h.UseTemplate)
		group.DELETE("/templates/:id", h.DeleteTemplate)
	}
}
