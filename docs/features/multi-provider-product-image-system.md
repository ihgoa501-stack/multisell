# 多方案商品图片系统开发规格

> 日期：2026-07-12
> 状态：`implemented_initial_deterministic / external_providers_gated`
> Owner 决策：完整建设多方案商品图片纵向单元
> 证据限制：本规格不代表 Provider 已配置、图片权利已取得、真实渠道已接受或经营效果已成立

## 1. 目标与边界

建设凌镜统一的商品图片制作、证据、审核和草稿接入系统。系统同时支持确定性处理、专业商品图片 API、通用生成模型和外部工具导入；所有路径共享同一资产、权利、预算、审核与审计规则。

```text
Owner 私人图片资产/已审货源图片
→ 创建图片任务并选择经营用途
→ 选择处理方案
→ 执行并保存不可变结果
→ 商品真实性、权利、渠道和视觉检查
→ Owner 选择精确文件
→ Listing 草稿内容审批
→ 独立受控发布
```

它不是通用绘画 SaaS，不服务外部用户，不训练自有基础模型，不自动发布，不以图片生成成功替代真实商品与渠道事实。

## 2. 完整能力范围

### 2.1 处理路径

| 路径 | 首批实现 | 主要用途 | 是否外部付费 |
|---|---|---|---:|
| `deterministic` | LingMirror | 裁剪、缩放、白底、格式、压缩、固定模板 | 否 |
| `commerce_ai` | Photoroom | 去背景、阴影、补光、商品场景和批量商品处理 | 是 |
| `precision_edit` | Adobe Firefly/Precise Composite | 局部修补、精确合成、扩图 | 是 |
| `creative_ai` | OpenAI Images | 场景副图和广告创意候选 | 是 |
| `manual_import` | Pebblely、Canva、Photoshop、Pixelcut 等 | 外部编辑结果回传 | 视外部工具而定 |
| `channel_native_import` | Shopify、Google、Amazon 等 | 渠道内置工具结果回传，仅限对应渠道 | 视渠道而定 |

首批接口完整支持上述六条路径。外部 API 适配器只有在配置和官方契约核验完成后才标为 `available`；未配置时明确 `unavailable`，不得以 mock 结果冒充。

### 2.2 图片用途

- 商品事实主图：真实商品主体，只允许确定性处理或明确保持主体的专业商品处理；
- 商品副图：尺寸、结构、包装、配件和使用说明；文字采用结构化事实与确定性排版；
- 场景副图：可生成背景，但不得改变商品或产生无证据暗示；
- 广告创意：与 Listing 图片分开审批，绑定广告位和法域；
- 内部概念图：永久标记 `concept_only`，不能进入可发布草稿。

### 2.3 明确不做

- 外部 SaaS、多租户、订阅、公共图片 API；
- 自训练基础模型、LoRA 市场或模型托管平台；
- 未经 Owner 选择的自动采用；
- 未经渠道决定的销售平台连接器扩张；
- 图片任务直接调用发布适配器；
- 小Q绕过领域服务、审批或审计执行付费处理。

## 3. 领域边界与现有能力复用

新领域建议为 `internal/domain/productimage/`，负责通用商品图片资产与处理。`sourcing1688` 只负责把已审货源图注册为来源资产，并在 Convert 前引用已批准图片；未来其他货源或商品模块可以复用同一领域。

复用：

- 受控来源快照和 SHA-256；
- 当前确定性裁剪、缩放、白底算法；
- Owner scope、RBAC、Approval、OperationLog、ToolBridge 幂等与审计；
- `product_media_asset`、草稿内容哈希和独立发布审批；
- 统一响应 envelope、分页和结构化错误。

改造：

- 当前 `sourcing1688.ImageProcessingRecord` 迁移/适配为统一处理结果；
- 旧 `imagegen` CRUD 只保留历史读取或迁移，不能继续成为生产写入口；
- `prismadapter` 仅作历史实现参考，不直接进入新领域。
- 新 Image Service 是 Prism 的正式替代，不长期并存；旧确定性算法迁入 deterministic processor，旧外部客户端、触发路由和发布前 Prism gate 按 `image-service-mcp-contract.md` 的迁移顺序冻结并移除。

