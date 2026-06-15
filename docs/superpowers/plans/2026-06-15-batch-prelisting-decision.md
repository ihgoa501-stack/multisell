# Batch Pre-Listing Decision Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add batch pre-listing business decisions so operators can evaluate many SKUs in one request and see approve, reject, needs-data, and error counts.

**Architecture:** Reuse the existing single-SKU `PreListingDecisionService.calculate(...)` path for each batch item to avoid a second decision algorithm. Add thin batch request/response schemas, a batch service method that isolates per-item errors, one backend endpoint, and a focused frontend batch page.

**Tech Stack:** Python 3.11+, FastAPI, Pydantic v2, SQLAlchemy 2.0 async, pytest, Vue 3, TypeScript, Vite, Naive UI.

---

## Starting Point

Implement this after the platform fee rules work is merged or present on the working branch.

Expected existing behavior:

- Single item endpoint: `POST /api/decisions/prelisting`.
- Single item service: `PreListingDecisionService.calculate(db, req)`.
- Permission: `decision:calculate`.
- Current single response can include platform fee rule fields:
  - `fixed_fee`
  - `advertising_fee`
  - `applied_platform_fee_rule_id`
  - `platform_fee_source`
  - `platform_fee_rule_summary`

Create a new branch before implementation:

```bash
git switch main
git pull
git switch -c codex/batch-prelisting-decision
```

If platform fee rules are not on `main` yet, branch from the completed platform fee branch instead:

```bash
git switch codex/platform-fee-rules
git switch -c codex/batch-prelisting-decision
```

## Scope

In scope:

- Batch backend API: `POST /api/decisions/prelisting/batch`.
- Per-item success/error isolation.
- Summary counts by recommendation.
- Reuse existing single-SKU decision calculation.
- Frontend batch decision page with editable rows and result table.
- Router menu entry protected by `decision:calculate`.
- Docs updates.

Out of scope:

- Persisting batch decision history.
- CSV/XLSX upload.
- Directly creating listing records from approved results.
- Automatic repricing.
- Parallel database execution.
- Real platform publishing.

## File Structure

Modify:

- `backend/app/decision/schemas.py` - add batch request and response models.
- `backend/app/decision/service.py` - add `calculate_batch(...)`.
- `backend/app/decision/router.py` - add batch endpoint.
- `backend/tests/test_prelisting_decision.py` - add batch tests.
- `frontend/src/api/modules/decision.ts` - add batch types and API function.
- `frontend/src/router/modules/decision.ts` - add batch route.
- `docs/PROJECT_STATUS.md` - document first version completion.
- `docs/ROADMAP.md` - update recommended next task.

Create:

- `frontend/src/views/decision/BatchPreListingDecision.vue` - batch page.

Do not create a database table in this task.

## API Contract

Endpoint:

```text
POST /api/decisions/prelisting/batch
```

Permission:

```text
decision:calculate
```

Request shape:

```json
{
  "items": [
    {
      "item_key": "row-1",
      "sku_id": 123,
      "destination_country": "RU",
      "target_sale_price": 5000,
      "platform_id": 1,
      "category_id": null,
      "platform_fee_pct": 10,
      "payment_fee_pct": 3,
      "other_fee": 100,
      "minimum_margin_pct": 20,
      "cargo_type": "normal"
    }
  ]
}
```

Response shape:

```json
{
  "summary": {
    "total_items": 1,
    "success_count": 1,
    "error_count": 0,
    "approve_count": 1,
    "reject_count": 0,
    "needs_data_count": 0,
    "average_profit_margin": 65.82
  },
  "items": [
    {
      "index": 0,
      "item_key": "row-1",
      "sku_id": 123,
      "status": "success",
      "result": {
        "sku_id": 123,
        "recommendation": "approve"
      },
      "error_message": null
    }
  ]
}
```

Rules:

- A batch can contain 1 to 100 items.
- Each batch item uses the same fields as `PreListingDecisionRequest`, plus optional `item_key`.
- One bad SKU must not fail the whole batch.
- `ValueError("SKU不存在")` from the single-item service becomes an item-level error.
- FastAPI/Pydantic validation errors such as missing `sku_id` still return HTTP 422 for the whole request.
- Summary counts only count successful calculation results for approve/reject/needs_data.
- `average_profit_margin` is the average of successful item margins only. If there are no successful items, return `0`.

