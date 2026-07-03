package sku

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles SKU and Product HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new SKU handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ── Product handlers ─────────────────────────────────────────────

// ListProducts returns a paginated list of products.
// GET /api/v1/products?page=1&size=20&search=xxx&category_id=&brand_id=&status=
// @Summary      List products
// @Description  Get paginated list of products
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        page        query  int     false  "Page number"
// @Param        size        query  int     false  "Page size"
// @Param        search      query  string  false  "Search keyword"
// @Param        category_id query  int     false  "Filter by category"
// @Param        brand_id    query  int     false  "Filter by brand"
// @Param        status      query  int     false  "Filter by status"
// @Success      200  {object}  response.PageResult
// @Security     BearerAuth
// @Router       /products [get]
func (h *Handler) ListProducts(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")

	categoryID, _ := strconv.ParseInt(c.Query("category_id"), 10, 64)
	brandID, _ := strconv.ParseInt(c.Query("brand_id"), 10, 64)
	var status *int16
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			sv := int16(v)
			status = &sv
		}
	}

	items, total, err := h.service.ListProducts(c.Request.Context(), p.Page, p.Size, search, categoryID, brandID, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list products: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetProduct returns a single product by ID.
// GET /api/v1/product-master/:id
// @Summary      Get product
// @Description  Get a single product by ID
// @Tags         products
// @Produce      json
// @Param        id  path  int  true  "Product ID"
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /products/{id} [get]
func (h *Handler) GetProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetProductByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get product: "+err.Error())
		return
	}

	response.Success(c, item)
}

// CreateProduct creates a new product.
// POST /api/v1/product-master
// @Summary      Create product
// @Description  Create a new product
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        body  body  Product  true  "Product data"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /products [post]
func (h *Handler) CreateProduct(c *gin.Context) {
	var p Product
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.CreateProduct(c.Request.Context(), &p); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create product: "+err.Error())
		return
	}

	response.Success(c, p)
}

// UpdateProduct updates an existing product.
// PUT /api/v1/product-master/:id
// @Summary      Update product
// @Description  Update an existing product
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id    path  int      true  "Product ID"
// @Param        body  body  Product  true  "Updated product data"
// @Success      200   {object}  response.Result
// @Security     BearerAuth
// @Router       /products/{id} [put]
func (h *Handler) UpdateProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var p Product
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	p.ID = id

	if err := h.service.UpdateProduct(c.Request.Context(), &p); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update product: "+err.Error())
		return
	}

	response.Success(c, p)
}

// DeleteProduct deletes a product.
// DELETE /api/v1/product-master/:id
// @Summary      Delete product
// @Description  Delete a product by ID
// @Tags         products
// @Produce      json
// @Param        id  path  int  true  "Product ID"
// @Success      200  {object}  response.Result
// @Security     BearerAuth
// @Router       /products/{id} [delete]
func (h *Handler) DeleteProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteProduct(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete product: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// ── SpecName handlers ────────────────────────────────────────────

// ListSpecs returns spec names (with values) for a product.
// GET /api/v1/product-master/:id/specs
func (h *Handler) ListSpecs(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	items, err := h.service.ListSpecNames(c.Request.Context(), productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list specs: "+err.Error())
		return
	}

	response.Success(c, items)
}

// CreateSpec creates a new spec name.
// POST /api/v1/product-master/:id/specs
func (h *Handler) CreateSpec(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var sn SpecName
	if err := c.ShouldBindJSON(&sn); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sn.ProductID = productID

	if err := h.service.CreateSpecName(c.Request.Context(), &sn); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create spec: "+err.Error())
		return
	}

	response.Success(c, sn)
}

// UpdateSpec updates a spec name.
// PUT /api/v1/product-master/:id/specs/:spec_id
func (h *Handler) UpdateSpec(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("spec_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var sn SpecName
	if err := c.ShouldBindJSON(&sn); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sn.ID = id

	if err := h.service.UpdateSpecName(c.Request.Context(), &sn); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update spec: "+err.Error())
		return
	}

	response.Success(c, sn)
}

