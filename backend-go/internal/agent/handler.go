package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles agent HTTP requests.
type Handler struct {
	service      *Service
	orchestrator *ai.Orchestrator
}

// NewHandler creates a new agent handler. The orchestrator is optional and
// wires agent run endpoints through the AI runtime; pass nil to disable.
func NewHandler(service *Service, orchestrator *ai.Orchestrator) *Handler {
	return &Handler{service: service, orchestrator: orchestrator}
}

// ListAgents GET /agents
func (h *Handler) ListAgents(c *gin.Context) {
	response.Success(c, h.service.List())
}

// GetAgent GET /agents/:id
func (h *Handler) GetAgent(c *gin.Context) {
	id := c.Param("id")
	a, ok := h.service.Get(id)
	if !ok {
		response.Error(c, http.StatusNotFound, "agent not found")
		return
	}
	response.Success(c, a)
}

// CreateAgent POST /agents — registering new agents is not supported in v1;
// the roster is canonical. Returns 409 to guide callers.
func (h *Handler) CreateAgent(c *gin.Context) {
	response.Error(c, http.StatusConflict, "agent roster is canonical; use the existing A1-A7 / G1-G3 agents")
}

// ExecuteAction POST /agents/:id/actions — triggers an agent run via the AI
// orchestrator. Body: { "decision_point": "...", "context": {...}, "stream": false }
func (h *Handler) ExecuteAction(c *gin.Context) {
	id := c.Param("id")
	if _, ok := h.service.Get(id); !ok {
		response.Error(c, http.StatusNotFound, "agent not found")
		return
	}
	if h.orchestrator == nil {
		response.Error(c, http.StatusServiceUnavailable, "AI orchestrator not configured")
		return
	}
	var body struct {
		DecisionPoint string                 `json:"decision_point" binding:"required"`
		Context       map[string]interface{} `json:"context"`
		Stream        bool                   `json:"stream"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.orchestrator.Run(&ai.RunAgentRequest{
		AgentID:       id,
		DecisionPoint: body.DecisionPoint,
		Context:       body.Context,
		Stream:        body.Stream,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// Evolution GET /agents/evolution — returns evolution config placeholder.
func (h *Handler) Evolution(c *gin.Context) {
	response.Success(c, gin.H{
		"enabled":   true,
		"rules":     []string{"margin_floor", "stock_alert_threshold", "discount_ceiling"},
		"episodes":  0,
		"last_run":  nil,
	})
}

// Entropy GET /agents/entropy — returns rule entropy placeholder.
func (h *Handler) Entropy(c *gin.Context) {
	response.Success(c, gin.H{
		"rule_count":       42,
		"conflict_count":   1,
		"entropy_score":    0.12,
		"health":           "ok",
		"warnings":         []string{},
	})
}

// Ensure *gorm.DB import is retained even if future edits drop a direct use.
var _ = gorm.ErrRecordNotFound
