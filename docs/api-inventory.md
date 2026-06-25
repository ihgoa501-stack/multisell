# API Endpoint Inventory

> Generated 2026-06-25. Covers `backend-go/` (Go/Gin) HTTP endpoints under `/api/v1`.

## Legend

| Column | Meaning |
|--------|---------|
| Status | ✅ handler+service exist, 🟡 handler exists but stub/unproven, ❌ not implemented, 🔧 frontend-only (no backend route) |
| Frontend Ref | `file.tsx` or `file.ts` — exact path under `frontend-next/src/app/` or `src/lib/` |

---

## 1. Auth & Public

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/health` | ✅ | inline | — |
| GET | `/api/v1/health` | ✅ | inline | — |
| GET | `/metrics` | ✅ | middleware.MetricsHandler | — |
| POST | `/api/v1/auth/login` | ✅ | handler.Login | `(auth)/login/page.tsx` |
| POST | `/api/v1/auth/register` | ✅ | handler.Register | — |
| POST | `/api/v1/auth/refresh` | ✅ | handler.Refresh | `lib/api-client.ts` |
| GET | `/api/v1/auth/me` | ✅ | handler.CurrentUser | — |
| WS | `/ws` | ✅ | realtime.Handler.ServeWS | — |

---

## 2. Dashboard (`/api/v1/dashboard`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/dashboard/overview` | ✅ | handler.Overview | `(main)/dashboard/page.tsx` |
| GET | `/api/v1/dashboard/orders` | ✅ | handler.Orders | — |
| GET | `/api/v1/dashboard/inventory` | ✅ | handler.Inventory | — |
| GET | `/api/v1/dashboard/exceptions` | ✅ | handler.Exceptions | — |

**Return types** (from `domain/dashboard/model.go`):
- `OverviewResponse` — `{ total_sales, total_orders, active_listings, pending_tasks, agent_status, ... }`
- `OrdersResponse` — `{ pending, processing, shipped, completed, cancelled, ... }`
- `InventoryResponse` — `{ total_sku, low_stock, out_of_stock, overstocked, ... }`
- `ExceptionsResponse` — `{ total, critical, warning, info, recent, ... }`

---

## 3. Products & SKU (`/api/v1/products`, `/api/v1/skus`, `/api/v1/spec-values`)

**Note:** Products and SKUs share the `internal/domain/sku/` package (one module covers both).

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/products` | ✅ | h.ListProducts | `(main)/products/page.tsx` |
| GET | `/api/v1/products/:id` | ✅ | h.GetProduct | `(main)/products/[id]/page.tsx` |
| POST | `/api/v1/products` | ✅ | h.CreateProduct | `(main)/products/create/page.tsx`, `products/page.tsx` |
| PUT | `/api/v1/products/:id` | ✅ | h.UpdateProduct | `products/page.tsx` |
| DELETE | `/api/v1/products/:id` | ✅ | h.DeleteProduct | `products/page.tsx` |
| GET | `/api/v1/products/:id/specs` | ✅ | h.ListSpecs | — |
| POST | `/api/v1/products/:id/specs` | ✅ | h.CreateSpec | — |
| PUT | `/api/v1/products/:id/specs/:spec_id` | ✅ | h.UpdateSpec | — |
| DELETE | `/api/v1/products/:id/specs/:spec_id` | ✅ | h.DeleteSpec | — |
| POST | `/api/v1/products/:id/specs/:spec_id/values` | ✅ | h.CreateSpecValue | — |
| GET | `/api/v1/products/:id/skus` | ✅ | h.ListSkusByProduct | — |
| GET | `/api/v1/skus` | ✅ | h.ListSkus | — |
| GET | `/api/v1/skus/:id` | ✅ | h.GetSku | — |
| POST | `/api/v1/skus` | ✅ | h.CreateSku | — |
| PUT | `/api/v1/skus/:id` | ✅ | h.UpdateSku | — |
| DELETE | `/api/v1/skus/:id` | ✅ | h.DeleteSku | — |
| PUT | `/api/v1/spec-values/:id` | ✅ | h.UpdateSpecValue | — |
| DELETE | `/api/v1/spec-values/:id` | ✅ | h.DeleteSpecValue | — |

**Return types** (from `domain/sku/model.go`):
- `ProductRecord` / `ProductDetail` — product with specs, variants, images
- `SkuResponse` — SKU-level product variant

---

## 4. Pricing (`/api/v1/prices`, `/api/v1/skus/:id/prices`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/prices` | ✅ | h.ListPrices | — |
| GET | `/api/v1/prices/:id` | ✅ | h.GetPrice | — |
| POST | `/api/v1/prices` | ✅ | h.SetPrice | — |
| PUT | `/api/v1/prices/:id` | ✅ | h.UpdatePrice | — |
| DELETE | `/api/v1/prices/:id` | ✅ | h.DeletePrice | — |
| GET | `/api/v1/skus/:id/prices` | ✅ | h.ListPricesBySKU | — |
| GET | `/api/v1/skus/:id/current-price` | ✅ | h.GetCurrentPrice | — |
| GET | `/api/v1/skus/:id/price-history` | ✅ | h.PriceHistory | — |

