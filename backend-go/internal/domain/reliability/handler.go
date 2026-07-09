package reliability

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles reliability / budget HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new reliability handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetBudget GET /reliability/budget
// @Summary      Get LLM monthly budget
// @Description  View current LLM monthly budget and spend
// @Tags         reliability
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /reliability/budget [get]
func (h *Handler) GetBudget(c *gin.Context) {
	b, err := h.service.GetBudget()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, InBudgetResponse(b))
}

// SetBudget PUT /reliability/budget
// @Summary      Set LLM monthly budget
// @Description  Set the monthly spending limit for LLM API calls
// @Tags         reliability
// @Accept       json
// @Produce      json
// @Param        body  body  BudgetInput  true  "Budget settings"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /reliability/budget [put]
func (h *Handler) SetBudget(c *gin.Context) {
	var input BudgetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	b, err := h.service.SetBudget(input.MonthlyLimitUSD)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, InBudgetResponse(b))
}
