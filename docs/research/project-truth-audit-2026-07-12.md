# 凌镜方向事实审计

> 审计日期：2026-07-12
> 审计范围：Owner 产品方向与现行文档一致性
> 状态：当前方向快照；不覆盖 2026-07-11 工程事实审计
> 最终方向：以文末 Owner 最终裁决及 [ADR-001](../decisions/ADR-001-owner-complete-commerce-platform.md) 为准；中间历史裁决保留用于追溯。

## 当前裁决

凌镜唯一开发路径是建设只供 Owner 本人使用的完整 AI 跨境电商经营平台。完整平台是目的地，按完整纵向单元推进；不服务外部软件用户，不安排客户访谈、设计伙伴、软件试点、SaaS、订阅、计费或软件商业化。

商品消费者仍然是 Owner 自营业务中的交易对手。对消费者购买、付款、签收、售后和最终利润的核验用于确认经营事实及经济结果，不得自动解释为因果实验或凌镜的软件需求验证。

## 证据等级

| 声明 | 等级 |
|---|---|
| Owner 明确凌镜只供本人使用、不会服务外部软件用户 | `policy` |
| 完整 Owner 自用跨境电商平台已确认为唯一开发路径 | `policy`；核心方向文档已同步 |
| 旧研究中关于 SaaS、设计伙伴或外部软件付费的内容 | `superseded` |
| 2026-07-11 审计记录的代码与测试结果 | 历史快照，本次未重跑 |
| 真实商品成交、售后闭合和最终净利润 | `unknown`，本次未连接外部经营数据 |

## 历史事实核验路径（不是唯一开发路径）

以下链条保留为历史事实核验路径，不构成完整平台架构、经营反馈或因果实验：

```text
候选市场比较
→ Owner 批准已选市场
→ 商品经营证据与反证
→ 商品机会
→ Owner 批准的经营行动
→ 非关联买家付款与签收
→ 售后与争议窗口关闭
→ 最终贡献利润与现金对账
→ 停止、换品、修正后再试或小幅加码
```

## 文档边界

- 当前产品边界以 `docs/SELF_USE_OPERATING_DIRECTION.md` 为准。
- 旧材料中的“真实付费需求”仅可解释为商品消费者真实付款，不代表外部软件需求。
- 旧材料中的 SaaS、外部客户、设计伙伴和软件试点路线不得进入当前任务队列。
- 真实商品成交或 Owner 自用效果不会自动改变产品边界；只有 Owner 新的明确决策才能改变方向。

## 本次未声明的事实

本次只修改和核对方向文档，没有重新运行代码测试，没有连接生产服务器、平台账号、订单、结算或银行数据。因此，本审计不提高任何工程或经营事实的证据等级。

## 补充审计：1688 受控草稿闭环（2026-07-12 12:06 CST）

本补充审计不改写上面的方向裁决，只记录其后的工程事实变化。

| 声明 | 等级 |
|---|---|
| 1688 受控采集、不可变快照、去重/变化、供应商与 SKU、完整成本、图片权利与实际处理、合规、本地化、类目规则、状态审批、实验追溯和独立发布审批已有代码与数据库约束 | `implemented` |
| 15 项逐项验收报告、可信 `controlled_fetch` 来源、草稿内容审批 SHA-256 与待审批防篡改触发器 | `implemented` |
| 后端全量 118 包 3080 个测试、Go build/vet、前端 24 文件 129 个测试与 91 页面生产构建 | `automated_verified` |
| 隔离 PostgreSQL 数据库迁移至 000097、服务 `/api/health` 与 `/api/ready`、Owner 鉴权空库读取、浏览器页面/受控采集弹窗/发布保护 | `manually_verified`（本地隔离环境） |
| 一个真实 1688 商品从真实页面采集到真实渠道草稿及平台返回结果 | `unknown`；未提供 Owner 已批准的市场、实验、真实 1688 URL/登录态和渠道凭据，本次未执行外部写入 |

## 补充审计：1688 真实工作流加固（2026-07-12 15:00 CST）

本节记录代码审查后的新增工程证据，不把自动验证升级为真实经营证据。

