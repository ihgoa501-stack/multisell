# 商品视觉生产与学习系统开发规格

> 初始日期：2026-07-12
> Owner 范围更新：2026-07-13
> 状态：`implemented / automated_verified / isolated_browser_verified / owner_real_sku_pending`
> Owner 决策：建设 Owner 自用的商品视觉生产与学习能力；先完成单 SKU 配方闭环，暂停无真实验证的 Provider 和功能扩张
> 证据限制：本规格不代表 Provider 已配置、图片权利已取得、真实渠道已接受或经营效果已成立

## 1. 目标与边界

建设凌镜统一的商品视觉生产与学习系统：把 exact SKU 的可信素材，通过确定性处理、生成式 AI、模板、人工处理或实拍，转化为适配具体渠道与经营用途的图片；保存完整制作配方、Owner 选择与拒绝原因、成本、渠道结果和经营效果，逐步学习“什么制作方法对什么商品、渠道和目标有效”。

AI 是重要生产手段，但不是产品边界。系统的核心资产不是某个 Provider、模型或孤立提示词，而是：

- exact SKU 商品事实与获权原始素材；
- 参考图、主体 mask、模板和品牌规则；
- 绑定模型版本、参数和输出的版本化制作配方；
- Owner 选择、拒绝、错误区域和返工反馈；
- 渠道规则、制作效率、发布结果与经营效果数据。

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

它不是通用绘画 SaaS、Canva/Photoshop 替代品或提示词库，不服务外部用户，不训练自有基础模型，不自动发布，不以图片生成成功或主观美观替代真实商品、渠道和经营效果。

### 1.1 当前开发单元：单 SKU 可用生产闭环

当前系统已有资产、任务、确定性处理、权利、预算、审核、图片集合、发布门禁和 Image Service，并已实现 exact SKU 配方冻结、候选反馈、返工继承和实时统计。2026-07-13 在隔离 PostgreSQL 与浏览器中完成“首轮候选 → 要求返工 → 第二轮候选 → 五类审核 → 选用”的工程验收；该证据是 `manually_verified` 的隔离工程行为，不是 Owner 真实 SKU、真实场景图、真实 Provider 或渠道效果验证，因此仍不能开始 3 SKU 对照验证。

当前唯一开发目标是完成：

```text
1 个真实 exact SKU + 场景副图用途
→ 绑定真实商品事实与获权素材
→ 选择人工导入、确定性模板或 1 个真实可调用 AI Provider
→ 保存完整 workflow recipe
→ 生成/导入候选并与原图并排检查
→ 记录选择、拒绝原因、错误区域与返工
→ Owner 批准精确最终文件
→ 自动汇总耗时、人工分钟、调用/外包成本、候选数与返工次数
```

完成标准不是 API、页面或测试单独存在，而是 Owner 能亲自在页面从真实素材走到批准的最终场景副图，并能读取完整配方和成本。首轮不使用 AI 重绘承担商品事实的主图主体。该闭环真实验收前，后续 Provider、批量和高级能力只保留为方向，不进入当前开发队列。

### 1.2 下一验证单元：3 SKU 三路线对照

只有第 1.1 节由 Owner 真实操作通过后，才进入：

```text
3 个真实 exact SKU × 1 个真实目标渠道 × 场景副图用途
→ A：现有人工/现成工具基线
→ B：真实商品图 + 确定性处理/模板
→ C：真实商品图 + 1 个 AI 场景 Provider + Owner 审批
→ 比较时间、人工分钟、总成本、首轮通过率、返工、事实错误、渠道结果和可比经营效果
```

## 2. 目标能力边界

### 2.1 处理路径

| 路径 | 当前处置 | 主要用途 | 是否外部付费 |
|---|---|---|---:|
| `deterministic` | 保留并作为验证路线 B | 裁剪、缩放、白底、格式、压缩、固定模板 | 否 |
| `commerce_ai` | 只选择 1 个 Provider 作为验证路线 C；当前真实 canary 仍为 `unknown` | 去背景、阴影、补光和商品场景 | 是 |
| `precision_edit` | 暂停扩张 | 局部修补、精确合成、扩图 | 是 |
| `creative_ai` | 暂停扩张 | 场景副图和广告创意候选 | 是 |
| `manual_import` | 保留，作为路线 A 和人工兜底的事实记录入口 | 外部编辑结果回传 | 视外部工具而定 |
| `channel_native_import` | 保留导入合同，不新增渠道接入 | 渠道内置工具结果回传，仅限对应渠道 | 视渠道而定 |

