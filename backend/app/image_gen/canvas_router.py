"""AI 生图 - 画布 API 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.auth.dependencies import require_permission, get_current_user
from app.common.schemas import Result
from app.image_gen.canvas_schemas import CanvasSaveRequest
from app.image_gen.canvas_service import CanvasService
from app.models import User

router = APIRouter(prefix="/canvas", tags=["AI 生图 - 画布"])


@router.post("", summary="保存画布")
async def save_canvas(
    req: CanvasSaveRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await CanvasService.save(
            db=db, user_id=current_user.id,
            product_id=req.product_id, name=req.name,
            layers=[l.model_dump() for l in req.layers],
        )
        return Result.ok(result)
    except ValueError as e:
        return Result.error(str(e))


@router.get("/{canvas_id}", summary="加载画布")
async def load_canvas(
    canvas_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:history")),
):
    result = await CanvasService.load(db=db, canvas_id=canvas_id)
    if not result:
        return Result.not_found("画布不存在")
    return Result.ok(result)


@router.get("", summary="商品画布列表")
async def list_canvases(
    product_id: int = Query(..., description="商品ID"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:history")),
):
    result = await CanvasService.list_by_product(
        db=db, product_id=product_id, page=page, page_size=page_size
    )
    return Result.ok(result)


@router.delete("/{canvas_id}", summary="删除画布")
async def delete_canvas(
    canvas_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        await CanvasService.delete(db=db, canvas_id=canvas_id, user_id=current_user.id)
        return Result.ok(message="画布已删除")
    except ValueError as e:
        return Result.error(str(e))
    except PermissionError as e:
        return Result(code=403, message=str(e))
