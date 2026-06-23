"""规格与SKU管理 - Pydantic Schema"""

from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, ConfigDict, Field


class SpecValueItem(BaseModel):
    """规格值项"""

    value: str = Field(..., min_length=1, max_length=100, description="规格值")


class SpecItem(BaseModel):
    """规格项"""

    name: str = Field(
        ..., min_length=1, max_length=100, description="规格名称（如：颜色）"
    )
    values: List[str] = Field(
        ..., min_length=1, description="规格值列表（如：['红','蓝']）"
    )


class SpecDefine(BaseModel):
    """定义规格模板"""

    specs: List[SpecItem] = Field(..., min_length=1, description="规格列表")


class SkuUpdate(BaseModel):
    """更新SKU"""

    price: Optional[float] = Field(None, description="销售价")
    cost_price: Optional[float] = Field(None, description="成本价")
    market_price: Optional[float] = Field(None, description="市场价")
    stock: Optional[int] = Field(None, description="库存")
    barcode: Optional[str] = Field(None, description="条形码")
    code: Optional[str] = Field(None, description="SKU编码")
    image: Optional[str] = Field(None, description="图片")
    weight: Optional[float] = Field(None, description="重量(kg)")
    sku_length_cm: Optional[float] = Field(None, gt=0, description="SKU包装长(cm)")
    sku_width_cm: Optional[float] = Field(None, gt=0, description="SKU包装宽(cm)")
    sku_height_cm: Optional[float] = Field(None, gt=0, description="SKU包装高(cm)")
    sku_weight_kg: Optional[float] = Field(None, gt=0, description="SKU包装重量(kg)")
    warning_stock: Optional[int] = Field(None, description="安全库存")
    status: Optional[int] = Field(None, description="状态")


class SpecNameVO(BaseModel):
    """规格名称响应"""

    id: int
    name: str
    sort_order: int = 0
    values: List["SpecValueVO"] = []

    model_config = ConfigDict(from_attributes=True)


class SpecValueVO(BaseModel):
    """规格值响应"""

    id: int
    value: str
    sort_order: int = 0

    model_config = ConfigDict(from_attributes=True)


class SkuVO(BaseModel):
    """SKU响应"""

    id: int
    product_id: int
    code: Optional[str] = None
    barcode: Optional[str] = None
    spec_desc: Optional[str] = None
    spec_values: Optional[dict] = None
    price: Optional[float] = None
    cost_price: Optional[float] = None
    market_price: Optional[float] = None
    stock: int = 0
    lock_stock: int = 0
    warning_stock: int = 0
    weight: Optional[float] = None
    sku_length_cm: Optional[float] = None
    sku_width_cm: Optional[float] = None
    sku_height_cm: Optional[float] = None
    sku_weight_kg: Optional[float] = None
    image: Optional[str] = None
    status: int = 1
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)