## 4. 核心领域模型

### 4.1 `product_image_asset`

不可变图片资产：`id`、`owner_id`、`product_id/sku_id`、`source_kind`、受控对象版本、SHA-256、MIME、尺寸、色彩空间、来源 URI、来源时间、真实性状态、父资产 ID、创建时间。对象采用内容寻址且不可覆盖。

### 4.2 `product_image_rights_grant`

绑定精确资产哈希，分别记录复制、修改、第三方 AI 处理、跨境传输、商业发布、平台再许可、商标、人物/肖像授权；包含授权人、权利链、用途、法域、渠道、Provider/地域、有效期、撤销状态、原始证据哈希和 Owner 核验。

### 4.3 `product_image_task`

Owner 的冻结经营意图：商品/SKU、图片用途、目标渠道规则、处理路径、允许/禁止变化、请求数量、预算、输入资产、请求 manifest SHA-256、状态、版本和审计上下文。

### 4.4 `product_image_attempt`

一次处理执行：processor、provider/model/version、稳定 Provider 幂等键、Provider 请求 ID、参数、prompt/mask、预计和实际费用、状态、错误、提交/对账时间。网络重试复用同一外部幂等键；只有明确的新生成意图才创建新键。

### 4.5 `product_image_review`

逐资产保存商品真实性、权利、渠道规则、声明、技术视觉和 Owner 审核；每项独立为 `passed/blocked/unknown`，保存证据和审核时间。Owner approval manifest 绑定资产、SKU、用途、渠道、规则、检查结果和最终派生文件哈希。

### 4.6 `product_image_cost_entry`

保存 Provider、存储、外部工具、人工/外包等现金成本及币种、汇率来源、观察时间和账单状态；连接项目预算账本，不允许拆任务绕过总预算。

## 5. Processor 契约

```go
type Processor interface {
	Code() string
	Capabilities(ctx context.Context) (Capabilities, error)
	Estimate(ctx context.Context, req EstimateRequest) (EstimateResult, error)
	Submit(ctx context.Context, req SubmitRequest) (Submission, error)
	Reconcile(ctx context.Context, ref ProviderRef) (ReconcileResult, error)
}
```

约束：

- 输入/输出使用领域 DTO，不泄漏 Provider JSON；
- 第三方响应一律按不可信输入校验；
- Capability 明确 operation、输入类型、最大尺寸、是否异步、是否支持查询/删除、费用口径和数据地域；
- `ReconcileResult` 使用判别联合：`processing/succeeded/failed/not_found/unknown`；
- 同一 Provider 只保留一个现行适配器版本，扩展字段保持向后兼容；
- Provider 特有原始响应加密保存或脱敏，不进入普通 Owner API；
- 不支持可靠对账或幂等的 Provider，生产付费调用失败关闭。

## 6. 状态机

任务：

```text
editing → pending_approval → approved
approved → running
running → review_ready | partially_ready | reconcile_required | failed
review_ready/partially_ready → owner_selected | rejected
owner_selected → prepared_media
```

attempt：

```text
created → submitting → submitted → processing
submitting/submitted/processing → reconcile_required
reconcile_required → processing | succeeded | failed | unknown
processing → succeeded | failed
```

终态不可覆盖；状态更新使用 PostgreSQL 约束和 `status + version` CAS。job/attempt/output ordinal、Provider request ID、Owner 选中主资产均有数据库唯一约束。

## 7. API

根路径：`/api/v1/product-images`。所有资源按 JWT Owner scope 查询；跨 Owner ID 返回 404。

