# 市面 AI Agent 调研（2026-07-12）

> 调研日期：2026-07-12
> 适用对象：凌镜 Owner 单人自用经营系统
> 来源范围：只采用厂商官方产品页、官方文档、官方博客与官方 GitHub。
> 边界：这是产品能力与部署形态盘点，不是实测报告。厂商宣称的效果、速度、准确率和生产力提升均未由本次调研独立验证。

## 1. 结论先行

1. **`actual`：市面上不存在统一标准的“agentic workshop”。** 更接近这一概念的是编码 Agent 已普遍具备的项目指令文件、Skills、MCP、Hooks、沙箱、审批和多任务界面，以及企业平台提供的 RBAC（按角色控制权限）、审计和数据连接器。
2. **`actual`：Agent 已分化为五类产品。** 通用个人执行型负责跨网页和应用完成任务；编码 Agent 在代码库、终端和 Git 中工作；研究 Agent 做多步检索和带来源报告；企业 Agent 把 Agent 接入公司数据与业务流程；开发框架供团队自己构建 Agent。
3. **`inferred`：对一人创业者，最佳起点是“一个主执行 Agent + 少数按需工具”，不是先搭建通用 Agent 平台。** 现成产品已覆盖指令文件、MCP、浏览器、终端、审批与隔离；自建基础设施只有在需要确定性状态机、不可变证据、业务审批和外部系统写入时才有必要。
4. **`policy`：凌镜当前冻结更多 Agent、MoA 和自治升级。** 因此本报告不能成为扩建 AgentOS 的依据。凌镜应复用现有 Codex/本地规则做开发与研究辅助，把真正需要固化的能力限制在候选市场 → 实验 → 订单/售后 → 最终利润事实链。
5. **`unknown`：没有产品可仅凭官方页面证明能可靠完成凌镜的真实跨境经营闭环。** 网页操作、研究报告、代码生成或测试通过，都不能证明市场、账号权限、真实成交或最终利润成立。

## 2. 证据标签

| 标签 | 本文含义 |
|---|---|
| `actual` | 官方资料直接确认的产品、功能、部署或价格事实 |
| `quoted` | 厂商自己的效果或定位说法，未由本次调研验证 |
| `inferred` | 根据官方能力与凌镜边界作出的适配判断 |
| `unknown` | 官方资料不足，或必须实际试用/接入后才能确认 |

## 3. 市场地图

| 类别 | 代表产品 | 主要形态 | 典型价值 | 对 Owner / 凌镜的初步适配 |
|---|---|---|---|---|
| 通用个人执行型 | ChatGPT agent、Manus、Perplexity Computer/Comet | 托管网页/桌面 Agent | 浏览网页、操作应用、制作文件、串联个人任务 | Owner 临时任务中等；核心经营事实链低 |
| 编码 Agent | OpenAI Codex、Claude Code、GitHub Copilot、Cursor、Gemini CLI | 本地 CLI/IDE/桌面 + 云端 Agent | 读写代码、运行命令/测试、提交 PR、并行任务 | 当前最适合直接复用 |
| 研究/浏览器 Agent | ChatGPT deep research、Gemini Deep Research、Perplexity Research | 托管研究 Agent/API | 多步搜索、阅读来源、形成带引用报告 | 候选市场线索较高；事实核验仍需独立证据 |
| 工作流/企业 Agent | Microsoft Copilot Studio、Google Vertex AI Agent Builder、Salesforce Agentforce、ServiceNow AI Agents | 企业 SaaS/PaaS | 接入企业数据、业务动作、身份权限与审计 | 当前过重，且偏离单人自用边界 |
| Agent 开发框架 | OpenAI Agents SDK、Google ADK、LangGraph、AutoGen、CrewAI | 开源库/自托管或配套云服务 | 自定义工具、状态、编排、多 Agent、追踪 | 只有明确缺口时局部采用，不先建平台 |

## 4. 通用个人执行型 Agent

### 4.1 ChatGPT agent

