package price

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers price routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	// Price CRUD routes
	prices := rg.Group("/prices")
	{
		prices.GET("", h.ListPrices)
		prices.GET("/:id", h.GetPrice)
		prices.POST("", h.SetPrice)
		prices.PUT("/:id", h.UpdatePrice)
		prices.DELETE("/:id", h.DeletePrice)
	}

	// SKU-scoped price routes
	skus := rg.Group("/skus")
	{
		skus.GET("/:id/prices", h.ListPricesBySKU)
		skus.GET("/:id/current-price", h.GetCurrentPrice)
		skus.GET("/:id/price-history", h.PriceHistory)
	}

	// Competitor Price routes
	cp := rg.Group("/competitor-prices")
	{
		cp.GET("", h.ListCompetitorPrices)
		cp.GET("/:id", h.GetCompetitorPrice)
		cp.POST("", h.CreateCompetitorPrice)
		cp.DELETE("/:id", h.DeleteCompetitorPrice)
	}

	// Pricing Strategy routes
	ps := rg.Group("/pricing-strategies")
	{
		ps.GET("", h.ListPricingStrategies)
		ps.GET("/:id", h.GetPricingStrategy)
		ps.POST("", h.SavePricingStrategy)
		ps.PUT("/:id", h.UpdatePricingStrategy)
		ps.DELETE("/:id", h.DeletePricingStrategy)
	}

	// Pricing Recommendation routes
	pr := rg.Group("/pricing-recommendations")
	{
		pr.GET("", h.ListRecommendations)
		pr.POST("/generate", h.GenerateRecommendation)
		pr.POST("/:id/apply", h.ApplyRecommendation)
	}
}
