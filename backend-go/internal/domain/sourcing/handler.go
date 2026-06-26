package sourcing

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles sourcing HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new sourcing handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Fetch POST /sourcing/fetch
// Accepts a 1688 URL, fetches product data, analyzes it, and saves a recommendation.
func (h *Handler) Fetch(c *gin.Context) {
	var req FetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: url is required")
		return
	}

	// Step 1: Fetch product data via ToolBridge.
	pageData, err := h.service.FetchProduct(c.Request.Context(), req.URL)
	if err != nil {
		response.Error(c, http.StatusBadGateway, err.Error())
		return
	}

	// Step 2: Analyze page quality.
	score, reason := h.service.AnalyzePage(pageData)

	// Step 3: Save recommendation and publish event.
	product, err := h.service.SaveRecommendation(c.Request.Context(), pageData, score, reason)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"product_id": product.ID,
		"title":      pageData.Title,
		"price":      pageData.Price,
		"score":      score,
		"status":     product.Status,
	})
}

// ListRecommendations GET /sourcing/recommendations
// Lists past recommendations with pagination.
func (h *Handler) ListRecommendations(c *gin.Context) {
	page := 1
	size := 20

	// Simple pagination from query params.
	if p := c.Query("page"); p != "" {
		if v, err := parseInt(p); err == nil && v > 0 {
			page = v
		}
	}
	if s := c.Query("size"); s != "" {
		if v, err := parseInt(s); err == nil && v > 0 && v <= 100 {
			size = v
		}
	}

	items, total, err := h.service.ListRecommendations(page, size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Paginated(c, items, total, page, size)
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
