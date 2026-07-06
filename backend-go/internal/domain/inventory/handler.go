package inventory

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles inventory HTTP requests.
type Handler struct {
	service     *Service
	approvalSvc *approval.Service
}

// NewHandler creates a new inventory handler.
func NewHandler(service *Service, approvalSvc *approval.Service) *Handler {
	return &Handler{service: service, approvalSvc: approvalSvc}
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

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "inventory_update",
			Requester:   operator,
			NewValue:    fmt.Sprintf("update inventory id=%d qty=%d", id, req.Quantity),
			Reason:      "inventory update requires approval",
			TargetType:  "inventory",
			TargetID:    id,
			RiskLevel:   "high",
			EntityType:  "inventory",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("inventory update requires approval (approval_id=%d)", apprReq.ID))
		return
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

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "inventory_lock",
			Requester:   operator,
			NewValue:    fmt.Sprintf("lock inventory id=%d qty=%d", id, req.Quantity),
			Reason:      "inventory lock requires approval",
			TargetType:  "inventory",
			TargetID:    id,
			RiskLevel:   "high",
			EntityType:  "inventory",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("inventory lock requires approval (approval_id=%d)", apprReq.ID))
		return
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

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "inventory_unlock",
			Requester:   operator,
			NewValue:    fmt.Sprintf("unlock inventory id=%d qty=%d", id, req.Quantity),
			Reason:      "inventory unlock requires approval",
			TargetType:  "inventory",
			TargetID:    id,
			RiskLevel:   "high",
			EntityType:  "inventory",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("inventory unlock requires approval (approval_id=%d)", apprReq.ID))
		return
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

// ── Cross-Platform Sync handlers ──────────────────────────────────────────

// SyncCrossPlatform runs a cross-platform inventory sync for a single product.
// POST /api/v1/inventory/sync-cross-platform/:productId
func (h *Handler) SyncCrossPlatform(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("productId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid productId")
		return
	}

	result, err := h.service.SyncAcrossPlatforms(c.Request.Context(), productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "sync failed: "+err.Error())
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "inventory_sync_cross_platform",
			Requester:   operator,
			NewValue:    fmt.Sprintf("sync inventory cross-platform product_id=%d", productID),
			Reason:      "cross-platform inventory sync requires approval",
			TargetType:  "inventory",
			TargetID:    productID,
			RiskLevel:   "high",
			EntityType:  "product",
			EntityID:    productID,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("cross-platform inventory sync requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	response.Success(c, result)
}

// OversellReport returns the oversell detection log with pagination.
// GET /api/v1/inventory/oversell-report?page=&size=
func (h *Handler) OversellReport(c *gin.Context) {
	p := common.ParsePagination(c)

	items, total, err := h.service.ListOversellReport(c.Request.Context(), p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list oversell report: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// ── Safety Config (#201) ──────────────────────────────────────────────

// GetSafetyConfig returns safety stock config for a SKU.
// GET /api/v1/inventory/safety-config/:sku_id
func (h *Handler) GetSafetyConfig(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("sku_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid sku_id")
		return
	}
	cfg, err := h.service.GetSafetyConfig(c.Request.Context(), skuID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "safety config not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get safety config: "+err.Error())
		return
	}
	response.Success(c, cfg)
}

// UpsertSafetyConfig creates or updates safety stock config.
// PUT /api/v1/inventory/safety-config/:sku_id
func (h *Handler) UpsertSafetyConfig(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("sku_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid sku_id")
		return
	}
	var cfg InventorySafetyConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	cfg.SkuID = skuID

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "safety_config_upsert",
			Requester:   operator,
			NewValue:    fmt.Sprintf("upsert safety config sku_id=%d", skuID),
			Reason:      "safety stock config changes require approval",
			TargetType:  "safety_config",
			TargetID:    skuID,
			RiskLevel:   "medium",
			EntityType:  "safety_config",
			EntityID:    skuID,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("safety config update requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.UpsertSafetyConfig(c.Request.Context(), &cfg); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to save safety config: "+err.Error())
		return
	}
	response.Success(c, cfg)
}

// ListSafetyConfigs returns all safety stock configs.
// GET /api/v1/inventory/safety-configs
func (h *Handler) ListSafetyConfigs(c *gin.Context) {
	items, err := h.service.ListSafetyConfigs(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list safety configs: "+err.Error())
		return
	}
	if items == nil {
		items = []InventorySafetyConfig{}
	}
	response.Success(c, items)
}

// ── Allocation (#201) ──────────────────────────────────────────────────

// AllocateStock returns allocation recommendation for a SKU.
// GET /api/v1/inventory/allocate/:sku_id
func (h *Handler) AllocateStock(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("sku_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid sku_id")
		return
	}
	rec, err := h.service.AllocateStock(c.Request.Context(), skuID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "allocation failed: "+err.Error())
		return
	}
	response.Success(c, rec)
}

// ── Dead Stock (#201) ──────────────────────────────────────────────────

// IdentifyDeadStock runs dead stock identification.
// POST /api/v1/inventory/dead-stock/analyze?threshold_days=90
func (h *Handler) IdentifyDeadStock(c *gin.Context) {
	thresholdDays, _ := strconv.Atoi(c.DefaultQuery("threshold_days", "90"))

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "dead_stock_analyze",
			Requester:   operator,
			NewValue:    fmt.Sprintf("analyze dead stock threshold=%d", thresholdDays),
			Reason:      "dead stock analysis requires approval",
			TargetType:  "dead_stock",
			RiskLevel:   "low",
			EntityType:  "dead_stock",
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("dead stock analysis requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	items, err := h.service.IdentifyDeadStock(c.Request.Context(), thresholdDays)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "dead stock analysis failed: "+err.Error())
		return
	}
	response.Success(c, items)
}

// ListDeadStockLogs returns dead stock detection history.
// GET /api/v1/inventory/dead-stock/logs?page=&size=
func (h *Handler) ListDeadStockLogs(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListDeadStockLogs(c.Request.Context(), p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list dead stock logs: "+err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}
