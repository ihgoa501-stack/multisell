"""物流供应商/渠道/区域/规则管理 API 测试"""

import pytest
from uuid import uuid4
from app.config import settings

from tests.auth_helpers import register_and_login, grant_permission


pytestmark = [pytest.mark.asyncio]


# ── Provider CRUD ────────────────────────────────────────────────────────


def _uc(name: str) -> str:
    """返回唯一编码。"""
    return f"{name}_{uuid4().hex[:6]}"


class TestProviderCRUD:
    """供应商 CRUD"""

    async def test_create_provider(self, async_client):
        """创建物流供应商"""
        resp = await async_client.post(
            "/api/shipping/providers",
            json={
                "name": "云途物流",
                "code": _uc("yuntu"),
                "contact": "张三",
                "phone": "13800000001",
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["name"] == "云途物流"
        assert data["data"]["code"].startswith("yuntu_")

    async def test_list_providers(self, async_client):
        """列表物流供应商"""
        await async_client.post(
            "/api/shipping/providers", json={"name": "A", "code": _uc("a")}
        )
        await async_client.post(
            "/api/shipping/providers", json={"name": "B", "code": _uc("b")}
        )
        resp = await async_client.get("/api/shipping/providers")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert len(data["data"]) >= 2

    async def test_update_provider(self, async_client):
        """更新物流供应商"""
        resp = await async_client.post(
            "/api/shipping/providers", json={"name": "原名称", "code": _uc("orig")}
        )
        pid = resp.json()["data"]["id"]
        resp = await async_client.put(
            f"/api/shipping/providers/{pid}",
            json={
                "name": "新名称",
                "contact": "李四",
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["data"]["name"] == "新名称"
        assert data["data"]["contact"] == "李四"

    async def test_delete_provider(self, async_client):
        """删除物流供应商"""
        resp = await async_client.post(
            "/api/shipping/providers", json={"name": "待删除", "code": _uc("del")}
        )
        pid = resp.json()["data"]["id"]
        resp = await async_client.delete(f"/api/shipping/providers/{pid}")
        assert resp.status_code == 200
        # 再次获取应已禁用
        resp = await async_client.get("/api/shipping/providers")
        updated = [p for p in resp.json()["data"] if p["id"] == pid]
        assert len(updated) > 0
        assert updated[0]["status"] == 0


class TestChannelCRUD:
    """渠道 CRUD"""

    async def _create_provider(self, async_client, name="测试供应商"):
        code = _uc("test_prov")
        resp = await async_client.post(
            "/api/shipping/providers", json={"name": name, "code": code}
        )
        return resp.json()["data"]["id"]

    async def test_create_channel(self, async_client):
        pid = await self._create_provider(async_client)
        resp = await async_client.post(
            "/api/shipping/channels",
            json={
                "provider_id": pid,
                "name": "美国专线",
                "code": _uc("us_line"),
                "volumetric_divisor": 6000,
                "cargo_types": ["normal", "battery"],
                "estimated_delivery_min": 7,
                "estimated_delivery_max": 15,
                "currency": "CNY",
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["name"] == "美国专线"

    async def test_list_channels(self, async_client):
        pid = await self._create_provider(async_client)
        await async_client.post(
            "/api/shipping/channels",
            json={
                "provider_id": pid,
                "name": "渠道A",
                "code": _uc("cha"),
                "volumetric_divisor": 6000,
                "cargo_types": ["normal"],
            },
        )
        resp = await async_client.get(f"/api/shipping/channels?provider_id={pid}")
        assert resp.status_code == 200
        assert len(resp.json()["data"]) >= 1

    async def test_update_channel(self, async_client):
        pid = await self._create_provider(async_client)
        resp = await async_client.post(
            "/api/shipping/channels",
            json={
                "provider_id": pid,
                "name": "原名",
                "code": _uc("orig_c"),
                "volumetric_divisor": 6000,
                "cargo_types": ["normal"],
            },
        )
        cid = resp.json()["data"]["id"]
        resp = await async_client.put(
            f"/api/shipping/channels/{cid}",
            json={
                "name": "新渠道名",
                "volumetric_divisor": 5000,
            },
        )
        assert resp.status_code == 200
        assert resp.json()["data"]["name"] == "新渠道名"
        assert resp.json()["data"]["volumetric_divisor"] == 5000

    async def test_delete_channel(self, async_client):
        pid = await self._create_provider(async_client)
        resp = await async_client.post(
            "/api/shipping/channels",
            json={
                "provider_id": pid,
                "name": "待删渠道",
                "code": _uc("del_c"),
                "volumetric_divisor": 6000,
                "cargo_types": ["normal"],
            },
        )
        cid = resp.json()["data"]["id"]
        resp = await async_client.delete(f"/api/shipping/channels/{cid}")
        assert resp.status_code == 200


class TestZoneCRUD:
    """区域 CRUD"""

    async def _create_channel(self, async_client):
        resp = await async_client.post(
            "/api/shipping/providers", json={"name": "P", "code": _uc("p_zone")}
        )
        pid = resp.json()["data"]["id"]
        resp = await async_client.post(
            "/api/shipping/channels",
            json={
                "provider_id": pid,
                "name": "C",
                "code": _uc("c_zone"),
                "volumetric_divisor": 6000,
                "cargo_types": ["normal"],
            },
        )
        return resp.json()["data"]["id"]

    async def test_create_zone(self, async_client):
        cid = await self._create_channel(async_client)
        resp = await async_client.post(
            f"/api/shipping/channels/{cid}/zones",
            json={
                "country_code": "US",
                "postal_code_from": "10000",
                "postal_code_to": "99999",
            },
        )
        assert resp.status_code == 200

    async def test_list_zones(self, async_client):
        cid = await self._create_channel(async_client)
        await async_client.post(
            f"/api/shipping/channels/{cid}/zones", json={"country_code": "US"}
        )
        resp = await async_client.get(f"/api/shipping/channels/{cid}/zones")
        assert resp.status_code == 200
        assert len(resp.json()["data"]) >= 1

    async def test_delete_zone(self, async_client):
        cid = await self._create_channel(async_client)
        resp = await async_client.post(
            f"/api/shipping/channels/{cid}/zones", json={"country_code": "DE"}
        )
        zid = resp.json()["data"]["id"]
        resp = await async_client.delete(f"/api/shipping/zones/{zid}")
        assert resp.status_code == 200


class TestRuleCRUD:
    """报价规则 CRUD"""

    async def _create_channel(self, async_client):
        resp = await async_client.post(
            "/api/shipping/providers", json={"name": "P2", "code": _uc("p_rule")}
        )
        pid = resp.json()["data"]["id"]
        resp = await async_client.post(
            "/api/shipping/channels",
            json={
                "provider_id": pid,
                "name": "C2",
                "code": _uc("c_rule"),
                "volumetric_divisor": 6000,
                "cargo_types": ["normal"],
            },
        )
        return resp.json()["data"]["id"]

    async def test_create_rule(self, async_client):
        cid = await self._create_channel(async_client)
        resp = await async_client.post(
            f"/api/shipping/channels/{cid}/rules",
            json={
                "rule_type": "fixed_plus_per_kg",
                "fixed_fee": 8,
                "per_kg_price": 42,
                "minimum_charge": 25,
                "rounding_increment": 0.1,
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["rule_type"] == "fixed_plus_per_kg"

    async def test_list_rules(self, async_client):
        cid = await self._create_channel(async_client)
        await async_client.post(
            f"/api/shipping/channels/{cid}/rules",
            json={
                "rule_type": "fixed_plus_per_kg",
                "fixed_fee": 8,
                "per_kg_price": 42,
            },
        )
        resp = await async_client.get(f"/api/shipping/channels/{cid}/rules")
        assert resp.status_code == 200
        assert len(resp.json()["data"]) >= 1

    async def test_update_rule(self, async_client):
        cid = await self._create_channel(async_client)
        resp = await async_client.post(
            f"/api/shipping/channels/{cid}/rules",
            json={
                "rule_type": "tiered_weight",
                "tier_config": [{"min_kg": 0, "max_kg": 1, "price": 50}],
            },
        )
        rid = resp.json()["data"]["id"]
        resp = await async_client.put(
            f"/api/shipping/rules/{rid}",
            json={
                "rule_type": "tiered_weight",
                "tier_config": [{"min_kg": 0, "max_kg": 2, "price": 80}],
            },
        )
        assert resp.status_code == 200

    async def test_delete_rule(self, async_client):
        cid = await self._create_channel(async_client)
        resp = await async_client.post(
            f"/api/shipping/channels/{cid}/rules",
            json={
                "rule_type": "fixed_plus_per_kg",
                "fixed_fee": 5,
                "per_kg_price": 30,
            },
        )
        rid = resp.json()["data"]["id"]
        resp = await async_client.delete(f"/api/shipping/rules/{rid}")
        assert resp.status_code == 200


class TestPermissions:
    """权限测试（AUTH_ENABLED=True）"""

    @pytest.fixture(autouse=True)
    def _enable_auth(self):
        original = settings.AUTH_ENABLED
        settings.AUTH_ENABLED = True
        yield
        settings.AUTH_ENABLED = original

    async def test_no_token_returns_401(self, async_client):
        """未登录返回 401"""
        resp = await async_client.post(
            "/api/shipping/providers", json={"name": "X", "code": _uc("x")}
        )
        assert resp.status_code == 401

    async def test_no_permission_returns_403(self, async_client):
        """无权限返回 403"""
        _, token = await register_and_login(async_client, "noperm")
        headers = {"Authorization": f"Bearer {token}"}
        resp = await async_client.post(
            "/api/shipping/providers",
            json={"name": "X", "code": _uc("x")},
            headers=headers,
        )
        assert resp.status_code == 403

    async def test_granted_manage_can_write(self, async_client):
        """有 shipping:manage 权限可写"""
        uid, token = await register_and_login(async_client, "can_manage")
        await grant_permission(uid, "shipping:manage")
        headers = {"Authorization": f"Bearer {token}"}
        resp = await async_client.post(
            "/api/shipping/providers",
            json={"name": "可管理", "code": _uc("mgmt")},
            headers=headers,
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200

    async def test_granted_view_can_read(self, async_client):
        """有 shipping:view 权限可读"""
        uid, token = await register_and_login(async_client, "can_view")
        await grant_permission(uid, "shipping:view")
        headers = {"Authorization": f"Bearer {token}"}
        resp = await async_client.get("/api/shipping/providers", headers=headers)
        assert resp.status_code == 200

    async def test_view_cannot_write(self, async_client):
        """仅有 view 权限不能写"""
        uid, token = await register_and_login(async_client, "view_only")
        await grant_permission(uid, "shipping:view")
        headers = {"Authorization": f"Bearer {token}"}
        resp = await async_client.post(
            "/api/shipping/providers",
            json={"name": "X", "code": _uc("x")},
            headers=headers,
        )
        assert resp.status_code == 403

    async def test_calculate_requires_permission(self, async_client):
        """计算接口需要 shipping:calculate 权限"""
        original = settings.AUTH_ENABLED
        settings.AUTH_ENABLED = False
        try:
            resp = await async_client.post(
                "/api/products",
                json={
                    "name": "鉴权计算测试",
                    "package_length_cm": 10,
                    "package_width_cm": 10,
                    "package_height_cm": 10,
                    "package_weight_kg": 0.5,
                    "cargo_type": "normal",
                },
            )
            assert resp.status_code == 200
            pid = resp.json()["data"]["id"]
            resp = await async_client.post(
                f"/api/products/{pid}/specs",
                json={"specs": [{"name": "颜色", "values": ["红"]}]},
            )
            assert resp.status_code == 200
            resp = await async_client.post(f"/api/products/{pid}/skus/generate")
            assert resp.status_code == 200
            sid = resp.json()["data"]["skus"][0]["id"]

            resp = await async_client.post(
                "/api/shipping/providers",
                json={
                    "name": "鉴权物流商",
                    "code": _uc("calc_provider"),
                },
            )
            assert resp.status_code == 200
            provider_id = resp.json()["data"]["id"]
            resp = await async_client.post(
                "/api/shipping/channels",
                json={
                    "provider_id": provider_id,
                    "name": "鉴权测试渠道",
                    "code": _uc("calc_channel"),
                    "volumetric_divisor": 6000,
                    "cargo_types": ["normal"],
                },
            )
            assert resp.status_code == 200
            channel_id = resp.json()["data"]["id"]
            resp = await async_client.post(
                f"/api/shipping/channels/{channel_id}/zones",
                json={"country_code": "US"},
            )
            assert resp.status_code == 200
            resp = await async_client.post(
                f"/api/shipping/channels/{channel_id}/rules",
                json={
                    "rule_type": "fixed_plus_per_kg",
                    "fixed_fee": 8,
                    "per_kg_price": 42,
                    "minimum_charge": 25,
                    "rounding_increment": 0.1,
                },
            )
            assert resp.status_code == 200
        finally:
            settings.AUTH_ENABLED = original

        uid, token = await register_and_login(async_client, "calc_user")
        await grant_permission(uid, "shipping:calculate")
        headers = {"Authorization": f"Bearer {token}"}

        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": sid,
                "quantity": 1,
                "destination_country": "US",
                "cargo_type": "normal",
            },
            headers=headers,
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 200
        assert len(resp.json()["data"]["results"]) >= 1

        uid2, token2 = await register_and_login(async_client, "no_calc")
        headers2 = {"Authorization": f"Bearer {token2}"}
        resp = await async_client.post(
            "/api/shipping/calculate",
            json={
                "sku_id": sid,
                "quantity": 1,
                "destination_country": "US",
                "cargo_type": "normal",
            },
            headers=headers2,
        )
        assert resp.status_code == 403

    async def test_write_creates_operation_log(self, async_client):
        """写操作产生审计日志"""
        uid, token = await register_and_login(async_client, "log_user")
        await grant_permission(uid, "shipping:manage")
        headers = {"Authorization": f"Bearer {token}"}
        resp = await async_client.post(
            "/api/shipping/providers",
            json={"name": "审计测试", "code": _uc("audit_p")},
            headers=headers,
        )
        assert resp.status_code == 200

        # 检查操作日志（需要 operation_log:view 权限）
        uid_view, token_view = await register_and_login(async_client, "log_viewer")
        await grant_permission(uid_view, "operation_log:view")
        resp = await async_client.get(
            "/api/operation-logs?module=shipping_provider",
            headers={"Authorization": f"Bearer {token_view}"},
        )
        assert resp.status_code == 200
        logs = resp.json().get("records", [])
        assert any(log["action"] == "create" for log in logs)
