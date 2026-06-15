"""平台结算导入 - 服务层"""

import csv
import io
from datetime import datetime
from typing import Optional

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Order, PlatformSettlementBatch, PlatformSettlementItem
from app.operation_log.service import OperationLogService

ALLOWED_TX_TYPES = {
    "sale", "platform_fee", "payment_fee", "refund",
    "adjustment", "payout", "tax", "other",
}

REQUIRED_COLUMNS = {"platform", "transaction_type", "amount"}


class SettlementService:

    @staticmethod
    def parse_csv(content: bytes) -> tuple[list[dict], list[str]]:
        """解析 CSV，返回 (rows, errors)。"""
        text = content.decode("utf-8-sig")
        reader = csv.DictReader(io.StringIO(text))
        headers = {h.strip() for h in reader.fieldnames or []}

        missing = REQUIRED_COLUMNS - headers
        if missing:
            raise ValueError(f"缺少必填列: {', '.join(sorted(missing))}")

        col_map = {
            "platform": "platform",
            "store_name": "store_name",
            "platform_order_no": "platform_order_no",
            "order_no": "order_no",
            "transaction_type": "transaction_type",
            "currency": "currency",
            "amount": "amount",
            "settled_at": "settled_at",
            "description": "description",
        }

        rows: list[dict] = []
        errors: list[str] = []

        for idx, raw in enumerate(reader, start=2):
            try:
                row = {}
                for csv_col, model_field in col_map.items():
                    raw_val = raw.get(csv_col, "").strip()
                    row[model_field] = raw_val if raw_val else None

                tx_type = (row.get("transaction_type") or "").lower()
                if tx_type not in ALLOWED_TX_TYPES:
                    errors.append(f"第{idx}行: 不支持的交易类型 '{tx_type}'")
                    continue

                if not row.get("platform"):
                    errors.append(f"第{idx}行: 平台不能为空")
                    continue

                try:
                    row["amount"] = float(row.get("amount") or 0)
                except (TypeError, ValueError):
                    errors.append(f"第{idx}行: 金额必须是数字")
                    continue

                if row.get("settled_at"):
                    try:
                        row["settled_at"] = datetime.fromisoformat(
                            row["settled_at"].replace("T", " ").replace("Z", "+00:00")
                        )
                    except ValueError:
                        row["settled_at"] = None

                row["raw_payload"] = dict(raw)
                rows.append(row)
            except Exception as e:
                errors.append(f"第{idx}行: 解析异常 {str(e)}")

        return rows, errors

    @staticmethod
    async def import_csv(
        db: AsyncSession,
        filename: str,
        content: bytes,
        operator: Optional[str] = None,
    ) -> dict:
        """导入结算 CSV，创建批次和行。"""
        rows, parse_errors = SettlementService.parse_csv(content)

        if parse_errors and not rows:
            return {
                "batch_id": None,
                "total_rows": 0,
                "imported_rows": 0,
                "error_rows": len(parse_errors),
                "errors": parse_errors,
            }

        # Detect platform name from first row
        platform_name = rows[0].get("platform") if rows else None

        batch = PlatformSettlementBatch(
            platform_name=platform_name,
            filename=filename,
            row_count=len(rows),
            import_status="imported",
            status="imported",
            created_by=operator,
        )
        db.add(batch)
        await db.flush()

        for row in rows:
            item = PlatformSettlementItem(
                batch_id=batch.id,
                row_number=rows.index(row) + 2,
                platform=row.get("platform"),
                store_name=row.get("store_name"),
                platform_order_no=row.get("platform_order_no"),
                order_no=row.get("order_no"),
                transaction_type=row.get("transaction_type", "").lower(),
                currency=row.get("currency") or "CNY",
                amount=float(row.get("amount", 0)),
                settled_at=row.get("settled_at"),
                description=row.get("description"),
                match_status="unmatched",
                raw_payload=row.get("raw_payload"),
            )

            # Try to match by order_no
            if item.order_no:
                stmt = select(Order).where(Order.order_no == item.order_no)
                result = await db.execute(stmt)
                order = result.scalar_one_or_none()
                if order:
                    item.match_status = "matched"
                    item.matched_order_id = order.id

            db.add(item)

        await db.flush()

        # Update batch counts
        matched_count = await SettlementService._count_by_match_status(db, batch.id, "matched")
        unmatched_count = await SettlementService._count_by_match_status(db, batch.id, "unmatched")
        batch.matched_count = matched_count
        batch.unmatched_count = unmatched_count
        await db.flush()

        await OperationLogService.log(
            db,
            module="settlement",
            action="import",
            resource_id=str(batch.id),
            content=f"导入平台结算: filename={filename}, rows={len(rows)}, matched={matched_count}",
            operator=operator or "system",
        )

        return {
            "batch_id": batch.id,
            "total_rows": len(rows),
            "imported_rows": len(rows),
            "error_rows": len(parse_errors),
            "errors": parse_errors,
        }

    @staticmethod
    async def _count_by_match_status(db, batch_id: int, status: str) -> int:
        stmt = select(func.count(PlatformSettlementItem.id)).where(
            PlatformSettlementItem.batch_id == batch_id,
            PlatformSettlementItem.match_status == status,
        )
        result = await db.execute(stmt)
        return result.scalar() or 0

    @staticmethod
    async def list_batches(db: AsyncSession, status: Optional[str] = None) -> list[dict]:
        stmt = select(PlatformSettlementBatch).order_by(
            PlatformSettlementBatch.created_at.desc()
        )
        if status:
            stmt = stmt.where(PlatformSettlementBatch.status == status)
        result = await db.execute(stmt)
        batches = result.scalars().all()
        return [
            {
                "id": b.id,
                "platform_name": b.platform_name,
                "filename": b.filename,
                "row_count": b.row_count or 0,
                "matched_count": b.matched_count or 0,
                "unmatched_count": b.unmatched_count or 0,
                "import_status": b.import_status,
                "status": b.status,
                "created_by": b.created_by,
                "created_at": b.created_at.isoformat() if b.created_at else None,
            }
            for b in batches
        ]

    @staticmethod
    async def get_batch(db: AsyncSession, batch_id: int) -> Optional[dict]:
        batch = await db.get(PlatformSettlementBatch, batch_id)
        if not batch:
            return None
        return {
            "id": batch.id,
            "platform_name": batch.platform_name,
            "filename": batch.filename,
            "row_count": batch.row_count or 0,
            "matched_count": batch.matched_count or 0,
            "unmatched_count": batch.unmatched_count or 0,
            "import_status": batch.import_status,
            "status": batch.status,
            "created_by": batch.created_by,
            "created_at": batch.created_at.isoformat() if batch.created_at else None,
        }

    @staticmethod
    async def list_items(
        db: AsyncSession,
        batch_id: int,
        match_status: Optional[str] = None,
    ) -> list[dict]:
        stmt = (
            select(PlatformSettlementItem)
            .where(PlatformSettlementItem.batch_id == batch_id)
            .order_by(PlatformSettlementItem.row_number)
        )
        if match_status:
            stmt = stmt.where(PlatformSettlementItem.match_status == match_status)

        result = await db.execute(stmt)
        items = result.scalars().all()
        return [
            {
                "id": it.id,
                "batch_id": it.batch_id,
                "row_number": it.row_number,
                "platform": it.platform,
                "store_name": it.store_name,
                "platform_order_no": it.platform_order_no,
                "order_no": it.order_no,
                "transaction_type": it.transaction_type,
                "currency": it.currency or "CNY",
                "amount": float(it.amount) if it.amount else 0,
                "settled_at": it.settled_at.isoformat() if it.settled_at else None,
                "description": it.description,
                "match_status": it.match_status,
                "matched_order_id": it.matched_order_id,
                "cost_layer": "actual",
                "created_at": it.created_at.isoformat() if it.created_at else None,
            }
            for it in items
        ]

    @staticmethod
    async def list_unmatched_items(
        db: AsyncSession,
    ) -> list[dict]:
        stmt = (
            select(PlatformSettlementItem)
            .where(PlatformSettlementItem.match_status == "unmatched")
            .order_by(PlatformSettlementItem.created_at.desc())
            .limit(200)
        )
        result = await db.execute(stmt)
        items = result.scalars().all()
        return [
            {
                "id": it.id,
                "batch_id": it.batch_id,
                "row_number": it.row_number,
                "platform": it.platform,
                "store_name": it.store_name,
                "order_no": it.order_no,
                "transaction_type": it.transaction_type,
                "currency": it.currency or "CNY",
                "amount": float(it.amount) if it.amount else 0,
                "settled_at": it.settled_at.isoformat() if it.settled_at else None,
                "description": it.description,
                "match_status": it.match_status,
                "cost_layer": "actual",
                "created_at": it.created_at.isoformat() if it.created_at else None,
            }
            for it in items
        ]
