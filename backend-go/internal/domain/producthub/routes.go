package producthub

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
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
	h := NewHandler(svc, versionSvc, freshnessSvc, relationSvc)

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

		// Variants sub-resource
		group.GET("/:id/variants", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "invalid id")
				return
			}
			items, err := variantSvc.ListByMaster(c.Request.Context(), id)
			if err != nil {
				response.InternalError(c, err)
				return
			}
			response.Success(c, items)
		})
		group.POST("/variants", func(c *gin.Context) {
			var v ProductVariant
			if err := c.ShouldBindJSON(&v); err != nil {
				response.Error(c, http.StatusBadRequest, err.Error())
				return
			}
			if err := variantSvc.Create(c.Request.Context(), &v); err != nil {
				response.InternalError(c, err)
				return
			}
			response.Success(c, v)
		})

		// Supplier offers sub-resource
		group.GET("/:id/offers", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "invalid id")
				return
			}
			items, err := offerSvc.ListByMaster(c.Request.Context(), id)
			if err != nil {
				response.InternalError(c, err)
				return
			}
			response.Success(c, items)
		})
		group.POST("/offers", func(c *gin.Context) {
			var o SupplierOffer
			if err := c.ShouldBindJSON(&o); err != nil {
				response.Error(c, http.StatusBadRequest, err.Error())
				return
			}
			if err := offerSvc.Create(c.Request.Context(), &o); err != nil {
				response.InternalError(c, err)
				return
			}
			response.Success(c, o)
		})

		// Sample requests sub-resource
		group.GET("/:id/samples", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "invalid id")
				return
			}
			items, err := sampleSvc.ListByMaster(c.Request.Context(), id)
			if err != nil {
				response.InternalError(c, err)
				return
			}
			response.Success(c, items)
		})
		group.POST("/samples", func(c *gin.Context) {
			var sr SampleRequest
			if err := c.ShouldBindJSON(&sr); err != nil {
				response.Error(c, http.StatusBadRequest, err.Error())
				return
			}
			if err := sampleSvc.Create(c.Request.Context(), &sr); err != nil {
				response.InternalError(c, err)
				return
			}
			response.Success(c, sr)
		})

		// Cost versions sub-resource
		group.GET("/:id/costs", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "invalid id")
				return
			}
			items, err := costSvc.ListByMaster(c.Request.Context(), id)
			if err != nil {
				response.InternalError(c, err)
				return
			}
			response.Success(c, items)
		})
		group.POST("/costs", func(c *gin.Context) {
			var cv CostVersion
			if err := c.ShouldBindJSON(&cv); err != nil {
				response.Error(c, http.StatusBadRequest, err.Error())
				return
			}
			if err := costSvc.Create(c.Request.Context(), &cv); err != nil {
				response.InternalError(c, err)
				return
			}
			response.Success(c, cv)
		})
		group.POST("/costs/:costId/confirm", func(c *gin.Context) {
			costID, err := strconv.ParseInt(c.Param("costId"), 10, 64)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "invalid cost id")
				return
			}
			if err := costSvc.Confirm(c.Request.Context(), costID); err != nil {
				response.InternalError(c, err)
				return
			}
			response.Success(c, gin.H{"id": costID, "status": "confirmed"})
		})
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
	}
	productsGroup.POST("/relations", h.CreateRelation)
	productsGroup.DELETE("/relations/:id", h.DeleteRelation)
}
