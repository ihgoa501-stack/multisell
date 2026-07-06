# 告警规则

| 规则 | 级别 | 条件 | 通知对象 |
|------|------|------|----------|
| R1: Agent 连续失败 | P1 | 同一 Agent 连续失败 > 3 次 | Owner |
| R2: 订单同步中断 | P1 | 任一平台同步中断 > 30 分钟 | Owner |
| R3: 利润率异常 | P2 | 任一 SKU 利润率 < -10% | Owner |
| R4a: LLM 预算超 80% | P3 | 月度 LLM 花费 > 预算 80% | Owner (警告) |
| R4b: LLM 预算超 100% | P0 | 月度 LLM 花费 ≥ 预算 100% | Owner + 自动停 Agent |
| R5: 同步失败率 | P2 | 任一平台同步失败率 > 5% | Owner |

## 通知目标

- Owner WebSocket: 实时推送
- 系统通知: `notifications` 表记录
