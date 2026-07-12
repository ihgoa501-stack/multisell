# AI 商品图片生成/编辑：供应商与渠道外部约束调研

> 调研日期：2026-07-12（Asia/Shanghai）
> 访问日期：下列网页均于 2026-07-12 访问
> 范围：程序化商品图片生成/编辑的成熟外部方案与渠道约束；不代表已购买、已调用 API、已取得商用授权或已选定销售渠道
> 证据等级：官方网页内容为 `quoted`；本文方案判断为 `inferred`；真实账号、合同、输出质量和渠道审核结果为 `unknown`

## 1. 结论

**`inferred`：当前最适合凌镜的第一步不是绑定某一家供应商，而是把“商品原图的受控编辑”定义成可替换 Provider 的能力。** 第一版只允许：从已核验图片权利的真实商品原图出发，做去/换背景、扩图、局部清理和受控场景合成；不得让模型凭空重画商品主图并冒充真实商品。

原因：

1. 三家成熟方案都能程序化生成和编辑，但模型不能保证 SKU 的形状、颜色、数量、标签、配件和文字完全不变。
2. 渠道规则可能直接要求主图是“实际商品的专业照片”。例如 Amazon 官方规则明确排除主图中的插画、mockup、占位图、文字和水印；所以“模型生成得像”不等于“渠道允许”。
3. 权利、隐私、内容凭证和费用都依赖具体产品、账号、区域与合同；供应商宣传不能代替 Owner 对源图权利和最终渠道规则的核验。

推荐的试验顺序（均为 `inferred`）：

- **首个技术 PoC：OpenAI Images**。单一 API 同时覆盖生成、编辑、多图参考、局部蒙版和多轮编辑，接入门槛相对低；但无 seed/严格几何约束，不能作为商品真实性裁决者。
- **需要 Google Cloud 数据地域或可复现 seed 时：Vertex AI Imagen**。Imagen 支持区域、GCS、SynthID 和在关闭水印时使用 seed；但商品保真仍须逐 SKU 人工验收。
- **需要企业合同、Adobe 工作流、Content Credentials 或特定 IP 赔偿条款时：Firefly Services**。不要仅因“commercially safe”宣传就默认拥有赔偿；资格取决于合同、功能、导出事件和产品说明。

在 Owner 尚未批准目标市场和销售渠道前，不把 Amazon 规则固化为全局规则；应保存为 `channel_rule_snapshot`，最终按“国家/地区 × 渠道 × 类目 × 观察时间”裁决。

## 2. 对凌镜而言，什么是“AI 作图”

**`inferred` 定义：** AI 作图是外部或本地生成式模型根据文本、真实图片、蒙版和结构/风格参考，生成或修改栅格图像的过程。对当前系统，它是一个**受控图片处理步骤**，不是商品事实来源，也不是图片权利证明。

应区分四类：

| 类型 | 输入 | 典型用途 | 当前建议 |
|---|---|---|---|
| 文生图 | 文本 | 概念草图、内部创意 | 不得作真实商品主图 |
| 图生图/参考生成 | 商品原图 + 文本/参考图 | 场景图、风格变化 | 可试验，必须比对商品事实 |
| 局部编辑 | 原图 + 蒙版 + 指令 | 去物、补边、修背景 | 第一版优先 |
| 合成/扩图/抠图 | 商品主体 + 新背景/画布 | 白底图、生活方式图 | 第一版优先，但保存原图和差异 |

AI 输出的默认真实性应为 `inferred`；只有其来源、处理链和视觉检查被记录，才可声明“处理发生”。即使人工通过，也只说明图片与输入商品在检查项上相符，不证明图片权利或渠道已接受。

## 3. 方案比较

