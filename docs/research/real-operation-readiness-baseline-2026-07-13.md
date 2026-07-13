# 凌镜真实经营就绪基线

> 日期：2026-07-13
> 对应计划：`docs/plan/REAL_OPERATION_READINESS_PLAN.md` 的 P0-1
> 结论：P0-1 清单已固定；最早未通过门槛是 P0-2 正式运行底座
> 证据边界：本文件只确认当前入口、工程证据和现场状态，不宣称任何真实经营结果已经发生

## 一句话结论

凌镜目前不是“没有底层数据系统”，而是已经有多条按 Owner 和经营对象保存事实的工程链，但正式服务器仍运行旧版本，备份恢复、当前版本部署、真实账号权限和真实外部回执尚未验收。因此现在不应再建设一个抽象的数据中心；应先把当前权威链部署到可恢复的正式环境，再从一条真实经营路径产生事实。

## 证据说明

- `actual`：本次只读检查直接观察到的当前状态；
- `implemented / automated_verified`：代码和自动测试已经证明的工程能力；
- `manually_verified`：在明确环境中人工执行并观察到的结果；
- `mock`：明确为模拟数据或模拟外部系统；
- `unknown`：当前没有足够证据，不能判断为已完成。

路由或页面存在只说明入口存在，不等于真实业务已经发生，也不等于生产可用。

## 正式环境现场基线

2026-07-13 14:35（Asia/Shanghai）按统一部署手册执行只读检查，未修改服务器、数据库或外部平台。

| 检查项 | 当前事实 | 证据等级 | 处置 |
|---|---|---|---|
| 服务器 | `lingmirror` 可通过既有 SSH 配置连接；Ubuntu 24.04，主机已运行约 1 天 21 小时 | `actual / manually_verified` | 保留，不重装 |
| 网络边界 | UFW 仅放行 22、80、443；Docker 未把 3000、5432、8080 暴露到公网 | `actual / manually_verified` | P0-2 复核云安全组和外部端口 |
| 运行容器 | frontend、backend、db、caddy 共 4 个容器在运行；前三者健康 | `actual / manually_verified` | 不能据此认定当前版本已部署 |
| 健康接口 | 容器内 `/api/health` 返回 alive；`/api/ready` 显示 database、event_bus、scheduler、traffic 为 true | `actual / manually_verified` | 部署当前版本后重验 |
| HTTPS | Caddy 使用 `Caddyfile.ip`；服务器本机经 HTTPS 访问健康接口返回 200 | `actual / manually_verified` | 公网证书信任和 Owner 浏览器访问仍为 `unknown` |
| 代码版本 | 已形成并推送干净发布候选 `codex/real-operation-readiness@9d6d1013`；服务器随后被另一任务切到 `codex/deploy-plugin-minimal-20260713@46319de8`，容器仍运行旧镜像 | `actual` | **停止并发覆盖部署**；待当前生产任务完成或明确交接后再部署 exact commit |
| 应用版本 | 服务器健康接口报告 `0.2.1`，低于仓库当前 `v0.3.0.0` 口径 | `actual` | P0-2 部署确定提交后统一版本声明 |
| 备份与恢复 | 服务器已有同日 pre-plugin 备份；本次新建 `multisell_2026-07-13_144223.dump` 及 SHA-256，权限 600；已恢复到隔离临时库并验证 179 张表和迁移版本 100，随后删除临时库，正式服务仍 ready；服务器外副本已保存到 Owner 本机 `/Users/lc/Backups/lingmirror/`，校验值匹配且 `pg_restore --list` 通过 | `actual / manually_verified` | 手工校验文件记录了服务器相对路径，已通过直接比较哈希绕开；后续应只用脚本生成可移植校验文件 |
| 监控与告警 | 当前运行容器中未见监控栈；告警投递未验证 | `actual / unknown` | P0-2 启动并验证一次真实告警投递 |
| Owner 登录 | 本次未使用或检查 Owner 凭据 | `unknown` | 部署后由 Owner 完成登录验收，不输出秘密值 |
| 外部账号与凭据 | 真实渠道、1688、承运商、结算、银行及付费 Provider 可用性未检查 | `unknown` | 到对应门槛时只检查权限和受控调用；真实写入须 exact Owner 批准 |

## 完整经营路径与唯一权威入口

“唯一权威入口”指本轮真实经营应使用的事实或动作入口。旧页面、兼容 CRUD、报表和 mock 接口不作为事实来源。

