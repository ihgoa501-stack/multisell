package producthub

import (
	"fmt"
	"math"
	"strings"

	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// discovered is the result type for auto-discovered relations.
type discovered struct {
	targetID     int64
	relationType string
	weight       float64
}

// RelationService handles product relationship graph operations.
type RelationService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewRelationService creates a new RelationService.
func NewRelationService(db *gorm.DB, logger *zap.Logger) *RelationService {
	return &RelationService{db: db, logger: logger}
}

// GetRelatedProducts returns all related products for a given product, grouped by type.
func (s *RelationService) GetRelatedProducts(productID int64) (*RelationListResponse, error) {
	// Query relations where product is either source or target (bidirectional).
	var relations []ProductRelation
	if err := s.db.Where("source_id = ? OR target_id = ?", productID, productID).
		Order("weight DESC").Find(&relations).Error; err != nil {
		return nil, fmt.Errorf("failed to query relations: %w", err)
	}

	// Collect related product IDs.
	type candidate struct {
		id             int64
		relationType   string
		weight         float64
		autoDiscovered bool
	}
	var candidates []candidate

	for _, r := range relations {
		var otherID int64
		if r.SourceID == productID {
			otherID = r.TargetID
		} else {
			otherID = r.SourceID
		}
		candidates = append(candidates, candidate{
			id:             otherID,
			relationType:   r.RelationType,
			weight:         r.Weight,
			autoDiscovered: r.AutoDiscovered,
		})
	}

	if len(candidates) == 0 {
		return &RelationListResponse{
			SourceID: productID,
			Groups:   []RelationGroup{},
		}, nil
	}

	// Batch load product names and images.
	ids := make([]int64, 0, len(candidates))
	seen := make(map[int64]int) // dedupe ids
	for _, c := range candidates {
		if _, exists := seen[c.id]; !exists {
			seen[c.id] = len(ids)
			ids = append(ids, c.id)
		}
	}

	var products []sku.Product
	if err := s.db.Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}

	productMap := make(map[int64]sku.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}

	// Group by relation type.
	typeMap := make(map[string][]RelatedProductResponse)
	typeOrder := []string{} // preserve insertion order
	for _, c := range candidates {
		p, ok := productMap[c.id]
		if !ok {
			continue // product may have been deleted
		}
		label := c.relationType
		if _, exists := typeMap[label]; !exists {
			typeOrder = append(typeOrder, label)
		}
		typeMap[label] = append(typeMap[label], RelatedProductResponse{
			ID:             p.ID,
			Name:           p.Name,
			MainImage:      p.MainImage,
			RelationType:   c.relationType,
			Weight:         c.weight,
			AutoDiscovered: c.autoDiscovered,
		})
	}

	groups := make([]RelationGroup, 0, len(typeOrder))
	for _, t := range typeOrder {
		label := RelationLabels[t]
		if label == "" {
			label = t
		}
		groups = append(groups, RelationGroup{
			RelationType: t,
			Label:        label,
			Items:        typeMap[t],
		})
	}

	return &RelationListResponse{
		SourceID: productID,
		Groups:   groups,
	}, nil
}

// CreateRelation manually creates a product relation. If it already exists (same
// source, target, and type), it updates the weight instead of creating a duplicate.
func (s *RelationService) CreateRelation(sourceID, targetID int64, relationType string, weight float64) (*ProductRelation, error) {
	if sourceID == targetID {
		return nil, fmt.Errorf("cannot create self-referencing relation")
	}

	// Look for existing relation (in either direction).
	var existing ProductRelation
	err := s.db.Where(
		"(source_id = ? AND target_id = ? AND relation_type = ?) OR (source_id = ? AND target_id = ? AND relation_type = ?)",
		sourceID, targetID, relationType, targetID, sourceID, relationType,
	).First(&existing).Error

	if err == nil {
		// Update existing relation weight.
		existing.Weight = weight
		existing.AutoDiscovered = false
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, fmt.Errorf("failed to update relation: %w", err)
		}
		return &existing, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing relation: %w", err)
	}

	// Create new relation.
	r := ProductRelation{
		SourceID:       sourceID,
		TargetID:       targetID,
		RelationType:   relationType,
		Weight:         weight,
		AutoDiscovered: false,
	}
	if r.Weight <= 0 {
		r.Weight = 0.5 // default weight for manual relations
	}

	if err := s.db.Create(&r).Error; err != nil {
		return nil, fmt.Errorf("failed to create relation: %w", err)
	}
	return &r, nil
}

