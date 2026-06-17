"""平台费用 - Pydantic Schema"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class PlatformFeeRuleCreate(BaseModel):
    """创建费用规则"""
    platform_id: int = Field(..., description="平台ID")
    country_code: Optional[str] = Field(None, description="国家代码，null表示平台默认")
    category_id: Optional[int] = Field(None, description="类目ID，null表示类目默认")
    fee_type: str = Field(..., description="commission/fixed/payment/storage/other")
    fee_rate_pct: float = Field(0, description="费率(%)")
    fixed_amount: float = Field(0, description="固定费用")
    min_amount: Optional[float] = Field(None, description="最低费用")
    max_amount: Optional[float] = Field(None, description="最高费用")
    currency: str = Field("CNY", description="币种")
    effective_from: Optional[datetime] = Field(None, description="生效时间")
    effective_to: Optional[datetime] = Field(None, description="失效时间")
    priority: int = Field(0, description="优先级，小值优先")
    status: str = Field("active", description="active/inactive")
    remark: Optional[str] = Field(None, description="备注")


class PlatformFeeRuleUpdate(BaseModel):
    """更新费用规则"""
    country_code: Optional[str] = None
    category_id: Optional[int] = None
    fee_type: Optional[str] = None
    fee_rate_pct: Optional[float] = None
    fixed_amount: Optional[float] = None
    min_amount: Optional[float] = None
    max_amount: Optional[float] = None
    currency: Optional[str] = None
    effective_from: Optional[datetime] = None
    effective_to: Optional[datetime] = None
    priority: Optional[int] = None
    status: Optional[str] = None
    remark: Optional[str] = None


class PlatformFeeRuleVO(BaseModel):
    """费用规则响应"""
    id: int
    platform_id: int
    country_code: Optional[str] = None
    category_id: Optional[int] = None
    fee_type: str
    fee_rate_pct: float = 0
    fixed_amount: float = 0
    min_amount: Optional[float] = None
    max_amount: Optional[float] = None
    currency: str = "CNY"
    effective_from: Optional[datetime] = None
    effective_to: Optional[datetime] = None
    priority: int = 0
    status: str = "active"
    remark: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class PlatformFeeRuleQuery(BaseModel):
    """费用规则查询参数"""
    platform_id: Optional[int] = Field(None, description="平台ID")
    country_code: Optional[str] = Field(None, description="国家代码")
    fee_type: Optional[str] = Field(None, description="费用类型")
    status: Optional[str] = Field(None, description="状态")
    page: int = Field(1, ge=1, description="页码")
    page_size: int = Field(20, ge=1, le=200, description="每页条数")


class PlatformFeeCalculateRequest(BaseModel):
    """费用计算请求"""
    platform_id: int = Field(..., description="平台ID")
    country_code: str = Field("RU", description="国家代码")
    category_id: Optional[int] = Field(None, description="类目ID")
    sale_price: float = Field(..., gt=0, description="售价")
    currency: str = Field("CNY", description="币种")


class PlatformFeeCalculateItem(BaseModel):
    """费用明细项"""
    fee_type: str
    rule_id: int
    description: str
    amount: float


class PlatformFeeCalculateResponse(BaseModel):
    """费用计算结果"""
    platform_id: int
    country_code: str
    category_id: Optional[int] = None
    sale_price: float
    total_fee: float
    items: list[PlatformFeeCalculateItem]
    rules_matched: int
