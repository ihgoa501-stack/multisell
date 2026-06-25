package supplier

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers supplier routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	// Supplier CRUD
	suppliers := rg.Group("/suppliers")
	{
		suppliers.GET("", h.List)
		suppliers.GET("/:id", h.Get)
		suppliers.POST("", h.Create)
		suppliers.PUT("/:id", h.Update)
		suppliers.DELETE("/:id", h.Delete)
	}

	// Product-Supplier association CRUD
	ps := rg.Group("/product-suppliers")
	{
		ps.GET("", h.ListProductSuppliers)
		ps.POST("", h.CreateProductSupplier)
		ps.PUT("/:id", h.UpdateProductSupplier)
		ps.DELETE("/:id", h.DeleteProductSupplier)
	}
}
