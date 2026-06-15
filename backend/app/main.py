"""凌镜 LingMirror — FastAPI 入口"""

import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from fastapi.exceptions import HTTPException
from fastapi.staticfiles import StaticFiles
from app.config import settings
import importlib, pkgutil

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# ========== 自动发现并注册所有模块路由 ==========
# 规则：每个 backend/app/<module>/ 只要 __init__.py 里导出 router 变量，
# 就会自动被注册到 /api 前缀下。新模块只需建目录 + router.py + __init__.py 即可，
# 不需要改 main.py 里的任何代码。
#
# 例外：app.common 导出了 upload_router（不从标准的 router 变量名），单独注册。
from app.common import upload_router as _upload_router

_registered_routers = set()


def _discover_routers(app_instance: FastAPI):
    """扫描 app/ 下所有子模块，自动注册有 router 的模块"""
    import app as _app_root

    for _importer, _modname, _ispkg in pkgutil.walk_packages(
        _app_root.__path__, prefix="app."
    ):
        if _ispkg:
            continue  # 只处理模块，不处理包
        _parent = _modname.rsplit(".", 1)[0]
        if _parent in _registered_routers:
            continue
        try:
            _pkg = importlib.import_module(_parent)
            if hasattr(_pkg, "router"):
                app_instance.include_router(_pkg.router, prefix="/api")
                _registered_routers.add(_parent)
        except Exception as _exc:
            logger.warning("模块 %s 路由注册失败: %s", _modname, _exc)

    # 单独注册 upload_router
    app_instance.include_router(_upload_router, prefix="/api")


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期"""
    # 生产环境强制开启鉴权
    if settings.is_production and not settings.AUTH_ENABLED:
        raise RuntimeError(
            "生产环境禁止关闭鉴权: 请设置 AUTH_ENABLED=True 或 DEBUG=True"
        )
    if settings.DEBUG:
        logger.warning(
            "DEBUG 模式启动 — 请确保已运行: cd backend && alembic upgrade head"
        )
    yield


app = FastAPI(
    title=settings.APP_NAME,
    version=settings.APP_VERSION,
    description=settings.APP_DESCRIPTION,
    lifespan=lifespan,
)

# 自动发现并注册路由
_discover_routers(app)

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
