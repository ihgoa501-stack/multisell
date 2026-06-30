package owner

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles Owner cockpit HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new Owner handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RiskSummary GET /owner/risk-summary
func (h *Handler) RiskSummary(c *gin.Context) {
	summary, err := h.service.RiskSummary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, summary)
}

// Suggestions GET /owner/suggestions
func (h *Handler) Suggestions(c *gin.Context) {
	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	items, err := h.service.Suggestions(limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// PlatformSyncStatus GET /owner/platform-sync
func (h *Handler) DecisionQueue(c *gin.Context) {
	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	items, err := h.service.DecisionQueue(limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

func (h *Handler) PlatformSyncStatus(c *gin.Context) {
	items, err := h.service.PlatformSyncStatus()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}
