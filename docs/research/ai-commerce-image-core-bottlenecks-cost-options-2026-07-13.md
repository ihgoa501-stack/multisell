# AI 电商作图：核心卡点、成本与方案（2026-07-13）

## 结论

AI 电商作图最大的卡点不是生成模型或 API 接入，而是持续证明输出图仍准确代表 exact SKU。颜色、材质、纹理、尺寸比例、数量、配件、包装和瑕疵中的任一偏差，都可能把“漂亮图片”变成错误商品声明。

最大的可变成本通常不是单次 API 调用，而是可信原始素材、人工验收和返工。项目应以“每个通过验收、可以发布的 SKU 图片组成本”为核心指标，而不是每张生成图价格。

## 证据等级

- `actual`：官方规则或官方公开价格直接说明。
- `quoted`：供应商对自身能力的公开陈述，未由本项目独立验证。
- `inferred`：根据规则、价格和工作流推导，需用真实 SKU 实验验证。
- `unknown`：当前没有本项目真实生产数据。

## 最大卡点

### 1. 商品真实性与一致性

- `actual`：eBay 要求图片准确代表商品；二手、损坏或有缺陷商品不能用库存图。卖家仍对 AI 生成的刊登内容负责。
- `actual`：Google Merchant 要求图片匹配具体变体，例如颜色、图案和材质。
- `inferred`：生成式模型擅长“看起来合理”，但电商主图需要“事实完全正确”。因此验收必须绑定 exact SKU、参考图和字段化检查，而不能只做审美评分。

### 2. 权利、渠道规则与可追溯性

- `actual`：eBay 明确警告未经许可使用第三方图片可能侵犯知识产权。
- `actual`：Google Merchant 要求 AI 生成图片保留或写入相应 IPTC 数字来源元数据。
- `inferred`：同一图片不能默认适合所有渠道；系统需要渠道规则配置、来源权利记录和生成历史，而不是只保存最终图片 URL。

### 3. 人工验收与返工

- `actual`：Photoroom 官方公开价中，基础去背景约为 0.02 美元/张，较完整的图片编辑约为 0.10 美元/张；实际套餐和企业合同另计。
- `inferred`：如果每个 SKU 需要多张候选、多轮修改和几分钟人工核对，人工时间、样品素材与返工会很快超过 API 调用费。
- `unknown`：本项目目前尚无真实 SKU 的首轮通过率、人工分钟数和返工轮次数据，不能给出可靠总成本。

## 成本模型

应统计：

`每套可发布图片成本 = 样品/原始素材 + 拍摄/修图人工 + 候选数 × API 单价 + 人审时间 + 返工 + 渠道衍生图 + 存储运维 + 错图/拒审/退货风险`

最关键的五个实测指标：

1. 首轮事实验收通过率；
2. 每套图片的人工审核分钟数；
3. 平均返工轮次；
4. 每套被接受图片的总成本；
5. 渠道拒审和因图片不符产生的投诉/退货。

## 可选方案

### A. 确定性编辑优先（推荐第一阶段）

用真实商品图做去背景、裁切、缩放、阴影和画布适配；不重新生成商品主体。适合主图，事实风险最低。可用 Photoroom、Adobe 或自建传统图像处理流水线。

### B. 参考图约束的生成式编辑

保留商品主体或蒙版，只生成背景、场景和局部装饰。适合副图、场景图和广告候选。必须并排对照 exact SKU 参考图，由 Owner 批准。

### C. 纯文生图重建商品

成本看似低、创意强，但最容易改动商品事实。不推荐用于主图或承担商品事实的详情图，只适合概念探索。

### D. 人工摄影/专业修图兜底

对高价值、反光、透明、复杂纹理、服装版型或 AI 多次失败的商品，直接转人工。它不是失败，而是风险控制分支。

### E. 渠道原生工具

适合单渠道快速试用，集成成本低；但能力、输出复用和审计证据可能受渠道限制。应作为执行器之一，而不是系统核心。

## 推荐的小闭环

先完成系统自身的单 SKU 可用生产闭环：Owner 能从真实素材创建场景副图任务，使用人工导入、确定性模板或一个真实可调用 AI Provider，查看候选与原图差异，记录拒绝原因和返工，批准最终文件，并读取完整制作配方与成本。底层模块、页面或自动测试存在不能替代这次真实操作验收。

系统具备验证资格后，只选一个真实 SKU 和一个目标渠道，制作一套 6 图：主图采用真实素材的确定性编辑；细节、包装和数量图使用真实照片；最多一张副图采用 AI 场景背景。所有输出保存输入来源、处理方式、模型/供应商、参数、成本和人工结论。单 SKU 闭环稳定后，再扩为3个 SKU 的人工、确定性模板和 AI 场景三路线比较。

通过条件：没有商品事实偏差，渠道规则检查通过，人工时间和每套总成本低于现有人工基线。停止条件：AI 连续改变商品事实，或者审核与返工时间高于人工工作流。

## 产品定位建议

应建设“商品图片治理与生产工作流”，而不是通用 AI 画图器。真正有价值的部分是 exact SKU 证据、渠道规则、审批、成本统计和可替换的执行器；模型/API 可替换，不应成为系统边界。

## 官方来源

- [eBay Picture policy](https://www.ebay.com/help/listing-policies/policies/picture-policy?id=4370)
- [eBay Adding pictures to your listings](https://www.ebay.com/help/selling/listings/photos-videos?id=4148)
- [eBay User Agreement](https://www.ebay.com/help/policies/user-agreement/user-agreement?id=4259)
- [eBay Intellectual property policy](https://www.ebay.com/help/policies/listing-policies/intellectual-property-policy?id=4349)
- [Google Merchant Center: AI-generated content](https://support.google.com/merchants/answer/14743464?hl=en)
- [Google Merchant Center: Image link specification](https://support.google.com/merchants/answer/6324350?hl=en)
- [Photoroom API pricing](https://www.photoroom.com/api/pricing)
- [Adobe Firefly Services API](https://developer.adobe.com/firefly-services/docs/firefly-api/api/)
- [C2PA specification](https://spec.c2pa.org/specifications/specifications/2.2/specs/C2PA_Specification.html)
- [IPTC Photo Metadata Standard 2025.1](http://www.iptc.org/std/photometadata/specification/IPTC-PhotoMetadata-2025.1.html)
