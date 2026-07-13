package purchase

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

func purchaseOwner(c *gin.Context) (int64, bool) {
	v := common.UserIDFromCtx(c)
	if v == nil || *v <= 0 {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return 0, false
	}
	return *v, true
}
func authorityError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, 404, "purchase not found")
	} else if errors.Is(err, ErrAuthorityConflict) {
		response.Error(c, 409, err.Error())
	} else {
		response.Error(c, 422, err.Error())
	}
}
func (h *Handler) CreateAuthority(c *gin.Context) {
	o, ok := purchaseOwner(c)
	if !ok {
		return
	}
	var in CreateAuthorityInput
	if e := c.ShouldBindJSON(&in); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	if k := c.GetHeader("Idempotency-Key"); k != "" {
		in.IdempotencyKey = k
	}
	v, e := h.service.CreateAuthority(c, o, in)
	if e != nil {
		authorityError(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) GetAuthority(c *gin.Context) {
	o, ok := purchaseOwner(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	v, e := h.service.GetAuthority(c, o, id)
	if e != nil {
		authorityError(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) ListAuthorities(c *gin.Context) {
	o, ok := purchaseOwner(c)
	if !ok {
		return
	}
	v, e := h.service.ListAuthorities(c, o)
	if e != nil {
		authorityError(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) ApproveAuthority(c *gin.Context) {
	o, ok := purchaseOwner(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ApproveAuthorityInput
	if e := c.ShouldBindJSON(&in); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	v, e := h.service.ApproveAuthority(c, o, id, in)
	if e != nil {
		authorityError(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) RecordSubmission(c *gin.Context)     { h.recordExternal(c, "submitted") }
func (h *Handler) RecordOrderReceipt(c *gin.Context)   { h.recordExternal(c, "ordered") }
func (h *Handler) RecordFailureReceipt(c *gin.Context) { h.recordExternal(c, "failed") }
func (h *Handler) RecordReceiving(c *gin.Context)      { h.recordExternal(c, "received") }
func (h *Handler) recordExternal(c *gin.Context, typ string) {
	o, ok := purchaseOwner(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ExternalFactInput
	if e := c.ShouldBindJSON(&in); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	v, e := h.service.RecordExternalFact(c, o, id, typ, in)
	if e != nil {
		authorityError(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) LegacyWriteFrozen(c *gin.Context) {
	response.Error(c, http.StatusGone, ErrLegacyWriteFrozen.Error())
}
