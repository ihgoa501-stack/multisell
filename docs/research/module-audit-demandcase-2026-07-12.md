# 候选市场模块独立审计

> 审计日期：2026-07-12
> 范围：`backend-go/internal/domain/demandcase/`、`/api/v1/demand-cases`、前端 `/demand-cases`
> 类型：只读调研；除本报告外未修改代码
> 当前方向：仅服务 Owner 自营跨境商品经营，不验证凌镜软件需求，不预设国家、平台、类目或候选数量

## 1. 结论

候选市场模块已经从 2026-07-11 的 `planned` 发展为一套可运行的研究基础设施：候选组合、八维证据、三类研究 run、原始 payload 摘要、反证、确定性裁决、JWT API 和 Owner 决策页面均已 `implemented`；本次聚焦后端测试 10 项通过，可标记为 `automated_verified`。这不等于候选市场比较、Owner 选定市场或真实经营验证已经完成。

当前最大风险是裁决门比文档定义宽松：手工证据可以用任意不同的 `run_id` 冒充独立反证且不绑定研究快照；`estimated` 可与 `quoted` 一样支撑全部维度；显式 `unknown` 冲突在已有另一条支持证据时不一定阻断；裁决也不要求 `data_reality_result`、来源交叉验证、证据新鲜度或非空停止线。因此，当前 `experiment_ready` 更准确的业务含义只能是“形式字段达到最低研究门槛”，不能视为“市场已选中”或“已获 Owner 批准”。

## 2. 模块业务定义

候选市场不是平台名，而是一个尚待比较的“国家/地区 × 目标消费者 × 需求场景 × 销售渠道”组合；代码另要求目标语言/地区 `target_locale`。候选不代表已经决定进入；已选市场还必须通过经营闸门并由 Owner 批准（`CONTEXT.md:7-13`；`backend-go/internal/domain/demandcase/model.go:31-43`）。

模块在当前唯一主线中的职责是：

1. 保存候选组合和停止条件；
2. 用需求、竞争、获客、履约、合规、收款、售后、利润可验证性八个维度记录支持、反证与冲突（`backend-go/internal/domain/demandcase/model.go:19-29,47-60`）；
3. 只接收 `scout_result / falsifier_result / data_reality_result` 三类研究输入，保存来源、采集时间、原始 payload 与 SHA-256（`backend-go/internal/domain/demandcase/research_contract.go:14-44`）；
4. 输出 `lead / evidence_missing / rejected / experiment_ready` 研究裁决和六行 Owner 决策卡（`backend-go/internal/domain/demandcase/model.go:5-18,89-118`）；
5. 只为后续 Owner 决定是否进入只读数据预检提供依据，不负责采购、发布、投放或证明真实付款（`backend-go/internal/domain/demandcase/service.go:136-144,193-214`；`docs/SELF_USE_OPERATING_DIRECTION.md:21-25,79-84`）。

## 3. “好”的可验证定义

一个好的候选市场模块，应同时满足以下可观察条件：

| 条件 | 可验证标准 |
|---|---|
| 定义完整 | 每案明确地区、消费者、具体需求场景、渠道、目标 locale、停止条件；不能仅填平台或宽泛人群。 |
| 同框比较 | 所有候选使用相同八维框架，列表或比较结果能解释相对优劣与淘汰原因，而不只是逐案存档。 |
| 来源可追溯 | 每条可用于裁决的证据都能回到不可变快照；快照原始字节摘要可复算，含来源与观察时间。 |
| 反证真正独立 | 反证 run 有独立执行身份、输入和快照，不与侦察 run 共享 run 身份；不能靠手填不同字符串证明独立。 |
| 真实性不跨级 | `unknown / mock / inferred` 必须阻断；`estimated` 只能表达估算，不能单独证明关键权限、费用、履约、收款、售后或利润可验证性。 |
| 冲突不静默 | 任一关键维度存在未解决冲突或明确 unknown，即使另有支持证据也保持 `evidence_missing`。 |
| 裁决保守 | `experiment_ready` 只表示可提交 Owner 审议；另有明确 Owner 批准事实后才能成为已选市场。 |
| Owner 隔离 | 未认证请求失败；任何读取、写入和快照引用都限定当前 Owner；非 Owner 账号不能执行 Owner 决策动作。 |
| 可复验 | 聚焦服务/API测试、数据库约束测试和前端浏览器流程覆盖核心正反路径；自动测试不冒充真实市场验证。 |
| 真实使用有效 | Owner 能用真实候选完成至少一轮同框比较，基于可追溯证据淘汰或选中一个市场；这才是 `manually_verified/external_observed`，当前尚未成立。 |

上述标准来自当前政策要求：市场选择必须比较关键经营维度，采集须绑定决策、来源、时间、口径和淘汰条件（`docs/SELF_USE_OPERATING_DIRECTION.md:23-37`），关键事实尽量由两个独立来源交叉验证且冲突必须显式保留（同文件 `96-105`）。

