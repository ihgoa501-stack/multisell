package agentos

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles AgentOS HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new AgentOS handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Overview GET /agentos
func (h *Handler) Overview(c *gin.Context) {
	ov, err := h.service.Overview()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, ov)
}

// WorkItems GET /agentos/work-items
func (h *Handler) WorkItems(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	f := &WorkItemsFilter{
		Status:    c.Query("status"),
		RiskLevel: c.Query("risk_level"),
		AgentID:   c.Query("agent_id"),
		SquadID:   c.Query("squad_id"),
	}
	items, total, err := h.service.WorkItems(limit, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"items": items, "total": total})
}

// Autonomy GET /agentos/autonomy
func (h *Handler) Autonomy(c *gin.Context) {
	response.Success(c, h.service.Autonomy())
}

// Status GET /agentos/status
func (h *Handler) Status(c *gin.Context) {
	response.Success(c, gin.H{"status": "running", "version": "0.1.0"})
}

// WorkItemDetail GET /agentos/work-items/:id
func (h *Handler) WorkItemDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid work item id")
		return
	}
	detail, err := h.service.WorkItemDetail(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.Error(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, detail)
}

// AgentTimeline GET /agentos/agent-timeline
func (h *Handler) AgentTimeline(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	entries, err := h.service.AgentTimeline(limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, entries)
}

// FailedRuns GET /agentos/failures
func (h *Handler) FailedRuns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.service.FailedRuns(limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}
