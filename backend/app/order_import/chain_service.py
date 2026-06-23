"""订单导入经营链路处理 - 服务层"""

from collections import defaultdict
from datetime import datetime, timezone
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.exceptions.service import ExceptionService
from app.finance.ledger_service import LedgerService
from app.operation_log.service import OperationLogService
from app.order_import.models import OrderImportBatch, OrderImportItem


IMPORT_ORDER_STATUSES = {"created_order", "imported", "skipped_duplicate"}


class OrderImportChainService:
    @staticmethod
    async def process_chain(
        db: AsyncSession,
        batch_id: int,
        operator: Optional[str] = None,
    ) -> dict:
        batch = await db.get(OrderImportBatch, batch_id)
        if not batch:
            raise ValueError("批次不存在")

        stmt = select(OrderImportItem).where(
            OrderImportItem.batch_id == batch_id,
            OrderImportItem.status.in_(IMPORT_ORDER_STATUSES),
            OrderImportItem.order_id.isnot(None),
        )
        result = await db.execute(stmt)
        items = list(result.scalars().all())

        order_ids = list({item.order_id for item in items if item.order_id})
        items_by_order: dict[int, list[OrderImportItem]] = defaultdict(list)
        for item in items:
            if item.order_id:
                items_by_order[item.order_id].append(item)

        ledger_rebuilt_count = 0
        exception_generated_count = 0
        chain_failure_count = 0

        for order_id in order_ids:
            order_items = items_by_order.get(order_id, [])
            try:
                await LedgerService.rebuild(db, order_id, operator=operator)
                ledger_rebuilt_count += 1
                for item in order_items:
                    item.chain_status = "ledger_rebuilt"
            except ValueError:
                chain_failure_count += 1
                for item in order_items:
                    item.chain_status = "chain_failed"
                    item.chain_failure_reason = "账本重建失败"

        exc_success = True
        try:
            exc_result = await ExceptionService.generate(db, operator=operator)
            exception_generated_count = exc_result.get("created_count", 0)
        except Exception:
            chain_failure_count += 1
            exc_success = False

        if exc_success:
            for item in items:
                if item.chain_status == "ledger_rebuilt":
                    item.chain_status = "exception_generated"

        batch.chain_status = (
            "chain_processed" if chain_failure_count == 0 else "chain_failed"
        )
        batch.ledger_rebuilt_count = ledger_rebuilt_count
        batch.exception_generated_count = exception_generated_count
        batch.chain_failure_count = chain_failure_count
        batch.processed_at = datetime.now(timezone.utc)

        await db.flush()

        await OperationLogService.log(
            db,
            module="order_import",
            action="process_chain",
            resource_id=str(batch_id),
            content=f"链路处理批次: {batch_id}, orders={len(order_ids)}, ledger={ledger_rebuilt_count}, exceptions={exception_generated_count}, failures={chain_failure_count}",
            operator=operator or "system",
        )

        return {
            "batch_id": batch_id,
            "processed_order_count": len(order_ids),
            "ledger_rebuilt_count": ledger_rebuilt_count,
            "exception_generated_count": exception_generated_count,
            "chain_failure_count": chain_failure_count,
            "chain_status": batch.chain_status,
        }

    @staticmethod
    async def get_chain_summary(
        db: AsyncSession,
        batch_id: int,
    ) -> dict:
        batch = await db.get(OrderImportBatch, batch_id)
        if not batch:
            raise ValueError("批次不存在")

        stmt = select(OrderImportItem).where(
            OrderImportItem.batch_id == batch_id,
            OrderImportItem.order_id.isnot(None),
        )
        result = await db.execute(stmt)
        items = list(result.scalars().all())
        order_ids = list({item.order_id for item in items if item.order_id})

        return {
            "batch_id": batch_id,
            "chain_status": batch.chain_status,
            "ledger_rebuilt_count": batch.ledger_rebuilt_count,
            "exception_generated_count": batch.exception_generated_count,
            "chain_failure_count": batch.chain_failure_count,
            "processed_at": batch.processed_at,
            "total_orders": len(order_ids),
        }
