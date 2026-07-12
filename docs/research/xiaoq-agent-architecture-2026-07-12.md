# 小Q Agent 架构官方资料调研

> 日期：2026-07-12
> 范围：只研究“单一主 Agent + 能力目录 + 权限审批 + 持续扩展”的技术模式。
> 来源限制：仅引用 OpenAI、Anthropic 与 Model Context Protocol 官方资料。
> 项目边界：小Q只服务凌镜 Owner 自营跨境商品实验，不是外部 SaaS，也不建议增加 Agent、MoA 或自治平台。

## 结论

`inferred`：小Q适合采用一个主 Agent、一个按任务动态暴露的能力目录、一个不受模型控制的策略与审批层，以及一条完整执行审计链。模型只能提出工具调用；真正执行、拒绝、暂停、恢复和验证必须由凌镜的确定性代码负责。

这不是“让小Q直接调用整个数据库”。“能调用整个系统”应解释为：每个业务模块通过版本化能力契约接入小Q；小Q只看见当前身份、实验状态和风险等级允许的能力。

## 官方资料支持的成熟模式

### 1. 单一主 Agent 可以组合多个工具，不必先建设多 Agent

`quoted`：OpenAI Agents SDK 把 Agent 定义为“模型 + 指令 + 工具 + 可选运行行为”，Runner 负责回合、工具、护栏与会话编排。[OpenAI Agents SDK：Agents](https://openai.github.io/openai-agents-python/agents/)

`inferred`：小Q第一阶段只需要一个 Owner-facing Agent。市场、实验、1688 草稿、订单和利润都作为能力域接入；不要把每个模块都变成一个 Agent。这样身份、审批语义和输出口径只有一套。

### 2. 能力目录应是明确契约，并按任务逐步发现

`quoted`：MCP 把外部能力分为 resources、tools 和 prompts；工具由 LLM 请求调用，但执行仍由宿主应用控制。[MCP：Build an MCP server](https://modelcontextprotocol.io/docs/develop/build-server)

`quoted`：MCP 官方指出，把大量工具定义一次性装入上下文会浪费 Token、增加延迟并降低模型表现，建议达到阈值后使用 progressive discovery，只在需要时加载完整工具定义。[MCP：Client Best Practices](https://modelcontextprotocol.io/docs/develop/clients/client-best-practices)

`quoted`：OpenAI Agents SDK 支持静态和动态工具过滤，动态过滤可以读取当前 run context、Agent 和 server 信息。[OpenAI Agents SDK：MCP](https://openai.github.io/openai-agents-python/mcp/)

`inferred`：凌镜应维护自己的 `Capability Registry`，每项能力至少声明：

- 稳定名称、版本、所属业务域；
- 输入/输出 schema；
- `read / propose / mutate / external_write / financial` 风险级别；
- 所需 Owner 身份、业务状态和 evidence 要求；
- 是否必须审批、是否允许重试、幂等键；
- 超时、成本上限、审计字段和结果验证器。

小Q每轮先查轻量能力索引，再只加载与当前任务相关的少量定义。新增业务功能时，只有完成能力契约、策略、测试和说明，才算完成“小Q接入”。

### 3. 权限与审批必须在模型之外强制执行

`quoted`：OpenAI Agents SDK 支持按工具或按调用设置审批；需要审批时运行会暂停，应用检查调用参数后批准或拒绝，再从保存的 RunState 恢复。[OpenAI Agents SDK：Human-in-the-loop](https://openai.github.io/openai-agents-python/human_in_the_loop/)

`quoted`：OpenAI 的 MCP 集成支持工具 allow/block list、动态过滤，以及 `always/never/按工具` 的审批策略。[OpenAI Agents SDK：MCP](https://openai.github.io/openai-agents-python/mcp/)

`quoted`：Anthropic 明确说明，模型只返回 tool request，真正运行工具的是应用；对于 shell，应以最小权限在容器或虚拟机中隔离、采用 allowlist、限制资源、记录命令并遮蔽凭证。[Anthropic：Bash tool](https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/bash-tool)

`quoted`：Codex 的官方安全说明也把 sandbox 与 approval 分开：sandbox 定义实际可写路径和网络边界，approval 决定何时必须停下来请求许可。[OpenAI：Running Codex safely](https://openai.com/index/running-codex-safely/)

`inferred`：小Q应采用以下硬规则：

- 只读查询可在 RBAC、业务状态和数据范围校验后自动执行；
- 建议类能力只生成结构化草案，不改变业务事实；
- 数据变更必须走现有 Service/Command，不允许直连数据库绕过状态机；
- 发布、采购、广告、退款、资金与最终事实确认始终暂停，展示对象、金额、渠道、证据和影响，等待 Owner 单次批准；
- 批准绑定具体 `tool_call_id + 参数摘要 + 数据版本`，参数或事实变化后必须重新批准；
- 对外写入必须有幂等键、回执保存、失败状态和人工可恢复路径。

### 4. 审计、成本和运行边界属于核心能力，不是后补功能

`quoted`：OpenAI Agents SDK 的 trace 可以记录模型生成、工具调用、审批/护栏及自定义事件，并允许控制是否包含敏感数据。[OpenAI Agents SDK：Tracing](https://openai.github.io/openai-agents-python/tracing/)

`quoted`：SDK 会聚合每次 run 的请求数、输入/输出 Token、缓存 Token 和推理 Token，可用于监控成本与执行限制。[OpenAI Agents SDK：Usage](https://openai.github.io/openai-agents-python/usage/)

`quoted`：Runner 可限制最大回合和本地 function tool 并发数。[OpenAI Agents SDK：Running agents](https://openai.github.io/openai-agents-python/running_agents/)

`inferred`：每次小Q运行至少应保存 `run_id、Owner、目标、加载的能力、模型/提示版本、Token/费用、每次调用参数摘要、审批决定、业务对象、证据引用、结果、错误和最终裁决`。敏感输入输出默认不进入第三方 trace；凌镜本地审计记录是经营裁决的权威来源。

## MCP 在小Q中的位置

`actual`：MCP 是连接能力的协议，不是授权系统、业务状态机或事实核验系统。[MCP：Architecture overview](https://modelcontextprotocol.io/docs/learn/architecture)

建议：

- 凌镜内部 Go 模块优先通过本地 capability adapter 调现有 Service，减少协议与网络复杂度；
- 只有确实需要跨进程、跨语言或第三方连接时才使用 MCP；
- MCP 工具仍必须经过凌镜统一 Registry、策略、审批和审计包装；
- 远程 MCP 只连接可信来源，并逐工具 allowlist。Anthropic 明确说明其目录收录不等于安全审计。[Anthropic：Claude Code Security](https://docs.anthropic.com/en/docs/claude-code/security)

## 主要风险

1. **“工具描述等于权限”**：`false`。工具注解和提示词只帮助模型判断，不能替代后端鉴权与状态校验。
2. **Prompt injection / 恶意外部内容**：平台网页、供应商描述或 MCP 返回值可能诱导小Q调用高风险工具。外部内容一律作为不可信数据，不能改变策略。
3. **Confused deputy（被借权代办）**：MCP 官方列出代理服务器可能被利用替攻击者取得授权的风险；应绑定 audience、Owner、资源范围并使用最小权限令牌。[MCP：Security Best Practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices)
4. **工具过多导致选择错误和 Token 膨胀**：使用分域索引、动态过滤和按需加载；不要把“整个系统”一次性塞进模型上下文。
5. **审批疲劳**：审批颗粒度过细会诱发无脑批准。低风险读自动化，高风险动作以清晰影响摘要一次一批，但资金/发布等不做跨会话永久批准。
6. **护栏覆盖误判**：OpenAI 官方说明 tool guardrail 只覆盖 function tools，不自动覆盖 hosted tools、MCP hosted tool、shell 等类别。[OpenAI Agents SDK：Guardrails](https://openai.github.io/openai-agents-python/guardrails/) 因此凌镜必须在统一执行网关再次校验，不能只依赖 SDK 护栏。
7. **把 Agent 回答当成经营事实**：小Q输出仍是 `inferred`；只有现有证据、Owner 核验、外部观察或对账流程才能提高事实等级。

## 对凌镜的最小落地建议

`inferred`：先完成一个可验证闭环，而不是直接“接管整个系统”：

1. 建立 `xiaoq` 运行记录与 Capability Contract；
2. 先接一个只读能力，例如读取一个候选市场案件并解释 evidence 缺口；
3. 验证能力筛选、Token/回合上限、RBAC、审计和失败呈现；
4. 再接一个 `propose` 能力，只生成建议草案；
5. 最后接一个可恢复的低风险内部 mutation，并完整走 Owner 审批；
6. 每个后续模块的完成定义增加“小Q接入检查”，但 `not_applicable` 可以是合法结论，避免为了覆盖率强行把所有功能暴露给模型。

通过标准：Owner 能从一次 run 中复核“小Q为什么选择该能力、读了什么证据、花了多少、是否改变状态、谁批准、结果是否成功”。在此之前，“小Q能调用整个系统”应保持 `planned`，不能写成 `implemented` 或 `verified`。
