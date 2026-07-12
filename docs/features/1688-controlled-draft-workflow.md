# 1688 货源到待上架草稿受控闭环

> 添加时间：2026-07-12
> 提出人：Owner
> 优先级：P0
> 状态：工程实现中；真实商品验收尚未完成

## 一句话说明

仅针对 Owner 已批准的候选市场、销售渠道和商品实验，把一个真实 1688 货源保存为不可变证据，经 Owner 复核后生成凌镜内部产品、SKU、图片/成本记录和待上架草稿；绝不自动发布到外部平台。

## 业务边界

- 1688 是供应来源，不是市场选择依据。
- 入口必须关联 `experiment_ready` 的 `demand_case` 和同一 Owner 的 active `experiment_case`。
- 快照保存原链接、采集时间、采集驱动、解析版本、原始 JSON 和 SHA-256；数据库拒绝更新或删除快照。
- 状态只允许严格流程：`pending_review → reviewed → draft_created`。
- 转草稿事务同时创建产品、SKU 映射、图片权利/处理记录、11 类成本输入、`product_listing(status=draft)` 和实验对象关联。
- 本流程没有平台适配器调用、发布按钮或发布状态转换。
- 旧的自由创建、更新、删除和“import” HTTP 路由不再注册，避免绕过证据与 Owner 门禁。

## API

- `GET /api/v1/sourcing-1688`
- `GET /api/v1/sourcing-1688/:id`
- `GET /api/v1/sourcing-1688/summary`
- `POST /api/v1/sourcing-1688/capture`
- `POST /api/v1/sourcing-1688/:id/review`
- `POST /api/v1/sourcing-1688/:id/convert-to-draft`

全部位于 JWT 保护组。采集人、复核人和草稿创建人以 JWT 用户身份为准，不信任请求体声明。

## 验收标准

### 自动验证

- 非 1688 HTTPS URL、无解析版本、无有效原始 JSON时拒绝采集。
- 未批准市场、不同 Owner、未关联实验时拒绝采集。
- 相同快照重试保持幂等；变化数据新增快照并回到待复核。
- 未复核记录不能转草稿。
- 图片缺少已核验权利或处理记录时拒绝。
- 11 类成本任一缺失，或真实性为 unknown/mock/inferred 时拒绝。
- 最终 listing 必须是 `draft`，且不产生外部调用。

### 真实人工验收

使用 1 个真实 1688 商品完成：真实页面采集 → Owner 对照快照复核 → 产品/SKU/图片/成本检查 → 渠道草稿保存与预览 → 反向核对完整追溯链。

该项目前为 `unknown`，在真实商品、登录授权、图片使用权和费用证据齐备前不得声明闭环完成或生产可用。

## 涉及模块

- 后端：`backend-go/internal/domain/sourcing1688/`
- 数据库：`backend-go/migrations/000084_sourcing_1688_draft_workflow.*.sql`
- 前端：`frontend-next/src/app/(main)/sourcing1688/page.tsx`

## 不在范围内

- 批量采集、自动采购、自动图片生成；
- 自动发布、价格/库存同步；
- 未经市场闸门的平台扩张；
- 用 Mock 数据代替真实商品验收。