## Task 1: Add Batch Schemas

**Files:**
- Modify: `backend/app/decision/schemas.py`
- Test: `backend/tests/test_prelisting_decision.py`

- [ ] **Step 1: Add failing schema/API tests**

Append these tests to `backend/tests/test_prelisting_decision.py`:

```python
@pytest.mark.asyncio
async def test_batch_prelisting_decision_returns_summary_and_item_results(
    async_client: AsyncClient,
):
    sku_id = await _create_test_data(async_client)

    resp = await async_client.post(
        "/api/decisions/prelisting/batch",
        json={
            "items": [
                {
                    "item_key": "approve-row",
                    "sku_id": sku_id,
                    "destination_country": "RU",
                    "target_sale_price": 5000,
                    "platform_fee_pct": 10,
                    "payment_fee_pct": 3,
                    "other_fee": 100,
                    "minimum_margin_pct": 20,
                    "cargo_type": "normal",
                },
                {
                    "item_key": "needs-data-row",
                    "sku_id": sku_id,
                    "destination_country": "ZZ",
                    "target_sale_price": 5000,
                    "platform_fee_pct": 10,
                    "payment_fee_pct": 3,
                    "other_fee": 100,
                    "minimum_margin_pct": 20,
                    "cargo_type": "normal",
                },
                {
                    "item_key": "missing-sku-row",
                    "sku_id": 999999,
                    "destination_country": "RU",
                    "target_sale_price": 5000,
                    "platform_fee_pct": 10,
                    "payment_fee_pct": 3,
                    "other_fee": 100,
                    "minimum_margin_pct": 20,
                    "cargo_type": "normal",
                },
            ]
        },
    )

    assert resp.status_code == 200, resp.text
    data = resp.json()["data"]
    assert data["summary"]["total_items"] == 3
    assert data["summary"]["success_count"] == 2
    assert data["summary"]["error_count"] == 1
    assert data["summary"]["approve_count"] == 1
    assert data["summary"]["needs_data_count"] == 1
    assert data["summary"]["reject_count"] == 0
    assert data["summary"]["average_profit_margin"] > 0

    items = data["items"]
    assert items[0]["index"] == 0
    assert items[0]["item_key"] == "approve-row"
    assert items[0]["status"] == "success"
    assert items[0]["result"]["recommendation"] == "approve"
    assert items[0]["error_message"] is None

    assert items[1]["status"] == "success"
    assert items[1]["result"]["recommendation"] == "needs_data"

    assert items[2]["status"] == "error"
    assert items[2]["result"] is None
    assert "SKU不存在" in items[2]["error_message"]


@pytest.mark.asyncio
async def test_batch_prelisting_decision_rejects_empty_items(async_client: AsyncClient):
    resp = await async_client.post(
        "/api/decisions/prelisting/batch",
        json={"items": []},
    )
    assert resp.status_code == 422
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision.py::test_batch_prelisting_decision_returns_summary_and_item_results tests/test_prelisting_decision.py::test_batch_prelisting_decision_rejects_empty_items -q
```

Expected:

- First test fails with `404 Not Found` because the batch route does not exist.
- Second test fails with `404 Not Found` for the same reason.

- [ ] **Step 3: Add batch schema classes**

In `backend/app/decision/schemas.py`, add these classes after `PreListingDecisionResponse`:

```python
class PreListingDecisionBatchItem(PreListingDecisionRequest):
    """批量上架前经营决策单行请求"""
    item_key: Optional[str] = Field(None, max_length=100, description="前端行标识，原样返回")


class PreListingDecisionBatchRequest(BaseModel):
    """批量上架前经营决策请求"""
    items: list[PreListingDecisionBatchItem] = Field(
        ...,
        min_length=1,
        max_length=100,
        description="批量测算行，最多100条",
    )


class PreListingDecisionBatchItemResult(BaseModel):
    """批量上架前经营决策单行结果"""
    index: int
    item_key: Optional[str] = None
    sku_id: Optional[int] = None
    status: str  # success / error
    result: Optional[PreListingDecisionResponse] = None
    error_message: Optional[str] = None


class PreListingDecisionBatchSummary(BaseModel):
    """批量上架前经营决策汇总"""
    total_items: int
    success_count: int
    error_count: int
    approve_count: int
    reject_count: int
    needs_data_count: int
    average_profit_margin: float


class PreListingDecisionBatchResponse(BaseModel):
    """批量上架前经营决策响应"""
    summary: PreListingDecisionBatchSummary
    items: list[PreListingDecisionBatchItemResult]
```