- `GET /processors`：返回已配置能力，不泄漏凭据；
- `POST /assets`：注册受控来源或上传外部结果；
- `GET /assets`、`GET /assets/:id`、`GET /assets/:id/content`；
- `POST /tasks`、`GET /tasks`、`GET /tasks/:id`；
- `POST /tasks/:id/approval-requests`；
- `POST /tasks/:id/executions`：消费目标绑定批准并执行；
- `POST /tasks/:id/reconciliations`；
- `POST /tasks/:id/retries`；
- `POST /tasks/:id/reviews`：提交 G1–G5；
- `POST /tasks/:id/decisions`：Owner 选择/拒绝；
- `POST /tasks/:id/media-preparations`：创建最终派生 media；
- `GET /tasks/:id/costs`。

列表全部分页；写命令携带 `expectedVersion` 和 `idempotencyKey`。错误使用标准 envelope，至少包括 `VALIDATION_ERROR`、`GATE_BLOCKED`、`PROVIDER_UNAVAILABLE`、`BUDGET_EXCEEDED`、`APPROVAL_REQUIRED`、`RECONCILE_REQUIRED`、`HASH_MISMATCH`、`VERSION_CONFLICT`。

## 8. 统一闸门

1. **来源真实性**：服务端受控 source asset 与商品/SKU/快照一致，客户端不能用相同 URL 替换字节；
2. **输入权利**：调用前，rights grant 覆盖精确资产、操作、Provider/地域和用途；
3. **隐私预检**：本地清除 EXIF/GPS，扫描人脸、二维码、面单、秘密和透明层；
4. **预算与批准**：付费调用具有最坏费用预占和目标绑定、可消费的 Owner approval；
5. **商品真实性**：输出与实物/可靠规格、原图、数量、结构、颜色、包装、配件和文字对账；
6. **声明与场景**：视觉暗示逐项关联商品事实，健康、安全、儿童、认证等高风险场景默认阻断；
7. **渠道规则**：规则快照绑定法域、渠道、站点、类目、用途/广告位、分发路径和有效时间；
8. **技术与来源凭证**：文件、差异、OCR、元数据、C2PA/IPTC 等按渠道要求检查；
9. **Owner 选择**：Owner 审核规范化全尺寸 rendition，批准精确 manifest；
10. **草稿与发布隔离**：任何图片变化使草稿批准失效；图片流程不能发布。

## 9. Owner 体验

在商品/货源详情增加统一“图片工作室”，而不是为每个 Provider 建页面：

1. 选择真实商品资产和用途；
2. 系统展示可用处理方案、能力、预计费用、数据地域与风险；
3. Owner 选择一个或多个方案并发起审批；
4. 统一查看进度、费用和失败；
5. 并排查看原图、候选、差异图、全尺寸细节和六类事实检查；
6. 选择、拒绝或要求重新处理；
7. 生成最终渠道派生图并进入 Listing 草稿审批。

人工导入必须记录工具、操作、费用、来源和已知参数；未知模型/版本保持 `unknown`。渠道内置结果默认只允许用于原渠道，跨渠道复用需重新审核。

## 10. 多方案选择规则

Owner 保持最终选择权，第一版不做 AI 自动路由。系统仅提供确定性推荐：

| 用途 | 默认方案 | 可选方案 |
|---|---|---|
| 主图白底/尺寸 | LingMirror deterministic | Photoroom |
| 去背景、阴影、补光 | Photoroom | deterministic |
| 精确局部修补/合成 | Adobe | manual import |
| 场景副图 | Photoroom | OpenAI、Pebblely 导入 |
| 广告创意 | OpenAI | Adobe、渠道工具导入 |
| 文字信息图 | deterministic template | Canva 导入 |

推荐不等于通过；输出仍必须经过统一闸门。

## 11. 失败、安全与运行保障

- Provider 接收请求但响应丢失：以稳定幂等键和 Provider 查询对账；无法查询则 `unknown`，禁止付费重试；
- 对象存储：`staging → hash verify → DB commit → finalize`，读取和审批重新校验哈希；
- 预算并发：事务内预占最坏费用，迟到费用归属 attempt；
- 429/5xx：按 Provider 契约有限重试，生产 mutation 复用原幂等键；
- 零输出、下载失败、MIME/尺寸/哈希异常：失败，不创建可审核资产；
- Provider 内容拒绝：明确失败，不自动改 prompt 绕过；
- 凭据来自批准的 secret storage，日志禁止记录 key 和敏感 payload；
- 保留/删除覆盖对象版本、缩略图、CDN、备份和 Provider 临时文件；无法证明删除时状态为 `unknown`；
- 指标：任务成功率、对账队列、P95 延迟、费用、输出阻断率、Owner 采用率、对象不一致和重复调用数；
- 告警：预算越线、长期 reconcile、Provider 契约失败、对象哈希错误和审批绕过尝试。

