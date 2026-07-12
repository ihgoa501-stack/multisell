package listing

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/listingtask"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles listing HTTP requests.
type Handler struct {
	service     *Service
	approvalSvc *approval.Service
}

// NewHandler creates a new listing handler.
func NewHandler(service *Service, approvalSvc *approval.Service) *Handler {
	return &Handler{service: service, approvalSvc: approvalSvc}
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

func parseParamInt64(c *gin.Context, param string) (int64, bool) {
	v, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid "+param)
		return 0, false
	}
	return v, true
}

// List GET /listings
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("status")
	search := c.Query("search")
	var platformID *int64
	if pidStr := c.Query("platform_id"); pidStr != "" {
		if v, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
			platformID = &v
		}
	}
	items, total, err := h.service.List(&p, platformID, status, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /listings/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	l, err := h.service.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, l)
}

// Create POST /listings
func (h *Handler) Create(c *gin.Context) {
	var in CreateListingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	l, err := h.service.Create(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, l)
}

// Update PUT /listings/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateListingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	l, err := h.service.Update(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, l)
}

// Delete DELETE /listings/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// Publish POST /listings/:id/publish
func (h *Handler) Publish(c *gin.Context) {
	_, ok := parseID(c)
	if !ok {
		return
	}
	response.Error(c, http.StatusPreconditionRequired, listingtask.ImageReleaseAttestationRequiredMessage)
}

// Sync POST /listings/:id/sync
func (h *Handler) Sync(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		Status      string `json:"status"`
		SyncMessage string `json:"sync_message"`
	}
	_ = c.ShouldBindJSON(&body) // optional body
	l, err := h.service.SyncStatus(id, body.Status, body.SyncMessage)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, l)
}

// ---------- Listing publish chain handlers ----------

// PublishProduct POST /listing/products/:product_id/publish/:platform_id
func (h *Handler) PublishProduct(c *gin.Context) {
	_, ok := parseParamInt64(c, "product_id")
	if !ok {
		return
	}
	_, ok = parseParamInt64(c, "platform_id")
	if !ok {
		return
	}
	response.Error(c, http.StatusPreconditionRequired, listingtask.ImageReleaseAttestationRequiredMessage)
}

// ListByProduct GET /listing/products/:product_id/listings
func (h *Handler) ListByProduct(c *gin.Context) {
	productID, ok := parseParamInt64(c, "product_id")
	if !ok {
		return
	}
	items, err := h.service.ListByProduct(productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"items": items, "total": len(items)})
}

// GetPlatformComparison GET /listing/products/:product_id/platform-comparison
func (h *Handler) GetPlatformComparison(c *gin.Context) {
	productID, ok := parseParamInt64(c, "product_id")
	if !ok {
		return
	}
	results, err := h.service.GetPlatformComparison(productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"items": results, "total": len(results)})
}

// CreateTasksFromDecisions POST /listing/listing-tasks/from-decisions
func (h *Handler) CreateTasksFromDecisions(c *gin.Context) {
	var in CreateTasksFromDecisionsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	tasks, err := h.service.CreateTasksFromDecisions(in.DecisionIDs)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "no decisions found for given ids")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"tasks": tasks, "created": len(tasks)})
}

// RecheckTask POST /listing/listing-tasks/:task_id/recheck
func (h *Handler) RecheckTask(c *gin.Context) {
	taskID, ok := parseParamInt64(c, "task_id")
	if !ok {
		return
	}
	task, err := h.service.RecheckTask(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing task not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, task)
}

// CancelTask POST /listing/listing-tasks/:task_id/cancel
func (h *Handler) CancelTask(c *gin.Context) {
	taskID, ok := parseParamInt64(c, "task_id")
	if !ok {
		return
	}
	var in CancelTaskInput
	_ = c.ShouldBindJSON(&in) // body optional
	task, err := h.service.CancelTask(taskID, in.Reason)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing task not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, task)
}

// PublishTask POST /listing/listing-tasks/:task_id/publish
func (h *Handler) PublishTask(c *gin.Context) {
	_, ok := parseParamInt64(c, "task_id")
	if !ok {
		return
	}

	response.Error(c, http.StatusPreconditionRequired, listingtask.ImageReleaseAttestationRequiredMessage)
}

// Suggest POST /listings/suggest
func (h *Handler) Suggest(c *gin.Context) {
	var in SuggestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	suggestion, err := h.service.GenerateSuggestion(c.Request.Context(), in.CandidateID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "candidate not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, suggestion)
}
