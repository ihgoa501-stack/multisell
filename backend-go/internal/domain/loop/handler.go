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

// BatchEvaluate POST /loop/batch-evaluate
// @Summary      Batch evaluate products
// @Description  Run the full evaluation pipeline for multiple products at once
// @Tags         loop
// @Accept       json
// @Produce      json
// @Param        body  body  BatchEvaluateInput  true  "Batch evaluate input"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /loop/batch-evaluate [post]
func (h *Handler) BatchEvaluate(c *gin.Context) {
	var in BatchEvaluateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(in.ProductIDs) == 0 {
		response.Error(c, http.StatusBadRequest, "product_ids is required")
		return
	}
	results := h.service.BatchEvaluate(in.ProductIDs, in.TriggeredBy)
	response.Success(c, results)
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