| 维度 | OpenAI Images | Google Vertex AI Imagen | Adobe Firefly Services |
|---|---|---|---|
| 成熟 API | Image API：生成、编辑；Responses API：多轮图像编辑 | REST/SDK；Imagen 4 生成，Imagen 3 编辑/定制 | REST；生成、填充、扩图、合成、放大、自定义模型 |
| 输入 | 文本；图像 bytes/Base64/URL/File ID；多图参考；蒙版 | 文本；基础图、蒙版、参考/控制图；可写入 GCS | 文本；上传后的 image ID 或受限来源预签名 URL；参考图 |
| 输出 | Base64；PNG/JPEG/WebP；尺寸、质量、压缩 | Base64 bytes 或 GCS；MIME、压缩、宽高比 | 异步 job/输出 URL；具体格式按端点/schema |
| 商品相关优势 | 多图参考、较强指令和文字表现、多轮编辑 | 区域控制、背景替换、seed、SynthID | 官方提供产品图规模化教程、对象合成、Photoshop/Lightroom 工作流 |
| 可复现性 | 文档未提供 seed；官方承认跨生成的一致性仍有限 | 官方称 seed 可使输出确定，但必须关闭支持模型的水印 | 查阅的公开 API schema 未确认通用 seed，`unknown` |
| 价格口径 | 按文本输入、图片输入、图片输出 token；当前文档给出 GPT Image 2 方图约 $0.006/$0.053/$0.211（低/中/高），编辑另计输入 | 按张：Imagen 4 Fast/4/4 Ultra 为 $0.02/$0.04/$0.06；Imagen 3 生成/编辑/定制 $0.04 | 公开开发文档未给出可直接核验的统一按张价；需 Adobe 销售/合同，`unknown` |
| 数据/训练 | API 输入输出默认不用于训练；保留期限/ZDR 资格需按账号核验 | 未经许可或指示不用于训练；滥用监控等场景仍可能有限保留，ZDR 有条件 | 本次未找到覆盖 Firefly Services API 的单一公开数据保留表；须合同/DPA 核验，`unknown` |
| 凭证/水印 | OpenAI 2025 官方发布称 `gpt-image-1` 输出含 C2PA；当前 GPT Image 2 API 指南未重申每个输出的具体凭证行为，需实测/合同核验 | `addWatermark` 默认 true 时加入 SynthID；可关闭；seed 要求关闭水印 | Adobe 表示 Firefly 及其 API 自动应用 Content Credentials；纯生成内容会附元数据，且可能存入公开凭证云 |
| 商用/版权 | API 条款与输出权利仍需 Owner/法务按现行服务协议核验；无“不侵权”保证的证据 | 需按 Google Cloud 服务条款和生成式 AI 条款核验；本次不宣称输出不侵权 | 指定合同和合格 Firefly 功能/导出事件可能有 IP 赔偿；不是所有账号/输出自动适用 |
| 速率/异步 | 账号 tier 的速率限制需登录后核验；复杂 prompt 官方称可达约 2 分钟 | 配额按项目/区域/模型，需账号核验；可直接返回或写 GCS | 官方默认 4 RPM、9000 RPD；新 API 示例为异步 job；更高限制联系客户经理 |
| 锁定风险 | 模型名、token 计价、Files/Responses 语义 | GCP IAM、区域、GCS、模型版本 | Adobe Admin Console、OAuth、组织授权、Adobe 工作流和合同 |

### 3.1 OpenAI Images — 官方事实

