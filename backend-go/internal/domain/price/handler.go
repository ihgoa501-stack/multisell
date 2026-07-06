package price

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/response"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Handler handles price HTTP requests.
type Handler struct {
	service     *Service
	approvalSvc *approval.Service
}

// NewHandler creates a new price handler.
func NewHandler(service *Service, approvalSvc *approval.Service) *Handler {
	return &Handler{service: service, approvalSvc: approvalSvc}
}

// ListPrices returns a paginated list of prices.
// GET /api/v1/prices?page=1&size=20&sku_id=&price_type=
func (h *Handler) ListPrices(c *gin.Context) {
	p := common.ParsePagination(c)
	skuID, _ := strconv.ParseInt(c.Query("sku_id"), 10, 64)
	priceType := c.Query("price_type")

	items, total, err := h.service.ListPrices(c.Request.Context(), p.Page, p.Size, skuID, priceType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list prices: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetPrice returns a single price by ID.
// GET /api/v1/prices/:id
func (h *Handler) GetPrice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetPriceByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "price not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get price: "+err.Error())
		return
	}

	response.Success(c, item)
}

// SetPrice creates or updates a price.
// POST /api/v1/prices
func (h *Handler) SetPrice(c *gin.Context) {
	var p Price
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	// ponytail: RequireApproval gates the mutation; caller must submit the approval
	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			ProductID:   p.SkuID,
			RequestType: "price_change",
			Requester:   operator,
			NewValue:    fmt.Sprintf("sku=%d price=%s type=%s", p.SkuID, p.Price.String(), p.PriceType),
			Reason:      "set price requires approval",
			TargetType:  "price",
			TargetID:    p.SkuID,
			RiskLevel:   "high",
			EntityType:  "price",
			EntityID:    p.SkuID,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("price change requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.SetPrice(c.Request.Context(), &p, operator); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to set price: "+err.Error())
		return
	}

	response.Success(c, p)
}

// UpdatePrice updates a price.
// PUT /api/v1/prices/:id
func (h *Handler) UpdatePrice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var p Price
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	p.ID = id

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			ProductID:   p.SkuID,
			RequestType: "price_change",
			Requester:   operator,
			NewValue:    fmt.Sprintf("update price id=%d sku=%d price=%s", id, p.SkuID, p.Price.String()),
			Reason:      "update price requires approval",
			TargetType:  "price",
			TargetID:    id,
			RiskLevel:   "high",
			EntityType:  "price",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("price update requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.UpdatePrice(c.Request.Context(), &p); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update price: "+err.Error())
		return
	}

	response.Success(c, p)
}

// DeletePrice deletes a price.
// DELETE /api/v1/prices/:id
func (h *Handler) DeletePrice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "price_delete",
			Requester:   operator,
			NewValue:    fmt.Sprintf("delete price id=%d", id),
			Reason:      "delete price requires approval",
			TargetType:  "price",
			TargetID:    id,
			RiskLevel:   "high",
			EntityType:  "price",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("price deletion requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.DeletePrice(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete price: "+err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

// ListPricesBySKU returns all prices for a SKU.
// GET /api/v1/skus/:id/prices
func (h *Handler) ListPricesBySKU(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	items, err := h.service.ListPricesBySKU(c.Request.Context(), skuID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list prices: "+err.Error())
		return
	}

	response.Success(c, items)
}

// GetCurrentPrice returns the current sale price for a SKU.
// GET /api/v1/skus/:id/current-price
func (h *Handler) GetCurrentPrice(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetCurrentPrice(c.Request.Context(), skuID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "no active price found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get current price: "+err.Error())
		return
	}

	response.Success(c, item)
}

// PriceHistory returns the price change history for a SKU.
// GET /api/v1/skus/:id/price-history
func (h *Handler) PriceHistory(c *gin.Context) {
	skuID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	p := common.ParsePagination(c)

	items, total, err := h.service.ListChangeLogs(c.Request.Context(), skuID, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get price history: "+err.Error())
		return
	}

	response.Paginated(c, items, total, p.Page, p.Size)
}

