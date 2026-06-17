"""AI 生图 - 业务逻辑"""

import asyncio
import logging
from typing import Optional, List
from datetime import datetime, timezone

from sqlalchemy import select, func, desc
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.models import ProductImageGen, Product
from app.common.utils import save_upload_file

logger = logging.getLogger(__name__)


# ====== 风格预设，辅助用户写 prompt ======
STYLE_PRESETS = {
    "product_white": {
        "label": "白底产品图",
        "suffix": "Professional product photography, pure white background, studio lighting, sharp focus, high detail, 8K",
    },
    "scene": {
        "label": "场景图",
        "suffix": "Lifestyle shot in a natural setting, ambient lighting, warm tones, realistic environment, professional photography",
    },
    "model": {
        "label": "模特展示",
        "suffix": "Model wearing the product, studio photography, soft lighting, natural pose, fashion editorial style",
    },
    "3d_render": {
        "label": "3D 渲染",
        "suffix": "3D render, isometric view, clean geometric background, vibrant colors, octane render, trending on artstation",
    },
}


class ImageGenService:

    @staticmethod
    async def generate(
        db: AsyncSession,
        user_id: int,
        product_id: int,
        prompt: str,
        style: str = "product_white",
        negative_prompt: str = "",
        size: str = "1024x1024",
        count: int = 1,
        batch_id: Optional[str] = None,
    ) -> dict:
        """生成图片"""
        # 1. 校验商品存在
        product = await db.get(Product, product_id)
        if not product:
            raise ValueError(f"商品不存在: product_id={product_id}")

        # 2. 组装完整 prompt
        style_info = STYLE_PRESETS.get(style, STYLE_PRESETS["product_white"])
        full_prompt = f"{prompt}. {style_info['suffix']}" if prompt else style_info['suffix']

        # 3. 创建生成记录
        record = ProductImageGen(
            product_id=product_id,
            prompt=prompt,
            style=style,
            negative_prompt=negative_prompt,
            size=size,
            requested_count=count,
            status="pending",
            created_by=user_id,
            batch_id=batch_id,
        )
        db.add(record)
        await db.flush()
        await db.refresh(record)
        gen_id = record.id

        try:
            # 4. 调用外部 API 生成
            image_urls = await ImageGenService._call_external_api(
                prompt=full_prompt,
                negative_prompt=negative_prompt,
                size=size,
                count=count,
            )

            # 5. 下载图片到本地存储
            local_urls = []
            for url in image_urls:
                local_url = await ImageGenService._download_image(url)
                if local_url:
                    local_urls.append(local_url)

            # 6. 更新记录
            record.status = "done"
            record.image_urls = local_urls
            await db.flush()

            return {
                "job_id": gen_id,
                "images": local_urls,
                "status": "done",
            }

        except Exception as e:
            logger.exception("图片生成失败 gen_id=%s", gen_id)
            record.status = "failed"
            record.error_message = str(e)
            await db.flush()
            return {
                "job_id": gen_id,
                "images": [],
                "status": "failed",
                "error": str(e),
            }

    @staticmethod
    async def save_to_product(
        db: AsyncSession,
        product_id: int,
        image_url: str,
        set_as_main: bool = False,
    ) -> dict:
        """保存生成图片到商品"""
        product = await db.get(Product, product_id)
        if not product:
            raise ValueError(f"商品不存在: product_id={product_id}")

        if set_as_main:
            product.main_image = image_url

        # 追加到图片列表（去重）
        current = product.images or []
        if not isinstance(current, list):
            current = []
        if image_url not in current:
            current.append(image_url)
        product.images = current
        await db.flush()

        return {"image_url": image_url, "set_as_main": set_as_main}

    @staticmethod
    async def get_history(
        db: AsyncSession,
        product_id: Optional[int] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> dict:
        """查询生成历史"""
        query = (
            select(
                ProductImageGen.id,
                ProductImageGen.product_id,
                ProductImageGen.prompt,
                ProductImageGen.style,
                ProductImageGen.status,
                ProductImageGen.image_urls,
                ProductImageGen.created_at,
                Product.name.label("product_name"),
            )
            .join(Product, ProductImageGen.product_id == Product.id, isouter=True)
            .order_by(desc(ProductImageGen.created_at))
        )

        if product_id:
            query = query.where(ProductImageGen.product_id == product_id)

        # 总数
        count_q = select(func.count()).select_from(query.subquery())
        total = (await db.execute(count_q)).scalar() or 0

        # 分页
        offset = (page - 1) * page_size
        query = query.offset(offset).limit(page_size)
        rows = (await db.execute(query)).all()

        items = []
        for r in rows:
            items.append({
                "id": r.id,
                "product_id": r.product_id,
                "prompt": r.prompt,
                "style": r.style,
                "status": r.status,
                "image_urls": r.image_urls or [],
                "created_at": r.created_at,
                "product_name": r.product_name,
            })

        return {"items": items, "total": total}

    @staticmethod
    async def batch_generate(
        db: AsyncSession,
        user_id: int,
        product_ids: List[int],
        prompt: str,
        style: str = "product_white",
        negative_prompt: str = "",
        size: str = "1024x1024",
        count: int = 1,
    ) -> dict:
        """批量生成图片 — 逐个商品调用 generate，共享一个 batch_id"""
        import uuid
        batch_id = uuid.uuid4().hex[:12]
        results = []
        success = 0
        failed = 0

        for pid in product_ids:
            try:
                res = await ImageGenService.generate(
                    db=db, user_id=user_id, product_id=pid,
                    prompt=prompt, style=style, negative_prompt=negative_prompt,
                    size=size, count=count, batch_id=batch_id,
                )
                # 拿商品名
                product = await db.get(Product, pid)
                pname = product.name if product else ""
                item = {
                    "product_id": pid,
                    "product_name": pname,
                    "job_id": res["job_id"],
                    "status": res["status"],
                    "images": res.get("images", []),
                    "error": res.get("error"),
                }
                results.append(item)
                if res["status"] == "done":
                    success += 1
                else:
                    failed += 1
            except Exception as e:
                results.append({
                    "product_id": pid, "product_name": "",
                    "job_id": 0, "status": "failed", "images": [], "error": str(e),
                })
                failed += 1

        return {
            "batch_id": batch_id,
            "total": len(product_ids),
            "success": success,
            "failed": failed,
            "results": results,
        }

    @staticmethod
    async def remove_background(
        db: AsyncSession,
        image_url: str,
    ) -> Optional[str]:
        """去背景 — 调用 remove.bg API，返回处理后的本地图片 URL"""
        import httpx

        api_key = settings.REMOVE_BG_API_KEY
        if not api_key:
            logger.warning("REMOVE_BG_API_KEY 未配置，使用 fallback 方案")
            return await ImageGenService._remove_bg_fallback(image_url)

        try:
            # 先下载原图
            async with httpx.AsyncClient(timeout=30, follow_redirects=True) as client:
                img_resp = await client.get(image_url)
                img_resp.raise_for_status()
                img_bytes = img_resp.bytes()

            # 调用 remove.bg
            async with httpx.AsyncClient(timeout=60) as client:
                resp = await client.post(
                    "https://api.remove.bg/v1.0/removebg",
                    headers={"X-Api-Key": api_key},
                    files={"image_file": ("image.jpg", img_bytes, "image/jpeg")},
                    data={"size": "auto"},
                )
                resp.raise_for_status()
                out_bytes = resp.content

            local_url = await save_upload_file(out_bytes, "nobg.jpg")
            return local_url

        except Exception as e:
            logger.warning("去背景失败: %s", e)
            return await ImageGenService._remove_bg_fallback(image_url)

    @staticmethod
    async def _remove_bg_fallback(image_url: str) -> Optional[str]:
        """去背景 fallback：直接复制原图（未配置 API Key 时兜底）"""
        import httpx
        try:
            async with httpx.AsyncClient(timeout=30, follow_redirects=True) as client:
                resp = await client.get(image_url)
                resp.raise_for_status()
            return await save_upload_file(resp.bytes(), "nobg_fallback.jpg")
        except Exception as e:
            logger.warning("去背景 fallback 失败: %s", e)
            return None

    # ========== Prompt 模板 CRUD ==========

    @staticmethod
    async def list_templates(
        db: AsyncSession,
        user_id: int,
        platform_code: Optional[str] = None,
        page: int = 1,
        page_size: int = 50,
    ) -> dict:
        """查询模板列表（共享的 + 自己创建的）"""
        from app.models import PromptTemplate

        conditions = [
            (PromptTemplate.is_shared == 1) | (PromptTemplate.created_by == user_id)
        ]
        if platform_code:
            conditions.append(
                (PromptTemplate.platform_code == platform_code) | (PromptTemplate.platform_code.is_(None))
            )

        query = (
            select(PromptTemplate)
            .where(*conditions)
            .order_by(desc(PromptTemplate.usage_count), desc(PromptTemplate.created_at))
        )
        count_q = select(func.count()).select_from(query.subquery())
        total = (await db.execute(count_q)).scalar() or 0

        offset = (page - 1) * page_size
        query = query.offset(offset).limit(page_size)
        rows = (await db.execute(query)).scalars().all()

        items = []
        for t in rows:
            items.append({
                "id": t.id,
                "name": t.name,
                "description": t.description,
                "prompt": t.prompt,
                "negative_prompt": t.negative_prompt or "",
                "style": t.style or "product_white",
                "size": t.size or "1024x1024",
                "platform_code": t.platform_code,
                "is_shared": bool(t.is_shared),
                "usage_count": t.usage_count or 0,
                "created_by": t.created_by,
                "created_at": t.created_at,
                "updated_at": t.updated_at,
            })

        return {"items": items, "total": total}

    @staticmethod
    async def create_template(
        db: AsyncSession,
        user_id: int,
        name: str,
        prompt: str,
        description: str = "",
        negative_prompt: str = "",
        style: str = "product_white",
        size: str = "1024x1024",
        platform_code: Optional[str] = None,
        is_shared: bool = True,
    ) -> dict:
        """创建 Prompt 模板"""
        from app.models import PromptTemplate

        tpl = PromptTemplate(
            name=name, description=description,
            prompt=prompt, negative_prompt=negative_prompt,
            style=style, size=size,
            platform_code=platform_code,
            is_shared=1 if is_shared else 0,
            created_by=user_id,
        )
        db.add(tpl)
        await db.flush()
        await db.refresh(tpl)
        return {"id": tpl.id, "name": tpl.name}

    @staticmethod
    async def update_template(
        db: AsyncSession,
        template_id: int,
        user_id: int,
        updates: dict,
    ) -> dict:
        """更新 Prompt 模板"""
        from app.models import PromptTemplate

        tpl = await db.get(PromptTemplate, template_id)
        if not tpl:
            raise ValueError(f"模板不存在: id={template_id}")
        # 仅创建者可修改
        if tpl.created_by and tpl.created_by != user_id:
            raise PermissionError("无权修改他人创建的模板")

        for key, val in updates.items():
            if val is not None and hasattr(tpl, key):
                if key == "is_shared":
                    val = 1 if val else 0
                setattr(tpl, key, val)

        await db.flush()
        return {"id": tpl.id, "name": tpl.name}

    @staticmethod
    async def delete_template(
        db: AsyncSession,
        template_id: int,
        user_id: int,
    ) -> None:
        """删除 Prompt 模板"""
        from app.models import PromptTemplate

        tpl = await db.get(PromptTemplate, template_id)
        if not tpl:
            raise ValueError(f"模板不存在: id={template_id}")
        if tpl.created_by and tpl.created_by != user_id:
            raise PermissionError("无权删除他人创建的模板")
        await db.delete(tpl)
        await db.flush()

    @staticmethod
    async def increment_template_usage(
        db: AsyncSession,
        template_id: int,
    ) -> None:
        """增加模板使用计数"""
        from app.models import PromptTemplate

        tpl = await db.get(PromptTemplate, template_id)
        if tpl:
            tpl.usage_count = (tpl.usage_count or 0) + 1
            await db.flush()

    # ========== 内部方法 ==========

    @staticmethod
    async def _call_external_api(
        prompt: str,
        negative_prompt: str = "",
        size: str = "1024x1024",
        count: int = 1,
    ) -> List[str]:
        """调用外部 AI 生图 API（默认 Replicate + FLUX.2 Pro）"""
        provider = settings.IMAGE_GEN_PROVIDER

        if provider == "replicate":
            return await ImageGenService._call_replicate(prompt, negative_prompt, size, count)
        elif provider == "openai":
            return await ImageGenService._call_openai(prompt, size, count)
        else:
            raise ValueError(f"不支持的图片生成服务商: {provider}")

    @staticmethod
    async def _call_replicate(
        prompt: str,
        negative_prompt: str = "",
        size: str = "1024x1024",
        count: int = 1,
    ) -> List[str]:
        """通过 Replicate API 调用 FLUX.2 Pro 生成图片"""
        import httpx

        api_key = settings.REPLICATE_API_KEY
        if not api_key:
            raise ValueError("REPLICATE_API_KEY 未配置")

        # 解析尺寸
        size_map = {
            "1024x1024": (1024, 1024),
            "768x1024": (768, 1024),
            "1024x768": (1024, 768),
            "1536x1024": (1536, 1024),
            "1024x1536": (1024, 1536),
        }
        width, height = size_map.get(size, (1024, 1024))

        # 调用 Replicate predictions API
        url = "https://api.replicate.com/v1/models/black-forest-labs/flux-2-pro/predictions"
        headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        }
        body = {
            "input": {
                "prompt": prompt,
                "negative_prompt": negative_prompt or None,
                "width": width,
                "height": height,
                "num_outputs": count,
                "output_format": "jpg",
                "guidance": 3.0,
                "num_inference_steps": 28,
            }
        }

        async with httpx.AsyncClient(timeout=30) as client:
            resp = await client.post(url, json=body, headers=headers)
            resp.raise_for_status()
            prediction = resp.json()

            # 轮询直到完成
            get_url = prediction["urls"]["get"]
            for _ in range(60):  # 最多等 60 秒
                await asyncio.sleep(1)
                poll = await client.get(get_url, headers=headers)
                poll.raise_for_status()
                status_data = poll.json()
                if status_data["status"] == "succeeded":
                    return status_data["output"] if isinstance(status_data["output"], list) else [status_data["output"]]
                elif status_data["status"] == "failed":
                    raise RuntimeError(f"Replicate 生成失败: {status_data.get('error', 'unknown')}")

            raise TimeoutError("Replicate 生成超时")

    @staticmethod
    async def _call_openai(
        prompt: str,
        size: str = "1024x1024",
        count: int = 1,
    ) -> List[str]:
        """通过 OpenAI DALL-E 3 生成图片"""
        import httpx

        api_key = settings.OPENAI_API_KEY
        if not api_key:
            raise ValueError("OPENAI_API_KEY 未配置")

        url = "https://api.openai.com/v1/images/generations"
        headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        }
        body = {
            "model": "dall-e-3",
            "prompt": prompt,
            "n": count,
            "size": size if size in ("1024x1024", "1024x1792", "1792x1024") else "1024x1024",
            "quality": "standard",
        }

        async with httpx.AsyncClient(timeout=60) as client:
            resp = await client.post(url, json=body, headers=headers)
            resp.raise_for_status()
            data = resp.json()
            return [item["url"] for item in data["data"]]

    @staticmethod
    async def _download_image(url: str) -> Optional[str]:
        """下载远程图片到本地 uploads 目录"""
        import httpx

        try:
            async with httpx.AsyncClient(timeout=30, follow_redirects=True) as client:
                resp = await client.get(url)
                resp.raise_for_status()
                content = resp.bytes()
                # 从 URL 推断文件名
                filename = url.split("/")[-1].split("?")[0] or "gen_image.jpg"
                if "." not in filename:
                    filename += ".jpg"
                local_url = await save_upload_file(content, filename)
                return local_url
        except Exception as e:
            logger.warning("下载远程图片失败: %s - %s", url, e)
            return None
