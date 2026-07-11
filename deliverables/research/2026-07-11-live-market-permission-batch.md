# 候选市场带日期公开资料侦察批次

> 后续状态覆盖：本文件是侦察与数据现实输出，不是最终权限决定。独立反证见 `2026-07-11-live-market-independent-falsification.md`；其最终结果为英国、美国 hold，日本 reject，本轮权限请求为 0。页面导入的是这些已完成输出，不在点击时联网。

> 观察日期：2026-07-11（Asia/Shanghai）
> 研究类型：跨市场、只读、外部公开资料
> 证据等级：公开来源为 `external_observed`；经营结论为 `unknown`，候选状态均为 `evidence_missing`
> 重要限制：本轮没有登录任何平台卖家账号、没有读取账号内搜索/点击/加购/购买/退货/广告/结算数据，也没有观察到凌镜自己的真实成交。因此本文不证明付费需求、商品机会、可盈利性或市场已被选中。

## 1. 本轮结论

本轮比较了三个不相同的“地区 × 可观察消费者 × 需求场景 × 销售渠道”组合：英国私人租客的严重潮湿/冷凝场景、美国有行动困难的老年人的非医疗日常辅助场景、日本 65 岁以上人群的非医疗日常辅助场景。

只有前两个组合具有“公开资料证明人群/问题确实存在，并且官方渠道文档说明账号内有进一步验证数据”的**权限申请线索**。日本组合只有人口规模事实，没有足够具体的需求场景事实，暂不进入权限申请。三个组合都不能进入商品实验。

推荐的下一步不是采购、上架或投广告，而是由 Owner 决定是否开通一个 Amazon 专业卖家账号的最小只读预检。若只允许一次预检，优先英国组合，因为官方住房调查直接观察到了场景发生率；这只是证据质量排序，不是市场胜出。

## 2. 证据边界与方法

- 只采用来源主体拥有该事实的一手资料：英国政府住房、税务、消费者与化学品监管资料；美国 Census/CDC/CPSC 资料；日本总务省统计局资料；Amazon 官方卖家与 SP-API 文档。
- “人群或问题存在”不等于“该人群会购买某个商品”。公开统计没有给出 Amazon 内搜索、购买、竞争、退货或净利润。
- Amazon 的公开页面只说明工具和字段存在。Product Opportunity Explorer、销售与流量报告、订单与结算数据都需要卖家账号或 API 授权，本文未读取其中任何值。
- 为减少不可逆风险，本批次只提出页面只读查看或非受限只读 API。不给 Listing、Feeds、Pricing 写权限，不创建商品，不改价，不投放广告。

## 3. 候选 A：英国私人租客 × 严重潮湿/冷凝 × Amazon.co.uk

### 候选定义

- **地区**：英国中的 England（公开需求证据仅覆盖 England；不可外推为整个 UK）
- **可观察消费者**：居住在私人租赁住房、住房调查记录到严重潮湿/冷凝问题的家庭
- **需求场景**：住户需要降低或管理室内潮湿/冷凝造成的日常影响；商品范围尚未确定
- **销售渠道**：Amazon.co.uk（Marketplace ID `A1F83G8C2ARO7P`）

### 公开支持证据

