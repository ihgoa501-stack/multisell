package sourcing1688

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ComplianceReviewPending  = "pending"
	ComplianceReviewApproved = "approved"
	ComplianceReviewRejected = "rejected"
)

var StandardPublishComplianceRequirementCodes = []string{
	"brand_ip", "patent", "certification", "dangerous_goods", "material", "labeling_instructions",
}

// SourcingComplianceEvidence is an independent, Owner-scoped compliance fact.
// It is deliberately separate from draft JSON and freezes the authority chain
// and the exact market/channel/SKU scope to which the evidence applies.
type SourcingComplianceEvidence struct {
	ID                   int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID              int64      `gorm:"column:owner_id;not null" json:"owner_id"`
	SourcingProductID    int64      `gorm:"column:sourcing_product_id;not null" json:"sourcing_product_id"`
	TaskLinkID           int64      `gorm:"column:task_link_id;not null" json:"task_link_id"`
	ProductOpportunityID int64      `gorm:"column:product_opportunity_id;not null" json:"product_opportunity_id"`
	SourceSnapshotID     int64      `gorm:"column:source_snapshot_id;not null" json:"source_snapshot_id"`
	ProductID            int64      `gorm:"column:product_id;not null" json:"product_id"`
	InternalSKUID        *int64     `gorm:"column:internal_sku_id" json:"internal_sku_id,omitempty"`
	CountryCode          string     `gorm:"column:country_code;size:16;not null" json:"country_code"`
	ChannelCode          string     `gorm:"column:channel_code;size:64;not null" json:"channel_code"`
	RequirementCode      string     `gorm:"column:requirement_code;size:120;not null" json:"requirement_code"`
	RequirementText      string     `gorm:"column:requirement_text;type:text;not null" json:"requirement_text"`
	EvidenceSource       string     `gorm:"column:evidence_source;type:text;not null" json:"evidence_source"`
	TruthStatus          string     `gorm:"column:truth_status;size:24;not null" json:"truth_status"`
	Scope                string     `gorm:"column:scope;type:text;not null" json:"scope"`
	IssuedAt             *time.Time `gorm:"column:issued_at" json:"issued_at,omitempty"`
	ObservedAt           time.Time  `gorm:"column:observed_at;not null" json:"observed_at"`
	ExpiresAt            *time.Time `gorm:"column:expires_at" json:"expires_at,omitempty"`
	RevokedAt            *time.Time `gorm:"column:revoked_at" json:"revoked_at,omitempty"`
	RevokedBy            *int64     `gorm:"column:revoked_by" json:"revoked_by,omitempty"`
	RevocationReason     string     `gorm:"column:revocation_reason;type:text;not null;default:''" json:"revocation_reason,omitempty"`
	ReviewStatus         string     `gorm:"column:review_status;size:16;not null;default:pending" json:"review_status"`
	ReviewedBy           *int64     `gorm:"column:reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt           *time.Time `gorm:"column:reviewed_at" json:"reviewed_at,omitempty"`
	ReviewNotes          string     `gorm:"column:review_notes;type:text;not null;default:''" json:"review_notes,omitempty"`
	CreatedBy            int64      `gorm:"column:created_by;not null" json:"created_by"`
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SourcingComplianceEvidence) TableName() string { return "sourcing_compliance_evidence" }

type CreateComplianceEvidenceInput struct {
	OwnerID         int64      `json:"-"`
	ProductID       int64      `json:"product_id" binding:"required"`
	InternalSKUID   *int64     `json:"internal_sku_id"`
	CountryCode     string     `json:"country_code" binding:"required"`
	ChannelCode     string     `json:"channel_code" binding:"required"`
	RequirementCode string     `json:"requirement_code" binding:"required"`
	RequirementText string     `json:"requirement_text" binding:"required"`
	EvidenceSource  string     `json:"evidence_source" binding:"required"`
	TruthStatus     string     `json:"truth_status" binding:"required"`
	Scope           string     `json:"scope" binding:"required"`
	IssuedAt        *time.Time `json:"issued_at"`
	ObservedAt      time.Time  `json:"observed_at" binding:"required"`
	ExpiresAt       *time.Time `json:"expires_at"`
}

type ReviewComplianceEvidenceInput struct {
	OwnerID  int64  `json:"-"`
	Decision string `json:"decision" binding:"required"`
	Notes    string `json:"notes" binding:"required"`
}

type RevokeComplianceEvidenceInput struct {
	OwnerID int64  `json:"-"`
	Reason  string `json:"reason" binding:"required"`
}

func (s *Service) ListComplianceEvidence(sourceID, taskLinkID, ownerID int64) ([]SourcingComplianceEvidence, error) {
	if _, err := requireTaskSourcingAuthority(s.db, sourceID, ownerID, taskLinkID); err != nil {
		return nil, err
	}
	var rows []SourcingComplianceEvidence
	err := s.db.Where("owner_id = ? AND sourcing_product_id = ? AND task_link_id = ?", ownerID, sourceID, taskLinkID).Order("id DESC").Find(&rows).Error
	return rows, err
}

func (s *Service) CreateComplianceEvidence(sourceID, taskLinkID int64, in *CreateComplianceEvidenceInput) (*SourcingComplianceEvidence, error) {
	if in == nil || in.OwnerID <= 0 || in.ProductID <= 0 || in.ObservedAt.IsZero() || strings.TrimSpace(in.CountryCode) == "" || strings.TrimSpace(in.ChannelCode) == "" || strings.TrimSpace(in.RequirementCode) == "" || strings.TrimSpace(in.RequirementText) == "" || strings.TrimSpace(in.EvidenceSource) == "" || strings.TrimSpace(in.Scope) == "" {
		return nil, fmt.Errorf("%w: complete compliance identity, requirement, source, scope and observation are required", ErrInvalidWorkflow)
	}
	truth := strings.TrimSpace(in.TruthStatus)
	if truth != "actual" && truth != "quoted" && truth != "estimated" && truth != "unknown" && truth != "mock" && truth != "inferred" {
		return nil, fmt.Errorf("%w: invalid truth status", ErrInvalidWorkflow)
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(in.ObservedAt) {
		return nil, fmt.Errorf("%w: expiry must be after observation", ErrInvalidWorkflow)
	}
	var out SourcingComplianceEvidence
	err := s.db.Transaction(func(tx *gorm.DB) error {
		link, err := requireTaskSourcingAuthority(tx, sourceID, in.OwnerID, taskLinkID)
		if err != nil {
			return err
		}
		var source Sourcing1688Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ?", sourceID, in.OwnerID).First(&source).Error; err != nil {
			return err
		}
		if source.SnapshotID == nil || source.ProductID == nil || *source.ProductID != in.ProductID {
			return fmt.Errorf("%w: source snapshot and converted product identity must match", ErrWorkflowGate)
		}
		if link.ProductOpportunityID == nil {
			return fmt.Errorf("%w: product opportunity authority required", ErrWorkflowGate)
		}
		var market demandCaseRow
		if err := tx.Where("id = ? AND owner_id = ?", link.DemandCaseID, in.OwnerID).First(&market).Error; err != nil {
			return fmt.Errorf("%w: selected market identity is unavailable", ErrWorkflowGate)
		}
		if strings.TrimSpace(market.Region) == "" || strings.TrimSpace(market.SalesChannel) == "" || !strings.EqualFold(strings.TrimSpace(in.CountryCode), strings.TrimSpace(market.Region)) || !strings.EqualFold(strings.TrimSpace(in.ChannelCode), strings.TrimSpace(market.SalesChannel)) {
			return fmt.Errorf("%w: compliance country and channel must match the frozen selected market", ErrWorkflowGate)
		}
		if in.InternalSKUID != nil {
			var count int64
			if err := tx.Table("sku").Where("id = ? AND product_id = ?", *in.InternalSKUID, in.ProductID).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return fmt.Errorf("%w: SKU does not belong to frozen product", ErrWorkflowGate)
			}
		}
		out = SourcingComplianceEvidence{OwnerID: in.OwnerID, SourcingProductID: sourceID, TaskLinkID: taskLinkID, ProductOpportunityID: *link.ProductOpportunityID, SourceSnapshotID: *source.SnapshotID, ProductID: in.ProductID, InternalSKUID: in.InternalSKUID, CountryCode: strings.ToUpper(strings.TrimSpace(in.CountryCode)), ChannelCode: strings.ToLower(strings.TrimSpace(in.ChannelCode)), RequirementCode: strings.TrimSpace(in.RequirementCode), RequirementText: strings.TrimSpace(in.RequirementText), EvidenceSource: strings.TrimSpace(in.EvidenceSource), TruthStatus: truth, Scope: strings.TrimSpace(in.Scope), IssuedAt: in.IssuedAt, ObservedAt: in.ObservedAt.UTC(), ExpiresAt: in.ExpiresAt, ReviewStatus: ComplianceReviewPending, CreatedBy: in.OwnerID}
		return tx.Create(&out).Error
	})
	return &out, err
}

func (s *Service) ReviewComplianceEvidence(sourceID, taskLinkID, evidenceID int64, in *ReviewComplianceEvidenceInput) (*SourcingComplianceEvidence, error) {
	if in == nil || in.OwnerID <= 0 || (in.Decision != ComplianceReviewApproved && in.Decision != ComplianceReviewRejected) || strings.TrimSpace(in.Notes) == "" {
		return nil, ErrInvalidWorkflow
	}
	var row SourcingComplianceEvidence
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := requireTaskSourcingAuthority(tx, sourceID, in.OwnerID, taskLinkID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ? AND sourcing_product_id = ? AND task_link_id = ?", evidenceID, in.OwnerID, sourceID, taskLinkID).First(&row).Error; err != nil {
			return err
		}
		if row.ReviewStatus != ComplianceReviewPending {
			return fmt.Errorf("%w: compliance review is immutable once decided", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		if in.Decision == ComplianceReviewApproved && (row.TruthStatus != "actual" || row.RevokedAt != nil || (row.ExpiresAt != nil && !row.ExpiresAt.After(now))) {
			return fmt.Errorf("%w: only current, non-revoked actual evidence can pass compliance", ErrWorkflowGate)
		}
		if err := tx.Model(&row).Updates(map[string]any{"review_status": in.Decision, "reviewed_by": in.OwnerID, "reviewed_at": now, "review_notes": strings.TrimSpace(in.Notes)}).Error; err != nil {
			return err
		}
		return tx.First(&row, evidenceID).Error
	})
	return &row, err
}

func (s *Service) RevokeComplianceEvidence(sourceID, taskLinkID, evidenceID int64, in *RevokeComplianceEvidenceInput) (*SourcingComplianceEvidence, error) {
	if in == nil || in.OwnerID <= 0 || strings.TrimSpace(in.Reason) == "" {
		return nil, ErrInvalidWorkflow
	}
	var row SourcingComplianceEvidence
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := requireTaskSourcingAuthority(tx, sourceID, in.OwnerID, taskLinkID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ? AND sourcing_product_id = ? AND task_link_id = ?", evidenceID, in.OwnerID, sourceID, taskLinkID).First(&row).Error; err != nil {
			return err
		}
		if row.RevokedAt != nil {
			return fmt.Errorf("%w: compliance evidence already revoked", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		if err := tx.Model(&row).Updates(map[string]any{"revoked_at": now, "revoked_by": in.OwnerID, "revocation_reason": strings.TrimSpace(in.Reason)}).Error; err != nil {
			return err
		}
		return tx.First(&row, evidenceID).Error
	})
	return &row, err
}

// RequireCurrentCompliance is the fail-closed integration seam for acceptance
// and publish gates: every required code needs a current approved actual fact.
func (s *Service) RequireCurrentCompliance(sourceID, taskLinkID, ownerID int64, requirementCodes []string, now time.Time) error {
	return requireCurrentCompliance(s.db, sourceID, taskLinkID, ownerID, requirementCodes, now)
}

func requireCurrentCompliance(db *gorm.DB, sourceID, taskLinkID, ownerID int64, requirementCodes []string, now time.Time) error {
	if _, err := requireTaskSourcingAuthority(db, sourceID, ownerID, taskLinkID); err != nil {
		return err
	}
	for _, code := range requirementCodes {
		var count int64
		err := db.Model(&SourcingComplianceEvidence{}).Where("owner_id = ? AND sourcing_product_id = ? AND task_link_id = ? AND requirement_code = ? AND truth_status = 'actual' AND review_status = 'approved' AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", ownerID, sourceID, taskLinkID, strings.TrimSpace(code), now.UTC()).Count(&count).Error
		if err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("%w: current approved actual compliance evidence missing for %s", ErrWorkflowGate, code)
		}
	}
	return nil
}
