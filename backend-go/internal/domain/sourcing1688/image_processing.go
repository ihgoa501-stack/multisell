package sourcing1688

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
	"gorm.io/gorm"
)

const maxSourceImageBytes = 10 << 20
const maxSourceImagePixels = 40_000_000
const imageProcessorVersion = "sourcing-image-v1"

type ImageProcessingRecord struct {
	ID                  int64           `gorm:"primaryKey" json:"id"`
	SourcingProductID   int64           `gorm:"not null;index" json:"sourcing_product_id"`
	SnapshotID          int64           `gorm:"not null;index" json:"snapshot_id"`
	SourceURL           string          `gorm:"type:text;not null" json:"source_url"`
	SourceSHA256        string          `gorm:"size:64;not null" json:"source_sha256"`
	ProcessedSHA256     string          `gorm:"size:64;not null" json:"processed_sha256"`
	OutputFormat        string          `gorm:"size:8;not null" json:"output_format"`
	OutputWidth         int             `gorm:"not null" json:"output_width"`
	OutputHeight        int             `gorm:"not null" json:"output_height"`
	Quality             int             `gorm:"not null" json:"quality"`
	ProcessorVersion    string          `gorm:"size:40;not null" json:"processor_version"`
	Operations          json.RawMessage `gorm:"type:jsonb;not null" json:"operations"`
	RightsEvidenceURI   string          `gorm:"type:text;not null" json:"rights_evidence_uri"`
	RightsTruthStatus   string          `gorm:"size:16;not null" json:"rights_truth_status"`
	RightsObservedAt    time.Time       `gorm:"not null" json:"rights_observed_at"`
	ChannelRuleURI      string          `gorm:"type:text;not null" json:"channel_rule_uri"`
	EvidenceFingerprint string          `gorm:"size:64;not null" json:"evidence_fingerprint"`
	ProcessedBytes      []byte          `gorm:"column:processed_bytes;not null" json:"-"`
	ProcessedBy         int64           `gorm:"not null" json:"processed_by"`
	CreatedAt           time.Time       `json:"created_at"`
}

func (ImageProcessingRecord) TableName() string { return "sourcing_1688_image_processing" }

type ProcessImageInput struct {
	SourcingProductID int64     `json:"sourcing_product_id" binding:"required"`
	SourceURL         string    `json:"source_url" binding:"required"`
	SourceBase64      string    `json:"source_base64" binding:"required"`
	Width             int       `json:"width" binding:"required,min=100,max=4000"`
	Height            int       `json:"height" binding:"required,min=100,max=4000"`
	Format            string    `json:"format" binding:"required,oneof=jpeg png"`
	Quality           int       `json:"quality" binding:"min=60,max=100"`
	RightsEvidenceURI string    `json:"rights_evidence_uri" binding:"required"`
	RightsTruthStatus string    `json:"rights_truth_status" binding:"required,eq=actual"`
	RightsObservedAt  time.Time `json:"rights_observed_at" binding:"required"`
	ChannelRuleURI    string    `json:"channel_rule_uri" binding:"required"`
	ProcessedBy       int64     `json:"processed_by"`
}

type ProcessImageResult struct {
	RecordID          int64           `json:"record_id"`
	ContentURL        string          `json:"content_url"`
	ProcessedSHA256   string          `json:"processed_sha256"`
	Width             int             `json:"width"`
	Height            int             `json:"height"`
	Format            string          `json:"format"`
	Operations        json.RawMessage `json:"operations"`
	RightsEvidenceURI string          `json:"rights_evidence_uri"`
	RightsObservedAt  time.Time       `json:"rights_observed_at"`
	ChannelRuleURI    string          `json:"channel_rule_uri"`
}

func centerCrop(src image.Image, targetWidth, targetHeight int) image.Image {
	b := src.Bounds()
	sourceRatio := float64(b.Dx()) / float64(b.Dy())
	targetRatio := float64(targetWidth) / float64(targetHeight)
	crop := b
	if sourceRatio > targetRatio {
		width := int(float64(b.Dy()) * targetRatio)
		x := b.Min.X + (b.Dx()-width)/2
		crop = image.Rect(x, b.Min.Y, x+width, b.Max.Y)
	} else if sourceRatio < targetRatio {
		height := int(float64(b.Dx()) / targetRatio)
		y := b.Min.Y + (b.Dy()-height)/2
		crop = image.Rect(b.Min.X, y, b.Max.X, y+height)
	}
	return subImage(src, crop)
}

func subImage(src image.Image, rect image.Rectangle) image.Image {
	if value, ok := src.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return value.SubImage(rect)
	}
	out := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(out, out.Bounds(), src, rect.Min, draw.Src)
	return out
}

func processImageBytes(source []byte, width, height int, format string, quality int) ([]byte, json.RawMessage, error) {
	if len(source) == 0 || len(source) > maxSourceImageBytes {
		return nil, nil, fmt.Errorf("%w: source image must be between 1 byte and 10 MiB", ErrInvalidWorkflow)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(source))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > 10_000 || config.Height > 10_000 || int64(config.Width)*int64(config.Height) > maxSourceImagePixels {
		return nil, nil, fmt.Errorf("%w: source image dimensions exceed safe limits", ErrInvalidWorkflow)
	}
	decoded, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: image decode failed", ErrInvalidWorkflow)
	}
	cropped := centerCrop(decoded, width, height)
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(canvas, canvas.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)
	var output bytes.Buffer
	switch format {
	case "jpeg":
		if quality == 0 {
			quality = 90
		}
		err = jpeg.Encode(&output, canvas, &jpeg.Options{Quality: quality})
	case "png":
		err = png.Encode(&output, canvas)
	default:
		return nil, nil, fmt.Errorf("%w: unsupported image format", ErrInvalidWorkflow)
	}
	if err != nil {
		return nil, nil, err
	}
	operations := json.RawMessage(fmt.Sprintf(`[{"operation":"center_crop"},{"operation":"resize","width":%d,"height":%d},{"operation":"white_background"}]`, width, height))
	return output.Bytes(), operations, nil
}

