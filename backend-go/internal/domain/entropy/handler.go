package entropy

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles entropy HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new entropy handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// userIDFromCtx extracts the user_id from JWT context.
func userIDFromCtx(c *gin.Context) (int64, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case int64:
		return x, true
	case int:
		return int64(x), true
	}
	return 0, false
}

// Dashboard GET /entropy/dashboard — 熵仪表盘摘要。
func (h *Handler) Dashboard(c *gin.Context) {
	uid, ok := userIDFromCtx(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	summary, err := h.service.GetEntropySummary(uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, summary)
}

// Health GET /entropy/health — 健康评分。
func (h *Handler) Health(c *gin.Context) {
	uid, ok := userIDFromCtx(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	scores, err := h.service.GetHealthScores(uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, scores)
}

// Spc GET /entropy/spc — SPC 控制图数据。
func (h *Handler) Spc(c *gin.Context) {
	uid, ok := userIDFromCtx(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	status, err := h.service.GetSpcStatus(uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, status)
}

// Defenses GET /entropy/defenses — 防线执行结果。
func (h *Handler) Defenses(c *gin.Context) {
	uid, ok := userIDFromCtx(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	summary, err := h.service.RunDefenses(uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, summary)
}

// Changes GET /entropy/changes — 变更日志（可选 ?source_type= 过滤）。
func (h *Handler) Changes(c *gin.Context) {
	uid, ok := userIDFromCtx(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	sourceType := c.Query("source_type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	entries, total, err := h.service.GetChangeLog(uid, sourceType, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, entries, total, page, pageSize)
}

// CheckPoint POST /entropy/spc/check — 手动检查 SPC 点。
func (h *Handler) CheckPoint(c *gin.Context) {
	uid, ok := userIDFromCtx(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	var body struct {
		AgentID       string  `json:"agent_id" binding:"required"`
		DecisionPoint string  `json:"decision_point" binding:"required"`
		MetricName    string  `json:"metric_name" binding:"required"`
		CurrentValue  float64 `json:"current_value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	status, err := h.service.spc.CheckPoint(uid, body.AgentID, body.DecisionPoint, body.MetricName, body.CurrentValue)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, status)
}
