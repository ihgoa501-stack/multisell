package sourcing

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers sourcing routes on the given router group.
// bridge is the ToolBridge for fetching product data; events is an optional EventPublisher.
// trendSources are optional MarketTrendSource implementations; if nil, default
// mock sources (AmazonBSRSource and KeywordTrendSource) are used.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, bridge ToolBridge, events EventPublisher, trendSources ...MarketTrendSource) {
	// Use default mock sources if none provided.
	sources := trendSources
	if len(sources) == 0 {
		sources = []MarketTrendSource{
			NewAmazonBSRSource(),
			NewKeywordTrendSource(),
		}
	}

	svc := NewService(db, logger, bridge, events, sources...)
	h := NewHandler(svc)

	group := rg.Group("/sourcing")
	{
		// r.GET("/api/v1/mock/unregistered-test")
		group.POST("/fetch", h.Fetch)
		group.GET("/recommendations", h.ListRecommendations)
		group.GET("/market-trends", h.MarketTrends)
		group.GET("/keyword-trends", h.KeywordTrends)
	}
}
