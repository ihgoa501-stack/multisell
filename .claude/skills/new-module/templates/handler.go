package {{ModuleName}}

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles {{module_name}} HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new {{module_name}} handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List returns a paginated list of {{module_name}}s.
// GET /api/v1/{{module_name}}s?page=1&size=20&search=xxx
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")

	items, total, err := h.service.List(c.Request.Context(), p.Page, p.Size, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list {{module_name}}s: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get returns a single {{module_name}} by ID.
// GET /api/v1/{{module_name}}s/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "{{module_name}} not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get {{module_name}}: "+err.Error())
		return
	}

	response.Success(c, item)
}

// Create creates a new {{module_name}}.
// POST /api/v1/{{module_name}}s
func (h *Handler) Create(c *gin.Context) {
	var m {{ModuleName}}
	if err := c.ShouldBindJSON(&m); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &m); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create {{module_name}}: "+err.Error())
		return
	}

	response.Success(c, m)
}

// Update updates an existing {{module_name}}.
// PUT /api/v1/{{module_name}}s/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var m {{ModuleName}}
	if err := c.ShouldBindJSON(&m); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	m.ID = id

	if err := h.service.Update(c.Request.Context(), &m); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update {{module_name}}: "+err.Error())
		return
	}

	response.Success(c, m)
}

// Delete deletes a {{module_name}}.
// DELETE /api/v1/{{module_name}}s/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete {{module_name}}: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}
