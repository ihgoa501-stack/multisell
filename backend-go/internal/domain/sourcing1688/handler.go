package sourcing1688

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles sourcing1688 HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new sourcing1688 handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func parseOptionalInt64(c *gin.Context, key string) *int64 {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

// List GET /sourcing1688
func (h *Handler) List(c *gin.Context) {
	ownerID, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	f := &ListFilter{
		Search:    c.Query("search"),
		Status:    c.Query("status"),
		ProductID: parseOptionalInt64(c, "product_id"),
	}
	items, total, err := h.service.ListOwned(ownerID, &p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /sourcing1688/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, ok := h.requireSourceOwner(c, id); !ok {
		return
	}
	p, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Create POST /sourcing1688
func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.Create(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Update PUT /sourcing1688/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.Update(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Delete DELETE /sourcing1688/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// Import POST /sourcing1688/:id/import
func (h *Handler) Import(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ImportInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.Import(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Reject POST /sourcing1688/:id/reject
func (h *Handler) Reject(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in RejectInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.Reject(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sourcing1688 product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Summary GET /sourcing1688/summary
func (h *Handler) Summary(c *gin.Context) {
	ownerID, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	sum, err := h.service.SummaryOwned(ownerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sum)
}

func workflowError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidWorkflow):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrWorkflowGate):
		response.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidLifecycleTransition):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		response.InternalError(c, err)
	}
}

func requireWorkflowActor(c *gin.Context) (int64, bool) {
	actor := common.UserIDFromCtx(c)
	if actor == nil || *actor <= 0 {
		response.Error(c, http.StatusUnauthorized, "authenticated Owner identity required")
		return 0, false
	}
	return *actor, true
}

func (h *Handler) requireSourceOwner(c *gin.Context, sourceID int64) (int64, bool) {
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return 0, false
	}
	if err := h.service.RequireSourceOwner(sourceID, actor); err != nil {
		workflowError(c, err)
		return 0, false
	}
	return actor, true
}

func (h *Handler) Lifecycle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, ok := h.requireSourceOwner(c, id); !ok {
		return
	}
	state, err := h.service.GetLifecycle(id)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, state)
}

func (h *Handler) CaptureFailed(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in CaptureFailureInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.ActorID = actor
	state, err := h.service.MarkCaptureFailed(id, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, state)
}

func (h *Handler) ReviewDecision(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in SourceReviewDecisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.OwnerID = actor
	state, err := h.service.DecideSourceReview(id, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, state)
}

func (h *Handler) SubmitDraftApproval(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in DraftApprovalSubmissionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.RequesterID = actor
	result, err := h.service.SubmitDraftApproval(id, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) DecideDraftApproval(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	approvalID, err := strconv.ParseInt(c.Param("approvalId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid approval id")
		return
	}
	var in DraftApprovalDecisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.OwnerID = actor
	result, err := h.service.DecideDraftApproval(id, approvalID, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, result)
}

func parseAttemptID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("attemptId"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid publish attempt id")
		return 0, false
	}
	return id, true
}

// RequestPublish creates the independent, high-risk Owner approval. It does
// not call the platform adapter.
func (h *Handler) RequestPublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in PublishRequestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := h.requireSourceOwner(c, id)
	if !ok {
		return
	}
	in.RequesterID = actor
	attempt, err := h.service.RequestPublish(id, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, attempt)
}

func (h *Handler) DecidePublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	attemptID, ok := parseAttemptID(c)
	if !ok {
		return
	}
	var in PublishDecisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := h.requireSourceOwner(c, id)
	if !ok {
		return
	}
	in.OwnerID = actor
	attempt, err := h.service.DecidePublish(id, attemptID, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, attempt)
}

func (h *Handler) ExecutePublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	attemptID, ok := parseAttemptID(c)
	if !ok {
		return
	}
	actor, ok := h.requireSourceOwner(c, id)
	if !ok {
		return
	}
	attempt, err := h.service.ExecutePublish(c.Request.Context(), id, attemptID, actor)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, attempt)
}

func (h *Handler) ListPublishRequests(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, ok := h.requireSourceOwner(c, id); !ok {
		return
	}
	items, err := h.service.ListPublishAttempts(id)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) ReconcilePublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	attemptID, ok := parseAttemptID(c)
	if !ok {
		return
	}
	var in PublishReconcileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := h.requireSourceOwner(c, id)
	if !ok {
		return
	}
	in.OwnerID = actor
	attempt, err := h.service.ReconcilePublish(c.Request.Context(), id, attemptID, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, attempt)
}

func (h *Handler) Snapshot(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, ok := h.requireSourceOwner(c, id); !ok {
		return
	}
	snapshot, err := h.service.GetSnapshot(id)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, snapshot)
}

func (h *Handler) Draft(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, ok := h.requireSourceOwner(c, id); !ok {
		return
	}
	draft, err := h.service.GetDraft(id)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, draft)
}

func (h *Handler) IdentityHistory(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if _, ok := h.requireSourceOwner(c, id); !ok {
		return
	}
	history, err := h.service.GetIdentityHistory(id)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, history)
}

func (h *Handler) ResolveDuplicate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ResolveDuplicateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.ReviewedBy = actor
	result, err := h.service.ResolveDuplicate(id, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ProcessImage(c *gin.Context) {
	var in ProcessImageInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.ProcessedBy = actor
	result, err := h.service.ProcessImage(&in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ProcessedImageContent(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	record, err := h.service.GetProcessedImage(id)
	if err != nil {
		workflowError(c, err)
		return
	}
	if _, ok := h.requireSourceOwner(c, record.SourcingProductID); !ok {
		return
	}
	contentType := "image/jpeg"
	if record.OutputFormat == "png" {
		contentType = "image/png"
	}
	c.Data(http.StatusOK, contentType, record.ProcessedBytes)
}

func (h *Handler) RecordCaptureFailure(c *gin.Context) {
	var in CaptureFailureRecordInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.AttemptedBy = actor
	result, err := h.service.RecordCaptureFailure(&in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ListCaptureFailures(c *gin.Context) {
	experimentID := c.Query("experiment_id")
	if experimentID == "" {
		response.Error(c, http.StatusBadRequest, "experiment_id required")
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	if err := h.service.RequireExperimentOwner(experimentID, actor); err != nil {
		workflowError(c, err)
		return
	}
	items, err := h.service.ListCaptureFailures(experimentID)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, items)
}

// Capture POST /sourcing-1688/capture stores immutable 1688 evidence.
func (h *Handler) Capture(c *gin.Context) {
	var in CaptureInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.CollectedBy = actor
	p, err := h.service.Capture(&in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, p)
}

// Review POST /sourcing-1688/:id/review is the explicit Owner gate.
func (h *Handler) Review(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ReviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.ReviewedBy = actor
	decision := SourceReviewDecisionInput{OwnerID: in.ReviewedBy, Action: "approve", Notes: in.Notes}
	p, err := h.service.DecideSourceReview(id, &decision)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, p)
}

// ConvertToDraft POST /sourcing-1688/:id/convert-to-draft creates no external side effect.
func (h *Handler) ConvertToDraft(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ConvertInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.CreatedBy = actor
	r, err := h.service.Convert(id, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, r)
}

func (h *Handler) UpdateDraft(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in ConvertInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	in.CreatedBy = actor
	result, err := h.service.UpdateDraft(id, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, result)
}
