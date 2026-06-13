"""平台管理 - Pydantic Schema"""
from datetime import datetime
from typing import Optional, Any
from pydantic import BaseModel, Field


class PlatformCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=100, description="平台名称")
    code: str = Field(..., min_length=1, max_length=50, description="平台代码（ozon, shopee, wb）")
    api_base_url: Optional[str] = Field(None, max_length=500, description="API基础地址")
    api_key: Optional[str] = Field(None, max_length=500, description="API密钥")
    client_id: Optional[str] = Field(None, max_length=200, description="Client ID")
    extra_config: Optional[Any] = Field(None, description="额外配置")
    sort_order: int = Field(0, description="排序")


class PlatformUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    code: Optional[str] = Field(None, min_length=1, max_length=50)
    api_base_url: Optional[str] = Field(None, max_length=500)
    api_key: Optional[str] = Field(None, max_length=500)
    client_id: Optional[str] = Field(None, max_length=200)
    extra_config: Optional[Any] = None
    status: Optional[int] = Field(None, description="状态: 0-禁用, 1-启用")
    sort_order: Optional[int] = None


class PlatformVO(BaseModel):
    id: int
    name: str
    code: str
    api_base_url: Optional[str] = None
    client_id: Optional[str] = None
    extra_config: Optional[Any] = None
    status: int = 1
    sort_order: int = 0
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True
