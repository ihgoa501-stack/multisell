# Spec: AI-Powered Multi-Platform Integration Layer

## Objective

Build an AI-powered integration layer that connects LingMirror to multiple e-commerce platforms (Amazon, Ozon, Shopify, Shopee, Lazada) without writing 600+ lines of hand-coded field mapping per platform, and surfaces **real** (not mock) profit data to the user.

### User Story

> As a cross-border seller, I connect my Amazon/Ozon/Shopify store to LingMirror. The system automatically pulls my orders, settlements, and returns, correctly maps them to the internal data model, and shows me my true platform-level profit — not an estimate. When I add a second platform, it works the same way with minimal extra effort.

### Success Criteria

1. **Real data pipeline**: A connected platform's orders, revenue, and fees appear on the Dashboard within minutes — verified against the platform's own UI.
2. **Platform independence**: Adding a new platform requires ~150 lines of Go (HTTP + auth + rate limiting) + ~50 lines of mapping prompts — not 600 lines of adapters.
3. **Per-SKU profit accuracy**: Profit calculated from real settlement data matches the platform's own reports within 5% margin (accounting for currency conversion timing differences).
4. **AI cost control**: Per-event AI mapping cost <$0.001 (cached deterministic rules do 80% of the work; LLM only for the ambiguous 20%).
5. **Ozon continuity**: Ozon's existing working pipeline is **not broken** during the transition — it's either ported to the new architecture with full regression pass, or left untouched and the new system runs in parallel.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go (existing) |
| LLM | Existing provider (`internal/ai/llm_provider.go`) |
| DB | PostgreSQL (existing) |
| Cache | In-process LRU (new, within mapper package) |
| Raw Events | New `platform_raw_event` table |
| Event Bus | Existing event bus for processed events |

## Commands

```bash
# Build (unchanged)
go build -o bin/server cmd/server/main.go

# Run (unchanged)
go run cmd/server/main.go

# Test integration layer
go test ./internal/domain/integrations/...
go test ./internal/domain/integrations/aimapper/...

# Test profit pipeline
go test ./internal/domain/price/...

# Run all backend tests (smoke check)
go test ./...

# AI mapper accuracy test
go test -run TestMapperAccuracy ./internal/domain/integrations/aimapper/ -v
```

## Project Structure

### New files

```
backend-go/internal/domain/integrations/
├── aimapper/                    # NEW: AI transformation layer
│   ├── mapper.go                #   entry point: Map(input) → DomainModel
│   ├── mapper_test.go           #   unit tests + accuracy tests
│   ├── cache.go                 #   deterministic mapping cache (LRU)
│   ├── cache_test.go
│   ├── prompt.go                #   prompt templates per platform
│   ├── prompt_test.go           #   verify prompt outputs match schema
│   ├── schema.go                #   output schema validation (num bounds, req fields)
│   └── platform/                #   platform-specific prompts
│       ├── amazon.go
│       ├── ozon.go
│       └── shopify.go
```

### Modified files

```
backend-go/internal/domain/integrations/
├── adapter.go                   # MINIMAL: remove field mapping from interface (or make optional)
├── ozon.go                      # SLIM DOWN: ~400 lines → ~150 lines (keep HTTP + auth, remove field mapping)
├── amazon.go                    # BUILD OUT: from stub to ~150 lines (HTTP + auth only)
├── types.go                     # ADD: MappedEvent, MappingConfidence types
├── webhook.go                   # UPDATE: store raw payload before AI mapping
├── registry.go                  # UNCHANGED

backend-go/internal/domain/dashboard/
├── service.go                   # UPDATE: use real profit data when available, fall back to estimated

backend-go/internal/domain/price/
├── service.go                   # ADD: profit calculation from real settlement data
├── engine.go                    # UPDATE: accept real cost inputs

backend-go/migrations/
├── 000070_platform_raw_event.up.sql   # NEW: raw event store table
├── 000070_platform_raw_event.down.sql
├── 000071_profit_recalculation.up.sql # NEW: profit table adjustments for real data
├── 000071_profit_recalculation.down.sql
```

### New DB tables

```sql
-- 000070 UP
CREATE TABLE platform_raw_event (
    id            BIGSERIAL PRIMARY KEY,
    platform_code VARCHAR(32)  NOT NULL,
    account_id    BIGINT       NOT NULL REFERENCES platform_integration_account(id),
    event_type    VARCHAR(64)  NOT NULL,   -- 'order', 'settlement', 'return', 'product_change'
    raw_payload   JSONB        NOT NULL,   -- platform-native JSON, verbatim
    mapped_result JSONB,                   -- AI-mapped domain model (nullable — mapping may fail)
    mapping_status VARCHAR(16) NOT NULL DEFAULT 'pending',  -- 'pending' | 'mapped' | 'failed'
    confidence    REAL,                    -- AI confidence score (0.0–1.0), NULL if deterministic
    received_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    mapped_at     TIMESTAMPTZ
);
CREATE INDEX idx_raw_event_platform ON platform_raw_event(platform_code, received_at DESC);
CREATE INDEX idx_raw_event_status ON platform_raw_event(mapping_status);
```

