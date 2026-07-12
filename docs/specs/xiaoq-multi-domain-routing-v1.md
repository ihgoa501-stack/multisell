# 小Q多领域消息与能力路由规范 v1

> 状态：`planned`（设计稿，不代表代码已实现）
> 日期：2026-07-12
> 范围：只读能力从 `demand_case` 扩展到 `experiment`、`sourcing1688`、`order` 等领域
> 约束：保持现有 `/api/v1/xiao-q/messages` 客户端兼容；不绕过 Owner 隔离、RBAC、审批、审计和领域状态机

## 1. 结论

推荐把小Q的消息入口稳定为“**消息 + 明确业务对象 Target + 可选能力提示 + 会话/关联标识**”，由服务端做确定性路由：

```text
HTTP鉴权
→ 请求规范化（旧 demand_case_id 转成 Target）
→ Target 解析及 Owner 范围校验
→ 根据 Target 类型按需取得候选 Capability
→ 确定性选择最小只读能力集合
→ 调用领域 Service 取得事实包
→ 记录 Trace / Capability Call / Evidence
→ 模型只解释已取得的事实包
```

模型不得自行拼表、猜 ID、决定权限或把未登记的工具变成可用能力。客户端提供的 `requested_capabilities` 只是缩小范围的提示，不能扩大服务端授权。

当前事实：现有 `MessageInput` 强制要求 `message + demand_case_id`；现有 `MessageResponse` 只表达单个候选市场；当前 active 能力仅为 `demand_case.read` 与 `demand_case.decision_card.read`。本文中的通用 Target、会话、correlation 和其他领域均为 `planned`。

## 2. 设计依据

