# 真实付费需求只读权限预检

> 当前状态：只生成权限决策，不执行登录或平台调用。

## 当前结论

- England 私人租客潮湿场景：独立反证后 `hold + evidence_missing`，暂不请求 Amazon 权限。
- 美国 65 岁以上行动困难场景：独立反证后 `hold + evidence_missing`，暂不请求 Amazon 权限。
- 日本老龄人口：当前候选表述 `reject`。

因此本轮权限请求数量为 0。只有新的具体场景证据推翻上述反证后，才重新生成最小只读权限卡。

完整侦察与独立反证见 `deliverables/research/2026-07-11-live-market-permission-batch.md` 和 `deliverables/research/2026-07-11-live-market-independent-falsification.md`。

## Owner 授权边界

当前只允许：Owner 本人登录对应 marketplace，人工查看 Product Opportunity Explorer，并保存 marketplace、查询词、niche 定义、字段、时间窗、无结果/低量结果、观察时间和快照。

当前禁止：Listings、Feeds、Pricing、广告、订单 PII、采购、发布、改价、投放或任何外部写入。需要真实 listing 或 transaction 才产生的数据保持 `requires_listing / requires_transaction`，不能伪装成只读授权。

## 通过与停止

只读预检通过只表示取得了可用于继续反证的账号内字段，不表示市场、需求或商品成立。出现以下任一情况即停止该渠道预检：

- 无工具权限或无 matching niche；
- 场景购买信号弱、季节性或替代解释更强；
- 退货、竞争或合规明显不利；
- 关键费用继续 unknown；
- Owner 不批准账号固定成本或最小只读访问。

平台真实调用必须在 Owner 明确授权后，按 `docs/ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md` 的环境和凭证规则执行；不得打印或写入研究文档中的密钥。
