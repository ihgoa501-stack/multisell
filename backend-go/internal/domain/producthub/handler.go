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

// nowFunc is a package-level variable for time.Now, overridable in tests.
var nowFunc = time.Now

// Handler handles product version HTTP requests.
type Handler struct {
	service   *Service
	version   *VersionService
	freshness FreshnessService
	relation  *RelationService
}

// NewHandler creates a new producthub handler.
func NewHandler(service *Service, version *VersionService, freshness FreshnessService, relation *RelationService) *Handler {
	return &Handler{service: service, version: version, freshness: freshness, relation: relation}
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

	now := nowFunc()
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
