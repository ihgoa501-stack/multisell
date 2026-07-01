# LingMirror 凌镜 — 已知风险清单

更新时间：2026-07-01

## 安全风险

| 风险 | 等级 | 缓解措施 | 状态 |
|------|------|----------|------|
| AI Agent 建议错误 | medium | 所有高风险操作需人工审批 | ✅ 已控制 |
| 外部平台凭证泄露 | high | 凭证走环境变量/secret storage，不硬编码 | ✅ 已控制 |
| RBAC 未完全覆盖 | medium | 关键路由已加 RequirePermission，持续补齐中 | 🔄 持续 |
| SQL 注入 | low | GORM 参数化查询 | ✅ 已控制 |
| JWT 密钥泄露 | high | 走环境变量，支持轮换 | ✅ 已控制 |

## 功能风险

| 风险 | 等级 | 缓解措施 | 状态 |
|------|------|----------|------|
| LLM 调用超时 | medium | Orchestrator 记录 failed trace，不阻塞其他 Agent | ✅ 已控制 |
| Agent 数据不完整 | low | completeness 引擎检查 12 维资料，缺失则 block | ✅ 已控制 |
| 平台同步数据延迟 | low | 同步状态可见，Owner dashboard 可监控 | ✅ 已控制 |
| 审批积压 | low | SLA 超期自动 escalate，Owner 仪表盘可见 | ✅ 已控制 |

## 技术风险

| 风险 | 等级 | 缓解措施 | 状态 |
|------|------|----------|------|
| 数据库迁移失败 | medium | down migration 存在，可回滚 | ✅ 已控制 |
| 全局锁/性能 | low | 当前单实例运行，扩展需加分布式锁 | 🔄 待评估 |
| 内存泄漏 | low | 基础 metrics 监控，Sentry 异常捕获 | 🔄 持续 |
