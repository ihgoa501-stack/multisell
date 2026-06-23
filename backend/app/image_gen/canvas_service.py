"""AI 生图 - 画布业务逻辑"""

import logging
from typing import Optional
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
            "layers": canvas.layers if canvas.layers else [],
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
            items.append(
                {
                    "id": c.id,
                    "product_id": c.product_id,
                    "name": c.name,
                    "layers": c.layers if c.layers else [],
                    "thumbnail": c.thumbnail,
                    "created_by": c.created_by,
                    "created_at": str(c.created_at),
                    "updated_at": str(c.updated_at),
                }
            )
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