## Code Style

### Mapping prompt template (Go)

Each platform gets a Go constant that acts as a prompt with schema instructions:

```go
// aimapper/platform/ozon.go

// OzonOrderMappingPrompt tells the LLM how to map Ozon posting JSON → internal Order.
// The {raw_json} placeholder is replaced with the actual payload at runtime.
const OzonOrderMappingPrompt = `You are a deterministic data transformer.
Map the following Ozon posting JSON to the internal order structure.

RULES:
- posting_number → order_sn (string, required)
- status → translate Ozon status to internal status using the mapping table below
- analytics_data.delivery_price → shipping_fee (string, keep "%.2f" format)
- financial_data.products → items array, each with sku_code (string), quantity (int), unit_price (string)
- If a product's quantity is 0, skip it
- If delivery_price is missing, use "0.00"

STATUS MAPPING:
  awaiting_packaging → pending
  awaiting_deliver → pending
  delivering → in_transit
  delivered → completed
  cancelled → cancelled

OUTPUT SCHEMA (JSON, must match exactly):
{"order_sn": string, "status": string, "total_amount": string,
 "shipping_fee": string, "items": [{"sku_code": string, "quantity": int, "unit_price": string}]}

INPUT JSON:
{raw_json}
`
```

### When deterministic rules are sufficient (the 80%)

Simple extraction goes through a fast path without LLM:

```go
// cache.go
// DetermineMappingStrategy checks if this payload can be mapped deterministically.
// Returns "deterministic" for known schemas, "llm" for complex/novel ones.
func DetermineMappingStrategy(raw []byte, platform string) string {
    // If the payload matches a known pattern (field names, nesting),
    // use fast path. Otherwise fall back to LLM.
}
```

## Testing Strategy

| Test Level | Location | What |
|-----------|----------|------|
| Unit | `aimapper/*_test.go` | Cache hit/miss, schema validation, strategy detection |
| Accuracy | `aimapper/mapper_test.go` | AI mapping against known-good fixtures: measure accuracy %, reject if <95% |
| Integration | `integrations/integrations_test.go` | Adapter → Mapper pipeline (with cached API responses, no live calls) |
| Smoke | `go test ./...` | No regressions in existing system |
| Manual | N/A | First 100 events per platform: inspect mapping_output vs raw_payload |

### Fixture directory

```
backend-go/internal/domain/integrations/aimapper/testdata/
├── ozon_order_001.json              # Known-good Ozon order
├── ozon_order_001_mapped.json       # Expected mapping result
├── amazon_settlement_001.json       # Amazon settlement report sample
├── amazon_settlement_001_mapped.json
└── ...
```

## Boundaries

### Always do:
- Store raw payloads **before** any AI mapping (recoverable debugging)
- Validate AI output against schema (types, required fields, numeric bounds) — reject if schema fails
- Log raw + mapped + confidence + latency per event (debuggability)
- Cache deterministic mappings (same input shape → same output, skip LLM)
- Run accuracy test suite before deploying a prompt change
- Keep Ozon working in old path until new path passes regression with same output

### Ask first:
- Changing a platform's mapping prompt (review accuracy delta first)
- Adding a new platform adapter (cost estimate + time estimate)
- Increasing per-event AI cost budget
- Changing the raw event table schema (migration plan needed)
- Removing deterministic fallback paths

### Never do:
- Skip schema validation on AI output
- Route money amounts through LLM without bounds checking (e.g., "price must be < 100× cost")
- Delete raw payloads before their mapped counterparts are verified
- Bypass human review on the first 100 events per platform
- Use LLM for auth, token exchange, webhook signature verification, or rate limiting

## Open Questions

1. **哪个平台先打穿？** Ozon 已有工作适配器 → 过渡最平滑，但用户更需要 Amazon。后者前置成本最高（SP-API 注册、IAM、$39.99/月）。
2. **AI 模型版本锁定？** 每次 prompt 变更时要不要固定所用的 LLM model+version？跑了 3 个月后模型升级导致行为漂移是真实风险。
3. **利润真实性如何验证？** 谁负责核对"系统显示的 $103.42 利润"和"Amazon 后台的 $103.42"？首次接入时需要人工对账。
4. **批处理窗口多长？** 收集 30 秒的历史事件一起送 LLM 能降成本，但订单确认有延迟。trade-off 需要定。
5. **Prompt 版本管理？** Prompt 存在 Go 常量里 vs .md 文件 vs 独立 JSON 文件？存 Go 里类型安全但变更要编译，存文件里可以热更新但复杂度高。

## Dependencies

```
Thin Adapters (one per platform)
    ↓
Raw Event Store → [AI Mapper] → Domain Model → DB
                       ↕
                Deterministic Cache (LRU)
```

**Adapter 之间没有依赖关系，可以逐个独立开发。** AI Mapper 依赖 Raw Event Store 就绪。Dashboard/Profit 依赖 Mapper 就绪。
