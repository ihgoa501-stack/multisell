package aftersales

import "context"

// OrderWriter provides order mutation operations for aftersales.
type OrderWriter interface {
	CancelOrder(ctx context.Context, orderID int64, operator string, remark string) error
}

// EventPublisher is the minimal interface for publishing supply chain events.
type EventPublisher interface {
	Publish(ctx context.Context, topic, source string, payload map[string]interface{}) (string, error)
}

// DeliveryChecker checks whether a shipment has been delivered for a given
// transaction on a platform. Used by the dispute evaluation engine.
type DeliveryChecker interface {
	IsDelivered(ctx context.Context, transactionID, platform string) (bool, error)
}