---

## 5. Inventory (`/api/v1/inventory`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/inventory` | ✅ | h.List | — |
| GET | `/api/v1/inventory/:id` | ✅ | h.Get | — |
| PUT | `/api/v1/inventory/:id` | ✅ | h.Update | — |
| POST | `/api/v1/inventory/:id/lock` | ✅ | h.Lock | — |
| POST | `/api/v1/inventory/:id/unlock` | ✅ | h.Unlock | — |
| GET | `/api/v1/inventory/logs` | ✅ | h.ListLogs | — |
| GET | `/api/v1/inventory/warehouses` | ✅ | h.ListWarehouses | — |
| POST | `/api/v1/inventory/warehouses` | ✅ | h.CreateWarehouse | — |
| GET | `/api/v1/inventory/warehouses/:id` | ✅ | h.GetWarehouse | — |
| PUT | `/api/v1/inventory/warehouses/:id` | ✅ | h.UpdateWarehouse | — |
| DELETE | `/api/v1/inventory/warehouses/:id` | ✅ | h.DeleteWarehouse | — |
| GET | `/api/v1/inventory/sku/:sku_id/warehouses` | ✅ | h.ListInventoryBySku | — |

**Return types** (from `domain/inventory/model.go`):
- `InventoryRecord` — `{ id, sku_id, warehouse_id, quantity, reserved, available }`
- `Warehouse` — `{ id, name, code, address, status }`

---

## 6. Orders (`/api/v1/order`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/order` | ✅ | h.List | — |
| GET | `/api/v1/order/summary` | ✅ | h.Summary | — |
| GET | `/api/v1/order/:id` | ✅ | h.Get | `(main)/orders/[id]/page.tsx` |
| POST | `/api/v1/order` | ✅ | h.Create | — |
| PUT | `/api/v1/order/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/order/:id` | ✅ | h.Delete | — |
| POST | `/api/v1/order/:id/status` | ✅ | h.UpdateStatus | — |

**Return types** (from `domain/order/model.go`):
- `OrderDetailResponse` — `{ id, platform_order_id, status, items, shipping, payment, ... }`

---

## 7. Listings (`/api/v1/listings`, `/api/v1/listing`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/listings` | ✅ | h.List | — |
| GET | `/api/v1/listings/:id` | ✅ | h.Get | — |
| POST | `/api/v1/listings` | ✅ | h.Create | `(main)/listings/create/page.tsx` |
| PUT | `/api/v1/listings/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/listings/:id` | ✅ | h.Delete | — |
| POST | `/api/v1/listings/:id/publish` | ✅ | h.Publish | — |
| POST | `/api/v1/listings/:id/sync` | ✅ | h.Sync | — |
| POST | `/api/v1/listing/products/:product_id/publish/:platform_id` | ✅ | h.PublishProduct | — |
| GET | `/api/v1/listing/products/:product_id/listings` | ✅ | h.ListByProduct | — |
| POST | `/api/v1/listing/listing-tasks/from-decisions` | ✅ | h.CreateTasksFromDecisions | — |
| POST | `/api/v1/listing/listing-tasks/:task_id/recheck` | ✅ | h.RecheckTask | — |
| POST | `/api/v1/listing/listing-tasks/:task_id/cancel` | ✅ | h.CancelTask | — |
| POST | `/api/v1/listing/listing-tasks/:task_id/publish` | ✅ | h.PublishTask | — |
| POST | `/api/v1/listing` | ❌ | **No handler** | `(main)/listings/create/page.tsx` calls `POST /v1/listing` but only `POST /v1/listings` exists |

**⚠ Gap:** Frontend `listings/create/page.tsx` calls `POST /v1/listing` (singular), but the backend only registers `POST /v1/listings` (plural). Needs alignment: either change frontend to `/v1/listings` or add a route alias.

**Return types** (from `domain/listing/model.go`):
- `ListingRecord` — `{ id, product_id, platform_id, store_id, status, listing_url, ... }`
- `PublishResult` — `{ task_id, status, errors[] }`

---

