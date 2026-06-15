"""物流运费 - Pydantic Schema"""

from datetime import datetime
from decimal import Decimal
from typing import Literal, Optional
from pydantic import BaseModel, Field, model_validator


# ── Provider ──────────────────────────────────────────────────────────────

class ProviderCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=200, description="供应商名称")
    code: Optional[str] = Field(None, max_length=50, description="编码")
    contact: Optional[str] = Field(None, max_length=100, description="联系人")
    phone: Optional[str] = Field(None, max_length=50, description="联系电话")
    remark: Optional[str] = Field(None, description="备注")


class ProviderUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=200)
    code: Optional[str] = Field(None, max_length=50)
    contact: Optional[str] = Field(None, max_length=100)
    phone: Optional[str] = Field(None, max_length=50)
    remark: Optional[str] = None
    status: Optional[int] = None


class ProviderVO(BaseModel):
    id: int
    name: str
    code: Optional[str] = None
    contact: Optional[str] = None
    phone: Optional[str] = None
    remark: Optional[str] = None
    status: int = 1
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True


# ── Channel ───────────────────────────────────────────────────────────────

class ChannelCreate(BaseModel):
    provider_id: int = Field(..., description="物流供应商ID")
    name: str = Field(..., min_length=1, max_length=200, description="渠道名称")
    code: Optional[str] = Field(None, max_length=50, description="渠道编码")
    volumetric_divisor: int = Field(6000, ge=1, description="抛重系数")
    cargo_types: list[str] = Field(default=["normal"], description="支持货品类型")
    estimated_delivery_min: Optional[int] = Field(None, ge=0, description="最短时效(天)")
    estimated_delivery_max: Optional[int] = Field(None, ge=0, description="最长时效(天)")
    currency: str = Field("CNY", max_length=10, description="报价币种")
    sort_order: int = Field(0, description="排序")


class ChannelUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=200)
    code: Optional[str] = Field(None, max_length=50)
    volumetric_divisor: Optional[int] = Field(None, ge=1)
    cargo_types: Optional[list[str]] = None
    estimated_delivery_min: Optional[int] = Field(None, ge=0)
    estimated_delivery_max: Optional[int] = Field(None, ge=0)
    currency: Optional[str] = Field(None, max_length=10)
    sort_order: Optional[int] = None
    status: Optional[int] = None


class ChannelVO(BaseModel):
    id: int
    provider_id: int
    provider_name: Optional[str] = None
    name: str
    code: Optional[str] = None
    volumetric_divisor: int = 6000
    cargo_types: Optional[list] = None
    estimated_delivery_min: Optional[int] = None
    estimated_delivery_max: Optional[int] = None
    currency: str = "CNY"
    sort_order: int = 0
    status: int = 1
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True


# ── Zone ──────────────────────────────────────────────────────────────────

class ZoneCreate(BaseModel):
    country_code: str = Field(..., min_length=2, max_length=10, description="国家代码(ISO 3166-1 alpha-2)")
    postal_code_from: Optional[str] = Field(None, max_length=20, description="邮编范围起始")
    postal_code_to: Optional[str] = Field(None, max_length=20, description="邮编范围截止")


class ZoneVO(BaseModel):
    id: int
    channel_id: int
    country_code: str
    postal_code_from: Optional[str] = None
    postal_code_to: Optional[str] = None
    status: int = 1
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True


# ── Quote Rule ────────────────────────────────────────────────────────────

class RuleCreate(BaseModel):
    zone_id: Optional[int] = Field(None, description="物流区域ID；为空表示渠道全局规则")
    rule_type: str = Field(..., description="规则类型")
    priority: int = Field(0, description="优先级")
    min_weight_kg: Optional[float] = Field(0, description="适用最小重量(kg)")
    max_weight_kg: Optional[float] = Field(None, description="适用最大重量(kg)")
    first_kg: Optional[float] = Field(0, description="首重(kg)")
    first_price: Optional[float] = Field(0, description="首重价格")
    additional_kg: Optional[float] = Field(0, description="续重单位(kg)")
    additional_price: Optional[float] = Field(0, description="续重单价")
    fixed_fee: Optional[float] = Field(0, description="固定费")
    per_kg_price: Optional[float] = Field(0, description="每公斤价格")
    minimum_charge: Optional[float] = Field(None, description="最低收费")
    tier_config: Optional[list] = Field(None, description="阶梯配置")
    surcharge_fixed: Optional[float] = Field(0, description="附加费(固定)")
    fuel_surcharge_pct: Optional[float] = Field(0, description="燃油附加费百分比")
    rounding_increment: Optional[float] = Field(0.1, description="向上取整增量(kg)")
    remark: Optional[str] = None


