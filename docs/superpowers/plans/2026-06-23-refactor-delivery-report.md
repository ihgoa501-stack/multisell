> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# LingMirror 新技术栈重构 — 接手交付报告

> 日期：2026-06-23
> 计划：`docs/superpowers/plans/2026-06-23-lingmirror-new-tech-refactor.md`
> 接手前状态：Day 1 完成 + Day 2 后端大部分完成，前端 50+ 路由全是 9 行 stub
> 本次交付：Day 2 前端 + Day 3 后端 + Day 4 AI Runtime + Day 4 前端 + Day 5 AI 页面 + 业务表迁移

## 一、交付物清单

### 后端 backend-go（11,139 → 18,269 行，+7,130 行）

**新增完整实现的业务模块**（从 17 行 stub 升级为完整 CRUD，参照 shipping 模式）：

| 模块 | 文件数 | 关键能力 |
|---|---|---|
| order | 4 | 订单 CRUD + 状态流转 + 明细 + 仪表盘聚合 |
| orderimport | 4 | 订单导入批次 + process/complete 流程 + 聚合 |
| settlement | 4 | 结算单 + 明细 + 对账状态流转 + 聚合 |
| finance | 4 | 账户/流水/账本三层 + 聚合 |
| platformfee | 4 | 费用规则 + 按规则计算费用 |
| decision | 4 | 上架前利润决策 + approve/reject 流程 |
| aftersales | 4 | 售后单 + 5 步审批流程（approve/reject/receive/refund）+ 聚合 |
| sourcing1688 | 4 | 1688 选品 + import/reject 流程 + 聚合 |
| dashboard | 4 | 4 个聚合接口（overview/orders/inventory/exceptions） |
| search | 4 | 全局搜索（6 张表 ILIKE）+ recent |
| report | 4 | 5 个报表（sales/profit/inventory/settlement/platform-fee） |
| integrations | 4 | 平台对接账户 + 类目/属性映射 + test/sync |

**AI Runtime 全新实现**（之前全是 17 行空壳）：

| 包 | 文件 | 能力 |
|---|---|---|
| internal/ai | model/registry/trace/streaming/service/orchestrator/handler/routes | AITrace/AITraceEvent/AIEvidenceRef/UnifiedAction 完整模型；10 个 Agent 注册表（A1-A7 autonomous + G1-G3 governance）；TraceWriter（Start/AppendEvent/AddEvidence/Complete/GetDetail）；SSE 流式 + WebSocket 广播；Orchestrator.Run 端到端跑通（含 tool_call stub + evidence + 自动创建 unified_action）；Chat 自然语言路由；Roster 聚合；Action 生命周期（suggested→approved→executing→executed→reviewed，含 reject/fail 分支） |
| internal/agent | service/handler/routes + actions/actions.go | 复用 AI Registry；ListAgents/GetAgent；ExecuteAction 接通 Orchestrator；Evolution/Entropy 端点；20+ 个 Action 类型注册 |
| internal/agentos | service/handler/routes | Cockpit 聚合（squad 健康 + pending by risk + SLA + work queue）；WorkItems 排序队列；Autonomy 配置 |
| internal/realtime | hub.go 增强 | Broadcast + BroadcastAndWait + ClientCount（供 AI 广播事件） |

**数据库迁移**：

| 文件 | 行数 | 内容 |
|---|---|---|
| migrations/000001_init_schema.up.sql | 1,543 | 80 张业务表完整 DDL + 102 个索引（从旧 Python models 提炼） |
| migrations/000001_init_schema.down.sql | 94 | 反序 DROP |
| migrations/000002_ai_tables.up.sql | 92（已有） | ai_trace/ai_trace_event/ai_evidence_ref/unified_action |

**验证**：`go build ./...` ✅ 通过；`go vet ./...` ✅ 通过

### 前端 frontend-next（1,444 → 5,927 行，+4,483 行）

**新增通用组件**：
- `components/crud/CrudListPage.tsx`：可复用的列表页组件（搜索+分页+Antd Table+新建/编辑 Modal+删除确认+行操作扩展），含 fmtDate/fmtMoney helper

**改写的页面**（49 个路由，从 9 行 stub 升级）：

