package supplychain

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles supply chain flow HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new supply chain flow handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List returns a paginated list of supply chain flows.
// GET /api/v1/supplychain/flows?page=1&size=20&source_type=&status=
func (h *Handler) List(c *gin.Context) {
	var req ListFlowsRequest
	req.Page = common.ParsePagination(c).Page
	req.Size = common.ParsePagination(c).Size
	req.SourceType = c.Query("source_type")
	req.Status = c.Query("status")

	items, total, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list supply chain flows: "+err.Error())
		return
	}

	response.Paginated(c, items, total, req.Page, req.Size)
}

// Get returns a single supply chain flow by ID.
// GET /api/v1/supplychain/flows/:id
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "missing id")
		return
	}

	flow, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "supply chain flow not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get supply chain flow: "+err.Error())
		return
	}

	response.Success(c, flow)
}

// GetEvents returns the timeline events for a supply chain flow.
// GET /api/v1/supplychain/flows/:id/events
func (h *Handler) GetEvents(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "missing id")
		return
	}

	flow, err := h.service.GetEvents(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "supply chain flow not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get supply chain flow events: "+err.Error())
		return
	}

	response.Success(c, flow)
}

// Create creates a new supply chain flow.
// POST /api/v1/supplychain/flows
func (h *Handler) Create(c *gin.Context) {
	var flow SupplyChainFlow
	if err := c.ShouldBindJSON(&flow); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &flow); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create supply chain flow: "+err.Error())
		return
	}

	response.Success(c, flow)
}

// Update updates a supply chain flow's status and summary fields.
// PUT /api/v1/supplychain/flows/:id
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "missing id")
		return
	}

	var req UpdateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.Update(c.Request.Context(), id, req); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update supply chain flow: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}
