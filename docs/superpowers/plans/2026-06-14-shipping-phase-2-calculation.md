> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Shipping Phase 2 Calculation Implementation Plan

> **For Reasonix / coding agent:** Execute this plan with TDD. Do not refactor unrelated code. Keep changes scoped to the shipping provider/channel/rule data model, shipping calculation, simple frontend management/calculator UI, permissions, tests, and docs.

**Goal:** Build MultiSell's first shipping-cost calculation system: maintain logistics providers/channels/rules and calculate comparable shipping quotes from SKU/package data.

**Architecture:** Add a new backend module `backend/app/shipping/` with standard `router.py`, `schemas.py`, `service.py`, `__init__.py`. Add shipping tables for providers, channels, destination zones, and quote rules. Use the existing Product/SKU logistics fields from Phase 1 as calculation input. Keep order shipping snapshots, real carrier API integrations, labels, tracking, and packing optimization out of this phase.

**Tech Stack:** FastAPI, SQLAlchemy async, Alembic, PostgreSQL, pytest, Vue 3, Naive UI, Vite.

---

## Must Read First

Read these files before editing:

- `docs/LOGISTICS_AND_SHIPPING_PRD.md`
- `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`
- `docs/LOGISTICS_QUOTE_RULE_EXAMPLES.md`
- `docs/superpowers/plans/2026-06-14-logistics-attributes-phase-1.md`
- `docs/PROJECT_STATUS.md`
- `docs/DEVELOPMENT_GUIDE.md`
- `docs/PERMISSIONS_AND_AUDIT.md`
- `backend/app/models.py`
- `backend/app/core/service.py`
- `backend/app/sku/router.py`
- `backend/app/auth/dependencies.py`
- `frontend/src/router/index.ts`
- `frontend/src/components/Layout.vue`
- `frontend/src/api/index.ts`

## Non-Negotiable Rules

- Write failing tests first.
- Verify focused tests fail for the expected reason before implementation.
- Do not implement real logistics carrier API calls.
- Do not implement order shipping snapshots in this phase.
- Do not implement packing / carton optimization.
- Do not change platform order sync.
- Do not reuse existing `Supplier` as logistics provider. Use independent `ShippingProvider`.
- Use existing Product/SKU package fields from Phase 1.
- Do not use `Sku.weight` as package weight.
- Do not expose or log secrets.
- Preserve `AUTH_ENABLED=False` behavior for existing tests.

---

## Scope

### In Scope

- `ShippingProvider` management.
- `ShippingChannel` management.
- `ShippingZone` destination country coverage.
- `ShippingQuoteRule` management.
- `POST /api/shipping/calculate`.
- Multiple channel quote comparison sorted by total shipping fee.
- Backend tests for quote calculation behavior.
- Simple frontend pages:
  - shipping provider/channel/rule maintenance
  - shipping calculator
- Permissions:
  - `shipping:view`
  - `shipping:manage`
  - `shipping:calculate`
- Documentation updates.

### Out Of Scope

- Real carrier APIs.
- Tracking number / label generation.
- Order shipping quote snapshot.
- Multi-package packing optimization.
- Platform order import.
- Financial reconciliation.

---

## Data Model

### `ShippingProvider`

Purpose: logistics provider / carrier / freight forwarder.

Suggested table: `shipping_provider`

Fields:

- `id`
- `name`
- `code`
- `contact`
- `phone`
- `remark`
- `status`
- `created_at`
- `updated_at`

### `ShippingChannel`

Purpose: a provider's specific shipping product.

Suggested table: `shipping_channel`

Fields:

- `id`
- `provider_id`
- `name`
- `code`
- `volumetric_divisor`
- `cargo_types` JSON list, e.g. `["normal", "battery"]`
- `estimated_delivery_min`
- `estimated_delivery_max`
- `currency`
- `sort_order`
- `status`
- `created_at`
- `updated_at`

### `ShippingZone`

Purpose: destination coverage for a channel.

Suggested table: `shipping_zone`

Fields:

- `id`
- `channel_id`
- `country_code`
- `postal_code_from`
- `postal_code_to`
- `status`
- `created_at`
- `updated_at`

### `ShippingQuoteRule`

