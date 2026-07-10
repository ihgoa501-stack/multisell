package sourcing

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ToolBridge defines the interface for fetching product data from external sources.
// Concrete implementations (PluginDriver, PlaywrightDriver, API1688Driver) are
// injected at construction time. The Route method returns toolbridge.PageData,
// which is converted to sourcing.PageData internally via fromToolbridgePageData.
type ToolBridge interface {
	// Route dispatches a fetch request to the appropriate driver and returns
	// structured toolbridge.PageData. Returns an error if all drivers are unavailable.
	Route(ctx context.Context, url string) (*toolbridge.PageData, error)
}

// EventPublisher defines the interface for publishing events to the event bus.
type EventPublisher interface {
	Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error)
}

// Service provides sourcing business logic for the A8 Agent.
type Service struct {
	db              *gorm.DB
	logger          *zap.Logger
	bridge          ToolBridge
	events          EventPublisher
	bsrSource       MarketTrendSource
	keywordSource   MarketTrendSource
}

// NewService creates a new sourcing service.
// bridge, events, and trend sources are optional: if nil, their respective
// operations will return errors.
func NewService(db *gorm.DB, logger *zap.Logger, bridge ToolBridge, events EventPublisher, trendSources ...MarketTrendSource) *Service {
	svc := &Service{
		db:     db,
		logger: logger,
		bridge: bridge,
		events: events,
	}

	// Assign trend sources if provided.
	for _, src := range trendSources {
		if src == nil {
			continue
		}
		switch src.Name() {
		case "amazon_bsr":
			svc.bsrSource = src
		case "keyword_trends":
			svc.keywordSource = src
		}
	}

	return svc
}