| 经营步骤 | Owner 入口与权威 API | 当前工程证据 | 当前真实状态 | 真正阻塞 | 下一验收动作 | 责任 |
|---|---|---|---|---|---|---|
| 0. 平台真相 | `/platform-truth`；`GET /api/v1/platform-truth` | `implemented / automated_verified` | 正式环境版本不是当前代码 | 当前合同尚未部署到正式环境 | 部署确定提交后检查全部领域均被分类 | AI |
| 1. 市场研究与选择 | `/demand-cases`；`/api/v1/demand-cases`、`/:id/owner-decisions` | `implemented / automated_verified` | 没有当前真实研究和 Owner 选择 | 国家×消费者×场景×渠道及八维证据缺失 | 用带来源、观察时间和独立反证的真实材料形成一个 Owner 决定 | AI 调查，Owner 决定 |
| 2. 商品机会 | `/demand-cases`；`/api/v1/product-opportunities`、`/:id/owner-decisions` | `implemented / automated_verified` | 没有真实已批准机会 | 必须先有最新 `selected` 市场决定 | 建立一个真实机会，完成反证、unknown、停止线和 Owner 批准 | AI 准备，Owner 决定 |
| 3. 1688 货源 | `/sourcing1688` 与配对插件；`/api/v1/extension/sourcing-1688/private-collections`、`/api/v1/sourcing-1688/:id/task-links`、`/fetch` | `implemented / automated_verified` | 无真实页面、供应商、样品或重复观察 | 插件、登录页面、真实商品和前两步授权缺失 | 采集一个真实页面，验证原始快照、断网恢复、供应商身份和变化观察 | Owner 操作，AI 核验 |
| 4. SKU、成本、合规与草稿 | `/sourcing1688`；task-link 下 SKU、成本、合规、素材、草稿和审批入口 | `implemented / automated_verified` | 关键经营字段均无真实证据 | 真实供应商声明、精确费用、合规和素材权利缺失 | 完成一个 canonical SKU、可复算成本版本、当前合规证据和 Owner 批准草稿 | AI 整理，Owner 审批 |
| 5. 商品图片 | `/product-images`；`/api/v1/product-images` 的任务、反馈、冻结图片集和 release attestation | `implemented / automated_verified`；真实 Provider 调用 `unknown` | 没有真实可发布图片集 | Provider、预算、权利、沙箱和发布证明未全部现场验收 | 先做沙箱/人工导入验收；付费调用前冻结预算和停止条件 | AI，付费时 Owner 批准 |
| 6. 采购 | `/purchase`；`/api/v1/purchase/authorities` 及 Owner approval、external submissions、order/failure receipts、receiving events | `implemented / automated_verified` | 无真实供应商提交、采购单或收货 | 真实供应商、准确成本、库存和 exact Owner 决定缺失 | 先验证请求哈希与审批；产生资金或下单前由 Owner 批准 exact 输入 | AI 准备，Owner 批准 |
| 7. 渠道发布 | `/sourcing1688`；task-link 下 publish request、decision、execute、reconcile、terminal observation | `implemented / automated_verified` | 无真实渠道发布 | 真实账号、权限、当前市场资格和正式环境未验证 | 使用 sandbox/只读权限先验；真实发布必须独立批准且结果未知时停止重试 | AI 准备，Owner 批准 |
| 8. 订单与库存 | `/orders`、`/inventory`；`POST /api/v1/platform-integrations/:id/order-events` 形成订单事实与库存 ledger | `implemented / automated_verified` | 仅有 mock 跨层验收，无真实订单 | 真实渠道事件、签名/账号级连接器和 SKU 映射未现场验证 | 摄取一条受控真实订单事件并核对同 Owner、账号、币种、SKU 和库存账本 | AI，外部事件由平台产生 |
| 9. 履约 | 订单/履约页面；`POST /api/v1/supply-chain/tracking/:id/carrier-events` | `implemented / automated_verified` | 无真实承运商事件或签收 | 真实承运商连接器和订单关联缺失 | 摄取不可变外部事件；只有 `external_observed delivered` 可形成真实签收 | AI，外部事件由承运商产生 |
| 10. 售后 | `/aftersales`；`/api/v1/aftersales/resolutions` 的请求、决定、执行和回执 | `implemented / automated_verified` | 无真实售后终局 | 真实请求、Owner 决定和平台回执缺失 | 对真实订单完成一次无售后观察期，或完成一笔售后至可信终局 | AI 准备，Owner 决定 |
| 11. 平台结算 | `/settlement`；`POST /api/v1/settlement/platform-accounts/:id/events` | `implemented / automated_verified` | 无真实结算 | 真实平台结算文件/事件和同订单绑定缺失 | 摄取一条原始结算事件并核对事件哈希、账号、订单、币种和明细 | AI，外部事件由平台产生 |
| 12. 最终利润 | `/profit`；`POST /api/v1/profit/order/:orderId/finalize` 与 final versions | `implemented / automated_verified` | 无真实最终利润 | 成本、履约、平台费、售后或结算任一缺失都会阻断 | 对同一订单从权威明细生成不可变利润版本并人工复算 | AI |
| 13. 现金到账 | `/finance`；`/api/v1/finance/cash-receipts`、`/cash-reconciliations` | `implemented / automated_verified` | 无真实银行/支付到账 | 银行凭证和平台净应收尚不存在 | 保存一条真实到账凭证；只在同对象、同币种、金额精确匹配时对账 | Owner 提供事实，AI 对账 |
| 14. 认识与下一行动 | `/business-decisions`、`/business-feedback`、`/xiaoq`；案卷→建议→Owner决定→行动→观察→下一建议 | `implemented / automated_verified`，但当前结果值仍可由调用方提交 | 无真实行动和后续结果；因果 `not_established` | 结果未由同一对象权威字段自动计算，观察期限也未形成完整门禁 | 先补同对象、执行后、冻结期限和系统计算；再完成两轮真实循环 | AI 建议，Owner 决定 |
| 15. 运行与恢复 | 唯一入口 `docs/ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md` | 手册和脚本 `implemented`；当前旧版本服务可运行；本地备份恢复已 `manually_verified` | 当前提交未部署，服务器外备份与告警未验证 | 干净发布点、外部备份、监控、Owner 登录 | 先完成 P0-2，再允许真实凭据或真实写入进入系统 | AI，重大外部动作 Owner 批准 |

