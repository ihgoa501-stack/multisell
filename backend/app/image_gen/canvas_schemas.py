"""AI 生图 - 画布数据模型"""

from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, Field


class CanvasLayerItem(BaseModel):
    id: str = Field(..., description="图层唯一ID")
    type: str = Field(..., description="类型: image/text/mask")
    fabric_json: dict = Field(..., description="Fabric.js 对象 JSON")


class CanvasSaveRequest(BaseModel):
    product_id: int = Field(..., description="关联商品ID")
    name: str = Field("未命名画布", max_length=200)
    layers: List[CanvasLayerItem] = Field(default_factory=list)


class CanvasItem(BaseModel):
    id: int
    product_id: int
    name: str
    layers: List[dict]
    thumbnail: Optional[str] = None
    created_by: int
    created_at: str
    updated_at: str


class CanvasListResponse(BaseModel):
    items: List[CanvasItem] = Field(default_factory=list)
    total: int = 0
