# LingMirror Image Service 与 MCP 技术合同

> 日期：2026-07-12
> 状态：`implemented_initial / paid_provider_execution_gated`
> 上位规格：`multi-provider-product-image-system.md`

## 1. 决策

在当前 monorepo 的 `services/image-service/` 建设独立内部服务。它负责图片字节、处理任务和 Provider 执行事实；凌镜后端继续拥有商品、权利、预算、Owner 审批、Listing 图片集合和发布放行事实。

```text
小Q/Agent ──MCP──→ LingMirror Backend Capability
                           │
                           │ 内部 HTTP + 一次性执行凭证
                           ▼
                  LingMirror Image Service
                    │ Worker / BlobStore
                    ▼
       deterministic / Photoroom / Adobe / OpenAI
```

MCP 不是大文件传输或生产任务的唯一协议；它只提供 Agent 可发现的受控工具。生产执行走内部 HTTP，图片走受控 BlobStore，异步处理走 durable queue/outbox。

## 2. 所有权边界

| 事实 | 权威系统 |
|---|---|
| 商品、SKU、来源主体、渠道和图片角色 | LingMirror |
| 图片权利、规则、预算、Owner 决定和发布放行 | LingMirror |
| Provider capability、attempt、request ID、技术状态和原始费用 | Image Service |
| 图片不可变字节、对象版本、SHA-256 和技术派生链 | Image Service |
| Listing image set 与最终经营成本 | LingMirror |

Lake 1 的 LingMirror 侧已将精确资产权利、五类逐项审核和成本账本作为 Listing image set 创建与冻结的双重门禁。Image Service 的任务成功、输出存在或 Provider 返回费用均不能替代这些 Owner 侧事实。

两个系统不得共同拥有同一状态。Image Service 的 `SUCCEEDED` 只说明技术处理成功；不能返回“合规、真实、可发布”。

## 3. 部署与目录

```text
services/image-service/
├── cmd/server/
├── internal/application/
├── internal/providers/{deterministic,photoroom,adobe,openai}/
├── internal/jobs/
├── internal/blobstore/
├── internal/auth/
├── internal/contracts/
├── mcp/
└── tests/
```

第一阶段同仓库、独立进程、独立数据库 schema/凭据和私有网络地址。只有出现独立发布周期或第二个内部消费者后，才重新评估是否拆仓并记录决定。

## 4. 内部 HTTP 合同

根路径 `/internal/v1`，只接受服务身份和 LingMirror 签发的短期执行令牌。

- `GET /processors`：能力、契约版本、可用性和安全等级；
- `POST /jobs`：创建或幂等重放任务；
- `GET /jobs/:id`：任务与 attempt 状态；
- `POST /jobs/:id/reconciliations`：安全查询 Provider；
- `POST /jobs/:id/retries`：仅合同允许的重试；
- `GET /jobs/:id/outputs`：输出元数据；
- `GET /blobs/:id/content`：授权流式读取；
- `POST /jobs/:id/remote-deletions`：Provider 支持且已批准时删除。

所有输入输出有 JSON Schema；第三方响应先按 Provider schema 校验。列表分页。错误统一为：

```json
{"error":{"code":"RECONCILE_REQUIRED","message":"Provider result is ambiguous","details":{}}}
```

核心错误：`VALIDATION_ERROR`、`UNAUTHORIZED`、`TOKEN_TARGET_MISMATCH`、`IDEMPOTENCY_CONFLICT`、`PROVIDER_UNAVAILABLE`、`BUDGET_TOKEN_INVALID`、`RECONCILE_REQUIRED`、`OUTPUT_INVALID`、`VERSION_CONFLICT`。

同一 idempotency key + 同一 request manifest 返回同一 job；同 key 不同 manifest 返回 409。HTTP 2xx 只表示本地任务已持久化，不表示 Provider 已执行。

## 5. 一次性执行令牌

LingMirror 在 Owner 批准并原子预占预算后签发短期 JWT/PASETO（实现时选一种），至少绑定：

