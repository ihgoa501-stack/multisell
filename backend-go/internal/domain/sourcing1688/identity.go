package sourcing1688

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SourcingChangeEvent struct {
	ID                 int64           `gorm:"primaryKey" json:"id"`
	SourcingProductID  int64           `gorm:"not null;index" json:"sourcing_product_id"`
	PreviousSnapshotID *int64          `json:"previous_snapshot_id,omitempty"`
	CurrentSnapshotID  int64           `gorm:"not null;index" json:"current_snapshot_id"`
	ChangeType         string          `gorm:"size:40;not null" json:"change_type"`
	BeforeValue        json.RawMessage `gorm:"type:jsonb" json:"before_value,omitempty"`
	AfterValue         json.RawMessage `gorm:"type:jsonb" json:"after_value,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

func (SourcingChangeEvent) TableName() string { return "sourcing_1688_change_event" }

type DuplicateCandidate struct {
	ID               int64      `gorm:"primaryKey" json:"id"`
	SourceProductID  int64      `gorm:"not null;index" json:"source_product_id"`
	MatchedProductID int64      `gorm:"not null;index" json:"matched_product_id"`
	MatchType        string     `gorm:"size:32;not null" json:"match_type"`
	Fingerprint      string     `gorm:"size:64;not null" json:"fingerprint"`
	Status           string     `gorm:"size:20;not null" json:"status"`
	ReviewedBy       *int64     `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

func (DuplicateCandidate) TableName() string { return "sourcing_1688_duplicate_candidate" }

type IdentityHistory struct {
	Snapshots  []Sourcing1688Snapshot `json:"snapshots"`
	Changes    []SourcingChangeEvent  `json:"changes"`
	Duplicates []DuplicateCandidate   `json:"duplicates"`
}

type ResolveDuplicateInput struct {
	ReviewedBy int64  `json:"reviewed_by"`
	Decision   string `json:"decision" binding:"required,oneof=same_product different_product"`
}

func productFingerprint(title string, variants json.RawMessage) (string, error) {
	var decoded any
	if len(variants) > 0 {
		if err := json.Unmarshal(variants, &decoded); err != nil {
			return "", fmt.Errorf("invalid variants for fingerprint: %w", err)
		}
	}
	normalizedTitle := strings.ToLower(strings.Join(strings.Fields(title), " "))
	payload, err := json.Marshal(struct {
		Title    string `json:"title"`
		Variants any    `json:"variants"`
	}{normalizedTitle, decoded})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func jsonValue(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func optionalFloatValue(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func (s *Service) recordIdentityAndChanges(tx *gorm.DB, p *Sourcing1688Product, previous *Sourcing1688Snapshot, current *Sourcing1688Snapshot) error {
	title := ""
	if current.ObservedTitle != nil {
		title = *current.ObservedTitle
	}
	variants := json.RawMessage(nil)
	if p.SkuVariants != nil {
		variants = *p.SkuVariants
	}
	fingerprint, err := productFingerprint(title, variants)
	if err != nil {
		return err
	}
	if err := tx.Model(p).Update("source_product_fingerprint", fingerprint).Error; err != nil {
		return err
	}
	p.SourceProductFingerprint = fingerprint
	current.ProductFingerprint = fingerprint
	if err := tx.Model(current).Update("product_fingerprint", fingerprint).Error; err != nil {
		return err
	}

	if previous != nil {
		changes := map[string][2]any{}
		if fmt.Sprint(optionalFloatValue(previous.ObservedPrice)) != fmt.Sprint(optionalFloatValue(current.ObservedPrice)) {
			changes["price"] = [2]any{optionalFloatValue(previous.ObservedPrice), optionalFloatValue(current.ObservedPrice)}
		}
		if previous.ObservedMOQ != current.ObservedMOQ {
			changes["moq"] = [2]any{previous.ObservedMOQ, current.ObservedMOQ}
		}
		if previous.ObservedSupplier != current.ObservedSupplier {
			changes["supplier"] = [2]any{previous.ObservedSupplier, current.ObservedSupplier}
		}
		if previous.ObservedSupplierBusinessID != current.ObservedSupplierBusinessID {
			changes["supplier_business_id"] = [2]any{previous.ObservedSupplierBusinessID, current.ObservedSupplierBusinessID}
		}
		if previous.SourceURL != current.SourceURL {
			changes["source_url"] = [2]any{previous.SourceURL, current.SourceURL}
		}
		if previous.ProductFingerprint != current.ProductFingerprint {
			changes["product_identity"] = [2]any{previous.ProductFingerprint, current.ProductFingerprint}
		}
		keys := make([]string, 0, len(changes))
		for key := range changes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			values := changes[key]
			event := SourcingChangeEvent{SourcingProductID: p.ID, PreviousSnapshotID: &previous.ID, CurrentSnapshotID: current.ID, ChangeType: key, BeforeValue: jsonValue(values[0]), AfterValue: jsonValue(values[1])}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&event).Error; err != nil {
				return err
			}
		}
	}

	var matches []Sourcing1688Product
	if err := tx.Where("source_product_fingerprint = ? AND id <> ?", fingerprint, p.ID).Order("id").Find(&matches).Error; err != nil {
		return err
	}
	for _, match := range matches {
		candidate := DuplicateCandidate{SourceProductID: p.ID, MatchedProductID: match.ID, MatchType: "content_fingerprint", Fingerprint: fingerprint, Status: "pending_review"}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&candidate).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetIdentityHistory(id int64) (*IdentityHistory, error) {
	var history IdentityHistory
	if err := s.db.Where("sourcing_product_id = ?", id).Order("id DESC").Find(&history.Snapshots).Error; err != nil {
		return nil, err
	}
	if len(history.Snapshots) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if err := s.db.Where("sourcing_product_id = ?", id).Order("id DESC").Find(&history.Changes).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("source_product_id = ? OR matched_product_id = ?", id, id).Order("id DESC").Find(&history.Duplicates).Error; err != nil {
		return nil, err
	}
	return &history, nil
}

func (s *Service) ResolveDuplicate(candidateID int64, in *ResolveDuplicateInput) (*DuplicateCandidate, error) {
	if in.ReviewedBy <= 0 || (in.Decision != "same_product" && in.Decision != "different_product") {
		return nil, ErrInvalidWorkflow
	}
	var candidate DuplicateCandidate
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&candidate, candidateID).Error; err != nil {
			return err
		}
		if candidate.Status != "pending_review" {
			return fmt.Errorf("%w: duplicate candidate already reviewed", ErrWorkflowGate)
		}
		var source Sourcing1688Product
		if err := tx.First(&source, candidate.SourceProductID).Error; err != nil {
			return err
		}
		var dc demandCaseRow
		if source.DemandCaseID == nil || tx.First(&dc, *source.DemandCaseID).Error != nil || dc.OwnerID != in.ReviewedBy {
			return fmt.Errorf("%w: duplicate decision requires workflow Owner", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		if err := tx.Model(&candidate).Updates(map[string]any{"status": in.Decision, "reviewed_by": in.ReviewedBy, "reviewed_at": now}).Error; err != nil {
			return err
		}
		candidate.Status, candidate.ReviewedBy, candidate.ReviewedAt = in.Decision, &in.ReviewedBy, &now
		return nil
	})
	return &candidate, err
}
