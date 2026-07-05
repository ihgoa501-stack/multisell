> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Excel Batch Pre-Listing Decision Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let operators download a decision template, upload an Excel file, preview row-level validation, run batch pre-listing decisions, and export the result workbook.

**Architecture:** Keep this as an extension of the existing decision module, not the product Excel module. Add a focused `backend/app/decision/excel_service.py` for workbook parsing/rendering, expose three decision Excel endpoints from `backend/app/decision/router.py`, and extend the existing batch decision page with upload, preview, and result export actions.

**Tech Stack:** Python 3.11+, FastAPI, Pydantic v2, openpyxl, SQLAlchemy 2.0 async, pytest, Vue 3, TypeScript, Vite, Naive UI.

---

## Starting Point

Implement this after batch pre-listing decision is merged or present on the working branch.

Expected existing backend:

- `POST /api/decisions/prelisting/batch`
- `PreListingDecisionBatchRequest`
- `PreListingDecisionBatchResponse`
- `PreListingDecisionService.calculate_batch(...)`
- Permission `decision:calculate`

Expected existing frontend:

- `frontend/src/views/decision/BatchPreListingDecision.vue`
- `calculateBatchPreListingDecision(...)`
- Batch input rows and result table already work manually.

Create a new branch before implementation:

```bash
git switch main
git pull
git switch -c codex/excel-batch-prelisting-decision
```

If batch decision is not on `main` yet, branch from the completed batch decision branch:

```bash
git switch codex/batch-prelisting-decision
git switch -c codex/excel-batch-prelisting-decision
```

## Scope

In scope:

- Download Excel template for batch pre-listing decision.
- Upload `.xlsx` file and parse rows into `PreListingDecisionBatchItem` payloads.
- Return row-level validation errors before calculation.
- Fill the batch decision page rows from preview output.
- Export batch decision results to `.xlsx`.
- Reuse existing batch calculation endpoint and service.
- Keep the same `decision:calculate` permission for template, preview, calculate, and export.

Out of scope:

- Persisting uploaded files.
- Persisting calculation history.
- `.csv` upload.
- Product/SKU lookup by external SKU code.
- Excel import that creates or updates products.
- Directly creating listing tasks from approved rows.

## Workbook Contract

Template sheet name:

```text
批量上架决策
```

Accepted input headers:

```text
行标识
SKU ID
目的国
目标售价
平台ID
类目ID
平台费率(%)
支付费率(%)
其他费用
最低利润率(%)
货品类型
```

Required input columns:

```text
SKU ID
目的国
目标售价
```

Default values when cells are blank:

```text
平台费率(%) -> 10
支付费率(%) -> 3
其他费用 -> 0
最低利润率(%) -> 20
货品类型 -> normal
```

Limits:

- Maximum 100 non-empty data rows.
- Empty rows are ignored.
- Uploaded file must end with `.xlsx`.
- Row validation errors are returned in preview response and do not throw HTTP 500.

## API Contract

Download template:

```text
GET /api/decisions/prelisting/batch/template
```

Preview upload:

```text
POST /api/decisions/prelisting/batch/preview
Content-Type: multipart/form-data
file=<xlsx>
```

Preview response:

```json
{
  "total_rows": 3,
  "valid_rows": 2,
  "error_rows": 1,
  "items": [
    {
      "row_number": 2,
      "item": {
        "item_key": "row-2",
        "sku_id": 1,
        "destination_country": "RU",
        "target_sale_price": 5000,
        "platform_id": null,
        "category_id": null,
        "platform_fee_pct": 10,
        "payment_fee_pct": 3,
        "other_fee": 0,
        "minimum_margin_pct": 20,
        "cargo_type": "normal"
      },
      "errors": []
    }
  ]
}
```

Export result:

```text
POST /api/decisions/prelisting/batch/export
Content-Type: application/json
```

Export request body is `PreListingDecisionBatchResponse`.

## File Structure

Create:

- `backend/app/decision/excel_service.py` - Excel template, upload parsing, result workbook export.
- `backend/tests/test_prelisting_decision_excel.py` - focused backend tests for template, preview, export, and permission.

Modify:

- `backend/app/decision/router.py` - add template, preview, and export routes.
- `backend/app/decision/schemas.py` - add preview row and preview response schemas.
- `frontend/src/api/modules/decision.ts` - add Excel preview/export API types and functions.
- `frontend/src/views/decision/BatchPreListingDecision.vue` - add upload/template/export UI.
- `docs/PROJECT_STATUS.md` - document first version completion.
- `docs/ROADMAP.md` - update recommended next task.

