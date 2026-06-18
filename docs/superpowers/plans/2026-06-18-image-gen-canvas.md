# Image Gen Canvas Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Fabric.js infinite canvas page for e-commerce product image/video generation and editing.

**Architecture:** New frontend page + extend existing backend ImageGenService. Fabric.js powers the canvas (zoom/pan/layers/mask). Backend adds canvas CRUD + inpaint/outpaint/video endpoints on top of existing replicate/openai providers.

**Tech Stack:** Vue 3 / TypeScript / Naive UI / Fabric.js 6 / FastAPI / SQLAlchemy async / Replicate API / FFmpeg

---

### Task 1: ProductCanvas Model + Alembic Migration

**Files:**
- Modify: `backend/app/models.py`
- Create: (Alembic auto-generates migration)

- [ ] **Step 1: Add ProductCanvas model to models.py**

Add after existing `ProductImageGen` model:

```python
class ProductCanvas(Base):
    __tablename__ = "product_canvases"

    id: Mapped[int] = mapped_column(primary_key=True)
    product_id: Mapped[int] = mapped_column(ForeignKey("products.id"))
    name: Mapped[str] = mapped_column(String(200), default="未命名画布")
    layers: Mapped[dict] = mapped_column(JSON)
    thumbnail: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    created_by: Mapped[int] = mapped_column(ForeignKey("users.id"))
    created_at: Mapped[datetime] = mapped_column(server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(server_default=func.now(), onupdate=func.now())
```

- [ ] **Step 2: Generate migration**

Run: `cd backend && .venv/bin/alembic revision --autogenerate -m "add product_canvases"`

- [ ] **Step 3: Apply migration**

Run: `cd backend && .venv/bin/alembic upgrade heads`

---

### Task 2: Canvas Backend — Schemas + Service + Router

**Files:**
- Create: `backend/app/image_gen/canvas_schemas.py`
- Create: `backend/app/image_gen/canvas_service.py`
- Create: `backend/app/image_gen/canvas_router.py`

- [ ] **Step 1: Canvas pydantic schemas**

```python
from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, Field


class CanvasLayerItem(BaseModel):
    id: str = Field(..., description="图层唯一ID")
    type: str = Field(..., description="类型: image/text/mask")
    fabric_json: dict = Field(..., description="Fabric.js 对象 JSON")


class CanvasSaveRequest(BaseModel):
    product_id: int = Field(..., description="关联商品ID")
    name: str = Field("未命名画布", max_length=200)
    layers: List[CanvasLayerItem] = Field(default_factory=list)


class CanvasItem(BaseModel):
    id: int
    product_id: int
    name: str
    layers: List[dict]
    thumbnail: Optional[str] = None
    created_by: int
    created_at: datetime
    updated_at: datetime


class CanvasListResponse(BaseModel):
    items: List[CanvasItem] = Field(default_factory=list)
    total: int = 0
```

- [ ] **Step 2: Canvas service**

```python
import logging
from typing import Optional, List
from sqlalchemy import select, func, desc
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import ProductCanvas

logger = logging.getLogger(__name__)


class CanvasService:

    @staticmethod
    async def save(
        db: AsyncSession,
        user_id: int,
        product_id: int,
        name: str,
        layers: list,
        canvas_id: Optional[int] = None,
    ) -> dict:
        if canvas_id:
            canvas = await db.get(ProductCanvas, canvas_id)
            if not canvas:
                raise ValueError(f"画布不存在: id={canvas_id}")
            canvas.layers = layers
            canvas.name = name
            await db.flush()
            await db.refresh(canvas)
        else:
            canvas = ProductCanvas(
                product_id=product_id,
                name=name,
                layers=layers,
                created_by=user_id,
            )
            db.add(canvas)
            await db.flush()
            await db.refresh(canvas)
        return {"id": canvas.id, "name": canvas.name}

    @staticmethod
    async def load(db: AsyncSession, canvas_id: int) -> Optional[dict]:
        canvas = await db.get(ProductCanvas, canvas_id)
        if not canvas:
            return None
        return {
            "id": canvas.id,
            "product_id": canvas.product_id,
            "name": canvas.name,
            "layers": canvas.layers,
            "thumbnail": canvas.thumbnail,
            "created_by": canvas.created_by,
            "created_at": str(canvas.created_at),
            "updated_at": str(canvas.updated_at),
        }

    @staticmethod
    async def list_by_product(
        db: AsyncSession, product_id: int, page: int = 1, page_size: int = 20
    ) -> dict:
        query = (
            select(ProductCanvas)
            .where(ProductCanvas.product_id == product_id)
            .order_by(desc(ProductCanvas.updated_at))
        )
        count_q = select(func.count()).select_from(query.subquery())
        total = (await db.execute(count_q)).scalar() or 0
        offset = (page - 1) * page_size
        query = query.offset(offset).limit(page_size)
        rows = (await db.execute(query)).scalars().all()
        items = []
        for c in rows:
            items.append({
                "id": c.id,
                "product_id": c.product_id,
                "name": c.name,
                "layers": c.layers,
                "thumbnail": c.thumbnail,
                "created_by": c.created_by,
                "created_at": str(c.created_at),
                "updated_at": str(c.updated_at),
            })
        return {"items": items, "total": total}

    @staticmethod
    async def delete(db: AsyncSession, canvas_id: int, user_id: int) -> None:
        canvas = await db.get(ProductCanvas, canvas_id)
        if not canvas:
            raise ValueError(f"画布不存在: id={canvas_id}")
        if canvas.created_by != user_id:
            raise PermissionError("无权删除他人画布")
        await db.delete(canvas)
        await db.flush()
```

- [ ] **Step 3: Canvas router**

```python
from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.auth.dependencies import require_permission, get_current_user
from app.common.schemas import Result
from app.image_gen.canvas_schemas import CanvasSaveRequest
from app.image_gen.canvas_service import CanvasService
from app.models import User

router = APIRouter(prefix="/image-gen/canvas", tags=["AI 生图 - 画布"])


@router.post("", summary="保存画布")
async def save_canvas(
    req: CanvasSaveRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await CanvasService.save(
            db=db, user_id=current_user.id,
            product_id=req.product_id, name=req.name,
            layers=[l.model_dump() for l in req.layers],
        )
        return Result.ok(result)
    except ValueError as e:
        return Result.error(str(e))


@router.get("/{canvas_id}", summary="加载画布")
async def load_canvas(
    canvas_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:history")),
):
    result = await CanvasService.load(db=db, canvas_id=canvas_id)
    if not result:
        return Result.not_found("画布不存在")
    return Result.ok(result)


@router.get("", summary="商品画布列表")
async def list_canvases(
    product_id: int = Query(..., description="商品ID"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:history")),
):
    result = await CanvasService.list_by_product(
        db=db, product_id=product_id, page=page, page_size=page_size
    )
    return Result.ok(result)


@router.delete("/{canvas_id}", summary="删除画布")
async def delete_canvas(
    canvas_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        await CanvasService.delete(db=db, canvas_id=canvas_id, user_id=current_user.id)
        return Result.ok(message="画布已删除")
    except (ValueError, PermissionError) as e:
        return Result.error(str(e))
```

