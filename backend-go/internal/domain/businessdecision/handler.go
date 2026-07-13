package businessdecision

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

type Handler struct{ s *Service }

func owner(c *gin.Context) (int64, bool) {
	v := common.UserIDFromCtx(c)
	if v == nil || *v <= 0 {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return 0, false
	}
	return *v, true
}
func cid(c *gin.Context) (int64, bool) {
	v, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || v <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return v, true
}
func fail(c *gin.Context, e error) {
	if errors.Is(e, gorm.ErrRecordNotFound) {
		response.Error(c, http.StatusNotFound, "resource not found")
	} else if errors.Is(e, ErrConflict) {
		response.Error(c, http.StatusConflict, e.Error())
	} else {
		response.Error(c, http.StatusUnprocessableEntity, e.Error())
	}
}
func (h *Handler) Create(c *gin.Context) {
	o, ok := owner(c)
	if !ok {
		return
	}
	var in CreateCaseInput
	if e := c.ShouldBindJSON(&in); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	if k := c.GetHeader("Idempotency-Key"); k != "" {
		in.IdempotencyKey = k
	}
	v, e := h.s.CreateCase(c, o, in)
	if e != nil {
		fail(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) Get(c *gin.Context) {
	o, ok := owner(c)
	if !ok {
		return
	}
	id, ok := cid(c)
	if !ok {
		return
	}
	v, e := h.s.Get(c, o, id)
	if e != nil {
		fail(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) List(c *gin.Context) {
	o, ok := owner(c)
	if !ok {
		return
	}
	v, e := h.s.List(c, o)
	if e != nil {
		fail(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) FactOptions(c *gin.Context) {
	o, ok := owner(c)
	if !ok {
		return
	}
	v, e := h.s.FactOptions(c, o, c.Query("object_type"))
	if e != nil {
		fail(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) Recommend(c *gin.Context) {
	o, ok := owner(c)
	if !ok {
		return
	}
	id, ok := cid(c)
	if !ok {
		return
	}
	var in RecommendInput
	if e := c.ShouldBindJSON(&in); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	if k := c.GetHeader("Idempotency-Key"); k != "" {
		in.IdempotencyKey = k
	}
	v, e := h.s.Recommend(c, o, id, in)
	if e != nil {
		fail(c, e)
		return
	}
	response.Success(c, v)
}
func (h *Handler) Decide(c *gin.Context) {
	o, ok := owner(c)
	if !ok {
		return
	}
	id, ok := cid(c)
	if !ok {
		return
	}
	var in DecideInput
	if e := c.ShouldBindJSON(&in); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	if k := c.GetHeader("Idempotency-Key"); k != "" {
		in.IdempotencyKey = k
	}
	v, e := h.s.Decide(c, o, id, in)
	if e != nil {
		fail(c, e)
		return
	}
	response.Success(c, v)
}