Do not create or modify database tables.

## Task 1: Add Preview Schemas And Excel Service Skeleton

**Files:**
- Modify: `backend/app/decision/schemas.py`
- Create: `backend/app/decision/excel_service.py`
- Create: `backend/tests/test_prelisting_decision_excel.py`

- [ ] **Step 1: Write failing schema/service tests**

Create `backend/tests/test_prelisting_decision_excel.py`:

```python
"""批量上架决策 Excel 导入导出测试。"""

import io

import openpyxl
import pytest
from httpx import AsyncClient

from app.decision.excel_service import PreListingDecisionExcelService


def _workbook_bytes(rows: list[list[object]]) -> bytes:
    wb = openpyxl.Workbook()
    ws = wb.active
    ws.title = "批量上架决策"
    for row in rows:
        ws.append(row)
    output = io.BytesIO()
    wb.save(output)
    return output.getvalue()


def _valid_rows() -> list[list[object]]:
    return [
        [
            "行标识",
            "SKU ID",
            "目的国",
            "目标售价",
            "平台ID",
            "类目ID",
            "平台费率(%)",
            "支付费率(%)",
            "其他费用",
            "最低利润率(%)",
            "货品类型",
        ],
        ["row-a", 1, "ru", 5000, None, None, None, None, None, None, None],
        ["row-b", 2, "MY", 1000, 7, 8, 12, 4, 9, 25, "battery"],
    ]


def test_parse_preview_maps_rows_and_defaults():
    content = _workbook_bytes(_valid_rows())

    preview = PreListingDecisionExcelService.parse_preview(content)

    assert preview.total_rows == 2
    assert preview.valid_rows == 2
    assert preview.error_rows == 0
    first = preview.items[0]
    assert first.row_number == 2
    assert first.item is not None
    assert first.item.item_key == "row-a"
    assert first.item.sku_id == 1
    assert first.item.destination_country == "RU"
    assert first.item.platform_fee_pct == 10
    assert first.item.payment_fee_pct == 3
    assert first.item.other_fee == 0
    assert first.item.minimum_margin_pct == 20
    assert first.item.cargo_type == "normal"

    second = preview.items[1].item
    assert second is not None
    assert second.platform_id == 7
    assert second.category_id == 8
    assert second.platform_fee_pct == 12
    assert second.payment_fee_pct == 4
    assert second.other_fee == 9
    assert second.minimum_margin_pct == 25
    assert second.cargo_type == "battery"


def test_parse_preview_returns_row_errors_without_crashing():
    content = _workbook_bytes(
        [
            ["行标识", "SKU ID", "目的国", "目标售价"],
            ["bad-row", "abc", "", -1],
        ]
    )

    preview = PreListingDecisionExcelService.parse_preview(content)

    assert preview.total_rows == 1
    assert preview.valid_rows == 0
    assert preview.error_rows == 1
    assert preview.items[0].row_number == 2
    assert preview.items[0].item is None
    assert any("SKU ID必须是整数" in err for err in preview.items[0].errors)
    assert any("目的国不能为空" in err for err in preview.items[0].errors)
    assert any("目标售价必须大于0" in err for err in preview.items[0].errors)


def test_parse_preview_rejects_more_than_100_rows():
    rows = [["SKU ID", "目的国", "目标售价"]]
    rows.extend([[idx, "RU", 100] for idx in range(1, 102)])
    content = _workbook_bytes(rows)

    with pytest.raises(ValueError, match="最多支持100行"):
        PreListingDecisionExcelService.parse_preview(content)
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision_excel.py::test_parse_preview_maps_rows_and_defaults tests/test_prelisting_decision_excel.py::test_parse_preview_returns_row_errors_without_crashing tests/test_prelisting_decision_excel.py::test_parse_preview_rejects_more_than_100_rows -q
```

Expected:

- FAIL during import because `app.decision.excel_service` does not exist.

- [ ] **Step 3: Add preview schemas**

In `backend/app/decision/schemas.py`, add these classes after `PreListingDecisionBatchResponse`:

```python
class PreListingDecisionExcelPreviewRow(BaseModel):
    """批量上架决策 Excel 预览单行"""
    row_number: int
    item: Optional[PreListingDecisionBatchItem] = None
    errors: list[str] = []


class PreListingDecisionExcelPreviewResponse(BaseModel):
    """批量上架决策 Excel 预览响应"""
    total_rows: int
    valid_rows: int
    error_rows: int
    items: list[PreListingDecisionExcelPreviewRow]
```

- [ ] **Step 4: Create Excel service**

