# Data Reality Audit — LingMirror Intelligence 与 Portfolio Launch OS

Date: 2026-07-11
Status: Research input, not product approval
Scope: Amazon, Shopee, Lazada, Shopify, Ozon；权威贸易/税则/合规/物流数据；市场情报竞品与定价
Method: 三个独立研究 Agent，优先官方/第一方来源；无法从官方来源确认的字段标记 `unknown`，不根据第三方博客补齐。

## Executive Verdict

1. **没有卖家授权，就没有真实销售数据。** 公开价格、评论、榜单、搜索建议和第三方销量估算都不能称为真实订单、退款、费用或结算。
2. **Amazon 是当前最适合验证真实 Marketplace 数据闭环的平台候选。** 官方 SP-API 文档覆盖商品、价格、库存、订单、退货、费用、财务、报告和刊登；但需要 Professional seller account、角色/权限和 OAuth/自授权。
3. **Shopify 是最快的集成测试面，不是市场需求数据源。** Dev store、GraphQL Admin API 和 webhook 适合验证工具与状态恢复，但 Shopify 没有跨店铺的市场搜索量、排名或真实需求数据。
4. **Shopee、Lazada、Ozon 不能在当前证据下宣称完整 capability。** 官方文档存在登录/JS/合作伙伴门槛，许多字段、费率、SLA 和 sandbox 能力必须在真实账号或平台批准后验证。
5. **公共权威数据适合国家/品类贸易与风险背景，不适合 SKU 销量预测。** Eurostat、TARIC、Safety Gate、ECB 等可以形成可追溯证据，但不能回答某个平台商品会卖多少。
6. **商业再分发权是 Intelligence 产品的第一风险。** UN Comtrade 未经书面许可禁止商业利用/自动下载/再分发；DHL rating data 也存在对外披露、存储和派生限制。能查询不等于能商业销售。
7. **现有竞品已经覆盖大量研究与运营功能。** LingMirror Intelligence 不能仅以“更多数据”竞争；可能的差异是跨市场证据、完整落地经济、许可与 provenance，以及从假设到 finalized 结果的可审计 Launch dossier。

## Evidence Classification

| Class | Definition | Can support | Cannot support |
|---|---|---|---|
| A — Seller-authorized truth | 店铺授权 API/导出中的订单、退款、费用、广告和结算 | 真实业务结果、对账和 finalized contribution | 未授权的全市场需求 |
| B — Official public fact | 政府、平台或权利机构公开的贸易、规则、费用、商品/价格信息 | 市场背景、规则、准入、公开竞争观察 | 真实销量、转化、利润 |
| C — Observable proxy | 评论、排名、搜索趋势、公开商品变化 | 机会假设、问题发现 | 订单量、GMV、因果需求 |
| D — Third-party estimate | 工具商基于 BSR、样本或模型估算的销量/搜索量 | 需标明方法和误差的比较信号 | 平台真实交易事实 |
| E — Synthetic/test | sandbox、dev store、fixture、模拟订单 | 工具契约、恢复、幂等和回放测试 | 任何商业验证 |

## Platform Capability Matrix

Legend: `Public` 无卖家授权；`Seller` 需店铺授权；`Partner` 还需平台/应用批准；`Unknown` 官方可访问材料不足；`Unavailable` 未发现对应平台能力。

