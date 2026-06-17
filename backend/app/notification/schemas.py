"""通知与预警 - Pydantic 模型"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class NotificationVO(BaseModel):
    """通知视图"""
    id: int
    user_id: int
    alert_type: str
    title: str
    content: Optional[str] = None
    link_url: Optional[str] = None
    severity: str
    is_read: bool
    source_id: Optional[str] = None
    created_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class AlertRuleVO(BaseModel):
    """预警规则视图"""
    id: int
    name: str
    alert_type: str
    enabled: bool
    config: Optional[dict] = None
    description: Optional[str] = None

    class Config:
        from_attributes = True


class AlertRuleUpdate(BaseModel):
    """更新预警规则"""
    enabled: Optional[bool] = None
    config: Optional[dict] = None
    description: Optional[str] = None


class UnreadCountVO(BaseModel):
    """未读数"""
    total: int = 0
    by_type: dict = {}