Create `backend/app/decision/excel_service.py`:

```python
"""批量上架决策 Excel 服务。"""

import io
from typing import Any

import openpyxl
from openpyxl.styles import Alignment, Border, Font, PatternFill, Side
from openpyxl.worksheet.datavalidation import DataValidation

from app.decision.schemas import (
    PreListingDecisionBatchItem,
    PreListingDecisionBatchResponse,
    PreListingDecisionExcelPreviewResponse,
    PreListingDecisionExcelPreviewRow,
)


def _text(value: Any) -> str:
    return str(value).strip() if value not in (None, "") else ""


def _normalize_header(value: Any) -> str:
    return _text(value).replace("（", "(").replace("）", ")")


def _int_or_none(value: Any, field_name: str, errors: list[str]) -> int | None:
    if value in (None, ""):
        return None
    try:
        return int(value)
    except (TypeError, ValueError):
        errors.append(f"{field_name}必须是整数")
        return None


def _required_int(value: Any, field_name: str, errors: list[str]) -> int | None:
    parsed = _int_or_none(value, field_name, errors)
    if parsed is None and value in (None, ""):
        errors.append(f"{field_name}不能为空")
    return parsed


def _float_or_default(value: Any, field_name: str, default: float, errors: list[str]) -> float:
    if value in (None, ""):
        return default
    try:
        parsed = float(value)
    except (TypeError, ValueError):
        errors.append(f"{field_name}必须是数字")
        return default
    if parsed < 0:
        errors.append(f"{field_name}不能小于0")
    return parsed


def _positive_float(value: Any, field_name: str, errors: list[str]) -> float | None:
    if value in (None, ""):
        errors.append(f"{field_name}不能为空")
        return None
    try:
        parsed = float(value)
    except (TypeError, ValueError):
        errors.append(f"{field_name}必须是数字")
        return None
    if parsed <= 0:
        errors.append(f"{field_name}必须大于0")
    return parsed


class PreListingDecisionExcelService:
    """批量上架决策 Excel 模板、预览和结果导出。"""

    SHEET_NAME = "批量上架决策"
    RESULT_SHEET_NAME = "测算结果"
    MAX_ROWS = 100
    HEADERS = [
        "行标识",
        "SKU ID",
        "目的国",
        "目标售价",
        "平台ID",
        "类目ID",
        "平台费率(%)",
        "支付费率(%)",
        "其他费用",
        "最低利润率(%)",
        "货品类型",
    ]
    RESULT_HEADERS = [
        "行号",
        "行标识",
        "SKU ID",
        "状态",
        "建议",
        "目的国",
        "目标售价",
        "商品成本",
        "运费",
        "平台费",
        "支付费",
        "固定交易费",
        "广告预留",
        "其他费用",
        "利润金额",
        "利润率(%)",
        "费用来源",
        "阻断原因",
        "警告",
        "错误",
    ]
    HEADER_FILL = PatternFill(start_color="2C3E50", end_color="2C3E50", fill_type="solid")
    HEADER_FONT = Font(bold=True, color="FFFFFF")
    THIN_BORDER = Border(
        left=Side(style="thin"),
        right=Side(style="thin"),
        top=Side(style="thin"),
        bottom=Side(style="thin"),
    )

    @staticmethod
    def _style_header(ws, headers: list[str]) -> None:
        for col, header in enumerate(headers, 1):
            cell = ws.cell(row=1, column=col, value=header)
            cell.fill = PreListingDecisionExcelService.HEADER_FILL
            cell.font = PreListingDecisionExcelService.HEADER_FONT
            cell.alignment = Alignment(horizontal="center")
            cell.border = PreListingDecisionExcelService.THIN_BORDER

    @staticmethod
    def export_template() -> io.BytesIO:
        wb = openpyxl.Workbook()
        ws = wb.active
        ws.title = PreListingDecisionExcelService.SHEET_NAME
        PreListingDecisionExcelService._style_header(ws, PreListingDecisionExcelService.HEADERS)
        widths = [18, 12, 12, 14, 12, 12, 14, 14, 12, 16, 12]
        for index, width in enumerate(widths, 1):
            ws.column_dimensions[openpyxl.utils.get_column_letter(index)].width = width

        cargo_validation = DataValidation(type="list", formula1='"normal,battery,liquid,sensitive"', allow_blank=True)
        cargo_validation.error = "请选择 normal / battery / liquid / sensitive"
        cargo_validation.errorTitle = "货品类型错误"
        ws.add_data_validation(cargo_validation)
        cargo_validation.add("K2:K101")

        output = io.BytesIO()
        wb.save(output)
        output.seek(0)
        return output

    @staticmethod
    def parse_preview(content: bytes) -> PreListingDecisionExcelPreviewResponse:
        wb = openpyxl.load_workbook(io.BytesIO(content), data_only=True)
        ws = wb.active
        if ws.max_row < 2:
            return PreListingDecisionExcelPreviewResponse(total_rows=0, valid_rows=0, error_rows=0, items=[])

        headers = [_normalize_header(cell.value) for cell in ws[1]]
        header_map = {header: idx for idx, header in enumerate(headers) if header}
        rows = []
        for row_number, row in enumerate(ws.iter_rows(min_row=2, values_only=True), start=2):
            if not any(value not in (None, "") for value in row):
                continue
            rows.append((row_number, row))

        if len(rows) > PreListingDecisionExcelService.MAX_ROWS:
            raise ValueError("最多支持100行批量决策")

        preview_rows: list[PreListingDecisionExcelPreviewRow] = []
        for row_number, row in rows:
            preview_rows.append(
                PreListingDecisionExcelService._parse_row(row_number, row, header_map)
            )

        valid_rows = sum(1 for item in preview_rows if item.item is not None)
        error_rows = len(preview_rows) - valid_rows
        return PreListingDecisionExcelPreviewResponse(
            total_rows=len(preview_rows),
            valid_rows=valid_rows,
            error_rows=error_rows,
            items=preview_rows,
        )

    @staticmethod
    def _cell(row: tuple[Any, ...], header_map: dict[str, int], name: str) -> Any:
        index = header_map.get(name)
        if index is None or index >= len(row):
            return None
        return row[index]

    @staticmethod
    def _parse_row(
        row_number: int,
        row: tuple[Any, ...],
        header_map: dict[str, int],
    ) -> PreListingDecisionExcelPreviewRow:
        errors: list[str] = []
        item_key = _text(PreListingDecisionExcelService._cell(row, header_map, "行标识")) or f"row-{row_number}"
        sku_id = _required_int(PreListingDecisionExcelService._cell(row, header_map, "SKU ID"), "SKU ID", errors)
        destination_country = _text(PreListingDecisionExcelService._cell(row, header_map, "目的国")).upper()
        if not destination_country:
            errors.append("目的国不能为空")
        target_sale_price = _positive_float(
            PreListingDecisionExcelService._cell(row, header_map, "目标售价"),
            "目标售价",
            errors,
        )
        platform_id = _int_or_none(PreListingDecisionExcelService._cell(row, header_map, "平台ID"), "平台ID", errors)
        category_id = _int_or_none(PreListingDecisionExcelService._cell(row, header_map, "类目ID"), "类目ID", errors)
        platform_fee_pct = _float_or_default(
            PreListingDecisionExcelService._cell(row, header_map, "平台费率(%)"),
            "平台费率(%)",
            10,
            errors,
        )
        payment_fee_pct = _float_or_default(
            PreListingDecisionExcelService._cell(row, header_map, "支付费率(%)"),
            "支付费率(%)",
            3,
            errors,
        )
        other_fee = _float_or_default(
            PreListingDecisionExcelService._cell(row, header_map, "其他费用"),
            "其他费用",
            0,
            errors,
        )
        minimum_margin_pct = _float_or_default(
            PreListingDecisionExcelService._cell(row, header_map, "最低利润率(%)"),
            "最低利润率(%)",
            20,
            errors,
        )
        cargo_type = _text(PreListingDecisionExcelService._cell(row, header_map, "货品类型")) or "normal"
        if cargo_type not in {"normal", "battery", "liquid", "sensitive"}:
            errors.append("货品类型必须是 normal / battery / liquid / sensitive")

        if errors or sku_id is None or target_sale_price is None:
            return PreListingDecisionExcelPreviewRow(row_number=row_number, item=None, errors=errors)

        item = PreListingDecisionBatchItem(
            item_key=item_key,
            sku_id=sku_id,
            destination_country=destination_country,
            target_sale_price=target_sale_price,
            platform_id=platform_id,
            category_id=category_id,
            platform_fee_pct=platform_fee_pct,
            payment_fee_pct=payment_fee_pct,
            other_fee=other_fee,
            minimum_margin_pct=minimum_margin_pct,
            cargo_type=cargo_type,
        )
        return PreListingDecisionExcelPreviewRow(row_number=row_number, item=item, errors=[])

    @staticmethod
    def export_results(data: PreListingDecisionBatchResponse) -> io.BytesIO:
        wb = openpyxl.Workbook()
        ws = wb.active
        ws.title = PreListingDecisionExcelService.RESULT_SHEET_NAME
        PreListingDecisionExcelService._style_header(ws, PreListingDecisionExcelService.RESULT_HEADERS)

        for row_idx, item in enumerate(data.items, start=2):
            result = item.result
            values = [
                item.index + 1,
                item.item_key or "",
                item.sku_id or "",
                "成功" if item.status == "success" else "错误",
                result.recommendation if result else "",
                result.destination_country if result else "",
                result.target_sale_price if result else "",
                result.product_cost if result else "",
                result.shipping_fee if result else "",
                result.platform_fee if result else "",
                result.payment_fee if result else "",
                result.fixed_fee if result else "",
                result.advertising_fee if result else "",
                result.other_fee if result else "",
                result.profit_amount if result else "",
                result.profit_margin if result else "",
                result.platform_fee_source if result else "",
                "；".join(result.blocking_reasons) if result else "",
                "；".join(result.warnings) if result else "",
                item.error_message or "",
            ]
            for col_idx, value in enumerate(values, 1):
                cell = ws.cell(row=row_idx, column=col_idx, value=value)
                cell.border = PreListingDecisionExcelService.THIN_BORDER

        for col_idx in range(1, len(PreListingDecisionExcelService.RESULT_HEADERS) + 1):
            ws.column_dimensions[openpyxl.utils.get_column_letter(col_idx)].width = 16

        output = io.BytesIO()
        wb.save(output)
        output.seek(0)
        return output
```

