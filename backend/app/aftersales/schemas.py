"""售后退货 — schemas"""
from datetime import datetime
from decimal import Decimal
from typing import Optional
from pydantic import BaseModel, ConfigDict, Field


class AfterSalesCreate(BaseModel):
    """创建退货申请"""
    order_id: int
    item_id: Optional[int] = None
    sku_id: int
    return_quantity: int = Field(..., gt=0)
    reason: str = Field(..., min_length=1, max_length=200)


class AfterSalesApprove(BaseModel):
    """审批通过"""
    approved_by: str
    refund_amount: Decimal = Field(..., gt=0)


class AfterSalesReject(BaseModel):
    """驳回"""
    rejected_by: str
    rejection_reason: str = Field(..., max_length=500)


class AfterSalesReceive(BaseModel):
    """入库验收"""
    received_by: str
    inspection_result: Optional[str] = None


class AfterSalesRefund(BaseModel):
    """退款"""
    refunded_by: str


class AfterSalesVO(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    order_id: int
    item_id: Optional[int] = None
    sku_id: int
    return_quantity: int
    reason: str
    status: str
    refund_amount: Optional[float] = None
    inspection_result: Optional[str] = None
    rejection_reason: Optional[str] = None
    created_by: Optional[str] = None
    approved_by: Optional[str] = None
    approved_at: Optional[datetime] = None
    rejected_by: Optional[str] = None
    rejected_at: Optional[datetime] = None
    received_by: Optional[str] = None
    received_at: Optional[datetime] = None
    refunded_by: Optional[str] = None
    refunded_at: Optional[datetime] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
