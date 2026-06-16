"""费用分摊 - Pydantic Schema"""

from typing import Optional

from pydantic import BaseModel, Field


class AllocationImportResponse(BaseModel):
    batch_id: int
    total_rows: int
    imported_rows: int
    error_rows: int
    errors: list[dict] = []


class AllocationItemResponse(BaseModel):
    id: int
    batch_id: int
    row_number: int
    sku_id: Optional[int] = None
    sku_code: Optional[str] = None
    order_id: Optional[int] = None
    quantity: int = 0
    weight_kg: Optional[float] = None
    volume_m3: Optional[float] = None
    item_value: Optional[float] = None
    allocation_factor: Optional[float] = None
    allocated_amount: float = 0
    cost_layer: str = "allocated"
    posted_to_ledger: bool = False
    created_at: Optional[str] = None
