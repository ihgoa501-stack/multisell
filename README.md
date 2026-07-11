# LingMirror — Owner 自用真实付费需求验证系统

> 技术项目名暂保留 `MultiSell`；当前只服务 Owner 本人的真实跨境经营，不面向外部客户。

凌镜现在以新技术栈为唯一活跃开发目标：

- Backend: `backend-go/` — Go / Gin / GORM / PostgreSQL
- Frontend: `frontend-next/` — Next.js / React / TypeScript / Ant Design

旧 Python/FastAPI + Vue 版本已于 2026-06-30 删除，历史代码保留在 git history 中。

## 许可证与使用限制

本项目为专有软件，不是开源项目，也不是免费授权项目。除非项目 Owner 另行签署书面授权，任何个人或组织不得使用、复制、修改、分发、部署、商用、托管、训练模型、反向工程，或基于本项目创建派生项目。

完整条款见 [LICENSE](LICENSE)。第三方依赖仍按其各自许可证执行。

## 核心定位

**真实数据 → 需求假设 → 独立反证 → Owner 批准的最小实验 → 陌生客户付款与签收 → 售后闭合后的正贡献利润。**

当前执行方向见 [docs/CURRENT_DIRECTION_AND_PRIORITIES.md](docs/CURRENT_DIRECTION_AND_PRIORITIES.md)。
完整经营边界见 [docs/SELF_USE_OPERATING_DIRECTION.md](docs/SELF_USE_OPERATING_DIRECTION.md)。当前没有选定目标客户、市场、类目或商品；Ozon、Shopee 和 Shopify 都只是待实证数据源。短期唯一目标是发现或证伪一个可核查的真实付费需求。

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
| 发布管理 | 生成发布任务、审批后进入发布流程、追踪发布状态 |
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
| [端到端教程](docs/tutorial-first-workflow.md) | 🆕 从安装到运行业务闭环，30 分钟端到端体验 |

### 📖 参考
| 文档 | 用途 |
|------|------|
| [当前方向与优先级](docs/CURRENT_DIRECTION_AND_PRIORITIES.md) | 当前产品方向、AgentOS 安全优先级、下一阶段建议 |
| [API 快速参考](docs/reference-api-quick.md) | 路由、认证、响应格式、中间件栈速查 |
| [模块目录](docs/reference-module-catalog.md) | 全部 60+ 后端领域模块一览 |
| [配置参考](docs/reference-configuration.md) | config.yaml + 环境变量完整说明 |
| [权限与审计](docs/PERMISSIONS_AND_AUDIT.md) | 鉴权规则、权限码、审计日志接入方式 |
| [API 端点清单](docs/api-inventory.md) | 完整 API 路由/Handler 对照表 |
| [AI & Agent 系统](docs/reference-ai-agent-system.md) | 🆕 LLM 编排、Agent 注册表、AgentOS 控制台、Trace 系统 |

### 🛠️ 操作指南
| 文档 | 用途 |
|------|------|
| [添加新领域模块](docs/howto-add-domain-module.md) | 添加完整 CRUD 模块的 step-by-step |
| [配置平台集成](docs/howto-platform-integrations.md) | 接入 Ozon / Shopee API |
| [创建自定义 Agent 规则](docs/howto-agent-rules.md) | 控制 Agent 决策边界和触发条件 |
| [运行测试与验证](docs/howto-test-and-verify.md) | Go 测试、前端测试、E2E、冒烟测试 |
| [服务器部署与测试](docs/ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md) | Owner 与 AI 共用的唯一服务器运行手册 |
| [使用 WebSocket 流式更新](docs/howto-websocket.md) | 连接 /ws 端点，接收 AI 流式输出 |
| [执行第一个业务闭环](docs/howto-first-business-loop.md) | 🆕 候选商品→完整度→利润→审批→上架的任务操作指南 |

### 🧠 解释
| 文档 | 用途 |
|------|------|
| [Agent Pipeline 和事件驱动编排](docs/explanation-agent-pipeline.md) | Agent 间如何通过 EventBus 通信和协作 |
| [两个核心业务闭环](docs/explanation-business-loops.md) | 🆕 商品→上架与订单→履约→结算两个主循环的设计 |
| [领域模块架构](docs/explanation-domain-architecture.md) | 🆕 60+ 模块的组织方式、协作模式和依赖关系 |

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
