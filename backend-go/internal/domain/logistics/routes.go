package logistics

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"go.uber.org/zap"
)

// RegisterRoutes registers logistics routes on the given router group.
// The logistics module is stateless (YAML-driven rate tables), so db is unused.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	// For now, use an empty rate table — production will load from YAML/DB.
	svc := NewService(nil)
	h := NewHandler(svc, logger)

	group := rg.Group("/logistics")
	{
		group.POST("/quote", h.GetQuotes)
	}
}
