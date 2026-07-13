package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers integrations routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, approvalSvc *approval.Service) {
	// Dynamically register the stateful mock adapters if not already registered.
	if _, ok := GetAdapter("mock_ozon"); !ok {
		RegisterAdapter("mock_ozon", NewMockOzonAdapter(db, logger))
	}
	if _, ok := GetAdapter("mock_shopee"); !ok {
		RegisterAdapter("mock_shopee", NewMockShopeeAdapter(db, logger))
	}

	svc := NewService(db, logger).WithApproval(approvalSvc)
	h := NewHandler(svc, approvalSvc)

	group := rg.Group("/platform-integrations")
	{
		// collection-level (static)
		group.GET("", h.List)
		group.GET("/owner-fact-options", h.OwnerFactOptions)
		group.POST("", h.Create)
		group.POST("/publish-to-ozon", h.PublishToOzon)
		group.POST("/write-back", h.WriteBack)
		group.POST("/write-back/:ref-id/retry", h.RetryWriteBack)

		// Stateful mock storefront seeding route
		group.POST("/mock/seed", func(c *gin.Context) {
			mockSvc := NewMockService(db, logger)
			if err := mockSvc.SeedStorefront(db); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "message": "seeded stateful mock storefront"})
		})

		// member-level (with :id)
		group.GET("/:id", h.Get)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
		group.POST("/:id/test", h.TestConnection)
		group.GET("/:id/mode", h.GetMode)
		group.PUT("/:id/mode", h.UpdateMode)
		group.POST("/:id/sync", h.Sync)
		group.POST("/:id/order-events", h.IngestOrderEvent)
		group.GET("/:id/categories", h.ListCategories)
		group.POST("/:id/categories", h.CreateCategory)
		group.GET("/:id/attributes", h.ListAttributes)
		group.POST("/:id/attributes", h.CreateAttribute)
		group.GET("/:id/ozon-products", h.ListOzonProducts)
	}
}
