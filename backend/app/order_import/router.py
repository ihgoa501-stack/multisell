"""订单导入 - 路由"""
from datetime import datetime
from io import BytesIO
from typing import Optional

from fastapi import APIRouter, Depends, File, Form, HTTPException, UploadFile
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.auth import require_permission
from app.common import PageResult, Result
from app.database import get_db
from app.models import User
from app.order_import.models import OrderImportBatch, OrderImportItem
from app.order_import.schemas import OrderImportBatchCreate, OrderImportBatchVO, OrderImportItemVO
from app.order_import.service import OrderImportService
from app.operation_log.service import OperationLogService

router = APIRouter(tags=["订单导入"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


@router.post("/order-imports/csv", summary="上传 CSV 订单导入")
async def import_orders_csv(
    file: UploadFile = File(...),
    adapter_code: Optional[str] = Form(None),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order_import:import")),
):
    if not file.filename or not file.filename.lower().endswith(".csv"):
        raise HTTPException(status_code=400, detail="仅支持 CSV 文件")

    adapter_code = (adapter_code or "csv_order").strip() or "csv_order"
    content = await file.read()
    if not content:
        raise HTTPException(status_code=400, detail="空文件")

    parsed = OrderImportService.parse_csv(content, source_filename=file.filename, adapter_code=adapter_code)
    batch = await OrderImportService.create_batch(db, {
        "adapter_code": adapter_code,
        "platform": parsed["platform"],
        "store_name": parsed["store_name"],
        "source_filename": parsed["source_filename"],
    }, operator=_operator(current_user))
    await OrderImportService.append_items(db, batch.id, parsed["rows"])
    batch = await OrderImportService.process_batch(db, batch.id, operator=_operator(current_user))

    await OperationLogService.log(
        db,
        module="order_import",
        action="import",
        resource_id=str(batch.id),
        content=f"导入订单批次: {file.filename}, adapter={adapter_code}, rows={batch.row_count}, created={batch.created_order_count}",
        operator=_operator(current_user),
    )
    await db.commit()
    return Result.ok(_batch_to_vo(batch))


@router.get("/order-imports", summary="列出订单导入批次")
async def list_import_batches(
    adapter_code: Optional[str] = None,
    page: int = 1,
    page_size: int = 20,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order_import:view")),
):
    batches = await OrderImportService.list_batches(db, adapter_code=adapter_code)
    total = len(batches)
    items = [_batch_to_vo(batch) for batch in batches]
    return PageResult.ok(items, total, page, page_size)


@router.get("/order-imports/{batch_id}", summary="批次详情")
async def get_import_batch(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order_import:view")),
):
    batch = await OrderImportService.get_batch(db, batch_id)
    if not batch:
        return Result.not_found("批次不存在")
    return Result.ok(_batch_to_vo(batch))


@router.get("/order-imports/{batch_id}/items", summary="批次明细")
async def list_import_items(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("order_import:view")),
):
    batch = await OrderImportService.get_batch(db, batch_id)
    if not batch:
        return Result.not_found("批次不存在")
    items = await OrderImportService.list_items(db, batch_id)
    return Result.ok([_item_to_vo(item) for item in items])


def _batch_to_vo(batch: OrderImportBatch) -> dict:
    return {
        "id": batch.id,
        "adapter_code": batch.adapter_code,
        "platform": batch.platform,
        "store_name": batch.store_name,
        "source_filename": batch.source_filename,
        "row_count": batch.row_count,
        "created_order_count": batch.created_order_count,
        "skipped_duplicate_count": batch.skipped_duplicate_count,
        "failed_count": batch.failed_count,
        "imported_by": batch.imported_by,
        "created_at": batch.created_at,
        "updated_at": batch.updated_at,
    }


def _item_to_vo(item: OrderImportItem) -> dict:
    return {
        "id": item.id,
        "batch_id": item.batch_id,
        "row_number": item.row_number,
        "platform_order_no": item.platform_order_no,
        "order_no": item.order_no,
        "sku_code": item.sku_code,
        "quantity": item.quantity,
        "unit_price": item.unit_price,
        "currency": item.currency,
        "recipient_name": item.recipient_name,
        "recipient_phone": item.recipient_phone,
        "country_code": item.country_code,
        "shipping_address": item.shipping_address,
        "shipping_fee": item.shipping_fee,
        "tracking_number": item.tracking_number,
        "paid_at": item.paid_at,
        "status": item.status,
        "failure_reason": item.failure_reason,
        "raw_payload": item.raw_payload,
        "created_at": item.created_at,
    }
