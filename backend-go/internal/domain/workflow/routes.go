package workflow

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, bus *eventbus.Bus, aiOrch *ai.Orchestrator, dispatcher *command.Dispatcher, logger *zap.Logger) {
	eng := NewEngine(db, bus, aiOrch, dispatcher, logger)
	h := NewHandler(eng)

	group := rg.Group("/workflow")
	{
		// Def CRUD.
		group.GET("/defs", h.ListDefs)
		group.GET("/defs/:id", h.GetDef)
		group.POST("/defs", h.CreateDef)
		group.PUT("/defs/:id", h.UpdateDef)
		group.DELETE("/defs/:id", h.DeleteDef)

		// Run lifecycle.
		group.POST("/defs/:defId/start", h.StartRun)
		group.POST("/runs/:id/pause", h.PauseRun)
		group.POST("/runs/:id/resume", h.ResumeRun)
		group.GET("/runs", h.ListRuns)
		group.GET("/runs/:id", h.GetRun)

		// External step advancement.
		group.POST("/runs/:id/advance", h.AdvanceStep)

		// Monitoring.
		group.GET("/monitor/stats", h.GetMonitorStats)
	}
}
