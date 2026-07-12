package demandcase

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

func decisionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, http.StatusNotFound, "resource not found")
	case errors.Is(err, ErrDecisionConflict), errors.Is(err, ErrMarketNotSelected), errors.Is(err, ErrOpportunityNotReady):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
	}
}

func (h *Handler) DecideMarket(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	var in MarketDecisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if key := c.GetHeader("Idempotency-Key"); key != "" {
		in.IdempotencyKey = key
	}
	row, err := h.service.DecideMarket(c.Request.Context(), id, owner, in)
	if err != nil {
		decisionError(c, err)
		return
	}
	response.Success(c, row)
}

func (h *Handler) LatestMarketDecision(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	row, err := h.service.LatestMarketDecision(c.Request.Context(), id, owner)
	if err != nil {
		decisionError(c, err)
		return
	}
	response.Success(c, row)
}

func (h *Handler) CreateProductOpportunity(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	var in ProductOpportunityInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	row, err := h.service.CreateProductOpportunity(c.Request.Context(), owner, in)
	if err != nil {
		decisionError(c, err)
		return
	}
	response.Success(c, row)
}

func (h *Handler) ListProductOpportunities(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	rows, err := h.service.ListProductOpportunities(c.Request.Context(), owner)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *Handler) GetProductOpportunity(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	row, err := h.service.GetProductOpportunity(c.Request.Context(), id, owner)
	if err != nil {
		decisionError(c, err)
		return
	}
	response.Success(c, row)
}

func (h *Handler) EvaluateProductOpportunity(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	row, blockers, err := h.service.EvaluateProductOpportunity(c.Request.Context(), id, owner)
	if err != nil {
		decisionError(c, err)
		return
	}
	response.Success(c, gin.H{"opportunity": row, "blockers": blockers})
}

func (h *Handler) DecideProductOpportunity(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := demandCaseID(c)
	if !ok {
		return
	}
	var in OpportunityDecisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if key := c.GetHeader("Idempotency-Key"); key != "" {
		in.IdempotencyKey = key
	}
	row, err := h.service.DecideProductOpportunity(c.Request.Context(), id, owner, in)
	if err != nil {
		decisionError(c, err)
		return
	}
	response.Success(c, row)
}
