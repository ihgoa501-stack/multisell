package producthub

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles product version HTTP requests.
type Handler struct {
	service    *Service
	version    *VersionService
	freshness  FreshnessService
	relation   *RelationService
	variant    *VariantService
	offer      *SupplierOfferService
	sample     *SampleService
	cost       *CostVersionService
	db         *gorm.DB
}

// NewHandler creates a new producthub handler.
func NewHandler(service *Service, version *VersionService, freshness FreshnessService, relation *RelationService, variant *VariantService, offer *SupplierOfferService, sample *SampleService, cost *CostVersionService, db *gorm.DB) *Handler {
	return &Handler{service: service, version: version, freshness: freshness, relation: relation, variant: variant, offer: offer, sample: sample, cost: cost, db: db}
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

func parseVersionID(c *gin.Context) (int64, bool) {
	idStr := c.Param("versionId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid version id")
		return 0, false
	}
	return id, true
}

// ---------- Version handlers ----------

// ListVersions GET /products/:id/versions
func (h *Handler) ListVersions(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	items, total, err := h.version.ListVersions(productID, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetVersion GET /products/:id/versions/:versionId
func (h *Handler) GetVersion(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	versionID, ok2 := parseVersionID(c)
	if !ok2 {
		return
	}
	v, err := h.version.GetVersion(versionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "version not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if v.ProductID != productID {
		response.Error(c, http.StatusNotFound, "version not found for this product")
		return
	}
	response.Success(c, v)
}

// Rollback POST /products/:id/versions/:versionId/rollback
func (h *Handler) Rollback(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	versionID, ok2 := parseVersionID(c)
	if !ok2 {
		return
	}
	restored, err := h.version.Rollback(productID, versionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "version or product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, restored)
}

// RecordDecision POST /products/:id/decisions
func (h *Handler) RecordDecision(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	var in DecisionRecordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	in.ProductID = productID
	record, err := h.service.RecordDecision(productID, &in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, record)
}

// ---------- Freshness handlers ----------

// GetProductFreshness GET /products/:id/freshness
func (h *Handler) GetProductFreshness(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	summaries, err := h.freshness.GetProductFreshness(c.Request.Context(), productID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, summaries)
}

// ListStaleProducts GET /products/freshness/stale
func (h *Handler) ListStaleProducts(c *gin.Context) {
	records, err := h.freshness.CheckFreshness(c.Request.Context())
	if err != nil {
		response.InternalError(c, err)
		return
	}

	now := time.Now()
	summaries := make([]FreshnessSummary, 0, len(records))
	for _, r := range records {
		label := computeFreshnessLabel(r, now)
		daysSince := int(now.Sub(r.LastVerifiedAt).Hours() / 24)
		summaries = append(summaries, FreshnessSummary{
			DataFreshness:  r,
			FreshnessLabel: label,
			DaysSinceCheck: daysSince,
		})
	}

	response.Success(c, StaleProductsResponse{
		Total:     len(summaries),
		Freshness: summaries,
	})
}

// VerifyDimension POST /products/:id/freshness/verify
func (h *Handler) VerifyDimension(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if err := h.freshness.RecordVerification(c.Request.Context(), productID, req.Dimension, req.CurrentValue); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "验证记录已保存"})
}

// ---------- Relation handlers ----------

// GetRelatedProducts GET /products/:id/relations
func (h *Handler) GetRelatedProducts(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	result, err := h.relation.GetRelatedProducts(productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取关联商品失败: "+err.Error())
		return
	}
	response.Success(c, result)
}

// CreateRelation POST /products/relations
func (h *Handler) CreateRelation(c *gin.Context) {
	var req RelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	r, err := h.relation.CreateRelation(req.SourceID, req.TargetID, req.RelationType, req.Weight)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, r)
}

// DeleteRelation DELETE /products/relations/:id
func (h *Handler) DeleteRelation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid relation id")
		return
	}
	if err := h.relation.DeleteRelation(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "关联关系不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "删除关联关系失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// AutoDiscoverRelations POST /products/:id/discover-relations
func (h *Handler) AutoDiscoverRelations(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	result, err := h.relation.AutoDiscoverRelations(productID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, result)
}

// ---------- Sub-resource handlers (extracted from anonymous inline) ----------

func (h *Handler) ListVariants(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	items, err := h.variant.ListByMaster(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateVariant(c *gin.Context) {
	var v ProductVariant
	if err := c.ShouldBindJSON(&v); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.variant.Create(c.Request.Context(), &v); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, v)
}

func (h *Handler) ListOffers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	items, err := h.offer.ListByMaster(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateOffer(c *gin.Context) {
	var o SupplierOffer
	if err := c.ShouldBindJSON(&o); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.offer.Create(c.Request.Context(), &o); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, o)
}

func (h *Handler) ListSamples(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	items, err := h.sample.ListByMaster(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateSample(c *gin.Context) {
	var sr SampleRequest
	if err := c.ShouldBindJSON(&sr); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.sample.Create(c.Request.Context(), &sr); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, sr)
}

func (h *Handler) ListCosts(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	items, err := h.cost.ListByMaster(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateCost(c *gin.Context) {
	var cv CostVersion
	if err := c.ShouldBindJSON(&cv); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.cost.Create(c.Request.Context(), &cv); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, cv)
}

func (h *Handler) ConfirmCost(c *gin.Context) {
	costID, err := strconv.ParseInt(c.Param("costId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid cost id")
		return
	}
	if err := h.cost.Confirm(c.Request.Context(), costID); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, gin.H{"id": costID, "status": "confirmed"})
}

func (h *Handler) GetProductSummary(c *gin.Context) {
	var total, active, draft int64
	h.db.Model(&ProductMaster{}).Count(&total)
	h.db.Model(&ProductMaster{}).Where("lifecycle_status = ?", "active").Count(&active)
	h.db.Model(&ProductMaster{}).Where("lifecycle_status IN ?",
		[]string{"idea", "researching", "sampling", "approved"},
	).Count(&draft)
	// ponytail: low_stock + expiring_certificates return 0 —
	//  need inventory + compliance module integration for real data
	response.Success(c, gin.H{
		"total_products":        total,
		"active_products":       active,
		"draft_products":        draft,
		"low_stock_products":    0,
		"expiring_certificates": 0,
	})
}

func (h *Handler) ListRecentDecisions(c *gin.Context) {
	type decisionRow struct {
		ID        int64     `json:"id"`
		ProductID int64     `json:"product_id"`
		Action    string    `json:"action"`
		Reason    string    `json:"reason"`
		CreatedAt time.Time `json:"created_at"`
	}
	var rows []decisionRow
	if err := h.db.Table("pre_listing_decision").
		Select("id, sku_id AS product_id, recommendation AS action, reasoning AS reason, created_at").
		Order("created_at DESC").
		Limit(10).
		Find(&rows).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, rows)
}
