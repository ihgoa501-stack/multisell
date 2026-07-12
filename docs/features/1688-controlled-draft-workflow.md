# 1688 货源到待上架草稿受控闭环

> 添加时间：2026-07-12
> 提出人：Owner
> 优先级：P0
> 状态：工程实现中；真实商品验收尚未完成

## 一句话说明

仅针对 Owner 已批准的候选市场、销售渠道和商品实验，把一个真实 1688 货源保存为不可变证据，经 Owner 复核后生成凌镜内部产品、SKU、图片/成本记录和待上架草稿；草稿阶段绝不自动发布，任何外部发布都必须另行申请、另行批准并再次显式执行。

## 业务边界

- 1688 是供应来源，不是市场选择依据。
- 入口必须关联 `experiment_ready` 的 `demand_case` 和同一 Owner 的 active `experiment_case`。
- 快照保存原链接、采集时间、采集驱动、解析版本、原始 JSON 和 SHA-256；数据库拒绝更新或删除快照。
- 主采集入口通过 Owner 浏览器扩展读取真实 1688 URL，保存实际返回的结构化响应和最多 256 KiB 的脱敏 DOM 页面结构；不会把该结构冒充未经处理的完整网页源码。
- 状态只允许严格流程：`capture_failed / pending_review → rejected / ready_for_product → editing → pending_approval → approved_draft`；批准后仍是内部草稿。
- 转草稿事务同时创建产品、SKU 三段映射、实际图片处理记录、10 类费用与独立汇率/贡献利润校验、`product_listing(status=draft)` 和实验对象关联。
- 同一 1688 offer 统一规范链接；内容指纹发现跨链接/跨供应商疑似同款并进入 Owner 待复核，价格、MOQ、供应商变化保存结构化事件。
- 图片处理真实执行解码、中心裁切、缩放和白底合成，保存源/结果 SHA-256 与处理版本；不会自动去水印或品牌标识。
- 本地化、渠道类目、变体、图片、配送模板、成本和 SKU 使用带来源与观察时间的确定性规则快照；blocker 非空时不能转草稿。
- 转草稿流程没有平台适配器调用、发布按钮或发布状态转换。独立发布流程要求第二条高风险 Owner 审批，冻结账号、SKU、价格、库存与请求哈希后才允许再次显式执行。
- 平台无错误响应只记为 `submitted`，不等于商品已创建或上线；超时/不确定结果记为 `reconcile_required`，只能用 Owner 核验的 `actual` 平台证据后置对账。
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
- `POST /api/v1/sourcing-1688/:id/submit-draft-approval`
- `POST /api/v1/sourcing-1688/:id/approvals/:approvalId/decision`
- `GET/POST /api/v1/sourcing-1688/:id/publish-requests`
- `POST /api/v1/sourcing-1688/:id/publish-requests/:attemptId/decision`
- `POST /api/v1/sourcing-1688/:id/publish-requests/:attemptId/execute`
- `POST /api/v1/sourcing-1688/:id/publish-requests/:attemptId/reconcile`

全部位于 JWT 保护组。采集人、复核人和草稿创建人以 JWT 用户身份为准，不信任请求体声明。

## 验收标准

### 自动验证

- 非 1688 HTTPS URL、无解析版本、无有效原始 JSON时拒绝采集。
- 未批准市场、不同 Owner、未关联实验时拒绝采集。
- 相同快照重试保持幂等；变化数据新增快照并回到待复核。
- 未复核记录不能转草稿。
- 图片缺少已核验权利或处理记录时拒绝。
- 10 类费用任一缺失，或必要汇率缺少可信来源，或真实性为 unknown/mock/inferred 时拒绝。
- 最终 listing 必须是 `draft`，且不产生外部调用。
- 草稿审批必须关联 canonical `approval_request`；批准和拒绝都记录审计，批准不能改变 listing 的 `draft` 状态。
- 发布申请必须使用另一条未过期的高风险 Owner 审批；批准动作本身不能调用平台，只有再次显式执行才能触发一次外部写。
- 发布请求和平台规范化返回保存 SHA-256；终态记录不可修改，失败或超时不得自动重试。

### 真实人工验收

使用 1 个真实 1688 商品完成：真实页面采集 → Owner 对照快照复核 → 产品/SKU/图片/成本检查 → 渠道草稿保存与预览 → 反向核对完整追溯链。

该项目前为 `unknown`，在真实商品、登录授权、图片使用权和费用证据齐备前不得声明闭环完成或生产可用。

## 涉及模块

- 后端：`backend-go/internal/domain/sourcing1688/`
- 数据库：`backend-go/migrations/000084_sourcing_1688_draft_workflow.*.sql`
- 数据库：`000085` 生命周期审批、`000086` 同款与变化、`000087` 图片处理、`000088` 采集失败记录、`000091` 独立发布审批与结果账本
- 采集扩展：`chrome-extension/`
- 前端：`frontend-next/src/app/(main)/sourcing1688/page.tsx`

## 不在范围内

- 批量采集、自动采购、自动图片生成；
- 自动发布、自动重试、价格/库存同步；
- 多个 active 店铺账号下的真实发布（现有适配器不能可靠按批准账号取凭据，当前门禁会拒绝）；
- 未经市场闸门的平台扩张；
- 用 Mock 数据代替真实商品验收。
