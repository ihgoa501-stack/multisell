"""商品管理 - 路由"""

from fastapi import APIRouter, Depends, Query, UploadFile, File
from fastapi.responses import StreamingResponse
from datetime import datetime
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import update, delete
from app.database import get_db
from app.common import Result, PageResult
from app.core.schemas import ProductCreate, ProductUpdate, ProductQuery
from app.core.service import ProductService, product_to_vo
from app.core.excel_service import ExcelService
from app.core.ai_service import AiEnhanceService
from app.core.detail_service import ProductDetailService
from app.auth import require_permission
from app.models import Product, User
from app.operation_log.service import OperationLogService

router = APIRouter(tags=["商品管理"])


def _operator(current_user: User) -> str:
    return current_user.username if current_user else "system"


async def _log_product_operation(
    db: AsyncSession,
    action: str,
    current_user: User,
    resource_id: str = None,
    content: str = None,
):
    await OperationLogService.log(
        db,
        module="product",
        action=action,
        resource_id=resource_id,
        content=content,
        operator=_operator(current_user),
    )


# ========== 批量操作 ==========


@router.post("/products/batch/status", summary="批量修改商品状态")
async def batch_update_status(
    data: dict,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("product:update")),
):
    ids = data.get("ids", [])
    status = data.get("status", 0)
    if not ids:
        return Result.bad_request("请选择商品")

    stmt = update(Product).where(Product.id.in_(ids)).values(status=status)
    await db.execute(stmt)
    await db.flush()
    await _log_product_operation(
        db,
        "batch_update_status",
        current_user,
        content=f"批量修改商品状态: ids={ids}, status={status}",
    )
    return Result.ok({"affected": len(ids)})


@router.post("/products/batch/delete", summary="批量删除商品")
async def batch_delete_products(
    data: dict,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("product:delete")),
):
    ids = data.get("ids", [])
    if not ids:
        return Result.bad_request("请选择商品")

    stmt = delete(Product).where(Product.id.in_(ids))
    result = await db.execute(stmt)
    await db.flush()
    await _log_product_operation(
        db,
        "batch_delete",
        current_user,
        content=f"批量删除商品: ids={ids}",
    )
    return Result.ok({"affected": result.rowcount})


# ========== Excel 导出导入 ==========
# NOTE: 这些路由必须在 /products/{product_id} 之前注册，
# 否则 "export" 和 "export-template" 会被当作 product_id 解析


@router.get("/products/export", summary="导出商品Excel")
async def export_products(
    name: str = Query(None, description="按名称筛选"),
    category_id: int = Query(None, description="按分类筛选"),
    status: int = Query(None, description="按状态筛选"),
    brand_id: int = Query(None, description="按品牌筛选"),
    cargo_type: str = Query(None, description="按货品类型筛选"),
    logistics_status: str = Query(None, description="按物流完整状态筛选"),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("product:export")),
):
    output = await ExcelService.export_products(db, name, category_id, status, brand_id, cargo_type, logistics_status)
    await _log_product_operation(
        db,
        "export",
        current_user,
        content=f"导出商品Excel: name={name}, category_id={category_id}, status={status}",
    )
    return StreamingResponse(
        output,
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": f"attachment; filename=products_{datetime.utcnow().strftime('%Y%m%d')}.xlsx"},
    )


@router.get("/products/export-template", summary="下载导入模板")
async def export_template(
    current_user: User = Depends(require_permission("product:export")),
):
    output = ExcelService.export_template()
    return StreamingResponse(
        output,
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": "attachment; filename=product_import_template.xlsx"},
    )


@router.post("/products/import", summary="从Excel导入商品")
async def import_products(
    file: UploadFile = File(...),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("product:import")),
):
    if not file.filename or not file.filename.endswith(('.xlsx', '.xls')):
        return Result.bad_request("请上传 .xlsx 或 .xls 文件")
    content = await file.read()
    result = await ExcelService.import_products(db, content)
    await _log_product_operation(
        db,
        "import",
        current_user,
        content=f"导入商品Excel: filename={file.filename}, imported={result['imported']}, total={result['total']}",
    )
    if result["errors"]:
        return Result.ok(result)
    return Result.ok(result)


# ========== CRUD ==========