Purpose: quote rule for a channel.

Suggested table: `shipping_quote_rule`

Fields:

- `id`
- `channel_id`
- `rule_type`
- `priority`
- `min_weight_kg`
- `max_weight_kg`
- `first_kg`
- `first_price`
- `additional_kg`
- `additional_price`
- `fixed_fee`
- `per_kg_price`
- `minimum_charge`
- `tier_config` JSON
- `surcharge_fixed`
- `fuel_surcharge_pct`
- `rounding_increment`
- `remark`
- `status`
- `created_at`
- `updated_at`

Supported `rule_type` values:

- `fixed_plus_per_kg`
- `first_weight_plus_increment`
- `tiered_weight`

Do not implement `manual_table` in this phase unless all required tests are already green and the implementation remains small.

---

## Calculation Rules

### Package Resolution

Input:

- `sku_id`
- `quantity`
- `destination_country`
- optional `postal_code`
- optional `cargo_type`

Package data:

- If SKU has all four override fields valid, use SKU package data:
  - `sku_length_cm`
  - `sku_width_cm`
  - `sku_height_cm`
  - `sku_weight_kg`
- Otherwise use product package data:
  - `package_length_cm`
  - `package_width_cm`
  - `package_height_cm`
  - `package_weight_kg`
- If fallback package data is incomplete, return a clear business error.
- Do not use `Sku.weight` as package weight.

### Weight Formula

```text
actual_weight_kg = package_weight_kg * quantity
volumetric_weight_kg = package_length_cm * package_width_cm * package_height_cm * quantity / volumetric_divisor
chargeable_weight_kg = max(actual_weight_kg, volumetric_weight_kg)
rounded_chargeable_weight_kg = ceil(chargeable_weight_kg / rounding_increment) * rounding_increment
```

### Channel Filtering

A channel is available only when:

- provider is active
- channel is active
- at least one active zone matches `destination_country`
- `cargo_type` is supported by channel `cargo_types`
- at least one active quote rule matches the rounded chargeable weight

### Fee Calculation

#### `fixed_plus_per_kg`

```text
base_fee = fixed_fee + rounded_chargeable_weight_kg * per_kg_price
```

#### `first_weight_plus_increment`

```text
if rounded_chargeable_weight_kg <= first_kg:
    base_fee = first_price
else:
    additional_units = ceil((rounded_chargeable_weight_kg - first_kg) / additional_kg)
    base_fee = first_price + additional_units * additional_price
```

#### `tiered_weight`

Use `tier_config` JSON:

```json
[
  {"min_kg": 0, "max_kg": 0.5, "price": 35},
  {"min_kg": 0.5, "max_kg": 1, "price": 48},
  {"min_kg": 1, "max_kg": 2, "price": 70}
]
```

Find the matching tier and use its fixed `price`.

### Final Fee

```text
fee_after_minimum = max(base_fee, minimum_charge) if minimum_charge exists else base_fee
fuel_surcharge_fee = (fee_after_minimum + surcharge_fixed) * fuel_surcharge_pct / 100
total_shipping_fee = fee_after_minimum + surcharge_fixed + fuel_surcharge_fee
```

Return detailed calculation fields for every channel result.

---

## API Draft

All endpoints use `/api/shipping`.

Provider:

- `GET /shipping/providers`
- `POST /shipping/providers`
- `PUT /shipping/providers/{provider_id}`
- `DELETE /shipping/providers/{provider_id}`

Channel:

- `GET /shipping/channels`
- `POST /shipping/channels`
- `PUT /shipping/channels/{channel_id}`
- `DELETE /shipping/channels/{channel_id}`

Zone:

- `POST /shipping/channels/{channel_id}/zones`
- `GET /shipping/channels/{channel_id}/zones`
- `DELETE /shipping/zones/{zone_id}`

Rule:

- `POST /shipping/channels/{channel_id}/rules`
- `GET /shipping/channels/{channel_id}/rules`
- `PUT /shipping/rules/{rule_id}`
- `DELETE /shipping/rules/{rule_id}`

Calculation:

- `POST /shipping/calculate`

Calculation request:

