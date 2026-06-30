package approval

import (
	"context"
	"fmt"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewAgentDecisionSubscriber creates an event bus handler that listens for
// agent.decided.* events and auto-creates approval requests when an agent's
// confidence is below 0.7.
func NewAgentDecisionSubscriber(db *gorm.DB, logger *zap.Logger) func(ctx context.Context, evt eventbus.Event) error {
	return func(ctx context.Context, evt eventbus.Event) error {
		payload := evt.Payload
		if payload == nil {
			return nil
		}

		// Extract confidence score from payload
		var confidence float64
		if c, ok := payload["confidence"].(float64); ok {
			confidence = c
		} else if c, ok := payload["confidence_score"].(float64); ok {
			confidence = c
		} else {
			// No confidence score, skip
			return nil
		}

		if confidence >= 0.7 {
			// High confidence, no approval needed
			return nil
		}

		// Extract agent ID from topic (agent.decided.{agent_id}.{decision_point})
		agentID := ""
		decisionPoint := ""
		if a, ok := payload["agent_id"].(string); ok {
			agentID = a
		}
		if d, ok := payload["decision_point"].(string); ok {
			decisionPoint = d
		}
		if agentID == "" && decisionPoint == "" {
			// Try to extract from topic
			_, err := fmt.Sscanf(evt.Topic, "agent.decided.%s.%s", &agentID, &decisionPoint)
			if err != nil {
				logger.Warn("cannot parse agent decision topic, skipping",
					zap.String("topic", evt.Topic),
				)
				return nil
			}
		}

		// Determine request type from decision point
		requestType := mapDecisionPointToRequestType(decisionPoint)

		// Extract product ID if available
		var productID int64
		if p, ok := payload["product_id"].(float64); ok {
			productID = int64(p)
		} else if p, ok := payload["sku_id"].(float64); ok {
			productID = int64(p)
		}

		// Get old/new values if available
		oldValue := ""
		if v, ok := payload["old_value"].(string); ok {
			oldValue = v
		}
		newValue := ""
		if v, ok := payload["new_value"].(string); ok {
			newValue = v
		}

		// Build reason from payload
		reason := fmt.Sprintf("Agent %s confidence %.2f below threshold 0.70", agentID, confidence)
		if r, ok := payload["reason"].(string); ok && r != "" {
			reason = r
		}

		svc := NewService(db, logger)
		input := &CreateApprovalInput{
			ProductID:   productID,
			RequestType: requestType,
			Requester:   agentID,
			OldValue:    oldValue,
			NewValue:    newValue,
			Reason:      reason,
		}

		req, err := svc.Create(input)
		if err != nil {
			logger.Error("failed to create approval request from agent decision",
				zap.String("topic", evt.Topic),
				zap.String("agent_id", agentID),
				zap.Float64("confidence", confidence),
				zap.Error(err),
			)
			return err
		}

		logger.Info("approval request created from agent decision",
			zap.Int64("approval_id", req.ID),
			zap.String("agent_id", agentID),
			zap.Float64("confidence", confidence),
			zap.String("request_type", requestType),
		)
		return nil
	}
}

// mapDecisionPointToRequestType maps agent decision points to approval request types.
func mapDecisionPointToRequestType(decisionPoint string) string {
	switch decisionPoint {
	case "stock_alert":
		return "publish"
	case "listing_optimize":
		return "content_update"
	case "price_review", "profit_check", "discount_risk_check", "discount_check":
		return "price_change"
	case "compliance_check":
		return "content_update"
	case "auto_reply", "system_health", "dashboard_overview", "warehouse_routing", "acos_analysis":
		return "publish"
	default:
		return "publish"
	}
}
