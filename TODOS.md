# TODOs

## Dual-Product Cathedral — Phase 3 Activation

### Capital Governor Enforcement

**What:** Upgrade the Phase 2 advisory/shadow Capital Governor into a hard enforcement boundary for every pre-registered spend channel.

**Why:** Advisory warnings cannot guarantee the experiment loss ceiling when purchases, advertising, inventory, logistics, fees, returns, labor or manual/off-platform spend remain outside enforcement.

**Context:** Accepted during `/plan-ceo-review` on 2026-07-11. Phase 2 must enumerate the full spend universe and show `external_spend_unenforced` for every unknown or unintegrated channel. Enforcement may start per workspace only after coverage is 100%, unknown spend is zero, external drift reconciliation works, and partial-failure/kill-switch drills pass. Any newly discovered manual spend invalidates the ceiling and freezes the next tranche.

**Effort:** L human / M with CC+gstack
**Priority:** P1
**Depends on:** Phase 2 advisory Capital Governor, all spend entry integrations, Finance Reviewer, external drift reconciliation and failure drills.

### Cross-Customer Do-Not-Launch Aggregation

**What:** Promote explicitly licensed reject, abstain and failure patterns from customer-private libraries into a rights-cleared cross-customer negative-evidence product.

**Why:** Private memory prevents one customer repeating mistakes; safe cross-customer patterns can become a differentiated Intelligence data asset that helps customers avoid already-known failure modes.

**Context:** Accepted during `/plan-ceo-review` on 2026-07-11. Phase 2 remains customer-private. Promotion requires multiple independent contributors, explicit purpose licenses, versioned quasi-identifier and attack-model configuration, reidentification testing, lineage deletion, unilateral consent withdrawal and downstream dossier invalidation. Never expose customer, SKU, supplier, contract-price or raw operating data.

**Effort:** L human / M with CC+gstack
**Priority:** P2
**Depends on:** Real private-library usage, multiple authorized sources, Data/Legal Reviewer and verified deletion/revocation propagation.

### Automated Evidence Warranty Remedies

**What:** Automate low-risk, low-value Warranty remedies such as service credits, replacement research or contract-defined refunds while keeping high-value and disputed claims human-reviewed.

**Why:** Manual Warranty operations validate liability boundaries, but become an SLA and gross-margin bottleneck as paid dossier volume grows.

**Context:** Accepted during `/plan-ceo-review` on 2026-07-11. Phase 2 uses human triage, one appeal, Legal escalation and fully audited remedies. Automation starts only after real claims establish stable defect classes, fraud patterns, mean liability cost and Legal/Finance-approved caps. Roll out one defect class at a time with a canary and kill switch.

**Effort:** M human / S–M with CC+gstack
**Priority:** P2
**Depends on:** Real Warranty claims, stable typed Error/Defect Registry, liability caps, fraud controls and Finance/Legal approval.

### Capped Real Outcome Fee Pilot

**What:** Convert shadow outcome billing into one capped, externally paid result-fee contract with a Design Partner.

**Why:** Outcome-aligned pricing can reduce first-purchase friction and prove economic value, but charging before attribution and adjustment rules reconcile would create damaging invoice disputes.

**Context:** Accepted during `/plan-ceo-review` on 2026-07-11. Phase 2 freezes baseline, attribution window, adjustments, cap and dispute SLA, then calculates but does not invoice shadow fees. A real pilot unlocks only after at least three finalized experiments reconcile inside the pre-registered tolerance with zero unresolved disputes. The recommendation system must never read contract fee rates.

**Effort:** M human plus 1–3 real decision cycles / S–M with CC+gstack
**Priority:** P2
**Depends on:** Three finalized experiments, Decision Flight Recorder, Finance/Tax/Legal review and one customer contract.

### Public API, Pricing and Self-Serve Onboarding

**What:** Progress from invite-only Design Partners to an invite-only API, public packages, self-service billing, workspace creation and platform connection.

**Why:** High-touch delivery validates value but cannot create repeatable low-marginal-cost software distribution.

**Context:** Accepted during `/plan-ceo-review` on 2026-07-11. Do not build public SaaS surfaces while customers still need founder-operated workflows. Unlock invite-only API only after independent completion, natural repeat/renewal, positive account gross margin, bounded support, tenant isolation, abuse controls, credential rotation and offboarding/deletion drills pass. Public registration and pricing require a separate Gate after API design partners succeed.

