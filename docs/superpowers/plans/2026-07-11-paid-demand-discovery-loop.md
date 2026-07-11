# 真实付费需求发现循环实施计划

> 设计规格：`docs/superpowers/specs/2026-07-11-paid-demand-discovery-loop-design.md`
>
> 当前原则：先建立可证伪的需求案件与证据裁决，再接真实只读数据；真实采购、发布、广告和资金动作不属于本计划的自动执行范围。

## 目标结果

第一阶段完成时，凌镜可以：

1. 接收 AI 自动产生的结构化需求假设和原始证据；
2. 强制另一条独立反证链攻击假设；
3. 用确定性状态机阻止搜索、评论、榜单、加购或普通订单冒充真实付费；
4. 向 Owner 只展示六行决策卡；
5. 对通过研究门槛的案件生成一个仍需 Owner 批准的实验草案；
6. 在接入真实订单后按付款、签收、售后和结算逐级裁决，并支持晚到退款重开；
7. 首个真实研究批次可以明确输出 `假设 / 证据不足 / 被污染 / 已驳回 / 可实验`，但在真实交易前绝不输出“已发现付费需求”。

## 明确不做

- 不预设 Ozon、Shopee 或 Shopify 是最终市场；
- 不预设目标客户、类目或商品；
- 不新建通用 Agent 平台、MoA、外部 SaaS 或多租户能力；
- 不让 LLM 决定状态转换或最终利润；
- 不自动采购、发布、投广告、退款、补货或花钱；
- 不用 mock、seed、亲友单或公开竞品数据验收商业结果。

## Task 1：修正当前事实源与旧流程入口

**目的**：确保任何 Agent 和开发者不会继续从“Ozon 商品采集→候选商品”开始。

**文件**：`docs/SELF_USE_OPERATING_DIRECTION.md`、`docs/CURRENT_DIRECTION_AND_PRIORITIES.md`、`docs/ROADMAP.md`、`TODOS.md`、`README.md`、`AGENTS.md`、`CLAUDE.md`、`docs/INDEX.md`。

**工作**：

1. 将当前主线改为 `需求假设→独立反证→Owner批准实验→真实裁决`；
2. 将 Ozon 标记为待验证数据源，而不是默认首发市场；
3. 将已有 Ozon 列表采集明确标记为竞争/价格线索能力，不是付费需求发现；
4. 将旧 `20→3→1` 改为研究漏斗，不允许 20 个商品链接充数；
5. 在文档索引加入设计、计划和三份对抗研究。

**验收**：活跃文档不再把 Ozon 写成已选市场或把商品链接写成真实机会；本地 Markdown 链接审计无新增断链；`git diff --check` 通过。

## Task 2：建立需求案件领域模型与数据库迁移

**目的**：给每个主张一个可审计、不可跨级的事实容器。

**新增文件**：

- `backend-go/internal/domain/demandcase/model.go`
- `backend-go/internal/domain/demandcase/service.go`
- `backend-go/internal/domain/demandcase/state_machine.go`
- `backend-go/internal/domain/demandcase/demandcase_test.go`
- `backend-go/migrations/000075_create_demand_cases.up.sql`
- `backend-go/migrations/000075_create_demand_cases.down.sql`

**核心模型**：`DemandCase`、`DemandEvidence`、`FalsificationCase`、`DemandExperiment`、`DemandEvent` 和 `DemandVerdict`。模型分别承载主张、原始证据、反证、冻结实验、只追加真实事件和当前裁决。

**先写失败测试**：

1. 搜索证据不能升级到 `paid_provisional`；
2. 订单创建不能升级到 `delivered_provisional`；
3. 关联单和测试单进入 `polluted`；
4. 未完成售后窗不能进入 `retained_provisional`；
5. 关键成本含 `unknown/estimated` 时不能进入 `proven_positive_contribution`；
6. 晚到退款会从终局改为 `reopened`；
7. 每次状态变化都有不可变事件和操作者。

**验收命令**：

```bash
cd backend-go
go test ./internal/domain/demandcase -count=1
go vet ./internal/domain/demandcase/...
```

## Task 3：提供只读案件 API 和六行 Owner 决策卡 API

**目的**：Owner 不接触内部 Agent 争论，只看案件事实和下一决定。

**新增/修改文件**：`backend-go/internal/domain/demandcase/{handler.go,routes.go,handler_test.go}`、`backend-go/internal/httpx/router.go`、路由目录（若当前项目要求）和 `docs/reference-module-catalog.md`。

**API**：

- `GET /api/v1/demand-cases`
- `GET /api/v1/demand-cases/:id`
- `GET /api/v1/demand-cases/:id/decision-card`
- `POST /api/v1/demand-cases`：仅内部 AI/授权操作者创建假设；
- `POST /api/v1/demand-cases/:id/evidence`：只追加证据；
- `POST /api/v1/demand-cases/:id/falsifications`：只追加反证；
- `POST /api/v1/demand-cases/:id/evaluate`：运行确定性裁判；
- `POST /api/v1/demand-cases/:id/experiment-proposal`：仅生成草案，不执行外部动作。

**安全**：全部使用 JWT；mutation 写 operation log；客户数据只返回去标识化字段；实验草案不得绕过 Approval；不提供 production execute 路由。

**六行卡契约**：

```json
{
  "hypothesis": "我们怀疑什么",
  "proven": "真实证据是什么",
  "not_proven": "这还不能证明什么",
  "strongest_counterevidence": "最强反证",
  "next_authority_or_cost": "下一步需要的权限或金额",
  "stop_condition": "不做会怎样以及停止线"
}
```

