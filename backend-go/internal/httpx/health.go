package httpx

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/platform/scheduler"
	"gorm.io/gorm"
)

type readinessDependencies struct {
	db               interface{ PingContext(context.Context) error }
	eventBusRunning  func() bool
	schedulerRunning func() bool
	acceptingTraffic func() bool
	version          string
}

func registerHealthRoutes(r *gin.Engine, db *gorm.DB, bus *eventbus.Bus, sched *scheduler.Scheduler, acceptingTraffic func() bool, version string) {
	sqlDB, err := db.DB()
	if err != nil {
		// A nil pinger makes readiness fail closed while liveness remains useful.
		registerHealthHandlers(r, readinessDependencies{eventBusRunning: bus.IsRunning, schedulerRunning: sched.IsRunning, acceptingTraffic: acceptingTraffic, version: version})
		return
	}
	registerHealthHandlers(r, readinessDependencies{db: sqlDB, eventBusRunning: bus.IsRunning, schedulerRunning: sched.IsRunning, acceptingTraffic: acceptingTraffic, version: version})
}

func registerHealthHandlers(r *gin.Engine, deps readinessDependencies) {
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive", "version": deps.version})
	})
	r.GET("/api/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		dbReady := deps.db != nil && deps.db.PingContext(ctx) == nil
		busReady := deps.eventBusRunning != nil && deps.eventBusRunning()
		schedulerReady := deps.schedulerRunning != nil && deps.schedulerRunning()
		trafficReady := deps.acceptingTraffic != nil && deps.acceptingTraffic()
		status := http.StatusOK
		state := "ready"
		if !dbReady || !busReady || !schedulerReady || !trafficReady {
			status = http.StatusServiceUnavailable
			state = "not_ready"
		}
		c.JSON(status, gin.H{
			"status":  state,
			"version": deps.version,
			"components": gin.H{
				"database":  dbReady,
				"event_bus": busReady,
				"scheduler": schedulerReady,
				"traffic":   trafficReady,
			},
		})
	})
}
