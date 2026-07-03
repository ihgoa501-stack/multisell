# 2026-07-03 多线并行合并 & 8月缺口梳理

## 完成情况

### Wave 1 — 优先合并 ✅

| PR | 内容 | 方式 |
|----|------|------|
| #260 | EventBus MutationGuard + ActionCatalog 系统动作 | merged to `main` |
| #248 | EventBus 幂等机制（旧） | closed as superseded by #260 |
| #234 | 领域状态机（approval/candidate/listing/loop） | merged to `main` |
| #235 | Owner Cockpit 状态机可视化页面 | merged to `main` |
| #237 | 状态机文档 + Owner 工作流指南 | merged to `main` |
| #252/#253 | CarrierAdapter 安全（SSRF防护 + approvalID集中校验） | merged to `main` as `30555e9f` |
| #245 | AgentOS 加固 fix | closed（基分支已合入 main） |

### Wave 2 — 并行合并 ✅

| PR | 内容 | 方式 |
|----|------|------|
| #247 | Phase 1 工作台（CollectLead/CandidateProduct 查询 + 前端） | merged to `main` |
| #258 | Fulfillment Cockpit Phase 6 前端 | merged to `main` |
| #255 | Traffic System P0/P1/P2（AgentAction验证/审批/审计） | merged to `main` |
| #259 | CandidateProduct 字段补全工作流 | merged to `main` |

### 串行 & 归档 ✅

| PR | 内容 | 方式 |
|----|------|------|
| #246 | first_km_scout 工作流 + collect-leads端点 | closed（已通过 #247/bd6385e5 包含） |

### 验证

```bash
cd backend-go && go test ./...   # 全部通过
cd backend-go && go vet ./...    # 干净
cd frontend-next && npm run build # 成功
cd frontend-next && npm run lint  # 0 errors
```

## 剩余 8 月缺口

按 Q3 战略，7 月计划中以下 P0/P1 尚未启动：

| 优先级 | 缺口 | 域 | 说明 |
|--------|------|----|------|
| 🔴 P0 | 售后退货 (aftersales) | 新 domain | 平台对账基础，建议 worktree 开发 |
| 🔴 P0 | Excel 批量运营 (import/export) | 新 domain | 批量编辑/导出，独立不冲突 |
| 🟡 P1 | 多仓库分配 (inventory) | inventory 域扩 | 扩展现有模型 |
| 🟡 P1 | AgentOS Phase 2 审批闭环 | agentos/approval | 完善审批链路、异常处理、审计 |

## 架构状态（合并后）

### candidate domain（之前 4 条线争夺，现在串行完毕）

```
service.go → GetCollectLead (PR#247) + ListCollectLeads (PR#247) +
             completeness engine (PR#259) + SkipFieldCheck (PR#259)
model.go  → CandidateProduct + completeness 字段 (PR#259)
handler.go → CollectLead 只读 + 候选商品增删改 + 完整度 API (PR#247/#259)
routes.go  → GET /candidates/collect-leads + GET candidates CRUD (PR#247/#259)
migrations → 000060-000062 (状态机) + 000063 (字段补全)
```

### platform 层（EventBus + Traffic System 同时合入）

```
eventbus      → mutation guard (PR#260) + idempotency (PR#260)
command       → AgentAction.Validate (PR#255)
actioncatalog → HighRiskActions (PR#255)
dispatchsafe  → Validate + AuditRecorder (PR#255)
statemachine  → 4 domain 状态机注册 (PR#234)
```

## 建议下一步

1. **尽快启动 aftersales（售后退货）** — 独立 domain，零冲突，可走 worktree
2. **Excel 批量运营** — 独立 feature，建议 `feat/excel-batch-operations` 分支
3. **2026-07-10 checkpoint** — 如果 aftersales 和 Excel 未启动，考虑回调 Traffic System 搁置释放资源
