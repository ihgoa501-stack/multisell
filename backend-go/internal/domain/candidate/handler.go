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
// @Summary      List candidates
// @Description  Get paginated list of candidate products
// @Tags         candidates
// @Accept       json
// @Produce      json
// @Param        page    query  int     false  "Page number"
// @Param        size    query  int     false  "Page size"
// @Param        status  query  string  false  "Filter by status"
// @Param        search  query  string  false  "Search keyword"
// @Success      200  {object}  response.PageResult
// @Security     BearerAuth
// @Router       /candidates [get]
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	filter := &ListCandidateFilter{
		Status:             c.Query("status"),
		Search:             c.Query("search"),
		CompletenessStatus: c.Query("completeness_status"),
	}
	items, total, err := h.service.List(&p, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// ListCollectLeads GET /candidates/collect-leads
// Read-only endpoint for viewing recent collect leads from the Chrome extension.
func (h *Handler) ListCollectLeads(c *gin.Context) {
	limit := 20
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && l > 0 {
		limit = l
	}
	items, err := h.service.ListCollectLeads(limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	total, _ := h.service.CountCollectLeads()
	response.Success(c, map[string]interface{}{
		"items": items,
		"total": total,
	})
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
// @Summary      Get candidate detail
// @Description  Get a single candidate product by ID
// @Tags         candidates
// @Produce      json
// @Param        id   path  int  true  "Candidate ID"
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /candidates/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.GetDetail(id)
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
// @Summary      Create candidate
// @Description  Create a new candidate product
// @Tags         candidates
// @Accept       json
// @Produce      json
// @Param        body  body  CreateCandidateInput  true  "Candidate data"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /candidates [post]
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
// @Summary      Update candidate
// @Description  Update an existing candidate product
// @Tags         candidates
// @Accept       json
// @Produce      json
// @Param        id    path  int                 true  "Candidate ID"
// @Param        body  body  UpdateCandidateInput  true  "Update data"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /candidates/{id} [put]
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
// @Summary      Delete candidate
// @Description  Delete a candidate product by ID
// @Tags         candidates
// @Produce      json
// @Param        id  path  int  true  "Candidate ID"
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /candidates/{id} [delete]
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
// @Summary      Count candidates
// @Description  Get total count of candidate products
// @Tags         candidates
// @Produce      json
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /candidates/count [get]
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

// respondErr maps common candidate errors to HTTP responses.
// Returns true if the error was handled (response already written).
func (h *Handler) respondErr(c *gin.Context, err error) bool {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, http.StatusNotFound, "candidate product not found")
		return true
	}
	if errors.Is(err, ErrFieldNotRecognized) || errors.Is(err, ErrRescrapeNoSource) {
		response.Error(c, http.StatusBadRequest, err.Error())
		return true
	}
	return false
}

// FillFields PUT /candidates/:id/fields
// @Summary      Fill candidate fields
// @Description  Manually fill completeness fields for a candidate
// @Tags         candidates
// @Accept       json
// @Produce      json
// @Param        id    path  int               true  "Candidate ID"
// @Param        body  body  FillFieldsInput   true  "Field values"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /candidates/{id}/fields [put]
func (h *Handler) FillFields(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in FillFieldsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.ManualFillFields(id, &in)
	if err != nil {
		if h.respondErr(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// SkipField POST /candidates/:id/skip-field
// Mark a field as intentionally skipped (cannot be provided), recalculate status.
func (h *Handler) SkipField(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in SkipFieldInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.SkipField(id, in.Field)
	if err != nil {
		if h.respondErr(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// Rescrape POST /candidates/:id/rescrape
// Trigger re-collection from the source URL, then recalculate status.
func (h *Handler) Rescrape(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.Rescrape(id)
	if err != nil {
		if h.respondErr(c, err) {
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}