---

### Task 3: Backend — Inpaint / Outpaint / Video

**Files:**
- Modify: `backend/app/image_gen/schemas.py`
- Modify: `backend/app/image_gen/service.py`
- Modify: `backend/app/image_gen/router.py`

- [ ] **Step 1: Add inpaint/outpaint/video schemas to schemas.py**

```python
class InpaintRequest(BaseModel):
    image_url: str = Field(..., description="原始图片URL")
    mask_base64: str = Field(..., description="mask 图片(base64), 白色区域为重绘区")
    prompt: str = Field(..., min_length=1, max_length=1000, description="描述重绘内容")
    negative_prompt: Optional[str] = Field("")


class OutpaintRequest(BaseModel):
    image_url: str = Field(..., description="原始图片URL")
    direction: str = Field("right", description="扩图方向: left/right/top/bottom")
    prompt: str = Field(..., min_length=1, max_length=1000, description="描述扩展内容")
    expand_ratio: float = Field(0.3, ge=0.1, le=1.0, description="扩展比例")


class VideoGenRequest(BaseModel):
    prompt: str = Field(..., min_length=1, max_length=2000, description="视频描述")
    image_url: Optional[str] = Field(None, description="起始图片(可选)")


class SlideshowRequest(BaseModel):
    image_urls: List[str] = Field(..., min_length=2, max_length=50, description="图片序列")
    duration_per_frame: float = Field(2.0, ge=0.5, le=10.0, description="每帧时长(秒)")
    transition: str = Field("fade", description="转场: fade/none")
    resolution: str = Field("1920x1080", description="视频分辨率")


class VideoJobStatus(BaseModel):
    job_id: str
    status: str
    video_url: Optional[str] = None
    error: Optional[str] = None
```

- [ ] **Step 2: Add inpaint/outpaint/video methods to service.py**

Add these methods to `ImageGenService`:

```python
@staticmethod
async def inpaint(
    db: AsyncSession,
    user_id: int,
    image_url: str,
    mask_base64: str,
    prompt: str,
    negative_prompt: str = "",
) -> dict:
    """局部重绘 — 调用 Replicate FLUX.2 Pro fill"""
    import httpx
    import base64

    # 如果 mask_base64 包含 data:URL 头，去掉
    if "," in mask_base64:
        mask_base64 = mask_base64.split(",")[1]

    api_key = settings.REPLICATE_API_KEY
    if not api_key:
        raise ValueError("REPLICATE_API_KEY 未配置")

    # 先下载原图并转 base64
    async with httpx.AsyncClient(timeout=30, follow_redirects=True) as client:
        img_resp = await client.get(image_url)
        img_resp.raise_for_status()
        image_b64 = base64.b64encode(img_resp.content).decode()

    url = "https://api.replicate.com/v1/models/black-forest-labs/flux-2-pro/predictions"
    headers = {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}
    body = {
        "input": {
            "prompt": prompt,
            "image": f"data:image/jpeg;base64,{image_b64}",
            "mask": f"data:image/png;base64,{mask_base64}",
            "negative_prompt": negative_prompt or None,
            "output_format": "jpg",
        }
    }

    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.post(url, json=body, headers=headers)
        resp.raise_for_status()
        prediction = resp.json()
        get_url = prediction["urls"]["get"]
        for _ in range(60):
            await asyncio.sleep(1)
            poll = await client.get(get_url, headers=headers)
            poll.raise_for_status()
            status_data = poll.json()
            if status_data["status"] == "succeeded":
                output_urls = status_data["output"]
                output_url = output_urls[0] if isinstance(output_urls, list) else output_urls
                local_url = await ImageGenService._download_image(output_url)
                return {"image_url": local_url}
            elif status_data["status"] == "failed":
                raise RuntimeError(f"Inpaint 失败: {status_data.get('error', 'unknown')}")
    raise TimeoutError("Inpaint 超时")


@staticmethod
async def outpaint(
    db: AsyncSession,
    user_id: int,
    image_url: str,
    direction: str,
    prompt: str,
    expand_ratio: float = 0.3,
) -> dict:
    """扩图 — 调用 Replicate FLUX outpaint"""
    import httpx
    api_key = settings.REPLICATE_API_KEY
    if not api_key:
        raise ValueError("REPLICATE_API_KEY 未配置")

    url = "https://api.replicate.com/v1/models/black-forest-labs/flux-2-pro/predictions"
    headers = {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}
    body = {
        "input": {
            "prompt": prompt,
            "image": image_url,
            "outpaint": direction,
            "outpaint_expand": expand_ratio,
            "output_format": "jpg",
        }
    }

    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.post(url, json=body, headers=headers)
        resp.raise_for_status()
        prediction = resp.json()
        get_url = prediction["urls"]["get"]
        for _ in range(60):
            await asyncio.sleep(1)
            poll = await client.get(get_url, headers=headers)
            poll.raise_for_status()
            status_data = poll.json()
            if status_data["status"] == "succeeded":
                output_urls = status_data["output"]
                output_url = output_urls[0] if isinstance(output_urls, list) else output_urls
                local_url = await ImageGenService._download_image(output_url)
                return {"image_url": local_url}
            elif status_data["status"] == "failed":
                raise RuntimeError(f"Outpaint 失败: {status_data.get('error', 'unknown')}")
    raise TimeoutError("Outpaint 超时")


@staticmethod
async def generate_video(
    db: AsyncSession,
    user_id: int,
    prompt: str,
    image_url: Optional[str] = None,
) -> dict:
    """AI 视频生成 — Replicate Stable Video Diffusion"""
    import httpx
    api_key = settings.REPLICATE_API_KEY
    if not api_key:
        raise ValueError("REPLICATE_API_KEY 未配置")

    input_data = {"prompt": prompt}
    if image_url:
        input_data["input_image"] = image_url

    url = "https://api.replicate.com/v1/models/stability-ai/stable-video-diffusion/predictions"
    headers = {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}
    body = {"input": input_data}

    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.post(url, json=body, headers=headers)
        resp.raise_for_status()
        prediction = resp.json()
        job_id = prediction["id"]

    return {"job_id": job_id, "status": "processing", "video_url": None}


@staticmethod
async def get_video_status(job_id: str) -> dict:
    """查询视频生成进度"""
    import httpx
    api_key = settings.REPLICATE_API_KEY
    if not api_key:
        raise ValueError("REPLICATE_API_KEY 未配置")

    url = f"https://api.replicate.com/v1/predictions/{job_id}"
    headers = {"Authorization": f"Bearer {api_key}"}

    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.get(url, headers=headers)
        resp.raise_for_status()
        data = resp.json()
        if data["status"] == "succeeded":
            output = data["output"]
            video_url = output[0] if isinstance(output, list) else output
            return {"job_id": job_id, "status": "done", "video_url": video_url}
        elif data["status"] == "failed":
            return {"job_id": job_id, "status": "failed", "video_url": None, "error": data.get("error")}
        return {"job_id": job_id, "status": "processing", "video_url": None}


@staticmethod
async def create_slideshow(
    db: AsyncSession,
    user_id: int,
    image_urls: List[str],
    duration_per_frame: float = 2.0,
    transition: str = "fade",
    resolution: str = "1920x1080",
) -> dict:
    """图片序列合成视频 — FFmpeg"""
    import httpx
    import os
    import subprocess
    import uuid

    output_filename = f"slideshow_{uuid.uuid4().hex[:12]}.mp4"
    output_path = os.path.join(settings.UPLOAD_DIR, output_filename)
    temp_dir = os.path.join(settings.UPLOAD_DIR, f"slideshow_{uuid.uuid4().hex[:12]}")
    os.makedirs(temp_dir, exist_ok=True)

    # 下载每张图片
    local_paths = []
    async with httpx.AsyncClient(timeout=30, follow_redirects=True) as client:
        for i, url in enumerate(image_urls):
            resp = await client.get(url)
            resp.raise_for_status()
            ext = "jpg"
            fpath = os.path.join(temp_dir, f"frame_{i:04d}.{ext}")
            with open(fpath, "wb") as f:
                f.write(resp.content)
            local_paths.append(fpath)

    # 用 FFmpeg 合成
    input_concat = os.path.join(temp_dir, "input.txt")
    with open(input_concat, "w") as f:
        for p in local_paths:
            f.write(f"file '{p}'\n")
            f.write(f"duration {duration_per_frame}\n")

    cmd = [
        "ffmpeg", "-y",
        "-f", "concat",
        "-safe", "0",
        "-i", input_concat,
        "-c:v", "libx264",
        "-pix_fmt", "yuv420p",
        "-r", "24",
        output_path,
    ]
    subprocess.run(cmd, capture_output=True, timeout=300)

    # 清理临时文件
    import shutil
    shutil.rmtree(temp_dir, ignore_errors=True)

    # 保存到 uploads
    from app.image_gen.service import save_upload_file
    with open(output_path, "rb") as f:
        video_bytes = f.read()
    local_url = await save_upload_file(video_bytes, output_filename)
    os.remove(output_path)

    return {"video_url": local_url, "duration": duration_per_frame * len(image_urls)}
```

