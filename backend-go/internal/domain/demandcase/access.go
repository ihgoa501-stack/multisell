package demandcase

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	AccessAvailable           = "available"
	AccessRequiresOwner       = "requires_owner_access"
	AccessRequiresListing     = "requires_listing"
	AccessRequiresTransaction = "requires_transaction"
	AccessUnavailable         = "unavailable"
	AccessUnknown             = "unknown"
)

type DataAccessRecord struct {
	ID                int64     `gorm:"primaryKey" json:"id"`
	DemandCaseID      int64     `gorm:"index;not null" json:"demand_case_id"`
	FieldName         string    `gorm:"size:120;not null" json:"field_name"`
	Status            string    `gorm:"size:32;not null" json:"status"`
	RequiredScope     string    `gorm:"size:160" json:"required_scope"`
	DecisionPurpose   string    `gorm:"type:text;not null" json:"decision_purpose"`
	RefusalImpact     string    `gorm:"type:text;not null" json:"refusal_impact"`
	SourceURI         string    `gorm:"type:text;not null" json:"source_uri"`
	RunID             string    `gorm:"size:80;not null" json:"run_id"`
	SnapshotID        int64     `gorm:"index;not null" json:"snapshot_id"`
	AccessMode        string    `gorm:"size:16;not null;default:read_only" json:"access_mode"`
	PreflightRequired bool      `gorm:"not null;default:false" json:"preflight_required"`
	CreatedAt         time.Time `json:"created_at"`
}

func (DataAccessRecord) TableName() string { return "demand_data_access" }

type PermissionRequest struct {
	FieldName       string `json:"field_name"`
	RequiredScope   string `json:"required_scope"`
	AccessMode      string `json:"access_mode"`
	DecisionPurpose string `json:"decision_purpose"`
	RefusalImpact   string `json:"refusal_impact"`
	SourceURI       string `json:"source_uri"`
}

func validAccessStatus(v string) bool {
	return v == AccessAvailable || v == AccessRequiresOwner || v == AccessRequiresListing || v == AccessRequiresTransaction || v == AccessUnavailable || v == AccessUnknown
}

func (s *Service) RecordDataAccess(ctx context.Context, ownerID int64, r *DataAccessRecord) error {
	if _, err := s.requireOwner(ctx, r.DemandCaseID, ownerID); err != nil {
		return err
	}
	if !validAccessStatus(r.Status) || strings.TrimSpace(r.FieldName) == "" || strings.TrimSpace(r.DecisionPurpose) == "" || strings.TrimSpace(r.RefusalImpact) == "" || strings.TrimSpace(r.SourceURI) == "" || strings.TrimSpace(r.RunID) == "" || r.SnapshotID <= 0 {
		return errors.New("invalid data access decision")
	}
	if r.AccessMode == "" {
		r.AccessMode = "read_only"
	}
	if r.AccessMode != "read_only" {
		return errors.New("only read-only access decisions are allowed")
	}
	if r.Status == AccessRequiresOwner && strings.TrimSpace(r.RequiredScope) == "" {
		return errors.New("owner access request requires a narrow read scope")
	}
	bad := strings.ToLower(r.RequiredScope)
	for _, token := range []string{"write", "listings", "feeds", "pricing"} {
		if strings.Contains(bad, token) {
			return errors.New("write-capable scope is forbidden in read-only preflight")
		}
	}
	var snap ResearchSnapshot
	if err := s.db.WithContext(ctx).Where("id = ? AND demand_case_id = ? AND owner_id = ? AND run_id = ?", r.SnapshotID, r.DemandCaseID, ownerID, r.RunID).First(&snap).Error; err != nil {
		return errors.New("data access decision requires a matching immutable research snapshot")
	}
	var existing DataAccessRecord
	err := s.db.WithContext(ctx).Where("demand_case_id = ? AND field_name = ? AND run_id = ?", r.DemandCaseID, r.FieldName, r.RunID).First(&existing).Error
	if err == nil {
		if existing.Status == r.Status && existing.RequiredScope == r.RequiredScope && existing.SnapshotID == r.SnapshotID {
			return nil
		}
		return errors.New("duplicate data access field has conflicting decision")
	}
	return s.db.WithContext(ctx).Create(r).Error
}

func (s *Service) PermissionRequests(ctx context.Context, id, ownerID int64) ([]PermissionRequest, error) {
	if _, err := s.requireOwner(ctx, id, ownerID); err != nil {
		return nil, err
	}
	var rows []DataAccessRecord
	if err := s.db.WithContext(ctx).Where("demand_case_id = ? AND preflight_required = ?", id, true).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	latest := map[string]DataAccessRecord{}
	for _, r := range rows {
		if _, ok := latest[r.FieldName]; !ok {
			latest[r.FieldName] = r
		}
	}
	out := make([]PermissionRequest, 0, len(latest))
	for _, r := range latest {
		if r.Status != AccessRequiresOwner {
			continue
		}
		out = append(out, PermissionRequest{FieldName: r.FieldName, RequiredScope: r.RequiredScope, AccessMode: r.AccessMode, DecisionPurpose: r.DecisionPurpose, RefusalImpact: r.RefusalImpact, SourceURI: r.SourceURI})
	}
	return out, nil
}
