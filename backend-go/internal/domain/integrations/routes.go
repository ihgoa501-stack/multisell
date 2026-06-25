package integrations

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers integrations routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/platform-integrations")
	{
		// collection-level (static)
		group.GET("", h.List)
		group.POST("", h.Create)

		// member-level (with :id)
		group.GET("/:id", h.Get)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
		group.POST("/:id/test", h.TestConnection)
		group.POST("/:id/sync", h.Sync)
		group.GET("/:id/categories", h.ListCategories)
		group.POST("/:id/categories", h.CreateCategory)
		group.GET("/:id/attributes", h.ListAttributes)
		group.POST("/:id/attributes", h.CreateAttribute)
		group.GET("/:id/ozon-products", h.ListOzonProducts)
	}
}
