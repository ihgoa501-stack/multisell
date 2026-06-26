package sourcing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ToolBridge defines the interface for fetching product data from external sources.
// Concrete implementations (PluginDriver, PlaywrightDriver, API1688Driver) are
// injected at construction time to avoid a direct dependency on the toolbridge package.
type ToolBridge interface {
	// Route dispatches a fetch request to the appropriate driver and returns
	// structured PageData. Returns an error if all drivers are unavailable.
	Route(ctx context.Context, url string) (*PageData, error)
}

// EventPublisher defines the interface for publishing events to the event bus.
type EventPublisher interface {
	Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error)
}

// Service provides sourcing business logic for the A8 Agent.
type Service struct {
	db      *gorm.DB
	logger  *zap.Logger
	bridge  ToolBridge
	events  EventPublisher
}

// NewService creates a new sourcing service.
// bridge and events are optional: if nil, FetchProduct and event publishing
// will return errors.
func NewService(db *gorm.DB, logger *zap.Logger, bridge ToolBridge, events EventPublisher) *Service {
	return &Service{
		db:     db,
		logger: logger,
		bridge: bridge,
		events: events,
	}
}

// FetchProduct fetches product data from the given URL via the ToolBridge.
func (s *Service) FetchProduct(ctx context.Context, url string) (*PageData, error) {
	if s.bridge == nil {
		return nil, fmt.Errorf("sourcing: ToolBridge not configured")
	}
	pageData, err := s.bridge.Route(ctx, url)
	if err != nil {
		s.logger.Warn("fetch product failed", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("sourcing: fetch failed: %w", err)
	}
	return pageData, nil
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

	// Determine the main image URL.
	imageURL := data.ImageFirst
	if imageURL == "" && len(data.Images) > 0 {
		imageURL = data.Images[0]
	}

	// Build source data JSON from PageData.
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("sourcing: marshal page data: %w", err)
	}
	sourceData := json.RawMessage(raw)

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

	// Store in sourcing1688 table.
	p1688 := sourcing1688.Sourcing1688Product{
		SourceURL:      data.SourceURL,
		SupplierName:   data.SupplierName,
		SupplierID1688: data.SupplierID,
		Price1688:      price,
		MinOrderQty:    moq,
		ImageURL:       imageURL,
		Status:         status,
		SourceData:     sourceData,
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
		rec := Recommendation{
			ID:            p.ID,
			SourceURL:     p.SourceURL,
			SupplierName:  p.SupplierName,
			Price:         p.Price1688,
			Status:        p.Status,
			ProductID1688: p.SupplierID1688,
			ImageURL:      p.ImageURL,
			CreatedAt:     p.CreatedAt.Format(time.RFC3339),
		}

		// Try to extract title from source_data if available.
		if p.SourceData != nil {
			var pd PageData
			if err := json.Unmarshal(p.SourceData, &pd); err == nil {
				rec.Title = pd.Title
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
