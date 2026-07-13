# AI 商品图片系统工程验证记录

> 日期：2026-07-12
> 最后复验：2026-07-13
> 依据：`ETHOS.md`、`multi-provider-product-image-system.md`、`image-service-mcp-contract.md`
> 边界：只记录当前代码和本地自动验证；只使用上述图片系统资料

## 当前裁决

`actual`：独立 Image Service、LingMirror 图片领域、Owner 页面、MCP、确定性处理、手工导入、精确权利/五轴审核/成本记录、冻结图片集合和发布证明已经进入代码。确定性路径可执行；Photoroom 具备默认关闭的 `sandbox_only` 接线，只有四门显式配置后才注册，production 环境禁止启动。当前没有凭据，因此运行状态仍为 `unavailable`，没有发生真实外呼。

`actual`：后端已有受控媒体发布 attempt 消费方，但没有真实平台 Adapter 实现该字节合同。旧 Listing 发布入口和旧 URL Adapter 均失败关闭，因此当前只能证明发布状态机与隔离边界，不能称为真实渠道发布闭环或生产可用。

## 已实现并自动验证

- 静态 JPEG/PNG 上传后重编码、去元数据、限制 10 MiB/40 MP、内容寻址保存和读取复算哈希；
- PostgreSQL 持久化 Job/Attempt/nonce、事务幂等、`FOR UPDATE SKIP LOCKED` lease 接管和同 Job 单一活跃 attempt；
- 确定性缩放与白色画布处理；
- OpenAI Image Edit 可插拔适配器合同：超时、响应上限、429/5xx 有限重试、稳定幂等键、请求 ID 和安全输出重编码；
- OpenAI 无密钥失败关闭，且即使配置密钥也尚未注册为可执行 operation；
- Owner JWT scope 的资产、任务、结果读取、图片集合与 MCP；
- 图片集合保存 task、job、manifest、operation、processor、output blob 的完整血缘；创建和冻结时重新核验 Image Service；
- Listing 必须通过现有商品关系证明 Owner 归属和渠道一致；
- 付费执行授权基础：独立 HMAC 密钥、Owner 批准记录、短期目标绑定令牌、nonce 持久化单次消费；
- 精确资产权利、撤权、五轴审核与成本记录；图片集合创建/冻结时重复核验；
- 手工导入和渠道内置导入保存工具、操作、模型版本、费用、父图片哈希和渠道限制，导入结果固定为 `unknown`；
- 不可变渠道规则快照、Owner 对精确 frozen set 的决定、逐 blob 复算和 HMAC release attestation；签发不等于发布；
- `/product-images` 页面可录入权利、五轴审核、成本、手工导入、渠道规则、图片集合决定并签发/读取证明；页面不提供证明消费按钮；
- MCP 提供 6 个严格 schema 工具；付费提交不接收 token、URL 或 Provider key，mutation notification 在写入前拒绝；
- MCP 有 10 个固定、独立、只读 XML evaluation；每题通过真实 Handler 执行至少两次只读工具调用并直接比对稳定答案；
- 创建任务前必须显式冻结 processor/purpose/channel/region，并验证输入资产精确、有效的复制和修改权；渠道内置导入限制随资产血缘保留，跨渠道失败关闭；
- 图片 API 与 MCP 需要 `product_image.owner`，迁移仅授予 active Owner/admin 角色；
- 渠道规则使用严格 schema v1，并在证明签发与消费时执行最小尺寸、格式、角色、数量和字节规则；空 `{}` 不能签发证明；
- 受控发布 attempt 只向 `ControlledPublisher` 交付重新读取并复算的精确 bytes/hash/MIME/role；100 并发单外呼，断流进入 `reconcile_required`，成功回执后证明才 `consumed`；
- 旧 URL Adapter 默认未注册，返回 `CONTROLLED_PUBLISHER_UNSUPPORTED` 且零外呼；
- 最终对抗审查发现并修复两条旧 URL 发布旁路：`platform-integrations/publish-to-ozon` 与 `sourcing1688` 旧发布执行的真实 Adapter 代码已移除；真实模式固定返回 `428 IMAGE_RELEASE_ATTESTATION_REQUIRED`，测试证明 Adapter 与业务数据库写入均为零；
- Owner 周期预算使用 PostgreSQL 事务级锁和严格十进制预占；付费批准绑定 exact task/version/manifest/provider，拆任务共享同一总额；100 并发 `0.02 USD` 在 `1.00 USD` 上限下恰好 50 次成功；
- 预算状态为 `reserved → claimed → spent`；外呼未知不释放，未 claim 才可取消/过期释放，实际及迟到费用均追加而不覆盖历史；
- Photoroom sandbox 任务固定三个操作、US、PNG、`0 USD`；执行令牌绑定 sandbox/水印/不可发布限制，PostgreSQL 原子消费一次 canary 配额和单次 provider submit claim；多实例并发只能取得一次外呼资格；
- Photoroom 输入权利在创建、批准、执行前重复核验复制、修改、第三方 AI 和跨境权限，并要求精确 provider/region；撤权或过期会在外呼前阻断；
- Photoroom Provider 原始结果是否带像素水印仍为 `unknown`；Image Service 会本地叠加明显的 `SANDBOX` 像素横幅，PNG 重编码后逐像素验证，只有验证成功才把 `sandbox/watermarked/non_publishable` 事实持久化到 Job 与 LingMirror Task；Image Set 创建/冻结、release attestation 签发/消费和 controlled publish 全部拒绝；MCP 与 Owner 页面只展示限制，不提供生产开关或密钥输入；
- Photoroom 客户端拒绝全部重定向并固定 Host；执行 token 精确绑定 watermarked 和不可变执行权利快照；环境值使用闭集，acceptance/production 服务密钥至少32字节且不能与执行密钥相同；
- Photoroom 执行使用数据库固定锁顺序原子 claim RightsGrant、Approval 和预算并创建不可变 ExecutionRightsSnapshot；真实 PostgreSQL 10轮×100路撤权/执行竞争全部满足“撤权先行零入队”或“执行先行恰好一份快照”；
- Blob readiness 实际执行 create/write/fsync/read/remove，并与 PostgreSQL readiness 同时通过才返回 ready；
- 旧 `/product-analysis/trigger-prism` 任意 URL 入口已停止注册，避免继续暴露旧抓取风险；
- 凌镜 `/product-images` 页面从后端读取真实 capability；加载失败、未配置或未知时禁止创建和执行。

