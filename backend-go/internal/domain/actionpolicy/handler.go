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

func (h *Handler) ListRules(c *gin.Context) {
	rules, err := h.service.ListRules()
	if err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	response.Success(c, rules)
}

func (h *Handler) CreateRule(c *gin.Context) {
	var rule PolicyRule
	if err := c.ShouldBindJSON(&rule); err != nil { response.Error(c, http.StatusBadRequest, err.Error()); return }
	if err := h.service.CreateRule(&rule); err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	response.Success(c, rule)
}

func (h *Handler) UpdateRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 { response.Error(c, http.StatusBadRequest, "invalid id"); return }
	var rule PolicyRule
	if err := c.ShouldBindJSON(&rule); err != nil { response.Error(c, http.StatusBadRequest, err.Error()); return }
	rule.ID = id
	if err := h.service.UpdateRule(&rule); err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	response.Success(c, rule)
}

func (h *Handler) DeleteRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 { response.Error(c, http.StatusBadRequest, "invalid id"); return }
	if err := h.service.DeleteRule(id); err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	response.Success(c, gin.H{"deleted": true})
}

func (h *Handler) Evaluate(c *gin.Context) {
	var ctx ActionContext
	if err := c.ShouldBindJSON(&ctx); err != nil { response.Error(c, http.StatusBadRequest, err.Error()); return }
	result, err := h.service.Evaluate(&ctx)
	if err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	response.Success(c, result)
}

func (h *Handler) GetRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 { response.Error(c, http.StatusBadRequest, "invalid id"); return }
	var rule PolicyRule
	if err := h.service.db.First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { response.Error(c, http.StatusNotFound, "rule not found"); return }
		response.Error(c, http.StatusInternalServerError, err.Error()); return
	}
	response.Success(c, rule)
}
