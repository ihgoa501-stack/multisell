# Owner 工作台说明

## 概述

Owner 工作台（`/owner`）是凌镜可信经营闭环的核心入口。Owner（卖家/经营者）通过工作台查看 Agent 建议、做出决策、追踪执行结果。

## Owner 现在能做什么

### 1. 查看风险摘要

`GET /api/v1/owner/risk-summary`

返回 Owner 经营总控台的聚合风险指标：

- `total_candidates` — 候选商品总数
- `missing_data_products` — 资料不完整商品数
- `low_profit_products` — 低利润/负利润商品数
- `pending_approvals` — 待审批的 listing task 数
- `sync_errors` — 同步失败数
- `total_recommendations` — 总建议数
- `list_ready_products` — 建议上架的商品数

### 2. 查看 Agent 建议

`GET /api/v1/owner/suggestions?limit=20`

返回最新的 Agent Listing 建议，每项包含：

| 字段 | 说明 |
|------|------|
| `id` | 建议 ID |
| `product_id` | 商品 ID |
| `product_title` | 商品标题 |
| `completeness_score` | 资料完整度评分 (0-100) |
| `profit_margin` | 预估利润率 (%) |
| `estimated_profit` | 预估利润 ($) |
| `decision` | Agent 建议：`list` / `cautious` / `skip` |
| `confidence` | 置信度 (0-1) |
| `reason` | 决策原因 |
| `risk_level` | 风险等级：`low` / `medium` / `high` |
| `risk_flags` | 风险标记 |
| `feedback_status` | 反馈状态：`pending` / `adopted` / `rejected` / `executed` / `execution_failed` |
| `feedback_note` | 反馈备注 |
| `listing_task_id` | 关联的上架任务 ID |
| `task_status` | 上架任务状态 |
| `approval_id` | 关联的审批请求 ID |
| `approval_status` | 审批状态 |

### 3. 决策队列

Owner 的典型决策流程：

```
查看建议 → 理解原因 → 批准/拒绝 → 跟踪执行
```

#### 采纳建议

`POST /api/v1/owner/suggestions/:id/feedback`

```json
{"action": "adopt", "note": "商品资料完整，利润良好，同意上架"}
```

采纳后自动：
1. 更新建议 `feedback_status` = `adopted`
2. 更新上架任务状态为 `pending_approval`
3. 创建审批请求（`entity_type=listing_task`）

#### 拒绝建议

```json
{"action": "reject", "note": "利润偏低，暂不上架"}
```

拒绝后自动：
1. 更新建议 `feedback_status` = `rejected`
2. 更新上架任务状态为 `rejected`

### 4. 审批操作

`PUT /api/v1/approval/:id/review`

```json
{"action": "approve", "reviewer": "owner", "review_note": "确认上架"}
```

或拒绝：
```json
{"action": "reject", "reviewer": "owner", "review_note": "需要重新评估"}
```

### 5. 执行上架（审批通过后）

`POST /api/v1/listing-task/:task_id/execute`

执行上架前自动通过 **执行门禁**（Execution Gate）：

1. 任务存在性验证
2. 幂等性检查（已完成的任务不重复执行）
3. 状态机校验（仅 approved 状态可执行）
4. ApprovalID 存在性
5. 审批记录校验（必须 approved、类型/ID 匹配）
6. 审计日志记录

## 风险等级说明

| 等级 | 判断条件 | 说明 |
|------|----------|------|
| `low` | decision=list 且 confidence>=0.6 | 低风险，可自动执行 |
| `medium` | decision=cautious 或 confidence<0.6 | 中风险，需要 Owner 判断 |
| `high` | decision=skip | 高风险，不建议上架 |

## Sandbox/Mock 说明

以下功能目前是沙盒模式，**不操作真实电商平台**：

- **上架执行** — `ExecuteTask` 仅更新数据库状态，不调用平台 API
- **同步状态** — `mock_sync_status` 提供模拟的同步状态
- **订单/结算** — `mock_order` / `mock_settlement` 提供演示数据

## 高风险动作门禁

以下动作需要经过审批 + 状态机双门禁：

| 动作 | 需要审批 | 可执行状态 |
|------|----------|------------|
| 上架（listing） | 是 | approved |
| 价格变更（price、planned） | 是 | —（待实现） |
| 下架（delist） | 是 | —（待实现） |
| 内容更新 | 否 | — |

当前仅实现了上架执行的门禁。其他高风险动作将在后续迭代中补充。

## 如何验证闭环

参见 [Testing Guide](./testing.md#闭环集成测试) 了解如何运行验证。

### 快速验证步骤

1. **创建候选商品**：`POST /api/v1/candidates`
2. **评估**：`POST /api/v1/loop/evaluate/:productId` → 生成建议 + listing task
3. **查看建议**：`GET /api/v1/owner/suggestions`
4. **采纳**：`POST /api/v1/owner/suggestions/:id/feedback {"action":"adopt"}`
5. **审批**：`PUT /api/v1/approval/:id/review {"action":"approve"}`
6. **执行**：`PUT /api/v1/listing-tasks/:id {"status":"approved","approval_id":<id>}`
7. **完成**：`POST /api/v1/listing-task/:id/execute`

或直接运行集成测试验证：
```bash
cd backend-go && go test ./internal/integrationtest/... -v -run TestTrustedClosedLoop
```
