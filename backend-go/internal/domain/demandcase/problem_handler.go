package demandcase

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

func problemID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid problem id")
		return 0, false
	}
	return id, true
}
func (h *Handler) ListProblems(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	rows, err := h.service.ListProblems(c.Request.Context(), owner)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, rows)
}
func (h *Handler) GetProblem(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := problemID(c)
	if !ok {
		return
	}
	d, err := h.service.GetProblem(c.Request.Context(), id, owner)
	if err != nil {
		response.Error(c, http.StatusNotFound, "problem case not found")
		return
	}
	response.Success(c, d)
}
func (h *Handler) CreateProblem(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	var p ProblemCase
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p.OwnerID = owner
	if err := h.service.CreateProblem(c.Request.Context(), &p); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, p)
}
func (h *Handler) AddProblemEvidence(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := problemID(c)
	if !ok {
		return
	}
	var in struct {
		Kind       string    `json:"kind"`
		Title      string    `json:"title"`
		SourceURI  string    `json:"source_uri"`
		ObservedAt time.Time `json:"observed_at"`
		Collector  string    `json:"collector"`
		RawSHA256  string    `json:"raw_sha256"`
		RawPayload string    `json:"raw_payload"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	e := ProblemEvidence{ProblemCaseID: id, Kind: in.Kind, Title: in.Title, SourceURI: in.SourceURI, ObservedAt: in.ObservedAt, Collector: in.Collector, RawSHA256: in.RawSHA256, RawPayload: in.RawPayload}
	if err := h.service.AddProblemEvidence(c.Request.Context(), owner, &e); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, e)
}
func (h *Handler) EvaluateProblem(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := problemID(c)
	if !ok {
		return
	}
	status, err := h.service.EvaluateProblem(c.Request.Context(), id, owner)
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, gin.H{"status": status})
}
func (h *Handler) PromoteProblem(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	id, ok := problemID(c)
	if !ok {
		return
	}
	var in struct {
		SalesChannel string `json:"sales_channel"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	d, err := h.service.PromoteProblem(c.Request.Context(), id, owner, in.SalesChannel)
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, d)
}

func (h *Handler) ImportReviewedProblemBatch(c *gin.Context) {
	owner, ok := demandOwnerID(c)
	if !ok {
		return
	}
	out, err := h.service.ImportReviewedProblemBatch(c.Request.Context(), owner)
	if err != nil {
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Success(c, out)
}
