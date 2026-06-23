"""库存管理 - Pydantic Schema"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, ConfigDict, Field


class InventoryUpdate(BaseModel):
    """更新库存"""

    warehouse: Optional[str] = Field(None, description="仓库")
    location: Optional[str] = Field(None, description="货位")
    quantity: int = Field(..., description="库存数量（最终值）")
    safety_stock: Optional[int] = Field(None, description="安全库存")
    remark: Optional[str] = Field(None, description="备注")


class InventoryCheck(BaseModel):
    """库存预占/释放"""

    sku_id: int = Field(..., description="SKU ID")
    quantity: int = Field(..., description="数量（正数为预占，负数为释放）")
    order_no: Optional[str] = Field(None, description="订单号")


class InventoryVO(BaseModel):
    """库存响应"""

    id: int = 0
    sku_id: int
    warehouse: str = "默认仓库"
    location: Optional[str] = None
    quantity: int = 0
    locked_quantity: int = 0
    available_quantity: int = 0
    safety_stock: int = 0
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)


class InventoryLogVO(BaseModel):
    """库存变动记录响应"""

    id: int
    sku_id: int
    change_type: str
    change_qty: int
    before_qty: Optional[int] = None
    after_qty: Optional[int] = None
    remark: Optional[str] = None
    operator: Optional[str] = None
    created_at: Optional[datetime] = None

    model_config = ConfigDict(from_attributes=True)
