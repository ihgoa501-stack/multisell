package operationlog

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers operationlog routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/operationlog")
	{
		group.GET("", h.List)
		group.GET("/:id", h.Get)
		// No POST route — audit entries are created exclusively by the
		// audit middleware (httpx/middleware/audit.go).
		// Exposing a public write endpoint undermines the audit chain of custody.
	}
}
