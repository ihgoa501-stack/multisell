# Shipping Manual Calculator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the shipping calculator so operators can calculate freight by manually entering package dimensions, weight, quantity, destination country, postal code, and cargo type, while keeping the existing SKU-based calculator.

**Architecture:** Keep a single `/shipping/calculate` API and single `ShippingCalculator.vue` page. Extend the request schema to support two calculation modes: `sku` mode resolves package data from SKU/product, and `manual` mode uses package data from the request. Reuse existing channel matching and rule calculation logic to avoid duplicate freight formulas.

**Tech Stack:** FastAPI, Pydantic, SQLAlchemy async, Vue 3 Composition API, Naive UI, pytest, existing shipping module.

---

## Business Requirement

The current shipping calculator is not enough because it only accepts `SKU ID`. Operators need a standalone calculator for quoting or comparing supplier rates before the product/order data is complete.

Required manual inputs:
- Package length in cm
- Package width in cm
- Package height in cm
- Package weight in kg
- Quantity
- Destination country code
- Postal code, optional
- Cargo type

Required output:
- Available logistics providers and channels
- Actual weight
- Volumetric weight
- Chargeable weight
- Base shipping fee
- Surcharge
- Fuel surcharge
- Total shipping fee
- Estimated delivery days
- Calculation detail

Important rule:
- Manual mode must use the same provider/channel/rule matching and pricing engine as SKU mode.
- Manual mode should not create products, SKUs, orders, or inventory records.

---

## Current Code Context

Existing files:

- `backend/app/shipping/schemas.py`
  - `CalculateRequest` currently requires `sku_id`.
  - `PackageInfo` supports `source`, length, width, height, weight.
  - `CalculateResponse` currently requires `sku_id`.

- `backend/app/shipping/service.py`
  - `CalculateService.calculate()` resolves package data by SKU only.
  - `_resolve_package(db, sku_id)` reads SKU package override, then product package fields.
  - `_find_active_channels()` already matches provider/channel/country/cargo type.
  - `_calculate_channel()` already calculates actual weight, volumetric weight, chargeable weight, fees, and totals.

- `backend/app/shipping/router.py`
  - `POST /shipping/calculate` uses `CalculateRequest`.

- `frontend/src/views/shipping/ShippingCalculator.vue`
  - Current page only has SKU ID, quantity, destination country, cargo type, postal code.

- `frontend/src/api/modules/shipping.ts`
  - `shippingApi.calculate(data)` already posts to `/shipping/calculate`.

---

## Target UX

The page should have two modes at the top:

```text
[ 手动输入 ] [ 按 SKU 计算 ]
```

Default mode:

```text
手动输入
```

Manual input form:
- Destination country, required, uppercase automatically
- Postal code, optional
- Cargo type, required, default `normal`
- Quantity, required, default `1`
- Length cm, required
- Width cm, required
- Height cm, required
- Weight kg, required

SKU input form:
- SKU ID, required
- Quantity, required
- Destination country, required
- Postal code, optional
- Cargo type, required

Result area:
- Summary card:
  - Calculation mode: `手动输入` or `SKU`
  - Destination country
  - Quantity
  - Package source:
    - `manual`
    - `sku`
    - `product`
  - Dimensions
  - Package weight
- Result table sorted by total fee ascending.
- Empty state:
  - `没有匹配的物流渠道，请检查目的地国家、货品类型或报价规则。`
- Error state:
  - Show backend validation message.

Usability requirements:
- Manual mode calculate button is disabled until all required fields are valid.
- Country code is normalized to uppercase before submit.
- Numeric inputs must be greater than 0.
- Keep layout dense and operational. This is a business tool, not a marketing screen.

---

## API Design

### Request

Extend `POST /shipping/calculate`.

Manual mode request:

```json
{
  "mode": "manual",
  "quantity": 2,
  "destination_country": "US",
  "postal_code": "90001",
  "cargo_type": "normal",
  "package": {
    "length_cm": 30,
    "width_cm": 20,
    "height_cm": 10,
    "weight_kg": 0.8
  }
}
```

SKU mode request:

