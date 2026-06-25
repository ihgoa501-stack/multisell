package search

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles search HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new search handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Search GET /search?q=keyword&limit=20
func (h *Handler) Search(c *gin.Context) {
	q := c.Query("q")
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	items, err := h.service.Search(q, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// Recent GET /search/recent
func (h *Handler) Recent(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	items := h.service.Recent(userID)
	response.Success(c, items)
}
