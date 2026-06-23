"""订单导入 - Pydantic 模型"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, ConfigDict, Field


class OrderImportRowData(BaseModel):
    """单条导入订单数据"""

    platform_order_no: str = Field(..., description="平台订单号")
    order_date: Optional[str] = None
    status: str = "paid"
    recipient_name: Optional[str] = None
    recipient_phone: Optional[str] = None
    shipping_address: Optional[str] = None
    country: Optional[str] = None
    items: list[dict] = Field(
        default_factory=list,
        description="商品明细: [{sku_code, product_name, quantity, unit_price}]",
    )
    total_amount: float = 0
    shipping_fee: float = 0
    platform_fee: float = 0
    payment_fee: float = 0
    currency: str = "CNY"
    cargo_type: str = "normal"


class OrderImportRequest(BaseModel):
    """导入订单请求"""

    source_type: str = Field(..., description="来源: ozon/shopee/wb/manual")
    platform_id: Optional[int] = None
    orders: list[OrderImportRowData] = Field(..., description="订单数据列表")


class OrderImportVO(BaseModel):
    """导入记录视图"""

    id: int
    platform_id: Optional[int] = None
    platform_name: Optional[str] = None
    source_type: str
    file_name: Optional[str] = None
    total_rows: int
    success_count: int
    error_count: int
    status: str
    error_detail: Optional[list] = None
    created_by: Optional[str] = None
    created_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class OrderImportResult(BaseModel):
    """导入结果"""

    import_id: int
    source_type: str
    total: int
    success: int
    failed: int
    errors: list[dict] = []
    orders: list[dict] = []
