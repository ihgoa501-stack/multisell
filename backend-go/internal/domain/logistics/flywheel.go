package logistics

import (
	"context"
	"math"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
)

// CarrierPerformance holds aggregate statistics for a shipping carrier/channel.
// A10 updates these on each supplychain.flywheel event so the rate engine
// can converge quoted prices closer to actual experienced costs.
type CarrierPerformance struct {
	ChannelName     string  `json:"channel_name"`
	ProviderName    string  `json:"provider_name"`
	TotalOrders     int     `json:"total_orders"`
	TotalCost       float64 `json:"total_cost"`
	AvgCost         float64 `json:"avg_cost"`
	AvgDeliveryDays float64 `json:"avg_delivery_days"`
	MinDeliveryDays int     `json:"min_delivery_days"`
	MaxDeliveryDays int     `json:"max_delivery_days"`
	LostPackages    int     `json:"lost_packages"`
	LossRate        float64 `json:"loss_rate"`
}

// CategoryPerformance holds aggregate statistics for a category x carrier
// combination. A8 uses this to refine sourcing recommendations by channel.
type CategoryPerformance struct {
	CategoryName   string  `json:"category_name"`
	ChannelName    string  `json:"channel_name"`
	TotalOrders    int     `json:"total_orders"`
	AvgCost        float64 `json:"avg_cost"`
	AvgDeliveryDays float64 `json:"avg_delivery_days"`
}

// carrierKey returns the map key for carrier performance (channel + provider).
func carrierKey(channel, provider string) string {
	return channel + "|" + provider
}

// categoryKey returns the map key for category performance (category + channel).
func categoryKey(category, channel string) string {
	return category + "|" + channel
}

// ---------------------------------------------------------------------------
// A10 flywheel handler
// ---------------------------------------------------------------------------

// HandleFlywheelEvent returns an eventbus.Handler that records fulfillment data
// into the logistics Service's carrier performance statistics (A10).
func (s *Service) HandleFlywheelEvent() eventbus.Handler {
	return func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload

		channelName, _ := payload["channel_name"].(string)
		providerName, _ := payload["provider_name"].(string)
		cost, _ := payload["actual_cost"].(float64)
		deliveryDays := parseDays(payload["delivery_days"])
		isLost, _ := payload["is_lost"].(bool)

		if channelName == "" {
			return nil // nothing to record
		}

		s.recordCarrierPerformance(channelName, providerName, cost, deliveryDays, isLost)
		return nil
	}
}

// HandleCategoryFlywheelEvent returns an eventbus.Handler that records
// fulfillment data into category x carrier statistics (consumed by A8).
func (s *Service) HandleCategoryFlywheelEvent() eventbus.Handler {
	return func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload

		channelName, _ := payload["channel_name"].(string)
		categoryName, _ := payload["category_name"].(string)
		cost, _ := payload["actual_cost"].(float64)
		deliveryDays := parseDays(payload["delivery_days"])

		if channelName == "" && categoryName == "" {
			return nil
		}

		s.recordCategoryPerformance(categoryName, channelName, cost, deliveryDays)
		return nil
	}
}

// ---------------------------------------------------------------------------
// Performance recording
// ---------------------------------------------------------------------------

// recordCarrierPerformance updates the in-memory carrier statistics with one
// fulfillment observation.
func (s *Service) recordCarrierPerformance(channel, provider string, cost float64, deliveryDays int, lost bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := carrierKey(channel, provider)
	cp, ok := s.carrierStats[key]
	if !ok {
		cp = &CarrierPerformance{
			ChannelName:     channel,
			ProviderName:    provider,
			MinDeliveryDays: deliveryDays,
			MaxDeliveryDays: deliveryDays,
		}
		s.carrierStats[key] = cp
	}

	cp.TotalOrders++
	cp.TotalCost += cost

	if deliveryDays < cp.MinDeliveryDays {
		cp.MinDeliveryDays = deliveryDays
	}
	if deliveryDays > cp.MaxDeliveryDays {
		cp.MaxDeliveryDays = deliveryDays
	}
	if lost {
		cp.LostPackages++
	}

	// Recalculate derived averages.
	cp.AvgCost = cp.TotalCost / float64(cp.TotalOrders)
	cp.AvgDeliveryDays = computeAvg(cp.AvgDeliveryDays, float64(deliveryDays), cp.TotalOrders)
	cp.LossRate = math.Round(float64(cp.LostPackages)/float64(cp.TotalOrders)*10000) / 100 // percentage, 2 decimals
}

// recordCategoryPerformance updates the in-memory category x carrier statistics.
func (s *Service) recordCategoryPerformance(category, channel string, cost float64, deliveryDays int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if category == "" {
		return
	}

	key := categoryKey(category, channel)
	cp, ok := s.categoryStats[key]
	if !ok {
		cp = &CategoryPerformance{
			CategoryName: category,
			ChannelName:  channel,
		}
		s.categoryStats[key] = cp
	}

	cp.TotalOrders++
	// Recalculate running averages.
	cp.AvgCost = computeAvg(cp.AvgCost, cost, cp.TotalOrders)
	cp.AvgDeliveryDays = computeAvg(cp.AvgDeliveryDays, float64(deliveryDays), cp.TotalOrders)
}

// ---------------------------------------------------------------------------
// Query methods
// ---------------------------------------------------------------------------

// GetCarrierPerformance returns a snapshot of all carrier performance stats.
func (s *Service) GetCarrierPerformance() []CarrierPerformance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CarrierPerformance, 0, len(s.carrierStats))
	for _, cp := range s.carrierStats {
		result = append(result, *cp)
	}
	return result
}

// GetCarrierPerformanceByChannel returns stats for a specific channel.
func (s *Service) GetCarrierPerformanceByChannel(channel, provider string) *CarrierPerformance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp, ok := s.carrierStats[carrierKey(channel, provider)]
	if !ok {
		return nil
	}
	cpCopy := *cp
	return &cpCopy
}

// GetCategoryPerformance returns a snapshot of all category x carrier stats.
func (s *Service) GetCategoryPerformance() []CategoryPerformance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CategoryPerformance, 0, len(s.categoryStats))
	for _, cp := range s.categoryStats {
		result = append(result, *cp)
	}
	return result
}

// GetCategoryPerformanceByCategory returns stats for a specific category.
func (s *Service) GetCategoryPerformanceByCategory(category string) []CategoryPerformance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []CategoryPerformance
	for _, cp := range s.categoryStats {
		if cp.CategoryName == category {
			result = append(result, *cp)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// computeAvg computes an incremental running average.
func computeAvg(currentAvg, newValue float64, count int) float64 {
	if count <= 1 {
		return newValue
	}
	return currentAvg + (newValue-currentAvg)/float64(count)
}

// parseDays extracts an int from a payload value that could be int64 or float64.
func parseDays(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
