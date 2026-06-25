package eventbus

import (
	"errors"
	"fmt"
)

// ErrSchemaValidation is returned when an event payload fails schema validation.
var ErrSchemaValidation = errors.New("eventbus: payload failed schema validation")

// TopicSchema validates event payloads for a given topic.
type TopicSchema interface {
	Validate(payload map[string]interface{}) error
}

// SchemaRegistry maps topic patterns to their TopicSchema validators.
// Lookup uses glob matching (the same matchTopic function the bus uses for
// subscriptions), so a pattern like "stock.*" matches "stock.alert".
type SchemaRegistry struct {
	schemas map[string]TopicSchema
}

// NewSchemaRegistry creates an empty SchemaRegistry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		schemas: make(map[string]TopicSchema),
	}
}

// Register binds a schema to a topic pattern. Topics are matched using the
// same glob rules as subscription patterns (e.g. "order.*").
func (r *SchemaRegistry) Register(topic string, schema TopicSchema) {
	r.schemas[topic] = schema
}

// Schema returns the schema matching the given topic, or false if none match.
func (r *SchemaRegistry) Schema(topic string) (TopicSchema, bool) {
	for pattern, schema := range r.schemas {
		if matchTopic(pattern, topic) {
			return schema, true
		}
	}
	return nil, false
}

// WithSchema returns a BusOption that configures payload schema validation.
func WithSchema(sr *SchemaRegistry) BusOption {
	return func(b *Bus) {
		b.schemaRegistry = sr
	}
}

// ---------------------------------------------------------------------------
// Concrete schema implementations
// ---------------------------------------------------------------------------

// StockAlertPayload validates payloads for stock alert events.
// Required fields: sku_id (string), quantity (int), threshold (int).
type StockAlertPayload struct{}

func (s StockAlertPayload) Validate(payload map[string]interface{}) error {
	return validateFields(payload,
		typedField{name: "sku_id", kind: fieldString},
		typedField{name: "quantity", kind: fieldInt},
		typedField{name: "threshold", kind: fieldInt},
	)
}

// ProfitWatchPayload validates payloads for profit watch events.
// Required fields: order_id (string), profit_margin (number), threshold (number).
type ProfitWatchPayload struct{}

func (s ProfitWatchPayload) Validate(payload map[string]interface{}) error {
	return validateFields(payload,
		typedField{name: "order_id", kind: fieldString},
		typedField{name: "profit_margin", kind: fieldFloat},
		typedField{name: "threshold", kind: fieldFloat},
	)
}

// ComplianceCheckPayload validates payloads for compliance check events.
// Required fields: listing_id (string), platform (string), rule_id (string).
type ComplianceCheckPayload struct{}

func (s ComplianceCheckPayload) Validate(payload map[string]interface{}) error {
	return validateFields(payload,
		typedField{name: "listing_id", kind: fieldString},
		typedField{name: "platform", kind: fieldString},
		typedField{name: "rule_id", kind: fieldString},
	)
}

// ---------------------------------------------------------------------------
// Internal field validation helpers
// ---------------------------------------------------------------------------

type fieldKind int

const (
	fieldString fieldKind = iota
	fieldInt
	fieldFloat
)

type typedField struct {
	name string
	kind fieldKind
}

func validateFields(payload map[string]interface{}, fields ...typedField) error {
	for _, f := range fields {
		val, ok := payload[f.name]
		if !ok {
			return fmt.Errorf("%w: missing required field %q", ErrSchemaValidation, f.name)
		}
		switch f.kind {
		case fieldString:
			if _, ok := val.(string); !ok {
				return fmt.Errorf("%w: field %q must be a string", ErrSchemaValidation, f.name)
			}
		case fieldInt:
			switch v := val.(type) {
			case int:
				// ok
			case float64:
				// JSON unmarshalling produces float64; accept whole numbers.
				if v != float64(int(v)) {
					return fmt.Errorf("%w: field %q must be an integer", ErrSchemaValidation, f.name)
				}
			default:
				return fmt.Errorf("%w: field %q must be an integer", ErrSchemaValidation, f.name)
			}
		case fieldFloat:
			switch val.(type) {
			case float64:
				// ok
			case int:
				// ok — Go callers may pass int where float is expected
			default:
				return fmt.Errorf("%w: field %q must be a number", ErrSchemaValidation, f.name)
			}
		}
	}
	return nil
}