```json
{
  "mode": "sku",
  "sku_id": 123,
  "quantity": 2,
  "destination_country": "US",
  "postal_code": "90001",
  "cargo_type": "normal"
}
```

Backward compatibility:
- If `mode` is omitted but `sku_id` exists, treat request as SKU mode.
- Existing SKU calculator tests should continue to pass after small schema updates.

### Response

Update response so `sku_id` is optional:

```json
{
  "mode": "manual",
  "sku_id": null,
  "quantity": 2,
  "destination_country": "US",
  "package": {
    "source": "manual",
    "length_cm": 30,
    "width_cm": 20,
    "height_cm": 10,
    "weight_kg": 0.8
  },
  "results": []
}
```

---

## Files To Modify

Backend:
- Modify: `backend/app/shipping/schemas.py`
- Modify: `backend/app/shipping/service.py`
- Modify: `backend/app/shipping/router.py` only if logging or response behavior needs small changes
- Modify: `backend/tests/test_shipping_calculation.py`

Frontend:
- Modify: `frontend/src/views/shipping/ShippingCalculator.vue`
- Modify: `frontend/src/api/modules/shipping.ts` only if adding TypeScript helper types

Docs:
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`

---

## Implementation Tasks

### Task 1: Backend Schema For Manual Calculation

**Files:**
- Modify: `backend/app/shipping/schemas.py`
- Test: `backend/tests/test_shipping_calculation.py`

- [ ] **Step 1: Add failing backend tests**

Add tests in `backend/tests/test_shipping_calculation.py`:

```python
class TestManualCalculation:

    async def test_manual_package_calculation(self, async_client):
        country = _unique_country()
        _, cid = await _seed_channel(async_client, country, {
            "volumetric_divisor": 6000,
            "fixed_fee": 8,
            "per_kg_price": 42,
            "minimum_charge": 25,
            "cargo_types": ["normal"],
        })

        resp = await async_client.post("/api/shipping/calculate", json={
            "mode": "manual",
            "quantity": 1,
            "destination_country": country,
            "cargo_type": "normal",
            "package": {
                "length_cm": 30,
                "width_cm": 20,
                "height_cm": 10,
                "weight_kg": 0.8,
            },
        })

        body = resp.json()
        assert body["code"] == 200
        assert body["data"]["mode"] == "manual"
        assert body["data"]["sku_id"] is None
        assert body["data"]["package"]["source"] == "manual"
        result = [item for item in body["data"]["results"] if item["channel_id"] == cid][0]
        assert result["actual_weight_kg"] == 0.8
        assert result["volumetric_weight_kg"] == 1.0
        assert result["chargeable_weight_kg"] == 1.0
        assert result["total_shipping_fee"] == 50.0
```

Add validation test:

```python
async def test_manual_package_requires_positive_dimensions(self, async_client):
    resp = await async_client.post("/api/shipping/calculate", json={
        "mode": "manual",
        "quantity": 1,
        "destination_country": "US",
        "cargo_type": "normal",
        "package": {
            "length_cm": 0,
            "width_cm": 20,
            "height_cm": 10,
            "weight_kg": 0.8,
        },
    })
    assert resp.status_code in (200, 422)
    if resp.status_code == 200:
        assert resp.json()["code"] == 400
```

- [ ] **Step 2: Add manual package schema**

In `backend/app/shipping/schemas.py`, add:

```python
from typing import Literal, Optional
```

If `Optional` already exists, only add `Literal`.

Add:

```python
class ManualPackageInput(BaseModel):
    length_cm: float = Field(..., gt=0, description="包装长(cm)")
    width_cm: float = Field(..., gt=0, description="包装宽(cm)")
    height_cm: float = Field(..., gt=0, description="包装高(cm)")
    weight_kg: float = Field(..., gt=0, description="包装重量(kg)")
```

- [ ] **Step 3: Extend `CalculateRequest`**

Replace current request with:

```python
class CalculateRequest(BaseModel):
    mode: Literal["sku", "manual"] = Field("sku", description="计算模式")
    sku_id: Optional[int] = Field(None, description="SKU ID")
    quantity: int = Field(1, ge=1, description="数量")
    destination_country: str = Field(..., min_length=2, max_length=10, description="目的地国家代码")
    postal_code: Optional[str] = Field(None, max_length=20, description="邮编")
    cargo_type: str = Field("normal", description="货品类型")
    package: Optional[ManualPackageInput] = Field(None, description="手动包裹信息")
