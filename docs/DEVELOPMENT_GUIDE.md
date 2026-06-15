# 凌镜 LingMirror Development Guide

## 本地启动

### 1. 启动数据库

```bash
docker compose up -d db
```

Docker PostgreSQL 默认会使用：

- 数据库：`product_management`
- 测试数据库：`product_management_test`
- 用户名：`postgres`
- 密码：`postgres`

### 2. 启动后端

```bash
cd backend
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
.venv/bin/alembic upgrade head
.venv/bin/python seed.py
.venv/bin/uvicorn app.main:app --reload --port 8001
```

后端 API 文档：

```text
http://localhost:8001/docs
```

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端本地地址：

```text
http://localhost:3001
```

## Docker 一键启动

```bash
docker compose up -d
```

默认地址：

- 前端：`http://localhost:3000`
- 后端：`http://localhost:8000/docs`

## 测试

后端测试：

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest -q
```

前端构建：

```bash
cd frontend
npm run build
```

## 项目结构

```text
backend/
  app/
    auth/            认证与权限依赖
    core/            商品核心、Excel、AI、聚合详情
    sku/             规格和 SKU
    price/           价格和调价记录
    inventory/       库存和库存日志
    supplier/        供应商和商品供应商绑定
    platform/        平台配置
    listing/         平台发布和 adapter
    order/           销售订单
    rbac/            角色、权限、用户角色绑定
    operation_log/   操作日志
    dashboard/       仪表盘和报表
    search/          全局搜索
  tests/             后端测试
  alembic/           数据库迁移

frontend/
  src/
    views/           页面
    router/          路由
    api/             API 调用封装
    components/      布局组件

docs/
  PROJECT_STATUS.md
  DEVELOPMENT_GUIDE.md
  PERMISSIONS_AND_AUDIT.md
  ROADMAP.md
  superpowers/plans/
```

## 后端模块约定

每个业务模块应遵守这个结构：

```text
backend/app/<module>/
  __init__.py
  router.py
  schemas.py
  service.py
```

约定：

- `router.py` 只处理 HTTP 请求、依赖注入和响应映射。
- `service.py` 放业务规则和数据库查询。
- `schemas.py` 放 Pydantic 请求 / 响应模型。
- 新模块在 `__init__.py` 暴露 `router` 后，会被 `app.main` 自动注册到 `/api`。

## 前端模块约定

页面放在：

```text
frontend/src/views/<module>/
```

可选模块路由放在：

```text
frontend/src/router/modules/<module>.ts
```

可选模块 API 放在：

```text
frontend/src/api/modules/<module>.ts
```

`router/index.ts` 和 `api/index.ts` 会自动加载 `modules/*.ts`。

## 新功能开发流程

推荐流程：

1. 在 `docs/superpowers/plans/` 写实现计划。
2. 先写失败测试。
3. 跑测试确认失败原因正确。
4. 写最小实现。
5. 跑局部测试。
6. 跑全量后端测试。
7. 跑前端 build。
8. 更新文档。

后端测试示例：

```bash
cd backend
python3 -m pytest tests/test_logistics_attributes.py -q
python3 -m pytest tests/test_listing.py -q
python3 -m pytest -q
```

前端验证示例：

```bash
cd frontend
npm run build
```

## 交给其他模型时的提示词

可以把下面这段直接发给其他模型：

```text
你接手的是 MultiSell 项目。先阅读 docs/PROJECT_STATUS.md、docs/DEVELOPMENT_GUIDE.md、docs/PERMISSIONS_AND_AUDIT.md 和 docs/ROADMAP.md。

要求：
- 不要重构无关代码。
- 先写测试，再实现。
- 遵守 backend/app/<module>/router.py + service.py + schemas.py 的边界。
- 如果新增状态变化接口，要接入 require_permission(...) 和 OperationLogService.log(...)。
- 完成后运行 cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q，以及 cd frontend && npm run build。
```
