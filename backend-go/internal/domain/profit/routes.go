package profit

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/exchangerate"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers profit summary routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	rateSvc := exchangerate.NewService(db, logger)
	svc := NewService(db, logger, rateSvc, 7.2)
	h := NewHandler(svc)

	r := rg.Group("/profit")
	{
		r.GET("/summaries", h.ListSummaries)
		r.GET("/summary/:productId", h.Summary)
		r.POST("/order/:orderId/calculate", h.CalculateOrderProfit)
	}
}
