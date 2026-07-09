package producthub

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers all Product Hub routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	// Services
	masterSvc := NewMasterService(db, logger)
	variantSvc := NewVariantService(db, logger)
	offerSvc := NewSupplierOfferService(db, logger)
	sampleSvc := NewSampleService(db, logger)
	costSvc := NewCostVersionService(db, logger)
	aggrSvc := NewAggregationService(db, logger)
	svc := NewService(db, logger)
	versionSvc := NewVersionService(db, logger)
	freshnessSvc := NewFreshnessService(db, logger)
	relationSvc := NewRelationService(db, logger)
	h := NewHandler(svc, versionSvc, freshnessSvc, relationSvc, variantSvc, offerSvc, sampleSvc, costSvc, db)

	// Handlers
	masterH := NewMasterHandler(masterSvc)
	hubH := NewHubHandler(aggrSvc)

	// Product Hub master CRUD and sub-resources
	group := rg.Group("/product-hub")
	{
		// Master CRUD
		group.GET("", masterH.List)
		group.GET("/:id", masterH.Get)
		group.POST("", masterH.Create)
		group.PUT("/:id", masterH.Update)
		group.DELETE("/:id", masterH.Delete)

		// Lifecycle
		group.POST("/:id/transition", masterH.TransitionLifecycle)

		// Aggregation
		group.GET("/:id/hub", hubH.GetHub)

		// Evidence trace
		group.GET("/:id/evidence", h.GetEvidence)

		// Variants sub-resource
		group.GET("/:id/variants", h.ListVariants)
		group.POST("/variants", h.CreateVariant)

		// Supplier offers sub-resource
		group.GET("/:id/offers", h.ListOffers)
		group.POST("/offers", h.CreateOffer)

		// Sample requests sub-resource
		group.GET("/:id/samples", h.ListSamples)
		group.POST("/samples", h.CreateSample)

		// Cost versions sub-resource
		group.GET("/:id/costs", h.ListCosts)
		group.POST("/costs", h.CreateCost)
		group.POST("/costs/:costId/confirm", h.ConfirmCost)
	}

	// Product details sub-routes (version history, freshness, relations)
	productsGroup := rg.Group("/products")
	{
		// Version history endpoints
		productsGroup.GET("/:id/versions", h.ListVersions)
		productsGroup.GET("/:id/versions/:versionId", h.GetVersion)
		productsGroup.POST("/:id/versions/:versionId/rollback", h.Rollback)

		// Decision recording with automatic snapshot
		productsGroup.POST("/:id/decisions", h.RecordDecision)

		// Freshness endpoints
		productsGroup.GET("/:id/freshness", h.GetProductFreshness)
		productsGroup.GET("/freshness/stale", h.ListStaleProducts)
		productsGroup.POST("/:id/freshness/verify", h.VerifyDimension)

		// Product relation endpoints
		productsGroup.GET("/:id/relations", h.GetRelatedProducts)
		productsGroup.POST("/:id/discover-relations", h.AutoDiscoverRelations)

		// Product dashboard summary (360)
		productsGroup.GET("/360/summary", h.GetProductSummary)

		// Product recent decision traces
		productsGroup.GET("/decision", h.ListRecentDecisions)
	}
	productsGroup.POST("/relations", h.CreateRelation)
	productsGroup.DELETE("/relations/:id", h.DeleteRelation)
}
