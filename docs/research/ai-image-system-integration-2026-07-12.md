# AI 作图与凌镜经营闭环接入审查

> 日期：2026-07-12
> 范围：只读审查当时代码；不修改业务代码，不连接真实图片模型、1688、销售渠道或生产数据
> 证据语言：沿用 `policy / planned / implemented / automated_verified / manually_verified / external_observed / reconciled / mock / inferred / unknown / superseded`

## 1. 结论

推荐把“AI 作图”在凌镜中定义为：**基于一个已通过 Owner 审查、权利可用的真实商品来源图，调用生成式图片模型产生候选商品图，并把输入、模型、提示词、输出、成本、权利与渠道检查完整留痕；候选图必须由 Owner 选择后，才允许进入现有 1688 内部草稿审批。**

第一版只做一个小闭环：

```text
已批准市场与 active 实验
→ 1688 真实受控快照与 Owner 来源审查
→ 选择一张有 actual 权利证据的来源图
→ 生成一组同用途候选图（建议仅“保持商品本体、替换背景/构图”）
→ 自动记录来源、模型、prompt、参数、输出哈希、费用与失败
→ 确定性安全检查
→ Owner 选择或拒绝
→ 复用现有 product_media_asset 与草稿内容哈希审批
→ listing 仍保持 draft
```

这不是“自动上架”，也不是另建一个图片平台。AI 输出只能是候选素材，不能成为商品真实性、图片权利、合规或渠道通过的证明。

## 2. 什么算 AI 作图

### 2.1 本项目中的边界定义

| 类型 | 是否称为 AI 作图 | 说明 |
|---|---:|---|
| 裁剪、缩放、压缩、白底、格式转换、水印 | 否 | 确定性图片处理；同样输入通常得到同样输出 |
| 抠图、去背景、超分、去物等模型处理 | 条件性 | 属于 AI 图片编辑，但不必然生成新场景；必须记录模型和输入输出 |
| 文生图、图生图、生成式填充、换背景、生成模特/场景 | 是 | 模型创造了输入图中没有的像素或内容 |
| AI 给 prompt、人工或传统程序出图 | 否 | AI 只参与文字建议，不代表图片由生成式模型产生 |

### 2.2 第一版允许的用途

- 从真实商品图生成**候选场景背景或构图变体**；商品主体应尽量保持不变。
- 为同一已批准销售渠道生成满足尺寸/留白要求的候选图。
- 生成结果只用于内部草稿比较，由 Owner 明确选择。

第一版不应包含：凭文字虚构商品、改变颜色/尺寸/材质/配件数量、生成不存在的认证或功效、生成真人背书、自动翻译并烧录未经核验的营销文案、自动发布。

## 3. 当前真实现状

### 3.1 已实现或已有自动验证

| 事实 | 等级 | 代码证据与限制 |
|---|---|---|
| 1688 受控采集在外部读取前校验 `experiment_ready` 市场、同 Owner active 实验、允许阶段、需求案件关联和 opportunity pass | `implemented` / 已有聚焦测试记录为 `automated_verified` | `backend-go/internal/domain/sourcing1688/controlled_fetch.go`；并不证明真实 1688 页面已经采集 |
| 来源图片处理要求来源 URL 存在于已审快照、Owner 匹配、图片权利为 `actual`、权利观察时间和渠道规则 URI 齐全 | `implemented` / 审计记录为 `automated_verified` | `backend-go/internal/domain/sourcing1688/image_processing.go` |
| 当前来源图片处理只做中心裁剪、缩放、白底、JPEG/PNG 编码，并保存输入/输出 SHA-256、操作、处理器版本和处理后字节 | `implemented` / 聚焦测试存在 | 这是确定性处理，**不是生成式 AI 作图** |
| 转草稿只接受匹配同一来源、快照和处理记录的 media；创建 product/SKU/media/cost/listing，并强制 listing 为 `draft` | `implemented`；当时未在本报告中重新验证 | `workflow_service.go:413-547`；不会调用平台适配器 |
| 草稿状态存在 `editing → pending_approval → approved_draft`，拒绝回到 `editing` | `implemented` / `lifecycle_service_test.go` 覆盖主要转换 | 提交审批时冻结 product/listing/SKU/media/cost 的内容 SHA-256；PostgreSQL 触发器阻止 pending 期间修改 |
| 真实发布使用另一套独立高风险审批和冻结请求；`submitted` 与 `reconcile_required` 都不等于真实上线 | `implemented` / 有聚焦测试 | 不应由 AI 作图流程直接触发 |
| 15 项验收报告已有“图片权利”“图片处理标准”“状态与审批”“独立发布安全”等检查 | `implemented` / 审计记为 `automated_verified` | 目前没有“生成式来源/模型/生成声明”专门检查 |
| 小Q 可只读解释 1688 来源、快照、草稿、成本和 media，并写 AI trace/evidence/event | `implemented` / `service_test.go` 有聚焦覆盖 | Capability 是 `sourcing_1688.controlled_draft.read`；不能生成、选择、批准或发布 |
| 独立 `imagegen` 模块有图片生成记录、画布、模板的 CRUD API 与测试 | `implemented` / 服务级 CRUD 测试为 `automated_verified` | 模型仅含 product、prompt、style、URL、status 等；没有 experiment/snapshot/rights/approval/audit/provider provenance 绑定 |
| `imagegen.Prism` 有多渠道缩放、白底和水印处理及单元测试 | `implemented` / `automated_verified` | 名称虽为 Prism，代码是确定性处理，不是生成式模型 |
| 可选 `prismadapter.Client` 能向外部 HTTP 服务发送 `image_url/platform/product_id` 并接收 output URL 与合规报告 | `implemented` | 当前在 product-analysis/listingtask 等旧路径使用；没有接入 sourcing1688 的证据链 |

