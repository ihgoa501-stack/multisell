# 小Q（xiaoq）模块技术质量审计

> 审计日期：2026-07-12
> 审计对象：`backend-go/internal/domain/xiaoq/`、`frontend-next/src/app/(main)/xiaoq/`、`frontend-next/src/features/xiaoq/` 及 Capability 契约
> 审计性质：仓库当前磁盘状态的只读技术审计；不评价回答是否带来经营结果
> 证据等级：源码存在为 `implemented`；本次实际运行通过为 `automated_verified`；真实模型稳定性、成本和生产体验为 `unknown`

## 结论

**总分：27/35，开发质量“良好，读权限和证据追溯基础扎实，但执行契约尚未完全落地”。**

小Q没有直接读写任意业务表，而是通过窄接口调用 demandcase、experiment、sourcing1688 的 Owner 范围 Service；输出始终标为 `mock` 或 `inferred` 且 `trusted=false`；每次运行建立 Trace、Capability 事件与 Evidence 引用；真实模型失败不会伪装成功。这些是好基础。

主要短板是 Capability 元数据与实际执行存在漂移：所有能力声明 5 秒超时，但需求案件和经营实验路径直接使用请求上下文调用领域服务和模型，没有应用该超时；Trace 的多步写入也不是单一原子工作流，部分写失败可能留下半成品记录。当前没有 P0，存在 2 个 P1。

## “开发得好 / 不好”的定义

开发得好是：能力有稳定、可执行的契约；只通过权威领域 Service；Owner 和 RBAC 双重隔离；模型只能解释而不能升级事实；超时、失败、取消、费用和证据都能追踪；真实模型错误不会降级为可信回答；前后端和真实路由组合经过验证。

开发得不好是：Capability 只是展示元数据，运行时不执行；Prompt 代替权限或事实校验；模型能看到敏感原始载荷；Trace 写一半却显示成功；超时和成本不可控；测试只用 fake reader/provider 而没有验证实际领域服务与权限链。评分标准：`5`=证据完整且无重要缺口，`4`=扎实但有局部债务，`3`=基本可用但有明显风险，`2`=脆弱，`1`=大部分缺失，`0`=不存在或已知错误。

## 七轴评分

| 轴 | 分数 | 裁决 |
|---|---:|---|
| 正确性 | 4/5 | 输入目标互斥、事实等级和失败语义清晰；两个主路径未执行声明的超时契约 |
| 可读性 / 复杂度 | 3/5 | 文件数量少、接口清楚，但 `service.go` 525 行复制四条编排流程，Trace/Provider/grounding 错误处理重复 |
| 架构边界 | 5/5 | 使用 `DemandCaseReader`、`ExperimentReader`、`Sourcing1688Reader`、`BusinessClosureReader` 窄接口，实际接线到领域 Service；没有业务写入或任意表访问 |
| 安全 / 权限 | 5/5 | JWT Owner 身份、`agent.read/write`、领域 Service Owner 范围校验、Trace Owner 过滤、PII/原始载荷排除均有代码和测试证据 |
| 性能 / 数据库 | 3/5 | 请求有输入长度和部分 5 秒超时；每条消息同步串行写多个 Trace/Evidence 记录，未见批量写、限流/成本预算或完整超时 |
| 测试质量 | 4/5 | 本次后端 race 34 项、相关前端测试通过；覆盖越权、stub、provider 失败、追溯和页面，但主要依赖 fake readers/providers，缺少真实跨域 HTTP/E2E |
| 可运维性 | 3/5 | Trace、provider、token、latency 和失败状态可见；但 Trace 多步持久化不原子，未见小Q专属指标/成本上限/卡死请求检测 |

## 主要发现

### P1：Capability 声明的 5 秒超时没有在所有 active 路径执行

- `capability.go:44-51` 为所有 active 能力声明 `TimeoutSeconds: 5`。
- business closure 在 `service.go:177-221`、sourcing1688 在 `service.go:240-284` 使用 `context.WithTimeout`。
- demand case 的领域读取与模型调用使用原始 `ctx`（`service.go:99-157`）；experiment 同样使用原始 `ctx`（`service.go:342-395`）。
- 影响：同样显示为 active 的能力实际故障边界不同；领域查询或模型卡住时，Owner 请求耗时依赖上游 HTTP/Provider，而不是 Capability 契约。
- 好的完成标准：建立统一 capability runner，在一次调用中同时约束领域读取、Trace 写入和模型请求；超时必须形成可查 Trace 的 `failed/timeout`，并为四类 target 添加超时/取消回归测试。

