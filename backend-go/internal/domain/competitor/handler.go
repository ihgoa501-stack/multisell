package competitor

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles competitor HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new competitor handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ── Competitor Product ───────────────────────────────────────────────

// List returns a paginated list of competitor products.
// GET /api/v1/competitors?page=1&size=20&platform=&search=
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	platform := c.Query("platform")
	search := c.Query("search")

	items, total, err := h.service.List(c.Request.Context(), p.Page, p.Size, platform, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list competitors: "+err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get returns a single competitor product.
// GET /api/v1/competitors/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "competitor not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get competitor: "+err.Error())
		return
	}
	response.Success(c, item)
}

// Create creates a new competitor product.
// POST /api/v1/competitors
func (h *Handler) Create(c *gin.Context) {
	var input CreateCompetitorInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	cp, err := h.service.Create(c.Request.Context(), &input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create competitor: "+err.Error())
		return
	}
	response.Success(c, cp)
}

// Update updates an existing competitor product.
// PUT /api/v1/competitors/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var cp CompetitorProduct
	if err := c.ShouldBindJSON(&cp); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	cp.ID = id
	if err := h.service.Update(c.Request.Context(), &cp); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update competitor: "+err.Error())
		return
	}
	response.Success(c, cp)
}

// Delete deletes a competitor product.
// DELETE /api/v1/competitors/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete competitor: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ── Price Snapshots ──────────────────────────────────────────────────

// RecordPrice records a price snapshot.
// POST /api/v1/competitors/:id/prices
func (h *Handler) RecordPrice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var input RecordPriceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	snapshot, err := h.service.RecordPrice(c.Request.Context(), id, &input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to record price: "+err.Error())
		return
	}
	response.Success(c, snapshot)
}

// ListPrices returns price snapshots.
// GET /api/v1/competitors/:id/prices?limit=30
func (h *Handler) ListPrices(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	snapshots, err := h.service.ListPrices(c.Request.Context(), id, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list prices: "+err.Error())
		return
	}
	if snapshots == nil {
		snapshots = []PriceSnapshot{}
	}
	response.Success(c, snapshots)
}

// GetPriceTrend returns price trend analysis.
// GET /api/v1/competitors/:id/trend
func (h *Handler) GetPriceTrend(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	trend, err := h.service.GetPriceTrend(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "failed to get trend: "+err.Error())
		return
	}
	response.Success(c, trend)
}
