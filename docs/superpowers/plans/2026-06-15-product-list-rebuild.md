# Product List Rebuild Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the product list into an operations-ready product data workspace, with logistics dimensions, weight, completeness filtering, bulk import/export, and clear repair paths for shipping calculation readiness.

**Architecture:** Keep the existing FastAPI + Vue 3 + Naive UI structure. Reuse the current `Product` logistics fields and `product_to_vo` response model, but extend query/filter/export/import behavior and redesign the product list table around shipping-readiness operations. Do not introduce new tables unless a task explicitly says so.

**Tech Stack:** FastAPI, SQLAlchemy async, Pydantic, Alembic, Vue 3 Composition API, Naive UI, openpyxl, pytest, Vitest only if already configured.

---

## Why This Is Needed

The current product list is not suitable for the next business phase.

Current problems:
- The list only shows a generic `物流完整/物流不完整` tag.
- Required shipping fields are hidden from the list: length, width, height, weight.
- Operators cannot filter products that are missing logistics data.
- Export/import templates do not include dimensions, weight, cargo type, or packaging fields.
- The product detail page also does not clearly show logistics readiness.
- Shipping calculation depends on package dimensions and weight, so incomplete product data must be visible and actionable before orders or shipping quotes can be trusted.

Business rule for this phase:
- Shipping calculation should use package dimensions and package weight.
- Product physical dimensions are still useful for product data, but logistics readiness must be based on package length, package width, package height, and package weight.
- Cargo type must be visible because shipping channels may restrict battery, liquid, or sensitive goods.
- Product list should let the operator find and fix incomplete products quickly.

---

## Current Code Context

Relevant existing files:

- `backend/app/models.py`
  - `Product` already has:
    - `product_length_cm`
    - `product_width_cm`
    - `product_height_cm`
    - `product_weight_kg`
    - `package_length_cm`
    - `package_width_cm`
    - `package_height_cm`
    - `package_weight_kg`
    - `cargo_type`

- `backend/app/core/schemas.py`
  - `ProductCreate`, `ProductUpdate`, `ProductVO`, `ProductQuery`.
  - `ProductVO` already returns logistics fields.
  - `ProductQuery` does not support logistics readiness filters.

- `backend/app/core/service.py`
  - `is_product_logistics_complete(product)` checks package fields.
  - `product_to_vo(product)` returns logistics fields and `logistics_status`.
  - `ProductService.list_products()` filters name/category/status/brand only.

- `backend/app/core/router.py`
  - `GET /products` calls `ProductService.list_products`.
  - `GET /products/export`, `GET /products/export-template`, and `POST /products/import` exist.

- `backend/app/core/excel_service.py`
  - Export currently omits logistics fields.
  - Template currently omits logistics fields.
  - Import currently has a column indexing bug risk and does not import logistics fields.

- `frontend/src/views/product/ProductList.vue`
  - Displays ID, name, category, unit, status, logistics tag, AI, platform, created time, actions.
  - Does not display dimensions/weight directly.
  - Does not support logistics completeness filter.

- `frontend/src/views/product/ProductForm.vue`
  - Already supports product dimensions, package dimensions, package weight, and cargo type.

- `frontend/src/views/product/ProductDetail.vue`
  - Does not show a dedicated logistics information card.

---

## Target Product List UX

The product list should become a dense operations table.

Required list columns:
- Selection checkbox
- ID
- Product name
- Category
- Status
- Cargo type
- Product dimensions: `L x W x H cm`
- Product weight: `kg`
- Package dimensions: `L x W x H cm`
- Package weight: `kg`
- Logistics readiness
- Created time
- Actions

Logistics readiness display:
- If complete: green tag `可计算运费`
- If incomplete: warning tag `缺物流数据`
- If incomplete, show missing fields as small text:
  - `缺包装长`
  - `缺包装宽`
  - `缺包装高`
  - `缺包装重量`