- [ ] **Step 5: Run service tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision_excel.py::test_parse_preview_maps_rows_and_defaults tests/test_prelisting_decision_excel.py::test_parse_preview_returns_row_errors_without_crashing tests/test_prelisting_decision_excel.py::test_parse_preview_rejects_more_than_100_rows -q
```

Expected:

- PASS.

- [ ] **Step 6: Commit service and schemas**

Run:

```bash
git add backend/app/decision/schemas.py backend/app/decision/excel_service.py backend/tests/test_prelisting_decision_excel.py
git commit -m "feat: parse batch decision excel preview"
```

## Task 2: Add Backend Excel Endpoints

**Files:**
- Modify: `backend/app/decision/router.py`
- Modify: `backend/tests/test_prelisting_decision_excel.py`

- [ ] **Step 1: Add failing endpoint tests**

Append these tests to `backend/tests/test_prelisting_decision_excel.py`:

```python
@pytest.mark.asyncio
async def test_download_batch_decision_template(async_client: AsyncClient):
    resp = await async_client.get("/api/decisions/prelisting/batch/template")

    assert resp.status_code == 200
    assert resp.headers["content-type"].startswith(
        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
    )
    wb = openpyxl.load_workbook(io.BytesIO(resp.content), data_only=True)
    ws = wb.active
    assert ws.title == "批量上架决策"
    assert [cell.value for cell in ws[1]][:3] == ["行标识", "SKU ID", "目的国"]


