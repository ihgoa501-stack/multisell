# Order Import Smoke Checklist

> 最后更新：2026-06-24
> 状态：旧 Python smoke 已归档；Go 新栈 smoke 待重建

## 当前结论

旧检查链路基于：

- `backend/`
- `/api/order-imports/*`
- `backend/tests/test_order_import_csv_adapter.py`
- `backend/tests/test_order_import_operational_chain.py`

当前活跃后端是 `backend-go/`，业务 API 前缀是 `/api/v1`。因此旧 checklist 不能直接执行。

## 当前 Go 新栈入口

相关模块：

- `backend-go/internal/domain/orderimport/`
- `backend-go/internal/domain/order/`
- `backend-go/internal/domain/finance/`
- `backend-go/internal/domain/exceptions/`

路由挂载：

- `backend-go/internal/httpx/router.go`
- 统一前缀：`/api/v1`

## 新栈 Smoke 应覆盖

目标链路：

```text
订单导入 -> 订单创建 -> 库存/财务/异常链路 -> 前端可见处理结果
```

最低验证项：

1. 登录并拿到 token。
2. 上传订单导入文件。
3. 查询导入批次。
4. 查询导入行。
5. 执行或触发处理链路。
6. 验证订单、财务账本、异常工作台结果。
7. 前端 `/order-import` 页面能展示批次和状态。

## 待补工作

1. 确认 Go `orderimport` 当前支持的文件格式和字段。
2. 补 Go 后端 focused tests。
3. 补 `/api/v1/*` curl smoke。
4. 补 `frontend-next` 页面 smoke / e2e。
5. 更新 `docs/demo-data/*.csv` 与 Go model 字段对齐。

## 当前可运行验证

在 smoke 重建前，先运行基础检查：

```bash
cd backend-go
go test ./...
go vet ./...

cd frontend-next
npm test
npm run build
```

## 历史 CSV

历史样例仍可作为字段设计参考：

- `docs/demo-data/order_import_demo.csv`

但旧接口 `/api/order-imports/*` 需要迁移为 Go 当前 `/api/v1/*` 路由后才能用于自动验收。