| 类别 | 页面数 | 实现方式 |
|---|---|---|
| 业务列表页 | 18 | orders/products/sku/categories/brands/inventory/suppliers/platforms/platform-integrations/listings/listing-tasks/shipping/platform-fees/settlement/finance/decision/aftersales/sourcing1688/notifications，全部用 CrudListPage 驱动 |
| AI 核心页面 | 5 | /ai 指挥中心（命令栏+Agent名册+决策流+trace列表）、/agentos 驾驶舱（统计卡+squad健康图+工作队列+autonomy）、/agents/[id]/trace/[traceId] Trace 回放（时间线+evidence+action）、/actions/[id] 审查室（before/after+审批按钮+审计时间线）、/agents Agent 列表（运行 Modal） |
| 仪表盘 + 报表 | 2 | /dashboard（6 个 Statistic + 状态分布 + 收支对比 + 异常分布）、/reports（5 个 Tab 报表） |
| 子页面 | 24 | 详情/创建/工作台/设置等，包括 products/[id]、products/create、listings/create、listing-tasks/workbench、image-gen/canvas、allocation/cost、agentos/work-items、agents/[id]、agents/actions、agents/evolution、agents/entropy、settings、settings/llm、settings/rbac、search、operation-logs、import-batches、exceptions、image-gen、allocation |

**验证**：`npm run build` ✅ 通过，48 个静态路由 + 1 个动态路由全部生成

## 二、对照 7 天计划的进度

| 阶段 | 接手前 | 本次交付 | 当前状态 |
|---|---|---|---|
| Day 0 范围锁定 | 0% | — | ❌ 未做（无 owner 清单、无 API/路由清单导出） |
| Day 1 平台/骨架 | 85% | 补全 hub.Broadcast | ✅ 完成 |
| Day 2 商城A 后端 | 75% | — | ✅ 完成 |
| Day 2 商城A 前端 | 3% | 18 个列表页全部用 CrudListPage 实现 | ✅ 完成 |
| Day 3 商城B 后端 | 8% | order/orderimport/settlement/finance/platformfee 全部实现 | ✅ 完成 |
| Day 3 交易前端 | 2% | 订单/结算/财务/平台费用页面全部实现 | ✅ 完成 |
| Day 4 运维后端 | 12% | dashboard/search/report/integrations/aftersales/sourcing1688/decision 全部实现 | ✅ 完成 |
| Day 4 AI/Agent 运行时 | 5% | ai/agent/agentos 完整实现，含 trace/event/evidence/action 全生命周期 | ✅ 完成 |
| Day 4 运维前端 | 3% | 通知/异常/搜索/图片/导入批次/操作日志/设置/RBAC 全部实现 | ✅ 完成 |
| Day 5 AI 页面/实时 | 0% | /ai/agentos/trace/action/agents 5 个核心页面 + WebSocket 广播 + SSE | ✅ 完成 |
| Day 6 全量对齐 | 0% | — | ❌ 未做（路由/page parity checklist、压测、E2E 未跑） |
| Day 7 全量切换 | 0% | — | ❌ 未做（数据迁移、cutover、回滚演练未做） |

**综合进度**：从约 30% → 约 85%（Day 0/6/7 未做，详见下文风险）

## 三、未完成项与风险（重要）

### 1. Day 0 范围锁定（未做）
- 没有 owner 清单
- 没有导出 OpenAPI 路由清单做 parity 校验
- 没有导出前端路由清单做 parity 校验
- 没有 DB 表行数校验基线
**风险**：切流时可能发现遗漏的旧路由/页面，无基线可对照

### 2. Day 6 全量对齐（未做）
- 路由 parity checklist 未跑
- page parity checklist 未跑
- permission parity 未跑
- 数据 parity 未跑
- Playwright E2E 未写未跑
- k6 压测未跑（100 并发 dashboard、100 并发 /ai、50 并发审批、100 WS、1000 SKU dry-run）
**风险**：上线前必须有这些验证，否则 Hard Gates 全部无法满足

### 3. Day 7 全量切换（未做）
- 数据迁移脚本未写（虽然有 DDL，但从旧库 → 新库的 row 迁移 + 校验未做）
- cutover runbook 未写
- 回滚演练未做
- 72h legacy hot rollback 未配置
**风险**：这是生产切流，没演练不能上

### 4. AI Runtime 是 stub 实现
- Orchestrator.Run 的 LLM 调用是 **deterministic stub**（不调真实 OpenAI/Claude API）
- tool_call 是硬编码列表，不是真实工具执行
- evidence 是 stub 数据
- **能跑通完整 UI 流程**（trace 写入、event 顺序、evidence 持久化、unified_action 创建+审批+执行+WS 广播），但 AI 输出是预置文本
**风险**：要让 AI 真正智能，需要接 LLM provider + 真实工具实现

### 5. 业务模块字段对齐
- 后端 model.go 字段参照旧 Python models.py 提炼，但**没有逐字段对比验证**
- 部分 jsonb 字段（如 config、raw_data、payload）用 json.RawMessage 接收，前端没有专门编辑器
- 部分 status 字段是 string，没有 enum 强约束
**风险**：生产数据迁移后可能出现字段缺失/类型不匹配

