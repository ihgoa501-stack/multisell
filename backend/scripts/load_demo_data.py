#!/usr/bin/env python3
"""
Demo sandbox data loader — Stage 13

幂等加载模拟经营数据，让凌镜 LingMirror 在没有真实数据的情况下，
也能完整演示和验证核心业务闭环：

商品 → SKU → 库存 → 物流报价 → 平台费用规则 → CSV 订单导入 →
运费账单 → 平台结算 → 利润账本 → 异常工作台 → 利润看板

用法:
    cd backend
    ./scripts/load_demo_data.py

可重复执行，不会创建重复数据。
"""

import asyncio
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from decimal import Decimal
from typing import Optional

from sqlalchemy import select, text
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker

from app.config import settings
from app.database import Base
from app.models import (
    Brand, Category, Platform, Product, SpecName, SpecValue, Sku,
    Inventory, Supplier, ShippingProvider, ShippingChannel, ShippingZone,
    ShippingQuoteRule, PlatformFeeRule, Permission,
    PlatformIntegrationAccount,
)
from app.auth.service import hash_password
from app.models import User


# ── 基础 seed 数据 ───────────────────────────────────────────────────

SEED_PERMISSIONS = [
    {"code": "product:view", "name": "查看商品", "module": "product"},
    {"code": "order:view", "name": "查看订单", "module": "order"},
    {"code": "order:create", "name": "创建订单", "module": "order"},
    {"code": "order:update", "name": "更新订单", "module": "order"},
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
    {"code": "order_import:import", "name": "导入订单", "module": "order_import"},
    {"code": "order_import:view", "name": "查看订单导入", "module": "order_import"},
    {"code": "order_import:process", "name": "处理订单导入链路", "module": "order_import"},
    {"code": "platform_fee:view", "name": "查看平台费用规则", "module": "platform_fee"},
    {"code": "platform_fee:manage", "name": "管理平台费用规则", "module": "platform_fee"},
    {"code": "platform_fee:calculate", "name": "匹配平台费用规则", "module": "platform_fee"},
    {"code": "decision:calculate", "name": "上架前决策计算", "module": "decision"},
    {"code": "listing:view", "name": "查看发布", "module": "listing"},
    {"code": "listing:task_manage", "name": "管理上架任务", "module": "listing"},
    {"code": "listing:publish", "name": "发布上架任务", "module": "listing"},
    {"code": "finance:report:view", "name": "查看财务报表", "module": "finance"},
    {"code": "dashboard:view", "name": "查看仪表盘", "module": "dashboard"},
    {"code": "platform_integration:view", "name": "查看平台集成", "module": "platform_integration"},
    {"code": "platform_integration:manage", "name": "管理平台集成", "module": "platform_integration"},
    {"code": "agent_action:propose", "name": "提议Agent动作", "module": "agent_action"},
    {"code": "agent_action:view", "name": "查看Agent动作", "module": "agent_action"},
]

