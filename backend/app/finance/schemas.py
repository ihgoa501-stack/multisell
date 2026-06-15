"""财务账本 - Pydantic Schema"""

from datetime import datetime
from typing import Optional

from pydantic import BaseModel


class LedgerEntryResponse(BaseModel):
    """账本条目响应"""
    id: int
    order_id: int
    entry_type: str
    amount: float
    currency: str = "CNY"
    cost_layer: str
    source_type: Optional[str] = None
    source_id: Optional[int] = None
    description: Optional[str] = None
    created_at: Optional[datetime] = None


class LedgerResponse(BaseModel):
    """账本响应"""
    order_id: int
    entries: list[LedgerEntryResponse]
    total_entries: int


class OrderProfitLedgerResponse(BaseModel):
    """订单利润账本汇总"""
    order_id: int
    revenue_amount: float
    product_cost: float
    shipping_cost: float
    platform_fee: float
    payment_fee: float
    refund: float
    adjustment: float
    other_fee: float
    profit_amount: float
    profit_margin: float
    shipping_cost_layer: str = "estimated"
    platform_fee_cost_layer: str = "estimated"
    profit_cost_layer: str = "estimated"
