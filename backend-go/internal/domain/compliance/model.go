package compliance

import (
	"time"

	"gorm.io/gorm"
)

// Check status constants.
const (
	StatusPass    = "pass"
	StatusWarning = "warning"
	StatusFail    = "fail"
)

// Risk level constants.
const (
	RiskLow     = "low"
	RiskMedium  = "medium"
	RiskHigh    = "high"
	RiskUnknown = "unknown"
)

// EvidenceItem represents one piece of evidence for a compliance finding.
type EvidenceItem struct {
	Rule      string `json:"rule"`
	Field     string `json:"field,omitempty"`
	Value     string `json:"value,omitempty"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp,omitempty"`
}

// CheckResult stores one compliance scan result per (product, platform, check_type) per day.
type CheckResult struct {
	ID               int64            `json:"id" gorm:"primaryKey"`
	ProductID        int64            `json:"product_id" gorm:"index;not null"`
	PlatformID       *int64           `json:"platform_id,omitempty"`
	CheckType        string           `json:"check_type" gorm:"default:'compliance'"`
	Status           string           `json:"status" gorm:"default:'pass'"`
	RiskLevel        *string          `json:"risk_level,omitempty"`
	RuleVersion      int              `json:"rule_version" gorm:"default:1"`
	Evidence         []EvidenceItem   `json:"evidence,omitempty" gorm:"serializer:json"`
	ScannedAt        time.Time        `json:"scanned_at" gorm:"column:scanned_at;not null"`
	NextScanAt       *time.Time       `json:"next_scan_at,omitempty"`
	IsSuppressed     bool             `json:"is_suppressed" gorm:"default:false"`
	SuppressedReason *string          `json:"suppressed_reason,omitempty"`
	SuppressedAt     *time.Time       `json:"suppressed_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	DeletedAt        gorm.DeletedAt   `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName overrides the default table name.
func (CheckResult) TableName() string {
	return "compliance_check_result"
}

// IsPass returns true if the result status is "pass".
func (r *CheckResult) IsPass() bool {
	return r.Status == StatusPass
}
