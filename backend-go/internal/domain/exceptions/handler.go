package exceptions

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles exceptions HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new exceptions handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseEID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

// List GET /api/v1/exceptions
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	f := ListFilter{
		SourceModule: c.Query("source_module"),
		SourceType:   c.Query("source_type"),
		Severity:     c.Query("severity"),
		Status:       c.Query("status"),
		AssignedTo:   c.Query("assigned_to"),
	}
	items, total, err := h.service.List(f, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /api/v1/exceptions/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseEID(c)
	if !ok {
		return
	}
	e, err := h.service.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "异常不存在")
		return
	}
	response.Success(c, e)
}

// Create POST /api/v1/exceptions
func (h *Handler) Create(c *gin.Context) {
	var e ExceptionItem
	if err := c.ShouldBindJSON(&e); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if e.SourceModule == "" || e.Title == "" {
		response.Error(c, http.StatusBadRequest, "source_module 和 title 不能为空")
		return
	}
	if err := h.service.Create(&e); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, e)
}

type resolveRequest struct {
	Note       string `json:"note"`
	ResolvedBy string `json:"resolved_by"`
}

// Resolve PUT /api/v1/exceptions/:id/resolve
func (h *Handler) Resolve(c *gin.Context) {
	id, ok := parseEID(c)
	if !ok {
		return
	}
	var req resolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// allow empty body; resolve with operator from query
		req.ResolvedBy = c.Query("operator")
		req.Note = c.Query("note")
	}
	if req.ResolvedBy == "" {
		req.ResolvedBy = c.Query("operator")
	}
	e, err := h.service.Resolve(id, req.ResolvedBy, req.Note)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, e)
}

type assignRequest struct {
	AssignedTo string `json:"assigned_to"`
}

// Assign PUT /api/v1/exceptions/:id/assign
func (h *Handler) Assign(c *gin.Context) {
	id, ok := parseEID(c)
	if !ok {
		return
	}
	var req assignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.AssignedTo = c.Query("assigned_to")
	}
	if req.AssignedTo == "" {
		response.Error(c, http.StatusBadRequest, "assigned_to 不能为空")
		return
	}
	e, err := h.service.Assign(id, req.AssignedTo)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, e)
}

// Update PUT /api/v1/exceptions/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseEID(c)
	if !ok {
		return
	}
	e, err := h.service.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "异常不存在")
		return
	}
	if err := c.ShouldBindJSON(e); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	e.ID = id
	if err := h.service.Update(e); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, e)
}

// Delete DELETE /api/v1/exceptions/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseEID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// AutoDetect POST /api/v1/exceptions/auto-detect
func (h *Handler) AutoDetect(c *gin.Context) {
	items, err := h.service.AutoDetect(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []ExceptionItem{}
	}
	response.Success(c, items)
}
