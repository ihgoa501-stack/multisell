"""财务管理 - Pydantic 模型"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class FinanceAccountCreate(BaseModel):
    """创建账户"""
    name: str = Field(..., description="账户名称")
    account_type: str = Field(..., description="platform/payment/bank/cash")
    platform_id: Optional[int] = None
    currency: str = "CNY"
    balance: float = 0


class FinanceAccountVO(BaseModel):
    """账户视图"""
    id: int
    name: str
    account_type: str
    platform_id: Optional[int] = None
    platform_name: Optional[str] = None
    currency: str
    balance: float
    status: int

    class Config:
        from_attributes = True


class FinanceTransactionCreate(BaseModel):
    """创建流水"""
    account_id: int = Field(..., description="账户ID")
    transaction_type: str = Field(..., description="revenue/cost/fee/refund/transfer")
    amount: float = Field(..., description="金额")
    currency: str = "CNY"
    order_id: Optional[int] = None
    settlement_id: Optional[int] = None
    platform_id: Optional[int] = None
    description: Optional[str] = None


class FinanceTransactionVO(BaseModel):
    """流水视图"""
    id: int
    account_id: int
    account_name: Optional[str] = None
    transaction_type: str
    amount: float
    currency: str
    order_id: Optional[int] = None
    settlement_id: Optional[int] = None
    platform_id: Optional[int] = None
    platform_name: Optional[str] = None
    description: Optional[str] = None
    transaction_date: Optional[datetime] = None
    created_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class ProfitSummary(BaseModel):
    """利润汇总"""
    period_start: str
    period_end: str
    total_revenue: float
    total_product_cost: float
    total_shipping_fee: float
    total_platform_fee: float
    total_payment_fee: float
    total_other_fee: float
    total_profit: float
    profit_margin: float
    order_count: int
    platform_breakdown: list[dict] = []


class FinanceReportQuery(BaseModel):
    """财务查询"""
    period_start: Optional[str] = None
    period_end: Optional[str] = None
    platform_id: Optional[int] = None
    group_by: str = "platform"  # platform / daily / monthly
