# 商品、货源与渠道准备纵向单元缺口审计

> 日期：2026-07-12
> 对应路线：ADR-001 第 3 单元
> 状态：开发中；不得声明完整或生产可用

## 已有工程证据

- 私人收藏、冻结商品机会授权、受控采集、不可变来源快照、Owner 复核、Listing 草稿、草稿审批、独立发布审批和超时对账已有实现及自动测试。
- 发布请求、决定、执行和失败对账均重新验证当前商品机会授权；市场暂停/拒绝或机会失效后 fail closed。
- 商品档案页的 Ozon 直接发布旁路已移除；任务选择改用唯一 `product_opportunity_id`，并展示冻结机会/批准 ID；小Q货源链接参数已修复。
- Owner 隔离的 1688 供应商身份及商品—供应商关系已绑定不可变来源快照，冲突 fail closed。
- 样品申请至接受/拒绝的严格事实链已绑定机会、任务、供应商、快照和 SKU；草稿批准要求已接受样品或不可变 Owner waiver。
- supplier/internal/channel 三段 SKU 已持久化为规范权威映射。
- 每个规范 SKU 可创建不可变精确成本版本：10 类金额使用最小货币单位，跨币种汇率使用十进制字符串并保存来源和观察时间；验收报告不再以旧浮点成本判定通过。
- 合规已提升为独立证据对象；只有当前、未撤销、Owner 批准的 `actual` 证据可通过六项验收。
- productimage 已实现 asset → job → output → 权利/五项审核 → frozen image set；旧 Listing 直接发布入口被封闭，必须使用受控发布尝试。
- 渠道发布已增加不可变终态证据：只有平台回执或受控对账可把 exact task 从 `submitted/reconcile_required` 推进为 `succeeded/failed`；执行、对账和终态写入均重新核验当前合规。

## 仍需开发

1. **变更失效联动**：成本、合规、图片集或渠道规则形成新版本后，必须使旧草稿/发布批准失效并要求 Owner 重新审议。
2. **图片链单一入口**：把 sourcing 页面直接引导到已绑定 Listing 的 productimage frozen image set，清理旧 `ImageProcessingRecord` 只能支撑历史验收的并行含义。
3. **跨层验收**：补 JWT/RBAC + 审批 + 浏览器 E2E，覆盖正常链、权限撤销、刷新恢复、超时不重试和图片回写。PostgreSQL 119 对迁移已完成全量 up → down → up 验证至版本 123。

## 证据边界

上述“已有”只属于 `implemented / automated_verified`。当前仓库没有真实供应商、真实样品、真实费用凭证、真实合规文件、可执行渠道账号、真实上线或消费者结果，因此这些经营事实仍为 `unknown`。