统一领域合同可以容纳上述路径，不代表六条路径都要在首批实现或当前开发。外部 API 适配器只有在真实需要、配置和官方契约核验完成后才标为 `available`；未配置时明确 `unavailable`，不得以 mock 结果冒充。

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
- 通用画布编辑器、提示词市场或海量提示词模板库；
- 把单独 prompt 字符串当成跨模型、跨版本稳定的核心资产；
- 在单 SKU 流程未真实稳定前建设批量生成；
- 在当前验证前新增视频、虚拟模特、自训练 LoRA 或更多 Provider；
- 用 AI 重绘主图商品主体替代真实商品证据；
- 仅凭图片更美观宣称点击、转化或利润改善。

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

一次处理执行：processor、provider/model/version、凌镜内部稳定幂等键、Provider 请求 ID、参数、prompt/mask、预计和实际费用、状态、错误、提交/对账时间。只有 Provider 官方契约明确支持并可验证去重时，才能把内部键作为 Provider 幂等键；否则付费请求只允许单次提交，响应未知后禁止自动重试。

这里保存的是完整版本化 `workflow recipe`，而不是孤立 prompt：目标用途、exact SKU 不可变事实、输入/参考资产、mask、模板/品牌规则、结构化创作意图、Provider 特定 prompt、模型版本、参数、渠道规则、候选和输出哈希必须可以共同追溯。模型或版本变化后，旧 recipe 必须重新验证，不能承诺原 prompt 自动复现旧结果。

### 4.5 `product_image_review`

逐资产保存商品真实性、权利、渠道规则、声明、技术视觉和 Owner 审核；每项独立为 `passed/blocked/unknown`，保存证据和审核时间。Owner approval manifest 绑定资产、SKU、用途、渠道、规则、检查结果和最终派生文件哈希。

### 4.6 `product_image_cost_entry`

保存 Provider、存储、外部工具、人工/外包等现金成本及币种、汇率来源、观察时间和账单状态；连接项目预算账本，不允许拆任务绕过总预算。

### 4.7 生产反馈与效果观察

逐个图片集合保存总耗时、人工审核分钟、候选数、首轮通过率、返工轮次、事实错误类型、渠道批准/拒绝和与图片不符有关的投诉/退货。曝光、点击、加购、转化等效果必须绑定商品、渠道、图片集合版本和观察窗口；价格、标题、流量、促销等混杂因素保持显式 `unknown`，没有受控比较时不得宣称图片导致经营结果变化。

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
- 不支持可靠查询或可验证 Provider 幂等的付费能力，只允许 Owner 明确接受的单次调用；响应未知后失败关闭并进入人工对账，绝不自动重试。

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

终态不可覆盖；Worker 使用同一 Repository 事务同时终结 job 与 attempt，Provider request ID 和输出/错误不会被拆成两个崩溃窗口；其他状态更新使用 PostgreSQL 约束和 `status + version` CAS。job/attempt/output ordinal、Provider request ID、Owner 选中主资产均有数据库唯一约束。

## 7. API

根路径：`/api/v1/product-images`。所有资源按 JWT Owner scope 查询；跨 Owner ID 返回 404。

- `GET /processors`：返回已配置能力，不泄漏凭据；
- `POST /assets`：注册受控来源或上传外部结果；
- `POST /manual-imports`、`GET /manual-imports`：上传并分页读取外部编辑结果；服务端冻结父资产 SHA-256、工具、操作、费用、模型/版本、来源时间和渠道限制，字节仍须经过 Image Service 清洗；
- `GET /assets`、`GET /assets/:id`、`GET /assets/:id/content`；
- `POST /tasks`、`GET /tasks`、`GET /tasks/:id`；
- `POST /tasks/:id/approval-requests`；
- `POST /tasks/:id/executions`：消费目标绑定批准并执行；
- `POST /tasks/:id/reconciliations`；
- `POST /tasks/:id/retries`；
- `POST /tasks/:id/reviews`：提交 G1–G5；
- `POST /tasks/:id/feedback`：冻结 `selected/rejected/rework_requested`、原因、错误区域、返工要求和人工审核秒数；
- `GET /recipes/:recipe_key/summary?sku_id=`：按 exact SKU 与配方实时汇总候选、采用、拒绝、返工、通过率、审核/生产耗时和已对账成本；
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