- [ ] **Step 3: Add endpoints to router.py**

```python
@router.post("/inpaint", summary="局部重绘")
async def inpaint_image(
    req: InpaintRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await ImageGenService.inpaint(
            db=db, user_id=current_user.id,
            image_url=req.image_url,
            mask_base64=req.mask_base64,
            prompt=req.prompt,
            negative_prompt=req.negative_prompt or "",
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"局部重绘失败: {str(e)}")


@router.post("/outpaint", summary="扩图")
async def outpaint_image(
    req: OutpaintRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await ImageGenService.outpaint(
            db=db, user_id=current_user.id,
            image_url=req.image_url,
            direction=req.direction,
            prompt=req.prompt,
            expand_ratio=req.expand_ratio,
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"扩图失败: {str(e)}")


@router.post("/video", summary="AI 生成视频")
async def generate_video(
    req: VideoGenRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await ImageGenService.generate_video(
            db=db, user_id=current_user.id,
            prompt=req.prompt,
            image_url=req.image_url,
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"视频生成失败: {str(e)}")


@router.get("/video/status/{job_id}", summary="视频生成进度")
async def video_status(
    job_id: str,
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:history")),
):
    try:
        result = await ImageGenService.get_video_status(job_id=job_id)
        return Result.ok(result)
    except Exception as e:
        return Result.error(str(e))


@router.post("/video/slideshow", summary="图片合成视频")
async def create_slideshow(
    req: SlideshowRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),
    _=Depends(require_permission("image_gen:generate")),
):
    try:
        result = await ImageGenService.create_slideshow(
            db=db, user_id=current_user.id,
            image_urls=req.image_urls,
            duration_per_frame=req.duration_per_frame,
            transition=req.transition,
            resolution=req.resolution,
        )
        return Result.ok(result)
    except Exception as e:
        return Result.error(f"视频合成失败: {str(e)}")
```

---

### Task 4: Frontend — Route + API Module + Composable

**Files:**
- Create: `frontend/src/router/modules/imageGenCanvas.ts`
- Create: `frontend/src/api/modules/imageGenCanvas.ts`
- Create: `frontend/src/views/image_gen_canvas/composables/useFabricCanvas.ts`
- Create: `frontend/src/views/image_gen_canvas/composables/useGeneration.ts`

- [ ] **Step 1: Route config**

```typescript
import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: 'image-gen-canvas',
    name: 'ImageGenCanvas',
    component: () => import('@/views/image_gen_canvas/CanvasEditor.vue'),
    meta: {
      title: '无限画布',
      icon: 'grid',
      menu: true,
      perm: 'image_gen:generate',
    },
  },
]
```

- [ ] **Step 2: API module**

