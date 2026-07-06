package listingtask

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/rbac"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles listing task HTTP requests.
type Handler struct {
	service *Service
	rbacSvc *rbac.Service // may be nil (RBAC check disabled)
}

// NewHandler creates a new listingtask handler.
func NewHandler(service *Service, rbacSvc *rbac.Service) *Handler {
	return &Handler{service: service, rbacSvc: rbacSvc}
}

func parseID(c *gin.Context) (int64, bool) {
	return parseIDParam(c, "id")
}

func parseIDParam(c *gin.Context, param string) (int64, bool) {
	idStr := c.Param(param)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid "+param)
		return 0, false
	}
	return id, true
}

func parseOptionalInt64(c *gin.Context, key string) *int64 {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

// getUserIDFromContext extracts the user_id from the gin context set by the auth middleware.
func getUserIDFromContext(c *gin.Context) (int64, string, bool) {
	uid, exists := c.Get("user_id")
	if !exists {
		return 0, "", false
	}
	var userID int64
	switch v := uid.(type) {
	case float64:
		userID = int64(v)
	case int64:
		userID = v
	case int:
		userID = int64(v)
	default:
		return 0, "", false
	}
	// Also try to get username/operator from context
	operator := fmt.Sprintf("user:%d", userID)
	if uname, exists := c.Get("username"); exists {
		if s, ok := uname.(string); ok {
			operator = s
		}
	}
	return userID, operator, true
}

// checkPermission checks if the authenticated user has the specified permission code.
func (h *Handler) checkPermission(c *gin.Context, permissionCode string) bool {
	if h.rbacSvc == nil {
		return true // RBAC disabled
	}
	userID, _, ok := getUserIDFromContext(c)
	if !ok {
		return false
	}
	perms, err := h.rbacSvc.GetUserPermissions(userID)
	if err != nil {
		return false
	}
	for _, p := range perms {
		if p == permissionCode {
			return true
		}
	}
	return false
}

// ---------- ListingTask handlers ----------

// List GET /listing-tasks
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("status")
	search := c.Query("search")
	platformID := parseOptionalInt64(c, "platform_id")
	productID := parseOptionalInt64(c, "product_id")
	items, total, err := h.service.List(&p, platformID, productID, status, search)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /listing-tasks/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	t, items, err := h.service.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing task not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"task": t, "items": items})
}

// Create POST /listing-tasks
func (h *Handler) Create(c *gin.Context) {
	var in CreateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.service.Create(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, t)
}

// CreateFromSuggestion POST /listing-tasks/from-suggestion
func (h *Handler) CreateFromSuggestion(c *gin.Context) {
	var in CreateFromSuggestionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	_, operator, ok := getUserIDFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	task, err := h.service.CreateFromSuggestion(in.CandidateID, operator)
	if err != nil {
		if err.Error() == "approval service not configured" {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, task)
}

// Update PUT /listing-tasks/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateTaskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.service.Update(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing task not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, t)
}

// Delete DELETE /listing-tasks/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing task not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------- ListingTaskItem handlers ----------

// ListItems GET /listing-tasks/:id/items
func (h *Handler) ListItems(c *gin.Context) {
	taskID, ok := parseID(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	items, total, err := h.service.ListItems(&p, taskID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// CreateItem POST /listing-tasks/:id/items
func (h *Handler) CreateItem(c *gin.Context) {
	taskID, ok := parseID(c)
	if !ok {
		return
	}
	var in CreateTaskItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	in.TaskID = taskID
	item, err := h.service.CreateItem(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// UpdateItem PUT /listing-tasks/:id/items/:item_id
func (h *Handler) UpdateItem(c *gin.Context) {
	itemID, ok := parseIDParam(c, "item_id")
	if !ok {
		return
	}
	var in UpdateTaskItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateItem(itemID, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "task item not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// DeleteItem DELETE /listing-tasks/:id/items/:item_id
func (h *Handler) DeleteItem(c *gin.Context) {
	itemID, ok := parseIDParam(c, "item_id")
	if !ok {
		return
	}
	if err := h.service.DeleteItem(itemID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "task item not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": itemID})
}

// ---------- Listing publish chain handlers ----------

// Execute POST /listing-task/:task_id/execute
func (h *Handler) Execute(c *gin.Context) {
	taskID, ok := parseIDParam(c, "task_id")
	if !ok {
		return
	}

	// RBAC: check listing_task:execute permission
	if !h.checkPermission(c, "listing_task:execute") {
		response.Error(c, http.StatusForbidden, "insufficient permissions: listing_task:execute required")
		return
	}

	// Get operator info from context
	_, operator, ok := getUserIDFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	task, err := h.service.ExecuteTask(taskID, operator)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing task not found")
			return
		}
		// Return 403 for gate/precondition violations that are permission-related
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, task)
}

// RetryFailed POST /listing-task/:task_id/retry-failed
func (h *Handler) RetryFailed(c *gin.Context) {
	taskID, ok := parseIDParam(c, "task_id")
	if !ok {
		return
	}
	task, err := h.service.RetryFailed(taskID)
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

// RetryItem POST /listing-task/:task_id/items/:item_id/retry
func (h *Handler) RetryItem(c *gin.Context) {
	taskID, ok := parseIDParam(c, "task_id")
	if !ok {
		return
	}
	itemID, ok := parseIDParam(c, "item_id")
	if !ok {
		return
	}
	item, err := h.service.RetryItem(taskID, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "task item not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// ---------- Review ----------

// Review GET /listing-tasks/:id/review — returns the task review with
// publish status, platform errors, and expected vs actual profit.
func (h *Handler) Review(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	review, err := h.service.ReviewTask(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "listing task not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, review)
}

// ---------- Stats / Bulk Retry ----------

// ListStats GET /listing-task/stats
func (h *Handler) ListStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, stats)
}

// RetryAll POST /listing-task/retry-all
func (h *Handler) RetryAll(c *gin.Context) {
	count, err := h.service.RetryAllTasks()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"retried": count})
}
