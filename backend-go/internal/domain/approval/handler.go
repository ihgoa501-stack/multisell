package approval

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler exposes HTTP handlers for approval request CRUD and review.
type Handler struct {
	service *Service
}

// NewHandler creates a new approval handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListApprovals godoc
// GET /api/v1/approval?page=1&size=20&status=pending&request_type=publish
func (h *Handler) ListApprovals(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("status")
	requestType := c.Query("request_type")

	items, total, err := h.service.List(p.Page, p.Size, status, requestType)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetApproval godoc
// GET /api/v1/approval/:id
func (h *Handler) GetApproval(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "审批ID无效")
		return
	}

	req, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "审批请求不存在")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, req)
}

// CreateApproval godoc
// POST /api/v1/approval
func (h *Handler) CreateApproval(c *gin.Context) {
	var input CreateApprovalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	req, err := h.service.Create(&input)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, req)
}

// ReviewApproval godoc
// PUT /api/v1/approval/:id/review
func (h *Handler) ReviewApproval(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "审批ID无效")
		return
	}

	var input ReviewApprovalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}
	if input.Action != "approve" && input.Action != "reject" {
		response.Error(c, http.StatusBadRequest, "动作必须是 approve 或 reject")
		return
	}

	req, err := h.service.Review(id, &input)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "审批请求不存在")
			return
		}
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, req)
}

// MyPending godoc
// GET /api/v1/approval/my?page=1&size=20
func (h *Handler) MyPending(c *gin.Context) {
	p := common.ParsePagination(c)

	items, total, err := h.service.MyPending(p.Page, p.Size)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// ApprovalStats godoc
// GET /api/v1/approval/stats
func (h *Handler) ApprovalStats(c *gin.Context) {
	stats, err := h.service.Stats()
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, stats)
}
