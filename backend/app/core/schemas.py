"""商品管理 - Pydantic Schema"""

from datetime import datetime
from typing import Optional, List
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
    page: int = Field(1, ge=1)
    page_size: int = Field(20, ge=1, le=100)
