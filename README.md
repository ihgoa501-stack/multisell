# LingMirror — Cross-border E-commerce AgentOS

> 技术项目名暂保留 `MultiSell`；对外产品品牌为 `凌镜 LingMirror`。

凌镜现在以新技术栈为唯一活跃开发目标：

- Backend: `backend-go/` — Go / Gin / GORM / PostgreSQL
- Frontend: `frontend-next/` — Next.js / React / TypeScript / Ant Design
- Legacy paused: `backend/` and `frontend/`

旧 Python/FastAPI + Vue 版本已暂停维护，仅用于行为对照、数据迁移和紧急回滚。具体规则见 [Active Stack Policy](docs/ACTIVE_STACK_POLICY.md)。

## 核心定位

**商品在这里创建 → AI加工 → 一键发布到多个平台。**

## 功能模块

| 模块 | 说明 |
|------|------|
| 商品管理 | 商品CRUD、批量操作、Excel导入导出、复制 |
| 分类管理 | 无限级分类树 |
| 品牌管理 | 品牌增删改查 |
| 规格与SKU | 规格定义、笛卡尔积自动生成SKU |
| 价格管理 | 多类型价格、批量调价、调价记录 |
| 库存管理 | 库存更新、安全库存预警、库存变动记录 |
| 供应商管理 | 供应商档案、商品-供应商绑定 |
| 平台管理 | 配置Ozon/Shopee等多平台API密钥 |
| 发布管理 | 一键发布商品到多平台、发布状态追踪 |
| AI增强 | AI生成商品标题/描述/SEO关键词 |
| 全局搜索 | 搜索商品/SKU/供应商（快捷键 `/`） |
| 仪表盘 | 数据总览、平台发布统计、近期动态 |
| 操作日志 | 系统操作审计记录 |

## 快速启动新版本

### Docker 一键启动

```bash
docker compose up -d
```

访问前端：http://localhost:3000

访问后端健康检查：http://localhost:8080/api/health

### 本地开发

先启动 PostgreSQL：

```bash
docker compose up -d db
```

**后端：**
```bash
cd backend-go
go run cmd/server/main.go
```

API base：http://localhost:8080/api/v1

**前端：**
```bash
cd frontend-next
npm install
npm run dev -- --hostname 127.0.0.1 --port 3000
```

访问 http://localhost:3000

### 启动旧版本（仅回滚/对照）

```bash
docker compose -f docker-compose.legacy.yml up -d
```

旧版本不再承接新功能。

## 测试

新后端：

```bash
cd backend-go
go test ./...
go vet ./...
```

新前端：

```bash
cd frontend-next
npm run build
npm run lint
```

## 项目文档

| 文档 | 用途 |
|------|------|
| [产品愿景与第一可用版本](docs/PRODUCT_VISION_AND_MVP.md) | 最终产品定位、第一阶段切入口、后续 Agent 开发方向 |
| [项目现状](docs/PROJECT_STATUS.md) | 当前已完成能力、已知限制、验证结果 |
| [项目收口与 Agent 协作规范](docs/PROJECT_GOVERNANCE_AND_AGENT_WORKFLOW.md) | Agent 分工、验收标准、协作流程 |
| [开发指南](docs/DEVELOPMENT_GUIDE.md) | 本地启动、测试、模块约定、交接提示词 |
| [权限与审计](docs/PERMISSIONS_AND_AUDIT.md) | 鉴权规则、权限码、审计日志接入方式 |
| [路线图](docs/ROADMAP.md) | 后续阶段优先级和每阶段待办 |
| [剩余开发总文档](docs/DEVELOPMENT_BACKLOG_AND_SPEC.md) | 还需要开发的全部事项、优先级、模块规格和验收标准 |
| [Active Stack Policy](docs/ACTIVE_STACK_POLICY.md) | 新旧版本边界、旧栈冻结规则、默认开发入口 |
| [新技术栈重构计划](docs/superpowers/plans/2026-06-23-lingmirror-new-tech-refactor.md) | Go + Next + AI 页面完整重构计划 |
| [生产切流 Runbook](docs/superpowers/plans/2026-06-23-cutover-runbook.md) | 切流、验证和回滚步骤 |

## 数据迁移

从旧版本迁移到新版本时，按新后端迁移 runbook 执行：

1. 在 staging 中导入旧库数据并重命名为 `legacy_*` 表。
2. 执行 `backend-go/migrations/000003_data_migration.up.sql`。
3. 执行 `backend-go/migrations/validate.sql`，确认行数、checksum、FK 完整性。

## 技术栈

- 后端：Go / Gin / GORM / PostgreSQL
- 前端：Next.js / React / TypeScript / Ant Design
- 部署：Docker / Docker Compose / Nginx

## Legacy Status

`backend/` and `frontend/` are paused. Do not add new product features there.
