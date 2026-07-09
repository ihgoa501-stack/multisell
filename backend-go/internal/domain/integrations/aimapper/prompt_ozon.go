package aimapper

// RegisterOzonPrompts registers all Ozon mapping prompts with the given registry.
func RegisterOzonPrompts(r *PromptRegistry) {
	r.Register("ozon", "order", ozonOrderMappingPrompt)
	r.Register("ozon", "settlement", ozonSettlementMappingPrompt)
	r.Register("ozon", "return", ozonReturnMappingPrompt)
}

// OzonOrderMappingPrompt maps Ozon posting JSON to the internal order structure.
// Placeholder {raw_json} is replaced at runtime with the actual payload.
const ozonOrderMappingPrompt = `You are a deterministic data transformer.
Map the following Ozon posting JSON to the internal order structure.

RULES:
- posting_number → order_sn (string, required)
- status → translate using: awaiting_packaging→pending, awaiting_deliver→pending, delivering→in_transit, delivered→completed, cancelled→cancelled
- If status doesn't match any known value, pass it through as-is lowercase
- analytics_data.delivery_price → shipping_fee (string with "%.2f" format, "0.00" if missing)
- financial_data.products → items array, each with: sku_code (string), quantity (int > 0, skip if 0), unit_price (string "%.2f")
- If total_amount is not present, calculate it as sum of items' price × quantity
- Return only the JSON, no explanation, no markdown

OUTPUT SCHEMA (JSON):
{"order_sn": string, "status": string, "total_amount": string, "shipping_fee": string, "items": [{"sku_code": string, "quantity": int, "unit_price": string}]}

INPUT JSON:
{raw_json}`

const ozonSettlementMappingPrompt = `You are a deterministic data transformer.
Map the following Ozon financial transaction JSON to the internal settlement structure.

RULES:
- operation_id → transaction_id (string, required)
- operation_date → occurred_at (string, keep ISO format)
- Map operation_type to transaction_type: sale→order_sale, refund→refund, delivery→shipping_fee, commission→platform_fee, payment_commission→payment_fee
- If operation_type doesn't match, pass it through as-is lowercase
- amount → amount (string "%.2f", absolute value, required)
- currency_code → currency (string, default "RUB")
- description → description (string, empty string if missing)
- posting_number from posting object → order_sn (string, empty string if missing)
- Return only the JSON array, no explanation, no markdown

OUTPUT SCHEMA (JSON):
{"transaction_id": string, "transaction_type": string, "order_sn": string, "amount": string, "currency": string, "occurred_at": string, "description": string}

INPUT JSON:
{raw_json}`

const ozonReturnMappingPrompt = `You are a deterministic data transformer.
Map the following Ozon return JSON to the internal return structure.

RULES:
- return_id → return_id (string, required)
- posting_number → order_sn (string, required)
- sku → sku_code (string, required)
- quantity → quantity (int, required)
- reason → reason (string)
- status → status (string, lowercase)
- created_at → created_at (string, keep ISO format)
- refund_amount → refund_amount (string "%.2f", "0.00" if missing)
- Return only the JSON, no explanation, no markdown

OUTPUT SCHEMA (JSON):
{"return_id": string, "order_sn": string, "sku_code": string, "quantity": int, "reason": string, "status": string, "created_at": string, "refund_amount": string}

INPUT JSON:
{raw_json}`
