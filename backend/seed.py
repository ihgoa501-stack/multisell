#!/usr/bin/env python3
"""
数据初始化脚本 — 独立可运行，不依赖 FastAPI 启动流程。

用法:
    cd backend
    pip install -r requirements.txt
    python seed.py

功能:
    1. 创建默认管理员账号 (admin / admin123)
    2. 创建演示分类数据 (至少10个分类)
    3. 创建示例平台 (Ozon / Shopee / Wildberries / 速卖通 / Temu)
    4. 创建示例品牌 (至少5个)
    5. 创建少量演示商品和 SKU
"""

import asyncio
import sys
import os

# 确保 backend/ 在 Python 路径中，使 app 包可导入
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from sqlalchemy import select
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker

from app.config import settings
from app.database import Base
from app.models import (
    User, Category, Platform, Brand,
    Product, SpecName, SpecValue, Sku,
    Inventory, Permission,
)
from app.auth.service import hash_password


# ── 种子数据 ──────────────────────────────────────────────────────────

SEED_CATEGORIES = [
    # (name, parent_name, level, sort_order) — parent_name="" 表示根分类
    ("服装",       "",  0, 1),
    ("男装",       "服装",  1, 1),
    ("女装",       "服装",  1, 2),
    ("电子产品",   "",  0, 2),
    ("手机配件",   "电子产品", 1, 1),
    ("智能穿戴",   "电子产品", 1, 2),
    ("家居用品",   "",  0, 3),
    ("厨房用具",   "家居用品", 1, 1),
    ("美妆个护",   "",  0, 4),
    ("护肤品",     "美妆个护", 1, 1),
    ("食品饮料",   "",  0, 5),
    ("休闲零食",   "食品饮料", 1, 1),
    ("母婴用品",   "",  0, 6),
    ("运动户外",   "",  0, 7),
    ("图书文具",   "",  0, 8),
    ("宠物用品",   "",  0, 9),
]

SEED_PLATFORMS = [
    ("Ozon",        "ozon",       "https://api.ozon.ru",          1),
    ("Shopee",      "shopee",     "https://api.shopee.com",       2),
    ("Wildberries", "wb",         "https://api.wildberries.ru",   3),
    ("速卖通",       "aliexpress", "https://api.aliexpress.com",  4),
    ("Temu",        "temu",       "https://api.temu.com",         5),
]

SEED_BRANDS = [
    ("TechPro",    "知名科技品牌，专注于智能数码产品"),
    ("NatureHome", "环保家居品牌，倡导自然生活方式"),
    ("StyleWear",  "时尚服装品牌，引领都市潮流"),
    ("BeautyGlow", "高端美妆品牌，源自韩国科研配方"),
    ("FreshFood",  "健康食品品牌，从农场到餐桌"),
    ("SportMax",   "专业运动品牌，助力极限挑战"),
]

