# AI 商品图片 Provider 官方合同核验

> 核验日期：2026-07-12（Asia/Shanghai）
> 证据范围：仅厂商官方开发文档、官方定价页、官方安全/隐私资料和官方服务条款
> 用途：为 `multi-provider-product-image-system.md` 的外部 Provider 开放决策提供输入
> 限制：没有真实凭据、账单或 sandbox 调用；本文不证明任何 Provider 已可用，也不是法律意见

## 1. 结论

首个可安全接入的真实 sandbox 候选是 **Photoroom Image Editing API sandbox**，但安全等级只能是 `sandbox_only`：官方明确提供每月 1,000 次带水印 sandbox 调用，能力最贴近商品图；同时，官方公开合同没有证明请求幂等、结果查询或 webhook，且同一输入重复调用会被视为两次调用。因此必须由 LingMirror 限制为带水印、不可发布、单次串行 canary，并在网络结果不确定时禁止自动重试。

Adobe 的 Firefly 异步接口在工程合同上最完整：有 job ID、状态查询和取消；Photoshop API 另有官方 webhook。但 Firefly Services 需要企业侧开通，公开资料没有给出本项目可直接采用的单次价格、明确数据保留/删除合同或 mutation 幂等保证，所以暂不作为首个 sandbox。

OpenAI Images 能力完整、可同步或 SSE 流式返回，但当前官方 Images 资源没有展示可恢复的图片 job 查询/取消/删除合同，也没有公开的图片请求幂等合同。断流后是否已计费和是否生成成功无法靠该资源恢复，因此生产付费自动重试必须关闭。

## 2. 官方合同矩阵

`unknown` 表示在本次查阅的官方资料中无法确认，不表示厂商一定不支持。

| 维度 | Photoroom | Adobe Firefly / Photoshop API | OpenAI Images |
|---|---|---|---|
| 主要能力 | Remove Background；Image Editing 提供背景、阴影、补光、扩图、商品美化等。官方提醒 Relight、文字移除、Expand、Beautifier、Upscale、Describe Any Change 可能改变商品，需人工验证 | Firefly：生成、填充、扩图、相似图、Object Composite、Upscale；Photoshop：PSD 图层编辑、渲染与转换 | Images API：文本生成、整图/局部编辑；支持尺寸、质量、格式、压缩、透明背景（依模型）和多图 |
| 调用形态 | 公开 quickstart 为同步 GET/POST，直接返回图片；异步 API 合同 `unknown` | Firefly 同时有同步接口和异步接口；异步返回 202、job ID、status URL、cancel URL。Photoshop 操作为异步 job | Images API 同步返回最终 base64；也支持 SSE partial/final image 流；不是持久化异步 job 合同 |
| 查询/取消 | `unknown` | Firefly 可按 job ID 查询并取消；Photoshop 可轮询 job | Images 资源只列 generate/edit/variation/streaming，图片 job 的 retrieve/cancel `unknown` |
| 幂等 | 未找到官方 idempotency key；官方明确同输入同参数调用两次算两次，且没有缓存 | `x-request-id` 官方定义为日志追踪 ID，不能据此推断幂等；mutation 幂等 `unknown` | Images API 的官方资源未展示 idempotency key；`unknown` |
| webhook | `unknown` | Photoshop API 官方支持 Adobe I/O Events webhook，通知 job 成功/失败；Firefly API webhook `unknown` | Images API 专用 webhook `unknown` |
| 费用 | Basic Remove Background：$0.02/图；Plus Image Editing：$0.10/图；成功调用扣量，错误调用不扣；Image Editing sandbox 每月 1,000 次且输出带水印 | 公开技术文档未给出可直接核算的 Firefly/Photoshop 单次价格；需要与 Adobe 代表/企业合同开通，记 `unknown` | 现行首选为 GPT Image 2，但其模型页只指向价格页/计算器，本次未取得可复算的公开单图数字，记 `unknown`。旧 GPT Image 1 页面仍列低/中/高 1024² 为 $0.011/$0.042/$0.167，不能代替 GPT Image 2 报价 |
| 限流 | 默认 60 图/分钟，超过返回 429；Enterprise 可提高 | Firefly 默认每组织 4 RPM、9,000 RPD；429 可按 `retry-after` 或退避，但 mutation 是否可安全重试仍取决于是否已受理 | 按 usage tier；GPT Image 2 免费层不支持，Tier 1 为 5 IPM，后续层级提高 |
| 数据地域 | 官方安全页称 GCP/AWS 美国；FAQ 称服务器位于美国东部；目前无 EU-only residency | 输入/输出可通过受支持的客户预签名存储 URL；Adobe 自身处理地域的精确合同 `unknown` | 支持国家/地区受协议限制；图片处理数据驻留地域的精确公开合同 `unknown` |
| 保留 | 同步图片在响应后丢弃；异步图片仅在 job 运行期间短暂保留后丢弃；运营日志 15 天，API access logs 1 年 | 官方安全资料说明服务会存储与服务有关的输入/输出信息，但本次未确认 Firefly/Photoshop 的统一具体期限，记 `unknown` | 默认 abuse monitoring logs 最长 30 天；Images generations/edits 无 application state；合格客户可申请 ZDR/MAM。即使启用，图片仍会做 CSAM 扫描，命中可能保留供人工复核 |
| 删除 | 官方称 API 图片不保存；对运行中异步图片的主动删除接口 `unknown` | Firefly 可取消运行中 job，但取消不等于删除；输入、输出、临时副本和日志的删除 API/SLA `unknown` | Images 资源未列图片对象删除接口；日志删除/提前清除 `unknown`，ZDR 需审批且有例外 |
| 输入权利/输出责任 | 通用条款适用；公开安全页的训练默认取决于合同：Enterprise 默认不训练，自助 API 默认参与但可在设置中 opt out。精确输入权利保证条款仍应在签约时复核 | Adobe 官方资料称 Firefly 输出属于 Customer Content、Adobe 不主张输出 IP；用户仍对输入限制、输出使用和不侵权负责，实际版权依当地法律 | Customer 保留 Input 权利并在法律允许范围拥有 Output；Customer 保证拥有提交 Input 所需全部权利，并负责评估输出。API IP indemnity 有明确例外，包括无输入权、商标商业使用、忽略安全功能等 |
| 失败/断流 | 429 和普通错误可识别；但同步响应断流且没有查询/幂等证据时，结果应记 `unknown`，自动付费重试为 0 | 异步 job 可用 status URL 对账，适合响应断流恢复；429/5xx 是否已创建 job 必须先以已有 job ref 对账，不能仅靠 `x-request-id` 重发 | SSE 或同步响应断流后，没有已确认的 Image job 查询合同；必须记 `unknown`，不得自动创建第二次付费请求 |