- Provider 接收请求但响应丢失：有官方查询能力时按精确请求编号查询；没有查询能力时保持 `RECONCILE_REQUIRED`，禁止付费重试。Owner 只能用可信证据结案为“追回输出”“确认未扣费”或“确认已扣费但无可恢复输出”；
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

## 13. 验证后再启动的开发切片

下列 Lake 是已有工程和可能的后续方向，不代表当前自动继续开发。当前活动任务只有第 1.1 节的单 SKU 可用生产闭环；该闭环由 Owner 真实验收后才进入第 1.2 节对照验证。只有验证显示明确的时间、成本、错误、素材供给或经营价值改善，Owner 才决定恢复对应 Lake。恢复后，每个切片都完成领域、API、Owner 体验、权限、审批、审计、错误、测试、运行验证和文档。

### Lake 1：统一资产与确定性处理

建立 asset/rights/task/review/cost，迁移现有确定性处理，完成导入、审核、草稿接入和对象一致性。

当前工程实现（2026-07-12，`automated_verified`）：

- `product_image_rights_grants` 将 Owner 与精确资产 SHA-256 绑定，逐项记录复制、修改、第三方 AI、跨境、商业发布、平台再许可、商标和肖像许可，以及用途、法域、渠道、Provider、地域、有效期、撤销、证据 SHA-256 和 Owner 核验；
- 五类审核分别保存商品真实性、权利、渠道规则、声明/场景和技术视觉的 `passed / blocked / unknown`，普通 API 输入不能把证据声明为 `actual`；
- `product_image_cost_entries` 保存预计/实际金额、严格十进制字符串、币种、汇率及来源、观察时间和账单状态；付费执行批准同时落一条预算上限预计成本；
- `product_image_budget_policies`、预算预占和追加式费用记录按 Owner、币种和周期共享总上限；付费审批在事务内预占，外呼前 `reserved → claimed`，结果未知时不得释放。可信未扣费证据必须先让 Image Service 原子停止精确任务，随后记录不可变 `no_charge` 并释放；真实账单使其进入 `spent`，确认已扣费但无可恢复输出记录 `charged_no_output` 终态；迟到费用继续追加并可标记 `over_budget`；
- 权利、审核和成本写命令使用 Owner scope、`expected_version` 与幂等键，读取接口分页；跨 Owner 对象按不存在处理；
- Listing image set 在创建和冻结时都会重新核验精确输出字节的有效发布权利和五类全通过。确定性处理不因成本未知而阻断；付费候选还必须存在非 `unknown` 成本记录。

Owner 尚未为真实商品录入权利证据或完成五类审核，因此这些工程门禁不能被解释为真实图片已获授权、渠道已接受或可生产发布。

预算 API：

- `POST /api/v1/product-images/budget-policies`
- `GET /api/v1/product-images/budget-policies`
- `GET /api/v1/product-images/budget-reservations`
- `POST /api/v1/product-images/budget-reservations/:reservation_id/cancel`
- `POST /api/v1/product-images/budget-reservations/:reservation_id/charges`
- `POST /api/v1/product-images/budget-reservations/:reservation_id/no-charge-reconciliations`

预算并发验收：周期上限 `1.00 USD`、100 个并发 `0.02 USD` 审批只能成功 50 个，总预占精确为 `1.0000`。这只证明数据库预占约束，不证明 Provider 账单或经营成本已对账。

### Lake 2：外部工具导入

如真实验证需要，继续维护 Pebblely、Canva、Photoshop及渠道内置结果的统一回传、来源标记、费用和渠道限制；不为每个外部工具分别建设集成。

当前工程状态：`implemented / automated_verified`。通用外部编辑与渠道内置结果都只做安全回传，不调用外部工具；渠道内置结果强制限定为原渠道。导入资产的真实性固定为 `unknown`，不会绕过权利或五类审核。

### Lake 3：专业商品 AI（当前仅保留隔离 canary，暂停扩张）

实现 Photoroom Provider、目标绑定付费审批、预算、幂等、对账、失败和 sandbox/生产契约验证。