// DeleteRelation removes a product relation by ID.
func (s *RelationService) DeleteRelation(id int64) error {
	result := s.db.Delete(&ProductRelation{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete relation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// AutoDiscoverRelations analyzes order data and product catalog to automatically
// discover relationships for a given product:
//   - cross_sell: frequently co-purchased products (same order)
//   - alternative: same category + similar name keywords
//   - variant: products with overlapping spec dimensions
func (s *RelationService) AutoDiscoverRelations(productID int64) (*RelationListResponse, error) {
	// Skip discovery if product doesn't exist.
	var p sku.Product
	if err := s.db.First(&p, productID).Error; err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// 1. Discover cross_sell from order data.
	crossSellProducts, err := s.discoverCrossSell(productID)
	if err != nil {
		s.logger.Warn("cross-sell discovery failed", zap.Int64("product_id", productID), zap.Error(err))
	}

	// 2. Discover alternatives from same category + similar name.
	alternatives, err := s.discoverAlternatives(productID, &p)
	if err != nil {
		s.logger.Warn("alternative discovery failed", zap.Int64("product_id", productID), zap.Error(err))
	}

	// 3. Discover variant candidates from products sharing spec dimensions.
	variants, err := s.discoverVariants(productID, &p)
	if err != nil {
		s.logger.Warn("variant discovery failed", zap.Int64("product_id", productID), zap.Error(err))
	}

	// Collect all discovered relations and persist them.
	all := make([]discovered, 0,
		len(crossSellProducts)+len(alternatives)+len(variants),
	)
	all = append(all, crossSellProducts...)
	all = append(all, alternatives...)
	all = append(all, variants...)

	for _, d := range all {
		// Upsert: check if relation already exists.
		var existing ProductRelation
		err := s.db.Where(
			"(source_id = ? AND target_id = ? AND relation_type = ?) OR (source_id = ? AND target_id = ? AND relation_type = ?)",
			productID, d.targetID, d.relationType, d.targetID, productID, d.relationType,
		).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			_ = s.db.Create(&ProductRelation{
				SourceID:       productID,
				TargetID:       d.targetID,
				RelationType:   d.relationType,
				Weight:         d.weight,
				AutoDiscovered: true,
			})
		} else if err == nil {
			// Update weight to max of existing and new.
			if d.weight > existing.Weight {
				s.db.Model(&existing).Update("weight", d.weight)
			}
		}
	}

	// Return the full relation list for this product.
	return s.GetRelatedProducts(productID)
}

// discoverCrossSell finds products that were frequently purchased together
// with the given product in the same order.
func (s *RelationService) discoverCrossSell(productID int64) ([]discovered, error) {
	type coPurchase struct {
		ProductID int64
		Count     int
	}

	// Find orders containing this product.
	var orderIDs []int64
	if err := s.db.Model(&order.OrderItem{}).
		Where("product_id = ?", productID).
		Pluck("DISTINCT order_id", &orderIDs).Error; err != nil {
		return nil, err
	}
	if len(orderIDs) == 0 {
		return nil, nil
	}

	// Find other products in those same orders.
	raw := s.db.Model(&order.OrderItem{}).
		Select("product_id, COUNT(*) as count").
		Where("order_id IN ? AND product_id != ?", orderIDs, productID).
		Group("product_id").
		Order("count DESC").
		Limit(10)

	var coPurchases []coPurchase
	if err := raw.Scan(&coPurchases).Error; err != nil {
		return nil, err
	}

	totalOrders := float64(len(orderIDs))
	result := make([]discovered, 0, len(coPurchases))
	for _, cp := range coPurchases {
		// Weight = co-purchase frequency (0.0-1.0).
		weight := math.Min(math.Round(float64(cp.Count)/totalOrders*100)/100, 1.0)
		if weight >= 0.1 {
			result = append(result, discovered{
				targetID:     cp.ProductID,
				relationType: "cross_sell",
				weight:       weight,
			})
		}
	}
	return result, nil
}

// discoverAlternatives finds products in the same category with similar names.
func (s *RelationService) discoverAlternatives(productID int64, p *sku.Product) ([]discovered, error) {
	if p.CategoryID == 0 {
		return nil, nil
	}

	// Extract meaningful keywords from product name (words >= 2 chars).
	keywords := extractKeywords(p.Name)
	if len(keywords) == 0 {
		return nil, nil
	}

	// Build ILIKE conditions for name similarity.
	var conditions []string
	var args []interface{}
	args = append(args, p.CategoryID, productID)
	for _, kw := range keywords {
		conditions = append(conditions, "name ILIKE ?")
		args = append(args, "%"+kw+"%")
	}
	likeClause := strings.Join(conditions, " OR ")

	type alternativeRow struct {
		ID   int64
		Name string
	}
	var rows []alternativeRow
	if err := s.db.Model(&sku.Product{}).
		Select("id, name").
		Where("category_id = ? AND id != ? AND ("+likeClause+")", args...).
		Limit(15).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]discovered, 0, len(rows))
	for _, r := range rows {
		// Score similarity by how many keywords match.
		matchCount := 0
		for _, kw := range keywords {
			if strings.Contains(strings.ToLower(r.Name), strings.ToLower(kw)) {
				matchCount++
			}
		}
		weight := math.Min(float64(matchCount)/float64(len(keywords)), 1.0)
		weight = math.Round(weight*100) / 100
		if weight >= 0.3 {
			result = append(result, discovered{
				targetID:     r.ID,
				relationType: "alternative",
				weight:       weight,
			})
		}
	}
	return result, nil
}