## 3. 各 Provider 的实现裁决

### 3.1 Photoroom：首个 `sandbox_only`

允许的第一条 canary 只做：一个 Owner 已核验权利的非敏感商品图，调用 Image Editing sandbox 的去背景/白底或 AI shadow，保留水印结果并永久标记 `sandbox / non_publishable`。

实施约束：

- 只开 POST 文件上传，不采用 Provider 代抓任意 URL；
- 关闭会改变商品主体的 Relight、文字移除、Expand、Beautifier、Upscale、Describe Any Change；
- 每个批准只允许一次真实 submit；超时或连接中断进入 `UNKNOWN`，人工查账，不自动重试；
- 本地保存请求 manifest、响应状态、耗时、输出哈希和 sandbox 水印证据；
- 自助账户先在设置中确认训练 opt-out，未确认前不得发送真实货源图；
- 不能把错误“不扣费”解释为断流一定不扣费，因为断流时客户端不知道 Provider 是否已成功处理。

### 3.2 Adobe：先做合同/账号准备，不做真实调用

Firefly 异步 job 的查询和取消适合 Image Service 的 `Submit → Reconcile` 模型，Object Composite/Fill/Expand 也符合精确编辑方向；Photoshop webhook 可作为未来减少轮询的补充。但在获得企业权限、价目、地域、保留/删除说明以及幂等/重复计费书面答复前，只能实现 fixture adapter，不开放真实 sandbox 或 production。

不得把 Photoshop API webhook 推广为 Firefly API webhook，也不得把 `x-request-id` 当成幂等键。

### 3.3 OpenAI：fixture 可做，真实付费保持关闭

建议将现行模型契约固定到明确 snapshot（当前官方列出 `gpt-image-2-2026-04-21`），并仅用于场景副图/广告创意候选，不用于商品事实主图。真实调用前必须解决断流后的重复费用决策：如果官方没有可查询 job 或幂等保证，运行器必须接受“单次提交后 UNKNOWN、人工决定是否新建意图”的限制。

不要用旧 GPT Image 1 的单图价格估算 GPT Image 2，也不要把“Customer 拥有 Output”解释为输出一定不侵权、一定可获得版权或一定适合商业发布。

## 4. 不开放生产的条件

任一条件未满足，对应 Provider 必须保持 `unavailable` 或 `sandbox_only`：