- `quoted`：当前指南列出 `gpt-image-2`、Image API 的生成/编辑端点，以及 Responses API 的多轮编辑；输出可配置尺寸、质量、格式和压缩。[Image generation guide](https://developers.openai.com/api/docs/guides/image-generation)
- `quoted`：Image API 返回 Base64，默认 PNG，也可 JPEG/WebP；复杂提示可能处理约 2 分钟，文字、跨图一致性和精确构图仍有局限。[Image generation guide](https://developers.openai.com/api/docs/guides/image-generation)
- `quoted`：当前指南的估算表中，GPT Image 2 的 1024×1024 输出约为低 $0.006、中 $0.053、高 $0.211；总成本还包括文本输入和编辑时的图片输入 token。[Image generation guide](https://developers.openai.com/api/docs/guides/image-generation)
- `quoted`：OpenAI 默认不使用 API 输入输出训练模型；符合条件的组织可配置保留期或申请零数据保留。[How data is used](https://openai.com/policies/how-your-data-is-used-to-improve-model-performance/)、[Business data](https://openai.com/business-data/)
- `quoted`：2025 年 `gpt-image-1` 发布说明称输出包含 C2PA 元数据；这不能自动外推为当前每个模型、格式和后处理链都保留凭证。[API image launch](https://openai.com/index/image-generation-api/)
- `unknown`：当前账号的组织验证、速率、数据保留资格、可用区域、人民币税费与最终合同权利。

### 3.2 Google Vertex AI Imagen — 官方事实

- `quoted`：Imagen 可文本生成；Imagen 编辑支持插入、移除、扩图和背景替换；可把 Base64 图像放在响应中或写入 Cloud Storage。[Generate images](https://cloud.google.com/vertex-ai/generative-ai/docs/image/generate-images)、[Edit images](https://cloud.google.com/vertex-ai/generative-ai/docs/image/edit-images-overview)
- `quoted`：`addWatermark` 默认 true 时加入可验证的 SynthID；要使用 seed 得到确定性输出，必须关闭水印。[Generate images](https://cloud.google.com/vertex-ai/generative-ai/docs/image/generate-images)
- `quoted`：官方价格表列 Imagen 4 Fast/4/4 Ultra 每张 $0.02/$0.04/$0.06，Imagen 3 生成/编辑/定制每张 $0.04；实际账单仍以区域 SKU 和账号为准。[Vertex AI pricing](https://cloud.google.com/vertex-ai/generative-ai/pricing)
- `quoted`：Google 表示未经客户事先许可或指示，不会用 Vertex AI 客户数据训练/微调模型；但滥用监控等场景可能有限保留，零保留需要满足文档条件。[Zero data retention](https://cloud.google.com/vertex-ai/generative-ai/docs/vertex-ai-zero-data-retention)
- `quoted`：Google 对 Imagen 3 商品参考图能力明确提示，某些“把商品放入不同场景并精确保留细节”的用例成功率低，且多商品/精确构图并非预期强项。[Subject customization](https://cloud.google.com/vertex-ai/generative-ai/docs/image/subject-customization)
- `unknown`：目标区域是否支持所需模型、项目配额、最终服务条款中的输出权利及赔偿范围。

### 3.3 Adobe Firefly Services — 官方事实

- `quoted`：Firefly API 提供生成、填充、扩图、编辑、放大和产品对象合成；支持自定义主体/风格模型。[Firefly API overview](https://developer.adobe.com/firefly-services/docs/firefly-api/)
- `quoted`：API 使用 OAuth server-to-server 和 API key；token 约 24 小时；生成示例使用异步端点。[Authentication](https://developer.adobe.com/firefly-services/docs/firefly-api/getting-started/)、[Quickstart](https://developer.adobe.com/firefly-services/docs/firefly-api/guides/)
- `quoted`：默认每组织 4 RPM、9000 RPD，超限返回 429，可按 `retry-after` 或退避重试；提高限制需联系客户经理。[Usage notes](https://developer.adobe.com/firefly-services/docs/firefly-api/getting-started/usage-notes/)
- `quoted`：Firefly 及 API 生成内容自动应用 Content Credentials；Adobe 说明纯 Firefly 生成资产的凭证会附着文件，并可能存储在公开 Content Credentials 云。[Creative Cloud Content Credentials](https://helpx.adobe.com/ca/creative-cloud/apps/adobe-content-authenticity/content-credentials/overview.html)、[Firefly Content Credentials](https://helpx.adobe.com/firefly/web/get-started/learn-the-basics/content-credentials-overview.html)
- `quoted`：Adobe 的 IP 赔偿只适用于协议链接到该产品说明、列出的合格功能/表面/导出事件并满足条款的客户；不能把宣传语当作自动保险。[Firefly product description](https://helpx.adobe.com/legal/product-descriptions/adobe-firefly.html)、[Generative AI product terms](https://www.adobe.com/cc-shared/assets/pdf/legal/servicetou/adobe-generative-ai-product-specific-terms-en-us-20260423.pdf)
- `unknown`：Firefly Services 的公开统一单价、当前 Owner 可购买套餐、API 数据保留/训练细节、赔偿资格及金额上限；必须拿正式报价、DPA 和订单条款核验。

## 4. 渠道规则：以 Amazon 为已调查样例，不预设最终渠道

### Amazon 官方规则快照

- `quoted`：Amazon 官方 Seller Central 说明，图片必须准确代表在售商品；主图应是**实际商品的专业照片**，纯白背景 RGB 255/255/255，不允许 graphics、illustrations、mockups、placeholders、混淆性道具、非商品文字、logo、水印或 inset image；类目规则冲突时类目规则优先；卖家必须拥有所提交图片的必要权利。[Amazon Product image requirements（官方内容镜像在 Seller Forums）](https://sellercentral.amazon.com/seller-forums/discussions/t/7366420bc9ccfb8656594e6edcf4ece6)
- `quoted`：Amazon 官方社区公告还列出图片最长边 500–10,000 像素，接受 JPEG/TIFF/PNG/非动画 GIF；不合规图片可能被拒绝、移除、修改或导致搜索抑制。[Listings Lounge: Product Image Requirements](https://sellercentral.amazon.com/seller-forums/discussions/t/4b3c4c39-6f8c-4312-aa0e-99982eb8f5e1)

**`inferred`：** 对 Amazon 样例，AI 生成的“虚构商品主图”风险不可接受；白底处理只能以真实商品照片为基底，并核对产品轮廓、颜色、数量、包装、随附配件、标签和比例。生活方式图也不能增加未包含配件或暗示不存在的功能。

### 不能泛化的地方

- Amazon 美国站样例不等于其他国家/渠道/类目的规则。
- 当前候选市场仍是 `evidence_missing`，Amazon US 未被批准为最终渠道。
- 上线前必须重新抓取目标站点、类目和图片位置（主图/附图/广告）的当日官方规则；保留 URL、观察时间、原始快照与 SHA-256。

## 5. 系统结合建议

### 5.1 Provider 中立契约（`inferred`）

系统只定义业务语义，不把 OpenAI/GCP/Adobe 字段扩散到领域层：

```text
ImageTransformationRequest
  operation: background_remove | background_replace | expand | cleanup | composite | concept_generate
  source_asset_ids[]
  mask_asset_id?
  prompt
  target_spec: width, height, format, background, channel_rule_snapshot_id
  provider_config: provider, model, region, quality, seed?
  experiment_id + sourcing_snapshot_id + owner_id

ImageTransformationResult
  immutable_input_hashes[] + prompt_hash
  provider + model/version + request_id
  output_asset_hash + mime + dimensions
  cost_amount/currency + cost_basis
  provenance: C2PA/SynthID/Content Credentials detection result
  status: submitted | completed | failed | reconcile_required
  verification: pending_owner_review | passed | rejected
```

### 5.2 必须保留的闸门（`inferred`）

1. **源图权利闸门**：没有来源 URL、抓取时间、许可/授权证据和商品快照，不允许送给外部模型。
2. **商品事实闸门**：生成后由 Owner 对颜色、形状、数量、文字、包装、配件、尺寸暗示逐项检查；AI 自检不能通过该闸门。
3. **渠道规则闸门**：目标渠道、国家和类目规则缺失时为 `unknown`，不得批准为渠道草稿。
4. **外部写审批**：生成/编辑是可逆内部操作；真实上传/发布继续沿用独立 Owner 审批、冻结请求、哈希和 `submitted/reconcile_required` 语义。
5. **隐私和凭证闸门**：禁止上传包含个人信息、供应商后台秘密、未授权人物或敏感文档的图片；记录区域和数据保留配置。
6. **成本闸门**：请求前估算上限，结果写实际费用；超过单案/单日预算停止，不自动重试内容错误。

### 5.3 与现有系统的边界（`inferred`）

- 应接在现有 `internal/domain/sourcing1688/` 的“实际图片处理”步骤中，关联 `experiment_id`、不可变来源快照和草稿审批；不要创建独立 SaaS、公共 API 或自治图片 Agent。
- 小Q若未来支持，只能通过登记 Capability 发起“创建内部图片处理草稿/读取状态”，不能自行批准商品事实、图片权利或渠道发布。
- 先做一类操作的真实闭环：**已授权真实商品图 → 白底/清理 → Owner 差异检查 → 渠道草稿验收报告**。在一件真实商品上通过前，不增加批量生成、品牌自定义模型或多供应商路由。

## 6. 供应商锁定与可迁移性

`inferred`：保存原始输入、通用操作类型、规范化 prompt、蒙版、目标规格、模型版本、输出哈希和人工验收，而不是只保存供应商 URL。供应商 URL 会过期，模型别名会变化，生成也通常不能逐像素重放。

最小迁移测试应固定同一组真实商品与验收清单，在不同 Provider 上比较：

- 商品事实错误率（最重要）；
- Owner 一次通过率；
- 渠道规则阻断率；
- 单张总成本和 P95 延迟；
- 失败/内容过滤/超时率；
- 内容凭证保留率；
- 数据地域和合同条件。

不要把三家输出“看起来都好”当作经营证据；只有目标渠道实际接受、买家未被误导且最终经营结果完成，才进入后续证据链。

## 7. 当前未知与下一步验证

### `unknown`

- Owner 是否拥有候选 1688 图片的再处理、上传给第三方模型和渠道发布权利。
- 目标国家/渠道/类目，以及该渠道是否允许 AI 编辑主图或附图。
- 三家在当前 Owner 账号、结算地区、合同下的真实价格、税费、配额、数据保留和赔偿范围。
- 对真实 SKU 的文字、包装、材质、颜色和配件保真率。
- C2PA/SynthID/Content Credentials 经下载、转码、压缩和渠道上传后是否仍可检测。

### 最小验证（不在本次执行）

1. Owner 先批准一个市场/渠道组合和一个拥有清晰图片权利的真实商品。
2. 为单一白底/清理操作设预算上限；分别生成少量候选，不发布。
3. 用固定检查表人工验收，记录错误而不是只挑最好看的结果。
4. 用目标渠道草稿/验收接口验证技术规则；真实发布仍需独立审批。
5. 只有单件闭环通过，才决定是否购买配额或接第二 Provider。

## 8. 来源清单

所有来源均为供应商或渠道官方一手资料，访问日期均为 2026-07-12：

- OpenAI：[Image generation](https://developers.openai.com/api/docs/guides/image-generation)、[API image launch](https://openai.com/index/image-generation-api/)、[How data is used](https://openai.com/policies/how-your-data-is-used-to-improve-model-performance/)、[Business data](https://openai.com/business-data/)
- Google Cloud：[Generate images](https://cloud.google.com/vertex-ai/generative-ai/docs/image/generate-images)、[Edit images](https://cloud.google.com/vertex-ai/generative-ai/docs/image/edit-images-overview)、[Subject customization](https://cloud.google.com/vertex-ai/generative-ai/docs/image/subject-customization)、[Pricing](https://cloud.google.com/vertex-ai/generative-ai/pricing)、[Zero data retention](https://cloud.google.com/vertex-ai/generative-ai/docs/vertex-ai-zero-data-retention)
- Adobe：[Firefly API overview](https://developer.adobe.com/firefly-services/docs/firefly-api/)、[API reference](https://developer.adobe.com/firefly-services/docs/firefly-api/api/)、[Quickstart](https://developer.adobe.com/firefly-services/docs/firefly-api/guides/)、[Usage notes](https://developer.adobe.com/firefly-services/docs/firefly-api/getting-started/usage-notes/)、[Image upload](https://developer.adobe.com/firefly-services/docs/firefly-api/guides/concepts/image-upload/)、[Content Credentials](https://helpx.adobe.com/ca/creative-cloud/apps/adobe-content-authenticity/content-credentials/overview.html)、[Product description](https://helpx.adobe.com/legal/product-descriptions/adobe-firefly.html)、[Generative AI terms](https://www.adobe.com/cc-shared/assets/pdf/legal/servicetou/adobe-generative-ai-product-specific-terms-en-us-20260423.pdf)
- Amazon：[Product image requirements](https://sellercentral.amazon.com/seller-forums/discussions/t/7366420bc9ccfb8656594e6edcf4ece6)、[Listings Lounge technical summary](https://sellercentral.amazon.com/seller-forums/discussions/t/4b3c4c39-6f8c-4312-aa0e-99982eb8f5e1)
