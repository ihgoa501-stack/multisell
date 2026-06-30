# How to 配置平台集成（Ozon / Shopee）

> 配置跨境电商平台 API 密钥，使凌镜可以发布商品、同步订单。

---

## 前置条件

- 拥有 Ozon / Shopee 卖家账号和 API 访问权限
- 凌镜后端已启动（默认 `http://localhost:8080`）

## 步骤

### 1. 添加平台配置

通过 Web UI：进入 **设置 → 平台管理**，点击"添加平台"。

或直接通过 API：

```bash
curl -X POST http://localhost:8080/api/v1/platforms \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "ozon",
    "name": "Ozon",
    "enabled": true
  }'
```

### 2. 配置 API 密钥

创建平台集成凭证：

```bash
curl -X POST http://localhost:8080/api/v1/integrations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "platform_code": "ozon",
    "credentials": {
      "client_id": "your-ozon-client-id",
      "api_key": "your-ozon-api-key"
    },
    "settings": {
      "sync_orders": true,
      "sync_interval_minutes": 15
    }
  }'
```

Ozon 需要的凭证字段: `client_id` + `api_key`
Shopee 需要的凭证字段: `partner_id` + `partner_key` + `shop_id`

### 3. 验证连接

```bash
curl http://localhost:8080/api/v1/integrations/ozon/status \
  -H "Authorization: Bearer <token>"
```

成功响应:

```json
{
  "code": 0,
  "data": {
    "platform": "ozon",
    "connected": true,
    "last_sync": "2026-06-30T10:00:00Z"
  }
}
```

### 4. 启用自动同步

平台集成配置中的 `sync_orders: true` 会自动注册一个 15 分钟间隔的定时同步任务（`scheduler.tick.ozon_sync`）。

手动触发同步：

```bash
curl -X POST http://localhost:8080/api/v1/integrations/ozon/sync \
  -H "Authorization: Bearer <token>"
```

## 适配器架构

每个平台实现 `internal/domain/integrations/adapter.go` 中的 `PlatformAdapter` 接口：

```go
type PlatformAdapter interface {
    Publish(listing *listing.Listing) (string, error)
    SyncStatus(platformID string) (string, error)
    ValidateCredentials(credentials map[string]string) error
    SyncInventory(skuID string, quantity int) error
    PushTracking(orderID, trackingNumber string) error
    FetchOrders(since time.Time) ([]Order, error)
}
```

通过 `RegisterAdapter("ozon", &Adapter{})` 注册（见 `registry.go`）。

## 验证

1. 在凌镜后台的"平台集成"页面看到连接状态为绿色
2. 创建一个商品并发布到该平台，检查发布状态
3. 等待 15 分钟，确认订单自动同步

## 故障排查

| 问题 | 原因与解决 |
|------|-----------|
| `Invalid credentials` | API 密钥过期或填错。到对应平台卖家后台重新生成。 |
| 商品发布失败 | 检查商品信息是否符合平台要求（必填字段、图片尺寸、类目 ID） |
| 订单不同步 | 检查 `sync_orders` 是否为 true；检查 API 调用额度是否耗尽 |
| `403 Forbidden` | 账号权限不足。确认 API 密钥具有商品管理和订单读取权限。 |

---

## 相关文档

- [How to 添加新领域模块](howto-add-domain-module.md)
- [参考 - API 快速参考](reference-api-quick.md)
