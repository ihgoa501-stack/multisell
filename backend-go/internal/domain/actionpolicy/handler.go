package actionpolicy

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// ──────────────────────────────────────────────────────────────
// PolicyRule handlers (existing)
// ──────────────────────────────────────────────────────────────

func (h *Handler) ListRules(c *gin.Context) {
	rules, err := h.service.ListRules()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, rules)
}

func (h *Handler) CreateRule(c *gin.Context) {
	var rule PolicyRule
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

func (h *Handler) UpdateRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var rule PolicyRule
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

func (h *Handler) HandleToggleRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.ToggleRule(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"toggled": true})
}

func (h *Handler) Evaluate(c *gin.Context) {
	var ctx ActionContext
	if err := c.ShouldBindJSON(&ctx); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.Evaluate(&ctx)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *Handler) GetRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var rule PolicyRule
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

// ──────────────────────────────────────────────────────────────
// ApprovalPolicy handlers (new)
// ──────────────────────────────────────────────────────────────

func (h *Handler) ListPolicies(c *gin.Context) {
	policies, err := h.service.ListPolicies()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, policies)
}

func (h *Handler) GetPolicy(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.service.GetPolicy(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "policy not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

func (h *Handler) CreatePolicy(c *gin.Context) {
	var p ApprovalPolicy
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.CreatePolicy(&p); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

func (h *Handler) UpdatePolicy(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var p ApprovalPolicy
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p.ID = id
	if err := h.service.UpdatePolicy(&p); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

func (h *Handler) DeletePolicy(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.DeletePolicy(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ──────────────────────────────────────────────────────────────
// ApprovalRequest handlers (new)
// ──────────────────────────────────────────────────────────────

func (h *Handler) ListRequests(c *gin.Context) {
	status := c.Query("status")
	reqs, err := h.service.ListRequests(status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, reqs)
}

func (h *Handler) GetRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	req, err := h.service.GetRequest(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "request not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, req)
}

func (h *Handler) HandleReview(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var in struct {
		Approve    bool   `json:"approve"`
		ReviewedBy string `json:"reviewed_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	req, err := h.service.Review(id, in.Approve, in.ReviewedBy)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, req)
}
