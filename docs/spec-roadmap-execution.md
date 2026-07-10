# Spec: Roadmap Execution — AI 决策质量平台 (#321)

## Objective

一人跨境卖家（年 GMV 500-2000 万，1-3 人团队）的 AI 决策质量平台。

**核心命题：帮卖家选得更准，不是做得更快。**

Four phases, parallel tracks where possible:

| Phase | Timeline | Track A (Code) | Track B (Market) |
|-------|----------|---------------|------------------|
| **0** | Week 1-2 | — | 卖家访谈（3-10 人），必问选品流程/痛点/付费意愿 |
| **0.5** | Week 3-5 | Phase 1 基建并行 | Concierge 试点（人工代运营+真实付款验证） |
| **1** | Week 2-6 | 5 层 guardrails 接入所有 LLM 调用路径 → EventBus correlation → Scheduler 健康 → ToolBridge → Approval/Audit 收尾 | — |
| **P3** | 待定 | ~~第一垂直应用：需求挖掘 → 选品分析~~ 降级。优先完成真实上架闭环 + 用户获取。 | — |

**退出条件**：Phase 0.5 结束时无人愿意付费 → 重新评估方向。

## Current State（调研摘要）

Phase 1 的"可立即开工"部分——guardrails wiring——需要先理解已有实现：

### Guardrails L1-L5 当前状态

| Layer | 实现 | 文件 | 接入情况 |
|-------|------|------|----------|
| L1 Input Guard | ✅ `PromptInjectionGuard` | `input_guard.go` | 仅在 ToolRegistry hook 中触发，LLM Chat 入口未检查 |
| L2 Call Guard | ✅ `PermissionGuard` | `call_guard.go` | ToolRegistry hook 中触发，已接线 |
| L3 Output Guard | ✅ `OutputGuard` (schema/business-rule) | `output_guard.go` | 有完整实现但未实际接入 LLM 输出路径 |
| L4 Execution Guard | ✅ `ExecutionGuard` (金额/数量阈值) | `execution_guard.go` | 在 `ExecuteAction` Gate 6 检查，ToolRegistry hook 也有 |
| L5 Rollback Guard | ✅ `RollbackGuard` | `rollback_guard.go` | 仅 inventory 工具在调用 `SetRollbackGuard` 后使用 |

**Guardrails Chain 创建位置**：`backend-go/internal/aios/setup/setup.go:77-83`
**Orchestrator 接线位置**：`backend-go/internal/httpx/router.go:186`
**ExecuteAction 检查位置**：`backend-go/internal/ai/service.go:279-313`

### LLM 调用路径（2026-07-09 更新）

| 路径 | 文件 | Guardrails | 说明 |
|------|------|-----------|------|
| REST `/ai/chat` | `handler.go:43` → `orch.Chat()` | ✅ L1 已接入 | 2026-07-09 新增：Chat 入口 L1 硬拦截 |
| WebSocket `/ws` | `websocket.go:14` → `orch.Chat()` | ✅ L1 已接入 | 复用 Chat() 的 guard |
| Orchestrator.Run | `runWithTimeout()` → `synthesizeOutput()` | ✅ L1+L3 已有 | `synthesizeOutput()` 内 L1 (soft) + L3 (prod hard) |
| ExecuteAction | `service.go:229` Gate 6 | ✅ L4 guardrails | 通过 `guard.Check()` |
| ToolRegistry.Call | `registry.go:177` via hook | ✅ L1-L5 全部 | setup.go 注册为 pre-call hook |
| LLM Gateway | `llmgateway/gateway.go:154` | 🔲 stub provider | 待用户验证后切换真实 provider |

**结论**：Guardrails 已在所有 LLM 调用路径覆盖。修正：初始调研未发现 `synthesizeOutput()` 内已存在 L1+L3 检查。

## Commands

```bash
# ===== Phase 0 (no code) =====
# 无运行命令。只做访谈、整理纪要。

# ===== Phase 1 =====
# Backend 测试
cd backend-go
go test ./internal/aios/guardrails/...   # guardrails 单元测试
go test ./internal/ai/...                # AI 模块测试（新增 guardrails 检查点）
go vet ./...

# Frontend（如涉及 UX 变更）
cd frontend-next
npm run lint
npm test
npm run build

# Full stack
docker compose up -d db
cd backend-go && go run cmd/server/main.go
cd frontend-next && npm run dev -- --hostname 127.0.0.1 --port 3000

# ===== P3（原 Phase 2） =====
# 降级。当前优先级：真实上架闭环 → 用户获取。
```

