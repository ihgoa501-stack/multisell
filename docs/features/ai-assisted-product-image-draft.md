# 真实 SKU 的 AI 辅助图片草稿方案（已被替代）

> 日期：2026-07-12
> 状态：`superseded`；当前执行规格为 `docs/features/multi-provider-product-image-system.md`
> 适用范围：凌镜 Owner 自营跨境商品实验内部系统
> 当前版本目标：一个实验 × 一个真实 SKU × 一个渠道 × 一张主图和最多两张副图

## 1. 最终决策

凌镜中的 AI 电商作图不是通用绘画工具，也不是自动上架能力。第一版只建设：

> 在已通过机会闸门的 Owner 经营实验中，以有权使用的真实商品图为锚点，调用可替换的生成式图片 Provider 产生背景替换或局部清理候选；系统保存完整证据，由 Owner 对照真实商品逐张选择，选中图片经过现有草稿哈希审批后仍保持 `product_listing.status=draft`。

第一版不允许凭文字重画商品主图，不生成商品文字，不自动选择，不批量运行，不自动发布。当前不直接批准工程开发；必须先完成第 12 节 Phase 0，证明这一问题值得自动化。

## 2. 要解决的具体问题

首轮只验证一个假设：

> AI 是否能在限定费用和时间内，把一张有明确处理权的真实商品图转换成 Owner 可以批准的渠道草稿图片，同时不改变商品颜色、结构、数量、包装、文字、配件和尺寸暗示。

当前基线和目标值必须在真实 SKU 确定后冻结：

| 指标 | 基线 | 首轮目标 |
|---|---:|---:|
| 人工完成一张合格图片的时间 | `unknown`，先实测 | AI 流程更短且证据完整 |
| 单张最终采用图片总成本 | `unknown` | Owner 冻结上限内 |
| 本任务请求的 N 张中至少一张一次通过 | `unknown` | 首轮只记录，不预设成功 |
| 严重商品事实错误 | 不适用 | 0；发生一次即停止采用该模型 |

图片漂亮、API 返回成功、文件尺寸正确或平台提交无报错，都不构成经营成功。经营结果仍由同一实验的成交、售后、结算和最终正贡献利润裁决。

## 3. 范围

### 3.1 第一版包含

- 输入一张来自受控快照的真实商品图；
- 首轮主图只支持纯白、无语义背景的 `background_cleanup`；场景背景和 `background_replace` 保持 deferred；
- 明确蒙版或商品主体保护区域；
- 每个任务最多生成三张候选；
- 首轮只接一个经 Phase 0 验证的 Provider；保留最小接口边界，但不预建第二 Provider 路由；
- 请求冻结、幂等、超时对账、费用和失败记录；
- 原图、蒙版、提示词、模型、结果和审批的不可变哈希；
- 确定性检查、辅助模型风险筛查和 Owner 人工核验；
- 选中结果进入现有 `sourcing1688` 图片处理与内部草稿审批；
- 小Q支持保持 `deferred`，首轮由页面直接展示状态、费用、阻断和证据。

### 3.2 第一版不包含

- 文生商品主图、商品本体重绘或虚构 SKU；
- AI 直接生成文字、认证、品牌标识或功效对比；
- 人物、真人模特、儿童、医疗或安全效果图；
- 批量多 SKU、多渠道生成或 Provider 自动路由；
- 模型训练、LoRA、品牌套件、通用画布、视频或自动 A/B；
- EventBus/Scheduler 自动触发；
- 小Q自由文本执行付费生成、选择、批准或发布；
- 任何平台自动上传和发布。

## 4. 接入现有系统

接入位置：

```text
候选市场与 opportunity gate 已通过
→ active experiment
→ 1688 受控采集与不可变快照
→ Owner 来源审查
→ 图片权利闸门
→ AI 图片候选任务（本方案）
→ Owner 选择
→ 现有确定性裁剪/缩放/格式处理
→ product_media_asset
→ editing → pending_approval → approved_draft
→ listing 仍为 draft
→ 另行创建发布审批
```

实现放在 `backend-go/internal/domain/sourcing1688/` 的经营领域内。旧 `imagegen` CRUD 和 `prismadapter` 可以提供接口设计参考，但不能直接成为经营事实源，也不能绕过实验、权利、审批和审计。

## 5. 数据与证据模型

### 5.1 `sourcing_image_generation_job`

至少记录：