## 12. 小Q

模块新增能力必须声明 `xiao_q_support`：

- `product_image.read`：领域和 Owner 页面稳定、权限与回归测试齐全后 `active`；
- `product_image.recommend_processor`：只读建议，可在有真实能力数据后 `active`；
- `product_image.request/execute/select/prepare_media`：首个发布版本保持 `deferred`。

小Q可以解释能力、证据、费用和阻断，不能把模型成功升级为商品真实、权利成立或渠道合规。

## 13. 开发切片

每个切片都完成领域、API、Owner 体验、权限、审批、审计、错误、测试、运行验证和文档；不是长期 90% 实现。

### Lake 1：统一资产与确定性处理

建立 asset/rights/task/review/cost，迁移现有确定性处理，完成导入、审核、草稿接入和对象一致性。

当前工程实现（2026-07-12，`automated_verified`）：

- `product_image_rights_grants` 将 Owner 与精确资产 SHA-256 绑定，逐项记录复制、修改、第三方 AI、跨境、商业发布、平台再许可、商标和肖像许可，以及用途、法域、渠道、Provider、地域、有效期、撤销、证据 SHA-256 和 Owner 核验；
- 五类审核分别保存商品真实性、权利、渠道规则、声明/场景和技术视觉的 `passed / blocked / unknown`，普通 API 输入不能把证据声明为 `actual`；
- `product_image_cost_entries` 保存预计/实际金额、严格十进制字符串、币种、汇率及来源、观察时间和账单状态；付费执行批准同时落一条预算上限预计成本；
- 权利、审核和成本写命令使用 Owner scope、`expected_version` 与幂等键，读取接口分页；跨 Owner 对象按不存在处理；
- Listing image set 在创建和冻结时都会重新核验精确输出字节的有效发布权利和五类全通过。确定性处理不因成本未知而阻断；付费候选还必须存在非 `unknown` 成本记录。

Owner 尚未为真实商品录入权利证据或完成五类审核，因此这些工程门禁不能被解释为真实图片已获授权、渠道已接受或可生产发布。

### Lake 2：外部工具导入

完整支持 Pebblely、Canva、Photoshop及渠道内置结果的统一回传、来源标记、费用和渠道限制。

### Lake 3：专业商品 AI

实现 Photoroom Provider、目标绑定付费审批、预算、幂等、对账、失败和 sandbox/生产契约验证。

### Lake 4：精确编辑与创意

实现 Adobe 与 OpenAI Provider；分别限制为精确编辑和场景/广告用途，不允许绕过主图规则。

### Lake 5：完整 Owner 工作室与小Q只读

统一对比、批量任务、全尺寸审核、成本视图、运行监控和满足契约后的只读 Capability。

## 14. 验收矩阵

- 单元测试：状态、门禁、权限、费用、manifest、Provider 响应解析；
- PostgreSQL 集成：唯一约束、CAS、并发预算、终态不可变、跨 Owner；
- Provider 合同：同步/异步、202、429、5xx、零输出、断流、迟到结果、同 key 不同 payload；
- 对象故障注入：put/DB/finalize 任一点崩溃后可恢复，篡改必阻断；
- 前端：配置状态、费用确认、差异查看、全尺寸审核和错误恢复；
- E2E：确定性、人工导入和每个已配置 Provider 至少一条完整草稿路径；
- 安全：IDOR、过期/错目标 approval、敏感元数据、未授权资产、日志泄密；
- 发布隔离：所有图片路径均无法直接调用平台 adapter。

