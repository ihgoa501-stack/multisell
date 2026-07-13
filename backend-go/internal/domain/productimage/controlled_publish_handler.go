package productimage

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

type PublishHandler struct{ service *PublishService }

func NewPublishHandler(service *PublishService) *PublishHandler {
	return &PublishHandler{service: service}
}

type executePublishRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required,max=100"`
}

func (h *PublishHandler) Execute(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := positivePathID(c, "attestation_id")
	if !ok {
		return
	}
	var req executePublishRequest
	if c.ShouldBindJSON(&req) != nil {
		problem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "idempotency_key is required")
		return
	}
	out, err := h.service.Execute(c.Request.Context(), owner, id, req.IdempotencyKey)
	if err != nil {
		publishProblem(c, out, err)
		return
	}
	c.JSON(http.StatusOK, response.Result{Code: 0, Message: "ok", Data: out})
}

func (h *PublishHandler) Get(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := positivePathID(c, "attempt_id")
	if !ok {
		return
	}
	out, err := h.service.Get(c.Request.Context(), owner, id)
	if err != nil {
		publishProblem(c, nil, err)
		return
	}
	response.Success(c, out)
}

func (h *PublishHandler) Reconcile(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := positivePathID(c, "attempt_id")
	if !ok {
		return
	}
	out, err := h.service.Reconcile(c.Request.Context(), owner, id)
	if err != nil {
		publishProblem(c, out, err)
		return
	}
	response.Success(c, out)
}

func positivePathID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || id <= 0 {
		problem(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid "+name)
		return 0, false
	}
	return id, true
}

func publishProblem(c *gin.Context, attempt *ImagePublishAttempt, err error) {
	data := any(nil)
	if attempt != nil {
		data = attempt
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		problem(c, http.StatusNotFound, "NOT_FOUND", "publication resource not found")
	case errors.Is(err, ErrUnsupportedPublisher):
		c.JSON(http.StatusConflict, response.Result{Code: 409, Message: "CONTROLLED_PUBLISHER_UNSUPPORTED: URL-based legacy adapters are not eligible", Data: data})
	case errors.Is(err, ErrReconcileRequired):
		c.JSON(http.StatusConflict, response.Result{Code: 409, Message: "RECONCILE_REQUIRED: external outcome is unknown; no automatic retry is allowed", Data: data})
	case errors.Is(err, ErrPublishInProgress), errors.Is(err, ErrAttestationConsumed), errors.Is(err, ErrAttestationExpired), errors.Is(err, ErrReleaseGateBlocked):
		c.JSON(http.StatusConflict, response.Result{Code: 409, Message: err.Error(), Data: data})
	case errors.Is(err, ErrInvalidInput):
		problem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid controlled publication request")
	default:
		var conflict *ConflictError
		if errors.As(err, &conflict) {
			problem(c, http.StatusConflict, conflict.Code, "publication request conflicts with immutable attempt")
			return
		}
		problem(c, http.StatusInternalServerError, "STORE_ERROR", "controlled publication failed")
	}
}
