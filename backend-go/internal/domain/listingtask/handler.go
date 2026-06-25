package listingtask

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles listing task HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new listingtask handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
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
	task, err := h.service.ExecuteTask(taskID)
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
