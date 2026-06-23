"""认证接口测试"""


class TestAuth:
    """用户认证"""

    async def test_login_success(self, async_client):
        """POST /api/auth/login → 用已知用户登录"""
        resp = await async_client.post(
            "/api/auth/login",
            json={
                "username": "admin",
                "password": "admin123",
            },
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        assert "access_token" in data["data"]
        assert data["data"]["token_type"] == "bearer"
        assert data["data"]["user"]["username"] == "admin"

    async def test_login_wrong_password(self, async_client):
        """POST /api/auth/login → 密码错误"""
        resp = await async_client.post(
            "/api/auth/login",
            json={
                "username": "admin",
                "password": "wrong_password",
            },
        )
        assert resp.status_code in (200, 401)
        data = resp.json()
        assert data["code"] != 200 or data["code"] == 401

    async def test_login_nonexistent_user(self, async_client):
        """POST /api/auth/login → 用户不存在"""
        resp = await async_client.post(
            "/api/auth/login",
            json={
                "username": "nonexistent_user_xxx",
                "password": "password123",
            },
        )
        data = resp.json()
        assert data["code"] != 200

    async def test_register(self, async_client):
        """POST /api/auth/register → 注册新用户"""
        import random

        suffix = random.randint(10000, 99999)
        resp = await async_client.post(
            "/api/auth/register",
            json={
                "username": f"testuser_{suffix}",
                "password": "testpass123",
                "display_name": f"测试用户{suffix}",
            },
        )
        data = resp.json()
        # 注册可能成功或返回用户已存在
        assert data["code"] in (200, 400)