- [ ] **Step 4: Run schema import check**

Run:

```bash
cd backend
.venv/bin/python - <<'PY'
from app.decision.schemas import PreListingDecisionBatchRequest

payload = {
    "items": [
        {
            "item_key": "row-1",
            "sku_id": 1,
            "destination_country": "RU",
            "target_sale_price": 100,
        }
    ]
}
obj = PreListingDecisionBatchRequest.model_validate(payload)
print(obj.items[0].item_key)
PY
```

Expected:

```text
row-1
```

- [ ] **Step 5: Commit schemas and failing tests**

Run:

```bash
git add backend/app/decision/schemas.py backend/tests/test_prelisting_decision.py
git commit -m "test: define batch prelisting decision contract"
```

## Task 2: Add Batch Service And Endpoint

**Files:**
- Modify: `backend/app/decision/service.py`
- Modify: `backend/app/decision/router.py`
- Test: `backend/tests/test_prelisting_decision.py`

- [ ] **Step 1: Add service method**

In `backend/app/decision/service.py`, extend imports:

```python
from app.decision.schemas import (
    PreListingDecisionBatchItemResult,
    PreListingDecisionBatchRequest,
    PreListingDecisionBatchResponse,
    PreListingDecisionBatchSummary,
    PreListingDecisionRequest,
    PreListingDecisionResponse,
)
```

Add this method inside `PreListingDecisionService` after `calculate(...)`:

```python
    @staticmethod
    async def calculate_batch(
        db: AsyncSession,
        req: PreListingDecisionBatchRequest,
    ) -> PreListingDecisionBatchResponse:
        item_results: list[PreListingDecisionBatchItemResult] = []
        approve_count = 0
        reject_count = 0
        needs_data_count = 0
        error_count = 0
        margin_sum = 0.0
        margin_count = 0

        for index, item in enumerate(req.items):
            try:
                single_req = PreListingDecisionRequest(
                    **item.model_dump(exclude={"item_key"})
                )
                result = await PreListingDecisionService.calculate(db, single_req)
                if result.recommendation == "approve":
                    approve_count += 1
                elif result.recommendation == "reject":
                    reject_count += 1
                elif result.recommendation == "needs_data":
                    needs_data_count += 1

                margin_sum += result.profit_margin
                margin_count += 1
                item_results.append(
                    PreListingDecisionBatchItemResult(
                        index=index,
                        item_key=item.item_key,
                        sku_id=item.sku_id,
                        status="success",
                        result=result,
                        error_message=None,
                    )
                )
            except ValueError as exc:
                error_count += 1
                item_results.append(
                    PreListingDecisionBatchItemResult(
                        index=index,
                        item_key=item.item_key,
                        sku_id=item.sku_id,
                        status="error",
                        result=None,
                        error_message=str(exc),
                    )
                )

        success_count = len(req.items) - error_count
        average_profit_margin = round(margin_sum / margin_count, 2) if margin_count else 0

        return PreListingDecisionBatchResponse(
            summary=PreListingDecisionBatchSummary(
                total_items=len(req.items),
                success_count=success_count,
                error_count=error_count,
                approve_count=approve_count,
                reject_count=reject_count,
                needs_data_count=needs_data_count,
                average_profit_margin=average_profit_margin,
            ),
            items=item_results,
        )
```

- [ ] **Step 2: Add router endpoint**

In `backend/app/decision/router.py`, update schema imports:

```python
from app.decision.schemas import (
    PreListingDecisionBatchRequest,
    PreListingDecisionBatchResponse,
    PreListingDecisionRequest,
    PreListingDecisionResponse,
)
```

Add this route after the existing single endpoint:

```python
@router.post("/prelisting/batch", summary="批量上架前经营决策")
async def batch_prelisting_decision(
    data: PreListingDecisionBatchRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("decision:calculate")),
) -> Result[PreListingDecisionBatchResponse]:
    result = await PreListingDecisionService.calculate_batch(db, data)
    return Result.ok(result)
```

