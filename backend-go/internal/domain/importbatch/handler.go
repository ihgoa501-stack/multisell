package importbatch

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles importbatch HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new importbatch handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseIBID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

// ListBatches GET /api/v1/importbatch
func (h *Handler) ListBatches(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListBatches(c.Query("type"), c.Query("status"), p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetBatch GET /api/v1/importbatch/:id
func (h *Handler) GetBatch(c *gin.Context) {
	id, ok := parseIBID(c)
	if !ok {
		return
	}
	b, err := h.service.GetBatch(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "批次不存在")
		return
	}
	response.Success(c, b)
}

// CreateBatch POST /api/v1/importbatch
func (h *Handler) CreateBatch(c *gin.Context) {
	var b ImportBatch
	if err := c.ShouldBindJSON(&b); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if b.Type == "" {
		response.Error(c, http.StatusBadRequest, "type 不能为空")
		return
	}
	if err := h.service.CreateBatch(&b); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, b)
}

// UpdateBatch PUT /api/v1/importbatch/:id
func (h *Handler) UpdateBatch(c *gin.Context) {
	id, ok := parseIBID(c)
	if !ok {
		return
	}
	b, err := h.service.GetBatch(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "批次不存在")
		return
	}
	if err := c.ShouldBindJSON(b); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	b.ID = id
	if err := h.service.UpdateBatch(b); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, b)
}

// DeleteBatch DELETE /api/v1/importbatch/:id
func (h *Handler) DeleteBatch(c *gin.Context) {
	id, ok := parseIBID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteBatch(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ListRows GET /api/v1/importbatch/:id/rows
func (h *Handler) ListRows(c *gin.Context) {
	id, ok := parseIBID(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	items, total, err := h.service.ListRows(id, c.Query("status"), p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}
