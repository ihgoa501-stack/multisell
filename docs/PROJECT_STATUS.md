# 凌镜 LingMirror Project Status

说明：`MultiSell` 是历史技术项目名；当前产品品牌为 `凌镜 LingMirror`。

更新时间：2026-06-24

## 当前结论

凌镜已完成全站新技术栈迁移。当前唯一活跃开发线是：

- Backend: `backend-go/`，Go / Gin / GORM / PostgreSQL
- Frontend: `frontend-next/`，Next.js / React / TypeScript / Ant Design
- API prefix: `/api/v1`
- Health check: `/api/health`

旧栈 `backend/`（Python / FastAPI）和 `frontend/`（Vue 3）已经暂停，只能用于行为对照、迁移参考、安全回滚或文档标注。新功能不得继续落到旧栈。

## 一句话说明

凌镜是跨境电商 AI AgentOS，核心流程是：

商品创建 -> SKU / 价格 / 库存维护 -> AI 优化与经营决策 -> 多平台发布 -> 订单、结算、财务、异常和 AgentOS 运营闭环。

## 入口与运行

| 项目 | 位置 |
|---|---|
| 后端入口 | `backend-go/cmd/server/main.go` |
| 后端路由汇总 | `backend-go/internal/httpx/router.go` |
| 后端配置 | `backend-go/configs/config.yaml`，支持环境变量覆盖 |
| 前端入口 | `frontend-next/src/app/` |
| 前端 API client | `frontend-next/src/lib/api-client.ts` |
| 前端菜单配置 | `frontend-next/src/config/menu.ts` |
| Docker 默认入口 | `docker-compose.yml` |

本地开发命令：

```bash
docker compose up -d db

cd backend-go
go run cmd/server/main.go

cd frontend-next
npm run dev -- --hostname 127.0.0.1 --port 3000
```

## 当前覆盖

### 后端

`backend-go/internal/httpx/router.go` 在 `/api/v1` 下注册了认证、RBAC、Agent、AgentOS 和业务域路由。当前业务域包括：

- 商品基础：`category`、`brand`、`sku`、`price`、`inventory`、`supplier`
- 平台与发布：`platform`、`integrations`、`listing`、`listingtask`
- 订单履约：`order`、`orderimport`、`shipping`、`platformfee`、`aftersales`
- 财务经营：`finance`、`settlement`、`decision`、`allocation`、`report`、`exchangerate`
- 运营支撑：`dashboard`、`search`、`notification`、`exceptions`、`operationlog`、`importbatch`
- AI / AgentOS：`ai`、`agent`、`agentos`、`agentrule`、`entropy`、`evolution`、`trustscore`、`actionpolicy`
- 选品与生图：`sourcing1688`、`imagegen`
- 实时能力：WebSocket `/ws`

### 前端

`frontend-next/src/app/` 已按业务域迁移到 Next App Router。当前 build 输出覆盖：

- Dashboard / AI / AgentOS / Action Center
- 商品、SKU、分类、品牌、库存、供应商
- 平台、平台集成、刊登、刊登任务、刊登任务详情
- 订单、订单详情、订单导入、售后
- 物流、平台费用、结算、结算详情、财务、经营决策、成本分摊
- 异常、通知、生图、画布、批量导入、操作日志、搜索、报表
- 设置、LLM 配置、RBAC、审批策略

侧边栏菜单目前有 41 个入口，已确认都能匹配到 `frontend-next/src/app` 下的实际页面。

## 验证状态

2026-06-24 复核结果：

| 检查 | 结果 | 说明 |
|---|---:|---|
| `cd backend-go && go test ./...` | 通过 | 8 个 Go 测试文件覆盖核心模块 |
| `cd backend-go && go vet ./...` | 通过 | 无 vet 输出 |
| `cd frontend-next && npm test` | 通过 | 11 个 test files，75 tests |
| `cd frontend-next && npm run build` | 通过 | Sentry auth token/source map 上传 warning 不阻塞 build |
| `cd frontend-next && npm run lint` | 失败 | 16 errors / 22 warnings，主要是 `any`、未使用变量和 `AuthGuard` hooks 规则 |

## 当前风险

### API 路径一致性

前端默认 API base 是 `http://localhost:8080/api`，因此业务调用应统一使用 `/v1/*` 路径，最终形成 `/api/v1/*`。当前仍发现部分调用缺少 `/v1` 前缀，例如：

- `/ai/actions`
- `/policy/rules`
- `/evolution/nudges`
- `/trust-scores/summary`

这些调用在默认配置下会请求 `/api/...`，无法命中 Go 后端当前的 `/api/v1/...` 路由。

### 前端 lint 门禁

当前 build 和测试通过，但 lint 未通过。典型问题：

- `@typescript-eslint/no-explicit-any`
- `@typescript-eslint/no-unused-vars`
- `react-hooks/set-state-in-effect`，见 `frontend-next/src/components/auth/AuthGuard.tsx`

### 测试覆盖

后端测试集中在 `ai`、`auth`、`aftersales`、`order`、`settlement`、`shipping`、`eventbus`、`rbac`，大量 domain 模块还没有 focused tests。前端组件测试较完整，但业务页面和 API 交互测试仍偏薄。

### 文档清理

旧 FastAPI / Vue 阶段文档仍然存在，阅读时应先看：

- `README.md`
- `AGENTS.md`
- `docs/ACTIVE_STACK_POLICY.md`
- `backend-go/README.md`
- `frontend-next/README.md`

历史文档中出现 `backend/app/*`、`frontend/src/views/*`、`/api/*` 时，默认按旧栈参考处理，不能直接作为当前实现事实。

## 下一步建议

1. 统一前端 API 调用路径，所有 `apiClient` 业务调用显式使用 `/v1/*`。
2. 修复 `frontend-next` lint，恢复前端质量门禁。
3. 为无测试的高风险 domain 模块补 focused tests，优先覆盖发布、库存、财务、AgentOS 动作链路。
4. 清理或标注旧栈文档，避免把 Vue/FastAPI 的历史缺口误读为当前缺口。
