# Implementation Plan: 凌镜 LingMirror — 可信 AgentOS 执行门禁收口

## Overview

当前凌镜已完成全栈技术迁移（v0.3.0.0），40+ 领域模块和 15 个 Agent。下一阶段从"增加模块"转为"执行门禁收口"——确保安全执行路径可靠、审计可追溯、Owner 可控制。

---

## Architecture Decisions

1. **每次一门禁，一次一个垂直切面** — 不并行推进 6 个 P0/P1 项，避免相互依赖和回滚范围膨胀。
2. **后端先于前端** — 先把后端门禁逻辑做实（可测试、可审计），再为前端提供统一的确认 UX。
3. **测试即契约** — 每项门禁必须有一个可以独立运行的测试证明它工作，不依赖手工 QA。
4. **单步可逆** — 每项变更不做大范围重构，改一两个文件就可以提供可检测的增量价值。
5. **从现有代码验证开始** — 先确认当前 EventBus/Scheduler 生命周期是否真的有问题，而不是凭文档假设去"修复"。

---

## Task List

### Phase 0: 基线验证（确认当前状态）

**为什么：** 所有 CURRENT_DIRECTION 中描述的"问题"可能是旧的或已被修复的。
先验证再改，避免为不存在的问题引入改动。

#### Task 0.1: 验证 EventBus/Scheduler 生命周期

**Description:** CURRENT_DIRECTION 说 EventBus context "被 defer 在了 router 初始化中"。
先写一个测试验证 bus 和 scheduler 是否在整个服务生命周期内保持运行。

**Acceptance criteria:**
- [x] 阅读 `NewRouter` 代码确认 `busCancel` 存储方式 — **已完成，busCancel 存储在 App.Cancel，没有 defer**
- [ ] 阅读 `main.go` 确认 shutdown 路径 — **已完成，scheduler.Shutdown → Cancel() → Bus.Stop 顺序正确**
- [ ] 写一个测试：启动 bus → publish → subscriber 收到事件 → stop → 确认停止
- [ ] 写一个测试：scheduler tick 发布事件 → bus subscriber 收到

**Verification:**
- [ ] Tests pass: `go test ./internal/platform/eventbus/...`
- [ ] Tests pass: `go test ./internal/platform/scheduler/...`
- [ ] No lifecycle issues found, or issue precisely documented

**Files likely touched:**
- `backend-go/internal/platform/eventbus/bus_test.go` (add lifecycle test)
- `backend-go/internal/platform/scheduler/scheduler_test.go` (add lifecycle test)
- `backend-go/internal/httpx/router.go` (only if confirmed issue)

**Estimated scope:** S (1-2 files)

---

#### Task 0.2: 验证批准/执行身份绑定

**Description:** 检查当前 `ReviewerFromCtx` 是否已经从 JWT 提取了正确的用户身份。

**Acceptance criteria:**
- [x] 已确认 `ReviewerFromCtx` 从 `c.Get("username")` 优先提取，其次是 `c.Get("user_id")`——来自 JWT 中间件
- [x] 已确认 `ActionDecisionInput` 结构体不包含 `operator` 字段——**客户端无法伪造**（需要确认 ActionDecisionInput 结构体字段）

**Verification:**
- [ ] 写测试：`POST /ai/actions/:id/approve` 验证 operator 来自 JWT，不是请求 body
- [ ] `go test ./internal/ai/...`

**Files likely touched:**
- `backend-go/internal/ai/handler_test.go`
- `backend-go/internal/ai/types.go` (检查 ActionDecisionInput 字段)

**Estimated scope:** S (1-2 files)

---

### Checkpoint: Baseline
- [ ] All Phase 0 tests pass
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] We now know which P0 items are real and which are already fixed
- [ ] **Review with human before proceeding**

---

### Phase 1: P0 — 统一执行门禁（ExecuteAction 安全加强）

#### Task 1.1: ExecuteAction 门禁链加固

**Description:** 当前 `ExecuteAction` 已经有一些检查（catalog validation、approval check、command dispatch）。
加固方向：
1. 确保每次 `execute` 都通过 audit 写 operationlog
2. 确保幂等守卫：已 `executed`/`failed` 的 action 不可再次 `execute`
3. 确保 executed_by 记录的是服务器端身份（不从客户端接受）