SEED_PERMISSIONS = [
    {"code": "product:view", "name": "查看商品", "module": "product"},
    {"code": "order:view", "name": "查看订单", "module": "order"},
    {"code": "order:create", "name": "创建订单", "module": "order"},
    {"code": "order:update", "name": "更新订单", "module": "order"},
    {"code": "order:update_status", "name": "更新订单状态", "module": "order"},
    {"code": "order:cancel", "name": "取消订单", "module": "order"},
    {"code": "category:view", "name": "查看分类", "module": "category"},
    {"code": "category:create", "name": "创建分类", "module": "category"},
    {"code": "category:update", "name": "更新分类", "module": "category"},
    {"code": "category:delete", "name": "删除分类", "module": "category"},
    {"code": "brand:view", "name": "查看品牌", "module": "brand"},
    {"code": "brand:create", "name": "创建品牌", "module": "brand"},
    {"code": "brand:update", "name": "更新品牌", "module": "brand"},
    {"code": "brand:delete", "name": "删除品牌", "module": "brand"},
    {"code": "search:view", "name": "全局搜索", "module": "search"},
    {"code": "decision:calculate", "name": "上架前决策计算", "module": "decision"},
    {"code": "platform_fee:view", "name": "查看平台费用规则", "module": "platform_fee"},
    {"code": "platform_fee:manage", "name": "管理平台费用规则", "module": "platform_fee"},
    {"code": "platform_fee:calculate", "name": "匹配平台费用规则", "module": "platform_fee"},
    {"code": "listing:view", "name": "查看发布", "module": "listing"},
    {"code": "listing:task_manage", "name": "管理上架任务", "module": "listing"},
    {"code": "listing:publish", "name": "发布上架任务", "module": "listing"},
    {"code": "shipping:bill:import", "name": "导入运费账单", "module": "shipping"},
    {"code": "shipping:bill:view", "name": "查看运费账单", "module": "shipping"},
    {"code": "shipping:reconcile", "name": "运费对账", "module": "shipping"},
    {"code": "settlement:import", "name": "导入平台结算", "module": "settlement"},
    {"code": "settlement:view", "name": "查看平台结算", "module": "settlement"},
    {"code": "settlement:match", "name": "匹配平台结算", "module": "settlement"},
    {"code": "finance:ledger:view", "name": "查看财务账本", "module": "finance"},
    {"code": "finance:ledger:rebuild", "name": "重建财务账本", "module": "finance"},
    {"code": "exception:generate", "name": "生成异常", "module": "exception"},
    {"code": "exception:view", "name": "查看异常", "module": "exception"},
    {"code": "exception:manage", "name": "管理异常", "module": "exception"},
]