DEMO_PRODUCTS = [
    {
        "name": "Demo 蓝牙耳机 Pro",
        "subtitle": "降噪 30h IPX5",
        "description": "Demo 商品 — 演示用蓝牙耳机",
        "brand_name": "TechPro",
        "category_name": "电子产品",
        "unit": "副",
        "skus": [
            {"code": "DEMO-BT-BLACK", "spec_desc": "黑色", "price": 299, "cost_price": 180, "stock": 200, "weight": 0.30},
            {"code": "DEMO-BT-WHITE", "spec_desc": "白色", "price": 299, "cost_price": 180, "stock": 150, "weight": 0.30},
        ],
    },
    {
        "name": "Demo 智能手表 S3",
        "subtitle": "心率血氧监测",
        "description": "Demo 商品 — 演示用智能手表",
        "brand_name": "TechPro",
        "category_name": "智能穿戴",
        "unit": "块",
        "skus": [
            {"code": "DEMO-WATCH-BLACK-SPORT", "spec_desc": "午夜黑-运动款", "price": 599, "cost_price": 350, "stock": 100, "weight": 0.25},
            {"code": "DEMO-WATCH-BLACK-LEATHER", "spec_desc": "午夜黑-皮质款", "price": 699, "cost_price": 400, "stock": 80, "weight": 0.28},
        ],
    },
    {
        "name": "Demo 男士休闲夹克",
        "subtitle": "防风 轻薄",
        "description": "Demo 商品 — 演示用夹克",
        "brand_name": "StyleWear",
        "category_name": "男装",
        "unit": "件",
        "skus": [
            {"code": "DEMO-JACKET-BLUE-L", "spec_desc": "深蓝-L", "price": 259, "cost_price": 130, "stock": 80, "weight": 0.50},
            {"code": "DEMO-JACKET-BLUE-XL", "spec_desc": "深蓝-XL", "price": 279, "cost_price": 140, "stock": 50, "weight": 0.55},
        ],
    },
    {
        "name": "Demo 不锈钢保温杯 500ml",
        "subtitle": "12h 保温 304 不锈钢",
        "description": "Demo 商品 — 演示用保温杯",
        "brand_name": "NatureHome",
        "category_name": "家居用品",
        "unit": "个",
        "skus": [
            {"code": "DEMO-CUP-WHITE", "spec_desc": "极光白", "price": 89, "cost_price": 45, "stock": 300, "weight": 0.35},
            {"code": "DEMO-CUP-BLACK", "spec_desc": "曜石黑", "price": 89, "cost_price": 45, "stock": 250, "weight": 0.35},
        ],
    },
    {
        "name": "Demo 玻尿酸保湿精华液 50ml",
        "subtitle": "三重玻尿酸 补水锁水",
        "description": "Demo 商品 — 演示用精华液",
        "brand_name": "BeautyGlow",
        "category_name": "护肤品",
        "unit": "瓶",
        "skus": [
            {"code": "DEMO-BTL-30ML", "spec_desc": "30ml", "price": 168, "cost_price": 80, "stock": 180, "weight": 0.12},
            {"code": "DEMO-BTL-50ML", "spec_desc": "50ml", "price": 238, "cost_price": 120, "stock": 120, "weight": 0.15},
        ],
    },
    {
        "name": "Demo 混合坚果礼盒 1kg",
        "subtitle": "6种坚果果干 科学配比",
        "description": "Demo 商品 — 演示用坚果礼盒",
        "brand_name": "FreshFood",
        "category_name": "休闲零食",
        "unit": "袋",
        "skus": [
            {"code": "DEMO-NUT-1KG", "spec_desc": "1kg", "price": 128, "cost_price": 75, "stock": 500, "weight": 1.0},
            {"code": "DEMO-NUT-500G", "spec_desc": "500g", "price": 69, "cost_price": 40, "stock": 600, "weight": 0.5},
        ],
    },
    {
        "name": "Demo 瑜伽垫 TPE双面防滑",
        "subtitle": "10mm 加厚",
        "description": "Demo 商品 — 演示用瑜伽垫",
        "brand_name": "SportMax",
        "category_name": "运动户外",
        "unit": "张",
        "skus": [
            {"code": "DEMO-YOGA-MAT-PURPLE-10", "spec_desc": "深紫-10mm", "price": 99, "cost_price": 45, "stock": 350, "weight": 0.8},
            {"code": "DEMO-YOGA-MAT-BLUE-10", "spec_desc": "湖蓝-10mm", "price": 99, "cost_price": 45, "stock": 300, "weight": 0.8},
        ],
    },
]


