package sourcing1688

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

var captureFailureCodes = map[string]bool{"login_required": true, "captcha_required": true, "network_error": true, "parse_error": true, "access_blocked": true, "source_unavailable": true, "collector_unavailable": true}
var sensitiveFailurePattern = regexp.MustCompile(`(?i)(cookie|token|authorization|password)\s*[:=]\s*[^\s,;]+`)
var htmlFailurePattern = regexp.MustCompile(`<[^>]*>`)

func safeCaptureFailureMessage(raw string) string {
	value := htmlFailurePattern.ReplaceAllString(raw, " ")
	value = sensitiveFailurePattern.ReplaceAllString(value, "$1=[redacted]")
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) > 500 {
		value = string([]rune(value)[:500])
	}
	return value
}

type CaptureAttempt struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	DemandCaseID  int64     `gorm:"not null;index" json:"demand_case_id"`
	ExperimentID  string    `gorm:"size:40;not null;index" json:"experiment_id"`
	SourceURL     string    `gorm:"type:text;not null" json:"source_url"`
	AttemptedAt   time.Time `gorm:"not null" json:"attempted_at"`
	Driver        string    `gorm:"size:40;not null" json:"driver"`
	ParserVersion string    `gorm:"size:40;not null" json:"parser_version"`
	Status        string    `gorm:"size:20;not null" json:"status"`
	ErrorCode     string    `gorm:"size:80;not null" json:"error_code"`
	ErrorMessage  string    `gorm:"type:text;not null" json:"error_message"`
	AttemptedBy   int64     `gorm:"not null" json:"attempted_by"`
	CreatedAt     time.Time `json:"created_at"`
}

func (CaptureAttempt) TableName() string { return "sourcing_1688_capture_attempt" }

type CaptureFailureRecordInput struct {
	DemandCaseID  int64     `json:"demand_case_id" binding:"required"`
	ExperimentID  string    `json:"experiment_id" binding:"required"`
	SourceURL     string    `json:"source_url" binding:"required"`
	AttemptedAt   time.Time `json:"attempted_at" binding:"required"`
	Driver        string    `json:"driver" binding:"required"`
	ParserVersion string    `json:"parser_version" binding:"required"`
	ErrorCode     string    `json:"error_code" binding:"required"`
	ErrorMessage  string    `json:"error_message" binding:"required"`
	AttemptedBy   int64     `json:"attempted_by"`
}

func (s *Service) RecordCaptureFailure(in *CaptureFailureRecordInput) (*CaptureAttempt, error) {
	if in == nil || in.AttemptedBy <= 0 || in.AttemptedAt.IsZero() || strings.TrimSpace(in.Driver) == "" || strings.TrimSpace(in.ParserVersion) == "" || !captureFailureCodes[in.ErrorCode] || strings.TrimSpace(in.ErrorMessage) == "" {
		return nil, ErrInvalidWorkflow
	}
	canonicalURL, err := canonical1688URL(in.SourceURL)
	if err != nil {
		return nil, err
	}
	attempt := CaptureAttempt{DemandCaseID: in.DemandCaseID, ExperimentID: in.ExperimentID, SourceURL: canonicalURL, AttemptedAt: in.AttemptedAt, Driver: in.Driver, ParserVersion: in.ParserVersion, Status: LifecycleCaptureFailed, ErrorCode: in.ErrorCode, ErrorMessage: safeCaptureFailureMessage(in.ErrorMessage), AttemptedBy: in.AttemptedBy}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var dc demandCaseRow
		if err := tx.First(&dc, in.DemandCaseID).Error; err != nil {
			return err
		}
		var exp experimentRow
		if err := tx.Where("experiment_id = ?", in.ExperimentID).First(&exp).Error; err != nil {
			return err
		}
		if dc.Status != "experiment_ready" || dc.OwnerID != in.AttemptedBy || exp.OwnerID != in.AttemptedBy || exp.Status != "active" || (exp.Stage != "product" && exp.Stage != "supply" && exp.Stage != "channel") {
			return fmt.Errorf("%w: capture failure must belong to the approved Owner workflow", ErrWorkflowGate)
		}
		var linkCount, gateCount int64
		if err := tx.Model(&objectLinkRow{}).Where("experiment_id = ? AND object_type = ? AND object_id = ?", in.ExperimentID, "demand_case", strconv.FormatInt(in.DemandCaseID, 10)).Count(&linkCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&gateRow{}).Where("experiment_id = ? AND stage = ? AND result = ?", in.ExperimentID, "opportunity", "pass").Count(&gateCount).Error; err != nil {
			return err
		}
		if linkCount == 0 || gateCount == 0 {
			return fmt.Errorf("%w: capture failure lacks approved market/opportunity linkage", ErrWorkflowGate)
		}
		return tx.Create(&attempt).Error
	})
	return &attempt, err
}

func (s *Service) ListCaptureFailures(experimentID string) ([]CaptureAttempt, error) {
	var items []CaptureAttempt
	err := s.db.Where("experiment_id = ?", experimentID).Order("id DESC").Find(&items).Error
	return items, err
}
