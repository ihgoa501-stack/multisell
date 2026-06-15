"""上架前经营决策 - 路由"""

import io
from datetime import datetime, timezone

from fastapi import APIRouter, Depends, File, HTTPException, UploadFile
from fastapi.responses import StreamingResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common.schemas import Result
from app.database import get_db
from app.decision.excel_service import PreListingDecisionExcelService
from app.decision.schemas import (
    PreListingDecisionBatchRequest,
    PreListingDecisionBatchResponse,
    PreListingDecisionExcelPreviewResponse,
    PreListingDecisionRequest,
    PreListingDecisionResponse,
)
from app.decision.service import PreListingDecisionService
from app.models import User

router = APIRouter(prefix="/decisions", tags=["上架决策"])


@router.post("/prelisting", summary="上架前经营决策")
async def prelisting_decision(
    data: PreListingDecisionRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("decision:calculate")),
) -> Result[PreListingDecisionResponse]:
    try:
        result = await PreListingDecisionService.calculate(db, data)
        return Result.ok(result)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))


@router.post("/prelisting/batch", summary="批量上架前经营决策")
async def prelisting_decision_batch(
    data: PreListingDecisionBatchRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("decision:calculate")),
) -> Result[PreListingDecisionBatchResponse]:
    result = await PreListingDecisionService.calculate_batch(db, data)
    return Result.ok(result)


@router.get("/prelisting/batch/template", summary="下载批量上架决策 Excel 模板")
async def download_prelisting_batch_template(
    current_user: User = Depends(require_permission("decision:calculate")),
):
    output = PreListingDecisionExcelService.generate_template()
    return StreamingResponse(
        io.BytesIO(output),
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": "attachment; filename=prelisting_decision_template.xlsx"},
    )


@router.post("/prelisting/batch/preview", summary="上传预览批量上架决策 Excel")
async def preview_prelisting_batch_excel(
    file: UploadFile = File(...),
    current_user: User = Depends(require_permission("decision:calculate")),
) -> Result[PreListingDecisionExcelPreviewResponse]:
    if not file.filename or not file.filename.lower().endswith(".xlsx"):
        raise HTTPException(status_code=400, detail="仅支持 .xlsx 文件")
    content = await file.read()
    try:
        preview = PreListingDecisionExcelService.parse_preview(content)
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    return Result.ok(preview)


@router.post("/prelisting/batch/export", summary="导出批量上架决策结果")
async def export_prelisting_batch_results(
    data: PreListingDecisionBatchResponse,
    current_user: User = Depends(require_permission("decision:calculate")),
):
    output = PreListingDecisionExcelService.export_results(data)
    filename = f"prelisting_decision_results_{datetime.now(timezone.utc).strftime('%Y%m%d%H%M%S')}.xlsx"
    return StreamingResponse(
        io.BytesIO(output),
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": f"attachment; filename={filename}"},
    )
