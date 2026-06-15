# Product Shipping Calculator Prefill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one-click freight estimate entry points from product list and product detail, passing product package dimensions, package weight, cargo type, and product context into the shipping calculator.

**Architecture:** This is a frontend-only workflow improvement. Reuse the existing manual shipping calculator query prefill support instead of adding backend APIs. Product list and product detail should build `/shipping/calculator` query params from already returned product logistics fields.

**Tech Stack:** Vue 3 Composition API, Vue Router, Naive UI, existing FastAPI product/shipping APIs.

---

## Current Problem

The product list already has a button:

```ts
router.push(`/shipping/calculator?product_id=${row.id}`)
```

But `ShippingCalculator.vue` does not read `product_id`. It only supports direct package query params:

```text
length_cm
width_cm
height_cm
weight_kg
country
cargo_type
quantity
sku_id
```

So the current product list button does not actually prefill dimensions or weight.

Product detail has a logistics card but no direct `试算运费` action.

---

## Target Behavior

From product list:
- If product has complete package logistics data, show active `运费试算` button.
- Clicking it routes to:

```text
/shipping/calculator?length_cm=30&width_cm=20&height_cm=10&weight_kg=0.8&cargo_type=normal&quantity=1&source_product_id=123&source_product_name=xxx
```

- Shipping calculator opens in manual mode.
- Package fields and cargo type are prefilled.
- User only needs to enter destination country and click `计算运费`.

If product logistics data is incomplete:
- Keep a visible action, but route to edit page or show warning.
- Recommended behavior:
  - Button text: `补物流`
  - Click: `/products/{id}/edit`
  - Message is not required because the action name is explicit.

From product detail:
- Logistics card should show:
  - `试算运费` if package logistics data is complete.
  - `补齐物流数据` if incomplete.
- `试算运费` routes to the same calculator query URL.

From shipping calculator:
- If `source_product_name` exists, show a small source hint:

```text
来源商品：xxx
```

- Do not auto-calculate. Operator still chooses country and clicks calculate.

---

## Files To Modify

Frontend:
- Modify: `frontend/src/views/product/ProductList.vue`
  - Replace current `product_id` calculator route with package query route.
  - Add reusable helper functions for completeness and query building.

- Modify: `frontend/src/views/product/ProductDetail.vue`
  - Add `试算运费` action in logistics card when logistics data is complete.
  - Add helper to build calculator route from `detail.product`.

- Modify: `frontend/src/views/shipping/ShippingCalculator.vue`
  - Read optional `source_product_id` and `source_product_name`.
  - Show source product hint.
  - Keep existing package query prefill behavior.

Docs:
- Modify: `docs/PROJECT_STATUS.md`
- Optional modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`

Backend:
- No backend change required for this phase.

---

## Route Contract

Product pages should pass:

```text
length_cm=<package_length_cm>
width_cm=<package_width_cm>
height_cm=<package_height_cm>
weight_kg=<package_weight_kg>
cargo_type=<cargo_type>
quantity=1
source_product_id=<product.id>
source_product_name=<product.name>
```

Do not pass `product_id` alone because the calculator currently cannot resolve product data from backend.

Use `router.push({ path: '/shipping/calculator', query })` instead of hand-written string concatenation. This avoids broken URLs when product names contain spaces, Chinese characters, or symbols.

---

## Implementation Tasks

### Task 1: Product List Route Prefill

**Files:**
- Modify: `frontend/src/views/product/ProductList.vue`

- [ ] **Step 1: Add logistics completeness helper**

Add below existing helper functions:

```ts
function hasCompletePackage(row: any) {
  return !!row.package_length_cm
    && !!row.package_width_cm
    && !!row.package_height_cm
    && !!row.package_weight_kg
}
```

- [ ] **Step 2: Add calculator query builder**

Add:

```ts
function buildCalculatorQuery(row: any) {
  return {
    length_cm: String(row.package_length_cm),
    width_cm: String(row.package_width_cm),
    height_cm: String(row.package_height_cm),
    weight_kg: String(row.package_weight_kg),
    cargo_type: row.cargo_type || 'normal',
    quantity: '1',
    source_product_id: String(row.id),
    source_product_name: row.name || '',
  }
}
```

- [ ] **Step 3: Add navigation helper**

Add:

```ts
function goShippingCalculator(row: any) {
  if (!hasCompletePackage(row)) {
    router.push(`/products/${row.id}/edit`)
    return
  }
  router.push({
    path: '/shipping/calculator',
    query: buildCalculatorQuery(row),
  })
}
```

- [ ] **Step 4: Replace current `运费试算` action**

Replace:

```ts
h(NButton, { size: 'small', ghost: true, type: 'info', onClick: () => router.push(`/shipping/calculator?product_id=${row.id}`) }, { default: () => '运费试算' }),
```

with:

```ts
h(
  NButton,
  {
    size: 'small',
    ghost: true,
    type: hasCompletePackage(row) ? 'info' : 'warning',
    onClick: () => goShippingCalculator(row),
  },
  { default: () => hasCompletePackage(row) ? '运费试算' : '补物流' },
),
```

- [ ] **Step 5: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
built
```

---

### Task 2: Product Detail Calculator Action

**Files:**
- Modify: `frontend/src/views/product/ProductDetail.vue`

- [ ] **Step 1: Add product completeness helper**

Add:

```ts
function hasCompletePackage(product: any) {
  return !!product?.package_length_cm
    && !!product?.package_width_cm
    && !!product?.package_height_cm
    && !!product?.package_weight_kg
}
```

