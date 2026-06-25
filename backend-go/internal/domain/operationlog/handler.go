package operationlog

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles operationlog HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new operationlog handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseOLID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// List GET /api/v1/operationlog
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	f := ListFilter{
		Module:   c.Query("module"),
		Action:   c.Query("action"),
		Operator: c.Query("operator"),
		From:     parseTime(c.Query("from")),
		To:       parseTime(c.Query("to")),
	}
	items, total, err := h.service.List(f, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /api/v1/operationlog/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseOLID(c)
	if !ok {
		return
	}
	l, err := h.service.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "日志不存在")
		return
	}
	response.Success(c, l)
}

// Create POST /api/v1/operationlog
func (h *Handler) Create(c *gin.Context) {
	var l OperationLog
	if err := c.ShouldBindJSON(&l); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if err := h.service.Create(&l); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, l)
}
