"""平台费用规则 - Pydantic Schema"""

from typing import Optional
from decimal import Decimal
from pydantic import BaseModel, Field


class PlatformFeeRuleCreate(BaseModel):
    """创建平台费用规则"""
    platform_id: int = Field(..., description="平台ID")
    site_code: Optional[str] = Field(None, description="站点/国家代码，空表示全局")
    category_id: Optional[int] = Field(None, description="本地类目ID")
    commission_pct: float = Field(0, ge=0, le=100, description="平台佣金比例(%)")
    payment_fee_pct: float = Field(0, ge=0, le=100, description="支付手续费比例(%)")
    fixed_fee: float = Field(0, ge=0, description="固定交易费")
    advertising_pct: float = Field(0, ge=0, le=100, description="广告/营销预留比例(%)")
    other_reserve_fee: float = Field(0, ge=0, description="其他固定预留费用")
    priority: int = Field(0, ge=0, description="优先级，值小优先")
    status: int = Field(1, description="状态: 0-禁用, 1-启用")
    remark: Optional[str] = Field(None, description="备注")


class PlatformFeeRuleUpdate(BaseModel):
    """更新平台费用规则（仅传递需要更新的字段）"""
    site_code: Optional[str] = None
    category_id: Optional[int] = None
    commission_pct: Optional[float] = None
    payment_fee_pct: Optional[float] = None
    fixed_fee: Optional[float] = None
    advertising_pct: Optional[float] = None
    other_reserve_fee: Optional[float] = None
    priority: Optional[int] = None
    status: Optional[int] = None
    remark: Optional[str] = None


class PlatformFeeRuleMatchRequest(BaseModel):
    """匹配平台费用规则请求"""
    platform_id: int = Field(..., description="平台ID")
    site_code: Optional[str] = Field(None, description="站点/国家代码")
    category_id: Optional[int] = Field(None, description="本地类目ID")


class PlatformFeeRuleVO(BaseModel):
    """平台费用规则视图"""
    id: int
    platform_id: int
    platform_name: Optional[str] = None
    site_code: Optional[str] = None
    category_id: Optional[int] = None
    commission_pct: float
    payment_fee_pct: float
    fixed_fee: float
    advertising_pct: float
    other_reserve_fee: float
    priority: int
    status: int
    remark: Optional[str] = None
    created_at: str
    updated_at: str
