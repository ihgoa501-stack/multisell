"""运费账单导入与对账 - 服务层"""

import csv
import io
from datetime import datetime, timezone
from decimal import Decimal
from typing import Optional

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    Order,
    OrderShippingSnapshot,
    ShippingBillBatch,
    ShippingBillItem,
)
from app.operation_log.service import OperationLogService

# 对账金额差异阈值（绝对金额）
AMOUNT_MISMATCH_THRESHOLD = Decimal("0.01")

# ============================================================
# CSV 列名映射 — 同时支持中文和英文两种格式
# ============================================================
# 英文格式（spec 要求）:
#   order_no,tracking_number,provider_name,channel_name,currency,actual_shipping_fee,surcharge_fee,billed_at
# 中文格式（已存在）:
#   运单号,订单号,物流商,渠道,目的国,计费重量(kg),实际运费,币种

COLUMN_MAP_EN_TO_MODEL = {
    "order_no": "order_no",
    "tracking_number": "tracking_number",
    "provider_name": "provider_name",
    "channel_name": "channel_name",
    "currency": "currency",
    "actual_shipping_fee": "actual_shipping_fee",
    "surcharge_fee": "surcharge_fee",
    "billed_at": "billed_at",
    "destination_country": "destination_country",
    "billed_weight_kg": "billed_weight_kg",
}

COLUMN_MAP_CN_TO_MODEL = {
    "运单号": "tracking_number",
    "订单号": "order_no",
    "物流商": "provider_name",
    "渠道": "channel_name",
    "目的国": "destination_country",
    "计费重量(kg)": "billed_weight_kg",
    "计费重量（kg）": "billed_weight_kg",
    "实际运费": "actual_shipping_fee",
    "币种": "currency",
    "附加费": "surcharge_fee",
    "账单日期": "billed_at",
}

# 英文格式必填列（tracking_number 和 order_no 至少一个即可）
REQUIRED_EN_COLUMNS_BASE = {"actual_shipping_fee"}
REQUIRED_EN_TRACKING_OR_ORDER = {"tracking_number", "order_no"}

# 中文格式必填列（运单号和订单号至少一个即可）
REQUIRED_CN_COLUMNS_BASE = {"物流商", "实际运费"}
REQUIRED_CN_TRACKING_OR_ORDER = {"运单号", "订单号"}


def _validate_required_columns(
    headers: set[str],
    base_required: set[str],
    tracking_or_order: set[str],
    format_name: str,
) -> None:
    """检查必填列和互斥列。"""
    missing_base = base_required - headers
    if missing_base:
        raise ValueError(
            f"缺少必填列: {', '.join(sorted(missing_base))}。"
        )
    # tracking_number 和 order_no 至少一个
    if not headers & tracking_or_order:
        raise ValueError(
            f"缺少追踪号或订单号列 ({', '.join(sorted(tracking_or_order))})，至少需要其中一个。"
            f"支持的格式：英文={list(COLUMN_MAP_EN_TO_MODEL.keys())} 或 中文={list(COLUMN_MAP_CN_TO_MODEL.keys())}"
        )


def _detect_and_get_column_map(headers: set[str]) -> tuple[dict[str, str], set[str], set[str]]:
    """检测 CSV 使用的是英文还是中文列名，返回 (column_map, base_required, tracking_or_order)。"""
    if "tracking_number" in headers or "actual_shipping_fee" in headers:
        return COLUMN_MAP_EN_TO_MODEL, REQUIRED_EN_COLUMNS_BASE, REQUIRED_EN_TRACKING_OR_ORDER
    return COLUMN_MAP_CN_TO_MODEL, REQUIRED_CN_COLUMNS_BASE, REQUIRED_CN_TRACKING_OR_ORDER