class DemoDataSummary:
    """Demo seed 执行摘要"""
    def __init__(self):
        self.products_created = 0
        self.products_updated = 0
        self.skus_created = 0
        self.skus_updated = 0
        self.inventory_seeded = 0
        self.shipping_rules_seeded = 0
        self.platform_fee_rules_seeded = 0
        self.csv_paths = []

    def __str__(self):
        return (
            f"products created/updated: {self.products_created}/{self.products_updated}\n"
            f"skus created/updated: {self.skus_created}/{self.skus_updated}\n"
            f"inventory seeded: {self.inventory_seeded}\n"
            f"shipping rules seeded: {self.shipping_rules_seeded}\n"
            f"platform fee rules seeded: {self.platform_fee_rules_seeded}\n"
            f"demo csv paths: {', '.join(self.csv_paths)}"
        )


# ── 辅助函数 ────────────────────────────────────────────────────────

async def get_or_create(session: AsyncSession, model, defaults: dict | None = None, **filters):
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


async def ensure_admin_user(session: AsyncSession):
    """确保 admin 用户存在"""
    stmt = select(User).where(User.username == "admin")
    result = await session.execute(stmt)
    admin = result.scalar_one_or_none()
    if admin:
        return admin
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
    return admin


async def ensure_demo_user(session: AsyncSession):
    """创建 demo 用户"""
    user, created = await get_or_create(
        session, User,
        defaults=dict(
            password_hash=hash_password("demo123"),
            display_name="Demo 演示用户",
            role="user",
            email="demo@multisell.com",
            status=1,
        ),
        username="demo",
    )
    return user, created


async def ensure_permissions(session: AsyncSession):
    """创建必要的权限码"""
    count = 0
    for item in SEED_PERMISSIONS:
        _, created = await get_or_create(
            session, Permission,
            defaults={"name": item["name"], "module": item["module"]},
            code=item["code"],
        )
        if created:
            count += 1
    return count


async def ensure_platforms(session: AsyncSession):
    """创建演示平台"""
    platforms_data = [
        ("Ozon", "ozon", "https://api.ozon.ru", 1),
        ("Shopee", "shopee", "https://api.shopee.com", 2),
        ("Wildberries", "wb", "https://api.wildberries.ru", 3),
    ]
    created_ids = []
    for name, code, api_base, sort_order in platforms_data:
        plat, created = await get_or_create(
            session, Platform,
            defaults=dict(name=name, api_base_url=api_base, sort_order=sort_order, status=1),
            code=code,
        )
        created_ids.append(plat.id)
    return created_ids


async def ensure_categories(session: AsyncSession):
    """创建演示分类"""
    cats = [
        ("电子产品", 0, 0, 1),
        ("智能穿戴", 0, 0, 2),
        ("男装", 0, 0, 3),
        ("家居用品", 0, 0, 4),
        ("护肤品", 0, 0, 5),
        ("休闲零食", 0, 0, 6),
        ("运动户外", 0, 0, 7),
    ]
    name_id_map = {}
    for name, parent_id, level, sort in cats:
        cat, _ = await get_or_create(
            session, Category,
            defaults=dict(parent_id=parent_id, level=level, sort_order=sort, status=1),
            name=name,
        )
        name_id_map[name] = cat.id
    return name_id_map


async def ensure_brands(session: AsyncSession):
    """创建演示品牌"""
    brands = [
        "TechPro", "NatureHome", "StyleWear", "BeautyGlow", "FreshFood", "SportMax",
    ]
    for name in brands:
        await get_or_create(session, Brand, defaults=dict(status=1, sort_order=0), name=name)


async def ensure_suppliers(session: AsyncSession):
    """创建演示供应商"""
    suppliers_data = [
        ("深圳华强电子有限公司", "supplier_wireless"),
        ("广州服装制造厂", "supplier_garment"),
        ("义乌家居日用厂", "supplier_home"),
    ]
    supplier_ids = {}
    for name, code in suppliers_data:
        sup, created = await get_or_create(
            session, Supplier,
            defaults=dict(status=1),
            name=name,
        )
        supplier_ids[name] = sup.id
    return supplier_ids


