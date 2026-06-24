package agentrule

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler exposes HTTP handlers for personal rule CRUD and evaluation.
type Handler struct {
	service *Service
}

// NewHandler creates a new personal rule handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListRules godoc
// GET /api/v1/agent-rules?agent_id=A5&decision_point=stock_alert&rule_type=veto
func (h *Handler) ListRules(c *gin.Context) {
	uid, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未认证")
		return
	}
	var userID int64
	switch v := uid.(type) {
	case float64:
		userID = int64(v)
	case int64:
		userID = v
	case int:
		userID = int64(v)
	default:
		response.Error(c, http.StatusUnauthorized, "无效的用户身份")
		return
	}
	agentID := c.Query("agent_id")
	decisionPoint := c.Query("decision_point")
	ruleType := c.Query("rule_type")

	rules, err := h.service.List(userID, agentID, decisionPoint, ruleType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取个人规则失败")
		return
	}
	response.Success(c, rules)
}

// GetRule godoc
// GET /api/v1/agent-rules/:id
func (h *Handler) GetRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "规则ID无效")
		return
	}

	var rule PersonalRule
	if err := h.service.db.First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "规则不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "获取规则失败")
		return
	}
	response.Success(c, rule)
}

// CreateRule godoc
// POST /api/v1/agent-rules
func (h *Handler) CreateRule(c *gin.Context) {
	var input CreateRuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// Enforce user_id from JWT, ignore request body
	if uid, exists := c.Get("user_id"); exists {
		switch v := uid.(type) {
		case float64:
			input.UserID = int64(v)
		case int64:
			input.UserID = v
		case int:
			input.UserID = int64(v)
		}
	}

	rule, err := h.service.Create(&input)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "创建规则失败")
		return
	}
	response.Success(c, rule)
}

// UpdateRule godoc
// PUT /api/v1/agent-rules/:id
func (h *Handler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "规则ID无效")
		return
	}

	var input UpdateRuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	rule, err := h.service.Update(id, &input)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "规则不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "更新规则失败")
		return
	}
	response.Success(c, rule)
}

// DeleteRule godoc
// DELETE /api/v1/agent-rules/:id
func (h *Handler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "规则ID无效")
		return
	}

	if err := h.service.Delete(id); err != nil {
		if err.Error() == "规则不存在" {
			response.Error(c, http.StatusNotFound, "规则不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "删除规则失败")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ToggleRule godoc
// POST /api/v1/agent-rules/:id/toggle
func (h *Handler) ToggleRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "规则ID无效")
		return
	}

	rule, err := h.service.ToggleEnabled(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "规则不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "切换规则状态失败")
		return
	}
	response.Success(c, rule)
}

// EvaluateRulesInput is the JSON body for the evaluate endpoint.
type evaluateRulesJSON struct {
	UserID        int64                  `json:"user_id" binding:"required"`
	AgentID       string                 `json:"agent_id" binding:"required"`
	DecisionPoint string                 `json:"decision_point" binding:"required"`
	Output        map[string]interface{} `json:"output" binding:"required"`
}

// EvaluateRules godoc
// POST /api/v1/agent-rules/evaluate
func (h *Handler) EvaluateRules(c *gin.Context) {
	var body evaluateRulesJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// Enforce user_id from JWT, ignore request body
	if uid, exists := c.Get("user_id"); exists {
		switch v := uid.(type) {
		case float64:
			body.UserID = int64(v)
		case int64:
			body.UserID = v
		case int:
			body.UserID = int64(v)
		}
	}

	result, err := h.service.Evaluate(body.UserID, body.AgentID, body.DecisionPoint, body.Output)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "规则评估失败")
		return
	}
	response.Success(c, result)
}
