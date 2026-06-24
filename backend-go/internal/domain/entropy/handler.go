package entropy

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetSummary GET /entropy
func (h *Handler) GetSummary(c *gin.Context) {
	userID := userIDFromCtx(c)
	summary, err := h.service.GetEntropySummary(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, summary)
}

// RunDefenses POST /entropy/defense
func (h *Handler) RunDefenses(c *gin.Context) {
	userID := userIDFromCtx(c)
	result, err := h.service.RunDefenses(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// GetHealthScores GET /entropy/health
func (h *Handler) GetHealthScores(c *gin.Context) {
	userID := userIDFromCtx(c)
	scores, err := h.service.GetHealthScores(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, scores)
}

// GetSpcStatus GET /entropy/spc
func (h *Handler) GetSpcStatus(c *gin.Context) {
	userID := userIDFromCtx(c)
	status, err := h.service.GetSpcStatus(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, status)
}

// GetChangeLog GET /entropy/changelog
func (h *Handler) GetChangeLog(c *gin.Context) {
	userID := userIDFromCtx(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	logs, total, err := h.service.GetChangeLog(userID, c.Query("source_type"), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, logs, total, page, pageSize)
}

// userIDFromCtx extracts user id from JWT context.
func userIDFromCtx(c *gin.Context) int64 {
	v, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	switch x := v.(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	}
	return 0
}
