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
func (h *Handler) Activate(c *gin.Context) {
	var input ActivateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "reason is required"})
		return
	}
	Activate(input.Reason)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "kill switch activated",
		"reason":  input.Reason,
	})
}

// Deactivate disengages the kill switch.
func (h *Handler) Deactivate(c *gin.Context) {
	Deactivate()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "kill switch deactivated",
	})
}

// RegisterRoutes registers kill switch management endpoints under the given group.
func RegisterRoutes(rg *gin.RouterGroup) {
	h := NewHandler()
	ks := rg.Group("/kill-switch")
	{
		ks.GET("/status", h.GetStatus)
		ks.POST("/activate", h.Activate)
		ks.POST("/deactivate", h.Deactivate)
	}
}
