"""AI 商品优化 — 服务层"""

import json
import asyncio
import re

import httpx
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.models import Product


class AiEnhanceService:
    """商品 AI 优化（标题、描述、SEO 关键词）"""

    @staticmethod
    async def enhance_product(db: AsyncSession, product_id: int) -> dict:
        """对商品执行 AI 优化，返回增强结果"""
        product = await db.get(Product, product_id)
        if not product:
            raise ValueError("商品不存在")

        name = product.name or ""
        subtitle = product.subtitle or ""
        description = product.description or ""

        # ----- 尝试调用真实 LLM -----
        if settings.LLM_API_KEY:
            result = await AiEnhanceService._call_llm_with_retry(
                name, subtitle, description
            )
        else:
            result = AiEnhanceService._fallback_data(name, description)

        # 保存到商品记录
        product.ai_title = result["title"]
        product.ai_description = result["description"]
        product.seo_keywords = result["keywords"]
        product.ai_status = "completed"
        await db.flush()

        return {
            "enhanced_title": result["title"],
            "enhanced_description": result["description"],
            "seo_keywords": result["keywords"],
            "ai_status": "completed",
            "message": "AI优化完成，请检查并确认",
        }

    @staticmethod
    async def _call_llm_with_retry(name: str, subtitle: str, description: str) -> dict:
        """调用 LLM，带 3 次重试，全部失败则回退"""
        prompt = f"""你是一个电商商品标题和描述优化专家。
请根据以下商品信息，生成优化的标题、描述和SEO关键词。

商品名称：{name}
商品副标题：{subtitle}
商品描述：{description}

请以JSON格式返回：
{{
  "title": "优化后的商品标题（不超过200字）",
  "description": "优化后的商品描述（包含产品特色、卖点、适合场景，200-500字）",
  "keywords": ["关键词1", "关键词2", "关键词3", "关键词4", "关键词5"]
}}
只返回JSON，不要额外说明。"""

        payload = {
            "model": settings.LLM_MODEL,
            "messages": [{"role": "user", "content": prompt}],
            "temperature": 0.7,
        }
        headers = {
            "Authorization": f"Bearer {settings.LLM_API_KEY}",
            "Content-Type": "application/json",
        }

        for attempt in range(3):
            try:
                async with httpx.AsyncClient(timeout=30) as client:
                    resp = await client.post(
                        settings.LLM_API_URL,
                        json=payload,
                        headers=headers,
                    )
                    resp.raise_for_status()
                    data = resp.json()
                    content = data["choices"][0]["message"]["content"]

                    # 尝试直接解析 JSON
                    try:
                        result = json.loads(content)
                    except json.JSONDecodeError:
                        json_match = re.search(r"```(?:json)?\s*([\s\S]*?)```", content)
                        if json_match:
                            result = json.loads(json_match.group(1))
                        else:
                            raise ValueError("无法解析LLM返回的JSON")

                    return {
                        "title": result.get("title", name),
                        "description": result.get("description", description or name),
                        "keywords": result.get("keywords", [name]),
                    }
            except Exception:
                if attempt < 2:
                    await asyncio.sleep(1 * (attempt + 1))

        # 全部失败 → 回退
        return AiEnhanceService._fallback_data(name, description or name)

    @staticmethod
    def _fallback_data(name: str, description: str) -> dict:
        """无 LLM Key 或全部重试失败时返回模拟增强数据"""
        return {
            "title": f"{name} - 高品质正品保障 厂家直销批发",
            "description": (
                f"【{name}】品质保障，正品货源。\n\n"
                f"产品特色：\n"
                f"✅ 优质材料，经久耐用\n"
                f"✅ 严格品控，质量可靠\n"
                f"✅ 厂家直供，价格优惠\n"
                f"✅ 支持批发/零售，快速发货\n\n"
                f"欢迎联系我们获取更多产品信息！"
            ),
            "keywords": [
                name,
                f"{name} 批发",
                f"{name} 厂家",
                f"{name} 价格",
            ],
        }
