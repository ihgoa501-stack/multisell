"""结算管理 - Pydantic 模型"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, ConfigDict, Field


class SettlementCreate(BaseModel):
    """导入结算单"""

    platform_id: int = Field(..., description="平台ID")
    settlement_no: str = Field(..., description="结算单号")
    period_start: Optional[datetime] = None
    period_end: Optional[datetime] = None
    currency: str = "CNY"
    total_revenue: float = 0
    total_fee: float = 0
    total_refund: float = 0
    total_net: float = 0
    raw_data: Optional[dict] = None


class SettlementUpdate(BaseModel):
    """更新结算单"""

    status: Optional[str] = None
    total_revenue: Optional[float] = None
    total_fee: Optional[float] = None
    total_refund: Optional[float] = None
    total_net: Optional[float] = None


class SettlementItemCreate(BaseModel):
    """添加结算明细"""

    transaction_type: str = Field(..., description="交易类型")
    transaction_id: Optional[str] = None
    order_no: Optional[str] = None
    order_id: Optional[int] = None
    sku_id: Optional[int] = None
    amount: float = 0
    fee: float = 0
    net: float = 0
    quantity: int = 0
    occurred_at: Optional[datetime] = None


class SettlementItemVO(BaseModel):
    """结算明细视图"""

    id: int
    settlement_id: int
    transaction_type: str
    transaction_id: Optional[str] = None
    order_no: Optional[str] = None
    order_id: Optional[int] = None
    sku_id: Optional[int] = None
    amount: float
    fee: float
    net: float
    quantity: int
    occurred_at: Optional[datetime] = None
    created_at: Optional[datetime] = None
    reconciliation_status: str
    reconciliation_note: Optional[str] = None
    reconciled_at: Optional[datetime] = None
    reconciled_by: Optional[str] = None

    model_config = ConfigDict(from_attributes=True)


class SettlementVO(BaseModel):
    """结算单视图"""

    id: int
    platform_id: int
    platform_name: Optional[str] = None
    settlement_no: str
    period_start: Optional[datetime] = None
    period_end: Optional[datetime] = None
    currency: str
    total_revenue: float
    total_fee: float
    total_refund: float
    total_net: float
    status: str
    item_count: int = 0
    matched_count: int = 0
    unmatched_count: int = 0
    discrepancy_count: int = 0
    imported_at: Optional[datetime] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class SettlementQuery(BaseModel):
    """结算单查询"""

    platform_id: Optional[int] = None
    status: Optional[str] = None
    keyword: Optional[str] = None


class SettlementReconcileRequest(BaseModel):
    """对账请求"""

    auto_match: bool = True
    strategy: str = "by_order_no"  # by_order_no / by_transaction_id


class SettlementImportResponse(BaseModel):
    """导入结果"""

    settlement_id: int
    settlement_no: str
    items_count: int
    total_revenue: float
    total_fee: float
    total_net: float
