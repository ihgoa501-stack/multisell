"""RBAC - Pydantic Schema"""
from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class RoleCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=100, description="角色名称")
    code: str = Field(..., min_length=1, max_length=100, description="角色代码")
    description: Optional[str] = Field(None, max_length=500, description="角色描述")


class RoleUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    code: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = Field(None, max_length=500)
    status: Optional[int] = Field(None, description="状态: 0-禁用, 1-启用")


class RoleVO(BaseModel):
    id: int
    name: str
    code: str
    description: Optional[str] = None
    status: int = 1
    permission_ids: list[int] = []
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class PermissionCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=100, description="权限名称")
    code: str = Field(..., min_length=1, max_length=100, description="权限代码")
    description: Optional[str] = Field(None, max_length=500, description="权限描述")
    module: Optional[str] = Field(None, max_length=100, description="所属模块")


class PermissionUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    code: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = Field(None, max_length=500)
    module: Optional[str] = Field(None, max_length=100)


class PermissionVO(BaseModel):
    id: int
    name: str
    code: str
    description: Optional[str] = None
    module: Optional[str] = None
    created_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class AssignRolesData(BaseModel):
    role_ids: list[int] = Field(..., description="角色ID列表")
