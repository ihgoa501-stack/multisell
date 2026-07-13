package businessdecision

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	h := &Handler{s: NewService(db)}
	read := rg.Group("/business-decisions", middleware.RequirePermission(db, "market.read"))
	read.GET("", h.List)
	read.GET("/fact-options", h.FactOptions)
	read.GET("/:id", h.Get)
	write := rg.Group("/business-decisions", middleware.RequirePermission(db, "market.write"))
	write.POST("", h.Create)
	write.POST("/:id/ai-recommendations", h.Recommend)
	decide := rg.Group("/business-decisions", middleware.RequirePermission(db, "market.decide"))
	decide.POST("/:id/owner-decisions", h.Decide)
}
