// Package supplychain implements the supply chain orchestration layer that
// bridges sourcing recommendations (A8) with logistics quoting (A10) and
// downstream fulfillment events.
//
// The Orchestrator listens for sourcing.recommend events via the event bus,
// translates them into supplychain.quote_requested events, and provides a
// tick handler for periodic no-op cycles.
package supplychain

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lingmirror/backend-go/internal/domain/aftersales"
	"github.com/lingmirror/backend-go/internal/domain/supplyevent"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Orchestrator coordinates supply chain operations across agents
// (A8 sourcing, A10 quoting, etc.) via the event bus.
type Orchestrator struct {
	bus           *eventbus.Bus
	db            *gorm.DB
	logger        *zap.Logger
	returnTracker *aftersales.ReturnRateTracker
}

// NewOrchestrator creates a new supply chain orchestrator.
func NewOrchestrator(bus *eventbus.Bus, db *gorm.DB, logger *zap.Logger, returnTracker *aftersales.ReturnRateTracker) *Orchestrator {
	return &Orchestrator{
		bus:           bus,
		db:            db,
		logger:        logger,
		returnTracker: returnTracker,
	}
}

// HandleRecommendEvent processes sourcing.recommend events.
// It extracts product info from the event payload and publishes
// a supplychain.quote_requested event to trigger A10 quoting.
func (o *Orchestrator) HandleRecommendEvent(ctx context.Context, evt eventbus.Event) error {
	payload := evt.Payload

	// Extract product ID, handling both direct int64 (in-process)
	// and float64 (JSON-deserialized from outbox/scheduler).
	var productID int64
	if id, ok := payload["id"].(int64); ok {
		productID = id
	} else if idFloat, ok := payload["id"].(float64); ok {
		productID = int64(idFloat)
	}

	sourceURL, _ := payload["source_url"].(string)

	quotePayload, err := supplyevent.ToPayload(supplyevent.QuoteRequested{
		ProductID:   productID,
		SourceURL:   sourceURL,
		Destination: "",
		WeightKg:    0,
		Timestamp:   time.Now(),
	})
	if err != nil {
		return err
	}

	_, err = o.bus.Publish(ctx, "supplychain.quote_requested", "supplychain", quotePayload)
	return err
}

// HandleTick processes scheduler ticks for the orchestrator.
// Currently a no-op placeholder.
func (o *Orchestrator) HandleTick(ctx context.Context, evt eventbus.Event) error {
	return nil
}

// HandleAftersaleReturn processes supplychain.aftersale.returned events.
// It creates a reverse supply-chain flow (return -> inspection -> refund/resend),
// tracks the return in the ReturnRateTracker, and transitions the flow status.
func (o *Orchestrator) HandleAftersaleReturn(ctx context.Context, evt eventbus.Event) error {
	payload := evt.Payload

	// Extract fields, handling both int64 and float64 numeric representations
	// since JSON deserialization through the event bus produces float64.
	aftersaleID := getInt64(payload, "aftersale_id")
	orderID := getInt64(payload, "order_id")
	skuID := getInt64(payload, "sku_id")
	quantity := int(getInt64(payload, "quantity"))

	reason, _ := payload["reason"].(string)

	// Track the return event in-memory.
	o.returnTracker.TrackReturn(skuID, quantity)

	// Build context JSON for the supply chain flow record.
	ctxData := map[string]interface{}{
		"aftersale_id": aftersaleID,
		"order_id":     orderID,
		"sku_id":       skuID,
		"quantity":     quantity,
		"reason":       reason,
	}
	rawCtx, err := json.Marshal(ctxData)
	if err != nil {
		o.logger.Error("failed to marshal return context",
			zap.Int64("aftersale_id", aftersaleID), zap.Error(err))
		return err
	}
	contextRaw := json.RawMessage(rawCtx)

	// Create the reverse supply chain flow in status "pending".
	// Generate a UUID here so the ID is available for the subsequent status
	// update — this also works in environments without gen_random_uuid() (e.g.
	// test databases).
	flow := &SupplyChainFlow{
		ID:         uuid.New().String(),
		SourceType: "aftersale_return",
		SourceID:   strconv.FormatInt(aftersaleID, 10),
		Status:     "pending",
		Context:    &contextRaw,
	}
	if err := o.db.WithContext(ctx).Create(flow).Error; err != nil {
		o.logger.Error("failed to create reverse supply chain flow",
			zap.Int64("aftersale_id", aftersaleID), zap.Error(err))
		return err
	}

	// Transition flow to "inspecting" once the return is tracked.
	if err := o.db.WithContext(ctx).
		Model(&SupplyChainFlow{}).
		Where("id = ?", flow.ID).
		Update("status", "inspecting").Error; err != nil {
		o.logger.Warn("failed to transition reverse flow to inspecting",
			zap.String("flow_id", flow.ID), zap.Error(err))
		// Non-fatal — flow was created; continue.
	}

	o.logger.Info("reverse logistics flow created",
		zap.String("flow_id", flow.ID),
		zap.Int64("aftersale_id", aftersaleID),
		zap.Int64("sku_id", skuID),
		zap.String("status", "inspecting"))

	return nil
}

