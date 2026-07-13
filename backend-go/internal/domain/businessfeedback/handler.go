package businessfeedback

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

type Handler struct{ svc *Service }

func ownerID(c *gin.Context) (int64, bool) {
	id := common.UserIDFromCtx(c)
	if id == nil || *id <= 0 {
		response.Error(c, http.StatusUnauthorized, "Owner身份无效")
		return 0, false
	}
	return *id, true
}
func pathID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "ID无效")
		return 0, false
	}
	return id, true
}
func writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotAuthorized):
		response.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, http.StatusNotFound, "经营对象不存在")
	default:
		response.Error(c, http.StatusConflict, err.Error())
	}
}

func (h *Handler) CreateAction(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var in CreateActionInput
	if c.ShouldBindJSON(&in) != nil {
		response.Error(c, 400, "请求无效")
		return
	}
	out, err := h.svc.CreateAction(c.Request.Context(), owner, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out)
}
func (h *Handler) List(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	decisionID, _ := strconv.ParseInt(c.Query("owner_decision_id"), 10, 64)
	out, err := h.svc.List(c.Request.Context(), owner, decisionID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out)
}
func (h *Handler) Get(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	out, err := h.svc.Get(c.Request.Context(), owner, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out)
}
func (h *Handler) Execute(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	out, err := h.svc.Execute(c.Request.Context(), owner, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out)
}
func (h *Handler) Observe(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in CreateObservationInput
	if c.ShouldBindJSON(&in) != nil {
		response.Error(c, 400, "请求无效")
		return
	}
	out, err := h.svc.CreateObservation(c.Request.Context(), owner, id, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out)
}
func (h *Handler) Recommend(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in CreateRecommendationInput
	if c.ShouldBindJSON(&in) != nil {
		response.Error(c, 400, "请求无效")
		return
	}
	out, err := h.svc.CreateRecommendation(c.Request.Context(), owner, id, in)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Success(c, out)
}
