"""平台发布适配器基础类型。"""

from dataclasses import dataclass
from datetime import datetime
from typing import Optional, Protocol
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Inventory, Platform, Price, Product, Sku


@dataclass(frozen=True)
class PublishResult:
    platform_product_id: str
    platform_sku: Optional[str]
    platform_url: str
    published_data: dict
    sync_message: Optional[str] = None


class ListingAdapter(Protocol):
    async def publish(
        self,
        *,
        product: Product,
        platform: Platform,
        skus: list[Sku],
        prices: dict[int, Price],
        inventories: dict[int, Inventory],
        db: Optional[AsyncSession] = None,
    ) -> PublishResult:
        """发布商品到平台并返回平台侧同步结果。"""

    async def sync_status(
        self,
        *,
        listing_id: int,
        platform: Platform,
        platform_product_id: str,
    ) -> str:
        """同步平台发布状态。"""

    async def validate_credentials(self, *, platform: Platform) -> bool:
        """校验平台凭证是否可用。"""

    async def push_tracking(
        self,
        *,
        platform: Platform,
        order_sn: str,
        tracking_number: str,
        carrier_code: str = "",
        db: Optional[AsyncSession] = None,
    ) -> bool:
        """将物流追踪号推回平台。"""

    async def fetch_orders(
        self,
        *,
        platform: Platform,
        since: datetime,
        db: Optional[AsyncSession] = None,
    ) -> list[dict]:
        """从平台拉取订单列表。返回 list[dict]，每个 dict 包含:
        - order_sn: str — 平台订单号
        - status: str — 平台状态
        - total_amount: str — 金额字符串
        - shipping_fee: str — 运费字符串
        - paid_at: str — ISO 时间
        - recipient_name: str
        - recipient_phone: str
        - shipping_address: str
        - items: list[{"sku_code": str, "quantity": int, "unit_price": str}]
        """

    async def fetch_settlements(
        self,
        *,
        platform: Platform,
        since: datetime,
        db: Optional[AsyncSession] = None,
    ) -> list[dict]:
        """从平台拉取结算/交易记录。返回 list[dict]，每条包含：
        - transaction_id: str
        - transaction_type: str — order_sale / refund / shipping_fee / platform_fee / payment_fee
        - order_sn: str (optional)
        - amount: str
        - fee: str (optional)
        - currency: str
        - occurred_at: str (ISO datetime)
        - description: str (optional)
        """

    async def fetch_returns(
        self,
        *,
        platform: Platform,
        since: datetime,
        db: Optional[AsyncSession] = None,
    ) -> list[dict]:
        """从平台拉取售后/退货申请。返回 list[dict]:
        - return_id: str — 平台退货ID
        - order_sn: str
        - sku_code: str
        - quantity: int
        - reason: str
        - status: str — platform's return status
        - created_at: str (ISO)
        - refund_amount: str (optional)
        """