- issuer/audience/key ID、Owner、task ID/version、approval execution ID；
- 输入及隐私清洗后资产 SHA-256；
- processor、adapter contract version、Provider account/project/endpoint/region/model；
- operation、规范化参数、prompt/mask hashes、数量；
- 最大含税费用、币种、幂等键、not-before、expiry、nonce。

Image Service 验证签名、audience、时间、nonce、manifest 和本地单次消费记录。参数任何变化均拒绝；令牌不能授权发布。密钥轮换保留有限重叠窗口并记录 key ID。

当前 P0 实现采用 HMAC SHA-256 短期令牌，密钥由 `IMAGE_SERVICE_EXECUTION_TOKEN_SECRET` 单独配置，禁止与内部 HTTP 的 `IMAGE_SERVICE_SHARED_SECRET` 复用。令牌只在 LingMirror 后端与 Image Service 之间传递；Owner API、MCP 输入输出和日志均不返回令牌。确定性处理保留无付费执行路径；任何非确定性执行必须先消费与 job、Owner、task/version、manifest、operation、processor、费用上限、币种、audience 和 nonce 绑定的一次性令牌。外部 Provider 尚未注册，因此即使授权有效也保持失败关闭，不产生真实外呼。

## 6. Job 与 Provider状态

```text
job: QUEUED → RUNNING → READY | PARTIAL | RECONCILE_REQUIRED | FAILED
attempt: CREATED → SUBMITTING → SUBMITTED → PROCESSING
         → SUCCEEDED | FAILED | RECONCILE_REQUIRED | UNKNOWN
blob: STAGED → READY | QUARANTINED | DELETE_PENDING | DELETED
```

Worker 使用数据库 lease 和 `FOR UPDATE SKIP LOCKED`；崩溃后可接管。外部 mutation 网络重试复用原 Provider 幂等键。Provider 不支持幂等/查询时标 `SANDBOX_ONLY` 或 `MANUAL_ONLY`；当前 Photoroom 在公开契约下不得标 `PRODUCTION_SAFE`。

## 7. BlobStore

合同：`PutStaged/Open/Promote/Delete`。对象 key 内容寻址且不可覆盖，保存 version/etag/SHA-256/MIME/尺寸。流程为：staging 写入 → 流式校验 → DB 记录 → outbox promote → ready。只有 ready 可返回 LingMirror；读取再次复算哈希。

拒绝 SVG/PDF/动画及未知 MIME；限制字节、像素、帧和解码尺寸；服务端抓取使用域名 allowlist，禁止私网、重定向重验和 SSRF。所有外发图片先本地重编码并清除非必要 EXIF/XMP/IPTC；渠道要求的来源字段在最终派生阶段按规则重新写入并验证。

## 8. MCP Server

Transport：远程使用 Streamable HTTP；本地开发可使用 stdio。工具返回 `structuredContent`，输入输出使用 Zod/JSON Schema，名称使用 `lingmirror_image_*`：

- `lingmirror_image_list_capabilities`：只读、分页；
- `lingmirror_image_estimate`：只读估算，不形成批准；
- `lingmirror_image_submit_approved_task`：只接收 LingMirror 内部 task ID 与幂等键；后端自行查找目标绑定批准并签发执行令牌，不接收任意 Provider key；
- `lingmirror_image_get_task`：只读状态；
- `lingmirror_image_list_outputs`：只读元数据和受控引用；
- `lingmirror_image_reconcile_task`：有外部读取，不创建新付费意图。

工具 annotations 明确 `readOnlyHint/destructiveHint/idempotentHint/openWorldHint`。错误必须告诉 Agent下一步是等待、请求 Owner批准、重新核验权利还是人工对账。MCP不暴露任意 URL抓取、任意文件系统、Provider凭据、发布、Owner审批或“标记合规”工具。

小Q不直接连接 Image Service 的生产 MCP；它调用 LingMirror 注册 Capability，由后端检查Owner scope和经营闸门，再转发受控请求。MCP handler与HTTP handler调用同一个 Application Service，禁止复制业务逻辑。

### 8.1 当前生产工具冻结合同

