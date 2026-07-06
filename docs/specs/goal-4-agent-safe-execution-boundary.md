# Spec: 目标 4 — 建立 Agent 安全执行边界

## Objective

把 Agent 从「输出文本」推进到「结构化建议 + 风险等级 + 审批状态」。
所有 Agent 的输出必须是结构化的 `UnifiedAction`（持久化）或 `AgentAction`（内存契约），
高风险生产 action 必须经过审批和审计，MOA 聚合输出明确的冲突、建议和风险。

## Acceptance Criteria

1. **Agent 输出结构化 action** — 每个 Agent 的 `Decide()` 返回后，Orchestrator 将其包装为 `UnifiedAction`，不再有自由文本输出
2. **每个 action 有 risk_level、mode、approval_required** — `UnifiedAction` / `AgentAction` 所有字段正确填充
3. **高风险 production action 不能绕过审批和审计** — `Validate()` 验证、`DispatchSafe` 拦截、审计事件发布
4. **MOA 聚合不是字符串拼接** — `synthesize()` 输出结构化冲突列表 + 建议 + 风险评估

## Tech Stack

Go 1.25, GORM, PostgreSQL via `backend-go/`

## Commands

```
cd backend-go && go test ./...   # all tests pass
cd backend-go && go vet ./...    # no lint issues
cd backend-go && go build -o /dev/null ./cmd/server/
```

## Project Structure (changes only)

| File | What changes |
|------|-------------|
| `internal/ai/orchestrator.go` | 确保所有 autonomy 等级都创建 UnifiedAction；advisory 级创建 auto-approved action |
| `internal/ai/moa.go` | 替换 `synthesize()` 从 `fmt.Sprintf` 为结构化输出；增强 conflict detection |
| `internal/ai/service.go` | 补充审计发布 |
| `internal/agent/impl/agent.go` | Agent 接口调整 |

## Code Style

语法和命名遵循项目现有风格。新增的结构体使用 JSON 标签，与 `UnifiedAction` 风格一致。

## Testing Strategy

- 不新增测试文件；在已有测试文件中追加用例
- 现有 `action_test.go` 已有完善的 DispatchSafe + Validate 测试 → 追加 AgentAction 创建测试
- MOA 测试: 如果 `moa_test.go` 不存在，在 `moa.go` 同目录新建。验证 synthesize 返回结构化 JSON，不包含 `fmt.Sprintf` 拼接
- 所有已有测试必须保持绿色

## Boundaries

- **Always:** 每个 Agent 执行后必须产生 UnifiedAction；高风险 action 必须触发审批创建
- **Ask first:** 修改 Agent 接口 `Decide()` 签名；新增外部依赖
- **Never:** 移除已有的审批/审计守卫逻辑；降低现有测试覆盖率

## Success Criteria

```
cd backend-go && go test ./...    # PASS
cd backend-go && go vet ./...     # PASS

# Manual verification:
# 1. A1 ProductScout "product_scout" 决策后产生 UnifiedAction
# 2. MOA Coordinator 的 synthesize() 输出含 conflicts/建议/risk_level 的结构化对象
# 3. 高风险 production action 被 DispatchSafe 拒绝（无 approval）
# 4. 高风险 production action 有 approval 时通过
```
