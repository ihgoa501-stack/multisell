# AI 电商作图：权利、人物与数据风险审计

> 审计日期：2026-07-12（Asia/Shanghai）
> 访问日期：2026-07-12（除非条目另有说明）
> 范围：Owner 自营商品图片的 AI 生成、编辑、内部审核与渠道使用
> 证据限制：只使用监管机关、法院/版权与知识产权机构、C2PA、OpenAI、Google Cloud、Adobe 的官方材料；未核验任何具体供应商合同、图片作者、目标国家法律或销售渠道条款
> 非法律意见：本文是工程与经营风险控制建议；具体争议需由适用法域的专业律师判断。

## 1. 先给结论

1. **卖货权不等于图片复制、修改或上传 AI 的权利。** `inferred`：商品买卖/分销授权解决的是销售商品；商品照片是独立作品，复制、改编、公开展示通常是版权人的专有权。美国版权法还明确，实物载体的所有权转移本身不转移其中作品的版权。[美国版权局：17 USC §202、§106](https://www.copyright.gov/title17/92chap2.html)，[美国版权局：版权基本权利](https://www.copyright.gov/what-is-copyright/)。适用范围：美国法的直接规则；“必须分别清权”作为跨境经营保守闸门。其他国家的例外和默示许可范围 `unknown`。
2. **把第三方图片上传到外部 AI，至少涉及复制和服务商处理；若让模型编辑，还可能涉及改编。** `inferred`：除非合同明确覆盖“向第三方 AI 服务传输/处理”和预定商业用途，不能从“供应商让我上架”自动推出有该权限。OpenAI 也要求客户保证拥有提供 Input 所需的一切权利、许可和权限。[OpenAI Services Agreement §4.3](https://cdn.openai.com/osa/openai-services-agreement.pdf)。适用范围：使用 OpenAI 商业服务；其他 Provider 应逐份核验。
3. **“用户拥有输出”不等于输出不侵权，也不等于能取得可执行版权。** `quoted`：OpenAI 仅在“OpenAI 与客户之间、法律允许范围内”将其对 Output 的权利转让给客户，同时由客户负责评估输出是否适合用途。[OpenAI Services Agreement §4](https://cdn.openai.com/osa/openai-services-agreement.pdf)。`quoted`：美国版权局认为纯 AI 生成材料不受版权保护；有人类创作贡献时，只保护人类创作部分。[美国版权局 AI 报告 Part 2](https://www.copyright.gov/ai/)。适用范围：前者是合同相对关系，后者是美国版权登记/版权法；其他法域 `unknown`。
4. **电商场景的商标风险不能交给 Provider 补偿兜底。** `quoted`：OpenAI API 补偿排除因客户在贸易或商业中使用输出而产生的商标或相关权利主张，也排除客户无权使用输入、已知可能侵权、绕过安全措施及修改/组合输出等情形。[OpenAI Service Terms，更新 2026-06-12](https://openai.com/policies/service-terms/)。`quoted`：Google Cloud 的生成输出补偿也排除因客户在贸易或商业中使用输出产生的商标相关权利主张，并限定为符合条件服务的未修改输出等条件。[Google Cloud Service Specific Terms](https://cloud.google.com/terms/service-terms/)。适用范围：对应商业合同及其资格条件；当前凌镜是否符合补偿资格 `unknown`。
5. **人物图需要独立处理肖像、公开权、隐私和个人信息。** `quoted`：USPTO 指出美国姓名、形象与肖像权主要受州法保护，各州不同；商业使用可能同时涉及商标法，并建议合同明确 AI 生成形象和数字替身。[USPTO：Name, image, and likeness，更新 2026-06-29](https://www.uspto.gov/trademarks/name-image-and-likeness)。`quoted`：中国《生成式人工智能服务管理暂行办法》要求不得侵害肖像、名誉、隐私和个人信息权益。[工信部官方文本](https://www.miit.gov.cn/zcfg/qtl/art/2023/art_f4e8f71ae1dc43b0980b962907b7738f.html)。适用范围分别为美国、中国境内公众生成式 AI 服务；具体广告投放地与模特合同效力 `unknown`。
6. **C2PA/Content Credentials 是来源和防篡改证据，不是合法、真实或商品一致性的证明。** `quoted`：C2PA 规范不对来源声明“好或坏”作价值判断，只验证声明是否与资产绑定、格式正确且未被篡改；其解释文件还明确 provenance 不能单独判断内容是否真实、准确或符合事实。[C2PA 2.2 规范](https://spec.c2pa.org/specifications/specifications/2.2/specs/C2PA_Specification)，[C2PA Explainer](https://c2pa.org/specifications/specifications/2.2/explainer/Explainer.html)。适用范围：技术证据；不是权利许可或渠道合规结论。

因此，凌镜不应设置一个笼统的 `image_rights = actual`。正确做法是把不同权利、人物同意、Provider 数据处理和目标渠道用途拆成分别核验的闸门；任何关键项为 `unknown` 时，外部 Provider 调用次数为零。

## 2. 风险分层

### P0：调用模型前必须阻断

| 风险问题 | 结论与证据等级 | 适用范围 / 未知 |
|---|---|---|
| 供应商允许销售，是否自动允许复制图片？ | `inferred`：否。照片作者通常是最初版权人，拥有复制、改编、公开展示等权利；转移实物不自动转移版权。[美国版权局摄影说明](https://www.copyright.gov/engage/photographers/)，[17 USC §202](https://www.copyright.gov/title17/92chap2.html) | 美国法直接支持；供应商所在地、摄影雇佣关系、平台上传许可 `unknown` |
| “可用于上架”是否允许修改背景？ | `inferred`：不应默认允许。编辑可能构成改编，授权范围需要覆盖修改及商业展示。[美国版权局：如何取得许可](https://www.copyright.gov/circs/circ16a.pdf) | 具体合同措辞和当地法例外 `unknown` |
| 是否允许上传第三方 AI？ | `inferred`：需要额外明确。上传产生第三方处理/复制，Provider 还要求输入方拥有必要权利。 | 供应商合同是否涵盖分包处理、跨境传输和模型服务 `unknown` |
| 图片中的品牌、包装、认证标志 | `inferred`：必须与真实商品和获授权销售范围一致；生成不存在的标志、错误包装或近似第三方品牌会产生商标、假冒与误导风险。Provider 的输出补偿普遍不覆盖贸易中的商标主张。 | 每个目标国、商品类别、渠道及授权链 `unknown` |
| 真人或可识别人物 | `quoted`：美国 NIL/公开权州法不同；中国官方规则明确保护肖像、隐私和个人信息。需要覆盖 AI 编辑、商业广告、国家/渠道、期限、再许可的模特同意。 | 人物身份、年龄、模特授权、投放地区 `unknown`；未成年人默认禁止首版 |
| 输入含个人信息或经营秘密 | `inferred`：人物照片可能是个人数据；后台截图、联系方式、报价和未公开包装还可能是秘密信息。无单独合法基础和最小化处理不得上传。 | 适用的数据保护法、跨境传输机制和数据驻留 `unknown` |

### P1：输出后、进入草稿前必须阻断

| 风险问题 | 结论与证据等级 | 适用范围 / 未知 |
|---|---|---|
| 输出是否独有 | `quoted`：OpenAI 条款说明输出可能不唯一，其他用户可能获得相似输出。[OpenAI Terms of Use](https://openai.com/policies/terms-of-use/) | 商业/API 合同具体版本需在启用时冻结 |
| 输出是否可受版权保护 | `quoted`：美国版权局只承认人类创作贡献；纯 AI 生成材料不受美国版权保护。[USCO AI Initiative / Part 2](https://www.copyright.gov/ai/) | 美国结论；目标国结论 `unknown` |
| Provider 是否保证不侵权 | `quoted`：OpenAI 与 Google 仅在特定付费合同和条件下提供有限补偿，输入无权、商标商业使用、已知风险、修改输出等可被排除。 | 是否购买符合资格的服务、通知和抗辩义务 `unknown` |
| 商品是否被 AI 改变 | `inferred`：版权清洁也不能证明商品颜色、结构、数量、包装文字、配件与实物一致。必须由 Owner 对原图、样品/可靠规格和输出原文件人工核验。 | 真实样品和可靠规格目前 `unknown` |
| 人物是否被虚构为代言/使用 | `inferred`：即使生成脸不是某个已知真人，也可能近似真人，或让消费者误以为存在真实使用体验/代言。首版应禁真人与拟真人。 | 各法域广告、消费者保护要求 `unknown` |
| 生成标识是否需要保留 | `quoted`：中国《人工智能生成合成内容标识办法》自 2025-09-01 施行，要求适用服务添加显式/隐式标识，用户发布生成合成内容应主动声明并使用标识功能，不得恶意删除或篡改。[中国网信办官方文本](https://www.cac.gov.cn/2025-03/14/c_1743654684782215.htm) | 是否直接适用于 Owner、Provider、销售渠道需具体判断；工程上应保留而非删除 |

## 3. 三类 Provider 的输入、输出与数据风险

### 3.1 OpenAI API

- `quoted`：商业协议下，客户保留 Input 权利，并在法律允许范围内拥有 Output；客户保证拥有提供 Input 所需的一切权利，并独自负责输出用途和适当性。[OpenAI Services Agreement](https://cdn.openai.com/osa/openai-services-agreement.pdf)。
- `quoted`：API 输入/输出默认不用于训练模型，除非客户主动选择共享。[OpenAI Data Controls](https://platform.openai.com/docs/models/default-usage-policies-by-endpoint)。
- `quoted`：默认滥用监控日志可含提示词、响应和元数据，最多保留 30 天；ZDR/MAM 需申请。`/v1/images/generations`、`edits`、`variations` 无应用状态保留，但默认仍有 30 天滥用监控；`gpt-image-1` 与 `gpt-image-1-mini` 可兼容 ZDR，DALL-E 2/3 不兼容。图片/文件输入还会接受 CSAM 扫描，命中潜在内容时即使启用 ZDR 也可能留存供人工复核。[同一官方 Data Controls 页面](https://platform.openai.com/docs/models/default-usage-policies-by-endpoint)。
- `inferred`：凌镜启用前必须冻结具体 endpoint、model snapshot、组织级数据控制和地区；“API 不训练”不能被写成“零保留”或“无人可见”。

### 3.2 Google Cloud Vertex AI / Imagen

- `quoted`：Google 未经客户事先许可或指示，不用客户数据训练或微调 AI/ML 模型；部分 grounding 功能存在固定 30 天存储例外。[Vertex AI Zero Data Retention](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/vertex-ai-zero-data-retention)。
- `quoted`：服务条款称，未经客户许可/指示，Google 不会在客户账户外保存提示数据超过生成输出所合理需要的时间，也不会在账户外保存输出。[Google Cloud Service Specific Terms](https://cloud.google.com/terms/service-terms/)。
- `quoted`：输出补偿只适用于列明的付费“Generative AI Indemnified Service”和符合条件的未修改输出；贸易中的商标主张、已知风险、绕过工具、收到侵权通知后继续使用、定制数据无权等不受保护。[同一条款](https://cloud.google.com/terms/service-terms/)。
- `inferred`：如果采用 Vertex，必须在每次模型/版本升级时重新核对 Imagen 是否仍列在当前 indemnified services 清单，以及项目地区和功能是否满足零保留要求。

### 3.3 Adobe Firefly

- `quoted`：Adobe 表示不会用客户内容训练 Firefly，Firefly 使用获得许可的 Adobe Stock 和版权过期的公版内容训练；Adobe 不主张客户内容或 Firefly 输出的所有权。[Adobe 官方生成式 AI 方法](https://www.adobe.com/ai/overview/firefly/gen-ai-approach.html)。
- `quoted`：Adobe 仍明确要求创作者自行判断合作伙伴模型是否适合具体项目；其用户指南禁止上传第三方受版权保护的参考图、生成侵权/商标内容或侵犯第三方隐私/数据权的个人信息。[Adobe Generative AI User Guidelines，更新 2026-05-15](https://www.adobe.com/it/legal/licenses-terms/adobe-gen-ai-user-guidelines.html)。
- `quoted`：Adobe 可用自动和人工方式审查提示、输入和输出以防滥用和过滤内容。[Adobe 官方用户指南](https://www.adobe.com/id_id/legal/licenses-terms/adobe-gen-ai-user-guidelines.html)。
- `quoted`：Firefly 对 100% 像素生成的资产自动附加 Content Credentials，并可能将凭证副本保存到 Adobe 公共 Content Credentials 云，以便恢复。[Adobe Content Credentials Overview，更新 2026-03-11](https://helpx.adobe.com/mena_en/firefly/web/get-started/learn-the-basics/content-credentials-overview.html)。
- `unknown`：凌镜拟使用的具体 Firefly API/企业套餐是否有 IP 补偿、补偿额度、排除项、输入保留和地域处理，必须以签约日具体订单和产品专用条款确认，不能从市场宣传页推定。

## 4. 凌镜必须保存的证据

### 4.1 权利证据必须拆成可判定字段

每张源图至少分别保存：

1. `source_image_owner`：摄影者/雇主/版权承继者；
2. `supplier_authority_basis`：供应商凭什么可授权该图；
3. `right_download_copy`：是否可下载和制作副本；
4. `right_modify_derivative`：是否可裁剪、去背、重构、生成式编辑；
5. `right_third_party_ai_processing`：是否可上传指定 Provider 处理；
6. `right_commercial_listing_ads`：是否可用于指定国家、渠道、类目、广告/详情页；
7. `right_sublicense_platform`：是否允许授予平台展示、压缩、分发所需许可；
8. `territory / channel / term / sku_scope`：地域、渠道、期限、SKU；
9. `people_release`：人物/模特是否同意 AI 编辑、商业广告、再许可、地域和期限；
10. `marks_authorization`：品牌、包装、认证标志、联名元素的授权；
11. `evidence_source_url_or_file_hash / observed_at / verified_by_owner`；
12. 每一项独立状态：`actual / quoted / unknown`，不得用总分或 AI 置信度替代。

`actual` 只代表 Owner 已核对原始合同/授权及观察时间；供应商聊天中一句“可以用”最多先记为 `quoted`，除非能确认说话主体有授权权力且用途范围完整。

### 4.2 每次模型调用必须保存

- 源图原始字节 SHA-256、受控存储地址、C2PA/元数据检测结果；
- 权利证据版本和批准哈希；
- Provider、产品、model snapshot、endpoint、账户/项目、处理地区；
- 生效条款 URL、条款版本/日期、页面或 PDF SHA-256；
- 是否训练、默认保留、ZDR/MAM 状态、人工审查例外；
- 完整 prompt、negative prompt、mask、参数、请求哈希、幂等键；
- Provider request ID、时间、状态、费用、失败/安全拒绝；
- 输出原始字节、SHA-256、尺寸、格式、Content Credentials 原文与验证结果；
- 自动检测与 Owner 人工核验分别记录；
- 选中/拒绝、用途、SKU、渠道和草稿审批哈希。

### 4.3 C2PA 的正确用法

- `inferred`：保存并验证 Provider 返回的原始凭证，不主动剥离；转码/裁剪后另建新的派生关系和哈希。
- `quoted`：C2PA 可证明声明与资产绑定且未被篡改，但不能证明声明真实、权利已清、人物已同意或商品没有被 AI 改变。[C2PA 规范](https://spec.c2pa.org/specifications/specifications/2.2/specs/C2PA_Specification)。
- `inferred`：即使渠道上传时丢失元数据，凌镜仍应保留原文件、凭证和派生链；缺少 C2PA 不能自动等于违规，存在 C2PA 也不能自动通过。

## 5. 推荐闸门

### Gate A：源图准入（任何 `unknown` 都不调用 Provider）

- [ ] 源图来自不可变受控快照，字节和 SHA-256 已保存；
- [ ] 授权主体身份与授权权力已由 Owner 核验；
- [ ] 复制、修改、第三方 AI 处理、商业渠道使用、平台再许可分别为 `actual`；
- [ ] 地域、渠道、期限、SKU 与本次任务一致；
- [ ] 品牌/包装/认证标志授权范围明确；
- [ ] 无真人、未成年人、个人信息、联系方式、后台截图、报价/成本或秘密资料；首版如检测到人物即阻断；
- [ ] Provider 数据地区、保留、训练、人工复核例外已接受；
- [ ] 生效条款与 Provider 配置已冻结并哈希。

### Gate B：输出准入（成功生成不等于通过）

- [ ] 输出原始字节、SHA-256、请求 ID、模型版本和费用完整；
- [ ] 未出现新的品牌、认证、包装文字、人物、受保护角色或疑似水印；
- [ ] 商品颜色、数量、结构、接口、尺寸关系、配件、包装与样品/可靠规格一致；
- [ ] Provider 安全过滤未被绕过；有拒绝时停止并交 Owner，Agent 不自动改写提示词规避；
- [ ] C2PA/标识已保存，后续处理不恶意删除；
- [ ] Owner 查看的是原始分辨率文件，并以 SHA-256 绑定选择；
- [ ] 目标国家和渠道的商标、人物、广告、AI 标识规则已另行核验。

### Gate C：草稿与发布

- [ ] 采用图片只进入现有 `product_listing.status=draft`；
- [ ] 换图使旧草稿审批失效并回到 `editing`；
- [ ] 发布另建 Owner 审批，冻结图片哈希和渠道请求；
- [ ] 平台返回无错误只记 `submitted`，不证明权利、商品真实或真实上线；
- [ ] 投诉/侵权通知到达后立即冻结该图片及相关发布，不继续使用并保全证据。

## 6. 首版范围裁决

推荐只做：**Owner 拥有或取得完整书面权利的一张真实商品图，去除原背景并生成纯白背景内部候选；不引入人物、不生成或修改品牌/包装文字、不自动发布。**

原因：

- `inferred`：真人场景同时扩大肖像、个人信息、虚假代言和 AI 标识风险；
- `inferred`：从零文生商品图无法可靠证明商品事实；
- `inferred`：白底候选仍可能改变商品，因此必须保留原图并人工逐项比对；
- `actual`（仓库审计）：当前真实受控商品、图片权利、目标渠道和费用证据仍是 `unknown`；工程模块即使实现，也不能宣称真实可用。

## 7. 开发前仍需取得的外部事实

1. `unknown`：一个真实 SKU 的源图作者和完整授权链；
2. `unknown`：授权是否明确覆盖指定外部 AI Provider、处理地区及商业发布；
3. `unknown`：目标国家/地区、消费者、类目和销售渠道；
4. `unknown`：目标渠道当前图片、品牌、人物和 AI 生成标识规则；
5. `unknown`：最终选择的 Provider、账户套餐、数据控制状态、处理地区与补偿资格；
6. `unknown`：适用法、争议管辖和人物/广告规则。

这些未知不阻塞建设“失败关闭”的领域状态机和测试 Provider，但必须阻塞真实图片上传、付费调用和“可用于卖货”的结论。

## 8. 官方来源索引

- 美国版权局：[版权是什么](https://www.copyright.gov/what-is-copyright/)、[17 USC 第 1 章（定义与专有权）](https://www.copyright.gov/title17/92chap1.html)、[第 2 章（所有权与转移）](https://www.copyright.gov/title17/92chap2.html)、[如何取得许可](https://www.copyright.gov/circs/circ16a.pdf)、[AI Initiative 与报告](https://www.copyright.gov/ai/)
- USPTO：[Name, image, and likeness](https://www.uspto.gov/trademarks/name-image-and-likeness)
- 中国监管：[生成式人工智能服务管理暂行办法](https://www.miit.gov.cn/zcfg/qtl/art/2023/art_f4e8f71ae1dc43b0980b962907b7738f.html)、[人工智能生成合成内容标识办法](https://www.cac.gov.cn/2025-03/14/c_1743654684782215.htm)
- 欧盟：[AI Act Regulation (EU) 2024/1689](https://eur-lex.europa.eu/eli/reg/2024/1689/oj/eng)、[欧委会 GDPR 个人信息说明](https://commission.europa.eu/law/law-topic/data-protection/information-individuals_en)
- OpenAI：[Services Agreement](https://cdn.openai.com/osa/openai-services-agreement.pdf)、[Service Terms](https://openai.com/policies/service-terms/)、[API Data Controls](https://platform.openai.com/docs/models/default-usage-policies-by-endpoint)
- Google Cloud：[Service Specific Terms](https://cloud.google.com/terms/service-terms/)、[Vertex AI Zero Data Retention](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/vertex-ai-zero-data-retention)
- Adobe：[Firefly 生成式 AI 方法](https://www.adobe.com/ai/overview/firefly/gen-ai-approach.html)、[Generative AI Product Specific Terms，生效 2026-04-23](https://www.adobe.com/cc-shared/assets/pdf/legal/servicetou/adobe-generative-ai-product-specific-terms-en-us-20260423.pdf)、[Content Credentials Overview](https://helpx.adobe.com/mena_en/firefly/web/get-started/learn-the-basics/content-credentials-overview.html)
- C2PA：[Content Credentials 2.2 规范](https://spec.c2pa.org/specifications/specifications/2.2/specs/C2PA_Specification)、[Explainer](https://c2pa.org/specifications/specifications/2.2/explainer/Explainer.html)
