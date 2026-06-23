"""1688 货源采集 - Pydantic Schema"""
from datetime import datetime
from typing import Optional
from pydantic import BaseModel, ConfigDict, Field


class CollectPayload(BaseModel):
    """油猴脚本采集 payload"""
    url: str = Field(..., max_length=1000, description="1688 商品链接")
    title: Optional[str] = Field(None, max_length=500, description="采集标题")
    price: Optional[float] = Field(None, description="采集供货价")
    moq: Optional[int] = Field(None, description="最小起订量")
    supplier: Optional[str] = Field(None, max_length=200, description="供应商名称")
    shop_url: Optional[str] = Field(None, max_length=1000, description="1688 店铺链接")
    shop_location: Optional[str] = Field(None, max_length=200, description="店铺地区")
    images: Optional[list[str]] = Field(None, description="图片列表")
    attributes: Optional[list[dict]] = Field(None, description="属性列表")
    skuVariants: Optional[list[dict]] = Field(None, description="SKU 变体")
    description: Optional[str] = Field(None, description="描述")
    length_cm: Optional[float] = Field(None, description="包装长(cm)")
    width_cm: Optional[float] = Field(None, description="包装宽(cm)")
    height_cm: Optional[float] = Field(None, description="包装高(cm)")
    weight_g: Optional[float] = Field(None, description="重量(克)")


class Sourcing1688ProductVO(BaseModel):
    """候选池响应视图"""
    id: int
    source_url: str
    title: Optional[str] = None
    price: Optional[float] = None
    moq: Optional[int] = None
    supplier_name: Optional[str] = None
    shop_url: Optional[str] = None
    shop_location: Optional[str] = None
    images: Optional[list] = None
    attributes: Optional[list] = None
    sku_variants: Optional[list] = None
    description: Optional[str] = None
    package_length_cm: Optional[float] = None
    package_width_cm: Optional[float] = None
    package_height_cm: Optional[float] = None
    package_weight_kg: Optional[float] = None
    status: str = "collected"
    product_id: Optional[int] = None
    supplier_id: Optional[int] = None
    collected_by: Optional[str] = None
    imported_by: Optional[str] = None
    imported_at: Optional[datetime] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class ImportPayload(BaseModel):
    """确认导入时的补充信息"""
    category_id: Optional[int] = Field(None, description="分类 ID")
    brand_id: Optional[int] = Field(None, description="品牌 ID")
    cargo_type: Optional[str] = Field("normal", description="货品类型")
    unit: Optional[str] = Field("件", description="单位")