// discoverVariants finds products with overlapping spec dimensions
// (same package dimensions/weight), suggesting they are variants of the same product.
func (s *RelationService) discoverVariants(productID int64, p *sku.Product) ([]discovered, error) {
	// Only discover variants if the product has spec dimensions.
	hasDimensions := p.ProductLengthCm.IsPositive() || p.ProductWidthCm.IsPositive() ||
		p.ProductHeightCm.IsPositive() || p.PackageLengthCm.IsPositive()

	if !hasDimensions {
		return nil, nil
	}

	// Find products with similar dimensions (within 20% tolerance).
	// Use package dimensions as primary match signal.
	tolerance := 0.20
	var similar []sku.Product

	query := s.db.Model(&sku.Product{}).
		Where("id != ?", productID)

	if p.PackageLengthCm.IsPositive() {
		lower := p.PackageLengthCm.InexactFloat64() * (1 - tolerance)
		upper := p.PackageLengthCm.InexactFloat64() * (1 + tolerance)
		query = query.Where("package_length_cm BETWEEN ? AND ?", lower, upper)
	}
	if p.PackageWidthCm.IsPositive() {
		lower := p.PackageWidthCm.InexactFloat64() * (1 - tolerance)
		upper := p.PackageWidthCm.InexactFloat64() * (1 + tolerance)
		query = query.Where("package_width_cm BETWEEN ? AND ?", lower, upper)
	}
	if p.PackageWeightKg.IsPositive() {
		lower := p.PackageWeightKg.InexactFloat64() * (1 - tolerance)
		upper := p.PackageWeightKg.InexactFloat64() * (1 + tolerance)
		query = query.Where("package_weight_kg BETWEEN ? AND ?", lower, upper)
	}

	if err := query.Limit(10).Find(&similar).Error; err != nil {
		return nil, err
	}

	result := make([]discovered, 0, len(similar))
	for _, s := range similar {
		// Score based on how many dimensions match.
		score := 0.0
		count := 0

		if p.PackageLengthCm.IsPositive() && s.PackageLengthCm.IsPositive() {
			diff := math.Abs(p.PackageLengthCm.InexactFloat64()-s.PackageLengthCm.InexactFloat64()) / p.PackageLengthCm.InexactFloat64()
			if diff <= tolerance {
				score += 1 - diff/tolerance
			}
			count++
		}
		if p.PackageWidthCm.IsPositive() && s.PackageWidthCm.IsPositive() {
			diff := math.Abs(p.PackageWidthCm.InexactFloat64()-s.PackageWidthCm.InexactFloat64()) / p.PackageWidthCm.InexactFloat64()
			if diff <= tolerance {
				score += 1 - diff/tolerance
			}
			count++
		}
		if p.PackageWeightKg.IsPositive() && s.PackageWeightKg.IsPositive() {
			diff := math.Abs(p.PackageWeightKg.InexactFloat64()-s.PackageWeightKg.InexactFloat64()) / p.PackageWeightKg.InexactFloat64()
			if diff <= tolerance {
				score += 1 - diff/tolerance
			}
			count++
		}

		if count > 0 {
			weight := math.Round(score/float64(count)*100) / 100
			if weight >= 0.5 {
				result = append(result, discovered{
					targetID:     s.ID,
					relationType: "variant",
					weight:       weight,
				})
			}
		}
	}
	return result, nil
}

// extractKeywords splits a product name into meaningful keywords (>= 2 chars).
func extractKeywords(name string) []string {
	parts := strings.Fields(name)
	keywords := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, ".,!?()[]{}-\"'")
		if len(p) >= 2 {
			keywords = append(keywords, p)
		}
	}
	return keywords
}