## 8. Listing Tasks (`/api/v1/listing-tasks`, `/api/v1/listing-task`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/listing-tasks` | ✅ | h.List | — |
| GET | `/api/v1/listing-tasks/:id` | ✅ | h.Get | `listing-tasks/[id]/page.tsx` |
| POST | `/api/v1/listing-tasks` | ✅ | h.Create | — |
| PUT | `/api/v1/listing-tasks/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/listing-tasks/:id` | ✅ | h.Delete | — |
| GET | `/api/v1/listing-tasks/:id/items` | ✅ | h.ListItems | — |
| POST | `/api/v1/listing-tasks/:id/items` | ✅ | h.CreateItem | — |
| PUT | `/api/v1/listing-tasks/:id/items/:item_id` | ✅ | h.UpdateItem | — |
| DELETE | `/api/v1/listing-tasks/:id/items/:item_id` | ✅ | h.DeleteItem | — |
| POST | `/api/v1/listing-task/:task_id/execute` | ✅ | h.Execute | `listing-tasks/[id]/page.tsx` |
| POST | `/api/v1/listing-task/:task_id/retry-failed` | ✅ | h.RetryFailed | `listing-tasks/[id]/page.tsx` |
| POST | `/api/v1/listing-task/:task_id/items/:item_id/retry` | ✅ | h.RetryItem | — |
| GET | `/api/v1/listing-task/stats` | ❌ | **Handler not found** | `listing-tasks/workbench/page.tsx` |
| POST | `/api/v1/listing-task/retry-all` | ❌ | **Handler not found** | `listing-tasks/workbench/page.tsx` |

**⚠ Gaps:** Frontend workbench calls `GET /v1/listing-task/stats` and `POST /v1/listing-task/retry-all`, neither exists in `listingtask/handler.go` or `routes.go`. These are Phase 2 additions.

**Return types** (from `domain/listingtask/model.go`):
- `TaskResponse` — `{ id, status, platform_id, product_id, total_items, completed, failed }`
- `TaskItemResponse` — `{ id, task_id, sku_id, platform_id, status, error }`

---

## 9. Supplier (`/api/v1/suppliers`, `/api/v1/product-suppliers`, `/api/v1/products/:id/supplier-comparison`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/suppliers` | ✅ | h.List | — |
| GET | `/api/v1/suppliers/:id` | ✅ | h.Get | — |
| POST | `/api/v1/suppliers` | ✅ | h.Create | — |
| PUT | `/api/v1/suppliers/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/suppliers/:id` | ✅ | h.Delete | — |
| GET | `/api/v1/product-suppliers` | ✅ | h.ListProductSuppliers | — |
| POST | `/api/v1/product-suppliers` | ✅ | h.CreateProductSupplier | — |
| PUT | `/api/v1/product-suppliers/:id` | ✅ | h.UpdateProductSupplier | — |
| DELETE | `/api/v1/product-suppliers/:id` | ✅ | h.DeleteProductSupplier | — |
| GET | `/api/v1/products/:id/supplier-comparison` | ✅ | h.GetSupplierComparison | `products/[id]/suppliers/page.tsx` |

---

## 10. Platforms & Stores (`/api/v1/platforms`, `/api/v1/stores`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/platforms` | ✅ | h.ListPlatforms | — |
| GET | `/api/v1/platforms/:id` | ✅ | h.GetPlatform | — |
| POST | `/api/v1/platforms` | ✅ | h.CreatePlatform | — |
| PUT | `/api/v1/platforms/:id` | ✅ | h.UpdatePlatform | — |
| DELETE | `/api/v1/platforms/:id` | ✅ | h.DeletePlatform | — |
| GET | `/api/v1/stores` | ✅ | h.ListStores | — |
| GET | `/api/v1/stores/:id` | ✅ | h.GetStore | — |
| POST | `/api/v1/stores` | ✅ | h.CreateStore | — |
| PUT | `/api/v1/stores/:id` | ✅ | h.UpdateStore | — |
| DELETE | `/api/v1/stores/:id` | ✅ | h.DeleteStore | — |

---

## 11. Platform Integrations (`/api/v1/platform-integrations`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/platform-integrations` | ✅ | h.List | — |
| POST | `/api/v1/platform-integrations` | ✅ | h.Create | — |
| GET | `/api/v1/platform-integrations/:id` | ✅ | h.Get | — |
| PUT | `/api/v1/platform-integrations/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/platform-integrations/:id` | ✅ | h.Delete | — |
| POST | `/api/v1/platform-integrations/:id/test` | ✅ | h.TestConnection | — |
| POST | `/api/v1/platform-integrations/:id/sync` | ✅ | h.Sync | — |
| GET | `/api/v1/platform-integrations/:id/categories` | ✅ | h.ListCategories | — |
| POST | `/api/v1/platform-integrations/:id/categories` | ✅ | h.CreateCategory | — |
| GET | `/api/v1/platform-integrations/:id/attributes` | ✅ | h.ListAttributes | — |
| POST | `/api/v1/platform-integrations/:id/attributes` | ✅ | h.CreateAttribute | — |

