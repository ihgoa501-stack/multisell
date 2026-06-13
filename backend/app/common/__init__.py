"""通用模块"""

from app.common.schemas import (
    Result, PageResult, PageParam, ProductStatus, IdSchema, IdsSchema, StatusSchema,
)
from app.common.upload import router as upload_router
from app.common.utils import build_tree, save_upload_file, allowed_file, generate_filename, utc_now

__all__ = [
    "Result", "PageResult", "PageParam", "ProductStatus",
    "IdSchema", "IdsSchema", "StatusSchema",
    "upload_router", "build_tree", "save_upload_file",
    "allowed_file", "generate_filename", "utc_now",
]
