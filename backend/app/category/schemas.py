"""分类管理 - Pydantic Schema"""
from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, ConfigDict, Field


class CategoryCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=100, description="分类名称")
    parent_id: int = Field(0, description="父分类ID")
    sort_order: int = Field(0, description="排序")


class CategoryUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    parent_id: Optional[int] = None
    sort_order: Optional[int] = None
    status: Optional[int] = None


class CategoryVO(BaseModel):
    id: int
    name: str
    parent_id: int = 0
    level: int = 0
    sort_order: int = 0
    status: int = 1
    children: List["CategoryVO"] = []
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)
