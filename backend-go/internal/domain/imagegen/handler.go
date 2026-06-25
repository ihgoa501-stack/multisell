package imagegen

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles imagegen HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new imagegen handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseIGID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

func atoi64OrZero(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// ===================== ProductImageGen =====================

// ListImageGens GET /api/v1/imagegen
func (h *Handler) ListImageGens(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListImageGens(
		atoi64OrZero(c.Query("product_id")),
		c.Query("batch_id"),
		c.Query("status"),
		p.Page, p.Size,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetImageGen GET /api/v1/imagegen/:id
func (h *Handler) GetImageGen(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	g, err := h.service.GetImageGen(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "记录不存在")
		return
	}
	response.Success(c, g)
}

// CreateImageGen POST /api/v1/imagegen
func (h *Handler) CreateImageGen(c *gin.Context) {
	var g ProductImageGen
	if err := c.ShouldBindJSON(&g); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if g.ProductID == 0 || g.Prompt == "" {
		response.Error(c, http.StatusBadRequest, "product_id 和 prompt 不能为空")
		return
	}
	if g.Status == "" {
		g.Status = "pending"
	}
	if err := h.service.CreateImageGen(&g); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, g)
}

// UpdateImageGenStatus PUT /api/v1/imagegen/:id/status
func (h *Handler) UpdateImageGenStatus(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	var req struct {
		Status       string          `json:"status" binding:"required"`
		ImageURLs    json.RawMessage `json:"image_urls"`
		ErrorMessage string          `json:"error_message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if err := h.service.UpdateImageGenStatus(id, req.Status, req.ImageURLs, req.ErrorMessage); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id, "status": req.Status})
}

// DeleteImageGen DELETE /api/v1/imagegen/:id
func (h *Handler) DeleteImageGen(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteImageGen(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== ProductCanvas =====================

// ListCanvases GET /api/v1/imagegen/canvas
func (h *Handler) ListCanvases(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListCanvases(atoi64OrZero(c.Query("product_id")), p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetCanvas GET /api/v1/imagegen/canvas/:id
func (h *Handler) GetCanvas(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	cv, err := h.service.GetCanvas(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "画布不存在")
		return
	}
	response.Success(c, cv)
}

// CreateCanvas POST /api/v1/imagegen/canvas
func (h *Handler) CreateCanvas(c *gin.Context) {
	var cv ProductCanvas
	if err := c.ShouldBindJSON(&cv); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if cv.ProductID == 0 {
		response.Error(c, http.StatusBadRequest, "product_id 不能为空")
		return
	}
	if cv.Name == "" {
		cv.Name = "未命名画布"
	}
	if err := h.service.CreateCanvas(&cv); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, cv)
}

// UpdateCanvas PUT /api/v1/imagegen/canvas/:id
func (h *Handler) UpdateCanvas(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	cv, err := h.service.GetCanvas(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "画布不存在")
		return
	}
	if err := c.ShouldBindJSON(cv); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	cv.ID = id
	if err := h.service.UpdateCanvas(cv); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, cv)
}

// DeleteCanvas DELETE /api/v1/imagegen/canvas/:id
func (h *Handler) DeleteCanvas(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteCanvas(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== PromptTemplate =====================

// ListTemplates GET /api/v1/imagegen/templates
func (h *Handler) ListTemplates(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListTemplates(
		c.Query("style"),
		c.Query("platform_code"),
		atoi64OrZero(c.Query("created_by")),
		p.Page, p.Size,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetTemplate GET /api/v1/imagegen/templates/:id
func (h *Handler) GetTemplate(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	t, err := h.service.GetTemplate(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "模板不存在")
		return
	}
	response.Success(c, t)
}

// CreateTemplate POST /api/v1/imagegen/templates
func (h *Handler) CreateTemplate(c *gin.Context) {
	var t PromptTemplate
	if err := c.ShouldBindJSON(&t); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if t.Name == "" || t.Prompt == "" {
		response.Error(c, http.StatusBadRequest, "name 和 prompt 不能为空")
		return
	}
	if err := h.service.CreateTemplate(&t); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, t)
}

// UpdateTemplate PUT /api/v1/imagegen/templates/:id
func (h *Handler) UpdateTemplate(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	t, err := h.service.GetTemplate(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "模板不存在")
		return
	}
	if err := c.ShouldBindJSON(t); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	t.ID = id
	if err := h.service.UpdateTemplate(t); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, t)
}

// UseTemplate POST /api/v1/imagegen/templates/:id/use — increments usage counter.
func (h *Handler) UseTemplate(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	if err := h.service.IncrementTemplateUsage(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id, "used": true})
}

// DeleteTemplate DELETE /api/v1/imagegen/templates/:id
func (h *Handler) DeleteTemplate(c *gin.Context) {
	id, ok := parseIGID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTemplate(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
