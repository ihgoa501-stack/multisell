package sentiment

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles sentiment HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// GetProductSentiment GET /api/v1/sentiment/:productId
func (h *Handler) GetProductSentiment(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的商品ID")
		return
	}

	sentiment, err := h.service.GetSentiment(productID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	if sentiment == nil {
		response.Success(c, gin.H{"product_id": productID, "message": "暂无情感数据"})
		return
	}
	response.Success(c, sentiment)
}

// RefreshSentiment POST /api/v1/sentiment/:productId/refresh
func (h *Handler) RefreshSentiment(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的商品ID")
		return
	}

	sentiment, err := h.service.CalculateSentiment(productID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, sentiment)
}

// ListNegativeSentiment GET /api/v1/sentiment/negative
func (h *Handler) ListNegativeSentiment(c *gin.Context) {
	items, err := h.service.ListNegativeSentiment()
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, items)
}
