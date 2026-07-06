# Spec: 目标1 — 恢复工程可信度

## Objective
修复项目中 "报告说全绿、实际不全绿" 的状态 — 消除编译错误、测试失败、merge conflict，让构建管道可信。

## Commands
```bash
cd backend-go
go build ./...                    # 检查编译
go vet ./...                      # 静态分析
go test ./...                     # 跑全部测试
```

## Project Structure
- `backend-go/internal/common/types.go` — UserIDFromCtx 去重
- `backend-go/internal/ai/` — 6 个文件有 merge conflict 标记
  - handler.go (3 conflicts)
  - service.go (6 conflicts)
  - orchestrator.go (3 conflicts)
  - routes.go (1 conflict)
  - model.go (1 conflict)
  - ai_test.go (5 conflicts)
- `AGENTS.md` — 更新测试状态

## Success Criteria
- [x] `UserIDFromCtx` 重复定义已修复 ✅
- [ ] `grep -rn "<<<<<<< " internal/ai/` 返回 0
- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] `go test ./...` 通过
- [ ] AGENTS.md 中测试状态更新

## Boundaries
- Always: 跑 `go test ./...` 和 `go vet ./...` 确认
- Ask first: 修改不属于修复范围的新文件
- Never: 删除测试、跳过验证步骤
