# Logistics Shipping Documentation Plan

> **For documentation agent:** This is a writing plan, not an implementation task. Do not edit backend or frontend code. Produce clear product and technical documentation for the logistics attributes and shipping-cost calculation system.

**Goal:** Create a complete documentation package that explains how MultiSell should store product logistics attributes and calculate shipping costs from multiple supplier / carrier quotes.

**Audience:** Product owner, future coding agents, backend developers, frontend developers, and operations users who need to maintain shipping quote rules.

**Output Documents:**

- `docs/LOGISTICS_AND_SHIPPING_PRD.md`
- `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`
- `docs/LOGISTICS_QUOTE_RULE_EXAMPLES.md`
- Update `docs/ROADMAP.md` with a short link to the new logistics workstream.

---

## Must Read First

Read these files before drafting:

- `README.md`
- `docs/PROJECT_STATUS.md`
- `docs/DEVELOPMENT_GUIDE.md`
- `docs/ROADMAP.md`
- `backend/app/models.py`
- `backend/app/core/schemas.py`
- `backend/app/sku/schemas.py`
- `backend/app/supplier/schemas.py`
- `backend/app/order/schemas.py`
- `frontend/src/views/product/ProductList.vue`
- `frontend/src/views/product/ProductForm.vue`
- `frontend/src/views/sku/SkuManage.vue`
- `frontend/src/views/order/OrderDetail.vue`

Use existing module names and data concepts from the codebase. Do not invent a different product architecture unless the doc explicitly explains why.

## Writing Rules

- Write in Chinese.
- Use precise business language, not marketing language.
- Separate product decisions from technical implementation details.
- Mark unresolved decisions clearly as `待确认`.
- Do not claim any feature is already implemented unless the current code proves it.
- Keep examples concrete: include sample dimensions, weight, quote rules, countries, and calculation output.
- Avoid writing code except small formulas or JSON-like examples when useful.

---

## Document 1: `docs/LOGISTICS_AND_SHIPPING_PRD.md`

Purpose: Product requirement document for logistics attributes and shipping calculation.

Required sections:

1. **背景**
   - Explain why length, width, height, and weight are mandatory for shipping cost calculation.
   - Explain why product dimensions and package dimensions must be separated.
   - Explain why SKU-level override is needed.

2. **目标**
   - Product / SKU can store logistics attributes.
   - Operators can know which products lack required shipping data.
   - System can calculate estimated shipping cost across supplier / carrier quotes.
   - Orders can store shipping cost snapshots.

3. **非目标**
   - Do not implement real carrier API integration in the first phase.
   - Do not solve warehouse packing optimization in the first phase.
   - Do not build a full accounting system in the first phase.

4. **用户角色**
   - Product operator.
   - Logistics operator.
   - Finance / profit analyst.
   - Admin.

5. **业务流程**
   - Product operator enters product / SKU logistics attributes.
   - Logistics operator maintains supplier channels and quote rules.
   - Operator selects SKU, quantity, destination country, and optional postal code.
   - System calculates chargeable weight.
   - System compares available shipping channels.
   - Operator chooses cheapest, fastest, or manually selected option.
   - Order stores final shipping quote snapshot.

6. **商品物流字段需求**
   - Product dimensions:
     - `product_length_cm`
     - `product_width_cm`
     - `product_height_cm`
     - `product_weight_kg`
   - Package dimensions:
     - `package_length_cm`
     - `package_width_cm`
     - `package_height_cm`
     - `package_weight_kg`
   - SKU override rules:
     - SKU package fields override product defaults.
     - If SKU field is empty, use product default.
     - If both are empty, logistics data is incomplete.

7. **商品列表展示需求**
   - Show package weight.
   - Show package size.
   - Show volumetric weight.
   - Show chargeable weight.
   - Show logistics data status: complete / incomplete.

8. **供应商与物流渠道需求**
   - Supplier can have multiple shipping channels.
   - Channel can support different countries / zones.
   - Channel can define volumetric divisor.
   - Channel can define quote rule type.
   - Channel can define available cargo types: normal, battery, liquid, sensitive.

9. **报价规则需求**
   Document these rule types:
   - Fixed fee plus per-kg price.
   - First weight plus additional weight.
   - Tiered weight price.
   - Minimum charge.
   - Extra surcharge.
   - Fuel surcharge percentage.

10. **运费计算输出**
    Required output fields:
    - supplier
    - channel
    - destination
    - actual weight
    - volumetric weight
    - chargeable weight
    - base shipping fee
    - surcharges
    - total shipping fee
    - currency
    - estimated delivery time
    - calculation explanation

11. **验收标准**
    - Product cannot be considered logistics-complete without package weight and package size.
    - SKU can override product package data.
    - A shipping calculator can compare multiple supplier channels.
    - Calculation result includes explainable fee breakdown.

---

## Document 2: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`

Purpose: Technical specification for later implementation.

Required sections:

1. **Current Code Context**
   - Summarize current Product, Sku, Supplier, Order, and ProductListing concepts.
   - Mention this is a design doc; implementation is not done yet.

2. **Recommended Data Model**
   Describe proposed entities:
   - Product logistics fields.
   - SKU logistics override fields.
   - `ShippingProvider`
   - `ShippingChannel`
   - `ShippingZone`
   - `ShippingQuoteRule`
   - `ShippingQuoteSnapshot`

