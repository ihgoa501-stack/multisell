# Prism 商品主图合规集成

## 概述

Prism 是商品主图生成与合规检测引擎。MultiSell 在商品发布执行阶段（`ExecuteTask`）调用 Prism，确保发往平台的商品主图符合各平台合规要求。

## 启用配置

### config.yaml

```yaml
prism:
  base_url: ""               # Prism 服务地址, 如 http://prism:8080
  api_key: ""                # Prism API key
  timeout: 30                # HTTP 超时秒数
  enabled: false             # 是否启用 Prism 图片合规
  strict: true               # Prism 服务异常时: true=阻塞, false=警告后继续用原图发布
```

### 环境变量

| 变量 | 对应配置 | 默认值 |
|------|---------|--------|
| `PRISM_BASE_URL` | `prism.base_url` | `""` |
| `PRISM_API_KEY` | `prism.api_key` | `""` |
| `PRISM_TIMEOUT` | `prism.timeout` | `30` |
| `PRISM_ENABLED` | `prism.enabled` | `false` |
| `PRISM_STRICT` | `prism.strict` | `true` |

### 启用步骤

1. 部署 Prism 服务
2. `PRISM_ENABLED=true PRISM_BASE_URL=http://your-prism:8080 PRISM_API_KEY=sk-xxx`
3. 重启 server
4. 日志出现 `Prism client initialized`

## 数据流

```
用户请求发布 (prism_enabled=true)
  └→ PublishProduct
       └→ 创建 product_listing + listing_task
       └→ published_data 写入 { prism: { enabled: true } }

执行发布任务 POST /listing-task/:id/execute
  └→ ExecuteTask
       ├→ 读 listing → published_data.prism.enabled
       ├→ 读 product → main_image
       ├→ 调用 Prism /api/v1/generate
       │    { image_url, platform(ozon/shopee/...), product_id }
       ├→ 根据 compliance_report.status 分支:
       │   pass    →  Prism 输出图作为平台主图，继续发布
       │   warning →  放行，task_item.result 记录 warnings
       │   fail    →  task→blocked, listing→blocked, 写入 last_error
       │   service err → strict→blocked, non-strict→放行
       └→ 写入 Prism 元数据到 published_data + task_item.result
```

## API 契约

Prism 服务端需实现以下接口：

### POST /api/v1/generate

**Headers:**
```
Authorization: Bearer <api_key>
Content-Type: application/json
```

**Request:**
```json
{
  "image_url": "https://cdn.multisell.com/products/123/main.jpg",
  "platform": "ozon",
  "product_id": 42
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `image_url` | string | 商品主图 URL |
| `platform` | string | 平台代码（ozon/shopee 等） |
| `product_id` | int64 | 商品 ID |

**Response (200):**
```json
{
  "job_id": "prism-job-abc123",
  "output_url": "https://cdn.prism.com/output/abc123.jpg",
  "compliance_report": {
    "status": "pass",
    "reasons": []
  },
  "risk_score": 0.1,
  "failure_reasons": []
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `job_id` | string | Prism 任务 ID |
| `output_url` | string | 合规处理后图片 URL |
| `compliance_report.status` | string | `pass` / `warning` / `fail` |
| `compliance_report.reasons` | []string | warning 或 fail 的原因说明 |
| `risk_score` | float64 | 0.0~1.0 风险评估 |
| `failure_reasons` | []string | 合规失败的具体原因列表（仅 fail 时有） |

## strict 模式

| strict | Prism 离线/报错 | 合规 fail |
|--------|----------------|-----------|
| `true` | task blocked | task blocked |
| `false` | 日志 warning，继续用原图发布 | task blocked（fail 永远阻塞） |

## Prism 存储位置

| 位置 | 内容 |
|------|------|
| `product_listing.published_data` jsonb | `{ prism: { job_id, output_url, compliance_report, risk_score, failure_reasons } }` |
| `listing_task_item.result` jsonb | 同上，前端可追溯 |
| `listing_task.last_error` | fail 时的错误描述 |

## 人工处理合规失败

1. 前端的 listing 列表看到 `blocked` 状态的记录
2. 修改商品图
3. 调用 `POST /listing-task/:id/retry-failed`
4. 重新执行 `POST /listing-task/:id/execute`
5. 自动重新走 Prism 检查

## 测试

```bash
# Prism adapter 包测试（mock，不依赖真实服务）
go test ./internal/prismadapter/...

# Listing task 测试（前置条件：prismSvc=nil，不连外部）
go test ./internal/domain/listingtask/...

# 全量
go test ./...
```