**验收命令**：`cd backend-go && go test ./internal/domain/demandcase ./internal/httpx -count=1`。

## Task 4：建立 AI 研究输入契约，不新增自治 Agent

**目的**：让外部 Codex 研究 Agent、现有 LLM 或未来调度器通过同一个可信接口提交结果，避免扩张内部 Agent roster。

**新增文件**：`backend-go/internal/domain/demandcase/research_contract.go`、`research_contract_test.go` 和 `docs/features/demand-research-agent-contract.md`。

**输入契约**：

- `scout_result`：只能创建 `lead`；
- `falsifier_result`：只能追加反证，不能批准实验；
- `data_reality_result`：只能标记字段可得性；
- `judge_request`：不接收 LLM 建议状态，只读取事实运行状态机。

**校验**：每个事实有来源和时间；推断标记 `hypothesis`；缺失值保持 `unknown`；raw payload hash 可重算；侦察和反证来自不同 run ID；无来源数字、客户画像或销量自动拒绝。

**验收命令**：`cd backend-go && go test ./internal/domain/demandcase -run ResearchContract -count=1`。

## Task 5：把真实研究批次导入证据账本

**目的**：不用模拟商品证明系统；用已完成的三份独立研究作为“研究流程证据”，但不把它们冒充市场需求。

**输入**：`deliverables/research/paid-demand-signal-map.md`、`paid-demand-falsification-protocol.md`、`demand-data-access-reality.md`。

**新增文件**：`backend-go/cmd/demand-research-import/main.go`、`main_test.go` 和 `backend-go/testdata/demand-research/` 下的结构化去敏固定样本。

**工作**：

1. 由 AI 将报告转换为严格 JSON 契约；
2. 导入时验证来源、事实/推断/未知和独立 run ID；
3. 建立一条“尚无真实付费需求”的基线案件；
4. 生成当前裁决 `evidence_missing`，列出所需账号只读证据；
5. 禁止把报告中的示例问题自动转成已确认客户需求。

**验收**：重复导入幂等；缺来源、伪造销量、把注意力标付款的样本被拒；导入后不存在 `paid_provisional` 以上案件。

## Task 6：平台数据访问预检，只读且零花费

**目的**：先证明我们能取得什么数据，再决定从哪个平台研究。

**新增/修改文件**：`backend-go/internal/domain/demandcase/platform_preflight.go`、测试、复用 `internal/domain/integrations/{ozon,shopee,shopify}.go`，新增 `docs/ops/PAID_DEMAND_READ_ONLY_PREFLIGHT.md`。

**平台判定**：`available / requires_owner_access / requires_listing / requires_transaction / unavailable / unknown`。

**顺序**：

1. 检查凭证是否配置，但不打印密钥；
2. 对已授权平台执行最小只读调用；
3. 保存脱敏响应结构、时间、scope 和错误；
4. 未授权平台只生成 Owner 授权卡，不尝试登录或绕过验证；
5. Shopify 没有目标流量时标记为经营测量工具，不标记为需求来源。

**验收**：至少一个平台得到真实 `available` 或明确 `requires_owner_access` 证据；不能以代码存在证明连通。

## Task 7：最小 Owner 页面，只呈现真实案件

**目的**：替换当前 Mock 决策入口的第一块真实信息，不重做整站视觉。

**新增/修改文件**：

- `frontend-next/src/app/(main)/demand-cases/page.tsx`
- `frontend-next/src/app/(main)/demand-cases/[id]/page.tsx`
- `frontend-next/src/types/demand-case.ts`
- `frontend-next/src/config/menu.ts`
- `frontend-next/e2e/tests/demand-case.spec.ts`

页面只显示状态、六行卡、证据来源、最强反证、未知项和“请求授权/查看实验草案”。不显示置信度分、虚构销量、Agent 投票或生产执行按钮。

**验收命令**：`cd frontend-next && npm test && npm run build`，以及 `cd frontend-next/e2e && npx playwright test tests/demand-case.spec.ts`。

## Task 8：完成首个自动研究—反证循环

**目的**：让多个 AI 对真实公开来源运行一轮，但诚实停在证据允许的位置。

**边界**：侦察、反证和数据现实三个独立 run；保存来源、时间和原始快照；最多产生 10 个假设；每条必须经过反证；无账号内聚合数据时不得宣称需求已确认；不购买、不发布、不投广告。

**结果**：被淘汰的假设、仍缺数据的假设、最多 3 个值得请求只读账号数据的案件；若没有存活案件，输出“本轮未发现可实验假设”，不强行推荐。

## Task 9：真实小额实验前置门

**目的**：只有研究和数据权限都通过后，才允许进入下一份独立计划。

**必须齐全**：具体需求—规格假设；平台/国家和可观察人群；同规格供应商、样品、合规、包装参数、物流和完整成本；预注册终点、排除规则、时间/曝光/预算上限和售后窗口；Owner 对采购、发布、广告分别批准；现金预算与不可回收损失上限冻结。

本计划到此停止。真实采购、发布和广告必须另开计划，并在动作发生时再次取得 Owner 批准。

## 全局验证

```bash
cd backend-go
go test ./...
go build ./...
go vet ./internal/domain/demandcase/... ./internal/httpx/...

cd ../frontend-next
npm test
npm run build

cd ..
git diff --check
```

已知的仓库级既有失败必须明确记录，不能用局部通过冒充全局通过，也不能顺手修改无关模块。

## 实施顺序与提交边界

每个 Task 独立完成红—绿—重构、局部验证和一次聚焦提交。不得将当前工作区中无关的部署、CI、历史文档或其他人的改动混入提交。

优先级：`Task 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → Task 9 审议`。
