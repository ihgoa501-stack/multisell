package loop

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles evaluation loop HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new loop handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Evaluate POST /loop/evaluate/:productId
func (h *Handler) Evaluate(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid productId")
		return
	}

	var in EvaluateInput
	c.ShouldBindJSON(&in)

	result, err := h.service.Evaluate(productID, in.TriggeredBy)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// GetRecommendations GET /loop/recommendations
func (h *Handler) GetRecommendations(c *gin.Context) {
	p := common.ParsePagination(c)
	decision := c.Query("decision")
	items, total, err := h.service.GetRecommendations(p.Page, p.Size, decision)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}
