package supplier

import (
	"context"
	"strings"
	"time"

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

// ── Supplier Score ─────────────────────────────────────────────────

// purchaseOrderRow is a minimal projection of purchase_order for scoring.
type purchaseOrderRow struct {
	ID               int64
	Status           string
	ExpectedDelivery *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// purchaseOrderItemRow is a minimal projection of purchase_order_item for scoring.
type purchaseOrderItemRow struct {
	PurchaseOrderID int64
	Quantity        int
	ReceivedQty     int
}

// GetScore returns the current supplier score.
func (s *Service) GetScore(ctx context.Context, supplierID int64) (*SupplierScore, error) {
	var score SupplierScore
	if err := s.db.WithContext(ctx).Where("supplier_id = ?", supplierID).First(&score).Error; err != nil {
		return nil, err
	}
	return &score, nil
}

// RecalculateScore computes all sub-scores from purchase order history and upserts the result.
func (s *Service) RecalculateScore(ctx context.Context, supplierID int64) (*SupplierScore, error) {
	now := time.Now()

	// 1. Fetch all purchase orders for this supplier
	var orders []purchaseOrderRow
	if err := s.db.WithContext(ctx).Table("purchase_order").
		Where("supplier_id = ?", supplierID).
		Order("created_at ASC").
		Find(&orders).Error; err != nil {
		return nil, err
	}

	totalOrders := len(orders)
	if totalOrders == 0 {
		score := &SupplierScore{
			SupplierID: supplierID,
			DataFreshness: 0,
		}
		return s.upsertScore(ctx, score)
	}

	// 2. Fetch all items for these orders
	orderIDs := make([]int64, 0, totalOrders)
	var lastOrderDate time.Time
	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
		if o.CreatedAt.After(lastOrderDate) {
			lastOrderDate = o.CreatedAt
		}
	}

	var items []purchaseOrderItemRow
	if err := s.db.WithContext(ctx).Table("purchase_order_item").
		Where("purchase_order_id IN ?", orderIDs).
		Find(&items).Error; err != nil {
		return nil, err
	}

	// 3. Calculate on-time delivery rate
	var completedCount int
	var onTimeCount int
	var totalLeadTimeDays float64
	var completedWithLeadTime int

	for _, o := range orders {
		if o.Status == "completed" || o.Status == "partially_received" {
			completedCount++

			// On-time: expected_delivery exists and completed (updated_at) <= expected
			if o.ExpectedDelivery != nil && *o.ExpectedDelivery != "" {
				expectedDate, err := time.Parse("2006-01-02", *o.ExpectedDelivery)
				if err == nil {
					if !o.UpdatedAt.After(expectedDate) {
						onTimeCount++
					}
				}
			}

			// Lead time days for completed orders
			leadDays := o.UpdatedAt.Sub(o.CreatedAt).Hours() / 24
			if leadDays > 0 {
				totalLeadTimeDays += leadDays
				completedWithLeadTime++
			}
		}
	}

	var onTimeDeliveryRate float64
	if completedCount > 0 {
		onTimeDeliveryRate = float64(onTimeCount) / float64(completedCount) * 100
	}

	// 4. Calculate order fulfillment %
	itemFulfillmentSum := 0.0
	itemCount := 0
	itemByOrder := make(map[int64][]purchaseOrderItemRow)
	for _, it := range items {
		itemByOrder[it.PurchaseOrderID] = append(itemByOrder[it.PurchaseOrderID], it)
	}
	// Consider only orders with items
	for _, oid := range orderIDs {
		its, ok := itemByOrder[oid]
		if !ok || len(its) == 0 {
			continue
		}
		var totalQty, totalReceived int
		for _, it := range its {
			totalQty += it.Quantity
			totalReceived += it.ReceivedQty
		}
		if totalQty > 0 {
			itemFulfillmentSum += float64(totalReceived) / float64(totalQty) * 100
			itemCount++
		}
	}

	var orderFulfillmentPct float64
	if itemCount > 0 {
		orderFulfillmentPct = itemFulfillmentSum / float64(itemCount)
	}

	// 5. Average lead time days
	var avgLeadTimeDays float64
	if completedWithLeadTime > 0 {
		avgLeadTimeDays = totalLeadTimeDays / float64(completedWithLeadTime)
	}

	// 6. Composite reliability score (weighted average)
	// Weights: on_time=35%, fulfillment=25%, lead_time=20%, quality=10%, communication=10%
	qualityPassRate := 0.0       // No quality data from purchase orders — defaults to 0
	communicationScore := 0.0    // No communication tracking — defaults to 0

	// Lead time score: inverse — shorter is better, cap at 100
	// Map avg_lead_time_days to a score: 30 days → 0, 0 days → 100
	var leadTimeScore float64
	if avgLeadTimeDays <= 0 {
		leadTimeScore = 100
	} else if avgLeadTimeDays >= 30 {
		leadTimeScore = 0
	} else {
		leadTimeScore = (1 - avgLeadTimeDays/30) * 100
	}

	reliabilityScore := onTimeDeliveryRate*0.35 +
		orderFulfillmentPct*0.25 +
		leadTimeScore*0.20 +
		qualityPassRate*0.10 +
		communicationScore*0.10

	// 7. Data freshness (days since last order)
	dataFreshness := int(now.Sub(lastOrderDate).Hours() / 24)

	score := &SupplierScore{
		SupplierID:          supplierID,
		OnTimeDeliveryRate:  onTimeDeliveryRate,
		QualityPassRate:     qualityPassRate,
		CommunicationScore:  communicationScore,
		OrderFulfillmentPct: orderFulfillmentPct,
		AvgLeadTimeDays:     avgLeadTimeDays,
		ReliabilityScore:    reliabilityScore,
		DataFreshness:       dataFreshness,
		LastOrderDate:       &lastOrderDate,
	}

	return s.upsertScore(ctx, score)
}

func (s *Service) upsertScore(ctx context.Context, score *SupplierScore) (*SupplierScore, error) {
	var existing SupplierScore
	result := s.db.WithContext(ctx).Where("supplier_id = ?", score.SupplierID).First(&existing)
	if result.Error != nil {
		// Insert new
		if err := s.db.WithContext(ctx).Create(score).Error; err != nil {
			return nil, err
		}
	} else {
		// Update existing
		score.ID = existing.ID
		score.CreatedAt = existing.CreatedAt
		if err := s.db.WithContext(ctx).Save(score).Error; err != nil {
			return nil, err
		}
	}
	return score, nil
}

// ListScoreboard returns all supplier scores ordered by reliability_score descending.
func (s *Service) ListScoreboard(ctx context.Context) ([]SupplierScore, error) {
	var scores []SupplierScore
	if err := s.db.WithContext(ctx).
		Order("reliability_score DESC, on_time_delivery_rate DESC").
		Find(&scores).Error; err != nil {
		return nil, err
	}
	return scores, nil
}

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

// ── Score History (#197) ───────────────────────────────────────────────

// GetScoreHistory returns the score history for a supplier, ordered by date desc.
func (s *Service) GetScoreHistory(ctx context.Context, supplierID int64, limit int) ([]SupplierScoreHistory, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var items []SupplierScoreHistory
	if err := s.db.WithContext(ctx).
		Where("supplier_id = ?", supplierID).
		Order("created_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// UpdateScoreManual manually sets a supplier's KPI score and logs a history entry.
func (s *Service) UpdateScoreManual(ctx context.Context, supplierID int64, kpiScore float64) (*Supplier, error) {
	sup, err := s.GetByID(ctx, supplierID)
	if err != nil {
		return nil, err
	}

	sup.KpiScore = kpiScore
	if err := s.db.WithContext(ctx).Save(sup).Error; err != nil {
		return nil, err
	}

	// Log a history entry with trigger="manual".
	history := &SupplierScoreHistory{
		SupplierID: supplierID,
		Trigger:    "manual",
	}
	if err := s.db.WithContext(ctx).Create(history).Error; err != nil {
		s.logger.Warn("failed to create score history entry", zap.Error(err))
	}

	return sup, nil
}

// RecordScoreSnapshot creates a score history snapshot from the current score.
func (s *Service) RecordScoreSnapshot(ctx context.Context, supplierID int64) error {
	score, err := s.GetScore(ctx, supplierID)
	if err != nil {
		return err
	}
	history := &SupplierScoreHistory{
		SupplierID:          supplierID,
		OnTimeDeliveryRate:  score.OnTimeDeliveryRate,
		QualityPassRate:     score.QualityPassRate,
		CommunicationScore:  score.CommunicationScore,
		OrderFulfillmentPct: score.OrderFulfillmentPct,
		AvgLeadTimeDays:     score.AvgLeadTimeDays,
		ReliabilityScore:    score.ReliabilityScore,
		Trigger:             "auto",
	}
	return s.db.WithContext(ctx).Create(history).Error
}
