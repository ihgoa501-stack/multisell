package sourcing1688

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles sourcing1688 HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new sourcing1688 handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func parseOptionalInt64(c *gin.Context, key string) *int64 {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

// List GET /sourcing1688
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &ListFilter{
		Search:    c.Query("search"),
		Status:    c.Query("status"),
		ProductID: parseOptionalInt64(c, "product_id"),
	}
	items, total, err := h.service.List(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /sourcing1688/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Create POST /sourcing1688
func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.Create(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Update PUT /sourcing1688/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.Update(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Delete DELETE /sourcing1688/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// Import POST /sourcing1688/:id/import
func (h *Handler) Import(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ImportInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.Import(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Reject POST /sourcing1688/:id/reject
func (h *Handler) Reject(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in RejectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.Reject(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Summary GET /sourcing1688/summary
func (h *Handler) Summary(c *gin.Context) {
	sum, err := h.service.Summary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sum)
}

func workflowError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidWorkflow):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrWorkflowGate):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		response.InternalError(c, err)
	}
}

func (h *Handler) Snapshot(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	snapshot, err := h.service.GetSnapshot(id)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, snapshot)
}

func (h *Handler) Draft(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	draft, err := h.service.GetDraft(id)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, draft)
}

// Capture POST /sourcing-1688/capture stores immutable 1688 evidence.
func (h *Handler) Capture(c *gin.Context) {
	var in CaptureInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if actor := common.UserIDFromCtx(c); actor != nil {
		in.CollectedBy = *actor
	}
	p, err := h.service.Capture(&in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, p)
}

// Review POST /sourcing-1688/:id/review is the explicit Owner gate.
func (h *Handler) Review(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ReviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if actor := common.UserIDFromCtx(c); actor != nil {
		in.ReviewedBy = *actor
	}
	p, err := h.service.Review(id, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, p)
}

// ConvertToDraft POST /sourcing-1688/:id/convert-to-draft creates no external side effect.
func (h *Handler) ConvertToDraft(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ConvertInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if actor := common.UserIDFromCtx(c); actor != nil {
		in.CreatedBy = *actor
	}
	r, err := h.service.Convert(id, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, r)
}
