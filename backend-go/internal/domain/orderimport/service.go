package orderimport

import (
	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides orderimport business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new orderimport service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated order imports with optional filter.
func (s *Service) List(p *common.Pagination, f *ListFilter) ([]OrderImport, int64, error) {
	q := s.db.Model(&OrderImport{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("file_name ILIKE ?", like)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.PlatformID != nil {
			q = q.Where("platform_id = ?", *f.PlatformID)
		}
		if f.SourceType != "" {
			q = q.Where("source_type = ?", f.SourceType)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []OrderImport
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns a single order import.
func (s *Service) Get(id int64) (*OrderImport, error) {
	var o OrderImport
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Create inserts a new order import.
func (s *Service) Create(in *CreateInput) (*OrderImport, error) {
	status := in.Status
	if status == "" {
		status = "pending"
	}
	sourceType := in.SourceType
	if sourceType == "" {
		sourceType = "manual"
	}
	o := OrderImport{
		PlatformID: in.PlatformID,
		SourceType: sourceType,
		FileName:   in.FileName,
		Status:     status,
		CreatedBy:  in.CreatedBy,
	}
	if in.TotalRows != nil {
		o.TotalRows = *in.TotalRows
	}
	if err := s.db.Create(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Update applies partial updates to an order import.
func (s *Service) Update(id int64, in *UpdateInput) (*OrderImport, error) {
	var o OrderImport
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.PlatformID != nil {
		updates["platform_id"] = *in.PlatformID
	}
	if in.SourceType != nil {
		updates["source_type"] = *in.SourceType
	}
	if in.FileName != nil {
		updates["file_name"] = *in.FileName
	}
	if in.TotalRows != nil {
		updates["total_rows"] = *in.TotalRows
	}
	if in.SuccessCount != nil {
		updates["success_count"] = *in.SuccessCount
	}
	if in.ErrorCount != nil {
		updates["error_count"] = *in.ErrorCount
	}
	if in.ErrorDetail != nil {
		updates["error_detail"] = *in.ErrorDetail
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if len(updates) == 0 {
		return &o, nil
	}
	if err := s.db.Model(&o).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Delete removes an order import by id.
func (s *Service) Delete(id int64) error {
	res := s.db.Delete(&OrderImport{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Process marks an order import as processing.
func (s *Service) Process(id int64) (*OrderImport, error) {
	var o OrderImport
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&o).Update("status", "processing").Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Complete finalizes an order import with success/error counts and status.
func (s *Service) Complete(id int64, in *CompleteInput) (*OrderImport, error) {
	var o OrderImport
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = "completed"
	}
	updates := map[string]interface{}{
		"success_count": in.SuccessCount,
		"error_count":   in.ErrorCount,
		"status":        status,
	}
	if len(in.ErrorDetail) > 0 {
		updates["error_detail"] = in.ErrorDetail
	}
	if err := s.db.Model(&o).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// Summary returns aggregation by status and total/success/error row counts.
func (s *Service) Summary() (*Summary, error) {
	var total int64
	if err := s.db.Model(&OrderImport{}).Count(&total).Error; err != nil {
		return nil, err
	}
	type statusCount struct {
		Status string
		Cnt    int64
	}
	var scs []statusCount
	if err := s.db.Model(&OrderImport{}).
		Select("status, COUNT(*) AS cnt").Group("status").Scan(&scs).Error; err != nil {
		return nil, err
	}
	byStatus := make(map[string]int64, len(scs))
	for _, sc := range scs {
		byStatus[sc.Status] = sc.Cnt
	}
	var agg struct {
		TotalRows    int64
		SuccessCount int64
		ErrorCount   int64
	}
	if err := s.db.Model(&OrderImport{}).
		Select("COALESCE(SUM(total_rows),0) AS total_rows, COALESCE(SUM(success_count),0) AS success_count, COALESCE(SUM(error_count),0) AS error_count").
		Scan(&agg).Error; err != nil {
		return nil, err
	}
	return &Summary{
		Total:        total,
		ByStatus:     byStatus,
		TotalRows:    agg.TotalRows,
		SuccessCount: agg.SuccessCount,
		ErrorCount:   agg.ErrorCount,
	}, nil
}