```

Validation rule:
- `mode="sku"` requires `sku_id`.
- `mode="manual"` requires `package`.

If this project uses Pydantic v2, implement:

```python
from pydantic import BaseModel, Field, model_validator

@model_validator(mode="after")
def validate_mode_payload(self):
    if self.mode == "sku" and not self.sku_id:
        raise ValueError("SKU模式必须填写sku_id")
    if self.mode == "manual" and self.package is None:
        raise ValueError("手动模式必须填写package")
    return self
```

If existing Pydantic version rejects `model_validator`, use a compatible root validator.

- [ ] **Step 4: Extend `CalculateResponse`**

Change:

```python
class CalculateResponse(BaseModel):
    mode: str = "sku"
    sku_id: Optional[int] = None
    quantity: int
    destination_country: str
    package: PackageInfo
    results: list[CalculateResultItem] = []
```

- [ ] **Step 5: Run focused test and confirm failure before service work**

Run:

```bash
cd backend && python3 -m pytest tests/test_shipping_calculation.py::TestManualCalculation -q
```

Expected:

```text
fails because service does not support manual package yet
```

---

### Task 2: Backend Service Manual Package Support

**Files:**
- Modify: `backend/app/shipping/service.py`
- Test: `backend/tests/test_shipping_calculation.py`

- [ ] **Step 1: Extract package resolution by mode**

In `CalculateService.calculate`, replace direct SKU package lookup:

```python
pkg = await _resolve_package(db, req.sku_id)
```

with:

```python
pkg = await _resolve_calculation_package(db, req)
```

Add:

```python
async def _resolve_calculation_package(db: AsyncSession, req: CalculateRequest) -> Optional[dict]:
    if req.mode == "manual":
        if req.package is None:
            return None
        return {
            "source": "manual",
            "length_cm": float(req.package.length_cm),
            "width_cm": float(req.package.width_cm),
            "height_cm": float(req.package.height_cm),
            "weight_kg": float(req.package.weight_kg),
        }
    if req.sku_id is None:
        return None
    return await _resolve_package(db, req.sku_id)
```

- [ ] **Step 2: Keep weight formula identical**

Do not change:

```python
actual_weight = pkg["weight_kg"] * req.quantity
base_volume = pkg["length_cm"] * pkg["width_cm"] * pkg["height_cm"] * req.quantity
```

This ensures manual mode and SKU mode calculate consistently.

- [ ] **Step 3: Update response construction**

Return:

```python
return CalculateResponse(
    mode=req.mode,
    sku_id=req.sku_id,
    quantity=req.quantity,
    destination_country=req.destination_country.upper(),
    package=PackageInfo(
        source=pkg["source"],
        length_cm=pkg["length_cm"],
        width_cm=pkg["width_cm"],
        height_cm=pkg["height_cm"],
        weight_kg=pkg["weight_kg"],
    ),
    results=results,
)
```

- [ ] **Step 4: Preserve existing SKU mode**

Existing requests like this must still work:

```json
{
  "sku_id": 1,
  "quantity": 1,
  "destination_country": "US",
  "cargo_type": "normal"
}
```

Because `mode` defaults to `sku`.

- [ ] **Step 5: Run shipping calculation tests**

Run:

```bash
cd backend && python3 -m pytest tests/test_shipping_calculation.py -q
```

Expected:

```text
passed
```

---

### Task 3: Frontend Calculator Two-Mode UI

**Files:**
- Modify: `frontend/src/views/shipping/ShippingCalculator.vue`

- [ ] **Step 1: Add mode state**

Add:

```ts
const mode = ref<'manual' | 'sku'>('manual')
```

Use a segmented control:

```vue
<n-radio-group v-model:value="mode" name="shipping-calculate-mode">
  <n-radio-button value="manual">手动输入</n-radio-button>
  <n-radio-button value="sku">按 SKU 计算</n-radio-button>
