# 1688 货源到待上架草稿受控闭环

> 添加时间：2026-07-12
> 提出人：Owner
> 优先级：P0
> 状态：`implemented / automated_verified`；真实商品、真实列表批次和渠道草稿的外部验收仍为`unknown`

## 一句话说明

仅针对Owner最新选择的市场决定和已经批准的商品机会，把一个真实1688货源保存为不可变证据，经Owner复核后生成凌镜内部产品、SKU、图片/成本记录和待上架草稿；`experiment`只保留事实追踪作用，不能授权。草稿阶段绝不自动发布，任何外部发布都必须另行申请、另行批准并再次显式执行。

## 业务边界

- 1688 是供应来源，不是市场选择依据。
- 进入受控研究前必须关联同一Owner最新`selected`市场决定下、经Owner批准且仍有效的`product_opportunity_id + opportunity_decision_id`；`experiment_id`仅用于trace，不是授权条件。
- 快照保存原链接、采集时间、采集驱动、解析版本、可信采集入口来源、原始响应字节和 SHA-256；PostgreSQL 使用 `BYTEA` 防止 JSONB 改写空白或键顺序，数据库拒绝更新或删除快照。只有服务器受控 `/fetch` 路径写入的 `controlled_fetch` 可进入真实验收，普通 `/capture` 固定标为 `manual_import`，不能通过伪造 driver/raw_html 冒充真实采集。
- 私人收藏入口通过Owner绑定的浏览器扩展上传固定schema的有限结构化页面观察，不克隆或上传整页DOM。后续服务器受控`/fetch`路径可为已授权任务保存实际响应字节和受限、脱敏的必要页面片段；两种证据来源不能混为同一可信等级。
- 状态只允许严格流程：`capture_failed / pending_review → rejected / ready_for_product → editing → pending_approval → approved_draft`；批准后仍是内部草稿。
- 提交草稿审批时对产品、listing 草稿、SKU、图片和成本的稳定字段统一计算 SHA-256，并写入 canonical approval；待审批期间数据库禁止修改这些内容，审批、发布申请、发布执行和验收报告都会重新核对该指纹。
- 转草稿事务同时创建产品、SKU三段映射、实际图片处理记录、10类费用与独立汇率/贡献利润校验、`product_listing(status=draft)`，并保存商品机会授权引用和可选trace引用。
- 同一 1688 offer 统一规范链接；变体数组顺序等无业务意义的差异会先规范化，标题轻微变化和跨链接/跨供应商疑似同款会进入同一 Owner 的待复核队列，价格、MOQ、供应商变化保存结构化事件，不向另一个 Owner 泄露内部记录。
- 图片处理真实执行解码、中心裁切、缩放和白底合成，保存源/结果 SHA-256 与处理版本；不会自动去水印或品牌标识。
- 供应商核验必须逐项保存结论、实际事实值、来源和观察时间；本地化语言必须与已批准市场的 `target_locale` 一致。渠道类目、变体、图片、配送模板、成本和 SKU 使用带来源与观察时间的确定性规则快照；blocker 非空时不能转草稿。
- 转草稿流程没有平台适配器调用、发布按钮或发布状态转换。独立发布申请仍用第二条高风险 Owner 审批冻结账号、SKU、价格、库存与请求哈希，但旧执行入口的图片是 URL，不能满足图片发布证明和受控字节合同；`approval_required / production` 执行固定在任何 adapter 调用和状态变更前返回 `IMAGE_RELEASE_ATTESTATION_REQUIRED`。它不会自行消费或伪造图片发布证明。
- `dry_run / sandbox` 在该旧入口只生成 `truth_status=mock` 的本地模拟记录，状态为 `mocked`，不解析或转发任意图片 URL，不调用凭据校验或平台 adapter，不改变 listing 的 `draft` 状态，也不能升级为平台观察终态。历史 `submitted / reconcile_required` 记录仍可按原证据规则读取和对账，但新记录只能经图片系统的 release attestation 与 `ControlledPublisher` 字节入口形成。
- 旧的自由创建、更新、删除和“import” HTTP 路由不再注册，避免绕过证据与 Owner 门禁。

## API

- `GET /api/v1/sourcing-1688`
- `GET /api/v1/sourcing-1688/:id`
- `GET /api/v1/sourcing-1688/summary`
- `POST /api/v1/sourcing-1688/capture`
- `POST /api/v1/sourcing-1688/fetch`
- `POST /api/v1/sourcing-1688/:id/review`
- `POST /api/v1/sourcing-1688/:id/convert-to-draft`
- `GET /api/v1/sourcing-1688/:id/identity-history`
- `POST /api/v1/sourcing-1688/duplicates/:id/resolve`
- `POST /api/v1/sourcing-1688/processed-images`
- `GET /api/v1/sourcing-1688/processed-images/:id/content`
- `POST /api/v1/sourcing-1688/capture-failures`
- `GET /api/v1/sourcing-1688/:id/lifecycle`
- `GET /api/v1/sourcing-1688/:id/acceptance-report`
- `POST /api/v1/sourcing-1688/:id/submit-draft-approval`
- `POST /api/v1/sourcing-1688/:id/approvals/:approvalId/decision`
- `GET/POST /api/v1/sourcing-1688/:id/publish-requests`
- `POST /api/v1/sourcing-1688/:id/publish-requests/:attemptId/decision`
- `POST /api/v1/sourcing-1688/:id/publish-requests/:attemptId/execute`
- `POST /api/v1/sourcing-1688/:id/publish-requests/:attemptId/reconcile`

