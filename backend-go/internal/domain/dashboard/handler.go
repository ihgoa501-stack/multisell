package dashboard

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles dashboard HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new dashboard handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Overview GET /dashboard/overview
// @Summary      Dashboard overview
// @Description  Get dashboard overview with key metrics
// @Tags         dashboard
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /dashboard/overview [get]
func (h *Handler) Overview(c *gin.Context) {
	o, err := h.service.Overview()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// Orders GET /dashboard/orders
// @Summary      Order trends
// @Description  Get order trend data for the dashboard
// @Tags         dashboard
// @Produce      json
// @Param        days  query  int  false  "Number of days (default 30)"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /dashboard/orders [get]
func (h *Handler) Orders(c *gin.Context) {
	days := 30
	if v := c.Query("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	items, err := h.service.OrdersTrend(days)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// Inventory GET /dashboard/inventory
// @Summary      Inventory health
// @Description  Get inventory health data for the dashboard
// @Tags         dashboard
// @Produce      json
// @Param        limit  query  int  false  "Max items (default 20)"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /dashboard/inventory [get]
func (h *Handler) Inventory(c *gin.Context) {
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	items, err := h.service.InventoryHealth(limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// Exceptions GET /dashboard/exceptions
// @Summary      Exception distribution
// @Description  Get exception distribution data for the dashboard
// @Tags         dashboard
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /dashboard/exceptions [get]
func (h *Handler) Exceptions(c *gin.Context) {
	items, err := h.service.ExceptionDistribution()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// RejectionReasons GET /dashboard/rejection-reasons
// @Summary      Rejection reason stats
// @Description  Get rejection reason statistics for the dashboard
// @Tags         dashboard
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /dashboard/rejection-reasons [get]
func (h *Handler) RejectionReasons(c *gin.Context) {
	items, err := h.service.GetRejectionReasonStats()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}
