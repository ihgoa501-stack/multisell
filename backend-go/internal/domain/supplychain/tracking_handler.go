package supplychain

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// TrackingHandler handles supply chain tracking HTTP requests.
type TrackingHandler struct {
	service *TrackingService
}

// NewTrackingHandler creates a new supply chain tracking handler.
func NewTrackingHandler(service *TrackingService) *TrackingHandler {
	return &TrackingHandler{service: service}
}

// List returns a paginated list of tracking records.
// GET /api/v1/supplychain/tracking?page=1&size=20&flow_id=&order_id=&status=&carrier_code=
func (h *TrackingHandler) List(c *gin.Context) {
	req := ListTrackingRequest{}
	req.FlowID = c.Query("flow_id")
	req.OrderID = c.Query("order_id")
	req.Status = c.Query("status")
	req.CarrierCode = c.Query("carrier_code")

	p := common.ParsePagination(c)
	req.Page = p.Page
	req.Size = p.Size

	items, total, err := h.service.List(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list tracking records: "+err.Error())
		return
	}

	response.Paginated(c, items, total, req.Page, req.Size)
}

// Get returns a single tracking record by ID.
// GET /api/v1/supplychain/tracking/:id
func (h *TrackingHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "missing id")
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "tracking record not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get tracking record: "+err.Error())
		return
	}

	response.Success(c, item)
}

// GetByFlow returns tracking records for a given flow ID.
// GET /api/v1/supplychain/tracking/flow/:flowId
func (h *TrackingHandler) GetByFlow(c *gin.Context) {
	flowID := c.Param("flowId")
	if flowID == "" {
		response.Error(c, http.StatusBadRequest, "missing flow_id")
		return
	}

	items, err := h.service.GetByFlowID(c.Request.Context(), flowID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get tracking records by flow: "+err.Error())
		return
	}

	response.Success(c, items)
}

// Create creates a new tracking record.
// POST /api/v1/supplychain/tracking
func (h *TrackingHandler) Create(c *gin.Context) {
	var req CreateTrackingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	item, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create tracking record: "+err.Error())
		return
	}

	response.Success(c, item)
}

// UpdateStatus updates a tracking record's status.
// PUT /api/v1/supplychain/tracking/:id/status
func (h *TrackingHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "missing id")
		return
	}

	var req UpdateTrackingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	item, err := h.service.UpdateStatus(c.Request.Context(), id, &req)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "tracking record not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update tracking record: "+err.Error())
		return
	}

	response.Success(c, item)
}
