package agentlearning

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles agent learning HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new agent learning handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetAllAccuracy GET /agent-learning/accuracy
func (h *Handler) GetAllAccuracy(c *gin.Context) {
	records, err := h.service.GetAllAccuracy()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if records == nil {
		records = []AgentAccuracy{}
	}
	response.Success(c, records)
}

// GetAccuracyByAgent GET /agent-learning/accuracy/:agentId
func (h *Handler) GetAccuracyByAgent(c *gin.Context) {
	agentID := c.Param("agentId")
	if agentID == "" {
		response.Error(c, http.StatusBadRequest, "agentId required")
		return
	}
	records, err := h.service.GetAccuracyByAgent(agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if records == nil {
		records = []AgentAccuracy{}
	}
	response.Success(c, records)
}

// ListEvaluations GET /agent-learning/evaluations
// Query params: agent_id, product_id
func (h *Handler) ListEvaluations(c *gin.Context) {
	agentID := c.Query("agent_id")
	var productID int64
	if pid := c.Query("product_id"); pid != "" {
		productID, _ = strconv.ParseInt(pid, 10, 64)
	}
	evals, err := h.service.ListEvaluations(agentID, productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if evals == nil {
		evals = []DecisionEvaluation{}
	}
	response.Success(c, evals)
}

// EvaluateDecision POST /agent-learning/evaluate
func (h *Handler) EvaluateDecision(c *gin.Context) {
	var req struct {
		DecisionTraceID int64  `json:"decision_trace_id" binding:"required"`
		Period          string `json:"period"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Period == "" {
		req.Period = "T+30"
	}
	if err := h.service.EvaluateDecision(req.DecisionTraceID, req.Period); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "ok"})
}

// RecalculateAccuracy POST /agent-learning/recalculate
func (h *Handler) RecalculateAccuracy(c *gin.Context) {
	var req struct {
		AgentID string `json:"agent_id" binding:"required"`
		Period  string `json:"period" binding:"required"` // 7d, 30d, 90d
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.RecalculateAccuracy(req.AgentID, req.Period); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "ok"})
}