async def ensure_shipping_providers_and_rules(session: AsyncSession, summary: DemoDataSummary):
    """创建物流供应商、渠道、区域和报价规则"""
    # 供应商 CDEK
    cdek, _ = await get_or_create(
        session, ShippingProvider,
        defaults=dict(code="CDEK", status=1, remark="Demo CDEK 物流"),
        name="CDEK",
    )
    # 俄语速运
    rus_post, _ = await get_or_create(
        session, ShippingProvider,
        defaults=dict(code="RUS_POST", status=1, remark="Demo 俄罗斯邮政"),
        name="Russian Post",
    )
    # Shopee Xpress
    shx, _ = await get_or_create(
        session, ShippingProvider,
        defaults=dict(code="SHX", status=1, remark="Demo Shopee Xpress"),
        name="Shopee Xpress",
    )

    # 渠道
    cdek_eco, _ = await get_or_create(
        session, ShippingChannel,
        defaults=dict(
            provider_id=cdek.id, code="CDEK_ECO", volumetric_divisor=6000,
            cargo_types=["normal"], estimated_delivery_min=10, estimated_delivery_max=20,
            currency="CNY", status=1,
        ),
        name="CDEK Economy",
    )
    cdek_std, _ = await get_or_create(
        session, ShippingChannel,
        defaults=dict(
            provider_id=cdek.id, code="CDEK_STD", volumetric_divisor=6000,
            cargo_types=["normal"], estimated_delivery_min=7, estimated_delivery_max=14,
            currency="CNY", status=1,
        ),
        name="CDEK Standard",
    )
    rus_ems, _ = await get_or_create(
        session, ShippingChannel,
        defaults=dict(
            provider_id=rus_post.id, code="RUS_EMS", volumetric_divisor=6000,
            cargo_types=["normal"], estimated_delivery_min=14, estimated_delivery_max=28,
            currency="CNY", status=1,
        ),
        name="EMS",
    )
    shx_std, _ = await get_or_create(
        session, ShippingChannel,
        defaults=dict(
            provider_id=shx.id, code="SHX_STD", volumetric_divisor=6000,
            cargo_types=["normal"], estimated_delivery_min=5, estimated_delivery_max=10,
            currency="SGD", status=1,
        ),
        name="Standard",
    )

    # 区域：RU
    ru_zone, _ = await get_or_create(
        session, ShippingZone,
        defaults=dict(country_code="RU", status=1),
        channel_id=cdek_eco.id,
    )
    ru_zone2, _ = await get_or_create(
        session, ShippingZone,
        defaults=dict(country_code="RU", status=1),
        channel_id=cdek_std.id,
    )
    ru_zone3, _ = await get_or_create(
        session, ShippingZone,
        defaults=dict(country_code="RU", status=1),
        channel_id=rus_ems.id,
    )
    sg_zone, _ = await get_or_create(
        session, ShippingZone,
        defaults=dict(country_code="SG", status=1),
        channel_id=shx_std.id,
    )

    # 报价规则
    rules_specs = [
        (cdek_eco.id, ru_zone.id, "fixed_plus_per_kg", 1, Decimal("20"), Decimal("2")),
        (cdek_std.id, ru_zone2.id, "fixed_plus_per_kg", 1, Decimal("25"), Decimal("3")),
        (rus_ems.id, ru_zone3.id, "fixed_plus_per_kg", 1, Decimal("15"), Decimal("1.5")),
        (shx_std.id, sg_zone.id, "first_weight_plus_increment", 1, Decimal("3"), 0),
    ]

    for ch_id, z_id, rtype, pri, base, per_kg in rules_specs:
        # first_weight_plus_increment needs first_kg
        defaults = dict(
            rule_type=rtype, priority=pri, first_kg=Decimal("0.5"),
            first_price=base, per_kg_price=per_kg, status=1,
        )
        # Use very specific filter: we need a unique identifier
        existing = await session.execute(
            select(ShippingQuoteRule).where(
                ShippingQuoteRule.channel_id == ch_id,
                ShippingQuoteRule.zone_id == z_id,
                ShippingQuoteRule.rule_type == rtype,
            )
        )
        if not existing.scalar_one_or_none():
            rule = ShippingQuoteRule(
                channel_id=ch_id, zone_id=z_id, **defaults
            )
            session.add(rule)
            summary.shipping_rules_seeded += 1

    await session.flush()
    return cdek, rus_post, shx


