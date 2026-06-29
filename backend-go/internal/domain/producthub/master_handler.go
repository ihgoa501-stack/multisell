package producthub

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// MasterHandler handles product master HTTP requests.
type MasterHandler struct {
	service *MasterService
}

// NewMasterHandler creates a MasterHandler.
func NewMasterHandler(svc *MasterService) *MasterHandler {
	return &MasterHandler{service: svc}
}

// List GET /api/v1/product-hub
func (h *MasterHandler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("lifecycle_status")
	items, total, err := h.service.List(c.Request.Context(), p.Page, p.Size, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /api/v1/product-hub/:id
func (h *MasterHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// Create POST /api/v1/product-hub
func (h *MasterHandler) Create(c *gin.Context) {
	var p ProductMaster
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Create(c.Request.Context(), &p); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Update PUT /api/v1/product-hub/:id
func (h *MasterHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var p ProductMaster
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p.ID = id
	if err := h.service.Update(c.Request.Context(), &p); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Delete DELETE /api/v1/product-hub/:id
func (h *MasterHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// TransitionLifecycle POST /api/v1/product-hub/:id/transition
func (h *MasterHandler) TransitionLifecycle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.TransitionLifecycle(c.Request.Context(), id, req.Status)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, p)
}
