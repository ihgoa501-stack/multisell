# 1688 受控货源到待上架草稿模块独立审计

> 审计日期：2026-07-12
> 审计范围：`backend-go/internal/domain/sourcing1688/`、`/api/v1/sourcing-1688`、前端 `/sourcing1688` 及直接相关迁移与测试
> 审计类型：只读代码/文档调研；除本报告外未修改代码或经营数据
> 当前裁决：工程闭环较完整，真实经营闭环尚未开始

## 1. 先给 Owner 的结论

这个模块的正确定位不是“1688 选品平台”，也不是“一键铺货”。它是一条 **只服务 Owner 已批准经营实验** 的受控事实链：

```text
已批准市场与 opportunity gate
→ 登录后的真实 1688 商品页受控采集
→ 原始字节快照与变化/同款判断
→ Owner 复核
→ 供应商、SKU、成本、图片权利、合规、语言和渠道规则证据
→ 内部 product + listing draft
→ Owner 草稿审批（批准后仍是 draft）
→ 如确需发布，再走完全独立的高风险审批与显式执行
```

按这个定义，当前模块的“好”不等于页面多、测试多或能调用平台，而是：任何缺失、未知、模拟或冲突证据都不能被包装成可上架结论；草稿批准不能自动触发真实发布；每个结果都能回到同一市场、实验、原始快照和 Owner 决定。

当前状态：

| 声明 | 裁决 |
|---|---|
| 受控采集、快照、去重/变化、草稿、15 项验收、审批和发布隔离的代码存在 | `implemented` |
| 本次聚焦测试：后端 86 个、前端 5 个通过 | `automated_verified`（本机，2026-07-12） |
| 隔离环境页面与迁移曾被人工验证 | `manually_verified`（历史审计快照，不是本次重验） |
| 当前验收库已存在真实 `sourcing_1688_product` | `unknown`；最近审计明确记录数为 0 |
| 真实 1688 原页、图片使用权、完整费用、合规和渠道规则 | `unknown` |
| 真实渠道已上线或平台返回已确认 | `unknown` |

因此，现在不应继续扩建功能。最有价值的下一步是用 **一个已通过市场闸门的真实商品** 跑一次只到 `approved_draft` 的最小验收；发布继续保持关闭，直到草稿证据全部通过且 Owner 另行决定。

## 2. 模块定义与边界

### 2.1 它负责什么

1. 只允许属于同一 Owner、状态为 `experiment_ready` 的候选市场、active 实验以及 opportunity pass 进入真实页面采集；外部页面调用前就先校验闸门（`controlled_fetch.go:48-77`）。
2. 保存精确原始载荷、SHA-256、采集驱动、解析版本、观察时间、1688 offer 身份、供应商业务身份，并为变化和疑似同款留痕（`workflow_service.go:53-75, 78-106, 109-199`）。
3. 将已复核来源转换为内部商品与待上架草稿，校验供应商、三段 SKU 映射、10 项成本与汇率、图片、合规、本地化、类目及渠道规则。
4. 通过独立草稿审批冻结内容；批准后 `product_listing.status` 仍必须为 `draft`（`lifecycle_service.go:148-202`；`acceptance_report.go:488-508`）。
5. 真实发布是第二条独立高风险流程，使用单独权限、审批、幂等键、冻结请求、显式执行及异常对账（`routes.go:46-50`；`publish_service.go:21-32, 237+`）。

### 2.2 它不负责什么

- 不决定哪个国家、消费者或渠道值得做；市场必须先由 `demandcase` 与 experiment opportunity gate 选定。
- 不因 1688 有商品就创建经营机会，更不能倒推市场。
- 不证明供应商可靠、图片可用、商品合规或成本真实；它只保存和裁决相应证据。
- 不把草稿批准解释为已发布，也不把平台无错误响应解释为已上线。
- 不负责采购、广告、订单、售后、结算或最终利润。
- 不服务外部 SaaS 用户，不建设多租户、批量铺货平台或更多平台连接器。当前 Owner 自用边界见 `docs/SELF_USE_OPERATING_DIRECTION.md:5-25, 60-75`。

## 3. “好”与“不好”的可验证定义

