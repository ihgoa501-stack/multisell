"""仪表盘 - 路由"""

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result
from app.dashboard.service import DashboardService

router = APIRouter(tags=["仪表盘"])


@router.get("/dashboard/stats", summary="仪表盘统计数据")
async def get_dashboard_stats(db: AsyncSession = Depends(get_db)):
    stats = await DashboardService.get_stats(db)
    return Result.ok(stats)


@router.get("/reports/product-stats", summary="商品统计")
async def get_product_stats(db: AsyncSession = Depends(get_db)):
    stats = await DashboardService.get_product_stats(db)
    return Result.ok(stats)


@router.get("/reports/platform-stats", summary="平台发布统计")
async def get_platform_stats(db: AsyncSession = Depends(get_db)):
    stats = await DashboardService.get_platform_stats(db)
    return Result.ok(stats)
