"""异常工作台 - Pydantic Schema"""

from datetime import datetime
from typing import Optional

from pydantic import BaseModel


class ExceptionItemResponse(BaseModel):
    """异常条目响应"""
    id: int
    source_module: str
    source_type: Optional[str] = None
    source_id: Optional[int] = None
    severity: str = "medium"
    status: str = "open"
    title: str
    description: Optional[str] = None
    recommended_action: Optional[str] = None
    assigned_to: Optional[str] = None
    resolved_at: Optional[str] = None
    resolved_by: Optional[str] = None
    note: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class ExceptionGenerateResponse(BaseModel):
    """生成异常响应"""
    created_count: int
    total_scanned: int


class ExceptionAssignRequest(BaseModel):
    assigned_to: str = ""


class ExceptionNoteRequest(BaseModel):
    note: str = ""
