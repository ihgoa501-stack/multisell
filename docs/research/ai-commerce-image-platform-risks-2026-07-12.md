# AI 电商作图：销售渠道与广告合规风险

> 调研日期 / 访问日期：2026-07-12
> 范围：商品主图、副图、生活方式图、广告创意中的 AI 生成或 AI 编辑图片
> 来源限制：仅平台、监管机构和广告自律机构的一手官方资料
> 证据标签：`quoted` = 官方来源明确陈述；`inferred` = 基于多项官方规则作出的系统建议；`unknown` = 本次未在官方公开资料中确认
> 重要限制：平台规则会按国家、类目、广告产品和账号变化。本文不是法律意见，也不证明任何市场或渠道已经被 Owner 选中。

## 结论

AI 电商作图并非“只要看起来像商品就能上传”。真正的合规对象至少同时受四层约束：

1. 图片必须准确表示实际出售的商品、变体、数量、配件和效果；
2. 主图与副图的允许内容不同，主图通常更严格；
3. 部分渠道明确要求 AI 来源标记或披露，且披露不能补救本身具有误导性的图片；
4. 商家必须在发布前持有权利与客观主张证据，平台审核通过不等于监管合规。

对凌镜的推荐是：**先确定目标渠道和类目，再生成；AI 图片默认只作为候选副图或内部草稿，除非渠道规则快照明确允许其作为主图；所有输出必须和真实 SKU、原图、权利证据、渠道规则版本、人工核验及文件哈希绑定。**（`inferred`）

## 1. 平台规则

### 1.1 Amazon

适用：Amazon 商品详情页；具体站点、类目可能有附加规则。

