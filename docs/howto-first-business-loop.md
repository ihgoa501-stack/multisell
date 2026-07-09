# How to: 执行第一个业务闭环

> 从候选商品到受控发布，完成完整的 "商品创建 → 评估 → 审批 → 上架" 流程。
> 前提：你已经按 [入门教程](tutorial-getting-started.md) 启动了本地开发环境。

---

## 概览

一条商品从选品采集到实际发布到电商平台，需要经过 5 个阶段：

```
采集 → 候选商品 → 完整度检查 → 利润测算 → 上架建议 → 审批 → 上架任务 → 发布
```

每个阶段都有决策点和安全门禁。

---

## 前置条件

- 本地后端在运行：`localhost:8080`
- 本地前端在运行：`localhost:3000`
- 已登录系统（获得 JWT Token）
- 数据库已 migrate（`docker compose up -d db`）

---

## Step 1: 创建候选商品

候选商品来自两个渠道：手动创建或 Agent 采集。这里展示手动创建。

**先获取 JWT Token：**

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
```

**创建候选商品：**

```bash
curl -s -X POST http://localhost:8080/api/v1/candidates \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "便携式蓝牙音箱 BTS-100",
    "description": "IPX7防水便携蓝牙音箱，支持TWS串联，续航12小时",
    "purchase_price": 45.00,
    "purchase_currency": "CNY",
    "package_weight_kg": 0.35,
    "package_length_cm": 12,
    "package_width_cm": 12,
    "package_height_cm": 10,
    "origin_country": "CN",
    "status": "draft",
    "source_url": "https://detail.1688.com/xxx",
    "source_platform": "1688"
  }'
```

成功返回：
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "title": "便携式蓝牙音箱 BTS-100",
    "status": "draft"
  }
}
```

**前端操作**：访问 `http://localhost:3000/candidates` → 点击 "新建候选商品" → 填写表单 → 保存。

---

## Step 2: 检查完整度

创建后评估商品资料完整度。系统从 12 个维度评分：

- 标题 ✓ / 描述 ✓ / 主图 / 图片 / 规格 / 采购价
- 物流重量尺寸 ✓ / 原产国 / 品牌 / 分类 / 供应商 / 关税编码

```bash
curl -s -X POST "http://localhost:8080/api/v1/completeness/check/1" \
  -H "Authorization: Bearer $TOKEN"
```

返回示例：
```json
{
  "code": 0,
  "data": {
    "product_id": 1,
    "overall_score": 58.3,
    "dimensions": {
      "title": 100,
      "description": 80,
      "main_image": 0,
      "images": 0,
      "specs": 0,
      "purchase_price": 100,
      "weight_dimensions": 100,
      "origin_country": 100,
      "brand": 0,
      "category": 0,
      "supplier": 0,
      "hs_code": 0
    },
    "missing_items": ["主图", "规格", "品牌", "分类", "供应商", "关税编码"],
    "level": "incomplete"
  }
}
```

**建议**：回补至少 70% 再进入下一步。缺失项（图片、规格、品牌等）可以通过 `PUT /v1/candidates/:id` 补充。

---

## Step 3: 测算利润

完整度达标后，计算利润。利润公式：

```
利润率 = (预计售价 - 采购成本 - 物流费 - 平台费 - 关税) / 预计售价
```

```bash
curl -s "http://localhost:8080/api/v1/profit/summary/1" \
  -H "Authorization: Bearer $TOKEN"
```

返回示例：
```json
{
  "code": 0,
  "data": {
    "product_id": 1,
    "purchase_price_cny": 45.00,
    "estimated_logistics": 18.50,
    "estimated_platform_fee": 8.00,
    "estimated_tariff": 3.50,
    "total_landed_cost": 75.00,
    "suggested_selling_price": 25.00,
    "profit_margin": 0.22,
    "verdict": "marginal"
  }
}
```

利润分级：
- `profitable` — 利润率 > 25%
- `marginal` — 利润率 10-25%（需关注的信号）
- `unprofitable` — 利润率 < 10%（不建议上架）

---

## Step 4: 执行全链路评估

使用 `loop` 模块串联完整度 + 利润 + 风险分析 + 上架建议：

```bash
curl -s -X POST "http://localhost:8080/api/v1/loop/evaluate/1" \
  -H "Authorization: Bearer $TOKEN"
```

返回包含：
- 完整度评分汇总
- 利润汇总
- 风险等级（low / medium / high）
- 上架建议（recommend / review / block）
- 建议的 listing task 配置

---

## Step 5: Owner 审批

评估结果出来后，进入决策流程：

```bash
# 查看待审批决策
curl -s "http://localhost:8080/api/v1/decision" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# 审批通过
curl -s -X POST "http://localhost:8080/api/v1/decision/1/approve" \
  -H "Authorization: Bearer $TOKEN"

# 或拒绝
curl -s -X POST "http://localhost:8080/api/v1/decision/1/reject" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "利润率过低，需重新定价"}'
```

**前端操作**：访问 `http://localhost:3000/decision/prelisting` → 查看决策详情 → 审批或拒绝。

---

## Step 6: 创建并执行上架任务

审批通过后，系统自动创建 ListingTask（初始状态为 `blocked`）。

```bash
# 查看上架任务
curl -s "http://localhost:8080/api/v1/listing-tasks" \
  -H "Authorization: Bearer $TOKEN"

# 发布（触发状态机转换）
curl -s -X POST "http://localhost:8080/api/v1/listing-task/1/execute" \
  -H "Authorization: Bearer $TOKEN"
```

**前端操作**：访问 `http://localhost:3000/listing-tasks` → 查看任务详情 → 执行。

---

## Step 7: 验证结果

检查上架任务的最终状态：

```bash
curl -s "http://localhost:8080/api/v1/listing-tasks/1" \
  -H "Authorization: Bearer $TOKEN"
```

状态机：
```
blocked → pending → executing → completed
                              → failed（可重试）
              ← cancelled
```

如果失败，可以重试：
```bash
curl -s -X POST "http://localhost:8080/api/v1/listing-task/1/retry-failed" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 完整流程示意图

```
手动录入/Agent采集
       ↓
  候选商品 ──→ [状态: draft]
       ↓
  完整度检查 ←── 补全缺失项
       ↓
  利润测算 ←── 调整售价/供应商
       ↓
  全链路评估
       ↓
 ┌── 决策 ──┐
 │          │
审批通过    拒绝 → 结束
 │
 ↓
ListingTask ──→ [状态: blocked]
 │              ↓
执行 ────────→ [pending → executing → completed]
 │
 ↓
商品出现在平台列表中
```

---

## 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| 完整度评分 < 60% | 缺少关键字段（图片/规格/品牌） | 补充字段后重新 check |
| 利润率为负 | 采购成本估算过低 | 检查成本配置或调高售价 |
| ListingTask 卡在 `blocked` | 未审批 | 先 `POST /decision/:id/approve` |
| Execution 返回 401 | Token 过期或未携带 | 重新 login 刷新 token |

---

## 下一步

- [Tutorial: 端到端业务流程](tutorial-first-workflow.md) — 从安装到运行的完整新手教程
- [模块目录](reference-module-catalog.md) — 全部 API 路由
- [AI & Agent 系统](reference-ai-agent-system.md) — 了解 Agent 如何自动执行此流程
