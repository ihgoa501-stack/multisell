# AI 商品图片系统的长期价值、提示词资产与建设边界

> 日期：2026-07-13
> 范围：Owner 自用的凌镜商品图片能力
> 方法：仅采用官方产品文档、官方平台规则和研究论文原文。厂商能力描述标记为 `quoted`，不自动视为独立验证结果。

## 1. 结论

AI 作图是正在快速成熟的生产手段，但凌镜不应以“拥有一套提示词”作为核心资产。

更准确的长期资产是：

> **商品事实 + 获权原始素材 + 可版本化制作配方 + Owner 选择反馈 + 渠道适配规则 + 发布后的效果和错误数据。**

提示词属于“可版本化制作配方”的一个字段。它可能提高同一模型、同一版本、相同输入条件下的复用效率，但不能脱离模型、参考图、遮罩、参数、模板和验收标准单独承诺稳定复现。

对凌镜而言，值得建设的不是通用图片编辑器或自研基础模型，而是一个把 exact SKU 的可信素材转化为不同用途图片，并持续学习“什么制作方法对什么商品、渠道和目标有效”的内部生产与证据系统。

## 2. 证据等级

- `actual`：本次直接查到的官方接口、官方规则或论文原文所描述的可核对内容。
- `quoted`：厂商对自身产品效果的陈述；能证明其定位和接口，不代表凌镜真实效果已经验证。
- `inferred`：基于多份一手证据形成的产品或架构判断。
- `unknown`：必须用凌镜真实商品、账号和经营数据验证的问题。

## 3. 商品级提示词是不是可靠的核心资产

### 3.1 可以积累，但不能作为孤立资产