- [ ] **Step 3: Run batch tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision.py::test_batch_prelisting_decision_returns_summary_and_item_results tests/test_prelisting_decision.py::test_batch_prelisting_decision_rejects_empty_items -q
```

Expected:

- Both tests pass.

- [ ] **Step 4: Add permission regression test**

Append this test to `backend/tests/test_prelisting_decision.py`:

```python
@pytest.mark.asyncio
async def test_batch_prelisting_decision_requires_decision_calculate_permission(
    async_client: AsyncClient,
    enable_auth,
):
    from tests.auth_helpers import register_and_login

    _uid, token = await register_and_login(async_client, "batch_decision_no_perm")
    resp = await async_client.post(
        "/api/decisions/prelisting/batch",
        json={
            "items": [
                {
                    "item_key": "row-1",
                    "sku_id": 1,
                    "destination_country": "RU",
                    "target_sale_price": 100,
                }
            ]
        },
        headers={"Authorization": f"Bearer {token}"},
    )

    assert resp.status_code == 403
```

- [ ] **Step 5: Run decision tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision.py -q
```

Expected:

- All tests in `test_prelisting_decision.py` pass.

- [ ] **Step 6: Commit backend batch API**

Run:

```bash
git add backend/app/decision/schemas.py backend/app/decision/service.py backend/app/decision/router.py backend/tests/test_prelisting_decision.py
git commit -m "feat: add batch prelisting decision api"
```

## Task 3: Add Frontend API Types

**Files:**
- Modify: `frontend/src/api/modules/decision.ts`

- [ ] **Step 1: Add TypeScript batch contracts**

In `frontend/src/api/modules/decision.ts`, add these interfaces after `PreListingDecisionResponse`:

```ts
export interface PreListingDecisionBatchItem extends PreListingDecisionRequest {
  item_key?: string | null
}

export interface PreListingDecisionBatchRequest {
  items: PreListingDecisionBatchItem[]
}

export interface PreListingDecisionBatchItemResult {
  index: number
  item_key?: string | null
  sku_id?: number | null
  status: 'success' | 'error'
  result?: PreListingDecisionResponse | null
  error_message?: string | null
}

export interface PreListingDecisionBatchSummary {
  total_items: number
  success_count: number
  error_count: number
  approve_count: number
  reject_count: number
  needs_data_count: number
  average_profit_margin: number
}

export interface PreListingDecisionBatchResponse {
  summary: PreListingDecisionBatchSummary
  items: PreListingDecisionBatchItemResult[]
}
```

Add this API function:

```ts
export function calculateBatchPreListingDecision(data: PreListingDecisionBatchRequest) {
  return http.post('/decisions/prelisting/batch', data)
}
```

- [ ] **Step 2: Run frontend type/build check**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 3: Commit frontend API types**

Run:

```bash
git add frontend/src/api/modules/decision.ts
git commit -m "feat: add batch decision api client"
```

## Task 4: Add Batch Decision Page

**Files:**
- Create: `frontend/src/views/decision/BatchPreListingDecision.vue`
- Modify: `frontend/src/router/modules/decision.ts`

- [ ] **Step 1: Add batch route**

Modify `frontend/src/router/modules/decision.ts` to export both routes:

```ts
import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'decisions/prelisting',
    name: 'PreListingDecision',
    component: () => import('@/views/decision/PreListingDecision.vue'),
    meta: {
      title: '上架决策',
      icon: 'analytics',
      menu: true,
      perm: 'decision:calculate',
    },
  },
  {
    path: 'decisions/prelisting-batch',
    name: 'BatchPreListingDecision',
    component: () => import('@/views/decision/BatchPreListingDecision.vue'),
    meta: {
      title: '批量上架决策',
      icon: 'analytics',
      menu: true,
      perm: 'decision:calculate',
    },
  },
]
```

- [ ] **Step 2: Create the batch page**

Create `frontend/src/views/decision/BatchPreListingDecision.vue`:

```vue
<template>
  <div>
    <h3 style="margin-bottom: 16px;">批量上架前经营决策</h3>

    <n-card title="批量测算">
      <n-space vertical :size="12">
        <n-alert type="info" :show-icon="false">
          最多一次测算 100 个 SKU。每行独立返回结果，单个 SKU 错误不会中断整批。
        </n-alert>

        <n-space>
          <n-button type="primary" @click="addRow">新增行</n-button>
          <n-button @click="removeSelectedRows" :disabled="checkedRowKeys.length === 0">删除选中</n-button>
          <n-button type="primary" :loading="loading" @click="handleCalculate">批量计算</n-button>
        </n-space>

        <n-data-table
          :columns="inputColumns"
          :data="rows"
          :row-key="rowKey"
          v-model:checked-row-keys="checkedRowKeys"
          :pagination="false"
          size="small"
        />
      </n-space>
    </n-card>

    <n-card v-if="batchResult" title="汇总结果" style="margin-top: 16px;">
      <n-descriptions :column="4" bordered>
        <n-descriptions-item label="总行数">{{ batchResult.summary.total_items }}</n-descriptions-item>
        <n-descriptions-item label="成功">{{ batchResult.summary.success_count }}</n-descriptions-item>
        <n-descriptions-item label="错误">{{ batchResult.summary.error_count }}</n-descriptions-item>
        <n-descriptions-item label="平均利润率">{{ batchResult.summary.average_profit_margin }}%</n-descriptions-item>
        <n-descriptions-item label="建议上架">{{ batchResult.summary.approve_count }}</n-descriptions-item>
        <n-descriptions-item label="不建议">{{ batchResult.summary.reject_count }}</n-descriptions-item>
        <n-descriptions-item label="数据不足">{{ batchResult.summary.needs_data_count }}</n-descriptions-item>
      </n-descriptions>
    </n-card>

    <n-card v-if="batchResult" title="明细结果" style="margin-top: 16px;">
      <n-data-table
        :columns="resultColumns"
        :data="batchResult.items"
        :pagination="{ pageSize: 20 }"
        size="small"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, reactive, ref } from 'vue'
import { NInput, NInputNumber, NSelect, NTag, useMessage } from 'naive-ui'
import {
  calculateBatchPreListingDecision,
  type PreListingDecisionBatchItem,
  type PreListingDecisionBatchItemResult,
  type PreListingDecisionBatchResponse,
} from '@/api/modules/decision'

type BatchInputRow = PreListingDecisionBatchItem & {
  key: string
}

const message = useMessage()
const loading = ref(false)
const checkedRowKeys = ref<string[]>([])
const batchResult = ref<PreListingDecisionBatchResponse | null>(null)
const rows = reactive<BatchInputRow[]>([createRow()])

const cargoTypeOptions = [
  { label: '普通', value: 'normal' },
  { label: '带电', value: 'battery' },
  { label: '液体', value: 'liquid' },
  { label: '敏感', value: 'sensitive' },
]

function createRow(): BatchInputRow {
  const key = `row-${Date.now()}-${Math.random().toString(16).slice(2)}`
  return {
    key,
    item_key: key,
    sku_id: null as unknown as number,
    destination_country: 'RU',
    target_sale_price: null as unknown as number,
    platform_id: null,
    category_id: null,
    platform_fee_pct: 10,
    payment_fee_pct: 3,
    other_fee: 0,
    minimum_margin_pct: 20,
    cargo_type: 'normal',
  }
}

function rowKey(row: BatchInputRow) {
  return row.key
}

function addRow() {
  if (rows.length >= 100) {
    message.warning('一次最多测算 100 行')
    return
  }
  rows.push(createRow())
}

function removeSelectedRows() {
  const selected = new Set(checkedRowKeys.value)
  for (let i = rows.length - 1; i >= 0; i -= 1) {
    if (selected.has(rows[i].key)) {
      rows.splice(i, 1)
    }
  }
  checkedRowKeys.value = []
  if (rows.length === 0) {
    rows.push(createRow())
  }
}

function validateRows() {
  for (const [idx, row] of rows.entries()) {
    if (!row.sku_id || !row.destination_country || !row.target_sale_price) {
      message.warning(`第 ${idx + 1} 行缺少 SKU ID、目的国或目标售价`)
      return false
    }
  }
  return true
}

function setNumberField(row: BatchInputRow, field: keyof BatchInputRow, value: number | null) {
  ;(row as unknown as Record<string, number | null>)[field as string] = value
}

async function handleCalculate() {
  if (!validateRows()) return

  loading.value = true
  batchResult.value = null
  try {
    const payload = {
      items: rows.map(({ key: _key, ...row }) => row),
    }
    const resp = await calculateBatchPreListingDecision(payload)
    batchResult.value = resp.data as unknown as PreListingDecisionBatchResponse
  } catch (err: any) {
    message.error(err?.response?.data?.message || err?.message || '批量测算失败')
  } finally {
    loading.value = false
  }
}

function renderNumberInput(row: BatchInputRow, field: keyof BatchInputRow, min = 0, precision = 2) {
  return h(NInputNumber, {
    value: row[field] as number | null,
    min,
    precision,
    style: 'width: 100%;',
    'onUpdate:value': (value: number | null) => {
      setNumberField(row, field, value)
    },
  })
}

const inputColumns = [
  { type: 'selection' as const },
  {
    title: 'SKU ID',
    key: 'sku_id',
    width: 120,
    render: (row: BatchInputRow) => renderNumberInput(row, 'sku_id', 1, 0),
  },
  {
    title: '目的国',
    key: 'destination_country',
    width: 110,
    render: (row: BatchInputRow) =>
      h(NInput, {
        value: row.destination_country,
        maxlength: 10,
        'onUpdate:value': (value: string) => {
          row.destination_country = value
        },
      }),
  },
  {
    title: '目标售价',
    key: 'target_sale_price',
    width: 130,
    render: (row: BatchInputRow) => renderNumberInput(row, 'target_sale_price', 0.01, 2),
  },
  {
    title: '平台ID',
    key: 'platform_id',
    width: 120,
    render: (row: BatchInputRow) => renderNumberInput(row, 'platform_id', 1, 0),
  },
  {
    title: '平台费率%',
    key: 'platform_fee_pct',
    width: 130,
    render: (row: BatchInputRow) => renderNumberInput(row, 'platform_fee_pct', 0, 1),
  },
  {
    title: '支付费率%',
    key: 'payment_fee_pct',
    width: 130,
    render: (row: BatchInputRow) => renderNumberInput(row, 'payment_fee_pct', 0, 1),
  },
  {
    title: '其他费用',
    key: 'other_fee',
    width: 120,
    render: (row: BatchInputRow) => renderNumberInput(row, 'other_fee', 0, 2),
  },
  {
    title: '最低利润率%',
    key: 'minimum_margin_pct',
    width: 140,
    render: (row: BatchInputRow) => renderNumberInput(row, 'minimum_margin_pct', 0, 1),
  },
  {
    title: '货品类型',
    key: 'cargo_type',
    width: 120,
    render: (row: BatchInputRow) =>
      h(NSelect, {
        value: row.cargo_type,
        options: cargoTypeOptions,
        'onUpdate:value': (value: string) => {
          row.cargo_type = value
        },
      }),
  },
]

const resultColumns = [
  { title: '行号', key: 'index', render: (row: PreListingDecisionBatchItemResult) => row.index + 1 },
  { title: 'SKU ID', key: 'sku_id' },
  {
    title: '状态',
    key: 'status',
    render: (row: PreListingDecisionBatchItemResult) =>
      h(
        NTag,
        { type: row.status === 'success' ? 'success' : 'error', size: 'small' },
        { default: () => (row.status === 'success' ? '成功' : '错误') },
      ),
  },
  {
    title: '建议',
    key: 'recommendation',
    render: (row: PreListingDecisionBatchItemResult) => row.result?.recommendation || '-',
  },
  {
    title: '利润率',
    key: 'profit_margin',
    render: (row: PreListingDecisionBatchItemResult) =>
      row.result ? `${row.result.profit_margin}%` : '-',
  },
  {
    title: '利润',
    key: 'profit_amount',
    render: (row: PreListingDecisionBatchItemResult) =>
      row.result ? `${row.result.profit_amount}` : '-',
  },
  {
    title: '费用来源',
    key: 'platform_fee_source',
    render: (row: PreListingDecisionBatchItemResult) =>
      row.result?.platform_fee_source === 'rule' ? '规则库' : row.result ? '手动输入' : '-',
  },
  {
    title: '原因/错误',
    key: 'message',
    render: (row: PreListingDecisionBatchItemResult) => {
      if (row.error_message) return row.error_message
      const reasons = row.result?.blocking_reasons || []
      const warnings = row.result?.warnings || []
      return [...reasons, ...warnings].join('；') || '-'
    },
  },
]
</script>
```