- `id / owner_id / experiment_id / demand_case_id`；
- `sourcing_product_id / snapshot_id`；
- 只能引用服务器受控的 `source_asset_id`；该资产必须以复合关系绑定 Owner、product、snapshot、SKU 和原始字节 SHA-256，禁止客户端重新上传字节冒充快照资产；
- `operation`，首版仅允许两种背景操作；
- 原图和蒙版的受控存储位置及 SHA-256；
- 不可变 `rights_grant_id`：绑定资产哈希、授权人及授权链、复制/修改/第三方 AI 处理/跨境传输/商业发布/平台再许可/商标/人物的独立授权、Provider/地域/用途/渠道、有效期、撤销状态和原文哈希；
- 渠道、站点、类目、图片用途和规则快照；
- 已验证的 `provider_profile_id`：绑定组织/项目、密钥指纹、端点、模型、区域、真实数据保留资格、条款版本、核验时间和过期时间；客户端声明不能证明 ZDR 或地域控制成立；
- 提示词、负面约束、参数及冻结请求 SHA-256；
- 请求数量、预计费用、费用上限、实际费用和币种；
- 幂等键、Provider 请求 ID、状态、时间和失败原因；
- 创建人、审批上下文和审计关联 ID。

### 5.2 `sourcing_image_generation_attempt`

一次 job 是 Owner 的一个冻结业务意图；每次向 Provider 提交是一个 attempt。至少记录：

- `id / job_id / attempt_no / parent_attempt_id`；
- 外部 mutation identity 固定绑定 job 与冻结请求哈希；同一外部意图的网络重试必须复用相同 Provider 幂等键。只有 Provider 明确证明前一意图未执行且 Owner 批准新的生成意图，才允许新键；另设唯一约束 `job_id + attempt_no`；
- Provider 请求 ID、提交状态、轮询/对账次数和最后对账时间；
- 每次请求的预计、实际费用、币种和计费状态；
- `submitting / submitted / provider_processing / reconcile_required / succeeded / failed_retryable / failed_final`；
- 超时、网络、内容拒绝、下载和哈希错误的结构化错误码。

job 幂等键确保一次 Owner 操作只建立一个逻辑任务；稳定 Provider 幂等键确保响应丢失时不会重复收费。重试可记录新 attempt，但同一外部意图必须转发原 Provider 幂等键。若没有 Provider request ID 且 Provider 不支持按幂等键查询，则失败关闭，禁止付费重试。

### 5.3 `sourcing_image_generation_output`

至少记录：

- `job_id / ordinal`；
- 原始模型输出的受控存储位置、SHA-256、格式、尺寸和色彩信息；
- `ai_generated / ai_edited / deterministic_transform` 标记；
- C2PA、SynthID、Content Credentials 等凭证的检测结果；
- 确定性检查结果和辅助模型风险结果；
- 六道闸门各自的 `passed / blocked / unknown`；
- Owner 决定、理由、审核人、时间和被批准的精确文件哈希；
- 派生的现有图片处理记录和 `product_media_asset` ID。

不得只保存可能过期或被替换的 Provider URL。生成发生可以记为事实，但画面中的商品声明不能因此升级为 `actual`。

对象使用内容寻址、不可覆盖的 key 和版本 ID。写入采用 `staging → 字节与哈希校验 → DB 提交 → finalize` 的可恢复协议；读取、Owner decision、prepare-media 和草稿审批均重新流式核验实际对象哈希。孤儿清理只能删除无数据库引用的 staging 对象，不能删除已引用版本。

## 6. Provider 契约

领域层只使用统一接口：

```go
type ImageProvider interface {
	Edit(ctx context.Context, req EditRequest) (*EditSubmission, error)
	Reconcile(ctx context.Context, providerRequestID string) (*EditResult, error)
}
```

Provider 适配器负责鉴权、请求转换、外部调用、响应下载和供应商错误映射；`sourcing1688` 负责所有经营闸门和状态裁决。

首版规则：

- 未配置 Provider 时返回 `blocked/provider_unavailable`，绝不返回 mock 成功；
- 外部写调用使用同一幂等键；
- 超时先通过 Provider 请求 ID 对账，再决定是否重试；
- 内容政策拒绝为明确失败，不自动改写提示词绕过；
- Provider 成功但零有效输出、下载失败或哈希异常均视为失败；
- 保存真实费用；不能取得实际费用时保持 `unknown`。

