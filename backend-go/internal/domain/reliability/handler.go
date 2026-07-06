package reliability

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles reliability HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new reliability handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetBudget GET /api/v1/reliability/budget
func (h *Handler) GetBudget(c *gin.Context) {
	b, err := h.service.GetBudget()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, b)
}

// SetBudget PUT /api/v1/reliability/budget
func (h *Handler) SetBudget(c *gin.Context) {
	var in BudgetConfig
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.SetBudget(in.MonthlyLimitUSD); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"monthly_limit_usd": in.MonthlyLimitUSD})
}
