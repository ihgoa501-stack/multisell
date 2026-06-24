package personalrule

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler exposes HTTP handlers for personal rule CRUD and rule application.
type Handler struct {
	service *Service
}

// NewHandler creates a new personal rule handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListRules godoc
// @Summary List personal rules for a user
// @Tags personal-rules
// GET /api/v1/agents/rules?user_id=1&agent_id=A5&decision_point=stock_alert
func (h *Handler) ListRules(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	agentID := c.Query("agent_id")
	decisionPoint := c.Query("decision_point")

	rules, err := h.service.ListRules(userID, agentID, decisionPoint)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if rules == nil {
		rules = []PersonalRule{}
	}
	response.Success(c, rules)
}

// GetRule godoc
// @Summary Get a single personal rule by ID
// @Tags personal-rules
// GET /api/v1/agents/rules/:id
func (h *Handler) GetRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var rule PersonalRule
	if err := h.service.db.First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "rule not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, rule)
}

// CreateRule godoc
// @Summary Create a new personal rule
// @Tags personal-rules
// POST /api/v1/agents/rules
func (h *Handler) CreateRule(c *gin.Context) {
	var rule PersonalRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.CreateRule(&rule); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, rule)
}

// UpdateRule godoc
// @Summary Update an existing personal rule
// @Tags personal-rules
// PUT /api/v1/agents/rules/:id
func (h *Handler) UpdateRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var rule PersonalRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	rule.ID = id
	if err := h.service.UpdateRule(&rule); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, rule)
}

// DeleteRule godoc
// @Summary Delete a personal rule
// @Tags personal-rules
// DELETE /api/v1/agents/rules/:id
func (h *Handler) DeleteRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.DeleteRule(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ApplyRulesInput is the JSON body for the apply endpoint.
type applyRulesJSON struct {
	UserID        int64                  `json:"user_id" binding:"required"`
	AgentID       string                 `json:"agent_id" binding:"required"`
	DecisionPoint string                 `json:"decision_point" binding:"required"`
	Output        map[string]interface{} `json:"output" binding:"required"`
}

// ApplyRules godoc
// @Summary Apply personal rules to a given output map
// @Tags personal-rules
// POST /api/v1/agents/rules/apply
func (h *Handler) ApplyRules(c *gin.Context) {
	var body applyRulesJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.ApplyRules(&ApplyRulesInput{
		UserID:        body.UserID,
		AgentID:       body.AgentID,
		DecisionPoint: body.DecisionPoint,
		Output:        body.Output,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}
