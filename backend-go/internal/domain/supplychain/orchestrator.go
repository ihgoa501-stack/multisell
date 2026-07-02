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
// operations. Implementations should persist suggestions for approval,
// not execute refunds, resends, or replenishment automatically.
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
	escalation    *EscalationManager
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

// WithActionCreator wires an ActionCreator for approval-gated high-risk actions.
// Without a creator, flows still progress and action creation is skipped.
func (o *Orchestrator) WithActionCreator(ac ActionCreator) *Orchestrator {
	o.actions = ac
	return o
}

// SetEscalationManager injects a 4-level EscalationManager (Issue #35) into the
// orchestrator state machine. When set, the orchestrator can route operational
// errors through Escalate(...) to apply the standard auto-retry → skip →
// manual-review → global-alert progression. When nil, Escalate is a no-op.
func (o *Orchestrator) SetEscalationManager(em *EscalationManager) *Orchestrator {
	o.escalation = em
	return o
}

// Escalate routes an escalation event through the configured EscalationManager.
// Returns nil when no manager is wired (the orchestrator logs and continues) so
// callers can invoke Escalate unconditionally without guarding for nil.
func (o *Orchestrator) Escalate(ctx context.Context, evt EscalationEvent) error {
	if o.escalation == nil {
		o.logger.Warn("orchestrator: escalation manager not configured, dropping event",
			zap.String("flow_id", evt.FlowID),
			zap.Int("level", int(evt.Level)),
			zap.String("error", evt.Error),
		)
		return nil
	}
	return o.escalation.Handle(ctx, evt)
}

// HandleRecommendEvent processes sourcing.recommend events.
// It creates a supply_chain_flow record (source_type="sourcing_recommend"),
// then publishes a supplychain.quote_requested event carrying the flow_id
// so downstream events can be correlated on the flow timeline.
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

	// Persist a supply_chain_flow record so the recommendation enters the
	// auditable flow timeline. The DB is optional — if absent (e.g. in
	// lightweight unit tests without a DB), we still publish the event
	// with a generated flow_id so downstream consumers can correlate.
	flowID := uuid.New().String()
	if o.db != nil {
		ctxData := map[string]interface{}{
			"product_id": productID,
			"source_url": sourceURL,
		}
		rawCtx, err := json.Marshal(ctxData)
		if err != nil {
			o.logger.Error("HandleRecommendEvent: failed to marshal context",
				zap.Int64("product_id", productID), zap.Error(err))
			return err
		}
		contextRaw := json.RawMessage(rawCtx)
		flow := &SupplyChainFlow{
			ID:         flowID,
			SourceType: "sourcing_recommend",
			SourceID:   strconv.FormatInt(productID, 10),
			Status:     "pending",
			Context:    &contextRaw,
		}
		if err := o.db.WithContext(ctx).Create(flow).Error; err != nil {
			o.logger.Error("HandleRecommendEvent: failed to create supply chain flow",
				zap.Int64("product_id", productID), zap.Error(err))
			return err
		}
		o.logger.Info("sourcing recommend flow created",
			zap.String("flow_id", flowID),
			zap.Int64("product_id", productID))
	}

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
	// Attach flow_id so the event can be linked back to the flow timeline.
	quotePayload["flow_id"] = flowID

	_, err = o.bus.Publish(ctx, "supplychain.quote_requested", "supplychain", quotePayload)
	return err
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

	o.createRefundResendAction(ctx, flow.ID, aftersaleID, orderID, skuID, quantity, reason)

	return nil
}

// createRefundResendAction creates an approval-gated suggestion for the
// operator. It never executes refunds or resends automatically.
func (o *Orchestrator) createRefundResendAction(ctx context.Context, flowID string, aftersaleID, orderID, skuID int64, quantity int, reason string) {
	if o.actions == nil {
		o.logger.Warn("HandleAftersaleReturn: no ActionCreator wired; skipping UnifiedAction creation",
			zap.String("flow_id", flowID), zap.Int64("aftersale_id", aftersaleID))
		return
	}
	payload := map[string]interface{}{
		"aftersale_id": aftersaleID,
		"order_id":     orderID,
		"sku_id":       skuID,
		"quantity":     quantity,
		"reason":       reason,
		"flow_id":      flowID,
	}
	desc := "退货已进入逆向物流流程，请审批退款或重发方案。"
	if reason != "" {
		desc += " 原因：" + reason
	}
	in := &ActionInput{
		SourceTable:        "supply_chain_flow",
		SourceID:           flowID,
		SourceType:         "aftersale_return",
		AgentID:            "A6",
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
	o.logger.Info("HandleAftersaleReturn: UnifiedAction created",
		zap.Int64("action_id", actionID),
		zap.String("flow_id", flowID),
		zap.Int64("aftersale_id", aftersaleID))
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

	o.createReplenishAction(flow.ID, skuID, currentStock, safetyStock, sellableDays)

	return nil
}

// createReplenishAction creates an approval-gated replenishment suggestion.
// It never places purchase orders automatically.
func (o *Orchestrator) createReplenishAction(flowID string, skuID int64, currentStock, safetyStock int, sellableDays float64) {
	if o.actions == nil {
		o.logger.Warn("HandleStockCritical: no ActionCreator wired; skipping UnifiedAction creation",
			zap.String("flow_id", flowID), zap.Int64("sku_id", skuID))
		return
	}
	payload := map[string]interface{}{
		"sku_id":        skuID,
		"current_stock": currentStock,
		"safety_stock":  safetyStock,
		"sellable_days": sellableDays,
		"flow_id":       flowID,
	}
	desc := "库存红色预警，建议创建补货单。该操作涉及采购资金和库存变化，必须人工审批。"
	in := &ActionInput{
		SourceTable:        "supply_chain_flow",
		SourceID:           flowID,
		SourceType:         "stock_critical",
		AgentID:            "A5",
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
	o.logger.Info("HandleStockCritical: UnifiedAction created",
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