class ShippingBillService:
    """运费账单导入与对账服务"""

    # ── Import ───────────────────────────────────────────────────────────────

    @staticmethod
    def parse_csv(content: bytes) -> tuple[list[dict], list[str]]:
        """解析CSV内容，返回 (rows, errors)。支持中英文两种列名格式。"""
        text = content.decode("utf-8-sig")
        reader = csv.DictReader(io.StringIO(text))
        headers = {h.strip() for h in reader.fieldnames or []}

        col_map, base_required, tracking_or_order = _detect_and_get_column_map(headers)

        # 检查必填列
        _validate_required_columns(headers, base_required, tracking_or_order, "英文" if "tracking_number" in headers or "actual_shipping_fee" in headers else "中文")

        rows: list[dict] = []
        errors: list[str] = []

        for idx, raw in enumerate(reader, start=2):
            try:
                row = {}
                for csv_col, model_field in col_map.items():
                    raw_val = raw.get(csv_col, "").strip()
                    row[model_field] = raw_val if raw_val else None

                # 保留原始数据
                raw_payload = dict(raw)

                # 必填校验
                if not row.get("tracking_number") and not row.get("order_no"):
                    errors.append(f"第{idx}行: 运单号和订单号不能同时为空")
                    continue
                if not row.get("provider_name"):
                    errors.append(f"第{idx}行: 物流商不能为空")
                    continue

                # 数值转换：actual_shipping_fee
                actual_fee_val = row.get("actual_shipping_fee")
                if not actual_fee_val:
                    errors.append(f"第{idx}行: 实际运费不能为空")
                    continue
                try:
                    row["actual_shipping_fee"] = float(actual_fee_val)
                except (TypeError, ValueError):
                    errors.append(f"第{idx}行: 实际运费必须是数字")
                    continue

                # 数值转换：surcharge_fee（可选）
                surcharge_val = row.get("surcharge_fee")
                if surcharge_val:
                    try:
                        row["surcharge_fee"] = float(surcharge_val)
                    except (TypeError, ValueError):
                        row["surcharge_fee"] = 0.0
                else:
                    row["surcharge_fee"] = 0.0

                # 计算 total_actual_fee
                row["total_actual_fee"] = row["actual_shipping_fee"] + row["surcharge_fee"]

                # 数值转换：billed_weight_kg（可选）
                if row.get("billed_weight_kg"):
                    try:
                        row["billed_weight_kg"] = float(row["billed_weight_kg"])
                    except (TypeError, ValueError):
                        row["billed_weight_kg"] = None

                # 解析 billed_at（可选）
                billed_at = row.get("billed_at")
                if billed_at:
                    try:
                        row["billed_at_dt"] = datetime.fromisoformat(billed_at)
                    except (ValueError, TypeError):
                        row["billed_at_dt"] = None
                else:
                    row["billed_at_dt"] = None

                # 默认币种
                if not row.get("currency"):
                    row["currency"] = "CNY"

                row["raw_payload"] = raw_payload
                rows.append(row)
            except Exception as e:
                errors.append(f"第{idx}行: 解析异常 {str(e)}")

        return rows, errors

    @staticmethod
    async def import_bills(
        db: AsyncSession,
        filename: str,
        content: bytes,
        operator: Optional[str] = None,
    ) -> dict:
        """导入运费账单，创建批次和行记录。"""
        rows, parse_errors = ShippingBillService.parse_csv(content)

        if parse_errors and not rows:
            return {
                "batch_id": None,
                "total_rows": 0,
                "imported_rows": 0,
                "error_rows": len(parse_errors),
                "errors": [{"row": None, "message": err} for err in parse_errors],
            }

        # 创建批次
        batch = ShippingBillBatch(
            source_filename=filename,
            row_count=len(rows),
            status="imported",
            created_by=operator,
        )
        # 检测是否所有行使用同一币种
        currencies = {r.get("currency", "CNY") for r in rows if r.get("currency")}
        if len(currencies) == 1:
            batch.currency = currencies.pop()
        elif len(currencies) > 1:
            batch.currency = "MULTI"
        db.add(batch)
        await db.flush()

        # 创建账单行
        for row in rows:
            item = ShippingBillItem(
                batch_id=batch.id,
                row_number=rows.index(row) + 2,
                tracking_number=row.get("tracking_number"),
                order_no=row.get("order_no"),
                provider_name=row.get("provider_name"),
                channel_name=row.get("channel_name"),
                destination_country=row.get("destination_country"),
                billed_weight_kg=row.get("billed_weight_kg"),
                currency=row.get("currency") or "CNY",
                actual_shipping_fee=row.get("actual_shipping_fee"),
                surcharge_fee=row.get("surcharge_fee", 0.0),
                total_actual_fee=row.get("total_actual_fee"),
                billed_at=row.get("billed_at_dt"),
                reconciliation_status="unmatched_bill",
                raw_payload=row.get("raw_payload"),
            )
            db.add(item)

        await db.flush()

        # 审计日志
        await OperationLogService.log(
            db,
            module="shipping_bill",
            action="import",
            resource_id=str(batch.id),
            content=f"导入运费账单: filename={filename}, rows={len(rows)}",
            operator=operator or "system",
        )

        return {
            "batch_id": batch.id,
            "total_rows": len(rows),
            "imported_rows": len(rows),
            "error_rows": len(parse_errors),
            "errors": [{"row": None, "message": err} for err in parse_errors],
        }

    # ── Reconciliation ──────────────────────────────────────────────────────

    @staticmethod
    async def reconcile_batch(
        db: AsyncSession,
        batch_id: int,
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        """对账单个批次的所有行。"""
        batch = await db.get(ShippingBillBatch, batch_id)
        if not batch:
            return None

        stmt = (
            select(ShippingBillItem)
            .where(ShippingBillItem.batch_id == batch_id)
        )
        result = await db.execute(stmt)
        items = result.scalars().all()

        matched_count = 0
        unmatched_count = 0
        mismatch_count = 0

        for item in items:
            # 跳过已手动处理的行
            if item.reconciliation_status in ("manual_resolved",):
                continue

            match_result = await ShippingBillService._match_item(db, item)

            if match_result is None:
                item.reconciliation_status = "unmatched_bill"
                unmatched_count += 1
            else:
                status = match_result["status"]
                item.reconciliation_status = status
                item.matched_order_id = match_result.get("order_id")
                item.matched_snapshot_id = match_result.get("snapshot_id")
                item.snapshot_shipping_fee = match_result.get("snapshot_amount")
                item.variance_amount = match_result.get("amount_diff")

                if status == "matched":
                    matched_count += 1
                else:
                    mismatch_count += 1

        # 更新批次汇总
        batch.matched_count = matched_count
        batch.unmatched_count = unmatched_count
        batch.mismatch_count = mismatch_count
        batch.status = "reconciled"
        await db.flush()

        # 审计日志
        await OperationLogService.log(
            db,
            module="shipping_bill",
            action="reconcile",
            resource_id=str(batch_id),
            content=f"对账完成: matched={matched_count}, mismatch={mismatch_count}, unmatched={unmatched_count}",
            operator=operator or "system",
        )

        return {
            "batch_id": batch.id,
            "total_items": len(items),
            "matched_count": matched_count,
            "mismatch_count": mismatch_count,
            "unmatched_count": unmatched_count,
        }

    @staticmethod
    async def _match_item(
        db: AsyncSession,
        item: ShippingBillItem,
    ) -> Optional[dict]:
        """匹配单条账单行到订单/快照。返回匹配结果或 None。"""
        order = None

        # 策略1: 按 tracking_number 匹配 Order
        if item.tracking_number:
            stmt = select(Order).where(
                Order.tracking_number == item.tracking_number
            )
            result = await db.execute(stmt)
            order = result.scalar_one_or_none()

        # 策略2: 按 order_no 匹配 Order
        if order is None and item.order_no:
            stmt = select(Order).where(Order.order_no == item.order_no)
            result = await db.execute(stmt)
            order = result.scalar_one_or_none()

        if order is None:
            return None  # unmatched_bill

        # 获取运费快照
        snap_stmt = select(OrderShippingSnapshot).where(
            OrderShippingSnapshot.order_id == order.id
        )
        snap_result = await db.execute(snap_stmt)
        snapshot = snap_result.scalar_one_or_none()

        if snapshot is None:
            # 订单存在但无快照 → missing_snapshot
            return {
                "status": "missing_snapshot",
                "order_id": order.id,
                "snapshot_id": None,
                "snapshot_amount": None,
                "amount_diff": None,
            }

        # 检查币种
        bill_currency = (item.currency or "CNY").strip().upper()
        snap_currency = (snapshot.currency or "CNY").strip().upper()
        if bill_currency != snap_currency:
            billed = Decimal(str(item.actual_shipping_fee or 0))
            snap_amt = Decimal(str(snapshot.total_shipping_fee or 0))
            diff = billed - snap_amt
            return {
                "status": "currency_mismatch",
                "order_id": order.id,
                "snapshot_id": snapshot.id,
                "snapshot_amount": float(snap_amt),
                "amount_diff": float(diff),
            }

        # 比较金额
        snap_amt = Decimal(str(snapshot.total_shipping_fee))
        billed = Decimal(str(item.actual_shipping_fee or 0))
        diff = billed - snap_amt

        if abs(diff) <= AMOUNT_MISMATCH_THRESHOLD:
            return {
                "status": "matched",
                "order_id": order.id,
                "snapshot_id": snapshot.id,
                "snapshot_amount": float(snap_amt),
                "amount_diff": float(diff),
            }
        else:
            return {
                "status": "amount_mismatch",
                "order_id": order.id,
                "snapshot_id": snapshot.id,
                "snapshot_amount": float(snap_amt),
                "amount_diff": float(diff),
            }

    # ── Reconciliation Summary ──────────────────────────────────────────────

    @staticmethod
    async def get_reconciliation_summary(
        db: AsyncSession,
    ) -> dict:
        """获取所有批次的对账汇总。"""
        # 统计各状态
        total_batches = await db.scalar(
            select(func.count(ShippingBillBatch.id))
        )
        reconciled_batches = await db.scalar(
            select(func.count(ShippingBillBatch.id)).where(
                ShippingBillBatch.status == "reconciled"
            )
        )

        # 统计所有行
        total_items = await db.scalar(
            select(func.count(ShippingBillItem.id))
        ) or 0

        # 各状态计数
        status_counts = {}
        for status_val in ["matched", "unmatched_bill", "missing_snapshot", "amount_mismatch", "currency_mismatch", "manual_resolved"]:
            cnt = await db.scalar(
                select(func.count(ShippingBillItem.id)).where(
                    ShippingBillItem.reconciliation_status == status_val
                )
            ) or 0
            if cnt > 0:
                status_counts[status_val] = cnt

        return {
            "total_batches": total_batches or 0,
            "reconciled_batches": reconciled_batches or 0,
            "total_items": total_items,
            "status_counts": status_counts,
        }

    # ── List / Get ──────────────────────────────────────────────────────────

    @staticmethod
    async def list_batches(
        db: AsyncSession,
        status: Optional[str] = None,
    ) -> list[dict]:
        """查询账单批次列表。"""
        stmt = select(ShippingBillBatch).order_by(
            ShippingBillBatch.created_at.desc()
        )
        if status:
            stmt = stmt.where(ShippingBillBatch.status == status)

        result = await db.execute(stmt)
        batches = result.scalars().all()
        return [
            {
                "id": b.id,
                "source_filename": b.source_filename,
                "row_count": b.row_count or 0,
                "matched_count": b.matched_count or 0,
                "mismatch_count": b.mismatch_count or 0,
                "unmatched_count": b.unmatched_count or 0,
                "currency": b.currency or "CNY",
                "status": b.status,
                "created_by": b.created_by,
                "created_at": b.created_at.isoformat() if b.created_at else None,
            }
            for b in batches
        ]

    @staticmethod
    async def get_batch(db: AsyncSession, batch_id: int) -> Optional[dict]:
        """获取单个批次详情。"""
        batch = await db.get(ShippingBillBatch, batch_id)
        if not batch:
            return None
        return {
            "id": batch.id,
            "source_filename": batch.source_filename,
            "row_count": batch.row_count or 0,
            "matched_count": batch.matched_count or 0,
            "mismatch_count": batch.mismatch_count or 0,
            "unmatched_count": batch.unmatched_count or 0,
            "currency": batch.currency or "CNY",
            "status": batch.status,
            "created_by": batch.created_by,
            "created_at": batch.created_at.isoformat() if batch.created_at else None,
        }

    @staticmethod
    async def list_items(
        db: AsyncSession,
        batch_id: int,
        status: Optional[str] = None,
    ) -> list[dict]:
        """查询账单行列表。"""
        stmt = (
            select(ShippingBillItem)
            .where(ShippingBillItem.batch_id == batch_id)
            .order_by(ShippingBillItem.row_number)
        )
        if status:
            stmt = stmt.where(ShippingBillItem.reconciliation_status == status)

        result = await db.execute(stmt)
        items = result.scalars().all()
        return [
            {
                "id": it.id,
                "batch_id": it.batch_id,
                "row_number": it.row_number,
                "reconciliation_status": it.reconciliation_status or "unmatched_bill",
                "tracking_number": it.tracking_number,
                "order_no": it.order_no,
                "provider_name": it.provider_name,
                "channel_name": it.channel_name,
                "destination_country": it.destination_country,
                "billed_weight_kg": float(it.billed_weight_kg) if it.billed_weight_kg else None,
                "currency": it.currency or "CNY",
                "actual_shipping_fee": float(it.actual_shipping_fee) if it.actual_shipping_fee else None,
                "surcharge_fee": float(it.surcharge_fee) if it.surcharge_fee else None,
                "total_actual_fee": float(it.total_actual_fee) if it.total_actual_fee else None,
                "billed_at": it.billed_at.isoformat() if it.billed_at else None,
                "matched_order_id": it.matched_order_id,
                "matched_snapshot_id": it.matched_snapshot_id,
                "snapshot_shipping_fee": float(it.snapshot_shipping_fee) if it.snapshot_shipping_fee else None,
                "variance_amount": float(it.variance_amount) if it.variance_amount else None,
                "note": it.note,
                "resolved_by": it.resolved_by,
                "resolved_at": it.resolved_at.isoformat() if it.resolved_at else None,
                "created_at": it.created_at.isoformat() if it.created_at else None,
            }
            for it in items
        ]

    # ── Manual Resolve ──────────────────────────────────────────────────────

    @staticmethod
    async def resolve_item(
        db: AsyncSession,
        item_id: int,
        note: str,
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        """手动解决账单差异。"""
        item = await db.get(ShippingBillItem, item_id)
        if not item:
            return None

        item.reconciliation_status = "manual_resolved"
        item.note = note
        item.resolved_by = operator
        item.resolved_at = datetime.now(timezone.utc)
        await db.flush()

        await OperationLogService.log(
            db,
            module="shipping_bill",
            action="resolve",
            resource_id=str(item_id),
            content=f"手动解决运费差异: note={note}",
            operator=operator or "system",
        )

        return {
            "id": item.id,
            "batch_id": item.batch_id,
            "row_number": item.row_number,
            "reconciliation_status": item.reconciliation_status,
            "note": item.note,
            "resolved_by": item.resolved_by,
            "resolved_at": item.resolved_at.isoformat() if item.resolved_at else None,
            "variance_amount": float(item.variance_amount) if item.variance_amount else None,
        }
