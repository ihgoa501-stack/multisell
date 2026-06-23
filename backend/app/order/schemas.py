"""订单管理 - Pydantic Schema"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class OrderItemCreate(BaseModel):
    sku_id: int = Field(..., description="SKU ID")
    quantity: int = Field(..., ge=1, description="数量")
    unit_price: Optional[float] = Field(None, ge=0, description="指定成交单价")


class OrderCreate(BaseModel):
    recipient_name: str = Field(..., min_length=1, max_length=100)
    recipient_phone: Optional[str] = Field(None, max_length=50)
    shipping_address: Optional[str] = Field(None, max_length=500)
    shipping_fee: float = Field(0, ge=0)
    platform_fee: float = Field(0, ge=0)
    payment_fee: float = Field(0, ge=0)
    other_fee: float = Field(0, ge=0)
    product_cost: float = Field(0, ge=0)
    payment_method: Optional[str] = Field(None, max_length=50)
    remark: Optional[str] = None
    items: list[OrderItemCreate] = Field(..., min_length=1)


class OrderStatusUpdate(BaseModel):
    status: str = Field(..., max_length=50)
    remark: Optional[str] = None
    tracking_number: Optional[str] = Field(
        None, max_length=200, description="运单号/追踪号"
    )
    operator: str = "system"


class OrderShippingQuoteBind(BaseModel):
    sku_id: int = Field(..., description="用于试算的 SKU ID")
    quantity: int = Field(1, ge=1, description="试算数量")
    destination_country: str = Field(..., min_length=2, max_length=10)
    postal_code: Optional[str] = Field(None, max_length=20)
    cargo_type: str = Field("normal", max_length=50)
    channel_id: Optional[int] = Field(None, description="为空时选择最低价渠道")


class OrderProfitInputsUpdate(BaseModel):
    platform_fee: Optional[float] = Field(None, ge=0)
    payment_fee: Optional[float] = Field(None, ge=0)
    other_fee: Optional[float] = Field(None, ge=0)
    product_cost: Optional[float] = Field(None, ge=0)


class OrderItemVO(BaseModel):
    id: int
    order_id: int
    sku_id: int
    product_id: int
    product_name: str
    sku_code: Optional[str] = None
    spec_desc: Optional[str] = None
    unit_price: float
    quantity: int
    subtotal: float


class OrderStatusLogVO(BaseModel):
    id: int
    order_id: int
    from_status: Optional[str] = None
    to_status: str
    operator: Optional[str] = None
    remark: Optional[str] = None
    is_current: bool = False
    created_at: Optional[datetime] = None


class OrderShippingSnapshotVO(BaseModel):
    id: int
    order_id: int
    sku_id: int
    quantity: int
    destination_country: str
    postal_code: Optional[str] = None
    cargo_type: Optional[str] = None
    package_source: Optional[str] = None
    package_length_cm: float
    package_width_cm: float
    package_height_cm: float
    package_weight_kg: float
    provider_id: int
    provider_name: str
    channel_id: int
    channel_name: str
    currency: str = "CNY"
    actual_weight_kg: float
    volumetric_weight_kg: float
    chargeable_weight_kg: float
    base_shipping_fee: float
    surcharge_fee: float = 0
    fuel_surcharge_fee: float = 0
    total_shipping_fee: float
    calculation_detail: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None


class OrderProfitVO(BaseModel):
    revenue_amount: float
    product_cost: float
    shipping_fee: float
    shipping_cost_layer: str = "estimated"
    platform_fee: float
    platform_fee_cost_layer: str = "estimated"
    payment_fee: float
    other_fee: float
    profit_amount: float
    profit_margin: float
    profit_cost_layer: str = "estimated"


class OrderVO(BaseModel):
    id: int
    order_no: str
    status: str
    recipient_name: Optional[str] = None
    recipient_phone: Optional[str] = None
    shipping_address: Optional[str] = None
    product_name: Optional[str] = None
    quantity: int = 0
    total_amount: float
    shipping_fee: float
    pay_amount: float
    platform_fee: float = 0
    payment_fee: float = 0
    other_fee: float = 0
    product_cost: float = 0
    profit_amount: float = 0
    profit_margin: float = 0
    profit: Optional[OrderProfitVO] = None
    shipping_snapshot: Optional[OrderShippingSnapshotVO] = None
    payment_method: Optional[str] = None
    remark: Optional[str] = None
    paid_at: Optional[datetime] = None
    shipped_at: Optional[datetime] = None
    delivered_at: Optional[datetime] = None
    cancelled_at: Optional[datetime] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    items: list[OrderItemVO] = []
    status_logs: list[OrderStatusLogVO] = []
