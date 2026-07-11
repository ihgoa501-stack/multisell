# 配置参考 (Configuration Reference)

> 凌镜 LingMirror 服务端配置项完整参考
> 来源: `backend-go/configs/config.yaml` + `internal/config/config.go`
> 更新日期: 2026-06-30

---

## 配置文件

默认配置: `backend-go/configs/config.yaml`

```yaml
server:
  port: 8080
  mode: debug           # debug | release | test
  version: "0.2.1"

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: multisell
  sslmode: disable      # 生产环境建议设为 require
  max_idle_conns: 10
  max_open_conns: 100
  schemas:              # PostgreSQL 业务 schema
    - order_module
    - finance_module
    - inventory_module
    - sku_module
    - settlement_module
  search_path: public,order_module,finance_module,inventory_module,sku_module,settlement_module

redis:
  addr: localhost:6379
  password: ""
  db: 0

cors:
  allowed_origins: "*"  # 生产环境请限制具体域名

metrics:
  enabled: true         # 开启 /metrics 供 Prometheus 抓取

jwt:
  secret: dev-secret-change-in-production
  expiry_hours: 24
  refresh_expiry_hours: 168

llm:
  daily_budget_usd: 0   # 0 = 不限制；按需设置日消耗上限（USD）
  provider: openai       # openai | anthropic | stub
  api_key: ""            # 通过 LLM_API_KEY 环境变量设置

prism:
  base_url: ""           # Prism 图片服务地址，如 http://prism:8080
  api_key: ""
  timeout: 30            # HTTP 超时秒数
  enabled: false         # 是否启用 Prism
  strict: true           # true=Prism 异常时阻塞发布；false=原图继续发布

log:
  level: debug
  format: console        # console | json

schemadrift:
  enabled: true
  on_drift: "warn"
```

## 环境变量覆盖

所有配置项均可通过环境变量覆盖。命名规则: `SECTION_KEY`（全大写，`.` 转为 `_`）。

| 环境变量 | 对应配置路径 | 类型 | 说明 |
|----------|-------------|------|------|
| `SERVER_PORT` | `server.port` | int | 服务监听端口 |
| `SERVER_MODE` | `server.mode` | string | `debug` / `release` / `test` |
| `DB_HOST` | `database.host` | string | PostgreSQL 主机 |
| `DB_PORT` | `database.port` | int | PostgreSQL 端口 |
| `DB_USER` | `database.user` | string | 数据库用户 |
| `DB_PASSWORD` | `database.password` | string | 数据库密码 |
| `DB_NAME` | `database.dbname` | string | 数据库名 |
| `DB_SSLMODE` | `database.sslmode` | string | SSL 模式 (`disable` / `require`) |
| `DB_MAX_IDLE_CONNS` | `database.max_idle_conns` | int | 最大空闲连接数 |
| `DB_MAX_OPEN_CONNS` | `database.max_open_conns` | int | 最大打开连接数 |
| `REDIS_ADDR` | `redis.addr` | string | Redis 地址 |
| `REDIS_PASSWORD` | `redis.password` | string | Redis 密码 |
| `REDIS_DB` | `redis.db` | int | Redis DB 编号 |
| `JWT_SECRET` | `jwt.secret` | string | JWT 签名密钥 |
| `JWT_EXPIRY_HOURS` | `jwt.expiry_hours` | int | Token 过期小时数 |
| `JWT_REFRESH_EXPIRY_HOURS` | `jwt.refresh_expiry_hours` | int | Refresh token 过期小时数 |
| `CORS_ALLOWED_ORIGINS` | `cors.allowed_origins` | string | 允许的跨域来源 |
| `METRICS_ENABLED` | `metrics.enabled` | bool | 是否启用 Prometheus 指标 |
| `SENTRY_DSN` | `sentry.dsn` | string | Sentry DSN（见 `config.yaml` 注释） |
| `LLM_API_KEY` | `llm.api_key` | string | LLM 提供商 API Key |
| `LLM_PROVIDER` | `llm.provider` | string | `openai` / `anthropic` / `stub` |
| `LLM_DAILY_BUDGET_USD` | `llm.daily_budget_usd` | float | 每日 LLM 预算上限（USD） |
| `PRISM_BASE_URL` | `prism.base_url` | string | Prism 服务地址 |
| `PRISM_API_KEY` | `prism.api_key` | string | Prism API Key |
| `PRISM_ENABLED` | `prism.enabled` | bool | 启用 Prism |
| `PRISM_STRICT` | `prism.strict` | bool | Prism 异常时是否阻塞 |
| `PRISM_TIMEOUT` | `prism.timeout` | int | Prism HTTP 超时(秒) |
| `LOG_LEVEL` | `log.level` | string | `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | `log.format` | string | `console` / `json` |

---

## 使用方式

### 1. Docker Compose（开发环境）

编辑 `docker-compose.yml` 中对应服务的环境变量，或直接使用 `configs/config.yaml`。

### 2. 本地开发

复制 `.env.example` 为 `.env`，按需修改。服务启动时自动读取。

### 3. 生产部署

本参考只解释配置项，不提供生产操作步骤。生产配置、Secret 和部署流程只能按 [Owner 与 AI 统一部署测试手册](ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md) 执行。**不要**将密钥直接提交到仓库。

---

## 相关文档

- [模块目录](reference-module-catalog.md) — 完整模块列表
- [API 快速参考](reference-api-quick.md) — 路由、权限、响应格式
- [Owner 与 AI 统一部署测试手册](ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md)
