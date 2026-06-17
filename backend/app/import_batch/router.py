from fastapi import APIRouter, Depends, Query, UploadFile, File
from fastapi.responses import StreamingResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.database import get_db
from app.common import Result, PageResult
from app.models import User
from app.import_batch.schemas import ImportBatchResponse, ImportCommitResponse
from app.import_batch.service import ImportBatchService, IMPORT_TYPES
from app.operation_log.service import OperationLogService

router = APIRouter(prefix="/import", tags=["导入管理"])


def _batch_to_vo(b) -> ImportBatchResponse:
    return ImportBatchResponse(
        id=b.id,
        type=b.type,
        file_name=b.file_name,
        status=b.status,
        total_rows=b.total_rows or 0,
        success_count=b.success_count or 0,
        error_count=b.error_count or 0,
        error_summary=b.error_summary,
        created_by=b.created_by,
        created_at=b.created_at,
    )


@router.get("/templates/{import_type}", summary="下载导入模板")
async def download_template(
    import_type: str,
    current_user: User = Depends(require_permission("import:view")),
):
    if import_type not in IMPORT_TYPES:
        return Result.bad_request(f"不支持的导入类型: {import_type}，可选: {', '.join(sorted(IMPORT_TYPES))}")

    output = ImportBatchService.generate_template(import_type)
    return StreamingResponse(
        output,
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": f"attachment; filename={import_type}_template.xlsx"},
    )


@router.post("/preview", summary="上传并预览导入文件")
async def preview_import(
    type: str = Query(..., description="导入类型: product/sku/price/inventory"),
    file: UploadFile = File(..., description="Excel文件"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("import:execute")),
):
    if type not in IMPORT_TYPES:
        return Result.bad_request(f"不支持的导入类型: {type}，可选: {', '.join(sorted(IMPORT_TYPES))}")

    if not file.filename or not file.filename.endswith((".xlsx", ".xls")):
        return Result.bad_request("仅支持 .xlsx 或 .xls 文件")

    file_bytes = await file.read()
    result = await ImportBatchService.preview(db, type, file_bytes, file.filename, current_user.username)

    await OperationLogService.log(
        db,
        module="import",
        action="preview",
        resource_id=str(result["batch_id"]),
        content=f"预览导入: type={type}, total={result['total_rows']}, errors={result['error_rows']}",
        operator=current_user.username,
    )

    return Result.ok(result)


@router.post("/commit/{batch_id}", summary="提交导入批次")
async def commit_import(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("import:execute")),
):
    try:
        result = await ImportBatchService.commit(db, batch_id, current_user.username)
    except ValueError as e:
        return Result.bad_request(str(e))

    await OperationLogService.log(
        db,
        module="import",
        action="commit",
        resource_id=str(batch_id),
        content=f"提交导入: type={result['type']}, success={result['success_count']}, errors={result['error_count']}",
        operator=current_user.username,
    )

    return Result.ok(ImportCommitResponse(**result))


@router.get("/batches", summary="导入批次列表")
async def list_batches(
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("import:view")),
):
    batches, total = await ImportBatchService.list_batches(db, page, page_size)
    records = [_batch_to_vo(b) for b in batches]
    return PageResult.ok(records, total, page, page_size)


@router.get("/batches/{batch_id}", summary="导入批次详情")
async def get_batch(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("import:view")),
):
    batch = await ImportBatchService.get_batch(db, batch_id)
    if not batch:
        return Result.not_found("批次不存在")
    return Result.ok(_batch_to_vo(batch))


@router.get("/batches/{batch_id}/errors", summary="下载错误报告")
async def download_errors(
    batch_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("import:view")),
):
    try:
        output = await ImportBatchService.generate_error_report(db, batch_id)
    except ValueError as e:
        return Result.bad_request(str(e))

    return StreamingResponse(
        output,
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": f"attachment; filename=import_batch_{batch_id}_errors.xlsx"},
    )