付费生成属于外部 mutation。执行前必须创建可消费、目标绑定的 Owner approval，冻结 owner、job、request SHA-256、Provider、模型、张数、最大费用、币种和有效期；执行事务必须同时 claim 持久化 tool execution 幂等记录并消费 approval。只有 Owner JWT 或普通写权限不能触发真实付费调用。

## 7. 状态机与 API

job 聚合状态：

```text
requested → running
running → succeeded | partially_succeeded | failed_retryable | failed_final
```

attempt 状态：

```text
submitting → submitted → provider_processing
submitting/submitted/provider_processing → reconcile_required
reconcile_required → provider_processing | succeeded | failed_retryable | failed_final
provider_processing → succeeded | failed_retryable | failed_final
```

首版不提供 cancel：外部请求一旦提交，必须完成对账和费用记录；Owner 可以拒绝所有输出，但不能把本地“取消”误写为外部任务已取消。`partially_succeeded` 仅表示请求的 N 张中至少一张已安全下载并进入 `pending_owner_review`，同时至少一张失败或被技术检查阻断；有效结果仍可继续审核。

候选状态：

```text
pending_checks → blocked | pending_owner_review
pending_owner_review → selected | rejected
selected → prepared_media
```

建议 API 全部位于 `/api/v1/sourcing-1688`：

- `POST /:id/image-generation-jobs`：重新校验闸门、冻结请求并发起生成；
- `GET /:id/image-generation-jobs`：查看任务列表；
- `GET /:id/image-generation-jobs/:jobId`：查看费用、失败和候选；
- `POST /:id/image-generation-jobs/:jobId/reconcile`：仅执行安全对账；
- `POST /:id/image-generation-jobs/:jobId/retry`：仅对可重试失败创建新 attempt；
- `POST /:id/image-generation-jobs/:jobId/outputs/:outputId/decision`：Owner 选择或拒绝；
- `POST /:id/image-generation-jobs/:jobId/outputs/:outputId/prepare-media`：把已选图片交给现有确定性处理链。

不提供 `publish`、`auto-select` 或“生成完成即替换草稿”的接口。

所有 mutation 仅允许 Owner，建议使用独立 `sourcing_image_generation.write` 权限；读取使用 `sourcing_image_generation.read`。服务端必须从 JWT 取得 Owner 身份，并重新核对路径中的 product、job、attempt 和 output 是否属于同一 Owner、experiment、SKU、snapshot 和 channel，不能相信客户端传入的关联 ID。Repository 查询以 `owner_id` 为首要条件，跨 Owner 对象统一返回 404；数据库复合约束保证关联对象 Owner 不漂移。

命令使用 `expected_version` 做乐观锁；重复 decision 和 prepare-media 必须幂等。至少定义标准错误：`gate_blocked`、`budget_exceeded`、`invalid_transition`、`reconcile_required`、`hash_mismatch`、`concurrent_update`、`provider_unavailable`。列表接口沿用项目统一分页。

## 8. 六道闸门

每张候选必须逐项裁决，任何关键 `unknown` 都不得用总分冲抵。G1–G5 全部通过后，候选才能进入 `pending_owner_review`；Owner 的 decision 才执行 G6。选中后进行确定性处理会产生派生文件和新哈希：系统必须保存“生成候选哈希 → 处理后 media 哈希”的派生链，最终草稿审批绑定处理后 media 哈希。任何后处理参数或文件变化都使旧 G6 和草稿批准失效，返回 `pending_owner_review` 或 `editing` 重新审核。

| 闸门 | 通过条件 | 典型阻断 |
|---|---|---|
| G1 商品真实性 | 与同一 SKU 的实物、可靠规格和原图对账 | 颜色、结构、数量、包装、文字、配件或比例改变 |
| G2 图片与人物权利 | submit 前 rights grant 已覆盖复制、修改、特定 Provider/区域的第三方 AI 处理和跨境传输；输出审核再检查商业发布、平台再许可、商标、人物及新出现元素 | 只有卖货授权；授权与资产哈希不匹配；已过期/撤销；出现人物 |
| G3 渠道规则 | 有“法域 × 渠道 × 站点 × 类目 × 用途/广告位 × 分发路径 × 时间”的官方原始规则快照 | 套用其他平台规则；规则过期、被替代或用途发生变化 |
| G4 文字与声明 | 所有明示和暗示声明有字段级证据 | AI 生成文字、认证、功效、尺寸或配件 |
| G5 技术与视觉质量 | 文件、清晰度、边缘、透明区、颜色、透视和差异检查通过 | 伪影、乱码、主体漂移、错误阴影或尺度感 |
| G6 Owner 审批 | Owner 批准具体 SKU、渠道、用途和精确文件哈希 | 只批准提示词或缩略图；批准后文件变化 |

