# 凌镜 LingMirror 测试说明

> 更新时间：2026-06-24
> 范围：当前活跃新栈 `backend-go/` + `frontend-next/`
> 状态：历史验证报告

## 使用说明

这是 2026-06-24 的历史验证报告，不再作为当前测试状态的事实源。
当前验证状态以 [PROJECT_STATUS.md](PROJECT_STATUS.md) 的“当前事实快照”为准。

如果需要交付或合并当前代码，应重新运行相关检查，并把新结果写入
[PROJECT_STATUS.md](PROJECT_STATUS.md)。

## 历史验证快照

| 检查 | 命令 | 结果 |
|---|---|---:|
| 后端测试 | `cd backend-go && go test ./...` | 通过 |
| 后端 vet | `cd backend-go && go vet ./...` | 通过 |
| 前端单测 | `cd frontend-next && npm test` | 通过，75 tests |
| 前端生产构建 | `cd frontend-next && npm run build` | 通过 |
| 前端 lint | `cd frontend-next && npm run lint` | 失败 |

## 当时已知问题

### 前端 lint

当前 lint 主要问题：

- `@typescript-eslint/no-explicit-any`
- `@typescript-eslint/no-unused-vars`
- `react-hooks/set-state-in-effect`

典型文件：

- `frontend-next/src/components/auth/AuthGuard.tsx`
- `frontend-next/src/app/(auth)/login/page.tsx`
- `frontend-next/src/app/(main)/decision/prelisting/page.tsx`
- `frontend-next/src/app/(main)/listing-tasks/[id]/page.tsx`
- `frontend-next/src/app/(main)/settlement/[id]/page.tsx`

### API 路径一致性

后端业务 API 统一为 `/api/v1/*`。前端默认 base 是 `http://localhost:8080/api`，因此 `apiClient` 调用应使用 `/v1/*`。

当时仍需修正的调用包括：

- `/ai/actions`
- `/policy/rules`
- `/evolution/nudges`
- `/trust-scores/summary`

这些路径一致性问题后续已有修复记录；不要把本段当作当前缺陷清单。

## 当时测试覆盖

### Go 后端

当时记录的 Go test 文件：

- `backend-go/internal/ai/ai_test.go`
- `backend-go/internal/auth/auth_test.go`
- `backend-go/internal/domain/aftersales/aftersales_test.go`
- `backend-go/internal/domain/order/order_test.go`
- `backend-go/internal/domain/settlement/settlement_test.go`
- `backend-go/internal/domain/shipping/shipping_test.go`
- `backend-go/internal/platform/eventbus/topic_test.go`
- `backend-go/internal/rbac/rbac_test.go`

### Next 前端

当时 Vitest 覆盖集中在共享 UI / CRUD 组件：

- `frontend-next/src/components/crud/__tests__/CrudListPage.test.tsx`
- `frontend-next/src/components/ui/__tests__/*.test.tsx`

## 当时建议的下一步

1. 修复 `frontend-next` lint。
2. 为 API 路径一致性补 smoke/e2e 检查。
3. 为高风险 Go domain 模块补 focused tests：发布、库存、财务、AgentOS action chain。
4. 为关键业务页面补 API mock + React component tests。

## 历史测试报告

2026-06-22 以前的 Chrome DevTools/Vue 页面测试报告属于旧前端阶段，只能作为历史参考，不能作为当前 `frontend-next/` 的验收结论。
