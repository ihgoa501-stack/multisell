# 凌镜经营学习链路代码审计

> 审计日期：2026-07-13
> 审计范围：`事实 → AI 建议 → Owner 决定 → 执行 → 结果观察 → 下一次建议`
> 审计方式：只读代码、迁移、页面与聚焦自动测试核验；未连接生产数据库或外部平台
> 对应路线：[ADR-001 第 6、7 纵向单元](../decisions/ADR-001-owner-complete-commerce-platform.md)

## 结论

凌镜**已经存在一条经营学习链的工程骨架**，不是“完全没有这种架构”：`businessdecision` 保存冻结事实、AI 建议与 Owner 决定，`businessfeedback` 保存受控行动、观测与下一动作建议，小Q能够读取决策案卷并保存一条 `inferred` 建议。

但当前只能判定为：

- `implemented / automated_verified`：分层记录、Owner 隔离、冻结哈希、精确授权、审批、幂等、审计、失败待对账和追加式记录已有代码与聚焦测试。
- `not established`：一条真实、可信、可重复的“经营学习循环”尚未成立。
- `unknown`：没有真实外部行动、行动后的真实经营结果或 Owner 长期使用记录。

主要原因不是缺少一个新“数据中心”，而是现有骨架的**关联和计算语义不够严格**：行动目标不一定与决策事实属于同一经营对象；观测中的目标值、实际值和偏差由请求方填写，并非从权威来源计算；观测也没有强制晚于行动执行。这样的记录可以追踪材料，但还不能可靠地支持系统学习。

## 证据等级

| 等级 | 本报告中的含义 |
|---|---|
| `implemented` | 当前工作树有模型、服务、API、迁移或页面 |
| `automated_verified` | 本次聚焦测试通过 |
| `inferred` | 根据代码关系作出的判断，尚未由真实经营运行验证 |
| `unknown` | 当前仓库与本次运行无法证明 |
| `not established` | 所需因果、时序或对象约束没有成立，不能宣称闭环 |

## 当前真实链路

```text
platform_order_ingest / purchase_authority
  → business_decision_fact_snapshot
  → business_ai_recommendation
  → business_owner_decision
  → business_controlled_action
  → DispatchSafe(Command)
  → business_action_observation
  → business_next_action_recommendation
```

这条链是两个相邻系统的组合，不是一个万能 `experiment`：

- `businessdecision`：事实、建议、Owner 决定。
- `businessfeedback`：受控行动、执行、观测、下一建议。
- `experiment`：仍是 `trace_only` 经营事实核验案卷，不提供经营授权或因果结论。
- 小Q：只接入 `businessdecision`；`businessfeedback` 仍为 `deferred`。

## 已有能力

### 1. 事实快照

