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
// @Summary      AgentOS overview
// @Description  Get AgentOS cockpit dashboard overview
// @Tags         agentos
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /agentos [get]
func (h *Handler) Overview(c *gin.Context) {
	ov, err := h.service.Overview()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, ov)
}

// WorkItems GET /agentos/work-items
// @Summary      List work items
// @Description  Get pending work items from all agents
// @Tags         agentos
// @Produce      json
// @Param        limit      query  int     false  "Max items (default 50)"
// @Param        status     query  string  false  "Filter by status"
// @Param        risk_level query  string  false  "Filter by risk level"
// @Param        agent_id   query  string  false  "Filter by agent ID"
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /agentos/work-items [get]
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
// @Summary      Agent autonomy status
// @Description  Get the autonomy configuration and current level for all agents
// @Tags         agentos
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /agentos/autonomy [get]
func (h *Handler) Autonomy(c *gin.Context) {
	response.Success(c, h.service.Autonomy())
}

// Status GET /agentos/status
// @Summary      AgentOS system status
// @Description  Get the current runtime status of the AgentOS
// @Tags         agentos
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /agentos/status [get]
func (h *Handler) Status(c *gin.Context) {
	response.Success(c, gin.H{"status": "running", "version": "0.1.0"})
}

// WorkItemDetail GET /agentos/work-items/:id
// @Summary      Get work item detail
// @Description  Get detail of a specific work item by ID
// @Tags         agentos
// @Produce      json
// @Param        id  path  int  true  "Work item ID"
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /agentos/work-items/{id} [get]
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

// TrafficSummary GET /agentos/traffic-summary
func (h *Handler) TrafficSummary(c *gin.Context) {
	summary, err := h.service.TrafficSummary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, summary)
}

// InterceptedActions GET /agentos/intercepted-actions
func (h *Handler) InterceptedActions(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, err := h.service.InterceptedActions(limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// AuditReplay GET /agentos/audit-replay/:correlation_id
func (h *Handler) AuditReplay(c *gin.Context) {
	correlationID := c.Param("correlation_id")
	if correlationID == "" {
		response.Error(c, http.StatusBadRequest, "correlation_id is required")
		return
	}
	replay, err := h.service.AuditReplay(correlationID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, replay)
}
