package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/integrations/aimapper"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// eventTopicMap maps platform event types to event bus topics.
var eventTopicMap = map[string]string{
	"listing_blocked":   "agent.decided.G3.listing_blocked",
	"listing_live":      "product.listing.live",
	"price_changed":     "product.price.changed",
	"inventory_changed": "product.inventory.changed",
}

// WebhookLog records a received webhook event for audit and debugging.
type WebhookLog struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Platform     string          `gorm:"column:platform" json:"platform"`
	EventType    string          `gorm:"column:event_type" json:"event_type"`
	RawPayload   json.RawMessage `gorm:"column:raw_payload;type:jsonb" json:"raw_payload,omitempty"`
	Status       string          `gorm:"column:status;default:received" json:"status"`
	MappedEvent  string          `gorm:"column:mapped_event" json:"mapped_event"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (WebhookLog) TableName() string { return "webhook_event_log" }

// RegisterWebhookRoutes registers webhook endpoints on the given router group.
// These routes are public (no auth) because external platforms need to call them.
func RegisterWebhookRoutes(rg *gin.RouterGroup, bus *eventbus.Bus, logger *zap.Logger) {
	h := &webhookHandler{bus: bus, logger: logger}
	rg.POST("/webhooks/:platform", h.ReceiveWebhook)
}

// RegisterWebhookRoutesWithPipeline registers webhook endpoints with an AI pipeline
// that runs after raw event storage.
func RegisterWebhookRoutesWithPipeline(rg *gin.RouterGroup, bus *eventbus.Bus, logger *zap.Logger, pipeline *aimapper.Pipeline, db *gorm.DB) {
	h := &webhookHandler{bus: bus, logger: logger, pipeline: pipeline, db: db}
	rg.POST("/webhooks/:platform", h.ReceiveWebhook)
}

// RegisterWebhookAdminRoutes registers webhook administration endpoints (JWT-protected).
func RegisterWebhookAdminRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	h := &webhookHandler{db: db, logger: logger}
	rg.GET("/platform-webhooks/config", h.GetConfig)
	rg.POST("/platform-webhooks/test-event", h.TestEvent)
}

type webhookHandler struct {
	db       *gorm.DB
	bus      *eventbus.Bus
	logger   *zap.Logger
	pipeline *aimapper.Pipeline
}

// ReceiveWebhook POST /api/webhooks/:platform
// Receives a generic webhook payload from an external e-commerce platform.
// All requests are signature-verified before any processing or event bus publication.
func (h *webhookHandler) ReceiveWebhook(c *gin.Context) {
	platform := strings.ToLower(c.Param("platform"))
	if platform == "" {
		response.Error(c, http.StatusBadRequest, "platform is required")
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to read request body")
		return
	}

	// Look up the platform adapter and verify the webhook signature.
	adapter, ok := GetAdapter(platform)
	if !ok {
		h.logger.Warn("webhook from unknown platform rejected",
			zap.String("platform", platform),
			zap.String("remote_addr", c.ClientIP()),
		)
		response.Error(c, http.StatusBadRequest, "unknown platform: "+platform)
		return
	}

	verifier, supportsVerification := adapter.(WebhookVerifier)
	if !supportsVerification {
		h.logger.Warn("webhook signature verification not supported for platform",
			zap.String("platform", platform),
			zap.String("remote_addr", c.ClientIP()),
		)
		response.Error(c, http.StatusInternalServerError,
			"webhook signature verification not configured for platform: "+platform)
		return
	}

	if !verifier.VerifyWebhookSignature(c.Request.Context(), body, c.Request.Header) {
		h.logger.Warn("webhook signature verification failed",
			zap.String("platform", platform),
			zap.String("remote_addr", c.ClientIP()),
		)
		response.Error(c, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	// Extract event type from the raw payload.
	eventType := detectEventType(raw)
	if eventType == "" {
		eventType = "unknown"
	}

	// Build a PlatformEvent from the raw payload.
	platformEvent := PlatformEvent{
		EventType: eventType,
		Data:      raw,
		OccurredAt: time.Now(),
	}

	// Extract platform_id if present
	if pid, ok := raw["platform_id"].(float64); ok {
		platformEvent.PlatformID = int64(pid)
	}
	// Extract product_id if present
	if pid, ok := raw["product_id"].(string); ok {
		platformEvent.ProductID = pid
	}
	// Extract sku_code if present
	if sku, ok := raw["sku_code"].(string); ok {
		platformEvent.SKUCode = sku
	}

	// Map to event bus topic.
	mappedTopic := ""
	if t, ok := eventTopicMap[eventType]; ok {
		mappedTopic = t
	} else {
		mappedTopic = "platform.event." + eventType
	}

	// Build the payload to publish to the bus.
	payload := map[string]interface{}{
		"platform":     platform,
		"platform_id":  platformEvent.PlatformID,
		"event_type":   eventType,
		"product_id":   platformEvent.ProductID,
		"sku_code":     platformEvent.SKUCode,
		"raw":          raw,
		"occurred_at":  platformEvent.OccurredAt.Format(time.RFC3339),
	}

	// Publish to the event bus.
	eventID, pubErr := h.bus.Publish(c.Request.Context(), mappedTopic, "webhook:"+platform, payload)
	if pubErr != nil {
		h.logger.Warn("webhook publish failed",
			zap.String("platform", platform),
			zap.String("event_type", eventType),
			zap.Error(pubErr),
		)
	}

	// Log to database if configured.
	if h.db != nil {
		payloadBytes, _ := json.Marshal(raw)
		logEntry := WebhookLog{
			Platform:    platform,
			EventType:   eventType,
			RawPayload:  payloadBytes,
			Status:      "received",
			MappedEvent: mappedTopic,
		}
		if err := h.db.Create(&logEntry).Error; err != nil {
			h.logger.Warn("failed to persist webhook log", zap.Error(err))
		}

		// Store as RawEvent and trigger AI pipeline in background.
		rawEvent := RawEvent{
			PlatformCode:  platform,
			EventType:     eventType,
			RawPayload:    payloadBytes,
			MappingStatus: "pending",
		}
		if err := h.db.Create(&rawEvent).Error; err != nil {
			h.logger.Warn("failed to persist raw event", zap.Error(err))
		} else if h.pipeline != nil {
			eventID := rawEvent.ID
			h.logger.Info("triggering pipeline for raw event", zap.Int64("event_id", eventID))
			go func(eid int64, pc, et string, rp json.RawMessage) {
				_, err := h.pipeline.ProcessRawEvent(context.Background(), eid, pc, et, rp)
				if err != nil {
					h.logger.Error("pipeline processing failed",
						zap.Int64("event_id", eid),
						zap.String("platform", pc),
						zap.Error(err),
					)
				}
			}(eventID, platform, eventType, payloadBytes)
		}
	}

	h.logger.Info("webhook received",
		zap.String("platform", platform),
		zap.String("event_type", eventType),
		zap.String("mapped_topic", mappedTopic),
		zap.String("event_id", eventID),
	)

	response.Success(c, gin.H{
		"status":     "received",
		"event_type": eventType,
		"mapped_to":  mappedTopic,
		"event_id":   eventID,
	})
}

// WebhookConfigResponse represents the webhook configuration status.
type WebhookConfigResponse struct {
	Platform      string `json:"platform"`
	WebhookURL    string `json:"webhook_url"`
	IsConfigured  bool   `json:"is_configured"`
	LastEventAt   string `json:"last_event_at,omitempty"`
	LastEventType string `json:"last_event_type,omitempty"`
}

// GetConfig GET /api/v1/platform-webhooks/config
// Returns webhook configuration status for all registered platforms.
func (h *webhookHandler) GetConfig(c *gin.Context) {
	if h.db == nil {
		response.Error(c, http.StatusInternalServerError, "database not configured")
		return
	}

	// Get all unique platforms from webhook_event_log
	type logSummary struct {
		Platform      string
		LastEventAt   string
		LastEventType string
	}
	var summaries []logSummary
	h.db.Model(&WebhookLog{}).
		Select("platform, MAX(created_at) as last_event_at, "+
			"(SELECT event_type FROM webhook_event_log w2 WHERE w2.platform = webhook_event_log.platform ORDER BY created_at DESC LIMIT 1) as last_event_type").
		Group("platform").
		Find(&summaries)

	// Build config for all registered platforms
	adapters := ListAdapters()
	var configs []WebhookConfigResponse
	for code := range adapters {
		config := WebhookConfigResponse{
			Platform:     code,
			WebhookURL:   fmt.Sprintf("/api/webhooks/%s", code),
			IsConfigured: true,
		}
		for _, s := range summaries {
			if s.Platform == code {
				config.LastEventAt = s.LastEventAt
				config.LastEventType = s.LastEventType
				break
			}
		}
		configs = append(configs, config)
	}

	response.Success(c, configs)
}

// TestEventRequest is the payload for testing a webhook event.
type TestEventRequest struct {
	Platform string                 `json:"platform" binding:"required"`
	EventType string                `json:"event_type" binding:"required"`
	Payload  map[string]interface{} `json:"payload"`
}

// TestEvent POST /api/v1/platform-webhooks/test-event
// Manually triggers a test webhook event for debugging.
func (h *webhookHandler) TestEvent(c *gin.Context) {
	var req TestEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	payload := req.Payload
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["event_type"] = req.EventType

	mappedTopic := ""
	if t, ok := eventTopicMap[req.EventType]; ok {
		mappedTopic = t
	} else {
		mappedTopic = "platform.event." + req.EventType
	}

	eventID, err := h.bus.Publish(c.Request.Context(), mappedTopic, "webhook:test:"+req.Platform, payload)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, fmt.Sprintf("publish failed: %v", err))
		return
	}

	h.logger.Info("test webhook event published",
		zap.String("platform", req.Platform),
		zap.String("event_type", req.EventType),
		zap.String("mapped_topic", mappedTopic),
		zap.String("event_id", eventID),
	)

	response.Success(c, gin.H{
		"status":      "published",
		"event_type":  req.EventType,
		"mapped_to":   mappedTopic,
		"event_id":    eventID,
	})
}

// detectEventType attempts to determine the event type from a raw webhook payload.
// Platforms use different field names — check common patterns.
func detectEventType(raw map[string]interface{}) string {
	// Direct event_type field.
	if et, ok := raw["event_type"].(string); ok && et != "" {
		return et
	}
	// Ozon-style: notification_type
	if nt, ok := raw["notification_type"].(string); ok && nt != "" {
		m := map[string]string{
			"price_changed":   "price_changed",
			"stock_changed":   "inventory_changed",
			"item_updated":    "listing_changed",
			"item_state":      "listing_state_changed",
			"item_created":    "listing_live",
			"item_rejected":   "listing_blocked",
			"item_moderation": "listing_moderation",
		}
		if t, ok := m[nt]; ok {
			return t
		}
		return nt
	}
	// Shopee-style: type field
	if t, ok := raw["type"].(string); ok && t != "" {
		m := map[string]string{
			"ITEM_STATUS_CHANGE":   "listing_state_changed",
			"ITEM_PRICE_UPDATE":    "price_changed",
			"ITEM_STOCK_UPDATE":    "inventory_changed",
			"ORDER_CREATED":        "order_placed",
		}
		if et, ok := m[t]; ok {
			return et
		}
		return t
	}
	return ""
}
