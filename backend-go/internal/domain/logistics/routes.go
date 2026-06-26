package logistics

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers logistics routes on the given gin group.
// The routes require authentication (applied at the caller level).
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	// Phase 1: use default rate tables (embedded sample YAML).
	// Phase 2+: load from database or file.
	entries, err := LoadRateTableFromYAML([]byte(SampleRateTableYAML))
	if err != nil {
		logger.Warn("logistics: failed to load default rate table, using empty engine", zap.Error(err))
		entries = []RateTableEntry{}
	}

	svc := NewService(entries)
	handler := NewHandler(svc)

	// POST /logistics/quote — get on-demand rate quotes
	rg.POST("/logistics/quote", handler.GetQuotes)

	// GET /logistics/carriers — list configured carriers
	rg.GET("/logistics/carriers", handler.ListCarriers)
}
