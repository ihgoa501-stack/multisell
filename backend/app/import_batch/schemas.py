from datetime import datetime
from typing import Optional
from pydantic import BaseModel


class ImportPreviewResponse(BaseModel):
    batch_id: int
    type: str
    total_rows: int
    valid_rows: int
    error_rows: int
    errors: list[dict]


class ImportBatchResponse(BaseModel):
    id: int
    type: str
    file_name: Optional[str] = None
    status: str
    total_rows: int = 0
    success_count: int = 0
    error_count: int = 0
    error_summary: Optional[str] = None
    created_by: Optional[str] = None
    created_at: Optional[datetime] = None

    class Config:
        from_attributes = True


class ImportCommitResponse(BaseModel):
    batch_id: int
    type: str
    success_count: int
    error_count: int
    imported: int
    errors: list[dict]
