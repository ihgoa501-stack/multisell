"""汇率 Pydantic 模型"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class ExchangeRateCreate(BaseModel):
    from_currency: str = Field(min_length=3, max_length=10, description="源货币")
    to_currency: str = Field(min_length=3, max_length=10, description="目标货币")
    rate: float = Field(gt=0, description="汇率")


class ExchangeRateUpdate(BaseModel):
    rate: float = Field(gt=0, description="汇率")


class ExchangeRateVO(BaseModel):
    id: int
    from_currency: str
    to_currency: str
    rate: float
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