当前工程状态：`sandbox_path_automated_verified / real_canary_unknown / production_forbidden`。Photoroom 只有同时满足显式 sandbox 开关、专用 API key、sandbox 账号确认和训练退出确认时才注册；仅 development/acceptance 可开启，production 环境开启会拒绝启动。任务固定为 US 区域、sandbox 环境、PNG、`0 USD` 和三个允许操作，消耗一次不可自动重置的 canary 次数。系统不假定 Provider 已添加像素水印；Image Service 对成功结果本地叠加明显的 `SANDBOX` 像素横幅，重编码后逐像素验证，验证成功才永久绑定 `sandbox + watermarked + non_publishable`。这些输出不能进入 Image Set、release attestation 或受控发布。HTTP 客户端拒绝所有重定向，断流、连接失败和 5xx 进入 `RECONCILE_REQUIRED`，没有自动重试。当前没有真实凭据，也没有执行真实网络 canary。

### Lake 4：精确编辑与创意（暂停）

OpenAI 场景编辑已接入显式关闭的生产付费路径，只允许 `gpt-image-2`、单张精确原图和冻结 Prompt；创建、批准、执行分别校验版权、任务版本、预算预占和单次提交门禁。OpenAI Images Edits 官方契约当前未提供可验证的请求幂等或按请求查询结果能力，因此系统不把 `Idempotency-Key` 请求头当作供应商去重证据：一旦响应不确定就禁止自动重试，并要求 Owner 对账。成功、响应异常和本地 Blob 写入失败均尽量保留脱敏 Provider request ID。`max_cost` 是凌镜预算预占上限，不是供应商硬封顶。当前仅为 `automated_verified`，没有项目专用凭据和 Owner 真实 SKU 外部运行证据，因此不得写成 `external_observed` 或生产验收完成；真实单次验收必须遵循 [`OPENAI_PRODUCT_IMAGE_OWNER_CANARY_RUNBOOK.md`](../ops/OPENAI_PRODUCT_IMAGE_OWNER_CANARY_RUNBOOK.md)。Adobe 仍只在真实缺口出现后评估。

### Lake 5：完整 Owner 工作室与小Q只读（等待真实验证）

真实验证通过后，按已观察到的重复需求建设统一对比、必要批量任务、全尺寸审核、成本视图、运行监控和满足契约后的只读 Capability。

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

## 16. 图片发布证明（Image Release Attestation）

当前后端已建立发布前证明与受控媒体交付合同，但尚无真实平台 Adapter 实现该新合同：

- `product_image_rule_snapshots` 保存渠道、站点、语言、类目和规则正文的不可变版本与规范哈希；新规则版本不会覆盖旧快照；
- `product_image_set_decisions` 保存 Owner 对精确 frozen image set 版本与 manifest 的批准或拒绝；
- `product_image_release_attestations` 绑定 Owner、Listing、商品、平台、账号、渠道、站点、语言、类目、Listing 快照、图片集、规则、Owner 决定、权利与五类审核 manifest；
- 每个 item 固定 ordinal、role、task、blob、SHA-256、MIME、宽高。签发和消费都会重新读取 Image Service 字节并复算 SHA-256；
- claims 使用规范 JSON、SHA-256 与独立 HMAC 密钥签名，保存 key ID、nonce、签发时间和过期时间；发布 attempt 与证明在同一事务内完成 `issued → claimed`，任何外呼不确定性进入 `reconcile_required`，只有确定成功回执才能进入 `consumed`；
- `ControlledPublisher` 只接收后端从 Image Service 重新读取并复算后的 bytes、SHA-256、MIME、role 和尺寸，不存在任意 URL 字段；100 个并发请求只能产生一次外呼；
- 旧 `PublishInput.MainImage/Images` URL Adapter 不满足新合同，默认不注册并以 `CONTROLLED_PUBLISHER_UNSUPPORTED` 失败关闭；
- 权利撤销、最新规则版本改变、Listing 内容改变、图片字节改变、审核或任务血缘改变、签名篡改和过期均失败关闭。

Owner API：

- `POST /api/v1/product-images/rule-snapshots`
- `POST /api/v1/product-images/image-sets/:set_id/decisions`
- `POST /api/v1/product-images/release-attestations`
- `GET /api/v1/product-images/release-attestations/:attestation_id`
- `POST /api/v1/product-images/release-attestations/:attestation_id/publish-attempts`
- `GET /api/v1/product-images/publish-attempts/:attempt_id`
- `POST /api/v1/product-images/publish-attempts/:attempt_id/reconcile`

