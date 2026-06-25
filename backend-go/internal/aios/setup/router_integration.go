package setup

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/aios/runtime"
	"github.com/lingmirror/backend-go/internal/platform/scheduler"
	"go.uber.org/zap"
)

// RegisterAIOSRoutes adds AIOS monitoring and management endpoints under the
// given Gin router group. Typically called from router.go with the protected
// API v1 group, e.g.:
//
//	cfg := setup.Initialize(db, bus, logger)
//	setup.RegisterAIOSRoutes(protected, cfg)
func RegisterAIOSRoutes(rg *gin.RouterGroup, cfg *Config) {
	// GET /api/v1/aios/health — AIOS subsystem health summary.
	rg.GET("/aios/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":        "ok",
			"runtime":       "initialized",
			"tools":         cfg.Registry.ToolCount(),
			"guardrails":    "configured",
			"agents":        len(cfg.Runtime.ListInstances()),
			"observability": cfg.Observability != nil,
		})
	})

	// GET /api/v1/aios/runtime/agents — list registered agent instances.
	rg.GET("/aios/runtime/agents", func(c *gin.Context) {
		instances := cfg.Runtime.ListInstances()
		type agentView struct {
			ID    string `json:"id"`
			State string `json:"state"`
		}
		view := make([]agentView, 0, len(instances))
		for _, inst := range instances {
			view = append(view, agentView{
				ID:    inst.Manifest.ID,
				State: inst.State.String(),
			})
		}
		c.JSON(200, gin.H{"agents": view})
	})

	// GET /api/v1/aios/tools — list all registered tools.
	rg.GET("/aios/tools", func(c *gin.Context) {
		toolList := cfg.Registry.List()
		type toolView struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
			Squad       string `json:"squad"`
			RiskLevel   string `json:"risk_level"`
		}
		view := make([]toolView, 0, len(toolList))
		for _, t := range toolList {
			view = append(view, toolView{
				Name:        t.Name,
				Version:     t.Version,
				Description: t.Description,
				Squad:       t.Squad,
				RiskLevel:   string(t.RiskLevel),
			})
		}
		c.JSON(200, gin.H{"tools": view, "count": len(toolList)})
	})
}

// parseDuration parses a human-readable interval string like "15m", "1h",
// "6h" into a time.Duration. Returns 0 and false on invalid input.
func parseDuration(s string) (time.Duration, bool) {
	if s == "" {
		return 0, false
	}
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, true
	}
	// Fallback: try numeric + unit manually for formats like "1hr".
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, false
	}
	unit := s[len(s)-1:]
	numStr := s[:len(s)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil {
		// Try two-char unit like "hr", "ms" etc.
		unit = s[len(s)-2:]
		numStr = s[:len(s)-2]
		n, err = strconv.Atoi(numStr)
		if err != nil {
			return 0, false
		}
	}
	switch unit {
	case "m", "min":
		return time.Duration(n) * time.Minute, true
	case "h", "hr":
		return time.Duration(n) * time.Hour, true
	case "s", "sec":
		return time.Duration(n) * time.Second, true
	default:
		return 0, false
	}
}

// SetupSchedulerAgentTriggers reads schedule-type triggers from all agents
// registered in the Runtime and creates corresponding scheduler.Task entries.
//
// Call this after all agents have been registered with Runtime.RegisterAgent
// but before the scheduler has started. If the scheduler is already running,
// Register will start each task immediately.
func SetupSchedulerAgentTriggers(sched *scheduler.Scheduler, rt *runtime.Runtime, logger *zap.Logger) {
	instances := rt.ListInstances()
	count := 0

	for _, inst := range instances {
		if inst.Manifest == nil {
			continue
		}
		for _, trigger := range inst.Manifest.Triggers {
			if trigger.Type != "schedule" {
				continue
			}
			d, ok := parseDuration(trigger.Interval)
			if !ok {
				logger.Warn("aios: skipping schedule trigger with unparsable interval",
					zap.String("agent_id", inst.Manifest.ID),
					zap.String("interval", trigger.Interval),
					zap.String("decision_point", trigger.DecisionPoint),
				)
				continue
			}

			taskID := "aios-" + inst.Manifest.ID + "-" + trigger.DecisionPoint

			sched.Register(scheduler.Task{
				ID:            taskID,
				AgentID:       inst.Manifest.ID,
				DecisionPoint: trigger.DecisionPoint,
				Interval:      d,
				Description:   inst.Manifest.Name + " / " + trigger.DecisionPoint,
			})
			count++
		}
	}

	logger.Info("aios: scheduler triggers registered",
		zap.Int("tasks", count),
		zap.Int("instances", len(instances)))
}