`businessdecision.Service.CreateCase` 会按 Owner 冻结一个来源对象，将来源内容序列化并保存 SHA-256 manifest；当前服务只接受 `platform_order_ingest` 和 `purchase_authority`。[service.go](../../backend-go/internal/domain/businessdecision/service.go#L61-L118)

`platform_order_ingest` 的 Owner 页面候选只列出 `external_observed + applied` 记录；采购候选列出 Owner 自己的采购请求。[service.go](../../backend-go/internal/domain/businessdecision/service.go#L218-L249)

判断：`implemented`。但当前事实入口只有两类对象，不是统一经营事实入口。

### 2. AI 建议与 Owner 决定分离

AI 建议绑定案卷 manifest，且服务拒绝建议声明为 `actual / external_observed / reconciled`。[service.go](../../backend-go/internal/domain/businessdecision/service.go#L121-L144)

Owner 的 `selected` 决定必须冻结 capability、command、target、输入 payload 与输入 SHA-256；非 selected 决定不得携带行动权限。Owner 决定还会核对案卷 manifest 和可选的建议 ID。[service.go](../../backend-go/internal/domain/businessdecision/service.go#L146-L197)

PostgreSQL 迁移把案卷、快照、AI 建议与 Owner 决定设为不可更新、不可删除。[000139](../../backend-go/migrations/000139_business_decision_case.up.sql#L28-L32)

判断：`implemented / automated_verified`。

### 3. 小Q只建议，不替 Owner 决定

小Q读取 Owner 隔离的经营决策案卷，将冻结事实加入 trace evidence；真实 Provider 的回答固定为 `inferred`，只有显式请求时才调用 `businessdecision.Recommend` 保存建议。stub Provider 只返回 `mock`，不会保存为可信建议。[xiaoq/service.go](../../backend-go/internal/domain/xiaoq/service.go#L265-L324)

小Q没有 `Decide` 和任意 Command 执行能力；能力合同也明确 `businessfeedback` 当前为 `deferred`。[XIAOQ_CAPABILITY_CONTRACT.md](../governance/XIAOQ_CAPABILITY_CONTRACT.md#L138)

判断：`implemented / automated_verified`；真实建议质量与 Owner 使用价值为 `unknown`。

### 4. 受控行动与执行安全

创建行动时会核对：

- 当前 Owner；
- 同一案卷最新决定必须为 `selected`；
- capability、command、target 和输入哈希必须与 Owner 决定完全一致；
- capability 必须来自当前 Dispatcher 已登记命令；
- Owner 级幂等键不能绑定不同输入。

见 [businessfeedback/service.go](../../backend-go/internal/domain/businessfeedback/service.go#L42-L100)。

执行前再次核对最新 Owner 决定，并通过 `DispatchSafe` 的 production/high-risk 审批、幂等与策略门。执行前审计失败会停止派发；进入派发后没有可靠成功回执则进入 `reconcile_required`，不会盲目重试。[businessfeedback/service.go](../../backend-go/internal/domain/businessfeedback/service.go#L125-L213)

判断：安全骨架 `implemented / automated_verified`。

### 5. 结果观测与下一建议分离

只有状态为 `succeeded` 的行动可以追加观测。观测来源限于同一订单范围内的：

- `platform_order_ingest`；
- `order_final_profit_version`；
- 单订单可归属的 `cash_reconciliation`。

来源会保存事实等级、来源 manifest 和观察时间；跨订单利润与多订单现金被拒绝。[businessfeedback/service.go](../../backend-go/internal/domain/businessfeedback/service.go#L215-L290)

下一动作建议必须先有至少一条观测，并固定为 `inferred / proposed`。[businessfeedback/service.go](../../backend-go/internal/domain/businessfeedback/service.go#L292-L303)

观测与下一建议在 PostgreSQL 中都是追加式不可变记录。[000140](../../backend-go/migrations/000140_business_controlled_action_feedback.up.sql#L22-L76)

判断：记录协议 `implemented / automated_verified`；学习结论有效性 `not established`。

## 关键断点

### P0：决策事实、行动目标与结果对象没有形成同一对象约束

订单事实案卷可以冻结订单 A，但 Owner 决定中的 `target_type / target_id` 可以填写任意 SKU；`CreateAction` 只核对它是否与决定一致，不核对目标 SKU 是否属于该订单。现有测试本身就是“订单 101 → SKU 1”，没有建立订单行与 SKU 1 的确定性关系。[businessfeedback/service_test.go](../../backend-go/internal/domain/businessfeedback/service_test.go#L69-L123)

结果观测只校验回到了案卷订单，却没有证明所执行的目标就是该订单对应商品。因此当前最多证明“某行动记录与某订单结果被放入同一案卷”，不能证明它们属于同一个经营对象，更不能形成可靠学习样本。

### P0：`actual_value` 和偏差不是从权威事实计算

`authoritativeSource` 只验证来源对象、Owner、订单范围、哈希和观察时间。随后保存的 `target_metric / target_value / actual_value / comparison_note` 全部来自请求输入，没有从利润版本或现金对账中读取对应数值。[businessfeedback/model.go](../../backend-go/internal/domain/businessfeedback/model.go#L72-L80) [businessfeedback/service.go](../../backend-go/internal/domain/businessfeedback/service.go#L215-L235)

这意味着可以引用一条真实利润记录，同时手填一个与该记录不一致的“实际利润”。数据库会保护来源哈希，却不会保护解释值的正确性。

### P0：没有行动前后时序与观察窗口约束

观测保存了来源 `observed_at`，但没有要求：

- `observed_at > action.executed_at`；
- 到达预先冻结的观察开始/结束时间；
- 结果发生在行动暴露期间；
- 没有使用行动前已经存在的历史结果。

因此旧利润或旧订单事实也可能被追加为“行动结果”。当前不能据此做前后比较。

### P0：PostgreSQL 契约与采购案卷服务发生漂移

当前 Go 服务允许 `purchase_authority` 创建经营决策案卷，[service.go](../../backend-go/internal/domain/businessdecision/service.go#L61-L90)；采购页面也会调用该入口。但迁移 `000139` 仍把 `business_decision_case.object_type` 限制为 `platform_order_ingest`，并把 `object_id` 外键固定指向 `platform_order_ingest(id)`。[000139](../../backend-go/migrations/000139_business_decision_case.up.sql#L1-L5)

后续迁移没有修改该约束。SQLite 聚焦测试通过不代表真实 PostgreSQL 能插入采购案卷。当前采购决策路径应判为：Go 层 `implemented / automated_verified`，PostgreSQL 运行可用性 `not established`。

### P1：命令“成功”的业务含义不统一

生产 Dispatcher 注册了 6 个命令并自动把它们全部暴露为 `command.<type>.v1`。[router.go](../../backend-go/internal/httpx/router.go#L177-L189) [router.go](../../backend-go/internal/httpx/router.go#L779-L785)

但其中：

- `replenish` 只返回一个 `pending_approval` 结果，没有创建权威采购请求；
- `listing_optimize` 只构造返回值，没有保存真实 Listing 草稿；
- `stock_alert` 创建通知失败时仍返回成功；
- `price_update` 在缺少有效 SKU 或价格时也可能不写数据而返回成功；
- `compliance_check` 找不到 SKU 或更新失败时仍返回成功。

见 [command/handlers.go](../../backend-go/internal/platform/command/handlers.go#L21-L209)。因此 `business_controlled_action.status=succeeded` 目前不统一表示“目标经营动作真实发生并有持久凭证”。

### P1：下一建议是人工文本记录，不是系统学习

`CreateRecommendation` 只检查存在任意观测，然后原样保存请求方提交的建议和理由；没有读取观测内容、计算偏差、检索相似历史或生成新案卷。[businessfeedback/service.go](../../backend-go/internal/domain/businessfeedback/service.go#L292-L303)

Owner 页面也直接提供“下一动作 / 理由”文本框。[business-decisions page](../../frontend-next/src/app/(main)/business-decisions/[id]/page.tsx#L33)

因此当前名称虽然叫 `next_action_recommendation`，实际是“人工追加一条 inferred 备忘”，不是自动学习机制。

### P1：下一建议没有进入下一轮

下一建议没有：

- manifest 或幂等键；
- 指向后续 `business_decision_case` 的关系；
- 被 Owner 接受、拒绝或 supersede 的服务流程；
- 再次执行后形成 `previous_action_id / next_case_id` 的链。

所以当前链停在“提出下一建议”，尚未完成 ADR-001 所要求的“下一动作再次进入市场”。

### P1：事实入口和结果入口仍过窄

决策案卷目前只接受订单摄取或采购请求；反馈观测只接受订单摄取、最终利润和现金对账。市场、商品机会、货源、Listing、库存、履约、售后等权威对象尚不能直接成为统一学习案卷的冻结输入或结果。

这不是要求立即建设万能对象模型；它说明当前代码只能验证一条很窄的订单/利润原型，不能宣称已经形成平台级学习底座。

### P2：治理注册表保留了重复边界

`platformtruth` 已把 `businessdecision`、`businessfeedback` 标为 `implemented/reuse`，同时还保留旧 `decision` 域为 `planned/rebuild`。[registry.go](../../backend-go/internal/domain/platformtruth/registry.go#L19-L31)

该状态可能是迁移期刻意保留，但当前没有明确说明最终由谁承载权威经营学习链。正式设计前需要收口职责，避免又建立第三套“底层数据/学习中心”。

## 可复用部分

不需要重建第二套数据库或通用数据中心。以下能力应直接复用：

1. `business_decision_fact_snapshot`：冻结输入、事实等级、观察时间和 manifest。
2. `business_ai_recommendation`：建议固定为非事实，并绑定 manifest。
3. `business_owner_decision`：Owner 不可变决定和 exact-action 授权。
4. `business_controlled_action`：最新决定复核、独立审批、幂等、审计与 `reconcile_required`。
5. 订单、利润和现金领域的权威事实与同订单约束。
6. 小Q的 Capability、trace、evidence 和 stub 降级边界。
7. `platformtruth` 的事实等级、Owner 范围和领域职责合同。

应改的是关联与计算合同，不是存储技术。

## 最小纵向验证建议

推荐先验证一条**单订单、单 SKU、最终利润**的受控价格行动，不扩展多租户，也不同时接所有领域。

### 必须先补齐的合同

1. 案卷冻结 `order_id + order_line_id + exact sku_id`，行动 target 必须由该快照确定，不能自由填写。
2. 冻结动作变量、原值、新值、目标指标、目标值、观察开始时间和结束时间。
3. 命令成功必须返回可持久核验的执行凭证；没有凭证只能进入 `reconcile_required`。
4. 观测必须发生在执行之后且位于冻结观察窗口内。
5. `actual_value` 必须由指定权威来源字段确定性计算，API 不接受调用方手填。
6. 偏差由服务端计算，并只记录 `support / counter / conflict / unknown`，不声明因果。
7. 下一建议绑定观测 manifest；Owner 接受后创建一个新的决策案卷，并显式保存 `previous_action_id`。

### 验收步骤

```text
同一 Owner 的真实订单与 SKU 事实
→ 冻结价格行动及观察窗口
→ 小Q生成 inferred 建议
→ Owner selected 决定
→ exact 审批与真实执行凭证
→ 窗口结束后读取同订单最终利润
→ 服务端计算目标/实际/偏差
→ inferred 下一建议
→ Owner接受后生成下一案卷
```

### 通过条件

- 刷新或重启后能恢复整条链。
- 任一 Owner、订单、SKU、输入哈希、审批或来源对象不一致都失败关闭。
- 观测早于执行、超出窗口、来自另一订单或手填数值都被拒绝。
- 命令没有可靠执行凭证时不得标记 `succeeded`。
- 下一建议不能自动执行，必须再次经过 Owner 决定。

### 停止条件

- 当前价格命令无法提供真实渠道执行凭证；或
- 当前真实经营没有可观察的单订单/单 SKU 结果；或
- 必须靠人工复制结果值才能完成偏差计算。

触发任一条件时，停止扩建“学习中心”，先补齐对应权威事实或执行入口。

## 本次验证

本次运行：

```text
go test ./internal/domain/businessdecision ./internal/domain/businessfeedback ./internal/domain/xiaoq ./internal/domain/platformtruth
```

结果：4 个 package，共 54 项测试通过，证据等级为 `automated_verified`。

这些测试证明当前工程规则在测试环境中成立，不证明 PostgreSQL 采购路径、真实外部执行、真实结果观察、建议质量、因果关系或经营价值。

## 不在本次范围

- 没有设计或推导多租户、跨用户共享学习、匿名聚合或 SaaS。
- 没有修改代码、迁移、运行数据库或外部平台。
- 没有把旧 `agentlearning`、AIOS 内存或 `experiment` 重新解释为权威经营学习系统。
