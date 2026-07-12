package sourcing1688

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

const controlledFetchTimeout = 30 * time.Second

// ReadOnlyPageFetcher is deliberately narrower than ToolBridge: this workflow
// can only retrieve structured page data and cannot execute mutation tools.
type ReadOnlyPageFetcher interface {
	FetchPage(context.Context, string) (*toolbridge.PageData, error)
}

type ControlledFetchInput struct {
	DemandCaseID int64  `json:"demand_case_id" binding:"required"`
	ExperimentID string `json:"experiment_id" binding:"required"`
	SourceURL    string `json:"source_url" binding:"required"`
}

type ControlledFetchResult struct {
	Product       *Sourcing1688Product `json:"product"`
	Driver        string               `json:"driver"`
	ParserVersion string               `json:"parser_version"`
	EvidenceKind  string               `json:"evidence_kind"`
}

type ControlledFetchHandler struct {
	service *Service
	bridge  ReadOnlyPageFetcher
}

func NewControlledFetchHandler(service *Service, bridge ReadOnlyPageFetcher) *ControlledFetchHandler {
	return &ControlledFetchHandler{service: service, bridge: bridge}
}

// validateControlledFetchGate runs before any external call. It mirrors the
// Capture gate so an unapproved market or mismatched Owner cannot reach a browser.
func (s *Service) validateControlledFetchGate(demandCaseID int64, experimentID string, ownerID int64) error {
	if demandCaseID <= 0 || strings.TrimSpace(experimentID) == "" || ownerID <= 0 {
		return ErrInvalidWorkflow
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var dc demandCaseRow
		if err := tx.First(&dc, demandCaseID).Error; err != nil {
			return err
		}
		var exp experimentRow
		if err := tx.Where("experiment_id = ?", experimentID).First(&exp).Error; err != nil {
			return err
		}
		if dc.Status != "experiment_ready" || dc.OwnerID != ownerID || exp.OwnerID != ownerID || exp.Status != "active" || (exp.Stage != "product" && exp.Stage != "supply" && exp.Stage != "channel") {
			return fmt.Errorf("%w: approved market, active experiment and authenticated Owner must match", ErrWorkflowGate)
		}
		var linkCount, gateCount int64
		if err := tx.Model(&objectLinkRow{}).Where("experiment_id = ? AND object_type = ? AND object_id = ?", experimentID, "demand_case", fmt.Sprint(demandCaseID)).Count(&linkCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&gateRow{}).Where("experiment_id = ? AND stage = ? AND result = ?", experimentID, "opportunity", "pass").Count(&gateCount).Error; err != nil {
			return err
		}
		if linkCount != 1 || gateCount == 0 {
			return fmt.Errorf("%w: approved opportunity linkage required", ErrWorkflowGate)
		}
		return nil
	})
}

func captureErrorCode(err error) string {
	text := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(text, "timeout"):
		return "network_error"
	case strings.Contains(text, "login"):
		return "login_required"
	case strings.Contains(text, "captcha"):
		return "captcha_required"
	case strings.Contains(text, "blocked"), strings.Contains(text, "forbidden"):
		return "access_blocked"
	default:
		return "source_unavailable"
	}
}