</n-radio-group>
```

- [ ] **Step 2: Split form state**

Use:

```ts
const form = reactive({
  sku_id: null as number | null,
  quantity: 1,
  destination_country: '',
  postal_code: '',
  cargo_type: 'normal',
  length_cm: null as number | null,
  width_cm: null as number | null,
  height_cm: null as number | null,
  weight_kg: null as number | null,
})
```

- [ ] **Step 3: Manual mode fields**

Render when `mode === 'manual'`:

```vue
<n-form-item-gi :span="6" label="包装长">
  <n-input-number v-model:value="form.length_cm" :min="0.01" :precision="2" style="width: 100%;">
    <template #suffix>cm</template>
  </n-input-number>
</n-form-item-gi>
<n-form-item-gi :span="6" label="包装宽">
  <n-input-number v-model:value="form.width_cm" :min="0.01" :precision="2" style="width: 100%;">
    <template #suffix>cm</template>
  </n-input-number>
</n-form-item-gi>
<n-form-item-gi :span="6" label="包装高">
  <n-input-number v-model:value="form.height_cm" :min="0.01" :precision="2" style="width: 100%;">
    <template #suffix>cm</template>
  </n-input-number>
</n-form-item-gi>
<n-form-item-gi :span="6" label="包装重量">
  <n-input-number v-model:value="form.weight_kg" :min="0.01" :precision="3" style="width: 100%;">
    <template #suffix>kg</template>
  </n-input-number>
</n-form-item-gi>
```

- [ ] **Step 4: SKU mode field**

Render SKU ID only when `mode === 'sku'`.

```vue
<n-form-item-gi v-if="mode === 'sku'" :span="6" label="SKU ID">
  <n-input-number v-model:value="form.sku_id" placeholder="输入SKU ID" :min="1" />
</n-form-item-gi>
```

- [ ] **Step 5: Shared fields**

Shared fields:
- Quantity
- Destination country
- Postal code
- Cargo type

Keep destination country normalized:

```ts
function normalizeCountry() {
  form.destination_country = (form.destination_country || '').trim().toUpperCase()
}
```

Call before submit.

- [ ] **Step 6: Add computed validity**

Add:

```ts
const canCalculate = computed(() => {
  if (!form.destination_country || !form.quantity) return false
  if (mode.value === 'sku') return !!form.sku_id
  return !!form.length_cm && !!form.width_cm && !!form.height_cm && !!form.weight_kg
})
```

Button:

```vue
<n-button type="primary" @click="handleCalculate" :loading="loading" :disabled="!canCalculate">
  计算运费
</n-button>
```

- [ ] **Step 7: Build request payload by mode**

In `handleCalculate`:

```ts
normalizeCountry()

const payload = mode.value === 'manual'
  ? {
      mode: 'manual',
      quantity: form.quantity,
      destination_country: form.destination_country,
      postal_code: form.postal_code || undefined,
      cargo_type: form.cargo_type,
      package: {
        length_cm: form.length_cm,
        width_cm: form.width_cm,
        height_cm: form.height_cm,
        weight_kg: form.weight_kg,
      },
    }
  : {
      mode: 'sku',
      sku_id: form.sku_id,
      quantity: form.quantity,
      destination_country: form.destination_country,
      postal_code: form.postal_code || undefined,
      cargo_type: form.cargo_type,
    }
```

- [ ] **Step 8: Update result header**

Header should not assume SKU always exists.

Use:

```vue
📊 计算结果 — {{ result.mode === 'manual' ? '手动包裹' : `SKU ${result.sku_id}` }} × {{ result.quantity }} → {{ result.destination_country }}
```

Package source display:

```ts
function packageSourceLabel(source?: string) {
  const map: Record<string, string> = {
    manual: '手动输入',
    sku: 'SKU包装',
    product: '商品包装',
  }
  return map[source || ''] || source || '-'
}
```

- [ ] **Step 9: Add empty guidance**

When result exists but `result.results.length === 0`, show:

```text
没有匹配的物流渠道，请检查目的地国家、货品类型或报价规则。
```

- [ ] **Step 10: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
built
```

---

### Task 4: Route Query Prefill From Product List

**Files:**
- Modify: `frontend/src/views/shipping/ShippingCalculator.vue`

- [ ] **Step 1: Read query params**