| 事实 | 证据等级 |
|---|---|
| 原始响应以 `BYTEA` 保存并按原字节计算 SHA-256；快照插入后不再补写指纹 | `automated_verified`：隔离 PostgreSQL 从零迁移后，`TestPostgresCaptureRespectsImmutableSnapshot` 验证原始字节、摘要与不可变触发器 |
| 000099 迁移新增采集请求唯一约束、单一未解决发布链、`listing.publish` 权限、市场语言和审批冻结修正 | `automated_verified`：隔离 PostgreSQL 上完成迁移至 000099、回退 1 次、重新上行 1 次并核对列、索引与权限 |
| 扩展真实字段名可被后端解析；扩展采集错误会立即返回；成功响应绑定服务器生成的 `collection_request_id` | `automated_verified`：ToolBridge、实时协议与受控采集聚焦测试含 race 检测通过 |
| 同款判断规范化变体顺序并识别标题轻微变化，且匹配与处置按 Owner 隔离 | `automated_verified`：sourcing1688 聚焦测试含 race 检测通过 |
| 供应商证据含事实值，草稿语言绑定已批准市场 `target_locale`，处理后图片通过受保护读取在页面实际预览 | `implemented`；后端契约测试、前端 18 个聚焦测试和 Next.js 生产构建通过，但图片视觉质量仍需真实商品人工确认 |
| 目标国家/地区、消费者、渠道、语言和真实 1688 商品的端到端经营验收 | `unknown`：本次仍未获得 Owner 批准的市场组合、真实 URL/登录态、图片权利、费用证据及渠道账号，不得宣称真实闭环完成 |

## 补充审计：Owner 授权 Agent 自选并录入候选（2026-07-12 15:44 CST）

Owner 明确要求 Agent 对可逆测试输入自行选择并录入。该授权允许建立候选与线索，不代表 Owner 已批准 opportunity gate，也不允许把公开索引升级为真实 1688 原页证据。

| 事实 | 证据等级 |
|---|---|
| 验收库 `multisell_acceptance_20260712` 已在备份后从迁移 000097 升至 000099 | `actual`；备份位于 `/tmp/multisell_acceptance_before_agent_input_20260712_154415.dump`，迁移状态为 `99 / dirty=false` |
| 已录入“美国 × Amazon US 日常猫毛清洁工具购买者 × en-US”候选市场，`demand_case.id=1` | `actual`；状态保持 `evidence_missing`，未生成 opportunity pass |
| 候选市场关联 3 个独立研究快照、9 条 support/counter/conflict 证据和 1 条 evidence_missing 裁决 | `actual`（数据库记录）；公开来源内容为 `quoted`，账号、费用、履约、售后和利润仍为 `unknown` |
| 已录入 1688 offer `692570310190` 为采集线索，关联 `collection_evidence.id=1` | `actual`（录入发生）；线索状态为 `pending_detail_collect / unverified`，证据哈希复算一致 |
| 商品标题、报价、MOQ 和供应商来自第三方公开索引 | `quoted`；来源为 `https://www.1688wholesale.com/zh-CHS/1688/china_alibaba_item/692570310190.html`，不得冒充 1688 原页受控采集 |
| `sourcing_1688_product` 真实受控商品记录、图片权利、完整成本、合规、SKU 与批准草稿 | `unknown`；当前记录数为 0，必须在登录后的真实 1688 页面通过 Owner 浏览器扩展采集后继续 |

自动测试、页面可见和隔离数据库就绪只证明工程链可运行，不能替代真实商品、图片使用权、真实成本、平台规则或真实发布结果的外部观察。

## 纠偏审计：现有 `experiment` 不构成经营闭环（2026-07-12）

Owner 明确否定“候选市场 → 选择商品 → 小额销售 → 付款/签收/退货/利润”属于经营闭环。该序列只是语言上首尾完整的交易与核验路径，工程上缺少目标控制、可执行变量、真实市场作用、可靠观测、偏差计算、反馈规则及下一轮执行。

| 声明 | 等级 |
|---|---|
| `experiment` 模块、状态机、对象关联、证据闸门和终局校验存在 | `implemented / automated_verified`（沿用已有工程证据） |
| 该模块能够追踪经营事实并阻止部分错误事实升级 | `inferred`；仍需真实 Owner 使用验证价值 |
| `experiment_id` 关联订单、售后、结算、利润和现金可证明因果关系 | 不成立；关联不等于因果 |
| 状态机走到终局或最终利润完成对账可证明经营反馈闭环 | 不成立；流程终止不等于反馈闭合 |
| 当前已实现工程意义上的经营反馈循环 | `not implemented` |

