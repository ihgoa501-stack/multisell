package candidate

import (
	"errors"
	"math/rand"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ErrDuplicateSourceURL is returned when a candidate with the same source_url already exists.
var ErrDuplicateSourceURL = errors.New("candidate with this source_url already exists")

// seedProducts defines the mock candidate products to seed.
var seedProducts = []struct {
	Title       string
	Description string
	PriceCNY    float64
	WeightKg    float64
	LengthCm    float64
	WidthCm     float64
	HeightCm    float64
	HSCode      string
	PlatformID  int64
	DestCountry string
	TargetPrice float64
	Status      string
}{
	{"智能蓝牙耳机 TWS Pro", "高品质蓝牙 5.3 无线耳机，支持主动降噪、IPX5 防水、30 小时续航。适用于运动、通勤和旅行。", 45.0, 0.15, 8, 6, 4, "851830", 1, "US", 18.99, "draft"},
	{"运动智能手环 Z9", "1.47 英寸 AMOLED 屏幕，心率血氧监测，IP68 防水，14 天续航。支持 Android/iOS。", 32.0, 0.08, 10, 4, 2, "910212", 1, "US", 12.99, "draft"},
	{"LED 触摸台灯", "三档亮度可调节，USB 充电，护眼无频闪，触控开关。适用于卧室、办公室。", 18.0, 0.35, 15, 12, 8, "940540", 1, "US", 8.99, "draft"},
	{"不锈钢保温杯 500ml", "真空双层 316 不锈钢，12 小时保温 8 小时保冷，食品级材质，防漏设计。", 22.0, 0.28, 22, 7, 7, "961700", 2, "RU", 9.99, "draft"},
	{"纯棉圆领短袖 T 恤", "100% 精梳棉，柔软透气，多色可选，尺码 S-3XL，适合日常穿着。", 15.0, 0.18, 25, 18, 2, "610910", 2, "RU", 6.99, "draft"},
	{"二合一无线充电座", "支持 iPhone/Android 15W 快充，带 LED 指示灯，过温过压保护。", 28.0, 0.12, 10, 8, 1.5, "850440", 1, "US", 14.99, "draft"},
	{"竹制切菜板套装", "天然竹子，防霉抗菌，带沥水架，轻便耐用。套装含 3 块不同尺寸。", 26.0, 0.65, 40, 25, 2, "441990", 3, "BR", 11.99, "draft"},
	{"便携式手持小风扇", "2000mAh 电池，三档风力，静音设计，可折叠。USB 充电，适合户外。", 9.0, 0.12, 8, 5, 12, "841451", 3, "BR", 5.99, "draft"},
	{"极简帆布背包", "复古风帆布材料，大容量主仓，加厚肩垫，带笔记本隔层。适合通勤和旅行。", 35.0, 0.3, 30, 15, 8, "420222", 1, "US", 16.99, "draft"},
	{"瑜伽垫 TPE 6mm", "环保 TPE 材料，双面防滑，高回弹减震。含背带和收纳绑带。", 20.0, 0.8, 180, 61, 0.6, "950691", 2, "RU", 8.99, "draft"},
	{"USB-C 扩展坞 7合1", "千兆网口 / HDMI 4K / USB 3.0 x3 / SD 读卡器 / PD 100W 充电。铝合金外壳。", 38.0, 0.08, 12, 5, 1.5, "847180", 1, "US", 19.99, "draft"},
	{"有机绿茶礼盒装", "精选明前龙井，250g 铁罐装。送礼自用两宜。产地浙江杭州。", 28.0, 0.35, 15, 12, 10, "090210", 3, "BR", 9.99, "draft"},
}

// Seed creates mock candidate products for demo/testing.
// Returns count of newly seeded products. Idempotent.
func (s *Service) Seed() (int, error) {
	var existing int64
	s.db.Model(&CandidateProduct{}).Where("is_seed_data = ?", true).Count(&existing)
	if existing > 0 {
		return 0, nil // already seeded
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	count := 0
	for _, sp := range seedProducts {
		// json.RawMessage for images — provide a minimal valid JSON array
		imagesJSON := `["https://via.placeholder.com/600x600.png?text=` + sp.Title + `"]`
		specJSON := `{"weight_g":"` + sp.Title + `","color":"Black"}`

		input := &CreateCandidateInput{
			Title:              sp.Title,
			Description:        sp.Description,
			MainImage:          "https://via.placeholder.com/800x800.png?text=" + sp.Title,
			Images:             []byte(imagesJSON),
			CategoryID:         ptrInt64(int64(rng.Intn(20) + 1)),
			BrandID:            ptrInt64(int64(rng.Intn(10) + 1)),
			SpecJSON:           []byte(specJSON),
			SupplierID:         ptrInt64(int64(rng.Intn(5) + 1)),
			PurchasePrice:      &sp.PriceCNY,
			PurchaseCurrency:   "CNY",
			PackageWeightKg:    &sp.WeightKg,
			PackageLengthCm:    &sp.LengthCm,
			PackageWidthCm:     &sp.WidthCm,
			PackageHeightCm:    &sp.HeightCm,
			HSCode:             sp.HSCode,
			OriginCountry:      "CN",
			TargetSalePrice:    &sp.TargetPrice,
			TargetCurrency:     "USD",
			TargetPlatformID:   &sp.PlatformID,
			DestinationCountry: sp.DestCountry,
			CreatedBy:          "system",
		}

		// Set a varied status so some products are incomplete
		input.Status = sp.Status

		c := CandidateProduct{
			Title:              input.Title,
			Description:        input.Description,
			MainImage:          input.MainImage,
			Images:             input.Images,
			CategoryID:         input.CategoryID,
			BrandID:            input.BrandID,
			SpecJSON:           input.SpecJSON,
			SupplierID:         input.SupplierID,
			PurchasePrice:      *input.PurchasePrice,
			PurchaseCurrency:   input.PurchaseCurrency,
			PackageWeightKg:    *input.PackageWeightKg,
			PackageLengthCm:    *input.PackageLengthCm,
			PackageWidthCm:     *input.PackageWidthCm,
			PackageHeightCm:    *input.PackageHeightCm,
			HSCode:             input.HSCode,
			OriginCountry:      input.OriginCountry,
			TargetSalePrice:    *input.TargetSalePrice,
			TargetCurrency:     input.TargetCurrency,
			TargetPlatformID:   input.TargetPlatformID,
			DestinationCountry: input.DestinationCountry,
			Status:             input.Status,
			IsSeedData:         true,
			CreatedBy:          input.CreatedBy,
		}
		if c.PurchaseCurrency == "" {
			c.PurchaseCurrency = "CNY"
		}
		if c.TargetCurrency == "" {
			c.TargetCurrency = "USD"
		}
		if c.OriginCountry == "" {
			c.OriginCountry = "CN"
		}
		if c.DestinationCountry == "" {
			c.DestinationCountry = "US"
		}
		if c.Status == "" {
			c.Status = "draft"
		}

		if err := s.db.Create(&c).Error; err != nil {
			s.logger.Warn("failed to seed candidate product", zap.String("title", sp.Title), zap.Error(err))
			continue
		}
		count++
	}
	return count, nil
}

func ptrInt64(n int64) *int64 { return &n }

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
// If source_url is provided, it checks for duplicates first.
// After creation, it computes and persists the completeness status.
func (s *Service) Create(in *CreateCandidateInput) (*CandidateProduct, error) {
	// Dedup check: if source_url is set, ensure it doesn't already exist
	if in.SourceURL != "" {
		var existing int64
		if err := s.db.Model(&CandidateProduct{}).
			Where("source_url = ?", in.SourceURL).
			Count(&existing).Error; err != nil {
			return nil, err
		}
		if existing > 0 {
			return nil, ErrDuplicateSourceURL
		}
	}

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
		SourceURL:          in.SourceURL,
		SourcePlatform:     in.SourcePlatform,
		RawPayload:         in.RawPayload,
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
	// Compute completeness before setting, but allow override via input
	if in.CompletenessStatus != "" {
		c.CompletenessStatus = in.CompletenessStatus
	} else {
		status, _ := computeCompleteness(&c)
		c.CompletenessStatus = status
	}
	if in.SourceURL != "" {
		now := time.Now()
		c.CollectedAt = &now
	}
	if err := s.db.Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateCollectLead saves a new CollectLead entry and returns its ID.
// CollectLead entries are list page leads (not full CandidateProducts).
func (s *Service) CreateCollectLead(lead *CollectLead) error {
	now := time.Now()
	lead.CollectedAt = &now
	return s.db.Create(lead).Error
}

// ListCollectLeads returns paginated collect leads, newest first.
// Read-only — no mutation.
func (s *Service) ListCollectLeads(p *common.Pagination, status string) ([]CollectLead, int64, error) {
	var items []CollectLead
	var total int64
	q := s.db.Model(&CollectLead{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if p == nil {
		p = &common.Pagination{Page: 1, Size: 20}
	}
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetCollectLeadByID returns a single collect lead.
func (s *Service) GetCollectLeadByID(id int64) (*CollectLead, error) {
	var lead CollectLead
	if err := s.db.First(&lead, id).Error; err != nil {
		return nil, err
	}
	return &lead, nil
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
func (s *Service) List(p *common.Pagination, status, search, sourcePlatform, completenessStatus string) ([]CandidateProduct, int64, error) {
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
	if sourcePlatform != "" {
		q = q.Where("source_platform = ?", sourcePlatform)
	}
	if completenessStatus != "" {
		q = q.Where("completeness_status = ?", completenessStatus)
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

// DedupResult represents a duplicate record group by source_url.
type DedupResult struct {
	SourceURL string `json:"source_url"`
	Count     int64  `json:"count"`
}

// Dedup returns records grouped by source_url that have duplicates,
// filtered by a minimum count threshold.
func (s *Service) Dedup(minDup int) ([]DedupResult, error) {
	if minDup < 2 {
		minDup = 2
	}
	var results []DedupResult
	if err := s.db.Model(&CandidateProduct{}).
		Select("source_url, COUNT(*) as count").
		Where("source_url != ''").
		Group("source_url").
		Having("COUNT(*) >= ?", minDup).
		Order("count DESC").
		Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// Count returns the total number of candidate products.
func (s *Service) Count() (int64, error) {
	var total int64
	if err := s.db.Model(&CandidateProduct{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