SEED_PRODUCTS = [
    {
        "name": "TechPro 无线蓝牙耳机 Pro",
        "subtitle": "降噪续航30小时 IPX5防水",
        "description": "采用最新的蓝牙5.3技术，支持主动降噪，续航长达30小时。配备充电仓，支持无线充电。",
        "brand_name": "TechPro",
        "category_name": "电子产品",
        "unit": "副",
        "specs": {"颜色": ["黑色", "白色"]},
        "skus": [
            {"spec_desc": "黑色", "price": 299.00, "cost_price": 180.00, "market_price": 399.00, "stock": 200},
            {"spec_desc": "白色", "price": 299.00, "cost_price": 180.00, "market_price": 399.00, "stock": 150},
        ],
    },
    {
        "name": "TechPro 智能手表 S3",
        "subtitle": "全天候心率血氧监测 AMOLED屏幕",
        "description": "1.5英寸AMOLED屏幕，支持心率/血氧/睡眠监测，100+运动模式，IP68防水。",
        "brand_name": "TechPro",
        "category_name": "智能穿戴",
        "unit": "块",
        "specs": {"颜色": ["午夜黑", "星光银"], "表带": ["运动款", "皮质款"]},
        "skus": [
            {"spec_desc": "午夜黑-运动款", "spec_values": {"颜色": "午夜黑", "表带": "运动款"}, "price": 599.00, "cost_price": 350.00, "market_price": 799.00, "stock": 100},
            {"spec_desc": "午夜黑-皮质款", "spec_values": {"颜色": "午夜黑", "表带": "皮质款"}, "price": 699.00, "cost_price": 400.00, "market_price": 899.00, "stock": 80},
            {"spec_desc": "星光银-运动款", "spec_values": {"颜色": "星光银", "表带": "运动款"}, "price": 599.00, "cost_price": 350.00, "market_price": 799.00, "stock": 90},
            {"spec_desc": "星光银-皮质款", "spec_values": {"颜色": "星光银", "表带": "皮质款"}, "price": 699.00, "cost_price": 400.00, "market_price": 899.00, "stock": 70},
        ],
    },
    {
        "name": "StyleWear 男士休闲夹克",
        "subtitle": "防风防泼水 轻薄便携",
        "description": "采用高密度尼龙面料，防风防泼水，轻薄可收纳。适合春秋季节日常通勤和户外旅行。",
        "brand_name": "StyleWear",
        "category_name": "男装",
        "unit": "件",
        "specs": {"颜色": ["深蓝", "卡其"], "尺码": ["M", "L", "XL"]},
        "skus": [
            {"spec_desc": "深蓝-M", "spec_values": {"颜色": "深蓝", "尺码": "M"}, "price": 259.00, "cost_price": 130.00, "market_price": 359.00, "stock": 60},
            {"spec_desc": "深蓝-L", "spec_values": {"颜色": "深蓝", "尺码": "L"}, "price": 259.00, "cost_price": 130.00, "market_price": 359.00, "stock": 80},
            {"spec_desc": "卡其-L", "spec_values": {"颜色": "卡其", "尺码": "L"}, "price": 259.00, "cost_price": 130.00, "market_price": 359.00, "stock": 70},
            {"spec_desc": "卡其-XL", "spec_values": {"颜色": "卡其", "尺码": "XL"}, "price": 279.00, "cost_price": 140.00, "market_price": 379.00, "stock": 50},
        ],
    },
    {
        "name": "NatureHome 不锈钢保温杯 500ml",
        "subtitle": "12小时保温 食品级304不锈钢",
        "description": "双层真空不锈钢杯身，12小时长效保温。食品级304不锈钢内胆，安全卫生。简约设计，多色可选。",
        "brand_name": "NatureHome",
        "category_name": "家居用品",
        "unit": "个",
        "specs": {"颜色": ["极光白", "曜石黑", "薄荷绿"]},
        "skus": [
            {"spec_desc": "极光白", "price": 89.00, "cost_price": 45.00, "market_price": 129.00, "stock": 300},
            {"spec_desc": "曜石黑", "price": 89.00, "cost_price": 45.00, "market_price": 129.00, "stock": 250},
            {"spec_desc": "薄荷绿", "price": 99.00, "cost_price": 50.00, "market_price": 139.00, "stock": 200},
        ],
    },
    {
        "name": "BeautyGlow 玻尿酸保湿精华液 30ml",
        "subtitle": "三重玻尿酸 深层补水锁水",
        "description": "含高浓度三重玻尿酸复合物，小分子深层渗透，中分子填充细纹，大分子锁水保湿。适合所有肤质。",
        "brand_name": "BeautyGlow",
        "category_name": "护肤品",
        "unit": "瓶",
        "specs": {"规格": ["30ml", "50ml"]},
        "skus": [
            {"spec_desc": "30ml", "price": 168.00, "cost_price": 80.00, "market_price": 228.00, "stock": 180},
            {"spec_desc": "50ml", "price": 238.00, "cost_price": 120.00, "market_price": 328.00, "stock": 120},
        ],
    },
    {
        "name": "FreshFood 混合坚果礼盒 1kg",
        "subtitle": "每日坚果 科学配比 新鲜直达",
        "description": "精选6种坚果果干科学配比：巴旦木、腰果、核桃、榛子、蔓越莓干、蓝莓干。独立小包装，锁住新鲜。",
        "brand_name": "FreshFood",
        "category_name": "休闲零食",
        "unit": "袋",
        "specs": {"规格": ["1kg（30包）", "500g（15包）"]},
        "skus": [
            {"spec_desc": "1kg（30包）", "price": 128.00, "cost_price": 75.00, "market_price": 168.00, "stock": 500},
            {"spec_desc": "500g（15包）", "price": 69.00, "cost_price": 40.00, "market_price": 89.00, "stock": 600},
        ],
    },
    {
        "name": "SportMax 瑜伽垫 TPE双面防滑",
        "subtitle": "加厚10mm 双面防滑 环保TPE材质",
        "description": "环保TPE材质，双面防滑纹理设计，10mm加厚缓冲。附赠收纳绑带和背包。",
        "brand_name": "SportMax",
        "category_name": "运动户外",
        "unit": "张",
        "specs": {"颜色": ["深紫", "湖蓝"], "厚度": ["6mm", "10mm"]},
        "skus": [
            {"spec_desc": "深紫-6mm", "spec_values": {"颜色": "深紫", "厚度": "6mm"}, "price": 79.00, "cost_price": 35.00, "market_price": 119.00, "stock": 400},
            {"spec_desc": "深紫-10mm", "spec_values": {"颜色": "深紫", "厚度": "10mm"}, "price": 99.00, "cost_price": 45.00, "market_price": 149.00, "stock": 350},
            {"spec_desc": "湖蓝-10mm", "spec_values": {"颜色": "湖蓝", "厚度": "10mm"}, "price": 99.00, "cost_price": 45.00, "market_price": 149.00, "stock": 300},
        ],
    },
]