## Project Structure

```
backend-go/internal/
├── aios/
│   ├── guardrails/          # L1-L5 guardrail implementations (Phase 1)
│   │   ├── guardrails.go    # Chain, Guardrail interface, GuardInput
│   │   ├── input_guard.go   # L1
│   │   ├── call_guard.go    # L2
│   │   ├── output_guard.go  # L3
│   │   ├── execution_guard.go # L4
│   │   ├── rollback_guard.go  # L5
│   │   └── *_test.go
│   ├── setup/setup.go       # AIOS 初始化，guardrails 链创建（可能需要改）
│   ├── llmgateway/          # LLM 网关（可能需要接入真实 provider）
│   └── toolregistry/        # 工具注册表，guardrails hook 接线（已接）
├── ai/
│   ├── orchestrator.go      # Orchestrator.Chat() — 需要接入 guardrails
│   ├── handler.go           # REST /ai/chat — 需要接入 guardrails
│   ├── service.go           # ExecuteAction — Gate 6 已接 guardrails
│   ├── websocket.go         # WebSocket — 需要接入 guardrails
│   └── llm_provider.go      # LLM 提供者（实际 HTTP 调用）
├── httpx/router.go          # 接线点（已接 guardrails → orchestrator）
└── platform/
    ├── actioncatalog/       # 动作目录（已集成）
    └── command/             # 命令调度器（已集成）

deliverables/
└── interviews/              # Phase 0 卖家访谈记录（新建）

docs/
└── spec-roadmap-execution.md  # 本文件
```

## Code Style

Phase 1 guardrails 接入遵循以下模式。核心原则：每个 LLM 调用路径加一个 `guardrails.Check()` 调用，不引入新抽象。

```go
// ✅ 对：在 Chat 入口加单行 guardrails 检查
func (o *Orchestrator) Chat(message string, userID int64) (*ChatResponse, error) {
    // L1: Input guard — check raw user input
    if o.guardrails != nil {
        gr, err := o.guardrails.Check(context.Background(), &guardrails.GuardInput{
            RawInput: message,
            UserID:   userID,
        })
        if err != nil || gr.Blocked {
            return nil, ErrBlockedByGuardrails
        }
    }
    // ... existing logic ...
}

// ✅ 对：输出后加 L3 检查
resp, err := o.provider.Chat(ctx, req)
if err == nil && o.guardrails != nil {
    gr, err := o.guardrails.Check(ctx, &guardrails.GuardInput{
        RawOutput: resp.Answer,
    })
    if err != nil || gr.Blocked {
        // 降级或标记
    }
}

// ✅ 对：在 orchestrator.go 的 Run() 方法中包装 LLM 调用
func (o *Orchestrator) Run(ctx context.Context, req *RunAgentRequest) (*RunResult, error) {
    // L1 on input
    if o.guardrails != nil {
        gr, err := o.guardrails.Check(ctx, &guardrails.GuardInput{
            RawInput: req.Input,
            AgentID:  req.AgentID,
            UserID:   req.UserID,
        })
        if err != nil || gr.Blocked {
            return nil, ErrBlockedByGuardrails
        }
    }
    // ... agent execution ...
    // L3 on output — before returning to caller
    if result != nil && o.guardrails != nil {
        gr, _ := o.guardrails.Check(ctx, &guardrails.GuardInput{
            RawOutput: result.Output,
        })
        if gr != nil && gr.Blocked {
            result.Output = "[blocked by guardrails]"
        }
    }
}
```

Key conventions:
- Go 函数名：`PascalCase`，`camelCase` 本地变量
- Error 命名：`ErrXxx` 包级变量
- GuardInput 复用已有结构体，不新增 wrapper
- 如果 guardrails 未配置 (`nil`)，跳过检查（已有 `if o.guardrails != nil` 模式）
- 所有变更加对应的 go test，至少一个 happy path + one guardrail-blocked case
- `ponytail:` 注释标记故意简化的实现，注明何时需要升级

## Testing Strategy

| Phase | Test Level | What |
|-------|-----------|------|
| 0 | N/A | 访谈记录在 `deliverables/interviews/` 下，Markdown 格式 |
| 0.5 | N/A | 付款凭证、反馈纪要 |
| 1 | `go test` | 每处新的 guardrails 调用点：happy path + blocked case |
| 1 | `go test ./internal/aios/guardrails/...` | guardrails 单元测试全覆盖 |
| 1 | `go vet ./...` | 静态分析无新增 issue |
| 1 | 端到端 | API 启动后，发一条含 prompt injection 的消息，确认被 guardrails 拦截 |
| **P3** | TBD | 原 Phase 2 降级，待真实上架闭环 + 用户获取后重新评估方向 |

