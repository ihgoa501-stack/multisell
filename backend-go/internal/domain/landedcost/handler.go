package landedcost

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles landedcost HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new landedcost handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Calculate POST /landed-cost/calculate
func (h *Handler) Calculate(c *gin.Context) {
	var req CalculateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.service.Calculate(&req)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, res)
}

// GetLandedCost GET /landed-cost/:productId
func (h *Handler) GetLandedCost(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid product_id")
		return
	}

	platformIDStr := c.Query("platform")
	if platformIDStr != "" {
		platformID, err := strconv.ParseInt(platformIDStr, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid platform_id")
			return
		}
		lc, err := h.service.GetByProductPlatform(productID, platformID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.Error(c, http.StatusNotFound, "no landed cost found for this product/platform")
				return
			}
			response.InternalError(c, err)
			return
		}
		response.Success(c, lc)
		return
	}

	// No platform filter — return all records for this product
	items, err := h.service.ListByProduct(productID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, items)
}

// CompareAcrossPlatforms GET /landed-cost/:productId/compare
func (h *Handler) CompareAcrossPlatforms(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid product_id")
		return
	}

	items, err := h.service.CompareAcrossPlatforms(productID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, items)
}
