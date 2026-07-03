package order

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles order HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new order handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
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

// List GET /order
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &OrderListFilter{
		Search:     c.Query("search"),
		Status:     c.Query("status"),
		PlatformID: parseOptionalInt64(c, "platform_id"),
	}
	items, total, err := h.service.List(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]OrderResponse, len(items))
	for i, o := range items {
		resp[i] = orderToResponse(o)
	}
	response.Paginated(c, resp, total, p.Page, p.Size)
}

// Get GET /order/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	detail, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, OrderDetailResponse{
		Order:      orderToResponse(detail.Order),
		Items:      detail.Items,
		StatusLogs: detail.StatusLogs,
	})
}

// Create POST /order
func (h *Handler) Create(c *gin.Context) {
	var in CreateOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.Create(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, orderToResponse(*o))
}

// Update PUT /order/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.Update(id, &in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, orderToResponse(*o))
}

// Delete DELETE /order/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// UpdateStatus POST /order/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		From     string `json:"from"`
		To       string `json:"to" binding:"required"`
		Operator string `json:"operator"`
		Remark   string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.UpdateStatus(id, body.From, body.To, body.Operator, body.Remark); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id, "status": body.To})
}

// Summary GET /order/summary
func (h *Handler) Summary(c *gin.Context) {
	sum, err := h.service.Summary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sum)
}
