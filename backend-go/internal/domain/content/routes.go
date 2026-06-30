package content

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/ai"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers content routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger, aiOrch *ai.Orchestrator) {
	svc := NewService(aiOrch, logger)
	h := NewHandler(svc)

	group := rg.Group("/content")
	{
		group.POST("/generate", h.GenerateContent)
		group.POST("/validate", h.ValidateContent)
	}
}