// FetchProduct fetches product data from the given URL via the ToolBridge.
func (s *Service) FetchProduct(ctx context.Context, url string) (*PageData, error) {
	if s.bridge == nil {
		return nil, fmt.Errorf("sourcing: ToolBridge not configured")
	}
	tbData, err := s.bridge.Route(ctx, url)
	if err != nil {
		s.logger.Warn("fetch product failed", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("sourcing: fetch failed: %w", err)
	}
	return fromToolbridgePageData(tbData), nil
}

// fromToolbridgePageData converts a toolbridge.PageData to sourcing.PageData.
// The two types differ in naming (PriceCNY vs Price, WeightKg vs PackageWeight, etc.)
// so this function handles the field mapping explicitly.
func fromToolbridgePageData(tb *toolbridge.PageData) *PageData {
	if tb == nil {
		return nil
	}

	specVariants := make([]SpecVariant, len(tb.SpecVariants))
	for i, sv := range tb.SpecVariants {
		specVariants[i] = SpecVariant{
			Spec:  sv.Spec,
			Price: sv.Price,
			Stock: sv.Stock,
		}
	}

	imageFirst := ""
	if len(tb.Images) > 0 {
		imageFirst = tb.Images[0]
	}

	return &PageData{
		SourceURL:      tb.SourceURL,
		CollectedAt:    tb.CollectedAt,
		Driver:         tb.Driver,
		Title:          tb.Title,
		Price:          tb.PriceCNY,
		PriceMin:       tb.PriceMinCNY,
		PriceMax:       tb.PriceMaxCNY,
		Currency:       "CNY",
		MOQ:            tb.MOQ,
		Images:         tb.Images,
		ImageFirst:     imageFirst,
		SpecVariants:   specVariants,
		SupplierName:   tb.SupplierName,
		SupplierScore:  tb.SupplierScore,
		PackageWeight:  tb.WeightKg,
		PackageLength:  tb.PackageLengthCm,
		PackageWidth:   tb.PackageWidthCm,
		PackageHeight:  tb.PackageHeightCm,
		Description:    tb.Description,
		Attributes:     nil,
	}
}

// AnalyzePage evaluates page quality and returns a score 1-10.
// Simple heuristic scoring based on available fields.
func (s *Service) AnalyzePage(data *PageData) (int, string) {
	if data == nil {
		return 0, "no data"
	}

	score := 0
	reasons := []string{}

	// Title quality (max 3 points)
	if len(data.Title) > 0 {
		score += 1
	}
	if len(data.Title) > 20 {
		score += 1
	}
	if len(data.Title) > 50 {
		score += 1
	}

	// Price quality (max 3 points)
	if data.Price > 0 {
		score += 2
		reasons = append(reasons, "price_ok")
	}
	if data.PriceMin != nil || data.PriceMax != nil {
		score += 1
	}

	// Image quality (max 3 points)
	imgCount := len(data.Images)
	if data.ImageFirst != "" && imgCount == 0 {
		imgCount = 1
	}
	if imgCount > 0 {
		score += 1
		reasons = append(reasons, "has_images")
	}
	if imgCount >= 3 {
		score += 1
	}
	if imgCount >= 5 {
		score += 1
	}

	// Supplier presence (1 point)
	if data.SupplierName != "" {
		score += 1
		reasons = append(reasons, "has_supplier")
	}

	// Clamp to 1-10
	if score < 1 {
		score = 1
	}
	if score > 10 {
		score = 10
	}

	reason := "ok"
	if len(reasons) > 0 {
		reason = reasons[0]
	}
	if score <= 3 {
		reason = "low_quality"
	}

	return score, reason
}

// SaveRecommendation stores a scored recommendation in the sourcing1688 table
// and publishes a sourcing.recommend event.
func (s *Service) SaveRecommendation(ctx context.Context, data *PageData, score int, reason string) (*sourcing1688.Sourcing1688Product, error) {
	if data == nil {
		return nil, fmt.Errorf("sourcing: cannot save nil page data")
	}

	// Build source data JSON from PageData.
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("sourcing: marshal page data: %w", err)
	}
	rawData := json.RawMessage(raw)

	// Determine price (use PriceMin if set, otherwise main price).
	price := data.Price
	if data.PriceMin != nil && *data.PriceMin > 0 {
		price = *data.PriceMin
	}

	moq := data.MOQ
	if moq <= 0 {
		moq = 1
	}

	status := "pending"
	if score >= 7 {
		status = "recommended"
	} else if score <= 3 {
		status = "low_quality"
	}

	// Marshal images into JSON for the images JSONB column.
	var imagesJSON *json.RawMessage
	if len(data.Images) > 0 {
		imgBytes, _ := json.Marshal(data.Images)
		imgRaw := json.RawMessage(imgBytes)
		imagesJSON = &imgRaw
	}

	// Marshal spec variants into JSON for the sku_variants JSONB column.
	var skuVariantsJSON *json.RawMessage
	if len(data.SpecVariants) > 0 {
		svBytes, _ := json.Marshal(data.SpecVariants)
		svRaw := json.RawMessage(svBytes)
		skuVariantsJSON = &svRaw
	}

	// Store in sourcing1688 table with columns matching the migration schema.
	p1688 := sourcing1688.Sourcing1688Product{
		SourceURL:    data.SourceURL,
		Title:        &data.Title,
		SupplierName: data.SupplierName,
		Price:        &price,
		MOQ:          moq,
		Images:       imagesJSON,
		SkuVariants:  skuVariantsJSON,
		Status:       status,
		RawData:      &rawData,
	}

	if err := s.db.WithContext(ctx).Create(&p1688).Error; err != nil {
		return nil, fmt.Errorf("sourcing: save recommendation: %w", err)
	}

	// Publish sourcing.recommend event to trigger pipeline chain.
	if s.events != nil {
		payload := map[string]interface{}{
			"id":         p1688.ID,
			"source_url": data.SourceURL,
			"title":      data.Title,
			"price":      price,
			"score":      score,
			"reason":     reason,
			"supplier":   data.SupplierName,
		}
		if _, pubErr := s.events.Publish(ctx, "sourcing.recommend", "A8", payload); pubErr != nil {
			s.logger.Warn("publish sourcing.recommend failed", zap.Error(pubErr))
		}
	}

	s.logger.Info("sourcing recommendation saved",
		zap.Int64("product_id", p1688.ID),
		zap.String("url", data.SourceURL),
		zap.Int("score", score),
	)

	return &p1688, nil
}