func (h *ControlledFetchHandler) Fetch(c *gin.Context) {
	var in ControlledFetchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	ownerID, ok := requireWorkflowActor(c)
	if !ok {
		return
	}
	canonicalURL, err := canonical1688URL(in.SourceURL)
	if err != nil {
		workflowError(c, err)
		return
	}
	if err := h.service.validateControlledFetchGate(in.DemandCaseID, in.ExperimentID, ownerID); err != nil {
		workflowError(c, err)
		return
	}
	if h.bridge == nil {
		_, auditErr := h.service.RecordCaptureFailure(&CaptureFailureRecordInput{DemandCaseID: in.DemandCaseID, ExperimentID: in.ExperimentID, SourceURL: canonicalURL, AttemptedAt: time.Now().UTC(), Driver: "toolbridge", ParserVersion: "unavailable", ErrorCode: "collector_unavailable", ErrorMessage: "read-only 1688 collector is not configured", AttemptedBy: ownerID})
		if auditErr != nil {
			response.InternalError(c, auditErr)
			return
		}
		response.Error(c, http.StatusServiceUnavailable, "read-only 1688 collector is not configured")
		return
	}

	ctx, cancel := context.WithTimeout(toolbridge.WithOwnerUserID(c.Request.Context(), ownerID), controlledFetchTimeout)
	defer cancel()
	page, err := h.bridge.FetchPage(ctx, canonicalURL)
	if err != nil {
		_, auditErr := h.service.RecordCaptureFailure(&CaptureFailureRecordInput{DemandCaseID: in.DemandCaseID, ExperimentID: in.ExperimentID, SourceURL: canonicalURL, AttemptedAt: time.Now().UTC(), Driver: "toolbridge", ParserVersion: "unavailable", ErrorCode: captureErrorCode(err), ErrorMessage: err.Error(), AttemptedBy: ownerID})
		if auditErr != nil {
			response.InternalError(c, fmt.Errorf("capture failed and attempt audit failed: %v: %w", auditErr, err))
			return
		}
		response.Error(c, http.StatusBadGateway, "1688 collection failed; the attempt was recorded")
		return
	}
	if page == nil || strings.TrimSpace(page.Driver) == "" || strings.TrimSpace(page.ParserVersion) == "" || strings.TrimSpace(page.SupplierBusinessID) == "" {
		msg := "collector response lacks driver, parser version, or supplier identity"
		_, auditErr := h.service.RecordCaptureFailure(&CaptureFailureRecordInput{DemandCaseID: in.DemandCaseID, ExperimentID: in.ExperimentID, SourceURL: canonicalURL, AttemptedAt: time.Now().UTC(), Driver: "toolbridge", ParserVersion: "unavailable", ErrorCode: "parse_error", ErrorMessage: msg, AttemptedBy: ownerID})
		if auditErr != nil {
			response.InternalError(c, auditErr)
			return
		}
		response.Error(c, http.StatusUnprocessableEntity, msg)
		return
	}
	returnedURL, urlErr := canonical1688URL(page.SourceURL)
	now := time.Now().UTC()
	if urlErr != nil || returnedURL != canonicalURL || page.PriceCNY <= 0 || page.MOQ <= 0 || strings.TrimSpace(page.SupplierName) == "" || (!page.CollectedAt.IsZero() && (page.CollectedAt.After(now.Add(5*time.Minute)) || page.CollectedAt.Before(now.Add(-24*time.Hour)))) {
		msg := "collector response URL, observation time, price, MOQ, or supplier identity is invalid"
		_, auditErr := h.service.RecordCaptureFailure(&CaptureFailureRecordInput{DemandCaseID: in.DemandCaseID, ExperimentID: in.ExperimentID, SourceURL: canonicalURL, AttemptedAt: now, Driver: page.Driver, ParserVersion: page.ParserVersion, ErrorCode: "parse_error", ErrorMessage: msg, AttemptedBy: ownerID})
		if auditErr != nil {
			response.InternalError(c, auditErr)
			return
		}
		response.Error(c, http.StatusUnprocessableEntity, msg)
		return
	}
	// Serialize the structured object actually returned by ToolBridge. We do not
	// invent HTML or claim access to bytes the collector did not provide.
	raw, err := json.Marshal(page)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	collectedAt := page.CollectedAt
	if collectedAt.IsZero() {
		collectedAt = time.Now().UTC()
	}
	images, _ := json.Marshal(page.Images)
	variants, _ := json.Marshal(page.SpecVariants)
	title, price, moq := page.Title, page.PriceCNY, page.MOQ
	product, err := h.service.Capture(&CaptureInput{DemandCaseID: in.DemandCaseID, ExperimentID: in.ExperimentID, SourceURL: canonicalURL, CollectedAt: collectedAt, CollectedBy: ownerID, Driver: page.Driver, ParserVersion: page.ParserVersion, RawPayload: raw, Title: &title, Price: &price, MOQ: &moq, SupplierName: page.SupplierName, SupplierBusinessID: page.SupplierBusinessID, Images: images, SkuVariants: variants})
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, ControlledFetchResult{Product: product, Driver: page.Driver, ParserVersion: page.ParserVersion, EvidenceKind: "structured_collector_response"})
}