生产入口为 JWT 保护的 `POST /api/v1/product-images/mcp`。所有工具从认证上下文取得 Owner，输入 schema 均为 `additionalProperties: false`；客户端不能传 `owner_id`、执行令牌、Provider key、URL 或文件路径。

| 工具 | 输入 | 行为 | annotations (`readOnly/destructive/idempotent/openWorld`) |
|---|---|---|---|
| `lingmirror_image_list_capabilities` | `page/page_size`，默认 1/20，最大 100 | 分页返回明确 `available/unavailable`；不根据名称推断可用 | `true/false/true/false` |
| `lingmirror_image_estimate` | `task_id` | deterministic 返回精确 `0 USD`；未配置外部 Provider 返回 `unavailable`；绝不创建批准、预算预占、attempt 或付费意图 | `true/false/true/false` |
| `lingmirror_image_get_task` | `task_id` | 返回 Owner 隔离的 LingMirror task 状态 | `true/false/true/false` |
| `lingmirror_image_list_outputs` | `task_id/page/page_size` | 只返回 SHA-256 与 `/api/v1/product-images/tasks/{id}/output/content` 受控引用，不返回任意 URL | `true/false/true/false` |
| `lingmirror_image_reconcile_task` | `task_id` | 只对账既有 Provider 意图；当前 deterministic 明确 `RECONCILE_NOT_SUPPORTED`，未配置外部 Provider 明确 `PROVIDER_UNAVAILABLE`，两者均不创建付费意图 | `false/false/true/true` |
| `lingmirror_image_submit_approved_task` | `task_id/idempotency_key` | deterministic 幂等提交；付费 task 由同一领域 Service 查找有效 Owner 批准与成本记录，服务端即时签 token 后仅通过私有 HTTP 发送 | `false/true/true/true` |

MCP 请求、响应、`structuredContent`、文本内容和工具 schema 均不得出现执行 token。JSON-RPC request 没有 `id` 时视为 notification：只读工具可执行并返回 HTTP 204 空响应；会消费批准或创建 attempt 的 mutation notification 必须在写入前拒绝并同样返回 204。显式 `id: null` 仍返回 JSON-RPC response。

工具级封闭错误统一包含 `error_code/message/next_action/retryable`。下一步只能是等待并重读、请求 Owner 对精确版本批准、修正参数、配置并验证 Provider，或人工对账；错误不得建议新建付费意图来绕过不确定状态。JSON-RPC 协议错误继续使用标准 `-32600/-32601/-32602/-32603`。

当前 route catalog 将混合读写的 MCP transport 登记为 `standard`，因为同一路径承载只读工具，不能在 HTTP 中间件层统一要求写审批；真正的付费 mutation 继续在 `Service.Execute` 内消费目标绑定批准、成本记录和一次性服务端令牌，MCP handler 不复制该门禁。

## 9. 回调与对账

Provider webhook（若支持）验证签名、时间窗和重放 nonce，原始响应先持久化再解析。没有可靠 webhook/query/idempotency 的 Provider，响应断流后进入 UNKNOWN，自动重试为零；Owner若承担重复费用必须在 LingMirror建立新意图和新批准。

Image Service向LingMirror发送的事件通过签名outbox回调，LingMirror按event ID幂等消费；回调失败不丢任务，可由查询接口恢复。

## 10. 安全与可观测性

- 双向服务认证、私有网络、最小权限和secret storage；
- Provider凭据不进入MCP、日志、数据库普通列或前端；
- 日志字段allowlist，prompt、URL和响应默认脱敏；
- 每任务 correlation ID、request manifest hash、approval ID和Provider ref；
- 指标：队列深度、lease过期、成功率、P95、费用、UNKNOWN、重复提交、blob不一致；
- kill switch按Provider/account/model/operation关闭新提交，不删除历史；
- 备份恢复后全量抽样复算blob hash。

## 11. 实施顺序

1. 冻结OpenAPI、MCP schemas、令牌claims和错误码；
2. 建服务骨架、数据库、Runner/outbox/lease和BYTEA BlobStore；
3. 完成deterministic processor与LingMirror端到端草稿路径；
4. 完成MCP只读工具和10个稳定、可验证的read-only evaluations；
5. 完成manual import与对象安全；
6. 接Photoroom sandbox并验证断流/重复费用边界；
7. 接Adobe、OpenAI；
8. 满足正式合同后逐Provider升级production安全等级；
9. 完成运维、备份恢复、canary和Owner runbook。