// ---------------------------------------------------------------------------
// Competitor Price Handlers
// ---------------------------------------------------------------------------

// CreateCompetitorPrice records a new competitor price observation.
// POST /api/v1/competitor-prices
func (h *Handler) CreateCompetitorPrice(c *gin.Context) {
	var req struct {
		SkuID          int64           `json:"sku_id" binding:"required"`
		Platform       string          `json:"platform"`
		CompetitorName string          `json:"competitor_name" binding:"required"`
		Price          decimal.Decimal `json:"price" binding:"required"`
		Currency       string          `json:"currency"`
		CapturedAt     *time.Time      `json:"captured_at"`
		SourceURL      string          `json:"source_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	cp := &CompetitorPrice{
		SkuID:          req.SkuID,
		Platform:       req.Platform,
		CompetitorName: req.CompetitorName,
		Price:          req.Price,
		Currency:       req.Currency,
		SourceURL:      req.SourceURL,
	}
	if req.CapturedAt != nil {
		cp.CapturedAt = *req.CapturedAt
	}
	if cp.Currency == "" {
		cp.Currency = "USD"
	}

	if err := h.service.CreateCompetitorPrice(c.Request.Context(), cp); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create competitor price: "+err.Error())
		return
	}
	response.Success(c, cp)
}

// ListCompetitorPrices returns paginated competitor prices.
// GET /api/v1/competitor-prices?page=1&size=20&sku_id=
func (h *Handler) ListCompetitorPrices(c *gin.Context) {
	p := common.ParsePagination(c)
	skuID, _ := strconv.ParseInt(c.Query("sku_id"), 10, 64)

	items, total, err := h.service.ListCompetitorPrices(c.Request.Context(), p.Page, p.Size, skuID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list: "+err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetCompetitorPrice returns a single competitor price by ID.
// GET /api/v1/competitor-prices/:id
func (h *Handler) GetCompetitorPrice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetCompetitorPriceByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "competitor price not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get: "+err.Error())
		return
	}
	response.Success(c, item)
}

// DeleteCompetitorPrice deletes a competitor price record.
// DELETE /api/v1/competitor-prices/:id
func (h *Handler) DeleteCompetitorPrice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "competitor_price_delete",
			Requester:   operator,
			NewValue:    fmt.Sprintf("delete competitor price id=%d", id),
			Reason:      "delete competitor price requires approval",
			TargetType:  "competitor_price",
			TargetID:    id,
			RiskLevel:   "medium",
			EntityType:  "competitor_price",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("competitor price deletion requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.DeleteCompetitorPrice(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------------------------------------------------------------------------
// Pricing Strategy Handlers
// ---------------------------------------------------------------------------

// SavePricingStrategy creates or updates a pricing strategy.
// POST /api/v1/pricing-strategies
func (h *Handler) SavePricingStrategy(c *gin.Context) {
	var req struct {
		SkuID        *int64      `json:"sku_id"`
		StrategyType string      `json:"strategy_type" binding:"required"`
		Parameters   interface{} `json:"parameters"`
		Active       *bool       `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	ps := &PricingStrategy{
		SkuID:        req.SkuID,
		StrategyType: req.StrategyType,
		Parameters:   "{}",
	}
	if req.Parameters != nil {
		b, err := json.Marshal(req.Parameters)
		if err == nil {
			ps.Parameters = string(b)
		}
	}
	if req.Active != nil {
		ps.Active = *req.Active
	} else {
		ps.Active = true
	}

	if err := h.service.SavePricingStrategy(c.Request.Context(), ps); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to save strategy: "+err.Error())
		return
	}
	response.Success(c, ps)
}

