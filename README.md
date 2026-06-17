# 🪞 凌镜 LingMirror — 跨境电商 AgentOS

> 技术项目名暂保留 `MultiSell`；对外产品品牌为 `凌镜 LingMirror`。

基于 **Python FastAPI + Vue 3 + PostgreSQL** 的 AI Agent 协作跨境电商运营平台。

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

## 快速启动

### Docker一键启动

```bash
docker compose up -d
```

访问前端：http://localhost:3000

访问后端 API：http://localhost:8000/docs

### 本地开发

先启动 PostgreSQL：

```bash
docker compose up -d db
```

**后端：**
```bash
cd backend
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
.venv/bin/alembic upgrade head
.venv/bin/python seed.py
.venv/bin/uvicorn app.main:app --reload --port 8001
```

API文档：http://localhost:8001/docs

**前端：**
```bash
cd frontend
npm install
npm run dev
```

访问 http://localhost:3001

## 测试

后端测试使用独立数据库 `product_management_test`。Docker 首次初始化 PostgreSQL 时会自动创建该测试库。

```bash
docker compose up -d db
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  python3 -m pytest -q
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
| [实施计划](docs/superpowers/plans/2026-06-13-multisell-stabilization-roadmap.md) | 分阶段工程实施计划 |

## 数据初始化

首次部署或需要演示数据时，运行数据初始化脚本：

```bash
cd backend
.venv/bin/pip install -r requirements.txt
.venv/bin/python seed.py
```

该脚本会独立创建以下演示数据（不依赖 FastAPI 启动流程）：
- 管理员账号：`admin` / `admin123`
- 16 个商品分类（含父子层级）
- 5 个跨境电商平台（Ozon、Shopee、Wildberries、速卖通、Temu）
- 6 个品牌
- 7 个演示商品及多规格 SKU、默认库存记录

> 如需重置数据库（删除所有表并重新填充），可使用运维工具：
> ```bash
> cd backend
> .venv/bin/python scripts/db_reset.py
> ```

## 技术栈

- 后端：Python 3.11+ / FastAPI / SQLAlchemy 2.0 / PostgreSQL
- 前端：Vue 3 / TypeScript / Naive UI / Pinia
- 部署：Docker / Docker Compose / Nginx