### 3.2 目前仍未知或不能成立

| 声明 | 等级 | 原因 |
|---|---|---|
| 当前已配置并成功调用真实生成式图片模型 | `unknown` | 配置可关闭；本次未连接运行环境或服务商 |
| `imagegen` CRUD 创建记录后会实际调模型并自动完成任务 | `unknown`，从所读主路径看没有该闭环 | 创建接口只插入 `pending` 记录，状态更新也是通用写接口 |
| 外部 Prism 的输出能证明商品本体保持真实 | `unknown` | 返回契约只有 URL、风险分和简化合规报告 |
| 生成图的商用权、来源图许可、人物/品牌/知识产权安全 | `unknown` | 必须按具体服务条款、来源授权和具体输出核验 |
| 生成图符合当前已批准渠道的最新图片规则 | `unknown` | 现有静态默认平台尺寸不能替代带来源和观察时间的渠道规则证据 |
| AI 作图提升点击率、转化或最终净利润 | `unknown` | 尚无真实受控实验与订单/利润对账 |

## 4. 推荐的最小端到端接入

### 4.1 接入位置

接在 `sourcing1688` 的 **Owner 来源审查通过之后、Convert 创建内部草稿 media 之前**。原因：此时已经有 approved market、active experiment、opportunity pass、不可变来源快照和 Owner 权利判断；同时还没有形成待审批草稿，更没有外部发布。

不建议把旧 `imagegen` CRUD 直接连到发布，也不建议让通用 `productanalysis/trigger-prism` 接管经营状态。可复用其 provider client/interface 思路，但经营规则必须留在 `sourcing1688` 领域 Service 内。

### 4.2 建议数据模型

建议新增两张领域表，不把生成信息塞进现有确定性 `ImageProcessingRecord`：

`sourcing_image_generation_job`

- `id`, `sourcing_product_id`, `snapshot_id`, `experiment_id`, `demand_case_id`, `owner_id`
- `purpose`：首版固定为 `background_or_composition_candidate`
- `source_image_url`, `source_sha256`, `rights_evidence_uri`, `rights_observed_at`
- `provider`, `model`, `model_version`, `prompt`, `negative_prompt`, `parameters_json`
- `channel_rule_uri`, `channel_rule_observed_at`
- `requested_count`, `status`, `idempotency_key`, `estimated_cost`, `actual_cost`, `currency`
- `request_sha256`, `provider_job_id`, `started_at`, `completed_at`, `error_code`, `error_message`
- `created_by`, `created_at`

`sourcing_image_generation_output`

- `id`, `job_id`, `ordinal`, `output_sha256`, `mime_type`, `width`, `height`, `stored_bytes/object_uri`
- `automated_check_status`, `automated_check_json`
- `owner_decision`: `pending / selected / rejected`
- `owner_decision_note`, `decided_by`, `decided_at`
- `derived_processing_record_id` 或最终 `product_media_asset_id`
- `truth_status` 固定表达“生成发生”的事实，不表达图中营销声明为真