func jsonContainsString(raw json.RawMessage, target string) bool {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	var walk func(any) bool
	walk = func(current any) bool {
		switch typed := current.(type) {
		case string:
			return typed == target
		case []any:
			for _, child := range typed {
				if walk(child) {
					return true
				}
			}
		case map[string]any:
			for _, child := range typed {
				if walk(child) {
					return true
				}
			}
		}
		return false
	}
	return walk(value)
}

func (s *Service) ProcessImage(in *ProcessImageInput) (*ProcessImageResult, error) {
	if in == nil || in.ProcessedBy <= 0 || in.SourcingProductID <= 0 || strings.TrimSpace(in.SourceURL) == "" || strings.TrimSpace(in.RightsEvidenceURI) == "" || in.RightsTruthStatus != "actual" || in.RightsObservedAt.IsZero() || strings.TrimSpace(in.ChannelRuleURI) == "" {
		return nil, ErrInvalidWorkflow
	}
	source, err := base64.StdEncoding.DecodeString(in.SourceBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid source_base64", ErrInvalidWorkflow)
	}
	processed, operations, err := processImageBytes(source, in.Width, in.Height, in.Format, in.Quality)
	if err != nil {
		return nil, err
	}
	sourceHash, processedHash := sha256.Sum256(source), sha256.Sum256(processed)
	quality := in.Quality
	if in.Format == "png" {
		quality = 100
	} else if quality == 0 {
		quality = 90
	}
	evidenceRaw := fmt.Sprintf("%s|%s|%s|%s", in.RightsEvidenceURI, in.RightsTruthStatus, in.RightsObservedAt.UTC().Format(time.RFC3339Nano), in.ChannelRuleURI)
	evidenceHash := sha256.Sum256([]byte(evidenceRaw))
	var record ImageProcessingRecord
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var product Sourcing1688Product
		if err := tx.First(&product, in.SourcingProductID).Error; err != nil {
			return err
		}
		if product.SnapshotID == nil || product.DemandCaseID == nil || (product.LifecycleStatus != LifecycleReadyForProduct && product.LifecycleStatus != LifecycleEditing) {
			return fmt.Errorf("%w: Owner-reviewed source and snapshot required", ErrWorkflowGate)
		}
		var snapshot Sourcing1688Snapshot
		if err := tx.First(&snapshot, *product.SnapshotID).Error; err != nil {
			return err
		}
		if !jsonContainsString(snapshot.RawPayload, in.SourceURL) {
			return fmt.Errorf("%w: source image URL is not present in the reviewed snapshot", ErrWorkflowGate)
		}
		var dc demandCaseRow
		if err := tx.First(&dc, *product.DemandCaseID).Error; err != nil {
			return err
		}
		if dc.OwnerID != in.ProcessedBy {
			return fmt.Errorf("%w: image processing requires workflow Owner", ErrWorkflowGate)
		}
		record = ImageProcessingRecord{SourcingProductID: product.ID, SnapshotID: *product.SnapshotID, SourceURL: in.SourceURL, SourceSHA256: hex.EncodeToString(sourceHash[:]), ProcessedSHA256: hex.EncodeToString(processedHash[:]), OutputFormat: in.Format, OutputWidth: in.Width, OutputHeight: in.Height, Quality: quality, ProcessorVersion: imageProcessorVersion, Operations: operations, RightsEvidenceURI: in.RightsEvidenceURI, RightsTruthStatus: in.RightsTruthStatus, RightsObservedAt: in.RightsObservedAt, ChannelRuleURI: in.ChannelRuleURI, EvidenceFingerprint: hex.EncodeToString(evidenceHash[:]), ProcessedBytes: processed, ProcessedBy: in.ProcessedBy}
		return tx.Where("sourcing_product_id = ? AND snapshot_id = ? AND source_sha256 = ? AND output_width = ? AND output_height = ? AND output_format = ? AND quality = ? AND processor_version = ? AND evidence_fingerprint = ?", record.SourcingProductID, record.SnapshotID, record.SourceSHA256, record.OutputWidth, record.OutputHeight, record.OutputFormat, record.Quality, record.ProcessorVersion, record.EvidenceFingerprint).FirstOrCreate(&record).Error
	})
	if err != nil {
		return nil, err
	}
	return &ProcessImageResult{RecordID: record.ID, ContentURL: fmt.Sprintf("/api/v1/sourcing-1688/processed-images/%d/content", record.ID), ProcessedSHA256: record.ProcessedSHA256, Width: record.OutputWidth, Height: record.OutputHeight, Format: record.OutputFormat, Operations: record.Operations, RightsEvidenceURI: record.RightsEvidenceURI, RightsObservedAt: record.RightsObservedAt, ChannelRuleURI: record.ChannelRuleURI}, nil
}

func (s *Service) GetProcessedImage(id int64) (*ImageProcessingRecord, error) {
	var record ImageProcessingRecord
	if err := s.db.First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &record, nil
}
