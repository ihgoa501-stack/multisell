"""操作日志 - Pydantic Schema"""
from datetime import datetime
from typing import Optional
from pydantic import BaseModel


class OperationLogVO(BaseModel):
    id: int
    module: Optional[str] = None
    action: Optional[str] = None
    resource_id: Optional[str] = None
    content: Optional[str] = None
    operator: Optional[str] = None
    ip: Optional[str] = None
    duration: Optional[int] = None
    created_at: Optional[datetime] = None

    class Config:
        from_attributes = True