规则快照保存官方 URL、redirect chain、locale、原始字节及哈希、抓取方式、`effective_from/to`、`superseded_at` 和最大有效期。prepare-media、草稿批准和发布执行都重新核对；渠道、类目、主/副图、广告位或规则变化会使 G3/G6 失效。平台要求的 AI 披露或来源元数据必须作为 G3 执行；元数据丢失即阻断。AI 标签不能补救误导性图片。

## 9. 检查方法

### 9.1 确定性检查

- MIME、尺寸、像素、文件大小、色彩空间和 SHA-256；
- 原图与结果的主体保护区像素差异；
- 蒙版外变化范围；
- OCR 检测意外文字；
- 品牌、认证、条码和包装文字区域是否变化；
- 元数据与内容凭证是否存在及是否在后处理中保留；
- 输出是否可解码，是否与任务、Owner、快照和 SKU 唯一关联。

### 9.2 辅助模型检查

辅助视觉模型可以标记颜色、数量、配件、文字、品牌、人物和结构疑点，但只能提供 `support/counter/conflict` 信号，不能通过任何闸门。

所有上传前检查必须在本地隔离环境执行：清除 EXIF/GPS 等非必要元数据，扫描二维码、条码、OCR、人物、面单、透明层和秘密信息；Provider 只接收清洗后新哈希对应的图片。首版任何人物、人脸、未知标识或语义场景都硬阻断，Owner 不能覆盖。

### 9.3 Owner 人工检查

页面并排显示原图、候选图、差异图、真实规格和风险提示。Owner 必须逐项确认：

- 商品颜色、材质和纹理；
- 数量、配件和包装；
- 结构、接口、按钮和尺寸比例；
- 商品及包装文字、品牌和认证；
- 背景、阴影、接触面和使用暗示；
- 是否存在人物或第三方元素。

## 10. Owner 页面与小Q

在现有 `/sourcing1688` 页面增加一个“AI 图片候选”区域，不新增独立图片平台：

1. 展示真实来源图、权利和渠道规则；
2. 选择首版允许的背景操作；
3. 展示模型、最多三张、预计费用和预算上限；
4. Owner 显式发起；
5. 并排显示原图、候选和差异；
6. 展示六道闸门及阻断原因；
7. Owner 选择一张或全部拒绝；
8. 选中图进入现有草稿编辑和内容哈希审批。

小Q第一版全部保持 `deferred`。只有记录到至少三次 Owner 因静态页面无法理解而作错或延迟决定，并且静态文案不能解决时，才重新评估只读 Capability；生成、选择和 prepare-media 能力继续保持 `deferred`。

## 11. 安全、隐私与成本

禁止上传包含以下内容的图片：

- 供应商后台、合同、报价、联系方式和内部成本；
- 未授权人物、私人住宅或个人信息；
- 未公开新品或受保密约束的设计；
- 无法确认使用权的品牌、认证或第三方素材。

请求前显示预计费用和最多生成数；记录实际费用、Owner 审核时间和返工次数。首轮建议由 Owner 冻结单任务费用上限，超过即阻断。系统不得自动连续调提示词，也不得因三张全部失败而扩大生成次数。

费用必须同时进入项目唯一预算账本：总现金上限 3,000 CNY，不可回收损失硬停止线 1,200 CNY。账本包含样品、Provider、存储、工具、失败输出和其他已承诺不可退费用；Owner 时间单独记录并设置小时上限。每次付费提交事务内原子预占最坏费用，不能通过拆分 job 绕过项目上限。

若 Provider 无法给出可验证的费用上限，真实调用失败关闭；超时、部分成功和迟到费用都归属具体 attempt，job 总费用为所有 attempt 的实际费用之和。币种和汇率来源、观察时间分别保存，不用估算值覆盖真实账单。

原图和输出建议放在受控对象存储中，数据库保存受保护引用和哈希。需要制定原图、拒绝图、Provider 临时文件和备份的保留/删除期限；删除文件时保留必要审计记录，不保留敏感原始 Provider payload。

阶段 B 前冻结保留策略：被采用的原图、输出和派生链随实验审计期保留；未采用输出默认保留 30 天后删除受控文件；Provider 临时文件在确认本地哈希保存并完成对账后尽快删除（若 Provider 支持）；审计表永久保留任务、哈希、时间、费用、状态和删除证明，但不保留敏感原始响应。具体期限仍须与备份政策和适用法律复核。

