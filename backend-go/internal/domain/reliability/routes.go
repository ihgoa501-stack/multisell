package reliability

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers reliability routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)
	group := rg.Group("/reliability")
	{
		group.GET("/budget", h.GetBudget)
		group.PUT("/budget", h.SetBudget)
	}
}
