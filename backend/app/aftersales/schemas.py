"""售后管理 - Pydantic Schema"""

from datetime import datetime
from decimal import Decimal
from typing import Optional

from pydantic import BaseModel, Field


class AfterSalesCreate(BaseModel):
    """创建售后单"""
    order_id: int = Field(..., description="订单ID")
    item_id: Optional[int] = Field(None, description="订单明细ID，全单退货时为空")
    sku_id: int = Field(..., description="SKU ID")
    return_quantity: int = Field(..., ge=1, description="退货数量")
    reason: str = Field(..., description="退货原因: defective/wrong_item/customer_return/other")
    refund_amount: float = Field(0, ge=0, description="退款金额")


class AfterSalesApprove(BaseModel):
    """审批售后单"""
    inspection_result: Optional[str] = Field(None, description="验货结果")


class AfterSalesReject(BaseModel):
    """驳回售后单"""
    rejection_reason: str = Field(..., min_length=1, description="驳回原因")


class AfterSalesReceive(BaseModel):
    """收货"""
    inspection_result: Optional[str] = Field(None, description="验货结果")


class AfterSalesRefund(BaseModel):
    """退款"""
    note: Optional[str] = Field(None, description="退款备注")


class AfterSalesOrderVO(BaseModel):
    """售后单响应"""
    id: int
    order_id: int
    order_no: Optional[str] = None
    item_id: Optional[int] = None
    sku_id: int
    sku_code: Optional[str] = None
    return_quantity: int
    reason: str
    status: str
    refund_amount: float = 0
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

    class Config:
        from_attributes = True
