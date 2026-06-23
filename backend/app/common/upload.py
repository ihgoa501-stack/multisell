"""文件上传路由"""

from fastapi import APIRouter, UploadFile, File
from app.common.schemas import Result
from app.common.utils import allowed_file, save_upload_file
from app.config import settings

router = APIRouter(tags=["文件上传"])


@router.post("/upload", summary="上传文件")
async def upload_file(file: UploadFile = File(...)):
    """上传图片/文件，返回访问URL"""
    if not file.filename:
        return Result.bad_request("文件名为空")

    if not allowed_file(file.filename):
        return Result.bad_request(
            f"不支持的文件格式，允许: {', '.join(settings.ALLOWED_EXTENSIONS)}"
        )

    # 检查文件大小
    content = await file.read()
    if len(content) > settings.MAX_UPLOAD_SIZE:
        return Result.bad_request(
            f"文件过大，最大 {settings.MAX_UPLOAD_SIZE // 1024 // 1024}MB"
        )

    url = await save_upload_file(content, file.filename)
    return Result.ok({"url": url})
