"""Demo seed 功能测试 — 不依赖 HTTP client，直接测试 seed 脚本和 CSV 文件。"""

import csv
import os

import pytest
from sqlalchemy import select

from app.database import async_session_factory
from app.models import (
    Product, Sku, Inventory,
    ShippingProvider, ShippingChannel, ShippingQuoteRule,
    PlatformFeeRule, PlatformIntegrationAccount, User, Permission,
)
from app.order_import.service import OrderImportService
from app.shipping.bill_service import ShippingBillService
from app.settlement.service import SettlementService


# ── CSV 解析测试（无需数据库） ─────────────────────────────────────

class TestDemoCsvParseable:
    """验证 demo CSV 文件可被现有 parser 解析。"""

    @staticmethod
    def _read_bytes(rel_path: str) -> bytes:
        demo_dir = os.path.join(os.path.dirname(__file__), "..", "..", "docs", "demo-data")
        with open(os.path.join(demo_dir, rel_path), "rb") as f:
            return f.read()

    def test_order_import_csv_parseable(self):
        content = self._read_bytes("order_import_demo.csv")
        result = OrderImportService.parse_csv(content, source_filename="order_import_demo.csv")
        assert len(result["rows"]) == 7
        assert result["platform"] == "Ozon"

    def test_order_import_csv_has_multi_sku_same_order(self):
        content = self._read_bytes("order_import_demo.csv")
        result = OrderImportService.parse_csv(content)
        count = sum(1 for r in result["rows"] if r["platform_order_no"] == "OZ-10001")
        assert count == 2

    def test_shipping_bill_csv_parseable(self):
        rows, errors = ShippingBillService.parse_csv(self._read_bytes("shipping_bill_demo.csv"))
        assert len(rows) == 5
        assert len(errors) == 0

    def test_shipping_bill_has_matched_and_unmatched(self):
        rows, _ = ShippingBillService.parse_csv(self._read_bytes("shipping_bill_demo.csv"))
        unmatched = [r for r in rows if r.get("tracking_number") and "UNMATCHED" in r["tracking_number"]]
        matched = [r for r in rows if r.get("tracking_number") and "UNMATCHED" not in r["tracking_number"]]
        matched_by_order = [r for r in rows if not r.get("tracking_number") and r.get("order_no")]
        assert len(unmatched) >= 1
        assert len(matched) >= 1 or len(matched_by_order) >= 1

    def test_settlement_csv_parseable(self):
        rows, errors = SettlementService.parse_csv(self._read_bytes("platform_settlement_demo.csv"))
        assert len(rows) > 0
        assert len(errors) == 0
        tx_types = {r["transaction_type"] for r in rows}
        assert "sale" in tx_types
        assert "platform_fee" in tx_types

    def test_settlement_has_matched_and_unmatched(self):
        """settlement demo 包含 matched 行和 unmatched 行。
        Unmatched 行的特征是 platform 包含 UNMATCHED 标识且无 platform_order_no。
        """
        rows, _ = SettlementService.parse_csv(self._read_bytes("platform_settlement_demo.csv"))
        # Unmatched: platform 名包含 UNMATCHED（因为这行没有 platform_order_no 和真实 order_no）
        unmatched = [r for r in rows if r.get("platform") and "UNMATCHED" in r["platform"].upper()]
        # Matched: 有 platform_order_no 的行
        matched = [r for r in rows if r.get("platform_order_no") or (r.get("order_no") and "UNMATCHED" not in (r.get("order_no") or "").upper())]
        assert len(unmatched) >= 1, f"期望 ≥1 unmatched, 实际 {len(unmatched)}"
        assert len(matched) >= 1, f"期望 ≥1 matched, 实际 {len(matched)}"


# ── 数据库测试（依赖 conftest prepare_db → 通过 async_client 触发） ──

class TestDemoSeed:
    """验证 demo seed 可导入且幂等。"""

    async def test_module_importable(self, async_client):
        import scripts.load_demo_data as mod
        assert hasattr(mod, "load_demo_data")

    async def test_idempotent(self, async_client):
        from scripts.load_demo_data import load_demo_data
        async with async_session_factory() as session:
            await load_demo_data(session)
            await session.commit()
            skus1 = (await session.execute(select(Sku))).scalars().all()
            count1 = len(skus1)

            await load_demo_data(session)
            await session.commit()
            skus2 = (await session.execute(select(Sku))).scalars().all()
            count2 = len(skus2)

        assert count1 == count2, f"重复执行不应新增 SKU: {count1} → {count2}"

    async def test_sku_codes_match_csv(self, async_client):
        demo_dir = os.path.join(os.path.dirname(__file__), "..", "..", "docs", "demo-data")
        with open(os.path.join(demo_dir, "order_import_demo.csv"), encoding="utf-8-sig") as f:
            csv_codes = {row["sku_code"].strip() for row in csv.DictReader(f) if row.get("sku_code", "").strip()}
        from scripts.load_demo_data import load_demo_data
        async with async_session_factory() as session:
            await load_demo_data(session)
            await session.commit()
            result = await session.execute(select(Sku).where(Sku.code.in_(csv_codes)))
            found = {s.code for s in result.scalars().all()}
        assert not (csv_codes - found)
        assert len(found) == len(csv_codes)


class TestDemoSeedCreatesData:
    """验证 demo seed 创建了所需实体。"""

    @staticmethod
    async def _load():
        from scripts.load_demo_data import load_demo_data
        async with async_session_factory() as session:
            await load_demo_data(session)
            await session.commit()

    async def test_users(self, async_client):
        await self._load()
        async with async_session_factory() as session:
            assert (await session.execute(select(User).where(User.username == "admin"))).scalar_one_or_none()
            assert (await session.execute(select(User).where(User.username == "demo"))).scalar_one_or_none()

    async def test_products(self, async_client):
        await self._load()
        async with async_session_factory() as session:
            prods = (await session.execute(select(Product).where(Product.name.like("Demo %")))).scalars().all()
            assert len(prods) >= 5

    async def test_skus(self, async_client):
        await self._load()
        async with async_session_factory() as session:
            for code in ("DEMO-BT-BLACK", "DEMO-BT-WHITE", "DEMO-NUT-1KG"):
                sku = (await session.execute(select(Sku).where(Sku.code == code))).scalar_one_or_none()
                assert sku
                assert sku.price and sku.cost_price

    async def test_inventory(self, async_client):
        await self._load()
        async with async_session_factory() as session:
            invs = (await session.execute(select(Inventory))).scalars().all()
            assert len(invs) >= 10

    async def test_shipping(self, async_client):
        await self._load()
        async with async_session_factory() as session:
            assert len((await session.execute(select(ShippingProvider))).scalars().all()) >= 2
            assert len((await session.execute(select(ShippingChannel))).scalars().all()) >= 3
            assert len((await session.execute(select(ShippingQuoteRule))).scalars().all()) >= 3

    async def test_platform_fee_rules(self, async_client):
        await self._load()
        async with async_session_factory() as session:
            assert len((await session.execute(select(PlatformFeeRule))).scalars().all()) >= 1

    async def test_csv_order_adapter(self, async_client):
        await self._load()
        async with async_session_factory() as session:
            accounts = (await session.execute(
                select(PlatformIntegrationAccount).where(PlatformIntegrationAccount.adapter_code == "csv_order")
            )).scalars().all()
            assert len(accounts) >= 1

    async def test_permissions(self, async_client):
        await self._load()
        async with async_session_factory() as session:
            perms = (await session.execute(
                select(Permission).where(Permission.code.like("order_import:%"))
            )).scalars().all()
            assert len(perms) >= 3
