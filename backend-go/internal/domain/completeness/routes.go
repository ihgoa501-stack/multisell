package completeness

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/exchangerate"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers completeness check routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	// Create profit service for economic estimates (nil-able; same pattern as profit.RegisterRoutes).
	rateSvc := exchangerate.NewService(db, logger)
	profitSvc := profit.NewService(db, logger, rateSvc, 7.2)

	svc := NewService(db, logger)
	svc.profitSvc = profitSvc
	h := NewHandler(svc)

	r := rg.Group("/completeness")
	{
		r.GET("/checks", h.ListChecks)
		r.POST("/check/:productId", h.Check)
	}

	// Enhanced completeness report with economic estimates.
	rg.POST("/candidates/:id/completeness", h.CheckEnhanced)
}
