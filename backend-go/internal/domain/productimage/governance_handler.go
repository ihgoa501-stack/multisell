package productimage

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

type GovernanceHandler struct{ service *Service }

func NewGovernanceHandler(service *Service) *GovernanceHandler {
	return &GovernanceHandler{service: service}
}

func resourceID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		problem(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid resource id")
		return 0, false
	}
	return id, true
}
func pagination(c *gin.Context) (int, int) { p := common.ParsePagination(c); return p.Page, p.Size }

func (h *GovernanceHandler) CreateRights(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var in RightsGrantInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid rights grant")
		return
	}
	item, err := h.service.CreateRightsGrant(c.Request.Context(), owner, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(201, response.Result{Code: 0, Message: "ok", Data: item})
}
func (h *GovernanceHandler) ListRights(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	page, size := pagination(c)
	items, total, err := h.service.ListRights(c.Request.Context(), owner, c.Query("asset_sha256"), page, size)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Paginated(c, items, total, page, size)
}
func (h *GovernanceHandler) RevokeRights(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := resourceID(c, "grant_id")
	if !ok {
		return
	}
	var in RevokeRightsInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid revocation")
		return
	}
	item, err := h.service.RevokeRightsGrant(c.Request.Context(), owner, id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, item)
}
func (h *GovernanceHandler) CreateReview(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	var in FiveAxisReviewInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid review")
		return
	}
	item, err := h.service.CreateFiveAxisReview(c.Request.Context(), owner, id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(201, response.Result{Code: 0, Message: "ok", Data: item})
}
func (h *GovernanceHandler) ListReviews(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	page, size := pagination(c)
	items, total, err := h.service.ListReviews(c.Request.Context(), owner, id, page, size)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Paginated(c, items, total, page, size)
}
func (h *GovernanceHandler) CreateFeedback(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	var in CandidateFeedbackInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid candidate feedback")
		return
	}
	item, err := h.service.CreateCandidateFeedback(c.Request.Context(), owner, id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(201, response.Result{Code: 0, Message: "ok", Data: item})
}
func (h *GovernanceHandler) RecipeSummary(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	item, err := h.service.RecipeSummary(c.Request.Context(), owner, c.Param("recipe_key"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, item)
}
func (h *GovernanceHandler) CreateCost(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	var in CostEntryInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid cost entry")
		return
	}
	item, err := h.service.CreateCostEntry(c.Request.Context(), owner, id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(201, response.Result{Code: 0, Message: "ok", Data: item})
}
func (h *GovernanceHandler) ListCosts(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := taskID(c)
	if !ok {
		return
	}
	page, size := pagination(c)
	items, total, err := h.service.ListCosts(c.Request.Context(), owner, id, page, size)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Paginated(c, items, total, page, size)
}
