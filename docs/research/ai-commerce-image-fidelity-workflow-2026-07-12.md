# AI 电商作图：商品保真风险与最小安全工作流

> 调研日期：2026-07-12（Asia/Shanghai）
> 范围：生成式 AI 对真实商品图的背景替换、场景化、构图调整；不讨论普通裁剪、缩放、白底抠图等确定性处理。
> 产品边界：只服务凌镜 Owner 自营商品实验；不建设外部 SaaS，不自动发布。
> 证据等级：`quoted` 为来源直接陈述；`inferred` 为基于来源的工程裁决；`unknown` 为必须用真实 SKU、渠道或账号核验的事项。

## 1. 结论

AI 电商作图不是“把提示词交给模型并保存结果”，而是：

> 用已获权利的真实商品图作为冻结输入，只允许模型修改明确遮罩区域；系统保存完整请求和输出字节，再用确定性检查、辅助模型检查和 Owner 对照实物/规格的人工验收，裁决候选图能否进入内部草稿。

最重要的风险是**商品事实被悄悄改变**。Google 官方将“单个商品换场景且换角度”标为低成功率，把多商品保身份、精确颜色/光线/风格同时保持列为非目标用法；OpenAI 官方也列出幻觉、属性绑定、编辑精度、小字和多语言文字等限制。商品图即使看起来更好，也不能证明颜色、数量、包装、文字、配件或尺寸关系仍真实。[Google Imagen subject customization](https://cloud.google.com/vertex-ai/generative-ai/docs/image/subject-customization)；[OpenAI 4o image generation limitations](https://openai.com/index/introducing-4o-image-generation/)

第一版应只验证一个闭环：

```text
已批准实验 + 一个真实 SKU + 一张有权使用的源图
→ 冻结源图、遮罩、模型、参数、预算和用途
→ 最多生成 3 张“只换背景、不改主体”的候选
→ 下载原始字节并计算 SHA-256
→ 自动阻断明显变化
→ Owner 对照源图、实物/可靠规格逐项验收
→ 选中图进入现有 1688 内部草稿
→ 草稿仍为 draft，换图使旧审批失效
```

## 2. 已核实的能力边界

| 声明 | 等级 | 对凌镜的含义 |
|---|---|---|
| Imagen 支持遮罩编辑、背景替换；输入和输出均可能被安全过滤，返回图数可能少于请求数 | `quoted` | “HTTP 成功”不等于返回了完整、可用的图片集合；必须逐张校验 |
| Google 称产品场景化保身份可能低成功；产品细节不要求精确保留的变化才是预期用法之一 | `quoted` | 不得把 reference image 或 prompt 当作身份保持保证 |
| OpenAI 明列 cropping、hallucinations、attribute binding、editing precision、dense/small text、multilingual text 等限制 | `quoted` | 包装、标签和多属性商品属于高风险；文字不能只靠肉眼缩略图验收 |
| OpenAI 2025 发布说明称详细图片可能生成长达约一分钟 | `quoted`（历史型号表现） | UI 和任务状态必须支持长延迟；不能同步阻塞或把客户端超时当失败事实 |
| Imagen 可用 seed 产生确定性输出，但 seed 与数字水印不能同时启用，返回顺序也不保证 | `quoted` | 不能只靠“重跑”；必须保存每个实际输出字节和哈希，并按哈希而非数组序号审批 |
| Google 负责 AI 指南称输出可能意外、错误或具误导性，视觉描述/VQA 也可能不准确和过度自信 | `quoted` | 不能用同类 AI 检查器单独证明商品真实；它只能辅助筛查 |
| Firefly/OpenAI 会附加 Content Credentials/C2PA；C2PA 规范承认裁剪、非兼容工具处理、元数据移除或损坏可使资产与 manifest 分离 | `quoted` | 元数据是辅助来源证据，不是唯一审计事实；系统需自行保存原文件、凭证检测结果和哈希 |
| 最新产品保真研究提出专门数据和 benchmark，报告现有开闭源编辑模型仍难保持品牌与细粒度文字 | `quoted`（2026 预印本，未视为独立复现） | “通用视觉质量高”不能替代 SKU 级保真测试 |

## 3. 风险矩阵

| 风险 | 影响 | 发生可能性 | 最小检测 | 放行规则 |
|---|---:|---:|---|---|
| 商品身份、结构、接口、纹理改变 | 极高 | 高 | 主体遮罩内像素差、局部特征比对、Owner 对照实物/规格 | 任一经营关键结构无法确认即阻断 |
| 颜色或材质被光照“美化” | 极高 | 高 | 色卡/可靠源图基准；主体区域 Lab 色差；人工确认材质反光 | 超阈值或肉眼可能误导即阻断；阈值需按真实 SKU 校准，当前 `unknown` |
| 数量、配件、包装内容增删 | 极高 | 中高 | 目标检测/分割计数 + 包装清单 + 人工逐项勾选 | 数量或附件不一致直接阻断 |
| 品牌、包装文字、规格、警示语乱码或改写 | 极高 | 高 | 原图与输出 OCR 逐字符 diff；原尺寸 100% 检视 | 关键文字必须逐字符一致；否则回退确定性合成 |
| 尺寸感、比例、使用效果被夸大 | 高 | 中高 | 轮廓/关键点比例、参照物合理性、人工检查 | 无真实尺寸依据的参照物或效果不得出现 |
| 背景违反渠道规则或遮挡主体 | 高 | 中 | 渠道规则快照 + 分割占比/边距/背景色检查 | 规则未确定为 `unknown` 时不能用于真实渠道 |
| 幻觉人物、手指、道具、徽标或声明 | 高 | 中高 | 对象检测、OCR、品牌/声明词扫描、人工检视 | 未在任务卡允许的对象一律拒绝 |
| 安全过滤导致少图、空结果或拒绝 | 中 | 中 | 按响应逐项检查结果、过滤原因和错误码 | 不得自动改写提示词绕过；Owner 决定是否修改任务 |
| 超时后重复请求、重复扣费 | 中高 | 中 | 业务幂等键、外部请求 ID、请求哈希、对账状态 | 状态不明时先 reconcile，禁止盲目重试 |
| 模型或提示词重写变化导致不可复现 | 高 | 中高 | 固定模型版本、记录 seed/完整参数/重写开关与最终提示词；保存原字节 | 不能复现不影响已保存输出审计，但禁止事后重生成替代已批准文件 |
| 压缩、裁剪或平台上传使 C2PA/元数据丢失 | 中 | 高 | 每个处理阶段解析 C2PA；保存原始 manifest/验证结果/哈希 | 凭证缺失不自动等于伪造，也不自动等于可信；内部事实链独立保存 |
| 自动视觉检查误报或漏报 | 高 | 中高 | 固定测试集、人工复核抽样、记录模型版本和置信度 | 自动检查只可阻断或提示，不能单独升级为 `actual` |
| 延迟、失败和返工令“每张成本”失真 | 中 | 高 | 记录每次请求、图数、费用、延迟、重试、人工分钟数 | 以“最终采用图总成本”计算，不以单次 API 标价计算 |
| 图片好看但没有经营效果 | 中高 | 高 | 渠道内受控 A/B；冻结价格、广告、库存和履约等变量 | 先通过真实性，再讨论点击/转化；利润、退货和争议需后续独立对账 |

## 4. 检测分层：机器先筛，Owner 最终裁决

### 4.1 确定性检查（必须）

1. 解码文件、MIME、尺寸、色彩空间、透明通道和文件大小；拒绝损坏、空白或格式不符的输出。
2. 保存 Provider 原始响应、原始图片字节、SHA-256、模型 ID、请求 ID、完整参数、源图/遮罩哈希、开始/结束时间、费用或计量单位。
3. 对主体保护区计算像素差和结构差；对只换背景任务，主体区有实质变化应默认阻断，而不是打低分继续。
4. 对包装和标签区域做 OCR，与冻结的真实文字清单逐字符比较；品牌、型号、容量、数量、合规警示任何变化均阻断。
5. 对商品/配件做检测、分割和计数；数量变化或新增未知对象阻断。
6. 对关键颜色区域计算 Delta E 等色差指标，但阈值必须用该 SKU 的真实照片和渠道压缩样本校准；尚未校准时为 `unknown`，不能假装通用阈值可靠。
7. 每次裁剪、压缩或转码后重新哈希，并记录父文件哈希形成派生链。

### 4.2 辅助模型检查（只能提供信号）

- 用独立视觉模型逐项回答“颜色/数量/文字/配件是否变化”，并要求给出不确定项；保存版本和原始回答。
- 不把高置信度当事实。Google 明确指出 VQA 可能过度自信，caption 可能遗漏复杂图像语境。[Google responsible AI for Imagen](https://cloud.google.com/vertex-ai/generative-ai/docs/image/responsible-ai-imagen)
- 不能让生成模型自己给自己的输出签字放行；不同 Agent 一致仍只是参考信号。

### 4.3 Owner 人工验收（必须）

Owner 必须看原尺寸文件，并同时看到源图、实物或可靠规格，不只看缩略图。逐项确认：

- 商品是否同一 SKU；
- 颜色、材质、结构、接口和比例；
- 商品数量、包装数量和全部配件；
- 品牌、型号、容量、标签、警示语；
- 背景是否暗示不存在的使用效果、尺寸或赠品；
- 目标渠道规则是否满足；
- AI 来源披露或凭证要求是否满足。

批准必须绑定：`output_sha256 + sku_id + experiment_id + target_channel + intended_use + rule_snapshot_id + reviewer + reviewed_at`。任何字节或用途改变都必须重新审批。

## 5. 最小安全工作流

1. **任务闸门**：实验 active、opportunity 已通过、源图来自不可变快照、四类图片权利已确认、渠道规则快照存在、预算充足；否则 Provider 调用次数为 0。
2. **冻结任务卡**：只允许一种 edit intent；首版为 `replace_background_only`。冻结源图、保护主体遮罩、提示词、负面约束、模型、版本、输出数（最多 3）、预算和超时策略。
3. **创建幂等任务**：以 Owner、SKU、源图哈希、遮罩哈希、规范化参数和任务版本生成业务幂等键；同键已有不确定/成功任务时禁止再次收费调用。
4. **调用 Provider**：异步执行；保存外部请求 ID。安全拒绝为业务拒绝，不自动规避；429/5xx 仅在确认未生成或可对账后指数退避并带抖动重试。
5. **获取与固化**：立即下载每个结果的原始字节；校验格式；写受控对象存储；计算哈希。只保存临时 URL 不合格。
6. **机器检查**：按 4.1 执行；任何硬性事实冲突进入 `blocked`，不得送入草稿。
7. **Owner 复核**：并排原图/候选图/规格/检查差异；Owner 可选中一张或全部拒绝。未选择即无可用图片。
8. **进入草稿**：选中输出生成派生媒体记录；草稿始终 `draft`；图片变化使旧内容哈希和审批失效；严禁调用平台发布适配器。
9. **真实渠道前复核**：按最新规则重新验证转码后的最终字节；提交成功只记 `submitted`，不代表上线、合规或经营有效。
10. **经营实验**：只有真实性和渠道规则都通过后，才能用渠道原生实验能力或受控流量做 A/B；一次只改变图片变量，预先定义指标和停止条件。

## 6. 重试、延迟与费用

- OpenAI 2025 官方示例价格约为低/中/高质量方图 `$0.02/$0.07/$0.19`，但价格会变，且输入图片、重试、存储和人工不包含在“单张输出”口径中；真实费用必须以账单和本次请求记录为准，当前账号可用价格为 `unknown`。[OpenAI image generation API announcement](https://openai.com/index/image-generation-api/)
- Google 当前公开价格页列 Imagen 3 生成/编辑/定制约 `$0.04/图`；同样只是公开标价，不证明凌镜账号、地区或折扣后的账单。[Vertex AI pricing](https://cloud.google.com/vertex-ai/generative-ai/pricing)
- 最有意义的成本指标是：

```text
最终采用图总成本
= 全部生成请求费用 + 失败/重复费用 + 下载/存储/检查费用
  + Owner 审核与返工时间成本
```

- 每项任务至少记录 `provider_latency_ms`、`download_latency_ms`、重试次数、返回/阻断/采用图数和 Owner 审核分钟数。
- 超时状态分为 `provider_definitely_failed` 与 `reconcile_required`。后者不可直接重试；只有 Provider 能确认未受理，或已完成结果对账后才可继续。

## 7. A/B 经营验证

Amazon 的官方 Seller Central 公告说明 Manage Your Experiments 支持主图 A/B，并提示两个版本需有足够差异才更可能得出结论；这证明“平台内随机分流实验”是可用模式，但不证明当前 Owner 账号、类目或渠道具备资格。[Amazon Manage Your Experiments announcement](https://sellercentral.amazon.com/seller-forums/discussions/t/8f353c9981adf8dba4423d6c455da286)；[Amazon experiment updates](https://sellercentral.amazon.com/seller-forums/discussions/t/d5e56fe5-fa97-459d-9021-e23e820c10e7)

凌镜应按以下顺序判断：

1. **制作闭环**：一张真实源图能否在预算和时间上限内生成至少一张 Owner 批准、事实无变化的草稿图。
2. **渠道可用**：最终转码文件是否被目标渠道接受并真实展示；接受不等于合法或效果好。
3. **经营效果**：只有流量足够且渠道支持随机分流时，比较原图与 AI 图；冻结标题、价格、优惠、广告、库存、履约承诺等混杂变量。
4. **指标分层**：主指标预先只选一个（如转化率）；同时观察点击、退货、争议、差评和最终贡献利润，不能用点击提升掩盖退货或利润恶化。
5. **不追逐噪声**：预先设样本量/最长期限/停止规则；中途频繁看数并换图会扩大假阳性。真实阈值取决于基线流量与最小可接受效果，当前为 `unknown`。

## 8. 第一轮停止条件

出现任一项即停止该次任务，不继续“多调几轮提示词”：

- 无真实 SKU、源图权利、可靠规格或渠道规则快照；
- 一次出现颜色、结构、数量、包装文字、型号、配件等严重商品事实变化；
- 三张候选全部未通过 Owner 核验；
- 达到预设费用或 Owner 审核时间上限；
- Provider 超时且无法对账；
- 安全政策拒绝，除非 Owner 明确修改合法任务内容；
- 自动检查无法解释或与 Owner 判断冲突；
- 最终字节无法绑定哈希、来源、模型和审批；
- 渠道实验无法隔离图片变量，或流量不足以支持预设判断。

失败后默认回退到现有确定性图片处理，不扩张到更多模型、批量作图或自动发布。

## 9. 仍未知、必须用真实输入验证

| 未知 | 最小验证 |
|---|---|
| 哪个模型对首个 SKU 最保真 | 同一冻结任务卡，各 Provider 最多 3 张；盲化人工评分 + 硬性事实检查 |
| 主体保护区允许多大像素/色差 | 用真实商品、实物/色卡、渠道压缩样本建立 SKU 级阈值 |
| OpenAI/Google/Adobe 当前账号可用性、地区、账单和数据设置 | 在 Owner 授权账号中读取控制台/合同并做一笔上限明确的真实调用 |
| 目标渠道是否要求 AI 披露、是否保留 C2PA | 保存带凭证测试图，在草稿或安全测试入口转码/上传后重新下载检查；不得据别的渠道推断 |
| AI 图是否改善经营结果 | 通过真实性后，在同一 SKU、同一时段的随机分流实验验证 |

## 10. 主要来源

- [OpenAI：4o image generation capabilities, limitations, latency and C2PA](https://openai.com/index/introducing-4o-image-generation/)
- [OpenAI：gpt-image-1 API and indicative pricing](https://openai.com/index/image-generation-api/)
- [OpenAI：API data controls for image endpoints](https://platform.openai.com/docs/models/default-usage-policies-by-endpoint)
- [Google：Imagen product/subject customization intended and unintended uses](https://cloud.google.com/vertex-ai/generative-ai/docs/image/subject-customization)
- [Google：Imagen editing and safety filtering](https://cloud.google.com/vertex-ai/generative-ai/docs/image/edit-images-overview)
- [Google：Responsible AI limitations for Imagen](https://cloud.google.com/vertex-ai/generative-ai/docs/image/responsible-ai-imagen)
- [Google：deterministic generation, seed/watermark trade-off](https://cloud.google.com/vertex-ai/generative-ai/docs/image/generate-deterministic-images)
- [Adobe：Content Credentials in Firefly](https://helpx.adobe.com/firefly/web/get-started/learn-the-basics/content-credentials-overview.html)
- [C2PA：Specification explainer, metadata removal and durable credentials](https://spec.c2pa.org/specifications/specifications/2.3/explainer/_attachments/Explainer.pdf)
- [EditVal：text-guided editing evaluation and property-preservation findings](https://arxiv.org/abs/2310.02426)
- [ProductConsistency：product identity/OCR preservation benchmark](https://arxiv.org/abs/2606.19103)（2026 预印本）

## 11. 证据边界

本次是外部资料调研，没有调用任何图片 Provider，没有获得真实 SKU、实物、图片权利、渠道账号或账单，也没有做视觉对比实验。因此：

- 官方能力与限制是 `quoted`；
- 风险矩阵、状态和工作流是 `inferred` 工程裁决；
- 模型对凌镜首个真实商品的保真率、成本、延迟和经营效果均为 `unknown`；
- 论文 benchmark 不能替代凌镜自己的真实 SKU 验收；
- 任何模型返回、自动检查通过或多个 Agent 一致，都不能将图片升级为真实商品事实。
