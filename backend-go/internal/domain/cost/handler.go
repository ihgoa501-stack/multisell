package cost

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles cost dashboard HTTP requests.
type Handler struct {
	service *Service
	budget  float64
}

// NewHandler creates a new cost handler.
func NewHandler(service *Service, budget float64) *Handler {
	return &Handler{service: service, budget: budget}
}

// Dashboard GET /cost/dashboard
func (h *Handler) Dashboard(c *gin.Context) {
	dash, err := h.service.GetDashboard(h.budget)
	if err != nil {
		c.Error(err)
		response.Error(c, http.StatusInternalServerError, "获取成本看板失败")
		return
	}
	response.Success(c, dash)
}
