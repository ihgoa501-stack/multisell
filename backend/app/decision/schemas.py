"""上架前经营决策 - Pydantic Schema"""

from typing import Optional
from pydantic import BaseModel, Field


class PreListingDecisionRequest(BaseModel):
    """上架前经营决策请求"""
    sku_id: int = Field(..., description="SKU ID")
    destination_country: str = Field(..., min_length=2, max_length=10, description="目的国代码(ISO 3166-1 alpha-2)")
    target_sale_price: float = Field(..., gt=0, description="目标售价")
    platform_id: Optional[int] = Field(None, description="平台ID；填写后尝试匹配平台费用规则")
    category_id: Optional[int] = Field(None, description="本地类目ID；为空时从SKU所属商品读取")
    platform_fee_pct: float = Field(default=10, ge=0, le=100, description="平台费率(%)")
    payment_fee_pct: float = Field(default=3, ge=0, le=100, description="支付费率(%)")
    other_fee: float = Field(default=0, ge=0, description="其他费用")
    minimum_margin_pct: float = Field(default=20, ge=0, le=100, description="最低利润率(%)")
    cargo_type: str = Field(default="normal", description="货品类型")


class PreListingDecisionResponse(BaseModel):
    """上架前经营决策响应"""
    sku_id: int
    destination_country: str
    target_sale_price: float
    product_cost: float
    shipping_fee: float
    platform_fee: float
    payment_fee: float
    fixed_fee: float = 0
    advertising_fee: float = 0
    other_fee: float
    profit_amount: float
    profit_margin: float
    recommendation: str  # approve / reject / needs_data
    blocking_reasons: list[str]
    warnings: list[str]
    applied_platform_fee_rule_id: Optional[int] = None
    platform_fee_source: str = "manual"  # manual / rule
    platform_fee_rule_summary: Optional[str] = None