@pytest.mark.asyncio
async def test_preview_batch_decision_excel_upload(async_client: AsyncClient):
    content = _workbook_bytes(_valid_rows())

    resp = await async_client.post(
        "/api/decisions/prelisting/batch/preview",
        files={
            "file": (
                "batch_decision.xlsx",
                content,
                "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            )
        },
    )

    assert resp.status_code == 200, resp.text
    data = resp.json()["data"]
    assert data["total_rows"] == 2
    assert data["valid_rows"] == 2
    assert data["error_rows"] == 0
    assert data["items"][0]["item"]["destination_country"] == "RU"


@pytest.mark.asyncio
async def test_preview_rejects_non_xlsx_upload(async_client: AsyncClient):
    resp = await async_client.post(
        "/api/decisions/prelisting/batch/preview",
        files={"file": ("batch_decision.csv", b"sku_id,destination_country", "text/csv")},
    )

    assert resp.status_code == 200
    assert resp.json()["code"] == 400
    assert "请上传 .xlsx 文件" in resp.json()["message"]
```

- [ ] **Step 2: Run endpoint tests to verify failure**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision_excel.py::test_download_batch_decision_template tests/test_prelisting_decision_excel.py::test_preview_batch_decision_excel_upload tests/test_prelisting_decision_excel.py::test_preview_rejects_non_xlsx_upload -q
```

Expected:

- FAIL with 404 because routes do not exist.

- [ ] **Step 3: Add routes**

In `backend/app/decision/router.py`, update imports:

```python
from datetime import datetime

from fastapi import APIRouter, Depends, File, HTTPException, UploadFile
from fastapi.responses import StreamingResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.decision.excel_service import PreListingDecisionExcelService
```

Add these routes after `prelisting_decision_batch(...)`:

```python
@router.get("/prelisting/batch/template", summary="下载批量上架决策模板")
async def download_prelisting_batch_template(
    current_user: User = Depends(require_permission("decision:calculate")),
):
    output = PreListingDecisionExcelService.export_template()
    return StreamingResponse(
        output,
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": "attachment; filename=prelisting_decision_template.xlsx"},
    )


@router.post("/prelisting/batch/preview", summary="预览批量上架决策Excel")
async def preview_prelisting_batch_excel(
    file: UploadFile = File(...),
    current_user: User = Depends(require_permission("decision:calculate")),
):
    if not file.filename or not file.filename.lower().endswith(".xlsx"):
        return Result.bad_request("请上传 .xlsx 文件")
    content = await file.read()
    try:
        preview = PreListingDecisionExcelService.parse_preview(content)
    except ValueError as exc:
        return Result.bad_request(str(exc))
    return Result.ok(preview)
```

- [ ] **Step 4: Run endpoint tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision_excel.py::test_download_batch_decision_template tests/test_prelisting_decision_excel.py::test_preview_batch_decision_excel_upload tests/test_prelisting_decision_excel.py::test_preview_rejects_non_xlsx_upload -q
```

Expected:

- PASS.

- [ ] **Step 5: Add export endpoint test**

Append this test to `backend/tests/test_prelisting_decision_excel.py`:

```python
@pytest.mark.asyncio
async def test_export_batch_decision_results(async_client: AsyncClient):
    payload = {
        "summary": {
            "total_items": 2,
            "success_count": 1,
            "error_count": 1,
            "approve_count": 1,
            "reject_count": 0,
            "needs_data_count": 0,
            "average_profit_margin": 42.5,
        },
        "items": [
            {
                "index": 0,
                "item_key": "row-a",
                "sku_id": 1,
                "status": "success",
                "result": {
                    "sku_id": 1,
                    "destination_country": "RU",
                    "target_sale_price": 5000,
                    "product_cost": 500,
                    "shipping_fee": 60,
                    "platform_fee": 500,
                    "payment_fee": 150,
                    "fixed_fee": 0,
                    "advertising_fee": 0,
                    "other_fee": 100,
                    "profit_amount": 3690,
                    "profit_margin": 73.8,
                    "recommendation": "approve",
                    "blocking_reasons": [],
                    "warnings": [],
                    "applied_platform_fee_rule_id": None,
                    "platform_fee_source": "manual",
                    "platform_fee_rule_summary": None,
                },
                "error_message": None,
            },
            {
                "index": 1,
                "item_key": "row-b",
                "sku_id": 999,
                "status": "error",
                "result": None,
                "error_message": "SKU不存在",
            },
        ],
    }

    resp = await async_client.post(
        "/api/decisions/prelisting/batch/export",
        json=payload,
    )

    assert resp.status_code == 200
    wb = openpyxl.load_workbook(io.BytesIO(resp.content), data_only=True)
    ws = wb.active
    assert ws.title == "测算结果"
    assert ws.cell(row=2, column=4).value == "成功"
    assert ws.cell(row=2, column=5).value == "approve"
    assert ws.cell(row=3, column=4).value == "错误"
    assert ws.cell(row=3, column=20).value == "SKU不存在"
```

- [ ] **Step 6: Add export route**

In `backend/app/decision/router.py`, add `PreListingDecisionBatchResponse` to the import list if it is not already imported, then add:

```python
@router.post("/prelisting/batch/export", summary="导出批量上架决策结果")
async def export_prelisting_batch_results(
    data: PreListingDecisionBatchResponse,
    current_user: User = Depends(require_permission("decision:calculate")),
):
    output = PreListingDecisionExcelService.export_results(data)
    filename = f"prelisting_decision_results_{datetime.utcnow().strftime('%Y%m%d%H%M%S')}.xlsx"
    return StreamingResponse(
        output,
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": f"attachment; filename={filename}"},
    )
```

- [ ] **Step 7: Run Excel endpoint tests**

Run:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_prelisting_decision_excel.py -q
```

Expected:

- All Excel decision tests pass.

- [ ] **Step 8: Commit backend routes**

Run:

```bash
git add backend/app/decision/router.py backend/tests/test_prelisting_decision_excel.py
git commit -m "feat: add batch decision excel endpoints"
```

## Task 3: Add Frontend API Client

**Files:**
- Modify: `frontend/src/api/modules/decision.ts`

- [ ] **Step 1: Add preview types and API functions**

In `frontend/src/api/modules/decision.ts`, add:

```ts
export interface PreListingDecisionExcelPreviewRow {
  row_number: number
  item?: PreListingDecisionBatchItem | null
  errors: string[]
}

export interface PreListingDecisionExcelPreviewResponse {
  total_rows: number
  valid_rows: number
  error_rows: number
  items: PreListingDecisionExcelPreviewRow[]
}

export function downloadBatchPreListingDecisionTemplate() {
  return http.get('/decisions/prelisting/batch/template', { responseType: 'blob' })
}

export function previewBatchPreListingDecisionExcel(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return http.post('/decisions/prelisting/batch/preview', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export function exportBatchPreListingDecisionResults(data: PreListingDecisionBatchResponse) {
  return http.post('/decisions/prelisting/batch/export', data, { responseType: 'blob' })
}
```

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 3: Commit frontend API client**

Run:

```bash
git add frontend/src/api/modules/decision.ts
git commit -m "feat: add batch decision excel client"
```

## Task 4: Add Upload, Preview, And Export To Batch Page

**Files:**
- Modify: `frontend/src/views/decision/BatchPreListingDecision.vue`

- [ ] **Step 1: Extend imports**

In `frontend/src/views/decision/BatchPreListingDecision.vue`, update Naive UI imports:

```ts
import { h, reactive, ref } from 'vue'
import { NInput, NInputNumber, NSelect, NTag, useMessage } from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
```

Update decision API imports:

```ts
import {
  calculateBatchPreListingDecision,
  downloadBatchPreListingDecisionTemplate,
  exportBatchPreListingDecisionResults,
  previewBatchPreListingDecisionExcel,
  type PreListingDecisionBatchItem,
  type PreListingDecisionBatchItemResult,
  type PreListingDecisionBatchResponse,
  type PreListingDecisionExcelPreviewResponse,
} from '@/api/modules/decision'
```

- [ ] **Step 2: Add template/upload/export controls**

In the first `<n-space>` toolbar, replace the current buttons with:

```vue
<n-space>
  <n-button type="primary" @click="addRow">新增行</n-button>
  <n-button @click="removeSelectedRows" :disabled="checkedRowKeys.length === 0">删除选中</n-button>
  <n-button @click="handleDownloadTemplate">下载模板</n-button>
  <n-upload
    :show-file-list="false"
    accept=".xlsx"
    :custom-request="handlePreviewUpload"
  >
    <n-button>上传预览</n-button>
  </n-upload>
  <n-button type="primary" :loading="loading" @click="handleCalculate">批量计算</n-button>
  <n-button v-if="batchResult" @click="handleExportResult">导出结果</n-button>
</n-space>
```

- [ ] **Step 3: Add preview error panel**

Add this block after the input table:

```vue
<n-alert
  v-if="previewErrors.length > 0"
  type="warning"
  :show-icon="false"
>
  <div v-for="err in previewErrors" :key="err">{{ err }}</div>
</n-alert>
```

- [ ] **Step 4: Add page state and helper functions**

In `<script setup>`, add:

```ts
const previewErrors = ref<string[]>([])

function downloadBlob(blob: Blob, filename: string) {
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

function applyPreview(preview: PreListingDecisionExcelPreviewResponse) {
  previewErrors.value = preview.items
    .filter((item) => item.errors.length > 0)
    .flatMap((item) => item.errors.map((err) => `第 ${item.row_number} 行：${err}`))

  const validItems = preview.items
    .filter((item) => item.item)
    .map((item) => item.item as PreListingDecisionBatchItem)

  rows.splice(0, rows.length)
  for (const item of validItems) {
    const key = item.item_key || `row-${Date.now()}-${Math.random().toString(16).slice(2)}`
    rows.push({
      key,
      ...item,
    })
  }
  if (rows.length === 0) {
    rows.push(createRow())
  }
}
```

- [ ] **Step 5: Add download, preview, and export handlers**

In `<script setup>`, add:

```ts
async function handleDownloadTemplate() {
  try {
    const resp = await downloadBatchPreListingDecisionTemplate()
    downloadBlob(resp as unknown as Blob, 'prelisting_decision_template.xlsx')
  } catch (err: any) {
    message.error(err?.message || '下载模板失败')
  }
}

async function handlePreviewUpload(options: { file: UploadFileInfo; onFinish: () => void; onError: () => void }) {
  const rawFile = options.file.file
  if (!rawFile) {
    message.error('未读取到上传文件')
    options.onError()
    return
  }
  try {
    const resp = await previewBatchPreListingDecisionExcel(rawFile)
    const preview = resp.data as unknown as PreListingDecisionExcelPreviewResponse
    applyPreview(preview)
    message.success(`解析成功：有效 ${preview.valid_rows} 行，错误 ${preview.error_rows} 行`)
    options.onFinish()
  } catch (err: any) {
    message.error(err?.message || '上传预览失败')
    options.onError()
  }
}

async function handleExportResult() {
  if (!batchResult.value) return
  try {
    const resp = await exportBatchPreListingDecisionResults(batchResult.value)
    downloadBlob(resp as unknown as Blob, 'prelisting_decision_results.xlsx')
  } catch (err: any) {
    message.error(err?.message || '导出结果失败')
  }
}
```

- [ ] **Step 6: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected:

- Build succeeds.

- [ ] **Step 7: Commit page integration**

Run:

```bash
git add frontend/src/views/decision/BatchPreListingDecision.vue
git commit -m "feat: add excel workflow to batch decision page"
```

## Task 5: Update Docs

**Files:**
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: Update project status**

In `docs/PROJECT_STATUS.md`, add this under the batch pre-listing decision section:

```markdown
### Excel 批量上架决策

状态：已完成第一版。

已实现：
- 下载批量上架决策 Excel 模板。
- 上传 `.xlsx` 后进行行级解析和校验预览。
- 有效行可自动填入批量决策页面。
- 错误行在页面展示行号和错误原因。
- 批量测算结果可导出为 Excel。

暂未实现：
- 保存导入历史。
- 保存测算历史。
- 从 Excel 中按商家 SKU 编码自动查找系统 SKU。
- 从 approve 结果直接创建平台发布任务。
```

- [ ] **Step 2: Update roadmap**

In `docs/ROADMAP.md`, replace the recommended next task block with:

````markdown
最推荐继续做：

```text
从批量决策到上架任务生成。
```

原因：

- Excel 批量决策已经能把运营表格转成可执行的利润判断。
- 下一步应把 approve 结果转成待发布任务或草稿发布记录。
- 这样才能从“判断是否值得上架”推进到“准备真正上架”。
````

- [ ] **Step 3: Commit docs**

Run:

```bash
git add docs/PROJECT_STATUS.md docs/ROADMAP.md
git commit -m "docs: document excel batch decision workflow"
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
  .venv/bin/python -m pytest tests/test_prelisting_decision.py tests/test_prelisting_decision_excel.py -q
```

Expected:

- All decision and decision Excel tests pass.

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
git log --oneline --decorate -8
```

Expected:

- Working tree is clean.
- Recent commits show Excel service, Excel endpoints, frontend workflow, and docs.

## Final Acceptance Criteria

The task is complete only when:

- `GET /api/decisions/prelisting/batch/template` returns a valid `.xlsx` template.
- `POST /api/decisions/prelisting/batch/preview` accepts `.xlsx`.
- Preview returns row-level errors without crashing the whole request.
- Uploads with more than 100 non-empty rows return a controlled bad request.
- `POST /api/decisions/prelisting/batch/export` returns a valid `.xlsx` result workbook.
- All new endpoints require `decision:calculate`.
- Batch page can download template, upload preview, fill valid rows, show preview errors, calculate, and export result.
- Focused backend tests pass.
- Backend full suite passes.
- Frontend production build passes.

## Recommended Agent Prompt

Give this to the implementing agent:

```text
你接手的是 /Users/lc/multisell 的 LingMirror / MultiSell 项目。

先阅读：
- docs/superpowers/plans/2026-06-15-excel-batch-prelisting-decision.md
- backend/app/decision/schemas.py
- backend/app/decision/router.py
- backend/app/decision/service.py
- frontend/src/api/modules/decision.ts
- frontend/src/views/decision/BatchPreListingDecision.vue
- backend/app/core/excel_service.py
- backend/app/shipping/service.py 中 ImportService 的解析写法

请在新分支 codex/excel-batch-prelisting-decision 上按计划逐任务执行。严格 TDD：先写失败测试，再写实现。不要新建数据库表，不要保存上传文件，不要做 CSV，不要从 approve 结果创建发布任务。

完成后必须运行：
- cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
- cd frontend && npm run build

交付时说明：
- 改了哪些文件
- 新增了哪些 API
- Excel 模板和预览字段规则
- 行级错误如何返回
- 结果导出包含哪些列
- 测试命令和结果
- 剩余限制
```
