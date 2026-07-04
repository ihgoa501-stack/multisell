package sku

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers SKU and Product routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	// Product routes
	products := rg.Group("/product-master")
	{
		products.GET("", h.ListProducts)
		products.GET("/:id", h.GetProduct)
		products.POST("", h.CreateProduct)
		products.PUT("/:id", h.UpdateProduct)
		products.DELETE("/:id", h.DeleteProduct)

		// Spec management under a product
		products.GET("/:id/specs", h.ListSpecs)
		products.POST("/:id/specs", h.CreateSpec)
		products.PUT("/:id/specs/:spec_id", h.UpdateSpec)
		products.DELETE("/:id/specs/:spec_id", h.DeleteSpec)

		// Spec value creation under a spec
		products.POST("/:id/specs/:spec_id/values", h.CreateSpecValue)

		// SKU listing per product
		products.GET("/:id/skus", h.ListSkusByProduct)
	}

	// Spec value routes (top-level for update/delete by ID)
	specValues := rg.Group("/spec-values")
	{
		specValues.PUT("/:id", h.UpdateSpecValue)
		specValues.DELETE("/:id", h.DeleteSpecValue)
	}

	// SKU routes
	skus := rg.Group("/skus")
	{
		skus.GET("", h.ListSkus)
		skus.GET("/:id", h.GetSku)
		skus.POST("", h.CreateSku)
		skus.PUT("/:id", h.UpdateSku)
		skus.DELETE("/:id", h.DeleteSku)
	}
}
