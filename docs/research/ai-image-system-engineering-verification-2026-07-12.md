# AI 商品图片系统工程验证记录

> 日期：2026-07-12
> 依据：`ETHOS.md`、`multi-provider-product-image-system.md`、`image-service-mcp-contract.md`
> 边界：只记录当前代码和本地自动验证；只使用上述图片系统资料

## 当前裁决

独立 Image Service、凌镜 Owner 页面、内部 HTTP 客户端、MCP 入口、确定性图片处理、Listing 图片集合和 OpenAI 适配器合同已经进入代码。当前可执行路径仍仅为无外部费用的确定性处理；OpenAI 适配器没有注册到生产 Worker，普通 API、页面和 MCP 均不能触发真实 OpenAI 外呼。

## 已实现并自动验证

- 静态 JPEG/PNG 上传后重编码、去元数据、限制 10 MiB/40 MP、内容寻址保存和读取复算哈希；
- 持久化 Job/Attempt、lease 接管、幂等创建和同 Job 单一活跃 attempt；
- 确定性缩放与白色画布处理；
- OpenAI Image Edit 可插拔适配器合同：超时、响应上限、429/5xx 有限重试、稳定幂等键、请求 ID 和安全输出重编码；
- OpenAI 无密钥失败关闭，且即使配置密钥也尚未注册为可执行 operation；
- Owner JWT 隔离的资产、任务、结果读取、图片集合与 MCP；
- 图片集合保存 task、job、manifest、operation、processor、output blob 的完整血缘；创建和冻结时重新核验 Image Service；
- Listing 必须通过现有商品关系证明 Owner 归属和渠道一致；
- 付费执行授权基础：独立 HMAC 密钥、Owner 批准记录、短期目标绑定令牌、nonce 持久化单次消费；
- 旧 `/product-analysis/trigger-prism` 任意 URL 入口已停止注册，避免继续暴露旧抓取风险；
- 凌镜 `/product-images` 页面从后端读取真实 capability；加载失败、未配置或未知时禁止创建和执行。

## 本次验证命令

```text
services/image-service: go test -race ./... → 53 tests / 9 packages passed
services/image-service: go vet ./... → passed
services/image-service: go build ./... → passed
backend focused race: productimage + imageservice + httpx + routecatalog → 87 tests / 4 packages passed
backend focused race（productimage、productanalysis、imageservice、httpx、routecatalog）→ 95 tests / 4 packages passed
backend full: go test ./... → 当前被并行修改中的 sourcing1688 生命周期测试 10 处失败；图片相关包通过
backend full: go vet ./... + go build ./... → passed
frontend selected suite（product-images、agent-upgrades、api-client）→ 24 tests passed
frontend TypeScript check → 当前被并行修改中的 sourcing1688 页面 2 处既有 props 缺失阻断；图片相关测试本身通过
frontend production build → 图片页面此前通过并包含 /product-images；最终工作树因上述 TypeScript 错误未能重跑成功
Docker production compose merge/config → passed，且生产密钥不继承开发默认值
migration file contract → 113 up/down pairs passed；未提供 DATABASE_URL，未执行真实 PostgreSQL migration
```

## 仍未完成或未知

- 未使用真实 OpenAI、Photoroom 或 Adobe 账号完成 sandbox/生产调用；
- 未提供真实商品、图片处理权利、渠道与类目规则，因此未做真实视觉和渠道验收；
- OpenAI 任务创建和付费执行仍未对 Owner 页面开放；预算目前记录批准上限，但尚未接入可对账的实际费用账本；
- Provider request ID 尚未写入持久化 attempt；
- Image Service 当前持久化为本地 JSON 快照，不等同于 PostgreSQL 生产队列；
- Prism 的旧运行客户端与任意 URL 入口已在路由装配层冻结；Listing/loop 兼容参数、旧数据和独立仓库尚未完成迁移；
- migration 000108/000116 尚未在真实 PostgreSQL 上行、回退和重跑验证；
- Docker daemon 未运行时无法证明镜像实际构建成功。

## 下一验收门

在开放首个外部 Provider 前，必须完成：真实 PostgreSQL migration 验证、实际费用落账与对账、Provider request ID 持久化、断流不重复收费测试、真实 sandbox canary，以及 Owner 页面明确展示预计/实际费用和授权状态。
