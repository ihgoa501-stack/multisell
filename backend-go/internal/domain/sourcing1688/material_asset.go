package sourcing1688

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	MaterialStatusActive   = "active"
	MaterialStatusArchived = "archived"
)

type SourcingMaterialAsset struct {
	ID                    int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID               int64      `gorm:"column:owner_id;not null;index" json:"owner_id"`
	SourcingProductID     int64      `gorm:"column:sourcing_product_id;not null" json:"sourcing_product_id"`
	TaskLinkID            int64      `gorm:"column:task_link_id;not null" json:"task_link_id"`
	SnapshotID            int64      `gorm:"column:snapshot_id;not null" json:"snapshot_id"`
	CanonicalSKUMappingID *int64     `gorm:"column:canonical_sku_mapping_id" json:"canonical_sku_mapping_id,omitempty"`
	Role                  string     `gorm:"column:role;size:16;not null" json:"role"`
	Ordinal               int        `gorm:"column:ordinal;not null" json:"ordinal"`
	SourceURL             string     `gorm:"column:source_url;type:text;not null" json:"source_url"`
	SourceSHA256          string     `gorm:"column:source_sha256;size:64;not null" json:"source_sha256"`
	MediaType             string     `gorm:"column:media_type;size:16;not null" json:"media_type"`
	MIMEType              string     `gorm:"column:mime_type;size:120;not null" json:"mime_type"`
	ByteSize              int64      `gorm:"column:byte_size;not null" json:"byte_size"`
	Width                 *int       `gorm:"column:width" json:"width,omitempty"`
	Height                *int       `gorm:"column:height" json:"height,omitempty"`
	DurationMS            *int64     `gorm:"column:duration_ms" json:"duration_ms,omitempty"`
	Status                string     `gorm:"column:status;size:16;not null" json:"status"`
	UsedAt                *time.Time `gorm:"column:used_at" json:"used_at,omitempty"`
	CreatedBy             int64      `gorm:"column:created_by;not null" json:"created_by"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SourcingMaterialAsset) TableName() string { return "sourcing_material_asset" }

type SourcingMaterialRightsEvidence struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AssetID      int64           `gorm:"column:asset_id;not null" json:"asset_id"`
	OwnerID      int64           `gorm:"column:owner_id;not null" json:"owner_id"`
	Version      int             `gorm:"column:version;not null" json:"version"`
	Status       string          `gorm:"column:status;size:16;not null" json:"status"`
	LicenseScope string          `gorm:"column:license_scope;type:text;not null" json:"license_scope"`
	Countries    json.RawMessage `gorm:"column:countries;type:jsonb;not null" json:"countries"`
	Channels     json.RawMessage `gorm:"column:channels;type:jsonb;not null" json:"channels"`
	Licensor     string          `gorm:"column:licensor;size:240;not null" json:"licensor"`
	SourceURI    string          `gorm:"column:source_uri;type:text;not null" json:"source_uri"`
	ObservedAt   time.Time       `gorm:"column:observed_at;not null" json:"observed_at"`
	ValidUntil   *time.Time      `gorm:"column:valid_until" json:"valid_until,omitempty"`
	SubmittedBy  int64           `gorm:"column:submitted_by;not null" json:"submitted_by"`
	ReviewedBy   *int64          `gorm:"column:reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time      `gorm:"column:reviewed_at" json:"reviewed_at,omitempty"`
	ReviewNote   string          `gorm:"column:review_note;type:text;not null" json:"review_note"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SourcingMaterialRightsEvidence) TableName() string { return "sourcing_material_rights_evidence" }

type SourcingMaterialRendition struct {
	ID                      int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AssetID                 int64     `gorm:"column:asset_id;not null" json:"asset_id"`
	OwnerID                 int64     `gorm:"column:owner_id;not null" json:"owner_id"`
	ImageProcessingRecordID int64     `gorm:"column:image_processing_record_id;not null" json:"image_processing_record_id"`
	CreatedBy               int64     `gorm:"column:created_by;not null" json:"created_by"`
	CreatedAt               time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SourcingMaterialRendition) TableName() string { return "sourcing_material_rendition" }

type CreateMaterialAssetInput struct {
	SnapshotID            int64  `json:"snapshot_id" binding:"required"`
	Role                  string `json:"role" binding:"required"`
	Ordinal               int    `json:"ordinal"`
	CanonicalSKUMappingID *int64 `json:"canonical_sku_mapping_id"`
	SourceURL             string `json:"source_url" binding:"required"`
	SourceSHA256          string `json:"source_sha256" binding:"required"`
	MediaType             string `json:"media_type" binding:"required"`
	MIMEType              string `json:"mime_type" binding:"required"`
	ByteSize              int64  `json:"byte_size"`
	Width                 *int   `json:"width"`
	Height                *int   `json:"height"`
	DurationMS            *int64 `json:"duration_ms"`
}

type CreateMaterialRightsInput struct {
	LicenseScope string     `json:"license_scope" binding:"required"`
	Countries    []string   `json:"countries" binding:"required"`
	Channels     []string   `json:"channels" binding:"required"`
	Licensor     string     `json:"licensor" binding:"required"`
	SourceURI    string     `json:"source_uri" binding:"required"`
	ObservedAt   time.Time  `json:"observed_at" binding:"required"`
	ValidUntil   *time.Time `json:"valid_until"`
}
type ReviewMaterialRightsInput struct {
	Decision   string `json:"decision" binding:"required"`
	ReviewNote string `json:"review_note" binding:"required"`
}
type MaterialOrderInput struct {
	Ordinal int `json:"ordinal"`
}
type MaterialRenditionInput struct {
	ImageProcessingRecordID int64 `json:"image_processing_record_id" binding:"required"`
}
type MaterialReviewNoteInput struct {
	ReviewNote string `json:"review_note" binding:"required"`
}

type MaterialRightsView struct {
	SourcingMaterialRightsEvidence
	EffectiveStatus string `json:"effective_status"`
}
type MaterialAssetView struct {
	SourcingMaterialAsset
	ProcessingStatus string                      `json:"processing_status"`
	Blocker          string                      `json:"blocker,omitempty"`
	PreviewURL       string                      `json:"preview_url,omitempty"`
	LatestRights     *MaterialRightsView         `json:"latest_rights,omitempty"`
	RightsVersions   []MaterialRightsView        `json:"rights_versions"`
	Renditions       []SourcingMaterialRendition `json:"renditions"`
}
type MaterialAssetList struct {
	Assets []MaterialAssetView `json:"assets"`
}

func validMaterialRole(role string) bool {
	switch role {
	case "main", "gallery", "sku", "detail", "video":
		return true
	}
	return false
}
func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (s *Service) requireMaterialContext(ownerID, sourceID, taskLinkID int64) (*Sourcing1688TaskLink, error) {
	if ownerID <= 0 || sourceID <= 0 || taskLinkID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	return findOwnedTaskLink(s.db, sourceID, ownerID, taskLinkID)
}

func (s *Service) CreateMaterialAsset(ownerID, sourceID, taskLinkID int64, in CreateMaterialAssetInput) (*SourcingMaterialAsset, error) {
	if _, err := s.requireMaterialContext(ownerID, sourceID, taskLinkID); err != nil {
		return nil, err
	}
	in.Role, in.SourceURL, in.SourceSHA256, in.MediaType, in.MIMEType = strings.TrimSpace(in.Role), strings.TrimSpace(in.SourceURL), strings.ToLower(strings.TrimSpace(in.SourceSHA256)), strings.TrimSpace(in.MediaType), strings.TrimSpace(in.MIMEType)
	if in.SnapshotID <= 0 || !validMaterialRole(in.Role) || in.Ordinal < 0 || in.SourceURL == "" || !validSHA256(in.SourceSHA256) || in.ByteSize < 0 || (in.MediaType != "image" && in.MediaType != "video") || (in.Role == "video") != (in.MediaType == "video") || !strings.HasPrefix(in.MIMEType, in.MediaType+"/") {
		return nil, ErrInvalidWorkflow
	}
	if in.Width != nil && *in.Width <= 0 || in.Height != nil && *in.Height <= 0 || in.DurationMS != nil && *in.DurationMS < 0 {
		return nil, ErrInvalidWorkflow
	}
	asset := SourcingMaterialAsset{OwnerID: ownerID, SourcingProductID: sourceID, TaskLinkID: taskLinkID, SnapshotID: in.SnapshotID, CanonicalSKUMappingID: in.CanonicalSKUMappingID, Role: in.Role, Ordinal: in.Ordinal, SourceURL: in.SourceURL, SourceSHA256: in.SourceSHA256, MediaType: in.MediaType, MIMEType: in.MIMEType, ByteSize: in.ByteSize, Width: in.Width, Height: in.Height, DurationMS: in.DurationMS, Status: MaterialStatusActive, CreatedBy: ownerID}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&Sourcing1688Snapshot{}).Where("id = ? AND sourcing_product_id = ?", in.SnapshotID, sourceID).Count(&count).Error; err != nil || count != 1 {
			return fmt.Errorf("%w: source snapshot mismatch", ErrWorkflowGate)
		}
		if in.CanonicalSKUMappingID != nil {
			if err := tx.Model(&SourcingSKUMapping{}).Where("id = ? AND owner_id = ? AND sourcing_product_id = ? AND task_link_id = ?", *in.CanonicalSKUMappingID, ownerID, sourceID, taskLinkID).Count(&count).Error; err != nil || count != 1 {
				return fmt.Errorf("%w: canonical SKU mapping mismatch", ErrWorkflowGate)
			}
		}
		if err := tx.Where("owner_id = ? AND task_link_id = ? AND role = ? AND ordinal = ? AND status = ?", ownerID, taskLinkID, in.Role, in.Ordinal, MaterialStatusActive).First(&SourcingMaterialAsset{}).Error; err == nil {
			return fmt.Errorf("%w: material ordinal already used", ErrWorkflowGate)
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if in.Role == "main" {
			if err := tx.Where("owner_id = ? AND task_link_id = ? AND role = 'main' AND status = ?", ownerID, taskLinkID, MaterialStatusActive).First(&SourcingMaterialAsset{}).Error; err == nil {
				return fmt.Errorf("%w: exactly one active main asset is allowed", ErrWorkflowGate)
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		return tx.Create(&asset).Error
	})
	return &asset, err
}

func effectiveMaterialRights(row SourcingMaterialRightsEvidence, now time.Time) MaterialRightsView {
	status := row.Status
	if status == "approved" && row.ValidUntil != nil && !row.ValidUntil.After(now) {
		status = "expired"
	}
	return MaterialRightsView{SourcingMaterialRightsEvidence: row, EffectiveStatus: status}
}

func (s *Service) ListMaterialAssets(ownerID, sourceID, taskLinkID int64) (*MaterialAssetList, error) {
	if _, err := s.requireMaterialContext(ownerID, sourceID, taskLinkID); err != nil {
		return nil, err
	}
	var assets []SourcingMaterialAsset
	if err := s.db.Where("owner_id = ? AND sourcing_product_id = ? AND task_link_id = ?", ownerID, sourceID, taskLinkID).Order("role ASC, ordinal ASC, id ASC").Find(&assets).Error; err != nil {
		return nil, err
	}
	result := &MaterialAssetList{Assets: make([]MaterialAssetView, 0, len(assets))}
	for _, asset := range assets {
		view := MaterialAssetView{SourcingMaterialAsset: asset, ProcessingStatus: "pending", PreviewURL: asset.SourceURL, RightsVersions: []MaterialRightsView{}, Renditions: []SourcingMaterialRendition{}}
		var rights []SourcingMaterialRightsEvidence
		if err := s.db.Where("asset_id = ? AND owner_id = ?", asset.ID, ownerID).Order("version DESC").Find(&rights).Error; err != nil {
			return nil, err
		}
		for _, row := range rights {
			view.RightsVersions = append(view.RightsVersions, effectiveMaterialRights(row, time.Now().UTC()))
		}
		if len(view.RightsVersions) > 0 {
			latest := view.RightsVersions[0]
			view.LatestRights = &latest
		}
		if err := s.db.Where("asset_id = ? AND owner_id = ?", asset.ID, ownerID).Order("id DESC").Find(&view.Renditions).Error; err != nil {
			return nil, err
		}
		if asset.MediaType == "video" {
			view.ProcessingStatus, view.Blocker = "blocked", "当前没有受控视频处理器；视频只能保存元数据和权利证据，不能进入草稿"
		} else if len(view.Renditions) > 0 {
			view.ProcessingStatus = "ready"
			view.PreviewURL = fmt.Sprintf("/api/v1/sourcing-1688/processed-images/%d/content", view.Renditions[0].ImageProcessingRecordID)
		}
		result.Assets = append(result.Assets, view)
	}
	return result, nil
}

func (s *Service) loadOwnedMaterial(tx *gorm.DB, ownerID, sourceID, taskLinkID, assetID int64, lock bool) (*SourcingMaterialAsset, error) {
	query := tx.Where("id = ? AND owner_id = ? AND sourcing_product_id = ? AND task_link_id = ?", assetID, ownerID, sourceID, taskLinkID)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var asset SourcingMaterialAsset
	if err := query.First(&asset).Error; err != nil {
		return nil, fmt.Errorf("%w: material asset not found in Owner task", ErrWorkflowGate)
	}
	return &asset, nil
}

func (s *Service) ReorderMaterialAsset(ownerID, sourceID, taskLinkID, assetID int64, ordinal int) (*SourcingMaterialAsset, error) {
	if ordinal < 0 {
		return nil, ErrInvalidWorkflow
	}
	if _, err := s.requireMaterialContext(ownerID, sourceID, taskLinkID); err != nil {
		return nil, err
	}
	var updated SourcingMaterialAsset
	err := s.db.Transaction(func(tx *gorm.DB) error {
		asset, err := s.loadOwnedMaterial(tx, ownerID, sourceID, taskLinkID, assetID, true)
		if err != nil {
			return err
		}
		if asset.Status != MaterialStatusActive {
			return fmt.Errorf("%w: archived material cannot be reordered", ErrWorkflowGate)
		}
		var count int64
		if err := tx.Model(&SourcingMaterialAsset{}).Where("owner_id = ? AND task_link_id = ? AND role = ? AND ordinal = ? AND status = ? AND id <> ?", ownerID, taskLinkID, asset.Role, ordinal, MaterialStatusActive, asset.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("%w: material ordinal already used", ErrWorkflowGate)
		}
		if err := tx.Model(asset).Updates(map[string]any{"ordinal": ordinal, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		return tx.First(&updated, asset.ID).Error
	})
	return &updated, err
}

func (s *Service) ArchiveMaterialAsset(ownerID, sourceID, taskLinkID, assetID int64) (*SourcingMaterialAsset, error) {
	if _, err := s.requireMaterialContext(ownerID, sourceID, taskLinkID); err != nil {
		return nil, err
	}
	var updated SourcingMaterialAsset
	err := s.db.Transaction(func(tx *gorm.DB) error {
		asset, err := s.loadOwnedMaterial(tx, ownerID, sourceID, taskLinkID, assetID, true)
		if err != nil {
			return err
		}
		if asset.UsedAt != nil {
			return fmt.Errorf("%w: material already used by a draft cannot be archived", ErrWorkflowGate)
		}
		if asset.Status != MaterialStatusActive {
			return fmt.Errorf("%w: material is already archived", ErrWorkflowGate)
		}
		if err := tx.Model(asset).Updates(map[string]any{"status": MaterialStatusArchived, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		return tx.First(&updated, asset.ID).Error
	})
	return &updated, err
}

func (s *Service) AddMaterialRights(ownerID, sourceID, taskLinkID, assetID int64, in CreateMaterialRightsInput) (*SourcingMaterialRightsEvidence, error) {
	if _, err := s.requireMaterialContext(ownerID, sourceID, taskLinkID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.LicenseScope) == "" || len(in.Countries) == 0 || len(in.Channels) == 0 || strings.TrimSpace(in.Licensor) == "" || strings.TrimSpace(in.SourceURI) == "" || in.ObservedAt.IsZero() || (in.ValidUntil != nil && !in.ValidUntil.After(in.ObservedAt)) {
		return nil, ErrInvalidWorkflow
	}
	countries, _ := json.Marshal(in.Countries)
	channels, _ := json.Marshal(in.Channels)
	var row SourcingMaterialRightsEvidence
	err := s.db.Transaction(func(tx *gorm.DB) error {
		asset, err := s.loadOwnedMaterial(tx, ownerID, sourceID, taskLinkID, assetID, true)
		if err != nil {
			return err
		}
		if asset.Status != MaterialStatusActive {
			return fmt.Errorf("%w: archived material cannot receive rights evidence", ErrWorkflowGate)
		}
		var maxVersion int
		if err := tx.Model(&SourcingMaterialRightsEvidence{}).Where("asset_id = ?", assetID).Select("COALESCE(MAX(version),0)").Scan(&maxVersion).Error; err != nil {
			return err
		}
		row = SourcingMaterialRightsEvidence{AssetID: assetID, OwnerID: ownerID, Version: maxVersion + 1, Status: "pending", LicenseScope: strings.TrimSpace(in.LicenseScope), Countries: countries, Channels: channels, Licensor: strings.TrimSpace(in.Licensor), SourceURI: strings.TrimSpace(in.SourceURI), ObservedAt: in.ObservedAt.UTC(), ValidUntil: in.ValidUntil, SubmittedBy: ownerID}
		return tx.Create(&row).Error
	})
	return &row, err
}

func (s *Service) ReviewMaterialRights(ownerID, sourceID, taskLinkID, assetID, evidenceID int64, in ReviewMaterialRightsInput) (*SourcingMaterialRightsEvidence, error) {
	if in.Decision != "approved" && in.Decision != "rejected" || strings.TrimSpace(in.ReviewNote) == "" {
		return nil, ErrInvalidWorkflow
	}
	if _, err := s.requireMaterialContext(ownerID, sourceID, taskLinkID); err != nil {
		return nil, err
	}
	var out SourcingMaterialRightsEvidence
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.loadOwnedMaterial(tx, ownerID, sourceID, taskLinkID, assetID, true); err != nil {
			return err
		}
		var row SourcingMaterialRightsEvidence
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND asset_id=? AND owner_id=?", evidenceID, assetID, ownerID).First(&row).Error; err != nil {
			return fmt.Errorf("%w: rights evidence not found", ErrWorkflowGate)
		}
		if row.Status != "pending" {
			return fmt.Errorf("%w: rights evidence already decided", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		if row.ValidUntil != nil && !row.ValidUntil.After(now) && in.Decision == "approved" {
			return fmt.Errorf("%w: expired rights evidence cannot be approved", ErrWorkflowGate)
		}
		if err := tx.Model(&row).Updates(map[string]any{"status": in.Decision, "reviewed_by": ownerID, "reviewed_at": now, "review_note": strings.TrimSpace(in.ReviewNote)}).Error; err != nil {
			return err
		}
		return tx.First(&out, row.ID).Error
	})
	return &out, err
}

func (s *Service) RevokeMaterialRights(ownerID, sourceID, taskLinkID, assetID, evidenceID int64, note string) (*SourcingMaterialRightsEvidence, error) {
	if strings.TrimSpace(note) == "" {
		return nil, ErrInvalidWorkflow
	}
	if _, err := s.requireMaterialContext(ownerID, sourceID, taskLinkID); err != nil {
		return nil, err
	}
	var out SourcingMaterialRightsEvidence
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.loadOwnedMaterial(tx, ownerID, sourceID, taskLinkID, assetID, true); err != nil {
			return err
		}
		var row SourcingMaterialRightsEvidence
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND asset_id=? AND owner_id=?", evidenceID, assetID, ownerID).First(&row).Error; err != nil {
			return fmt.Errorf("%w: rights evidence not found", ErrWorkflowGate)
		}
		if row.Status != "approved" {
			return fmt.Errorf("%w: only approved rights can be revoked", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		if err := tx.Model(&row).Updates(map[string]any{"status": "revoked", "reviewed_by": ownerID, "reviewed_at": now, "review_note": strings.TrimSpace(note)}).Error; err != nil {
			return err
		}
		return tx.First(&out, row.ID).Error
	})
	return &out, err
}

func (s *Service) AttachMaterialRendition(ownerID, sourceID, taskLinkID, assetID int64, in MaterialRenditionInput) (*SourcingMaterialRendition, error) {
	if in.ImageProcessingRecordID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	if _, err := s.requireMaterialContext(ownerID, sourceID, taskLinkID); err != nil {
		return nil, err
	}
	var rendition SourcingMaterialRendition
	err := s.db.Transaction(func(tx *gorm.DB) error {
		asset, err := s.loadOwnedMaterial(tx, ownerID, sourceID, taskLinkID, assetID, true)
		if err != nil {
			return err
		}
		if asset.MediaType != "image" {
			return fmt.Errorf("%w: no controlled video processor is registered", ErrWorkflowGate)
		}
		var record ImageProcessingRecord
		if err := tx.Where("id=? AND sourcing_product_id=? AND snapshot_id=? AND processed_by=?", in.ImageProcessingRecordID, sourceID, asset.SnapshotID, ownerID).First(&record).Error; err != nil {
			return fmt.Errorf("%w: image processing record mismatch", ErrWorkflowGate)
		}
		if record.SourceSHA256 != asset.SourceSHA256 {
			return fmt.Errorf("%w: rendition source hash mismatch", ErrWorkflowGate)
		}
		rendition = SourcingMaterialRendition{AssetID: assetID, OwnerID: ownerID, ImageProcessingRecordID: record.ID, CreatedBy: ownerID}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rendition).Error
	})
	return &rendition, err
}

func (s *Service) MarkMaterialUsed(ownerID, sourceID, taskLinkID, assetID int64) (*SourcingMaterialAsset, error) {
	link, err := s.requireMaterialContext(ownerID, sourceID, taskLinkID)
	if err != nil {
		return nil, err
	}
	var market demandCaseRow
	if err := s.db.Where("id = ? AND owner_id = ?", link.DemandCaseID, ownerID).First(&market).Error; err != nil || strings.TrimSpace(market.Region) == "" || strings.TrimSpace(market.SalesChannel) == "" {
		return nil, fmt.Errorf("%w: task market scope is incomplete", ErrWorkflowGate)
	}
	var out SourcingMaterialAsset
	err = s.db.Transaction(func(tx *gorm.DB) error {
		asset, err := s.loadOwnedMaterial(tx, ownerID, sourceID, taskLinkID, assetID, true)
		if err != nil {
			return err
		}
		if asset.Status != MaterialStatusActive || asset.MediaType == "video" {
			return fmt.Errorf("%w: blocked material cannot enter a draft", ErrWorkflowGate)
		}
		var rights SourcingMaterialRightsEvidence
		if err := tx.Where("asset_id=? AND owner_id=?", assetID, ownerID).Order("version DESC").First(&rights).Error; err != nil || rights.Status != "approved" || rights.ValidUntil != nil && !rights.ValidUntil.After(time.Now().UTC()) || !materialRightsCoverTask(rights, market) {
			return fmt.Errorf("%w: current approved rights evidence is required", ErrWorkflowGate)
		}
		var renditions int64
		if err := tx.Model(&SourcingMaterialRendition{}).Where("asset_id=? AND owner_id=?", assetID, ownerID).Count(&renditions).Error; err != nil {
			return err
		}
		if renditions == 0 {
			return fmt.Errorf("%w: a controlled static rendition is required", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		if err := tx.Model(asset).Updates(map[string]any{"used_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.First(&out, asset.ID).Error
	})
	return &out, err
}

func materialRightsCoverTask(rights SourcingMaterialRightsEvidence, market demandCaseRow) bool {
	if strings.TrimSpace(rights.LicenseScope) == "" {
		return false
	}
	var countries, channels []string
	if json.Unmarshal(rights.Countries, &countries) != nil || json.Unmarshal(rights.Channels, &channels) != nil {
		return false
	}
	contains := func(values []string, expected string) bool {
		for _, value := range values {
			if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(expected)) {
				return true
			}
		}
		return false
	}
	return contains(countries, market.Region) && contains(channels, market.SalesChannel)
}

func requireDraftMaterialsReady(tx *gorm.DB, ownerID, sourceID, taskLinkID int64) error {
	if !tx.Migrator().HasTable(&SourcingMaterialAsset{}) {
		return nil // isolated fixtures created before migration 000148
	}
	var assets []SourcingMaterialAsset
	if err := tx.Where("owner_id=? AND sourcing_product_id=? AND task_link_id=? AND status=?", ownerID, sourceID, taskLinkID, MaterialStatusActive).Find(&assets).Error; err != nil || len(assets) == 0 {
		return err
	}
	var market demandCaseRow
	if err := tx.Joins("JOIN sourcing_1688_task_link l ON l.demand_case_id = demand_case.id").Where("l.id=? AND l.owner_id=?", taskLinkID, ownerID).First(&market).Error; err != nil {
		return fmt.Errorf("%w: task market scope is unavailable", ErrWorkflowGate)
	}
	hasMain := false
	for _, asset := range assets {
		hasMain = hasMain || asset.Role == "main"
		if asset.UsedAt == nil || asset.MediaType != "image" {
			return fmt.Errorf("%w: every active material must be ready and marked used", ErrWorkflowGate)
		}
		var rights SourcingMaterialRightsEvidence
		if err := tx.Where("asset_id=? AND owner_id=?", asset.ID, ownerID).Order("version DESC").First(&rights).Error; err != nil || rights.Status != "approved" || rights.ValidUntil != nil && !rights.ValidUntil.After(time.Now().UTC()) || !materialRightsCoverTask(rights, market) {
			return fmt.Errorf("%w: current scoped material rights are required", ErrWorkflowGate)
		}
		var renditions int64
		if err := tx.Model(&SourcingMaterialRendition{}).Where("asset_id=? AND owner_id=?", asset.ID, ownerID).Count(&renditions).Error; err != nil || renditions == 0 {
			return fmt.Errorf("%w: controlled material rendition is required", ErrWorkflowGate)
		}
	}
	if !hasMain {
		return fmt.Errorf("%w: one ready main material is required", ErrWorkflowGate)
	}
	return nil
}
