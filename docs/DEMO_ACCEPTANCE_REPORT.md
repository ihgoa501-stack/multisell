# Demo Acceptance Report

## Meta

| Field | Value |
|---|---|
| Date | 2026-06-17 |
| Branch | `codex/demo-acceptance` |
| Acceptance reference | Current branch HEAD after acceptance fixes |
| Backend | `DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test /Users/lc/multisell/backend/.venv/bin/python -m uvicorn app.main:app --reload --port 8000` |
| Frontend | `cd frontend && npm run dev` |

## Demo Seed Result

```bash
cd backend && DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  /Users/lc/multisell/backend/.venv/bin/python scripts/load_demo_data.py
```

**Result:** ✅ Pass
- products created/updated: 7/0
- skus created/updated: 14/0
- inventory seeded: 14
- shipping rules seeded: 4
- platform fee rules seeded: 6
- demo csv paths exist and accessible
- Second execution: idempotent

## Page-by-Page Acceptance

### 1. Login
**Result:** ✅ PASS
- demo/demo123 可登录
- 修复：seed 脚本中 demo 用户 role 改为 admin（原为 user，无权限）

### 2. Products / SKU / Inventory
**Result:** ✅ PASS
- API `GET /api/products` 返回 7 个 Demo 商品
- SKU 通过 `GET /api/products/{product_id}/skus` 可见（没有独立的 `/api/skus` 全局列表）

### 3. Shipping Calculator
**Result:** ✅ PASS（API 验证）
- `POST /api/shipping/calculate` 支持 sku 和 manual 两种模式
- 需传递 `mode=sku&sku_id=1` 或 `mode=manual&package={...}`
- 存在已 seed 的报价规则（CDEK Economy / Standard, EMS, Shopee Xpress Standard）

### 4. Pre-listing Decision
**Result:** ✅ PASS（API 验证）
- `POST /api/decisions/prelisting` 接收 `sku_id`, `target_sale_price`, `destination_country` 等参数

### 5. CSV Order Import
**Result:** ✅ PASS
- API 工作正常：可见多 SKU 合并、创建订单数
- 上传格式：`adapter_code` 作为 Form 字段，`file` 作为 File 字段
- 修复：Demo CSV 地址字段已加引号，避免逗号导致 `shipping_fee/tracking_number/paid_at` 错位。
- 最新结果：7 行记录，创建 6 个订单，失败 0 行。

### 6. Process Chain
**Result:** ✅ PASS
- `POST /api/order-imports/{batch_id}/process-chain` 成功重建账本并生成异常

### 7. Shipping Bill Import & Reconcile
**Result:** ⚠️ WARN
- 导入成功（5 行）
- 对账结果：matched=0, mismatch=3, unmatched=2
- 说明：对账接口可运行并能分类异常；当前 demo 未创建运费快照，所以不会产生 matched，适合作为异常工作台演示输入。

### 8. Settlement Import
**Result:** ✅ PASS
- 导入成功（18 行）
- platform_order_no 可匹配到导入订单

### 9. Exception Workbench
**Result:** ✅ PASS
- `POST /api/exceptions/generate` 生成 23 条异常
- `GET /api/exceptions` 返回异常列表

### 10. Profit Dashboard
**Result:** ⚠️ PARTIAL
- Profit Summary 返回 revenue=0
- **原因：** 已通过 process-chain 重建账本，但重建过程需要订单中的 product_cost 等字段才有利润数据。Demo CSV 没有 product_cost 字段
- Negative Profit Orders: 0
- 需要补充：订单导入后需要设置订单的 product_cost 才能正确计算利润

### 11. Frontend Build
**Result:** ✅ PASS
- `npm run build` 零警告通过

## Issues Found

### Fixed

| # | Issue | Fix | Status |
|---|---|---|---|
| 1 | Demo user has no permissions (role=user, empty permissions) | Updated seed to set role="admin" and update existing users | ✅ |
| 2 | `order_import_batch` / `order_import_item` tables not created by seed script | Added `from app.order_import import models` to seed imports | ✅ |
| 3 | ensure_demo_user only creates, doesn't update existing | Updated to update role/display_name on existing | ✅ |

### Known / Not Fixed (documented)

| # | Issue | Impact | Workaround |
|---|---|---|---|
| 4 | Demo 订单没有运费快照 | Shipping bill reconcile 无 matched，主要产生 mismatch/unmatched | Stage 14.2 增加 demo 快照或导入后自动快照 |
| 5 | Profit dashboard revenue=0 | Process chain → ledger rebuild doesn't have product_cost data from CSV | Add `product_cost` field to order_import_demo.csv |
| 6 | No independent `/api/skus` list endpoint (only `/api/products/{product_id}/skus`) | Demo scripts must look up SKU through product detail | Use product-scoped endpoint |
| 7 | Negative profit report is empty | Demo cannot yet show negative-profit BI card | Add product_cost and settlement/fee data to imported orders |

## Next Steps

1. **Immediate (Stage 14.1):** Add `product_cost` field to order import CSV so profit calculation works
2. **Immediate (Stage 14.2):** Add demo order shipping snapshots so shipping bills can produce matched and amount_mismatch examples
3. **Short-term:** Add a standalone SKU list endpoint
4. **Demo docs:** Update DEMO_SCENARIO.md with accurate API paths and expected results
5. **Long-term:** Add automated E2E test that runs the full acceptance script

## Verification

```bash
# Backend tests
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  /Users/lc/multisell/backend/.venv/bin/python -m pytest tests/test_demo_seed.py -q
# → 17 passed

# Frontend build
cd frontend && npm run build
# → ✓ 4292 modules transformed, ✓ built in 2.10s

# API acceptance
cd backend && /Users/lc/multisell/backend/.venv/bin/python scripts/acceptance_api.py
# → PASS: 12  FAIL: 0  WARN: 3  TOTAL: 15
```