| Capability | Amazon | Shopee | Lazada | Shopify | Ozon |
|---|---|---|---|---|---|
| 市场需求/搜索量 | Seller；Data Kiosk/Brand Analytics 类能力，非公共市场 API | Unknown | Unknown | Unavailable；没有跨店需求索引 | Unknown / seller analytics 可能存在但未证实 |
| 公共商品/价格/评论/排名 | Catalog/Pricing 需 Seller/Partner；评论与排名 API 未证实 | Unknown | Unknown | Seller 只能访问本店商品；无跨店评论/排名 | Unknown |
| 库存 | Seller | Seller/Partner | Seller/Partner | Seller | Seller |
| 广告曝光/点击 | Advertiser + Ads API 授权 | Unknown | Unknown | 非原生广告网络市场数据 | Unknown |
| 订单 | Seller；PII 另有 restricted role | Seller/Partner | Seller/Partner | Seller；默认历史窗口有限 | Seller |
| 退款/退货 | Seller | Seller/Partner | Unknown | Seller | Unknown |
| 平台费用 | Seller；Product Fees/Finances/Reports | Unknown | Seller/Partner，声明/费用字段可见 | Seller，取决于支付/计划能力 | Unknown/Seller |
| 结算 | Seller；Finances/Reports | Unknown | Seller/Partner | Shopify Payments eligible stores | Unknown/Seller |
| 刊登发布/更新/状态 | Seller；Listings/Feeds | Seller/Partner | Seller/Partner | Seller；GraphQL Admin | Seller |
| 测试环境 | Static sandbox；部分 API dynamic sandbox | Unknown | 官方存在 test-order 流程，完整 sandbox 未证实 | Dev store 最清晰 | Unknown |

### Amazon