必须持久化真实输出内容或受控对象存储及哈希，不能只存可能过期、可被替换的外部 URL。

### 4.3 建议 API

全部位于现有 `/api/v1/sourcing-1688`，沿用 JWT 与 Owner scope：

- `POST /:id/image-generation-jobs`：校验闸门、冻结请求并创建 job；不接受任意 product。
- `GET /:id/image-generation-jobs` 与 `GET /:id/image-generation-jobs/:jobId`：读取任务、费用、失败和输出。
- `POST /:id/image-generation-jobs/:jobId/retry`：只允许可重试失败，复用原冻结输入并生成新的 attempt。
- `POST /:id/image-generation-jobs/:jobId/outputs/:outputId/decision`：Owner `select/reject`，必须备注。
- `POST /:id/image-generation-jobs/:jobId/outputs/:outputId/prepare-media`：把已选输出经过确定性尺寸/格式处理，生成可进入 Convert 的 media 记录。

不提供 `publish`、`auto-select` 或“生成完成即替换草稿”的 API。

### 4.4 状态机

```text
job: requested
  → running
  → succeeded | partially_succeeded | failed_retryable | failed_final | cancelled

output: pending_checks
  → blocked | pending_owner_review
  → selected | rejected
  → prepared_media
```

关键规则：

- 创建任务前重新检查市场、实验、opportunity gate、同 Owner、来源快照和 `actual` 权利证据。
- provider 成功但零有效输出应为失败，不能记 `succeeded`。
- 只有 `pending_owner_review` 可以选择；只有一个主图候选可被选为 main。
- 草稿已进入 `pending_approval` 后禁止新生成结果静默写入；应先拒绝/撤回并回到 `editing`，再明确替换、重新计算内容哈希并重新审批。
- `approved_draft` 的图发生任何变化，原批准不得继续有效。

## 5. 审批、审计和证据

### 5.1 审批边界

- **生成动作本身**：中风险、会产生外部调用和费用。首版建议 Owner 显式点击，页面展示用途、来源图、模型、预计费用、最大张数和失败后是否收费。若单次/累计费用超过 Owner 设置上限，应进入独立审批或直接阻断。
- **采用生成图**：必须 Owner 选择；选择不是合规批准。
- **草稿内容批准**：继续复用现有 `sourcing_1688_draft` 审批及 SHA-256 防篡改。
- **真实发布**：继续使用现有独立 `sourcing_1688_publish` 高风险审批；生成流程无权复用或跳过它。

### 5.2 审计记录至少回答

- 为什么生成、服务哪个 `experiment_id` 和哪项渠道决策；
- 使用了哪一个不可变快照、来源图哈希和权利证据；
- 谁请求，哪个 provider/model/version，实际 prompt/参数是什么；
- 外部请求/响应哈希、provider job ID、时间、费用和失败；
- 哪些自动检查通过/失败；
- Owner 选择了哪张、为何选择；
- 最终进入哪个 processing record、media、draft 和 approval。

建议 action：`image_generation.request / completed / failed / output.selected / output.rejected / media.prepared`。外部调用要有 correlation ID、幂等键、超时和敏感字段脱敏。

## 6. 失败处理

| 失败 | 系统行为 |
|---|---|
| provider 未配置 | `blocked/provider_unavailable`，不创建伪成功 URL |
| 超时或网络中断 | `failed_retryable`；先按 provider job ID 对账，不能盲重试造成重复费用 |
| 内容策略拒绝 | `failed_final`，保留供应商错误码和脱敏说明 |
| 输出下载失败/哈希不一致 | 输出 `blocked`；不得进入 media |
| 商品主体、颜色、数量或关键结构疑似变化 | 输出 `blocked` 或强制 Owner 复核；首版宁可拦截 |
| 含品牌、水印、中文文字、虚假认证或人物风险 | 输出 `blocked`；不得用模型自报“合规”直接通过 |
| 权利证据过期/撤回、渠道规则变化、市场或实验闸门撤销 | 停止生成和 prepare-media；已有候选保留审计但不得采用 |
| Owner 拒绝全部候选 | 回到来源图/人工图片处理，不自动扩大 prompt 或调用次数 |
| 草稿已 pending/approved 后想换图 | 使当前内容批准失效并回到 editing；重新提交完整草稿审批 |