1. 未用真实账号确认 endpoint、模型 snapshot、账号/项目、地域和限流；
2. 未获得可复算的当前费用口径、税费、失败/取消/断流计费规则和账单对账方法；
3. 没有稳定 Provider 幂等键，且同步/流式请求又没有可查询 job；
4. 无法证明输入图片权利覆盖复制、修改、第三方 AI、跨境传输、商业用途和目标 Provider/地域；
5. 训练 opt-out、保留、日志、删除和子处理方状态尚未确认；
6. 429、5xx、超时、响应截断、零输出、恶意 MIME、同 key 不同 payload 的 fixture 未通过；
7. 没有最坏费用预占、Owner 目标绑定批准、单次 nonce、kill switch 和逐 attempt 对账；
8. 输出未经过商品真实性、权利、渠道规则、声明/场景和技术视觉五类审核；
9. sandbox/试用结果仍可能被 Listing image set、release attestation 或平台发布路径采用；
10. 无法在日志、MCP transcript、前端和普通数据库列中证明 Provider key、原始 URL、敏感输入已被隔离。

## 5. 下一步可执行验证

1. 先创建 Photoroom 自助 API 团队并确认训练 opt-out；不绑定生产发布权限。
2. 用专门的非敏感测试图和 Image Editing sandbox 执行一次带水印 canary；预算上限为 0 美元付费费用、最多 1 次 submit。
3. 人工断开响应连接做一次故障测试，确认系统进入 `UNKNOWN` 且不会重发；该测试会消耗 sandbox 配额，必须单独批准。
4. 从 API dashboard 核对调用和扣量事实，记录访问时间与截图/账单证据。
5. 在 Photoroom 真实合同完成前不升级生产；随后再分别向 Adobe 和 OpenAI 索取本文 unknown 项的书面答案。

## 6. 官方来源（均访问于 2026-07-12）

### Photoroom

- [Image Editing API Quickstart](https://docs.photoroom.com/image-editing-api-plus-plan/quickstart-guide)
- [API FAQ：限流、存储、服务器位置](https://docs.photoroom.com/getting-started/frequently-asked-questions)
- [API Pricing 说明](https://docs.photoroom.com/getting-started/pricing)
- [Image Editing Pricing：重复调用与缓存](https://docs.photoroom.com/image-editing-api-plus-plan/pricing)
- [API Pricing & Plans：sandbox 与单图价格](https://www.photoroom.com/api/pricing)
- [Data Security & Privacy](https://www.photoroom.com/platform/security)
- [Terms and Conditions](https://www.photoroom.com/legal/terms-and-conditions)
- [Privacy Policy](https://www.photoroom.com/legal/privacy)

### Adobe

- [Firefly Async API 指南](https://developer.adobe.com/firefly-services/docs/firefly-api/guides/how-tos/using-async-apis)
- [Firefly API Reference：status/cancel](https://developer.adobe.com/firefly-services/docs/firefly-api/api/)
- [Firefly Technical Usage Notes：限流与存储域](https://developer.adobe.com/firefly-services/docs/firefly-api/getting-started/usage-notes/)
- [Object Composite Guide](https://developer.adobe.com/firefly-services/docs/firefly-api/guides/how-tos/object-composite/)
- [Photoshop API：PSD renditions 与异步 job](https://developer.adobe.com/firefly-services/docs/photoshop/guides/rendering-and-conversions/)
- [Photoshop API Webhooks / Adobe I/O Events](https://developer.adobe.com/firefly-services/docs/photoshop/getting-started/webhooks/)
- [Firefly Services credentials](https://developer.adobe.com/firefly-services/docs/guides/get-started)
- [Adobe Firefly Data and Content Usage](https://business.adobe.com/content/dam/dx/us/en/resources/sdk/adobe-firefly-data-and-content-usage/adobe-firefly-data-and-content-usage.pdf)
- [Adobe Firefly Services Security Fact Sheet](https://www.adobe.com/cc-shared/assets/pdf/trust-center/ungated/whitepapers/creative-cloud/adobe-firefly-services-security-fact-sheet.pdf)

### OpenAI

- [Image generation guide](https://developers.openai.com/api/docs/guides/image-generation)
- [Images API Reference](https://developers.openai.com/api/reference/resources/images)
- [Image streaming events](https://platform.openai.com/docs/api-reference/images-streaming)
- [GPT Image 2 model](https://developers.openai.com/api/docs/models/gpt-image-2)
- [GPT Image 1 model and historical prices](https://developers.openai.com/api/docs/models/gpt-image-1)
- [API data controls](https://platform.openai.com/docs/models/default-usage-policies-by-endpoint)
- [OpenAI Services Agreement](https://openai.com/policies/services-agreement/)
- [Service Terms](https://openai.com/policies/service-terms/)
