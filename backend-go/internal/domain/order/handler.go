package order

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles order HTTP requests.
type Handler struct {
	service     *Service
	approvalSvc *approval.Service
}

// NewHandler creates a new order handler.
func NewHandler(service *Service, approvalSvc *approval.Service) *Handler {
	return &Handler{service: service, approvalSvc: approvalSvc}
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
// @Summary      List orders
// @Description  Get paginated list of orders
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        page        query  int     false  "Page number"
// @Param        size        query  int     false  "Page size"
// @Param        search      query  string  false  "Search keyword"
// @Param        status      query  string  false  "Filter by status"
// @Param        platform_id query  int     false  "Filter by platform ID"
// @Success      200  {object}  response.PageResult
// @Security     BearerAuth
// @Router       /order [get]
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
// @Summary      Get order
// @Description  Get order detail by ID
// @Tags         orders
// @Produce      json
// @Param        id  path  int  true  "Order ID"
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /order/{id} [get]
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

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "order_update",
			Requester:   operator,
			NewValue:    fmt.Sprintf("update order id=%d", id),
			Reason:      "order update requires approval",
			TargetType:  "order",
			TargetID:    id,
			RiskLevel:   "medium",
			EntityType:  "order",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("order update requires approval (approval_id=%d)", apprReq.ID))
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

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "order_delete",
			Requester:   operator,
			NewValue:    fmt.Sprintf("delete order id=%d", id),
			Reason:      "order deletion requires approval",
			TargetType:  "order",
			TargetID:    id,
			RiskLevel:   "medium",
			EntityType:  "order",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("order deletion requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// UpdateStatus POST /order/:id/status
// @Summary      Update order status
// @Description  Transition order status via state machine
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        id    path  int     true  "Order ID"
// @Param        body  body  object{from=string,to=string,operator=string,remark=string}  true  "Status transition"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /order/{id}/status [post]
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

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "order_status_change",
			Requester:   operator,
			NewValue:    fmt.Sprintf("order %d status %s -> %s", id, body.From, body.To),
			Reason:      "order status change requires approval",
			TargetType:  "order",
			TargetID:    id,
			RiskLevel:   "medium",
			EntityType:  "order",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("order status change requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.UpdateStatus(id, body.From, body.To, body.Operator, body.Remark); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id, "status": body.To})
}

// Summary GET /order/summary
// @Summary      Order summary
// @Description  Get order summary statistics
// @Tags         orders
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /order/summary [get]
func (h *Handler) Summary(c *gin.Context) {
	sum, err := h.service.Summary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sum)
}
