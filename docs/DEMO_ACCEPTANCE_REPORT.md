# Demo Acceptance Report

> 最后更新：2026-06-24
> 状态：历史验收报告，旧栈结果已归档；新栈 demo acceptance 待重建

## 当前结论

2026-06-17 的 demo acceptance 基于旧 Python/FastAPI + Vue 栈：

- 后端：`backend/`
- 前端：`frontend/`
- API：`/api/*`
- Demo seed：`backend/scripts/load_demo_data.py`
- Acceptance script：`backend/scripts/acceptance_api.py`

当前项目已经完成全站迁移，活跃栈变为：

- 后端：`backend-go/`
- 前端：`frontend-next/`
- API：`/api/v1/*`

因此旧验收结果只作为历史参考，不再代表当前新栈 demo readiness。

## 当前新栈验证状态

2026-06-24：

| 检查 | 结果 |
|---|---:|
| `cd backend-go && go test ./...` | 通过 |
| `cd backend-go && go vet ./...` | 通过 |
| `cd frontend-next && npm test` | 通过 |
| `cd frontend-next && npm run build` | 通过 |
| `cd frontend-next && npm run lint` | 失败 |

## 新栈 Demo Acceptance 待办

1. 新增 Go demo seed。
2. 新增新栈 acceptance script，使用 `/api/v1/*`。
3. 覆盖登录、商品、SKU、库存、物流报价、决策、刊登任务、订单、结算、财务、异常、AI action。
4. 补前端 E2E 或 smoke test。
5. 重新生成本报告。

## 历史结果摘要

旧栈验收曾覆盖：

- demo/demo123 登录
- 商品 / SKU / 库存
- 运费计算器
- 上架前决策
- CSV 订单导入
- 经营链路处理
- 运费账单导入与对账
- 平台结算导入
- 异常工作台
- 利润看板
- 前端 build

旧栈已知限制：

- Demo 订单没有完整运费快照
- Profit dashboard revenue 依赖 CSV 中的成本字段
- 部分 API 路径是旧 `/api/*`，需要迁移到 `/api/v1/*`

详见 `docs/DEMO_SCENARIO.md` 的新栈重建建议。
