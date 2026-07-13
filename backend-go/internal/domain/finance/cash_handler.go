package finance

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

func cashOwner(c *gin.Context) (int64, bool) {
	id := common.UserIDFromCtx(c)
	if id == nil || *id <= 0 {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return 0, false
	}
	return *id, true
}

func cashError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCashValidation):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrCashNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrCashIdempotencyConflict), errors.Is(err, ErrCashObjectConflict):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "cash authority operation failed")
	}
}

func (h *Handler) CreateCashReceipt(c *gin.Context) {
	owner, ok := cashOwner(c)
	if !ok {
		return
	}
	var in CreateCashReceiptInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	row, replay, err := h.service.CreateCashReceipt(c.Request.Context(), owner, in)
	if err != nil {
		cashError(c, err)
		return
	}
	if replay {
		c.Header("Idempotent-Replay", "true")
	}
	response.Success(c, row)
}

func (h *Handler) ListCashReceipts(c *gin.Context) {
	owner, ok := cashOwner(c)
	if !ok {
		return
	}
	rows, err := h.service.ListCashReceipts(c.Request.Context(), owner)
	if err != nil {
		cashError(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *Handler) CreateCashReconciliation(c *gin.Context) {
	owner, ok := cashOwner(c)
	if !ok {
		return
	}
	var in CreateCashReconciliationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	row, replay, err := h.service.CreateCashReconciliation(c.Request.Context(), owner, in)
	if err != nil {
		cashError(c, err)
		return
	}
	if replay {
		c.Header("Idempotent-Replay", "true")
	}
	response.Success(c, row)
}

func (h *Handler) ListCashReconciliations(c *gin.Context) {
	owner, ok := cashOwner(c)
	if !ok {
		return
	}
	rows, err := h.service.ListCashReconciliations(c.Request.Context(), owner)
	if err != nil {
		cashError(c, err)
		return
	}
	response.Success(c, rows)
}
