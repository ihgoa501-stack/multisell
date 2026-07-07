package integrations

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers integrations routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, approvalSvc *approval.Service) {
	svc := NewService(db, logger).WithApproval(approvalSvc)
	h := NewHandler(svc, approvalSvc)

	group := rg.Group("/platform-integrations")
	{
		// collection-level (static)
		group.GET("", h.List)
		group.POST("", h.Create)
		group.POST("/publish-to-ozon", h.PublishToOzon)
		group.POST("/write-back", h.WriteBack)
		group.POST("/write-back/:ref-id/retry", h.RetryWriteBack)

		// member-level (with :id)
		group.GET("/:id", h.Get)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
		group.POST("/:id/test", h.TestConnection)
		group.GET("/:id/mode", h.GetMode)
		group.PUT("/:id/mode", h.UpdateMode)
		group.POST("/:id/sync", h.Sync)
		group.GET("/:id/categories", h.ListCategories)
		group.POST("/:id/categories", h.CreateCategory)
		group.GET("/:id/attributes", h.ListAttributes)
		group.POST("/:id/attributes", h.CreateAttribute)
		group.GET("/:id/ozon-products", h.ListOzonProducts)
	}
}