`actual`：Adobe Firefly 的生成接口显式要求调用方指定模型版本，并且不同模型版本支持的参数并不相同；例如 negative prompt、参考图、输出分辨率和 variation 数量存在版本差异。[Adobe Firefly API Reference](https://developer.adobe.com/firefly-services/docs/firefly-api/api/)

`actual`：Adobe 的接口把 prompt、结构参考图、风格参考图、强度、seed、尺寸、preset 和 custom model ID 分开建模；局部填充还需要原图与 mask，而不是只靠文字。[Adobe Firefly API Reference](https://developer.adobe.com/firefly-services/docs/firefly-api/api/)

`actual`：OpenAI 官方把上传图片和多轮上下文作为保持一致性的关键输入，同时明确列出裁切、幻觉、属性绑定和编辑精度等限制。[Introducing 4o Image Generation](https://openai.com/index/introducing-4o-image-generation/)

`actual`：已有研究把 prompt engineering 描述为反复试错问题；另一项研究提出把用户输入自动适配成“模型偏好的提示词”，说明相同意图需要面向模型进行转换，而非存在天然通用的固定咒语。[Design Guidelines for Prompt Engineering Text-to-Image Generative Models](https://arxiv.org/abs/2109.06977)、[Optimizing Prompts for Text-to-Image Generation](https://arxiv.org/abs/2212.09611)

`actual`：Midjourney 官方明确提示 V6 的旧 style codes 在 V7 可能不再产生相同风格；其模型版本说明也显示新版本对提示词的理解方式发生变化。[Midjourney Style Reference](https://docs.midjourney.com/hc/en-us/articles/32180011136653-Style-Reference)、[Midjourney Version](https://docs.midjourney.com/hc/en-us/articles/32199405667853-Version)

`actual`：Google Imagen 只在相同 seed、相同输入参数及固定模型条件下说明确定性生成，并且平台可对用户 prompt 做增强改写。这进一步说明要保存模型版本、完整参数和改写后的 prompt，而不是只存用户原句。[Google deterministic image generation](https://cloud.google.com/vertex-ai/generative-ai/docs/image/generate-deterministic-images)

`inferred`：商品提示词可以复用的是**意图、结构和约束**，不是某段文字在所有模型和版本上的像素级结果。例如“保留商品主体、生成浅色厨房台面背景、左侧留文字空间”可以是耐久意图；具体英文措辞、负面词、权重语法和参数需要按 provider/model/version 适配。

`unknown`：没有一手证据支持“一条商品提示词能够跨主流模型、未来版本和不同随机采样稳定得到相同质量”。凌镜不得把这一假设写成已证实事实。

### 3.2 正确的保存单位：制作配方，而不是 prompt 字符串

建议每次生成保存一个可复算的 `workflow recipe`：

- 目标：主图、场景副图、广告素材、尺寸说明图等；
- exact SKU 和必须保持不变的事实；
- 原图、参考图、mask 和模板版本；
- provider、model、model version、调用能力；
- 结构化创作意图和 provider-specific prompt；
- seed、尺寸、比例、强度、preset 等参数；
- 禁止变化项和渠道规则版本；
- 候选输出、Owner 选择/拒绝理由、返工操作；
- 最终资产、发布位置、观察窗口和效果指标。

这样即使模型改变，系统仍能保留“想达到什么、用了什么证据、为何选择、结果如何”，并重新编译到新模型。

## 4. 哪些资产更耐久

| 资产 | 耐久性判断 | 原因 |
|---|---|---|
| exact SKU 商品事实 | 很高 | 决定图片能否准确代表实际售卖变体 |
| 获权原图、商品参考图、细节图 | 很高 | 跨模型可复用，也是人工验真的基准 |
| Owner 接受/拒绝及局部错误标注 | 很高 | 记录真实偏好和失败模式，不依赖单一模型 |
| 渠道规则及规则版本 | 很高 | 决定能否发布，但需持续更新 |
| 发布后的渠道、曝光和经营效果数据 | 很高 | 能回答图片是否有效；必须控制混杂因素 |
| 模板、版式、brand kit | 高 | 字体、颜色、Logo、布局比自然语言更确定 |
| mask、商品轮廓、结构参考 | 高 | 把可修改区域与商品主体分离，降低事实漂移 |
| 结构化 workflow recipe | 高 | 能迁移和重新执行，保留过程证据 |
| provider-specific prompt | 中低 | 在固定模型版本内有用，跨版本需要重验 |
| 某次生成 seed | 低到中 | 仅在供应商支持且模型版本固定时有复现价值 |
| 单张“看起来不错”的输出 | 中 | 可直接使用，但若没有来源、SKU和效果关系，学习价值有限 |

这是一项 `inferred` 排序，不是行业统一评分；应由真实运行数据修正。

## 5. 优秀产品实际在怎样做

### 5.1 用参考图、mask 和结构控制，而非只写提示词

`actual`：Adobe Firefly 提供结构参考、风格参考、相似图片、局部填充 mask、对象合成和 custom model 等不同控制面。[Firefly API](https://developer.adobe.com/firefly-services/docs/firefly-api/api/)

`quoted`：Adobe 将 Custom Models 定位为捕获品牌风格、角色或商品，并通过版本化 asset ID 在工作流中复用；这证明其产品设计把“一致性资产”建模为模型、素材、preset 和参数的组合，而非 prompt 库。[Adobe Firefly API Overview](https://developer.adobe.com/firefly-services/docs/firefly-api/)

`actual`：eBay 的官方卖家工具先移除真实商品图背景，再选择纯色、影棚或 AI 场景；官方还建议 AI 背景只用于少数图片，并避免背景喧宾夺主。[eBay Take great photos](https://www.ebay.com/sellercenter/listings/photo-tips)

### 5.2 用模板和品牌系统保证一致性

`quoted`：Canva Brand Kit 集中管理字体、Logo、颜色、图像、图形、品牌模板和使用指南，并支持在设计中替换品牌资产。[Canva Brand Kit](https://www.canva.com/en_in/pro/brand-kit/)

`inferred`：对确定性的 Logo、字体、颜色、边距、文案位置和渠道尺寸，模板/规则引擎比“让生成模型每次理解一遍”更可靠。AI 应负责需要变化的场景和创意部分，不应替代所有确定性合成。

### 5.3 用人工偏好反馈逐轮收敛

`actual`：Google Research 的 PASTA 工作把用户连续选择建模为偏好反馈，并用它调整下一轮 prompt expansion；论文报告人工评价优于基线。这支持“选择历史比最终 prompt 更有长期学习价值”的方向，但尚不能直接证明电商转化效果。[Preference Adaptive and Sequential Text-to-Image Generation](https://research.google/pubs/preference-adaptive-and-sequential-text-to-image-generation/)

`actual`：另一项 Google Research 工作把反馈细化到错误区域以及未被图像表达的关键词，说明“为什么拒绝”比单一通过/失败标签更有信息量。[Rich Human Feedback for Text to Image Generation](https://research.google/pubs/rich-human-feedback-for-text-to-image-generation/)

### 5.4 以 exact variant 和渠道结果闭环

`actual`：Google Merchant 要求主图准确展示整个商品，正确匹配颜色、图案、材质及具体 variant；通用图、占位图、错误颜色或未售卖配件可能导致拒登。[Google Merchant image_link requirements](https://support.google.com/merchants/answer/6324350?hl=en)

`actual`：Google Merchant 可按 product ID 查看图片、批准/拒绝状态、可见性和点击数，说明渠道结果必须绑定具体商品与时间，而不是只形成一套脱离发布的图片库。[Viewing and understanding product data](https://support.google.com/merchants/answer/16488801?hl=en)

`inferred`：长期最有价值的学习对象不是“哪条 prompt 好看”，而是“某类商品 + 某种用途 + 某个渠道 + 某个制作配方，在受控比较下的通过率、人工时间、点击/转化和退货风险”。

## 6. 凌镜现在应该做什么

### 6.1 建议保留和建设

1. **商品事实层**：所有图片绑定 exact SKU、变体、包含物、真实参考图和权利来源。
2. **用途层**：明确主图、详情图、场景图、广告创意等不同任务，规则不可混用。
3. **可替换执行层**：去背景、模板合成、生成背景、局部编辑、人工修图和实拍都是 provider，不假设 AI 必须处理全部图片。
4. **版本化制作配方**：保存模型、prompt、参考图、mask、模板、参数和输出哈希。
5. **候选与反馈层**：保存 Owner 选择、拒绝原因、错误区域、返工动作和最终批准。
6. **渠道适配与验收**：按目标渠道检查尺寸、变体、主图/副图、元数据和禁止内容。
7. **结果学习层**：记录制作时间、调用成本、首轮通过率、返工次数、平台拒登、点击/转化以及与图片相关的退货投诉。

### 6.2 暂时不要做

1. 不自研基础图像生成模型。
2. 不建设面向所有人的通用 Canva/Photoshop 编辑器。
3. 不把“提示词市场”或海量 prompt 模板库当作当前产品目标。
4. 不先接很多 provider；先用一条确定性处理路径和一个生成 provider 跑通真实验证。
5. 不承诺全自动发布；在事实保真和效果数据不足前保留 Owner 审批。
6. 不用 AI 重绘商品主图主体来替代真实商品证据。
7. 不因图片更漂亮就宣称经营效果更好；必须连接实际渠道数据，并区分曝光、点击、转化和退货。

## 7. 开发前置与最小真实验证

### 7.1 当前开发前置：先让系统具备验证资格

`actual`：当前仓库已有图片资产、任务、确定性处理、权利、预算、审核、图片集合、发布门禁和独立 Image Service 等工程能力；真实 AI Provider 仍未形成足够次数的可用验证，Owner 页面也尚未由 Owner 走通“真实素材 → 候选 → 对比/拒绝/返工 → 最终批准 → 配方与成本”的完整操作。

因此当前不能直接开始3个 SKU 对照。下一开发目标应先完成一个真实 exact SKU 的场景副图生产闭环，包含完整 recipe、候选对比、拒绝原因、错误区域、返工和成本统计，并由 Owner 亲自操作验收。只有这一步通过，系统才具备比较实验资格。

### 7.2 下一步：3 SKU 最小真实比较

单 SKU 可用生产闭环通过后，再做一个小但完整的比较实验：

#### 样本

- 选择 3 个真实、已确定 exact SKU 的商品，而不是只选一个最容易成功的商品；
- 每个商品准备获权原图和必须保持不变的事实清单；
- 选择一个真实目标渠道和一个明确用途，例如“商品场景副图”；
- 暂不使用 AI 重绘商品主图主体。

#### 三条制作路线

- A：现有人工/现成工具基线；
- B：真实商品抠图 + 固定模板/确定性处理；
- C：真实商品抠图或参考图 + AI 场景生成 + Owner 审批。

#### 必须保存的数据

- 每套图的总耗时、人工分钟数、API/外包成本；
- 候选数、首轮通过率、返工轮次；
- 商品事实错误类型和错误区域；
- 使用的完整 recipe，而不只是最终 prompt；
- 渠道批准/拒绝；
- 在可比流量条件下的点击、加购、转化，以及后续与图片不符有关的退货/投诉。

#### 判断标准

`planned`：具体阈值应在跑基线后由 Owner确认，不能先伪造行业标准。

至少回答：

1. C 是否比 A/B 更快或更便宜？
2. C 是否在不提高商品事实错误的前提下增加可用素材？
3. 保存完整 recipe 和拒绝理由后，第二个相似商品是否明显减少试错？
4. 图片差异是否带来可观察的渠道效果差异，而不是只有主观“更好看”？

如果 AI 路线不能改善任何一项，继续保留确定性图片流水线，不应为了趋势扩大生成能力。如果改善成立，再逐类目积累 recipe、失败模式与效果数据。

## 8. 当前未知

- `unknown`：凌镜当前具体商品类目中，AI 场景图相对模板和人工方法的净收益。
- `unknown`：多少历史选择数据足以形成稳定的商品/渠道配方推荐。
- `unknown`：图片对点击、转化和退货的因果影响；同期价格、标题、流量、促销会造成混杂。
- `unknown`：不同 provider/model/version 的迁移成本和输出稳定性。
- `unknown`：哪些真实商品类别必须实拍或专业修图。

这些未知不要求停止全部图片建设，但要求暂停无验证的范围扩张，并先完成真实对照。