---

## 12. Shipping (`/api/v1/shipping`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/shipping/quote` | ✅ | h.Quote | — |
| GET | `/api/v1/shipping/providers` | ✅ | h.ListProviders | — |
| GET | `/api/v1/shipping/providers/:id` | ✅ | h.GetProvider | — |
| POST | `/api/v1/shipping/providers` | ✅ | h.CreateProvider | — |
| PUT | `/api/v1/shipping/providers/:id` | ✅ | h.UpdateProvider | — |
| DELETE | `/api/v1/shipping/providers/:id` | ✅ | h.DeleteProvider | — |
| GET | `/api/v1/shipping/channels` | ✅ | h.ListChannels | — |
| GET | `/api/v1/shipping/channels/:id` | ✅ | h.GetChannel | — |
| POST | `/api/v1/shipping/channels` | ✅ | h.CreateChannel | — |
| PUT | `/api/v1/shipping/channels/:id` | ✅ | h.UpdateChannel | — |
| DELETE | `/api/v1/shipping/channels/:id` | ✅ | h.DeleteChannel | — |
| GET | `/api/v1/shipping/zones` | ✅ | h.ListZones | — |
| POST | `/api/v1/shipping/zones` | ✅ | h.CreateZone | — |
| DELETE | `/api/v1/shipping/zones/:id` | ✅ | h.DeleteZone | — |
| GET | `/api/v1/shipping/rules` | ✅ | h.ListRules | — |
| POST | `/api/v1/shipping/rules` | ✅ | h.CreateRule | — |
| DELETE | `/api/v1/shipping/rules/:id` | ✅ | h.DeleteRule | — |
| GET | `/api/v1/shipping/bill-batches` | ✅ | h.ListBillBatches | — |
| GET | `/api/v1/shipping/bill-batches/:id` | ✅ | h.GetBillBatch | — |
| POST | `/api/v1/shipping/bill-batches` | ✅ | h.CreateBillBatch | — |
| DELETE | `/api/v1/shipping/bill-batches/:id` | ✅ | h.DeleteBillBatch | — |
| GET | `/api/v1/shipping/bill-batches/:id/items` | ✅ | h.ListBillItems | — |

---

## 13. Platform Fee (`/api/v1/platform-fee`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/platform-fee/calculate` | ✅ | h.Calculate | — |
| GET | `/api/v1/platform-fee` | ✅ | h.List | — |
| GET | `/api/v1/platform-fee/:id` | ✅ | h.Get | — |
| POST | `/api/v1/platform-fee` | ✅ | h.Create | — |
| PUT | `/api/v1/platform-fee/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/platform-fee/:id` | ✅ | h.Delete | — |

---

## 14. Order Import (`/api/v1/order-import`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/order-import` | ✅ | h.List | — |
| GET | `/api/v1/order-import/summary` | ✅ | h.Summary | — |
| GET | `/api/v1/order-import/:id` | ✅ | h.Get | — |
| POST | `/api/v1/order-import` | ✅ | h.Create | — |
| PUT | `/api/v1/order-import/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/order-import/:id` | ✅ | h.Delete | — |
| POST | `/api/v1/order-import/:id/process` | ✅ | h.Process | — |
| POST | `/api/v1/order-import/:id/complete` | ✅ | h.Complete | — |

---

## 15. Settlement (`/api/v1/settlement`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/settlement/summary` | ✅ | h.Summary | — |
| GET | `/api/v1/settlement` | ✅ | h.List | — |
| GET | `/api/v1/settlement/:id` | ✅ | h.Get | `(main)/settlement/[id]/page.tsx` |
| POST | `/api/v1/settlement` | ✅ | h.Create | — |
| PUT | `/api/v1/settlement/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/settlement/:id` | ✅ | h.Delete | — |
| POST | `/api/v1/settlement/:id/reconcile` | ✅ | h.Reconcile | `(main)/settlement/[id]/page.tsx` |
| POST | `/api/v1/settlement/:id/items` | ✅ | h.AddItem | — |
| GET | `/api/v1/settlement/:id/items` | ✅ | h.ListItems | — |
| PUT | `/api/v1/settlement/items/:item_id/reconciliation` | ✅ | h.UpdateItemReconciliation | — |

**Return types** (from `domain/settlement/model.go`):
- `SettlementDetailResponse` — `{ id, store_id, period_start, period_end, total_revenue, platform_fees, net_revenue, items[] }`

---

