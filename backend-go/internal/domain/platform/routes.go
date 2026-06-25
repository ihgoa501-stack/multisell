package platform

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers platform & store routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	platforms := rg.Group("/platforms")
	{
		platforms.GET("", h.ListPlatforms)
		platforms.GET("/:id", h.GetPlatform)
		platforms.POST("", h.CreatePlatform)
		platforms.PUT("/:id", h.UpdatePlatform)
		platforms.DELETE("/:id", h.DeletePlatform)
	}

	stores := rg.Group("/stores")
	{
		stores.GET("", h.ListStores)
		stores.GET("/:id", h.GetStore)
		stores.POST("", h.CreateStore)
		stores.PUT("/:id", h.UpdateStore)
		stores.DELETE("/:id", h.DeleteStore)
	}
}