当前代码状态分析（已完成）：
- `ExecuteActionService` 已有 `s.cat.ValidateProduction()` 检查
- 已有状态转换守卫：`suggested/approved → executing → executed/failed`
- 已有 `RequiresApproval` 检查
- `executed_by` 来自 `common.ReviewerFromCtx(c)`（JWT 提取）✅
- 缺：operationlog 审计写入

**Acceptance criteria:**
- [ ] ExecuteAction 在 dispatch 前后写入 operationlog
- [ ] 已 `executed` 或 `failed` 的 action 再次 execute 返回明确的错误（已部分实现—状态转换检查）
- [ ] 编写 ExecuteAction 门禁测试：需要批准但未批准的 action 执行失败
- [ ] 编写 ExecuteAction 幂等测试：已执行的 action 不可再次执行

**Verification:**
- [ ] `go test ./internal/ai/...` passes
- [ ] Audit log entries exist for execute operations

**Files likely touched:**
- `backend-go/internal/ai/service.go` (add audit logging)
- `backend-go/internal/ai/service_test.go` (add test coverage)

**Estimated scope:** M (3-4 files)

---

#### Task 1.2: 确认 ActionDecisionInput 不从客户端接受 operator 字段

**Description:** 回顾 ActionDecisionInput 结构体，确保 approve/reject/execute 的 operator 始终来自服务器端 JWT。

当前已确认：
- `ApproveAction` handler 传入 `common.ReviewerFromCtx(c)` ✅
- `RejectAction` handler 同上 ✅
- `ExecuteAction` handler 同上 ✅

**Acceptance criteria:**
- [ ] 确认 ActionDecisionInput 结构体没有 operator 字段（客户端不能提交）
- [ ] 否则，加入安全过滤：忽略客户端提供的 operator

**Files likely touched:**
- `backend-go/internal/ai/types.go` (检查 ActionDecisionInput)
- `backend-go/internal/ai/handler.go` (过滤客户端 operator 如果存在)

**Estimated scope:** XS (1 file)

---

### Checkpoint: Phase 1
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] ExecuteAction 已具有完整的审批→审计→幂等链
- [ ] **Review with human before proceeding**

---

### Phase 2: P0 — Admin Approval RBAC 绑定

#### Task 2.1: Approval 服务集成 RBAC 检查

**Description:** 确保 approve/reject 操作受 RBAC 保护，只有具有批准权限的用户可以操作。

**Acceptance criteria:**
- [ ] Handler 层添加 `middleware.RequirePermission("approval:approve")` 或类似检查
- [ ] 或 service 层注入当前用户角色做检查
- [ ] 编写测试验证无权限用户无法批准

**Verification:**
- [ ] `go test ./internal/domain/approval/...` passes
- [ ] `go test ./internal/rbac/...` passes

**Files likely touched:**
- `backend-go/internal/ai/handler.go` (add permission check)
- `backend-go/internal/domain/approval/service.go` (add RBAC integration if needed)
- `backend-go/internal/rbac/...` (if new permission needed)

**Estimated scope:** M (3-5 files)

---

### Checkpoint: Phase 2
- [ ] `go test ./...` passes
- [ ] Approval 的身份绑定和 RBAC 集成完成
- [ ] **Review with human before proceeding**

---

### Phase 3: P1 — 审计日志敏感字段脱敏

#### Task 3.1: 审计日志脱敏过滤器

**Description:** 写入 operationlog 前，对 `Content` 字段中的敏感信息（token、password、secret、API key）进行脱敏。

**Acceptance criteria:**
- [ ] 定义一个通用的脱敏函数（可使用正则匹配敏感字段名/值）
- [ ] 在 Audit 中间件和所有显式调用 `operationlog.Service.Log` 的地方应用脱敏
- [ ] 编写测试验证包含 `password=secret123` 的请求日志中变为 `password=***`

**Verification:**
- [ ] `go test ./internal/domain/operationlog/...` passes
- [ ] Test proves sensitive field is redacted

