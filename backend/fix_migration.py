"""临时修复：为 sku 表添加物流字段"""
import asyncio
from sqlalchemy import text
from sqlalchemy.ext.asyncio import create_async_engine
from app.config import settings


async def main():
    engine = create_async_engine(settings.DATABASE_URL)
    async with engine.begin() as conn:
        # SKU 级字段（锁库存 + 物流）
        await conn.execute(text("""
            ALTER TABLE sku
            ADD COLUMN IF NOT EXISTS lock_stock integer default 0,
            ADD COLUMN IF NOT EXISTS sku_length_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS sku_width_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS sku_height_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS sku_weight_kg numeric(10,2),
            ADD COLUMN IF NOT EXISTS package_length_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS package_width_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS package_height_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS package_weight_kg numeric(10,2)
        """))
        # Product 级物流字段
        await conn.execute(text("""
            ALTER TABLE product
            ADD COLUMN IF NOT EXISTS product_length_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS product_width_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS product_height_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS product_weight_kg numeric(10,2),
            ADD COLUMN IF NOT EXISTS package_length_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS package_width_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS package_height_cm numeric(10,2),
            ADD COLUMN IF NOT EXISTS package_weight_kg numeric(10,2),
            ADD COLUMN IF NOT EXISTS cargo_type varchar(50) default 'normal'
        """))
        # Product AI 字段
        await conn.execute(text("""
            ALTER TABLE product
            ADD COLUMN IF NOT EXISTS ai_title varchar(500),
            ADD COLUMN IF NOT EXISTS ai_description text,
            ADD COLUMN IF NOT EXISTS seo_keywords jsonb,
            ADD COLUMN IF NOT EXISTS ai_status varchar(50) default 'pending',
            ADD COLUMN IF NOT EXISTS platform_statuses jsonb
        """))
        # Inventory locked_quantity
        await conn.execute(text("""
            ALTER TABLE inventory
            ADD COLUMN IF NOT EXISTS locked_quantity integer default 0
        """))
        # 平台 api_key 字段
        await conn.execute(text("""
            ALTER TABLE platform
            ADD COLUMN IF NOT EXISTS api_key text
        """))
        print("所有字段已添加")
    await engine.dispose()


asyncio.run(main())