@router.post("/products", summary="创建商品")
async def create_product(
    data: ProductCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("product:create")),
):
    product = await ProductService.create(db, data)
    await _log_product_operation(
        db,
        "create",
        current_user,
        resource_id=str(product.id),
        content=f"创建商品: {product.name}",
    )
    return Result.ok(product_to_vo(product))


@router.put("/products/{product_id}", summary="更新商品")
async def update_product(
    product_id: int,
    data: ProductUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("product:update")),
):
    product = await ProductService.update(db, product_id, data)
    if not product:
        return Result.not_found("商品不存在")
    await _log_product_operation(
        db,
        "update",
        current_user,
        resource_id=str(product.id),
        content=f"更新商品: {product.name}",
    )
    return Result.ok(product_to_vo(product))


@router.get("/products/{product_id}", summary="商品详情")
async def get_product(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("product:view")),
):
    product = await ProductService.get_by_id(db, product_id)
    if not product:
        return Result.not_found("商品不存在")
    return Result.ok(product_to_vo(product))


@router.get("/products", summary="商品列表")
async def list_products(
    name: str = Query(None, description="商品名称"),
    category_id: int = Query(None, description="分类ID"),
    status: int = Query(None, description="状态"),
    brand_id: int = Query(None, description="品牌ID"),
    cargo_type: str = Query(None, description="货品类型: normal/battery/liquid/sensitive"),
    logistics_status: str = Query(None, description="物流完整状态: complete/incomplete"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("product:view")),
):
    query = ProductQuery(name=name, category_id=category_id, status=status, brand_id=brand_id, cargo_type=cargo_type, logistics_status=logistics_status, page=page, page_size=page_size)
    products, total = await ProductService.list_products(db, query)
    items = [product_to_vo(p) for p in products]
    return PageResult.ok(items, total, page, page_size)


@router.delete("/products/{product_id}", summary="删除商品")
async def delete_product(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("product:delete")),
):
    try:
        ok = await ProductService.delete(db, product_id)
    except ValueError as e:
        return Result.bad_request(str(e))
    if not ok:
        return Result.not_found("商品不存在")
    await _log_product_operation(
        db,
        "delete",
        current_user,
        resource_id=str(product_id),
        content=f"删除商品: {product_id}",
    )
    return Result.ok(message="删除成功")


# ========== 复制 ==========


@router.post("/products/{product_id}/duplicate", summary="复制商品")
async def duplicate_product(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("product:create")),
):
    product = await ProductService.get_by_id(db, product_id)
    if not product:
        return Result.not_found("商品不存在")

    new_product = Product(
        name=f"{product.name} (副本)",
        subtitle=product.subtitle,
        description=product.description,
        brand_id=product.brand_id,
        category_id=product.category_id,
        unit=product.unit,
        main_image=product.main_image,
        images=product.images,
        product_length_cm=product.product_length_cm,
        product_width_cm=product.product_width_cm,
        product_height_cm=product.product_height_cm,
        product_weight_kg=product.product_weight_kg,
        package_length_cm=product.package_length_cm,
        package_width_cm=product.package_width_cm,
        package_height_cm=product.package_height_cm,
        package_weight_kg=product.package_weight_kg,
        cargo_type=product.cargo_type,
        status=0,
    )
    db.add(new_product)
    await db.flush()
    await db.refresh(new_product)
    await _log_product_operation(
        db,
        "duplicate",
        current_user,
        resource_id=str(new_product.id),
        content=f"复制商品: {product_id} -> {new_product.id}",
    )

    return Result.ok(product_to_vo(new_product))


# ========== 聚合详情 ==========


@router.get("/products/{product_id}/detail", summary="商品聚合详情")
async def get_product_detail(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    _current_user: User = Depends(require_permission("product:view")),
):
    try:
        data = await ProductDetailService.get_detail(db, product_id)
        return Result.ok(data)
    except ValueError as e:
        return Result.not_found(str(e))


# ========== AI 优化 ==========


@router.post("/products/{product_id}/ai-enhance", summary="AI优化商品信息")
async def ai_enhance_product(
    product_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("product:ai")),
):
    try:
        result = await AiEnhanceService.enhance_product(db, product_id)
        await _log_product_operation(
            db,
            "ai_enhance",
            current_user,
            resource_id=str(product_id),
            content=f"AI优化商品: {product_id}",
        )
        return Result.ok(result)
    except ValueError as e:
        return Result.not_found(str(e))
