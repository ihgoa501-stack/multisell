"""上架前经营决策 - 路由"""

from fastapi import APIRouter, Depends, File, HTTPException, Response, UploadFile
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common.schemas import Result
from app.database import get_db
from app.decision.excel_service import PreListingDecisionExcelService
from app.decision.schemas import (
    CompareDecisionRequest,
    PreListingDecisionBatchItemResult,
    PreListingDecisionBatchRequest,
    PreListingDecisionBatchResponse,
    PreListingDecisionBatchSummary,
    PreListingDecisionRequest,
    PreListingDecisionResponse,
)
from app.decision.service import PreListingDecisionService
from app.models import Platform, User

router = APIRouter(prefix="/decisions", tags=["上架决策"])


@router.post("/prelisting", summary="上架前经营决策")
async def prelisting_decision(
    data: PreListingDecisionRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("decision:calculate")),
):
    try:
        result = await PreListingDecisionService.calculate(db, data)
        return Result.ok(result)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))


@router.post("/prelisting/compare", summary="多平台经营决策对比")
async def compare_prelisting_decision(
    data: CompareDecisionRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("decision:calculate")),
):
    """对同一SKU在多个平台下分别计算经营决策，方便运营对比选择最佳平台"""
    results = []
    for platform_id in data.platform_ids:
        # 验证平台存在
        stmt = select(Platform).where(Platform.id == platform_id)
        platform = (await db.execute(stmt)).scalar_one_or_none()
        if not platform:
            continue
        req = PreListingDecisionRequest(
            sku_id=data.sku_id,
            destination_country=data.destination_country,
            target_sale_price=data.target_sale_price,
            platform_id=platform_id,
            platform_fee_pct=0,
            payment_fee_pct=data.payment_fee_pct,
            other_fee=data.other_fee,
            minimum_margin_pct=data.minimum_margin_pct,
            cargo_type=data.cargo_type,
        )
        try:
            r = await PreListingDecisionService.calculate(db, req)
            # Lookup platform name
            stmt = select(Platform.name).where(Platform.id == platform_id)
            platform_name = (await db.execute(stmt)).scalar_one_or_none() or f"平台{platform_id}"
            results.append({
                "platform_id": platform_id,
                "platform_name": platform_name,
                "product_cost": r.product_cost,
                "shipping_fee": r.shipping_fee,
                "platform_fee": r.platform_fee,
                "payment_fee": r.payment_fee,
                "total_cost": round(
                    r.product_cost + r.shipping_fee + r.platform_fee + r.payment_fee + data.other_fee, 2
                ),
                "profit_amount": r.profit_amount,
                "profit_margin": r.profit_margin,
                "recommendation": r.recommendation,
                "blocking_reasons": r.blocking_reasons,
                "warnings": r.warnings,
            })
        except (ValueError, HTTPException):
            continue
    return Result.ok({
        "sku_id": data.sku_id,
        "destination_country": data.destination_country,
        "target_sale_price": data.target_sale_price,
        "results": results,
    })


@router.post("/prelisting/batch", summary="批量上架前经营决策")
async def batch_prelisting_decision(
    data: PreListingDecisionBatchRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("decision:calculate")),
):
    items: list[PreListingDecisionBatchItemResult] = []
    profit_margins: list[float] = []
    approve_count = 0
    reject_count = 0
    needs_data_count = 0

    for index, item in enumerate(data.items):
        try:
            result = await PreListingDecisionService.calculate(db, item)
            if result.recommendation == "approve":
                approve_count += 1
            elif result.recommendation == "reject":
                reject_count += 1
            elif result.recommendation == "needs_data":
                needs_data_count += 1
            profit_margins.append(result.profit_margin)
            items.append(PreListingDecisionBatchItemResult(
                index=index,
                item_key=item.item_key,
                sku_id=item.sku_id,
                status="success",
                result=result,
            ))
        except Exception as e:
            items.append(PreListingDecisionBatchItemResult(
                index=index,
                item_key=item.item_key,
                sku_id=item.sku_id,
                status="error",
                error_message=str(e),
            ))

    success_count = sum(1 for item in items if item.status == "success")
    error_count = len(items) - success_count
    summary = PreListingDecisionBatchSummary(
        total_items=len(items),
        success_count=success_count,
        error_count=error_count,
        approve_count=approve_count,
        reject_count=reject_count,
        needs_data_count=needs_data_count,
        average_profit_margin=round(sum(profit_margins) / len(profit_margins), 2) if profit_margins else 0,
    )
    return Result.ok(PreListingDecisionBatchResponse(summary=summary, items=items))


@router.get("/prelisting/batch/template", summary="下载批量上架决策模板")
async def download_batch_prelisting_template(
    current_user: User = Depends(require_permission("decision:calculate")),
):
    content = PreListingDecisionExcelService.generate_template()
    return Response(
        content=content,
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": 'attachment; filename="prelisting_decision_template.xlsx"'},
    )


@router.post("/prelisting/batch/preview", summary="预览批量上架决策 Excel")
async def preview_batch_prelisting_excel(
    file: UploadFile = File(...),
    current_user: User = Depends(require_permission("decision:calculate")),
):
    try:
        content = await file.read()
        preview = PreListingDecisionExcelService.parse_preview(content)
        return Result.ok(preview)
    except ValueError as e:
        return Result.bad_request(str(e))
    except Exception as e:
        return Result.bad_request(f"Excel解析失败: {e}")


@router.post("/prelisting/batch/export", summary="导出批量上架决策结果")
async def export_batch_prelisting_results(
    data: PreListingDecisionBatchResponse,
    current_user: User = Depends(require_permission("decision:calculate")),
):
    content = PreListingDecisionExcelService.export_results(data)
    return Response(
        content=content,
        media_type="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers={"Content-Disposition": 'attachment; filename="prelisting_decision_results.xlsx"'},
    )
