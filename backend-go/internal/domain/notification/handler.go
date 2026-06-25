package notification

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles notification HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new notification handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseNID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

func atoiPtr(s string) *int {
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

func atoi64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// List GET /api/v1/notification
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	uid, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未认证")
		return
	}
	var userID int64
	switch v := uid.(type) {
	case float64:
		userID = int64(v)
	case int64:
		userID = v
	case int:
		userID = int64(v)
	}
	f := ListFilter{
		UserID:    userID,
		AlertType: c.Query("alert_type"),
		Severity:  c.Query("severity"),
		IsRead:    atoiPtr(c.Query("is_read")),
	}
	items, total, err := h.service.List(f, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /api/v1/notification/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseNID(c)
	if !ok {
		return
	}
	n, err := h.service.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "通知不存在")
		return
	}
	response.Success(c, n)
}

// Create POST /api/v1/notification
func (h *Handler) Create(c *gin.Context) {
	var n Notification
	if err := c.ShouldBindJSON(&n); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if n.UserID == 0 || n.AlertType == "" || n.Title == "" {
		response.Error(c, http.StatusBadRequest, "user_id, alert_type, title 不能为空")
		return
	}
	if err := h.service.Create(&n); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, n)
}

// MarkAsRead PUT /api/v1/notification/:id/read
func (h *Handler) MarkAsRead(c *gin.Context) {
	id, ok := parseNID(c)
	if !ok {
		return
	}
	if err := h.service.MarkAsRead(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id, "is_read": 1})
}

// MarkAllRead PUT /api/v1/notification/read-all?user_id=
func (h *Handler) MarkAllRead(c *gin.Context) {
	userID := atoi64(c.Query("user_id"))
	if userID == 0 {
		response.Error(c, http.StatusBadRequest, "user_id 不能为空")
		return
	}
	if err := h.service.MarkAllRead(userID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"user_id": userID, "marked": true})
}

// UnreadCount GET /api/v1/notification/unread-count?user_id=
func (h *Handler) UnreadCount(c *gin.Context) {
	userID := atoi64(c.Query("user_id"))
	if userID == 0 {
		response.Error(c, http.StatusBadRequest, "user_id 不能为空")
		return
	}
	count, err := h.service.UnreadCount(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"user_id": userID, "unread_count": count})
}

// Delete DELETE /api/v1/notification/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseNID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== Alert Rules =====================

// ListAlertRules GET /api/v1/notification/alert-rules
func (h *Handler) ListAlertRules(c *gin.Context) {
	rules, err := h.service.ListAlertRules()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, rules)
}

// CreateAlertRule POST /api/v1/notification/alert-rules
func (h *Handler) CreateAlertRule(c *gin.Context) {
	var r AlertRule
	if err := c.ShouldBindJSON(&r); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if r.Name == "" || r.AlertType == "" {
		response.Error(c, http.StatusBadRequest, "name 和 alert_type 不能为空")
		return
	}
	if err := h.service.CreateAlertRule(&r); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// UpdateAlertRule PUT /api/v1/notification/alert-rules/:id
func (h *Handler) UpdateAlertRule(c *gin.Context) {
	id, ok := parseNID(c)
	if !ok {
		return
	}
	r, err := h.service.GetAlertRule(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "规则不存在")
		return
	}
	if err := c.ShouldBindJSON(r); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	r.ID = id
	if err := h.service.UpdateAlertRule(r); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// DeleteAlertRule DELETE /api/v1/notification/alert-rules/:id
func (h *Handler) DeleteAlertRule(c *gin.Context) {
	id, ok := parseNID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteAlertRule(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
