package mock

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles mock data HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new mock handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Seed POST /mock/seed
func (h *Handler) Seed(c *gin.Context) {
	if err := h.service.SeedMockData(); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "mock data seeded"})
}

// ListOrders GET /mock/orders
func (h *Handler) ListOrders(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListOrders(p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// ListSettlements GET /mock/settlements
func (h *Handler) ListSettlements(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListSettlements(p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// SyncStatuses GET /mock/sync-statuses
func (h *Handler) SyncStatuses(c *gin.Context) {
	items, err := h.service.GetSyncStatuses()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}
