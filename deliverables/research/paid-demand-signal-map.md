# 跨境电商“真实付费需求”信号地图

> 研究日期：2026-07-11
> 范围：Ozon、Shopee、Shopify，以及政府/平台公开一手资料
> 目的：判断哪些数据能证明“有人正在为某个问题付钱”，而不是推荐市场、类目、商品或客户。

## 先说结论

**单独的搜索量、曝光、点击、收藏、加购、评论数、公开“已售”数字、类目 GMV 大盘，都不能证明真实付费需求。** 它们最多证明注意、考虑、历史交易痕迹或宏观市场存在。

在本研究范围内，最可靠的可取得证据来自**店铺自有的一方订单与资金记录**：具体 SKU 的非测试订单，存在成功支付/捕获记录，之后经过足够观察窗仍未取消、退款或拒付，并且最好已经妥投。若还要证明“为某个问题付钱”，必须再把该 SKU 的明确用途/规格，与订单、退货原因或购买后反馈做可审计关联；“卖出了一个商品”并不自动等于“验证了研究者假设的问题”。

公开资料无法提供竞争对手逐笔支付、最终净成交、退款或买家问题标签。因此，在没有本店真实实验订单前，最多只能形成“值得验证的问题”，不能宣称发现了真实付费需求。

## 一、证据等级

| 等级 | 可观察证据 | 能证明什么 | 不能证明什么 |
|---|---|---|---|
| **P5 最强：最终净付费** | SKU 级成功支付/捕获 + 妥投/完成 + 观察窗后无取消、退款、拒付；金额、币种、数量可核对 | 有人在该地区、该价格、该时点为该商品完成了净付款 | 若商品对应多个用途，仍不能单凭订单确认买家具体要解决的问题；也不能直接证明可持续利润 |
| **P4 强：已付费但未最终结算** | 成功支付/捕获的非测试订单，但仍在退货/争议窗内 | 买家实际付过钱 | 最终保留收入、满意度、净利润 |
| **P3 中：已下单/已完成结账** | 订单创建、COD 下单、待支付、授权但未捕获 | 有明确购买承诺或结账行为 | 钱已到账；COD 尤其可能拒收/取消 |
| **P2 弱：交易近端行为** | 加购、发起结账、询盘、优惠券领取 | 比浏览更接近购买 | 实际付款；行为可能受促销、机器人、误触影响 |
| **P1 注意力** | 搜索、曝光、点击、商品页访问、收藏、评论/评分数量 | 有注意或历史互动 | 当前有人付款、金额、净成交、未满足问题 |
| **P0 背景** | 政府电商销售大盘、进出口额、平台总 GMV | 某地区/行业存在经济活动 | 某个平台、某 SKU、某问题的付费需求 |

### 升级规则

只有同时满足以下条件，才把线索升级为“真实付费需求（P5）”：

1. **对象明确**：订单行能落到具体 SKU/变体和明确、可检验的需求—规格假设；
2. **付款真实**：支付/捕获交易状态成功，排除测试单、免费单、员工单和仅授权未捕获；
3. **履约真实**：已妥投/完成，而不是仅创建订单；COD 必须以签收及结算为准；
4. **净额真实**：扣除取消、退款、退货、拒付；退款对象和数量能回连订单行；
5. **时间成熟**：经过平台适用的退货/争议观察窗；报告必须写明截止日，不能把窗口内订单当最终净成交；
6. **地域可核对**：使用订单收货国家/地区或平台市场，不用 IP、语言或流量来源代替购买地区；
7. **可复现**：不是单个偶发订单。最低建议为同一需求—规格假设在不同日期出现多笔独立 P5 订单；阈值应在实验前预注册，不能看到结果后修改；
8. **问题归因有证据**：商品页明确承诺的用途/规格，或购买后的一方原因字段/访谈能支持“为什么买”。若没有，只能写“该 SKU 有净付费”，不能写“该问题已验证”。

## 二、事实 / 推断 / 未知

### 事实（官方资料直接支持）

