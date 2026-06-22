"""平台集成 - Pydantic Schema"""

from datetime import datetime
from typing import Optional, Any
from pydantic import BaseModel, ConfigDict, Field


# ── Adapter 注册表 ──────────────────────────────────────────────────────

class AdapterCapabilityResponse(BaseModel):
    adapter_code: str
    display_name: str
    supports_listing_publish: bool
    supports_order_import: bool
    supports_settlement_import: bool
    supports_tracking_sync: bool
    auth_type: str


# ── 平台账号/连接配置 ────────────────────────────────────────────────────

class PlatformIntegrationAccountCreate(BaseModel):
    platform_id: int
    adapter_code: str = Field(..., min_length=1, max_length=50)
    account_name: str = Field(..., min_length=1, max_length=200)
    credentials: Optional[dict[str, str]] = None


class PlatformIntegrationAccountUpdate(BaseModel):
    account_name: Optional[str] = Field(None, min_length=1, max_length=200)
    status: Optional[str] = None  # draft/active/disabled
    credentials: Optional[dict[str, str]] = None


class PlatformIntegrationAccountResponse(BaseModel):
    id: int
    platform_id: int
    platform_name: Optional[str] = None
    adapter_code: str
    account_name: str
    status: str
    credential_metadata: Optional[Any] = None
    created_by: Optional[str] = None
    updated_by: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class TestConnectionResponse(BaseModel):
    success: bool
    message: str


# ── 类目映射 ────────────────────────────────────────────────────────────

class PlatformCategoryMappingCreate(BaseModel):
    platform_id: int
    adapter_code: str = Field(..., min_length=1, max_length=50)
    local_category_id: int
    platform_category_id: str = Field(..., min_length=1, max_length=200)
    platform_category_name: Optional[str] = None
    platform_category_path: Optional[str] = None


class PlatformCategoryMappingResponse(BaseModel):
    id: int
    platform_id: int
    platform_name: Optional[str] = None
    adapter_code: str
    local_category_id: int
    local_category_name: Optional[str] = None
    platform_category_id: str
    platform_category_name: Optional[str] = None
    platform_category_path: Optional[str] = None
    created_by: Optional[str] = None
    created_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


# ── 属性映射 ────────────────────────────────────────────────────────────

class PlatformAttributeMappingCreate(BaseModel):
    platform_id: int
    adapter_code: str = Field(..., min_length=1, max_length=50)
    local_attribute: str = Field(..., min_length=1, max_length=100)
    platform_attribute: str = Field(..., min_length=1, max_length=200)
    default_value: Optional[str] = None


class PlatformAttributeMappingResponse(BaseModel):
    id: int
    platform_id: int
    platform_name: Optional[str] = None
    adapter_code: str
    local_attribute: str
    platform_attribute: str
    default_value: Optional[str] = None
    created_by: Optional[str] = None
    created_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)
