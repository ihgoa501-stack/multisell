package compliance

import (
	"errors"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides compliance domain business logic.
type Service struct {
	db              *gorm.DB
	logger          *zap.Logger
	adapter         *A7Adapter
	freshnessWriter *FreshnessWriter
}

// NewService creates a new compliance Service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{
		db:              db,
		logger:          logger.Named("compliance"),
		adapter:         NewA7Adapter(),
		freshnessWriter: NewFreshnessWriter(db),
	}
}

// CheckProduct runs a compliance check for a given product via the A7 agent
// and persists the result.
func (s *Service) CheckProduct(input *CheckInput, platformID *int64) (*CheckResult, error) {
	output, confidence, err := s.adapter.RunCheck(input)
	if err != nil {
		return nil, fmt.Errorf("run compliance check: %w", err)
	}

	now := time.Now()
	riskLevel := output.RiskLevel
	if riskLevel == "" {
		riskLevel = RiskUnknown
	}

	status := StatusPass
	if riskLevel == RiskHigh || riskLevel == RiskUnknown {
		status = StatusFail
	} else if riskLevel == RiskMedium {
		status = StatusWarning
	}

	evidence := make([]EvidenceItem, 0, 4)
	evidence = append(evidence, EvidenceItem{
		Rule:      "risk_level",
		Field:     "risk_level",
		Value:     riskLevel,
		Source:    "A7ComplianceGuard",
		Timestamp: now.Format(time.RFC3339),
	})
	if len(output.Certifications) > 0 {
		certs := ""
		for i, c := range output.Certifications {
			if i > 0 {
				certs += ", "
			}
			certs += c
		}
		evidence = append(evidence, EvidenceItem{
			Rule:   "certifications",
			Field:  "required_certifications",
			Value:  certs,
			Source: "A7ComplianceGuard",
		})
	}
	if len(output.Restrictions) > 0 {
		rest := ""
		for i, r := range output.Restrictions {
			if i > 0 {
				rest += ", "
			}
			rest += r
		}
		evidence = append(evidence, EvidenceItem{
			Rule:   "restrictions",
			Field:  "restrictions",
			Value:  rest,
			Source: "A7ComplianceGuard",
		})
	}
	if len(output.BlockedPlatforms) > 0 {
		blocked := ""
		for i, p := range output.BlockedPlatforms {
			if i > 0 {
				blocked += ", "
			}
			blocked += p
		}
		evidence = append(evidence, EvidenceItem{
			Rule:   "blocked_platforms",
			Field:  "blocked_platforms",
			Value:  blocked,
			Source: "A7ComplianceGuard",
		})
	}

	// Include confidence as custom evidence field
	evidenceRule := "confidence"
	if confidence < 0.7 {
		evidenceRule = "low_confidence"
	}
	evidence = append(evidence, EvidenceItem{
		Rule:      evidenceRule,
		Field:     "confidence",
		Value:     fmt.Sprintf("%.2f", confidence),
		Source:    "A7ComplianceGuard",
		Timestamp: now.Format(time.RFC3339),
	})

	result := &CheckResult{
		ProductID:  0, // caller should update if they have the product ID
		PlatformID: platformID,
		CheckType:  "compliance",
		Status:     status,
		RiskLevel:  &riskLevel,
		Evidence:   evidence,
		ScannedAt:  now,
	}

	if err := s.SaveResult(result); err != nil {
		return nil, fmt.Errorf("save compliance result: %w", err)
	}

	s.logger.Info("compliance check completed",
		zap.String("product_name", input.ProductName),
		zap.String("risk_level", riskLevel),
		zap.String("status", status),
		zap.Float64("confidence", confidence),
	)

	return result, nil
}

// SaveResult persists a compliance check result with idempotency.
// Deletes any existing result for the same (product_id, platform_id, check_type, scanned_at::date)
// before creating, ensuring one result per product per platform per day.
func (s *Service) SaveResult(result *CheckResult) error {
	platformVal := 0
	if result.PlatformID != nil {
		platformVal = int(*result.PlatformID)
	}
	err := s.db.Exec(
		`DELETE FROM compliance_check_result
		 WHERE product_id = ? AND COALESCE(platform_id, 0) = ?
		   AND check_type = ? AND DATE(scanned_at) = DATE(?)`,
		result.ProductID, platformVal, result.CheckType, result.ScannedAt,
	).Error
	if err != nil {
		return err
	}
	if err := s.db.Create(result).Error; err != nil {
		return err
	}

	// Non-fatal: freshness is advisory.
	if err := s.freshnessWriter.RecordVerification(result.ProductID, result.Status); err != nil {
		s.logger.Warn("failed to update freshness", zap.Int64("product_id", result.ProductID), zap.Error(err))
	}
	return nil
}

// ListResults returns a paginated, filtered list of compliance check results.
// Pass 0 for productID to skip filtering on that dimension.
// Pass an empty string for status or riskLevel to skip filtering on that dimension.
func (s *Service) ListResults(p *common.Pagination, status, riskLevel string, productID int64) ([]CheckResult, int64, error) {
	q := s.db.Model(&CheckResult{})
	if productID > 0 {
		q = q.Where("product_id = ?", productID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if riskLevel != "" {
		q = q.Where("risk_level = ?", riskLevel)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []CheckResult
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// SuppressResult marks a compliance result as suppressed with a given reason.
// Returns gorm.ErrRecordNotFound if the result does not exist.
func (s *Service) SuppressResult(id int64, reason string) error {
	result := &CheckResult{}
	if err := s.db.First(result, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"is_suppressed":      true,
		"suppressed_reason":  reason,
		"suppressed_at":      &now,
	}
	return s.db.Model(result).Updates(updates).Error
}