## 16. Finance (`/api/v1/finance`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/finance/profit/calculate` | ✅ | h.CalculateProfit | — |
| POST | `/api/v1/finance/profit/batch-calculate` | ✅ | h.BatchCalculateProfit | — |
| GET | `/api/v1/finance/profit/summary` | ✅ | h.GetProfitSummary | — |
| GET | `/api/v1/finance/profit/ranking` | ✅ | h.GetSKUProfitRanking | — |
| GET | `/api/v1/finance/summary` | ✅ | h.Summary | — |
| GET | `/api/v1/finance/profit-summary` | ✅ | h.ProfitSummary | — |
| GET | `/api/v1/finance/ledger` | ✅ | h.ListLedger | — |
| POST | `/api/v1/finance/mock` | ✅ | h.Mock | — |
| GET | `/api/v1/finance/orders/:order_id/ledger` | ✅ | h.ListOrderLedger | — |
| GET | `/api/v1/finance/orders/:order_id/profit` | ✅ | h.OrderProfit | — |
| POST | `/api/v1/finance/orders/:order_id/ledger/rebuild` | ✅ | h.RebuildOrderLedger | — |
| GET | `/api/v1/finance/accounts` | ✅ | h.ListAccounts | — |
| POST | `/api/v1/finance/accounts` | ✅ | h.CreateAccount | — |
| GET | `/api/v1/finance/accounts/:id` | ✅ | h.GetAccount | — |
| PUT | `/api/v1/finance/accounts/:id` | ✅ | h.UpdateAccount | — |
| DELETE | `/api/v1/finance/accounts/:id` | ✅ | h.DeleteAccount | — |
| GET | `/api/v1/finance/transactions` | ✅ | h.ListTransactions | — |
| POST | `/api/v1/finance/transactions` | ✅ | h.CreateTransaction | — |

---

## 17. Decision (`/api/v1/decision`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/decision` | ✅ | h.List | — |
| GET | `/api/v1/decision/summary` | ✅ | h.Summary | — |
| GET | `/api/v1/decision/:id` | ✅ | h.Get | — |
| POST | `/api/v1/decision` | ✅ | h.Create | `(main)/decision/prelisting/page.tsx` |
| PUT | `/api/v1/decision/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/decision/:id` | ✅ | h.Delete | — |
| POST | `/api/v1/decision/:id/approve` | ✅ | h.Approve | `(main)/decision/prelisting/page.tsx` |
| POST | `/api/v1/decision/:id/reject` | ✅ | h.Reject | `(main)/decision/prelisting/page.tsx` |

---

## 18. AI / AgentOS (`/api/v1/ai`, `/api/v1/agents`, `/api/v1/agentos`)

### AI (`/api/v1/ai`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/ai/chat` | ✅ | h.Chat | `(main)/ai/page.tsx`, `components/copilot/CopilotPanel.tsx` |
| POST | `/api/v1/ai/run` | ✅ | h.RunAgent | `(main)/ai/page.tsx` |
| GET | `/api/v1/ai/traces` | ✅ | h.ListTraces | `(main)/ai/page.tsx`, `(main)/agents/[id]/page.tsx` |
| GET | `/api/v1/ai/actions` | ✅ | h.ListActions | `(main)/ai/page.tsx`, `(main)/actions/page.tsx`, `lib/actions-api.ts` |
| GET | `/api/v1/ai/agents` | ✅ | h.Roster | `(main)/ai/page.tsx` |
| GET | `/api/v1/ai/agents/specs` | ✅ | h.AgentSpecs | — |
| POST | `/api/v1/ai/actions` | ✅ | h.CreateAction | — |
| GET | `/api/v1/ai/traces/:trace_id` | ✅ | h.GetTrace | `agents/[id]/trace/[traceId]/page.tsx` |
| GET | `/api/v1/ai/actions/:id` | ✅ | h.GetAction | `actions/[id]/page.tsx`, `lib/actions-api.ts` |
| POST | `/api/v1/ai/actions/:id/approve` | ✅ | h.ApproveAction | `ai/page.tsx`, `agentos/page.tsx`, `actions/page.tsx`, `agents/[id]/trace/[traceId]/page.tsx` |
| POST | `/api/v1/ai/actions/:id/reject` | ✅ | h.RejectAction | Same as approve |
| POST | `/api/v1/ai/actions/:id/execute` | ✅ | h.ExecuteAction | Same as approve |
| POST | `/api/v1/ai/actions/:id/review` | ✅ | h.ReviewAction | `actions/[id]/page.tsx` |