### Guardrails 接入验收检查（Phase 1 已完成）
- [x] REST `/api/v1/ai/chat` — L1 input guard 检查用户输入
- [x] WebSocket `/ws` — L1 input guard 检查用户消息
- [x] Orchestrator.Run — L1 + L3 检查 input/output
- [x] 所有 LLM 调用路径返回 guardrails-blocked 错误时不暴露内部细节
- [x] guardrails nil-safe（未配置时跳过，不 panic）
- [x] 测试覆盖率：每处新增检查点至少 2 个测试 case

## Boundaries

### Always Do
- 每个 LLM 调用路径加 guardrails 检查
- L1 在用户输入进入 LLM 前检查
- L3 在 LLM 输出后检查
- guardrails = nil 时跳过所有检查（不破坏未接线的环境）
- 变更对应文件的测试：至少 happy path + blocked case
- 卖家访谈记录用 `deliverables/interviews/YYYY-MM-DD-{seller-alias}.md` 格式

### Ask First
- 修改 guardrails Chain 接口（`GuardInput`/`GuardResult` 结构体）
- 添加新 guardrail 层（L0 或 L6）
- 修改 `ExecuteAction` 中 guardrails 的触发位置（Gate 6）
- 为 LLM Gateway 切换非 stub provider 到生产
- 任何涉及外部平台写回、价格变更、库存变更、订单状态的代码
- 扩大 Agent 的 autonomous execution 范围
- 为 P3（原 Phase 2）做任何 scaffold（当前优先级：真实上架闭环 → 用户获取）

### Never Do
- 不写 spec 就进入编码
- 不通过 guardrails 检查直接执行 LLM Chat（Phase 1 交付后）
- 在 guardrails 检查错误中泄露 LLM API key 或内部路径
- 卖家访谈期间写功能代码（Phase 0 约束）
- 删除或绕过已有的 guardrails 检查
- 在真实卖家数据上使用非 stub LLM provider 之前未确认数据合规

## Success Criteria

- [ ] Phase 0: 10 个卖家访谈记录 + GTM 备忘录
- [ ] Phase 0.5: 至少 1 个卖家愿意付费（Concierge 验证）
- [x] Phase 1: guardrails L1+L3 接入所有 LLM 调用路径
- [x] Phase 1: guardrails 测试通过，无 vet issue
- [x] Phase 1: EventBus correlation 层完成
- [x] Phase 1: Scheduler 健康检查 + 重试队列完成
- [x] Phase 1: ToolBridge 完整性验证完成
- [x] Phase 1: Approval/Audit 收尾完成
- [ ] P3（原 Phase 2）: ~~第一垂直功能上线~~ 降级。当前优先级：真实上架闭环 → 用户获取
- [ ] 至少 1 个用户表示愿意为正式产品付费

## Open Questions

1. **LLM 数据源**：P3（原 Phase 2）需求分析工具的数据从哪来？Amazon Review API？爬虫？卖家手动输入？降级后待后续评估。
2. **LLM Gateway**：当前使用 stub provider。短期内应继续使用 stub（无额外 LLM 成本），有真实用户后再切换。
3. **EventBus correlation**：issue 列在 Phase 1 但优先级靠后。是否在 Phase 1 guardrails 之后自动进入？
4. **定价模式**：SaaS/按结果/免费增值？待 Phase 0 访谈验证。
5. **供应链整合**：AI 代替工厂沟通在实操中可行？待 Phase 0.5 Concierge 试点验证。

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-07-09 | #321 覆盖旧方向 | 从「可信 AgentOS 执行门禁收口」转向「AI 决策质量平台——帮卖家选得更准」 |
| 2026-07-09 | Phase 0 访谈 + Phase 1 guardrails 并行执行 | guardrails 接入已在代码知识范围内，可独立完成；Phase 0 不需开发时间 |
| 2026-07-09 | 完整路线图 spec 覆盖 Phase 0-2 | 一个文档对齐所有阶段方向 |
| 2026-07-09 | P0-P1 执行门禁视为已完成基础版 | PR #276 已合并，Guardrails L1-L5 都有实现但 LLM 入口未接入 |
| 2026-07-10 | Phase 2 降级至 P3 | 没有卖家访谈验证选品方向。优先完成真实上架闭环 + 用户获取。 |
