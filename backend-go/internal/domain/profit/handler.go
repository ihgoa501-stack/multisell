package profit

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles profit summary HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new profit handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Summary GET /profit/summary/:productId
func (h *Handler) Summary(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid productId")
		return
	}

	var in SummaryInput
	c.ShouldBindJSON(&in)

	result, err := h.service.Calculate(productID, in.CalculatedBy)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// ListSummaries GET /profit/summaries
func (h *Handler) ListSummaries(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("status")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	items, total, err := h.service.ListSummaries(p.Page, p.Size, status, startDate, endDate)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// CalculateOrderProfit POST /profit/order/:orderId/calculate
func (h *Handler) CalculateOrderProfit(c *gin.Context) {
	orderIDStr := c.Param("orderId")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid orderId")
		return
	}

	result, err := h.service.CalculateOrderProfit(c.Request.Context(), uint(orderID))
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, result)
}

func finalAuthorityParams(c *gin.Context) (int64, int64, bool) {
	ownerID := common.UserIDFromCtx(c)
	orderID, err := strconv.ParseInt(c.Param("orderId"), 10, 64)
	if ownerID == nil || *ownerID <= 0 || err != nil || orderID <= 0 {
		response.Error(c, http.StatusBadRequest, "valid Owner and orderId are required")
		return 0, 0, false
	}
	return *ownerID, orderID, true
}

func (h *Handler) AllocateOrderProductCost(c *gin.Context) {
	ownerID, orderID, ok := finalAuthorityParams(c)
	if !ok {
		return
	}
	var in AllocateOrderProductCostInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "valid exact cost allocation is required")
		return
	}
	out, err := h.service.AllocateOrderProductCost(c.Request.Context(), ownerID, orderID, in)
	if err != nil {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}
	response.Success(c, out)
}

func (h *Handler) FinalizeOrderProfit(c *gin.Context) {
	ownerID, orderID, ok := finalAuthorityParams(c)
	if !ok {
		return
	}
	out, err := h.service.FinalizeOrderProfit(c.Request.Context(), ownerID, orderID)
	if err != nil {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}
	response.Success(c, out)
}

func (h *Handler) ListFinalOrderProfitVersions(c *gin.Context) {
	ownerID, orderID, ok := finalAuthorityParams(c)
	if !ok {
		return
	}
	out, err := h.service.ListFinalOrderProfitVersions(c.Request.Context(), ownerID, orderID)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, out)
}
