"""订单管理 - 服务层"""

from datetime import datetime
from decimal import Decimal
from typing import Optional
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession
from app.finance.cost_layers import (
    COST_LAYER_ESTIMATED,
    COST_LAYER_SNAPSHOT,
    resolve_profit_cost_layer,
)
from app.inventory.service import InventoryService
from app.models import Order, OrderItem, OrderShippingSnapshot, OrderStatusLog, Price, Product, Sku
from app.order.schemas import OrderCreate, OrderProfitInputsUpdate, OrderShippingQuoteBind
from app.shipping.schemas import CalculateRequest
from app.shipping.service import CalculateService


ALLOWED_TRANSITIONS = {
    "pending": {"paid", "cancelled"},
    "paid": {"shipped", "cancelled"},
    "shipped": {"delivered"},
    "delivered": {"completed"},
    "completed": set(),
    "cancelled": set(),
}


def _money(value) -> float:
    return float(value or 0)


def _decimal(value) -> Decimal:
    return Decimal(str(value or 0))


def _pct(numerator: Decimal, denominator: Decimal) -> Decimal:
    if denominator <= 0:
        return Decimal("0")
    return (numerator / denominator) * Decimal("100")


def _item_to_dict(item: OrderItem) -> dict:
    return {
        "id": item.id,
        "order_id": item.order_id,
        "sku_id": item.sku_id,
        "product_id": item.product_id,
        "product_name": item.product_name,
        "sku_code": item.sku_code,
        "spec_desc": item.spec_desc,
        "unit_price": _money(item.unit_price),
        "quantity": item.quantity,
        "subtotal": _money(item.subtotal),
    }


def _log_to_dict(log: OrderStatusLog, current_status: str) -> dict:
    return {
        "id": log.id,
        "order_id": log.order_id,
        "from_status": log.from_status,
        "to_status": log.to_status,
        "operator": log.operator,
        "remark": log.remark,
        "is_current": log.to_status == current_status,
        "created_at": log.created_at,
    }


def _shipping_snapshot_to_dict(snapshot: OrderShippingSnapshot) -> dict:
    return {
        "id": snapshot.id,
        "order_id": snapshot.order_id,
        "sku_id": snapshot.sku_id,
        "quantity": snapshot.quantity,
        "destination_country": snapshot.destination_country,
        "postal_code": snapshot.postal_code,
        "cargo_type": snapshot.cargo_type,
        "package_source": snapshot.package_source,
        "package_length_cm": _money(snapshot.package_length_cm),
        "package_width_cm": _money(snapshot.package_width_cm),
        "package_height_cm": _money(snapshot.package_height_cm),
        "package_weight_kg": float(snapshot.package_weight_kg or 0),
        "provider_id": snapshot.provider_id,
        "provider_name": snapshot.provider_name,
        "channel_id": snapshot.channel_id,
        "channel_name": snapshot.channel_name,
        "currency": snapshot.currency or "CNY",
        "actual_weight_kg": float(snapshot.actual_weight_kg or 0),
        "volumetric_weight_kg": float(snapshot.volumetric_weight_kg or 0),
        "chargeable_weight_kg": float(snapshot.chargeable_weight_kg or 0),
        "base_shipping_fee": _money(snapshot.base_shipping_fee),
        "surcharge_fee": _money(snapshot.surcharge_fee),
        "fuel_surcharge_fee": _money(snapshot.fuel_surcharge_fee),
        "total_shipping_fee": _money(snapshot.total_shipping_fee),
        "calculation_detail": snapshot.calculation_detail,
        "created_at": snapshot.created_at,
        "updated_at": snapshot.updated_at,
    }


