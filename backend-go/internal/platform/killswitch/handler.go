package killswitch

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for managing the production write kill switch.
type Handler struct{}

// NewHandler creates a kill switch HTTP handler.
func NewHandler() *Handler {
	return &Handler{}
}

// GetStatus returns the current kill switch state.
// Read-only, no special permission required beyond auth.
func (h *Handler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"active": IsActive(),
		"reason": Reason(),
	})
}

// ActivateInput is the JSON body for activating the kill switch.
type ActivateInput struct {
	Reason string `json:"reason" binding:"required"`
}

// Activate engages the kill switch with a reason.
// Requires RBAC permission (configured at route registration).
func (h *Handler) Activate(c *gin.Context) {
	var input ActivateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "reason is required"})
		return
	}

	Activate(input.Reason)

	// Audit: operator info from JWT (set by middleware.Auth).
	_ = c.GetString("username")
	operator := "anonymous"
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok && s != "" {
			operator = s
		}
	}
	logAudit(operator, "activate", input.Reason)

	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"message":  "kill switch activated",
		"reason":   input.Reason,
		"operator": operator,
	})
}

// Deactivate disengages the kill switch.
// Requires RBAC permission (configured at route registration).
func (h *Handler) Deactivate(c *gin.Context) {
	Deactivate()

	operator := "anonymous"
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok && s != "" {
			operator = s
		}
	}
	logAudit(operator, "deactivate", "")

	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"message":  "kill switch deactivated",
		"operator": operator,
	})
}

// logAudit writes a structured audit entry through the log (currently stdout).
// Upgrade to DB-backed operationlog.LogStructured when a DB reference is available
// at the killswitch package level.
func logAudit(operator, action, reason string) {
	logger := auditLogger()
	if logger == nil {
		return
	}
	switch action {
	case "activate":
		logger.Warn("kill switch activated",
			"operator", operator,
			"reason", reason,
		)
	case "deactivate":
		logger.Warn("kill switch deactivated",
			"operator", operator,
		)
	}
}

// RegisterRoutes registers kill switch management endpoints.
// To add RBAC, pass a subgroup with middleware.RequirePermission:
//
//	admin := protected.Group("", middleware.RequirePermission(db, "admin.system"))
//	killswitch.RegisterRoutes(admin)
func RegisterRoutes(rg *gin.RouterGroup) {
	h := NewHandler()
	ks := rg.Group("/kill-switch")
	{
		ks.GET("/status", h.GetStatus)
		ks.POST("/activate", h.Activate)
		ks.POST("/deactivate", h.Deactivate)
	}
}
