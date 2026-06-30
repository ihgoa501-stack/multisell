package supplychain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/supplyevent"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides supply chain flow business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	bus    *eventbus.Bus
}

// NewService creates a new supply chain flow service.
// bus is optional — if nil, flywheel events will not be published.
func NewService(db *gorm.DB, logger *zap.Logger, bus *eventbus.Bus) *Service {
	return &Service{db: db, logger: logger, bus: bus}
}

// List returns a paginated list of supply chain flows with optional filters.
func (s *Service) List(ctx context.Context, req ListFlowsRequest) ([]SupplyChainFlow, int64, error) {
	var items []SupplyChainFlow
	var total int64

	q := s.db.WithContext(ctx).Model(&SupplyChainFlow{})
	if req.SourceType != "" {
		q = q.Where("source_type = ?", req.SourceType)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}
	offset := (req.Page - 1) * req.Size
	if err := q.Order("created_at DESC").Offset(offset).Limit(req.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID retrieves a single supply chain flow by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*SupplyChainFlow, error) {
	var flow SupplyChainFlow
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&flow).Error; err != nil {
		return nil, err
	}
	return &flow, nil
}

// FlowEventsResponse is the payload returned by GetEvents. It bundles the
// flow record with the ordered list of event_outbox rows that reference it.
type FlowEventsResponse struct {
	Flow   *SupplyChainFlow `json:"flow"`
	Events []EventOutboxRow `json:"events"`
}

// EventOutboxRow is a read-only projection of the event_outbox table for the
// supply chain flow timeline. The table is created by migration 000007 and
// written to by the event bus whenever an event is published.
type EventOutboxRow struct {
	ID        int64                  `gorm:"column:id" json:"id"`
	Topic     string                 `gorm:"column:topic" json:"topic"`
	Source    string                 `gorm:"column:source" json:"source"`
	Payload   map[string]interface{} `gorm:"column:payload;serializer:json" json:"payload"`
	Priority  int                    `gorm:"column:priority" json:"priority"`
	Status    string                 `gorm:"column:status" json:"status"`
	CreatedAt time.Time              `gorm:"column:created_at" json:"created_at"`
}

// TableName pins EventOutboxRow to the event_outbox table created by migration 000007.
func (EventOutboxRow) TableName() string { return "event_outbox" }

// GetEvents retrieves the flow plus its event timeline from event_outbox.
//
// Events are matched to the flow via the "flow_id" key in the payload JSON.
// The orchestrator attaches flow_id when publishing supplychain.* events
// (e.g. supplychain.quote_requested, supplychain.flywheel), so each event
// tied to this flow appears in the timeline ordered by created_at.
//
// The JSON-path filter is dialect-aware: PostgreSQL uses payload->>'flow_id'
// while SQLite uses json_extract(payload, '$.flow_id') for in-memory tests.
func (s *Service) GetEvents(ctx context.Context, id string) (*FlowEventsResponse, error) {
	var flow SupplyChainFlow
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&flow).Error; err != nil {
		return nil, err
	}

	var events []EventOutboxRow
	q := s.db.WithContext(ctx).Model(&EventOutboxRow{}).Order("created_at ASC")
	switch s.db.Dialector.Name() {
	case "postgres":
		q = q.Where("payload->>'flow_id' = ?", id)
	default:
		// SQLite (in-memory tests) and any other dialect: use json_extract.
		q = q.Where("json_extract(payload, '$.flow_id') = ?", id)
	}
	// Ignore table-missing errors so the endpoint degrades gracefully when
	// event_outbox has not been provisioned (e.g. legacy test fixtures).
	if err := q.Find(&events).Error; err != nil {
		s.logger.Warn("supplychain: failed to query event_outbox for flow timeline",
			zap.String("flow_id", id), zap.Error(err))
		events = nil
	}

	return &FlowEventsResponse{Flow: &flow, Events: events}, nil
}

// Create inserts a new supply chain flow.
func (s *Service) Create(ctx context.Context, flow *SupplyChainFlow) error {
	return s.db.WithContext(ctx).Create(flow).Error
}

// Update updates the status and summary fields of an existing supply chain flow.
// When the status transitions to "completed", it publishes a supplychain.flywheel
// event to close the data flywheel loop (A10 carrier_performance, A8 category_performance).
func (s *Service) Update(ctx context.Context, id string, req UpdateFlowRequest) error {
	updates := map[string]interface{}{
		"status": req.Status,
	}
	if req.Context != nil {
		updates["context"] = req.Context
	}
	if req.CarrierSummary != nil {
		updates["carrier_summary"] = req.CarrierSummary
	}
	if req.FinancialSummary != nil {
		updates["financial_summary"] = req.FinancialSummary
	}
	if req.ErrorLog != nil {
		updates["error_log"] = req.ErrorLog
	}
	if err := s.db.WithContext(ctx).Model(&SupplyChainFlow{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}

	// When status transitions to "completed", publish the flywheel event.
	if req.Status == "completed" && s.bus != nil {
		s.publishFlywheel(ctx, id, req)
	}

	return nil
}

// publishFlywheel builds and publishes a supplychain.flywheel event from the
// completed flow's carrier_summary and financial_summary fields.
func (s *Service) publishFlywheel(ctx context.Context, id string, req UpdateFlowRequest) {
	flywheel := buildFlywheelEvent(id, req)
	payload, err := supplyevent.ToPayload(flywheel)
	if err != nil {
		s.logger.Warn("supplychain: failed to marshal flywheel event", zap.String("flow_id", id), zap.Error(err))
		return
	}
	_, err = s.bus.Publish(ctx, "supplychain.flywheel", "supplychain", payload)
	if err != nil {
		s.logger.Warn("supplychain: failed to publish flywheel event", zap.String("flow_id", id), zap.Error(err))
	}
}

// buildFlywheelEvent extracts fulfillment data from the update request.
// It parses carrier_summary and financial_summary JSONB fields for the
// actual cost, delivery time, and loss info that flow back into the
// carrier_performance and category_performance statistics.
func buildFlywheelEvent(id string, req UpdateFlowRequest) supplyevent.FlywheelEvent {
	evt := supplyevent.FlywheelEvent{
		FlowID:    id,
		Timestamp: time.Now(),
	}

	// Extract carrier data from carrier_summary JSONB.
	if req.CarrierSummary != nil {
		var cs struct {
			Channel      string `json:"channel"`
			Provider     string `json:"provider"`
			Category     string `json:"category"`
			DeliveryDays int    `json:"delivery_days"`
			IsLost       bool   `json:"is_lost"`
		}
		if err := json.Unmarshal(*req.CarrierSummary, &cs); err == nil {
			evt.ChannelName = cs.Channel
			evt.ProviderName = cs.Provider
			evt.CategoryName = cs.Category
			evt.DeliveryDays = cs.DeliveryDays
			evt.IsLost = cs.IsLost
		}
	}

	// Extract financial data from financial_summary JSONB.
	if req.FinancialSummary != nil {
		var fs struct {
			ActualCost float64 `json:"actual_cost"`
			Currency   string  `json:"currency"`
		}
		if err := json.Unmarshal(*req.FinancialSummary, &fs); err == nil {
			evt.ActualCost = fs.ActualCost
			if fs.Currency != "" {
				evt.Currency = fs.Currency
			}
		}
	}

	return evt
}
