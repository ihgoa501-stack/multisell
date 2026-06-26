package aftersales

import (
	"context"
	"strconv"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/platform"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DefaultSyncInterval is the default polling interval for the return sync
// worker (10 minutes, matching the legacy Python worker's 600s interval).
const DefaultSyncInterval = 10 * time.Minute

// ReturnSyncWorker polls platform adapters and creates AfterSalesOrder
// records for new returns fetched from e-commerce platforms.
//
// Usage:
//
//	w := NewReturnSyncWorker(db, logger)
//	w.Start()
//	// ... later ...
//	w.Stop()
type ReturnSyncWorker struct {
	db       *gorm.DB
	logger   *zap.Logger
	interval time.Duration
	stopCh   chan struct{}
}

// NewReturnSyncWorker creates a new ReturnSyncWorker with the given DB and
// logger. Use Start() / WithInterval() / Stop() to control the lifecycle.
func NewReturnSyncWorker(db *gorm.DB, logger *zap.Logger) *ReturnSyncWorker {
	return &ReturnSyncWorker{
		db:       db,
		logger:   logger,
		interval: DefaultSyncInterval,
		stopCh:   make(chan struct{}),
	}
}

// WithInterval sets a custom polling interval. Must be called before Start().
func (w *ReturnSyncWorker) WithInterval(d time.Duration) *ReturnSyncWorker {
	w.interval = d
	return w
}

// Start begins the background polling loop in a goroutine. The worker
// immediately runs one sync cycle on start, then polls every interval.
func (w *ReturnSyncWorker) Start() {
	w.logger.Info("ReturnSyncWorker started", zap.Duration("interval", w.interval))
	go w.loop()
}

// Stop signals the background loop to stop after the current cycle
// completes.
func (w *ReturnSyncWorker) Stop() {
	close(w.stopCh)
}

// loop is the main background goroutine.
func (w *ReturnSyncWorker) loop() {
	// Run one cycle immediately on start.
	w.syncOnce()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.syncOnce()
		case <-w.stopCh:
			w.logger.Info("ReturnSyncWorker stopped")
			return
		}
	}
}

// syncOnce executes one full sync cycle: queries active platforms, fetches
// returns via each platform adapter, and creates AfterSalesOrder records.
func (w *ReturnSyncWorker) syncOnce() {
	w.logger.Debug("ReturnSyncWorker: starting sync cycle")

	var platforms []platform.Platform
	if err := w.db.Where("status = ?", 1).Find(&platforms).Error; err != nil {
		w.logger.Error("ReturnSyncWorker: failed to query platforms", zap.Error(err))
		return
	}

	if len(platforms) == 0 {
		w.logger.Debug("ReturnSyncWorker: no active platforms configured")
		return
	}

	// Look back 1 hour (matching legacy Python worker behaviour).
	since := time.Now().Add(-1 * time.Hour)

	for _, pl := range platforms {
		w.syncPlatform(pl, since)
	}
}

// syncPlatform fetches returns from a single platform adapter and processes
// each return record.
func (w *ReturnSyncWorker) syncPlatform(pl platform.Platform, since time.Time) {
	adapter, ok := integrations.GetAdapter(pl.Code)
	if !ok {
		w.logger.Debug("ReturnSyncWorker: no adapter registered",
			zap.String("platform_code", pl.Code))
		return
	}

	returns, err := adapter.FetchReturns(context.Background(), &integrations.FetchReturnsInput{
		PlatformID: pl.ID,
		Since:      since,
	})
	if err != nil {
		w.logger.Error("ReturnSyncWorker: FetchReturns failed",
			zap.String("platform_code", pl.Code),
			zap.Error(err))
		return
	}

	for _, r := range returns {
		w.processReturn(r)
	}
}

// processReturn looks up the local Order and Sku for a platform return, then
// creates an AfterSalesOrder record (with dedup).
func (w *ReturnSyncWorker) processReturn(r *integrations.PlatformReturn) {
	// Look up local Order by order_no.
	var ord order.Order
	if err := w.db.Where("order_no = ?", r.OrderSN).First(&ord).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			w.logger.Warn("ReturnSyncWorker: order not found for return",
				zap.String("order_sn", r.OrderSN),
				zap.String("return_id", r.ReturnID))
		} else {
			w.logger.Error("ReturnSyncWorker: error looking up order",
				zap.String("order_sn", r.OrderSN),
				zap.Error(err))
		}
		return
	}

	// Look up local Sku by code.
	var s *sku.Sku
	if r.SkuCode != "" {
		s = new(sku.Sku)
		if err := w.db.Where("code = ?", r.SkuCode).First(s).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				w.logger.Warn("ReturnSyncWorker: Sku not found for return",
					zap.String("sku_code", r.SkuCode),
					zap.String("return_id", r.ReturnID))
			} else {
				w.logger.Error("ReturnSyncWorker: error looking up Sku",
					zap.String("sku_code", r.SkuCode),
					zap.Error(err))
			}
			return
		}
	}

	// Dedup: skip if an AfterSalesOrder already exists for (order_id, sku_id).
	if s != nil {
		var existing AfterSalesOrder
		err := w.db.Where("order_id = ? AND sku_id = ?", ord.ID, s.ID).First(&existing).Error
		if err == nil {
			w.logger.Debug("ReturnSyncWorker: return already exists, skipping",
				zap.String("return_id", r.ReturnID),
				zap.Int64("aftersales_id", existing.ID))
			return
		} else if err != gorm.ErrRecordNotFound {
			w.logger.Error("ReturnSyncWorker: error checking existing aftersales order",
				zap.String("return_id", r.ReturnID),
				zap.Error(err))
			return
		}
	}

	// Default reason.
	reason := r.Reason
	if reason == "" {
		reason = "平台发起退货"
	}

	// Default quantity to 1 if zero.
	quantity := r.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	// Parse refund amount if provided.
	var refundAmount float64
	if r.RefundAmount != "" {
		if v, err := strconv.ParseFloat(r.RefundAmount, 64); err == nil {
			refundAmount = v
		} else {
			w.logger.Warn("ReturnSyncWorker: invalid refund_amount",
				zap.String("return_id", r.ReturnID),
				zap.String("refund_amount", r.RefundAmount))
		}
	}

	aso := AfterSalesOrder{
		OrderID:          ord.ID,
		ReturnQuantity:   quantity,
		Reason:           reason,
		Status:           "pending",
		RefundAmount:     refundAmount,
		CreatedBy:        "system",
		InspectionResult: "",
	}

	// Set SkuID if we found a matching Sku, and look up OrderItem for ItemID.
	if s != nil {
		aso.SkuID = &s.ID

		var item order.OrderItem
		if err := w.db.Where("order_id = ? AND sku_id = ?", ord.ID, s.ID).First(&item).Error; err == nil {
			aso.ItemID = &item.ID
		}
	}

	if err := w.db.Create(&aso).Error; err != nil {
		w.logger.Error("ReturnSyncWorker: failed to create AfterSalesOrder",
			zap.String("return_id", r.ReturnID),
			zap.Error(err))
		return
	}

	w.logger.Info("ReturnSyncWorker: created AfterSalesOrder",
		zap.Int64("aftersales_id", aso.ID),
		zap.String("return_id", r.ReturnID),
		zap.String("order_sn", r.OrderSN),
		zap.String("sku_code", r.SkuCode))
}
