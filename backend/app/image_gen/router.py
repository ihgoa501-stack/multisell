"""AI 生图 - API 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_db
from app.auth.dependencies import require_permission, get_current_user
from app.common.schemas import Result
from app.operation_log.service import OperationLogService
from app.models import User
from app.image_gen.schemas import (
    GenerateImageRequest,
    BatchGenerateRequest,
    SaveImageRequest,
    RemoveBgRequest,
    PromptTemplateCreate,
    PromptTemplateUpdate,
    InpaintRequest,
    OutpaintRequest,
    VideoGenRequest,
    SlideshowRequest,
)
from app.image_gen.service import ImageGenService

router = APIRouter(prefix="/image-gen", tags=["AI 生图"])


@router.post("/generate", summary="生成商品图片")
async def generate_image(
    req: GenerateImageRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    """为指定商品生成 AI 图片"""
    try:
        result = await ImageGenService.generate(
            db=db,
            user_id=current_user.id,
            product_id=req.product_id,
            prompt=req.prompt,
            style=req.style,
            negative_prompt=req.negative_prompt or "",
            size=req.size,
            count=req.count,
        )
        await OperationLogService.log(
            db,
            module="image_gen",
            action="generate",
            resource_id=str(req.product_id),
            operator_id=current_user.id,
            content=f"生图: product_id={req.product_id}, style={req.style}",
        )
        return Result.ok(result)
    except ValueError as e:
        return Result.error(str(e))
    except Exception as e:
        return Result.error(f"图片生成失败: {str(e)}")


@router.post("/save", summary="保存图片到商品")
async def save_image(
    req: SaveImageRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:save")),
):
    """将已生成的图片保存到商品的 main_image 或 images 列表"""
    try:
        result = await ImageGenService.save_to_product(
            db=db,
            product_id=req.product_id,
            image_url=req.image_url,
            set_as_main=req.set_as_main,
        )
        action_detail = "设为主图" if req.set_as_main else "加入图库"
        await OperationLogService.log(
            db,
            module="image_gen",
            action="save",
            resource_id=str(req.product_id),
            operator_id=current_user.id,
            content=f"保存图片到商品: {action_detail}",
        )
        return Result.ok(result)
    except ValueError as e:
        return Result.error(str(e))


@router.get("/history", summary="查询生成历史")
async def get_history(
    product_id: int = Query(None, description="商品ID，为空查全部"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:history")),
):
    """查询某个商品或全部的 AI 生图历史"""
    result = await ImageGenService.get_history(
        db=db,
        product_id=product_id,
        page=page,
        page_size=page_size,
    )
    return Result.ok(result)


@router.post("/batch-generate", summary="批量生成商品图片")
async def batch_generate(
    req: BatchGenerateRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    """为多个商品批量生成 AI 图片"""
    try:
        result = await ImageGenService.batch_generate(
            db=db,
            user_id=current_user.id,
            product_ids=req.product_ids,
            prompt=req.prompt,
            style=req.style,
            negative_prompt=req.negative_prompt or "",
            size=req.size,
            count=req.count,
        )
        await OperationLogService.log(
            db,
            module="image_gen",
            action="batch_generate",
            resource_id=req.batch_id or ",".join(str(x) for x in req.product_ids[:5]),
            operator_id=current_user.id,
            content=f"批量生图: {len(req.product_ids)} 个商品, style={req.style}",
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"批量生图失败: {str(e)}")


@router.post("/remove-bg", summary="图片去背景")
async def remove_background(
    req: RemoveBgRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:edit")),
):
    """对指定图片做去背景处理"""
    try:
        result = await ImageGenService.remove_background(
            db=db,
            image_url=req.image_url,
        )
        if result:
            return Result.ok({"url": result})
        return Result.error("去背景处理失败")
    except Exception as e:
        return Result.error(f"去背景失败: {str(e)}")


# ====== Prompt 模板 CRUD ======


@router.get("/templates", summary="查询 Prompt 模板列表")
async def list_templates(
    platform_code: str = Query(None, description="按平台筛选"),
    page: int = Query(1, ge=1),
    page_size: int = Query(50, ge=1, le=200),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:history")),
):
    """查询可用的 Prompt 模板（共享的 + 自己创建的）"""
    result = await ImageGenService.list_templates(
        db=db,
        user_id=current_user.id,
        platform_code=platform_code,
        page=page,
        page_size=page_size,
    )
    return Result.ok(result)


@router.post("/templates", summary="创建 Prompt 模板")
async def create_template(
    req: PromptTemplateCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    """创建新的 Prompt 模板"""
    try:
        result = await ImageGenService.create_template(
            db=db,
            user_id=current_user.id,
            name=req.name,
            prompt=req.prompt,
            description=req.description or "",
            negative_prompt=req.negative_prompt or "",
            style=req.style,
            size=req.size,
            platform_code=req.platform_code,
            is_shared=req.is_shared,
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"创建模板失败: {str(e)}")


@router.put("/templates/{template_id}", summary="更新 Prompt 模板")
async def update_template(
    template_id: int,
    req: PromptTemplateUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    """更新 Prompt 模板（仅创建者可修改）"""
    try:
        updates = {k: v for k, v in req.model_dump().items() if v is not None}
        result = await ImageGenService.update_template(
            db=db,
            template_id=template_id,
            user_id=current_user.id,
            updates=updates,
        )
        return Result.ok(result)
    except ValueError as e:
        return Result.error(str(e))
    except PermissionError as e:
        return Result(code=403, message=str(e))
    except Exception as e:
        return Result.error(f"更新模板失败: {str(e)}")


@router.delete("/templates/{template_id}", summary="删除 Prompt 模板")
async def delete_template(
    template_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    """删除 Prompt 模板（仅创建者可删除）"""
    try:
        await ImageGenService.delete_template(
            db=db,
            template_id=template_id,
            user_id=current_user.id,
        )
        return Result.ok(message="模板已删除")
    except ValueError as e:
        return Result.error(str(e))
    except PermissionError as e:
        return Result(code=403, message=str(e))
    except Exception as e:
        return Result.error(f"删除模板失败: {str(e)}")


@router.post("/inpaint", summary="局部重绘")
async def inpaint_image(
    req: InpaintRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await ImageGenService.inpaint(
            db=db,
            user_id=current_user.id,
            image_url=req.image_url,
            mask_base64=req.mask_base64,
            prompt=req.prompt,
            negative_prompt=req.negative_prompt or "",
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"局部重绘失败: {str(e)}")


@router.post("/outpaint", summary="扩图")
async def outpaint_image(
    req: OutpaintRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await ImageGenService.outpaint(
            db=db,
            user_id=current_user.id,
            image_url=req.image_url,
            direction=req.direction,
            prompt=req.prompt,
            expand_ratio=req.expand_ratio,
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"扩图失败: {str(e)}")


@router.post("/video", summary="AI 生成视频")
async def generate_video(
    req: VideoGenRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await ImageGenService.generate_video(
            db=db,
            user_id=current_user.id,
            prompt=req.prompt,
            image_url=req.image_url,
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"视频生成失败: {str(e)}")


@router.get("/video/status/{job_id}", summary="视频生成进度")
async def video_status(
    job_id: str,
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:history")),
):
    try:
        result = await ImageGenService.get_video_status(job_id=job_id)
        return Result.ok(result)
    except Exception as e:
        return Result.error(str(e))


@router.post("/video/slideshow", summary="图片合成视频")
async def create_slideshow(
    req: SlideshowRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await ImageGenService.create_slideshow(
            db=db,
            user_id=current_user.id,
            image_urls=req.image_urls,
            duration_per_frame=req.duration_per_frame,
            transition=req.transition,
            resolution=req.resolution,
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"视频合成失败: {str(e)}")