class RuleUpdate(BaseModel):
    zone_id: Optional[int] = None
    rule_type: Optional[str] = None
    priority: Optional[int] = None
    min_weight_kg: Optional[float] = None
    max_weight_kg: Optional[float] = None
    first_kg: Optional[float] = None
    first_price: Optional[float] = None
    additional_kg: Optional[float] = None
    additional_price: Optional[float] = None
    fixed_fee: Optional[float] = None
    per_kg_price: Optional[float] = None
    minimum_charge: Optional[float] = None
    tier_config: Optional[list] = None
    surcharge_fixed: Optional[float] = None
    fuel_surcharge_pct: Optional[float] = None
    rounding_increment: Optional[float] = None
    remark: Optional[str] = None
    status: Optional[int] = None


class RuleVO(BaseModel):
    id: int
    channel_id: int
    zone_id: Optional[int] = None
    country_code: Optional[str] = None
    rule_type: str
    priority: int = 0
    min_weight_kg: Optional[float] = 0
    max_weight_kg: Optional[float] = None
    first_kg: Optional[float] = 0
    first_price: Optional[float] = 0
    additional_kg: Optional[float] = 0
    additional_price: Optional[float] = 0
    fixed_fee: Optional[float] = 0
    per_kg_price: Optional[float] = 0
    minimum_charge: Optional[float] = None
    tier_config: Optional[list] = None
    surcharge_fixed: Optional[float] = 0
    fuel_surcharge_pct: Optional[float] = 0
    rounding_increment: Optional[float] = 0.1
    remark: Optional[str] = None
    status: int = 1
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True


# ── Calculate ─────────────────────────────────────────────────────────────

class ManualPackageInput(BaseModel):
    length_cm: float = Field(..., gt=0, description="包装长(cm)")
    width_cm: float = Field(..., gt=0, description="包装宽(cm)")
    height_cm: float = Field(..., gt=0, description="包装高(cm)")
    weight_kg: float = Field(..., gt=0, description="包装重量(kg)")


class CalculateRequest(BaseModel):
    mode: Literal["sku", "manual"] = Field("sku", description="计算模式")
    sku_id: Optional[int] = Field(None, description="SKU ID")
    quantity: int = Field(1, ge=1, description="数量")
    destination_country: str = Field(..., min_length=2, max_length=10, description="目的地国家代码")
    postal_code: Optional[str] = Field(None, max_length=20, description="邮编")
    cargo_type: str = Field("normal", description="货品类型")
    package: Optional[ManualPackageInput] = Field(None, description="手动包裹信息")

    @model_validator(mode="after")
    def validate_mode_payload(self):
        if self.mode == "sku" and not self.sku_id:
            raise ValueError("SKU模式必须填写sku_id")
        if self.mode == "manual" and self.package is None:
            raise ValueError("手动模式必须填写package")
        return self


class PackageInfo(BaseModel):
    source: str = Field(..., description="数据来源: manual/sku/product")
    length_cm: float
    width_cm: float
    height_cm: float
    weight_kg: float


class CalculateResultItem(BaseModel):
    provider_id: int
    provider_name: str
    channel_id: int
    channel_name: str
    currency: str = "CNY"
    actual_weight_kg: float
    volumetric_weight_kg: float
    chargeable_weight_kg: float
    base_shipping_fee: float
    minimum_applied: bool = False
    surcharge_fee: float = 0
    fuel_surcharge_fee: float = 0
    total_shipping_fee: float
    estimated_delivery_min: Optional[int] = None
    estimated_delivery_max: Optional[int] = None
    calculation_detail: str = ""


class CalculateResponse(BaseModel):
    mode: str = "sku"
    sku_id: Optional[int] = None
    quantity: int
    destination_country: str
    package: PackageInfo
    results: list[CalculateResultItem] = []
