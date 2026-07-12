# 凌镜 AI 基础建设总规划：小Q 与 Evidence Workshop

> 日期：2026-07-12
> 状态：待 Owner 审阅；获批后按阶段执行
> 适用边界：Owner 单人自用跨境商品实验系统
> 历史计划：原 AI-Native AgentOS 扩张计划已冻结，归档见 `tasks/ai-native-agentos-execution-plan.md`

## 1. 结论

凌镜可以继续建设 AI 基础设施。系统只建设一个面向 Owner 的主 Agent，名字叫 **小Q**；不建设一组互相竞争的常驻 Agent，也不以 Agent 数量、自治程度或界面规模作为目标。

小Q 是 Owner 与凌镜整个系统之间的统一对话和执行入口：它能理解 Owner 的经营问题，读取被授权的系统数据，调用各业务模块提供的工具，解释结果，提出下一步，并把需要现实写入的动作送入审批，而不是绕过系统直接操作数据库或平台。

本规划把所谓 “agentic workshop” 定义为小Q底层的 **Evidence Workshop（证据工作间）**：

```text
Owner 对小Q提出经营问题
→ 受控分配研究任务
→ 调用已批准的模型与工具
→ 保存原始来源和不可变快照
→ 独立产生支持、反证和数据现实结果
→ 用确定性规则裁决
→ Owner 批准高风险动作
→ 关联真实订单、售后、结算和现金
→ 用最终利润决定停止、换品、修正或小幅加码
```

它不是一个面向外部用户的软件产品，而是小Q调用现有凌镜经营事实链时使用的受控 AI 执行层。

## 1.1 小Q的一句话定义

> 小Q是凌镜唯一的 Owner 经营 Agent：会看、会查、会解释、会建议、会提交受控动作，但不能伪造事实，也不能绕过权限、审批、审计和经营状态机。

## 1.2 “能调用整个系统”的准确含义

“整个系统”不是给小Q数据库超级权限，而是每个业务模块通过稳定、可测试的 **Capability（能力接口）** 向小Q开放允许的操作。

每项能力必须声明：

- 能解决什么 Owner 问题；
- 输入、输出和事实来源；
- read / suggest / mutate 风险类别；
- 需要的 RBAC 权限和 Owner 审批；
- 是否有外部副作用；
- 幂等、超时、失败和恢复方式；
- 审计、trace 和 evidence 如何关联；
- 小Q如何用普通语言解释执行结果。

因此，小Q最终可以覆盖整个系统，但权限是逐项登记和逐项验收的，不是一开始无限开放。

## 2. 大目标

### 长期愿景（3—5 年，`vision`）

让 Owner 可以对小Q说出一句经营问题，由小Q启动一条受控工作流，自动完成系统查询、资料发现、来源保存、独立反证、数据刷新和异常重试；所有现实写操作仍经过明确审批，最终由可对账的经营结果而不是 AI 信心决定下一步。

长期成功不是“无人值守”，而是：

- Owner 不再日常抄数和拼接资料；
- AI 的每个重要结论都能追溯到来源、时间和原始快照；
- AI 无权把推断升级成事实；
- 采购、发布、广告、退款和资金动作保持 Owner 控制；
- 系统能从一次次真实实验中积累可复用的经营证据，但不跨案件伪造确定性；
- 至少连续三轮小规模经营实验都能完整走到售后、结算、现金与最终利润裁决。

### 12 个月目标（`planned`）

完成一条可重复的 Owner 自用 AI 商品实验循环：