| 维度 | 好 | 不好 / 必须阻断 |
|---|---|---|
| 市场前提 | 明确国家/地区、消费者、渠道、语言；同一 Owner；`experiment_ready`、active experiment、opportunity pass 与对象关联同时成立 | 因采集器可用而先采商品；市场仍 `evidence_missing`；跨 Owner 或跨实验复用来源 |
| 采集真实性 | 只把 `controlled_fetch`、`plugin` 驱动、服务器 `req_` 请求号、同一规范化 1688 URL、非空 `raw_html`、解析版本、观察时间和正确原始哈希同时成立视为真实受控采集 | 手工导入、第三方索引、历史快照或缺字段响应冒充真实 1688 原页 |
| 快照 | 原始响应按原字节存储并计算 SHA-256；同载荷幂等；新观察生成新快照；变化有结构化事件；疑似同款由 Owner 裁决 | 覆盖旧快照；只保存解析结果；哈希由规范化 JSON 计算；变化或同款结论无留痕 |
| 供应商 | 身份、经营年限、成交、MOQ、混批、交期、样品、退换货 8 项均有值、来源和观察时间 | 只凭店铺名、销量或 AI 总结给“可信”标签 |
| SKU | 供应商 SKU → 内部 SKU → 渠道 SKU 一一映射，颜色/尺寸/材质/包装齐全且与持久化 SKU 一致 | 只保留一个模糊规格字符串；变体数量或编码不一致 |
| 成本 | 采购、国内运费、包装、跨境物流、平台费、支付费、广告、税、关税、退货损失 10 项齐全；另有汇率和收入证据；事实等级与来源可复算 | 用商品价代替落地成本；缺项默认为 0；估算写成 actual；费用不属于同一实验 |
| 图片权利 | 每张图绑定 Owner 核验的 `actual` 权利证据、观察时间、处理记录和内容哈希；处理后图片实际可预览、目检 | “来自 1688”就推断可商用；勾选即算有权；只记操作计划未生成图片；未目检即通过 |
| 图片处理 | 真实执行裁切/缩放/白底等操作，保存处理器版本、输入输出哈希、字节、尺寸和渠道规则 | 前端字段写着“已处理”，但无输出字节或哈希；水印、中文、品牌标识未核验 |
| 合规 | 品牌/IP、专利、认证、危险品、材质、标签/说明书 6 项均有 `actual` 证据 | 供应商口头说法或 AI 推断直接通过；关键项 unknown 仍生成可上架结论 |
| 本地化与渠道 | 语言等于已批准市场 `target_locale`；类目属性、变体、图片、配送模板与当前渠道规则均冻结并有来源 | 默认语言/规则沿用其他市场；连接器存在就假设渠道契约可用 |
| 草稿审批 | `editing → pending_approval → approved_draft`；审批冻结内容哈希；审批期间数据库阻止商品、listing、SKU、图片、成本变更；批准后 listing 仍为 draft | 审批对象和实际内容不同；批准时内容可被修改；草稿审批顺带调用平台 |
| 发布隔离 | 单独 `listing.publish` 权限、高风险审批、唯一幂等键、单一未解决链、冻结适配器请求、显式执行、超时后禁止盲重试并要求 actual 对账 | 复用草稿审批；并发创建多条发布链；无错误响应即标记 live；超时直接重试 |
| 最终验收 | 15 项逐项输出 `passed / blocked / unknown`；只有真实 controlled raw 且前 14 项全 pass，第 15 项才 pass | 用模块存在、自动测试、页面可见或 Agent 共识代替真实商品验收 |

15 项标准不是另造的一套指标，而是代码中的统一裁决：目标市场、原始采集、去重/更新、供应商、SKU、完整成本、图片权利、图片处理、合规、本地化、类目规则、生命周期审批、追溯、发布安全、真实商品端到端验收（`acceptance_report.go:139-152`）。第 15 项严格依赖 controlled raw 与前 14 项全 pass（`acceptance_report.go:632-645`）。

## 4. 当前工程证据

### 4.1 已实现且设计合理的部分

- API 按 `product.read`、`product.write` 和独立 `listing.publish` 权限拆分；读、受控写、真实发布的表面清楚（`routes.go:17-50`）。
- 受控采集在浏览器调用前校验市场/实验/Owner；采集器被收窄为只读 `FetchPage`，不能执行任意写工具（`controlled_fetch.go:20-24, 48-77`）。
- 返回结果必须来自 plugin、带 `req_`、标题、供应商身份、原始 HTML、有效 URL/时间/价格/MOQ；失败会留采集失败记录，不会制造空成功（`controlled_fetch.go:114-196`）。
- 服务按原始载荷字节计算 SHA-256，同载荷幂等，新载荷生成新快照，并禁止已转草稿来源再次采集（`workflow_service.go:53-75, 144-152, 187-199`）。
- 数据库把原始载荷改为 `BYTEA`，为 `collection_request_id` 加唯一索引（`migrations/000099_sourcing_real_workflow_hardening.up.sql:4-14`）。
- 草稿审批保存内容 SHA-256；待审批期间数据库触发器冻结 product、listing、SKU、media、cost（`migrations/000097_sourcing_draft_content_approval.up.sql:1-30, 32-83`）。
- 发布请求表冻结市场实验、平台账号、幂等键、请求/响应哈希和状态；数据库触发器阻止执行后的请求和终态结果被改写（`migrations/000091_sourcing_publish_approval.up.sql:1-27, 35-80`）。
- 000099 进一步限制同一来源只能有一条未解决发布链，并新增 `listing.publish` 权限（`migrations/000099_sourcing_real_workflow_hardening.up.sql:16-20, 66-76`）。
- 页面明确提示“草稿批准不会发布”“submitted 不等于真实上线”“reconcile_required 禁止重试”，且处理后图片需要加载预览再目检（`frontend-next/src/app/(main)/sourcing1688/page.tsx:81-90, 439-458, 487-524, 587-592`）。

