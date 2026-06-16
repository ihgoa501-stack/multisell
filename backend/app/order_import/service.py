"""订单导入 - 服务层"""
from datetime import datetime, date
from decimal import Decimal
from io import StringIO
from typing import Optional
from uuid import uuid4

import csv
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.inventory.service import InventoryService
from app.models import Order, OrderItem, OrderStatusLog, Platform, Sku, Product
from app.order.schemas import OrderCreate, OrderItemCreate
from app.order.service import OrderService, order_to_dict
from app.order_import.schemas import OrderImportItemCreate
from app.order_import import models


IMPORT_ALLOWED_STATUSES = {"imported", "created_order", "skipped_duplicate", "failed"}

CSV_FIELDS = [
    "platform",
    "store_name",
    "platform_order_no",
    "order_no",
    "sku_code",
    "quantity",
    "unit_price",
    "currency",
    "recipient_name",
    "recipient_phone",
    "country_code",
    "shipping_address",
    "shipping_fee",
    "paid_at",
]


class OrderImportService:

    @staticmethod
    def _parse_paid_at(value: Optional[str]) -> Optional[datetime]:
        if not value or not str(value).strip():
            return None
        text = str(value).strip()
        for fmt in ("%Y-%m-%d %H:%M:%S", "%Y-%m-%d", "%m/%d/%Y %H:%M", "%m/%d/%Y"):
            try:
                return datetime.strptime(text, fmt)
            except ValueError:
                continue
        return None

    @staticmethod
    async def _find_sku(db: AsyncSession, sku_code: str) -> Optional[Sku]:
        stmt = select(Sku).where(Sku.code == sku_code, Sku.status == 1).limit(1)
        result = await db.execute(stmt)
        return result.scalar_one_or_none()

    @staticmethod
    async def _find_platform_by_name(db: AsyncSession, name: Optional[str]) -> Optional[Platform]:
        if not name:
            return None
        stmt = select(Platform).where(Platform.name == name, Platform.status == 1).limit(1)
        result = await db.execute(stmt)
        return result.scalar_one_or_none()

    @staticmethod
    async def _find_external_order(db: AsyncSession, adapter_code: str, platform_order_no: Optional[str], exclude_batch_id: Optional[int] = None) -> Optional[Order]:
        if not platform_order_no:
            return None
        stmt = (
            select(Order)
            .join(models.OrderImportItem, models.OrderImportItem.order_id == Order.id)
            .join(models.OrderImportBatch, models.OrderImportBatch.id == models.OrderImportItem.batch_id)
            .where(
                models.OrderImportBatch.adapter_code == adapter_code,
                models.OrderImportItem.platform_order_no == platform_order_no,
            )
        )
        if exclude_batch_id is not None:
            stmt = stmt.where(models.OrderImportItem.batch_id != exclude_batch_id)
        stmt = stmt.limit(1)
        result = await db.execute(stmt)
        return result.scalar_one_or_none()

    @staticmethod
    async def create_order_from_import(db: AsyncSession, batch: models.OrderImportBatch, payload: dict, operator: Optional[str] = None) -> dict:
        sku_code = str(payload.get("sku_code") or "").strip()
        sku = await OrderImportService._find_sku(db, sku_code)
        if not sku:
            raise ValueError(f"SKU不存在: {sku_code}")

        quantity = int(payload.get("quantity") or 1)
        unit_price = payload.get("unit_price")
        if unit_price is None or str(unit_price).strip() == "":
            unit_price = float(sku.price or 0)

        fee_fields = ["shipping_fee", "platform_fee", "payment_fee", "other_fee", "product_cost"]
        fee_values = {field: float(payload.get(field) or 0) for field in fee_fields}

        recipient_name = str(payload.get("recipient_name") or "").strip()
        if not recipient_name:
            raise ValueError("缺少收件人姓名")

        item_payload = OrderItemCreate(
            sku_id=sku.id,
            quantity=quantity,
            unit_price=float(unit_price),
        )
        order_create = OrderCreate(
            recipient_name=recipient_name,
            recipient_phone=(payload.get("recipient_phone") or None),
            shipping_address=(payload.get("shipping_address") or None),
            items=[item_payload],
            **fee_values,
        )
        order_dict = await OrderService.create(db, order_create)
        order_id = order_dict["id"]

        # 将 tracking_number 和 shipping_fee 写入订单
        order = await db.get(Order, order_id)
        tracking_number = payload.get("tracking_number")
        if tracking_number and str(tracking_number).strip():
            order.tracking_number = str(tracking_number).strip()
        await db.flush()
        await db.refresh(order)

        paid_at_dt = OrderImportService._parse_paid_at(payload.get("paid_at"))
        if paid_at_dt:
            order.status = "paid"
            order.paid_at = paid_at_dt
            log = OrderStatusLog(
                order_id=order.id,
                from_status="pending",
                to_status="paid",
                operator=operator or "system",
                remark="CSV导入支付",
            )
            db.add(log)
            await db.flush()
            await db.refresh(order)
        return order_dict

    @staticmethod
    async def process_batch(db: AsyncSession, batch_id: int, operator: Optional[str] = None) -> models.OrderImportBatch:
        batch = await db.get(models.OrderImportBatch, batch_id)
        if not batch:
            raise ValueError("批次不存在")

        stmt = (
            select(models.OrderImportItem)
            .where(models.OrderImportItem.batch_id == batch_id)
            .order_by(models.OrderImportItem.id)
        )
        result = await db.execute(stmt)
        items = list(result.scalars().all())

        batch.row_count = len(items)
        batch.created_order_count = 0
        batch.skipped_duplicate_count = 0
        batch.failed_count = 0

        created_orders: dict[str, dict] = {}

        for item in items:
            payload = item.raw_payload or {}
            try:
                existing = await OrderImportService._find_external_order(db, batch.adapter_code, item.platform_order_no, exclude_batch_id=batch.id)
                if existing:
                    item.order_id = existing.id
                    item.order_no = existing.order_no
                    item.status = "skipped_duplicate"
                    item.failure_reason = None
                    batch.skipped_duplicate_count += 1
                    continue

                if not item.sku_code or not str(item.sku_code).strip():
                    item.status = "failed"
                    item.failure_reason = "缺少SKU编码"
                    batch.failed_count += 1
                    continue

                sku = await OrderImportService._find_sku(db, str(item.sku_code).strip())
                if not sku:
                    item.status = "failed"
                    item.failure_reason = f"SKU不存在: {item.sku_code}"
                    batch.failed_count += 1
                    continue

                external_key = f"{batch.adapter_code}:{item.platform_order_no or ''}"
                if external_key in created_orders:
                    parent = created_orders[external_key]

                    quantity = int(item.quantity or 1)
                    unit_price = Decimal(str(float(item.unit_price or sku.price or 0)))
                    subtotal = unit_price * quantity

                    order_item = OrderItem(
                        order_id=parent["id"],
                        sku_id=sku.id,
                        product_id=sku.product_id,
                        product_name="",
                        sku_code=sku.code,
                        spec_desc=sku.spec_desc or "",
                        unit_price=unit_price,
                        quantity=quantity,
                        subtotal=subtotal,
                    )
                    db.add(order_item)

                    order_obj = await db.get(Order, parent["id"])
                    if order_obj:
                        order_obj.total_amount = Decimal(str(order_obj.total_amount or 0)) + subtotal
                        OrderService._recalculate_profit(order_obj)

                    await InventoryService.lock_stock(
                        db, sku_id=sku.id, quantity=quantity,
                        order_no=parent["order_no"], operator=operator or "system",
                    )

                    await db.flush()

                    item.order_id = parent["id"]
                    item.order_no = parent["order_no"]
                    item.status = "imported"
                else:
                    order_dict = await OrderImportService.create_order_from_import(db, batch, payload, operator)
                    item.order_id = order_dict["id"]
                    item.order_no = order_dict["order_no"]
                    item.status = "created_order"
                    batch.created_order_count += 1
                    created_orders[external_key] = {"id": order_dict["id"], "order_no": order_dict["order_no"]}
            except ValueError as exc:
                item.status = "failed"
                item.failure_reason = str(exc)
                batch.failed_count += 1
            except Exception as exc:
                item.status = "failed"
                item.failure_reason = f"导入异常: {exc}"
                batch.failed_count += 1

        batch.imported_by = operator or batch.imported_by
        await db.flush()
        await db.refresh(batch)
        return batch

    @staticmethod
    async def create_batch(db: AsyncSession, data: dict, operator: Optional[str] = None) -> models.OrderImportBatch:
        adapter_code = str(data.get("adapter_code") or "csv_order").strip()
        batch = models.OrderImportBatch(
            adapter_code=adapter_code,
            platform=data.get("platform"),
            store_name=data.get("store_name"),
            source_filename=data.get("source_filename"),
            row_count=0,
            created_order_count=0,
            skipped_duplicate_count=0,
            failed_count=0,
            imported_by=operator,
        )
        db.add(batch)
        await db.flush()
        await db.refresh(batch)
        return batch

    @staticmethod
    async def append_items(db: AsyncSession, batch_id: int, rows: list[dict]) -> list[models.OrderImportItem]:
        items = []
        for idx, row in enumerate(rows, start=1):
            item = models.OrderImportItem(
                batch_id=batch_id,
                row_number=row.get("row_number") or idx,
                platform=row.get("platform"),
                store_name=row.get("store_name"),
                platform_order_no=str(row.get("platform_order_no") or "").strip() or None,
                order_no=str(row.get("order_no") or "").strip() or None,
                sku_code=str(row.get("sku_code") or "").strip(),
                quantity=int(row.get("quantity") or 1),
                unit_price=float(row.get("unit_price") or 0),
                currency=str(row.get("currency") or "CNY"),
                recipient_name=row.get("recipient_name"),
                recipient_phone=row.get("recipient_phone"),
                country_code=row.get("country_code"),
                shipping_address=row.get("shipping_address"),
                shipping_fee=float(row.get("shipping_fee") or 0),
                tracking_number=row.get("tracking_number"),
                paid_at=row.get("paid_at"),
                status="imported",
                raw_payload=row,
            )
            db.add(item)
            items.append(item)
        await db.flush()
        for item in items:
            await db.refresh(item)
        return items

    @staticmethod
    async def get_batch(db: AsyncSession, batch_id: int) -> Optional[models.OrderImportBatch]:
        return await db.get(models.OrderImportBatch, batch_id)

    @staticmethod
    async def list_batches(db: AsyncSession, adapter_code: Optional[str] = None) -> list[models.OrderImportBatch]:
        stmt = select(models.OrderImportBatch).order_by(models.OrderImportBatch.id.desc())
        if adapter_code:
            stmt = stmt.where(models.OrderImportBatch.adapter_code == adapter_code)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    async def list_items(db: AsyncSession, batch_id: int) -> list[models.OrderImportItem]:
        stmt = select(models.OrderImportItem).where(models.OrderImportItem.batch_id == batch_id).order_by(models.OrderImportItem.id)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    def parse_csv(content: bytes, source_filename: Optional[str] = None, adapter_code: str = "csv_order") -> dict:
        text = content.decode("utf-8-sig")
        reader = csv.DictReader(StringIO(text))
        fieldnames = reader.fieldnames or []
        normalized_fieldnames = [name.strip() for name in fieldnames] if fieldnames else []
        adapter_code = (adapter_code or "csv_order").strip() or "csv_order"
        source_filename = source_filename or "upload.csv"
        platform = None
        store_name = None
        rows: list[dict] = []
        for row_number, row in enumerate(reader, start=1):
            plain = {k.strip(): (v.strip() if isinstance(v, str) else v) for k, v in row.items() if k}
            if platform is None:
                platform = plain.get("platform")
            if store_name is None:
                store_name = plain.get("store_name")
            payload = {
                "row_number": row_number,
                "platform": plain.get("platform"),
                "store_name": plain.get("store_name"),
                "platform_order_no": plain.get("platform_order_no"),
                "order_no": plain.get("order_no"),
                "sku_code": plain.get("sku_code"),
                "quantity": plain.get("quantity"),
                "unit_price": plain.get("unit_price"),
                "currency": plain.get("currency") or "CNY",
                "recipient_name": plain.get("recipient_name"),
                "recipient_phone": plain.get("recipient_phone"),
                "country_code": plain.get("country_code"),
                "shipping_address": plain.get("shipping_address"),
                "shipping_fee": plain.get("shipping_fee"),
                "tracking_number": plain.get("tracking_number"),
                "paid_at": plain.get("paid_at"),
            }
            rows.append(payload)

        return {
            "adapter_code": adapter_code,
            "source_filename": source_filename,
            "platform": platform,
            "store_name": store_name,
            "normalized_fieldnames": normalized_fieldnames,
            "rows": rows,
        }