工程通过只能记 `automated_verified`；真实 Provider 调用为注明环境的 `manually_verified`；渠道真实展示为 `external_observed`；图片对经营的因果影响在没有受控比较时保持 `unknown`。

## 15. 当前未知与实施前置

- Provider 账号、当前价格、配额、地区、数据保留和正式合同；
- 第一个真实商品和图片权利；
- 目标渠道与类目规则；
- Adobe Precise Composite 的实际可用契约；
- 不同方案对真实 SKU 的保真率和 Owner 人工成本。

这些未知阻止对应 Provider 的生产可用声明，但不阻止按上述合同完成领域、适配器、失败关闭和自动测试。

## 16. 文档关系

- 本文是多方案图片系统的当前开发规格；
- `ai-assisted-product-image-draft.md` 自本文生效后为 `superseded` 历史方案；
- 外部工具、API、工作流和风险报告保留为事实来源；
- 代码或外部状态变化后生成新的带日期工程验证记录，不静默提高证据等级。

## 17. 多 Agent 审查后的强制前置条件

本节优先于前文中与之冲突的简写。以下 P0 在对应 Lake 开工前必须写入详细设计与测试，不得留给实现者自行猜测。

### 17.1 正确的图片主体身份

图片通常在 `sourcing1688.Convert` 创建正式 product/SKU 前处理，因此 asset 不能只依赖 `product_id/sku_id`。统一身份改为：

```text
owner_id + subject_type + subject_id + source_snapshot_id
```

Convert 后新增不可变 binding 关联正式 product/SKU，禁止回写替换原来源主体。数据库和服务端必须保证整条关联链属于同一 Owner。

### 17.2 Listing 图片集合，而非单张图片

新增 `product_image_set` 和 `product_image_set_item`：绑定 listing、channel、locale、version、角色、顺序、asset hash 和渠道 rendition。主图、不同副图、尺寸图、包装图及广告封面分别有角色约束；Owner 与草稿审批绑定整个 set manifest。任何图片替换、调序、语言、用途、渠道或最终字节变化都产生新版本并使旧批准失效。

### 17.3 状态和证据拆分

新增 `product_image_output`，并把状态拆成：

```text
intent: editing → pending_approval → approved | rejected | expired | cancelled
execution: queued → running → review_ready | partially_ready | reconcile_required | failed_before_submit | failed
selection: pending → selected | rejected | stale
image_set: editing → pending_approval → approved_draft | rejected | stale
```

pending 后冻结 manifest；任何输入、用途、费用、Provider、规则或参数变化都 clone 新版本。自动检测通过只证明检测行为为 `automated_verified`，不能把权利、真实性或“没有发现风险”升级为 `actual`。

### 17.4 Provider Runner 与异步执行

当前 ToolBridge 没有可直接复用的通用图片 Execute 合同。第一版使用 productimage 自有 `ProcessorRunner + durable attempt/outbox`；worker 以 lease、`next_attempt_at` 和 `FOR UPDATE SKIP LOCKED` 领取任务，崩溃可接管。若未来扩展 ToolBridge，必须先形成正式的 Execute、payload hash、durable result 和异步 reconcile 合同。

`POST /executions` 只在本地事务中创建或重放同一个 attempt；同一 key、同一 payload 返回同一资源，同一 key、不同 payload 返回 409。外部 worker 的提交生命周期独立于 HTTP 2xx，Provider reference 和结果持久化前不得把外部执行标为完成。

### 17.5 Photoroom 与 Provider 安全等级

每个 Provider 必须标记：`contract_defined / sandbox_only / manual_only / production_safe / unavailable`。公开资料未确认 Photoroom 同步二进制 API 具有可靠幂等、任务查询或 webhook；因此 Photoroom 当前只能 `sandbox_only/manual_only`。只有正式采购合同或官方能力证明能够防止响应断流后的重复收费并支持可靠对账，才能升级为 `production_safe`。

生产付费执行的 approval manifest 必须绑定 Owner、task/version、输入及清洗后字节哈希、processor/adapter version、Provider 账号/项目、endpoint/region/model、完整参数/prompt/mask、数量、最坏含税费用、币种和过期时间。预算预占、attempt 创建和 approval 单次消费在同一数据库事务中完成，并以 approval ID 唯一约束阻止并发双花。

