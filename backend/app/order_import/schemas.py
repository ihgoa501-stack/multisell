"""订单导入 - Pydantic Schema"""
from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class OrderImportItemCreate(BaseModel):
    row_number: int
    platform_order_no: Optional[str] = None
    order_no: Optional[str] = None
    sku_code: str
    quantity: int = Field(..., ge=1)
    unit_price: Optional[float] = None
    currency: str = Field("CNY", min_length=1, max_length=10)
    recipient_name: Optional[str] = None
    recipient_phone: Optional[str] = None
    country_code: Optional[str] = None
    shipping_address: Optional[str] = None
    shipping_fee: Optional[float] = None
    tracking_number: Optional[str] = None
    paid_at: Optional[str] = None
    raw_payload: Optional[dict] = None


class OrderImportBatchCreate(BaseModel):
    adapter_code: str = Field(default="csv_order", min_length=1, max_length=50)
    platform: Optional[str] = None
    store_name: Optional[str] = None
    source_filename: Optional[str] = None
    items: list[OrderImportItemCreate] = Field(default_factory=list, max_length=5000)


class OrderImportItemVO(BaseModel):
    id: int
    batch_id: int
    row_number: int
    platform_order_no: Optional[str] = None
    order_no: Optional[str] = None
    sku_code: Optional[str] = None
    quantity: Optional[int] = None
    unit_price: Optional[float] = None
    currency: Optional[str] = None
    recipient_name: Optional[str] = None
    recipient_phone: Optional[str] = None
    country_code: Optional[str] = None
    shipping_address: Optional[str] = None
    shipping_fee: Optional[float] = None
    tracking_number: Optional[str] = None
    paid_at: Optional[str] = None
    status: str
    failure_reason: Optional[str] = None
    raw_payload: Optional[dict] = None
    created_at: Optional[datetime] = None


class OrderImportBatchVO(BaseModel):
    id: int
    adapter_code: str
    platform: Optional[str] = None
    store_name: Optional[str] = None
    source_filename: Optional[str] = None
    row_count: int = 0
    created_order_count: int = 0
    skipped_duplicate_count: int = 0
    failed_count: int = 0
    imported_by: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
