# Tutorial: 从安装到运行第一个业务闭环

> 你将搭建完整的凌镜开发环境，创建一个候选商品，执行完整度-利润-评估-审批-上架的完整流程。
> 预计时间：30 分钟。

---

## 你会学到什么

- 启动凌镜的本地开发环境
- 创建和管理候选商品
- 检查商品完整度
- 测算跨平台利润
- 执行全链路评估并审批
- 创建上架任务

---

## 你需要什么

- **操作系统**：macOS / Linux 均可
- **Docker Desktop**（或 Docker + docker-compose）：PostgreSQL 依赖
- **Go 1.25+**：后端运行
- **Node.js 22+**：前端运行
- **curl**（或任何 HTTP 客户端）：测试 API
- **Git**：克隆项目

---

## Step 1: 启动数据库

凌镜依赖 PostgreSQL。最简单的方式是用 Docker：

```bash
# 在项目根目录
docker compose up -d db
```

验证：
```bash
docker compose ps
# 应看到 db 服务的状态为 Up
```

默认连接参数（定义在 `docker-compose.yml`）：
- Host: `localhost:5432`
- Database: `multisell`
- User: `multisell`
- Password: `multisell`

---

## Step 2: 启动后端

```bash
cd backend-go
go run cmd/server/main.go
```

第一次运行会：
1. 自动执行数据库迁移（创建所有表）
2. 注入 Mock 种子数据（候选商品 + 订单 + 结算数据）
3. 启动 HTTP 服务器在 `:8080`
4. 启动 EventBus、Scheduler、Agent 系统

验证后端运行：
```bash
curl http://localhost:8080/api/health
# 返回: {"status":"ok"} 或 {"code":0,"data":{"status":"ok"}}
```

**你可能看到了什么：**
后端启动时会输出以下日志：
```
Starting LingMirror server...
Running migrations...
Seeding mock data...
Server listening on :8080
```

---

## Step 3: 启动前端（可选）

如果需要操作 UI 而不只是 API：

```bash
cd frontend-next
npm install     # 首次运行需要
npm run dev -- --hostname 127.0.0.1 --port 3000
```

访问 `http://localhost:3000` → 使用默认账号登录：
- 用户名：`admin`
- 密码：`admin123`

---

## Step 4: 创建你的第一个候选商品

**API 方式：**

```bash
# 登录
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])" 2>/dev/null)

echo "Token: ${TOKEN:0:20}..."

# 创建候选商品
curl -s -X POST http://localhost:8080/api/v1/candidates \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "无线蓝牙耳机 Pro-200",
    "description": "真无线立体声蓝牙耳机，充电仓续航24小时，IPX5防水",
    "purchase_price": 65.00,
    "purchase_currency": "CNY",
    "package_weight_kg": 0.08,
    "package_length_cm": 6,
    "package_width_cm": 6,
    "package_height_cm": 3,
    "origin_country": "CN",
    "source_platform": "1688"
  }'
```

成功输出：
```json
{"code":0,"data":{"id":2,"title":"无线蓝牙耳机 Pro-200","status":"draft","...":"..."}}
```

**前端方式（如果启动了前端）：**
1. 打开 `http://localhost:3000/candidates`
2. 点击 "新建候选商品"
3. 填写商品信息（标题、描述、采购价、重量尺寸）
4. 保存

---

## Step 5: 检查商品完整度

刚才创建的商品只填了基础信息。检查一下缺什么：

```bash
curl -s -X POST "http://localhost:8080/api/v1/completeness/check/2" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

结果示例：
```json
{
  "code": 0,
  "data": {
    "product_id": 2,
    "overall_score": 58.3,
    "level": "incomplete",
    "missing_items": ["主图", "规格", "品牌", "分类", "供应商", "关税编码"]
  }
}
```

**现在图片还不具备，但为了方便演示，我们这样操作：** 先补充必要的文本字段。

```bash
# 更新分类（假设已有分类 ID 1）
curl -s -X PUT "http://localhost:8080/api/v1/candidates/2" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "category_id": 1,
    "brand": "BlastTech"
  }'
```

---

## Step 6: 测算利润

完整度满足最低要求后，运行利润测算：

```bash
curl -s "http://localhost:8080/api/v1/profit/summary/2" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

