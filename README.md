# LingMirror — Cross-border E-commerce AgentOS

> 技术项目名暂保留 `MultiSell`；对外产品品牌为 `凌镜 LingMirror`。

凌镜现在以新技术栈为唯一活跃开发目标：

- Backend: `backend-go/` — Go / Gin / GORM / PostgreSQL
- Frontend: `frontend-next/` — Next.js / React / TypeScript / Ant Design

旧 Python/FastAPI + Vue 版本已于 2026-06-30 删除，历史代码保留在 git history 中。

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
| AI选品 | A8 Agent 利润分析引擎，1688 选品采集与评估 |
| 物流费率 | A10 物流费率引擎，四类定价模式，YAML 费率表配置 |
| 工具桥接 | Agent 插件执行桥接，WebSocket → Chrome 扩展采集 |
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

### 启动旧版本

旧版本已于 2026-06-30 删除，历史代码在 git history 中。

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

### 🚀 入门
| 文档 | 用途 |
|------|------|
| [入门教程](docs/tutorial-getting-started.md) | 从零搭建开发环境到创建第一个商品（新手上路首选） |

### 📖 参考
| 文档 | 用途 |
|------|------|
| [API 快速参考](docs/reference-api-quick.md) | 路由、认证、响应格式、中间件栈速查 |
| [模块目录](docs/reference-module-catalog.md) | 全部 60+ 后端领域模块一览 |
| [配置参考](docs/reference-configuration.md) | config.yaml + 环境变量完整说明 |
| [权限与审计](docs/PERMISSIONS_AND_AUDIT.md) | 鉴权规则、权限码、审计日志接入方式 |
| [API 端点清单](docs/api-inventory.md) | 完整 API 路由/Handler 对照表 |

### 🛠️ 操作指南
| 文档 | 用途 |
|------|------|
| [添加新领域模块](docs/howto-add-domain-module.md) | 添加完整 CRUD 模块的 step-by-step |
| [配置平台集成](docs/howto-platform-integrations.md) | 接入 Ozon / Shopee API |
| [创建自定义 Agent 规则](docs/howto-agent-rules.md) | 控制 Agent 决策边界和触发条件 |
| [运行测试与验证](docs/howto-test-and-verify.md) | Go 测试、前端测试、E2E、冒烟测试 |
| [配置与部署](docs/howto-deploy.md) | Docker 生产部署、Nginx/Caddy、备份恢复 |
| [使用 WebSocket 流式更新](docs/howto-websocket.md) | 连接 /ws 端点，接收 AI 流式输出 |

### 🧠 解释
| 文档 | 用途 |
|------|------|
| [Agent Pipeline 和事件驱动编排](docs/explanation-agent-pipeline.md) | Agent 间如何通过 EventBus 通信和协作 |

### 📚 领域指南
| 文档 | 用途 |
|------|------|
| [AI 选品使用指南](docs/sourcing-guide.md) | A8 选品引擎使用说明与 API 参考 |
| [物流费率引擎指南](docs/logistics-guide.md) | A10 物流费率引擎配置与调用 |
| [ToolBridge 指南](docs/toolbridge-guide.md) | Agent 工具桥接插件开发 |
| [Chrome 扩展指南](docs/chrome-extension-guide.md) | 选品助手扩展安装与协议 |

### 🏛️ 架构与治理
| 文档 | 用途 |
|------|------|
| [系统架构设计 v1](docs/system-architecture-design-v1.md) | 九层架构、数据流、Agent 编排 |
| [AIOS 基础设施架构](docs/aios-architecture.md) | 11 个 AIOS 内核模块设计 |
| [产品愿景与 MVP](docs/PRODUCT_VISION_AND_MVP.md) | 最终产品定位、第一阶段切入口 |
| [项目现状](docs/PROJECT_STATUS.md) | 当前完成能力、已知限制 |
| [路线图](docs/ROADMAP.md) | 后续阶段优先级和每阶段待办 |
| [项目治理与 Agent 协作规范](docs/PROJECT_GOVERNANCE_AND_AGENT_WORKFLOW.md) | Agent 分工、验收标准 |
| [Owner-First 开发协议](docs/governance/OWNER_FIRST_PROTOCOL.md) | 非技术 Owner 如何提需求和验收 |
| [平台宪法](docs/governance/PLATFORM_CONSTITUTION.md) | 系统分层、风险等级、禁止操作 |
| [Agent 开发协议](docs/governance/AGENT_DEVELOPMENT_PROTOCOL.md) | 多 Agent 开工/review/QA/交接规则 |
| [Kernel 契约](docs/governance/KERNEL_CONTRACTS.md) | EventBus/Command/Scheduler/ToolBridge 等接口契约 |
| [开发指南](docs/DEVELOPMENT_GUIDE.md) | 本地启动、测试、模块约定、交接提示词 |
| [Active Stack Policy](docs/ACTIVE_STACK_POLICY.md) | 新旧版本边界、旧栈冻结规则 |

## 数据迁移

从旧版本迁移到新版本时，按新后端迁移 runbook 执行：

1. 执行 `backend-go/migrations/000003_data_migration.up.sql`。
2. 执行 `backend-go/migrations/validate.sql`，确认行数、checksum、FK 完整性。

## 技术栈

- 后端：Go / Gin / GORM / PostgreSQL
- 前端：Next.js / React / TypeScript / Ant Design
- 部署：Docker / Docker Compose / Nginx

旧栈已于 2026-06-30 删除。历史代码保留在 git history 中。