Required filters:
- Product name
- Status
- Cargo type
- Logistics status:
  - All
  - Complete
  - Incomplete
- Category and brand filters are optional for this phase if existing selector data is cumbersome; keep backend support.

Required actions:
- Detail
- Edit
- SKU
- Shipping test: route to `/shipping/calculator?product_id=<id>` only if the calculator can safely accept this later. If the current calculator cannot accept product ID, add a disabled button with tooltip `后续接入商品带入`.
- Duplicate
- Delete

Bulk actions:
- Batch status remains.
- Batch delete remains.
- Add `只看缺物流数据` as a quick filter button.
- Add export with current filters.

Responsive behavior:
- Desktop: dense table, horizontal scroll allowed.
- Mobile/narrow width: table may scroll horizontally; do not hide logistics fields silently.

---

## Files To Modify

Backend:
- Modify: `backend/app/core/schemas.py`
  - Add query fields for `cargo_type` and `logistics_status`.
  - Add helper fields to `ProductVO` if needed:
    - `missing_logistics_fields: list[str]`
    - `package_volume_weight_kg: Optional[float]`

- Modify: `backend/app/core/service.py`
  - Add missing logistics field calculation helper.
  - Extend list filters.
  - Return missing fields and volume weight in `product_to_vo`.

- Modify: `backend/app/core/router.py`
  - Accept `cargo_type` and `logistics_status` query params for `/products`.
  - Pass those params to `ProductQuery`.
  - Include those params in export route.

- Modify: `backend/app/core/excel_service.py`
  - Export product dimensions and package dimensions.
  - Export cargo type and logistics status.
  - Template must include required logistics columns.
  - Import must parse logistics columns robustly by header name, not hardcoded indexes.

- Add or modify tests:
  - `backend/tests/test_product_list_logistics.py`
  - Existing product tests may need small updates if schema changed.

Frontend:
- Modify: `frontend/src/views/product/ProductList.vue`
  - Redesign columns.
  - Add cargo type and logistics status filters.
  - Render dimensions and missing fields clearly.
  - Keep existing import/export/delete/status flows.

- Modify: `frontend/src/views/product/ProductDetail.vue`
  - Add a `物流信息` card.

- Modify: `frontend/src/api/index.ts`
  - If needed, type or pass new query params. Existing `productApi.list(params)` may already be sufficient.

Docs:
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`
- Optional modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`

---

## Data Contract

### `GET /products` Query Params

Add:

```text
cargo_type=normal|battery|liquid|sensitive
logistics_status=complete|incomplete
```

Existing params stay:

```text
name
category_id
brand_id
status
page
page_size
```

### Product List Response Fields

Each product row should include:

```json
{
  "id": 1,
  "name": "Example",
  "category_name": "Category",
  "status": 1,
  "status_name": "上架",
  "product_length_cm": 10.0,
  "product_width_cm": 5.0,
  "product_height_cm": 3.0,
  "product_weight_kg": 0.2,
  "package_length_cm": 12.0,
  "package_width_cm": 6.0,
  "package_height_cm": 4.0,
  "package_weight_kg": 0.3,
  "cargo_type": "normal",
  "logistics_status": "complete",
  "logistics_status_name": "物流完整",
  "missing_logistics_fields": [],
  "package_volume_weight_kg": 0.058
}
```

`package_volume_weight_kg` should use default divisor `6000`:

```text
length_cm * width_cm * height_cm / 6000
```

This is only a preview field. The final shipping calculator may use channel-specific divisors.

---

## Implementation Tasks

### Task 1: Backend Product Logistics Response

**Files:**
- Modify: `backend/app/core/schemas.py`
- Modify: `backend/app/core/service.py`
- Test: `backend/tests/test_product_list_logistics.py`

- [ ] **Step 1: Add failing tests for logistics fields**

Create `backend/tests/test_product_list_logistics.py`.