```typescript
import http from '@/api/http'

export interface CanvasLayer {
  id: string
  type: 'image' | 'text' | 'mask'
  fabric_json: Record<string, any>
}

export interface CanvasSaveRequest {
  product_id: number
  name?: string
  layers: CanvasLayer[]
}

export interface CanvasItem {
  id: number
  product_id: number
  name: string
  layers: Record<string, any>[]
  thumbnail?: string
  created_by: number
  created_at: string
  updated_at: string
}

export interface CanvasListResponse {
  items: CanvasItem[]
  total: number
}

export interface InpaintRequest {
  image_url: string
  mask_base64: string
  prompt: string
  negative_prompt?: string
}

export interface OutpaintRequest {
  image_url: string
  direction: string
  prompt: string
  expand_ratio?: number
}

export interface VideoGenRequest {
  prompt: string
  image_url?: string
}

export interface SlideshowRequest {
  image_urls: string[]
  duration_per_frame?: number
  transition?: string
  resolution?: string
}

export const imageGenCanvasApi = {
  saveCanvas(data: CanvasSaveRequest, canvasId?: number) {
    const url = canvasId ? `/image-gen/canvas/${canvasId}` : '/image-gen/canvas'
    return canvasId ? http.put(url, data) : http.post('/image-gen/canvas', data)
  },

  loadCanvas(canvasId: number) {
    return http.get<CanvasItem>(`/image-gen/canvas/${canvasId}`)
  },

  listCanvases(productId: number, params?: { page?: number; page_size?: number }) {
    return http.get<CanvasListResponse>('/image-gen/canvas', {
      params: { product_id: productId, ...params },
    })
  },

  deleteCanvas(canvasId: number) {
    return http.delete(`/image-gen/canvas/${canvasId}`)
  },

  inpaint(data: InpaintRequest) {
    return http.post<{ image_url: string }>('/image-gen/inpaint', data)
  },

  outpaint(data: OutpaintRequest) {
    return http.post<{ image_url: string }>('/image-gen/outpaint', data)
  },

  generateVideo(data: VideoGenRequest) {
    return http.post<{ job_id: string; status: string }>('/image-gen/video', data)
  },

  getVideoStatus(jobId: string) {
    return http.get<{ job_id: string; status: string; video_url?: string }>(
      `/image-gen/video/status/${jobId}`
    )
  },

  createSlideshow(data: SlideshowRequest) {
    return http.post<{ video_url: string; duration: number }>('/image-gen/video/slideshow', data)
  },
}
```

- [ ] **Step 3: useFabricCanvas composable**

```typescript
import { ref, onMounted, onUnmounted, type Ref } from 'vue'
import { fabric } from 'fabric'

export function useFabricCanvas(canvasEl: Ref<HTMLCanvasElement | null>) {
  const canvas = ref<fabric.Canvas | null>(null)
  const zoom = ref(1)
  const selectedObject = ref<fabric.Object | null>(null)

  function initCanvas() {
    if (!canvasEl.value) return
    const el = canvasEl.value
    const parent = el.parentElement!
    const c = new fabric.Canvas(el, {
      width: parent.clientWidth,
      height: parent.clientHeight,
      selection: true,
      preserveObjectStacking: true,
      backgroundColor: '#f0f0f0',
    })

    c.on('mouse:wheel', (opt) => {
      const delta = opt.e.deltaY
      let z = c.getZoom()
      z *= 0.999 ** delta
      z = Math.max(0.1, Math.min(5, z))
      c.zoomToPoint({ x: (opt.e as MouseEvent).offsetX, y: (opt.e as MouseEvent).offsetY }, z)
      zoom.value = z
      opt.e.preventDefault()
    })

    c.on('selection:created', (e) => { selectedObject.value = e.selected?.[0] || null })
    c.on('selection:updated', (e) => { selectedObject.value = e.selected?.[0] || null })
    c.on('selection:cleared', () => { selectedObject.value = null })

    canvas.value = c
  }

  function addImage(url: string) {
    if (!canvas.value) return
    fabric.Image.fromURL(url, (img) => {
      img.set({
        left: 100 + Math.random() * 200,
        top: 100 + Math.random() * 200,
        cornerColor: '#1890ff',
        cornerSize: 8,
        transparentCorners: false,
      })
      canvas.value!.add(img)
      canvas.value!.setActiveObject(img)
      canvas.value!.renderAll()
    }, { crossOrigin: 'anonymous' })
  }

  function addText(text: string) {
    if (!canvas.value) return
    const tb = new fabric.Textbox(text, {
      left: 200, top: 200,
      fontSize: 32,
      fill: '#333',
      fontFamily: 'Arial',
    })
    canvas.value.add(tb)
    canvas.value.setActiveObject(tb)
    canvas.value.renderAll()
  }

  function deleteSelected() {
    if (!canvas.value || !selectedObject.value) return
    canvas.value.remove(selectedObject.value)
    canvas.value.renderAll()
    selectedObject.value = null
  }

  function bringForward() {
    if (!canvas.value || !selectedObject.value) return
    canvas.value.bringForward(selectedObject.value)
    canvas.value.renderAll()
  }

  function sendBackward() {
    if (!canvas.value || !selectedObject.value) return
    canvas.value.sendBackwards(selectedObject.value)
    canvas.value.renderAll()
  }

  function exportImage(format: 'png' | 'jpeg' = 'png'): string {
    if (!canvas.value) return ''
    return canvas.value.toDataURL({ format, multiplier: 2 })
  }

  function getCanvasJSON(): object {
    if (!canvas.value) return {}
    return canvas.value.toJSON()
  }

  function loadFromJSON(json: object) {
    if (!canvas.value) return
    canvas.value.loadFromJSON(json, () => {
      canvas.value!.renderAll()
    })
  }

  function setZoom(z: number) {
    if (!canvas.value) return
    canvas.value.setZoom(z)
    zoom.value = z
    canvas.value.renderAll()
  }

  function fitToScreen() {
    if (!canvas.value) return
    canvas.value.setZoom(1)
    zoom.value = 1
    canvas.value.renderAll()
  }

  onMounted(() => {
    initCanvas()
  })

  onUnmounted(() => {
    canvas.value?.dispose()
  })

  return {
    canvas,
    zoom,
    selectedObject,
    addImage,
    addText,
    deleteSelected,
    bringForward,
    sendBackward,
    exportImage,
    getCanvasJSON,
    loadFromJSON,
    setZoom,
    fitToScreen,
  }
}
```

- [ ] **Step 4: useGeneration composable**

