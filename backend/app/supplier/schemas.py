"""供应商管理 - Pydantic Schema"""
from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, Field


class SupplierCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=200, description="供应商名称")
    contact_person: Optional[str] = Field(None, max_length=100, description="联系人")
    contact_phone: Optional[str] = Field(None, max_length=50, description="联系电话")
    email: Optional[str] = Field(None, max_length=200, description="邮箱")
    address: Optional[str] = Field(None, max_length=500, description="地址")
    remark: Optional[str] = Field(None, description="备注")


class SupplierUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=200)
    contact_person: Optional[str] = Field(None, max_length=100)
    contact_phone: Optional[str] = Field(None, max_length=50)
    email: Optional[str] = Field(None, max_length=200)
    address: Optional[str] = Field(None, max_length=500)
    status: Optional[int] = None
    remark: Optional[str] = None


class SupplierVO(BaseModel):
    id: int
    name: str
    contact_person: Optional[str] = None
    contact_phone: Optional[str] = None
    email: Optional[str] = None
    address: Optional[str] = None
    status: int = 1
    remark: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class ProductSupplierBind(BaseModel):
    """绑定商品-供应商"""
    product_id: int = Field(..., description="商品ID")
    supplier_id: int = Field(..., description="供应商ID")
    supply_price: Optional[float] = Field(None, description="供货价")
    min_order_qty: int = Field(1, description="最小起订量")


class ProductSupplierVO(BaseModel):
    """商品-供应商关联响应"""
    id: int
    product_id: int
    supplier_id: int
    supplier_name: Optional[str] = None
    supply_price: Optional[float] = None
    min_order_qty: int = 1
    created_at: Optional[datetime] = None

    class Config:
        from_attributes = True
