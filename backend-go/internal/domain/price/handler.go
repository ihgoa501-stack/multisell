package price

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles price HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new price handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListPrices returns a paginated list of prices.
// GET /api/v1/prices?page=1&size=20&sku_id=&price_type=
func (h *Handler) ListPrices(c *gin.Context) {
	p := common.ParsePagination(c)
	skuID, _ := strconv.ParseInt(c.Query("sku_id"), 10, 64)
	priceType := c.Query("price_type")

	items, total, err := h.service.ListPrices(c.Request.Context(), p.Page, p.Size, skuID, priceType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list prices: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetPrice returns a single price by ID.
// GET /api/v1/prices/:id
func (h *Handler) GetPrice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetPriceByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "price not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get price: "+err.Error())
		return
	}

	response.Success(c, item)
}

// SetPrice creates or updates a price.
// POST /api/v1/prices
func (h *Handler) SetPrice(c *gin.Context) {
	var p Price
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if err := h.service.SetPrice(c.Request.Context(), &p, operator); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to set price: "+err.Error())
		return
	}

	response.Success(c, p)
}

// UpdatePrice updates a price.
// PUT /api/v1/prices/:id
func (h *Handler) UpdatePrice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var p Price
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	p.ID = id

	if err := h.service.UpdatePrice(c.Request.Context(), &p); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update price: "+err.Error())
		return
	}

	response.Success(c, p)
}

// DeletePrice deletes a price.
// DELETE /api/v1/prices/:id
func (h *Handler) DeletePrice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeletePrice(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete price: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// ListPricesBySKU returns all prices for a SKU.
// GET /api/v1/skus/:id/prices
func (h *Handler) ListPricesBySKU(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	items, err := h.service.ListPricesBySKU(c.Request.Context(), skuID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list prices: "+err.Error())
		return
	}

	response.Success(c, items)
}

// GetCurrentPrice returns the current sale price for a SKU.
// GET /api/v1/skus/:id/current-price
func (h *Handler) GetCurrentPrice(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetCurrentPrice(c.Request.Context(), skuID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "no active price found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get current price: "+err.Error())
		return
	}

	response.Success(c, item)
}

// PriceHistory returns the price change history for a SKU.
// GET /api/v1/skus/:id/price-history
func (h *Handler) PriceHistory(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	p := common.ParsePagination(c)

	items, total, err := h.service.ListChangeLogs(c.Request.Context(), skuID, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get price history: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}