### 4.2 本次自动验证

| 命令 | 结果 | 能证明什么 | 不能证明什么 |
|---|---|---|---|
| `cd backend-go && go test ./internal/domain/sourcing1688/` | 86 passed | 聚焦领域服务、验证器、受控采集、生命周期、验收报告与发布安全在测试环境通过 | 真实 1688 登录、外部页面字段、生产 PostgreSQL、真实平台账号 |
| `cd frontend-next && npm test -- --run 'src/app/(main)/sourcing1688/page.test.ts' 'src/lib/__tests__/api-client-sourcing1688.test.ts'` | 2 files / 5 tests passed | 草稿/发布/对账 payload 与 API 客户端契约通过 | 完整浏览器交互、视觉质量、真实上传与真实发布 |

上述仅为 `automated_verified`，没有运行全仓测试、生产构建、浏览器 E2E 或生产数据库验证。

## 5. 当前经营证据状态

最近真相审计记录：验收库里的候选市场仍为 `evidence_missing`，没有 opportunity pass；1688 offer 只是第三方公开索引形成的 `pending_detail_collect / unverified` 线索；`sourcing_1688_product` 记录数为 0（`docs/research/project-truth-audit-2026-07-12.md:73-86`）。

据此当前逐项状态应保守裁决为：

| 范围 | 当前状态 | 说明 |
|---|---|---|
| 市场前提 | `blocked` / `unknown` | 已有候选仍 `evidence_missing`，未通过 opportunity gate |
| 原始受控采集 | `unknown` | 没有真实 `sourcing_1688_product`，第三方索引不是 controlled fetch |
| 不可变快照、变化/去重 | `implemented` / `automated_verified`；经营实例 `unknown` | 机制存在，但没有本次真实实例 |
| 供应商与 SKU | `unknown` | 标题、报价和供应商线索仅为 `quoted`，未形成受控草稿证据 |
| 完整成本与汇率 | `unknown` | 真实采购、运费、平台费、税费、退货损失等未取得 |
| 图片权利与实际质量 | `unknown` | 没有 Owner actual 权利凭证及真实处理后目检结果 |
| 合规 | `unknown` | 未取得具体商品 × 目标市场的六项 actual 证据 |
| 草稿审批 | `implemented`；经营实例 `unknown` | 状态机和防篡改存在，没有真实 approved_draft |
| 真实发布 | `unknown`，当前不应执行 | 没有已批准市场、草稿、渠道账号与平台返回证据 |

## 6. 主要风险与不足

### P0：没有真实输入，不能判断模块是否真的好用

当前最大问题不是缺代码，而是零个真实受控商品通过。自动测试能证明防线按预期工作，不能证明浏览器扩展能稳定读取当前 1688 页面、字段口径正确、图片能实际处理、Owner 能看懂 15 项阻断原因。

### P1：`actual` 仍依赖 Owner 对外部凭证的真实性判断

图片权利与六项合规要求为 `actual`，代码会校验证据 URI、时间、Owner 和哈希绑定（`acceptance_report.go:394-449`）；但系统无法仅靠 URI 判断凭证本身是否真实、是否覆盖目标国家/渠道、是否仍在有效期。因此“字段齐全”最多证明审计链完整，不等于法律或平台层面已经确认。

### P1：验收报告的“真实商品端到端”终点只到内部批准草稿

第 15 项文案明确是“到内部批准草稿”（`acceptance_report.go:640-645`），不等于采购成功、平台上线、成交或盈利。报告使用者若只看 `ready/passed` 而忽略边界，仍可能过度解读。当前方向下这个终点设计是合理的，但所有页面和后续报告必须继续保留免责声明。

### P1：平台 `submitted` 不是上线事实

系统已把 `submitted` 与 `succeeded` 分开，并对不明确结果要求 reconcile；这是正确的。不过真实平台适配器、账号权限、线上契约及状态同步仍为 `unknown`。未经 actual 平台页面/API 证据，不能把 submitted 写成上线。

