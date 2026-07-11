# 需求数据可得性与自动化现实审计

> 审计日期：2026-07-11
> 范围：Ozon、Shopee、Shopify；官方开发者文档、官方卖家帮助、卖家后台/导出与公开页面；结合当前 LingMirror 仓库只读审计
> 核心问题：在尚未选市场、类目和商品时，AI 能否自动发现真实付费需求？

## 一、结论先行

**不能。** 在尚未确定国家站点、目标人群、具体需求、商品规格、售价和履约方案时，AI 最多能自动生成并排序一批**待验证需求假设**，不能发现已经被证明的“真实付费需求”。

原因不是 AI 不够聪明，而是平台可取得的数据有天然边界：

1. **公开页面**能观察当前商品、标价、促销、评分、评论、配送承诺和搜索结果，但通常看不到真实曝光、搜索人数、加购、成交、取消、退货、广告成本和卖家最终到账。
2. **卖家 API、后台和导出**主要返回“这个已授权店铺、这个已存在 listing、这些已发生订单”的第一方数据，不是全市场任意商品的真实销量数据库。
3. **平台趋势、热门查询、热销榜和缺失商品工具**比公开页面强，但仍是聚合或滞后信号；它们不能证明新卖家以自己的规格、价格、交付时效和广告预算也能成交。
4. **Shopify 是自有店基础设施，不自带 marketplace 需求。** 没有可验证流量时，零访问、零加购、零订单只证明没有流量，不能证明商品无人愿意购买。
5. **付费意愿是干预后的结果。** 必须把一个具体报价真实展示给目标买家，才知道是否有人点击、加购、付款、签收，并在退货与结算后留下正的最终净利润。

因此应把证据分成四级：

| 等级 | 能回答什么 | 不能回答什么 |
|---|---|---|
| 公开市场观察 | 有哪些替代品、价格、评论问题、供给密度 | 真实搜索量、真实销量、新卖家可获得流量 |
| 平台聚合分析 | 哪些查询/类目/商品近期受关注或成交 | Owner 的具体 SKU 会不会卖、会花多少广告费 |
| 自己店铺的 listing/流量/订单 | 自己的报价获得了多少曝光、点击、加购、下单 | 订单能否签收、是否退货、最终是否赚钱 |
| 完整真实实验 | 有效成交、拒收退货、结算到账、最终净利润 | 下一批一定复现；仍需重复实验 |

**经营含义：** LingMirror 可以自动化“发现—抓取—留证—去重—排序—刷新—提醒”，但不能把模型推断、公开页热度或平台榜单标记成“已验证付费需求”。采购、发布、广告、补货和资金动作仍需 Owner 批准。

## 二、数据源矩阵

标记说明：

- `无销售`：不要求已有订单；可能仍要求卖家账号或后台权限。
- `需 listing`：需要自己店铺已有商品或 SKU，数据才有意义。
- `需订单/流量`：只有真实访问、广告、订单或结算发生后才产生。
- `付费/特权`：平台计划、Premium、受保护数据审批或单独权限会扩大范围。

### 2.1 Ozon

