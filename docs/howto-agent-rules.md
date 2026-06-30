# How to 创建自定义 Agent 规则

> 为凌镜 AI Agent 编写行为规则，控制 Agent 的决策边界和触发条件。

---

## 前置条件

- 了解 Agent 清单（参见[模块目录 - AI & Agent 层](reference-module-catalog.md#2-ai--agent-层)）
- 熟悉 Agent 决策点概念（decision point）
- 后端开发环境已就绪

## 概念

Agent 规则 (Agent Rule) 定义 Agent 在某决策点上的行为约束：

```
Agent 收到调度 → 进入决策点 → 检查规则 → 执行动作 → 发布 agent.decided.* 事件
                         ↓
                    规则可以: 允许 / 禁止 / 修改参数 / 要求审批
```

每条规则包含：
- **Agent ID** (`A5`, `G3` 等)
- **决策点** (`stock_alert`, `discount_risk_check` 等)
- **条件表达式** — 什么情况下规则生效
- **动作** — `allow` / `block` / `require_approval` / `modify_params`
- **优先级** — 多条规则冲突时的排序

## 步骤

### 1. 通过 API 创建规则

```bash
curl -X POST http://localhost:8080/api/v1/agent-rules \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "A5",
    "decision_point": "stock_alert",
    "name": "高价商品不自动调价",
    "condition": {
      "field": "product.price",
      "operator": "gt",
      "value": 1000
    },
    "action": "block",
    "priority": 100,
    "enabled": true,
    "description": "单价超过 1000 的商品，A5 库存预警不自动建议调价"
  }'
```

### 2. 规则字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `agent_id` | string | Agent 标识（A5, G3, A6 等） |
| `decision_point` | string | 决策点名称 |
| `name` | string | 规则名称（唯一标识） |
| `condition` | object | 触发条件（field + operator + value） |
| `action` | string | 动作: `allow`, `block`, `require_approval`, `modify_params` |
| `priority` | int | 优先级（值越大越优先） |
| `enabled` | bool | 是否启用 |
| `description` | string | 规则说明 |

### 3. 条件运算符

| operator | 说明 | 示例 |
|----------|------|------|
| `eq` | 等于 | `{"field": "stock", "operator": "eq", "value": 0}` |
| `gt` | 大于 | `{"field": "price", "operator": "gt", "value": 1000}` |
| `lt` | 小于 | `{"field": "confidence", "operator": "lt", "value": 0.5}` |
| `in` | 包含于 | `{"field": "platform", "operator": "in", "value": ["ozon", "shopee"]}` |
| `contains` | 包含 | `{"field": "product_name", "operator": "contains", "value": "奢侈品"}` |

### 4. 规则冲突处理

当多条规则匹配时，按优先级排序。同优先级按创建时间倒序（最新的优先）。

- `block` 动作权重最高——任一条规则 block 就 block
- `require_approval` 需要 Owner 在 AgentOS 总控台手动审批
- `allow` 覆盖同优先级的 block 规则
- `modify_params` 需要上层决策是否采纳修改

### 5. 通过 AgentOS UI 管理规则

打开 `http://localhost:3000/agents/actions` → 选择 Agent → 进入「规则」标签页：

- 查看该 Agent 的全部规则
- 启用/禁用规则
- 测试规则（输入模拟数据验证规则效果）

### 6. 查看规则执行日志

每条规则匹配结果会被审计日志记录。查询：

```bash
curl http://localhost:8080/api/v1/operation-logs \
  -H "Authorization: Bearer <token>" \
  -d '{"module": "agent:A5", "action": "rule_match"}'
```

## 最佳实践

- **先从 `block` 开始**：新规则先用 block 模式测试，确认匹配逻辑正确后改为 `require_approval` 或 `allow`
- **优先级间隔用 100**：留出插入空间，不要用 1,2,3 这种紧邻优先级
- **规则越具体越好**：`price > 1000 AND platform = ozon` 比裸 `price > 1000` 更少误伤
- **周期审查**：每季度检查一次规则有效性，删除过期的

## 验证

```bash
# 确认规则已创建
curl http://localhost:8080/api/v1/agent-rules \
  -H "Authorization: Bearer <token>"

# 手动触发 A5 检查
curl -X POST http://localhost:8080/api/v1/agents/A5/run \
  -H "Authorization: Bearer <token>" \
  -d '{"decision_point": "stock_alert", "context": {"product_id": 1, "price": 2000}}'

# 查看 A5 决策日志是否包含规则命中记录
```

## 故障排查

| 问题 | 原因与解决 |
|------|-----------|
| 规则从未匹配 | 检查 `condition.field` 路径是否与 Agent 决策上下文字段一致 |
| Agent 不运行 | 确认规则不是 `action: block` 类型导致被拦截；检查调度器是否在运行 |
| 规则被忽略 | 检查优先级是否被更高优先级覆盖；检查规则 `enabled` 是否为 true |

---

## 相关文档

- [参考 - 模块目录（AI 治理模块）](reference-module-catalog.md#8-ai-治理模块)
- [解释 - Agent Pipeline 和事件驱动编排](explanation-agent-pipeline.md)
- [系统架构 - 决策全生命周期](../system-architecture-design-v1.md#6-决策全生命周期状态机)