### Agents (`/api/v1/agents`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/agents` | ✅ | h.ListAgents | `(main)/agents/page.tsx` |
| GET | `/api/v1/agents/evolution` | ✅ | h.Evolution | — |
| GET | `/api/v1/agents/entropy` | ✅ | h.Entropy | `(main)/agents/entropy/page.tsx` |
| POST | `/api/v1/agents` | ✅ | h.CreateAgent | — |
| GET | `/api/v1/agents/:id` | ✅ | h.GetAgent | `(main)/agents/[id]/page.tsx` |
| POST | `/api/v1/agents/:id/actions` | ✅ | h.ExecuteAction | `agents/page.tsx` |

### Agent Rules (`/api/v1/agents/rules`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/agents/rules` | ✅ | h.ListRules | — |
| GET | `/api/v1/agents/rules/:id` | ✅ | h.GetRule | — |
| POST | `/api/v1/agents/rules` | ✅ | h.CreateRule | — |
| PUT | `/api/v1/agents/rules/:id` | ✅ | h.UpdateRule | — |
| DELETE | `/api/v1/agents/rules/:id` | ✅ | h.DeleteRule | — |
| POST | `/api/v1/agents/rules/apply` | ✅ | h.ApplyRules | — |

### AgentOS (`/api/v1/agentos`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/agentos` | ✅ | h.Overview | `(main)/agentos/page.tsx` |
| GET | `/api/v1/agentos/status` | ✅ | h.Status | — |
| GET | `/api/v1/agentos/work-items` | ✅ | h.WorkItems | `(main)/agentos/page.tsx`, `agentos/work-items/page.tsx` |
| GET | `/api/v1/agentos/autonomy` | ✅ | h.Autonomy | `(main)/agentos/page.tsx` |

---

## 19. Trust Scores (`/api/v1/trust-scores`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/trust-scores` | ✅ | h.List | — |
| GET | `/api/v1/trust-scores/:agent_id` | ✅ | h.GetByAgent | — |
| POST | `/api/v1/trust-scores/recalculate` | ✅ | h.Recalculate | `(main)/agents/trust/page.tsx` |
| POST | `/api/v1/trust-scores/eligible` | ✅ | h.Eligible | — |
| PUT | `/api/v1/trust-scores/:agent_id/level` | ✅ | h.UpdateLevel | — |
| POST | `/api/v1/trust-scores/auto-upgrade` | ✅ | h.AutoUpgrade | `(main)/agents/trust/page.tsx` |
| GET | `/api/v1/trust-scores/summary` | ✅ | h.Summary | `(main)/agents/trust/page.tsx` |

---

## 20. Evolution & Entropy

### Evolution (`/api/v1/evolution`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/evolution/nudges` | ✅ | h.ListNudges | — |
| POST | `/api/v1/evolution/nudges/evaluate` | ✅ | h.EvaluateNudges | `(main)/agents/evolution/page.tsx` |
| POST | `/api/v1/evolution/nudges/:id/accept` | ✅ | h.AcceptNudge | — |
| POST | `/api/v1/evolution/nudges/:id/dismiss` | ✅ | h.DismissNudge | — |

### Entropy (`/api/v1/entropy`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/entropy` | ✅ | h.GetSummary | — |
| POST | `/api/v1/entropy/defense` | ✅ | h.RunDefenses | — |
| GET | `/api/v1/entropy/health` | ✅ | h.GetHealthScores | — |
| GET | `/api/v1/entropy/spc` | ✅ | h.GetSpcStatus | — |
| GET | `/api/v1/entropy/changelog` | ✅ | h.GetChangeLog | — |

---

## 21. RBAC (`/api/v1/rbac`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/rbac/roles` | ✅ | h.ListRoles | — |
| POST | `/api/v1/rbac/roles` | ✅ | h.CreateRole | — |
| GET | `/api/v1/rbac/roles/:id` | ✅ | h.GetRole | — |
| PUT | `/api/v1/rbac/roles/:id` | ✅ | h.UpdateRole | — |
| DELETE | `/api/v1/rbac/roles/:id` | ✅ | h.DeleteRole | — |
| GET | `/api/v1/rbac/roles/:id/permissions` | ✅ | h.GetRolePermissions | — |
| POST | `/api/v1/rbac/roles/:id/permissions` | ✅ | h.AssignRolePermissions | — |
| GET | `/api/v1/rbac/permissions` | ✅ | h.ListPermissions | — |
| POST | `/api/v1/rbac/permissions` | ✅ | h.CreatePermission | — |
| GET | `/api/v1/rbac/permissions/:id` | ✅ | h.GetPermission | — |
| PUT | `/api/v1/rbac/permissions/:id` | ✅ | h.UpdatePermission | — |
| DELETE | `/api/v1/rbac/permissions/:id` | ✅ | h.DeletePermission | — |
| GET | `/api/v1/rbac/current/permissions` | ✅ | h.GetCurrentUserPermissions | `stores/permission-store.ts` |
| GET | `/api/v1/rbac/users/:id/roles` | ✅ | h.GetUserRoles | — |
| POST | `/api/v1/rbac/users/:id/roles` | ✅ | h.AssignUserRoles | — |

