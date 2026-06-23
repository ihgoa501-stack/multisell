"""订单导入 - 服务层

支持从平台导出文件或API数据导入订单。
当前使用模拟数据/CSV解析，后续接入平台API后替换为真实数据源。
"""

import csv
import io
import logging
from datetime import datetime, timezone
from decimal import Decimal
from typing import Optional

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import (
    Order, OrderItem, OrderStatusLog,
    Platform, Sku,
    OrderImport,
)
from app.order_import.schemas import OrderImportRowData

logger = logging.getLogger(__name__)


class OrderImportService:

    @staticmethod
    async def import_orders(
        db: AsyncSession,
        source_type: str,
        orders_data: list[OrderImportRowData],
        platform_id: Optional[int] = None,
        created_by: Optional[str] = None,
    ) -> dict:
        """批量导入订单

        对每条数据:
        1. 按 platform_order_no 去重
        2. 解析 SKU 信息
        3. 创建 Order + OrderItem 记录

        返回 { import_id, total, success, failed, errors, orders }
        """
        import_record = OrderImport(
            platform_id=platform_id,
            source_type=source_type,
            total_rows=len(orders_data),
            status="processing",
            created_by=created_by or "system",
        )
        db.add(import_record)
        await db.flush()

        success_count = 0
        error_count = 0
        errors = []
        created_orders = []

        for idx, row in enumerate(orders_data):
            try:
                # 查重 - 按平台订单号
                dup = await db.execute(
                    select(Order).where(Order.order_no == row.platform_order_no)
                )
                if dup.scalar_one_or_none():
                    error_count += 1
                    errors.append({"row": idx, "order_no": row.platform_order_no, "error": "订单号已存在"})
                    continue

                # 解析商品明细
                items_data = []
                total_amount = Decimal("0")
                for item_data in row.items:
                    sku = None
                    sku_code = item_data.get("sku_code", "")
                    if sku_code:
                        sku_result = await db.execute(
                            select(Sku).where(Sku.code == sku_code)
                        )
                        sku = sku_result.scalar_one_or_none()

                    unit_price = Decimal(str(item_data.get("unit_price", 0)))
                    quantity = int(item_data.get("quantity", 1))
                    subtotal = unit_price * quantity
                    total_amount += subtotal

                    items_data.append({
                        "sku_id": sku.id if sku else None,
                        "product_id": sku.product_id if sku else None,
                        "product_name": item_data.get("product_name", ""),
                        "sku_code": sku_code,
                        "spec_desc": sku.spec_desc if sku else item_data.get("spec_desc", ""),
                        "unit_price": float(unit_price),
                        "quantity": quantity,
                        "subtotal": float(subtotal),
                    })

                # 创建订单
                order_data = {
                    "order_no": row.platform_order_no,
                    "status": "paid",
                    "recipient_name": row.recipient_name or "",
                    "recipient_phone": row.recipient_phone or "",
                    "shipping_address": row.shipping_address or "",
                    "total_amount": float(total_amount),
                    "shipping_fee": row.shipping_fee,
                    "pay_amount": float(total_amount) + row.shipping_fee,
                    "platform_fee": row.platform_fee,
                    "payment_fee": row.payment_fee,
                    "paid_at": datetime.now(timezone.utc),
                    "created_at": datetime.now(timezone.utc),
                }

                order = Order(**order_data)
                db.add(order)
                await db.flush()

                # 创建订单明细
                for item in items_data:
                    order_item = OrderItem(
                        order_id=order.id,
                        **item,
                    )
                    db.add(order_item)

                # 创建状态记录
                status_log = OrderStatusLog(
                    order_id=order.id,
                    from_status=None,
                    to_status="paid",
                    operator=created_by or "system",
                    remark=f"从{source_type}导入",
                )
                db.add(status_log)

                created_orders.append({
                    "id": order.id,
                    "order_no": order.order_no,
                    "items_count": len(items_data),
                    "total_amount": float(total_amount),
                })
                success_count += 1

            except Exception as e:
                error_count += 1
                errors.append({"row": idx, "order_no": row.platform_order_no, "error": str(e)})

        # 更新导入记录
        import_record.success_count = success_count
        import_record.error_count = error_count
        import_record.error_detail = errors
        import_record.status = "completed" if error_count == 0 else "completed"
        await db.flush()
        await db.refresh(import_record)

        return {
            "import_id": import_record.id,
            "source_type": source_type,
            "total": len(orders_data),
            "success": success_count,
            "failed": error_count,
            "errors": errors,
            "orders": created_orders,
        }

    @staticmethod
    async def list_imports(
        db: AsyncSession,
        source_type: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[dict], int]:
        """查询导入记录"""
        stmt = select(OrderImport)
        count_stmt = select(func.count()).select_from(OrderImport)

        if source_type:
            stmt = stmt.where(OrderImport.source_type == source_type)
            count_stmt = count_stmt.where(OrderImport.source_type == source_type)

        total = (await db.execute(count_stmt)).scalar() or 0
        offset = (page - 1) * page_size
        stmt = stmt.order_by(OrderImport.created_at.desc()).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        records = result.scalars().all()

        rows = []
        for r in records:
            platform_name = None
            if r.platform_id:
                p = await db.get(Platform, r.platform_id)
                if p:
                    platform_name = p.name

            rows.append({
                "id": r.id,
                "platform_id": r.platform_id,
                "platform_name": platform_name,
                "source_type": r.source_type,
                "file_name": r.file_name,
                "total_rows": r.total_rows,
                "success_count": r.success_count,
                "error_count": r.error_count,
                "status": r.status,
                "created_by": r.created_by,
                "created_at": r.created_at.isoformat() if r.created_at else None,
            })

        return rows, total

    @staticmethod
    async def parse_csv(content: str, source_type: str) -> list[OrderImportRowData]:
        """解析平台导出的CSV文件为导入数据

        支持格式:
          - ozon: Ozon 标准导出格式
          - shopee: Shopee 标准导出格式
          - wb: Wildberries 标准导出格式
        """
        reader = csv.DictReader(io.StringIO(content))
        orders_map: dict[str, OrderImportRowData] = {}

        for row in reader:
            platform_order_no = (
                row.get("Номер заказа") or  # Ozon (Russian)
                row.get("Order ID") or       # Shopee
                row.get("Номер") or          # WB (Russian)
                row.get("订单号") or          # 中文
                row.get("platform_order_no") or
                row.get("order_no") or
                ""
            )
            if not platform_order_no:
                continue

            if platform_order_no not in orders_map:
                orders_map[platform_order_no] = OrderImportRowData(
                    platform_order_no=platform_order_no,
                    order_date=row.get("Дата") or row.get("Date") or row.get("日期") or row.get("paid_at"),
                    status="paid",
                    recipient_name=row.get("Получатель") or row.get("Recipient") or row.get("收件人") or row.get("recipient_name"),
                    recipient_phone=row.get("Телефон") or row.get("Phone") or row.get("电话") or row.get("recipient_phone"),
                    shipping_address=row.get("Адрес") or row.get("Address") or row.get("地址") or row.get("shipping_address"),
                    country=row.get("Страна") or row.get("Country") or row.get("国家") or row.get("country_code"),
                    total_amount=float(row.get("Сумма") or row.get("Amount") or row.get("金额") or 0),
                    shipping_fee=float(row.get("Доставка") or row.get("Shipping") or row.get("运费") or row.get("shipping_fee") or 0),
                    platform_fee=float(row.get("Комиссия") or row.get("Fee") or row.get("平台费") or 0),
                )

            # 商品明细
            sku_code = row.get("SKU") or row.get("Артикул") or row.get("sku_code") or ""
            product_name = row.get("Товар") or row.get("Product Name") or row.get("商品名称") or ""
            quantity = int(row.get("Количество") or row.get("Quantity") or row.get("数量") or row.get("quantity") or 1)
            unit_price = float(row.get("Цена") or row.get("Price") or row.get("单价") or row.get("unit_price") or 0)

            orders_map[platform_order_no].items.append({
                "sku_code": sku_code,
                "product_name": product_name,
                "quantity": quantity,
                "unit_price": unit_price,
            })

        return list(orders_map.values())

    @staticmethod
    async def generate_mock_orders(
        db: AsyncSession,
        platform_id: int,
        count: int = 5,
    ) -> list[dict]:
        """生成模拟订单数据，用于演示"""
        import random

        sku_stmt = select(Sku).limit(10)
        skus = (await db.execute(sku_stmt)).scalars().all()

        if not skus:
            return []

        rows = []
        for i in range(count):
            sku = random.choice(skus)
            qty = random.randint(1, 3)
            unit_price = float(sku.price or 99)
            rows.append(OrderImportRowData(
                platform_order_no=f"SIM-{datetime.now(timezone.utc).strftime('%Y%m%d')}-{i+1:04d}",
                recipient_name=f"测试用户{i+1}",
                recipient_phone=f"1380000{i+1:04d}",
                shipping_address=f"测试地址第{i+1}号",
                country="RU",
                total_amount=unit_price * qty,
                shipping_fee=random.choice([0, 15, 30]),
                platform_fee=round(unit_price * qty * 0.08, 2),
                items=[{
                    "sku_code": sku.code or f"sku-{sku.id}",
                    "product_name": f"模拟商品-{sku.id}",
                    "quantity": qty,
                    "unit_price": unit_price,
                }],
            ))

        return rows