def order_to_dict(
    order: Order,
    items: list[OrderItem],
    logs: Optional[list[OrderStatusLog]] = None,
    shipping_snapshot: Optional[dict] = None,
) -> dict:
    total_quantity = sum(item.quantity for item in items)
    if not items:
        product_name = None
    elif len(items) == 1:
        product_name = items[0].product_name
    else:
        product_name = f"{items[0].product_name} 等{len(items)}件"

    shipping_cost_layer = COST_LAYER_SNAPSHOT if shipping_snapshot else COST_LAYER_ESTIMATED
    platform_fee_cost_layer = COST_LAYER_ESTIMATED
    profit_cost_layer = resolve_profit_cost_layer(shipping_cost_layer, platform_fee_cost_layer)

    return {
        "id": order.id,
        "order_no": order.order_no,
        "status": order.status,
        "recipient_name": order.recipient_name,
        "recipient_phone": order.recipient_phone,
        "shipping_address": order.shipping_address,
        "product_name": product_name,
        "quantity": total_quantity,
        "total_amount": _money(order.total_amount),
        "shipping_fee": _money(order.shipping_fee),
        "pay_amount": _money(order.pay_amount),
        "platform_fee": _money(order.platform_fee),
        "payment_fee": _money(order.payment_fee),
        "other_fee": _money(order.other_fee),
        "product_cost": _money(order.product_cost),
        "profit_amount": _money(order.profit_amount),
        "profit_margin": float(order.profit_margin or 0),
        "profit": {
            "revenue_amount": _money(order.total_amount),
            "product_cost": _money(order.product_cost),
            "shipping_fee": _money(order.shipping_fee),
            "shipping_cost_layer": shipping_cost_layer,
            "platform_fee": _money(order.platform_fee),
            "platform_fee_cost_layer": platform_fee_cost_layer,
            "payment_fee": _money(order.payment_fee),
            "other_fee": _money(order.other_fee),
            "profit_amount": _money(order.profit_amount),
            "profit_margin": float(order.profit_margin or 0),
            "profit_cost_layer": profit_cost_layer,
        },
        "shipping_snapshot": shipping_snapshot,
        "payment_method": order.payment_method,
        "remark": order.remark,
        "paid_at": order.paid_at,
        "shipped_at": order.shipped_at,
        "delivered_at": order.delivered_at,
        "cancelled_at": order.cancelled_at,
        "created_at": order.created_at,
        "updated_at": order.updated_at,
        "items": [_item_to_dict(item) for item in items],
        "status_logs": [_log_to_dict(log, order.status) for log in (logs or [])],
    }


