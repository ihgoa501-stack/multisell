package producthub

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers product hub routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	versionSvc := NewVersionService(db, logger)
	freshnessSvc := NewFreshnessService(db, logger)
	relationSvc := NewRelationService(db, logger)
	h := NewHandler(svc, versionSvc, freshnessSvc, relationSvc)

	group := rg.Group("/products")
	{
		// Version history endpoints
		group.GET("/:id/versions", h.ListVersions)
		group.GET("/:id/versions/:versionId", h.GetVersion)
		group.POST("/:id/versions/:versionId/rollback", h.Rollback)

		// Decision recording with automatic snapshot
		group.POST("/:id/decisions", h.RecordDecision)

		// Freshness endpoints
		group.GET("/:id/freshness", h.GetProductFreshness)
		group.GET("/freshness/stale", h.ListStaleProducts)
		group.POST("/:id/freshness/verify", h.VerifyDimension)

		// Product relation endpoints
		group.GET("/:id/relations", h.GetRelatedProducts)
		group.POST("/:id/discover-relations", h.AutoDiscoverRelations)
	}
	group.POST("/relations", h.CreateRelation)
	group.DELETE("/relations/:id", h.DeleteRelation)
}