## 12. 开发阶段

### Phase 0：先证明值得自动化（当前唯一获准阶段）

工程预算为 0；只使用现有工具和一次手工记录，最长 7 天：

1. 取得一个真实 SKU、当前批次实物多角度照片、尺寸参照、变体和包装/配件清单；仅有供应商 `quoted` 规格不足以通过主图真实性闸门；
2. 取得一张权利范围明确覆盖特定 Provider 处理的图片；
3. 确定一个渠道、类目、图片用途和当日官方规则；
4. 比较四种当前解决方法：原图直用、现有确定性白底、一次性人工/外包修图、一个 Provider 的手工沙盒编辑；
5. 从输入准备到 hash-bound 草稿记录总现金、Owner 总时间、返工、事实错误和权利风险；
6. 在试验前冻结通过阈值，例如 AI 相比最佳非 AI 基线至少节省 30% Owner 时间、现金不更高且严重商品事实错误为 0。具体数字由 Owner 在试验前确认，结果出现后不得改阈值。

任一前提在 7 天内仍为 `unknown`，或最佳成熟方案更便宜/更快/风险更低，归档模块并停止开发。一次结果只裁决该 SKU 当前任务，不证明模型保真率或可重复性。

### 阶段 A：领域闭环

- 数据库迁移、唯一约束和不可变字段；
- 领域模型、状态机、Owner scope 和所有前置闸门；
- 测试 Provider；
- API、费用、失败、幂等和对账；
- 选中输出接入现有图片处理和草稿审批。

进入条件：Phase 0 达到预冻结阈值并由 Owner 明确批准开发。完成标准：无需真实密钥即可证明所有业务门禁和失败路径；不满足条件时外部调用次数为零。

### 阶段 B：真实 Provider

- 按开发当日官方文档核验 OpenAI Images 契约、价格、组织资格和数据保留；
- 实现 OpenAI 适配器及契约测试；
- 禁用 mock 回退；
- 在隔离环境使用非敏感、明确授权的测试图片验证一次调用和费用记录。

完成标准：只证明 Provider 调用和证据保存为 `manually_verified`，不证明商品可用。

### 阶段 C：Owner 页面与草稿闭环

- 生成发起、任务状态、候选对比、差异图和逐项审核；
- 草稿 pending/approved 后换图使旧审批失效；
- 小Q只读解释；
- 前端测试和浏览器 E2E。

完成标准：一个真实 SKU 的一张图片进入内部草稿，仍保持 `draft`。

### 阶段 D：真实渠道观察

- 取得目标渠道当日规则快照；
- Owner 独立批准并执行发布；
- 观察精确哈希图片是否被渠道接受并真实展示；
- 记录投诉、退货、经营指标和最终利润，但不把一次结果外推为可重复性。

完成标准：渠道展示只能记为 `external_observed`；经营有效必须另行通过实验事实链裁决。

### 证据等级分层

- **工程完成**：状态机、闸门和测试通过，为 `automated_verified`；
- **真实内部可用**：一个获权真实 SKU 使用真实 Provider，通过 G1–G6 并进入 hash-bound draft，为注明环境的 `manually_verified`；
- **渠道接受**：目标渠道真实展示精确派生图片，为 `external_observed`；
- **制图流程有效**：相对预冻结的最佳非 AI 基线，Owner 总时间、现金、返工和严重事实错误达到阈值；
- **经营结果记录**：成交、售后和利润只作为同一实验的外部结果，若价格、广告、流量、文案或库存同时变化，AI 图片的增量影响保持 `unknown`，不得把关联写成因果。

## 13. 工程验收

以下全部通过才算工程闭环完成：

1. market、experiment、opportunity、snapshot、rights 或 channel rule 任一闸门失败时，Provider 调用次数为零；
2. 同一幂等键不重复收费或创建两个逻辑任务；
3. 超时后先对账，不盲目重试；
4. Provider 成功但无输出、下载失败或哈希异常不能进入候选审核；
5. 未经 Owner 选择的图片不能进入 media；
6. 被选图片必须绑定同一 Owner、experiment、SKU、snapshot 和渠道；
7. pending approval 期间不能静默换图；换图后旧批准失效；
8. Convert 后 listing 始终为 `draft`；生成链绝不调用平台适配器；
9. 未配置密钥、测试环境和 stub 路径均明确标记，不能伪装成真实成功；
10. 后端领域测试、迁移约束、Provider 契约测试、前端测试和核心 E2E 通过。

