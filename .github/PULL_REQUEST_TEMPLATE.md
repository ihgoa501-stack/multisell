## 改了什么

<!-- 简要描述这次 PR 修改的内容 -->

## 风险等级

- [ ] **低 (low)** — 文档、注释、测试、UI 文案、纯重构（行为不变）、CI 配置
- [ ] **中 (medium)** — 新增只读 API、Agent 建议逻辑、非敏感 CRUD、仪表盘、报表
- [ ] **高 (high)** — 价格/库存/订单变更、平台发布、退款、权限修改、AI 执行逻辑、数据迁移

## 发布通道

- [ ] **Read-only (fast track)** — 仅文档/UI/纯重构，单 Reviewer 即可
- [ ] **Suggestion (standard)** — 只读功能，Test Green + 代码所有者
- [ ] **Approval-required (slow)** — 写操作/高风险，Business Verified + Owner 审批
- [ ] **Production-write (slowest)** — 生产写回/支付/权限，Beta Accepted + Owner Decision Log

## 测试

- [ ] `scripts/verify_all.sh` 已通过，或下方逐项列明未运行/失败原因
- [ ] `cd backend-go && go build ./...` 通过
- [ ] `cd backend-go && go vet ./...` 通过
- [ ] `cd backend-go && go test ./...` 通过
- [ ] `cd frontend-next && npm test` 通过
- [ ] `cd frontend-next && npm run lint` 通过
- [ ] `cd frontend-next && npm run build` 通过
- [ ] `cd frontend-next/e2e && npm run e2e` 通过，且主链路未 skip
- [ ] 涉及的模块有新增或更新测试

## 验收状态

- [ ] Dev Done：代码已实现
- [ ] Test Green：自动化检查全绿
- [ ] Business Verified：业务闭环已验证
- [ ] Beta Accepted：允许进入受控试运行
- [ ] 本 PR 不声称业务验收或 Beta 验收完成

## 文档同步

- [ ] `AGENTS.md` 已更新（如涉及模块名/路径/API 变更）
- [ ] `CLAUDE.md` 已更新（如涉及）
- [ ] `docs/INDEX.md` 已更新（如涉及新模块或路径变更）
- [ ] `docs/BETA_ACCEPTANCE_REPORT.md` 已更新（如声称 Beta/试运行验收）
- [ ] `docs/KNOWN_ISSUES.md` 已更新（如存在 FAIL/BLOCKED/SKIPPED 验收项）
- [ ] 本 PR 不涉及文档变更

## 风险

- [ ] 包含 breaking change（API、数据库、配置不兼容）
- [ ] 需要数据库 migration（请注明 migration 编号）
- [ ] 涉及敏感操作（价格/库存/订单/发布/权限修改）
- [ ] 高风险动作已证明：未审批不能执行、服务端身份绑定、审计日志、敏感字段脱敏、dry-run 不写生产

## 验证步骤

<!-- 如果 Reviewer/AI 要验收这个功能，应该怎么操作？ -->
<!-- 例如：打开 XXX 页面 → 点击 YYY → 期望看到 ZZZ -->

## 未通过 / 未运行项

<!-- 只能使用 PASS / FAIL / SKIPPED / BLOCKED / NOT RUN。禁止写 “PASS with known issue”。 -->
<!-- 对 FAIL/BLOCKED/SKIPPED 项，必须在 docs/KNOWN_ISSUES.md 写 owner / deadline / impact。 -->