| 数据源 | 可取得的数据 | 前置条件 | 对需求判断的价值 | 关键限制 |
|---|---|---|---|---|
| Ozon 消费者公开搜索/商品页 | 商品标题、当前/活动价、规格、评分、评论数及内容、卖家/配送承诺、可售状态；固定地区下的搜索结果样本 | 无需卖家销售；网页可能要求地区、登录、验证码或浏览器会话 | 竞品、价格带、评论痛点、供给密度的观察证据 | 看不到真实曝光、搜索人数、加购、订单、取消、退货、广告成本；排序受城市、个性化、促销和库存影响。官方说明同一商品可能因地区库存不可用，[商品库存可用性](https://docs.ozon.com/common/en/tovary/harakteristiki-i-nalichie/) |
| 卖家后台“分析工具/在 Ozon 卖什么” | 趋势、热门商品、搜索查询、缺失商品、缺货订阅、类目与竞争等平台聚合信号 | 需可用卖家账号；具体模块、国家和账号是否开放必须登录实证；通常不要求自己已有订单 | 形成类目和需求假设的最强平台内入口之一 | 聚合、时间窗和口径受平台定义约束；热门不等于可进入、更不等于利润。[Ozon 官方分析工具](https://docs.ozon.com/global/en/analytics/analytics-and-metrics/analytics-tools/) |
| “我的商品查询”及 Seller API `/v1/analytics/product-queries*` | 自己商品对应查询的搜索用户、看到商品用户、位置、浏览、订单/GMV、转化等 | 需 Seller API `Client-Id`/`Api-Key`，并且需自己的 SKU/listing 和实际曝光；完整历史/扩展数据与 Premium 或 Premium Plus 有关 | 判断“自己的商品在哪些查询下被看见并成交” | 不是全市场关键词数据库；无 listing 或无曝光时没有可用证据。Ozon 官方开发者公告明确这些方法对应“我的商品查询”，且 Premium 扩展时间和数据量：[官方公告](https://dev.ozon.ru/news/512-Novye-metody-dlia-raboty-s-analitikoi-po-zaprosam-tovarov-v-Seller-API/) |
| Seller API 通用分析 `/v1/analytics/data` | 按 SKU、日期等维度读取自己店铺的浏览、加购、订单等指标（可用指标以当期接口为准） | Seller API 凭证；需自己的商品产生行为 | 自动刷新 listing 漏斗 | 只覆盖授权卖家自己的数据；不能用于无商品阶段的全市场需求发现。官方文档入口：[Ozon Seller API](https://docs.ozon.ru/api/seller/) |
| 商品、库存、订单、退货、财务 API | 自己商品状态与库存；FBS posting；订单商品/价格；退货原因与状态；财务交易与费用 | 卖家账号、API 凭证和相应数据；订单/退货/结算需真实事件已经发生 | 验证有效成交、售后与最终净利润所必需 | 不能在实验前告诉你结果；费用类型和晚到调整必须对账。Ozon 把退货、应计和销售分析分别列为工具，[分析工具](https://docs.ozon.com/global/en/analytics/analytics-and-metrics/analytics-tools/) |
| 卖家后台导出/报表 | 商品、订单、销售、退货、应计、佣金、结算等 CSV/XLSX/报表（具体项以账号为准） | 登录卖家后台；部分报表需已有订单/结算 | API 缺口的可审计兜底；适合财务复核 | 可能异步生成、口径和可用期不同；浏览器下载需要登录/验证码和 Owner 授权 |
| 物流、佣金、促销与计算器 | 中国发货线路、重量/货值分组、时效、费率文件；商品卡佣金；活动折扣规则 | 公开规则可先读；最终可用线路、佣金和报价需账号、类目、SKU、包装参数 | 判断单位经济和硬淘汰 | 历史文件不能替代当期报价；促销折扣可能由卖家承担。[Partner Delivery](https://docs.ozon.com/global/en/fulfillment/rfbs/logistic-settings/partner-delivery-ozon/)、[促销](https://docs.ozon.com/global/en/promotion/promotions/promo/) |

**Ozon 的现实边界：** 无 listing 时，AI 可以从公开页和账号内聚合分析形成线索；有 listing 和曝光后，才能读到“我的商品查询”和漏斗；有订单、退货和结算后，才能判断有效成交和最终净利润。Premium 主要扩大分析深度/历史，不会把未经实验的机会变成已验证需求。

### 2.2 Shopee

Shopee 是按国家/站点经营的 marketplace；“Shopee 数据”不是单一市场数据。未先指定站点，币种、费用、物流、类目规则和消费者行为都不能合并比较。

| 数据源 | 可取得的数据 | 前置条件 | 对需求判断的价值 | 关键限制 |
|---|---|---|---|---|
| 各站点公开搜索/商品页 | 商品、价格、折扣、评分、评论、已显示的销量/售出标签（若该站点页面展示）、店铺信息、配送承诺 | 无需自己的销售；需固定国家站点、收货地区和采集时间；常受登录、验证码和反自动化影响 | 竞品、价格带、评论痛点、供给密度 | 页面展示数字是平台 UI 口径，不等同可审计订单明细；跨站点不可直接合并；看不到竞争者真实曝光、转化、退款和利润 |
| Seller Centre 的 Business Insights/数据中心 | 自己店铺的流量、商品表现、转化、销售、买家与营销表现；可用维度以目标站点后台为准 | 已开通目标站点店铺并获得相应员工权限；商品表现需 listing/流量，销售表现需订单 | 验证自己的 listing 漏斗 | 无店、无站点、无 listing 时不可用；站点差异必须账号内核验，不能把其他国家帮助页当当前权限 |
| Seller Centre 导出 | 商品、订单、收入/钱包、营销或表现报表（具体菜单依站点） | 店铺登录和导出权限；订单/收入报表需真实交易 | API 不开放或字段不足时的审计兜底 | 需人工登录/验证码；可用报表和时间范围存在站点差异，必须保存原文件和口径 |
| Shopee Open Platform 商品 API | 自己店铺商品列表、详情、状态、库存与价格；发布/更新等写操作 | 开发者/合作伙伴应用、Partner ID/Key、店铺 OAuth 授权、对应 scope；需要已开通店铺 | 自动同步自己 listing 与库存 | 不是全市场商品研究 API；写操作是高风险外部动作。官方入口：[Shopee Open Platform](https://open.shopee.com/) |
| Shopee Open Platform 订单/物流/退货退款 API | 自己店铺订单列表与详情、履约状态、物流、退货退款（具体能力依站点和应用权限） | 店铺 OAuth、相应 scope；必须已有订单/售后事件 | 有效成交和售后归因 | 只覆盖授权店铺；不能预知需求；字段与端点必须在选定站点的当前官方文档和真实账号中验证 |
| 财务/钱包/结算数据 | 自己店铺的交易、费用、收入或余额；API 是否开放取决于站点/合作伙伴权限，后台导出通常更现实 | 已有订单与结算；可能需要单独 finance 权限 | 最终净利润对账 | 不能用订单金额替代到账；当前 LingMirror 的 Shopee 代码在财务失败时会用订单估算，这只能是 `estimated`，不能称最终结算 |
| 广告/营销数据 | 自己广告活动的曝光、点击、花费、转化（后台或获批营销接口，依站点） | 广告账户、真实投放、预算和权限 | 真实获客成本与付费流量转化 | 无投放就没有数据；第三方应用未必获得营销权限；广告成交只证明该价格×流量组合 |

**Shopee 的现实边界与阻塞：** 官方开放平台文档的详细端点、scope 和站点适用性往往需要开发者登录/应用资格后查看。因 Owner 尚未选定并开通一个国家站点，本审计不把任何跨站点的营销、财务或 Business Insights 字段当成已取得能力。第一步不是开发更多 Shopee 代码，而是选定站点后，由 Owner 完成入驻/OAuth 授权，导出后台可见菜单和权限清单。

### 2.3 Shopify

| 数据源 | 可取得的数据 | 前置条件 | 对需求判断的价值 | 关键限制 |
|---|---|---|---|---|
| Shopify Admin 商品/库存 API | 自己商店的产品、变体、价格、库存和发布状态 | 已有 Shopify 店铺、应用安装与 `read_products`/库存等 scope；数据需要自己创建 | 管理实验商品 | 不是 marketplace 竞品或市场需求数据；空店只能返回空数据 |
| GraphQL Admin Orders | 自己店铺订单、客户、商品行、财务与履约状态 | `read_orders` 等 scope；必须已有订单。默认只可读取最近 60 天，读取更早订单需 `read_all_orders` 审批 | 验证付款、订单和履约 | 只覆盖自己的店；订单仍不等于最终净利润。[Shopify Order 官方文档](https://shopify.dev/docs/api/admin-graphql/latest/objects/order) |
| Shopify Analytics 后台 | 销售、sessions、转化、加购、到达结账、渠道/地区/设备、商品表现等自己的店铺数据 | Shopify 订阅店铺；员工需 Analytics 权限；流量指标需在线商店真实访问 | 自有站漏斗与渠道归因 | 没有流量时没有需求证据；主要功能官方称各 Shopify 订阅计划可用，[Shopify Analytics](https://help.shopify.com/en/manual/reports-and-analytics/shopify-reports/new-analytics) |
| Live View | 当前访客、sessions、活跃购物车、到达结账、购买、热门地点/商品 | 在线商店渠道和真实访问 | 观察一次发布/推广的即时反馈 | 实时窗口小，可能与第三方统计不同；不能代替长期实验。[Live View](https://help.shopify.com/en/manual/reports-and-analytics/shopify-reports/live-view) |
| 后台报表导出 | 大多数报表可导出 CSV、XML、JSONL 或 Parquet；销售、获客、库存等类别 | 店铺与 Analytics 权限；报表内容取决于已产生的数据 | 可审计归档和离线分析 | 报表是“你的店铺数据”，不提供全网需求。[导出报表](https://help.shopify.com/en/manual/reports-and-analytics/shopify-reports/report-types/custom-reports/export-reports) |
| ShopifyQL GraphQL API | 程序化查询自己店铺的销售、订单等报表数据 | `read_reports`；涉及客户字段还需 `read_customers` 和受保护客户数据 Level 2 资格 | 自动化经营分析 | 特定权限审批，不是公开市场数据库。[ShopifyQL API](https://shopify.dev/docs/apps/build/shopifyql/graphql-admin-api) |
| 市场、支付、广告/渠道数据 | Shopify Markets 配置、自己的渠道归因、支付/退款等 | 相应 scope、支付渠道和真实经营事件；广告平台数据通常还需对应广告账户授权 | 验证国家、流量来源、支付与退款 | Shopify Markets 是商家配置的目标市场，不是市场需求数据库；广告花费来自外部渠道授权 |

**Shopify 的现实边界：** Shopify 能很好地测量“被带到自己商店的人如何行动”，但不会自动带来这些人。若无既有受众、SEO 排名、创作者渠道或获批的小额广告，AI 创建商品页后得到零订单不能区分“商品无需求”和“无人看见商品”。因此当前不适合作为首轮 marketplace 需求发现工具。

## 三、自动化边界

### 3.1 可以安全自动化（只读/建议模式）

1. 固定国家站点、城市、登录状态和时间，采集公开搜索结果与商品详情快照。
2. 抽取标题、规格、价格、促销、评分、评论、配送承诺，去重并聚类为“需求—规格—价格带”线索。
3. 对评论做问题归类，但保留原文、URL、时间和商品 ID，模型总结不能替代原始证据。
4. 在 Owner 授权后读取卖家 API、后台导出和报表，自动保存原始 payload、口径、时间窗与权限状态。
5. 将字段标为 `actual / quoted / estimated / unknown / conflict`；缺失时保持 unknown。
6. 计算公开可复核的代理指标：价格分布、评价量分布、头部集中、交付时效分布、评论问题频率。
7. 生成 20 个待验证假设、硬淘汰项、3 个候选建议和需要 Owner 决策的证据卡。
8. 上线后同步自己的曝光、点击、加购、订单、退款、退货和结算，触发预算与止损提醒。

### 3.2 不能诚实自动得出的结论

1. 从公开搜索排名推断真实销量或关键词总搜索量。
2. 从评论数推断近期销量；评论是累计且评论率未知。
3. 从平台“热门”推断新卖家可获得同等流量。
4. 从搜索/点击/加购直接推断愿意付款、签收并保留商品。
5. 从竞品标价推断 Owner 能实现的成交价；促销、品牌、配送和评价资产不同。
6. 从订单金额推断最终净利润；退货、拒收、平台费、广告、汇兑和晚到调整尚未结清。
7. 从供应商展示价推断可交付同规格质量和最终到岸成本。
8. 在未指定 Shopee 站点、未登录 Ozon/Shopify 真实账号时，声称某 API/报表已对 Owner 开放。

### 3.3 必须通过真实小额实验才能知道

| 未知问题 | 为什么研究数据不能回答 | 最小验证动作 |
|---|---|---|
| 新 listing 能否获得有效曝光 | 平台排序与账号、价格、库存、配送、内容、促销有关 | 发布 1 个具体 SKU，固定观察期，读取真实 impressions |
| 买家是否愿意付该价格 | 搜索和评论不是支付行为 | 展示可成交报价，记录点击→加购→结账→付款 |
| 需要多少广告费获得一单 | CPC、转化率随词、素材、价格和竞争实时变化 | Owner 批准的小额、有日限额的单变量广告测试 |
| 交付承诺是否可信 | 线路表是报价，不是该包裹实际妥投 | 发出真实订单或可追踪测试包裹，记录全链路时效/附加费 |
| 商品质量是否匹配承诺 | 供应商页面和样品描述不可替代验收 | 先买样、测量、功能/耐久/包装验收 |
| 取消、拒收、退货和争议率 | 新品、国家、配送和描述共同决定 | 非关联买家真实订单，观察完整退货/争议窗口 |
| 最终净利润是否为正 | 费用和汇兑有晚到调整 | 等平台结算与所有凭证齐全后复算 |
| 是否可复现 | 一单可能是偶然或促销补贴 | 在止损线内做下一小批/下一周期复验 |

## 四、结合当前仓库的现实审计

### 4.1 已有能力

- `backend-go/internal/domain/integrations/ozon.go` 已实现 Ozon 卖家凭证读取、商品发布/状态、库存、订单、退货和财务交易读取的代码路径。
- `backend-go/internal/domain/integrations/shopee.go` 有商品、订单、退货和财务读取的代码路径。
- `backend-go/internal/domain/integrations/shopify.go` 有 Shopify 商店商品/订单等适配路径。
- `backend-go/internal/platform/toolbridge/bridge.go` 已有页面驱动降级和 list-page 采集接口。
- `chrome-extension/content-script-list.ts` 当前能从 Ozon/供应商列表页抽取可见商品卡的标题、价格、URL、图片、原始文本/HTML。
- Candidate 模块已有 `CollectLead`、不可变 `CollectionEvidence` 和 `GET /api/v1/candidates/collection-evidence/:id` 证据读取方向。

这些能力适合成为“可追溯采集与经营归因底座”，但不等于已经取得真实平台数据。

### 4.2 当前不能声称已完成的能力

1. **当前公开列表采集字段过窄。** 主要是标题、价格、URL、图片和原始卡片；没有平台官方搜索量、曝光、加购、订单、退货或结算。
2. **Ozon 集成是卖家经营 API，不是全市场需求 API。** 当前代码重点是已有商品、订单、退货与财务；尚未见对 Ozon 聚合“在 Ozon 卖什么”或商品查询分析的已验证生产采集。
3. **Shopee 凭证验证明确未实现。** `ValidateCredentials` 返回 not yet implemented；因此不能把代码中写出的端点当作真实账号已验收能力。
4. **Shopee 财务存在估算降级。** finance API 失败后，代码会从订单构造 settlement；这必须标成 `estimated`，不能进入“最终净利润”。
5. **任何适配器都缺少本次审计范围内的账号级实证。** 代码能发请求不代表 Owner 的账号、scope、站点、API 版本和字段当前可用。
6. **浏览器采集受页面变化与反自动化影响。** CSS selector 成功不证明字段完整，也不能绕过登录、验证码、地区选择或平台条款。
7. **Shopify 数据只属于自己的店。** 即使 API 完整，也不能在没有流量时自动发现市场需求。

因此当前正确标签是：

| 能力 | 状态 |
|---|---|
| 公开页面商品线索采集 | `implemented, needs live QA` |
| 不可变页面证据关联 | `implemented/in progress, needs migration + live verification` |
| Ozon 卖家订单/退货/财务读取 | `code exists, account-level unverified` |
| Ozon 全市场需求分析采集 | `not demonstrated` |
| Shopee OAuth/凭证可用性 | `blocked/unimplemented validation` |
| Shopee 最终结算 | `not demonstrated; estimation fallback unsafe for final` |
| Shopify 自店订单分析 | `code exists, requires store + traffic/orders` |
| AI 自动发现“已验证付费需求” | `impossible before real experiment` |

## 五、最小可行研究流程

目标不是让 Owner 手工抄数，而是让 AI 自动做大部分只读工作，Owner 只处理身份、授权、冲突和花钱决定。

### 阶段 0：平台与账号闸门（0 元采购）

1. Ozon 登录并核验主体、合同、收款、类目、API Key 权限、后台分析菜单、当前物流和费用入口。
2. 保存脱敏截图/导出、URL、时间、账号/站点、权限结果；任何无法确认项保持 `unknown`。
3. Ozon 六项硬闸失败，则选定**一个具体 Shopee 国家站点**后重复入驻、收款、物流、费用和数据权限核验。
4. Shopify 只有已有可验证目标流量和可用支付渠道时才进入候选。

**输出：** 一个通过闸门的平台；没有通过则停止，不采购。

### 阶段 1：自动形成 20 个需求—规格假设

1. 从平台官方聚合分析（能取到时）获取趋势、查询、热门/缺失商品等候选词。
2. 固定地区和时间采集公开搜索结果前 20–50 个商品及详情/评论快照。
3. AI 聚类为不同的“购买任务 + 关键规格 + 价格带”，换颜色/换卖家不算新线索。
4. 每条至少保留：官方信号、3–5 个直接竞品、评论痛点、价格带、交付时效、未知项。
5. 硬淘汰敏感合规、不可运、疑似侵权、重/易碎/液体/带电、价格明显覆盖不了保守成本的线索。

**输出：** 20 个有来源但仍未验证付费需求的假设。

### 阶段 2：供应商与成本验证，压到 3 个机会

1. 对同一冻结规格向至少 2 家独立供应商索取书面报价。
2. 取得样品、MOQ、包装后尺寸重量、国内运费、交期、质量/赔付和合规资料。
3. 用 Owner 账号和当期官方文件获取类目佣金、可用线路、物流报价、退件处置、收款/汇兑费用。
4. 关键成本必须为 `actual` 或仍有效的 `quoted`；`unknown` 不能被模型均值填补。
5. 样品实测通过后，才称“合格商品机会”。

**输出：** 3 个需求—规格—供应商—合规—物流—完整成本相连的机会。

### 阶段 3：Owner 批准 1 个真实小额实验

批准对象必须具体到：一个 SKU、供应商/批次、数量、售价、线路、广告上限、总预算、不可回收损失和停止规则。发布、采购、广告和资金动作分别留审批与审计。

**输出：** 1 个可停止、可归因的实验，不是“爆款承诺”。

### 阶段 4：真实漏斗与最终净利润

依次记录：

`曝光 → 点击/商品页访问 → 加购 → 结账 → 付款订单 → 发货 → 签收 → 退货/争议窗口关闭 → 最终结算 → 最终净利润`

每个阶段都允许得出不同结论：无曝光先修流量；有曝光无点击修定位/素材；有点击无付款修价格、信任或交付；付款后拒收/退货修描述、质量或物流。一次只改一个主要变量，并受新增损失上限约束。

## 六、阻塞与人工授权点

### 6.1 当前明确阻塞

| 阻塞 | 为什么 AI 不能自行解决 | 解锁人/证据 |
|---|---|---|
| Ozon 真实账号权限与后台菜单未知 | 需要 Owner 身份、登录、验证码和账号合同 | Owner 登录授权；保存脱敏权限清单 |
| Ozon API Key/scope 未生产验收 | 代码存在不等于凭证有效 | Owner 创建/授权只读 key；执行最小只读请求 |
| Ozon 聚合需求数据具体可见范围未知 | 与国家、账号、订阅和菜单开放有关 | 账号内截图/导出/API 响应；记录 Premium 状态 |
| Shopee 尚未指定国家站点/开店 | 各站费用、数据、物流和 API 资格不同 | Owner 选择站点并完成入驻 |
| Shopee Open Platform 应用/OAuth 未验收 | 需要开发者资格、Partner 凭证与店铺授权 | Owner 授权；真实 OAuth 和 scope 清单 |
| Shopify 缺目标流量与支付闭环 | AI 不能凭空产生可归因买家 | 已有受众证据，或 Owner 批准有限广告实验 |
| 具体 SKU、包装参数与合规归类未定 | 佣金、物流、税费、认证都依赖具体商品 | 冻结规格、实测样品、候选 HS/ТН ВЭД 与书面核验 |
| 真实获客成本、拒收退货和利润未知 | 只在真实交易后产生 | 小额实验、完整观察期、平台结算与现金凭证 |

### 6.2 必须由 Owner 授权或审批

**仅需访问授权，不代表花钱：**

- 登录、验证码、cookie/浏览器会话；
- 创建只读 API 凭证、OAuth 授权和员工 Analytics 权限；
- 下载含经营/客户/财务数据的报表；
- 允许保存脱敏原始证据及规定保留期。

**每次都需业务审批并留痕：**

- 采购样品或首批货；
- 发布/修改外部 listing；
- 设置价格、折扣、库存和物流；
- 启动广告及日/总预算；
- 发货、退款、退货处置和补偿；
- 补货或扩大实验。

AI 可以准备动作、计算风险和提醒，但不得替 Owner 执行上述外部写入或资金动作。

## 七、建议的验收口径

本阶段“自动化需求研究完成”不能用“AI 给了 20 个商品”验收，应同时满足：

1. 20 条是不同的需求—规格—价格带假设，不是 20 个商品链接。
2. 每条均有不可变原始证据、来源、时间、地区、采集方式和指标口径。
3. 公开观察、平台聚合、自己 listing、自己订单四类证据没有混写。
4. 任何推断都标成 `estimated/hypothesis`；缺失值为 `unknown`。
5. 进入 3 个候选前，每个都有两家同规格报价、样品验证、合规路径、真实包装参数和当期物流/费用证据。
6. 只有独立买家真实付款并签收，才称有效成交；只有退货/争议关闭且结算完整，才称最终净利润。
7. 没有真实小额实验，不得出现“已发现付费需求”结论。

## 八、官方来源与来源边界

### Ozon

- [Ozon 官方：Analytics tools](https://docs.ozon.com/global/en/analytics/analytics-and-metrics/analytics-tools/) — 平台分析工具种类与经营数据入口。
- [Ozon for Developers：商品查询分析 API 公告](https://dev.ozon.ru/news/512-Novye-metody-dlia-raboty-s-analitikoi-po-zaprosam-tovarov-v-Seller-API/) — `/v1/analytics/product-queries*` 对应自己的商品查询，Premium 扩大范围。
- [Ozon Seller API 官方入口](https://docs.ozon.ru/api/seller/) — 卖家 API 身份与接口参考。
- [Ozon Partner Delivery](https://docs.ozon.com/global/en/fulfillment/rfbs/logistic-settings/partner-delivery-ozon/) — 中国发货、动态费率/时效和商品限制。
- [Ozon 商品库存可用性](https://docs.ozon.com/common/en/tovary/harakteristiki-i-nalichie/) — 地区与库存影响公开商品可见/可售。
- [Ozon 促销](https://docs.ozon.com/global/en/promotion/promotions/promo/) — 平台/卖家促销会影响展示和成交价。

### Shopee

- [Shopee Open Platform 官方入口](https://open.shopee.com/) — 应用注册、店铺授权与 Open API 文档入口。
- [Shopee 中国卖家官方入口](https://shopee.cn/) — 跨境卖家入驻、站点和经营工具入口。

Shopee 的精确 API scope、Business Insights 字段、财务/营销接口和导出范围必须在**选定国家站点 + 已授权开发者/卖家账号**中复核。本报告刻意不使用非官方 API 汇总网站填补这些空白。

### Shopify

- [Shopify GraphQL Admin Order](https://shopify.dev/docs/api/admin-graphql/latest/objects/order) — 订单 scope 与默认 60 天边界。
- [ShopifyQL with GraphQL Admin API](https://shopify.dev/docs/apps/build/shopifyql/graphql-admin-api) — `read_reports`、客户数据权限和程序化报表。
- [Shopify Analytics](https://help.shopify.com/en/manual/reports-and-analytics/shopify-reports/new-analytics) — 自己店铺的访客、表现与交易分析；主要功能的计划可用性。
- [Shopify Live View](https://help.shopify.com/en/manual/reports-and-analytics/shopify-reports/live-view) — sessions、加购、结账和购买的实时漏斗。
- [Shopify Reports](https://help.shopify.com/en/manual/reports-and-analytics/shopify-reports/report-types) — 报表是自己店铺数据，类别包括获客、库存和销售。
- [Shopify 导出报表](https://help.shopify.com/en/manual/reports-and-analytics/shopify-reports/report-types/custom-reports/export-reports) — CSV/XML/JSONL/Parquet 导出。

### 研究限制

- 官方页面、API、订阅权益和站点规则会变化；任何花钱决定前必须以 Owner 当日真实账号、当前官方文档和原始响应重新取证。
- 本报告验证的是“官方宣称的数据边界”和“仓库代码中存在的路径”，不是生产账号连通性证明。
- 未使用第三方文章，也未把非官方镜像文档作为结论依据。
