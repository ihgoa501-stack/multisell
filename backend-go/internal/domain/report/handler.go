package report

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles report HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new report handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseOptionalInt64(c *gin.Context, key string) *int64 {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

// Sales GET /report/sales
func (h *Handler) Sales(c *gin.Context) {
	from, to := parseRange(c.Query("from"), c.Query("to"))
	platformID := parseOptionalInt64(c, "platform_id")
	r, err := h.service.Sales(from, to, platformID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// Profit GET /report/profit
func (h *Handler) Profit(c *gin.Context) {
	from, to := parseRange(c.Query("from"), c.Query("to"))
	platformID := parseOptionalInt64(c, "platform_id")
	r, err := h.service.Profit(from, to, platformID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// Inventory GET /report/inventory
func (h *Handler) Inventory(c *gin.Context) {
	r, err := h.service.Inventory()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// Settlement GET /report/settlement
func (h *Handler) Settlement(c *gin.Context) {
	from, to := parseRange(c.Query("from"), c.Query("to"))
	platformID := parseOptionalInt64(c, "platform_id")
	r, err := h.service.Settlement(from, to, platformID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// PlatformFee GET /report/platform-fee
func (h *Handler) PlatformFee(c *gin.Context) {
	from, to := parseRange(c.Query("from"), c.Query("to"))
	platformID := parseOptionalInt64(c, "platform_id")
	r, err := h.service.PlatformFee(from, to, platformID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}