### 17.6 BlobStore 与旧数据迁移

仓库当前主要把处理后图片保存在 PostgreSQL `BYTEA`，不能假定对象存储已经存在。先定义 `BlobStore`：`PutStaged/Open/Promote/Delete`，并保存 `staged/ready/quarantined/delete_pending`、object key/version/etag/hash；事务 outbox 驱动 promote/reconcile，只有 `ready` 资产可以审核。

Lake 1 可继续使用 BYTEA 完成正确领域链，但必须明确旧 `processed_bytes` 双读、回填、哈希核对、前端切换、停止旧写、回滚和保留只读的迁移顺序。不得在新存储未就绪时声称对象存储完成。

### 17.7 图片预算账户

通用外部 Provider 预算账本目前不存在，必须在首次付费 Lake 中建设 `image_budget_account` 与 `image_budget_reservation`。金额使用 NUMERIC 与 ISO 币种，状态为 `reserved/captured/released/late_charge`；事务原子预占最坏费用，账单迟到和汇率差异不能被静默覆盖。

### 17.8 权利、规则与发布的跨阶段重验

权利在 submit、结果采用、prepare-media、Listing approval 和 publish execution 五个阶段重验。manual/channel import 不继承父图授权，默认 `pending_rights`。渠道规则在 prepare、草稿批准和发布执行时重新核验；渠道、站点、类目、图片角色、广告位或规则变化会使 selection 与 image set 变为 `stale`。

Owner 必须批准最终可发布字节及对象 version，不只是中间候选。发布领域必须强制消费 `image_release_attestation`，其中绑定 listing version、整组 media hashes、渠道规则和 Owner 图片决定；直接写 media、历史资产、`concept_only` 或 blocked review 都不能创建或执行发布。

### 17.9 API 与迁移修正

- 拆分服务端受控的 `/assets/from-sourcing-snapshots` 与外部 `/imports`，客户端不能声明受控来源或真实性；
- 所有新 mutation 登记 route catalog、风险级别、权限、action type 和 approval target；
- Convert 新增 `approved_product_image_set_id` 双读路径，验证 Owner、来源主体、快照、review manifest、用途和渠道；
- 先双读旧记录，再回填、切换前端、停止旧写，旧表保留只读；
- PostgreSQL 明确 CHECK、复合 FK、partial unique、immutable trigger、CAS、父资产无环和跨 Owner 约束；SQLite 单测不能替代这些验证。

### 17.10 Lake 的可用性与 Owner 体验

统一合同从 Lake 1 起容纳六类来源，但不能把它们都写成已实现。状态必须分别报告 `implemented/configured/sandbox_verified/production_verified`。Lake 1 自身就必须具有完整 Owner 页面和整组图片审核；不能等到 Lake 5 才可用。

外部工具统一通过 tool registry 和 manual/channel-native import，不为 Pebblely、Canva、Photoshop分别建设伪 API 集成。批量能力只有在单 SKU 流程真实稳定且 Owner 明确启用后进入活动范围。Provider 推荐必须展示依据、适用合同、价格、地域和观察时间；“precision”等名称不能被写成已证明保真。

### 17.11 强制新增测试

- Provider 已处理但响应在返回途中断流：自动调用次数必须为 1，不能普通 retry；
- 100 个并发 execution：同一 approval 只有一次预算预占和外部提交；
- 权利在任一重验阶段撤销：后续外呼、采用、prepare 和发布均为 0；
- 最终字节、顺序、元数据、对象版本或渠道规则变化：旧图片与发布批准全部失效；
- 直接 SQL/API 塞入未 attested asset：发布失败关闭；
- 对象 put/DB/promote 任一点崩溃：重启后收敛，篡改对象必阻断；
- 真实 Provider 调用不进入普通 CI；deterministic/manual E2E 与 Provider fixture 必跑，sandbox 作为隔离 canary。