## 14. 真实验收卡

开始真实调用前必须填完：

```text
experiment_id：
已批准 demand_case 与 market readiness：
active experiment 与 opportunity gate：
受控 snapshot_id、来源和 SHA-256：
真实 SKU 与样品/可靠规格：
源图 SHA-256：
复制权：passed / blocked / unknown
修改权：passed / blocked / unknown
上传第三方 AI：passed / blocked / unknown
商业发布与平台再许可：passed / blocked / unknown
商标使用权：passed / blocked / unknown
人物/肖像权：passed / blocked / unknown
个人信息与隐私：passed / blocked / unknown
目标地区、渠道、站点和类目：
图片用途：主图 / 副图
渠道规则快照及观察时间：
Provider、模型、区域和数据保留设置：
最多生成：3
费用上限：
Owner 审核时间上限：
失败后回退：现有确定性图片处理
```

关键项存在 `unknown` 时，只能完成工程测试，不能调用真实 Provider 或生成可发布候选。

## 15. 停止条件

发生任一情况立即停止当前 job，不自动扩大范围：

- 商品身份、颜色、结构、数量、包装、文字或配件出现一次严重改变：阻断该 output，并暂停该 SKU × 模型组合；只有 Owner 查看根因和新证据后才能重新启用；
- 图片权利、渠道规则或关键商品声明为 `unknown`；
- 出现未授权人物、商标、认证或第三方设计；
- 超过任务费用或 Owner 审核时间上限；
- 三张候选全部不合格；
- Provider 超时后无法完成对账；
- 批准后文件哈希变化；
- 外部发布缺少独立 Owner 审批；
- 第一个真实 SKU 尚未闭环，却准备增加第二 Provider、批量化或更多用途。

项目级熔断：首个 Phase 0 三张候选全部拒绝、一次严重商品事实错误、连续任务无净节省、项目不可回收承诺达到 1,200 CNY 或总现金承诺达到 3,000 CNY时，阻断全部新 attempt、Provider 扩张和相关开发。只有 Owner 基于新的外部证据显式解冻；不能因更换 SKU、模型或拆分 job 绕过。

## 16. 当前事实与未知

### 已有事实

- `implemented`：现有 `sourcing1688` 已有不可变来源快照、确定性图片处理、权利证据、media、草稿内容哈希、Owner 审批和独立发布保护；
- `implemented`：当前裁剪、缩放和白底处理不是生成式 AI；
- `quoted`：成熟 Provider 提供生成和编辑 API，但官方同时承认商品身份保持、文字、属性绑定和编辑精度存在限制；
- `quoted`：部分渠道对商品真实性、AI 来源元数据或 AI 披露有具体要求，不能全球一刀切。

### 当前未知

- 第一个真实 SKU、真实样品或可靠规格；
- 源图是否允许复制、修改、上传第三方模型和商业发布；
- 最终渠道及当日类目规则；
- 当前账号价格、配额、数据保留和合同保护；
- 哪个模型对该 SKU 最保真；
- 真实返工率、Owner 审核时间、渠道接受率和经营效果。

这些未知不阻止阶段 A 的工程建设，但会阻止真实 Provider 调用和“真实可用”结论。

## 17. 依据

本方案综合以下报告；外部事实、适用范围和官方链接以原报告为准：

- `docs/research/ai-image-definition-2026-07-12.md`
- `docs/research/ai-image-system-integration-2026-07-12.md`
- `docs/research/ai-image-provider-policy-2026-07-12.md`
- `docs/research/ai-commerce-image-platform-risks-2026-07-12.md`
- `docs/research/ai-commerce-image-rights-risks-2026-07-12.md`
- `docs/research/ai-commerce-image-fidelity-workflow-2026-07-12.md`

## 18. Owner 下一项决策

当前只推荐批准 Phase 0，不批准阶段 A。Owner 先提供或批准寻找：一个真实 SKU 与当前批次样品、一张具有明确特定 Provider 处理权的图片、目标渠道、人工/确定性基线和手工试验费用上限。

只有 Phase 0 达到预冻结阈值，才重新提交阶段 A 决策。届时第一个开发切片依次实现：状态转换表、权限矩阵、服务器前置条件 schema、job/attempt/output 三表迁移及契约测试；完成后再编写单 Provider 适配器。