`unknown`：现有真实平台 Adapter 仍只支持旧 URL 合同，没有一个已经实现 `ControlledPublisher` 并在真实 sandbox/production 验证。因此当前可证明“旧 Adapter 无法绕过、受控 attempt 状态机正确”，不能宣称任何真实渠道已经发布成功或受到该证明保护。

旧 `/api/v1/platform-integrations/publish-to-ozon` 和通用 `write-back`/retry 已从路由注册与 mutation 目录退役；`sourcing1688` 旧发布执行也已移除真实 Adapter 调用代码。真实发布只允许经过 task-link 冻结、Owner 独立批准、release attestation 与终态回执的受控链路。

## 15. 当前未知与实施前置

- Provider 账号、当前价格、配额、地区、数据保留和正式合同；
- 第一个真实商品和图片权利；
- 目标渠道与类目规则；
- Adobe Precise Composite 的实际可用契约；
- 不同方案对真实 SKU 的保真率和 Owner 人工成本。

这些未知阻止对应 Provider 的生产可用声明，但不阻止按上述合同完成领域、适配器、失败关闭和自动测试。

## 16. 文档关系

- 本文是商品视觉生产与学习系统的当前权威规格；
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

当前 Photoroom 接线默认关闭，使用 `IMAGE_SERVICE_PHOTOROOM_SANDBOX_ENABLED=true`、API key、sandbox 账号确认和 training opt-out 确认四门同时开启。Image Service 的只读 capability 必须明确返回 `sandbox_only / us / sandbox / watermarked / non_publishable / quota_remaining=1`，LingMirror 才允许创建、批准和执行。这里的 `watermarked` 是 Image Service 本地重编码并逐像素验证的 SANDBOX 横幅事实，不是对 Provider 原始输出的推测。任务创建、批准和执行都会重新核验精确的复制、修改、第三方 AI 与跨境权利，Provider 与 region 不接受通配绕过。执行令牌精确绑定 task/version、manifest、operation、region、provider environment、sandbox/watermarked/non-publishable、`0 USD` 和 nonce。

Photoroom HTTP 客户端拒绝全部重定向，固定正式 Host，并禁止 Base URL 携带 userinfo、query 或 fragment。执行阶段以固定锁顺序在同一事务锁定 Task、精确 RightsGrant、Approval 和预算预占，创建不可变 `ExecutionRightsSnapshot` 后才签发执行令牌；撤权先提交则零入队，执行先 claim 则冻结该次 grant ID/version/evidence，后续撤权只影响新执行。部署环境只接受 `development / acceptance / production`，Photoroom 只允许前两者；acceptance/production 服务密钥至少32字节且不得复用执行令牌密钥。

生产付费执行的 approval manifest 必须绑定 Owner、task/version、输入及清洗后字节哈希、processor/adapter version、Provider 账号/项目、endpoint/region/model、完整参数/prompt/mask、数量、最坏含税费用、币种和过期时间。预算预占、attempt 创建和 approval 单次消费在同一数据库事务中完成，并以 approval ID 唯一约束阻止并发双花。

### 17.6 BlobStore 与旧数据迁移

当前 Image Service 的 Job/Attempt/nonce/Provider submit claim 保存在 PostgreSQL，图片字节保存在内容寻址的本地持久卷；它不是对象存储，也没有完成备份恢复演练。readiness 会实际写入、fsync、读回并清理探针文件，但这只能证明当前卷可用。

Lake 1 可继续使用 BYTEA 完成正确领域链，但必须明确旧 `processed_bytes` 双读、回填、哈希核对、前端切换、停止旧写、回滚和保留只读的迁移顺序。不得在新存储未就绪时声称对象存储完成。

### 17.7 图片预算账户

预算政策、预占和追加式费用账本已实现。金额使用 PostgreSQL NUMERIC 与 ISO 币种，状态为 `reserved/claimed/spent/released`；事务原子预占最坏费用，结果未知时不能释放，账单迟到和超支只追加新记录而不覆盖历史。免费 Photoroom sandbox 另有一次性 canary 次数门禁，因为 `0 USD` 不等于不消耗 Provider 配额。

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
