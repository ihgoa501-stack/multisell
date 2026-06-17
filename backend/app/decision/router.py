"""上架前经营决策 - 路由"""

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common.schemas import Result
from app.database import get_db
from app.decision.schemas import (
    CompareDecisionRequest,
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