- [ ] **Step 3: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 4: Commit frontend page**

Run:

```bash
git add frontend/src/router/modules/decision.ts frontend/src/views/decision/BatchPreListingDecision.vue
git commit -m "feat: add batch decision page"
```

## Task 5: Update Docs

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: Update project status**

In `docs/PROJECT_STATUS.md`, add this under the existing “上架前经营决策” section or directly after it:

```markdown
### 批量上架前经营决策

状态：已完成第一版。

已实现：
- 批量提交多个 SKU 的上架前利润测算。
- 每一行复用单 SKU 决策逻辑，包含运费、平台费用规则、利润率和推荐结论。
- 单行 SKU 不存在等错误不会中断整批，结果中按行返回错误信息。
- 返回 approve / reject / needs_data / error 汇总数量。
- 前端提供批量录入、批量计算、汇总和明细结果表。

暂未实现：
- 批量导入 Excel。
- 保存批量测算历史。
- 从 approve 结果直接生成平台发布任务。
```

- [ ] **Step 2: Update roadmap**

In `docs/ROADMAP.md`, replace the recommended next task block with:

````markdown
最推荐继续做：

```text
Excel 批量导入批量决策。
```

原因：

- 批量上架前经营决策已经具备后端和页面闭环。
- 运营真实工作流通常来自表格，而不是逐行手动录入。
- 下一步应支持上传 Excel、行级校验、错误下载和批量决策结果导出。
````