```json
{
  "sku_id": 42,
  "quantity": 2,
  "destination_country": "US",
  "postal_code": "10001",
  "cargo_type": "normal"
}
```

Calculation response:

```json
{
  "sku_id": 42,
  "quantity": 2,
  "destination_country": "US",
  "package": {
    "source": "sku",
    "length_cm": 30,
    "width_cm": 20,
    "height_cm": 10,
    "weight_kg": 0.8
  },
  "results": [
    {
      "provider_id": 1,
      "provider_name": "云途物流",
      "channel_id": 1,
      "channel_name": "美国普货",
      "currency": "CNY",
      "actual_weight_kg": 1.6,
      "volumetric_weight_kg": 2.0,
      "chargeable_weight_kg": 2.0,
      "base_shipping_fee": 92.0,
      "minimum_applied": false,
      "surcharge_fee": 0,
      "fuel_surcharge_fee": 0,
      "total_shipping_fee": 92.0,
      "estimated_delivery_min": 7,
      "estimated_delivery_max": 15,
      "calculation_detail": "固定费8 + 计费重2.0kg × 42 = 92"
    }
  ]
}
```

---

## Permissions And Audit

Use:

- `shipping:view` for provider/channel/zone/rule read endpoints.
- `shipping:manage` for provider/channel/zone/rule create/update/delete endpoints.
- `shipping:calculate` for `POST /shipping/calculate`.

Write audit logs for successful management changes:

- `module="shipping_provider"`, actions `create`, `update`, `delete`
- `module="shipping_channel"`, actions `create`, `update`, `delete`
- `module="shipping_zone"`, actions `create`, `delete`
- `module="shipping_quote_rule"`, actions `create`, `update`, `delete`

Do not write audit logs for pure calculation unless the project owner explicitly asks to keep calculation history.

---

## Frontend Scope

Add simple UI only. Do not over-design.

Suggested files:

- Create: `frontend/src/views/shipping/ShippingManage.vue`
- Create: `frontend/src/views/shipping/ShippingCalculator.vue`
- Create: `frontend/src/router/modules/shipping.ts`
- Create or Modify: `frontend/src/api/modules/shipping.ts`

UI requirements:

- Shipping management page:
  - provider list
  - channel list
  - zone list per channel
  - quote rule list per channel
  - simple modals/forms for create/update
- Shipping calculator page:
  - input SKU ID, quantity, destination country, optional postal code, cargo type
  - show results table sorted by total fee
  - show actual weight, volumetric weight, chargeable weight, total fee, time range

Permission metadata:

- management route requires `shipping:manage`
- calculator route requires `shipping:calculate`

---

## TDD Task Breakdown

### Task 1: Backend Calculation Tests

**Files:**

- Create: `backend/tests/test_shipping_calculation.py`

Write tests first for:

- product package fallback when SKU override is absent
- SKU package override when all four SKU fields are present
- missing package data blocks calculation
- volumetric weight greater than actual weight
- actual weight greater than volumetric weight
- `fixed_plus_per_kg`
- `first_weight_plus_increment`
- `tiered_weight`
- minimum charge
- fixed surcharge
- fuel surcharge percentage
- inactive provider/channel/rule excluded
- cargo type mismatch excluded
- destination country mismatch excluded
- results sorted by `total_shipping_fee`

Run expected failing test:

```bash
cd backend && python3 -m pytest tests/test_shipping_calculation.py -q
```

Expected: fails because shipping module/tables/endpoints do not exist yet.

### Task 2: Backend Management API Tests

**Files:**

- Create: `backend/tests/test_shipping_management.py`

Test:

- provider create/list/update/delete
- channel create/list/update/delete
- zone create/list/delete
- rule create/list/update/delete
- no token returns 401 when `AUTH_ENABLED=True`
- user without permission returns 403
- granted `shipping:manage` can write
- granted `shipping:view` can read
- successful writes create operation logs

Run expected failing test:

```bash
cd backend && python3 -m pytest tests/test_shipping_management.py -q
```

### Task 3: Alembic Migration And Models

**Files:**

- Create: `backend/alembic/versions/20260614_02_add_shipping_tables.py`
- Modify: `backend/app/models.py`

Add SQLAlchemy models and migration for:

