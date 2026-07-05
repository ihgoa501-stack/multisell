# 凌镜 LingMirror — 功能测试报告

> 测试日期：2026-07-05
> 测试范围：前端 88 个页面 + 后端 API
> 测试方法：浏览器导航 + API curl

---

## 一、测试结果总览

### 前端页面（可访问/渲染）

| 分组 | 页面 | 状态 | 备注 |
|------|------|------|------|
| **Owner** | 经营总控台 /owner | ✅ 渲染 | 标注 Mock，无真实数据 |
| | 候选商品 /candidates | ✅ 渲染 | 标注 Mock |
| | 审批管理 /approval | ✅ 渲染 | 有3个tab，但标注 Mock（与实际不符） |
| | 数据概览 /dashboard | ✅ 渲染 | 标注 Mock |
| **商品** | AI 选品 /sourcing | ✅ | CRUD 表格 |
| | 产品档案 /product-hub | ✅ | CRUD 表格 |
| | 类目 /categories | ✅ | CRUD 表格 |
| | 品牌 /brands | ✅ | CRUD 表格 |
| | SKU /sku | ✅ | 13列表格 |
| | 库存 /inventory | ✅ | 有 tab（库位/调拨） |
| | 供应商 /suppliers | ✅ | CRUD 表格 |
| | 刊登管理 /listings | ✅ | |
| | 刊登任务 /listing-tasks | ✅ | |
| | 平台集成 /platform-integrations | ✅ | 沙箱模式 |
| **订单** | 订单 /orders | ✅ | |
| | 物流 /shipping | ✅ | |
| | 履约中枢 /fulfillment | ✅ | 沙箱模式 |
| | 供应链追踪 /supplychain | ✅ | |
| | 售后 /aftersales | ✅ | |
| | 采购订单 /purchase | ✅ | |
| | 1688采购 /sourcing1688 | ✅ | |
| **AgentOS** | AI 指挥中心 /ai | ✅ | 最完整页面之一 |
| | Agent 列表 /agents | ✅ | 17个Agent，Squad统计 |
| | Action 中心 /agents/actions | ✅ | |
| | 信任与自主度 /agents/trust | ✅ | 进度条+重算按钮 |
| | 工作队列 /agentos/work-items | ✅ | |
| | 代谢评分 /metabolism | ✅ | |
| | 异常监控 /exceptions | ✅ | |
| | 操作日志 /operation-logs | ✅ | |
| **系统** | LLM 配置 /settings/llm | ✅ | |
| | 审批策略 /settings/policy | ✅ | |
| | 设计系统 /design-system | ✅ | 壳模式 |
| | 登录 /login | ✅ | 品牌深色风格 |

### 后端 API

| API | 状态 | 说明 |
|-----|------|------|
| `/api/health` | ✅ 200 | |
| `/api/v1/auth/login` | ✅ JWT 返回 | |
| `/api/v1/auth/register` | ✅ 账号创建 | |
| `/api/v1/ai/agents` | ✅ 17个Agent | |
| `/api/v1/approval` | ✅ | |
| `/api/v1/categories` | ✅ | |
| `/api/v1/brands` | ✅ | |
| `/api/v1/suppliers` | ✅ | |
| `/api/v1/decision` | ✅ | |
| `/api/v1/platforms` | ✅ | |
| `/api/v1/inventory` | ⚠️ 403 | 权限不足 |
| `/api/v1/product-hub` | ⚠️ 403 | 权限不足 |

---

## 二、发现的问题

### P0 — 阻断

无。系统已可登录、渲染所有页面。

### P1 — 严重

| # | 问题 | 影响 |
|---|------|------|
| 1 | **全页面数据为空** | 无种子数据，所有表格0条，图表无数据，测试无意义 |
| 2 | **前端 dev server 会意外退出** | 无守护进程，服务不可靠 |
| 3 | **`inventory` + `product-hub` API 返回 403** | `operator` 角色无这两个模块的读取权限 |

### P2 — 中

| # | 问题 |
|---|------|
| 4 | **日志与会话不稳定** — 页面导航后 session 丢失（localStorage 存了 token 但 useEffect 未及时水合） |
| 5 | **审批管理标注 Mock 但 UI 完整** — 菜单状态标记与实际情况不符 |
| 6 | **Next.js 开发工具提示 2-5 个 issue** — 页面切换时 issue 数变化 |
| 7 | **8080 端口被 nginx 占用** — 默认配置期望 8080 被后端使用 |

### P3 — 低

| # | 问题 |
|---|------|
| 8 | Owner 总控台 /owner 页面 Mock — 核心决策页面不可用 |
| 9 | 履约中枢 /fulfillment 沙箱模式 — 核心履约逻辑未实现 |
| 10 | 设计系统 /design-system 壳模式 |
| 11 | 前端 dev 模式（Turbopack），非生产构建 |

---

## 三、关键发现

1. **登录页设计精美** — 深色品牌风格，"凌镜 ✦"Logo + 副标题"跨境电商 AI Agent 工作台"
2. **Agent 页面功能最完整** — 列表页有 17 个 Agent 真实数据、Squad 分组统计、运行按钮
3. **AI 指挥中心有完整 Agent 卡片面板** — 每个 Agent 含名称/Squad/决策点/自主度
4. **信任与自主度页面复杂** — 含进度条、信任分公式说明、重算/自动升级按钮
5. **库存页面有完整 tab 设计** — 库位管理/调拨管理双面板
6. **审批管理 UI 完整** — 待审批/已审批/全部 三 tab，含审批人绑定（decided_by = JWT identity）

---

## 四、建议

1. **加种子数据** — 最优先，不然看不到功能到底能不能用
2. **修复 inventory/product-hub 403** — 给 `operator` 角色配置缺失的权限
3. **改审批管理菜单标记** —从 Mock 改为正常
4. **考虑用 docker compose 跑 backend 服务镜像** — 避免端口冲突和进程退出问题
5. **加 `/owner` 页面的真实数据** — Owner 总控台是核心入口
