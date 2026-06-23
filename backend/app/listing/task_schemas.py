"""上架任务队列 - Pydantic Schema"""

from typing import Optional

from pydantic import BaseModel, Field


class ListingTaskDecisionResult(BaseModel):
    """决策结果快照（字段与 PreListingDecisionResponse 对齐）"""

    sku_id: int
    destination_country: str
    target_sale_price: float
    product_cost: float = 0
    shipping_fee: float = 0
    platform_fee: float = 0
    payment_fee: float = 0
    fixed_fee: float = 0
    advertising_fee: float = 0
    other_fee: float = 0
    profit_amount: float = 0
    profit_margin: float = 0
    recommendation: str
    blocking_reasons: list[str] = []
    warnings: list[str] = []
    platform_fee_source: str = "manual"


class ListingTaskCreateFromDecisionItem(BaseModel):
    """从决策创建单行上架任务"""

    item_key: Optional[str] = Field(None, max_length=100)
    sku_id: int
    platform_id: int
    decision_result: ListingTaskDecisionResult


class ListingTaskCreateFromDecisionRequest(BaseModel):
    """从决策创建上架任务请求"""

    items: list[ListingTaskCreateFromDecisionItem] = Field(
        ...,
        min_length=1,
        max_length=100,
    )


class ListingTaskCreateResult(BaseModel):
    """创建结果单行"""

    id: int
    product_id: int
    platform_id: int
    status: str
    missing_requirements: list[str]
    source_item_key: Optional[str] = None


class ListingTaskCreateFromDecisionResponse(BaseModel):
    """从决策创建上架任务响应"""

    created_count: int
    reused_count: int
    skipped_count: int
    tasks: list[ListingTaskCreateResult]
    skipped: list[dict] = []


class ListingTaskResponse(BaseModel):
    """上架任务列表响应项"""

    id: int
    product_id: int
    product_name: str
    platform_id: int
    platform_name: str
    sku_id: Optional[int] = None
    product_listing_id: Optional[int] = None
    source_type: str = "decision"
    source_item_key: Optional[str] = None
    status: str
    missing_requirements: list[str] = []
    target_sale_price: Optional[float] = None
    target_profit_margin: Optional[float] = None
    destination_country: Optional[str] = None
    last_error: Optional[str] = None
    created_by: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None
