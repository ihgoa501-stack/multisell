package inventory

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles inventory HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new inventory handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ── Inventory handlers ────────────────────────────────────────────

// List returns a paginated list of inventory records.
// GET /api/v1/inventory?page=1&size=20&sku_id=&warehouse=
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	skuID, _ := strconv.ParseInt(c.Query("sku_id"), 10, 64)
	warehouse := c.Query("warehouse")

	items, total, err := h.service.List(c.Request.Context(), p.Page, p.Size, skuID, warehouse)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list inventory: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get returns a single inventory record by ID.
// GET /api/v1/inventory/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "inventory not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get inventory: "+err.Error())
		return
	}

	response.Success(c, item)
}

// Update updates the stock quantity of an inventory record.
// PUT /api/v1/inventory/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Quantity int    `json:"quantity"`
		Remark   string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if err := h.service.UpdateStock(c.Request.Context(), id, req.Quantity, operator, req.Remark); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update stock: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id, "quantity": req.Quantity})
}

// Lock locks stock for an inventory record.
// POST /api/v1/inventory/:id/lock
func (h *Handler) Lock(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if err := h.service.LockStock(c.Request.Context(), id, req.Quantity, operator); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to lock stock: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id, "locked": req.Quantity})
}

// Unlock unlocks stock for an inventory record.
// POST /api/v1/inventory/:id/unlock
func (h *Handler) Unlock(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if err := h.service.UnlockStock(c.Request.Context(), id, req.Quantity, operator); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to unlock stock: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id, "unlocked": req.Quantity})
}

// ListLogs returns inventory change logs.
// GET /api/v1/inventory/logs?sku_id=&page=&size=
func (h *Handler) ListLogs(c *gin.Context) {
	p := common.ParsePagination(c)
	skuID, _ := strconv.ParseInt(c.Query("sku_id"), 10, 64)

	items, total, err := h.service.ListLogs(c.Request.Context(), skuID, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list logs: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// ── Warehouse handlers ───────────────────────────────────────────

// ListWarehouses returns a paginated list of warehouses.
// GET /api/v1/inventory/warehouses?page=1&size=20&search=xxx
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

// GetWarehouse returns a single warehouse by ID.
// GET /api/v1/inventory/warehouses/:id
func (h *Handler) GetWarehouse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetWarehouseByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "warehouse not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get warehouse: "+err.Error())
		return
	}

	response.Success(c, item)
}

// CreateWarehouse creates a new warehouse.
// POST /api/v1/inventory/warehouses
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
// PUT /api/v1/inventory/warehouses/:id
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
// DELETE /api/v1/inventory/warehouses/:id
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

// ListInventoryBySku returns inventory per warehouse for a SKU.
// GET /api/v1/inventory/sku/:sku_id/warehouses
func (h *Handler) ListInventoryBySku(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("sku_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid sku_id")
		return
	}

	items, err := h.service.ListInventoryBySku(c.Request.Context(), skuID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list inventory: "+err.Error())
		return
	}

	response.Success(c, items)
}

// ── Bin Location handlers ──────────────────────────────────────────

// ListLocations returns a paginated list of bin locations.
// GET /api/v1/inventory/locations?warehouse=&page=&size=
func (h *Handler) ListLocations(c *gin.Context) {
	p := common.ParsePagination(c)
	warehouse := c.Query("warehouse")

	items, total, err := h.service.ListLocations(warehouse, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list locations: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// ── Transfer handlers ──────────────────────────────────────────────

// ListTransfers returns a paginated list of inventory transfers.
// GET /api/v1/inventory/transfers?sku_id=&status=&page=&size=
func (h *Handler) ListTransfers(c *gin.Context) {
	p := common.ParsePagination(c)
	skuID, _ := strconv.ParseInt(c.Query("sku_id"), 10, 64)
	status := c.Query("status")

	items, total, err := h.service.ListTransfers(skuID, status, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list transfers: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}