- [ ] **Step 2: Add detail calculator navigation**

Add:

```ts
function goShippingCalculator() {
  const product = detail.value.product
  if (!hasCompletePackage(product)) {
    router.push(`/products/${productId}/edit`)
    return
  }
  router.push({
    path: '/shipping/calculator',
    query: {
      length_cm: String(product.package_length_cm),
      width_cm: String(product.package_width_cm),
      height_cm: String(product.package_height_cm),
      weight_kg: String(product.package_weight_kg),
      cargo_type: product.cargo_type || 'normal',
      quantity: '1',
      source_product_id: String(product.id),
      source_product_name: product.name || '',
    },
  })
}
```

- [ ] **Step 3: Update logistics card action**

Current logistics card only shows `补齐物流数据` when incomplete.

Replace action template with:

```vue
<template #action>
  <n-space>
    <n-button
      v-if="detail.product?.logistics_status === 'complete'"
      size="small"
      type="info"
      @click="goShippingCalculator"
    >
      试算运费
    </n-button>
    <n-button
      v-else
      size="small"
      type="warning"
      @click="router.push(`/products/${productId}/edit`)"
    >
      补齐物流数据
    </n-button>
  </n-space>
</template>
```

- [ ] **Step 4: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
built
```

---

### Task 3: Shipping Calculator Source Product Hint

**Files:**
- Modify: `frontend/src/views/shipping/ShippingCalculator.vue`

- [ ] **Step 1: Add source product state**

Add:

```ts
const sourceProduct = ref<{ id?: string; name?: string }>({})
```

- [ ] **Step 2: Parse source query params**

Inside `onMounted`, add:

```ts
if (q.source_product_id || q.source_product_name) {
  sourceProduct.value = {
    id: typeof q.source_product_id === 'string' ? q.source_product_id : undefined,
    name: typeof q.source_product_name === 'string' ? q.source_product_name : undefined,
  }
}
```

- [ ] **Step 3: Show source hint in calculator form**

Below the mode switch, add:

```vue
<n-alert
  v-if="sourceProduct.id || sourceProduct.name"
  type="info"
  style="margin-bottom: 12px;"
  :show-icon="false"
>
  来源商品：{{ sourceProduct.name || `ID ${sourceProduct.id}` }}
</n-alert>
```

- [ ] **Step 4: Keep manual mode behavior**

Do not auto-calculate.

Reason:
- Product does not know destination country.
- Operator may need to choose cargo type or quantity.

- [ ] **Step 5: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
built
```

---

### Task 4: Documentation Update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Optional modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`

- [ ] **Step 1: Update project status**

Add:

```markdown
### 商品到运费计算器预填

已支持：
- 商品列表完整物流数据商品可一键进入运费计算器。
- 商品详情物流卡片可一键试算运费。
- 跳转时自动带入包装长、宽、高、重量、货品类型和商品来源。
- 物流数据不完整时引导回商品编辑页补齐数据。
```

- [ ] **Step 2: Update logistics spec if needed**

Add:

```markdown
商品页跳转运费计算器时使用 query 参数传入包装字段，不通过 `product_id` 让计算器二次查询商品。这样计算器保持纯试算工具，不依赖商品接口，也不写入业务数据。
```

---

## Verification Commands

Run:

```bash
cd frontend && npm run build
```

Run:

```bash
git diff --check
```

Optional backend sanity check, because this phase should not touch backend:

```bash
cd backend && python3 -m pytest tests/test_shipping_calculation.py -q
```

Expected final state:
- Frontend build passes.
- `git diff --check` passes.
- Product list `运费试算` sends package query params, not only `product_id`.
- Product detail logistics card has `试算运费` when complete.
- Shipping calculator displays source product hint and prefilled manual package fields.

---

## Manual Browser Check

1. Login as admin.
2. Open `/products`.
3. Find a product with `可计算运费`.
4. Click `运费试算`.
5. Confirm `/shipping/calculator` opens in manual mode.
6. Confirm length, width, height, weight, and cargo type are prefilled.
7. Confirm source product hint is visible.
8. Enter country, for example `US`.
9. Click `计算运费`.
10. Return to `/products`.
11. Find a product with `缺物流数据`.
12. Click `补物流`.
13. Confirm it opens the product edit page.
14. Open a complete product detail page.
15. Confirm logistics card shows `试算运费`.

---

## Handoff Prompt For Another Agent

```text
请阅读并严格执行这个规划文档：
/Users/lc/multisell/docs/superpowers/plans/2026-06-15-product-shipping-prefill.md

目标是打通商品页到运费计算器：商品列表和商品详情一键进入物流计算器，并自动带入包装长宽高、包装重量、货品类型、数量和商品来源。不要新增后端接口，不要只传 product_id；必须使用计算器已经支持的 query 参数完成预填。

执行要求：
1. 不要重构无关模块。
2. 不要回滚其他人的改动。
3. 只改前端和文档，除非发现必须修复的阻断问题。
4. 完成后运行：cd frontend && npm run build；git diff --check。
5. 汇报改动文件、构建结果、是否有遗留风险。
```

---

## Next Planning Queue

After this workflow is implemented, plan these in order:

1. **Shipping quote comparison snapshot**
   - Save manual calculation result as a quote comparison record for later review.

2. **SKU-level logistics override cleanup**
   - Make SKU package fields visible in SKU list/detail and allow SKU-specific freight prefill.

3. **Shipping import preview**
   - Validate supplier quotation sheets before committing rules.

4. **Platform order ingestion**
   - Bring real platform orders into the system so shipping/profit workflows can run on real data.

