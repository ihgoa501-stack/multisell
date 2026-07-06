package reliability

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles reliability HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new reliability handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetAgentStatus returns all agent heartbeat statuses.
// GET /api/v1/reliability/agent-status
func (h *Handler) GetAgentStatus(c *gin.Context) {
	statuses, err := h.service.GetAgentStatus(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get agent status: "+err.Error())
		return
	}
	response.Success(c, AgentStatusResponse{
		Statuses: statuses,
		Total:    int64(len(statuses)),
	})
}

// GetLLMCost returns LLM cost summary for a given period.
// GET /api/v1/reliability/llm-cost?period=today|week|month
func (h *Handler) GetLLMCost(c *gin.Context) {
	period := c.DefaultQuery("period", "today")
	resp, err := h.service.GetLLMCost(c.Request.Context(), period)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get LLM cost: "+err.Error())
		return
	}
	response.Success(c, resp)
}

// GetFailures returns pending failure records.
// GET /api/v1/reliability/failures
func (h *Handler) GetFailures(c *gin.Context) {
	records, err := h.service.GetFailures(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get failures: "+err.Error())
		return
	}
	response.Success(c, records)
}

// ResolveFailure marks a failure record as resolved.
// POST /api/v1/reliability/failures/:id/resolve
func (h *Handler) ResolveFailure(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.ResolveFailure(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to resolve failure: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