3. **Field Definitions**
   Include exact field names, units, nullable behavior, and validation:
   - dimensions must be positive decimals.
   - weight must be positive decimal.
   - centimeters and kilograms are canonical internal units.
   - currency code uses ISO-like uppercase string such as `CNY`, `USD`.

4. **Fallback Rules**
   - SKU package field overrides product package field.
   - Product package field is fallback.
   - Product dimensions do not automatically equal package dimensions unless explicitly copied.
   - Missing package data blocks shipping calculation.

5. **Chargeable Weight Formula**
   Include:

   ```text
   volumetric_weight_kg = package_length_cm * package_width_cm * package_height_cm / volumetric_divisor
   chargeable_weight_kg = max(package_weight_kg, volumetric_weight_kg)
   ```

   Explain rounding:
   - default round up to 0.1 kg or provider-specific increment.
   - provider quote rule controls rounding increment.

6. **Quote Rule Types**
   Describe a normalized rule format for:
   - `fixed_plus_per_kg`
   - `first_weight_plus_increment`
   - `tiered_weight`
   - `manual_table`

7. **Calculation Algorithm**
   Step-by-step:
   - Resolve product / SKU package data.
   - Validate destination.
   - Find active channels.
   - Check cargo restrictions.
   - Calculate volumetric weight.
   - Calculate chargeable weight.
   - Apply quote rule.
   - Apply minimum charge.
   - Apply surcharges.
   - Return sorted comparison results.

8. **API Draft**
   Propose endpoints:
   - `GET /api/shipping/providers`
   - `POST /api/shipping/providers`
   - `GET /api/shipping/channels`
   - `POST /api/shipping/channels`
   - `POST /api/shipping/calculate`
   - `POST /api/orders/{id}/shipping-quote`

9. **Frontend Draft**
   Pages / UI:
   - Product form logistics section.
   - SKU manage logistics override section.
   - Shipping provider/channel management page.
   - Shipping calculator panel.
   - Order shipping quote snapshot display.

10. **Testing Plan**
    Include backend tests for:
    - fallback from SKU to product package fields.
    - missing dimensions blocks calculation.
    - volumetric weight greater than actual weight.
    - actual weight greater than volumetric weight.
    - minimum charge.
    - tiered quote.
    - inactive channel excluded.
    - cargo restriction excluded.

---

## Document 3: `docs/LOGISTICS_QUOTE_RULE_EXAMPLES.md`

Purpose: Operator-facing examples showing how different quote rules calculate fees.

Required examples:

1. **Fixed Fee Plus Per Kg**

Example:

```text
Provider: 云途
Channel: 美国普货
Actual weight: 0.8 kg
Package size: 30 x 20 x 10 cm
Volumetric divisor: 6000
Fixed fee: 8 CNY
Per kg: 42 CNY
Minimum charge: 25 CNY
```

Show calculation and final total.

2. **First Weight Plus Additional Weight**

Example:

```text
First 100g: 20 CNY
Additional 100g: 5 CNY
Chargeable weight: 0.35 kg
```

Show rounding to 400g if increment is 100g.

3. **Tiered Weight**

Example:

```text
0-0.5 kg: 35 CNY
0.5-1 kg: 48 CNY
1-2 kg: 70 CNY
```

4. **Multiple Provider Comparison**

Show a table comparing:

- 云途
- 燕文
- DHL

Include cost, time, chargeable weight, and reason for recommendation.

5. **Cargo Restriction Example**

Show why a battery product cannot use a normal goods channel.

---

## Update `docs/ROADMAP.md`

Add a short Phase entry or update existing Phase 5 / next steps:

- Product logistics attributes.
- Shipping provider quote rules.
- Shipping calculator.
- Order shipping quote snapshots.

Keep it short; link to the three new docs.

---

## Acceptance Checklist

The documentation agent is done only when:

- `docs/LOGISTICS_AND_SHIPPING_PRD.md` exists.
- `docs/LOGISTICS_SHIPPING_TECH_SPEC.md` exists.
- `docs/LOGISTICS_QUOTE_RULE_EXAMPLES.md` exists.
- `docs/ROADMAP.md` links to the logistics documentation.
- The PRD clearly says product / SKU package dimensions and weight are mandatory for shipping calculation.
- The tech spec includes chargeable weight formula and quote rule types.
- The examples doc includes at least three concrete shipping calculations.
- No backend or frontend code is modified.
- `git diff --check` passes.

## Suggested Agent Prompt

```text
你接手 MultiSell 的物流和运费文档任务。

只写文档，不写代码。

先阅读：
- docs/superpowers/plans/2026-06-14-logistics-shipping-documentation-plan.md
- docs/PROJECT_STATUS.md
- docs/DEVELOPMENT_GUIDE.md
- docs/ROADMAP.md
- backend/app/models.py
- backend/app/core/schemas.py
- backend/app/sku/schemas.py
- backend/app/supplier/schemas.py
- backend/app/order/schemas.py

产出：
- docs/LOGISTICS_AND_SHIPPING_PRD.md
- docs/LOGISTICS_SHIPPING_TECH_SPEC.md
- docs/LOGISTICS_QUOTE_RULE_EXAMPLES.md
- 更新 docs/ROADMAP.md，加入物流/运费文档链接

要求：
- 中文写作。
- 不修改任何 backend 或 frontend 代码。
- 明确商品和 SKU 必须有包装长宽高重量，用于运费计算。
- 明确体积重、计费重、抛重系数、报价规则、供应商渠道对比。
- 给出具体计算例子。
- 最后运行 git diff --check。
```