async def ensure_platform_fee_rules(session: AsyncSession, summary: DemoDataSummary):
    """创建平台费用规则"""
    # 获取 platform IDs
    platforms = {p.name: p.id for p in (await session.execute(select(Platform))).scalars().all()}

    rules = [
        (platforms.get("Ozon"), None, "RU", Decimal("10"), Decimal("1"), Decimal("0")),
        (platforms.get("Ozon"), None, None, Decimal("12"), Decimal("1"), Decimal("0")),
        (platforms.get("Shopee"), None, "SG", Decimal("8"), Decimal("2"), Decimal("0")),
        (platforms.get("Shopee"), None, None, Decimal("10"), Decimal("2"), Decimal("0")),
        (platforms.get("Wildberries"), None, "RU", Decimal("15"), Decimal("1"), Decimal("0")),
        (platforms.get("Wildberries"), None, None, Decimal("15"), Decimal("1"), Decimal("0")),
    ]

    for pid, cat_id, site, comm, pay, fixed in rules:
        if pid is None:
            continue
        existing = await session.execute(
            select(PlatformFeeRule).where(
                PlatformFeeRule.platform_id == pid,
                PlatformFeeRule.site_code == site,
            )
        )
        if existing.scalar_one_or_none():
            continue
        rule = PlatformFeeRule(
            platform_id=pid,
            category_id=cat_id,
            site_code=site,
            commission_pct=comm,
            payment_fee_pct=pay,
            fixed_fee=fixed,
            priority=10,
            status=1,
            remark="Demo rule",
        )
        session.add(rule)
        summary.platform_fee_rules_seeded += 1
    await session.flush()


async def ensure_csv_order_adapter(session: AsyncSession):
    """创建 csv_order adapter 配置"""
    platforms = await session.execute(select(Platform))
    platforms = platforms.scalars().all()
    for plat in platforms:
        existing = await session.execute(
            select(PlatformIntegrationAccount).where(
                PlatformIntegrationAccount.platform_id == plat.id,
                PlatformIntegrationAccount.adapter_code == "csv_order",
            )
        )
        if not existing.scalar_one_or_none():
            account = PlatformIntegrationAccount(
                platform_id=plat.id,
                adapter_code="csv_order",
                account_name=f"{plat.name} CSV Order Import",
                status="active",
                credential_metadata=[],
                created_by="demo_seed",
            )
            session.add(account)
    await session.flush()


