package productimage

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Capabilities(c *gin.Context) {
	if _, ok := ownerID(c); !ok {
		return
	}
	configured := h.service != nil && h.service.image != nil
	response.Success(c, []gin.H{
		{"code": "deterministic", "name": "凌镜标准处理", "configured": configured, "operations": []string{"DETERMINISTIC_RESIZE"}},
		{"code": "photoroom", "name": "Photoroom", "configured": false, "operations": []string{}},
		{"code": "adobe", "name": "Adobe Firefly", "configured": false, "operations": []string{}},
		{"code": "openai", "name": "OpenAI Images", "configured": false, "operations": []string{}},
	})
}

func ownerID(c *gin.Context) (int64, bool) {
	id := common.UserIDFromCtx(c)
	if id == nil || *id <= 0 {
		problem(c, http.StatusUnauthorized, "UNAUTHORIZED", "Owner authentication required")
		return 0, false
	}
	return *id, true
}
func taskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		problem(c, 400, "VALIDATION_ERROR", "invalid task id")
		return 0, false
	}
	return id, true
}

func (h *Handler) UploadAsset(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		problem(c, 422, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if err != nil || len(body) == 0 || len(body) > 10<<20 {
		problem(c, 422, "VALIDATION_ERROR", "file must be between 1 byte and 10 MiB")
		return
	}
	asset, err := h.service.CreateAsset(c.Request.Context(), owner, header.Filename, header.Header.Get("Content-Type"), body)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "ok", Data: asset})
}
func (h *Handler) CreateTask(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var in CreateTaskInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid task request")
		return
	}
	task, err := h.service.CreateTask(c.Request.Context(), owner, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "ok", Data: task})
}
func (h *Handler) ListTasks(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	items, total, err := h.service.ListTasks(c.Request.Context(), owner, p.Page, p.Size)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}
func (h *Handler) GetTask(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	item, err := h.service.GetTask(c.Request.Context(), owner, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, item)
}
func (h *Handler) Execute(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	var in ExecutionInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "idempotency_key is required")
		return
	}
	item, err := h.service.Execute(c.Request.Context(), owner, id, in.IdempotencyKey)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, response.Result{Code: 0, Message: "accepted", Data: item})
}
func (h *Handler) ApproveExecution(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	var in ApprovalInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid approval request")
		return
	}
	item, err := h.service.ApproveExecution(c.Request.Context(), owner, id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "approved", Data: item})
}
func (h *Handler) Attempts(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	items, err := h.service.Attempts(c.Request.Context(), owner, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}
func (h *Handler) OutputContent(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	b, media, err := h.service.OutputContent(c.Request.Context(), owner, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.Header("Content-Type", media)
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, media, b)
}

func problem(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"code": status, "message": message, "data": gin.H{"error_code": code}})
}
func writeServiceError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		problem(c, 404, "NOT_FOUND", "resource not found")
		return
	}
	if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrTruthRequiresOwner) {
		problem(c, 422, "VALIDATION_ERROR", err.Error())
		return
	}
	if errors.Is(err, ErrGateBlocked) {
		problem(c, 409, "GATE_BLOCKED", "valid image rights and five passed reviews are required")
		return
	}
	if errors.Is(err, ErrOutputHashMismatch) {
		problem(c, 502, "HASH_MISMATCH", "image output integrity verification failed")
		return
	}
	var conflict *ConflictError
	if errors.As(err, &conflict) {
		message := "request conflicts with current state"
		switch conflict.Code {
		case "IDEMPOTENCY_CONFLICT":
			message = "idempotency key conflicts with another request"
		case "APPROVAL_REQUIRED":
			message = "active Owner execution approval required"
		case "VERSION_CONFLICT":
			message = "task version or processor changed"
		case "BUDGET_COST_REQUIRED":
			message = "paid execution requires a current budget and cost record"
		}
		problem(c, 409, conflict.Code, message)
		return
	}
	var api *imageservice.APIError
	if errors.As(err, &api) {
		status := api.StatusCode
		if status < 400 || status > 599 {
			status = 502
		}
		problem(c, status, api.Code, api.Message)
		return
	}
	problem(c, 502, "IMAGE_SERVICE_ERROR", "image service unavailable")
}
