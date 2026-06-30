package aftersales

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles aftersales HTTP requests.
type Handler struct {
	service        *Service
	disputeService *DisputeService
}

// NewHandler creates a new aftersales handler.
func NewHandler(service *Service, disputeService *DisputeService) *Handler {
	return &Handler{service: service, disputeService: disputeService}
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

// List GET /aftersales
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &ListFilter{
		Search:  c.Query("search"),
		Status:  c.Query("status"),
		OrderID: parseOptionalInt64(c, "order_id"),
	}
	items, total, err := h.service.List(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /aftersales/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	o, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "aftersales order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// Create POST /aftersales
func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.Create(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// Update PUT /aftersales/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.Update(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "aftersales order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// Delete DELETE /aftersales/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "aftersales order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// Approve POST /aftersales/:id/approve
func (h *Handler) Approve(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ApproveInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.Approve(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "aftersales order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// Reject POST /aftersales/:id/reject
func (h *Handler) Reject(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in RejectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.Reject(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "aftersales order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// Receive POST /aftersales/:id/receive
func (h *Handler) Receive(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ReceiveInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.Receive(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "aftersales order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// Refund POST /aftersales/:id/refund
func (h *Handler) Refund(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in RefundInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.Refund(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "aftersales order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// Summary GET /aftersales/summary
func (h *Handler) Summary(c *gin.Context) {
	sum, err := h.service.Summary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sum)
}

// ---------------------------------------------------------------------------
// Dispute Handlers
// ---------------------------------------------------------------------------

// CreateDispute POST /aftersales/disputes
func (h *Handler) CreateDispute(c *gin.Context) {
	var in CreateDisputeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	dc, err := h.disputeService.CreateCase(c.Request.Context(), &in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, dc)
}

// GetDispute GET /aftersales/disputes/:id
func (h *Handler) GetDispute(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	dc, err := h.disputeService.GetCase(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "dispute case not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, dc)
}

// ListDisputes GET /aftersales/disputes
func (h *Handler) ListDisputes(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &DisputeListFilter{
		Platform:  c.Query("platform"),
		ClaimType: c.Query("claim_type"),
		Status:    c.Query("status"),
	}
	items, total, err := h.disputeService.ListCases(c.Request.Context(), &p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// EvaluateDispute POST /aftersales/disputes/:id/evaluate
func (h *Handler) EvaluateDispute(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result, err := h.disputeService.EvaluateCase(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "dispute case not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// AutoDecideDispute POST /aftersales/disputes/:id/auto-decide
func (h *Handler) AutoDecideDispute(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result, err := h.disputeService.AutoDecide(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "dispute case not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// UpdateDisputeStatus PUT /aftersales/disputes/:id/status
func (h *Handler) UpdateDisputeStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	dc, err := h.disputeService.UpdateDisputeStatus(c.Request.Context(), id, in.Status, in.Reason)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "dispute case not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, dc)
}
