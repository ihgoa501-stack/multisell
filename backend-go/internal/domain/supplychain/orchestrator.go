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

// ActionInput is the supplychain-local shape for creating a UnifiedAction.
// It mirrors the subset of ai.CreateActionInput that the orchestrator needs;
// keeping it local avoids importing internal/ai from this domain package.
type ActionInput struct {
	SourceTable        string
	SourceID           string
	SourceType         string
	AgentID            string
	SquadID            string
	ActionType         string
	BusinessObjectType string
	BusinessObjectID   string
	Title              string
	Description        string
	Payload            map[string]interface{}
	RiskLevel          string
	RequiresApproval   bool
	ProposedBy         string
}

// ActionCreator creates approval-gated UnifiedActions for high-risk
// operations (refund, resend, replenishment, etc.). Implementations should
// persist the action in the "suggested" state so it shows up in the
// approval queue.
type ActionCreator interface {
	CreateAction(in *ActionInput) (int64, error)
}

// Orchestrator coordinates supply chain operations across agents
// (A8 sourcing, A10 quoting, etc.) via the event bus.
type Orchestrator struct {
	bus           *eventbus.Bus
	db            *gorm.DB
	logger        *zap.Logger
	returnTracker *aftersales.ReturnRateTracker
	actions       ActionCreator
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

// WithActionCreator wires an ActionCreator (typically an adapter around
// *ai.Service) so the orchestrator can create approval-gated UnifiedActions
// for refund/resend/replenishment operations. Without a creator the
// orchestrator logs a warning and skips action creation — flows still work.
func (o *Orchestrator) WithActionCreator(ac ActionCreator) *Orchestrator {
	o.actions = ac
	return o
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
// tracks the return in the ReturnRateTracker, transitions the flow status, and
// creates an approval-gated UnifiedAction so a human operator can approve the
// actual refund/resend — the orchestrator never executes high-risk operations
// automatically.
//
// The return-rate tracker update closes the data flywheel to A8 sourcing:
// elevated return rates for a SKU surface in A8's next scan via the
// sku_return_stats table, downgrading the SKU's sourcing score.
func (o *Orchestrator) HandleAftersaleReturn(ctx context.Context, evt eventbus.Event) error {
	payload := evt.Payload

	// Extract fields, handling both int64 and float64 numeric representations
	// since JSON deserialization through the event bus produces float64.
	aftersaleID := getInt64(payload, "aftersale_id")
	orderID := getInt64(payload, "order_id")
	skuID := getInt64(payload, "sku_id")
	quantity := int(getInt64(payload, "quantity"))

	reason, _ := payload["reason"].(string)

	// Track the return in the DB-backed return-rate tracker. This is the
	// writeback to A8: elevated return rates surface in A8's next scan and
	// downgrade the SKU's sourcing score.
	if o.returnTracker != nil && skuID != 0 {
		if err := o.returnTracker.TrackReturn(skuID, quantity); err != nil {
			o.logger.Warn("HandleAftersaleReturn: failed to track return rate",
				zap.Int64("sku_id", skuID), zap.Error(err))
		}
	}

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

	// Create an approval-gated UnifiedAction for the refund/resend decision.
	// The orchestrator only SUGGESTS the action; a human operator must approve
	// before any money moves or goods ship. Risk is high because refunds move
	// money out and resends consume inventory.
	o.createRefundResendAction(ctx, flow.ID, aftersaleID, orderID, skuID, quantity, reason)

	o.logger.Info("reverse logistics flow created",
		zap.String("flow_id", flow.ID),
		zap.Int64("aftersale_id", aftersaleID),
		zap.Int64("sku_id", skuID),
		zap.String("status", "inspecting"))

	return nil
}

// createRefundResendAction creates a UnifiedAction in the "suggested" state
// for the refund/resend decision. The action requires human approval — the
// orchestrator never executes refunds or resends automatically. Failures are
// logged but do not fail the handler: the flow itself was already created,
// and operators can still drive it manually from the cockpit.
func (o *Orchestrator) createRefundResendAction(ctx context.Context, flowID string, aftersaleID, orderID, skuID int64, quantity int, reason string) {
	if o.actions == nil {
		o.logger.Warn("HandleAftersaleReturn: no ActionCreator wired; skipping UnifiedAction creation",
			zap.String("flow_id", flowID), zap.Int64("aftersale_id", aftersaleID))
		return
	}
	payload := map[string]interface{}{
		"flow_id":     flowID,
		"aftersale_id": aftersaleID,
		"order_id":    orderID,
		"sku_id":      skuID,
		"quantity":    quantity,
		"reason":      reason,
		"options":     []string{"refund", "resend"},
	}
	desc := "客户发起退货 (aftersale_id=" + strconv.FormatInt(aftersaleID, 10) +
		", sku_id=" + strconv.FormatInt(skuID, 10) +
		", qty=" + strconv.Itoa(quantity) + "). 请审批：退款 或 重发。"
	in := &ActionInput{
		SourceTable:        "supply_chain_flow",
		SourceID:           flowID,
		SourceType:         "aftersale_return",
		AgentID:            "A6", // A6 aftersales management owns refund decisions
		SquadID:            "service",
		ActionType:         "refund_or_resend",
		BusinessObjectType: "aftersales_order",
		BusinessObjectID:   strconv.FormatInt(aftersaleID, 10),
		Title:              "售后退货审批：退款/重发 #" + strconv.FormatInt(aftersaleID, 10),
		Description:        desc,
		Payload:            payload,
		RiskLevel:          "high",
		RequiresApproval:   true,
		ProposedBy:         "supplychain.orchestrator",
	}
	actionID, err := o.actions.CreateAction(in)
	if err != nil {
		o.logger.Warn("HandleAftersaleReturn: failed to create UnifiedAction",
			zap.String("flow_id", flowID),
			zap.Int64("aftersale_id", aftersaleID),
			zap.Error(err))
		return
	}
	o.logger.Info("refund/resend UnifiedAction created",
		zap.Int64("action_id", actionID),
		zap.String("flow_id", flowID),
		zap.Int64("aftersale_id", aftersaleID))
}

// HandleStockCritical processes supplychain.stock.critical events.
//
// When A5 stock_alert detects a red (critical) stock level, it publishes
// this event. The orchestrator:
//  1. Creates a supply_chain_flow record (status pending → sourcing_requested).
//  2. Publishes a sourcing.rescan event so A8 can scan for replacement products.
//  3. Creates an approval-gated UnifiedAction for replenishment — actual
//     purchase orders are never auto-placed; a human operator must approve.
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

	// Create an approval-gated UnifiedAction for replenishment. The
	// orchestrator only SUGGESTS the replenishment — placing an actual
	// purchase order moves money and inventory, so a human must approve.
	o.createReplenishAction(flow.ID, skuID, currentStock, safetyStock, sellableDays)

	o.logger.Info("stock critical flow created",
		zap.String("flow_id", flow.ID),
		zap.Int64("sku_id", skuID),
		zap.String("status", "sourcing_requested"))

	return nil
}

// createReplenishAction creates a UnifiedAction in the "suggested" state for
// the replenishment decision. Risk is high because replenishment places a
// purchase order (money out + inventory in). Failures are logged but do not
// fail the handler — the flow and sourcing.rescan event already happened.
func (o *Orchestrator) createReplenishAction(flowID string, skuID int64, currentStock, safetyStock int, sellableDays float64) {
	if o.actions == nil {
		o.logger.Warn("HandleStockCritical: no ActionCreator wired; skipping UnifiedAction creation",
			zap.String("flow_id", flowID), zap.Int64("sku_id", skuID))
		return
	}
	payload := map[string]interface{}{
		"flow_id":       flowID,
		"sku_id":        skuID,
		"current_stock": currentStock,
		"safety_stock":  safetyStock,
		"sellable_days": sellableDays,
		"suggested_action": "place_purchase_order",
	}
	desc := "库存红色预警 (sku_id=" + strconv.FormatInt(skuID, 10) +
		", 当前库存=" + strconv.Itoa(currentStock) +
		", 安全库存=" + strconv.Itoa(safetyStock) +
		", 可售天数=" + strconv.FormatFloat(sellableDays, 'f', 1, 64) +
		"). 请审批：是否下达采购补货单。"
	in := &ActionInput{
		SourceTable:        "supply_chain_flow",
		SourceID:           flowID,
		SourceType:         "stock_critical",
		AgentID:            "A5", // A5 inventory alert owns replenishment suggestions
		SquadID:            "logistics",
		ActionType:         "replenish_order",
		BusinessObjectType: "sku",
		BusinessObjectID:   strconv.FormatInt(skuID, 10),
		Title:              "库存红色预警补货审批：SKU #" + strconv.FormatInt(skuID, 10),
		Description:        desc,
		Payload:            payload,
		RiskLevel:          "high",
		RequiresApproval:   true,
		ProposedBy:         "supplychain.orchestrator",
	}
	actionID, err := o.actions.CreateAction(in)
	if err != nil {
		o.logger.Warn("HandleStockCritical: failed to create UnifiedAction",
			zap.String("flow_id", flowID),
			zap.Int64("sku_id", skuID),
			zap.Error(err))
		return
	}
	o.logger.Info("replenish UnifiedAction created",
		zap.Int64("action_id", actionID),
		zap.String("flow_id", flowID),
		zap.Int64("sku_id", skuID))
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