// HandleStockCritical processes supplychain.stock.critical events.
//
// When A5 stock_alert detects a red (critical) stock level, it publishes
// this event. The orchestrator creates a supply_chain_flow record and
// publishes a sourcing.rescan event so A8 can scan for replacement products.
func (o *Orchestrator) HandleStockCritical(ctx context.Context, evt eventbus.Event) error {
	payload := evt.Payload

	skuID := getInt64(payload, "sku_id")
	currentStock := int(getInt64(payload, "current_stock"))
	safetyStock := int(getInt64(payload, "safety_stock"))
	sellableDays := getFloat64(payload, "sellable_days")

	if skuID == 0 {
		o.logger.Warn("HandleStockCritical: missing or zero sku_id, skipping")
		return nil
	}

	ctxData := map[string]interface{}{
		"sku_id":        skuID,
		"current_stock": currentStock,
		"safety_stock":  safetyStock,
		"sellable_days": sellableDays,
	}
	rawCtx, err := json.Marshal(ctxData)
	if err != nil {
		o.logger.Error("HandleStockCritical: failed to marshal context",
			zap.Int64("sku_id", skuID), zap.Error(err))
		return err
	}
	contextRaw := json.RawMessage(rawCtx)

	flow := &SupplyChainFlow{
		ID:         uuid.New().String(),
		SourceType: "stock_critical",
		SourceID:   strconv.FormatInt(skuID, 10),
		Status:     "pending",
		Context:    &contextRaw,
	}
	if err := o.db.WithContext(ctx).Create(flow).Error; err != nil {
		o.logger.Error("HandleStockCritical: failed to create supply chain flow",
			zap.Int64("sku_id", skuID), zap.Error(err))
		return err
	}

	rescanPayload := map[string]interface{}{
		"sku_id":        skuID,
		"current_stock": currentStock,
		"safety_stock":  safetyStock,
		"sellable_days": sellableDays,
		"flow_id":       flow.ID,
	}
	_, pubErr := o.bus.Publish(ctx, "sourcing.rescan", "supplychain", rescanPayload)
	if pubErr != nil {
		o.logger.Warn("HandleStockCritical: failed to publish sourcing.rescan",
			zap.Int64("sku_id", skuID), zap.Error(pubErr))
	}

	if err := o.db.WithContext(ctx).
		Model(&SupplyChainFlow{}).
		Where("id = ?", flow.ID).
		Update("status", "sourcing_requested").Error; err != nil {
		o.logger.Warn("HandleStockCritical: failed to transition flow to sourcing_requested",
			zap.String("flow_id", flow.ID), zap.Error(err))
	}

	o.logger.Info("stock critical flow created",
		zap.String("flow_id", flow.ID),
		zap.Int64("sku_id", skuID),
		zap.String("status", "sourcing_requested"))

	return nil
}

// getInt64 extracts an int64 from an event payload, handling both int64 and
// float64 representations that arise from in-process vs. JSON-deserialized events.
func getInt64(payload map[string]interface{}, key string) int64 {
	if v, ok := payload[key].(int64); ok {
		return v
	}
	if v, ok := payload[key].(float64); ok {
		return int64(v)
	}
	return 0
}

// getFloat64 extracts a float64 from an event payload, handling direct float64
// and int64 representations.
func getFloat64(payload map[string]interface{}, key string) float64 {
	if v, ok := payload[key].(float64); ok {
		return v
	}
	if v, ok := payload[key].(int64); ok {
		return float64(v)
	}
	return 0
}