### P2：前端测试覆盖集中在 payload，不等于页面全流程 QA

当前页面测试验证结构化 payload 和风险文案（`page.test.ts:29-82`），但没有证明从真实采集弹窗到 15 项验收、图片预览、审批和异常对账的完整浏览器流程。应在真实商品输入出现后补一次人工浏览器验收，而不是先扩建测试框架。

## 7. 最小真实验证方案（不触发发布）

目标：只验证“一个真实 1688 商品能否安全成为 Owner 批准的内部待上架草稿”，不验证采购、发布或盈利。

前置停止条件：

- 候选市场未达到 `experiment_ready` 或 experiment opportunity gate 未 pass：停止。
- 不是登录后的真实 1688 offer URL，或浏览器扩展无法返回 plugin + `req_` + raw HTML：停止，不用手工导入冒充。
- 无法取得图片使用权、六项合规或关键费用证据：保持 unknown/blocked，不采购、不发布。
- 任何动作可能突破 3,000 CNY 总预算或 1,200 CNY 不可回收损失线：停止并交 Owner 决定。

执行顺序：

1. 选择一个已通过 opportunity gate 的 active 实验，只选一个 1688 offer。
2. 在 Owner 已登录的 1688 商品页执行 `/fetch`，核对 URL、观察时间、raw HTML、`collection_request_id` 与 SHA-256。
3. 再采一次：相同原始载荷应幂等；如价格/MOQ/供应商或变体变化，应生成新快照和变化事件。
4. Owner 复核来源和疑似同款。
5. 逐项补齐供应商 8 项、SKU 三段映射、10 项成本 + 汇率、图片权利与实际处理、合规 6 项、本地化和渠道规则。缺项必须保留 unknown/blocked。
6. 生成内部草稿，确认 listing 仍为 `draft`；提交草稿审批后尝试修改关键内容应被阻止；Owner 审批后再次确认仍未调用平台。
7. 打开 15 项报告：要求每项证据引用可回到来源和时间，只有前 14 项都 pass，第 15 项才 pass。
8. 到此停止。不要创建或执行发布请求；发布另立真实渠道验证任务。

通过标准：一个具体商品的 15 项报告全部 pass，Owner 能实际预览处理后图片，草稿批准后 listing 仍为 draft，过程中没有外部写入。
失败标准：任何关键证据只能靠手工断言、第三方索引、AI 推断或默认 0 才能通过；或草稿审批产生平台调用。
仍然未知：供应商实际履约、商品质量、采购结果、平台上线、买家成交、售后与最终净利润。

## 8. 不扩大范围的建议

1. **现在不开发新功能。** 先完成上述一个真实商品的 `approved_draft` 验收。
2. 不新增平台、批量采集、自动采购、自动发布、供应商评分 Agent、MoA 或仪表盘。
3. 不把第三方索引线索导入 `sourcing_1688_product` 并标成真实；它只能留在 collection evidence。
4. 第一轮真实验证发现的问题，只修阻断当前闭环的字段解析、错误处理、证据展示或状态门；相邻功能记入后续，不进入当前任务。
5. 若 15 项全部通过，再由 Owner 单独决定是否开启一个渠道的发布验证；该决定不能由本报告或测试结果代替。

## 9. 证据索引

- 当前方向：`docs/SELF_USE_OPERATING_DIRECTION.md:5-25, 60-75, 77-111`
- 最新事实状态：`docs/research/project-truth-audit-2026-07-12.md:48-86`
- API 与权限：`backend-go/internal/domain/sourcing1688/routes.go:11-51`
- 受控采集：`backend-go/internal/domain/sourcing1688/controlled_fetch.go:18-196`
- 快照与工作流门：`backend-go/internal/domain/sourcing1688/workflow_service.go:53-202`
- 草稿审批：`backend-go/internal/domain/sourcing1688/lifecycle_service.go:148-202, 208+`
- 15 项验收：`backend-go/internal/domain/sourcing1688/acceptance_report.go:139-149, 155-645`
- 发布隔离：`backend-go/internal/domain/sourcing1688/publish_service.go:21-32, 77-103, 194-235, 237+`
- 前端 Owner 页面：`frontend-next/src/app/(main)/sourcing1688/page.tsx:81-90, 197-237, 439-592`
- 迁移级约束：`backend-go/migrations/000091_sourcing_publish_approval.up.sql:1-80`、`000097_sourcing_draft_content_approval.up.sql:1-83`、`000099_sourcing_real_workflow_hardening.up.sql:1-76`
- 前端聚焦测试：`frontend-next/src/app/(main)/sourcing1688/page.test.ts:29-82`