class OrderService:
    @staticmethod
    async def create(db: AsyncSession, data: OrderCreate) -> dict:
        order = Order(
            order_no=await OrderService._generate_order_no(db),
            status="pending",
            recipient_name=data.recipient_name,
            recipient_phone=data.recipient_phone,
            shipping_address=data.shipping_address,
            shipping_fee=Decimal(str(data.shipping_fee)),
            platform_fee=Decimal(str(data.platform_fee)),
            payment_fee=Decimal(str(data.payment_fee)),
            other_fee=Decimal(str(data.other_fee)),
            product_cost=Decimal(str(data.product_cost)),
            payment_method=data.payment_method,
            remark=data.remark,
        )
        db.add(order)
        await db.flush()

        items: list[OrderItem] = []
        total_amount = Decimal("0")
        for input_item in data.items:
            sku, product = await OrderService._get_sku_product(db, input_item.sku_id)
            unit_price = Decimal(str(input_item.unit_price)) if input_item.unit_price is not None else await OrderService._get_sale_price(db, sku)
            subtotal = unit_price * input_item.quantity
            total_amount += subtotal
            item = OrderItem(
                order_id=order.id,
                sku_id=sku.id,
                product_id=product.id,
                product_name=product.name,
                sku_code=sku.code,
                spec_desc=sku.spec_desc,
                unit_price=unit_price,
                quantity=input_item.quantity,
                subtotal=subtotal,
            )
            db.add(item)
            items.append(item)

        order.total_amount = total_amount
        order.pay_amount = total_amount + Decimal(str(data.shipping_fee))
        OrderService._recalculate_profit(order)

        # 锁定库存
        try:
            for item in items:
                await InventoryService.lock_stock(
                    db,
                    sku_id=item.sku_id,
                    quantity=item.quantity,
                    order_no=order.order_no,
                    operator="system",
                )
        except ValueError as e:
            raise ValueError(str(e))

        log = OrderStatusLog(
            order_id=order.id,
            from_status=None,
            to_status="pending",
            operator="system",
            remark="创建订单",
        )
        db.add(log)
        await db.flush()
        await db.refresh(order)
        for item in items:
            await db.refresh(item)
        await db.refresh(log)
        return order_to_dict(order, items, [log])

    @staticmethod
    async def list_orders(db: AsyncSession, status: Optional[str], page: int, page_size: int) -> tuple[list[dict], int]:
        stmt = select(Order)
        if status:
            stmt = stmt.where(Order.status == status)
        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0
        stmt = stmt.order_by(Order.created_at.desc()).offset((page - 1) * page_size).limit(page_size)
        result = await db.execute(stmt)
        orders = list(result.scalars().all())
        rows = []
        for order in orders:
            items = await OrderService._get_items(db, order.id)
            rows.append(order_to_dict(order, items))
        return rows, total

    @staticmethod
    async def get_detail(db: AsyncSession, order_id: int) -> Optional[dict]:
        order = await db.get(Order, order_id)
        if not order:
            return None
        items = await OrderService._get_items(db, order.id)
        logs = await OrderService._get_logs(db, order.id)
        shipping_snapshot = await OrderService._get_shipping_snapshot(db, order.id)
        return order_to_dict(order, items, logs, shipping_snapshot)

    @staticmethod
    async def update_status(db: AsyncSession, order_id: int, status: str, remark: str = None, operator: str = "system") -> Optional[dict]:
        order = await db.get(Order, order_id)
        if not order:
            return None
        if status not in ALLOWED_TRANSITIONS:
            raise ValueError("未知订单状态")
        allowed = ALLOWED_TRANSITIONS.get(order.status, set())
        if status not in allowed:
            raise ValueError(f"订单状态不能从 {order.status} 变更为 {status}")

        old_status = order.status
        order.status = status
        now = datetime.utcnow()
        if status == "paid":
            order.paid_at = now
        elif status == "shipped":
            order.shipped_at = now
        elif status == "delivered":
            order.delivered_at = now
        elif status == "cancelled":
            order.cancelled_at = now

        # 库存变动
        items = await OrderService._get_items(db, order.id)
        if old_status == "pending" and status == "paid":
            for item in items:
                await InventoryService.confirm_locked_stock_deduction(
                    db, sku_id=item.sku_id, quantity=item.quantity,
                    order_no=order.order_no, operator=operator,
                )
        elif old_status == "pending" and status == "cancelled":
            for item in items:
                await InventoryService.release_locked_stock(
                    db, sku_id=item.sku_id, quantity=item.quantity,
                    order_no=order.order_no, operator=operator,
                )
        # paid -> cancelled does not restore physical stock in this phase.
        # Returns/refunds need a separate after-sale workflow.

        log = OrderStatusLog(
            order_id=order.id,
            from_status=old_status,
            to_status=status,
            operator=operator,
            remark=remark,
        )
        db.add(log)
        await db.flush()
        await db.refresh(order)
        return await OrderService.get_detail(db, order.id)

    @staticmethod
    def _recalculate_profit(order: Order) -> None:
        revenue = _decimal(order.total_amount)
        costs = (
            _decimal(order.product_cost)
            + _decimal(order.shipping_fee)
            + _decimal(order.platform_fee)
            + _decimal(order.payment_fee)
            + _decimal(order.other_fee)
        )
        profit = revenue - costs
        order.profit_amount = profit
        order.profit_margin = _pct(profit, revenue)
        order.pay_amount = revenue + _decimal(order.shipping_fee)

    @staticmethod
    async def bind_shipping_quote(db: AsyncSession, order_id: int, data: OrderShippingQuoteBind) -> Optional[dict]:
        order = await db.get(Order, order_id)
        if not order:
            return None

        calc = await CalculateService.calculate(
            db,
            CalculateRequest(
                sku_id=data.sku_id,
                quantity=data.quantity,
                destination_country=data.destination_country,
                postal_code=data.postal_code,
                cargo_type=data.cargo_type,
            ),
        )
        if not calc.results:
            raise ValueError("没有可用物流报价")

        if data.channel_id is not None:
            selected = next((item for item in calc.results if item.channel_id == data.channel_id), None)
            if selected is None:
                raise ValueError("指定物流渠道不可用")
        else:
            selected = calc.results[0]

        existing = await OrderService._get_shipping_snapshot_model(db, order_id)
        snapshot = existing or OrderShippingSnapshot(order_id=order_id)
        snapshot.sku_id = data.sku_id
        snapshot.quantity = data.quantity
        snapshot.destination_country = calc.destination_country
        snapshot.postal_code = data.postal_code
        snapshot.cargo_type = data.cargo_type
        snapshot.package_source = calc.package.source
        snapshot.package_length_cm = Decimal(str(calc.package.length_cm))
        snapshot.package_width_cm = Decimal(str(calc.package.width_cm))
        snapshot.package_height_cm = Decimal(str(calc.package.height_cm))
        snapshot.package_weight_kg = Decimal(str(calc.package.weight_kg))
        snapshot.provider_id = selected.provider_id
        snapshot.provider_name = selected.provider_name
        snapshot.channel_id = selected.channel_id
        snapshot.channel_name = selected.channel_name
        snapshot.currency = selected.currency
        snapshot.actual_weight_kg = Decimal(str(selected.actual_weight_kg))
        snapshot.volumetric_weight_kg = Decimal(str(selected.volumetric_weight_kg))
        snapshot.chargeable_weight_kg = Decimal(str(selected.chargeable_weight_kg))
        snapshot.base_shipping_fee = Decimal(str(selected.base_shipping_fee))
        snapshot.surcharge_fee = Decimal(str(selected.surcharge_fee))
        snapshot.fuel_surcharge_fee = Decimal(str(selected.fuel_surcharge_fee))
        snapshot.total_shipping_fee = Decimal(str(selected.total_shipping_fee))
        snapshot.calculation_detail = selected.calculation_detail
        if existing is None:
            db.add(snapshot)

        order.shipping_fee = Decimal(str(selected.total_shipping_fee))
        OrderService._recalculate_profit(order)

        await db.flush()
        await db.refresh(order)
        await db.refresh(snapshot)
        return await OrderService.get_detail(db, order_id)

    @staticmethod
    async def update_profit_inputs(db: AsyncSession, order_id: int, data: OrderProfitInputsUpdate) -> Optional[dict]:
        order = await db.get(Order, order_id)
        if not order:
            return None
        update_data = data.model_dump(exclude_unset=True)
        for field, value in update_data.items():
            setattr(order, field, Decimal(str(value)))
        OrderService._recalculate_profit(order)
        await db.flush()
        await db.refresh(order)
        return await OrderService.get_detail(db, order_id)

    @staticmethod
    async def _generate_order_no(db: AsyncSession) -> str:
        prefix = datetime.utcnow().strftime("MS%Y%m%d%H%M%S")
        count = await db.scalar(select(func.count()).select_from(Order)) or 0
        return f"{prefix}{count + 1:04d}"

    @staticmethod
    async def _get_sku_product(db: AsyncSession, sku_id: int) -> tuple[Sku, Product]:
        stmt = (
            select(Sku, Product)
            .join(Product, Sku.product_id == Product.id)
            .where(Sku.id == sku_id)
        )
        result = await db.execute(stmt)
        row = result.one_or_none()
        if not row:
            raise ValueError("SKU不存在")
        return row

    @staticmethod
    async def _get_sale_price(db: AsyncSession, sku: Sku) -> Decimal:
        stmt = (
            select(Price)
            .where(Price.sku_id == sku.id, Price.price_type == "sale_price", Price.status == 1)
            .order_by(Price.created_at.desc())
            .limit(1)
        )
        result = await db.execute(stmt)
        price = result.scalar_one_or_none()
        if price:
            return Decimal(str(price.price))
        return Decimal(str(sku.price or 0))

    @staticmethod
    async def _get_items(db: AsyncSession, order_id: int) -> list[OrderItem]:
        result = await db.execute(
            select(OrderItem).where(OrderItem.order_id == order_id).order_by(OrderItem.id)
        )
        return list(result.scalars().all())

    @staticmethod
    async def _get_logs(db: AsyncSession, order_id: int) -> list[OrderStatusLog]:
        result = await db.execute(
            select(OrderStatusLog).where(OrderStatusLog.order_id == order_id).order_by(OrderStatusLog.id)
        )
        return list(result.scalars().all())

    @staticmethod
    async def _get_shipping_snapshot_model(db: AsyncSession, order_id: int) -> Optional[OrderShippingSnapshot]:
        result = await db.execute(
            select(OrderShippingSnapshot).where(OrderShippingSnapshot.order_id == order_id)
        )
        return result.scalar_one_or_none()

    @staticmethod
    async def _get_shipping_snapshot(db: AsyncSession, order_id: int) -> Optional[dict]:
        snapshot = await OrderService._get_shipping_snapshot_model(db, order_id)
        return _shipping_snapshot_to_dict(snapshot) if snapshot else None