1. 候选市场研究有三个独立 run：侦察、反证、数据现实；
2. 每个 run 的输入、模型、工具、来源、Token、费用、失败和输出可追踪；
3. 关键结论只有在来源与观察时间合格时才能进入需求案件；
4. 已选市场能够进入商品机会、1688 货源草稿和 Owner 审批；
5. 真实实验关联流量、订单、付款、签收、售后、结算、现金和最终利润；
6. 完成至少三轮真实实验裁决，不要求三轮都盈利；
7. 只有在重复使用证明确实减少 Owner 的研究或核验工作后，才讨论扩大自动化。
8. 当前经营主线涉及的每个新功能都同步提供小Q Capability、权限规则、审计和验收测试；没有小Q接入不阻止功能发布，但该功能不得宣称“小Q可调用”。

### 90 天主目标（`planned`）

交付一个最小但完整的小Q：Owner 可以在统一对话入口询问一个候选市场问题，小Q能读取需求案件、受控地产生三份独立研究结果、保存来源快照，通过需求案件闸门形成“继续取证 / 淘汰 / 可实验”建议，并让 Owner 看懂为什么。

### 第一阶段通过标准

- 一个真实候选问题完成 `scout_result → falsifier_result → data_reality_result`；
- 三个 run 输入和输出不可混用，重复提交幂等；
- 每条关键证据都有来源 URL/API、观察时间、原始 payload、SHA-256 和真实性状态；
- 至少一条反证实际改变或阻止结论，而不是装饰性反证；
- unknown、mock 或 inferred 关键字段不能进入 `experiment_ready`；
- Owner 能在一个页面看到结论、依据、反证、未知、成本和下一步；
- Owner 能在小Q对话中追问依据，并跳转到对应需求案件、run、evidence 和审批；
- 同一任务相较纯人工基线，人工整理与复核时间下降；具体阈值在首次基线测量后冻结。

### 失败与停止条件

- 连续两次受限试运行都出现来源错配、事实升级或跨 run 污染；
- 工具要求持有不必要的生产写权限；
- 人工复核时间没有下降，反而因复杂编排增加；
- 90 天内仍无法让一个真实候选问题完成端到端裁决；
- 建设开始转向更多 Agent、通用工作台、外部 SaaS、多租户或展示性仪表盘；
- 首轮经营资金纪律仍按现行政策：总现金上限 3,000 CNY、不可回收损失硬停止线 1,200 CNY；本规划不自动授权任何资金动作。

触发停止条件时，停止扩建，保留证据，回到单 Agent + 人工审批流程。

## 3. 当前事实基线

### `implemented / automated_verified`

- 已有 Go/Next 主栈、Auth、RBAC、Approval、Audit、EventBus、Command、Scheduler 和 ToolBridge；
- `demandcase` 已定义三个研究输入类型和不可变 payload 校验；
- `experiment` 已有支持/反证/冲突、真实性状态和部分终局门禁；
- `sourcing1688` 已有受控草稿流程，不能直接发布；
- ToolBridge 已区分工具类别，mutation 工具缺少 approval ID 时会阻止执行；
- AI trace、成本控制、guardrails 和 action policy 已有代码基础。

### `implemented but not trusted as real AI`

- `internal/ai/orchestrator.go` 仍会生成 stub tool call、stub evidence 和 deterministic fallback；
- 多 Agent 聚合主要是结构化模板，不能视为真实独立推理；
- 旧 AgentOS 页面和部分导航存在 mock 或历史方向；
- 连接器存在不代表账号、权限、市场或生产契约有效。

### `unknown`

- 当前模型供应商在目标语言和目标市场问题上的真实准确率；
- 浏览器、API 和第三方 MCP 对目标数据源的可用性；
- 每个合格研究 run 的真实 Token、费用和人工复核成本；
- AI 是否能稳定降低 Owner 工作量；
- 是否存在真实成交、售后闭合和正最终净利润。

## 4. 架构决策

### 4.0 小Q采用“一个身份、一个能力目录、多个确定性服务”

小Q只有一个稳定身份 `xiao_q` 和一个 Owner 对话入口。它不复制业务逻辑，内部通过 Capability Catalog（能力目录）发现并调用现有领域服务。

