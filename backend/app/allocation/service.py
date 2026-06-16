"""费用分摊 - 服务层"""

import csv
import io
from decimal import Decimal, ROUND_HALF_UP
from typing import Optional

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    CostAllocationBatch,
    CostAllocationItem,
    FinanceLedgerEntry,
    Order,
    Sku,
)
from app.finance.cost_layers import COST_LAYER_ALLOCATED
from app.finance.ledger_service import ENTRY_ALLOCATED_COST
from app.operation_log.service import OperationLogService

ALLOCATION_METHODS = ["quantity", "weight", "volume", "value"]


class AllocationService:

    @staticmethod
    def parse_csv(content: bytes) -> tuple[list[dict], list[str]]:
        text = content.decode("utf-8-sig")
        reader = csv.DictReader(io.StringIO(text))
        headers = {h.strip() for h in reader.fieldnames or []}

        if "sku_code" not in headers:
            raise ValueError("缺少必填列: sku_code")

        rows: list[dict] = []
        errors: list[str] = []

        for idx, raw in enumerate(reader, start=2):
            try:
                row = {
                    "sku_code": raw.get("sku_code", "").strip(),
                    "order_no": raw.get("order_no", "").strip() or None,
                    "quantity": int(float(raw.get("quantity", 0) or 0)),
                    "weight_kg": float(raw.get("weight", 0) or 0) if raw.get("weight") else None,
                    "volume_m3": float(raw.get("volume", 0) or 0) if raw.get("volume") else None,
                    "item_value": float(raw.get("item_value", 0) or 0) if raw.get("item_value") else None,
                    "raw_payload": dict(raw),
                }
                if not row["sku_code"]:
                    errors.append(f"第{idx}行: SKU编码不能为空")
                    continue
                rows.append(row)
            except Exception as e:
                errors.append(f"第{idx}行: 解析异常 {str(e)}")

        return rows, errors

    @staticmethod
    async def import_csv(
        db: AsyncSession,
        filename: str,
        content: bytes,
        allocation_type: str,
        allocation_method: str,
        total_amount: float,
        currency: str = "CNY",
        operator: Optional[str] = None,
    ) -> dict:
        rows, parse_errors = AllocationService.parse_csv(content)

        if parse_errors and not rows:
            return {"batch_id": None, "total_rows": 0, "imported_rows": 0, "error_rows": len(parse_errors), "errors": parse_errors}

        batch = CostAllocationBatch(
            allocation_type=allocation_type,
            allocation_method=allocation_method,
            total_amount=Decimal(str(total_amount)),
            currency=currency,
            source_filename=filename,
            row_count=len(rows),
            status="imported",
            created_by=operator,
        )
        db.add(batch)
        await db.flush()

        for row in rows:
            # Resolve sku_id by code
            sku = None
            order = None
            if row["sku_code"]:
                stmt = select(Sku).where(Sku.code == row["sku_code"])
                result = await db.execute(stmt)
                sku = result.scalar_one_or_none()

            if row.get("order_no"):
                stmt = select(Order).where(Order.order_no == row["order_no"])
                result = await db.execute(stmt)
                order = result.scalar_one_or_none()

            item = CostAllocationItem(
                batch_id=batch.id,
                row_number=rows.index(row) + 2,
                sku_id=sku.id if sku else None,
                sku_code=row["sku_code"],
                order_id=order.id if order else None,
                quantity=row["quantity"],
                weight_kg=row.get("weight_kg"),
                volume_m3=row.get("volume_m3"),
                item_value=row.get("item_value"),
                cost_layer="allocated",
                raw_payload=row.get("raw_payload"),
            )
            db.add(item)

        await db.flush()

        await OperationLogService.log(
            db, module="allocation", action="import",
            resource_id=str(batch.id),
            content=f"导入费用分摊: type={allocation_type}, method={allocation_method}, total={total_amount}",
            operator=operator or "system",
        )

        return {"batch_id": batch.id, "total_rows": len(rows), "imported_rows": len(rows), "error_rows": len(parse_errors), "errors": parse_errors}

    @staticmethod
    async def calculate(
        db: AsyncSession,
        batch_id: int,
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        batch = await db.get(CostAllocationBatch, batch_id)
        if not batch:
            return None

        stmt = select(CostAllocationItem).where(CostAllocationItem.batch_id == batch_id)
        result = await db.execute(stmt)
        items = result.scalars().all()

        if not items:
            return None

        total_amount = Decimal(str(batch.total_amount))
        method = batch.allocation_method

        # Compute factors
        factors: list[Decimal] = []
        for item in items:
            if method == "quantity":
                factors.append(Decimal(str(item.quantity or 0)))
            elif method == "weight":
                factors.append(Decimal(str(item.weight_kg or 0)))
            elif method == "volume":
                factors.append(Decimal(str(item.volume_m3 or 0)))
            elif method == "value":
                factors.append(Decimal(str(item.item_value or 0)))

        total_factor = sum(factors)

        if total_factor <= 0:
            raise ValueError("分摊因子总和为0，无法计算")

        # Allocate with rounding
        allocated_sum = Decimal("0")
        for i, item in enumerate(items):
            factor = factors[i]
            if i == len(items) - 1:
                # Last item: remainder
                raw_amt = total_amount - allocated_sum
            else:
                ratio = factor / total_factor
                raw_amt = (total_amount * ratio).quantize(Decimal("0.01"), rounding=ROUND_HALF_UP)
            allocated_sum += raw_amt
            item.allocation_factor = factor
            item.allocated_amount = raw_amt

        batch.status = "calculated"
        await db.flush()

        await OperationLogService.log(
            db, module="allocation", action="calculate",
            resource_id=str(batch_id),
            content=f"计算分摊: method={method}, total={total_amount}",
            operator=operator or "system",
        )

        return AllocationService._batch_detail(batch, items)

    @staticmethod
    async def post_to_ledger(
        db: AsyncSession,
        batch_id: int,
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        batch = await db.get(CostAllocationBatch, batch_id)
        if not batch:
            return None

        stmt = select(CostAllocationItem).where(
            CostAllocationItem.batch_id == batch_id,
            CostAllocationItem.posted_to_ledger == 0,
        )
        result = await db.execute(stmt)
        items = result.scalars().all()

        posted_count = 0
        for item in items:
            if not item.order_id:
                continue

            # Create ledger entry
            entry = FinanceLedgerEntry(
                order_id=item.order_id,
                entry_type=ENTRY_ALLOCATED_COST,
                amount=-Decimal(str(item.allocated_amount or 0)),
                currency=batch.currency or "CNY",
                cost_layer=COST_LAYER_ALLOCATED,
                source_type="cost_allocation_item",
                source_id=item.id,
                description=f"费用分摊: {batch.allocation_type} ({batch.allocation_method})",
            )
            db.add(entry)
            item.posted_to_ledger = 1
            posted_count += 1

        batch.posted_count = (batch.posted_count or 0) + posted_count
        if batch.posted_count >= (batch.row_count or 0):
            batch.status = "posted"
        await db.flush()

        await OperationLogService.log(
            db, module="allocation", action="post_to_ledger",
            resource_id=str(batch_id),
            content=f"入账分摊: posted={posted_count}",
            operator=operator or "system",
        )

        return {"batch_id": batch.id, "posted_count": posted_count}

    @staticmethod
    async def list_batches(db: AsyncSession, status: Optional[str] = None) -> list[dict]:
        stmt = select(CostAllocationBatch).order_by(CostAllocationBatch.created_at.desc())
        if status:
            stmt = stmt.where(CostAllocationBatch.status == status)
        result = await db.execute(stmt)
        return [AllocationService._batch_to_dict(b) for b in result.scalars().all()]

    @staticmethod
    async def get_batch(db: AsyncSession, batch_id: int) -> Optional[dict]:
        batch = await db.get(CostAllocationBatch, batch_id)
        if not batch:
            return None
        return AllocationService._batch_to_dict(batch)

    @staticmethod
    async def list_items(db: AsyncSession, batch_id: int) -> list[dict]:
        stmt = select(CostAllocationItem).where(CostAllocationItem.batch_id == batch_id).order_by(CostAllocationItem.row_number)
        result = await db.execute(stmt)
        items = result.scalars().all()
        return [AllocationService._item_to_dict(it) for it in items]

    @staticmethod
    def _batch_detail(batch, items) -> dict:
        return {
            "batch_id": batch.id,
            "allocation_type": batch.allocation_type,
            "allocation_method": batch.allocation_method,
            "total_amount": float(batch.total_amount),
            "currency": batch.currency or "CNY",
            "status": batch.status,
            "items": [AllocationService._item_to_dict(it) for it in items],
        }

    @staticmethod
    def _batch_to_dict(b: CostAllocationBatch) -> dict:
        return {
            "id": b.id,
            "allocation_type": b.allocation_type,
            "allocation_method": b.allocation_method,
            "total_amount": float(b.total_amount),
            "currency": b.currency or "CNY",
            "source_filename": b.source_filename,
            "row_count": b.row_count or 0,
            "status": b.status,
            "posted_count": b.posted_count or 0,
            "created_by": b.created_by,
            "created_at": b.created_at.isoformat() if b.created_at else None,
        }

    @staticmethod
    def _item_to_dict(it: CostAllocationItem) -> dict:
        return {
            "id": it.id,
            "batch_id": it.batch_id,
            "row_number": it.row_number,
            "sku_id": it.sku_id,
            "sku_code": it.sku_code,
            "order_id": it.order_id,
            "quantity": it.quantity or 0,
            "weight_kg": float(it.weight_kg) if it.weight_kg else None,
            "volume_m3": float(it.volume_m3) if it.volume_m3 else None,
            "item_value": float(it.item_value) if it.item_value else None,
            "allocation_factor": float(it.allocation_factor) if it.allocation_factor else None,
            "allocated_amount": float(it.allocated_amount) if it.allocated_amount else 0,
            "cost_layer": it.cost_layer or "allocated",
            "posted_to_ledger": bool(it.posted_to_ledger),
            "created_at": it.created_at.isoformat() if it.created_at else None,
        }