## 4. “不好”的可验证定义

出现以下任一情况，可判定模块在该项“不好”或尚未达标：

- 把已有 Ozon、Shopee 或采集器当成市场选择结论；
- 缺少八维任一关键项仍给出可实验结论；
- 以 `unknown / mock / inferred`、无来源、无观察时间或不可回溯快照的证据放行；
- 仅凭不同 `run_id` 文本声称反证独立；
- `estimated` 费用或权限被当成已核实可用；
- 有未解决冲突却因另一条支持证据而放行；
- `experiment_ready` 被页面或下游解释为 Owner 已批准市场；
- 只展示单案，没有同一框架的候选比较与淘汰记录；
- 测试或页面存在被描述为真实市场、生产可用或真实成交已经成立。

## 5. 当前证据状态

| 声明 | 状态 | 证据 |
|---|---|---|
| 当前仅供 Owner 自用，候选市场不预设国家或平台 | `policy` | `docs/SELF_USE_OPERATING_DIRECTION.md:7-25` |
| 候选、八维证据、裁决、研究批次与快照模型存在 | `implemented` | `backend-go/internal/domain/demandcase/model.go:5-118`；迁移 `000078_create_demand_cases.up.sql:1-39`、`000079_create_demand_research_snapshots.up.sql:1-18` |
| 三类输入、payload 哈希、同 run 幂等和角色 run ID 分离存在 | `implemented` | `backend-go/internal/domain/demandcase/research_contract.go:14-18,44-109` |
| API 路由存在并位于 JWT 保护组 | `implemented` | `backend-go/internal/domain/demandcase/routes.go:9-21`；`backend-go/internal/httpx/router.go:592-593,861` |
| 列表、详情、快照和六行决策卡页面存在 | `implemented` | `frontend-next/src/app/(main)/demand-cases/page.tsx:9-20`；`frontend-next/src/app/(main)/demand-cases/[id]/page.tsx:9-18` |
| 八维、独立反证、unknown、冲突、actual 禁入、哈希和幂等测试 | `automated_verified` | 本次 `rtk go test ./internal/domain/demandcase`：1 包、10 项通过；具体断言见 `service_test.go:18-165`、`research_contract_test.go:19-75`、`handler_test.go:26-54` |
| 验收数据库已有 1 个美国/Amazon 候选、3 个快照、9 条证据，状态 evidence_missing | `actual`（引用既有审计，未在本次重新连接数据库） | `docs/research/project-truth-audit-2026-07-12.md:78-86` |
| 这些公开来源内容证明账号、费用、履约、售后或利润 | `unknown`；公开内容仅 `quoted` | 同上 `:80-86` |
| 候选市场已经完成同框比较并由 Owner 选定 | `unknown/未完成` | `docs/PROJECT_STATUS.md:9-20` |
| 前端核心浏览器流程、生产部署、真实账号与外部数据 | `unknown` | 本次未运行浏览器 E2E、未连接生产或外部平台；2026-07-11 审计亦明确当时无核心 E2E（`docs/research/project-truth-audit-2026-07-11.md:88-94`） |

## 6. 关键漏洞

### P0：直接证据接口可伪造“独立反证”

`POST /:id/evidence` 与 `POST /:id/falsifications` 接受调用方填写的 `run_id`（`handler.go:89-119`）。`AddEvidence` 只校验它非空，不要求 `snapshot_id > 0` 或快照属于同案/同 Owner/对应 run（`service.go:35-48`）。`Evaluate` 仅以反证 run 字符串未出现在支持 run 集合中判断“独立”（`service.go:77-116`）。因此两个手工字符串即可满足独立反证条件。这违背“研究输入限定为三类 run、原始 payload 与快照一致”的现行边界。

最小复现：通过普通 evidence 接口为八维写入同一 `scout-a` 的 quoted support，再通过 falsifications 写入 `fake-b` quoted counter，均不绑定快照；调用 evaluate 可得到 `experiment_ready`。

### P0：关键 unknown 冲突可能不阻断

冲突只有在 `usableEvidence` 为真时才加入 blocker（`service.go:99-101,132-134`）。`unknown` 被定义为不可用，所以若某维已有 quoted support，同时另有明确 `unknown` conflict，该 unknown conflict 自身不会阻断。当前测试只覆盖 quoted conflict，未覆盖 unknown conflict 与已有 support 并存（`service_test.go:118-143`）。这与“关键维度为 unknown 只能 evidence_missing”不一致。

### P1：`estimated` 可单独放行全部关键维度

`usableEvidence` 只排除 unknown/mock/inferred，因此 `estimated` 与 `quoted` 同样可满足八维支持（`service.go:132-134`）。对平台权限、真实费用、履约、收款、售后和利润可验证性而言，估算不能证明可用；当前没有按维度区分最低真实性等级。

### P1：未强制数据现实 run、来源多样性、时效或快照完整性

