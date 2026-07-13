# Owner 经营决策与反馈案卷

状态：`implemented / automated_verified`（工程事实）；真实经营使用仍为 `unknown`。

Owner 入口为 `/business-decisions`。案卷只能从 Owner 隔离的权威事实选择器创建，目前支持 `platform_order_ingest` 与 `purchase_authority`。创建时冻结事实 snapshot 与 manifest，不触发外部行动。

API：

- `GET/POST /api/v1/business-decisions`：Owner 案卷列表与创建；
- `GET /api/v1/business-decisions/fact-options`：只读权威事实选择器，不产生新事实；
- `GET /api/v1/business-decisions/:id`：冻结事实、AI 建议和 Owner 决定；
- `GET/POST /api/v1/business-feedback/actions`：Owner 隔离的受控行动列表与创建；
- `GET /api/v1/business-feedback/actions/:id`：执行、权威观测与下一动作建议的可恢复明细。

`selected` Owner 决定必须冻结 capability、command、target 和服务端规范化后的 exact input payload/SHA-256。受控行动创建和执行会再次逐字段核验最新决定与审批；新决定可撤销尚未执行的旧行动。观测只接受同一经营对象的登记权威事实，下一动作建议固定为 `inferred`，不会自动形成下一次 Owner 决定或外部执行。

迁移 `000143_business_decision_input_payload` 为新决定保存 exact payload。历史 selected 决定只有 SHA 而没有 payload 时可继续只读追溯，但页面会失败关闭，不能据此重建执行载荷。