- `ShippingProvider`
- `ShippingChannel`
- `ShippingZone`
- `ShippingQuoteRule`

Run:

```bash
cd backend && python3 -m alembic upgrade head
```

Expected: migration succeeds.

### Task 4: Shipping Backend Module

**Files:**

- Create: `backend/app/shipping/__init__.py`
- Create: `backend/app/shipping/router.py`
- Create: `backend/app/shipping/schemas.py`
- Create: `backend/app/shipping/service.py`

Implement:

- CRUD services.
- calculation service.
- package data resolution.
- rule application.
- channel filtering.
- sorted results.
- permission dependencies.
- audit logs for write operations.

Run:

```bash
cd backend && python3 -m pytest tests/test_shipping_calculation.py tests/test_shipping_management.py -q
```

Expected: all focused shipping tests pass.

### Task 5: Frontend API And Routes

**Files:**

- Create: `frontend/src/api/modules/shipping.ts`
- Create: `frontend/src/router/modules/shipping.ts`

Implement:

- `shippingApi` provider/channel/zone/rule CRUD.
- `shippingApi.calculate`.
- routes for management and calculator pages.
- route meta permissions.

Run:

```bash
cd frontend && npm run build
```

Expected: build passes or fails only because views are not created yet.

### Task 6: Frontend Pages

**Files:**

- Create: `frontend/src/views/shipping/ShippingManage.vue`
- Create: `frontend/src/views/shipping/ShippingCalculator.vue`

Implement simple pages:

- Do not build complex nested UI.
- Use tables and modals/forms.
- Calculator can accept SKU ID manually in first version.
- Results table must show provider, channel, actual weight, volumetric weight, chargeable weight, total fee, currency, delivery time.

Run:

```bash
cd frontend && npm run build
```

Expected: build passes.

### Task 7: Documentation Updates

**Files:**

- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`
- Modify: `docs/PERMISSIONS_AND_AUDIT.md`
- Optional Modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`

Update:

- Shipping Phase 2 status.
- New permission codes:
  - `shipping:view`
  - `shipping:manage`
  - `shipping:calculate`
- Verification results.

### Task 8: Final Verification

Run:

```bash
cd backend && python3 -m pytest -q
cd frontend && npm run build
git diff --check
```

Expected:

- all backend tests pass
- frontend build passes
- no whitespace errors

---

## Acceptance Criteria

This phase is complete only when:

- shipping provider/channel/zone/rule tables exist and migrate successfully
- management CRUD endpoints exist
- shipping calculate endpoint returns multi-channel comparison
- calculation supports fixed-plus-kg, first-weight-increment, tiered-weight
- calculation supports volumetric weight, chargeable weight, rounding, minimum charge, fixed surcharge, fuel surcharge
- inactive providers/channels/rules are excluded
- cargo type mismatch is excluded
- destination country mismatch is excluded
- backend focused shipping tests pass
- full backend tests pass
- frontend build passes
- docs are updated

---

## Prompt To Hand To Reasonix

```text
请实现 MultiSell Shipping Phase 2：运费计算基础系统。

先阅读：
- docs/superpowers/plans/2026-06-14-shipping-phase-2-calculation.md
- docs/LOGISTICS_AND_SHIPPING_PRD.md
- docs/LOGISTICS_SHIPPING_TECH_SPEC.md
- docs/LOGISTICS_QUOTE_RULE_EXAMPLES.md
- docs/PROJECT_STATUS.md
- docs/DEVELOPMENT_GUIDE.md

严格按计划执行：
- 先写测试，再实现。
- 新增 ShippingProvider、ShippingChannel、ShippingZone、ShippingQuoteRule。
- 新增 backend/app/shipping 模块。
- 实现 POST /api/shipping/calculate。
- 支持体积重、计费重、取整、固定费+每kg、首重续重、阶梯价、最低收费、附加费、燃油附加费、货品类型过滤、国家过滤。
- 前端做简单管理页和计算页。
- 接入 shipping:view、shipping:manage、shipping:calculate。
- 不接真实物流 API。
- 不做订单运费快照。
- 不做装箱优化。

完成后运行：
cd backend && python3 -m pytest -q
cd frontend && npm run build
git diff --check
```
