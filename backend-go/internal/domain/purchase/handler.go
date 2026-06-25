package purchase

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles purchase HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new purchase handler.
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

// ---------- Order handlers ----------

// CreateOrder POST /purchase/orders
func (h *Handler) CreateOrder(c *gin.Context) {
	var in CreateOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.CreateOrder(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// ApproveOrder POST /purchase/orders/:id/approve
func (h *Handler) ApproveOrder(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	o, err := h.service.ApproveOrder(id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, o)
}

// ReceiveOrder POST /purchase/orders/:id/receive
func (h *Handler) ReceiveOrder(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ReceiveOrderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.service.ReceiveOrder(id, &in)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, o)
}

// CancelOrder POST /purchase/orders/:id/cancel
func (h *Handler) CancelOrder(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	o, err := h.service.CancelOrder(id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, o)
}

// ListOrders GET /purchase/orders
func (h *Handler) ListOrders(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &PurchaseOrderListFilter{
		Status:     c.Query("status"),
		SupplierID: parseOptionalInt64(c, "supplier_id"),
		Search:     c.Query("search"),
	}
	items, total, err := h.service.ListOrders(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetOrder GET /purchase/orders/:id
func (h *Handler) GetOrder(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	o, err := h.service.GetOrder(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "purchase order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, o)
}

// ---------- Suggestion handlers ----------

// ListSuggestions GET /purchase/suggestions
func (h *Handler) ListSuggestions(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListSuggestions(&p, c.Query("status"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GenerateSuggestions POST /purchase/suggestions/generate
func (h *Handler) GenerateSuggestions(c *gin.Context) {
	items, err := h.service.GenerateSuggestions()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// ---------- Supplier handlers ----------

// CreateSupplier POST /purchase/suppliers
func (h *Handler) CreateSupplier(c *gin.Context) {
	var in CreateSupplierInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	sup, err := h.service.CreateSupplier(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sup)
}

// UpdateSupplier PUT /purchase/suppliers/:id
func (h *Handler) UpdateSupplier(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateSupplierInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	sup, err := h.service.UpdateSupplier(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "supplier not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sup)
}

// DeleteSupplier DELETE /purchase/suppliers/:id
func (h *Handler) DeleteSupplier(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteSupplier(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ListSuppliers GET /purchase/suppliers
func (h *Handler) ListSuppliers(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListSuppliers(&p, c.Query("search"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetSupplier GET /purchase/suppliers/:id
func (h *Handler) GetSupplier(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	sup, err := h.service.GetSupplier(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "supplier not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sup)
}

// GetSupplierKPI GET /purchase/suppliers/:id/kpi
func (h *Handler) GetSupplierKPI(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	kpi, err := h.service.GetSupplierKPI(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "supplier not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, kpi)
}
