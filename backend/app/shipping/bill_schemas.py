"""运费账单导入与对账 - Pydantic Schema"""

from datetime import datetime
from typing import Optional

from pydantic import BaseModel, ConfigDict, Field


from app.finance.cost_layers import COST_LAYER_ACTUAL


# ── Bill Batch ──────────────────────────────────────────────────────────────

class BillBatchResponse(BaseModel):
    """账单批次响应"""
    id: int
    source_filename: str
    row_count: int = 0
    matched_count: int = 0
    mismatch_count: int = 0
    unmatched_count: int = 0
    currency: str = "CNY"
    status: str = "imported"
    created_by: Optional[str] = None
    created_at: Optional[str] = None

    model_config = ConfigDict(from_attributes=True)


class BillImportResponse(BaseModel):
    """导入响应"""
    batch_id: Optional[int] = None
    total_rows: int
    imported_rows: int
    error_rows: int
    errors: list[dict] = []


# ── Bill Item ───────────────────────────────────────────────────────────────

class BillItemResponse(BaseModel):
    """账单行响应"""
    id: int
    batch_id: int
    row_number: int
    reconciliation_status: str = "unmatched_bill"
    actual_shipping_cost_layer: str = COST_LAYER_ACTUAL
    tracking_number: Optional[str] = None
    order_no: Optional[str] = None
    provider_name: Optional[str] = None
    channel_name: Optional[str] = None
    destination_country: Optional[str] = None
    billed_weight_kg: Optional[float] = None
    currency: str = "CNY"
    actual_shipping_fee: Optional[float] = None
    surcharge_fee: Optional[float] = None
    total_actual_fee: Optional[float] = None
    billed_at: Optional[str] = None
    matched_order_id: Optional[int] = None
    matched_snapshot_id: Optional[int] = None
    snapshot_shipping_fee: Optional[float] = None
    variance_amount: Optional[float] = None
    note: Optional[str] = None
    resolved_by: Optional[str] = None
    resolved_at: Optional[str] = None
    created_at: Optional[str] = None

    model_config = ConfigDict(from_attributes=True)


class BillItemResolveRequest(BaseModel):
    """手动解决请求"""
    note: str = Field(..., min_length=1, max_length=1000, description="解决说明")


# ── Reconciliation ─────────────────────────────────────────────────────────

class BillReconcileResponse(BaseModel):
    """对账结果"""
    batch_id: int
    total_items: int
    matched_count: int
    mismatch_count: int
    unmatched_count: int


class BillReconciliationSummaryResponse(BaseModel):
    """对账汇总"""
    total_batches: int
    reconciled_batches: int
    total_items: int
    status_counts: dict[str, int]