**Effort:** L–XL human / M–L with CC+gstack
**Priority:** P2
**Depends on:** S4 retention/economics, stable versioned API, billing, abuse prevention, support operations and security/deletion verification.

## Metabolism M1 — Phase 1 Migration

- **What:** Migration for `metabolism_log` table + `event_outbox` indexed columns
- **Why:** M1 scores records and needs a table to store score results. `event_outbox` needs `excreted_at` (tagged for deletion) and `excretion_reason` (why it was scored/excreted) columns for scheduled cleanup in Phase 2.
- **Context:** Added during /plan-eng-review on 2026-06-26. Phase 1 is dry-run (no actual deletion), but the schema should ship from day 1 so Phase 2 doesn't need a second migration.
- **Action:** `backend-go/migrations/XXX_add_metabolism.sql` — CREATE TABLE metabolism_log + ALTER TABLE event_outbox ADD COLUMNS.
- **Depends on:** Design approval of MetabolismModel fields (see design doc).
- **Blocked by:** Nothing.

## UI Redesign — 未完成项目

### P5: 跨页一致性检查

- **What:** 检查 P2-P4 三个页面（prelisting、dashboard、agentos）之间是否存在样式 drift — 边距不一致、字体不一致、颜色偏差、共享组件使用方式不一致。
- **Why:** P1 创建了 PageContainer / AgentDecisionPanel 等共享组件，P2-P4 可能各自改写了样式造成漂移。
- **Action:** 逐页面对比 padding、font-size、color token、AgentDecisionPanel 使用方式。输出差异清单并修复。
- **Blocked by:** P1-P4 完成（已交付）。

### Mock 数据 → 真实 API 迁移

- **What:** 三个新页面目前使用硬编码 mock 数据。替换为从后端 API 获取真实数据。
  - `/decision/prelisting`：用 `/api/v1/decision` 系列接口
  - `/dashboard`：用 `/api/v1/dashboard/overview`
  - `/agentos`：用 `/api/v1/agentos`、`/api/v1/agentos/work-items`
- **Why:** Mock 数据不能用于生产。
- **Action:** 替换每个页面的 `mockSkus`/`mockPriority`/`mockWorkItems` 为 React Query 的 `useQuery` 调用后端 API。保留 mock 作为 fallback。
- **Depends on:** 后端对应 API endpoint 已就绪。
- **Blocked by:** `backend-go` 对应 handler 的验证。

### 旧 40+ 页面在新 layout 中的视觉验证

- **What:** P1 将 layout 从三栏改为四栏，20+ 原有 Ant Design CRUD 页面自动被新 shell 包裹。需要逐个验证它们在新 shell 中的渲染效果——宽度、滚动、边框、嵌套。
- **Why:** 旧页面可能在窄的中心区域显示异常（表格被截断、按钮重叠等）。
- **Action:** 在本地 dev 环境中逐一访问旧页面，截图检查布局。修复发现的异常。
- **Blocked by:** P1 完成（已交付）。

### 组件测试覆盖

- **What:** P1 新增的共享组件（DomainSidebar、AgentDecisionPanel、DecisionCard、RiskBadge、PageContainer）缺少测试，需要至少覆盖 loading/empty/error/normal 四种状态。
- **Why:** 按 review 决策要求全量测试覆盖。
- **Action:** 为每个新增组件创建 `*.test.tsx` 文件，使用 Vitest + React Testing Library 渲染四种状态并断言。
- **Blocked by:** P1 完成（已交付）。

### Ant Design 6 暗色 Token 收敛

- **What:** P1 在 `AntdProvider.tsx` 中设置了基础 token 覆盖（colorBgBase、colorBgContainer），但部分 Ant Design 组件的暗色适配可能不完整（如 Table 表头、Modal 遮罩、Dropdown 菜单）。
- **Why:** 如果 AntD 默认暗色 token 和新 DESIGN.md 不一致，会出现肉眼可见的对比度差异。
- **Action:** 在浏览器中逐组件检查 Table、Modal、Dropdown、Select、DatePicker 的暗色渲染，补充 token 覆盖。
- **Blocked by:** P1 完成（已交付）。
