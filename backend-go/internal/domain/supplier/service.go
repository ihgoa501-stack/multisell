package supplier

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides supplier business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new supplier service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ── Supplier ──────────────────────────────────────────────────────

// List returns a paginated list of suppliers with optional search.
func (s *Service) List(ctx context.Context, page, size int, search string) ([]Supplier, int64, error) {
	var items []Supplier
	var total int64

	q := s.db.WithContext(ctx).Model(&Supplier{})
	if search != "" {
		q = q.Where("name ILIKE ? OR contact_person ILIKE ? OR contact_phone ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListAll returns all enabled suppliers (for dropdown selectors).
func (s *Service) ListAll(ctx context.Context) ([]Supplier, error) {
	var items []Supplier
	if err := s.db.WithContext(ctx).Where("status = 1").Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetByID retrieves a single supplier by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*Supplier, error) {
	var sup Supplier
	if err := s.db.WithContext(ctx).First(&sup, id).Error; err != nil {
		return nil, err
	}
	return &sup, nil
}

// Create inserts a new supplier.
func (s *Service) Create(ctx context.Context, sup *Supplier) error {
	sup.Name = strings.TrimSpace(sup.Name)
	if sup.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(sup).Error
}

// Update saves changes to an existing supplier.
func (s *Service) Update(ctx context.Context, sup *Supplier) error {
	sup.Name = strings.TrimSpace(sup.Name)
	if sup.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(sup).Error
}

// Delete removes a supplier by ID (hard delete).
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&Supplier{}, id).Error
}

// ── ProductSupplier ──────────────────────────────────────────────

// ListProductSuppliers returns product-supplier associations for a product.
func (s *Service) ListProductSuppliers(ctx context.Context, productID int64) ([]ProductSupplier, error) {
	var items []ProductSupplier
	q := s.db.WithContext(ctx).Model(&ProductSupplier{})
	if productID > 0 {
		q = q.Where("product_id = ?", productID)
	}
	if err := q.Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CreateProductSupplier inserts a new product-supplier association.
func (s *Service) CreateProductSupplier(ctx context.Context, ps *ProductSupplier) error {
	return s.db.WithContext(ctx).Create(ps).Error
}

// UpdateProductSupplier saves changes to an existing product-supplier association.
func (s *Service) UpdateProductSupplier(ctx context.Context, ps *ProductSupplier) error {
	return s.db.WithContext(ctx).Save(ps).Error
}

// DeleteProductSupplier removes a product-supplier association by ID.
func (s *Service) DeleteProductSupplier(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&ProductSupplier{}, id).Error
}

// GetSupplierComparison returns a product's suppliers side-by-side.
func (s *Service) GetSupplierComparison(ctx context.Context, productID int64) (*SupplierComparisonResponse, error) {
	type productName struct {
		ID   int64  `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	var p productName
	if err := s.db.WithContext(ctx).Table("product").Select("id,name").First(&p, productID).Error; err != nil {
		return nil, err
	}

	type row struct {
		SupplierID    int64            `gorm:"column:supplier_id"`
		SupplierName  string           `gorm:"column:supplier_name"`
		SupplyPrice   *decimal.Decimal `gorm:"column:supply_price"`
		MinOrderQty   int              `gorm:"column:min_order_qty"`
		SpecSummary   string           `gorm:"column:spec_summary"`
	}
	var rows []row
	q := s.db.WithContext(ctx).Table("product_supplier ps").
		Select("ps.supplier_id, s.name AS supplier_name, ps.supply_price, ps.min_order_qty").
		Joins("JOIN supplier s ON s.id = ps.supplier_id").
		Where("ps.product_id = ?", productID).
		Order("ps.id ASC")

	// left join sourcing_1688_product for spec summary
	q = q.Joins("LEFT JOIN sourcing_1688_product sp ON sp.product_id = ps.product_id AND sp.supplier_id_1688::bigint = ps.supplier_id")

	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}

	suppliers := make([]SupplierRow, 0, len(rows))
	for _, r := range rows {
		suppliers = append(suppliers, SupplierRow{
			SupplierID:   r.SupplierID,
			SupplierName: r.SupplierName,
			SupplyPrice:  r.SupplyPrice,
			MinOrderQty:  r.MinOrderQty,
			SpecSummary:  r.SpecSummary,
		})
	}

	return &SupplierComparisonResponse{
		ProductID:   p.ID,
		ProductName: p.Name,
		Suppliers:   suppliers,
	}, nil
}
