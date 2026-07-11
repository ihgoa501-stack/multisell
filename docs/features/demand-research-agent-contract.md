# 候选市场 AI 研究输入契约

> 生效日期：2026-07-11。适用于 `/api/v1/demand-cases/research/import`。

## 三类独立运行

- `scout_result`：只能建立候选市场线索与支持性公开证据，不能批准实验。
- `falsifier_result`：必须使用不同 `run_id`，只能追加反证。
- `data_reality_result`：只记录字段可得性，不得提交需求结论。

每次提交必须包含来源 URL、采集时间、原始 payload 和可重算 SHA-256。相同 Owner 下重复 `run_id + run_type` 只有来源、批次和 hash 完全一致时才视为幂等；内容变化会拒绝。

每个 run 还必须保存 `collector` 身份。`falsifier_result` 的 collector 必须与同案件 scout 不同；只修改 run ID 不能冒充独立反证。

## 数据可得性固定状态

`available / requires_owner_access / requires_listing / requires_transaction / unavailable / unknown`

只有当前预检必需且最新状态为 `requires_owner_access` 的字段可以生成权限卡，并且必须写明最窄只读 scope、服务的经营决定和拒绝授权的后果。后续独立 run 的 `available` 可以取代旧 unknown/未授权状态；历史记录保留但不永久阻塞。`requires_listing` 与 `requires_transaction` 不是只读权限，必须等待未来独立实验和 Owner 审批。

## 禁止事项

- 不能把搜索、评论、榜单、公开销量估计写成付款。
- 不能提交无来源数字、虚构消费者画像或模型补值。
- 不能复用侦察 run 充当独立反证。
- 不能通过研究接口采购、发布、投放、改价或操作资金。
- API 默认不返回原始 payload；Owner 页面只显示来源、hash、裁决和必要的去敏字段。