---

## 22. Categories & Brands

### Categories (`/api/v1/categories`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/categories` | ✅ | h.List | — |
| GET | `/api/v1/categories/tree` | ✅ | h.Tree | — |
| GET | `/api/v1/categories/:id` | ✅ | h.Get | — |
| POST | `/api/v1/categories` | ✅ | h.Create | — |
| PUT | `/api/v1/categories/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/categories/:id` | ✅ | h.Delete | — |

### Brands (`/api/v1/brands`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/brands` | ✅ | h.List | — |
| GET | `/api/v1/brands/:id` | ✅ | h.Get | — |
| POST | `/api/v1/brands` | ✅ | h.Create | — |
| PUT | `/api/v1/brands/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/brands/:id` | ✅ | h.Delete | — |

---

## 23. Reports (`/api/v1/report`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/report/sales` | ✅ | h.Sales | `(main)/reports/page.tsx` |
| GET | `/api/v1/report/profit` | ✅ | h.Profit | `(main)/reports/page.tsx` |
| GET | `/api/v1/report/inventory` | ✅ | h.Inventory | `(main)/reports/page.tsx` |
| GET | `/api/v1/report/settlement` | ✅ | h.Settlement | `(main)/reports/page.tsx` |
| GET | `/api/v1/report/platform-fee` | ✅ | h.PlatformFee | `(main)/reports/page.tsx` |

---

## 24. Other Modules

### Image Gen (`/api/v1/imagegen`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET/POST/PUT/DELETE | `/api/v1/imagegen[/:id]` | ✅ | Various | — |
| GET/POST/PUT/DELETE | `/api/v1/imagegen/canvas[/:id]` | ✅ | Various | — |
| GET/POST/PUT/DELETE | `/api/v1/imagegen/templates[/:id]` | ✅ | Various | — |
| POST | `/api/v1/imagegen/templates/:id/use` | ✅ | UseTemplate | — |
| PUT | `/api/v1/imagegen/:id/status` | ✅ | UpdateImageGenStatus | — |

### Search (`/api/v1/search`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/search` | ✅ | h.Search | `(main)/search/page.tsx` |
| GET | `/api/v1/search/recent` | ✅ | h.Recent | — |

### Notification (`/api/v1/notification`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET/POST/PUT/DELETE | `/api/v1/notification[/:id]` | ✅ | Various (List, Get, Create, MarkAsRead, Delete) | — |
| GET | `/api/v1/notification/unread-count` | ✅ | h.UnreadCount | — |
| PUT | `/api/v1/notification/read-all` | ✅ | h.MarkAllRead | — |
| CRUD | `/api/v1/notification/alert-rules[/:id]` | ✅ | Various (ListAlertRules, CreateAlertRule, etc.) | — |

### Operation Log (`/api/v1/operationlog`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/operationlog` | ✅ | h.List | — |
| GET | `/api/v1/operationlog/:id` | ✅ | h.Get | — |
| POST | `/api/v1/operationlog` | ✅ | h.Create | — |

### Product Analysis (`/api/v1/product-analysis`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/product-analysis/analyze` | ✅ | h.Analyze | — |
| GET | `/api/v1/product-analysis/analyses` | ✅ | h.ListAnalyses | — |
| GET | `/api/v1/product-analysis/analyses/:id` | ✅ | h.GetAnalysis | — |
| POST | `/api/v1/product-analysis/analyses/:id/feedback` | ✅ | h.RecordFeedback | — |

### Aftersales (`/api/v1/aftersales`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET/POST/PUT/DELETE | `/api/v1/aftersales[/:id]` | ✅ | CRUD + Approve, Reject, Receive, Refund | — |
| GET | `/api/v1/aftersales/summary` | ✅ | h.Summary | — |

### Exceptions (`/api/v1/exceptions`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET/POST/PUT/DELETE | `/api/v1/exceptions[/:id]` | ✅ | CRUD + Resolve, Assign | — |

### Allocation (`/api/v1/allocation`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| CRUD | `/api/v1/allocation/warehouses[/:id]` | ✅ | Warehouses | — |
| CRUD | `/api/v1/allocation/rules[/:id]` | ✅ | Rules | — |
| CRUD | `/api/v1/allocation/cost/batches[/:id]` | ✅ | Batches | — |