```text
Owner
  ↓
小Q：理解意图、组织上下文、解释结果
  ↓
Capability Catalog：能力发现、schema、风险、权限
  ↓
ToolBridge / Command / Domain Service
  ↓
DemandCase / Experiment / Sourcing1688 / Order / Aftersales / Settlement
  ↓
Evidence + Trace + Audit + Approval
```

小Q不能直接拼接 SQL、直接修改领域表，或用自由文本代替 Command 和领域服务。

### 4.1 不新建第二套 Agent 内核

复用现有 Go Kernel、ToolBridge、Approval、Audit、Trace、Cost Control 和领域状态机。暂不引入 LangGraph、CrewAI、AutoGen、Google ADK 或新的 Python sidecar。

只有当一个明确、可测试的能力缺口连续阻塞两个真实 run，且现有 Go 实现成本明显更高时，才允许做一个隔离原型；原型不直接进入生产事实链。

### 4.2 模型负责建议，领域规则负责裁决

LLM 可以发现、提取、归纳、提出假设和反证；不能决定真实性等级、跨过关键 unknown、修改预算和停止条件，也不能最终确认利润。

### 4.3 一个运行时，三个独立角色

不建设更多对 Owner 暴露的常驻 Agent。第一阶段由小Q在后台按需启动三个隔离研究角色：

- Scout：寻找支持和候选线索；
- Falsifier：主动寻找足以淘汰候选的证据；
- Data Reality：核验账号、字段、地区、时间、权限和数据口径是否真实可得。

三者共用同一受控运行时，但使用独立输入快照、独立上下文和独立输出；Falsifier 在提交前不得读取 Scout 的结论文本。

这些是小Q的内部研究模式，不是三个独立产品、三个自治员工或三个新增导航入口。

### 4.7 新功能必须遵守“小Q同步协议”

以后新增或修改业务功能时，必须在设计阶段回答：

1. 小Q是否应该读取它？
2. 小Q是否应该提出建议？
3. 小Q是否可以提交动作？动作风险是什么？
4. 需要哪些 Capability schema、权限、审批、审计和证据？
5. 小Q如何向 Owner解释成功、失败和未知？
6. 哪些自动测试证明小Q没有越权？

若答案是“暂不接入”，必须显式记录 `xiao_q_support: deferred` 和原因，不能让小Q猜测或直接调用内部实现。

### 4.4 工具按风险分层

```text
L0 本地只读：读取规则、schema、历史快照
L1 外部只读：搜索、抓取公开页面、调用只读 API
L2 建议写入：保存内部草稿、研究结果和待复核证据
L3 外部经营写入：发布、采购、广告、订单、退款、资金
```

第一阶段只启用 L0—L2。L3 不属于 Evidence Workshop 自动执行范围，必须走独立 Owner 审批、审计和幂等控制。

### 4.5 MCP 是连接方式，不是信任来源

MCP 或插件只负责连接工具。每个工具仍需登记：用途、输入输出、凭据范围、数据去向、读写副作用、超时、重试、费用、审计和停用开关。第三方 MCP 默认不可信，未审计前不得接触生产凭据。

### 4.6 Token 优化服从证据完整性

先减少无关上下文、重复抓取和无效并行，再考虑摘要和缓存。原始证据、关键反证、来源与观察时间不得为了省 Token 被摘要替代。

## 5. 阶段路线

### Phase 0：冻结小Q边界与基线（1 周）

目标：明确哪些是当前事实链、哪些是 stub/历史能力，并测量一次纯人工研究基线。

交付：小Q身份和职责契约、Capability Contract、运行契约、工具清单、模型清单、成本口径、人工基线、禁用项清单。此阶段不接生产写权限。

通过门：Owner 能用普通语言说明 Workshop 会做什么、不会做什么；一个基线问题有人工耗时和结果记录。

### Phase 1：小Q只读对话与单 run 可信执行（2—3 周）

