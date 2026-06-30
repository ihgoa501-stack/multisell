package producthub

import (
	"encoding/json"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/sku"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VersionService handles product version snapshot management.
type VersionService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewVersionService creates a new VersionService.
func NewVersionService(db *gorm.DB, logger *zap.Logger) *VersionService {
	return &VersionService{db: db, logger: logger}
}

// CreateVersion records a new ProductVersion with the provided snapshot data.
func (s *VersionService) CreateVersion(productID int64, agentID, reason string, snapshot json.RawMessage, versionData json.RawMessage) (*ProductVersion, error) {
	if snapshot == nil {
		// Auto-capture snapshot from product table if not provided.
		var p sku.Product
		if err := s.db.First(&p, productID).Error; err != nil {
			return nil, err
		}
		raw, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		snapshot = raw
	}
	if versionData == nil {
		versionData = snapshot
	}

	v := ProductVersion{
		ProductID:   productID,
		VersionData: versionData,
		Snapshot:    snapshot,
		AgentID:     agentID,
		Reason:      reason,
	}
	if err := s.db.Create(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVersions returns paginated versions for a product.
func (s *VersionService) ListVersions(productID int64, page, size int) ([]ProductVersion, int64, error) {
	var total int64
	q := s.db.Model(&ProductVersion{}).Where("product_id = ?", productID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}
	var items []ProductVersion
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetVersion returns a single version by id.
func (s *VersionService) GetVersion(id int64) (*ProductVersion, error) {
	var v ProductVersion
	if err := s.db.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// Rollback restores a product's key fields from a snapshot.
func (s *VersionService) Rollback(productID, versionID int64) (*sku.Product, error) {
	// Load the version record.
	var v ProductVersion
	if err := s.db.First(&v, versionID).Error; err != nil {
		return nil, err
	}
	if v.ProductID != productID {
		return nil, gorm.ErrRecordNotFound
	}

	// Parse snapshot into product fields.
	var snapshot sku.Product
	if err := json.Unmarshal(v.Snapshot, &snapshot); err != nil {
		return nil, err
	}

	// Before rolling back, record the current state as a new version.
	currentSnapshot, err := json.Marshal(snapshot)
	if err == nil {
		rollbackVersion := ProductVersion{
			ProductID:   productID,
			VersionData: v.VersionData,
			Snapshot:    currentSnapshot,
			AgentID:     "system",
			Reason:      "rollback: pre-rollback snapshot before restoring version " + v.CreatedAt.Format(time.RFC3339),
		}
		_ = s.db.Create(&rollbackVersion)
	}

	// Restore product fields that are safe to rollback.
	updates := map[string]interface{}{
		"name":              snapshot.Name,
		"subtitle":          snapshot.Subtitle,
		"description":       snapshot.Description,
		"brand_id":          snapshot.BrandID,
		"category_id":       snapshot.CategoryID,
		"unit":              snapshot.Unit,
		"status":            snapshot.Status,
		"main_image":        snapshot.MainImage,
		"images":            snapshot.Images,
		"product_length_cm": snapshot.ProductLengthCm,
		"product_width_cm":  snapshot.ProductWidthCm,
		"product_height_cm": snapshot.ProductHeightCm,
		"product_weight_kg": snapshot.ProductWeightKg,
		"package_length_cm": snapshot.PackageLengthCm,
		"package_width_cm":  snapshot.PackageWidthCm,
		"package_height_cm": snapshot.PackageHeightCm,
		"package_weight_kg": snapshot.PackageWeightKg,
		"cargo_type":        snapshot.CargoType,
		"ai_title":          snapshot.AiTitle,
		"ai_description":    snapshot.AiDescription,
		"seo_keywords":      snapshot.SeoKeywords,
		"ai_status":         snapshot.AiStatus,
		"platform_statuses": snapshot.PlatformStatuses,
	}

	if err := s.db.Model(&sku.Product{}).Where("id = ?", productID).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Return the restored product.
	var restored sku.Product
	if err := s.db.First(&restored, productID).Error; err != nil {
		return nil, err
	}
	return &restored, nil
}
