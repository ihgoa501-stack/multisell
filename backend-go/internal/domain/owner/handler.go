package owner

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
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
// @Summary      Risk summary
// @Description  Get owner cockpit risk summary
// @Tags         owner
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /owner/risk-summary [get]
func (h *Handler) RiskSummary(c *gin.Context) {
	summary, err := h.service.RiskSummary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, summary)
}

// Suggestions GET /owner/suggestions
// @Summary      Owner suggestions
// @Description  Get AI-generated suggestions for the business owner
// @Tags         owner
// @Produce      json
// @Param        limit  query  int  false  "Max items (default 20)"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /owner/suggestions [get]
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
// @Summary      Platform sync status
// @Description  Get sync status of all connected platform accounts
// @Tags         owner
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /owner/platform-sync [get]
func (h *Handler) PlatformSyncStatus(c *gin.Context) {
	items, err := h.service.PlatformSyncStatus()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// Feedback POST /owner/suggestions/:id/feedback
func (h *Handler) Feedback(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid suggestion id")
		return
	}

	var in FeedbackInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.RecordFeedback(id, &in); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "feedback recorded"})
}

// AgentActivity GET /owner/agent-activity
// @Summary      Agent activity
// @Description  Get recent agent activity timeline
// @Tags         owner
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /owner/agent-activity [get]
func (h *Handler) AgentActivity(c *gin.Context) {
	data, err := h.service.AgentActivity()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, data)
}

// PipelineChain GET /owner/pipeline-chain
func (h *Handler) PipelineChain(c *gin.Context) {
	data, err := h.service.PipelineChain()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, data)
}

// GetDecisionQueue GET /owner/decision-queue
// @Summary      Decision queue
// @Description  Get paginated decision queue with filtering, sorting, and status summary
// @Tags         owner
// @Produce      json
// @Param        display_status          query  string  false  "Filter by status: waiting_data, ready_for_decision, pending_approval, executing, completed, failed"
// @Param        min_completeness_score  query  number  false  "Minimum completeness score filter"
// @Param        min_profit_margin       query  number  false  "Minimum profit margin filter"
// @Param        platform_id             query  int     false  "Filter by target platform ID"
// @Param        destination_country     query  string  false  "Filter by destination country"
// @Param        search                  query  string  false  "Search by title or reason"
// @Param        sort_by                 query  string  false  "Sort field: completeness_score, profit_margin, estimated_profit, confidence, created_at"
// @Param        sort_order              query  string  false  "Sort order: asc or desc"
// @Param        page                    query  int     false  "Page number"
// @Param        size                    query  int     false  "Page size"
// @Success      200     {object}  response.Result
// @Security     BearerAuth
// @Router       /owner/decision-queue [get]
func (h *Handler) GetDecisionQueue(c *gin.Context) {
	filter := &DecisionQueueFilter{
		DisplayStatus:       c.Query("display_status"),
		MinCompletenessScore: parseQueryFloat(c.Query("min_completeness_score")),
		MinProfitMargin:      parseQueryFloat(c.Query("min_profit_margin")),
		PlatformID:           parseQueryInt(c.Query("platform_id")),
		DestinationCountry:   c.Query("destination_country"),
		Search:               c.Query("search"),
		SortBy:               c.Query("sort_by"),
		SortOrder:            c.Query("sort_order"),
	}
	p := common.ParsePagination(c)
	filter.Page = p.Page
	filter.Size = p.Size

	items, total, err := h.service.DecisionQueue(filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	summary, _ := h.service.DecisionQueueSummary()
	response.Success(c, gin.H{
		"items":   items,
		"total":   total,
		"page":    p.Page,
		"size":    p.Size,
		"summary": summary,
	})
}

// parseQueryFloat safely parses a float64 from a query string.
func parseQueryFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseQueryInt safely parses an int64 from a query string.
func parseQueryInt(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