// UpdatePricingStrategy updates an existing pricing strategy.
// PUT /api/v1/pricing-strategies/:id
func (h *Handler) UpdatePricingStrategy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		SkuID        *int64      `json:"sku_id"`
		StrategyType string      `json:"strategy_type" binding:"required"`
		Parameters   interface{} `json:"parameters"`
		Active       *bool       `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	ps := &PricingStrategy{
		ID:           id,
		SkuID:        req.SkuID,
		StrategyType: req.StrategyType,
		Parameters:   "{}",
	}
	if req.Parameters != nil {
		b, err := json.Marshal(req.Parameters)
		if err == nil {
			ps.Parameters = string(b)
		}
	}
	if req.Active != nil {
		ps.Active = *req.Active
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "pricing_strategy_update",
			Requester:   operator,
			NewValue:    fmt.Sprintf("update pricing strategy id=%d type=%s", id, req.StrategyType),
			Reason:      "update pricing strategy requires approval",
			TargetType:  "pricing_strategy",
			TargetID:    id,
			RiskLevel:   "high",
			EntityType:  "pricing_strategy",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("pricing strategy update requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.SavePricingStrategy(c.Request.Context(), ps); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update strategy: "+err.Error())
		return
	}
	response.Success(c, ps)
}

// GetPricingStrategy returns a pricing strategy by ID.
// GET /api/v1/pricing-strategies/:id
func (h *Handler) GetPricingStrategy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	item, err := h.service.GetPricingStrategyByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "pricing strategy not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get strategy: "+err.Error())
		return
	}
	response.Success(c, item)
}

// ListPricingStrategies returns paginated pricing strategies.
// GET /api/v1/pricing-strategies?page=1&size=20&sku_id=
func (h *Handler) ListPricingStrategies(c *gin.Context) {
	p := common.ParsePagination(c)
	skuID, _ := strconv.ParseInt(c.Query("sku_id"), 10, 64)

	items, total, err := h.service.ListPricingStrategies(c.Request.Context(), p.Page, p.Size, skuID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list strategies: "+err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// DeletePricingStrategy deletes a pricing strategy.
// DELETE /api/v1/pricing-strategies/:id
func (h *Handler) DeletePricingStrategy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "pricing_strategy_delete",
			Requester:   operator,
			NewValue:    fmt.Sprintf("delete pricing strategy id=%d", id),
			Reason:      "delete pricing strategy requires approval",
			TargetType:  "pricing_strategy",
			TargetID:    id,
			RiskLevel:   "high",
			EntityType:  "pricing_strategy",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("pricing strategy deletion requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.DeletePricingStrategy(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete strategy: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------------------------------------------------------------------------
// Pricing Recommendation Handlers
// ---------------------------------------------------------------------------

// GenerateRecommendation generates a pricing recommendation for a SKU.
// POST /api/v1/pricing-recommendations/generate
func (h *Handler) GenerateRecommendation(c *gin.Context) {
	var req GenerateRecommendationInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	rec, err := h.service.GenerateRecommendation(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to generate: "+err.Error())
		return
	}
	response.Success(c, rec)
}

// ListRecommendations returns paginated pricing recommendations.
// GET /api/v1/pricing-recommendations?page=1&size=20&sku_id=
func (h *Handler) ListRecommendations(c *gin.Context) {
	p := common.ParsePagination(c)
	skuID, _ := strconv.ParseInt(c.Query("sku_id"), 10, 64)

	items, total, err := h.service.ListRecommendations(c.Request.Context(), p.Page, p.Size, skuID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list recommendations: "+err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// ApplyRecommendation marks a recommendation as applied.
// POST /api/v1/pricing-recommendations/:id/apply
func (h *Handler) ApplyRecommendation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	if h.approvalSvc != nil {
		apprReq, err := h.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "pricing_recommendation_apply",
			Requester:   operator,
			NewValue:    fmt.Sprintf("apply pricing recommendation id=%d", id),
			Reason:      "applying a pricing recommendation requires approval",
			TargetType:  "pricing_recommendation",
			TargetID:    id,
			RiskLevel:   "high",
			EntityType:  "pricing_recommendation",
			EntityID:    id,
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Error(c, http.StatusForbidden, fmt.Sprintf("applying pricing recommendation requires approval (approval_id=%d)", apprReq.ID))
		return
	}

	if err := h.service.ApplyRecommendation(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to apply: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