**Files likely touched:**
- `backend-go/internal/domain/operationlog/service.go` (add redaction)
- `backend-go/internal/domain/operationlog/redact.go` (new file — redaction logic)
- `backend-go/internal/domain/operationlog/redact_test.go` (new test file)
- `backend-go/internal/httpx/middleware/audit.go` (apply redaction)

**Estimated scope:** M (3-4 files)

---

### Checkpoint: Phase 3
- [ ] `go test ./...` passes
- [ ] Audit 日志不会泄露敏感字段
- [ ] **Review with human before proceeding**

---

### Phase 4: P1 — 外部平台写安全

#### Task 4.1: 平台集成 dry-run/sandbox/production 模式

**Description:** Ozon/Shopee 等外部平台的发布、改价、改库存操作需要区分 dry-run、sandbox 和 production。

当前代码状态（已确认）：
- `internal/domain/integrations/` 已定义 `PlatformAdapter` 接口
- Ozon/Shopee 适配器已实现

**Acceptance criteria:**
- [ ] 在 `PlatformAdapter` 或集成服务层添加 mode 参数（dry-run | sandbox | production）
- [ ] dry-run 模式打印/记录将要执行的操作但不发送到外部平台
- [ ] sandbox 模式使用平台的沙箱 API endpoint（如果有）
- [ ] 无真实凭证时不执行 production 写操作
- [ ] 操作日志记录 mode

**Verification:**
- [ ] `go test ./internal/domain/integrations/...` passes
- [ ] 测试验证 dry-run 不发送真实请求

**Files likely touched:**
- `backend-go/internal/domain/integrations/adapter.go` (add mode to adapter if needed)
- `backend-go/internal/domain/integrations/service.go` (add mode check)
- `backend-go/internal/domain/integrations/types.go` (add mode type)

**Estimated scope:** M (3-5 files)

---

### Checkpoint: Phase 4
- [ ] `go test ./...` passes
- [ ] 平台写操作已通过 mode 控制
- [ ] **Review with human before proceeding**

---

### Phase 5: P1 — 前端高风险动作确认 UX

#### Task 5.1: 共享确认弹窗组件

**Description:** 创建一个统一的 `HighRiskConfirmDialog` 组件，用于 publish、approve、execute、price、inventory、refund 等高风险操作。

必须展示的信息：
- 目标对象
- 风险等级
- Before/After 值（如果有）
- 环境模式（production / sandbox / mock）
- 审批要求
- 预期后果
- 审计去向
- 回滚/恢复说明（当可用时）

**Acceptance criteria:**
- [ ] 创建 `src/components/ui/HighRiskConfirmDialog.tsx`
- [ ] 支持 Ant Design 的 `Modal.confirm` 风格或自定义弹窗
- [ ] 集成到 publish/approve/execute 操作中（至少一个示例）
- [ ] `npm run build` 通过

**Verification:**
- [ ] `npm test` passes
- [ ] `npm run build` succeeds
- [ ] Manual check: approval action shows risk dialog with expected fields

**Files likely touched:**
- `frontend-next/src/components/ui/HighRiskConfirmDialog.tsx` (new)
- `frontend-next/src/lib/api-client.ts` or specific page hooks

**Estimated scope:** M (3-4 files)

---

### Checkpoint: Phase 5
- [ ] `npm test` passes
- [ ] `npm run build` passes
- [ ] High-risk action confirmation pattern works end-to-end
- [ ] **Human review of the UX before finalizing**

---

### Phase 6: 全局验证与文档更新

#### Task 6.1: 全量测试验证

**Description:** 所有门禁修改完成后，运行全量测试并清理遗留问题。

**Acceptance criteria:**
- [ ] `cd backend-go && go test ./...` — 全绿
- [ ] `cd backend-go && go vet ./...` — 无输出
- [ ] `cd frontend-next && npm test` — 全绿
- [ ] `cd frontend-next && npm run build` — 通过
- [ ] 烟雾测试 `backend-go/scripts/smoke_test.sh` — 通过（或问题明确记录）

**Files likely touched:**
- None (verification only)

**Estimated scope:** S (documentation updates only)

---

#### Task 6.2: 文档更新

