# API 快速参考 (API Quick Reference)

> 凌镜 LingMirror API 路由、权限和响应格式速查
> 更新日期: 2026-07-12
> 完整清单: [完整 API 参考](reference-api-complete.md)

---

## 基础信息

| 项目 | 值 |
|------|-----|
| API 前缀 | `/api/v1` |
| 健康检查 | `GET /api/health`, `GET /api/v1/health` |
| WebSocket | `GET /ws` |
| 指标 | `GET /metrics`（需开启 `metrics.enabled`） |
| 端口 | 8080 (可通过 `SERVER_PORT` 覆盖) |

---

## 认证方式

**JWT Bearer Token**，放在 `Authorization` 头：

```
Authorization: Bearer <token>
```

### 公共路由（无需认证）

| 方法 | 路径 |
|------|------|
| POST | `/api/v1/auth/login` |
| POST | `/api/v1/auth/register` |
| POST | `/api/v1/auth/refresh` |
| GET | `/api/health` |
| GET | `/api/v1/health` |

`GET /api/v1/auth/me` 需要有效 JWT，不是公共路由。

### Token 生命周期

- Access Token: 24 小时（可通过 `JWT_EXPIRY_HOURS` 配置）
- Refresh Token: 168 小时（7 天）
- 通过 `POST /api/v1/auth/refresh` 续期

---

## 标准响应格式

### 成功响应 (`code=0`)

```json
{
  "code": 0,
  "message": "ok",
  "data": { ... }
}
```

### 错误响应 (`code≠0`)

```json
{
  "code": 400,
  "message": "参数错误：商品名称不能为空"
}
```

5xx 错误信息在 `release` 模式下自动脱敏，返回通用消息 "internal server error"。

### 分页响应

```json
{
  "code": 0,
  "message": "ok",
  "data": [ ... ],
  "total": 100,
  "page": 1,
  "size": 20
}
```

分页参数: `?page=1&size=20`，通过 `common.ParsePagination(c)` 解析。
排序参数: `?sort_by=created_at&sort_order=desc`，通过 `common.ParseSort(c)` 解析。

---

## 中间件栈

```
请求 → CORS → RequestID → Metrics(opt-in) → RecoveryWithSentry → Audit → Auth(JWT) → RBAC → Handler
```

- **CORS**: 根据 `cors.allowed_origins` 配置
- **RequestID**: 为每个请求生成唯一 ID（`X-Request-ID` 头）
- **Metrics**: 可选，记录请求计数/延迟到 Prometheus
- **RecoveryWithSentry**: panic 恢复 + Sentry 上报
- **Audit**: 写操作自动记录到 `operationlog` 表
- **Auth**: JWT 验证（仅 `/api/v1` protected 组）
- **RequirePermission**: RBAC 权限码检查（可选，按路由组）

---

## 权限码速查

| 权限码 | 保护的路由组 | 说明 |
|--------|-------------|------|
| `rbac.manage` | `/api/v1/rbac/*` | RBAC 管理 |
| `finance.read` | `/api/v1/finance/*` | 财务数据读取 |
| `report.read` | `/api/v1/reports/*` | 报表读取 |

---

## 主要业务路由组

| 路由前缀 | 模块 | 方法 |
|----------|------|------|
| `/api/v1/products` | 商品 | CRUD |
| `/api/v1/skus` | SKU | CRUD |
| `/api/v1/categories` | 分类 | CRUD + 树结构 |
| `/api/v1/brands` | 品牌 | CRUD |
| `/api/v1/prices` | 价格 | CRUD + 批量调价 |
| `/api/v1/inventory` | 库存 | CRUD + 安全库存 |
| `/api/v1/suppliers` | 供应商 | CRUD |
| `/api/v1/purchases` | 采购 | CRUD |
| `/api/v1/orders` | 订单 | CRUD + 详情 |
| `/api/v1/shipping` | 运费 | CRUD |
| `/api/v1/settlement` | 结算 | CRUD + 详情 |
| `/api/v1/dashboard` | 仪表盘 | Overview/Orders/Inventory/Exceptions |
| `/api/v1/search` | 搜索 | 全文搜索 |
| `/api/v1/platforms` | 平台 | CRUD |
| `/api/v1/listings` | 刊登 | CRUD |
| `/api/v1/listing-tasks` | 刊登任务 | CRUD + 工作台 |
| `/api/v1/logistics` | 物流费率 | 配置 + 报价计算 |
| `/api/v1/image-gen` | 商品生图 | 生成请求 |
| `/api/v1/aftersales` | 售后 | CRUD + 纠纷 |
| `/api/v1/approvals` | 审批 | CRUD |
| `/api/v1/xiao-q` | 唯一 Owner Agent 小Q | 受控 Capability、消息与 Trace |
| `/api/v1/ai/traces` | 旧 Agent 历史审计 | 仅 GET，`audit.read` 且按 Owner 隔离 |
| `/api/v1/ai/actions` | 旧 Action 历史审计 | 仅 GET，`audit.read` 且按 Owner 隔离 |

---

## WebSocket

- 端点: `GET /ws`
- 协议: Gorilla WebSocket
- 用途: 认证后的实时业务更新
- 集成: `realtime.NewHub` + `realtime.NewHandler`
- 旧通用 AI Chat handler 已解除绑定；小Q使用 `/api/v1/xiao-q` 受控 HTTP 契约

---

## 相关文档

- [模块目录](reference-module-catalog.md) — 完整模块列表
- [配置参考](reference-configuration.md) — config.yaml + 环境变量
- [完整 API 参考](reference-api-complete.md) — 从 Gin 运行时路由表生成的完整路由清单
- [旧 API 端点清单](api-inventory.md) — 2026-07-03 历史快照，不再作为事实源
- [权限与审计](PERMISSIONS_AND_AUDIT.md)
