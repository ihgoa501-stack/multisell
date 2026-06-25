# 凌镜 LingMirror Demo 场景

> 最后更新：2026-06-24
> 状态：旧栈 demo 已归档；新栈 demo seed 待重建

## 当前结论

此前的 Stage 13 Demo Seed / Sandbox Scenario 基于旧 Python/FastAPI 后端和 Vue 前端：

- `backend/scripts/load_demo_data.py`
- `backend/scripts/acceptance_api.py`
- `frontend/`
- `/api/*`

这些内容现在只作为历史参考。当前活跃新栈是：

- `backend-go/`
- `frontend-next/`
- `/api/v1/*`

截至 2026-06-24，`backend-go/` 下没有等价的 demo seed 脚本，因此不能继续按旧 demo 文档执行。

## 当前可运行验证

启动新栈：

```bash
docker compose up -d db

cd backend-go
go run cmd/server/main.go

cd frontend-next
npm run dev -- --hostname 127.0.0.1 --port 3000
```

基础验证：

```bash
curl http://localhost:8080/api/health

cd backend-go
go test ./...
go vet ./...

cd frontend-next
npm test
npm run build
```

## 新栈 Demo 应覆盖的业务闭环

建议重建一个 Go + Next demo seed，覆盖以下场景：

1. 登录和 RBAC
2. 商品、SKU、库存、价格基础数据
3. 物流供应商、渠道、区域、报价规则
4. 平台和平台费用规则
5. 上架前决策
6. 刊登任务与任务详情
7. 订单和订单详情
8. 订单导入批次
9. 结算和对账
10. 财务利润报表
11. 异常工作台
12. AI chat、trace、action 审批
13. AgentOS 工作队列、熵监控、信任分

## 新栈 Demo Seed 建议

建议新增：

```text
backend-go/cmd/seed-demo/main.go
```

或：

```text
backend-go/internal/demo/
```

要求：

- 幂等，可重复执行。
- 使用当前 GORM models。
- 写入 `multisell` 开发库。
- 可选参数控制清空/追加。
- 输出 demo 用户、demo SKU、demo 订单和导入文件路径。

## 新栈 Demo 数据建议

| 实体 | 建议数量 | 说明 |
|---|---:|---|
| 用户 | 2 | admin / demo |
| 商品 | 7 | 覆盖电子、服装、家居、美妆、食品、运动 |
| SKU | 14 | 每商品 2 个 SKU |
| 库存 | 14 | 覆盖正常、低库存、锁定库存 |
| 平台 | 5 | Ozon、Shopee、Wildberries、AliExpress、Temu |
| 物流渠道 | 4 | 覆盖 RU / SEA 示例 |
| 平台费用规则 | 6 | 支持决策测算 |
| 订单 | 6 | 覆盖 paid、shipped、completed、cancelled |
| 结算行 | 10+ | 覆盖 matched / unmatched / mismatch |
| 异常 | 5+ | 覆盖库存、利润、物流、结算 |
| AI actions | 5+ | 覆盖 suggested / approved / executed / rejected |

## 历史 demo 数据

历史 CSV 文件仍在：

- `docs/demo-data/order_import_demo.csv`
- `docs/demo-data/shipping_bill_demo.csv`
- `docs/demo-data/platform_settlement_demo.csv`

它们可以作为新 Go demo seed 的输入样例，但旧 API 路径需要迁移到 `/api/v1/*`。

## 待办

1. 新增 Go demo seed。
2. 更新 demo CSV 字段以匹配 Go models。
3. 新增端到端 demo acceptance 脚本。
4. 将 `docs/DEMO_ACCEPTANCE_REPORT.md` 更新为新栈验收结果。
