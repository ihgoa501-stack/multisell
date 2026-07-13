package productanalysis

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles product analysis HTTP requests.
type Handler struct{ service Service }

// NewHandler creates a new Handler.
func NewHandler(svc Service) *Handler {
	return &Handler{service: svc}
}

// Analyze POST /api/v1/product-analysis/analyze
func (h *Handler) Analyze(c *gin.Context) {
	var in AnalyzeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	userID := currentUserID(c)

	// Rate limit: simple in-memory per-user throttle
	if !acquireSlot(userID) {
		response.Error(c, http.StatusTooManyRequests, "操作太频繁，请稍后再试")
		return
	}
	defer releaseSlot(userID)

	result, err := h.service.Analyze(&in, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// GetAnalysis GET /api/v1/product-analysis/analyses/:id
func (h *Handler) GetAnalysis(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}
	userID := currentUserID(c)

	a, err := h.service.GetAnalysis(id, userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, a)
}

// ListAnalyses GET /api/v1/product-analysis/analyses
func (h *Handler) ListAnalyses(c *gin.Context) {
	userID := currentUserID(c)
	p := common.ParsePagination(c)
	filter := &ListFilter{
		UserID: userID,
		Status: c.Query("status"),
		Page:   p.Page,
		Size:   p.Size,
	}
	items, total, err := h.service.ListAnalyses(filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// RecordFeedback POST /api/v1/product-analysis/analyses/:id/feedback
func (h *Handler) RecordFeedback(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var in FeedbackInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	userID := currentUserID(c)

	if err := h.service.RecordFeedback(id, &in, userID); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "反馈已记录"})
}

// ---- rate limiter ----

var (
	slotsMu sync.Mutex
	slots   = make(map[string]time.Time)
)

func currentUserID(c *gin.Context) string {
	if userID, ok := c.Get("user_id"); ok {
		if s, ok := userID.(string); ok && s != "" {
			return s
		}
	}
	return "anonymous"
}

func acquireSlot(key string) bool {
	slotsMu.Lock()
	defer slotsMu.Unlock()

	now := time.Now()
	if last, ok := slots[key]; ok && now.Sub(last) < time.Second {
		return false
	}
	slots[key] = now
	return true
}

func releaseSlot(key string) {
	slotsMu.Lock()
	defer slotsMu.Unlock()

	delete(slots, key)
}
