// Package aimapper maps raw platform events to domain models.
// ponytail: simple field-copy mapping for now. Replace with AI (LLM) prompt when
// event variety grows beyond what field-mapping can handle.
package aimapper

import (
	"context"
	"encoding/json"
	"fmt"
)

// MapResult holds the output of event-to-domain mapping.
type MapResult struct {
	DomainModel map[string]interface{} `json:"domain_model"`
	TargetTable string                 `json:"target_table"`
	Confidence  float64                `json:"confidence"`
}

// Mapper maps raw platform events to domain models.
type Mapper struct{}

// NewMapper creates a Mapper.
func NewMapper() *Mapper { return &Mapper{} }

// MapEvent maps a raw event payload to a domain model.
// platformCode identifies the source platform (e.g. "ozon").
// eventType identifies the kind of event ("order", "settlement", "return", etc.).
func (m *Mapper) MapEvent(ctx context.Context, platformCode, eventType string, rawPayload json.RawMessage) (*MapResult, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(rawPayload, &raw); err != nil {
		return nil, fmt.Errorf("aimapper: unmarshal raw payload: %w", err)
	}
	switch eventType {
	case "order":
		return m.mapOrder(raw), nil
	case "settlement":
		return m.mapSettlement(raw), nil
	case "return":
		return m.mapReturn(raw), nil
	default:
		return &MapResult{DomainModel: raw, TargetTable: "unknown", Confidence: 0.1}, nil
	}
}

// mapOrder maps Ozon order fields to domain model fields.
func (m *Mapper) mapOrder(raw map[string]interface{}) *MapResult {
	dm := map[string]interface{}{
		"order_no":      raw["posting_number"],
		"status":        raw["status"],
		"total_amount":  raw["total_amount"],
		"shipping_fee":  raw["shipping_fee"],
		"paid_at":       raw["in_process_at"],
		"platform_code": raw["platform_code"],
	}
	if itemsRaw, ok := raw["items"].([]interface{}); ok {
		var items []map[string]interface{}
		for _, it := range itemsRaw {
			if item, ok := it.(map[string]interface{}); ok {
				items = append(items, map[string]interface{}{
					"sku_code":   item["sku_code"],
					"quantity":   item["quantity"],
					"unit_price": item["unit_price"],
				})
			}
		}
		dm["items"] = items
	}
	return &MapResult{DomainModel: dm, TargetTable: "sales_order", Confidence: 0.85}
}

// mapSettlement maps Ozon settlement fields to domain model fields.
func (m *Mapper) mapSettlement(raw map[string]interface{}) *MapResult {
	return &MapResult{
		DomainModel: map[string]interface{}{
			"transaction_id":   raw["operation_id"],
			"transaction_type": raw["operation_type"],
			"order_no":         raw["posting_number"],
			"amount":           raw["amount"],
			"currency":         raw["currency_code"],
			"occurred_at":      raw["operation_date"],
			"description":      raw["description"],
			"platform_code":    raw["platform_code"],
		},
		TargetTable: "settlement_item",
		Confidence:  0.85,
	}
}

// mapReturn maps Ozon return fields to domain model fields.
func (m *Mapper) mapReturn(raw map[string]interface{}) *MapResult {
	dm := map[string]interface{}{
		"return_id":     raw["return_id"],
		"order_no":      raw["posting_number"],
		"sku_code":      raw["sku"],
		"quantity":      raw["quantity"],
		"reason":        raw["reason"],
		"status":        raw["status"],
		"created_at":    raw["created_at"],
		"platform_code": raw["platform_code"],
	}
	if v, ok := raw["refund_amount"]; ok {
		dm["refund_amount"] = v
	}
	return &MapResult{DomainModel: dm, TargetTable: "after_sales_order", Confidence: 0.85}
}
