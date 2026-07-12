package category

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles category HTTP requests.
type Handler struct {
	service *Service
}

type updateCategoryInput struct {
	Name      *string `json:"name"`
	ParentID  *int64  `json:"parent_id"`
	Level     *int    `json:"level"`
	SortOrder *int    `json:"sort_order"`
	Status    *int16  `json:"status"`
}

// NewHandler creates a new category handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List returns a paginated list of categories.
// GET /api/v1/categories?page=1&size=20&search=xxx
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")

	items, total, err := h.service.List(c.Request.Context(), p.Page, p.Size, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list categories: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get returns a single category by ID.
// GET /api/v1/categories/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "category not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get category: "+err.Error())
		return
	}

	response.Success(c, item)
}

// Create creates a new category.
// POST /api/v1/categories
func (h *Handler) Create(c *gin.Context) {
	var cat Category
	if err := c.ShouldBindJSON(&cat); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &cat); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create category: "+err.Error())
		return
	}

	response.Success(c, cat)
}

// Update updates an existing category.
// PUT /api/v1/categories/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	cat, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "category not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get category: "+err.Error())
		return
	}

	var input updateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if input.Name != nil {
		cat.Name = *input.Name
	}
	if input.ParentID != nil {
		cat.ParentID = *input.ParentID
	}
	if input.Level != nil {
		cat.Level = *input.Level
	}
	if input.SortOrder != nil {
		cat.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		cat.Status = *input.Status
	}

	if err := h.service.Update(c.Request.Context(), cat); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update category: "+err.Error())
		return
	}

	response.Success(c, cat)
}

// Delete deletes a category.
// DELETE /api/v1/categories/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete category: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// Tree returns the category tree.
// GET /api/v1/categories/tree
func (h *Handler) Tree(c *gin.Context) {
	tree, err := h.service.Tree(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to build tree: "+err.Error())
		return
	}

	response.Success(c, tree)
}