1. 英国政府 2024–25 English Housing Survey 记录：2024 年 England 有 5% 住宅存在潮湿问题，私人租赁住宅为 10%，高于社会租赁的 7% 和自有住房的 4%；严重冷凝/霉菌约涉及 823,000 套住宅。该调查只记录足以进入 HHSRS 风险评估的较严重情况，不包括轻微问题。来源：[EHS 2024–25 home insulation fact sheet](https://www.gov.uk/government/statistics/english-housing-survey-2024-to-2025-home-insulation-fact-sheet/english-housing-survey-2024-to-2025-home-insulation-fact-sheet)、[EHS 2024–25 headline report PDF](https://assets.publishing.service.gov.uk/media/697a0f35005d288bf850deb2/2024-25_EHS_Headline_Report_on_Housing_Quality_and_Energy_Efficiency.pdf)（观察于 2026-07-11）。
2. 2023–24 EHS 的深入分析指出，潮湿成因复杂，建筑失修、通风不足、保温和住户行为可能共同作用；私人租赁住宅在卧室、客厅、浴室和厨房均有记录。该事实支持“场景存在”，也同时否定“买一个简单商品就能解决根因”。来源：[EHS 2023–24 drivers and impacts of housing quality](https://www.gov.uk/government/statistics/english-housing-survey-2023-to-2024-drivers-and-impacts-of-housing-quality/english-housing-survey-2023-to-2024-drivers-and-impacts-of-housing-quality)（观察于 2026-07-11）。
3. Amazon 官方 SP-API Marketplace IDs 文档列出 UK Marketplace ID；Amazon 官方 Product Opportunity Explorer 说明卖家登录后可按 niche 查看搜索、购买、评价、价格、退货及饱和度等数据。这证明有下一步验证入口，不证明该场景在 Amazon.co.uk 的需求。来源：[Amazon Marketplace IDs](https://developer-docs.amazon.com/sp-api/lang-US/docs/marketplace-ids)、[Product Opportunity Explorer](https://sell.amazon.com/tools/product-opportunity-explorer/)（观察于 2026-07-11）。

### 最强反证

- 官方调查明确指出潮湿通常由建筑结构、通风、保温、供暖和住户行为共同造成。租客可能没有改造权限，消费品可能只能缓解表象，不能解决根因；廉价吸湿用品也可能是高度同质化、低毛利和季节性商品。
- 若产品宣称杀灭或控制霉菌，可能进入英国生物杀灭产品监管。HSE 要求先判断是否属于 biocidal product，并核验活性物质和供应商资格；普通“家居用品”假设不能绕过该问题。来源：[HSE — How to get a biocidal product on the UK market](https://www.hse.gov.uk/biocides/products/market.htm)（观察于 2026-07-11）。
- 海外卖家还必须在具体货物流、货值和客户类型下确认 VAT 责任；线上远程销售通常还涉及取消、退货和退款义务。来源：[HMRC — selling through an online marketplace](https://www.gov.uk/guidance/charging-vat-when-using-an-online-marketplace-to-sell-goods-to-customers-in-the-uk)、[GOV.UK — distance selling](https://www.gov.uk/online-and-distance-selling-for-businesses/distance-selling)（观察于 2026-07-11）。

### 精确的数据访问状态与最小权限

| 数据 | 当前状态 | 所需权限/动作 | 通过条件 |
|---|---|---|---|
| 问题发生率 | 公开、已观察 | 无 | 仅证明 England 私人租赁潮湿问题存在 |
| Amazon 搜索量、购买行为、价格、评价、退货、niche 饱和度 | 账号内、未观察 | Owner 登录 UK 卖家账号，在 Seller Central → Growth → Product Opportunity Explorer 只读查看；先不接 API | 保存观察时间和原始导出/截图；必须同时记录无结果或低量结果 |
| 自有 listing 的页面流量和销售 | 账号内、当前不存在 | 不能作为市场预检的前置证据；`GET_SALES_AND_TRAFFIC_REPORT` 只在自有 ASIN/销售发生后有意义 | 不得用空值解释为零需求 |
| 订单、退款、结算、真实费用 | 账号内、当前不存在 | 仅在 Owner 批准实验后，另行授权 Orders/Finance 只读角色 | 与同一订单和银行入账对账后才可确认利润/现金 |
| 合规 | 商品级、未知 | 在选择具体产品前按产品逐项核验 HSE/OPSS、VAT、远程销售义务 | 任何杀霉/抗菌宣称未厘清时自动淘汰 |

**当前裁决：`permission_candidate` + `evidence_missing`。** 只值得申请一次只读需求预检，不值得采购或上架。

## 4. 候选 B：美国有行动困难的老年人 × 非医疗日常辅助 × Amazon.com

### 候选定义

- **地区**：United States
- **可观察消费者**：65 岁以上、报告行动困难（walking or climbing stairs）的社区成年人
- **需求场景**：不涉及诊断、治疗或医疗宣称的日常取放、整理或轻量辅助；具体商品未确定
- **销售渠道**：Amazon.com（Marketplace ID `ATVPDKIKX0DER`）

### 公开支持证据

1. CDC 的 Disability and Health Data System 将 mobility 定义为严重行走或爬楼困难，并提供州和全国层面的公开数据。CDC 对 2022 BRFSS 的发布称，65 岁以上成年人报告任一残障的比例为 43.9%；CDC 的专题说明还称行动困难是老年人中最常见的类型，约四分之一。来源：[CDC DHDS overview](https://www.cdc.gov/dhds/about/overview.html)、[CDC 2022 disability release](https://www.cdc.gov/media/releases/2024/s0716-Adult-disability.html)、[CDC disability type and access analysis](https://www.cdc.gov/disability-and-health/articles-documents/disabilities-health-care-access.html)（观察于 2026-07-11）。
2. Census 2023 ACS S1810 提供年龄分组与具体 disability type 的官方估计；这使“消费者”可以按年龄和功能困难观察，而不是泛称“老人市场”。来源：[Census ACS 2023 S1810](https://data.census.gov/table/ACSST1Y2023.S1810?moe=false)（观察于 2026-07-11）。
3. Amazon 官方资料说明 Product Opportunity Explorer 需要注册卖家账号并登录 Seller Central，之后才可查看 niche 的搜索、购买、评价、价格、退货和竞争数据；US Marketplace ID 由官方 SP-API 文档确认。来源：[Product Opportunity Explorer](https://sell.amazon.com/tools/product-opportunity-explorer/)、[Amazon Marketplace IDs](https://developer-docs.amazon.com/sp-api/lang-US/docs/marketplace-ids)（观察于 2026-07-11）。

### 最强反证

- 人口和功能困难并不证明消费者需要、能使用或愿意购买某种跨境商品；购买者也可能是照护者而不是本人。泛化的“老年用品”会把多个不同功能问题错误合并。
- 产品若承重、用于防跌、接触食品、面向儿童或带医疗宣称，合规和伤害责任可能显著上升。CPSC 明确指出进口商/线上卖家需要识别适用的测试、标签和认证要求，平台还可以施加高于联邦最低要求的内部门槛。来源：[CPSC Online Sellers' Safety Guide](https://www.cpsc.gov/Business--Manufacturing/Online-Sellers-Safety-Guide)、[CPSC FAQ](https://www.cpsc.gov/FAQ/Online-Sellers-Safety-Guide-FAQs)（观察于 2026-07-11）。
- Amazon US 专业计划公开价为每月 USD 39.99，另有 referral fee、FBA 和广告等可能费用；仅有毛销售额不能证明利润。来源：[Amazon US pricing](https://sell.amazon.com/pricing?mons_sel_locale=en_US)（观察于 2026-07-11）。

### 精确的数据访问状态与最小权限

| 数据 | 当前状态 | 所需权限/动作 | 通过条件 |
|---|---|---|---|
| 人群与功能困难 | 公开、已观察 | 无 | 仅证明功能困难人群存在 |
| 场景相关搜索、购买、竞争、退货 | 账号内、未观察 | Owner 登录 US Seller Central，Product Opportunity Explorer 只读；按“功能场景词”而非“老人用品”宽词查询 | 至少保存 niche 定义、时间窗、搜索/购买/退货/竞争字段及反向查询；无匹配也必须保存 |
| API 自动读取 | 未授权 | 如后续确需自动化：专业卖家账号 + Owner 作为 Primary User 注册 private SP-API app + 最小非受限只读角色 + self-authorization refresh token | 当前阶段不申请 Listing/Feeds/Pricing 写权限；密钥不得进入研究文档 |
| 商品合规 | 未知 | 具体候选商品逐项运行 CPSC Regulatory Robot，并核验 supplier certificates/recalls | 任何承重、防跌、儿童或医疗边界未厘清时淘汰 |
| 真实利润 | 不存在 | 只有经 Owner 批准的小实验后才能收集订单、退款、平台费用、物流、结算和银行入账 | 最终利润与现金回收分开核验 |

Amazon 官方 SP-API 文档说明：private seller app 需要专业卖家账号，且 Primary User 才能自授权；角色控制具体资源访问，缺少角色会返回 403。来源：[SP-API registration overview](https://developer-docs.amazon.com/sp-api/lang-en_EN/docs/sp-api-registration-overview)、[private app self-authorization](https://developer-docs.amazon.com/sp-api/docs/self-authorization)、[SP-API roles](https://developer-docs.amazon.com/sp-api/docs/direct-to-consumer-shipping-restricted-role?ld=usb2bshorturl)（观察于 2026-07-11）。

**当前裁决：`permission_candidate` + `evidence_missing`。** 证据弱于候选 A；只能做只读场景验证。

## 5. 候选 C：日本 65 岁以上人群 × 非医疗日常辅助 × Amazon.co.jp

### 公开证据

日本总务省统计局公布：截至 2024-10-01，65 岁以上人口为 36.243 million，占总人口 29.3%；Amazon 官方 Marketplace IDs 文档列出 Japan Marketplace ID `A1VC38T7YXB528`。来源：[Statistics Bureau of Japan — 2024 population estimates](https://www.stat.go.jp/english/data/jinsui/2024np/index.htm)、[Amazon Marketplace IDs](https://developer-docs.amazon.com/sp-api/lang-US/docs/marketplace-ids)（观察于 2026-07-11）。

### 淘汰理由 / 最强反证

年龄结构只是人口统计，不是具体需求。当前一手来源没有把一个可观察的日本消费者群体连接到明确的日常场景，也没有任何 Amazon.co.jp 账号内搜索、购买、退货、竞争或履约证据。把“老龄化”直接转成“适老商品机会”属于 `inferred`。此外，本轮未完成日本商品标签、消费者保护、进口主体和税务的商品级核验。

**当前裁决：`hold_no_permission_request` + `evidence_missing`。** 不申请账号权限，不创建案件中的可实验结论。只有找到日本官方的具体功能困难/生活场景资料后才重开。

## 6. 跨候选共同的数据现实

Amazon 的 Analytics Reports 官方文档显示：Search Query Performance 需要 Brand Analytics 角色且需加入 Brand Registry；Sales and Traffic 报告虽对 sellers 可用，但它是卖家自有目录按 ASIN/日期汇总的销售与页面流量，不是进入市场前的全市场需求证明。来源：[Amazon SP-API Analytics Reports](https://developer-docs.amazon.com/sp-api/lang-tr_TR/docs/report-type-values-analytics)（观察于 2026-07-11）。

因此第一步应使用 Seller Central 内 Product Opportunity Explorer 的人工只读证据快照。Brand Registry、广告数据、订单 PII、Listing/Feeds/Pricing 写权限都不是本批次的必要权限。若 Seller Central 账号没有 Product Opportunity Explorer、对应 marketplace 未开通、查询无 matching niche，必须如实记录为阻塞或反证，不得用公开畅销榜或第三方估算补成“付费需求”。

## 7. Owner 六行决策卡

1. **怀疑什么**：公开存在的潮湿或行动困难，是否在指定 Amazon marketplace 中形成可观察的搜索→购买行为，并能留下安全利润。
2. **已证明什么**：英国私人租赁住宅潮湿发生率较高；美国老年人中行动困难可被官方数据观察；Amazon 官方说明卖家账号内存在进一步需求/竞争/退货字段。
3. **尚未证明什么**：具体商品、付费需求、目标价格、获客成本、竞争强度、退货、履约、合规成本、最终净利润和现金回收。
4. **最强反证**：英国潮湿根因可能不能靠消费品解决；美国辅助用品可能带来伤害/合规风险；人口统计可能根本不转化为购买。
5. **下一步所需权限或金额**：只需 Owner 批准一个 marketplace 的 Seller Central 人工只读预检；若账号尚不存在，先只核验开户资格和固定费用，不采购、不上架、不投广告。US 专业计划公开价为 USD 39.99/月；UK 实际费用必须在对应站点开户页再次核验。
6. **停止条件**：工具无权限/无 matching niche；场景购买信号弱或季节性过强；退货/竞争明显不利；商品必须依赖未厘清的杀霉、医疗、防跌或承重宣称；关键费用仍未知；Owner 不愿承担账号固定费或合规验证成本。

## 8. 可导入结构化表

> 下面仅用于创建/更新候选研究案件。`permission_candidate` 不等于 `experiment_ready`；`actual` 一律为 false。

| candidate_key | region | observable_consumer | need_scenario | sales_channel | scout_status | evidence_status | paid_demand_status | strongest_counter | public_source_urls | account_only_fields | minimum_permission_needed | forbidden_permissions_now | next_gate | stop_condition |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `GB-ENG-PRIVATE-RENTER-DAMP-AMZUK-20260711` | `GB-ENG` | `private-renter households in dwellings with survey-recorded serious damp/condensation` | `manage daily effects of serious indoor damp/condensation; product unspecified` | `amazon.co.uk:A1F83G8C2ARO7P` | `permission_candidate` | `evidence_missing` | `unknown` | `root cause may be structural/ventilation; consumer product may not solve it; biocidal claims add regulation` | `https://www.gov.uk/government/statistics/english-housing-survey-2024-to-2025-home-insulation-fact-sheet/english-housing-survey-2024-to-2025-home-insulation-fact-sheet ; https://www.gov.uk/government/statistics/english-housing-survey-2023-to-2024-drivers-and-impacts-of-housing-quality/english-housing-survey-2023-to-2024-drivers-and-impacts-of-housing-quality ; https://www.hse.gov.uk/biocides/products/market.htm ; https://sell.amazon.com/tools/product-opportunity-explorer/` | `niche search volume,purchase behavior,pricing,reviews,returns,saturation; later own orders/refunds/fees/settlement` | `Owner interactive read-only Seller Central Product Opportunity Explorer for UK marketplace` | `Listings,Feeds,Pricing,Ads,Orders PII,external writes` | `independent account-data falsification` | `no access or no matching niche; weak purchase signal; adverse returns/competition; biocidal boundary unresolved; key costs unknown` |
| `US-65PLUS-MOBILITY-DAILY-AID-AMZUS-20260711` | `US` | `community adults 65+ reporting serious walking/climbing-stairs difficulty` | `non-medical daily retrieval/organization/light assistance; product unspecified` | `amazon.com:ATVPDKIKX0DER` | `permission_candidate` | `evidence_missing` | `unknown` | `functional difficulty does not establish purchase; caregiver may be buyer; load-bearing/fall/medical boundary raises harm and compliance risk` | `https://www.cdc.gov/dhds/about/overview.html ; https://www.cdc.gov/disability-and-health/articles-documents/disabilities-health-care-access.html ; https://data.census.gov/table/ACSST1Y2023.S1810?moe=false ; https://www.cpsc.gov/Business--Manufacturing/Online-Sellers-Safety-Guide ; https://sell.amazon.com/tools/product-opportunity-explorer/` | `niche search volume,purchase behavior,pricing,reviews,returns,saturation; later own orders/refunds/fees/settlement` | `Owner interactive read-only Seller Central Product Opportunity Explorer for US marketplace; optional later private SP-API non-restricted read roles` | `Listings,Feeds,Pricing,Ads,Orders PII,external writes` | `independent account-data falsification plus product-level CPSC classification` | `no matching niche; ambiguous consumer/buyer; safety boundary unresolved; adverse returns/competition; key costs unknown` |
| `JP-65PLUS-UNSPECIFIED-AID-AMZJP-20260711` | `JP` | `population age 65+ (need segment not yet observable)` | `unspecified non-medical daily assistance` | `amazon.co.jp:A1VC38T7YXB528` | `hold_no_permission_request` | `evidence_missing` | `unknown` | `aging statistic alone is not a need or purchase signal` | `https://www.stat.go.jp/english/data/jinsui/2024np/index.htm ; https://developer-docs.amazon.com/sp-api/lang-US/docs/marketplace-ids` | `not requested` | `none` | `all account permissions and external writes` | `find primary Japan source for a precise functional/need scenario first` | `no precise official need evidence` |

## 9. 批次决策

- 可提交 Owner 审批的只读预检：候选 A；候选 B 作为备选，不应同时付费开多个站点。
- 本轮淘汰/暂停：候选 C。
- 不允许的解释：A 或 B “市场已选中”“需求已验证”“值得采购”“能赚钱”。
- 下一份证据必须来自与本研究不同的 run，并优先寻找否定结果；账号内快照需记录 marketplace、查询词、工具字段、时间窗、观察时间和不可变哈希。
