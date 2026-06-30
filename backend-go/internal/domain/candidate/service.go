package candidate

import (
	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides candidate product business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new candidate service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// Create inserts a new candidate product.
func (s *Service) Create(in *CreateCandidateInput) (*CandidateProduct, error) {
	c := CandidateProduct{
		Title:              in.Title,
		Description:        in.Description,
		MainImage:          in.MainImage,
		Images:             in.Images,
		CategoryID:         in.CategoryID,
		BrandID:            in.BrandID,
		SpecJSON:           in.SpecJSON,
		SupplierID:         in.SupplierID,
		PurchaseCurrency:   in.PurchaseCurrency,
		HSCode:             in.HSCode,
		OriginCountry:      in.OriginCountry,
		TargetCurrency:     in.TargetCurrency,
		TargetPlatformID:   in.TargetPlatformID,
		DestinationCountry: in.DestinationCountry,
		CreatedBy:          in.CreatedBy,
	}
	if in.PurchasePrice != nil {
		c.PurchasePrice = *in.PurchasePrice
	}
	if in.PackageWeightKg != nil {
		c.PackageWeightKg = *in.PackageWeightKg
	}
	if in.PackageLengthCm != nil {
		c.PackageLengthCm = *in.PackageLengthCm
	}
	if in.PackageWidthCm != nil {
		c.PackageWidthCm = *in.PackageWidthCm
	}
	if in.PackageHeightCm != nil {
		c.PackageHeightCm = *in.PackageHeightCm
	}
	if in.TargetSalePrice != nil {
		c.TargetSalePrice = *in.TargetSalePrice
	}
	if in.PurchaseCurrency == "" {
		c.PurchaseCurrency = "CNY"
	}
	if in.TargetCurrency == "" {
		c.TargetCurrency = "USD"
	}
	if in.OriginCountry == "" {
		c.OriginCountry = "CN"
	}
	if in.DestinationCountry == "" {
		c.DestinationCountry = "US"
	}
	if in.Status != "" {
		c.Status = in.Status
	} else {
		c.Status = "draft"
	}
	if err := s.db.Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByID returns a single candidate product.
func (s *Service) GetByID(id int64) (*CandidateProduct, error) {
	var c CandidateProduct
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// List returns paginated candidate products with optional filters.
func (s *Service) List(p *common.Pagination, status, search string) ([]CandidateProduct, int64, error) {
	var items []CandidateProduct
	var total int64
	q := s.db.Model(&CandidateProduct{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("title ILIKE ? OR description ILIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Update patches a candidate product by id.
func (s *Service) Update(id int64, in *UpdateCandidateInput) (*CandidateProduct, error) {
	var c CandidateProduct
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.MainImage != nil {
		updates["main_image"] = *in.MainImage
	}
	if in.Images != nil {
		updates["images"] = *in.Images
	}
	if in.CategoryID != nil {
		updates["category_id"] = *in.CategoryID
	}
	if in.BrandID != nil {
		updates["brand_id"] = *in.BrandID
	}
	if in.SpecJSON != nil {
		updates["spec_json"] = *in.SpecJSON
	}
	if in.SupplierID != nil {
		updates["supplier_id"] = *in.SupplierID
	}
	if in.PurchasePrice != nil {
		updates["purchase_price"] = *in.PurchasePrice
	}
	if in.PurchaseCurrency != nil {
		updates["purchase_currency"] = *in.PurchaseCurrency
	}
	if in.PackageWeightKg != nil {
		updates["package_weight_kg"] = *in.PackageWeightKg
	}
	if in.PackageLengthCm != nil {
		updates["package_length_cm"] = *in.PackageLengthCm
	}
	if in.PackageWidthCm != nil {
		updates["package_width_cm"] = *in.PackageWidthCm
	}
	if in.PackageHeightCm != nil {
		updates["package_height_cm"] = *in.PackageHeightCm
	}
	if in.HSCode != nil {
		updates["hs_code"] = *in.HSCode
	}
	if in.OriginCountry != nil {
		updates["origin_country"] = *in.OriginCountry
	}
	if in.TargetSalePrice != nil {
		updates["target_sale_price"] = *in.TargetSalePrice
	}
	if in.TargetCurrency != nil {
		updates["target_currency"] = *in.TargetCurrency
	}
	if in.TargetPlatformID != nil {
		updates["target_platform_id"] = *in.TargetPlatformID
	}
	if in.DestinationCountry != nil {
		updates["destination_country"] = *in.DestinationCountry
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.UpdatedBy != nil {
		updates["updated_by"] = *in.UpdatedBy
	}
	if len(updates) == 0 {
		return &c, nil
	}
	if err := s.db.Model(&c).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Delete removes a candidate product by id.
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&CandidateProduct{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Count returns the total number of candidate products.
func (s *Service) Count() (int64, error) {
	var total int64
	if err := s.db.Model(&CandidateProduct{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
