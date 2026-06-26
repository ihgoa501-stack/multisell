package allocation

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles allocation HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new allocation handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ── Warehouse handlers ───────────────────────────────────────────

// ListWarehouses returns a paginated list of warehouses.
// GET /api/v1/allocation/warehouses?page=1&size=20&search=xxx
func (h *Handler) ListWarehouses(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")

	items, total, err := h.service.ListWarehouses(c.Request.Context(), p.Page, p.Size, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list warehouses: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// CreateWarehouse creates a new warehouse.
// POST /api/v1/allocation/warehouses
func (h *Handler) CreateWarehouse(c *gin.Context) {
	var w Warehouse
	if err := c.ShouldBindJSON(&w); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.CreateWarehouse(c.Request.Context(), &w); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create warehouse: "+err.Error())
		return
	}

	response.Success(c, w)
}

// UpdateWarehouse updates a warehouse.
// PUT /api/v1/allocation/warehouses/:id
func (h *Handler) UpdateWarehouse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var w Warehouse
	if err := c.ShouldBindJSON(&w); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	w.ID = id

	if err := h.service.UpdateWarehouse(c.Request.Context(), &w); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update warehouse: "+err.Error())
		return
	}

	response.Success(c, w)
}

// DeleteWarehouse deletes a warehouse.
// DELETE /api/v1/allocation/warehouses/:id
func (h *Handler) DeleteWarehouse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteWarehouse(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete warehouse: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// ── AllocationRule handlers ──────────────────────────────────────

// ListRules returns a paginated list of allocation rules.
// GET /api/v1/allocation/rules?page=1&size=20&warehouse_id=
func (h *Handler) ListRules(c *gin.Context) {
	p := common.ParsePagination(c)
	warehouseID, _ := strconv.ParseInt(c.Query("warehouse_id"), 10, 64)

	items, total, err := h.service.ListRules(c.Request.Context(), p.Page, p.Size, warehouseID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list rules: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// CreateRule creates a new allocation rule.
// POST /api/v1/allocation/rules
func (h *Handler) CreateRule(c *gin.Context) {
	var r AllocationRule
	if err := c.ShouldBindJSON(&r); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.CreateRule(c.Request.Context(), &r); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create rule: "+err.Error())
		return
	}

	response.Success(c, r)
}

// UpdateRule updates an allocation rule.
// PUT /api/v1/allocation/rules/:id
func (h *Handler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var r AllocationRule
	if err := c.ShouldBindJSON(&r); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	r.ID = id

	if err := h.service.UpdateRule(c.Request.Context(), &r); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update rule: "+err.Error())
		return
	}

	response.Success(c, r)
}

// DeleteRule deletes an allocation rule.
// DELETE /api/v1/allocation/rules/:id
func (h *Handler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteRule(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete rule: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// ── CostAllocationBatch handlers ────────────────────────────────

// ListBatches returns a paginated list of cost allocation batches.
// GET /api/v1/allocation/cost/batches?page=1&size=20&allocation_type=
func (h *Handler) ListBatches(c *gin.Context) {
	p := common.ParsePagination(c)
	allocationType := c.Query("allocation_type")

	items, total, err := h.service.ListBatches(c.Request.Context(), p.Page, p.Size, allocationType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list batches: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// CreateBatch creates a new cost allocation batch.
// POST /api/v1/allocation/cost/batches
func (h *Handler) CreateBatch(c *gin.Context) {
	var b CostAllocationBatch
	if err := c.ShouldBindJSON(&b); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.CreateBatch(c.Request.Context(), &b); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create batch: "+err.Error())
		return
	}

	response.Success(c, b)
}

// GetBatch returns a single cost allocation batch by ID with its items.
// GET /api/v1/allocation/cost/batches/:id
func (h *Handler) GetBatch(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	batch, err := h.service.GetBatchByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "batch not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get batch: "+err.Error())
		return
	}

	// Also fetch items for the batch detail view
	p := common.ParsePagination(c)
	items, total, _ := h.service.ListBatchItems(c.Request.Context(), id, p.Page, p.Size)

	response.Success(c, gin.H{
		"batch": batch,
		"items": items,
		"total": total,
	})
}

// ComputeAllocation computes cost allocation for a batch.
// POST /api/v1/allocation/cost/:batchId/compute
func (h *Handler) ComputeAllocation(c *gin.Context) {
	batchID, err := strconv.ParseInt(c.Param("batchId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid batch id")
		return
	}

	if err := h.service.ComputeAllocation(c.Request.Context(), batchID); err != nil {
		response.Error(c, http.StatusInternalServerError, "allocation computation failed: "+err.Error())
		return
	}

	response.Success(c, gin.H{"batch_id": batchID})
}

// ── AutoAllocate handlers ──────────────────────────────────────────

// AutoAllocate distributes available inventory for a SKU across warehouses.
// POST /api/v1/allocation/auto-allocate/:skuId
func (h *Handler) AutoAllocate(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("skuId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid sku_id")
		return
	}

	if err := h.service.AutoAllocate(c.Request.Context(), skuID); err != nil {
		response.Error(c, http.StatusInternalServerError, "auto-allocate failed: "+err.Error())
		return
	}

	response.Success(c, gin.H{"sku_id": skuID, "status": "allocated"})
}