裁决只看 evidence 行，不要求存在 `data_reality_result` 快照，也不要求八维证据来自多个来源、来源与市场适用范围一致、证据仍在有效期内，或 `raw_sha256` 对数据库原始 payload 重新复算（`service.go:66-129`）。数据库的 `snapshot_id` 只是默认 0 的普通 bigint，没有外键到快照（`000079_create_demand_research_snapshots.up.sql:17-18`）。

### P1：候选比较与 Owner 批准事实链尚未实现

后端只有逐案 List/Get/Evaluate/DecisionCard，没有比较结果或 Owner 批准路由（`routes.go:9-21`）；前端列表展示逐案状态，没有同维度比较、淘汰轨迹或批准动作（页面 `page.tsx:10-20`）。`experiment_ready` 更新由 Evaluate 直接完成（`service.go:121-129`），但系统没有独立的 `selected_market/owner_approved` 事实。这印证 `docs/PROJECT_STATUS.md:15` 的“未完成”。

### P1：研究导入可能生成重复候选并发生模糊归属

每个新的 scout run 都无条件创建 DemandCase（`research_contract.go:80-84`），数据库也没有候选自然键或 batch 内唯一约束（`000078...:1-13`）。后续 falsifier/data reality 按六字段 `.First` 查找（`research_contract.go:85-88`），相同组合有重复行时可能附着到非预期案件。Batch 记录未直接冻结一个候选身份。

### P2：停止条件非必填，Owner 权限只等同于“已登录”

Create 校验地区、消费者、场景、渠道和 locale，但不要求停止条件（`service.go:21-26`），与采集前必须定义淘汰条件的政策有差距。API 从 JWT 用户 ID 直接当作 Owner ID（`handler.go:18-25`），未见 DemandCase 路由级 Owner 角色/RBAC；在仍保留其他账号的环境里，任何已登录用户都可建立和评估自己的“Owner”案件。虽有行级隔离（`service.go:29-32,176-190`），但没有唯一 Owner 身份门。

### P2：前端可能误导和缺少操作闭环

列表把 `experiment_ready` 标成“可申请预检”，措辞比“市场已选中”安全（页面 `page.tsx:9`），但详情无 Evaluate、补证、冲突处置或 Owner 批准入口，只能读取。证据来源列对空 URI 仍渲染“查看原始来源”（详情页 `:16`）；无 blockers 时显示“研究阶段未记录阻塞”（`:15`），这不等于已核验通过。当前未见该页面的专属前端单测或 E2E。

## 7. 最小验证方法

建议先用一个隔离测试库、两个候选组合完成以下最小验证，不连接真实写操作：

1. **自动负向测试**：证明无 snapshot 的手工 run、unknown conflict、仅 estimated 的关键维度、缺 data reality run、重复候选组合均不能得到 `experiment_ready`。
2. **约束测试**：对 `snapshot_id` 建立真实外键/归属验证后，验证跨 Owner、跨 case、错误 run 引用被拒绝；复算 raw payload SHA-256。
3. **API 鉴权测试**：未登录为 401；非 Owner 角色不能创建、导入、评估；Owner A 不能读取或引用 Owner B 的案件和快照。
4. **浏览器最小闭环**：Owner 查看两个候选的同框八维差异 → 打开原始来源 → 看见反证与 unknown → 淘汰一个 → 对另一个仅提交批准审议；页面不得出现采购、发布或投放入口。
5. **真实但只读验收**：为一个真实候选导入三个真正独立 run，人工抽查每条关键结论可回到原始快照；账号权限、真实费用等未外部观察前保持 unknown。

通过标准：前四项自动/人工工程验证通过，只能提升为 `automated_verified/manually_verified`；第 5 项取得外部来源与观察时间后，相关字段才可为 `external_observed`。即使市场获批，也不能证明真实付款、售后闭合或最终利润。

## 8. 不扩大范围的建议

推荐只收紧当前小闭环，不新增平台、Agent、MoA、仪表盘或大型重构：

1. 先堵两个 P0：裁决仅接受绑定有效研究快照的证据；任何 unknown/conflict 都按维度阻断。
2. 再收紧真实性：为八维定义最低证据门，`estimated` 不得单独通过权限、费用、履约、收款、售后和利润可验证性。
3. 补一个最小“候选比较 → Owner 批准已选市场”事实状态，明确分离 `experiment_ready` 与 `owner_approved`，不要把批准塞回现有状态名。
4. 修正重复候选归属，并强制非空停止线；保持三类 run，不增加新 Agent 类型。
5. 只补上述风险对应的后端负向测试和一个核心浏览器 E2E。得到一次真实 Owner 使用反馈前，不增加相邻功能。

当前最需要 Owner 验证的不是“页面是否好看”，而是：用两个真实候选时，系统能否让 Owner 看清同框差异、最强反证、未知成本和停止线，并安全地只选出一个进入下一步只读预检。
