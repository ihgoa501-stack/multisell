"""通用API数据模型"""

from datetime import datetime
from typing import Optional, Generic, TypeVar, List, Any
from pydantic import BaseModel

T = TypeVar("T")


class Result(BaseModel, Generic[T]):
    """统一响应"""
    code: int = 200
    message: str = "ok"
    data: Optional[T] = None

    @staticmethod
    def ok(data: Any = None, message: str = "ok"):
        return Result(data=data, message=message)

    @staticmethod
    def error(message: str = "error", code: int = 500):
        return Result(code=code, message=message, data=None)

    @staticmethod
    def unauthorized(message: str = "未授权"):
        return Result(code=401, message=message)

    @staticmethod
    def not_found(message: str = "资源不存在"):
        return Result(code=404, message=message)

    @staticmethod
    def bad_request(message: str = "请求参数错误"):
        return Result(code=400, message=message)


class PageResult(BaseModel, Generic[T]):
    """分页响应"""
    code: int = 200
    message: str = "ok"
    records: List[T] = []
    total: int = 0
    page: int = 1
    page_size: int = 20

    @staticmethod
    def ok(records: List[T], total: int, page: int = 1, page_size: int = 20):
        return PageResult(records=records, total=total, page=page, page_size=page_size)


class PageParam(BaseModel):
    """分页参数"""
    page: int = 1
    page_size: int = 20


class ProductStatus:
    """商品状态"""
    DRAFT = 0
    ON_SHELF = 1
    OFF_SHELF = 2

    STATUS_MAP = {
        DRAFT: "草稿",
        ON_SHELF: "上架",
        OFF_SHELF: "下架",
    }


class IdSchema(BaseModel):
    id: int


class IdsSchema(BaseModel):
    ids: list[int]


class StatusSchema(BaseModel):
    status: int
