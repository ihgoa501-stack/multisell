"""AI 生图 - 数据模型"""

from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, Field


class GenerateImageRequest(BaseModel):
    """生成图片请求"""

    product_id: int = Field(..., description="商品ID")
    prompt: str = Field(..., min_length=1, max_length=2000, description="正向提示词")
    negative_prompt: Optional[str] = Field("", description="反向提示词")
    style: str = Field(
        "product_white", description="风格: product_white/scene/model/3d_render"
    )
    size: str = Field("1024x1024", description="图片尺寸")
    count: int = Field(1, ge=1, le=4, description="生成数量 1-4")


class BatchGenerateRequest(BaseModel):
    """批量生成图片请求"""

    product_ids: List[int] = Field(
        ..., min_length=1, max_length=50, description="商品ID列表(1-50)"
    )
    prompt: str = Field(..., min_length=1, max_length=2000, description="正向提示词")
    negative_prompt: Optional[str] = Field("", description="反向提示词")
    style: str = Field(
        "product_white", description="风格: product_white/scene/model/3d_render"
    )
    size: str = Field("1024x1024", description="图片尺寸")
    count: int = Field(1, ge=1, le=4, description="每商品生成数量 1-4")


class BatchGenerateItem(BaseModel):
    """批量生成 — 单个商品的结果"""

    product_id: int
    product_name: Optional[str] = None
    job_id: int
    status: str
    images: List[str] = Field(default_factory=list)
    error: Optional[str] = None


class BatchGenerateResponse(BaseModel):
    """批量生成响应"""

    batch_id: str = Field(..., description="批次UUID")
    total: int = 0
    success: int = 0
    failed: int = 0
    results: List[BatchGenerateItem] = Field(default_factory=list)


class GenerateImageResponse(BaseModel):
    """生成图片响应"""

    job_id: int = Field(..., description="生成任务ID")
    images: List[str] = Field(default_factory=list, description="生成图片URL列表")
    status: str = Field("pending", description="状态: pending/done/failed")
    error: Optional[str] = None


class SaveImageRequest(BaseModel):
    """保存图片到商品请求"""

    product_id: int = Field(..., description="商品ID")
    image_url: str = Field(..., description="图片URL")
    set_as_main: bool = Field(False, description="是否设为主图")


class RemoveBgRequest(BaseModel):
    """去背景请求"""

    image_url: str = Field(..., description="原始图片URL")


class GenHistoryItem(BaseModel):
    """生成历史条目"""

    id: int
    product_id: int
    prompt: str
    style: str
    status: str
    image_urls: List[str]
    created_at: datetime
    product_name: Optional[str] = None


class GenHistoryResponse(BaseModel):
    """生成历史响应"""

    items: List[GenHistoryItem] = Field(default_factory=list)
    total: int = 0


# ====== Prompt 模板 ======


class PromptTemplateCreate(BaseModel):
    """创建模板请求"""

    name: str = Field(..., min_length=1, max_length=200, description="模板名称")
    description: Optional[str] = Field("", description="模板描述")
    prompt: str = Field(..., min_length=1, max_length=2000, description="正向提示词")
    negative_prompt: Optional[str] = Field("", description="反向提示词")
    style: str = Field("product_white", description="风格")
    size: str = Field("1024x1024", description="图片尺寸")
    platform_code: Optional[str] = Field(None, description="关联平台代码")
    is_shared: bool = Field(True, description="是否团队共享")


class PromptTemplateUpdate(BaseModel):
    """更新模板请求"""

    name: Optional[str] = Field(None, max_length=200)
    description: Optional[str] = None
    prompt: Optional[str] = Field(None, max_length=2000)
    negative_prompt: Optional[str] = None
    style: Optional[str] = None
    size: Optional[str] = None
    platform_code: Optional[str] = None
    is_shared: Optional[bool] = None


class PromptTemplateItem(BaseModel):
    """模板条目"""

    id: int
    name: str
    description: Optional[str] = None
    prompt: str
    negative_prompt: Optional[str] = None
    style: str
    size: str
    platform_code: Optional[str] = None
    is_shared: bool
    usage_count: int
    created_by: Optional[int] = None
    created_at: datetime
    updated_at: datetime


# ====== 编辑 / 视频 ======


class InpaintRequest(BaseModel):
    """局部重绘请求"""

    image_url: str = Field(..., description="原始图片URL")
    mask_base64: str = Field(..., description="mask 图片(base64), 白色区域为重绘区")
    prompt: str = Field(..., min_length=1, max_length=1000, description="描述重绘内容")
    negative_prompt: Optional[str] = Field("", description="反向提示词")


class OutpaintRequest(BaseModel):
    """扩图请求"""

    image_url: str = Field(..., description="原始图片URL")
    direction: str = Field("right", description="扩图方向: left/right/top/bottom")
    prompt: str = Field(..., min_length=1, max_length=1000, description="描述扩展内容")
    expand_ratio: float = Field(0.3, ge=0.1, le=1.0, description="扩展比例")


class VideoGenRequest(BaseModel):
    """AI 视频生成请求"""

    prompt: str = Field(..., min_length=1, max_length=2000, description="视频描述")
    image_url: Optional[str] = Field(None, description="起始图片(可选)")


class SlideshowRequest(BaseModel):
    """图片合成视频请求"""

    image_urls: List[str] = Field(
        ..., min_length=2, max_length=50, description="图片序列"
    )
    duration_per_frame: float = Field(2.0, ge=0.5, le=10.0, description="每帧时长(秒)")
    transition: str = Field("fade", description="转场: fade/none")
    resolution: str = Field("1920x1080", description="视频分辨率")
