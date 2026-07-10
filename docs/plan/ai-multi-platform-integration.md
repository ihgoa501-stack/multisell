# Implementation Plan: AI-Powered Multi-Platform Integration Layer

## Overview

Replace hand-written field-mapping per platform with an AI-powered transformation layer. Each platform gets a thin (~150 line) communication adapter, and fields/events/categories are mapped via LLM prompts. The profit pipeline switches from `estimated_profit` (mock) to real settlement data. Ozon is the first platform ported to the new architecture as validation; Amazon follows.

## Architecture Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| First validation platform | Ozon | Existing working adapter → can verify AI mapper output against real data side-by-side before cutting over |
| Prompt storage | Go constants | Type safety, versioned with code, simpler deploy for single-dev team. No hot-reload needed at this stage |
| Batch window | 30 seconds | Balances latency vs cost; tunable via config after first month of production data |
| Model locking | Lock to current model | Prevents behavioral drift on mapping output between deploys. Pin in `aimapper/mapper.go` |
| Money validation | Stark bounds check | Every mapped numeric value checked: `0 < price < 100×cost`, reject + flag if out of bounds |
| Ozon transition | Parallel run first | Old adapter + new AI mapper both run; results compared; cut over only when accuracy >99% for 100+ events |
| Deterministic fast path | In-process LRU cache | Keyed by (platform, event_type, input_sha256). Simple field extraction (order_sn, dates, status codes) caught here — no LLM needed |

## Open Questions (pending user input)

1. **利润验证账号？** 是否有真实 Ozon 卖家账号可以用来对账验证 new path vs old path 的利润数值？
2. **Amazon SP-API 注册？** Amazon adapter 需要卖家账号 + SP-API 开发者注册（$39.99/月）。要不要先搞定这个？

## Task List

### Phase 0: Foundation

#### Task 1: Create `platform_raw_event` table and Go model

**Description:** Add the migration for the raw event store table and the corresponding Go model. This is the persistence backbone for the AI mapper — every platform event gets stored here before AI transformation.

**Acceptance criteria:**
- [ ] Migration `000070_platform_raw_event.up.sql` creates the table with correct columns, indexes, and foreign key
- [ ] Go model `RawEvent` in `internal/domain/integrations/model.go` or new `raw_event.go` with GORM tags
- [ ] Down migration drops the table
- [ ] Migration runs cleanly on dev database (test with existing data)

**Verification:**
- [ ] `go build ./...` passes
- [ ] Test: create a RawEvent, read it back, verify all fields

**Dependencies:** None

**Files likely touched:**
- `backend-go/migrations/000070_platform_raw_event.up.sql`
- `backend-go/migrations/000070_platform_raw_event.down.sql`
- `backend-go/internal/domain/integrations/raw_event.go` (NEW)

**Estimated scope:** Small (2 files, 1 new)

---

#### Task 2: Build AI mapper core framework

**Description:** Build the `aimapper/` package: the `Mapper` interface, `SchemaValidator` (output type + bounds checking), `StrategyDetector` (decides deterministic vs LLM path), and the LRU deterministic cache. Wire them together into a `MapEvent()` entry point.

**Acceptance criteria:**
- [ ] `Mapper` interface: `MapEvent(ctx, raw: RawEvent) → (domain: DomainModel, confidence: float64, error)`
- [ ] `StrategyDetector` correctly identifies deterministic mappings (simple field extraction) vs LLM-needed mappings (complex structure, free text)
- [ ] `SchemaValidator` rejects: missing required fields, wrong types, numeric values outside configured bounds (e.g., negative prices, prices > 1M)
- [ ] LRU cache stores deterministic mappings by (platform, event_type, input_sha256) and returns cached result when same input seen again
- [ ] Cache respects max size (configurable, default 10k entries)
- [ ] All three components have unit tests

**Verification:**
- [ ] `go test ./internal/domain/integrations/aimapper/...` passes
- [ ] Benchmark: deterministic path < 1ms, cache hit < 100μs

