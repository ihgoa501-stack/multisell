package businessfeedback

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, _ *zap.Logger, dispatcher *command.Dispatcher, policy command.PolicyChecker, capabilities []string) {
	h := &Handler{svc: NewService(db, dispatcher, policy, capabilities)}
	g := r.Group("/business-feedback", middleware.RequirePermission(db, "business_feedback.owner"))
	g.GET("/actions", h.List)
	g.GET("/actions/:id", h.Get)
	g.POST("/actions", h.CreateAction)
	g.POST("/actions/:id/execute", h.Execute)
	g.POST("/actions/:id/observations", h.Observe)
	g.POST("/actions/:id/next-recommendations", h.Recommend)
}