## 12. 验收

- 同key同payload并发100次只产生一个job；不同payload返回409；
- 令牌任一claim变化、过期、重放或错audience均0次外呼；
- Provider已处理但响应断流时不自动重复付费；
- worker在任一崩溃点重启后收敛；
- blob篡改、MIME伪造、图片炸弹、SSRF均阻断；
- MCP不能绕过LingMirror审批、预算或发布；
- deterministic/manual E2E必跑，Provider fixture必跑，真实sandbox仅定期canary；
- Image Service成功不能使Listing自动批准或发布；
- 发布字节必须与LingMirror image release attestation一致。

工程通过为`automated_verified`；真实Provider调用为注明环境的`manually_verified`；真实渠道结果仍由LingMirror记录为`external_observed`。

## 14. 审查裁决：MCP拓扑与Prism迁移

本节优先于前文冲突简写。

### 14.1 MCP唯一生产入口

生产MCP Server属于LingMirror Backend；Image Service生产环境只暴露私有HTTP，最多保留dev-only stdio调试适配器。Agent只向LingMirror MCP提交内部task引用，执行令牌由后端从安全存储即时签发并仅用于server-to-server HTTP，绝不进入模型参数、MCP transcript或客户端日志。所有工具从当前认证subject解析Owner scope，客户端不能声明Owner。

### 14.2 Prism替代状态

Image Service目标上正式替代Prism，但当前状态只能记为`planned replacement`，尚不能宣称已覆盖。inventory必须同时覆盖：

- multisell内`imagegen.Prism`、`prismadapter`、productanalysis、listingtask、loop、listing JSON、配置、路由策略、自动文档和前端硬编码；
- 独立Git仓库`/Users/lc/prism`中的异步服务、七段pipeline、人工review、YAML templates、Tongyi/Fal.ai Provider、compliance validator、S3封装、metrics、Go client、测试与实验资产；
- `product_image_gen`、`product_canvases`、`prompt_template`及legacy迁移，以及`/image-gen`列表、详情、canvas页面。

每项必须裁决`reuse/rewrite/superseded/archive`。独立Prism仓库先记录commit并打只读归档tag，不直接删除历史。

### 14.3 旧合同事实

multisell旧client请求`/api/v1/generate`并期待同步`output_url/compliance_report`；独立Prism暴露`/v1/generate`并返回202后查询job。旧链应记为`implemented but contract-incompatible / production unknown`，不得作为已工作golden基线。旧Prism output URL仅见于metadata，尚无证据证明真正替换了平台发布字节。

迁移不提供伪同步facade；逐调用方改为异步任务。数据迁移可回滚，发布attestation切换只能forward-fix/fail-closed，不能恢复未经批准的旧直接外呼。

### 14.4 新增服务前置

- 增加输入资产进入Image Service的受控pull或短期单对象上传合同；
- create、retry、reconcile、remote-delete分别使用action-bound服务命令凭证；
- 回调携带job version、attempt number和单调sequence；
- 补health/readiness、迁移锁、优雅停机、lease drain、资源限制、备份恢复和密钥轮换；
- MCP逐工具定义input/output schema、annotations、scope和净化后的封闭错误；
- 评测覆盖跨Owner、过期/重放批准、manifest变化、并发submit、恶意Provider错误、output越权和断线重放。

### 14.5 Prism安全切换顺序

1. 先冻结`trigger-prism`直接外呼，并禁止`prismStrict=false`在失败后继续发布；
2. 建立两仓完整inventory和能力disposition；
3. 迁移可复用算法、fixtures和历史数据，不迁移mock回退或启发式“合规通过”；
4. 逐调用方切到Image Service异步合同；
5. 发布领域原子启用image release attestation并fail-closed演练；
6. 双读、回填、数量/哈希对账、停止旧写；
7. 迁移前端、routecatalog、mutation policy、配置、审计和文档；
8. 观察无调用后删除multisell旧client和死代码，归档独立Prism仓库。

