"""平台结算导入 - Pydantic Schema"""

from typing import Optional

from pydantic import BaseModel, Field


class SettlementItemResponse(BaseModel):
    """结算行响应"""
    id: int
    batch_id: int
    row_number: int
    platform: Optional[str] = None
    store_name: Optional[str] = None
    platform_order_no: Optional[str] = None
    order_no: Optional[str] = None
    transaction_type: str
    currency: str = "CNY"
    amount: float = 0
    settled_at: Optional[str] = None
    description: Optional[str] = None
    match_status: str = "unmatched"
    matched_order_id: Optional[int] = None
    cost_layer: str = "actual"
    created_at: Optional[str] = None


class SettlementBatchResponse(BaseModel):
    """结算批次响应"""
    id: int
    platform_name: Optional[str] = None
    filename: str
    row_count: int = 0
    matched_count: int = 0
    unmatched_count: int = 0
    import_status: str = "imported"
    status: str = "imported"
    created_by: Optional[str] = None
    created_at: Optional[str] = None

    class Config:
        from_attributes = True


class SettlementImportResponse(BaseModel):
    """导入响应"""
    batch_id: int
    total_rows: int
    imported_rows: int
    error_rows: int
    errors: list[dict] = []