- SP-API 官方说明覆盖 Data Kiosk、Catalog、Pricing、Inventory、Orders、Returns、Product Fees、Finances、Reports、Listings 和 Feeds。
  Sources: [SP-API overview](https://developer-docs.amazon.com/sp-api/lang-US/docs/what-is-the-selling-partner-api), [API references](https://developer-docs.amazon.com/sp-api/lang-US/reference/welcome-to-api-references)
- Ads 数据走独立 Ads API，需要广告主授权。
  Source: [Amazon Ads API](https://advertising.amazon.com/about-api/)
- 私有 seller app 仍需要 Professional plan；公开 app 需要 Amazon approval/Appstore 与 seller OAuth，敏感字段受角色和安全审查限制。
  Sources: [Onboarding overview](https://developer-docs.amazon.com/sp-api/docs/onboarding-overview), [Registration overview](https://developer-docs.amazon.com/sp-api/lang-en_EN/docs/sp-api-registration-overview)
- 美国卖家计划公开价格为 Individual `$0.99/item` 或 Professional `$39.99/month`，另有类目 referral fee、FBA 和广告费用。
  Source: [Amazon selling plans and pricing](https://sell.amazon.com/pricing)
- Sandbox 成功是测试证据，不是真实销售数据。

### Shopify

- 当前官方主 API 是 GraphQL Admin；REST Admin 已进入 legacy，新 public app 使用 GraphQL。
  Sources: [Shopify API documentation](https://shopify.dev/docs/api), [REST legacy notice](https://shopify.dev/docs/api/admin-rest/latest)
- 订单、退款、商品、库存和本店支付数据需要 merchant consent/scopes。
  Sources: [Order object](https://shopify.dev/docs/api/admin-graphql/2026-01/objects/Order), [Refund object](https://shopify.dev/docs/api/admin-graphql/latest/objects/refund)
- Dev store 是最快的工具与 webhook 测试面，但 synthetic orders 必须标记 Test。
- Shopify 不提供跨店铺市场搜索量、平台排名或全市场销量，因此不能单独支撑“选市场/选产品”。
- Shopify API Terms 禁止未授权 scraping、系统化采集、商品/商业索引和 benchmarking。
  Source: [Shopify API License and Terms](https://www.shopify.com/legal/api-terms)

### Lazada

- Seller business data 需要 OAuth2 授权，覆盖 SG/MY/TH/VN/ID/PH。
  Source: [Lazada seller OAuth](https://developer.alibaba.com/docs/doc.htm?articleId=108260&docType=1&treeId=499)
- 商业 ERP 类应用可能需要在 Service Marketplace 发布并由卖家订阅；in-house app 存在 whitelist 限制。
  Source: [Lazada authorization policy](https://developer.alibaba.com/docs/doc.htm?articleId=108056&docType=1&treeId=499)
- 商品、订单、声明/费用和发布相关 scope 可在官方授权界面观察到，但实时 SLA、rate limit 和完整 sandbox 仍需账号验证。
  Source: [Lazada authorization screen](https://auth.lazada.com/oauth/authorize?client_id=115721&force_auth=true&redirect_uri=&response_type=code)

### Shopee 与 Ozon

- 官方 Open/Seller API 存在，但当前公开可访问文档不足以可靠确认所有字段、权限、rate limit、费用和 sandbox。
  Sources: [Shopee Open Platform docs](https://open.shopee.com/documents), [Ozon Seller API](https://docs.ozon.ru/api/seller/)
- 结论必须保持 `Unknown`，不能用聚合博客或非官方 SDK 推断生产 capability。

## Authoritative Public Data Sources

| Source | Measures | Useful granularity | Access | Commercial reuse status | Main limitation |
|---|---|---|---|---|---|
| Eurostat Comext | EU 货物进出口流 | 月/年、reporter、partner、CN8、value、quantity | 免费 API/bulk，无认证 | 多数可复用并署名；存在 reporter/粒度例外 | 贸易流，不是消费者或平台销量 |
| UN Comtrade | 全球贸易流 | 年/月、国家、HS6，部分 tariff line | API/download | 未经 UN 书面许可禁止商业利用、自动下载和再分发 | 许可是 Intelligence 商业化阻断 |
| EU TARIC | EU 税则、优惠、配额、反倾销、禁限措施 | CN/TARIC、origin、effective date | 官方查询/数据传输 | 专项商业再分发许可未确认 | 规则查询，不是 binding ruling |
| Access2Markets | 税费、手续、原产地和产品要求 | 国家对 + HS code | Web；公开稳定 bulk/API 未确认 | 数据库受版权保护，商业再分发未知 | 适合人工核验，不宜默认抓取销售 |
| EU Safety Gate | 危险产品/召回警报 | 事件、产品、来源国、风险、措施 | 网站/PDF，完整 bulk API 未确认 | 可复用但需指定署名；图片/IP 另核验 | 没有在售量分母，不能计算召回率 |
| EUIPO/TMview | 商标申请与注册 | 标志、状态、类别、申请人、日期 | API Portal/TMview | API licence 与来源局条款需逐项核验 | 不等于法律 clearance |
| ECB EXR | 官方参考汇率 | 日频货币对 | 免费 SDMX API/bulk | 发布前补齐专项许可核验 | 不是实际结算汇率 |
| DHL MyDHL | 单个账户/承运商报价与时效 | 路线、重量、服务、indicative rate | 需账户与批准凭证 | 官方条款限制存储、派生和对外披露 | 不能做公开物流价格数据库 |
| Google Trends API Alpha | 搜索兴趣 proxy | 词、国家/地区、时间 | 申请制 alpha | 数据商用/持久化条款未明确 | 不是购买或平台行为，不能作为硬依赖 |

### Important Official Sources

- [Eurostat international trade database](https://ec.europa.eu/eurostat/web/international-trade-in-goods/database)
- [Eurostat API introduction](https://ec.europa.eu/eurostat/web/user-guides/data-browser/api-data-access/api-introduction)
- [Eurostat copyright notice](https://ec.europa.eu/eurostat/help/copyright-notice)
- [UN Comtrade](https://comtradeplus.un.org/)
- [UN Comtrade license](https://comtradeplus.un.org/LicenseAgreement)
- [EU TARIC](https://taxation-customs.ec.europa.eu/customs-4/calculation-customs-duties/customs-tariff/eu-customs-tariff-taric_en)
- [Access2Markets](https://trade.ec.europa.eu/access-to-markets/en/home)
- [EU Safety Gate](https://ec.europa.eu/safety-gate-alerts)
- [ECB exchange-rate dataset](https://data.ecb.europa.eu/data/datasets/EXR/structure)
- [DHL Express MyDHL API](https://developer.dhl.com/api-reference/dhl-express-mydhl-api?lang=en)
- [Google Trends API Alpha](https://developers.google.com/search/apis/trends)

## Public Data Gaps That Cannot Be Filled Honestly

Public authoritative data does not provide:

- SKU/ASIN/listing-level real units sold, GMV or profit.
- Marketplace conversion, impressions, ads, inventory, refunds and settlement without seller authorization.
- A clean split between marketplace, independent-site, wholesale and offline demand in customs data.
- Actual cross-border parcel cost and delivery-time distribution across all carriers and customer contracts.
- Final tariff classification, origin ruling, product certification or legal clearance for one concrete product.
- Consumer-demand causality from import value or Google search interest.
- Safety/recall rates when only alert events exist without an exposure denominator.

Any product claiming these fields must identify seller-authorized, licensed commercial or estimated sources. `Unknown` is preferable to fabricated completeness.

## Commercial Alternatives and Official Pricing

Pricing checked 2026-07-11. Promotions and annual billing may change.

| Product | Verified role | Official pricing snapshot | Data caveat | Workflow boundary |
|---|---|---|---|---|
| Amazon Product Opportunity Explorer | Amazon-native niches, search/purchase trends, reviews, returns, saturation | Included with Professional seller account, `$39.99/month + fees` in US | Amazon data; terms with insufficient volume may be absent; no public POE raw API found | Stops at insight/strategy; wider Seller Central handles execution |
| Jungle Scout Catalyst / Cobalt | Amazon product/keyword research, suppliers, listing, inventory and brand intelligence | Starter `$29/mo`; Growth `$49`; Brand Owner `$129`; Cobalt custom | Vendor estimates; enterprise terms do not warrant all data accurate/complete | Broad suite, but no verified governed thesis→compliance→publication→finalized learning record |
| Helium 10 | Amazon/TikTok/Walmart research, listings, ads and operations | Platinum `$129/mo`; Diamond `$359`; Enterprise from `$1,499/mo` | Xray sales are estimates based on category BSR; accuracy not guaranteed | Broad action suite without verified end-to-end evidence/approval/causal ledger |
| SellerSprite | Amazon multi-market product/category/keyword research and listing optimization | Basic `$39`; Standard `$79`; Advanced `$129`; VIP `$189` monthly; APIs `$149–$3,999/mo` examples | Vendor terms disclaim accuracy/completeness/currentness | Strong export/API and insights; no verified supplier/compliance/publication control loop |

Sources:

- [Amazon Product Opportunity Explorer](https://sell.amazon.com/tools/product-opportunity-explorer/)
- [Jungle Scout pricing](https://www.junglescout.com/pricing/)
- [Jungle Scout product overview](https://support.junglescout.com/hc/en-us/articles/26234503954967-What-is-Jungle-Scout-Amazon-Growth-Tools-and-Market-Intelligence-for-Brands-Agencies-and-Sellers)
- [Helium 10 pricing](https://www.helium10.com/pricing/)
- [Helium 10 sales estimator method](https://kb.helium10.com/hc/en-us/articles/1260805907550-How-Do-I-Use-the-Sales-Estimator-in-the-Chrome-Extension)
- [SellerSprite pricing](https://www.sellersprite.com/price?from=home)
- [SellerSprite API pricing](https://www.sellersprite.com/en/help/about-api)
- [SellerSprite terms](https://www.sellersprite.com/en/help/terms)

## Competitive Implication

The market already offers inexpensive research tools and expensive data/API products. Therefore:

- “AI product research” is not a differentiated category.
- “More product data” is not enough to win against platform-native or established vendor data.
- A defensible Intelligence hypothesis must focus on licensed cross-market normalization, transparent source/estimate boundaries, complete landed economics, compliance context and decision-ready evidence.
- Portfolio Launch OS differentiation is not a bigger dashboard. It is a governed dossier linking thesis, sources, assumptions, supplier evidence, compliance, approval, listing, actual transactions and finalized learning.

No official source reviewed proves that one existing product provides this entire governed causal record. This is an inference from product documentation, not proof of an empty market.

## First-Lake Options

### Option 1 — Shopify Integration Lake

Purpose: validate auth, tools, webhooks, listing state, orders/refunds and durable recovery using a dev store.

Pros:
- Best official developer experience and fastest legal test surface.
- Good for ToolContract, workflow, replay and observability.

Limits:
- Synthetic orders are not commercial evidence.
- No marketplace-wide demand data.

### Option 2 — Amazon Marketplace Truth Lake

Purpose: use a real Professional seller account and private SP-API authorization to validate listings, orders, refunds, fees and settlements.

Pros:
- Best documented Marketplace path in this audit.
- Can produce seller-authorized operational truth.

Limits:
- Requires account, roles, security work and real activity.
- Market opportunity tools remain account/UI gated; public observations are not sales.

### Option 3 — EU Public Intelligence Lake

Purpose: build a commercial-rights-reviewed country/category evidence product from Eurostat trade data, tariff/risk sources and reference FX.

Pros:
- Strongest no-seller-credential official source chain identified.
- Suitable for testing provenance, classification versions, evidence graphs and licensing.

Limits:
- Measures imports and regulation, not marketplace demand or SKU sales.
- TARIC, Access2Markets, Safety Gate assets and ECB data require source-specific reuse review.

The research Agent proposed `Germany × toys` because EU trade, tariff, recall and FX sources are rich. This report **does not recommend toys as the first commerce category**: child-product compliance and safety risk make it unsuitable without explicit Legal Reviewer approval. Use it only as a data-coverage demonstration, or choose a lower-risk category after a compliance screen.

## Recommended Parallel Path

Because LingMirror Intelligence and Portfolio Launch OS are separate products and codebases:

1. **Intelligence:** begin with the EU Public Intelligence Lake after commercial reuse review. Select a low-risk category only after Legal Reviewer screening. The first paid output should be a decision-grade evidence dossier, not a generic dashboard.
2. **Portfolio Launch OS:** use Shopify dev store for tool/runtime verification, while preparing Amazon Professional seller authorization for the first genuine Marketplace Truth Lake.
3. **Commercial validation:** interview buyers against actual alternatives and invoices. Pricing hypotheses from competitor anchors are not willingness-to-pay evidence.
4. **Integration contract:** Intelligence exposes versioned evidence artifacts. Launch OS stores received evidence snapshots and provenance; it never directly queries the Intelligence database.

## Buyer Interview Questions

1. Show the last market/product decision for which you paid data. What invoice or budget line covered it?
2. Which metric do you trust enough to commit inventory cash, and which do you independently verify?
3. What was the last data-driven launch that failed, which signal failed, and what did it cost?
4. Would you pay for raw/exportable data, an API, or a recommendation with provenance? Rank them.
5. At what point must market research connect to supplier quote, landed cost and compliance before it becomes operationally valuable?
6. Would a well-supported “do not launch” recommendation be worth paying for?
7. Choose among `$99/month` insight-only, `$399/month` workflow and `$1,500/month` governed controls. Which has budget today and who owns it?
8. What proof would cause you to switch from your current tool: backtest, pilot, hours saved, avoided failed launch or finalized ROI?
9. Would you authorize seller data if the software provider also operates its own store? What contractual boundary is required?
10. Which outputs must be exportable or deletable before procurement will approve the product?

## Decision Required

The next Owner decision is not “which product to build first.” Both products are approved to run in parallel. The decision is which first Lake each product will use:

- Intelligence: EU official-data evidence product, or another geography after source audit.
- Portfolio Launch OS: Shopify integration lake plus Amazon seller-authorized marketplace lake, or another platform after account verification.

No product should claim “real market demand” until it has seller-authorized outcomes or explicitly labels the signal as trade flow/proxy/estimate.
