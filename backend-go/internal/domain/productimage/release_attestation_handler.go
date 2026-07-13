package productimage

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

type ReleaseHandler struct{ service *ReleaseService }

func NewReleaseHandler(service *ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{service: service}
}

type ruleSnapshotRequest struct {
	Channel        string          `json:"channel"`
	Site           string          `json:"site"`
	Locale         string          `json:"locale"`
	CategoryID     int64           `json:"category_id"`
	Rules          json.RawMessage `json:"rules"`
	EffectiveAt    time.Time       `json:"effective_at"`
	ExpiresAt      *time.Time      `json:"expires_at"`
	IdempotencyKey string          `json:"idempotency_key"`
}

func (h *ReleaseHandler) CreateRule(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var req ruleSnapshotRequest
	if c.ShouldBindJSON(&req) != nil {
		problem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid rule snapshot")
		return
	}
	out, err := h.service.CreateRuleSnapshot(c.Request.Context(), owner, CreateRuleSnapshotInput{Channel: req.Channel, Site: req.Site, Locale: req.Locale, CategoryID: req.CategoryID, Rules: req.Rules, EffectiveAt: req.EffectiveAt, ExpiresAt: req.ExpiresAt, IdempotencyKey: req.IdempotencyKey})
	if err != nil {
		releaseProblem(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "ok", Data: out})
}

type setDecisionRequest struct {
	Decision        string `json:"decision"`
	Reason          string `json:"reason"`
	IdempotencyKey  string `json:"idempotency_key"`
	ExpectedVersion uint   `json:"expected_version"`
}

func (h *ReleaseHandler) DecideSet(c *gin.Context) {
	owner, setID, ok := imageSetOwnerAndID(c)
	if !ok {
		return
	}
	var req setDecisionRequest
	if c.ShouldBindJSON(&req) != nil {
		problem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid image-set decision")
		return
	}
	out, err := h.service.DecideImageSet(c.Request.Context(), owner, setID, DecideImageSetInput{Decision: req.Decision, Reason: req.Reason, IdempotencyKey: req.IdempotencyKey, ExpectedVersion: req.ExpectedVersion})
	if err != nil {
		releaseProblem(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "ok", Data: out})
}

type issueAttestationRequest struct {
	ImageSetID        uint64 `json:"image_set_id"`
	RuleSnapshotID    int64  `json:"rule_snapshot_id"`
	PlatformAccountID int64  `json:"platform_account_id"`
	Site              string `json:"site"`
	IdempotencyKey    string `json:"idempotency_key"`
	TTLSeconds        int64  `json:"ttl_seconds"`
}

func (h *ReleaseHandler) Issue(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var req issueAttestationRequest
	if c.ShouldBindJSON(&req) != nil {
		problem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid attestation request")
		return
	}
	out, err := h.service.Issue(c.Request.Context(), owner, IssueAttestationInput{ImageSetID: req.ImageSetID, RuleSnapshotID: req.RuleSnapshotID, PlatformAccountID: req.PlatformAccountID, Site: req.Site, IdempotencyKey: req.IdempotencyKey, TTL: time.Duration(req.TTLSeconds) * time.Second})
	if err != nil {
		releaseProblem(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "ok", Data: out})
}

func (h *ReleaseHandler) Get(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("attestation_id")), 10, 64)
	if err != nil || id <= 0 {
		problem(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid attestation id")
		return
	}
	out, err := h.service.Get(c.Request.Context(), owner, id)
	if err != nil {
		releaseProblem(c, err)
		return
	}
	response.Success(c, out)
}

func releaseProblem(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		problem(c, http.StatusNotFound, "NOT_FOUND", "image release resource not found")
	case errors.Is(err, ErrInvalidInput):
		problem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid image release request")
	case errors.Is(err, ErrReleaseGateBlocked), errors.Is(err, ErrAttestationExpired), errors.Is(err, ErrAttestationConsumed):
		problem(c, http.StatusConflict, "RELEASE_GATE_BLOCKED", err.Error())
	default:
		var conflict *ConflictError
		if errors.As(err, &conflict) {
			problem(c, http.StatusConflict, conflict.Code, "request conflicts with existing immutable record")
			return
		}
		problem(c, http.StatusInternalServerError, "STORE_ERROR", "image release operation failed")
	}
}