## 7. 小Q Capability

第一版建议：

- 保留现有 `sourcing_1688.controlled_draft.read` 为 active，并扩充只读 OwnerView，展示 job 状态、成本、模型来源、选中输出、阻断和未知；不能返回敏感原始 provider payload。
- 新增 `sourcing_1688.image_generation.read` 可为 active，前提是 Owner scope、证据追踪、失败路径和回归测试齐全。
- `sourcing_1688.image_generation.request`、`select`、`prepare_media` 先标为 `deferred`。首版由 Owner 页面显式操作；不要让小Q自由文本直接执行外部付费生成或替换草稿。
- 小Q只能称输出为“AI 生成候选图（generation actual）”；对商品声明、权利、合规和效果必须分别保留 `unknown/quoted/actual`，不能因为模型或检查器通过而升级。

## 8. 绝对不该接的地方

- 不接候选市场比较阶段：没有选定市场与渠道时生成图不能改变市场决策，只会提前花钱。
- 不从普通 `imagegen` CRUD 直接写 `product_media_asset`、listing 或平台适配器。
- 不接 EventBus/Scheduler 自动批量生成；当前方向冻结更多自治升级，且会产生不可控费用。
- 不让小Q绕过 Capability、RBAC、Owner scope、审批或审计。
- 不把生成图片或 provider 的 `compliance_report` 当成图片权利、商品真实性、渠道合规或真实上线的外部证据。
- 不将生成完成状态映射成 `approved_draft`、`submitted`、`published` 或实验 gate pass。
- 不扩大到 SaaS、多租户、公共 API、模板市场、外部设计伙伴或“跨平台图片平台”。旧研究中的 Prism 产品化路线属于当前方向下的 `superseded` 材料。
- 不预设 Ozon、Shopee、Amazon 等平台规格；渠道规则必须来自已批准市场的当前证据。

## 9. 最小验收标准

工程验收只证明实现行为，不证明经营效果：

1. 不满足 market/experiment/opportunity/source-rights 任一闸门时，provider 调用次数为 0。
2. 同一幂等键不会重复收费或生成两个逻辑 job；超时后先对账。
3. 输入快照、来源图、模型、prompt、参数、输出字节和费用均有不可变哈希/记录。
4. provider 成功但输出缺失、无法下载或检查失败，不能进入 media。
5. 未经 Owner `selected` 的输出不能进入 Convert。
6. 已选输出必须经过现有确定性处理/渠道规则检查，并绑定同一 snapshot/source/experiment。
7. pending approval 期间不能替换图片；改图后原内容哈希审批失效。
8. Convert 后 listing 始终为 `draft`；生成链绝不调用平台 adapter。
9. 小Q只读输出保留 provider、模型、成本和 truth status，stub 回答明确为 `mock`。
10. 用一个真实、已获权利的 1688 商品人工走通后，才可把该次运行记为 `external_observed`；只有真实经营 A/B 数据和最终利润对账后，才能讨论是否有用。

## 10. 本次审查限制

- 本次未运行测试，因此本报告不提高任何自动验证证据等级。
- 未读取生产配置、未调用真实图片模型、未检查任何供应商最新服务条款或销售渠道最新图片规则。
- 未证明当前候选市场、商品、图片权利、费用、渠道账号或真实经营效果已经成立。

## 11. 主要代码依据

- `backend-go/internal/domain/sourcing1688/controlled_fetch.go`
- `backend-go/internal/domain/sourcing1688/image_processing.go`
- `backend-go/internal/domain/sourcing1688/workflow_service.go`
- `backend-go/internal/domain/sourcing1688/lifecycle_service.go`
- `backend-go/internal/domain/sourcing1688/draft_content.go`
- `backend-go/internal/domain/sourcing1688/acceptance_report.go`
- `backend-go/internal/domain/sourcing1688/publish_service.go`
- `backend-go/internal/domain/xiaoq/capability.go`
- `backend-go/internal/domain/xiaoq/service.go`
- `backend-go/internal/domain/imagegen/`
- `backend-go/internal/prismadapter/`
- `backend-go/migrations/000097_sourcing_draft_content_approval.up.sql`
