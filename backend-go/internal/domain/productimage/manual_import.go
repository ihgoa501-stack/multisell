package productimage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	ManualImportKind        = "manual_import"
	ChannelNativeImportKind = "channel_native_import"
)

var manualImportAmountPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$`)

// ManualImport is immutable provenance for bytes edited outside LingMirror.
// It is evidence of an import, not evidence of image rights or publishability.
type ManualImport struct {
	ID                 int64     `json:"id" gorm:"primaryKey"`
	OwnerID            int64     `json:"owner_id" gorm:"not null;uniqueIndex:uidx_product_image_manual_import_owner_idem;index:idx_product_image_manual_import_owner_created"`
	AssetID            int64     `json:"asset_id" gorm:"not null;index"`
	AssetSHA           string    `json:"asset_sha256" gorm:"size:64;not null"`
	ParentAssetID      int64     `json:"parent_asset_id" gorm:"not null;index"`
	ParentAssetSHA     string    `json:"parent_asset_sha256" gorm:"size:64;not null"`
	ImportKind         string    `json:"import_kind" gorm:"size:32;not null"`
	Tool               string    `json:"tool" gorm:"size:100;not null"`
	Operation          string    `json:"operation" gorm:"size:255;not null"`
	FeeAmount          string    `json:"fee_amount" gorm:"size:32;not null"`
	FeeCurrency        string    `json:"fee_currency" gorm:"size:3;not null"`
	Model              string    `json:"model" gorm:"size:100;not null"`
	ModelVersion       string    `json:"model_version" gorm:"size:100;not null"`
	OriginalChannel    string    `json:"original_channel,omitempty" gorm:"size:64"`
	ChannelRestriction string    `json:"channel_restriction" gorm:"size:64;not null"`
	SourceObservedAt   time.Time `json:"source_observed_at" gorm:"not null"`
	Truth              string    `json:"truth" gorm:"size:20;not null;default:unknown"`
	IdempotencyKey     string    `json:"idempotency_key" gorm:"size:100;not null;uniqueIndex:uidx_product_image_manual_import_owner_idem"`
	RequestHash        string    `json:"request_hash" gorm:"size:64;not null"`
	CreatedAt          time.Time `json:"created_at" gorm:"index:idx_product_image_manual_import_owner_created"`
}

func (ManualImport) TableName() string { return "product_image_manual_imports" }

type ManualImportInput struct {
	ParentAssetID      int64     `json:"parent_asset_id"`
	ParentAssetSHA     string    `json:"parent_asset_sha256"`
	ImportKind         string    `json:"import_kind"`
	Tool               string    `json:"tool"`
	Operation          string    `json:"operation"`
	FeeAmount          string    `json:"fee_amount"`
	FeeCurrency        string    `json:"fee_currency"`
	Model              string    `json:"model"`
	ModelVersion       string    `json:"model_version"`
	OriginalChannel    string    `json:"original_channel"`
	ChannelRestriction string    `json:"channel_restriction"`
	SourceObservedAt   time.Time `json:"source_observed_at"`
	IdempotencyKey     string    `json:"idempotency_key"`
}

func (s *Service) CreateManualImport(ctx context.Context, ownerID int64, in ManualImportInput, filename, contentType string, body []byte) (*ManualImport, error) {
	in.ParentAssetSHA = strings.ToLower(strings.TrimSpace(in.ParentAssetSHA))
	in.ImportKind, in.Tool, in.Operation = strings.TrimSpace(in.ImportKind), strings.TrimSpace(in.Tool), strings.TrimSpace(in.Operation)
	in.FeeAmount, in.FeeCurrency = strings.TrimSpace(in.FeeAmount), strings.ToUpper(strings.TrimSpace(in.FeeCurrency))
	in.Model, in.ModelVersion = strings.TrimSpace(in.Model), strings.TrimSpace(in.ModelVersion)
	in.OriginalChannel, in.ChannelRestriction = strings.TrimSpace(in.OriginalChannel), strings.TrimSpace(in.ChannelRestriction)
	in.IdempotencyKey, filename, contentType = strings.TrimSpace(in.IdempotencyKey), strings.TrimSpace(filename), strings.TrimSpace(contentType)
	if ownerID <= 0 || in.ParentAssetID <= 0 || !isSHA256(in.ParentAssetSHA) ||
		(in.ImportKind != ManualImportKind && in.ImportKind != ChannelNativeImportKind) ||
		in.Tool == "" || in.Operation == "" || !manualImportAmountPattern.MatchString(in.FeeAmount) ||
		!allowedExecutionCurrency(in.FeeCurrency) || in.Model == "" || in.ModelVersion == "" ||
		in.ChannelRestriction == "" || in.SourceObservedAt.IsZero() || in.SourceObservedAt.After(time.Now().UTC().Add(5*time.Minute)) ||
		in.IdempotencyKey == "" || len(in.IdempotencyKey) > 100 || filename == "" || len(body) == 0 || s.image == nil {
		return nil, ErrInvalidInput
	}
	if in.ImportKind == ChannelNativeImportKind && (in.OriginalChannel == "" || in.ChannelRestriction != in.OriginalChannel) {
		return nil, ErrInvalidInput
	}
	var parent Asset
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", in.ParentAssetID, ownerID).First(&parent).Error; err != nil {
		return nil, err
	}
	if parent.SHA256 != in.ParentAssetSHA {
		return nil, &ConflictError{Code: "PARENT_HASH_MISMATCH"}
	}
	inputDigest := sha256.Sum256(body)
	manifest, _ := json.Marshal(struct {
		Input   ManualImportInput `json:"input"`
		FileSHA string            `json:"file_sha256"`
	}{in, hex.EncodeToString(inputDigest[:])})
	requestDigest := sha256.Sum256(manifest)
	requestHash := hex.EncodeToString(requestDigest[:])
	var existing ManualImport
	if err := s.db.WithContext(ctx).Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; err == nil {
		if existing.RequestHash != requestHash {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	remote, err := s.image.PutBlob(ctx, contentType, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("upload manual import bytes: %w", err)
	}
	if !isSHA256(remote.BlobID) {
		return nil, errors.New("upload manual import bytes: Image Service returned an invalid content address")
	}
	parentID := parent.ID
	asset := &Asset{OwnerID: ownerID, BlobID: remote.BlobID, Filename: filename, ContentType: contentType, SizeBytes: int64(len(body)), SHA256: remote.BlobID, Truth: TruthUnknown, SourceKind: in.ImportKind, ParentAssetID: &parentID, ParentAssetSHA: parent.SHA256, ChannelRestriction: cleanScope(in.ChannelRestriction)}
	item := &ManualImport{OwnerID: ownerID, AssetSHA: remote.BlobID, ParentAssetID: parent.ID, ParentAssetSHA: parent.SHA256, ImportKind: in.ImportKind, Tool: in.Tool, Operation: in.Operation, FeeAmount: in.FeeAmount, FeeCurrency: in.FeeCurrency, Model: in.Model, ModelVersion: in.ModelVersion, OriginalChannel: in.OriginalChannel, ChannelRestriction: in.ChannelRestriction, SourceObservedAt: in.SourceObservedAt.UTC(), Truth: TruthUnknown, IdempotencyKey: in.IdempotencyKey, RequestHash: requestHash}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(asset).Error; err != nil {
			return err
		}
		item.AssetID = asset.ID
		return tx.Create(item).Error
	}); err != nil {
		return nil, fmt.Errorf("persist manual import: %w", err)
	}
	return item, nil
}

func (s *Service) ListManualImports(ctx context.Context, ownerID int64, page, size int) ([]ManualImport, int64, error) {
	if ownerID <= 0 {
		return nil, 0, ErrInvalidInput
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	q := s.db.WithContext(ctx).Model(&ManualImport{}).Where("owner_id = ?", ownerID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []ManualImport{}
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func isSHA256(v string) bool {
	if len(v) != 64 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil && v == strings.ToLower(v)
}
