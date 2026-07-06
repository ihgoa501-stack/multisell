package supplier

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles supplier HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new supplier handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ── Supplier handlers ─────────────────────────────────────────────

// List returns a paginated list of suppliers.
// GET /api/v1/suppliers?page=1&size=20&search=xxx
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")

	items, total, err := h.service.List(c.Request.Context(), p.Page, p.Size, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list suppliers: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get returns a single supplier by ID.
// GET /api/v1/suppliers/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "supplier not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get supplier: "+err.Error())
		return
	}

	response.Success(c, item)
}

// Create creates a new supplier.
// POST /api/v1/suppliers
func (h *Handler) Create(c *gin.Context) {
	var sup Supplier
	if err := c.ShouldBindJSON(&sup); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &sup); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create supplier: "+err.Error())
		return
	}

	response.Success(c, sup)
}

// Update updates an existing supplier.
// PUT /api/v1/suppliers/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var sup Supplier
	if err := c.ShouldBindJSON(&sup); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sup.ID = id

	if err := h.service.Update(c.Request.Context(), &sup); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update supplier: "+err.Error())
		return
	}

	response.Success(c, sup)
}

// Delete deletes a supplier.
// DELETE /api/v1/suppliers/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete supplier: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// ── ProductSupplier handlers ──────────────────────────────────────

// ListProductSuppliers returns product-supplier associations.
// GET /api/v1/product-suppliers?product_id=
func (h *Handler) ListProductSuppliers(c *gin.Context) {
	productID, _ := strconv.ParseInt(c.Query("product_id"), 10, 64)

	items, err := h.service.ListProductSuppliers(c.Request.Context(), productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list product suppliers: "+err.Error())
		return
	}

	response.Success(c, items)
}

// CreateProductSupplier creates a product-supplier association.
// POST /api/v1/product-suppliers
func (h *Handler) CreateProductSupplier(c *gin.Context) {
	var ps ProductSupplier
	if err := c.ShouldBindJSON(&ps); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.CreateProductSupplier(c.Request.Context(), &ps); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create product supplier: "+err.Error())
		return
	}

	response.Success(c, ps)
}

// UpdateProductSupplier updates a product-supplier association.
// PUT /api/v1/product-suppliers/:id
func (h *Handler) UpdateProductSupplier(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var ps ProductSupplier
	if err := c.ShouldBindJSON(&ps); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	ps.ID = id

	if err := h.service.UpdateProductSupplier(c.Request.Context(), &ps); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update product supplier: "+err.Error())
		return
	}

	response.Success(c, ps)
}

// DeleteProductSupplier deletes a product-supplier association.
// DELETE /api/v1/product-suppliers/:id
func (h *Handler) DeleteProductSupplier(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteProductSupplier(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete product supplier: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// ── Supplier Score handlers ───────────────────────────────────────

// GetScore returns the supplier's credibility score.
// GET /api/v1/suppliers/:id/score
func (h *Handler) GetScore(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	score, err := h.service.GetScore(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "score not found, run recalculate first")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get score: "+err.Error())
		return
	}

	response.Success(c, score)
}

// RecalculateScore triggers score recalculation from purchase order history.
// POST /api/v1/suppliers/:id/recalculate
func (h *Handler) RecalculateScore(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	score, err := h.service.RecalculateScore(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to recalculate score: "+err.Error())
		return
	}

	response.Success(c, score)
}

// ListScoreboard returns all supplier scores sorted by reliability.
// GET /api/v1/suppliers/scoreboard
func (h *Handler) ListScoreboard(c *gin.Context) {
	scores, err := h.service.ListScoreboard(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list scoreboard: "+err.Error())
		return
	}

	if scores == nil {
		scores = []SupplierScore{}
	}

	response.Success(c, scores)
}

// GetSupplierComparison returns a product vs its suppliers side-by-side.
// GET /api/v1/product-suppliers/comparison?product_id=
func (h *Handler) GetSupplierComparison(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid product id")
		return
	}

	result, err := h.service.GetSupplierComparison(c.Request.Context(), productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get supplier comparison: "+err.Error())
		return
	}

	response.Success(c, result)
}

// ── Score History (#197) ───────────────────────────────────────────────

// GetScoreHistory returns score history for a supplier.
// GET /api/v1/suppliers/:id/score-history?limit=30
func (h *Handler) GetScoreHistory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	items, err := h.service.GetScoreHistory(c.Request.Context(), id, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get score history: "+err.Error())
		return
	}
	if items == nil {
		items = []SupplierScoreHistory{}
	}
	response.Success(c, items)
}

// UpdateScoreManual manually sets a supplier's KPI score.
// PUT /api/v1/suppliers/:id/kpi-score
func (h *Handler) UpdateScoreManual(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		KpiScore float64 `json:"kpi_score" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	sup, err := h.service.UpdateScoreManual(c.Request.Context(), id, req.KpiScore)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "supplier not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update score: "+err.Error())
		return
	}
	response.Success(c, sup)
}

// RecordScoreSnapshot creates a score history snapshot.
// POST /api/v1/suppliers/:id/score-snapshot
func (h *Handler) RecordScoreSnapshot(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.RecordScoreSnapshot(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to record snapshot: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