```typescript
import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { imageGenCanvasApi } from '@/api/modules/imageGenCanvas'
import type { CanvasLayer } from '@/api/modules/imageGenCanvas'

export function useGeneration() {
  const message = useMessage()
  const generating = ref(false)
  const videoGenerating = ref(false)
  const videoJobId = ref<string | null>(null)
  const videoUrl = ref<string | null>(null)

  async function inpaint(
    imageUrl: string,
    maskBase64: string,
    prompt: string,
    negativePrompt?: string
  ) {
    generating.value = true
    try {
      const resp = await imageGenCanvasApi.inpaint({
        image_url: imageUrl,
        mask_base64: maskBase64,
        prompt,
        negative_prompt: negativePrompt,
      })
      const data = resp.data as any
      message.success('重绘完成')
      return data?.image_url || null
    } catch (e: any) {
      message.error(e?.response?.data?.message || '重绘失败')
      return null
    } finally {
      generating.value = false
    }
  }

  async function outpaint(
    imageUrl: string,
    direction: string,
    prompt: string,
    expandRatio?: number
  ) {
    generating.value = true
    try {
      const resp = await imageGenCanvasApi.outpaint({
        image_url: imageUrl,
        direction,
        prompt,
        expand_ratio: expandRatio,
      })
      const data = resp.data as any
      message.success('扩图完成')
      return data?.image_url || null
    } catch (e: any) {
      message.error(e?.response?.data?.message || '扩图失败')
      return null
    } finally {
      generating.value = false
    }
  }

  async function startVideoGen(prompt: string, imageUrl?: string) {
    videoGenerating.value = true
    videoUrl.value = null
    try {
      const resp = await imageGenCanvasApi.generateVideo({ prompt, image_url: imageUrl })
      const data = resp.data as any
      videoJobId.value = data.job_id
      pollVideoStatus(data.job_id)
    } catch (e: any) {
      message.error(e?.response?.data?.message || '视频生成失败')
      videoGenerating.value = false
    }
  }

  async function pollVideoStatus(jobId: string) {
    const interval = setInterval(async () => {
      try {
        const resp = await imageGenCanvasApi.getVideoStatus(jobId)
        const data = resp.data as any
        if (data.status === 'done') {
          videoUrl.value = data.video_url
          videoGenerating.value = false
          videoJobId.value = null
          message.success('视频生成完成')
          clearInterval(interval)
        } else if (data.status === 'failed') {
          message.error(data.error || '视频生成失败')
          videoGenerating.value = false
          videoJobId.value = null
          clearInterval(interval)
        }
      } catch {
        clearInterval(interval)
        videoGenerating.value = false
      }
    }, 3000)
  }

  async function createSlideshow(
    imageUrls: string[],
    durationPerFrame?: number
  ) {
    videoGenerating.value = true
    try {
      const resp = await imageGenCanvasApi.createSlideshow({
        image_urls: imageUrls,
        duration_per_frame: durationPerFrame,
      })
      const data = resp.data as any
      videoUrl.value = data.video_url
      message.success('视频合成完成')
      return data.video_url
    } catch (e: any) {
      message.error(e?.response?.data?.message || '视频合成失败')
      return null
    } finally {
      videoGenerating.value = false
    }
  }

  return {
    generating,
    videoGenerating,
    videoJobId,
    videoUrl,
    inpaint,
    outpaint,
    startVideoGen,
    createSlideshow,
  }
}
```

---

### Task 5: Frontend — FabricCanvas.vue Component

**Files:**
- Create: `frontend/src/views/image_gen_canvas/components/FabricCanvas.vue`

```vue
<template>
  <div ref="containerRef" class="fabric-canvas-container">
    <canvas ref="canvasEl" />
    <!-- 工具栏覆盖层 -->
    <div class="absolute top-3 left-3 flex gap-1 z-10">
      <n-button size="tiny" quaternary @click="fitToScreen">
        <template #icon><svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"/></svg></template>
      </n-button>
      <n-button size="tiny" quaternary @click="setZoom(1)">100%</n-button>
      <span class="text-[11px] text-[var(--text-tertiary)] flex items-center">{{ Math.round(zoom * 100) }}%</span>
    </div>
    <!-- 右键菜单 -->
    <n-dropdown
      v-if="contextMenuVisible"
      :show="contextMenuVisible"
      :x="contextMenuX"
      :y="contextMenuY"
      :options="contextMenuOptions"
      @select="handleContextMenu"
      @clickoutside="contextMenuVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, provide } from 'vue'
import { useFabricCanvas } from '../composables/useFabricCanvas'
import type { DropdownOption } from 'naive-ui'

const emit = defineEmits<{
  addImage: [url: string]
  requestEdit: [action: string]
}>()

const containerRef = ref<HTMLDivElement | null>(null)
const canvasEl = ref<HTMLCanvasElement | null>(null)

const {
  canvas, zoom, selectedObject,
  addImage, addText, deleteSelected,
  bringForward, sendBackward,
  exportImage, getCanvasJSON, loadFromJSON,
  setZoom, fitToScreen,
} = useFabricCanvas(canvasEl)

// 右键菜单状态
const contextMenuVisible = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextMenuOptions: DropdownOption[] = [
  { label: '置顶', key: 'bring-forward' },
  { label: '置底', key: 'send-backward' },
  { label: '复制', key: 'duplicate' },
  { label: '删除', key: 'delete' },
  { type: 'divider' as const },
  { label: '局部重绘', key: 'inpaint' },
  { label: '扩图', key: 'outpaint' },
  { label: '去背景', key: 'remove-bg' },
  { type: 'divider' as const },
  { label: '导出此图层', key: 'export-layer' },
]

function handleContextMenu(key: string) {
  contextMenuVisible.value = false
  if (key === 'delete') deleteSelected()
  else if (key === 'bring-forward') bringForward()
  else if (key === 'send-backward') sendBackward()
  else if (key === 'duplicate') duplicateSelected()
  else if (key === 'export-layer') exportSelectedLayer()
  else emit('requestEdit', key)
}

function duplicateSelected() {
  if (!canvas.value || !selectedObject.value) return
  selectedObject.value.clone((cloned: fabric.Object) => {
    cloned.set({ left: (cloned.left || 0) + 20, top: (cloned.top || 0) + 20 })
    canvas.value!.add(cloned)
    canvas.value!.setActiveObject(cloned)
    canvas.value!.renderAll()
  })
}

function exportSelectedLayer() {
  if (!selectedObject.value) return
  const dataUrl = selectedObject.value.toDataURL({ multiplier: 2 })
  const link = document.createElement('a')
  link.download = 'layer.png'
  link.href = dataUrl
  link.click()
}

function onRightClick(e: MouseEvent) {
  e.preventDefault()
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  contextMenuVisible.value = true
}

onMounted(() => {
  containerRef.value?.addEventListener('contextmenu', onRightClick)
})

onUnmounted(() => {
  containerRef.value?.removeEventListener('contextmenu', onRightClick)
})

// 暴露父组件调用的方法
defineExpose({ addImage, addText, getCanvasJSON, loadFromJSON, exportImage, setZoom, fitToScreen, canvas })
</script>

<style scoped>
.fabric-canvas-container {
  width: 100%;
  height: 100%;
  position: relative;
  overflow: hidden;
  background: #f0f0f0;
}
.fabric-canvas-container :deep(canvas) {
  display: block;
}
</style>
```