// ListRecommendations returns paginated recommendations from the sourcing1688 table.
func (s *Service) ListRecommendations(page, size int) ([]Recommendation, int64, error) {
	var total int64
	q := s.db.Model(&sourcing1688.Sourcing1688Product{})

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var products []sourcing1688.Sourcing1688Product
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&products).Error; err != nil {
		return nil, 0, err
	}

	recs := make([]Recommendation, 0, len(products))
	for _, p := range products {
		price := 0.0
		if p.Price != nil {
			price = *p.Price
		}

		rec := Recommendation{
			ID:           p.ID,
			SourceURL:    p.SourceURL,
			SupplierName: p.SupplierName,
			Price:        price,
			Status:       p.Status,
			CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		}

		// Try to extract title and other fields from raw_data if available.
		if p.RawData != nil {
			var pd PageData
			if err := json.Unmarshal(*p.RawData, &pd); err == nil {
				rec.Title = pd.Title
				rec.ImageURL = pd.ImageFirst
				// If no separate score field, estimate from status.
				switch p.Status {
				case "recommended":
					rec.Score = 8
				case "low_quality":
					rec.Score = 2
				default:
					rec.Score = 5
				}
			}
		}

		recs = append(recs, rec)
	}

	return recs, total, nil
}

// FetchMarketTrends returns Amazon BSR trend data for the given product category.
// Returns an error if the BSR source is not configured.
func (s *Service) FetchMarketTrends(ctx context.Context, category string) ([]MarketTrendItem, error) {
	if s.bsrSource == nil {
		return nil, fmt.Errorf("sourcing: Amazon BSR source not configured")
	}
	return s.bsrSource.FetchTrends(ctx, category)
}

// FetchKeywordTrends returns keyword search volume and competition data for
// the given search keyword. Returns an error if the keyword source is not configured.
func (s *Service) FetchKeywordTrends(ctx context.Context, keyword string) ([]MarketTrendItem, error) {
	if s.keywordSource == nil {
		return nil, fmt.Errorf("sourcing: keyword trend source not configured")
	}
	return s.keywordSource.FetchTrends(ctx, keyword)
}

// CategorySummary aggregates market data for one product category.
type CategorySummary struct {
	Category     string  `json:"category"`
	ProductCount int     `json:"product_count"`
	AvgRank      float64 `json:"avg_rank"`
	AvgPriceMin  float64 `json:"avg_price_min"`
	AvgPriceMax  float64 `json:"avg_price_max"`
	AvgRating    float64 `json:"avg_rating"`
	TotalReviews int     `json:"total_reviews"`
	DemandScore  float64 `json:"demand_score"`
}

// FetchMarketOverview returns aggregated market demand data across all categories.
func (s *Service) FetchMarketOverview(ctx context.Context) ([]CategorySummary, error) {
	if s.bsrSource == nil {
		return nil, fmt.Errorf("sourcing: BSR source not configured")
	}
	items, err := s.bsrSource.FetchTrends(ctx, "")
	if err != nil {
		return nil, err
	}
	type agg struct {
		count       int
		rankSum     float64
		priceMinSum float64
		priceMaxSum float64
		ratingSum   float64
		reviews     int
	}
	groups := make(map[string]*agg)
	for _, it := range items {
		g, ok := groups[it.Category]
		if !ok {
			g = &agg{}
			groups[it.Category] = g
		}
		g.count++
		g.rankSum += float64(it.Rank)
		g.reviews += it.ReviewCount
		g.ratingSum += it.AvgRating
		// Parse price range "CNY XX-YY"
		parts := strings.Split(it.PriceRange, "-")
		if len(parts) == 2 {
			minP := parsePrice(parts[0])
			maxP := parsePrice(parts[1])
			g.priceMinSum += minP
			g.priceMaxSum += maxP
		}
	}
	var result []CategorySummary
	for cat, g := range groups {
		demandScore := (float64(g.reviews)/1000.0)*0.3 + (5.0-g.rankSum/float64(g.count))*0.3 + (g.ratingSum/float64(g.count))*0.2
		if demandScore > 10 {
			demandScore = 10
		}
		result = append(result, CategorySummary{
			Category:     cat,
			ProductCount: g.count,
			AvgRank:      round2(g.rankSum / float64(g.count)),
			AvgPriceMin:  round2(g.priceMinSum / float64(g.count)),
			AvgPriceMax:  round2(g.priceMaxSum / float64(g.count)),
			AvgRating:    round2(g.ratingSum / float64(g.count)),
			TotalReviews: g.reviews,
			DemandScore:  round2(demandScore),
		})
	}
	// Sort by demand score descending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].DemandScore > result[j].DemandScore
	})
	return result, nil
}

func parsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	var val float64
	for _, r := range s {
		if r >= '0' && r <= '9' || r == '.' {
			val = val*10 + float64(r-'0')
		}
	}
	return val
}
