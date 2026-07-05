> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Shipping Rate Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let operators import carrier/forwarder quote sheets into MultiSell and use country-specific shipping rules in calculations.

**Architecture:** Add optional `zone_id` to `ShippingQuoteRule`. Existing rules with `zone_id = null` remain global channel rules; imported rate rows create provider, channel, destination zone, and a zone-specific rule. Shipping calculation filters rules by the matched destination zone first, falling back to global rules when no zone-specific rule exists.

**Tech Stack:** FastAPI, SQLAlchemy async ORM, Alembic, openpyxl/csv, pytest/httpx, Vue 3, Naive UI.

---

## Scope

- Import `.xlsx` and `.csv` files through `POST /api/shipping/import-rules`.
- Required import columns:
  - `provider_name`
  - `channel_name`
  - `country_code`
  - `rule_type`
- Supported optional columns:
  - `provider_code`, `channel_code`, `volumetric_divisor`, `cargo_types`, `currency`
  - `estimated_delivery_min`, `estimated_delivery_max`
  - `priority`, `fixed_fee`, `per_kg_price`, `first_kg`, `first_price`
  - `additional_kg`, `additional_price`, `minimum_charge`
  - `surcharge_fixed`, `fuel_surcharge_pct`, `rounding_increment`
- Do not implement real carrier APIs.
- Do not implement label/tracking/reconciliation.

## Tasks

### Task 1: Zone-Specific Rule Backend

- [ ] Add tests proving US and DE on the same channel can use different rates.
- [ ] Add nullable `zone_id` to `ShippingQuoteRule`.
- [ ] Add Alembic migration `20260615_01_add_shipping_rule_zone_id.py`.
- [ ] Update rule serializers to expose `zone_id` and `country_code`.
- [ ] Update calculation to pick active zone-specific rules first, then global rules.

### Task 2: Import API

- [ ] Add tests for `.xlsx` import creating provider/channel/zone/rule.
- [ ] Add tests for invalid rows returning row-level errors.
- [ ] Implement `ImportService.parse_file`.
- [ ] Implement `ImportService.import_rules`.
- [ ] Add `POST /api/shipping/import-rules` with `shipping:manage` permission and audit log.

### Task 3: Frontend

- [ ] Add `shippingApi.importRules(file)`.
- [ ] Add upload button to `ShippingManage.vue`.
- [ ] Show import summary and errors.
- [ ] Reload providers/channels/rules after successful import.

### Task 4: Verification

Run:

```bash
cd backend && python3 -m alembic upgrade head
cd backend && python3 -m pytest tests/test_shipping_rate_import.py tests/test_shipping_calculation.py tests/test_shipping_management.py -q
cd backend && python3 -m pytest -q
cd frontend && npm run build
git diff --check
```