Use `useRoute`.

Support:

```text
/shipping/calculator?length_cm=30&width_cm=20&height_cm=10&weight_kg=0.8&country=US&cargo_type=normal
```

Optional:

```text
/shipping/calculator?sku_id=123&country=US
```

- [ ] **Step 2: Prefill mode**

Rules:
- If query has `sku_id`, set `mode='sku'`.
- If query has package fields, set `mode='manual'`.
- If no query, default manual.

- [ ] **Step 3: Parse numeric values safely**

Add:

```ts
function numberFromQuery(value: unknown) {
  const raw = Array.isArray(value) ? value[0] : value
  if (raw == null || raw === '') return null
  const parsed = Number(raw)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null
}
```

- [ ] **Step 4: Do not auto-calculate**

Prefill only. User should click `计算运费` manually.

Reason:
- Avoid surprising API calls.
- Operator may need to change country or cargo type first.

---

### Task 5: Documentation Update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`

- [ ] **Step 1: Update project status**

Add:

```markdown
### 运费计算器手动试算

已支持：
- 手动输入包装长、宽、高、重量、数量、国家、货品类型计算运费。
- 保留 SKU 计算模式，SKU 模式继续从 SKU 或商品包装字段读取物流数据。
- 手动模式和 SKU 模式共用同一套渠道匹配、体积重、计费重、报价规则计算逻辑。
```

- [ ] **Step 2: Update logistics spec**

Add API note:

```markdown
`POST /api/shipping/calculate` 支持两种模式：

- `mode=sku`：根据 `sku_id` 解析 SKU 包装字段，缺失时回退商品包装字段。
- `mode=manual`：直接使用请求中的 `package.length_cm`、`package.width_cm`、`package.height_cm`、`package.weight_kg` 计算运费，不写入商品或订单数据。
```

---

## Verification Commands

Run all commands before reporting complete:

```bash
cd backend && python3 -m pytest tests/test_shipping_calculation.py -q
```

```bash
cd backend && python3 -m pytest -q
```

```bash
cd frontend && npm run build
```

```bash
git diff --check
```

Expected final state:
- Existing SKU shipping calculation still works.
- Manual package shipping calculation works.
- Frontend calculator defaults to manual input mode.
- Result table still sorts by total fee ascending.
- No unrelated module rewrites.

---

## Manual Browser Check

After implementation:

1. Login as admin.
2. Open `/shipping/calculator`.
3. Confirm default mode is `手动输入`.
4. Fill:
   - Country: `US`
   - Quantity: `1`
   - Cargo type: `普通`
   - Length: `30`
   - Width: `20`
   - Height: `10`
   - Weight: `0.8`
5. Click `计算运费`.
6. Confirm result shows package source `手动输入`.
7. Confirm actual weight, volume weight, chargeable weight, and total fee appear.
8. Switch to `按 SKU 计算`.
9. Fill a valid SKU ID and country.
10. Confirm SKU mode still returns results.

---

## Handoff Prompt For Another Agent

```text
请阅读并严格执行这个规划文档：
/Users/lc/multisell/docs/superpowers/plans/2026-06-15-shipping-manual-calculator.md

目标是升级物流计算页面：默认支持手动输入包装长宽高、重量、数量、国家、货品类型来计算运费，同时保留按 SKU ID 计算。后端必须复用现有渠道匹配和报价规则计算逻辑，不能写入商品、SKU、订单或库存数据。

执行要求：
1. 不要重构无关模块。
2. 不要回滚其他人已有改动。
3. 先补后端测试，再改 schema/service，再改前端页面。
4. 完成后运行文档里的 Verification Commands。
5. 汇报改动文件、测试结果、前端构建结果、遗留风险。
```

---

## Next Planning Queue

After this calculator is implemented, plan these in order:

1. **Product list to calculator prefill**
   - From product list or product detail, one click carries package dimensions into calculator.

2. **Shipping quote comparison snapshot**
   - Save manual calculation result as a quote comparison record for later review.

3. **SKU-level logistics override cleanup**
   - Make SKU logistics fields visible and easy to maintain.

4. **Shipping import preview**
   - Validate supplier quotation sheets before committing rules.

