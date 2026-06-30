package compliance

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles compliance HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new compliance handler.
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

// checkRequest is the JSON body for POST /compliance/check.
type checkRequest struct {
	ProductID   int64  `json:"product_id" binding:"required"`
	PlatformID  *int64 `json:"platform_id,omitempty"`
	ProductName string `json:"product_name" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Country     string `json:"country" binding:"required"`
	Platform    string `json:"platform" binding:"required"`
}

// Check handles POST /api/v1/compliance/check
func (h *Handler) Check(c *gin.Context) {
	var req checkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	input := &CheckInput{
		ProductName: req.ProductName,
		Category:    req.Category,
		Country:     req.Country,
		Platform:    req.Platform,
	}
	result, err := h.service.CheckProduct(input, req.PlatformID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	// Patch the saved result with the actual product_id/platform_id (CheckProduct
	// saves with default ProductID=0 since the handler owns the mapping).
	updates := map[string]interface{}{"product_id": req.ProductID}
	if req.PlatformID != nil {
		updates["platform_id"] = *req.PlatformID
	}
	if err := h.service.db.Model(result).Updates(updates).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	result.ProductID = req.ProductID
	if req.PlatformID != nil {
		result.PlatformID = req.PlatformID
	}

	// Update data_freshness for compliance dimension.
	_ = h.service.db.Exec(
		`UPDATE data_freshness SET last_verified_at = NOW(), status = ?
		 WHERE dimension = 'compliance' AND product_id = ?`,
		result.Status, req.ProductID,
	)

	response.Success(c, result)
}

// scanRequest is the JSON body for POST /compliance/scan
type scanRequest struct {
	ProductIDs []int64 `json:"product_ids,omitempty"`
	PlatformID *int64  `json:"platform_id,omitempty"`
}

// Scan handles POST /api/v1/compliance/scan — triggers batch scan.
func (h *Handler) Scan(c *gin.Context) {
	var req scanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = req // product-ids filtering deferred to Task 6

	scanner := NewScanner(h.service.db, h.service.logger)
	ctx := context.Background()
	scanResult, err := scanner.ScanPaginated(ctx)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.Success(c, scanResult)
}

// ListResults handles GET /api/v1/compliance/results
func (h *Handler) ListResults(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("status")
	riskLevel := c.Query("risk_level")

	productIDStr := c.Query("product_id")
	var productID int64
	if productIDStr != "" {
		productID, _ = strconv.ParseInt(productIDStr, 10, 64)
	}

	items, total, err := h.service.ListResults(&p, status, riskLevel, productID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetResult handles GET /api/v1/compliance/results/:id
func (h *Handler) GetResult(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var result CheckResult
	if err := h.service.db.First(&result, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "result not found")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, result)
}

// suppressRequest is the JSON body for PUT /compliance/results/:id/suppress
type suppressRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// SuppressResult handles PUT /api/v1/compliance/results/:id/suppress
func (h *Handler) SuppressResult(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req suppressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.SuppressResult(id, req.Reason); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}