### Exchange Rates (`/api/v1/exchange-rates`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/exchange-rates` | ✅ | h.List | — |
| POST | `/api/v1/exchange-rates` | ✅ | h.Create | — |
| DELETE | `/api/v1/exchange-rates/:id` | ✅ | h.Delete | — |
| PUT | `/api/v1/exchange-rates/:from_currency/:to_currency` | ✅ | h.UpdateByPair | — |
| GET | `/api/v1/exchange-rates/:from_currency/:to_currency/latest` | ✅ | h.GetLatest | — |

### Import Batch (`/api/v1/importbatch`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET/POST/PUT/DELETE | `/api/v1/importbatch[/:id]` | ✅ | CRUD | — |
| GET | `/api/v1/importbatch/:id/rows` | ✅ | ListRows | — |

### Purchase (`/api/v1/purchase`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET/POST | `/api/v1/purchase/orders[/:id]` | ✅ | ListOrders, CreateOrder, GetOrder, ApproveOrder, ReceiveOrder, CancelOrder | — |
| GET/POST | `/api/v1/purchase/suggestions` | ✅ | ListSuggestions, GenerateSuggestions | — |
| CRUD | `/api/v1/purchase/suppliers[/:id]` | ✅ | Supplier CRUD + GetSupplierKPI | — |

### Sourcing 1688 (`/api/v1/sourcing1688`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET/POST/PUT/DELETE | `/api/v1/sourcing1688[/:id]` | ✅ | CRUD + Import, Reject | — |
| GET | `/api/v1/sourcing1688/summary` | ✅ | h.Summary | — |

### Support (`/api/v1/support`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| CRUD + Reply/Close/Messages | `/api/v1/support/conversations[/:id]` | ✅ | Various | — |
| CRUD | `/api/v1/support/templates[/:id]` | ✅ | Various | — |
| List/Add/Check/Delete | `/api/v1/support/blacklist` | ✅ | Various | — |

### Action Policy (`/api/v1/policy`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| CRUD | `/api/v1/policy/rules[/:id]` | ✅ | ListRules, GetRule, CreateRule, UpdateRule, DeleteRule | `(main)/settings/policy/page.tsx` |
| POST | `/api/v1/policy/evaluate` | ✅ | h.Evaluate | — |

### Agent Rule (`/api/v1/agent-rules`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| CRUD + Toggle | `/api/v1/agent-rules[/:id]` | ✅ | Various | — |
| POST | `/api/v1/agent-rules/evaluate` | ✅ | h.EvaluateRules | — |

---

## 25. Missing Endpoints (Frontend calls without backend routes)

These endpoints are referenced by frontend code but have **no backend handler or route** registered.

| Method | Path | Frontend File | Notes |
|--------|------|---------------|-------|
| GET | `/v1/listing-task/stats` | `listing-tasks/workbench/page.tsx` | Returns `{ pending, total }` — no handler |
| POST | `/v1/listing-task/retry-all` | `listing-tasks/workbench/page.tsx` | Batch retry — no handler |
| GET | `/v1/settings/llm` | `settings/llm/page.tsx` | LLM config — no settings module exists |
| POST | `/v1/listing` | `listings/create/page.tsx` | **Path mismatch** — backend has `POST /v1/listings` (plural). Needs alignment. |

---

## 26. Summary Statistics

| Category | Count |
|----------|-------|
| **Total registered routes** | ~225 |
| **Modules with full CRUD (handler + service)** | 38 of 38 |
| **Modules with handler stubs only** | 0 |
| **Frontend-referenced endpoints** | ~65 |
| **Frontend calls with no backend route** | 3 + 1 path mismatch |

---

## 27. Key Architecture Notes

1. **Products and SKUs share a single module** (`internal/domain/sku/`). The `sku` package handles both `/products` and `/skus` routes. There is no separate `domain/products/` directory.
2. **Listings split across two prefixes**: CRUD under `/listings`, publish chain under `/listing`. This is intentional (regular CRUD vs. operational publishing).
3. **Settings module does not exist** in backend-go. The frontend settings/llm page calls `/v1/settings/llm` but no module is registered in `router.go` for it. Likely a Phase 2 addition.
4. **Listing task workbench** (`GET /v1/listing-task/stats` and `POST /v1/listing-task/retry-all`) are called by the frontend workbench page but have no backend handlers. These must be implemented in `listingtask/handler.go` and registered in `listingtask/routes.go`.
5. **Scheduler-driven agents** (A0-A7, G0-G3) have no HTTP endpoints — they run as event bus subscriptions on cron ticks, not as REST API calls.
6. **Frontend path mismatch**: `POST /v1/listing` (frontend) vs `POST /v1/listings` (backend) for listing creation. Gin routing may silently normalize this, but it is still a drift risk.