你会看到：
```json
{
  "code": 0,
  "data": {
    "purchase_price_cny": 65.00,
    "estimated_logistics": 12.50,
    "estimated_platform_fee": 6.00,
    "estimated_tariff": 2.50,
    "total_landed_cost": 86.00,
    "suggested_selling_price": 18.00,
    "profit_margin": 0.31,
    "verdict": "profitable"
  }
}
```

利润分级：
- `profitable` (利润率 > 25%) ✅ — 可以继续
- `marginal` (10-25%) ⚠️ — 需关注
- `unprofitable` (< 10%) ❌ — 不建议上架

**如果利润不可观怎么办？** 调整采购价、物流选择、或者目标售价。修改字段后重新 check。

---

## Step 7: 运行全链路评估

这是串联所有环节的最终评估：

```bash
curl -s -X POST "http://localhost:8080/api/v1/loop/evaluate/2" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

输出包含：
- 完整度评分 → `completeness_score`
- 利润汇总 → `profit_summary`
- 风险等级 → `risk_level` (low/medium/high)
- 上架建议 → `recommendation` (recommend/review/block)
- 建议的 listing 配置

**如果建议是 "block"**，说明某个关键门槛未到（比如完整度太低、利润为负）。检查反馈并修正。

---

## Step 8: 审批决策

评估完成后会生成一条决策记录。查看并审批：

```bash
# 查看待审批决策列表
curl -s "http://localhost:8080/api/v1/decision" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "
import sys,json
data = json.load(sys.stdin)
for d in data.get('data', []):
    print(f\"决策 #{d['id']}: {d.get('title','')} — 状态: {d.get('status','')}\")
"

# 审批通过（假设决策 ID 为 1）
curl -s -X POST "http://localhost:8080/api/v1/decision/1/approve" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Step 9: 发布上架任务

审批通过后，发布任务会被创建（状态 `blocked`）。执行它：

```bash
# 查看上架任务
curl -s "http://localhost:8080/api/v1/listing-tasks" \
  -H "Authorization: Bearer $TOKEN"

# 执行任务
curl -s -X POST "http://localhost:8080/api/v1/listing-task/1/execute" \
  -H "Authorization: Bearer $TOKEN"
```

状态转换：`blocked → pending → executing → completed`

如果失败，可以重试：
```bash
curl -s -X POST "http://localhost:8080/api/v1/listing-task/1/retry-failed" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Step 10: 查看结果

在 Dashboard 上查看发布结果：

```bash
curl -s "http://localhost:8080/api/v1/dashboard/overview" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

或打开 `http://localhost:3000/dashboard` 查看前端可视化面板。

---

## 你完成了什么

你刚刚完成了凌镜的第一个完整业务闭环：

```
✅ Docker 启动 PostgreSQL
✅ 后端启动 + 自动 migrate
✅ 创建候选商品
✅ 检查 12 维完整度
✅ 测算跨平台利润
✅ 运行全链路评估
✅ 审批决策
✅ 执行上架任务
```

---

## 现在可以试试

- **用 Agent 采集候选商品**：启动 Chrome 扩展（见 `chrome-extension/`），打开 1688 页面，通过 `A12 CollectionAgent` 自动采集
- **配置真实平台集成**：在 Settings 中添加 Ozon / Shopee API Key，然后尝试 dry-run 发布
- **检查 Dashboard**：查看系统概览、待审批 Action、Agent 运行状态
- **尝试 AgentOS 控制台**：打开 `http://localhost:3000/agentos` 查看分析队列

---

## 故障排除

| 问题 | 可能原因 | 解决 |
|------|----------|------|
| `docker compose up -d db` 失败 | Docker 未运行 | 启动 Docker Desktop |
| 后端启动后立即退出 | 端口占用或配置错误 | 检查 `:8080` 端口、`.env` 文件 |
| `401 Unauthorized` | Token 无效或过期 | 重新登录获取新 Token |
| 完整度一直 < 60% | 图片/规格等关键字段缺失 | 先创建带 image_url 的完整商品 |
| 利润始终为负 | Mock 数据中成本 > 售价 | 调高售价或联系 Admin 修改成本参数 |
| ListingTask 无法执行 | 未审批或状态不匹配 | 先审批决策，确认任务状态正确 |

---

## 相关文档

- [How-to: 执行业务闭环](howto-first-business-loop.md) — API 参考版
- [入门教程](tutorial-getting-started.md) — 环境搭建基础版
- [AI & Agent 系统参考](reference-ai-agent-system.md) — Agent 执行原理
- [Why: 两个业务闭环](explanation-business-loops.md) — 设计背后的思考
