"""订单导入经营链路处理 - 服务层"""

from collections import defaultdict
from datetime import datetime, timezone
from decimal import Decimal
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.exceptions.service import ExceptionService
from app.finance.ledger_service import LedgerService
from app.models import Order, OrderShippingSnapshot, ShippingProvider, ShippingChannel
from app.operation_log.service import OperationLogService
from app.order_import.models import OrderImportBatch, OrderImportItem


IMPORT_ORDER_STATUSES = {"created_order", "imported", "skipped_duplicate"}


class OrderImportChainService:

    @staticmethod
    async def _ensure_csv_provider(db: AsyncSession) -> tuple[ShippingProvider, ShippingChannel]:
        stmt = select(ShippingProvider).where(ShippingProvider.code == "CSV_IMPORT").limit(1)
        provider = (await db.execute(stmt)).scalar_one_or_none()
        if not provider:
            provider = ShippingProvider(name="CSV Import", code="CSV_IMPORT", status=1)
            db.add(provider)
            await db.flush()
        stmt = select(ShippingChannel).where(
            ShippingChannel.code == "CSV", ShippingChannel.provider_id == provider.id,
        ).limit(1)
        channel = (await db.execute(stmt)).scalar_one_or_none()
        if not channel:
            channel = ShippingChannel(
                name="CSV", code="CSV", provider_id=provider.id,
                volumetric_divisor=6000, cargo_types=["normal"],
                estimated_delivery_min=1, estimated_delivery_max=99,
                currency="CNY", status=1,
            )
            db.add(channel)
            await db.flush()
        return provider, channel

    @staticmethod
    async def _create_order_shipping_snapshot(db: AsyncSession, order: Order) -> None:
        existing_stmt = select(OrderShippingSnapshot).where(OrderShippingSnapshot.order_id == order.id).limit(1)
        existing = (await db.execute(existing_stmt)).scalar_one_or_none()
        if existing or not order.tracking_number:
            return
        from app.models import OrderItem
        stmt = select(OrderItem).where(OrderItem.order_id == order.id).limit(1)
        item = (await db.execute(stmt)).scalar_one_or_none()
        if not item:
            return
        provider, channel = await OrderImportChainService._ensure_csv_provider(db)
        shipping_fee = Decimal(str(order.shipping_fee or 0))
        snapshot = OrderShippingSnapshot(
            order_id=order.id,
            sku_id=item.sku_id,
            quantity=item.quantity,
            destination_country="CSV",
            cargo_type="normal",
            package_source="csv_import",
            package_length_cm=Decimal("0"),
            package_width_cm=Decimal("0"),
            package_height_cm=Decimal("0"),
            package_weight_kg=Decimal("0"),
            provider_id=provider.id,
            provider_name=provider.name,
            channel_id=channel.id,
            channel_name=channel.name,
            currency="CNY",
            actual_weight_kg=Decimal("0"),
            volumetric_weight_kg=Decimal("0"),
            chargeable_weight_kg=Decimal("0"),
            base_shipping_fee=shipping_fee,
            total_shipping_fee=shipping_fee,
            calculation_detail="CSV导入快照",
        )
        db.add(snapshot)
        await db.flush()

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
        snapshot_count = 0

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

        for order_id in order_ids:
            order = await db.get(Order, order_id)
            if order and order.tracking_number:
                await OrderImportChainService._create_order_shipping_snapshot(db, order)
                snapshot_count += 1

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

        batch.chain_status = "chain_processed" if chain_failure_count == 0 else "chain_failed"
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
