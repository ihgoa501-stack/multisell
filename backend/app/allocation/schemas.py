"""库存分配 - Pydantic 模型"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, ConfigDict, Field


class WarehouseCreate(BaseModel):
    """创建仓库"""
    name: str = Field(..., description="仓库名称")
    code: Optional[str] = None
    address: Optional[str] = None
    contact: Optional[str] = None
    phone: Optional[str] = None
    is_default: bool = False


class WarehouseUpdate(BaseModel):
    """更新仓库"""
    name: Optional[str] = None
    code: Optional[str] = None
    address: Optional[str] = None
    contact: Optional[str] = None
    phone: Optional[str] = None
    is_default: Optional[bool] = None
    status: Optional[int] = None


class WarehouseVO(BaseModel):
    """仓库视图"""
    id: int
    name: str
    code: Optional[str] = None
    address: Optional[str] = None
    contact: Optional[str] = None
    phone: Optional[str] = None
    is_default: bool
    status: int
    created_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class AllocationRuleCreate(BaseModel):
    """创建分配规则"""
    name: str = Field(..., description="规则名称")
    priority: int = 0
    rule_type: str = Field(..., description="percentage/fixed/priority")
    warehouse_id: int = Field(..., description="仓库ID")
    allocation_pct: float = 100
    allocation_qty: int = 0


class AllocationRuleVO(BaseModel):
    """分配规则视图"""
    id: int
    name: str
    priority: int
    rule_type: str
    warehouse_id: int
    warehouse_name: Optional[str] = None
    allocation_pct: float
    allocation_qty: int
    status: int

    model_config = ConfigDict(from_attributes=True)


class InventoryWarehouseVO(BaseModel):
    """仓库库存视图"""
    id: int
    sku_id: int
    warehouse_id: int
    warehouse_name: Optional[str] = None
    quantity: int
    locked_quantity: int
    safety_stock: int
    available_qty: int = 0

    model_config = ConfigDict(from_attributes=True)


class InventoryAllocateRequest(BaseModel):
    """库存分配请求"""
    sku_id: int = Field(..., description="SKU ID")
    warehouse_id: int = Field(..., description="目标仓库ID")
    quantity: int = Field(..., gt=0, description="分配数量")


class AllocationResult(BaseModel):
    """分配结果"""
    sku_id: int
    sku_code: Optional[str] = None
    total_available: int
    allocated: list[dict] = []
    warnings: list[str] = []