自本节起，当前文档中的“经营实验模块”统一按“经营事实核验案卷”理解；旧文档中的“经营实验闭环”“业务闭环”和“真实闭环完成”属于 `superseded` 表述。不得围绕该错误模型继续扩建。代码是否重命名或重构不在本次文档修改授权范围内。

## 补充方向裁决：1688私人采集箱（2026-07-12）

Owner明确纠正此前“采集前必须关联经营实验”的产品路径。新的产品顺序为：1688页面主动点击 → 保存到Owner私人采集箱 → 检查整理 → 决定继续研究时关联选品任务 → 通过经营闸门后进入现有受控草稿流程。

| 声明 | 等级 |
|---|---|
| Owner允许1688商品在未关联选品任务时先保存为私人收藏 | `policy` |
| 私人收藏固定为`unverified_lead`，页面字段最高为`quoted`，不构成商品机会或经营证据 | `policy` |
| 经营闸门从“采集前”移动到“进入待研究商品/草稿前” | `policy` |
| 私人采集箱、主动点击接口、状态机和插件新交互已经实现 | `planned`；截至本补充裁决尚未修改代码或重新验证 |
| 当前旧插件、旧自动采集路径和现有数据库是否符合新路径 | `unknown`；必须在实施前完成代码与模型审计 |

## Owner 最终方向裁决：完整平台是唯一开发路径（2026-07-12）

Owner 明确确认：凌镜的开发目标不是一个小工具、单一行动卡或孤立的“最小闭环”，而是只供 Owner 本人使用的完整 AI 跨境电商经营平台。AI 降低工程实现成本，因此不得以单人、代码量或传统团队周期主动缩小已经确认的平台目标。

| 声明 | 等级 |
|---|---|
| 完整 Owner 自用跨境电商平台是唯一开发路径 | `policy`；Owner 明确确认 |
| 平台按完整纵向单元推进，小单元不是产品上限 | `policy` |
| 经营事实系统与经营决策系统必须分开建模并在平台内协作 | `policy`；目标架构尚为 `planned` |
| 现有代码已经构成上述完整平台 | `unknown`；必须按新架构重新映射，不能沿用旧完成声明 |
| 现有 `experiment` 已成为真实反馈或因果系统 | 不成立；前述纠偏继续有效 |
| 外部 SaaS、多租户、订阅、计费和公共 API 因“完整平台”而解冻 | 不成立；仍为 `superseded / frozen` |

唯一权威决策记录为 `docs/decisions/ADR-001-owner-complete-commerce-platform.md`。后续计划、TODO、PR、QA 和发布必须映射到该路径；无法映射的工作不得进入开发队列。

## 1688浏览器采集重构的新增工程证据（同日后续）

- `implemented / automated_verified`：Owner私人收藏、设备绑定插件凭证、页面预览确认、提交前关键摘要复读、URL与页面商品ID一致性、`sourcing1688.private.v1`、原始/结构/请求信封三类哈希、跨重启request_id对账、私人工作副本、多个选品任务关联展示和草稿转换幂等已进入当前代码并有聚焦自动测试。
- `implemented / not externally_verified`：插件页面字段仍只是1688页面声明，最高为`quoted`；以上工程能力不证明供应商身份、价格、SKU、图片权利或可成交性真实。
- `unknown`：迁移000101—000104尚未在真实PostgreSQL执行；真实登录Chrome与真实1688页面的逐字段、重复商品、下架/验证码、断网/重启验收尚未完成。
- `incomplete`：私人采集失败记录、明确的`reconcile_required`持久状态、接口级载荷限制、每个任务独立草稿工作流和完整采集箱状态机仍在开发，不得将插件称为完成或生产可用。

## 1688浏览器采集重构补充工程审计（2026-07-12 21:00 CST）

本节取代上一节对工程缺口的旧快照，但不改变真实1688页面与经营事实仍为`unknown`的结论。