- `quoted`：Amazon 官方 Seller Forums 的规则说明称，每件商品至少一张图片；所有图片最长边 500–10,000 像素，且须清晰。违规图片可能被拒绝、移除或修改，违规可导致商品从搜索结果中被抑制。[Amazon Product Image Requirements](https://sellercentral.amazon.com/seller-forums/discussions/t/4b3c4c39-6f8c-4312-aa0e-99982eb8f5e1)
- `quoted`：Amazon 官方回复概括主图要求为纯白背景、商品约占图片 85%，准确展示所售商品，只展示所售商品并尽量少用道具；生活方式图通常不允许作为主图。[Amazon: Lifestyle Image as Main Image](https://sellercentral.amazon.com/seller-forums/discussions/t/b2c7291a-2db6-43e5-ac05-b1f7d493420b)
- `unknown`：本次在无需登录可访问的 Amazon 官方资料中，未找到一条适用于所有站点和商品图片的通用“AI 生成必须披露”条款。
- `inferred`：不能因 Amazon 未明确禁止 AI 就将全合成商品作为主图。主图的准确性、白底、商品占比和只展示所售内容仍然适用；AI 改变颜色、形状、数量、附件、包装文字或尺寸感时应阻断。

### 1.2 eBay

适用：eBay 商品刊登；部分品类另有规则。

- `quoted`：图片不得不准确表示商品，不得使用占位图、边框、附加文字/艺术元素/营销材料或任何水印；二手、损坏或有缺陷商品不得用库存图。图片最长边至少 500 像素。[eBay Picture policy](https://www.ebay.com/help/listing-policies/policies/picture-policy?id=4370)
- `quoted`：图片和文字必须真实、准确且不侵犯第三方权利。eBay 建议使用自己的图片；可使用 eBay 商品目录内容。新车库存图须获供应商许可并披露图片并非实际车辆，二手车辆必须展示实际车辆。[eBay Images, videos and text policy](https://www.ebay.com/help/policies/listing-policies/images-videos-text-policy?id=4240)
- `quoted`：违反政策可导致内容移除、警告、活动限制或账号暂停。[eBay Picture policy](https://www.ebay.com/help/listing-policies/policies/picture-policy?id=4370)
- `unknown`：本次未找到 eBay 面向普通商品图片的专门 AI 披露规则。
- `inferred`：AI 场景合成不能掩盖二手品相，不能添加未随单附送的物品，也不能用“AI 生成”水印破坏 eBay 的无水印要求；如需披露，应优先使用刊登字段或描述，而不是擅自在主图加字，且须复核具体品类规则。

### 1.3 Etsy

适用：Etsy 商品及店铺；其“AI 创作商品”规则与“用 AI 修饰普通商品照片”不是同一概念。

- `quoted`：卖家提示词驱动的 AI 创作可以作为“由卖家设计”的商品，但必须在商品描述中披露 AI 使用；单独销售 AI 提示词包不允许。[Etsy stance on AI creations](https://www.etsy.com/seller-handbook/article/1275449912004)；[Etsy What Can I Sell](https://help.etsy.com/hc/en-us/articles/360024112614-What-Can-I-Sell-on-Etsy)
- `quoted`：Etsy 对“由卖家制作”的实体商品要求使用卖家自己的照片；商品仍须符合其 Creativity Standards，违反者可被移除。[Etsy What Can I Sell](https://help.etsy.com/hc/en-us/articles/360024112614-What-Can-I-Sell-on-Etsy)；[Reasons a Listing May Be Removed](https://help.etsy.com/hc/en-us/articles/34707360607511-Reasons-a-Listing-May-Be-Removed-Under-Etsy-s-Creativity-Standards)
- `inferred`：如果出售的是普通实体商品，不能把“AI 艺术允许销售”误读为“允许用完全虚构图片代替实际商品照片”。系统应分别记录 `product_is_ai_created` 与 `listing_photo_is_ai_edited`，不能共用一个披露开关。
- `unknown`：本次未确认 Etsy 对仅更换实体商品照片背景是否一律要求 AI 披露；应在选定商品类型和站点后再次核验。

### 1.4 Shopify

适用：Shopify 商家后台内置媒体生成功能；店铺面向消费者时仍受销售地法律及接入渠道规则约束。

- `quoted`：Shopify Magic 可替换纯色背景，或根据提示修改背景、光线等；Shopify 不支持或认可从网络上传图片或侵犯版权。[Shopify media generation](https://help.shopify.com/en/manual/shopify-admin/productivity-tools/shopify-magic/media-generation)
- `quoted`：Shopify 不主张商家使用该工具创建图片的所有权，也不限制商家在 Shopify 内外用于经营；AI 生成内容带不可见水印/元数据。[Shopify media generation](https://help.shopify.com/en/manual/shopify-admin/productivity-tools/shopify-magic/media-generation)
- `inferred`：这是工具使用说明，不是对商品真实性、广告主张或第三方权利的合规担保。商家仍需独立核验输出。
- `unknown`：本次未找到 Shopify 店铺层面的统一“消费者可见 AI 图片披露”要求；若同步到 Google、Meta、TikTok 或 marketplace，适用的是目标渠道自己的规则。

### 1.5 Google Merchant Center / Shopping

适用：Google Merchant Center 的免费商品信息和 Shopping 广告；规则按 feed 属性区分主图、附加图和生活方式图。

- `quoted`：所有生成式 AI 创建的图片必须保留表明 AI 来源的 IPTC `DigitalSourceType` 元数据；官方列出 `TrainedAlgorithmicMedia`、`CompositeSynthetic`、`AlgorithmicMedia`，不得移除嵌入的来源元数据。[Google AI-generated content](https://support.google.com/merchants/answer/14743464?hl=en)
- `quoted`：该要求适用于 `image_link`、`additional_image_link` 和 `lifestyle_image_link`。[Google product data specification](https://support.google.com/merchants/answer/7052112?hl=en)
- `quoted`：主图应准确完整展示实际商品，尽量少或不用场景布置；不得使用与实际商品无关的通用图、占位图、Logo 替代图、促销文字、价格、CTA、水印或边框。变体应匹配正确颜色、图案和材料。[Google image_link](https://support.google.com/merchants/answer/6324350?hl=en)
- `quoted`：附加图可包含商品场景、使用方式、图形或插画；生活方式图应展示真实场景，但也不得含促销元素、文字、边框或填充。[Google product data specification](https://support.google.com/merchants/answer/7052112?hl=en)；[Google lifestyle_image_link](https://support.google.com/merchants/answer/9103186?hl=en)
- `quoted`：不符合图像要求的商品可被拒登，停止显示于广告和免费商品信息，直到修正。[Google image_link](https://support.google.com/merchants/answer/6324350?hl=en)
- `inferred`：凌镜必须保留/写入 AI 来源元数据，并在所有裁剪、压缩、格式转换后重新检查；只在数据库里记“AI 生成”但导出文件丢失元数据，不满足 Google 文件级要求。

### 1.6 TikTok 广告与商业内容

适用：TikTok Ads、Spark Ads/商业内容；具体功能可用性和法律要求因国家而异。

- `quoted`：完全 AI 生成或被 AI 显著修改的图片、视频、音频必须使用免责声明；广告管理器提供 AI-generated content disclaimer。[TikTok ad disclaimers](https://ads.tiktok.com/help/article/about-ad-disclaimers-in-tiktok-ads-manager)
- `quoted`：TikTok 允许显著 AI 编辑内容，但要求使用 AIGC 标签，或清晰的免责声明、说明、文字水印或贴纸；未披露的 AI 内容可被拒绝或限制。[TikTok misleading and false content](https://ads.tiktok.com/help/article/tiktok-ads-policy-misleading-and-false-content?lang=en)
- `quoted`：TikTok 将光线、亮度、色彩饱和度、背景移除/修改、降噪列为“不显著”的 AI 编辑；若不确定，建议谨慎标记。[TikTok misleading and false content](https://ads.tiktok.com/help/article/tiktok-ads-policy-misleading-and-false-content?lang=en)
- `quoted`：广告与落地页不得在商品、价格、优惠等信息上不一致，不得显示商品 A 而落地页销售商品 B；夸大效果及可能导致错误效果印象的前后对比不允许。[TikTok misleading and false content](https://ads.tiktok.com/help/article/tiktok-ads-policy-misleading-and-false-content?lang=en)
- `quoted`：TikTok 商业内容必须启用 Commercial Content Disclosure；未正确披露可能失去 For You 分发资格并限制流量。[TikTok commercial content disclosure](https://ads.tiktok.com/help/article/about-the-commercial-content-disclosure-setting-for-advertisers)
- `inferred`：背景替换在 TikTok 定义中通常属于轻微编辑，但若生成场景暗示不存在的功能、使用效果、人物背书或实际环境，仍可能构成显著修改或误导；系统应按输出实际改变内容判定，而不是按操作按钮名称判定。

### 1.7 Meta / Facebook / Instagram

适用：Meta 广告和购物广告。

- `quoted`：Meta 的广告审查会检查图片、视频、文字、定向和目的地；广告购物方案还受 Commerce Policies 约束。审查主要自动化，也可能人工复核。[Meta Ads Review Policy](https://www.facebook.com/business/ads/review-policy-guidelines)
- `quoted`：Meta 官方提供背景生成、图片扩展、文字生成和静态图动画等生成式广告功能。[Meta Advantage+ Creative](https://www.facebook.com/business/ads/meta-advantage-plus/creative)
- `unknown`：本次在无需登录可访问的 Meta 官方资料中，没有确认一条适用于普通电商商品 AI 图片的统一强制 AI 披露规则，也未取得一份可稳定引用的完整 Meta 欺骗性广告条款页面。
- `inferred`：不能把“Meta 自己提供生成工具”解释为输出自动合规。发布前仍需按商品真实性、落地页一致性、第三方权利和所售地区法律进行核验；Meta 审核通过只代表平台当次接受，不构成法律或真实性证明。

## 2. 监管与广告自律规则

### 2.1 美国 FTC

适用：面向美国消费者的广告。

- `quoted`：广告必须真实、不具欺骗性、有证据支持且不得不公平；FTC 会从合理消费者角度审视广告整体，包括文字和图片，并同时考察明示与暗示主张。广告运行前必须有证据。[FTC Advertising FAQs](https://www.ftc.gov/business-guidance/resources/advertising-faqs-guide-small-business)
- `quoted`：如果广告声称“眼见为实”地展示产品效果，画面必须真实表示产品能做到的效果；虚构或增强演示可能构成欺骗。[FTC: Less than meets the eye](https://www.ftc.gov/business-guidance/blog/2014/01/less-meets-eye)
- `quoted`：必要披露必须清晰显著；埋在小字、无关文字或容易错过位置的披露通常无效。[FTC Advertising FAQs](https://www.ftc.gov/business-guidance/resources/advertising-faqs-guide-small-business)；[FTC Full Disclosure](https://www.ftc.gov/business-guidance/blog/2014/09/full-disclosure)
- `inferred`：一句“图片由 AI 生成”不能修复错误颜色、夸大尺寸、虚构配件或虚假使用效果，因为 FTC 关注广告的整体净印象和消费者获得的商品事实。

### 2.2 欧盟

适用：面向欧盟消费者的商业行为；各成员国负责落地和执法。

- `quoted`：《不公平商业行为指令》规定，即使个别信息事实正确，商业行为若通过整体呈现欺骗或可能欺骗普通消费者，并影响其交易决定，也可构成误导；重要遗漏同样受规制。[EU Unfair Commercial Practices Directive](https://eur-lex.europa.eu/eli/dir/2005/29/oj/eng)
- `quoted`：《欧盟 AI 法》第 50 条要求生成合成音频、图像、视频或文本的 AI 系统提供者使输出具有机器可读标记并可检测；部署者使用 AI 创建或操纵构成 deep fake 的图像、音频或视频时，应披露其人工生成或操纵来源。[EU AI Act](https://eur-lex.europa.eu/eli/reg/2024/1689/oj/eng)
- `inferred`：高度逼真、看似真实商品或真实使用场景的合成图可能进入 deep fake 定义，但普通背景移除或不实质改变输入语义的标准辅助编辑可能不同。具体电商图是否触发第 50 条需结合输出与实施时间作法律核验，不能一概而论。
- `unknown`：本文没有裁决某张具体商品图是否属于 EU AI Act 的 deep fake，也没有核验各成员国的额外消费者法和语言要求。

### 2.3 英国

适用：在英国媒体投放或面向英国消费者的广告。

- `quoted`：英国政府说明，广告不得包含虚假或欺骗性信息，也不得遗漏重要信息；执法可包括最高 30 万英镑或全球营业额 10%（取较高者）的罚款。[GOV.UK Marketing and advertising law](https://www.gov.uk/marketing-advertising-law/regulations-that-affect-advertising)
- `quoted`：ASA/CAP 会评估广告整体印象。图片展示错误商品、未包含的额外物品，或夸大商品质量、大小，可能误导。[ASA: Avoiding misleading imagery](https://www.asa.org.uk/news/a-picture-says-a-thousand-words-avoiding-misleading-imagery-in-ads.html)
- `quoted`：CAP 要求广告主在发布前持有客观主张的书面证据；AI 或自动化制作不转移广告主责任。[ASA substantiation](https://www.asa.org.uk/news/evidently-evidence-is-everything-six-tips-on-sound-substantiation.html)；[ASA AI and Deepfakes](https://www.asa.org.uk/news/ai-and-deepfakes-four-things-advertisers-need-to-know-before-they-hit-run.html)
- `quoted`：ASA 明确指出 AI 图如被用来表达产品功效而不能准确反映真实功效，可能误导；AI 偏见还可能造成不负责任的刻板印象。[ASA Generative AI & Advertising](https://www.asa.org.uk/news/generative-ai-advertising-decoding-ai-regulation.html)

## 3. 跨渠道高风险清单

以下是依据上述官方规则汇总的系统风险（均为 `inferred`，需用目标渠道规则快照裁决）：

| 风险 | 典型 AI 变化 | 必须核验的真实性证据 |
|---|---|---|
| 商品身份错误 | 型号、接口、按钮、纹理、品牌标识变化 | SKU 规格、真实样品/受控原图、变体映射 |
| 颜色与材料误导 | 色温美化导致颜色改变；塑料变金属/皮革 | 实物色卡或可靠规格、原图、人工对照 |
| 数量与附件误导 | 多生成一件、增加收纳盒/配件/食物 | 包装清单、订单可交付内容 |
| 尺寸和能力夸大 | 人手/房间比例错误、虚构承重、防水、清洁效果 | 尺寸、测试报告、客观主张证据 |
| 不真实使用场景 | 商品处于不支持的环境或由不存在的用户背书 | 使用说明、安全限制、人物授权 |
| 包装文字幻觉 | 虚构认证、成分、保质期、警示、Logo | 实际包装、合规文件、商标许可 |
| 权利侵害 | 复制他人图片、人物、角色、品牌或设计 | 原图来源、下载/修改/模型上传/商业使用许可 |
| AI 标记丢失 | 压缩、裁剪、转码清除 IPTC/C2PA | 原始输出、导出文件元数据扫描、哈希 |
| 主图违规 | 加场景、文字、边框、水印、非实际商品 | 目标渠道 + 站点 + 类目主图规则快照 |
| 广告与落地页不一致 | 创意展示 A，落地页/实际交付是 B | 创意版本、商品页版本、SKU 与落地页快照 |

## 4. 凌镜应如何做

### 4.1 发布前硬闸门

以下任一项不满足，Provider 可以生成内部候选图，但候选不得进入渠道草稿；涉及权利或敏感资料缺失时，连外部 Provider 调用也应阻断（`inferred`）：

1. 已绑定明确的国家/地区、渠道、站点、类目、图片用途（主图/副图/广告）；
2. 已绑定真实 SKU、变体、包装清单、尺寸、颜色、材料和已验证原图；
3. 已分别确认原图的下载、修改、上传第三方模型和商业发布权；
4. 保存目标渠道规则 URL、访问时间、原文摘要与不可变哈希；
5. 逐项检查模型是否改变商品事实或生成未交付内容；
6. 导出文件满足尺寸、背景、文字、水印、元数据和格式规则；
7. 广告中的每个客观或暗示性效果主张均有发布前证据；
8. Owner 审核绑定完整文件哈希、SKU、用途、渠道和版本；换图或重新转码后重新审核。

### 4.2 不应采用的“一刀切”规则

- 不应默认“AI 图片都要在画面上加字”：eBay 和 Google 主图对叠加文字/水印有限制，而 Google 要求的是文件元数据；TikTok 则提供平台标签。（`inferred`）
- 不应默认“背景替换永远无需披露”：TikTok把它列为轻微编辑，但 Google 对所有生成式 AI 图片要求保留来源元数据；EU/地区法还需具体判断。（`quoted` + `inferred`）
- 不应默认“副图就可以虚构”：副图可允许场景和插画，但仍不得误导商品、配件、效果或交付内容。（`inferred`）
- 不应把平台 `submitted/approved` 记为合规事实。它最多是平台当次处理结果，真实性、权利和法律合规仍需独立证据。（`inferred`）

### 4.3 推荐数据与验收输出

每张图片至少保存（`inferred`）：

- 原始输入与输出文件、SHA-256、像素、格式和文件大小；
- Provider、模型/版本、提示词、参数、请求 ID、生成时间和费用；
- AI 来源元数据/C2PA/IPTC 的生成前后扫描结果；
- 操作类型与实际改变区域；
- 真实 SKU/变体/包装清单关联；
- 源图四项权利证据；
- 渠道/站点/类目/用途及规则快照；
- 商品事实逐项人工核验结果；
- Owner 的选择、拒绝原因和审批哈希；
- 实际发布文件哈希、平台响应及后续下架/拒登记录。

## 5. 当前仍未知

- `unknown`：Amazon、eBay、Meta、Shopify 是否在特定站点、类目或账号界面存在未公开或需登录查看的额外 AI 图片规则。
- `unknown`：未来实际选择的国家、渠道和商品类目，因此本文不能替代该次任务的最新规则核验。
- `unknown`：某张具体 AI 商品图是否触发 EU AI Act 的 deep fake 披露义务。
- `unknown`：供应商图片是否具有下载、修改、上传模型和跨渠道商业使用的完整授权。
- `unknown`：平台在压缩或重新编码图片时是否保留所有来源元数据；应在真实渠道进行文件级实测。

## 6. 可执行裁决

第一版不需要建设“自动判定全球合规”的规则引擎。应建设一个**按渠道规则快照失败关闭的证据闸门**：规则未知时返回 `unknown/blocked`，而不是猜测允许；规则明确后才按主图、副图或广告用途生成和验收。这样既能支持未来任何被选中的渠道，也不会预设 Amazon、TikTok 或其他平台是当前方向。（`inferred`）
