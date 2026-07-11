package experiment

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func ownerID(c *gin.Context) (int64, bool) {
	id := common.UserIDFromCtx(c)
	if id == nil || *id <= 0 {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return 0, false
	}
	return *id, true
}
func (h *Handler) List(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	x, n, e := h.service.List(c.Request.Context(), owner, p.Page, p.Size)
	if e != nil {
		response.Error(c, 500, e.Error())
		return
	}
	response.Paginated(c, x, n, p.Page, p.Size)
}
func (h *Handler) Get(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	x, e := h.service.GetDetail(c.Request.Context(), c.Param("experimentId"), owner)
	if errors.Is(e, gorm.ErrRecordNotFound) {
		response.Error(c, 404, "experiment not found")
		return
	}
	if e != nil {
		response.Error(c, 500, e.Error())
		return
	}
	response.Success(c, x)
}
func (h *Handler) Create(c *gin.Context) {
	var x ExperimentCase
	if e := c.ShouldBindJSON(&x); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	x.OwnerID = owner
	if e := h.service.Create(c.Request.Context(), &x); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	response.Success(c, x)
}
func (h *Handler) Update(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var x ExperimentCase
	if e := c.ShouldBindJSON(&x); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	if e := h.service.Update(c.Request.Context(), c.Param("experimentId"), owner, &x); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	response.Success(c, x)
}
func (h *Handler) Delete(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id := c.Param("experimentId")
	if e := h.service.Delete(c.Request.Context(), id, owner); e != nil {
		response.Error(c, 500, e.Error())
		return
	}
	response.Success(c, gin.H{"experiment_id": id})
}
func (h *Handler) AddEvidence(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var x EvidenceRecord
	if e := c.ShouldBindJSON(&x); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	x.ExperimentID = c.Param("experimentId")
	if e := h.service.AddEvidence(c.Request.Context(), owner, &x); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	response.Success(c, x)
}
func (h *Handler) VerifyEvidence(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	evidenceID, err := strconv.ParseInt(c.Param("evidenceId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid evidence id")
		return
	}
	x, err := h.service.VerifyEvidence(c.Request.Context(), c.Param("experimentId"), evidenceID, owner)
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, x)
}
func (h *Handler) AddObjectLink(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var x ObjectLink
	if e := c.ShouldBindJSON(&x); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	x.ExperimentID = c.Param("experimentId")
	if e := h.service.AddObjectLink(c.Request.Context(), owner, &x); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	response.Success(c, x)
}
func (h *Handler) EvaluateGate(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var x GateInput
	if e := c.ShouldBindJSON(&x); e != nil {
		response.Error(c, 400, e.Error())
		return
	}
	x.DecidedBy = owner
	g, e := h.service.EvaluateGate(c.Request.Context(), c.Param("experimentId"), owner, x)
	if e != nil {
		response.Error(c, http.StatusUnprocessableEntity, e.Error())
		return
	}
	response.Success(c, g)
}
func (h *Handler) OwnerSummary(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	x, e := h.service.OwnerSummary(c.Request.Context(), c.Param("experimentId"), owner)
	if e != nil {
		response.Error(c, 500, e.Error())
		return
	}
	response.Success(c, x)
}