| 声明 | 证据等级 |
|---|---|
| 私人采集失败记录、Owner隔离、有限错误码、持久化`receiving / saved / not_saved / reconcile_required`请求收据已实现 | `implemented / automated_verified`；sourcing1688、auth和routecatalog聚焦测试通过 |
| 插件区分服务端明确未保存与仅404无收据；404不再被升级为确定未保存 | `implemented / automated_verified`；插件对账测试覆盖保存、明确未保存、无收据和断网 |
| SKU解析失败只留下诊断记录，不再生成终局`not_saved`收据，仍允许按`parse_failed`保存私人收藏 | `implemented / automated_verified`；后端回归测试覆盖终局收据数量与SKU无阻断 |
| 私人采集不再克隆或上传整页DOM；原始证据限于固定结构化payload及三类哈希 | `implemented / automated_verified`；DOM fixture明确断言无`raw_html` |
| 插件已识别非商品页、未登录、验证码/风控、下架占位、页面加载中和SKU未稳定，并在初次读取及确认提交前失败关闭 | `implemented / automated_verified`；6类脱敏fixture及预览后页面变化回归测试 |
| 采集箱显示六类关键字段完整度、多次观察数、任务关联数、状态筛选，并支持未进入事实链的私人收藏归档/恢复 | `implemented / automated_verified`；前端8项聚焦测试、Next生产构建95页面通过 |
| 同一Owner商品的每个task link拥有独立草稿、乐观锁编辑、草稿审批、发布审批、执行与对账状态；旧商品级入口只投影主任务 | `implemented / automated_verified`；迁移000112、000118，sourcing1688聚焦测试与前端任务级测试通过；真实渠道执行仍为unknown |
| 重复商品409只返回Owner隔离安全摘要，插件逐项显示本次与已有标题、价格、MOQ、供应商、SKU数、图片数和观察时间差异 | `implemented / automated_verified`；后端Owner隔离/泄漏测试和content-script DOM测试覆盖 |
| 迁移000001—000120可在本机隔离真实PostgreSQL完成全量上行、最新回退/上行、全量回退及再次全量上行 | `manually_verified`（隔离数据库）；最终版本120，未触碰现有`multisell`库 |
| 插件TypeScript与自动测试 | `automated_verified`；32项测试通过 |
| `/sourcing1688`前端聚焦测试与生产构建 | `automated_verified`；16项聚焦测试、95页面Next生产构建通过 |
| 后端全量测试 | `automated_verified`；当前工作树121个包共3259项测试通过，Go build与vet通过 |
| 真实登录Chrome中的真实1688商品逐字段采集、插件侧载、重复观察和断网恢复 | `unknown`；浏览器安全策略阻止Agent直接导航该详情页，本次没有取得外部页面证据 |
| 真实供应商身份、可成交价格、SKU/库存、图片权利和平台规则 | `unknown`；页面声明即使采集成功也最高为`quoted` |

## 补充审计：市场机会权威链与货源权限收口（2026-07-12）

本节记录 ADR-001 第 2 单元完成后的工程证据，不提升真实市场或真实经营事实等级。

| 声明 | 等级 |
|---|---|
| 2—4 个候选可按八维证据、反证、冲突、unknown 和 Owner 决定同框比较 | `implemented / automated_verified` |
| 系统评估、Owner 市场决定、商品机会及 Owner 机会批准已分离；新增证据使旧 verdict 失效 | `implemented / automated_verified` |
| 研究批次、快照、案件、Owner、run、来源和观察时间由 PostgreSQL 外键/触发器约束，快照追加后不可改删 | `automated_verified`：真实 PostgreSQL 专项约束测试通过 |
| 1688 受控采集、复核、草稿、验收、小Q只读和发布全边界使用冻结商品机会批准授权；experiment 只追踪 | `implemented / automated_verified` |
| 全新临时 PostgreSQL 完成 108 对迁移全上行、全回退和再次上行至版本 111 | `automated_verified`；临时库已删除，未触碰现有数据 |
| 后端全量 121 包 3207 个测试、Go build、相关 vet、407 条 mutation 路由安全清单 | `automated_verified` |
| 前端候选比较测试、定向 ESLint 与 95 页面 Next.js 生产构建 | `automated_verified` |
| 真实市场选择、真实商品机会、真实渠道权限、消费者付款及利润 | `unknown`；本次没有连接外部经营事实 |
