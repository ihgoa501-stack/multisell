package sourcing1688

import (
	"encoding/json"
	"fmt"

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

func (s *Service) RequireSourceOwner(id, ownerID int64) error {
	if ownerID <= 0 {
		return ErrWorkflowGate
	}
	var count int64
	err := s.db.Table("sourcing_1688_product AS sp").
		Joins("LEFT JOIN demand_case dc ON dc.id = sp.demand_case_id").
		Where("sp.id = ? AND (sp.owner_id = ? OR (sp.owner_id = 0 AND dc.owner_id = ?))", id, ownerID, ownerID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("%w: source does not belong to authenticated Owner", ErrWorkflowGate)
	}
	return nil
}

func (s *Service) RequireExperimentOwner(experimentID string, ownerID int64) error {
	var count int64
	err := s.db.Model(&experimentRow{}).Where("experiment_id = ? AND owner_id = ?", experimentID, ownerID).Count(&count).Error
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("%w: experiment does not belong to authenticated Owner", ErrWorkflowGate)
	}
	return nil
}

// List returns paginated sourcing1688 products with optional filter.
func (s *Service) List(p *common.Pagination, f *ListFilter) ([]Sourcing1688Product, int64, error) {
	q := s.db.Model(&Sourcing1688Product{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("LOWER(supplier_name) LIKE LOWER(?) OR LOWER(source_url) LIKE LOWER(?)", like, like)
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

// ListOwned is the HTTP-facing list. It prevents an authenticated non-owner
// account from seeing another Owner's sourcing evidence.
func (s *Service) ListOwned(ownerID int64, p *common.Pagination, f *ListFilter) ([]Sourcing1688Product, int64, error) {
	if ownerID <= 0 {
		return nil, 0, ErrWorkflowGate
	}
	q := s.db.Model(&Sourcing1688Product{}).
		Joins("LEFT JOIN demand_case dc ON dc.id = sourcing_1688_product.demand_case_id").
		Where("sourcing_1688_product.owner_id = ? OR (sourcing_1688_product.owner_id = 0 AND dc.owner_id = ?)", ownerID, ownerID)
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("LOWER(COALESCE(sourcing_1688_product.title, '')) LIKE LOWER(?) OR LOWER(sourcing_1688_product.supplier_name) LIKE LOWER(?) OR LOWER(sourcing_1688_product.source_url) LIKE LOWER(?)", like, like, like)
		}
		if f.Status != "" {
			q = q.Where("sourcing_1688_product.status = ?", f.Status)
		}
		if f.LifecycleStatus != "" {
			q = q.Where("sourcing_1688_product.lifecycle_status = ?", f.LifecycleStatus)
		}
		if f.ProductID != nil {
			q = q.Where("sourcing_1688_product.product_id = ?", *f.ProductID)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Sourcing1688Product
	if err := q.Order("sourcing_1688_product.id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListPrivateCollectionBox enriches the ordinary Owner-scoped product list
// with read-only collection metadata. Field statuses come from the immutable
// snapshot referenced by the product; they are never inferred from zero values.
func (s *Service) ListPrivateCollectionBox(ownerID int64, p *common.Pagination, f *ListFilter) ([]PrivateCollectionListItem, int64, error) {
	products, total, err := s.ListOwned(ownerID, p, f)
	if err != nil || len(products) == 0 {
		return nil, total, err
	}
	ids := make([]int64, 0, len(products))
	for _, product := range products {
		ids = append(ids, product.ID)
	}
	type countRow struct {
		SourcingProductID int64
		Count             int64
	}
	var observationCounts, taskCounts []countRow
	if err := s.db.Model(&Sourcing1688Snapshot{}).Select("sourcing_product_id, COUNT(*) AS count").Where("sourcing_product_id IN ?", ids).Group("sourcing_product_id").Scan(&observationCounts).Error; err != nil {
		return nil, 0, err
	}
	if err := s.db.Model(&Sourcing1688TaskLink{}).Select("sourcing_product_id, COUNT(*) AS count").Where("owner_id = ? AND sourcing_product_id IN ?", ownerID, ids).Group("sourcing_product_id").Scan(&taskCounts).Error; err != nil {
		return nil, 0, err
	}
	observationByProduct, taskByProduct := map[int64]int64{}, map[int64]int64{}
	for _, row := range observationCounts {
		observationByProduct[row.SourcingProductID] = row.Count
	}
	for _, row := range taskCounts {
		taskByProduct[row.SourcingProductID] = row.Count
	}

	snapshotIDs := make([]int64, 0, len(products))
	for _, product := range products {
		if product.SnapshotID != nil {
			snapshotIDs = append(snapshotIDs, *product.SnapshotID)
		}
	}
	var snapshots []Sourcing1688Snapshot
	if len(snapshotIDs) > 0 {
		if err := s.db.Where("id IN ?", snapshotIDs).Find(&snapshots).Error; err != nil {
			return nil, 0, err
		}
	}
	snapshotByID := make(map[int64]Sourcing1688Snapshot, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotByID[snapshot.ID] = snapshot
	}

	items := make([]PrivateCollectionListItem, 0, len(products))
	for _, product := range products {
		statuses := map[string]string{}
		if product.SnapshotID != nil {
			if snapshot, ok := snapshotByID[*product.SnapshotID]; ok {
				var payload struct {
					FieldStatuses map[string]string `json:"field_statuses"`
				}
				if json.Unmarshal(snapshot.RawPayload, &payload) == nil && payload.FieldStatuses != nil {
					statuses = payload.FieldStatuses
				}
			}
		}
		items = append(items, PrivateCollectionListItem{Sourcing1688Product: product, FieldStatuses: statuses, ObservationCount: observationByProduct[product.ID], TaskLinkCount: taskByProduct[product.ID]})
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
		CollectedBy:  in.CollectedBy,
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

func (s *Service) SummaryOwned(ownerID int64) (*Summary, error) {
	if ownerID <= 0 {
		return nil, ErrWorkflowGate
	}
	base := s.db.Model(&Sourcing1688Product{}).
		Joins("LEFT JOIN demand_case dc ON dc.id = sourcing_1688_product.demand_case_id").
		Where("sourcing_1688_product.owner_id = ? OR (sourcing_1688_product.owner_id = 0 AND dc.owner_id = ?)", ownerID, ownerID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}
	type statusCount struct {
		Status string
		Cnt    int64
	}
	var scs []statusCount
	if err := base.Select("sourcing_1688_product.status, COUNT(*) AS cnt").
		Group("sourcing_1688_product.status").Scan(&scs).Error; err != nil {
		return nil, err
	}
	byStatus := make(map[string]int64, len(scs))
	for _, sc := range scs {
		byStatus[sc.Status] = sc.Cnt
	}
	return &Summary{Total: total, ByStatus: byStatus}, nil
}