- Shopify `OrderTransaction` 是订单相关支付交易，包含 `kind`、`status`、金额、处理时间、网关、`test` 等字段；访问需 `read_orders` 或 `read_marketplace_orders`。它明确区分支付、退款等交易生命周期。[Shopify：OrderTransaction](https://shopify.dev/docs/api/admin-graphql/latest/objects/OrderTransaction)
- Shopify `Order` 提供订单行、创建/处理时间、财务与履约状态、收货地址、交易、退款、退货、总收款与总退款等字段；默认只能读取最近 60 天订单，更早订单需额外 `read_all_orders` 权限。[Shopify：Order](https://shopify.dev/docs/api/admin-graphql/latest/objects/Order)；[Shopify：访问范围](https://shopify.dev/docs/api/usage/access-scopes)
- Shopify 的 `Refund` 记录本身**不保证钱已真正退回**，必须检查关联 `OrderTransaction.status`；这说明“退款对象存在”与“资金交易成功”必须分开判断。[Shopify：Refund](https://shopify.dev/docs/api/admin-graphql/latest/objects/Refund)
- Shopify 退货行可提供数量、标准化退货原因及买家备注；读取需要 `read_returns` 或 `read_marketplace_returns`。官方原因包括损坏/缺陷、与描述不符、尺寸过大/过小、不想要、错发等。[Shopify：ReturnLineItem](https://shopify.dev/docs/api/admin-graphql/latest/objects/ReturnLineItem)；[Shopify：ReturnReason](https://shopify.dev/docs/api/admin-graphql/latest/enums/ReturnReason)
- Shopify 的购买旅程是**订单关联的归因资料**，只覆盖最长 30 天归因窗；它能连接购买前会话与订单，但会话本身仍不是付款。[Shopify：CustomerJourney](https://shopify.dev/docs/api/admin-graphql/latest/objects/CustomerJourney)
- ShopifyQL 可按 `sales`、`orders` 等数据源查询销售/订单指标，需 `read_reports`，并涉及受保护客户数据要求；汇总结果仍应与订单交易核对。[Shopify：shopifyqlQuery](https://shopify.dev/docs/api/admin-graphql/latest/queries/shopifyqlQuery)
- Ozon Seller API 官方接口包括财务交易、FBO/FBS 发货单、分析等卖家私有数据面；调用需要卖家 `Client-Id` 与 `Api-Key`。财务交易接口 `/v3/finance/transaction/list` 单次请求最长一个月，能按日期、交易类型、发货单号查询应计项目。[Ozon Seller API 官方文档入口](https://docs.ozon.ru/api/seller/)；[Ozon 官方开发者：API 密钥规则](https://dev.ozon.ru/news/649-Obnovlenie-pravil-raboty-s-API-kliuchami-Vazhnye-izmeneniia-v-rabote-s-Ozon-Seller-API/)
- Shopee 官方 Seller Education 资料显示卖家后台有销售报告，能按“全部、未付款、待发货、运输中、完成、取消、退款/退货”等状态导出；Business Insights 同时展示销售、订单和商品浏览，因此必须把成交字段与流量字段分开。[Shopee 官方销售报告教材](https://cdngarenanow-a.akamaihd.net/shopee/seller/seller_cms/9473fdf75e16ecbda6e3255b34f849a2/Finance%20Course.pdf)；[Shopee 官方数据经营教材](https://cdngarenanow-a.akamaihd.net/shopee/seller/seller_cms/1056e5d0c187f40912426fb92c104fc7/%5BMY%5D%20Data-Driven%20Business.pdf)
- 菲律宾统计局把电商交易定义为通过网络下单，付款和交付可在线或离线完成；其数据按行业和地区汇总。例如 2022 年批零业电商销售按行业组、全国及地区发布，但没有平台、店铺、SKU 或具体问题粒度。[菲律宾统计局：2022 批发零售业 ASPBI](https://psa.gov.ph/statistics/wholesale-and-retail-trade/aspbi)

### 推断（由事实支持，但不是平台直接结论）

- **成功支付 + 最终无退款**比“订单数”更接近付费需求，因为订单可能未付款、COD 拒收、取消或之后退款。
- **退货原因可反证需求—规格匹配**：例如“尺寸不合”“与描述不符”说明曾付款，但也说明当前规格/表达未稳定满足需求；不应把毛订单全部计为成功验证。
- **跨店公开数据只能做发现线索**。即使公开页显示销量或评论，也缺少支付状态、退货、拒付、测试/刷单排除及订单地区，最高只能列为 P1–P2。
- **政府电商销售可用于市场闸门的背景校验，不可用于商品验证**。其粒度和时滞不足以把收入归因到某个平台、SKU 或问题。
- 对自用首轮实验，最小闭环应是：需求—规格假设 → 上架/报价 → 自有订单行 → 成功支付 → 妥投 → 观察窗后净留存 → 退货原因/反馈。缺少任一关键环节都应降级表述。

### 未知（官方公开资料不足或需账户内验证）

- 本研究未登录任何账户，因此不知道 Owner 当前 Ozon/Shopee/Shopify 店铺实际开通了哪些报表、API 权限、历史保留期和地区字段。
- Ozon 官方公开文档当前未在无需交互/授权的页面中稳定暴露所有分析指标的最新字段表；具体 `ordered_units`、`delivered_units`、`returns`、`cancellations` 的现行可用组合、最大分析时间窗及跨境地区字段，需要在官方接口控制台以实际卖家权限只读验证。
- Shopee Open Platform 文档入口通常要求开发者登录；公开的官方教材能确认报表与状态分类，但不能证明 2026 年每个市场的 API 字段、最大查询窗、数据留存和权限完全一致。
- Ozon/Shopee 是否向普通卖家提供**竞争对手**的净支付、退款与买家地区数据：未发现官方公开证据；在获得明确官方权限说明前，应视为不可取得。
- 各平台/国家、商品类型和买家身份适用的退货、退款、拒付最终观察窗并不统一，必须按真实店铺政策和订单市场逐单确定。
- 从订单推断“买家具体问题”的可靠性未知，除非商品只服务单一明确用途，或另有购买原因/售后原因的一方证据。

## 三、各平台可取得的数据与边界

| 平台/来源 | 可取得字段（与付费判断有关） | 时间窗 | 地区粒度 | 权限 | 主要限制与最高等级 |
|---|---|---|---|---|---|
| **Ozon Seller 私有订单/发货单** | 发货单号、创建时间、状态、商品/SKU、数量、价格；FBO/FBS 路径分开 | 具体接口依版本；需实测。订单应按创建/状态时间增量留档 | 店铺所属市场；收货/仓库/配送字段能否稳定得到买家地区需实测 | 卖家后台或 Seller API `Client-Id` + `Api-Key` | 仅本店；订单创建/发货不等于付款。单独最高 P3；与资金及售后联结后可到 P5。[官方 API](https://docs.ozon.ru/api/seller/)
| **Ozon Seller 财务交易** | 交易/操作 ID、日期、类型、金额、销售应计、佣金、配送/退回费用、发货单号、商品项 | `/v3/finance/transaction/list` 单请求最多 1 个月；历史总保留期需实测 | 主要是店铺/发货单/仓库维度；不能假定有买家城市 | 财务类 API Key 权限 | 应计不必然等于最终净利润；需把退款、退货、服务费与同一发货单/SKU 对齐。成功销售应计可 P4，成熟净额可 P5。[官方 API](https://docs.ozon.ru/api/seller/)
| **Ozon 分析/推广** | 订单件数、收入、曝光、访问、加购、转化、取消/退货等（实际可用组合需控制台确认） | 最新最大查询窗未知；应每日留存证据快照 | 常见为 SKU/类目/品牌/日等聚合；买家地区未知 | 卖家分析权限；部分能力可能受套餐/角色限制 | 曝光到加购仅 P1–P2；“下单收入”若未排退款至多 P3/P4；不能看竞争者逐笔净支付。[官方 API](https://docs.ozon.ru/api/seller/)
| **Shopee Seller Centre 销售报告/订单** | 订单号、商品/变体、数量、金额、未付款/待发货/运输中/完成/取消/退款退货状态（各市场导出列需实测） | 官方公开教材未给出统一最大历史窗；按市场后台实测并定期导出 | shop/market（国家站）确定；更细收货地区取决于订单导出与隐私权限 | 对应站点卖家/员工角色 | 仅本店；COD 订单尤其不能以“已下单”当付款。完成且对账成熟后可 P5。[官方销售报告教材](https://cdngarenanow-a.akamaihd.net/shopee/seller/seller_cms/9473fdf75e16ecbda6e3255b34f849a2/Finance%20Course.pdf)
| **Shopee Business Insights** | 销售额、订单量，以及浏览/流量等经营指标 | 可按月/季/年回看是官方教材给出的经营用法；精确保留期未知 | 通常店铺/商品；公开教材未证明买家地区维度 | 卖家后台权限 | 销售汇总必须与订单完成、退款/退货、钱包结算对账；浏览量仅 P1。[官方数据经营教材](https://cdngarenanow-a.akamaihd.net/shopee/seller/seller_cms/1056e5d0c187f40912426fb92c104fc7/%5BMY%5D%20Data-Driven%20Business.pdf)
| **Shopee Open Platform** | 理论上可读本店订单状态与详情；2026 各站点具体字段需登录官方开发者门户确认 | 未知，按接口/站点验证 | 至少 shop/market；更细粒度未知 | Partner/app 授权 + 店铺授权；能力因市场/卖家类型可能不同 | 无授权不能读；旧版官方指南不能当当前字段保证。最高等级取决于能否同时取得完成、结算和退款数据。[Shopee Open Platform](https://open.shopee.com/)
| **Shopify Admin GraphQL：订单/交易** | SKU/变体、数量、金额/币种、`createdAt`/`processedAt`、财务/履约状态、`fullyPaid`、`totalReceivedSet`、`totalRefundedSet`、交易 `kind/status/test`、收货地址 | 默认最近 60 天；更早需 `read_all_orders`；大数据可用 Bulk Operations | 收货地址可到国家/省市（受保护客户数据规则）；店铺/市场与呈现币种也可用 | 安装并授权 app；`read_orders`；全历史另需审批 `read_all_orders` | 只覆盖该商户；手工标记付款、测试交易、外部网关需排除/核验。成功捕获 P4，成熟净额 P5。[Order](https://shopify.dev/docs/api/admin-graphql/latest/objects/Order)；[OrderTransaction](https://shopify.dev/docs/api/admin-graphql/latest/objects/OrderTransaction)
| **Shopify：退款/退货** | 退款金额、行项目、数量、处理时间、交易状态；退货数量、原因、买家备注 | 随订单可见性；订单超过 60 天时受全订单权限影响 | 跟随订单地区，不是独立市场估计 | `read_orders`；退货另需 `read_returns` | `Refund` 存在不代表退款交易成功；必须看交易状态。用于从毛付费降到净付费及识别失败原因。[Refund](https://shopify.dev/docs/api/admin-graphql/latest/objects/Refund)；[ReturnLineItem](https://shopify.dev/docs/api/admin-graphql/latest/objects/ReturnLineItem)
| **Shopify：旅程/报告** | 首末访问、转化天数、30 日内购买前会话；ShopifyQL 的 sales/orders 汇总 | Journey 最长 30 天归因；订单默认 60 天；报告查询窗依数据权限 | 访问来源/订单地区可辅助切分，但不能互相替代 | `read_orders`；ShopifyQL 需 `read_reports` 并满足受保护客户数据要求 | 归因不是因果；流量不等于付款；汇总需回查交易。[CustomerJourney](https://shopify.dev/docs/api/admin-graphql/latest/objects/CustomerJourney)；[ShopifyQL](https://shopify.dev/docs/api/admin-graphql/latest/queries/shopifyqlQuery)
| **政府一手统计（菲律宾示例）** | 电商销售额，按行业组、全国/地区；年度调查 | 年度，发布存在明显时滞 | 全国/行政区/行业，不到平台、店铺、SKU、问题 | 公开，无账号 | 只能 P0；定义允许线下付款/交付，且无法归因 Shopee 或具体商品。[PSA ASPBI](https://psa.gov.ph/statistics/wholesale-and-retail-trade/aspbi)

## 四、注意力与付费必须分开的字段

| 只能证明注意/考虑 | 交易承诺但未证明付款 | 可支持真实付款 | 用于把毛付款修正为净付款 |
|---|---|---|---|
| 搜索词/搜索量、曝光、排名、点击、访问、停留、收藏、评论/评分数、广告点击 | 加购、发起结账、订单创建、COD 下单、支付授权、待发货 | 非测试的成功 capture/sale、`fullyPaid` 且交易状态成功、平台成功销售应计 | 取消、退款交易状态、退货数量/原因、拒付/争议、妥投/完成状态、平台费用与退货运费 |

特别提醒：评论能证明有人留下过内容，可能来自真实购买者，也可能存在赠品、历史购买、合并变体或平台规则差异；即使是“已验证购买”，也只证明该评论关联过购买，不等于当前仍有人付费，更不等于未满足需求。

## 五、最多 5 个值得进一步验证的问题

以下均是**问题，不是已确认需求**：

1. 对一个预先定义的“需求—规格”假设，Ozon 本店能否在连续 30 天获得至少若干笔来自不同日期/买家的成功销售应计，并在适用售后窗后仍保持正的 SKU 级净收入？
2. Shopee 某一国家站中，某需求—规格假设的“完成且已结算订单”是否能在排除 COD 拒收、取消和退货后重复出现，而不是只在一次补贴活动中出现？
3. 同一明确用途的 Shopify 单品，在不同流量来源和不同周次是否都产生非测试成功捕获，并且退货原因不集中于“与描述不符/缺陷/尺寸问题”？
4. 当同一需求—规格假设在两个平台测试时，净付费差异究竟来自地区需求，还是价格、配送时效、促销、页面表达和平台人群差异？需要怎样的对照才能区分？
5. 商品订单证明的是买家为“该商品”付钱，还是为我们假设的“该问题”付钱？能否通过单一用途页面、变体选择、购买后自愿问题或售后原因获得不含诱导的归因证据？

## 六、建议的验证记录（不涉及访问账号）

未来由 Owner 明确授权后，每个实验至少保存以下只读证据链；本研究没有执行这些动作：

```text
假设 ID / 明确问题 / SKU 与规格 / 平台与国家站
订单 ID / 订单行 ID / 下单时间 / 数量 / 成交价与币种
支付交易 ID / kind / status / test / captured_at
履约状态 / delivered_or_completed_at
取消 / 退款交易 / 退货数量与原因 / 拒付
观察窗截止日 / 截止日净件数 / 截止日净收入
平台费用、物流、采购和税费（用于利润，不用于证明付款本身）
证据快照 ID / 采集时间 / 官方接口或报表来源
```

判定语言必须与等级一致：

- P1：**“出现注意力线索。”**
- P2：**“出现购买近端行为，尚未证明付款。”**
- P3：**“出现订单承诺，尚未证明最终付款。”**
- P4：**“出现已付款订单，仍在售后观察窗内。”**
- P5：**“截至某日，出现可核对的最终净付费；问题归因强度为高/中/低。”**

## 来源完整性说明

- 全文只使用 Ozon、Shopee、Shopify 和政府官方域名/官方分发 CDN 的资料；未使用媒体榜单、培训机构文章、第三方选品分或 AI 生成市场判断。
- Ozon 与 Shopee 的部分最新接口字段在公开、无需登录的页面不可稳定读取，因此明确列为“未知/需授权后实测”，没有用第三方镜像补成事实。
- 本报告没有访问、读取或修改任何用户/卖家账号，也没有推荐具体市场、类目、商品或客户。