- `actual`：ChatGPT agent 可使用可视浏览器、代码解释器、Apps 和受支持的终端，能浏览网站、处理上传文件、连接第三方数据源、填写表单和编辑表格；执行中可暂停澄清或确认。[官方帮助中心](https://help.openai.com/en/articles/11752874-chatgpt-agent)
- `actual`：它是 OpenAI 托管的 ChatGPT 能力，面向 Plus、Pro、Business、Enterprise 和 Edu；官方帮助页在调研日列出 Plus 40 次/月、Pro 400 次/月、Business/Enterprise 40 次/月，灵活定价工作区为 30 credits/次。限制会变化，应在购买当天复核同页。
- `actual`：Apps 可连接外部工具和数据，部分 Apps 可执行动作；个人方案若开启“Improve the model for everyone”，Apps 访问的信息可能用于改进模型，Business/Enterprise/Edu 默认不用于训练。[Apps 官方说明](https://help.openai.com/en/articles/11487775-apps-in-chatgpt)
- `unknown`：官方页面不能证明其对任意网站、验证码、反爬、跨境平台权限和高风险写操作都稳定有效。
- `inferred`：适合 Owner 做临时网页任务和资料整理；不适合绕过凌镜的 Owner 审批、证据快照与经营状态机直接写生产平台。

### 4.2 Manus 与 Perplexity Computer/Comet

- `actual`：两者官方定位均已从问答扩展到可浏览、生成文件/应用或执行电脑任务的托管 Agent；Perplexity 官方套餐页将 Comet Assistant 明确列为 Browser Agent，并把 Research、Create files and apps 与 Computer 分开计量。[Perplexity 套餐说明](https://www.perplexity.ai/help-center/en/articles/11187416-which-perplexity-subscription-plan-is-right-for-you)
- `actual`：Perplexity Max 官方价为 200 美元/月或 2,000 美元/年，并提供更高 Browser Agent、Research 和文件/应用创建限额。[Perplexity Max](https://www.perplexity.ai/help-center/en/articles/11680686-perplexity-max)
- `unknown`：本次只找到有限且会动态变化的 Manus 官方公开材料，无法用稳定官方文档确认其细粒度权限、MCP、审计和当前准确价格，因此不据营销展示作能力结论。[Manus 官网](https://manus.im/)
- `inferred`：此类产品可作为临时执行面，但对于凌镜，价格、数据外传、网页提示注入和不可审计写入风险高于其当前必要价值，优先级低于已有 Codex + 受控连接器。

## 5. 编码 Agent

### 5.1 OpenAI Codex

- `actual`：Codex 提供桌面 App、CLI、IDE、Web/Cloud 等形态；桌面 App 支持多个 Agent/任务、独立线程和 Git worktree 隔离，并可审查 diff。[Codex App 官方介绍](https://openai.com/index/introducing-the-codex-app/)
- `actual`：默认限制 Agent 编辑当前目录或分支，使用系统级沙箱；网络或更高权限命令需审批，也可按项目/团队配置规则。Codex 支持 MCP，官方同时建议人工审查变更和部署。[Codex 升级与安全说明](https://openai.com/index/introducing-upgrades-to-codex/)
- `actual`：企业可通过 Permissions & Roles、RBAC 和 Compliance API 管控与记录 Codex 使用。[Enterprise 发布说明](https://help.openai.com/en/articles/10128477-chatgpt-enterprise-edu-release-notes)
- `actual`：调研日官方说明 Codex 已包含在 ChatGPT Free、Go、Plus、Pro、Business、Edu、Enterprise，额度随方案变化；额外 credits 的精确成本需在购买时查看账户定价。[使用 Codex 的官方说明](https://help.openai.com/en/articles/11369540-getting-started-with-codex)
- `inferred`：这是凌镜当前最合适的主开发 Agent，因为仓库规则、权限、Skills、Hooks、MCP 和多任务能力已经构成“workshop”的大部分，不需要另建同名平台。

### 5.2 Claude Code

- `actual`：Claude Code 是本地终端编码 Agent，AI 处理需要联网；支持交互与非交互模式、恢复会话、限制最大 turns、权限模式和 JSON 输出。[安装文档](https://docs.anthropic.com/en/docs/claude-code/getting-started)、[CLI 文档](https://docs.anthropic.com/en/docs/claude-code/cli-usage)
- `actual`：支持 MCP；官方 CLI 明确提供权限提示与 `--dangerously-skip-permissions`，后者标注需谨慎。官方高级指南还覆盖 CLAUDE.md、Subagents 及按子 Agent 限定工具权限。[Anthropic MCP 文档](https://docs.anthropic.com/en/docs/mcp)、[官方高级指南](https://resources.anthropic.com/hubfs/Claude%20Code%20Advanced%20Patterns_%20Subagents%2C%20MCP%2C%20and%20Scaling%20to%20Real%20Codebases.pdf)
- `unknown`：套餐包含量和 API 计费会随模型变化，本报告不写死价格；使用前应查 [Anthropic 官方定价](https://www.anthropic.com/pricing)。
- `inferred`：能力与 Codex 高度重叠。对一人团队同时长期维护两套主 Agent 会增加规则漂移和复核成本，除非做针对性第二意见或模型对比，否则没有必要双主栈。

### 5.3 GitHub Copilot

- `actual`：覆盖 GitHub、IDE、CLI 和 Cloud agent；支持 agent mode、MCP、自定义 instructions/agents、代码审查，并可把任务委派给 Agent 生成 PR。[官方产品页](https://github.com/features/copilot)、[Agent 页](https://github.com/features/copilot/agents)
- `actual`：调研日个人版官方价为 Free 0 美元、Pro 10 美元/月、Pro+ 39 美元/月、Max 100 美元/月；Agent、CLI、代码审查等消耗 GitHub AI Credits，超额为按量计费。[官方套餐页](https://github.com/features/copilot/plans)
- `actual`：组织方案提供许可、政策和用量控制；不同功能与套餐的 MCP/Cloud agent 权限并不完全相同，须按官方对照表核对。
- `inferred`：适合 GitHub 原生 issue→PR→review 流程；凌镜现阶段已有 Codex，增加 Copilot 只有在 GitHub 云端委派/组织策略产生明确净收益时才值得。

### 5.4 Cursor

- `actual`：Cursor 是 IDE + 前台 Agent + 云端 Background Agents；支持 MCP、Skills、Hooks 和 Cloud agents。Background Agent 在 Cursor 的 AWS 隔离 VM 中运行，需向 GitHub App 授予仓库读写权限，有互联网访问，并会自动运行终端命令。[Background Agents 官方文档](https://docs.cursor.com/background-agent)
- `actual`：官方明确警告：云端 Agent 自动运行命令会带来提示注入和数据外传风险；Privacy Mode 下不训练代码，但任务期间仍需在其基础设施存放代码。
- `actual`：调研日官网个人 Pro 为 20 美元/月、Teams 为 40 美元/用户/月；Enterprise 提供仓库、模型、MCP、自动运行、浏览器和网络控制、审计日志等。[官方定价](https://cursor.com/pricing)
- `inferred`：适合偏 IDE 的开发者；Owner 当前主要通过对话驱动开发，迁移成本可能高于收益。不能因其“全自动”就放松生产写权限。

### 5.5 Gemini CLI

- `actual`：Google 官方开源终端 Agent，内置文件、Shell、网页抓取和 Google Search，支持 MCP、非交互脚本、GEMINI.md、checkpoint 和 token caching。[官方 GitHub](https://github.com/google-gemini/gemini-cli)
- `actual`：官方 GitHub 在调研日列出个人 Google 账号免费层 60 requests/min、1,000 requests/day；实际可用模型、地区和限额需登录确认。
- `actual`：修改文件和执行 Shell 默认需人工确认；可配置 trusted folders。沙箱可用 macOS Seatbelt、Docker/Podman 等方式，但官方配置文档也指出沙箱默认关闭。[工具文档](https://github.com/google-gemini/gemini-cli/blob/main/docs/reference/tools.md)、[沙箱文档](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/sandbox.md)
- `inferred`：适合低成本第二模型、Google Search 辅助和脚本任务；若用于凌镜必须显式开启沙箱并限制目录/网络，不能把免费额度当作采用理由。

## 6. 研究与浏览器 Agent

### 6.1 ChatGPT deep research

- `actual`：可使用公开网页、指定网站、上传文件和启用的 Apps；先提出研究计划，用户可修改、观察进度和中断，输出带引用/来源链接，并可导出 Markdown、Word、PDF。[官方帮助](https://help.openai.com/en/articles/10500283-deep-research-faq)
- `actual`：2026-02 官方更新称可连接任意 MCP 或 App，并把网页搜索限制到可信网站。[官方发布页](https://openai.com/index/introducing-deep-research/)
- `unknown`：引用存在不等于引用准确支持结论；对动态价格、费用、法规和平台后台数据仍需逐项检查原文、地区、时间与口径。
- `inferred`：适合候选市场的公开线索侦察和反证初稿，不应直接把报告结论写成 `actual` 或跨过需求案件证据闸门。

### 6.2 Gemini Deep Research

- `actual`：先生成可修改/批准的研究计划，再迭代搜索、阅读、补缺口，形成带原始链接的报告；可导出 Google Docs。[Google 官方介绍](https://blog.google/products-and-platforms/products/gemini/google-gemini-deep-research/)
- `actual`：可把 Gmail、Drive（Docs/Slides/Sheets/PDF）和 Chat 与网页来源一起用于研究；Google 官方称该能力对所有 Gemini 用户开放，但地区和账户可用性仍需实测。[Workspace 集成公告](https://blog.google/products-and-platforms/products/gemini/deep-research-workspace-app-integration/)
- `actual`：Google 还通过 Gemini Interactions API 提供 Deep Research Agent，允许开发者嵌入应用。[开发者公告](https://blog.google/innovation-and-ai/technology/developers-tools/deep-research-agent-gemini-api/)
- `inferred`：若 Owner 经营资料主要在 Google Workspace，信息整合优势明显；但凌镜当前没有必要仅为此迁移事实源。

### 6.3 Perplexity Research

- `actual`：官方说明 Research 会迭代搜索、阅读文档并调整计划，生成综合报告，可导出 PDF/文档或转为 Page；在 Web、移动端与 Mac App 可用。[Research 官方帮助](https://www.perplexity.ai/help-center/en/articles/10738684-what-is-research-mode)
- `quoted`：官方称会执行数十次搜索、读取数百来源并通常数分钟完成；本次未独立验证覆盖率、速度和准确性。
- `actual`：Free 只有有限 Research 使用，Pro/Max/Enterprise 提升额度；具体频次见实时套餐页。[官方套餐说明](https://www.perplexity.ai/help-center/en/articles/11187416-which-perplexity-subscription-plan-is-right-for-you)
- `inferred`：适合快速广搜和来源发现；如果目标是严格的一手来源审计，需要额外限制域名并人工剔除二手转述。

## 7. 工作流 / 企业 Agent

（本节只比较能力边界，不建议凌镜当前采购。企业产品通常按席位、用量、连接器和合同组合报价，公开价不能代表总成本。）

| 产品 | `actual` 官方能力与形态 | 权限/治理 | 对凌镜判断 |
|---|---|---|---|
| Microsoft Copilot Studio | 低代码构建 Agent，连接 Microsoft 365、Dataverse、外部动作与 MCP，可发布到多渠道。[官方产品页](https://www.microsoft.com/en-us/microsoft-365-copilot/pricing/copilot-studio) | Entra 身份、环境/DLP、连接器、发布与审计控制。[治理文档](https://learn.microsoft.com/en-gb/microsoft-copilot-studio/security-and-governance) | `inferred`：适合 Microsoft 企业流程，当前单人自用过重；公开价格存在预购/按量等多种口径，采购时需按地区复核 |
| Google Vertex AI Agent Builder / ADK | 在 Google Cloud 上构建、部署和管理 Agent，结合模型、搜索/数据、工具、MCP registry 与受管 Agent Engine。[官方文档](https://docs.cloud.google.com/agent-builder) | IAM、Cloud 审计与受管运行环境 | `inferred`：适合已在 GCP 的生产 Agent；凌镜当前没有迁云理由 |
| AWS Bedrock Agents Classic / AgentCore | Classic 支持 action groups、knowledge bases、guardrails、memory、code interpretation 和 supervisor/协作者式多 Agent。[官方文档](https://docs.aws.amazon.com/en_us/bedrock/latest/userguide/agents-multi-agent-collaboration.html) | IAM、Guardrails、trace；`actual`：官方已将其称为 Agents Classic，并标明 2026-07-30 起不再向新客户开放，应评估 AgentCore。[官方产品页](https://aws.amazon.com/bedrock/agents/) | `inferred`：适合 AWS 企业栈；当前不应为凌镜引入。AgentCore 当前完整费用本轮 `unknown` |
| Salesforce Agentforce | 在 Salesforce 数据与业务动作上构建面向员工/客户的 Agent；官方资料列出 MCP client/gateway 与多 Agent 编排能力。[官方产品页](https://www.salesforce.com/agentforce/) | Salesforce 权限、信任层、审计/护栏由其平台承载；官方价包含 credits、conversation 或用户许可等多口径，[需按用例核价](https://www.salesforce.com/agentforce/pricing/) | `inferred`：只有 Salesforce 是核心事实源时才合理 |
| ServiceNow AI Agents | 在 Now Platform 工作流中执行 IT、客服等任务。[官方产品页](https://www.servicenow.com/products/ai-agents.html) | 继承企业工作流、身份、审批和审计 | `inferred`：明显超出凌镜当前规模与边界 |

## 8. Agent 开发框架

| 框架 | `actual` 能力/部署 | MCP / 多 Agent / 治理 | 对一人创业者与凌镜 |
|---|---|---|---|
| OpenAI Agents SDK | 开源 SDK，用于工具调用、handoff、guardrails、sessions 和 tracing；代码可自托管，模型/API 通常为托管服务。[官方文档](https://openai.github.io/openai-agents-python/) | 支持 MCP 与多 Agent handoff；应用权限仍由开发者实现 | `inferred`：若凌镜明确采用 OpenAI API 做三种独立研究 run，可局部用；不能替代业务状态机 |
| Google Agent Development Kit (ADK) | 开源、模型和部署相对中立，可在本地开发并部署到 Vertex AI Agent Engine 等。[官方文档](https://google.github.io/adk-docs/) | Tools、MCP、多 Agent、评估；云端 IAM 取决于部署 | `inferred`：适合 Google 生态，当前不是必要依赖 |
| LangGraph | 开源低层编排框架，以图和持久状态支持长时、可恢复、人机协作 Agent；可自托管，也有 LangSmith 部署/追踪服务。[官方文档](https://docs.langchain.com/oss/python/langgraph/overview) | 多 Agent 是可组合模式；企业自托管可提供 SSO/RBAC 等，但完整部署依赖也更重。[部署文档](https://docs.langchain.com/langsmith/self-hosted) | `inferred`：只有现有 Go 状态机无法满足明确 LLM 编排需求时再评估；公开云价见[官方定价](https://www.langchain.com/pricing) |
| Microsoft AutoGen | 微软官方开源的事件驱动多 Agent 框架，提供 AgentChat 与 Core 等层次。[官方 GitHub](https://github.com/microsoft/autogen) | 多 Agent 强；生产身份、审批和审计仍需自行建设 | `inferred`：研究/原型价值高，凌镜当前扩 MoA 属冻结范围 |
| CrewAI | 官方开源框架，以 Crews 和 Flows 组织角色型 Agent 与确定性流程，并提供企业控制面。[官方文档](https://docs.crewai.com/) | Tools、多 Agent、流程与追踪；MCP/企业能力依版本与方案 | `inferred`：上手直观，但角色数量容易制造“看似分工”的复杂度，不建议因概念新颖引入 |

## 9. 横向判断：所谓 “agentic workshop” 市场上已经有哪些部件

| Workshop 部件 | 市面已有实现 | 仍需项目自己负责 |
|---|---|---|
| 项目上下文/规则 | AGENTS.md、CLAUDE.md、GEMINI.md、Copilot instructions、Cursor rules/skills | 当前业务边界、事实等级、停止条件是否正确 |
| Token 优化 | 上下文按需加载、缓存、摘要、模型选择、额度/credits | 什么信息值得读、预算上限、错误压缩是否丢失关键事实 |
| 子 Agent 分配 | Codex 多线程/worktrees、Claude subagents、Gemini subagents、框架 handoff | 是否真的可独立、如何验收、冲突如何裁决；数量不是质量 |
| 工具/MCP | 多数编码 Agent 和主流 SDK 已支持 MCP | 最小权限、凭据、允许的读写动作、提示注入与数据外传防护 |
| 安全执行 | 沙箱、trusted folders、命令审批、云 VM 隔离 | 生产账号、资金、发布、采购和删除动作的 Owner 审批 |
| 治理/审计 | 企业 RBAC、策略、审计日志、Compliance API | 业务级不可变证据、来源时间、订单/结算/现金对账 |
| 可观察性 | terminal logs、diff、traces、run history | 指标是否代表真实经营结果，而非 Agent 活跃度 |

`inferred`：所以“先放一套本地文件就不会乱跑”只说对了一半。本地文件能统一意图，但只有执行层沙箱、权限、审批、预算、日志和业务状态机才能形成真正护栏。

## 10. 对 Owner 与凌镜的推荐

### 推荐组合（当前）

1. **主开发与本地执行：继续使用现有 Codex。** 复用仓库 `AGENTS.md`、Skills、沙箱/权限和工具配置，不新建 AI Workshop 产品。
2. **市场研究：按任务使用一种 Deep Research 工具，输出只进入待核验证据。** 要求保存来源 URL、观察时间、原始材料、支持/反证作用和真实性状态。
3. **真实平台动作：继续走凌镜现有/待完善的确定性领域服务和 Owner 审批。** 任何浏览器 Agent 都不能自行采购、发布、投广告、退款或移动资金。
4. **Agent 框架：暂不新增。** 只有当三个独立研究 run 的具体实现证明现有 Go 编排无法满足，而且缺口能写成可测试接口时，再在 OpenAI Agents SDK、Google ADK 或 LangGraph 中做一项小型对比原型。

### 不建议当前做

- 不复制 Codex/Claude/Cursor 已有的通用工作台、MCP 管理器、多 Agent 面板或 Token 仪表盘；
- 不为“看起来像公司架构”购买 Copilot Studio、Agentforce、Vertex Agent Builder 等企业控制面；
- 不用多个 Agent 的一致意见提高证据等级；
- 不把研究报告、页面截图、代码模块或测试通过写成真实市场成立、生产可用或最终利润成立；
- 不让通用 Agent 直接持有跨境平台、支付、银行或广告账户的无限写权限。

## 11. 最小验证方法

若 Owner 想比较产品，不做大规模迁移，只用同一份只读任务测试 2–3 个候选：

1. 任务：针对一个已定义的候选市场问题，找 5 条官方一手证据和 3 条独立反证；
2. 限制：不登录生产账号、不写外部系统、最多 60 分钟、成本上限事先固定；
3. 记录：有效来源率、错误引用数、遗漏反证数、人工复核时间、总成本；
4. 通过：来源可访问且真正支持字段，事实等级正确，人工修正工作明显下降；
5. 失败：引用错配、把推断写成事实、无法限定来源、要求危险权限或复核成本没有下降；
6. 裁决：只选一个主工具；另一个最多保留为独立反证工具。

## 12. 仍然未知

- 各产品在 Owner 所在地区、账户和当前套餐下的实际可用性、速度与限额；
- 对目标跨境平台登录、验证码、动态页面、地区限制和反爬的真实兼容性；
- 研究 Agent 对俄语等目标语言、当地法规和平台后台资料的准确率；
- MCP Server 本身的安全质量，以及第三方 MCP 是否会保留或外传数据；
- 用任何一个 Agent 完成凌镜完整经营事实链的真实成本与失败率。

这些未知必须通过受限实测或真实外部证据解决，不能靠更多产品页或更多 Agent 投票解决。