## 15. Prism 替代与迁移详细清单

Image Service 正式覆盖旧 Prism，不作为两个长期并存项目。该裁决只改变未来开发路径，不代表旧代码已经删除。

### 15.1 现有 Prism 的真实分类

| 现有部分 | 当前事实 | 新归属 |
|---|---|---|
| `domain/imagegen/prism.go` | 确定性缩放、白底、水印和内存缓存，不是生成式 AI | 迁移到 deterministic processor |
| `domain/imagegen` CRUD/canvas/template | 通用记录与画布，缺经营证据和审批 | 有用数据迁移；旧写入口冻结 |
| `prismadapter` | 薄 HTTP generate client，返回 URL 与简化 compliance report | 由标准 Processor adapter/Runner 取代 |
| `productanalysis/trigger-prism` | 可直接触发外部 Prism | 迁移到 LingMirror productimage 任务与审批 |
| `listingtask` Prism gate | 发布前调用 Prism，并把简化报告写入 published data | 由 image release attestation 强制门禁取代 |

旧 `POST /listings/:id/publish`、`POST /listing/products/:product_id/publish/:platform_id`、`POST /listing/listing-tasks/:task_id/publish` 以及 `listingtask.PublishHook` 已失败关闭。即使配置 Prism 或商品主图包含任意 URL，也不会调用 Prism 或平台 adapter；调用方必须改用受控发布 attempt。
| `loop` Prism 依赖 | 历史工作流依赖 | 迁移或冻结，不进入新图片执行链 |
| `PRISM_*` 配置 | 旧外部服务配置 | 过渡期兼容，最终由 Image Service provider config 取代 |

旧 Prism 的“合规报告”不能升级为图片权利、商品真实性或渠道合规事实。

### 15.2 迁移顺序

1. 建立新合同、Image Service deterministic processor 与 LingMirror productimage 资产；
2. 对旧 Prism 数据、配置、路由和调用方生成完整 inventory；
3. 将确定性算法及测试迁入新 processor，做逐字节或明确容差的 golden 对比；
4. 新增兼容 facade，使旧调用方暂时转向新任务/结果，不再直接访问外部 Prism；
5. `listingtask` 改为只验证 LingMirror 签发的 image release attestation，禁止在发布过程中临时生成或“边检查边发布”；
6. 双读旧数据并回填资产、attempt、结果和来源状态；无法证明来源的历史记录标 `legacy/unknown`；
7. 前端移除“Prism 已配置/合规通过”等误导展示，改为统一图片工作室状态；
8. 停止 `trigger-prism`、旧 CRUD 和旧配置的新写入，保留有期限的只读查询；
9. 观察期内确认无调用、无数据差异和可回滚后，删除旧路由、适配器、配置和死代码；
10. 同步更新 AGENTS.md、CLAUDE.md、docs/INDEX.md、自动路由目录和运行手册，并生成新的工程验证记录。

### 15.3 迁移保护

- 不直接删除旧表、路由或配置；先 inventory、双读、回填、停止写、观察、再删除；
- 旧 URL-only 结果必须下载并校验后才能成为新资产，无法取得原字节则保持历史引用；
- 旧 compliance `pass` 不自动转换为新 review passed；
- 迁移不能触发 Provider、发布或其他外部写；
- 旧 API 在过渡期返回明确 deprecated 信息和替代资源，但不得静默改变响应语义；
- 回滚可恢复旧只读路径，但不得恢复未经批准的直接外部生成。

### 15.4 完成条件

- 仓库中不存在生产调用 `prismadapter.Generate` 的路径；
- 发布路径只接受有效 image release attestation；
- 旧确定性测试全部在新 processor 下通过；
- 历史记录迁移数量、哈希和 unknown 数量可对账；
- `PRISM_*` 不再是生产必需配置；
- 旧路由、文档和前端展示全部标记或完成迁移；
- 新工程验证记录只声明工程迁移结果，不宣称真实图片或渠道已经验证。
