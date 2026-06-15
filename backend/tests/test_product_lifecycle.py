"""商品生命周期一致性测试"""


async def _create_product(async_client, name: str = "生命周期商品"):
    resp = await async_client.post("/api/products", json={"name": name, "unit": "件"})
    assert resp.status_code == 200
    data = resp.json()
    assert data["code"] == 200
    return data["data"]


async def _define_color_specs(async_client, product_id: int):
    resp = await async_client.post(
        f"/api/products/{product_id}/specs",
        json={
            "specs": [
                {"name": "颜色", "values": ["红色", "蓝色"]},
            ]
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["code"] == 200
    return data["data"]


async def _generate_skus(async_client, product_id: int):
    resp = await async_client.post(f"/api/products/{product_id}/skus/generate")
    assert resp.status_code == 200
    data = resp.json()
    assert data["code"] == 200
    return data["data"]["skus"]


class TestProductLifecycle:
    async def test_regenerating_same_specs_preserves_sku_inventory(self, async_client):
        product = await _create_product(async_client, "SKU幂等商品")
        await _define_color_specs(async_client, product["id"])
        first_skus = await _generate_skus(async_client, product["id"])
        first_sku_id = first_skus[0]["id"]

        inv_resp = await async_client.put(
            f"/api/inventory/{first_sku_id}",
            json={"quantity": 12, "warehouse": "默认仓库", "safety_stock": 2},
        )
        assert inv_resp.status_code == 200
        assert inv_resp.json()["code"] == 200

        second_skus = await _generate_skus(async_client, product["id"])

        assert [sku["id"] for sku in second_skus] == [sku["id"] for sku in first_skus]

        inventory_resp = await async_client.get(f"/api/inventory/{first_sku_id}")
        assert inventory_resp.status_code == 200
        assert inventory_resp.json()["data"]["quantity"] == 12

    async def test_sku_list_uses_inventory_quantity_as_stock(self, async_client):
        product = await _create_product(async_client, "SKU库存商品")
        await _define_color_specs(async_client, product["id"])
        skus = await _generate_skus(async_client, product["id"])
        sku_id = skus[0]["id"]

        await async_client.put(
            f"/api/inventory/{sku_id}",
            json={"quantity": 9, "warehouse": "默认仓库", "safety_stock": 1},
        )

        resp = await async_client.get(f"/api/products/{product['id']}/skus")

        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        returned = {sku["id"]: sku for sku in data["data"]}
        assert returned[sku_id]["stock"] == 9

    async def test_product_with_skus_cannot_be_deleted(self, async_client):
        product = await _create_product(async_client, "删除保护商品")
        await _define_color_specs(async_client, product["id"])
        await _generate_skus(async_client, product["id"])

        delete_resp = await async_client.delete(f"/api/products/{product['id']}")

        assert delete_resp.status_code == 200
        assert delete_resp.json()["code"] == 400

        get_resp = await async_client.get(f"/api/products/{product['id']}")
        assert get_resp.status_code == 200
        assert get_resp.json()["code"] == 200
