package imagegen

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides imagegen business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new imagegen service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ===================== ProductImageGen =====================

// ListImageGens returns image-gen records, optionally filtered by product/batch/status.
func (s *Service) ListImageGens(productID int64, batchID, status string, page, size int) ([]ProductImageGen, int64, error) {
	q := s.db.Model(&ProductImageGen{})
	if productID > 0 {
		q = q.Where("product_id = ?", productID)
	}
	if batchID != "" {
		q = q.Where("batch_id = ?", batchID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var items []ProductImageGen
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetImageGen returns a single image-gen record.
func (s *Service) GetImageGen(id int64) (*ProductImageGen, error) {
	var g ProductImageGen
	if err := s.db.First(&g, id).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

// CreateImageGen inserts a new image-gen request (status defaults to pending).
func (s *Service) CreateImageGen(g *ProductImageGen) error {
	return s.db.Create(g).Error
}

// UpdateImageGenStatus updates the status and result fields of an image-gen record.
func (s *Service) UpdateImageGenStatus(id int64, status string, imageUrls []byte, errMsg string) error {
	return s.db.Model(&ProductImageGen{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       status,
			"image_urls":   imageUrls,
			"error_message": errMsg,
		}).Error
}

// DeleteImageGen removes an image-gen record.
func (s *Service) DeleteImageGen(id int64) error {
	return s.db.Delete(&ProductImageGen{}, id).Error
}

// ===================== ProductCanvas =====================

// ListCanvases returns canvases for a product.
func (s *Service) ListCanvases(productID int64, page, size int) ([]ProductCanvas, int64, error) {
	q := s.db.Model(&ProductCanvas{})
	if productID > 0 {
		q = q.Where("product_id = ?", productID)
	}
	var total int64
	q.Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var items []ProductCanvas
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetCanvas returns a single canvas.
func (s *Service) GetCanvas(id int64) (*ProductCanvas, error) {
	var cv ProductCanvas
	if err := s.db.First(&cv, id).Error; err != nil {
		return nil, err
	}
	return &cv, nil
}

// CreateCanvas inserts a new canvas.
func (s *Service) CreateCanvas(cv *ProductCanvas) error {
	return s.db.Create(cv).Error
}

// UpdateCanvas updates an existing canvas.
func (s *Service) UpdateCanvas(cv *ProductCanvas) error {
	return s.db.Save(cv).Error
}

// DeleteCanvas removes a canvas.
func (s *Service) DeleteCanvas(id int64) error {
	return s.db.Delete(&ProductCanvas{}, id).Error
}

// ===================== PromptTemplate =====================

// ListTemplates returns prompt templates, optionally filtered by style/platform/creator.
func (s *Service) ListTemplates(style, platformCode string, createdBy int64, page, size int) ([]PromptTemplate, int64, error) {
	q := s.db.Model(&PromptTemplate{})
	if style != "" {
		q = q.Where("style = ?", style)
	}
	if platformCode != "" {
		q = q.Where("platform_code = ?", platformCode)
	}
	if createdBy > 0 {
		q = q.Where("created_by = ?", createdBy)
	}
	var total int64
	q.Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var items []PromptTemplate
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetTemplate returns a single prompt template.
func (s *Service) GetTemplate(id int64) (*PromptTemplate, error) {
	var t PromptTemplate
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateTemplate inserts a new prompt template.
func (s *Service) CreateTemplate(t *PromptTemplate) error {
	return s.db.Create(t).Error
}

// UpdateTemplate updates an existing prompt template.
func (s *Service) UpdateTemplate(t *PromptTemplate) error {
	return s.db.Save(t).Error
}

// IncrementTemplateUsage bumps the usage_count for a template.
func (s *Service) IncrementTemplateUsage(id int64) error {
	return s.db.Model(&PromptTemplate{}).Where("id = ?", id).
		UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).Error
}

// DeleteTemplate removes a prompt template.
func (s *Service) DeleteTemplate(id int64) error {
	return s.db.Delete(&PromptTemplate{}, id).Error
}
