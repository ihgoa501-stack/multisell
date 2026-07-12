# 小Q Capability Contract

> 生效日期：2026-07-12
> 适用范围：所有希望被小Q读取、解释、建议或调用的凌镜功能
> 产品边界：小Q只服务 Owner，不是外部 SaaS 或通用 Agent 平台

## 1. 目标

小Q是凌镜唯一面向 Owner 的经营 Agent。它不能直接访问任意数据库表或内部函数；业务模块必须通过登记的 Capability（能力接口）向小Q开放能力。

“功能已实现”和“功能已接入小Q”是两个独立事实：

- 模块代码、API 或页面存在：`implemented`；
- Capability、权限、审计、失败处理和小Q回归测试齐全：`xiao_q_support=active`；
- 只有规划或接口占位：`planned`，不得显示为小Q可用能力。

## 2. 唯一身份

- 稳定 Agent ID：`xiao_q`
- 显示名：`小Q`
- 当前用户：Owner
- 默认模式：`read_only`
- 小Q输出默认是 `inferred`；模型回答不能把证据升级为 `actual`。

不得再为一个业务模块建立另一个 Owner-facing Agent。内部确定性服务、隔离研究 run 或历史 Agent 可以作为实现细节，但不形成第二个产品入口。

## 3. Capability 必填字段

每项能力必须登记：

```text
id
version
domain
description
input_schema
output_schema
risk
required_permission
approval_required
execution_modes
external_side_effects
idempotency_required
evidence_policy
timeout_seconds
retry_limit
audit_action_type
owner_explanation
status
```

`status` 只能是：

- `active`：代码、权限和测试均可用；
- `deprecated`：仍可追溯但不再用于新请求；
- `deferred`：明确决定暂不接入；
- `disabled`：因安全、故障或成本被停用。

## 4. 风险级别

| 类型 | 行为 | 默认要求 |
|---|---|---|
| `read` | 读取已授权的内部或外部数据 | Owner 范围校验、Trace |
| `suggest` | 生成草案或建议，不改变事实 | 结构化输出、证据引用 |
| `mutate` | 修改内部业务状态 | RBAC、Command/Service、审计、幂等 |
| `external_write` | 发布、同步或写外部平台 | 独立 Owner 审批、回执、恢复 |
| `financial` | 采购、广告、退款、结算或资金 | 独立 Owner 审批、金额和凭证复核 |

小Q不得使用自由文本、Prompt 或工具描述绕过这些分类。

## 5. 执行规则

1. 小Q只看见与当前 Owner、业务对象状态和权限匹配的 active 能力。
2. 内部 Go 模块优先调用已有领域 Service；不得由小Q直接写业务表。
3. MCP只用于跨进程或第三方连接，仍必须经过相同的权限、审批和审计包装。
4. 高风险批准必须绑定具体 capability、参数摘要、对象版本和幂等键；参数或事实变化后重新审批。
5. Capability、Trace、Evidence 或最终状态写入失败时，不能返回伪成功。
6. mock、stub、unknown 或 inferred 不得通过经营事实闸门。
7. 每次调用必须能回答：为什么调用、读取什么、使用什么证据、花费多少、是否改变状态、谁批准、结果在哪里。

## 6. 新功能同步规则

每个新功能的 Spec、计划或 PR 必须包含：

```text
xiao_q_support: active | deferred | not_applicable
reason:
capabilities:
permissions:
approval_and_audit:
tests:
owner_explanation:
```

- `active`：本次同步交付 Capability 和测试；
- `deferred`：记录未接入原因和重新评估条件；
- `not_applicable`：功能不应暴露给小Q，并解释原因。

缺少该声明不代表必须强行接入，但该功能不得宣称已被小Q调用。

## 7. 验收清单

- [ ] Capability 只调用权威领域 Service/Command；
- [ ] Owner范围和RBAC在服务端校验；
- [ ] 输入输出 schema 有版本；
- [ ] 事实等级、evidence、run和snapshot可追溯；
- [ ] 失败、超时、取消和禁用对Owner可见；
- [ ] 真实模型失败不回退成可信答案；
- [ ] 高风险能力经过审批、审计和幂等；
- [ ] 小Q页面只展示实际 active 能力；
- [ ] 聚焦测试和端到端契约测试通过。

## 8. 当前能力

截至2026-07-12，第一版只允许：

- `demand_case.read`
- `demand_case.decision_card.read`
- `experiment.read`
- `experiment.gate_status.read`
- `sourcing_1688.controlled_draft.read`

经营实验能力只读取当前 Owner 的案件详情、原始事实等级、已有闸门决定、阻断项与终局状态；不新增或核验证据，不执行闸门评估，不改变实验状态。1688能力只读取与当前 Owner、候选市场、实验和不可变快照一致的受控内部草稿；原始采集载荷、平台发布载荷、内部 URI 和未经独立真实性字段支持的图片权利事实不进入模型上下文，且不能发布、采购或批准。其他模块均为 `deferred`，不能解释为小Q已经能够调用整个系统。
