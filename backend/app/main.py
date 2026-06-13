"""MultiSell — FastAPI 入口"""

from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles
from app.config import settings
from app.database import init_db

# ========== 导入路由 ==========
from app.core import router as core_router
from app.category import router as category_router
from app.sku import router as sku_router
from app.price import router as price_router
from app.inventory import router as inventory_router
from app.supplier import router as supplier_router
from app.common import upload_router
from app.brand import router as brand_router
from app.dashboard import router as dashboard_router
from app.operation_log import router as operation_log_router
from app.platform import router as platform_router
from app.listing import router as listing_router
from app.search import router as search_router
from app.auth import router as auth_router
from fastapi import Request
from fastapi.responses import JSONResponse
from fastapi.exceptions import HTTPException


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期"""
    # 启动时：初始化数据库
    if settings.DEBUG:
        await init_db()
    yield
    # 关闭时：清理资源


app = FastAPI(
    title=settings.APP_NAME,
    version=settings.APP_VERSION,
    description=settings.APP_DESCRIPTION,
    lifespan=lifespan,
)

# ========== 挂载路由 ==========
app.include_router(core_router, prefix="/api")
app.include_router(category_router, prefix="/api")
app.include_router(sku_router, prefix="/api")
app.include_router(price_router, prefix="/api")
app.include_router(inventory_router, prefix="/api")
app.include_router(supplier_router, prefix="/api")
app.include_router(brand_router, prefix="/api")
app.include_router(dashboard_router, prefix="/api")
app.include_router(operation_log_router, prefix="/api")
app.include_router(platform_router, prefix="/api")
app.include_router(listing_router, prefix="/api")
app.include_router(search_router, prefix="/api")
app.include_router(auth_router, prefix="/api")
app.include_router(upload_router, prefix="/api")

# ========== 挂载静态文件（上传目录） ==========
import os
os.makedirs(settings.UPLOAD_DIR, exist_ok=True)
app.mount(settings.STATIC_URL, StaticFiles(directory=settings.UPLOAD_DIR), name="static")


@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    return JSONResponse(
        status_code=exc.status_code,
        content={"code": exc.status_code, "message": exc.detail, "data": None},
    )


@app.get("/api/health", tags=["系统"])
async def health():
    """健康检查"""
    return {"status": "ok", "service": settings.APP_NAME}
