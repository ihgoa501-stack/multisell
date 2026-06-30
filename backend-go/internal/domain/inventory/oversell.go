package inventory

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OversellWatcher periodically checks all products with active listings for
// oversell conditions and publishes events when detected.
type OversellWatcher struct {
	db     *gorm.DB
	bus    *eventbus.Bus
	logger *zap.Logger
	svc    *Service
}

// NewOversellWatcher creates a new OversellWatcher.
func NewOversellWatcher(db *gorm.DB, bus *eventbus.Bus, logger *zap.Logger) *OversellWatcher {
	return &OversellWatcher{
		db:     db,
		bus:    bus,
		logger: logger.Named("oversell_watcher"),
		svc:    NewService(db, logger),
	}
}

// RunOnce executes one full sweep of all products that have active listings.
func (w *OversellWatcher) RunOnce(ctx context.Context) {
	// Discover all product IDs that have at least one active listing.
	var productIDs []int64
	if err := w.db.WithContext(ctx).
		Model(&struct{ ProductID int64 }{}).
		Table("product_listing").
		Where("status IN ('active','live','published')").
		Distinct("product_id").
		Pluck("product_id", &productIDs).Error; err != nil {
		w.logger.Error("failed to query active product IDs", zap.Error(err))
		return
	}

	if len(productIDs) == 0 {
		w.logger.Debug("no active listings found, skipping oversell sweep")
		return
	}

	w.logger.Info("oversell sweep starting",
		zap.Int("products_with_active_listings", len(productIDs)))

	for _, pid := range productIDs {
		result, err := w.svc.SyncAcrossPlatforms(ctx, pid)
		if err != nil {
			w.logger.Warn("oversell sync failed for product",
				zap.Int64("product_id", pid),
				zap.Error(err))
			continue
		}

		if result.OversellDetected {
			w.logger.Warn("oversell detected",
				zap.Int64("product_id", pid),
				zap.Int("available", result.AvailableInventory),
				zap.Int("committed", result.TotalCommitted),
				zap.Int("oversell_by", result.OversellBy))

			// Publish event for downstream handling (e.g., alert agents).
			_, pubErr := w.bus.Publish(ctx, "inventory.oversell.detected", "oversell_watcher", map[string]interface{}{
				"product_id":           pid,
				"available_inventory":  result.AvailableInventory,
				"total_committed":      result.TotalCommitted,
				"oversell_by":          result.OversellBy,
				"platform_breakdown":   result.PlatformBreakdown,
				"detected_at":          time.Now(),
			})
			if pubErr != nil {
				w.logger.Warn("failed to publish oversell event",
					zap.Int64("product_id", pid),
					zap.Error(pubErr))
			}
		}
	}

	w.logger.Info("oversell sweep complete")
}
