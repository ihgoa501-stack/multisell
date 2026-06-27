package sourcing1688

import (
	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides sourcing1688 business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new sourcing1688 service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated sourcing1688 products with optional filter.
func (s *Service) List(p *common.Pagination, f *ListFilter) ([]Sourcing1688Product, int64, error) {
	q := s.db.Model(&Sourcing1688Product{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("supplier_name ILIKE ? OR source_url ILIKE ?", like, like)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.ProductID != nil {
			q = q.Where("product_id = ?", *f.ProductID)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Sourcing1688Product
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns a single sourcing1688 product.
func (s *Service) Get(id int64) (*Sourcing1688Product, error) {
	var p Sourcing1688Product
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Create inserts a new sourcing1688 product.
func (s *Service) Create(in *CreateInput) (*Sourcing1688Product, error) {
	status := in.Status
	if status == "" {
		status = "collected"
	}
	p := Sourcing1688Product{
		SourceURL:    in.SourceURL,
		Title:        in.Title,
		SupplierName: in.SupplierName,
		SupplierID:   in.SupplierID,
		ShopURL:      in.ShopURL,
		ShopLocation: in.ShopLocation,
		Description:  in.Description,
		Status:       status,
		RawData:      in.RawData,
	}
	if in.Price != nil {
		p.Price = in.Price
	}
	if in.MOQ != nil {
		p.MOQ = *in.MOQ
	} else {
		p.MOQ = 1
	}
	if err := s.db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Update applies partial updates to a sourcing1688 product.
func (s *Service) Update(id int64, in *UpdateInput) (*Sourcing1688Product, error) {
	var p Sourcing1688Product
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.SourceURL != nil {
		updates["source_url"] = *in.SourceURL
	}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Price != nil {
		updates["price"] = *in.Price
	}
	if in.MOQ != nil {
		updates["moq"] = *in.MOQ
	}
	if in.SupplierName != nil {
		updates["supplier_name"] = *in.SupplierName
	}
	if in.ShopURL != nil {
		updates["shop_url"] = *in.ShopURL
	}
	if in.ShopLocation != nil {
		updates["shop_location"] = *in.ShopLocation
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.ProductID != nil {
		updates["product_id"] = *in.ProductID
	}
	if in.SupplierID != nil {
		updates["supplier_id"] = *in.SupplierID
	}
	if in.CollectedBy != nil {
		updates["collected_by"] = *in.CollectedBy
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.RawData != nil {
		updates["raw_data"] = *in.RawData
	}
	if len(updates) == 0 {
		return &p, nil
	}
	if err := s.db.Model(&p).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Delete removes a sourcing1688 product by id.
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&Sourcing1688Product{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Import marks a sourcing1688 product as imported.
func (s *Service) Import(id int64, in *ImportInput) (*Sourcing1688Product, error) {
	var p Sourcing1688Product
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&p).Update("status", "imported").Error; err != nil {
		return nil, err
	}
	s.logger.Info("sourcing1688 product imported",
		zap.Int64("id", id),
		zap.String("imported_by", in.ImportedBy),
	)
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Reject marks a sourcing1688 product as rejected.
func (s *Service) Reject(id int64, in *RejectInput) (*Sourcing1688Product, error) {
	var p Sourcing1688Product
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&p).Update("status", "rejected").Error; err != nil {
		return nil, err
	}
	s.logger.Info("sourcing1688 product rejected",
		zap.Int64("id", id),
		zap.String("rejected_by", in.RejectedBy),
		zap.String("reason", in.Reason),
	)
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Summary returns aggregation by status.
func (s *Service) Summary() (*Summary, error) {
	var total int64
	if err := s.db.Model(&Sourcing1688Product{}).Count(&total).Error; err != nil {
		return nil, err
	}
	type statusCount struct {
		Status string
		Cnt    int64
	}
	var scs []statusCount
	if err := s.db.Model(&Sourcing1688Product{}).
		Select("status, COUNT(*) AS cnt").Group("status").Scan(&scs).Error; err != nil {
		return nil, err
	}
	byStatus := make(map[string]int64, len(scs))
	for _, sc := range scs {
		byStatus[sc.Status] = sc.Cnt
	}
	return &Summary{Total: total, ByStatus: byStatus}, nil
}