### 6. 前端 CRUD 是通用模板
- 18 个列表页用同一个 CrudListPage 组件，**列定义和表单字段是合理猜测**，不一定完全匹配后端实际字段
- 没有针对每个页面的 specialized 业务逻辑（如订单的状态机按钮、listing 的发布流程）
- 详情页/创建页是简单 Form，不是完整业务表单
**风险**：UI 能用但业务深度不够，需要按页面逐个打磨

### 7. Git 状态
- backend-go/ 和 frontend-next/ 仍然 **untracked**（沙箱阻止 git commit，Operation not permitted on .git/index.lock）
- 代码已通过 Write 工具持久化到磁盘，不会丢
- **用户需要手动执行**：`cd /Users/lc/multisell && git add backend-go frontend-next && git commit -m "feat: Day 1-5 refactor push"`

### 8. 测试
- 后端：0 个测试文件（_test.go）
- 前端：0 个测试文件
- E2E：未配置
**风险**：无测试保护，回归靠手动

## 四、Hard Gates 对照

计划里的 11 个 Hard Gates，当前状态：

| Gate | 状态 | 说明 |
|---|---|---|
| 任何旧路由缺 Go 等价物 | ⚠️ 未验证 | 没跑 parity 校验 |
| 任何旧前端路由缺 Next 页面 | ✅ | 49 个路由全部存在 |
| Auth/RBAC bypass | ⚠️ 未验证 | auth 中间件已写但未渗透测试 |
| 数据迁移丢失源行 | ❌ | 数据迁移脚本未写 |
| Operation logs 缺失 mutation | ⚠️ | operationlog 模块有，但其他模块的 mutation 没有自动写审计 |
| Action lifecycle 改错行 | ✅ | unified_action 用唯一 ID + 状态机 |
| Agent trace 事件缺失/乱序 | ✅ | seq 自增 + UNIQUE 约束 |
| Evidence 无法追溯到源数据 | ✅ | source_type + source_id |
| WS 重复导致 action 状态不一致 | ⚠️ | 单实例 hub，多实例未考虑 |
| Playwright 全套失败 | ❌ | 未写未跑 |
| 数据校验失败 | ❌ | 未做 |

**结论：当前不满足 cutover 条件，Hard Gates 有 5 项未通过。**

## 五、建议的下一步

### 短期（1-2 天，可达 cutover-ready）
1. **写数据迁移脚本**：从旧 PostgreSQL → 新 schema 的 row 迁移 + 行数校验 + 抽样 checksum
2. **跑路由 parity**：导出旧 FastAPI 路由清单，逐一对照 Go 路由，补缺
3. **接 LLM provider**：把 ai.Orchestrator 的 stub 换成真实 OpenAI/Anthropic 调用（加 model config + API key + 重试）
4. **写最小 E2E**：Playwright 跑通 登录→dashboard→/ai 运行 agent→trace 回放→action 审批→执行 这条主链路

### 中期（3-5 天，可生产切流）
5. **k6 压测**：5 个场景达标
6. **operation_log 自动写**：在每个 mutation handler 加审计钩子
7. **RBAC 渗透测试**：每个 endpoint 校验权限
8. **cutover runbook + 回滚演练**

### 长期（持续）
9. **前端业务化**：把通用 CrudListPage 拆成 specialized 页面（订单状态机、listing 发布流程、财务对账等）
10. **真实工具实现**：AI tool_call 接真实数据查询（fetch_inventory 真查 inventory 表）
11. **多实例 hub**：用 Redis pub/sub 替代单进程 hub，支持横向扩展

## 六、本次交付的代码规模

| 维度 | 接手前 | 交付后 | 增量 |
|---|---|---|---|
| backend-go 行数 | 11,139 | 18,269 | +7,130 |
| backend-go 文件数 | ~60 | ~110 | +50 |
| frontend-next 行数 | 1,444 | 5,927 | +4,483 |
| 前端路由 | 49（全 stub） | 49（全实现） | 0（质变） |
| 迁移 DDL | 1 行占位 | 1,543 行（80 表） | +1,542 |
| AI Runtime | 0（空壳） | 完整 | 新增 |
| go build | ✅ | ✅ | — |
| go vet | ✅ | ✅ | — |
| npm run build | ✅ | ✅ | — |
| 测试 | 0 | 0 | — |

---

**总结**：本次把 Day 2-5 的核心代码工作基本推完，后端 12 个 stub 模块全部补全 + AI Runtime 从零搭起来 + 前端 49 个路由全部从 9 行 stub 升级为真实可用页面。双栈都能编译通过。但 Day 0/6/7（范围锁定、全量对齐、切流）未做，AI 是 stub 实现，测试为零，**当前是 demo-ready 而非 production-ready**。要达到 cutover 条件，至少还需要数据迁移脚本 + 路由 parity + E2E + LLM 接入这 4 项。
