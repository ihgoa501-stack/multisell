package consolidation

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers consolidation routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewConsolidationService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/consolidation")
	{
		// Group CRUD
		group.POST("/groups", h.CreateGroup)
		group.GET("/groups", h.ListGroups)
		group.GET("/groups/:groupId", h.GetGroup)

		// Items within a group
		group.POST("/groups/:groupId/items", h.AddItem)
		group.GET("/groups/:groupId/items", h.GetGroupItems)
		group.DELETE("/groups/:groupId/items/:itemId", h.RemoveItem)

		// Negotiation
		group.POST("/groups/:groupId/negotiate", h.NegotiateGroup)
	}
}