# ── 辅助函数 ────────────────────────────────────────────────────────

async def get_or_create(session: AsyncSession, model, defaults: dict | None = None, **filters):
    """查找已有记录，若不存在则创建（可指定默认字段值）。"""
    stmt = select(model).filter_by(**filters)
    result = await session.execute(stmt)
    instance = result.scalar_one_or_none()
    if instance:
        return instance, False
    init_data = {**filters, **(defaults or {})}
    instance = model(**init_data)
    session.add(instance)
    await session.flush()
    return instance, True


async def seed_users(session: AsyncSession):
    """创建默认管理员账号。"""
    print("  └─ 创建管理员账号...")
    stmt = select(User).where(User.username == "admin")
    result = await session.execute(stmt)
    admin = result.scalar_one_or_none()
    if admin:
        print(f"     → 管理员已存在 (id={admin.id})，跳过")
        return
    admin = User(
        username="admin",
        password_hash=hash_password("admin123"),
        display_name="系统管理员",
        role="admin",
        email="admin@multisell.com",
        status=1,
    )
    session.add(admin)
    await session.flush()
    print(f"     ✔ 管理员 admin / admin123 已创建 (id={admin.id})")


async def seed_permissions(session: AsyncSession):
    """创建基础权限码。"""
    print("  └─ 创建权限码...")
    for item in SEED_PERMISSIONS:
        permission, created = await get_or_create(
            session,
            Permission,
            defaults={"name": item["name"], "module": item["module"]},
            code=item["code"],
        )
        if created:
            print(f"     ✔ {permission.code}")
        else:
            print(f"     → {permission.code} 已存在，跳过")


async def seed_categories(session: AsyncSession) -> dict[str, int]:
    """创建演示分类数据，返回 {分类名: id} 映射。"""
    print("  └─ 创建分类...")
    name_id_map: dict[str, int] = {}

    for name, parent_name, level, sort_order in SEED_CATEGORIES:
        cat, created = await get_or_create(
            session, Category,
            defaults=dict(level=level, sort_order=sort_order, status=1),
            name=name,
        )
        if created:
            cat.parent_id = name_id_map.get(parent_name, 0)
            await session.flush()
            print(f"     ✔ {name} (id={cat.id})")
        else:
            print(f"     → {name} 已存在 (id={cat.id})，跳过")
        name_id_map[name] = cat.id

    return name_id_map


async def seed_platforms(session: AsyncSession):
    """创建示例平台。"""
    print("  └─ 创建平台...")
    for name, code, api_base, sort_order in SEED_PLATFORMS:
        platform, created = await get_or_create(
            session, Platform,
            defaults=dict(name=name, api_base_url=api_base, sort_order=sort_order, status=1),
            code=code,
        )
        if created:
            print(f"     ✔ {name} ({code}) (id={platform.id})")
        else:
            print(f"     → {name} ({code}) 已存在 (id={platform.id})，跳过")


async def seed_brands(session: AsyncSession):
    """创建示例品牌。"""
    print("  └─ 创建品牌...")
    for name, desc in SEED_BRANDS:
        brand, created = await get_or_create(
            session, Brand,
            defaults=dict(description=desc, status=1, sort_order=0),
            name=name,
        )
        if created:
            print(f"     ✔ {name} (id={brand.id})")
        else:
            print(f"     → {name} 已存在 (id={brand.id})，跳过")