async def ensure_demo_products_and_skus(session: AsyncSession, summary: DemoDataSummary):
    """创建 demo 商品和 SKU"""
    brands_map = {b.name: b.id for b in (await session.execute(select(Brand))).scalars().all()}
    categories_map = {c.name: c.id for c in (await session.execute(select(Category))).scalars().all()}

    for prod_data in DEMO_PRODUCTS:
        brand_id = brands_map.get(prod_data["brand_name"], 0)
        cat_id = categories_map.get(prod_data["category_name"], 0)

        existing = (await session.execute(
            select(Product).where(Product.name == prod_data["name"])
        )).scalar_one_or_none()
        if existing:
            product = existing
        else:
            product = Product(
                name=prod_data["name"],
                subtitle=prod_data.get("subtitle", ""),
                description=prod_data.get("description", ""),
                brand_id=brand_id or 0,
                category_id=cat_id,
                unit=prod_data.get("unit", "件"),
                status=1,
            )
            session.add(product)
            await session.flush()
            summary.products_created += 1

        for sku_data in prod_data["skus"]:
            existing_sku = (await session.execute(
                select(Sku).where(Sku.code == sku_data["code"])
            )).scalar_one_or_none()
            if existing_sku:
                sku = existing_sku
            else:
                sku = Sku(
                    product_id=product.id,
                    code=sku_data["code"],
                    spec_desc=sku_data["spec_desc"],
                    price=sku_data["price"],
                    cost_price=sku_data["cost_price"],
                    stock=sku_data["stock"],
                    warning_stock=10,
                    weight=sku_data.get("weight", 0.30),
                    status=1,
                )
                session.add(sku)
                await session.flush()
                summary.skus_created += 1

            # 库存
            existing_inv = (await session.execute(
                select(Inventory).where(Inventory.sku_id == sku.id)
            )).scalar_one_or_none()
            if not existing_inv:
                inv = Inventory(
                    sku_id=sku.id,
                    warehouse="默认仓库",
                    quantity=sku_data["stock"],
                    safety_stock=10,
                )
                session.add(inv)
                summary.inventory_seeded += 1

    await session.flush()


# ── 主函数 ──────────────────────────────────────────────────────────

async def load_demo_data(session: AsyncSession) -> DemoDataSummary:
    """加载全部 demo 数据。幂等可重复执行。"""
    summary = DemoDataSummary()

    print("  ├─ 确保 admin 用户...")
    await ensure_admin_user(session)

    print("  ├─ 确保 demo 用户...")
    demo_user, demo_created = await ensure_demo_user(session)
    if demo_created:
        print("  │   ✔ demo / demo123 已创建")
    else:
        print("  │   → demo 用户已存在")

    print("  ├─ 确保权限码...")
    perm_count = await ensure_permissions(session)
    print(f"  │   {'✔ 新增' if perm_count else '→ 已存在'} {perm_count} 权限码")

    print("  ├─ 确保分类...")
    await ensure_categories(session)

    print("  ├─ 确保平台...")
    await ensure_platforms(session)

    print("  ├─ 确保品牌...")
    await ensure_brands(session)

    print("  ├─ 确保供应商...")
    await ensure_suppliers(session)

    print("  ├─ 确保平台费用规则...")
    await ensure_platform_fee_rules(session, summary)

    print("  ├─ 确保物流供应商/渠道/报价规则...")
    await ensure_shipping_providers_and_rules(session, summary)

    print("  ├─ 确保 csv_order adapter...")
    await ensure_csv_order_adapter(session)

    print("  ├─ 确保 demo 商品和 SKU...")
    await ensure_demo_products_and_skus(session, summary)

    # demo CSV 路径
    repo_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    demo_dir = os.path.join(repo_root, "docs", "demo-data")
    summary.csv_paths = [
        os.path.join(demo_dir, "order_import_demo.csv"),
        os.path.join(demo_dir, "shipping_bill_demo.csv"),
        os.path.join(demo_dir, "platform_settlement_demo.csv"),
    ]

    return summary


async def main():
    """CLI 入口"""
    engine = create_async_engine(settings.DATABASE_URL, echo=False)
    session_factory = async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)

    print(f"\n{'='*55}")
    print("  凌镜 LingMirror Demo 数据加载")
    print(f"{'='*55}\n")
    print(f"📦 数据库: {settings.DATABASE_URL}")
    print("🔄 连接中...")

    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    print("✔ 数据表已就绪\n")

    async with session_factory() as session:
        try:
            print("📋 加载 demo 数据...")
            summary = await load_demo_data(session)
            await session.commit()

            print(f"\n{'='*55}")
            print("  ✅ Demo 数据加载完成！")
            print(f"{'='*55}")
            print()
            print(summary)

        except Exception as e:
            await session.rollback()
            print(f"\n❌ 加载失败: {e}")
            raise
        finally:
            await engine.dispose()


if __name__ == "__main__":
    asyncio.run(main())
