package ai

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles AI HTTP requests.
type Handler struct {
	service      *Service
	orchestrator *Orchestrator
	streamer     *Streamer
}

// NewHandler creates a new AI handler.
func NewHandler(service *Service, orchestrator *Orchestrator, streamer *Streamer) *Handler {
	return &Handler{service: service, orchestrator: orchestrator, streamer: streamer}
}

// Chat POST /ai/chat
// @Summary      AI chat
// @Description  Send a message to the AI assistant and get a response (supports streaming)
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        body  body  ChatInput  true  "Chat message"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /ai/chat [post]
func (h *Handler) Chat(c *gin.Context) {
	var in ChatInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	// Route through orchestrator.
	userID := userIDFromCtx(c)
	result, err := h.orchestrator.Chat(in.Message, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if in.Stream {
		answer := ""
		if s, ok := result.Output["recommendation"].(string); ok {
			answer = s
		}
		h.streamer.StreamChat(c, result.TraceID, result.AgentID, answer)
		return
	}
	response.Success(c, &ChatResponse{
		TraceID:    result.TraceID,
		AgentID:    result.AgentID,
		Answer:     stringify(result.Output["recommendation"]),
		Confidence: result.Confidence,
		RiskLevel:  result.RiskLevel,
		Actions:    actionsOrNil(result.Action),
	})
}

// RunAgent POST /ai/run
// @Summary      Run AI agent
// @Description  Execute an agent decision point with the given context
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        body  body  RunAgentRequest  true  "Agent run request"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /ai/run [post]
func (h *Handler) RunAgent(c *gin.Context) {
	var in RunAgentRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	in.UserID = userIDFromCtx(c)
	result, err := h.orchestrator.Run(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// ListTraces GET /ai/traces
// @Summary      List AI traces
// @Description  Get paginated list of AI agent execution traces
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        page           query  int     false  "Page number"
// @Param        size           query  int     false  "Page size"
// @Param        search         query  string  false  "Search keyword"
// @Param        agent_id       query  string  false  "Filter by agent ID"
// @Param        status         query  string  false  "Filter by status"
// @Param        decision_point query  string  false  "Filter by decision point"
// @Success      200  {object}  response.PageResult
// @Security     BearerAuth
// @Router       /ai/traces [get]
func (h *Handler) ListTraces(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &TraceListFilter{
		Search:        c.Query("search"),
		AgentID:       c.Query("agent_id"),
		Status:        c.Query("status"),
		DecisionPoint: c.Query("decision_point"),
	}
	items, total, err := h.service.ListTraces(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetTrace GET /ai/traces/:trace_id
// @Summary      Get AI trace
// @Description  Get a single execution trace by trace ID
// @Tags         ai
// @Produce      json
// @Param        trace_id  path  string  true  "Trace UUID"
// @Success      200       {object}  response.Result
// @Security     BearerAuth
// @Router       /ai/traces/{trace_id} [get]
func (h *Handler) GetTrace(c *gin.Context) {
	traceID := c.Param("trace_id")
	if traceID == "" {
		response.Error(c, http.StatusBadRequest, "trace_id required")
		return
	}
	detail, err := h.service.GetTrace(traceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "trace not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, detail)
}

// ListActions GET /ai/actions
func (h *Handler) ListActions(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &ActionListFilter{
		Search:   c.Query("search"),
		AgentID:  c.Query("agent_id"),
		Status:   c.Query("status"),
		RiskLevel: c.Query("risk_level"),
		SquadID:  c.Query("squad_id"),
	}
	items, total, err := h.service.ListActions(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetAction GET /ai/actions/:id
func (h *Handler) GetAction(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	a, err := h.service.GetAction(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "action not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, a)
}

// CreateAction POST /ai/actions
func (h *Handler) CreateAction(c *gin.Context) {
	var in CreateActionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	a, err := h.service.CreateAction(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, a)
}

// ApproveAction POST /ai/actions/:id/approve
func (h *Handler) ApproveAction(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ActionDecisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	a, err := h.service.ApproveAction(id, common.ReviewerFromCtx(c), in.Reason, common.UserIDFromCtx(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.broadcastActionUpdate(a, "action_approved")
	response.Success(c, a)
}

// RejectAction POST /ai/actions/:id/reject
func (h *Handler) RejectAction(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ActionDecisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	a, err := h.service.RejectAction(id, common.ReviewerFromCtx(c), in.Reason, common.UserIDFromCtx(c))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.broadcastActionUpdate(a, "action_rejected")
	response.Success(c, a)
}

// ExecuteAction POST /ai/actions/:id/execute
func (h *Handler) ExecuteAction(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ActionDecisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	a, err := h.service.ExecuteAction(id, common.ReviewerFromCtx(c), in.Reason, common.UserIDFromCtx(c))
	if err != nil {
		if errors.Is(err, ErrApprovalRequired) {
			response.Error(c, http.StatusForbidden, "action requires approval before execution")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.broadcastActionUpdate(a, "action_executed")
	response.Success(c, a)
}

// ReviewAction POST /ai/actions/:id/review
func (h *Handler) ReviewAction(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	a, err := h.service.ReviewAction(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, a)
}

// Roster GET /ai/agents
// @Summary      Agent roster
// @Description  Get list of all registered AI agents
// @Tags         ai
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /ai/agents [get]
func (h *Handler) Roster(c *gin.Context) {
	roster, err := h.service.Roster()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, roster)
}

// AgentSpecs GET /ai/agents/specs
// @Summary      Agent specs
// @Description  Get the full registry of agent specifications
// @Tags         ai
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /ai/agents/specs [get]
func (h *Handler) AgentSpecs(c *gin.Context) {
	response.Success(c, h.orchestrator.Registry().Agents)
}

// broadcastActionUpdate pushes an action lifecycle event to all WS clients.
func (h *Handler) broadcastActionUpdate(a *UnifiedAction, event string) {
	if h.streamer == nil {
		return
	}
	h.streamer.BroadcastEvent(&SSEEvent{
		Event:     event,
		TraceID:   a.TraceID,
		AgentID:   a.AgentID,
		Data:      map[string]interface{}{"action_id": a.ID, "status": a.Status, "title": a.Title},
		Timestamp: a.UpdatedAt,
	})
}

// parseID parses the :id path param.
func parseID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	if idStr == "" {
		response.Error(c, http.StatusBadRequest, "id required")
		return 0, false
	}
	var id int64
	for _, ch := range idStr {
		if ch < '0' || ch > '9' {
			response.Error(c, http.StatusBadRequest, "invalid id")
			return 0, false
		}
		id = id*10 + int64(ch-'0')
	}
	return id, true
}


// userIDFromCtx extracts user id from the JWT context (set by auth middleware).
func userIDFromCtx(c *gin.Context) *int64 {
	v, exists := c.Get("user_id")
	if !exists {
		return nil
	}
	switch x := v.(type) {
	case int64:
		return &x
	case int:
		n := int64(x)
		return &n
	case float64:
		n := int64(x)
		return &n
	}
	return nil
}

func stringify(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func actionsOrNil(a *UnifiedAction) []UnifiedAction {
	if a == nil {
		return nil
	}
	return []UnifiedAction{*a}
}