- 本项目的 [小Q Capability Contract](../governance/XIAOQ_CAPABILITY_CONTRACT.md) 要求小Q只调用登记的领域 Service/Command，保留权限、事实等级、证据、运行和快照追踪。
- OpenAI Agents SDK把本地运行上下文与模型可见上下文分开；Owner ID、权限、依赖和审批状态应留在服务端运行上下文，不能靠 Prompt 保护。[官方 Context 文档](https://openai.github.io/openai-agents-python/context/)
- OpenAI Agents SDK用 `group_id` 关联同一会话的多次 Trace，而每次运行仍有独立 `trace_id`；本文据此区分 `conversation_id`、`correlation_id` 和 `trace_id`。[官方 Tracing 文档](https://openai.github.io/openai-agents-python/tracing/)
- OpenAI Agents SDK Session 用稳定 session ID 保存多轮历史，并明确不要同时叠加两套延续机制；小Q第一版只使用凌镜自己的 `conversation_id`，不同时混用 provider conversation ID。[官方 Sessions 文档](https://openai.github.io/openai-agents-python/sessions/)
- 官方文档建议把只在需要时获取的数据暴露为函数工具，而不是始终塞入上下文；本文据此采用按 Target 类型加载 Capability、按实际选择执行的渐进披露。[官方 Context 文档](https://openai.github.io/openai-agents-python/context/)

这些资料是架构参考，不证明凌镜现有实现已经具备上述能力。

## 3. 核心概念与标识

| 字段 | 含义 | 生命周期 | 谁生成 |
|---|---|---|---|
| `conversation_id` | Owner 与小Q的一条会话 | 多轮消息 | 服务端；客户端可续用已获 ID |
| `client_message_id` | 一次客户端发送/重试的稳定 ID | 单条用户消息 | 客户端 UUID |
| `correlation_id` | 一次 Owner 意图及其重试/后续异步工作的关联 ID | 一次逻辑操作 | 服务端默认生成；可信客户端可回传 |
| `trace_id` | 一次服务端执行记录 | 每次 HTTP 执行 | Trace 服务生成 |
| `target` | 本次问题涉及的明确业务对象 | 单次消息 | 客户端选择，服务端验证 |
| `capability_call_id` | 一次能力调用 | 单次能力调用 | 服务端生成 |

规则：

1. `trace_id` 不充当会话 ID；一次重试产生新 Trace，但复用同一个 `client_message_id` 和 `correlation_id`。
2. `conversation_id` 必须绑定 Owner；请求其他 Owner 的会话统一返回 not found，避免泄露存在性。
3. 会话历史只帮助理解代词和上一轮选择，不是经营事实源。每轮都重新读取 Target 的权威状态。
4. Target ID 使用字符串传输，兼容数字主键、实验字符串 ID 和未来复合外部 ID，禁止前端用 JavaScript `number` 承载超大整数。

## 4. Go DTO

建议新增 DTO，不立即删除现有字段：

```go
type TargetType string

const (
    TargetDemandCase   TargetType = "demand_case"
    TargetExperiment   TargetType = "experiment"
    TargetSourcing1688 TargetType = "sourcing1688_case"
    TargetOrder        TargetType = "order"
)

type MessageTarget struct {
    Type     TargetType `json:"type" binding:"required"`
    ID       string     `json:"id" binding:"required"`
    Relation string     `json:"relation,omitempty"` // primary | related
    Version  string     `json:"version,omitempty"`  // 可选乐观读取版本/快照版本
}

type MessageRequest struct {
    SchemaVersion         string          `json:"schema_version,omitempty"`
    Message               string          `json:"message" binding:"required"`
    Targets               []MessageTarget `json:"targets,omitempty"`
    RequestedCapabilities []string        `json:"requested_capabilities,omitempty"`
    ConversationID        string          `json:"conversation_id,omitempty"`
    CorrelationID         string          `json:"correlation_id,omitempty"`
    ClientMessageID       string          `json:"client_message_id,omitempty"`

    // Legacy V1 compatibility. New clients MUST NOT send it together with targets.
    DemandCaseID int64 `json:"demand_case_id,omitempty"`
}

type ResolvedTarget struct {
    Type           TargetType `json:"type"`
    ID             string     `json:"id"`
    Relation       string     `json:"relation"`
    DisplayLabel   string     `json:"display_label,omitempty"`
    DomainStatus   string     `json:"domain_status,omitempty"`
    ResourceVersion string    `json:"resource_version,omitempty"`
}

type CapabilityCall struct {
    CallID       string          `json:"call_id"`
    CapabilityID string          `json:"capability_id"`
    Version      string          `json:"version"`
    Target       MessageTarget   `json:"target"`
    Status       string          `json:"status"` // succeeded | failed | skipped
    ErrorCode    string          `json:"error_code,omitempty"`
    DurationMs   int64           `json:"duration_ms"`
    EvidenceRefs []EvidenceRef   `json:"evidence_refs"`
}

type EvidenceRef struct {
    RefID          string `json:"ref_id"`
    SourceType     string `json:"source_type"`
    SourceID       string `json:"source_id"`
    Title          string `json:"title"`
    Summary        string `json:"summary,omitempty"`
    TruthStatus    string `json:"truth_status"`
    SourceURL      string `json:"source_url,omitempty"`
    ObservedAt     string `json:"observed_at,omitempty"`
    RunID          string `json:"run_id,omitempty"`
    SnapshotID     string `json:"snapshot_id,omitempty"`
    SnapshotSHA256 string `json:"snapshot_sha256,omitempty"`
}

type MessageResponseV2 struct {
    SchemaVersion  string           `json:"schema_version"` // xiao-q.message.v2
    TraceID        string           `json:"trace_id"`
    ConversationID string           `json:"conversation_id"`
    CorrelationID  string           `json:"correlation_id"`
    ClientMessageID string          `json:"client_message_id,omitempty"`
    AgentID        string           `json:"agent_id"`
    Mode           string           `json:"mode"`
    Answer         string           `json:"answer"`
    TruthStatus    string           `json:"truth_status"`
    Trusted        bool             `json:"trusted"`
    Targets        []ResolvedTarget `json:"targets"`
    CapabilityCalls []CapabilityCall `json:"capability_calls"`
    Evidence       []EvidenceRef    `json:"evidence"`
    Unknowns       []string         `json:"unknowns"`
    Warnings       []string         `json:"warnings"`
    Links          []ResponseLink   `json:"links"`
    Provenance     Provenance       `json:"provenance"`

    // Legacy response projection when primary target is demand_case.
    DemandCaseID int64 `json:"demand_case_id,omitempty"`
}
```

### 4.1 事实等级

`EvidenceRef.truth_status` 只能是 `actual | quoted | estimated | inferred | unknown | mock`。回答自身默认为 `inferred`；stub回答为 `mock`。`trusted=false` 的语义保留为“模型回答本身不是权威经营事实”，不应因为引用了 `actual` 证据而变成 `true`。

领域裁决、状态和金额必须保持领域 Service 返回的原值及证据引用。模型不得把：

- `paid` 推断为已签收；
- `delivered` 推断为售后窗口已关闭；
- `profit_amount` 推断为已对账最终利润；
- 工程 `implemented` 推断为生产或外部事实已验证。

### 4.2 兼容规范化

```go
func NormalizeMessageRequest(in MessageRequest) (NormalizedRequest, *APIError) {
    switch {
    case len(in.Targets) == 0 && in.DemandCaseID > 0:
        // 旧客户端等价于一个 primary demand_case target
    case len(in.Targets) > 0 && in.DemandCaseID == 0:
        // 新请求
    case len(in.Targets) > 0 && in.DemandCaseID > 0:
        // 拒绝：避免两个事实源冲突
    default:
        // 拒绝：缺少 target
    }
}
```

旧请求 `{message, demand_case_id}` 的成功响应继续保留所有现有字段和 `demand_case_id`；只新增字段，不改现有含义。旧客户端无需理解 `schema_version`、`targets` 或 `capability_calls`。

## 5. TypeScript DTO

```ts
export type XiaoQTargetType =
  | 'demand_case'
  | 'experiment'
  | 'sourcing1688_case'
  | 'order';

export interface XiaoQTarget {
  type: XiaoQTargetType;
  id: string;
  relation?: 'primary' | 'related';
  version?: string;
}

export interface XiaoQMessageRequestV2 {
  schema_version: 'xiao-q.message.v2';
  message: string;
  targets: XiaoQTarget[];
  requested_capabilities?: string[];
  conversation_id?: string;
  correlation_id?: string;
  client_message_id: string;
}

export interface XiaoQLegacyMessageRequest {
  message: string;
  demand_case_id: number;
}

export type XiaoQMessageRequest =
  | XiaoQMessageRequestV2
  | XiaoQLegacyMessageRequest;

export interface XiaoQCapabilityCall {
  call_id: string;
  capability_id: string;
  version: string;
  target: XiaoQTarget;
  status: 'succeeded' | 'failed' | 'skipped';
  error_code?: string;
  duration_ms: number;
  evidence_refs: XiaoQEvidenceRef[];
}

export interface XiaoQMessageResponseV2 extends XiaoQMessageResponse {
  schema_version: 'xiao-q.message.v2';
  conversation_id: string;
  correlation_id: string;
  client_message_id?: string;
  targets: Array<XiaoQTarget & {
    display_label?: string;
    domain_status?: string;
    resource_version?: string;
  }>;
  capability_calls: XiaoQCapabilityCall[];
  evidence: XiaoQEvidenceRef[];
  warnings: string[];
}
```

前端解析器必须容忍旧响应缺少新增字段，并将其规范化为空数组；不得根据链接或回答文本反推 Target。

## 6. Capability 路由

### 6.1 能力命名与首批规划

能力 ID 使用 `<domain>.<resource>.<verb>` 或兼容已有 ID；版本独立于消息 schema。

| Target | 最小只读能力 | 状态 | 激活前置条件 |
|---|---|---|---|
| `demand_case` | `demand_case.read`、`demand_case.decision_card.read`（均为 v1） | `active` | 已有实现与测试 |
| `experiment` | `experiment.case.read`、`experiment.gates.read`、`experiment.owner_summary.read`（均规划 v1） | `deferred` | Owner隔离接口、事实/对象链接完整映射、契约测试 |
| `sourcing1688_case` | `sourcing1688.case.read`、`sourcing1688.validation.read`（均规划 v1） | `deferred` | 仅允许已关联合规市场和实验；来源快照与图片/成本证据可追溯 |
| `order` | `order.detail.read`、`order.fulfillment_facts.read`、`order.settlement_links.read`（均规划 v1） | `deferred` | **先补 Owner 范围读取**；敏感字段脱敏；结算/利润/现金严格分态 |

`order.Service.Get(id)` 当前没有 Owner 参数，不能直接包装为 active Capability。必须先由订单领域提供按 Owner 鉴权的权威读取方法；小Q适配层不得自行查询订单表补救。

### 6.2 候选能力按需加载

1. Capability Registry先按 `status=active`、Owner权限、执行模式 `read_only` 过滤。
2. 再按 `Target.Type` 取得该领域能力；不把全系统能力说明全部送进模型。
3. 客户端若给出 `requested_capabilities`，取它与候选集合的交集；出现非候选 ID 返回明确错误，不静默扩大。
4. 领域 Route Policy根据“问题类别 + Target状态”选择最小集合；安全关键选择必须是代码规则，不由模型自由决定。
5. 仅把实际成功调用得到的事实包送给模型；能力失败时不得让模型依据记忆补答案。

建议路由接口：

```go
type RouteContext struct {
    OwnerID       int64
    Message       string
    Targets       []ResolvedTarget
    Allowed       []Capability
    ConversationID string
    CorrelationID string
}

type RoutePlan struct {
    Intent          string
    Calls           []PlannedCapabilityCall
    RequiredAll     bool
    PromptPolicyID  string
}

type CapabilityRouter interface {
    Plan(context.Context, RouteContext) (RoutePlan, error)
}
```

### 6.3 确定性路由优先级

按以下顺序裁决，首次失败即停止：

1. 验证 schema、消息长度（继续使用 2000 rune 上限）、Target数量（v1最多 8，且仅1个 `primary`）。
2. 从 JWT取得 Owner ID；忽略请求体里的任何 owner/user字段。
3. 验证会话属于当前 Owner。
4. 逐个解析 Target，执行 Owner隔离与资源状态检查。跨Owner和不存在均返回同一 `TARGET_NOT_FOUND`。
5. 验证多个 Target 是否存在权威关联：例如 `experiment.demand_case_id`、sourcing case的 `experiment_id`、订单通过实验对象链接关联。客户端声称 `related` 不构成关联证据。
6. 取得每个 Target 的 active、read_only Capability。
7. 根据显式问题类别选择最小能力：
   - “结论/能否下一步/为什么”：主对象 detail + gate/decision card；
   - “证据/最强反证/缺什么”：detail + evidence；
   - “订单付款/签收”：order detail + fulfillment facts，不能自动加载利润；
   - “最终利润/现金”：settlement links + final profit + cash reconciliation，三者分别呈现。
8. 若语义有歧义且选择不同能力会改变答案，不猜测，返回 `ROUTE_AMBIGUOUS` 及可选 Target/问题提示。
9. 先执行能力并记录调用，再调用模型。`RequiredAll=true` 时任一必需能力失败，整次回答失败；不能以部分事实生成完整结论。

禁止“先让LLM选任意工具再校验”的路由方式。未来可让模型在已过滤的低风险候选中做建议，但服务端仍必须验证计划并记录选择理由。

## 7. 多Target规则

- 默认只允许一个 primary Target；related Target只能由前端展示当前权威链路后由Owner勾选，或由服务端从 primary解析。
- v1不支持任意同类型对象批量比较。市场比较应使用单独、经过验证的 comparison Capability，而不是塞入多个 `demand_case.read` 后让模型自由比较。
- 如果 Target链不一致（例如订单不属于该实验），返回 `TARGET_RELATION_MISMATCH`，不向模型暴露任一对象内容。
- 每个 evidence ref必须能指回产生它的 capability call和 Target；去重只能合并相同 `(source_type, source_id, snapshot_sha256)`，不能只按标题合并。

## 8. 会话与上下文策略

建议新增服务端 `xiao_q_conversations` 与 `xiao_q_messages`（实施时另做迁移设计），但第一阶段可以只生成会话ID并保存最小元数据。安全规则：

1. 只保留最近 N 轮或压缩摘要；摘要 `truth_status=inferred`，不能作为 Evidence。
2. 每轮保存原始 Target、解析后的版本、实际 Capability调用、Trace ID和回答事实等级。
3. 新一轮未提供 Target时，只有在同一会话上一轮存在唯一 primary Target且未被删除时才可继承；响应加 `warnings=["TARGET_INHERITED_FROM_CONVERSATION"]`。第一版更保守：仍要求显式 Target。
4. 更换 primary Target时前端明确显示“当前对象已切换”，不静默携带旧对象事实。
5. Provider侧 continuation ID若未来引入，只作为内部实现字段，不暴露成凌镜 `conversation_id`，也不与本地历史双重拼接。
6. Trace默认不保存不必要的订单收件人电话、地址、支付信息；Evidence摘要做字段级脱敏。

## 9. API与错误码

保持现有路由：

```text
POST /api/v1/xiao-q/messages
GET  /api/v1/xiao-q/identity
GET  /api/v1/xiao-q/capabilities
GET  /api/v1/xiao-q/traces/:trace_id
```

`GET /capabilities` 可向后兼容增加查询：

```text
?target_type=experiment&mode=read_only
```

未传查询参数仍返回所有当前Owner可见的 active能力。不得把 deferred能力伪装成 available；若产品需要展示路线图，应另取静态说明，不混入运行时能力列表。

建议错误 envelope（保持现有顶层 `code/message/data`，新增稳定机器码）：

```json
{
  "code": 400,
  "message": "请选择一个经营对象后再提问",
  "data": {
    "error_code": "XQ_TARGET_REQUIRED",
    "trace_id": "",
    "correlation_id": "corr_...",
    "retryable": false,
    "details": {}
  }
}
```

| HTTP | `error_code` | 触发条件 | 可重试 |
|---:|---|---|---|
| 400 | `XQ_INVALID_REQUEST` | schema、消息、ID格式或数量非法 | 否 |
| 400 | `XQ_TARGET_REQUIRED` | 新旧Target均缺少 | 否 |
| 400 | `XQ_TARGET_CONFLICT` | 同时传 `targets` 和 `demand_case_id` | 否 |
| 400 | `XQ_TARGET_RELATION_MISMATCH` | 多对象无权威关联 | 否 |
| 409 | `XQ_ROUTE_AMBIGUOUS` | 无法安全确定问题对象/能力 | 修正请求后 |
| 403 | `XQ_CAPABILITY_FORBIDDEN` | Owner无所需权限 | 否 |
| 404 | `XQ_TARGET_NOT_FOUND` | 对象不存在或不属于Owner | 否 |
| 404 | `XQ_CONVERSATION_NOT_FOUND` | 会话不存在或不属于Owner | 否 |
| 409 | `XQ_CAPABILITY_UNAVAILABLE` | 能力 deferred/disabled/无active版本 | 稍后可能 |
| 422 | `XQ_DOMAIN_STATE_UNSUPPORTED` | 对象状态不允许该只读解释 | 状态变化后 |
| 502 | `XQ_PROVIDER_FAILED` | LLM失败；返回已生成Trace ID但不伪造回答 | 是 |
| 503 | `XQ_CAPABILITY_FAILED` | 必需领域读取失败/超时 | 是 |
| 504 | `XQ_RUN_TIMEOUT` | 全链路超时 | 是 |

鉴权失败继续使用全局 401/403。内部错误信息不得直接返回；Trace可供Owner查看经过脱敏的失败阶段。

## 10. 前端交互

### 10.1 目标选择

替换当前“手填候选市场 ID”为统一 Target Picker：

1. 先选业务类型（候选市场、实验、1688草稿、订单）；只显示存在 active能力的类型。
2. 通过对应领域的 Owner范围搜索 API选择具体对象，显示业务标签和状态，不要求Owner记数字ID。
3. 选择后显示“当前提问对象”卡片，包含类型、名称/编号、状态、事实更新时间和来源页面链接。
4. 从业务详情页进入 `/xiaoq?target_type=experiment&target_id=...` 时，前端仍调用后端验证，URL参数不视为已授权。
5. 对 related对象由服务端返回权威关联后展示；用户不能自由输入任意关联ID。

### 10.2 消息与结果

- 每次发送生成 `client_message_id`，网络重试复用；新问题生成新的值。
- 第一条成功响应后保存 `conversation_id`；同页后续消息回传。
- 回答卡按“结论（inferred）/ 系统事实 / 反证与未知 / 能力调用 / 执行记录”分区。
- `mock`、`unknown`、`inferred` 使用醒目标识；领域状态和模型解释视觉上分开。
- Capability失败时展示失败能力和Trace，不把旧回答留在界面上冒充本次结果。
- 切换Target时保留历史但插入对象切换分隔线，输入区始终显示当前Target。
- 前端不得根据关键词自行决定实际Capability，只能发送可选提示；最终调用清单以响应为准。

### 10.3 渐进发布

1. 先让页面同时支持 legacy request和V2 response解析。
2. 后端上线Normalize与V2新增字段，旧 `demand_case_id` 测试保持通过。
3. 前端改用 Target Picker并发送V2。
4. 激活 `experiment` 只读能力；验证后再逐域激活。
5. `order` 最后接入，且必须先完成Owner隔离和敏感数据裁剪。

## 11. 测试矩阵

### 11.1 请求兼容与校验

| 用例 | 期望 |
|---|---|
| 旧 `{message,demand_case_id}` | 200；等价规范化；保留旧响应字段 |
| V2单 `demand_case` Target | 与旧请求调用相同两个active能力 |
| 同时传旧ID与targets | 400 `XQ_TARGET_CONFLICT` |
| 无Target、空消息、超2000 rune、9个Targets | 对应400；不创建模型调用 |
| Target ID为超大数字字符串 | 不丢精度；交由领域解析 |
| 两个primary Targets | 400 `XQ_INVALID_REQUEST` |

### 11.2 权限与隔离

| 用例 | 期望 |
|---|---|
| Owner A读取Owner B对象 | 404 `XQ_TARGET_NOT_FOUND`；Trace/日志不泄露标题 |
| Owner A复用Owner B conversation | 404 `XQ_CONVERSATION_NOT_FOUND` |
| 缺少Capability权限 | 403；不调用领域Service/模型 |
| order尚无Owner隔离Reader | Registry不得标active；请求409 unavailable |
| Trace读取跨Owner | 404，保持现有行为 |

### 11.3 路由与按需加载

| 问题/Target | 期望调用 | 明确不调用 |
|---|---|---|
| demand case“最强反证” | case read + decision card | experiment/order |
| experiment“为什么不能下一步” | case + gates + owner summary | sourcing/order（除非显式且有关联） |
| order“付款了吗” | detail + fulfillment facts | final profit/cash |
| order“最终赚了多少且钱回来了吗” | settlement links + final profit + cash reconciliation | 未关联其他订单 |
| 指定不属于候选集的requested capability | 409 unavailable/403 forbidden；不静默忽略 |
| 任一RequiredAll能力失败 | 整体503；模型不被调用 |

### 11.4 事实与证据

| 用例 | 期望 |
|---|---|
| Evidence为mock/inferred/unknown | 原样保留；不能进入已证明结论 |
| 订单paid但未delivered | 回答不得声称签收 |
| 有profit记录但结算未reconciled | 不得称最终利润 |
| final profit已确认但cash未回收 | 两个独立状态分别展示 |
| 相同标题、不同snapshot | 保留两条，不错误去重 |
| LLM把事实升级 | 输出验证器拒绝/降级并记录Trace；不得返回可信结论 |

### 11.5 会话、重试与追踪

| 用例 | 期望 |
|---|---|
| 同conversation两轮 | 两个trace_id，同conversation/group；各自token独立记录 |
| 网络重试同client_message_id | 新Trace或幂等复用策略明确；correlation保持一致，不重复展示两条用户消息 |
| 切换primary Target | 新响应只使用新Target事实；显示切换warning |
| provider失败 | 502 + trace/correlation；无stub可信回退 |
| capability超时 | 503/504；调用状态failed；无模型补写 |
| Trace敏感字段检查 | 电话、地址、支付凭证不进入默认Trace payload |

### 11.6 前端

- legacy响应解析回归；新增字段缺失不崩溃。
- 只有active Target类型可选择；deferred不显示为可用。
- Picker使用Owner范围API，错误和空态可理解。
- 发送、重试、换Target、会话续问的ID行为符合规范。
- Answer Card正确区分系统事实、模型推断、未知和mock。
- 从业务详情深链进入后仍以服务端验证结果为准。
- 键盘、加载、失败、移动端布局和无障碍标签测试。

## 12. 实施验收门

任何领域只有同时满足以下证据，才能把 `xiao_q_support` 从 `deferred` 改为 `active`：

- 领域提供Owner隔离的权威只读Service接口，而非小Q直接查表；
- Capability Registry包含完整契约、版本、权限、超时、Evidence Policy；
- Target resolver与关系验证通过跨Owner及错链测试；
- Evidence保留事实等级、来源、观察时间、run和snapshot（适用时）；
- 路由只按需加载并通过“未调用无关能力”测试；
- LLM/provider失败不会回退为可信答案；
- Trace、conversation、correlation和token记录可追溯且敏感字段已裁剪；
- Go单元/契约测试、前端测试及至少一条真实浏览器流程通过。

工程测试通过只证明测试环境行为，不证明真实市场、订单、结算、利润或现金事件已经发生。

## 13. 明确不做

- 不在本规范中开放 mutate、external_write 或 financial 能力。
- 不让小Q拥有任意SQL、任意HTTP或“调用整个系统”的超级工具。
- 不为了统一入口重写现有领域状态机。
- 不把旧A1-A12/G0-G3整批注册为小Q能力。
- 不允许一个自由Prompt同时跨多个无关联业务对象检索。
- 不把会话记忆或模型总结当成经营证据。