### P1：Trace 是多步非原子写入，部分失败可能留下难解释的半成品

- 每次调用依次执行 `Start → AppendEvent → AddEvidence(循环) → provider.Chat → AppendEvent → Complete`；任一步失败再尝试 `Complete failed`（`service.go:95-174, 191-237, 248-302, 338-412`）。
- 如果 Trace 存储本身在中途失败，补写失败状态也可能失败；此时已经创建的 trace/evidence 留在数据库，但无法保证有终局状态。`failTrace` 还故意忽略 `Complete` 的错误（`service.go:502-505`）。
- 影响：不是业务数据错误，但会削弱“每次调用都能回答发生了什么”的审计承诺。
- 好的完成标准：TraceWriter 提供可恢复的 run lifecycle（开始、事件追加、终局状态），启动时或读取时识别超时未完成 run；失败状态持久化要有重试/告警，不能只靠同一故障存储再写一次。

### P2：Capability schema 是描述性 map，不是运行时可验证契约

- `Capability.InputSchema/OutputSchema` 使用 `map[string]interface{}`，当前值如 `{"demand_case_id":"positive_int64"}`；`SendMessage` 另写手工 if 分支校验。
- 风险：新增能力时元数据、请求结构和实际分支容易漂移，现有超时问题就是同类信号。
- 建议：使用带版本的 typed schema/validator，并以表驱动方式绑定 capability→权限→timeout→handler；启动测试验证所有 active 能力都有唯一 handler、输入校验和输出类型。

### P2：测试尚未证明真实跨模块与 HTTP 组合

- 后端 34 项覆盖面较好，但 Service 测试主要注入 fake reader/provider/trace；Handler 测试自行构造路由。前端测试 mock API；未发现 `/xiaoq` 从真实 JWT/RBAC 到真实 demandcase/experiment/sourcing1688 Service 再到 Trace 的 Playwright/契约验收。
- 建议：增加隔离 PostgreSQL 集成测试：两个 Owner 交叉读取必须失败；真实领域对象可生成只属于当前 Owner 的 Trace；provider 超时/失败不返回成功答案；页面只展示 active 能力和可追溯链接。

### P2：权限命名和运行成本控制需要收紧

- 所有只读 Capability 的元数据都声明 `RequiredPermission: agent.write`，消息路由也要求 `agent.write`。这在安全上是收紧而非越权，但名称与只读语义不一致，会使权限维护者误判。
- 能记录 tokens/latency，但 Capability 文本明确“模型费用预算仍未确定”；模块内未见每请求 token/cost 上限之外的日预算、并发限制或专属指标。
- 建议：决定并文档化“发起模型推理”为何属于 write；若保留，重命名为更准确的 `agent.invoke`。在有真实调用前先设单请求、并发和日费用硬上限，不扩展更多能力。

## 已验证证据

- `automated_verified`：`cd backend-go && go test -race ./internal/domain/xiaoq`，34 项通过，退出码 0。
- `automated_verified`：相关前端聚焦测试合计 4 文件、24 项通过；其中小Q API、组件和页面测试 3 文件。
- `implemented`：`routes.go:14-23` 的真实领域 Service 接线与 RBAC；`handler.go:18-24,71-86` 的 JWT Owner 和 Trace Owner入口；`service.go:508-519` 的 Trace 所有权过滤；`capability.go:73-89` 的窄接口。
- `unknown`：未调用真实生产模型；没有模型费用和延迟分布；没有真实浏览器/生产权限链验收；测试通过不能证明回答内容正确。

## 推荐顺序与停止条件

1. 先统一执行 Capability 的 timeout/取消/失败契约，并补四类 target 的测试。
2. 再解决 Trace 未完成 run 的恢复和告警；未完成前不要把 Trace 描述为完整审计账本。
3. 建一条真实跨域集成测试和一条浏览器验收，再考虑新增能力。
4. 明确 `agent.write` 的语义和模型费用硬上限；在真实成本未知时不增加自治或外部写能力。

达到“基础好”的最低条件：两个 P1 关闭；所有 active Capability 的元数据与运行时一致；越权、超时、Provider 失败和 Trace 恢复在真实数据库/路由组合中自动验证；小Q继续保持只读且模型输出永远不能升级事实。