目标：先让 Owner 能和小Q对话，让小Q安全读取一个需求案件；再让一个 Scout run 真正调用模型和只读工具，完整保存 trace、来源和失败，而不是扩展多 Agent。

交付：小Q对话会话、意图路由、只读 Capability 调用、统一 Run Contract、真实 provider 调用、工具调用记录、预算硬限制、超时/取消、结构化输出校验、stub 显式标记。

通过门：小Q能正确回答一个需求案件的当前状态并给出事实链接；一个真实 Scout run 可重复执行；失败不会伪装成成功；任何 stub 输出均不能进入需求案件。

### Phase 2：独立三 run 与证据导入（2—3 周）

目标：完成 Scout、Falsifier、Data Reality 三个隔离 run，并接入 `demandcase`。

交付：上下文隔离、run provenance、幂等提交、来源快照、冲突标记、关键字段 completeness gate。

通过门：至少一个候选被反证阻止或退回补证；重复 run 不产生重复事实；关键 unknown 阻止可实验结论。

### Phase 3：小Q Owner 决策工作台（2 周）

目标：让 Owner 不读技术 trace 也能判断候选。

交付：小Q六行决策卡——候选、支持、反证、未知、成本/风险、建议下一步；可在对话中追问并下钻来源和原始快照；支持继续取证、淘汰、批准市场三种受控动作。

通过门：页面没有 mock 经营数字；每个决定能回溯到 run 和 evidence ID；Owner 操作有审计记录。

### Phase 4：已选市场到商品实验（3—4 周）

目标：把已批准市场安全接到商品机会和 1688 草稿，不增加平台方向。

交付：market approval → experiment → sourcing1688 的稳定引用；商品/供应商/成本/合规证据；草稿审批保持 draft；外部发布仍被阻止。

通过门：一个案件能从已批准市场形成待上架草稿，且没有任何自动生产发布。

### Phase 5：真实经营结果闭环（4—8 周，取决于现实售后周期）

目标：用真实订单和结算裁决实验，而不是继续优化 Agent 表现。

交付：非关联买家核验、付款、签收、售后观察期、结算、现金、最终利润接线；停止规则；四选一结论。

通过门：至少一次实验走到最终裁决。盈利不是工程通过条件；凭证完整和裁决正确才是。

### Phase 6：小Q重复性与有限自动化（只有前三轮真实实验后）

目标：只自动化已经重复出现、人工规则稳定、失败可恢复的环节。

候选：定时刷新公开价格/费用、失败重试、冲突排队、来源失效提醒、重复证据去重。

禁止：自动改价、自动采购、自动投放、自动退款、自动资金操作，以及未经新 Owner 决策的更多 Agent/MoA。

## 6. 依赖关系

```text
边界与基线
  ↓
小Q Identity + Capability Contract
  ↓
统一 Run Contract
  ├─→ Provider / Token / Cost
  ├─→ Tool Registry / MCP policy
  ├─→ Trace / Audit / Failure states
  └─→ Evidence snapshot
            ↓
      三个独立 research run
            ↓
       DemandCase gates
            ↓
      Owner market approval
            ↓
 Experiment → Sourcing1688 draft
            ↓
 Order → Aftersales → Settlement → Cash
            ↓
       Final profit decision
            ↓
   Repeated-use automation review
```

## 7. 统一运行契约

每次 AI run 至少记录：

```text
run_id
run_type
experiment_id / demand_case_id
business_question
input_snapshot_hash
model_provider / model_name / prompt_version
allowed_tools / denied_tools
started_at / completed_at / status
token_input / token_output / estimated_cost
source_count / counter_source_count
raw_output_hash / parsed_output_version
failure_code / retry_of
correlation_id / audit_id
```

状态至少区分：`queued / running / awaiting_owner / completed / failed / blocked / canceled / superseded`。

`completed` 只说明运行完成，不说明结论为真；真实性由证据和领域闸门裁决。