## 本次验证命令

```text
services/image-service: go test -race ./... → 95 tests / 12 packages passed
services/image-service: go vet ./... → passed
services/image-service: go build ./... → passed
backend productimage race → passed，包含受控发布并发100、断流、篡改、跨Owner和unsupported adapter
backend full: go test ./... → 3421 tests / 122 packages passed
backend full: go vet ./... + go build ./... → passed
frontend product-images suite → 25 tests passed
frontend TypeScript check → passed
frontend production build → passed，包含 `/product-images`
Docker Compose config → 使用三个独立、满足长度要求的临时测试 secret 时 passed；缺 secret 时按配置失败关闭
migration file contract → 137 up/down pairs passed
真实本地 PostgreSQL 16 临时库 → 全部 migration（含 `000145_product_image_sandbox_output_facts`、`000146_product_image_execution_rights_snapshot`）up passed；最新迁移 down/up rollback smoke passed；Image Service 多实例 submit/一次 canary 原子消费、图片预算100并发预占和撤权/执行100路互斥通过；临时库随后删除
Prism migration inventory → passed
最终对抗复验 → 五个相关后端包 race 411 passed；两套旧 URL 发布生产代码中 `adapter.Publish(...)` 零命中；未发现新增 P0/P1
```

## 仍未完成或未知

- 未使用真实 OpenAI、Photoroom 或 Adobe 账号完成 sandbox/生产调用。Photoroom 已有默认关闭的 `sandbox_only` 任务链，但没有配置 API key、sandbox 账号确认和 training opt-out 确认，因此当前 capability 为 unavailable；这不证明真实账号、endpoint、配额、输出质量或真实错误语义可用；
- 未提供真实商品、图片处理权利、渠道与类目规则，因此未做真实视觉和渠道验收；
- 尚无真实平台 Adapter 实现 `ControlledPublisher`，因此没有真实平台回执或外部发布观察；
- 并发预算预占已经自动验证；真实 Provider 账单对账仍未发生，迟到费用和超支路径目前只有自动测试证据；
- Provider request ID、断流后查询和实际费用回写尚未用真实 Provider 验证；
- Blob 仍为本地持久卷，不是对象存储；readiness 能发现卷不可写，但没有完成备份恢复演练；
- Prism 旧运行客户端、Product Analysis 触发器、ListingTask/Loop 依赖、配置和前端硬编码已清除；历史 `imagegen` 数据/算法和独立仓库保留待归档与数据迁移；
- 本机 Docker daemon 未运行，因此没有证明镜像实际 build/start；
- 对抗审查发现的输入权利、渠道限制继承、结构化渠道规则、Owner 专属权限和远端任务身份问题已完成代码修复和自动验证；真实数据人工验收仍未知。

## 下一验收门

下一验收只允许一次 Photoroom 带水印、不可发布的真实 sandbox canary：Owner 提供已确认权利的非敏感商品图、sandbox 账号/API key、training opt-out 证据和明确的一次外呼授权。其后仍需人工核对输出、request ID、配额变化和断流语义；生产 Provider 与真实渠道发布继续关闭。
