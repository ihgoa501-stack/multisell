"""通用工具类"""

import uuid
from datetime import datetime, timezone
from pathlib import Path
from app.config import settings


def utc_now() -> datetime:
    """获取当前UTC时间"""
    return datetime.now(timezone.utc)


def generate_filename(original: str) -> str:
    """生成唯一文件名"""
    ext = original.rsplit(".", 1)[-1].lower() if "." in original else "bin"
    return f"{uuid.uuid4().hex}.{ext}"


def allowed_file(filename: str) -> bool:
    """检查文件扩展名是否允许"""
    ext = filename.rsplit(".", 1)[-1].lower() if "." in filename else ""
    return ext in settings.ALLOWED_EXTENSIONS


async def save_upload_file(file_bytes: bytes, filename: str) -> str:
    """保存上传文件，返回相对路径"""
    safe_name = generate_filename(filename)
    upload_path = Path(settings.UPLOAD_DIR)
    upload_path.mkdir(parents=True, exist_ok=True)
    file_path = upload_path / safe_name
    with open(file_path, "wb") as f:
        f.write(file_bytes)
    return f"{settings.STATIC_URL}/{safe_name}"


def build_tree(items: list[dict], parent_id: int = 0) -> list[dict]:
    """构建树形结构（通用递归）"""
    tree = []
    for item in items:
        if item.get("parent_id", 0) == parent_id:
            children = build_tree(items, item["id"])
            if children:
                item["children"] = children
            tree.append(item)
    return tree