**Dependencies:** None (doesn't need real data or DB)

**Files likely touched:**
- `backend-go/internal/domain/integrations/aimapper/mapper.go` (NEW)
- `backend-go/internal/domain/integrations/aimapper/schema.go` (NEW)
- `backend-go/internal/domain/integrations/aimapper/cache.go` (NEW)
- `backend-go/internal/domain/integrations/aimapper/prompt.go` (NEW) — base prompt template structure
- `backend-go/internal/domain/integrations/aimapper/mapper_test.go` (NEW)
- `backend-go/internal/domain/integrations/aimapper/cache_test.go` (NEW)

**Estimated scope:** Medium (4 files + tests)

---

#### Task 3: Wire raw event capture into webhook handler

**Description:** Update the existing webhook handler so that when a platform sends a webhook, the raw payload is stored in `platform_raw_event` before (or regardless of) any AI processing. This ensures zero data loss — every event that arrives is at least captured in its native format.

**Acceptance criteria:**
- [ ] Every incoming webhook (any platform) stores the raw payload as a `RawEvent` row
- [ ] The `event_type` field is auto-detected as best-effort (existing `detectEventType`), with fallback to `"unknown"`
- [ ] The existing webhook processing pipeline (event bus publish) continues to work unchanged — this is additive, not replacing anything
- [ ] New webhook log includes `raw_event_id` linking to the stored raw event

**Verification:**
- [ ] Send a test webhook via `POST /api/v1/platform-webhooks/test-event` → `platform_raw_event` has the record
- [ ] Existing webhook → event bus path still works (verify with log message)
- [ ] `go test ./internal/domain/integrations/...` passes

**Dependencies:** Task 1 (raw event table must exist)

**Files likely touched:**
- `backend-go/internal/domain/integrations/webhook.go`
- `backend-go/internal/domain/integrations/webhook_test.go`
- `backend-go/internal/domain/integrations/raw_event.go` (extend if needed)

**Estimated scope:** Small (2 files modified)

---

### Checkpoint 1: Foundation Complete
- [ ] Raw event table exists, migration clean
- [ ] AI mapper creates + validates + caches output
- [ ] Webhook stores raw payloads without breaking existing pipeline
- [ ] All tests green

---

### Phase 1: Ozon — First Platform Migration

#### Task 4: Slim Ozon adapter to communication-only

**Description:** Strip field-mapping logic from `ozon.go`. The adapter becomes a thin layer that handles HTTP communication (auth, JSON calls, pagination, retry) and returns **raw JSON bytes**. The `FetchOrders`, `FetchSettlements`, `FetchReturns` and other data-read methods change their return type to `([]byte, error)` — the caller (AI mapper) handles transformation.

**Acceptance criteria:**
- [ ] Ozon adapter methods return raw JSON (`[]byte`) instead of parsed Go structs
- [ ] All auth + HTTP + pagination logic preserved
- [ ] Adapter line count drops from ~593 to ~180 (auth + HTTP helpers + one raw call per endpoint)
- [ ] `PlatformAdapter` interface updated — or better: add a `FetchRaw(ctx, endpoint, payload) → ([]byte, error)` method to the interface so existing callers don't break
- [ ] Existing direct callers of Ozon adapter's structured returns are updated to go through the mapper

**Verification:**
- [ ] `go build ./...` passes
- [ ] Any existing tests that use Ozon adapter's structured returns are updated
- [ ] Manual: hit Ozon API through slim adapter, verify raw JSON is valid

**Dependencies:** Task 2 (mapper framework exists)

**Files likely touched:**
- `backend-go/internal/domain/integrations/ozon.go` (heavy rewrite)
- `backend-go/internal/domain/integrations/adapter.go` (add `FetchRaw` to interface)
- `backend-go/internal/domain/integrations/types.go` (maybe — simplify old types if they're no longer used externally)
- `backend-go/internal/domain/integrations/ozon_test.go`
- Any file that calls `OzonAdapter.ListProducts()` or similar structured methods

**Estimated scope:** Medium (3-5 files)

---

#### Task 5: Write Ozon mapping prompts and register with mapper

**Description:** For each Ozon endpoint (orders, settlements, returns, products, webhook events), write the mapping prompt that tells the AI how to transform the raw JSON into the internal domain model. Register the prompts with the mapper framework. Add deterministic fast-paths for simple extractions (order_sn, status codes).

**Acceptance criteria:**
- [ ] Prompt for Ozon `FetchOrders` output → `[]PlatformOrder` — verified against 5 known Ozon JSON fixtures
- [ ] Prompt for Ozon `FetchSettlements` output → `[]PlatformSettlement` — verified against 5 fixtures
- [ ] Prompt for Ozon `FetchReturns` output → `[]PlatformReturn` — verified against 3 fixtures
- [ ] Prompt for Ozon webhook event type detection — replaces `detectEventType` Ozon entries
- [ ] Deterministic fast-path hits for: `posting_number → order_sn`, `status` (known enum → internal status), dates (ISO format unchanged)
- [ ] Test fixtures committed: `aimapper/testdata/ozon_order_*.json` + expected output
- [ ] Accuracy test: `go test -run TestOzonMapping` runs all fixtures through mapper, reports accuracy % per field

**Verification:**
- [ ] `go test ./internal/domain/integrations/aimapper/... -run TestOzon` passes with ≥95% accuracy on fixtures
- [ ] Each fixture maps without hitting LLM if it matches a deterministic pattern (verify via metrics)

**Dependencies:** Task 2, Task 4

**Files likely touched:**
- `backend-go/internal/domain/integrations/aimapper/platform/ozon.go` (NEW — Ozon prompt constants)
- `backend-go/internal/domain/integrations/aimapper/prompt.go` (update — register Ozon prompts)
- `backend-go/internal/domain/integrations/aimapper/testdata/ozon_order_001.json` (NEW)
- `backend-go/internal/domain/integrations/aimapper/testdata/ozon_order_001_mapped.json` (NEW)
- `backend-go/internal/domain/integrations/aimapper/testdata/ozon_settlement_001.json` (NEW)
- `backend-go/internal/domain/integrations/aimapper/mapper_test.go` (add Ozon accuracy tests)

**Estimated scope:** Medium (4-6 files)

---

#### Task 6: Wire Ozon raw events through mapper pipeline

**Description:** Complete the pipeline: when a raw Ozon event arrives (webhook or polled), it flows through the mapper and the domain model result is persisted. This is the first end-to-end integration test of the new architecture.

**Acceptance criteria:**
- [ ] Incoming Ozon webhook → stored as RawEvent → processed by mapper → domain model saved
- [ ] Scheduled Ozon order sync uses new path (slim adapter → raw store → mapper → domain model)
- [ ] Old mapping path removed or demoted to logging-only comparison (side-by-side for verification)
- [ ] Processing latency: end-to-end from raw event to domain model ≤ 30s for batch, ≤ 3s for single webhook

**Verification:**
- [ ] Run Ozon sync via scheduler, verify domain DB tables (sales_order, profit_summary) have the same data as before
- [ ] Compare old-path result vs new-path result for 10 Ozon orders — must match field-by-field
- [ ] `go test ./internal/domain/integrations/...` passes

**Dependencies:** Task 1, Task 3, Task 4, Task 5

**Files likely touched:**
- `backend-go/internal/domain/integrations/service.go` (update Ozon sync to use mapper pipeline)
- `backend-go/internal/domain/integrations/webhook.go` (trigger mapper after storing raw)
- `backend-go/internal/domain/integrations/aimapper/mapper.go` (expose `MapAndPersist()` or equivalent)
- `backend-go/internal/domain/integrations/integrations_test.go` (new integration test)

**Estimated scope:** Medium (3-4 files)

---

### Checkpoint 2: Ozon Migrated
- [ ] All Ozon data flows through AI mapper
- [ ] Output matches old path 100% on 10+ sampled orders/settlements
- [ ] No regression in webhook → event bus path
- [ ] `go test ./...` passes
- [ ] Dashboard still shows correct numbers (nothing broke)
- [ ] Human review: inspect 10 mapped events, confirm quality

---

### Phase 2: Real Profit Data

#### Task 7: Build profit calculation from real settlement data

**Description:** Create a new profit calculation pipeline that uses AI-mapped settlement data (from `platform_raw_event` where `event_type='settlement'` and `mapping_status='mapped'`) instead of `estimated_profit` from `profit_summary`. Calculate per-SKU profit = revenue - platform_fees - shipping - returns.

**Acceptance criteria:**
- [ ] New settlement → profit calculation reads mapped `PlatformSettlement` data (not mock/profit_summary)
- [ ] Fees broken down by type (platform_fee, payment_fee, shipping, refund) — not lump sum
- [ ] Profit data written to a new or extended table (or `profit_summary` with new `source='real'` flag)
- [ ] Backfill: profit for historical Ozon orders recalculated from available settlement data
- [ ] Monthly profit/revenue in Dashboard and DailyBrief uses real data when available, falls back to estimated

**Verification:**
- [ ] Run profit calculation against Ozon settlement data → profit numbers match Ozon's own financial reports within 5%
- [ ] Dashboard shows real profit (not 0) for Ozon-connected account
- [ ] Existing `estimated_profit` pipeline still works (fallback path for non-Ozon platforms)
- [ ] `go test ./internal/domain/price/...` passes

**Dependencies:** Task 5, Task 6

**Files likely touched:**
- `backend-go/internal/domain/price/service.go` (add `CalculateRealProfit(settlements)` method)
- `backend-go/internal/domain/price/model.go` (extend ProfitSummary or add RealProfit model)
- `backend-go/internal/domain/dashboard/service.go` (switch to real data when `source='real'`)
- `backend-go/migrations/000071_profit_recalculation.up.sql` (table adjustments if needed)
- `backend-go/internal/domain/price/price_test.go`

**Estimated scope:** Medium (4-5 files)

---

#### Task 8: Update Dashboard to display real profit

**Description:** The Dashboard and DailyBrief already have the fields for profit/revenue/cost. Update the service layer to source real profit data from the new pipeline when it exists, falling back to `estimated_profit` only when real data isn't available. Platform connection status already works.

**Acceptance criteria:**
- [ ] Dashboard overview shows real` order_total` and `order_revenue` from synced platform data — not from mock
- [ ] DailyBrief's `today_profit` / `month_profit` uses real settlement data
- [ ] `negative_margin_skus` list uses real (not estimated) profit margins
- [ ] When platform is disconnected or data not yet synced, fallback message says "数据同步中" instead of showing 0
- [ ] Frontend shows a small "数据来源: Ozon | Amazon" indicator per number

**Verification:**
- [ ] Load Dashboard with Ozon connected + data synced — numbers look real (non-rounded, vary per day)
- [ ] Disconnect Ozon → numbers show "同步中" or clear empty state — not $0.00
- [ ] `go build ./...` passes

**Dependencies:** Task 7

**Files likely touched:**
- `backend-go/internal/domain/dashboard/service.go` (2-3 queries updated)
- `backend-go/internal/domain/dashboard/model.go` (maybe add a `DataSources` field)
- `frontend-next/src/app/(main)/dashboard/page.tsx` (display data source tag)

**Estimated scope:** Medium (3 files)

---

### Checkpoint 3: Real Profit Displayed
- [ ] Ozon-connected account shows real profit numbers
- [ ] Dashboard fallback to "同步中" when no real data
- [ ] Manual Ozon → dashboard comparison passes sniff test
- [ ] All backend tests pass

---

### Phase 3: Amazon Adapter

#### Task 9: Build thin Amazon communication adapter

**Description:** Create a real (non-stub) Amazon SP-API adapter. This covers auth (IAM roles, LWA, OAuth token exchange), product listing fetch, order polling, and settlement report download. It returns raw JSON bytes — no field mapping.

**Acceptance criteria:**
- [ ] Amazon SP-API auth flow works: developer registration → IAM → LWA → token
- [ ] `FetchRaw` for orders (SP-API `getOrders` endpoint) returns valid JSON
- [ ] `FetchRaw` for settlements (SP-API `getFinancialEvents`) returns valid JSON
- [ ] Token refresh handled transparently (expiry detection + auto-refresh before 401)
- [ ] Rate limiting: Amazon's per-seller-profile quota respected (no 429s under normal load)
- [ ] Sandbox mode available for testing without real API calls
- [ ] Ozon adapter-level code reuse: HTTP pool, auth storage, error handling shared where possible

**Verification:**
- [ ] `TestAmazonFetchRaw` with cached sandbox response returns expected JSON shape
- [ ] `TestAmazonAuth` verifies token flow (requires real/mocked credentials)
- [ ] `go build ./...` passes

**Dependencies:** Task 4 (slim adapter pattern proven)

**Files likely touched:**
- `backend-go/internal/domain/integrations/amazon.go` (full rewrite from stub)
- `backend-go/internal/domain/integrations/amazon_test.go`
- `backend-go/internal/domain/integrations/registry.go` (already registered, no change)
- `backend-go/internal/domain/integrations/types.go` (maybe — Amazon-specific request types)

**Estimated scope:** Medium-Large (2-3 files, but complex auth logic)

---

#### Task 10: Write Amazon mapping prompts

**Description:** Following the Ozon pattern, write mapping prompts for Amazon order JSON → `PlatformOrder`, settlement reports → `PlatformSettlement`, and returns → `PlatformReturn`. Amazon's data format is more complex (nested objects, multiple currency fields, tax breakdowns).

**Acceptance criteria:**
- [ ] Amazon order mapping prompt handles: `getOrders` response format (flat vs FBA), multi-item orders, tax breakdown
- [ ] Amazon settlement prompt handles: Settlement Report format (tab-delimited or financial events JSON) → `PlatformSettlement` with proper fee breakdown
- [ ] Test fixtures committed with 5 Amazon order samples + 3 settlement samples
- [ ] Deterministic fast-path for simple extractions (order_id, dates, currency)
- [ ] Money validation catches: price < 0 (reject), fee > order total (reject with flag)

**Verification:**
- [ ] `go test -run TestAmazonMapping` passes with ≥95% accuracy on fixtures
- [ ] Manual: compare 5 mapped Amazon orders against Amazon Seller Central UI

**Dependencies:** Task 9, Task 5 (pattern to follow)

**Files likely touched:**
- `backend-go/internal/domain/integrations/aimapper/platform/amazon.go` (NEW)
- `backend-go/internal/domain/integrations/aimapper/prompt.go` (register Amazon prompts)
- `backend-go/internal/domain/integrations/aimapper/testdata/amazon_order_001.json` (NEW)
- `backend-go/internal/domain/integrations/aimapper/testdata/amazon_settlement_001.json` (NEW)
- `backend-go/internal/domain/integrations/aimapper/mapper_test.go` (add Amazon accuracy tests)

**Estimated scope:** Medium (4-5 files)

---

#### Task 11: Wire Amazon into end-to-end pipeline

**Description:** Connect the Amazon adapter → raw event store → mapper → domain model pipeline. Set up Amazon order sync on a schedule (same pattern as existing Ozon SyncOzonOrders at 15-minute cadence).

**Acceptance criteria:**
- [ ] Amazon order sync runs on schedule (every 30 min — Amazon is less real-time than Ozon)
- [ ] Mapped Amazon orders land in `sales_order` table with correct platform tag
- [ ] Profit from Amazon settlement data calculated and displayed on Dashboard
- [ ] Amazon connection status shows in Dashboard's platform_connections
- [ ] Error handling: rate-limit backoff, auth failure notification, sync_status=error with readable message

**Verification:**
- [ ] Run Amazon sync with sandbox/mocked data → orders appear in DB
- [ ] Dashboard shows Amazon revenue alongside Ozon (or fallback "同步中")
- [ ] Rate-limit simulation → adapter backs off and retries, doesn't crash
- [ ] `go test ./internal/domain/integrations/...` passes

**Dependencies:** Task 8, Task 9, Task 10

**Files likely touched:**
- `backend-go/internal/domain/integrations/service.go` (add Amazon sync)
- `backend-go/internal/domain/dashboard/service.go` (already multi-platform — verify nothing breaks)
- `backend-go/internal/domain/integrations/integrations_test.go` (Amazon integration tests)

**Estimated scope:** Medium (2-3 files)

---

### Checkpoint 4: Amazon Live
- [ ] Amazon orders syncing, profit real
- [ ] Dashboard shows numbers from both Ozon and Amazon
- [ ] One Ozon account + one Amazon account both connected and showing real data
- [ ] Manual verification against platform UIs passes
- [ ] `go test ./...` passes

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| AI hallucinated prices ($1 → $100) | High — monetary loss | `SchemaValidator` rejects prices outside bounds (0 < price < 100×cost). Human review of first 100 events per platform. |
| AI mapping latency too high for webhook | Medium — UX degradation | Deterministic fast-path handles 80% in <1ms. LLM path runs async (store raw → return 200 → process later). Webhook response never blocks on LLM. |
| Amazon SP-API rate limiting blocks sync | Medium — stale data | Adapter queues requests within per-profile quota. Backoff + retry with exponential. Alert when sync lag > 1 hour. |
| LLM model update breaks existing prompts | Medium — silent regression | Pin model version in code. Accuracy tests run CI-style: if accuracy drops below threshold, deployment blocked. |
| Ozon API changes break slimming | Medium — Ozon goes down | Parallel run: old path continues working. New path errors are logged, not blocking. Cutover is a config flag. |
| Cost overrun on LLM calls | Medium — bill shock | LRU cache, deterministic pre-filter, batch window of 30s, per-event cost logged + alert threshold. Hard cap via env var. |

## Summary

```
Phase 0: Foundation     (Tasks 1-3)   —  mapper framework, raw event store, webhook wiring
Phase 1: Ozon Migration (Tasks 4-6)   —  slim adapter, Ozon prompts, end-to-end pipeline
Phase 2: Real Profit    (Tasks 7-8)   —  settlement → profit calculation, dashboard display
Phase 3: Amazon         (Tasks 9-11)  —  Amazon thin adapter, prompts, pipeline wiring

Total: 11 tasks, 4 checkpoints
Estimated: ~4-6 weeks for a single developer (depending on Amazon SP-API registration)
```

The plan is designed so **every task leaves the system in a working state** — no half-built infrastructure that blocks other work. Ozon continues to work through Phase 0 and the early part of Phase 1.