async def seed_products_and_skus(session: AsyncSession, category_ids: dict[str, int]):
    """创建演示商品和 SKU。"""
    print("  └─ 创建商品和 SKU...")

    # 品牌 name → id
    brands_map = {b.name: b.id for b in (await session.execute(select(Brand))).scalars().all()}

    for prod_data in SEED_PRODUCTS:
        brand_id = brands_map.get(prod_data["brand_name"], 0)
        cat_id = category_ids.get(prod_data["category_name"], 0)

        # 按名称去重
        existing = (await session.execute(
            select(Product).where(Product.name == prod_data["name"])
        )).scalar_one_or_none()
        if existing:
            print(f"     → 商品「{prod_data['name']}」已存在，跳过")
            continue

        product = Product(
            name=prod_data["name"],
            subtitle=prod_data.get("subtitle", ""),
            description=prod_data.get("description", ""),
            brand_id=brand_id or 0,
            category_id=cat_id,
            unit=prod_data.get("unit", "件"),
            status=1,  # 上架
        )
        session.add(product)
        await session.flush()
        print(f"     ✔ 商品「{product.name}」(id={product.id})")

        # 创建规格名称 + 规格值
        for spec_name, spec_values in prod_data.get("specs", {}).items():
            sn = SpecName(product_id=product.id, name=spec_name, sort_order=0)
            session.add(sn)
            await session.flush()
            for idx, val in enumerate(spec_values):
                sv = SpecValue(spec_name_id=sn.id, product_id=product.id, value=val, sort_order=idx)
                session.add(sv)
            await session.flush()

        # 创建 SKU + 库存
        for sku_data in prod_data["skus"]:
            spec_desc = sku_data["spec_desc"]
            spec_values = sku_data.get("spec_values", {})
            sku = Sku(
                product_id=product.id,
                code=f"{product.id}-{spec_desc[:20]}",
                spec_desc=spec_desc,
                spec_values=spec_values if spec_values else None,
                price=sku_data["price"],
                cost_price=sku_data["cost_price"],
                market_price=sku_data["market_price"],
                stock=sku_data["stock"],
                warning_stock=10,
                weight=0.30,
                status=1,
            )
            session.add(sku)
            await session.flush()

            inv = Inventory(
                sku_id=sku.id,
                warehouse="默认仓库",
                quantity=sku_data["stock"],
                safety_stock=10,
            )
            session.add(inv)
            print(f"       SKU {sku.code}: ¥{sku.price} (库存:{sku.stock})")

    await session.flush()


# ── 主流程 ──────────────────────────────────────────────────────────

async def seed():
    """执行全部数据初始化。"""
    engine = create_async_engine(settings.DATABASE_URL, echo=False)
    session_factory = async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)

    print(f"\n{'='*50}")
    print("  MultiSell 数据初始化脚本")
    print(f"{'='*50}\n")
    print(f"📦 数据库: {settings.DATABASE_URL}")
    print("🔄 连接中...")

    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    print("✔ 数据表已就绪\n")

    async with session_factory() as session:
        try:
            print("📋 第1步：创建管理员账号")
            await seed_users(session)

            print("\n📋 第2步：创建权限码")
            await seed_permissions(session)

            print("\n📋 第3步：创建分类数据")
            category_ids = await seed_categories(session)

            print("\n📋 第4步：创建平台数据")
            await seed_platforms(session)

            print("\n📋 第5步：创建品牌数据")
            await seed_brands(session)

            print("\n📋 第6步：创建商品和 SKU")
            await seed_products_and_skus(session, category_ids)

            await session.commit()
            print(f"\n{'='*50}")
            print("  ✅ 数据初始化完成！")
            print(f"{'='*50}\n")

        except Exception as e:
            await session.rollback()
            print(f"\n❌ 初始化失败: {e}")
            raise
        finally:
            await engine.dispose()


def main():
    asyncio.run(seed())


if __name__ == "__main__":
    main()
