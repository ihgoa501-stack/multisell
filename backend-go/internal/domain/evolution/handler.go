package evolution

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListNudges GET /evolution/nudges
func (h *Handler) ListNudges(c *gin.Context) {
	userIDStr := c.Query("user_id")
	var userID *int64
	if userIDStr != "" {
		if id, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			userID = &id
		}
	}
	agentID := c.Query("agent_id")
	status := c.Query("status")

	nudges, err := h.service.ListNudges(userID, agentID, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, nudges)
}

// EvaluateNudges POST /evolution/nudges/evaluate
func (h *Handler) EvaluateNudges(c *gin.Context) {
	nudges, err := h.service.EvaluateNudges()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{
		"nudges_created": len(nudges),
		"nudges":         nudges,
	})
}

// AcceptNudge POST /evolution/nudges/:id/accept
func (h *Handler) AcceptNudge(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.AcceptNudge(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "accepted"})
}

// DismissNudge POST /evolution/nudges/:id/dismiss
func (h *Handler) DismissNudge(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.DismissNudge(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "dismissed"})
}
