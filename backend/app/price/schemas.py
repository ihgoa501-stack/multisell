"""价格管理 - Pydantic Schema"""

from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, ConfigDict, Field


class PriceCreate(BaseModel):
    """设置价格"""

    sku_id: int = Field(..., description="SKU ID")
    price_type: str = Field(
        ...,
        description="价格类型: sale_price/market_price/cost_price/vip_price/wholesale_price",
    )
    price: float = Field(..., gt=0, description="价格")
    start_time: Optional[datetime] = Field(None, description="生效时间")
    end_time: Optional[datetime] = Field(None, description="失效时间")


class PriceBatchCreate(BaseModel):
    """批量调价"""

    sku_ids: List[int] = Field(..., min_length=1, description="SKU ID列表")
    price_type: str = Field(..., description="价格类型")
    price: float = Field(..., gt=0, description="新价格")
    start_time: Optional[datetime] = Field(None, description="生效时间")
    end_time: Optional[datetime] = Field(None, description="失效时间")


class PriceVO(BaseModel):
    """价格响应"""

    id: int
    sku_id: int
    price_type: str
    price: float
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    status: int = 1
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class PriceChangeLogVO(BaseModel):
    """调价记录响应"""

    id: int
    sku_id: int
    old_price: Optional[float] = None
    new_price: Optional[float] = None
    price_type: Optional[str] = None
    change_type: Optional[str] = None
    operator: Optional[str] = None
    remark: Optional[str] = None
    created_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)