上述受控经营接口位于JWT保护组。只读接口要求`product.read`，编辑和草稿审批要求`product.write`，真实渠道发布申请、批准、执行和对账另要求`listing.publish`。插件私人收藏另使用`/api/v1/extension/sourcing-1688/private-collections`和设备绑定的`sourcing1688.collect`窄权限；私人收藏关联已批准商品机会后，才进入JWT保护的受控读取、复核和草稿链。操作者身份以服务端凭证为准，不信任请求体声明。

`acceptance-report` 是只读且按 Owner 隔离的逐项验收报告。它只根据当前数据库中该商品的持久化事实返回 15 项 `passed / blocked / unknown`；不返回原始 HTML 或冻结发布请求。只有 15 项全部为 `passed` 时 `ready=true`。代码存在、页面可见或自动测试通过不会提高真实验收状态。

## 验收标准

### 自动验证

- 非 1688 HTTPS URL、无解析版本、无有效原始 JSON时拒绝采集。
- 市场未被最新Owner决定选中、商品机会未获批准或Owner不一致时，拒绝进入受控采集与草稿；未关联机会的私人收藏仍可保持`unverified_lead`。
- 相同快照重试保持幂等；变化数据新增快照并回到待复核。
- 未复核记录不能转草稿。
- 图片缺少已核验权利或处理记录时拒绝。
- 10 类费用任一缺失，或必要汇率缺少可信来源，或真实性为 unknown/mock/inferred 时拒绝。
- 最终 listing 必须是 `draft`，且不产生外部调用。
- 草稿审批必须关联 canonical `approval_request`；批准和拒绝都记录审计，批准不能改变 listing 的 `draft` 状态。
- 发布申请必须使用另一条未过期的高风险 Owner 审批；批准动作本身不能调用平台。旧执行入口即使所有审批、合规和冻结内容都有效，也必须以 HTTP 428 和 `IMAGE_RELEASE_ATTESTATION_REQUIRED` 失败关闭，保持 attempt=`approved`、listing=`draft`，且平台 adapter 调用次数为零。
- `dry_run / sandbox` 只能留下不含外部引用的本地 `mocked` 结果及 SHA-256；任意 URL 都不能到达 adapter，也不能产生 `submitted / succeeded / external_observed`。历史平台返回和终态记录继续保持不可修改，失败或超时不得自动重试。
- 同一货源同一时间最多存在一个未解决的发布流程；更换幂等键也不能并行创建第二条外部写路径。
- 受保护的处理后图片由前端携带 Owner 令牌读取并实际显示；图片无法显示时不能当作已经人工确认。

### 真实人工验收

下游受控草稿链使用1个真实1688详情商品完成：真实页面采集 → Owner对照快照复核 → 产品/SKU/图片/成本检查 → 渠道草稿保存与预览 → 反向核对完整追溯链。该专项验收不替代插件首页/搜索页列表批次验收。

该项目前为 `unknown`，在真实商品、登录授权、图片使用权和费用证据齐备前不得声明闭环完成或生产可用。

## 涉及模块

- 后端：`backend-go/internal/domain/sourcing1688/`
- 数据库：`backend-go/migrations/000084_sourcing_1688_draft_workflow.*.sql`
- 数据库：`000085` 生命周期审批、`000086` 同款与变化、`000087` 图片处理、`000088` 采集失败记录、`000091` 独立发布审批与结果账本、`000095` 可信采集入口来源、`000097` 草稿内容审批指纹、`000099` 原始字节/采集请求/市场语言/RBAC/单发布链加固
- 采集扩展：`chrome-extension/`
- 前端：`frontend-next/src/app/(main)/sourcing1688/page.tsx`

## 不在范围内

- 自动翻页、整店采集、无人值守爬取、自动采购、自动图片生成；
- 首页/搜索页由Owner主动触发的“采集选中/采集本页”只创建私人未验证线索，不等于受控货源采集、批量铺货或草稿生成；
- 自动发布、自动重试、价格/库存同步；
- 多个 active 店铺账号下的真实发布（现有适配器不能可靠按批准账号取凭据，当前门禁会拒绝）；
- 未经市场闸门的平台扩张；
- 用 Mock 数据代替真实商品验收。