When editing this block inside Markdown, use four backticks around the outer snippet if needed so nested fences render correctly.

- [ ] **Step 3: Commit docs**

Run:

```bash
git add docs/PROJECT_STATUS.md docs/ROADMAP.md
git commit -m "docs: document batch prelisting decision"
```

## Task 6: Final Verification

**Files:**
- Read: repository root
- Modify: none unless verification reveals a real bug

- [ ] **Step 1: Run focused backend tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision.py -q
```

Expected:

- All pre-listing decision tests pass.

- [ ] **Step 2: Run backend full suite**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest -q
```

Expected:

- Full backend suite passes.

- [ ] **Step 3: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 4: Check git state**

Run:

```bash
git status --short --branch
git log --oneline --decorate -6
```

Expected:

- Working tree is clean.
- Recent commits show backend API, frontend page, and docs.

## Final Acceptance Criteria

The task is complete only when:

- `POST /api/decisions/prelisting/batch` exists.
- The endpoint requires `decision:calculate`.
- A batch with mixed approve, needs_data, and missing SKU returns HTTP 200.
- Missing SKU is returned as an item-level error.
- Empty items list returns HTTP 422.
- Summary counts are correct.
- Frontend has a protected menu entry for batch decision.
- Frontend can add rows, remove selected rows, submit batch calculation, and show summary plus details.
- Backend decision tests pass.
- Backend full test suite passes.
- Frontend build passes.

## Recommended Agent Prompt

Give this to the implementing agent:

```text
你接手的是 /Users/lc/multisell 的 LingMirror / MultiSell 项目。

先阅读：
- docs/superpowers/plans/2026-06-15-batch-prelisting-decision.md
- backend/app/decision/schemas.py
- backend/app/decision/service.py
- backend/app/decision/router.py
- backend/tests/test_prelisting_decision.py
- frontend/src/api/modules/decision.ts
- frontend/src/router/modules/decision.ts
- frontend/src/views/decision/PreListingDecision.vue

请在新分支 codex/batch-prelisting-decision 上按计划逐任务执行。严格 TDD：先写失败测试，再写实现。批量计算必须复用现有单 SKU 决策逻辑，不要新建数据库表，不要接 Excel，不要接真实平台发布。

完成后必须运行：
- cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
- cd frontend && npm run build

交付时说明：
- 改了哪些文件
- 新增了哪些 API
- 批量结果和错误隔离如何工作
- 测试命令和结果
- 剩余限制
```