---

### Task 6: Frontend — SideToolbar + ImagePanel

**Files:**
- Create: `frontend/src/views/image_gen_canvas/components/SideToolbar.vue`
- Create: `frontend/src/views/image_gen_canvas/components/ImagePanel.vue`

- [ ] **Step 1: SideToolbar.vue**

```vue
<template>
  <div class="side-toolbar">
    <!-- 工具按钮 -->
    <n-button :type="activeTool === 'select' ? 'primary' : 'tertiary'" size="small" @click="$emit('selectTool', 'select')">
      <template #icon><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 3l7.07 16.97 2.51-7.39 7.39-2.51L3 3z"/></svg></template>
    </n-button>
    <n-button :type="activeTool === 'generate' ? 'primary' : 'tertiary'" size="small" @click="$emit('selectTool', 'generate')">
      <template #icon><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3v18M3 12h18"/></svg></template>
    </n-button>
    <n-button :type="activeTool === 'text' ? 'primary' : 'tertiary'" size="small" @click="$emit('selectTool', 'text')">
      <template #icon><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 7 4 4 20 4 20 7"/><line x1="9" y1="20" x2="15" y2="20"/><line x1="12" y1="4" x2="12" y2="20"/></svg></template>
    </n-button>
    <n-button :type="activeTool === 'video' ? 'primary' : 'tertiary'" size="small" @click="$emit('selectTool', 'video')">
      <template #icon><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg></template>
    </n-button>
    <n-divider style="margin:8px 0" />
    <n-button size="small" quaternary @click="$emit('save')">
      <template #icon><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg></template>
    </n-button>
    <n-button size="small" quaternary @click="$emit('export')">
      <template #icon><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg></template>
    </n-button>
  </div>
</template>

<script setup lang="ts">
defineProps<{ activeTool: string }>()
defineEmits<{ selectTool: [tool: string]; save: []; export: [] }>()
</script>

<style scoped>
.side-toolbar {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  background: white;
  border-right: 1px solid var(--border-light);
  width: 48px;
  align-items: center;
}
</style>
```

- [ ] **Step 2: ImagePanel.vue**

```vue
<template>
  <div class="image-panel">
    <div class="panel-header">
      <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">图片</span>
    </div>
    <div class="panel-body">
      <!-- 搜索商品 -->
      <div class="px-3 py-2">
        <n-input v-model:value="searchQuery" size="tiny" placeholder="搜索商品..." />
      </div>
      <!-- 快捷生成 -->
      <div class="px-3 pb-2">
        <n-button size="tiny" block secondary @click="$emit('openGenerate')">
          <template #icon><svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3v18M3 12h18"/></svg></template>
          生图
        </n-button>
      </div>
      <!-- 已生成图片列表 -->
      <div class="px-3 pb-2">
        <label class="text-[11px] text-[var(--text-tertiary)]">拖入画布</label>
      </div>
      <div class="grid grid-cols-2 gap-1.5 px-3 pb-3">
        <div v-for="(img, i) in images" :key="i"
          class="relative aspect-square rounded-[4px] overflow-hidden border border-[var(--border-light)] cursor-grab active:cursor-grabbing bg-[var(--bg-subtle)]"
          draggable="true"
          @dragstart="onDragStart($event, img.url)">
          <img :src="img.url" class="w-full h-full object-cover" />
        </div>
        <div v-if="images.length === 0" class="col-span-2 text-center py-6 text-[11px] text-[var(--text-tertiary)]">
          暂无图片
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ images: { url: string; name?: string }[] }>()
defineEmits<{ openGenerate: []; addToCanvas: [url: string] }>()

const searchQuery = ref('')

function onDragStart(e: DragEvent, url: string) {
  e.dataTransfer?.setData('text/plain', url)
}
</script>

<style scoped>
.image-panel {
  width: 220px;
  background: white;
  border-left: 1px solid var(--border-light);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}
.panel-header {
  padding: 12px 16px 8px;
  border-bottom: 1px solid var(--border-light);
}
.panel-body {
  flex: 1;
  overflow-y: auto;
}
</style>
```

---

### Task 7: Frontend — VideoPanel.vue

**Files:**
- Create: `frontend/src/views/image_gen_canvas/components/VideoPanel.vue`

```vue
<template>
  <div class="video-panel">
    <div class="panel-header">
      <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">视频</span>
    </div>
    <div class="p-3 space-y-3">
      <!-- 模式切换 -->
      <n-tabs v-model:value="mode" size="small" type="segment">
        <n-tab key="ai" name="ai">AI 生成</n-tab>
        <n-tab key="slideshow" name="slideshow">Slideshow</n-tab>
      </n-tabs>

      <!-- AI 视频 -->
      <div v-if="mode === 'ai'" class="space-y-2">
        <n-input v-model:value="aiPrompt" type="textarea" :rows="2" placeholder="描述视频内容，例如：产品 360 度旋转展示" />
        <n-button size="small" type="primary" block :loading="videoGenerating" :disabled="!aiPrompt" @click="handleGenerateVideo">
          生成视频
        </n-button>
        <div v-if="videoJobId" class="text-[11px] text-[var(--text-tertiary)]">
          任务 ID: {{ videoJobId }}，处理中...
        </div>
      </div>

      <!-- Slideshow -->
      <div v-if="mode === 'slideshow'" class="space-y-2">
        <div class="text-[12px] text-[var(--text-secondary)]">已选 {{ selectedFrameUrls.length }} 帧</div>
        <div class="flex flex-wrap gap-1">
          <div v-for="(url, i) in selectedFrameUrls" :key="i"
            class="w-12 h-12 rounded-[4px] overflow-hidden border border-[var(--border-light)] relative">
            <img :src="url" class="w-full h-full object-cover" />
            <button class="absolute -top-1 -right-1 w-4 h-4 bg-red-500 text-white rounded-full text-[8px] flex items-center justify-center"
              @click="removeFrame(i)">×</button>
          </div>
        </div>
        <n-input-number v-model:value="durationPerFrame" :min="0.5" :max="10" :step="0.5" size="tiny">
          <template #prefix>每帧</template>
          <template #suffix>秒</template>
        </n-input-number>
        <n-button size="small" type="primary" block :loading="videoGenerating" :disabled="selectedFrameUrls.length < 2"
          @click="handleCreateSlideshow">
          合成视频
        </n-button>
      </div>

      <!-- 视频预览 -->
      <div v-if="videoUrl" class="mt-2">
        <video :src="videoUrl" controls class="w-full rounded-[6px]" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  videoGenerating: boolean
  videoJobId: string | null
  videoUrl: string | null
}>()

const emit = defineEmits<{
  generateVideo: [prompt: string]
  createSlideshow: [urls: string[], duration: number]
  addFrame: []
}>()

const mode = ref<'ai' | 'slideshow'>('ai')
const aiPrompt = ref('')
const selectedFrameUrls = ref<string[]>([])
const durationPerFrame = ref(2)

// 供父组件调用来添加帧
function addFrame(url: string) {
  if (!selectedFrameUrls.value.includes(url)) {
    selectedFrameUrls.value.push(url)
  }
}

function removeFrame(i: number) {
  selectedFrameUrls.value.splice(i, 1)
}

function handleGenerateVideo() {
  if (aiPrompt.value) emit('generateVideo', aiPrompt.value)
}

function handleCreateSlideshow() {
  if (selectedFrameUrls.value.length >= 2) {
    emit('createSlideshow', [...selectedFrameUrls.value], durationPerFrame.value)
  }
}

defineExpose({ addFrame })
</script>

<style scoped>
.video-panel {
  width: 280px;
  background: white;
  border-left: 1px solid var(--border-light);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}
.panel-header {
  padding: 12px 16px 8px;
  border-bottom: 1px solid var(--border-light);
}
</style>
```

