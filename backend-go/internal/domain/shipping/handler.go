package shipping

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles shipping HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new shipping handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
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

// ---------- Provider ----------

// ListProviders GET /shipping/providers
func (h *Handler) ListProviders(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")
	items, total, err := h.service.ListProviders(&p, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetProvider GET /shipping/providers/:id
func (h *Handler) GetProvider(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	prov, err := h.service.GetProvider(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "provider not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, prov)
}

// CreateProvider POST /shipping/providers
func (h *Handler) CreateProvider(c *gin.Context) {
	var in CreateProviderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	prov, err := h.service.CreateProvider(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, prov)
}

// UpdateProvider PUT /shipping/providers/:id
func (h *Handler) UpdateProvider(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateProviderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	prov, err := h.service.UpdateProvider(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "provider not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, prov)
}

// DeleteProvider DELETE /shipping/providers/:id
func (h *Handler) DeleteProvider(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteProvider(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "provider not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------- Channel ----------

// ListChannels GET /shipping/channels
func (h *Handler) ListChannels(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")
	providerID := parseOptionalInt64(c, "provider_id")
	items, total, err := h.service.ListChannels(&p, providerID, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetChannel GET /shipping/channels/:id
func (h *Handler) GetChannel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ch, err := h.service.GetChannel(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "channel not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, ch)
}

// CreateChannel POST /shipping/channels
func (h *Handler) CreateChannel(c *gin.Context) {
	var in CreateChannelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	ch, err := h.service.CreateChannel(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, ch)
}

// UpdateChannel PUT /shipping/channels/:id
func (h *Handler) UpdateChannel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateChannelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	ch, err := h.service.UpdateChannel(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "channel not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, ch)
}

// DeleteChannel DELETE /shipping/channels/:id
func (h *Handler) DeleteChannel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteChannel(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "channel not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------- Zone ----------

// ListZones GET /shipping/zones
func (h *Handler) ListZones(c *gin.Context) {
	p := common.ParsePagination(c)
	channelID := parseOptionalInt64(c, "channel_id")
	items, total, err := h.service.ListZones(&p, channelID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// CreateZone POST /shipping/zones
func (h *Handler) CreateZone(c *gin.Context) {
	var in CreateZoneInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	z, err := h.service.CreateZone(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, z)
}

// DeleteZone DELETE /shipping/zones/:id
func (h *Handler) DeleteZone(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteZone(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "zone not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------- Quote Rule ----------

// ListRules GET /shipping/rules
func (h *Handler) ListRules(c *gin.Context) {
	p := common.ParsePagination(c)
	channelID := parseOptionalInt64(c, "channel_id")
	items, total, err := h.service.ListRules(&p, channelID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// CreateRule POST /shipping/rules
func (h *Handler) CreateRule(c *gin.Context) {
	var in CreateQuoteRuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	r, err := h.service.CreateRule(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// DeleteRule DELETE /shipping/rules/:id
func (h *Handler) DeleteRule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteRule(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "rule not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------- Quote ----------

// Quote POST /shipping/quote
func (h *Handler) Quote(c *gin.Context) {
	var in QuoteRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.service.Quote(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, resp)
}

// ---------- Bill Batch ----------

// ListBillBatches GET /shipping/bill-batches
func (h *Handler) ListBillBatches(c *gin.Context) {
	p := common.ParsePagination(c)
	providerID := parseOptionalInt64(c, "provider_id")
	items, total, err := h.service.ListBillBatches(&p, providerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetBillBatch GET /shipping/bill-batches/:id
func (h *Handler) GetBillBatch(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	b, items, err := h.service.GetBillBatch(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "bill batch not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"batch": b, "items": items})
}

// CreateBillBatch POST /shipping/bill-batches
func (h *Handler) CreateBillBatch(c *gin.Context) {
	var in CreateBillBatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	b, err := h.service.CreateBillBatch(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, b)
}

// DeleteBillBatch DELETE /shipping/bill-batches/:id
func (h *Handler) DeleteBillBatch(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteBillBatch(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "bill batch not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ListBillItems GET /shipping/bill-batches/:id/items
func (h *Handler) ListBillItems(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	items, total, err := h.service.ListBillItems(&p, id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// ---- Phase 1: Fulfillment Intelligence OS Handlers ----

// QuoteUnified POST /shipping/quote-unified
func (h *Handler) QuoteUnified(c *gin.Context) {
	var req QuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	resp, err := h.service.QuoteUnified(&req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, resp)
}

// CreateSnapshot POST /shipping/snapshots
func (h *Handler) CreateSnapshot(c *gin.Context) {
	var req CreateSnapshotInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	snap, err := h.service.CreateSnapshot(&req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, snap)
}

// GetSnapshot GET /shipping/snapshots/:orderId
func (h *Handler) GetSnapshot(c *gin.Context) {
	orderID, err := strconv.ParseInt(c.Param("orderId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid order_id")
		return
	}
	snap, err := h.service.GetSnapshotByOrderID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "snapshot not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, snap)
}

// ReconcileBillBatch POST /shipping/bill-batches/:id/reconcile
func (h *Handler) ReconcileBillBatch(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid batch_id")
		return
	}
	result, err := h.service.ReconcileBillBatch(batchID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// ListBillAnomalies GET /shipping/bill-batches/:id/anomalies
func (h *Handler) ListBillAnomalies(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid batch_id")
		return
	}
	items, err := h.service.ListBillAnomalies(batchID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// ReviewBillItem PUT /shipping/bill-items/:id/review
func (h *Handler) ReviewBillItem(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid item_id")
		return
	}
	var req struct {
		ReviewStatus string `json:"review_status" binding:"required"`
		Note         string `json:"note"`
		ResolvedBy   string `json:"resolved_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.ReviewBillItem(id, req.ReviewStatus, req.Note, req.ResolvedBy); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "ok"})
}

// ListRuleVersions GET /shipping/rules/:id/versions
func (h *Handler) ListRuleVersions(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid channel_id")
		return
	}
	rules, total, err := h.service.ListRuleVersions(channelID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, rules, total, 1, int(total))
}