## 并存入口的处置边界

这些入口可以保留兼容，但本轮不能被当作权威事实链：

| 并存入口 | 当前判断 | 本轮规则 |
|---|---|---|
| `/api/v1/purchase/orders` 写接口 | 已固定返回 Gone | 保留失败关闭测试，不再作为采购入口 |
| `/api/v1/listings/:id/publish`、`/api/v1/listing/.../publish` | 当前返回需要 image release attestation，不执行旧发布 | 保留失败关闭测试；真实发布只走受控发布尝试 |
| `/api/v1/aftersales/:id/refund` 与 disputes | 直接退款和旧争议接口已失败关闭 | 售后真实终局只走 `/aftersales/resolutions` |
| `/api/v1/aftersales` 旧 Create/Update/Delete/Approve/Reject/Receive | 兼容 CRUD 仍可达，但不形成可信外部退款事实 | P0-2 前确认 Owner 页面不把它们展示为权威终局；必要时进一步冻结写入 |
| `/api/v1/settlement` 旧 CRUD、reconcile、items | 与新的平台事实入口并存 | 真实结算只认 platform account event；兼容记录不得进入最终利润 |
| `/owner` 与 `/api/v1/owner/*` | 页面混合 mock 展示、真实审批和假成功文案 | 本地已将页面跳转到 `/platform-truth`，后端入口仅在 `development` 注册，`acceptance` 与 `production` 失败关闭；尚未部署，生产状态仍需复核 |
| `/api/v1/finance/mock` | 明确 mock，且会写入权威财务 ledger 表 | 本地已从路由注册与安全目录移除；尚未部署，部署前不得宣称生产已关闭 |
| `/api/v1/platform-integrations/publish-to-ozon`、`/write-back` 和 retry | 旧兼容写入会生成 sandbox/dry-run 假成功，并可能触发外部 adapter | 本地已从路由注册与安全目录移除；尚未部署，真实发布继续只走受控 task-link 发布链 |
| `/api/v1/platform-integrations/mock/seed` 和 mock adapters | 明确 mock | 本地已限制为 `development` 注册，`acceptance` 与 `production` 失败关闭；尚未部署，正式环境仍需用运行时路由表复核 |
| `/api/v1/platform-integrations/publish-to-ozon`、`write-back` | 不是当前受控发布权威入口 | 本地已从路由移除；部署后复核运行时路由表 |
| tracking 的手工 status 更新 | 只能是内部状态 | 不得生成 `actual_delivery`；真实签收只认承运商事件 |
| `docs/api-inventory.md` 和旧模块目录中的 ✅ | 只表示曾发现路由 | 不作为可用性、真实性或当前权威入口证据 |

## P0-1 通过判断

P0-1 已通过，理由是：

1. ADR-001 的必要经营步骤均已映射到一个本轮权威入口；
2. 每一步均记录了工程证据、真实状态、阻塞、下一验收动作和责任；
3. mock、兼容路径、手工状态和外部回执已分开；
4. 正式环境、版本、备份、监控、Owner 登录和外部权限的当前状态已明确记录为事实或 unknown；
5. 没有用模块数量、页面存在或测试通过替代真实经营完成。

## 当前唯一下一步

进入 P0-2，但第一动作不是直接部署：

1. 等待或接管当前 `codex/deploy-plugin-minimal-20260713` 生产任务，禁止两个任务同时切换服务器代码；
2. 以已验证发布候选 `9d6d1013` 为基准，确认是否需要吸收当前插件分支的独有改动；
3. 部署 exact commit，并将数据库从当前迁移版本 100 升级到候选版本 151；
4. 部署当前版本后验证 HTTPS、Owner 登录、RBAC、审计、迁移、监控、告警和恢复；
5. 在 P0-2 通过前，不把真实渠道、供应商、银行或付费 Provider 凭据接入正式写路径。

## 主要证据来源

- `docs/research/project-truth-audit-2026-07-13.md`
- `docs/research/product-supply-channel-gap-audit-2026-07-12.md`
- `docs/research/order-inventory-fulfillment-aftersales-gap-audit-2026-07-12.md`
- `docs/research/settlement-profit-cash-gap-audit-2026-07-12.md`
- `docs/research/business-decision-feedback-gap-audit-2026-07-12.md`
- `docs/research/xiaoq-owner-collaboration-audit-2026-07-12.md`
- 当前工作树的领域路由、服务和 mutation policy
- 2026-07-13 对 `lingmirror` 的只读 SSH、容器、健康接口、防火墙、版本和备份目录检查
