"""商品管理 - Pydantic Schema"""

from datetime import datetime
from typing import Literal, Optional, List
from pydantic import BaseModel, Field


class ProductCreate(BaseModel):
    """创建商品"""
    name: str = Field(..., min_length=1, max_length=200, description="商品名称")
    subtitle: Optional[str] = Field(None, max_length=500, description="副标题")
    description: Optional[str] = Field(None, description="商品描述")
    brand_id: Optional[int] = Field(None, description="品牌ID")
    category_id: Optional[int] = Field(None, description="分类ID")
    unit: str = Field("件", max_length=20, description="单位")
    main_image: Optional[str] = Field(None, description="主图URL")
    images: Optional[List[str]] = Field(None, description="图片列表")
    status: int = Field(0, description="状态: 0-草稿, 1-上架, 2-下架")
    product_length_cm: Optional[float] = Field(None, gt=0, description="商品长(cm)")
    product_width_cm: Optional[float] = Field(None, gt=0, description="商品宽(cm)")
    product_height_cm: Optional[float] = Field(None, gt=0, description="商品高(cm)")
    product_weight_kg: Optional[float] = Field(None, gt=0, description="商品重量(kg)")
    package_length_cm: Optional[float] = Field(None, gt=0, description="包装长(cm)")
    package_width_cm: Optional[float] = Field(None, gt=0, description="包装宽(cm)")
    package_height_cm: Optional[float] = Field(None, gt=0, description="包装高(cm)")
    package_weight_kg: Optional[float] = Field(None, gt=0, description="包装重量(kg)")
    cargo_type: Literal["normal", "battery", "liquid", "sensitive"] = Field("normal", description="货品类型")


class ProductUpdate(BaseModel):
    """更新商品"""
    name: Optional[str] = Field(None, min_length=1, max_length=200)
    subtitle: Optional[str] = Field(None, max_length=500)
    description: Optional[str] = None
    brand_id: Optional[int] = None
    category_id: Optional[int] = None
    unit: Optional[str] = Field(None, max_length=20)
    main_image: Optional[str] = None
    images: Optional[List[str]] = None
    status: Optional[int] = None
    product_length_cm: Optional[float] = Field(None, gt=0)
    product_width_cm: Optional[float] = Field(None, gt=0)
    product_height_cm: Optional[float] = Field(None, gt=0)
    product_weight_kg: Optional[float] = Field(None, gt=0)
    package_length_cm: Optional[float] = Field(None, gt=0)
    package_width_cm: Optional[float] = Field(None, gt=0)
    package_height_cm: Optional[float] = Field(None, gt=0)
    package_weight_kg: Optional[float] = Field(None, gt=0)
    cargo_type: Optional[Literal["normal", "battery", "liquid", "sensitive"]] = None


class ProductVO(BaseModel):
    """商品响应"""
    id: int
    name: str
    subtitle: Optional[str] = None
    description: Optional[str] = None
    brand_id: Optional[int] = None
    category_id: Optional[int] = None
    category_name: Optional[str] = None
    brand_name: Optional[str] = None
    unit: str = "件"
    status: int = 0
    status_name: str = ""
    main_image: Optional[str] = None
    images: Optional[list] = None
    product_length_cm: Optional[float] = None
    product_width_cm: Optional[float] = None
    product_height_cm: Optional[float] = None
    product_weight_kg: Optional[float] = None
    package_length_cm: Optional[float] = None
    package_width_cm: Optional[float] = None
    package_height_cm: Optional[float] = None
    package_weight_kg: Optional[float] = None
    cargo_type: str = "normal"
    logistics_status: str = "incomplete"
    logistics_status_name: str = "物流不完整"
    missing_logistics_fields: list[str] = []
    package_volume_weight_kg: Optional[float] = None
    ai_status: Optional[str] = None
    platform_statuses: Optional[dict] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class ProductQuery(BaseModel):
    """商品查询参数"""
    name: Optional[str] = Field(None, description="商品名称(模糊搜索)")
    category_id: Optional[int] = Field(None, description="分类ID")
    status: Optional[int] = Field(None, description="状态")
    brand_id: Optional[int] = Field(None, description="品牌ID")
    cargo_type: Optional[Literal["normal", "battery", "liquid", "sensitive"]] = Field(None, description="货品类型")
    logistics_status: Optional[Literal["complete", "incomplete"]] = Field(None, description="物流完整状态")
    page: int = Field(1, ge=1)
    page_size: int = Field(20, ge=1, le=100)