---

### Task 8: Frontend — CanvasEditor.vue (Main View)

**Files:**
- Create: `frontend/src/views/image_gen_canvas/CanvasEditor.vue`

```vue
<template>
  <div class="canvas-editor h-full flex flex-col">
    <!-- 顶部：选择商品 + 画布操作 -->
    <div class="flex items-center gap-3 px-4 py-2 bg-white border-b border-[var(--border-light)] shrink-0">
      <n-select
        v-model:value="selectedProductId"
        :options="productOptions"
        placeholder="选择商品"
        filterable
        style="width:240px"
        size="small"
        @update:value="loadCanvases"
      />
      <n-select
        v-model:value="activeCanvasId"
        :options="canvasOptions"
        placeholder="选择画布"
        style="width:200px"
        size="small"
        clearable
      />
      <n-button size="tiny" secondary @click="showGenerateModal = true">
        <template #icon><svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3v18M3 12h18"/></svg></template>
        生图
      </n-button>
      <n-button size="tiny" quaternary @click="saveCanvas">保存画布</n-button>
      <n-button size="tiny" quaternary @click="exportCanvas">导出</n-button>
      <div class="flex-1" />
      <n-button size="tiny" quaternary @click="activeTool = 'video'">视频</n-button>
    </div>

    <!-- 主体：三栏布局 -->
    <div class="flex flex-1 overflow-hidden">
      <SideToolbar
        :activeTool="activeTool"
        @selectTool="activeTool = $event"
        @save="saveCanvas"
        @export="exportCanvas"
      />

      <!-- 中间：无限画布 -->
      <div class="flex-1 relative">
        <FabricCanvas ref="fabricCanvasRef" @requestEdit="handleEdit" />
      </div>

      <!-- 右侧面板：根据工具切换 -->
      <ImagePanel
        v-if="activeTool !== 'video'"
        ref="imagePanelRef"
        :images="generatedImages"
        @openGenerate="showGenerateModal = true"
      />

      <VideoPanel
        v-if="activeTool === 'video'"
        ref="videoPanelRef"
        :videoGenerating="videoGenerating"
        :videoJobId="videoJobId"
        :videoUrl="videoUrl"
        @generateVideo="handleGenerateVideo"
        @createSlideshow="handleCreateSlideshow"
      />
    </div>

    <!-- 生图弹窗 -->
    <n-modal v-model:show="showGenerateModal" title="AI 生图" preset="card" style="width:560px;max-width:90vw;">
      <template #header>
        <div class="text-base font-semibold">AI 生图</div>
      </template>
      <div class="space-y-3">
        <n-input v-model:value="genPrompt" type="textarea" :rows="2" placeholder="描述你想要的图片内容..." />
        <div class="grid grid-cols-3 gap-2">
          <n-select v-model:value="genStyle" :options="styleOptions" size="small" />
          <n-select v-model:value="genSize" :options="sizeOptions" size="small" />
          <n-input-number v-model:value="genCount" :min="1" :max="4" size="small" />
        </div>
        <n-button type="primary" block :loading="generating" :disabled="!genPrompt || !selectedProductId" @click="handleGenerate">
          生成
        </n-button>
      </div>
    </n-modal>

    <!-- 编辑弹窗 -->
    <n-modal v-model:show="showEditModal" title="AI 编辑" preset="card" style="width:480px;max-width:90vw;">
      <div class="space-y-3">
        <img v-if="editImageUrl" :src="editImageUrl" class="w-full max-h-48 object-contain rounded-[6px] bg-[var(--bg-subtle)]" />
        <div v-if="editMode === 'inpaint' || editMode === 'outpaint'">
          <n-input v-model:value="editPrompt" type="textarea" :rows="2" :placeholder="editPlaceholder" />
        </div>
        <div v-if="editMode === 'outpaint'" class="flex gap-2">
          <n-button v-for="d in directions" :key="d.value" size="tiny" :type="editDirection === d.value ? 'primary' : 'default'"
            @click="editDirection = d.value">{{ d.label }}</n-button>
        </div>
        <n-button type="primary" block :loading="generating" :disabled="!editPrompt" @click="confirmEdit">
          确认{{ editMode === 'remove-bg' ? '去背景' : editMode === 'inpaint' ? '重绘' : '扩图' }}
        </n-button>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { productApi } from '@/api'
import { imageGenApi } from '@/api/modules/imageGen'
import { imageGenCanvasApi } from '@/api/modules/imageGenCanvas'
import { useGeneration } from './composables/useGeneration'
import SideToolbar from './components/SideToolbar.vue'
import FabricCanvas from './components/FabricCanvas.vue'
import ImagePanel from './components/ImagePanel.vue'
import VideoPanel from './components/VideoPanel.vue'

const message = useMessage()
const fabricCanvasRef = ref<InstanceType<typeof FabricCanvas> | null>(null)
const videoPanelRef = ref<InstanceType<typeof VideoPanel> | null>(null)

const {
  generating, videoGenerating, videoJobId, videoUrl,
  inpaint, outpaint, startVideoGen, createSlideshow: doCreateSlideshow,
} = useGeneration()

// 商品 / 画布
const selectedProductId = ref<number | null>(null)
const activeCanvasId = ref<number | null>(null)
const products = ref<any[]>([])
const canvases = ref<any[]>([])
const activeTool = ref('select')
const generatedImages = ref<{ url: string; name?: string }[]>([])

const productOptions = computed(() =>
  products.value.map(p => ({ label: p.name || `商品 #${p.id}`, value: p.id }))
)

