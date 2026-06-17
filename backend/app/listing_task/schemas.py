"""多平台上架任务 - 数据模型"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel


class ListingTaskCreate(BaseModel):
    name: str
    product_ids: list[int]
    platform_ids: list[int]


class ListingTaskItemVO(BaseModel):
    id: int
    task_id: int
    product_id: int
    platform_id: int
    product_name: Optional[str] = None
    platform_name: Optional[str] = None
    platform_code: Optional[str] = None
    status: str
    result: Optional[dict] = None
    error_message: Optional[str] = None
    retry_count: int = 0
    executed_at: Optional[datetime] = None


class ListingTaskVO(BaseModel):
    id: int
    name: str
    status: str
    total_count: int
    success_count: int
    failed_count: int
    created_by: Optional[int] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    items: list[ListingTaskItemVO] = []
