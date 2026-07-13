package aftersales

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

func ownerID(c *gin.Context) (int64, bool) {
	id := common.UserIDFromCtx(c)
	if id == nil || *id <= 0 {
		response.Error(c, http.StatusUnauthorized, "Owner identity required")
		return 0, false
	}
	return *id, true
}

func resolutionError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, http.StatusNotFound, "resolution not found")
		return
	}
	response.Error(c, http.StatusConflict, err.Error())
}

func (h *Handler) CreateResolution(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var in CreateResolutionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	out, err := h.service.CreateResolution(owner, &in)
	if err != nil {
		resolutionError(c, err)
		return
	}
	response.Success(c, out)
}
func (h *Handler) GetResolution(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	out, err := h.service.GetResolutionDetail(owner, id)
	if err != nil {
		resolutionError(c, err)
		return
	}
	response.Success(c, out)
}
func (h *Handler) DecideResolution(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in struct {
		Decision       string `json:"decision" binding:"required"`
		Reason         string `json:"reason" binding:"required"`
		IdempotencyKey string `json:"idempotency_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	out, err := h.service.DecideResolution(owner, owner, id, ResolutionDecisionInput{Decision: in.Decision, Reason: in.Reason, IdempotencyKey: in.IdempotencyKey})
	if err != nil {
		resolutionError(c, err)
		return
	}
	response.Success(c, out)
}
func (h *Handler) SubmitResolution(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in struct {
		ExternalRequestID string `json:"external_request_id" binding:"required"`
		IdempotencyKey    string `json:"idempotency_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	out, err := h.service.SubmitResolution(owner, id, ResolutionExecutionInput{ExternalRequestID: in.ExternalRequestID, IdempotencyKey: in.IdempotencyKey})
	if err != nil {
		resolutionError(c, err)
		return
	}
	response.Success(c, out)
}
func (h *Handler) RecordResolutionReceipt(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in struct {
		Outcome           string          `json:"outcome" binding:"required"`
		SourceType        string          `json:"source_type" binding:"required"`
		EvidenceID        string          `json:"evidence_id" binding:"required"`
		ExternalReceiptID string          `json:"external_receipt_id" binding:"required"`
		ObservedAt        time.Time       `json:"observed_at" binding:"required"`
		ActualMinor       int64           `json:"actual_minor"`
		Currency          string          `json:"currency" binding:"required"`
		FailureCode       string          `json:"failure_code"`
		Payload           json.RawMessage `json:"receipt_payload" binding:"required"`
		PayloadSHA256     string          `json:"receipt_sha256" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	out, err := h.service.RecordResolutionReceipt(owner, id, ResolutionReceiptInput{Outcome: in.Outcome, SourceType: in.SourceType, EvidenceID: in.EvidenceID, ExternalReceiptID: in.ExternalReceiptID, ObservedAt: in.ObservedAt, ActualMinor: in.ActualMinor, Currency: in.Currency, FailureCode: in.FailureCode, Payload: in.Payload, PayloadSHA256: in.PayloadSHA256})
	if err != nil {
		resolutionError(c, err)
		return
	}
	response.Success(c, out)
}

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
	// The legacy endpoint used an internal database transition as proof that an
	// external refund succeeded. Keep the route as an explicit fail-closed
	// compatibility boundary; callers must use the Owner resolution workflow and
	// provide a platform receipt or controlled reconciliation evidence.
	response.Error(c, http.StatusPreconditionRequired, "legacy direct refund is disabled; use /aftersales/resolutions")
}

// AutoDecide POST /aftersales/:id/auto-decide
func (h *Handler) AutoDecide(c *gin.Context) {
	response.Success(c, gin.H{
		"status":  "unavailable",
		"message": "售后 Agent 自动处理功能开发中",
	})
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
	legacyDisputeDisabled(c)
}

// GetDispute GET /aftersales/disputes/:id
func (h *Handler) GetDispute(c *gin.Context) {
	legacyDisputeDisabled(c)
}

// ListDisputes GET /aftersales/disputes
func (h *Handler) ListDisputes(c *gin.Context) {
	legacyDisputeDisabled(c)
}

// EvaluateDispute POST /aftersales/disputes/:id/evaluate
func (h *Handler) EvaluateDispute(c *gin.Context) {
	legacyDisputeDisabled(c)
}

// AutoDecideDispute POST /aftersales/disputes/:id/auto-decide
func (h *Handler) AutoDecideDispute(c *gin.Context) {
	legacyDisputeDisabled(c)
}

// UpdateDisputeStatus PUT /aftersales/disputes/:id/status
func (h *Handler) UpdateDisputeStatus(c *gin.Context) {
	legacyDisputeDisabled(c)
}

func legacyDisputeDisabled(c *gin.Context) {
	response.Error(c, http.StatusPreconditionRequired, "legacy dispute API is disabled; use /aftersales/resolutions with kind=dispute")
}
