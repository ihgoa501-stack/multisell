package demandcase

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func demandOwnerID(c *gin.Context) (int64, bool) {
	id := common.UserIDFromCtx(c)
	if id == nil || *id <= 0 {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return 0, false
	}
	return *id, true
}

func demandCaseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid demand case id")
		return 0, false
	}
	return id, true
}

func (h *Handler) Create(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	var input DemandCase
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	input.OwnerID = owner
	if err := h.service.Create(c.Request.Context(), &input); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, input)
}

func (h *Handler) List(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	rows, total, err := h.service.List(c.Request.Context(), owner, p.Page, p.Size)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Paginated(c, rows, total, p.Page, p.Size)
}

func (h *Handler) Get(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	d, err := h.service.Get(c.Request.Context(), id, owner)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, http.StatusNotFound, "demand case not found")
		return
	}
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, d)
}

func (h *Handler) AddEvidence(c *gin.Context) {
	h.addEvidence(c, "")
}

func (h *Handler) AddFalsification(c *gin.Context) {
	h.addEvidence(c, EvidenceCounter)
}

func (h *Handler) addEvidence(c *gin.Context, forcedKind string) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	var input DemandEvidence
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	input.DemandCaseID = id
	if forcedKind != "" {
		input.Kind = forcedKind
	}
	if err := h.service.AddEvidence(c.Request.Context(), owner, &input); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, input)
}

func (h *Handler) Evaluate(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	v, err := h.service.Evaluate(c.Request.Context(), id, owner)
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, v)
}

func (h *Handler) DecisionCard(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	card, err := h.service.DecisionCard(c.Request.Context(), id, owner)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, http.StatusNotFound, "demand case not found")
		return
	}
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, card)
}

func (h *Handler) ImportResearch(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	var input ResearchResult
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.ImportResearchResult(c.Request.Context(), owner, input)
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *Handler) RecordDataAccess(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	var input DataAccessRecord
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	input.DemandCaseID = id
	if err := h.service.RecordDataAccess(c.Request.Context(), owner, &input); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, input)
}
func (h *Handler) PermissionRequests(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	items, err := h.service.PermissionRequests(c.Request.Context(), id, owner)
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, items)
}
func (h *Handler) ImportReviewedBatch(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	out, err := h.service.ImportReviewedMarketPermissionBatch(c.Request.Context(), owner)
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, out)
}
