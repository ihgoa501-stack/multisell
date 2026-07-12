package productimage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	ImageSetDraft  = "draft"
	ImageSetFrozen = "frozen"
)

var (
	ErrImageSetInvalid   = errors.New("invalid image set")
	ErrImageSetFrozen    = errors.New("image set is frozen")
	ErrImageSetNotFrozen = errors.New("image set is not frozen")
)

// ImageSet is one immutable, Owner-selected version of the complete image
// arrangement for a listing, channel and locale. A changed image, role or
// order is represented by a new version; a frozen row is never edited.
type ImageSet struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	OwnerID      uint64         `gorm:"not null;uniqueIndex:ux_image_set_scope_version,priority:1" json:"owner_id"`
	ListingID    uint64         `gorm:"not null;uniqueIndex:ux_image_set_scope_version,priority:2;index" json:"listing_id"`
	Channel      string         `gorm:"size:64;not null;uniqueIndex:ux_image_set_scope_version,priority:3" json:"channel"`
	Locale       string         `gorm:"size:32;not null;uniqueIndex:ux_image_set_scope_version,priority:4" json:"locale"`
	Version      uint           `gorm:"not null;uniqueIndex:ux_image_set_scope_version,priority:5" json:"version"`
	BasedOnSetID *uint64        `gorm:"index" json:"based_on_set_id,omitempty"`
	Status       string         `gorm:"size:16;not null;index" json:"status"`
	ManifestSHA  string         `gorm:"size:64" json:"manifest_sha256,omitempty"`
	SelectedBy   *uint64        `json:"selected_by,omitempty"`
	SelectedAt   *time.Time     `json:"selected_at,omitempty"`
	FrozenAt     *time.Time     `json:"frozen_at,omitempty"`
	Items        []ImageSetItem `gorm:"foreignKey:ImageSetID;constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (ImageSet) TableName() string { return "product_image_sets" }

// ImageSetItem binds the exact immutable asset bytes to their listing role.
// Channel and locale are repeated deliberately so the approved manifest is
// self-contained and cannot silently inherit a changed parent default.
type ImageSetItem struct {
	ID                uint64    `gorm:"primaryKey" json:"id"`
	ImageSetID        uint64    `gorm:"not null;uniqueIndex:ux_image_set_item_ordinal,priority:1;uniqueIndex:ux_image_set_item_role_ordinal,priority:1;index" json:"image_set_id"`
	Role              string    `gorm:"size:32;not null;uniqueIndex:ux_image_set_item_role_ordinal,priority:2" json:"role"`
	Ordinal           uint      `gorm:"not null;uniqueIndex:ux_image_set_item_ordinal,priority:2;uniqueIndex:ux_image_set_item_role_ordinal,priority:3" json:"ordinal"`
	Locale            string    `gorm:"size:32;not null" json:"locale"`
	Channel           string    `gorm:"size:64;not null" json:"channel"`
	AssetSHA          string    `gorm:"size:64;not null" json:"asset_sha256"`
	TaskID            int64     `gorm:"not null;index" json:"task_id"`
	OutputBlobID      string    `gorm:"size:64;not null" json:"output_blob_id"`
	TaskManifestHash  string    `gorm:"size:64;not null" json:"task_manifest_hash"`
	Operation         string    `gorm:"size:64;not null" json:"operation"`
	Processor         string    `gorm:"size:64;not null" json:"processor"`
	ImageServiceJobID string    `gorm:"size:100;not null" json:"image_service_job_id"`
	CreatedAt         time.Time `json:"created_at"`
}

func (ImageSetItem) TableName() string { return "product_image_set_items" }

type ImageSetItemInput struct {
	Role              string `json:"role"`
	Ordinal           uint   `json:"ordinal"`
	Locale            string `json:"locale"`
	Channel           string `json:"channel"`
	AssetSHA          string `json:"asset_sha256"`
	TaskID            int64  `json:"task_id"`
	OutputBlobID      string `json:"output_blob_id"`
	TaskManifestHash  string `json:"task_manifest_hash"`
	Operation         string `json:"operation"`
	Processor         string `json:"processor"`
	ImageServiceJobID string `json:"image_service_job_id"`
}

type CreateImageSetInput struct {
	OwnerID   uint64              `json:"owner_id"`
	ListingID uint64              `json:"listing_id"`
	Channel   string              `json:"channel"`
	Locale    string              `json:"locale"`
	Items     []ImageSetItemInput `json:"items"`
}

// ImageSetService owns set versioning and freezing. The mutex prevents two
// goroutines in one process from allocating the same next version; the unique
// database index remains the final cross-process guard.
type ImageSetService struct {
	db *gorm.DB
	mu sync.Mutex
}

func NewImageSetService(db *gorm.DB) *ImageSetService { return &ImageSetService{db: db} }

func ImageSetModels() []any { return []any{&ImageSet{}, &ImageSetItem{}} }

func (s *ImageSetService) CreateDraft(ctx context.Context, in CreateImageSetInput) (*ImageSet, error) {
	return s.createVersion(ctx, in, nil)
}

// Revise creates a draft successor. The base must be a frozen set owned by the
// same Owner. It never mutates or unfreezes the approved version.
func (s *ImageSetService) Revise(ctx context.Context, ownerID, baseID uint64, items []ImageSetItemInput) (*ImageSet, error) {
	var base ImageSet
	if err := s.db.WithContext(ctx).First(&base, "id = ? AND owner_id = ?", baseID, ownerID).Error; err != nil {
		return nil, err
	}
	if base.Status != ImageSetFrozen {
		return nil, ErrImageSetNotFrozen
	}
	return s.createVersion(ctx, CreateImageSetInput{
		OwnerID: ownerID, ListingID: base.ListingID, Channel: base.Channel,
		Locale: base.Locale, Items: items,
	}, &baseID)
}

func (s *ImageSetService) createVersion(ctx context.Context, in CreateImageSetInput, basedOn *uint64) (*ImageSet, error) {
	clean, err := normalizeImageSetInput(in)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var created ImageSet
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var latest uint
		if err := tx.Model(&ImageSet{}).
			Where("owner_id = ? AND listing_id = ? AND channel = ? AND locale = ?", clean.OwnerID, clean.ListingID, clean.Channel, clean.Locale).
			Select("COALESCE(MAX(version), 0)").Scan(&latest).Error; err != nil {
			return err
		}
		created = ImageSet{
			OwnerID: clean.OwnerID, ListingID: clean.ListingID, Channel: clean.Channel,
			Locale: clean.Locale, Version: latest + 1, BasedOnSetID: basedOn, Status: ImageSetDraft,
		}
		created.Items = make([]ImageSetItem, len(clean.Items))
		for i, item := range clean.Items {
			created.Items[i] = ImageSetItem{Role: item.Role, Ordinal: item.Ordinal, Locale: item.Locale, Channel: item.Channel, AssetSHA: item.AssetSHA, TaskID: item.TaskID, OutputBlobID: item.OutputBlobID, TaskManifestHash: item.TaskManifestHash, Operation: item.Operation, Processor: item.Processor, ImageServiceJobID: item.ImageServiceJobID}
		}
		return tx.Create(&created).Error
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// SelectAndFreeze records the Owner decision and freezes the canonical bytes.
// Replaying the same Owner selection is idempotent.
func (s *ImageSetService) SelectAndFreeze(ctx context.Context, ownerID, setID uint64) (*ImageSet, error) {
	var result ImageSet
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("ordinal ASC, id ASC") }).
			First(&result, "id = ? AND owner_id = ?", setID, ownerID).Error; err != nil {
			return err
		}
		if result.Status == ImageSetFrozen {
			if result.SelectedBy != nil && *result.SelectedBy == ownerID && result.ManifestSHA != "" {
				return nil
			}
			return ErrImageSetFrozen
		}
		if err := validateImageSet(result.Channel, result.Locale, result.Items); err != nil {
			return err
		}
		manifest, err := CanonicalImageSetManifest(result)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		updates := map[string]any{
			"status": ImageSetFrozen, "manifest_sha": manifest,
			"selected_by": ownerID, "selected_at": now, "frozen_at": now,
		}
		res := tx.Model(&ImageSet{}).Where("id = ? AND owner_id = ? AND status = ?", setID, ownerID, ImageSetDraft).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrImageSetFrozen
		}
		result.Status, result.ManifestSHA = ImageSetFrozen, manifest
		result.SelectedBy, result.SelectedAt, result.FrozenAt = &ownerID, &now, &now
		return nil
	})
	return &result, err
}

func (s *ImageSetService) Get(ctx context.Context, ownerID, setID uint64) (*ImageSet, error) {
	var set ImageSet
	err := s.db.WithContext(ctx).Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("ordinal ASC, id ASC") }).
		First(&set, "id = ? AND owner_id = ?", setID, ownerID).Error
	return &set, err
}

type canonicalImageSet struct {
	OwnerID   uint64                  `json:"owner_id"`
	ListingID uint64                  `json:"listing_id"`
	Channel   string                  `json:"channel"`
	Locale    string                  `json:"locale"`
	Version   uint                    `json:"version"`
	Items     []canonicalImageSetItem `json:"items"`
}

type canonicalImageSetItem struct {
	Role              string `json:"role"`
	Ordinal           uint   `json:"ordinal"`
	Locale            string `json:"locale"`
	Channel           string `json:"channel"`
	AssetSHA          string `json:"asset_sha256"`
	TaskID            int64  `json:"task_id"`
	OutputBlobID      string `json:"output_blob_id"`
	TaskManifestHash  string `json:"task_manifest_hash"`
	Operation         string `json:"operation"`
	Processor         string `json:"processor"`
	ImageServiceJobID string `json:"image_service_job_id"`
}

// CanonicalImageSetManifest hashes only frozen business fields, never row IDs,
// timestamps or JSON map order. Item input order therefore cannot change it.
func CanonicalImageSetManifest(set ImageSet) (string, error) {
	items := append([]ImageSetItem(nil), set.Items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Ordinal != items[j].Ordinal {
			return items[i].Ordinal < items[j].Ordinal
		}
		if items[i].Role != items[j].Role {
			return items[i].Role < items[j].Role
		}
		return items[i].AssetSHA < items[j].AssetSHA
	})
	manifest := canonicalImageSet{OwnerID: set.OwnerID, ListingID: set.ListingID, Channel: set.Channel, Locale: set.Locale, Version: set.Version, Items: make([]canonicalImageSetItem, len(items))}
	for i, item := range items {
		manifest.Items[i] = canonicalImageSetItem{Role: item.Role, Ordinal: item.Ordinal, Locale: item.Locale, Channel: item.Channel, AssetSHA: item.AssetSHA, TaskID: item.TaskID, OutputBlobID: item.OutputBlobID, TaskManifestHash: item.TaskManifestHash, Operation: item.Operation, Processor: item.Processor, ImageServiceJobID: item.ImageServiceJobID}
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

var allowedImageRoles = map[string]struct{}{
	"main": {}, "gallery": {}, "detail": {}, "size": {}, "packaging": {}, "ad_cover": {},
}

func normalizeImageSetInput(in CreateImageSetInput) (CreateImageSetInput, error) {
	in.Channel, in.Locale = strings.ToLower(strings.TrimSpace(in.Channel)), strings.TrimSpace(in.Locale)
	if in.OwnerID == 0 || in.ListingID == 0 || in.Channel == "" || in.Locale == "" {
		return in, fmt.Errorf("%w: owner, listing, channel and locale are required", ErrImageSetInvalid)
	}
	items := make([]ImageSetItem, len(in.Items))
	for i := range in.Items {
		in.Items[i].Role = strings.ToLower(strings.TrimSpace(in.Items[i].Role))
		in.Items[i].Channel = strings.ToLower(strings.TrimSpace(in.Items[i].Channel))
		in.Items[i].Locale = strings.TrimSpace(in.Items[i].Locale)
		in.Items[i].AssetSHA = strings.ToLower(strings.TrimSpace(in.Items[i].AssetSHA))
		in.Items[i].OutputBlobID = strings.ToLower(strings.TrimSpace(in.Items[i].OutputBlobID))
		in.Items[i].TaskManifestHash = strings.ToLower(strings.TrimSpace(in.Items[i].TaskManifestHash))
		in.Items[i].Operation = strings.TrimSpace(in.Items[i].Operation)
		in.Items[i].Processor = strings.TrimSpace(in.Items[i].Processor)
		in.Items[i].ImageServiceJobID = strings.TrimSpace(in.Items[i].ImageServiceJobID)
		items[i] = ImageSetItem{Role: in.Items[i].Role, Ordinal: in.Items[i].Ordinal, Locale: in.Items[i].Locale, Channel: in.Items[i].Channel, AssetSHA: in.Items[i].AssetSHA, TaskID: in.Items[i].TaskID, OutputBlobID: in.Items[i].OutputBlobID, TaskManifestHash: in.Items[i].TaskManifestHash, Operation: in.Items[i].Operation, Processor: in.Items[i].Processor, ImageServiceJobID: in.Items[i].ImageServiceJobID}
	}
	if err := validateImageSet(in.Channel, in.Locale, items); err != nil {
		return in, err
	}
	return in, nil
}

func validateImageSet(channel, locale string, items []ImageSetItem) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: at least one item is required", ErrImageSetInvalid)
	}
	ordinals := make(map[uint]struct{}, len(items))
	roleOrdinals := make(map[string]map[uint]struct{})
	mainCount, adCoverCount := 0, 0
	for _, item := range items {
		if _, ok := allowedImageRoles[item.Role]; !ok {
			return fmt.Errorf("%w: unsupported role %q", ErrImageSetInvalid, item.Role)
		}
		if item.Ordinal == 0 {
			return fmt.Errorf("%w: ordinal starts at 1", ErrImageSetInvalid)
		}
		if _, exists := ordinals[item.Ordinal]; exists {
			return fmt.Errorf("%w: duplicate ordinal %d", ErrImageSetInvalid, item.Ordinal)
		}
		ordinals[item.Ordinal] = struct{}{}
		if roleOrdinals[item.Role] == nil {
			roleOrdinals[item.Role] = map[uint]struct{}{}
		}
		if _, exists := roleOrdinals[item.Role][item.Ordinal]; exists {
			return fmt.Errorf("%w: duplicate role/ordinal", ErrImageSetInvalid)
		}
		roleOrdinals[item.Role][item.Ordinal] = struct{}{}
		if item.Channel != channel || item.Locale != locale {
			return fmt.Errorf("%w: item scope differs from set", ErrImageSetInvalid)
		}
		if len(item.AssetSHA) != 64 {
			return fmt.Errorf("%w: asset SHA-256 must be 64 hex characters", ErrImageSetInvalid)
		}
		if item.TaskID <= 0 || item.OutputBlobID != item.AssetSHA || len(item.TaskManifestHash) != 64 || item.Operation == "" || item.Processor == "" || item.ImageServiceJobID == "" {
			return fmt.Errorf("%w: complete task and processor lineage is required", ErrImageSetInvalid)
		}
		if _, err := hex.DecodeString(item.TaskManifestHash); err != nil {
			return fmt.Errorf("%w: task manifest SHA-256 is not hex", ErrImageSetInvalid)
		}
		if _, err := hex.DecodeString(item.AssetSHA); err != nil {
			return fmt.Errorf("%w: asset SHA-256 is not hex", ErrImageSetInvalid)
		}
		if item.Role == "main" {
			mainCount++
		}
		if item.Role == "ad_cover" {
			adCoverCount++
		}
	}
	for ordinal := uint(1); ordinal <= uint(len(items)); ordinal++ {
		if _, ok := ordinals[ordinal]; !ok {
			return fmt.Errorf("%w: ordinals must be contiguous", ErrImageSetInvalid)
		}
	}
	if mainCount != 1 {
		return fmt.Errorf("%w: exactly one main image is required", ErrImageSetInvalid)
	}
	if adCoverCount > 1 {
		return fmt.Errorf("%w: at most one ad_cover image is allowed", ErrImageSetInvalid)
	}
	return nil
}
