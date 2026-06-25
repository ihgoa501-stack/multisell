package platform

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles platform & store HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new platform handler.
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

// ---------- Platform handlers ----------

// ListPlatforms GET /platforms
func (h *Handler) ListPlatforms(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")
	items, total, err := h.service.ListPlatforms(&p, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetPlatform GET /platforms/:id
func (h *Handler) GetPlatform(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.service.GetPlatform(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "platform not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// CreatePlatform POST /platforms
func (h *Handler) CreatePlatform(c *gin.Context) {
	var in CreatePlatformInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.CreatePlatform(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// UpdatePlatform PUT /platforms/:id
func (h *Handler) UpdatePlatform(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdatePlatformInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.UpdatePlatform(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "platform not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// DeletePlatform DELETE /platforms/:id
func (h *Handler) DeletePlatform(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeletePlatform(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "platform not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------- Store handlers ----------

// ListStores GET /stores
func (h *Handler) ListStores(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")
	var platformID *int64
	if pidStr := c.Query("platform_id"); pidStr != "" {
		if v, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
			platformID = &v
		}
	}
	items, total, err := h.service.ListStores(&p, platformID, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetStore GET /stores/:id
func (h *Handler) GetStore(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	st, err := h.service.GetStore(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "store not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, st)
}

// CreateStore POST /stores
func (h *Handler) CreateStore(c *gin.Context) {
	var in CreateStoreInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	st, err := h.service.CreateStore(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, st)
}

// UpdateStore PUT /stores/:id
func (h *Handler) UpdateStore(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateStoreInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	st, err := h.service.UpdateStore(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "store not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, st)
}

// DeleteStore DELETE /stores/:id
func (h *Handler) DeleteStore(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteStore(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "store not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