const canvasOptions = computed(() =>
  canvases.value.map(c => ({ label: c.name, value: c.id }))
)

// 生图
const showGenerateModal = ref(false)
const genPrompt = ref('')
const genStyle = ref('product_white')
const genSize = ref('1024x1024')
const genCount = ref(1)

const styleOptions = [
  { label: '白底产品图', value: 'product_white' },
  { label: '场景图', value: 'scene' },
  { label: '模特展示', value: 'model' },
  { label: '3D 渲染', value: '3d_render' },
]

const sizeOptions = [
  { label: '1024×1024', value: '1024x1024' },
  { label: '768×1024', value: '768x1024' },
  { label: '1024×768', value: '1024x768' },
  { label: '1536×1024', value: '1536x1024' },
  { label: '1024×1536', value: '1024x1536' },
]

// 编辑
const showEditModal = ref(false)
const editMode = ref<'inpaint' | 'outpaint' | 'remove-bg'>('inpaint')
const editImageUrl = ref('')
const editPrompt = ref('')
const editDirection = ref('right')
const directions = [
  { label: '← 左', value: 'left' },
  { label: '→ 右', value: 'right' },
  { label: '↑ 上', value: 'top' },
  { label: '↓ 下', value: 'bottom' },
]

const editPlaceholder = computed(() => ({
  inpaint: '描述要重绘的区域内容...',
  outpaint: '描述扩展区域的内容...',
  'remove-bg': '',
})[editMode.value])

async function loadProducts() {
  try {
    const resp = await productApi.list({ page: 1, page_size: 200 })
    const data = resp.data as any
    products.value = data?.records || data?.items || []
  } catch { message.warning('加载商品失败') }
}

async function loadCanvases() {
  if (!selectedProductId.value) return
  try {
    const resp = await imageGenCanvasApi.listCanvases(selectedProductId.value)
    const data = resp.data as any
    canvases.value = data?.items || []
  } catch { /* 静默 */ }
}

async function saveCanvas() {
  if (!selectedProductId.value || !fabricCanvasRef.value) return
  const layers = fabricCanvasRef.value.getCanvasJSON()
  const layersArr = [{
    id: 'root',
    type: 'image' as const,
    fabric_json: layers,
  }]
  try {
    await imageGenCanvasApi.saveCanvas({
      product_id: selectedProductId.value,
      name: `画布 ${new Date().toLocaleString()}`,
      layers: layersArr,
    }, activeCanvasId.value || undefined)
    message.success('画布已保存')
    loadCanvases()
  } catch { message.error('保存失败') }
}

function exportCanvas() {
  if (!fabricCanvasRef.value) return
  const dataUrl = fabricCanvasRef.value.exportImage('png')
  const link = document.createElement('a')
  link.download = 'canvas.png'
  link.href = dataUrl
  link.click()
}

async function handleGenerate() {
  if (!selectedProductId.value || !genPrompt.value) return
  try {
    const resp = await imageGenApi.generate({
      product_id: selectedProductId.value,
      prompt: genPrompt.value,
      style: genStyle.value,
      size: genSize.value,
      count: genCount.value,
    })
    const data = resp.data as any
    if (data?.images) {
      for (const url of data.images) {
        generatedImages.value.push({ url, name: genPrompt.value.slice(0, 30) })
        fabricCanvasRef.value?.addImage(url)
      }
    }
    message.success(`生成完成`)
    showGenerateModal.value = false
  } catch (e: any) {
    message.error(e?.response?.data?.message || '生成失败')
  }
}

async function handleEdit(action: string) {
  if (!fabricCanvasRef.value?.selectedObject) {
    message.warning('请先选中画布中的图片')
    return
  }
  const obj = fabricCanvasRef.value.selectedObject as any
  if (action === 'inpaint' || action === 'outpaint') {
    editImageUrl.value = obj.toDataURL({ multiplier: 1 })
    editMode.value = action
    editPrompt.value = ''
    showEditModal.value = true
  } else if (action === 'remove-bg') {
    editImageUrl.value = obj.toDataURL({ multiplier: 1 })
    editMode.value = 'remove-bg'
    showEditModal.value = false
    // 直接去背景
    try {
      const resp = await imageGenApi.removeBg(obj.toDataURL({ multiplier: 1 }))
      const data = resp.data as any
      if (data?.url) {
        fabricCanvasRef.value?.addImage(data.url)
        message.success('去背景完成')
      }
    } catch { message.error('去背景失败') }
  }
}

async function confirmEdit() {
  let result: string | null = null
  if (editMode.value === 'inpaint') {
    result = await inpaint(editImageUrl.value, '', editPrompt.value)
  } else if (editMode.value === 'outpaint') {
    result = await outpaint(editImageUrl.value, editDirection.value, editPrompt.value)
  }
  if (result && fabricCanvasRef.value) {
    fabricCanvasRef.value.addImage(result)
    generatedImages.value.push({ url: result })
  }
  showEditModal.value = false
}

async function handleGenerateVideo(prompt: string) {
  await startVideoGen(prompt, undefined)
}

async function handleCreateSlideshow(urls: string[], duration: number) {
  const result = await doCreateSlideshow(urls, duration)
  if (result) message.success('视频合成完成')
}

onMounted(() => {
  loadProducts()
})
</script>

<style scoped>
.canvas-editor {
  height: calc(100vh - 56px);
}
</style>
```

---

### Task 9: Install Fabric.js + Register Route

- [ ] **Step 1: Install Fabric.js**

Run: `cd frontend && npm install fabric@6 @types/fabric`

- [ ] **Step 2: Register route in router index**

Check `frontend/src/router/index.ts` to ensure the module is auto-merged (per project conventions, modules under `frontend/src/router/modules/` are auto-merged). Verify the route exists at `/image-gen-canvas`.

- [ ] **Step 3: Run frontend build check**

Run: `cd frontend && npx vue-tsc --noEmit --skipLibCheck` to verify TypeScript

---

### Task 10: Verify Backend

- [ ] **Step 1: Run existing tests**

Run: `cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_image_gen.py -q` (if exists) or run general test suite to ensure nothing broken.

- [ ] **Step 2: Verify server starts**

Run: `cd backend && .venv/bin/python -c "from app.image_gen.canvas_service import CanvasService; print('OK')"`
