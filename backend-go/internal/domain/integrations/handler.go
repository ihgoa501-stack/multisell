package integrations

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles integrations HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new integrations handler.
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

// List GET /platform-integrations
// @Summary      List integrations
// @Description  Get paginated list of platform integration accounts
// @Tags         integrations
// @Accept       json
// @Produce      json
// @Param        page        query  int     false  "Page number"
// @Param        size        query  int     false  "Page size"
// @Param        search      query  string  false  "Search keyword"
// @Param        platform_id query  int     false  "Filter by platform"
// @Param        status      query  string  false  "Filter by status"
// @Success      200  {object}  response.PageResult
// @Security     BearerAuth
// @Router       /platform-integrations [get]
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &AccountListFilter{
		Search:     c.Query("search"),
		PlatformID: parseOptionalInt64(c, "platform_id"),
		Status:     c.Query("status"),
	}
	items, total, err := h.service.List(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /platform-integrations/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	a, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "integration account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, a)
}

// Create POST /platform-integrations
func (h *Handler) Create(c *gin.Context) {
	var in CreateAccountInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	a, err := h.service.Create(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, a)
}

// Update PUT /platform-integrations/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateAccountInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	a, err := h.service.Update(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "integration account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, a)
}

// Delete DELETE /platform-integrations/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// TestConnection POST /platform-integrations/:id/test
// @Summary      Test integration connection
// @Description  Test whether a platform integration account has valid credentials
// @Tags         integrations
// @Produce      json
// @Param        id  path  int  true  "Integration account ID"
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /platform-integrations/{id}/test [post]
func (h *Handler) TestConnection(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	r, err := h.service.TestConnection(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "integration account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, r)
}

// GetMode GET /platform-integrations/:id/mode
func (h *Handler) GetMode(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	mode, err := h.service.GetMode(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "integration account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"execution_mode": mode})
}

// UpdateMode PUT /platform-integrations/:id/mode
func (h *Handler) UpdateMode(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in struct {
		ExecutionMode int8 `json:"execution_mode" binding:"gte=0,lte=3"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.UpdateMode(id, in.ExecutionMode); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "integration account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"execution_mode": in.ExecutionMode})
}

// Sync POST /platform-integrations/:id/sync
// @Summary      Sync integration
// @Description  Trigger a sync of orders and products from the platform
// @Tags         integrations
// @Produce      json
// @Param        id  path  int  true  "Integration account ID"
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /platform-integrations/{id}/sync [post]
func (h *Handler) Sync(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	a, err := h.service.TriggerSync(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "integration account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": a.ID, "sync_status": a.SyncStatus})
}

// ListOzonProducts GET /platform-integrations/:id/ozon-products
func (h *Handler) ListOzonProducts(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	products, err := h.service.ListOzonProducts(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if products == nil {
		products = []OzonProduct{}
	}
	response.Success(c, products)
}

// PublishToOzon POST /platform-integrations/publish-to-ozon
// @Summary      Publish to Ozon
// @Description  Publish a product listing to the Ozon platform
// @Tags         integrations
// @Accept       json
// @Produce      json
// @Param        body  body  PublishToOzonInput  true  "Publish request"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /platform-integrations/publish-to-ozon [post]
func (h *Handler) PublishToOzon(c *gin.Context) {
	var in PublishToOzonInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	mode, err := h.service.GetMode(in.AccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "integration account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	ctx := WithExecutionMode(c.Request.Context(), ExecutionMode(mode))
	if in.ApprovalID != nil {
		ctx = WithApprovalID(ctx, *in.ApprovalID)
	}
	result, err := h.service.PublishToOzon(ctx, &in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}
func (h *Handler) ListCategories(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListCategoryMappings(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// CreateCategory POST /platform-integrations/:id/categories
func (h *Handler) CreateCategory(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in CreateCategoryMappingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.service.CreateCategoryMapping(id, &in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, m)
}

// ListAttributes GET /platform-integrations/:id/attributes
func (h *Handler) ListAttributes(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListAttributeMappings(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// CreateAttribute POST /platform-integrations/:id/attributes
func (h *Handler) CreateAttribute(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in CreateAttributeMappingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.service.CreateAttributeMapping(id, &in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, m)
}