**Description:** 更新 `docs/PROJECT_STATUS.md` 记录本次门禁收口的状态。

**Acceptance criteria:**
- [ ] PROJECT_STATUS.md 更新当前验证状态为 ✅ 已通过
- [ ] 记录本次完成的 P0/P1 条目
- [ ] 更新 SPEC.md 以反映实现的变更

**Files likely touched:**
- `docs/PROJECT_STATUS.md`
- `docs/SPEC.md` (mark resolved items)

**Estimated scope:** S (1-2 files)

---

### Checkpoint: Final
- [ ] All acceptance criteria met
- [ ] `go test ./...` + `npm test` + `npm run build` all pass
- [ ] Documentation updated
- [ ] Ready for PR / merge review

---

## Dependency Graph

```
Phase 0 (Baseline verification)
├── Task 0.1: EventBus/Scheduler lifecycle verify
├── Task 0.2: Approval identity verify
│
├── Phase 1 (Execution gate) — independent of Phase 2
│   └── Task 1.1: ExecuteAction gate chain
│   └── Task 1.2: ActionDecisionInput operator check
│
├── Phase 2 (RBAC binding) — independent of Phase 1
│   └── Task 2.1: Approval RBAC integration
│
├── Phase 3 (Audit redaction) — independent of all
│   └── Task 3.1: Redact sensitive fields
│
├── Phase 4 (Platform write safety) — prefers Phase 1 done
│   └── Task 4.1: Dry-run/sandbox/production modes
│
├── Phase 5 (Frontend UX) — prefers Phase 1-2 done (UI calls backend)
│   └── Task 5.1: HighRiskConfirmDialog
│
└── Phase 6 (Global verification)
    └── Task 6.1-6.2: Tests + docs
```

**可并行的阶段：** Phase 1、2、3 互不依赖，可并行推进。
**必须顺序的：** Phase 4 建议 Phase 1 完成后进行（复用执行门禁模式）。Phase 5 建议后端接口确认后。

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Phase 0 发现 EventBus/Scheduler 无问题** | Low — 白费时间？不，确认状态本身就是价值 | 如无问题，Phase 0 结论就是"已验证无问题"——不需要修复 |
| **Phase 1 改 ExecuteAction 引入回归** | Med — 影响现有 AI action 执行 | 测试覆盖所有状态转换路径再改 |
| **Phase 2 RBAC 改造影响现有 API** | Med — 授权失败可能导致功能不可用 | 测试验证有权限用户可以正常操作 |
| **Phase 3 脱敏过强漏掉业务数据** | Low — 审计日志变模糊 | 只对明确敏感字段脱敏（password, secret, token, key），不脱敏通用 content |
| **Phase 5 前端确认弹窗与 Ant Design 6 不兼容** | Low — 样式不一致 | 使用 Ant Design 内置 Modal.confirm，不做自定义重渲染 |
| **并行 Phase 1/2/3 导致 merge 冲突** | Med — 三批人改不同文件 | 用 worktree 隔离，每 Phase 独立 PR |

---

## 执行节奏建议

```
推荐执行顺序（一次一门禁）：
  1. Phase 0（基线验证）—— 1 个 session
  2. Phase 1（执行门禁）—— 1-2 个 sessions     ← 最高优先级
  3. Phase 2（RBAC 绑定）—— 1 session
  4. Phase 3（审计脱敏）—— 1 session
  ——— checkpoint: 后端 3 个 P0/P1 完成 ———
  5. Phase 4（平台写安全）—— 1 session
  6. Phase 5（前端 UX）—— 1 session
  7. Phase 6（全局验证）—— 1 session
```

---

## Open Questions

1. **Phase 0 验证结果** — 如果发现 EventBus/Scheduler 生命周期和审批身份绑定已经是正确的，是记录为"已验证通过"还是仍然修改代码？
2. **ActionDecisionInput 结构体** — 需要确认其字段列表。如果现有的确没有 operator 字段，Task 1.2 就是验证通过即可。
3. **前端覆盖范围** — 是做一个通用确认组件并集成到 1 个操作中演示，还是所有 approve/execute/publish 操作一次性替换？