## 7.1 小Q Capability Contract

每个可调用能力至少登记：

```text
capability_id / version
business_purpose
domain_owner
input_schema / output_schema
risk_level
required_permission
approval_required
execution_modes
external_side_effects
idempotency_required
evidence_policy
timeout / retry_policy
audit_action_type
owner_explanation_template
status: active / deprecated / deferred
```

新功能的完成定义增加四项：Capability 登记、权限测试、失败解释和小Q回归测试。仅纯内部实现且小Q无需调用时可以标记 deferred。

## 8. 预算与资源纪律

规划预算先采用上限而非承诺：

- Phase 0：只读盘点和基线，不新增付费平台；
- Phase 1—3：模型与外部工具试验预算建议上限 500 CNY，超过前由 Owner 决定；
- 单 run 必须有 Token/费用硬上限和超时；
- 同一失败最多自动重试两次，之后进入人工复核；
- 不因免费额度引入第二套长期主 Agent；
- 真实经营资金继续使用现有 3,000 / 1,200 CNY 纪律，与 AI 基础设施预算分开记录。

金额为 `planned ceiling`，不是已发生支出。

## 9. 衡量指标

### 主指标

- 完成一次合格候选裁决所需的 Owner 人工时间；
- 有效来源率：真正支持对应字段的来源占比；
- 反证命中率：实际导致淘汰、降级或补证的反证比例；
- 关键事实升级错误数；
- 从 run 到 evidence 再到决策的可追溯率；
- 真实实验完成最终裁决的比例。

### 护栏指标

- 未审批外部写入次数必须为 0；
- mock/stub 穿过经营闸门次数必须为 0；
- 跨 run 上下文污染次数必须为 0；
- 无来源关键结论进入 `experiment_ready` 次数必须为 0；
- 重复外部副作用次数必须为 0。

### 不作为进展的指标

Agent 数量、Prompt 数量、MCP 数量、代码行数、页面数、研究报告数、Token 消耗和模型“信心”。

## 10. 风险与应对

| 风险 | 影响 | 应对 |
|---|---|---|
| 把营销能力当真实能力 | 错误架构决策 | 只用受限任务实测，厂商声明标 `quoted` |
| stub 与真实模型混淆 | 伪造经营证据 | 输出强制 provenance；stub 永不进入业务闸门 |
| 多 Agent 相互污染 | 反证失效 | 独立输入快照、上下文隔离、提交后再汇总 |
| MCP/浏览器泄露凭据 | 账号与资金风险 | 最小权限、只读优先、第三方 MCP 审计、kill switch |
| Token 优化丢失证据 | 无法复核 | 原始快照独立保存，摘要不替代证据 |
| 自动化超过 Owner 理解 | 系统再次失控 | 每阶段一个 Owner 可见闭环，过门后才继续 |
| 建设替代真实经营 | 没有成交反馈 | Phase 4 后优先真实实验，不继续扩基础设施 |

## 11. 明确不做

- 不开发面向外部用户的 AI Workshop；
- 不把小Q改造成面向外部用户的通用助手；
- 不建设通用 Agent Builder、Agent 商店或工作流市场；
- 不新增 SaaS、多租户、订阅、计费和公共 API；
- 不用新框架重写现有 Go Kernel；
- 不扩充 Agent 名单、MoA 或自治等级；
- 不让 AI 自动执行价格、库存、订单、采购、广告、退款和资金动作；
- 不用多个 Agent 的一致意见提高事实等级；
- 不因长期愿景跳过第一轮真实候选和真实经营实验。

## 12. Owner 决策点

本规划获批只代表允许从 Phase 0 开始，不代表一次批准所有后续阶段。每个 Phase 都需要看见上一阶段的证据后再继续。

当前推荐决定：**确认“小Q是凌镜唯一 Owner Agent”的产品方向，并批准 Phase 0；暂不批准 Phase 1—6 的实现。**