Test cases:
- Product with all package fields returns `logistics_status=complete`.
- Product missing package weight returns `logistics_status=incomplete`.
- Missing fields include Chinese labels suitable for UI display.
- Volume weight is calculated when package dimensions exist.

Suggested test names:

```python
async def test_product_list_returns_logistics_readiness(client, auth_headers):
    ...

async def test_product_list_reports_missing_logistics_fields(client, auth_headers):
    ...
```

Use existing auth helpers and product creation patterns from:
- `backend/tests/test_logistics_attributes.py`
- `backend/tests/test_product_lifecycle.py`

- [ ] **Step 2: Extend `ProductVO`**

In `backend/app/core/schemas.py`, add:

```python
missing_logistics_fields: list[str] = []
package_volume_weight_kg: Optional[float] = None
```

- [ ] **Step 3: Add helpers in `service.py`**

Add helper behavior:

```python
def missing_logistics_fields(product: Product) -> list[str]:
    missing = []
    if not product.package_length_cm or float(product.package_length_cm) <= 0:
        missing.append("包装长")
    if not product.package_width_cm or float(product.package_width_cm) <= 0:
        missing.append("包装宽")
    if not product.package_height_cm or float(product.package_height_cm) <= 0:
        missing.append("包装高")
    if not product.package_weight_kg or float(product.package_weight_kg) <= 0:
        missing.append("包装重量")
    return missing
```

Volume weight:

```python
def package_volume_weight_kg(product: Product) -> float | None:
    values = [
        product.package_length_cm,
        product.package_width_cm,
        product.package_height_cm,
    ]
    if not all(value is not None and float(value) > 0 for value in values):
        return None
    return round(
        float(product.package_length_cm)
        * float(product.package_width_cm)
        * float(product.package_height_cm)
        / 6000,
        3,
    )
```

- [ ] **Step 4: Wire helpers into `product_to_vo`**

`product_to_vo` must populate:

```python
missing_logistics_fields=missing_logistics_fields(product),
package_volume_weight_kg=package_volume_weight_kg(product),
```

- [ ] **Step 5: Run focused tests**

Run:

```bash
cd backend && python3 -m pytest tests/test_product_list_logistics.py -q
```

Expected:

```text
passed
```

---

### Task 2: Backend Product List Filters

**Files:**
- Modify: `backend/app/core/schemas.py`
- Modify: `backend/app/core/service.py`
- Modify: `backend/app/core/router.py`
- Test: `backend/tests/test_product_list_logistics.py`

- [ ] **Step 1: Add failing filter tests**

Add tests:

```python
async def test_product_list_filters_by_cargo_type(client, auth_headers):
    ...

async def test_product_list_filters_by_incomplete_logistics(client, auth_headers):
    ...
```

Expected behavior:
- `cargo_type=battery` only returns battery products.
- `logistics_status=incomplete` only returns products missing one or more package logistics fields.
- `logistics_status=complete` only returns products with package length, width, height, and weight.

- [ ] **Step 2: Extend `ProductQuery`**

In `backend/app/core/schemas.py`:

```python
cargo_type: Optional[Literal["normal", "battery", "liquid", "sensitive"]] = Field(None, description="货品类型")
logistics_status: Optional[Literal["complete", "incomplete"]] = Field(None, description="物流完整状态")
```

- [ ] **Step 3: Extend `/products` route params**

In `backend/app/core/router.py`, add query params:

```python
cargo_type: str = Query(None, description="货品类型"),
logistics_status: str = Query(None, description="物流完整状态: complete/incomplete"),
```

Pass into `ProductQuery`.

- [ ] **Step 4: Extend `ProductService.list_products`**

Add:

```python
if query.cargo_type:
    stmt = stmt.where(Product.cargo_type == query.cargo_type)

if query.logistics_status == "complete":
    stmt = stmt.where(
        Product.package_length_cm.is_not(None),
        Product.package_width_cm.is_not(None),
        Product.package_height_cm.is_not(None),
        Product.package_weight_kg.is_not(None),
        Product.package_length_cm > 0,
        Product.package_width_cm > 0,
        Product.package_height_cm > 0,
        Product.package_weight_kg > 0,
    )
elif query.logistics_status == "incomplete":
    stmt = stmt.where(
        or_(
            Product.package_length_cm.is_(None),
            Product.package_width_cm.is_(None),
            Product.package_height_cm.is_(None),
            Product.package_weight_kg.is_(None),
            Product.package_length_cm <= 0,
            Product.package_width_cm <= 0,
            Product.package_height_cm <= 0,
            Product.package_weight_kg <= 0,
        )
    )
```

`or_` is already imported in `service.py`.

- [ ] **Step 5: Run focused tests**

Run:

```bash
cd backend && python3 -m pytest tests/test_product_list_logistics.py -q
```

Expected:

```text
passed
```

---

### Task 3: Product Excel Import And Export Upgrade

**Files:**
- Modify: `backend/app/core/router.py`
- Modify: `backend/app/core/excel_service.py`
- Test: `backend/tests/test_product_list_logistics.py`

- [ ] **Step 1: Add failing tests**

Add tests:
- Export includes logistics columns.
- Template includes logistics columns.
- Import reads logistics columns by header name.
- Import result reports row-level errors for invalid numeric logistics values.

Required columns:

```text
商品名称
副标题
单位
状态
商品长(cm)
商品宽(cm)
商品高(cm)
商品重量(kg)
包装长(cm)
包装宽(cm)
包装高(cm)
包装重量(kg)
货品类型
```

- [ ] **Step 2: Update export headers**

In `ExcelService.export_products`, replace headers with the full set above plus:

```text
ID
分类
物流状态
创建时间
```

Keep the workbook readable with wider columns for name and numeric fields.

- [ ] **Step 3: Update template headers**

In `ExcelService.export_template`, use the import-required columns.

Add dropdown validation:
- Status: `草稿,上架,下架`
- Cargo type: `normal,battery,liquid,sensitive`

- [ ] **Step 4: Rewrite import parsing by header**

Do not use positional `row[1]`, `row[2]` style.

Build a header map:

```python
headers = [str(cell.value).strip() if cell.value else "" for cell in ws[1]]
header_map = {name: index for index, name in enumerate(headers)}
```

Read values by header name:

```python
def cell(row, name):
    index = header_map.get(name)
    if index is None or index >= len(row):
        return None
    return row[index]
```

Convert numbers through one helper:

```python
def to_positive_float(value, field_name, row_idx):
    if value in (None, ""):
        return None
    try:
        number = float(value)
    except (TypeError, ValueError):
        raise ValueError(f"{field_name}必须是数字")
    if number <= 0:
        raise ValueError(f"{field_name}必须大于0")
    return number
```

- [ ] **Step 5: Include export filters**

In `backend/app/core/router.py`, `export_products` should accept:

```python
brand_id
cargo_type
logistics_status
```

Either update `ExcelService.export_products` to accept those filters directly or pass a `ProductQuery`.

- [ ] **Step 6: Run tests**

Run:

```bash
cd backend && python3 -m pytest tests/test_product_list_logistics.py tests/test_product_lifecycle.py -q
```

Expected:

```text
passed
```

---

### Task 4: Product List Frontend Redesign

**Files:**
- Modify: `frontend/src/views/product/ProductList.vue`

- [ ] **Step 1: Add query fields**

Extend current query:

```ts
const query = reactive({
  name: '',
  status: null as number | null,
  cargo_type: null as string | null,
  logistics_status: null as string | null,
  page: 1,
  page_size: 20,
})
```

- [ ] **Step 2: Add filter options**

Add:

```ts
const cargoTypeOptions = [
  { label: '普通货品', value: 'normal' },
  { label: '带电', value: 'battery' },
  { label: '液体', value: 'liquid' },
  { label: '敏感货', value: 'sensitive' },
]

const logisticsStatusOptions = [
  { label: '可计算运费', value: 'complete' },
  { label: '缺物流数据', value: 'incomplete' },
]
```

- [ ] **Step 3: Add helper render functions**

In `ProductList.vue`:

```ts
function formatDimensions(length?: number, width?: number, height?: number) {
  if (!length || !width || !height) return '-'
  return `${length} x ${width} x ${height} cm`
}

function formatWeight(weight?: number) {
  if (!weight) return '-'
  return `${weight} kg`
}

function cargoTypeLabel(value?: string) {
  const map: Record<string, string> = {
    normal: '普通',
    battery: '带电',
    liquid: '液体',
    sensitive: '敏感',
  }
  return map[value || 'normal'] || value || '-'
}
```

- [ ] **Step 4: Redesign filter form**

The filter form should include:
- Product name input
- Status select
- Cargo type select
- Logistics readiness select
- Search
- Reset
- Quick filter button: `只看缺物流数据`

Quick filter behavior:

```ts
function showIncompleteOnly() {
  query.logistics_status = 'incomplete'
  query.page = 1
  fetchData()
}
```

- [ ] **Step 5: Replace columns**

Use these columns:

```ts
const columns = [
  { type: 'selection' as const },
  { title: 'ID', key: 'id', width: 70, fixed: 'left' as const },
  { title: '商品名称', key: 'name', minWidth: 220, ellipsis: { tooltip: true } },
  { title: '分类', key: 'category_name', width: 120 },
  { title: '状态', key: 'status_name', width: 90, render: ... },
  { title: '货品', key: 'cargo_type', width: 90, render: ... },
  { title: '商品尺寸', key: 'product_dimensions', width: 150, render: ... },
  { title: '商品重量', key: 'product_weight_kg', width: 100, render: ... },
  { title: '包装尺寸', key: 'package_dimensions', width: 150, render: ... },
  { title: '包装重量', key: 'package_weight_kg', width: 100, render: ... },
  { title: '计费体积重', key: 'package_volume_weight_kg', width: 110, render: ... },
  { title: '物流状态', key: 'logistics_status', width: 180, render: ... },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '操作', width: 330, fixed: 'right' as const, render: ... },
]
```

Logistics status render:
- Complete:

```ts
h(NTag, { type: 'success', size: 'small' }, { default: () => '可计算运费' })
```

- Incomplete:

```ts
h('div', [
  h(NTag, { type: 'warning', size: 'small' }, { default: () => '缺物流数据' }),
  h('div', { style: 'margin-top:4px;color:#d03050;font-size:12px;' },
    `缺: ${(row.missing_logistics_fields || []).join('、') || '未知'}`
  )
])
```

- [ ] **Step 6: Make table horizontally scrollable**

Set:

```vue
<n-data-table
  :columns="columns"
  :data="data"
  :loading="loading"
  :pagination="pagination"
  :scroll-x="1700"
  :row-key="(row: any) => row.id"
  ...
/>
```

- [ ] **Step 7: Export with new filters**

In `handleExport`, include:

```ts
params: {
  name: query.name || undefined,
  status: query.status ?? undefined,
  cargo_type: query.cargo_type || undefined,
  logistics_status: query.logistics_status || undefined,
}
```

- [ ] **Step 8: Reset all filters**

Reset should clear:

```ts
query.name = ''
query.status = null
query.cargo_type = null
query.logistics_status = null
query.page = 1
fetchData()
```

