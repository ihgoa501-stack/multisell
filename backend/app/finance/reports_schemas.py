"""财务报表 - Pydantic Schema"""

from typing import Optional

from pydantic import BaseModel


class ProfitSummaryResponse(BaseModel):
    revenue_amount: float = 0
    product_cost: float = 0
    shipping_cost: float = 0
    platform_fee: float = 0
    payment_fee: float = 0
    refund: float = 0
    adjustment: float = 0
    allocated_cost: float = 0
    other_fee: float = 0
    profit_amount: float = 0
    profit_margin: float = 0


class OrderProfitItem(BaseModel):
    order_id: int
    order_no: str
    revenue_amount: float
    product_cost: float
    shipping_cost: float
    platform_fee: float
    payment_fee: float
    other_fee: float
    allocated_cost: float = 0
    refund: float = 0
    adjustment: float = 0
    profit_amount: float
    profit_margin: float
    shipping_cost_layer: str = "estimated"
    platform_fee_cost_layer: str = "estimated"
    profit_cost_layer: str = "estimated"


class CostVarianceItem(BaseModel):
    order_id: int
    order_no: str
    snapshot_amount: Optional[float] = None
    bill_amount: Optional[float] = None
    variance_amount: Optional[float] = None
    variance_pct: Optional[float] = None
    status: str = "no_data"


class NegativeProfitItem(BaseModel):
    order_id: int
    order_no: str
    profit_amount: float
    profit_margin: float
    shipping_cost_layer: str = "estimated"
    platform_fee_cost_layer: str = "estimated"


class CostLayerMixResponse(BaseModel):
    layers: list[dict] = []