// DeleteSpec deletes a spec name.
// DELETE /api/v1/product-master/:id/specs/:spec_id
func (h *Handler) DeleteSpec(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("spec_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteSpecName(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete spec: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// ── SpecValue handlers ────────────────────────────────────────────

// CreateSpecValue creates a new spec value.
// POST /api/v1/product-master/:id/specs/:spec_id/values
func (h *Handler) CreateSpecValue(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	specNameID, err := strconv.ParseInt(c.Param("spec_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid spec_id")
		return
	}

	var sv SpecValue
	if err := c.ShouldBindJSON(&sv); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sv.ProductID = productID
	sv.SpecNameID = specNameID

	if err := h.service.CreateSpecValue(c.Request.Context(), &sv); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create spec value: "+err.Error())
		return
	}

	response.Success(c, sv)
}

// UpdateSpecValue updates a spec value.
// PUT /api/v1/spec-values/:id
func (h *Handler) UpdateSpecValue(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var sv SpecValue
	if err := c.ShouldBindJSON(&sv); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sv.ID = id

	if err := h.service.UpdateSpecValue(c.Request.Context(), &sv); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update spec value: "+err.Error())
		return
	}

	response.Success(c, sv)
}

// DeleteSpecValue deletes a spec value.
// DELETE /api/v1/spec-values/:id
func (h *Handler) DeleteSpecValue(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteSpecValue(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete spec value: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// ── SKU handlers ──────────────────────────────────────────────────

// ListSkus returns a paginated list of SKUs.
// GET /api/v1/skus?page=1&size=20&search=xxx&product_id=
// @Summary      List SKUs
// @Description  Get paginated list of SKUs
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        page       query  int     false  "Page number"
// @Param        size       query  int     false  "Page size"
// @Param        search     query  string  false  "Search keyword"
// @Param        product_id query  int     false  "Filter by product ID"
// @Success      200  {object}  response.PageResult
// @Security     BearerAuth
// @Router       /skus [get]
func (h *Handler) ListSkus(c *gin.Context) {
	p := common.ParsePagination(c)
	search := c.Query("search")
	productID, _ := strconv.ParseInt(c.Query("product_id"), 10, 64)

	items, total, err := h.service.ListSkus(c.Request.Context(), p.Page, p.Size, search, productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list skus: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// ListSkusByProduct returns all SKUs for a product.
// GET /api/v1/product-master/:id/skus
func (h *Handler) ListSkusByProduct(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	items, err := h.service.ListSkusByProduct(c.Request.Context(), productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list skus: "+err.Error())
		return
	}

	response.Success(c, items)
}

// GetSku returns a single SKU by ID.
// GET /api/v1/skus/:id
func (h *Handler) GetSku(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetSkuByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "sku not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get sku: "+err.Error())
		return
	}

	response.Success(c, item)
}

// CreateSku creates a new SKU.
// POST /api/v1/skus
func (h *Handler) CreateSku(c *gin.Context) {
	var sk Sku
	if err := c.ShouldBindJSON(&sk); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.service.CreateSku(c.Request.Context(), &sk); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create sku: "+err.Error())
		return
	}

	response.Success(c, sk)
}

// UpdateSku updates an existing SKU.
// PUT /api/v1/skus/:id
func (h *Handler) UpdateSku(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var sk Sku
	if err := c.ShouldBindJSON(&sk); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	sk.ID = id

	if err := h.service.UpdateSku(c.Request.Context(), &sk); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update sku: "+err.Error())
		return
	}

	response.Success(c, sk)
}

// DeleteSku deletes a SKU.
// DELETE /api/v1/skus/:id
func (h *Handler) DeleteSku(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteSku(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete sku: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}
