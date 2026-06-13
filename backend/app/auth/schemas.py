"""认证 - Pydantic Schema"""
from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field


class UserRegister(BaseModel):
    username: str = Field(..., min_length=3, max_length=100, description="用户名")
    password: str = Field(..., min_length=6, max_length=200, description="密码")
    display_name: Optional[str] = Field(None, max_length=200, description="显示名称")
    email: Optional[str] = Field(None, max_length=200, description="邮箱")


class UserLogin(BaseModel):
    username: str = Field(..., description="用户名")
    password: str = Field(..., description="密码")


class UserVO(BaseModel):
    id: int
    username: str
    display_name: Optional[str] = None
    role: str = "user"
    email: Optional[str] = None
    status: int = 1
    created_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class TokenVO(BaseModel):
    access_token: str
    token_type: str = "bearer"
    user: UserVO
