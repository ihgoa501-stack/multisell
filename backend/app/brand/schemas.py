"""品牌管理 - Pydantic Schema"""
from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class BrandCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=200, description="品牌名称")
    logo: Optional[str] = Field(None, max_length=500, description="Logo URL")
    description: Optional[str] = Field(None, description="品牌描述")
    sort_order: int = Field(0, description="排序")


class BrandUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=200)
    logo: Optional[str] = Field(None, max_length=500)
    description: Optional[str] = None
    sort_order: Optional[int] = None
    status: Optional[int] = Field(None, description="状态: 0-禁用, 1-启用")


class BrandVO(BaseModel):
    id: int
    name: str
    logo: Optional[str] = None
    description: Optional[str] = None
    status: int = 1
    sort_order: int = 0
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True
