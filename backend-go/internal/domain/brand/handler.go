package brand

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles brand HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new brand handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List returns a paginated list of brands.
// GET /api/v1/brands?page=1&size=20&search=xxx
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")

	items, total, err := h.service.List(c.Request.Context(), p.Page, p.Size, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list brands: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get returns a single brand by ID.
// GET /api/v1/brands/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "brand not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get brand: "+err.Error())
		return
	}

	response.Success(c, item)
}

// Create creates a new brand.
// POST /api/v1/brands
func (h *Handler) Create(c *gin.Context) {
	var b Brand
	if err := c.ShouldBindJSON(&b); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &b); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create brand: "+err.Error())
		return
	}

	response.Success(c, b)
}

// Update updates an existing brand.
// PUT /api/v1/brands/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var b Brand
	if err := c.ShouldBindJSON(&b); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	b.ID = id

	if err := h.service.Update(c.Request.Context(), &b); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update brand: "+err.Error())
		return
	}

	response.Success(c, b)
}

// Delete deletes a brand.
// DELETE /api/v1/brands/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete brand: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}
