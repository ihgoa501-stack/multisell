package candidate

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles candidate product HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new candidate handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

// List GET /candidates
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("status")
	search := c.Query("search")
	sourcePlatform := c.Query("source_platform")
	completenessStatus := c.Query("completeness_status")
	items, total, err := h.service.List(&p, status, search, sourcePlatform, completenessStatus)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// ListCollectLeads GET /candidates/collect-leads
// Read-only endpoint for viewing recent collect leads from the Chrome extension.
// Supports pagination (page, size) and status filter.
func (h *Handler) ListCollectLeads(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("status")
	items, total, err := h.service.ListCollectLeads(&p, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetCollectLead GET /candidates/collect-leads/:id
func (h *Handler) GetCollectLead(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.GetCollectLeadByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "collect lead not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// Get GET /candidates/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "candidate product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// Create POST /candidates
func (h *Handler) Create(c *gin.Context) {
	var in CreateCandidateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.Create(&in)
	if err != nil {
		if errors.Is(err, ErrDuplicateSourceURL) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// Update PUT /candidates/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateCandidateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.Update(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "candidate product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// Delete DELETE /candidates/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "candidate product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// Count GET /candidates/count
func (h *Handler) Count(c *gin.Context) {
	total, err := h.service.Count()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"total": total})
}

// Dedup GET /candidates/dedup
func (h *Handler) Dedup(c *gin.Context) {
	minDup := 2
	if v := c.Query("min_dup"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 1 {
			minDup = parsed
		}
	}
	results, err := h.service.Dedup(minDup)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, results)
}

// Seed POST /candidates/seed
func (h *Handler) Seed(c *gin.Context) {
	count, err := h.service.Seed()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"seeded": count, "message": "种子数据生成成功"})
}