- [ ] **Step 9: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
built
```

---

### Task 5: Product Detail Logistics Card

**Files:**
- Modify: `frontend/src/views/product/ProductDetail.vue`

- [ ] **Step 1: Add logistics card**

Add a card titled `物流信息`.

It should show:
- Product dimensions
- Product weight
- Package dimensions
- Package weight
- Volume weight preview
- Cargo type
- Logistics readiness
- Missing fields if incomplete

- [ ] **Step 2: Add helpers**

Use the same helpers as ProductList:

```ts
function formatDimensions(length?: number, width?: number, height?: number) {
  if (!length || !width || !height) return '-'
  return `${length} x ${width} x ${height} cm`
}

function formatWeight(weight?: number) {
  if (!weight) return '-'
  return `${weight} kg`
}
```

- [ ] **Step 3: Add repair action**

If logistics incomplete, show button:

```text
补齐物流数据
```

Click route:

```ts
router.push(`/products/${productId}/edit`)
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

### Task 6: Documentation Update

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`
- Optional modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`

- [ ] **Step 1: Update project status**

Add a section:

```markdown
### 商品列表物流数据工作台

已完成：
- 商品列表展示商品尺寸、商品重量、包装尺寸、包装重量、货品类型。
- 支持按货品类型和物流完整状态筛选。
- 支持导出/导入物流字段。
- 商品详情展示物流信息和缺失字段。
```

- [ ] **Step 2: Update roadmap**

Mark product logistics data readiness as complete under the logistics/shipping phase.

- [ ] **Step 3: Update logistics spec if needed**

Add note:

```markdown
运费计算前置条件：商品或 SKU 必须具备可用于物流报价的包装长、宽、高、重量。商品列表提供物流完整性筛选，作为运营补齐数据入口。
```

---

## Verification Commands

Run all commands before reporting complete:

```bash
cd backend && python3 -m pytest tests/test_product_list_logistics.py -q
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
- Backend tests pass.
- Frontend build passes.
- No whitespace errors.
- Product list visibly includes required logistics fields.
- Products missing package logistics data are easy to find and fix.

---

## Manual Browser Check

After implementation, start the app normally and check:

1. Login as admin.
2. Open `/products`.
3. Confirm table shows:
   - 商品尺寸
   - 商品重量
   - 包装尺寸
   - 包装重量
   - 货品类型
   - 物流状态
4. Click `只看缺物流数据`.
5. Confirm only incomplete products appear.
6. Edit one incomplete product and fill package length, width, height, weight.
7. Return to product list.
8. Confirm the product now shows `可计算运费`.
9. Download export file and confirm logistics fields exist.
10. Download import template and confirm logistics fields exist.

---

## Handoff Prompt For Another Agent

```text
请阅读并严格执行这个规划文档：
/Users/lc/multisell/docs/superpowers/plans/2026-06-15-product-list-rebuild.md

目标是重做商品列表，让它成为运费计算前的数据工作台。必须显示商品长宽高重量、包装长宽高重量、货品类型、物流完整状态；必须支持按货品类型和物流完整状态筛选；必须升级 Excel 导入导出模板；必须在商品详情补充物流信息卡片。

执行要求：
1. 不要重构无关模块。
2. 不要回滚其他人已有改动。
3. 先补测试，再改后端，再改前端。
4. 完成后运行文档里的 Verification Commands。
5. 汇报改动文件、测试结果、前端构建结果、遗留风险。
```

---

## After This Plan: Next Planning Queue

After this product list rebuild is implemented, plan the next modules in this order:

1. **SKU-level logistics override**
   - Some SKUs may have different packaging from the base product.
   - Shipping calculation should prefer SKU package fields, then fallback to product package fields.

2. **Order inventory closure**
   - Create order locks stock.
   - Cancel pending order releases stock.
   - Paid order deducts physical stock.

3. **Shipping quote versioning**
   - Supplier/channel quote rules need effective dates, expiration dates, and version rollback.

4. **Shipping import preview**
   - Add preview and validation before importing supplier quote sheets.

5. **Platform order ingestion**
   - Import platform orders from Excel/API so shipping and profit can run on real orders.

