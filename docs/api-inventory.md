# API Endpoint Inventory

> **注意**：这是 2026-07-03 的历史快照，不再作为 API 路由事实源。
> 当前完整清单见 [完整 API 参考](reference-api-complete.md)，它由 Gin 运行时路由表生成。
> 本文仅为兼容旧链接而保留。
>
> Generated 2026-07-03. Covers `backend-go/` (Go/Gin) HTTP endpoints under `/api/v1`.
> Last updated: 2026-07-03

## Legend

| Column | Meaning |
|--------|---------|
| Status | ✅ handler+service exist, 🟡 handler exists but stub/unproven, 🔧 frontend-only (no backend route) |
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
| POST | `/api/v1/webhooks/:platform` | ✅ | h.ReceiveWebhook | — |
| WS | `/ws` | ✅ | realtime.Handler.ServeWS | — |
| WS | `/ws/extension` | ✅ | extHandler.ServeWS | — |

---

## 2. Dashboard (`/api/v1/dashboard`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/dashboard/overview` | ✅ | h.Overview | `(main)/dashboard/page.tsx` |
| GET | `/api/v1/dashboard/orders` | ✅ | h.Orders | — |
| GET | `/api/v1/dashboard/inventory` | ✅ | h.Inventory | — |
| GET | `/api/v1/dashboard/exceptions` | ✅ | h.Exceptions | — |
| GET | `/api/v1/dashboard/rejection-reasons` | ✅ | h.RejectionReasons | — |

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
| POST | `/api/v1/products` | ✅ | h.CreateProduct | `(main)/products/create/page.tsx` |
| PUT | `/api/v1/products/:id` | ✅ | h.UpdateProduct | — |
| DELETE | `/api/v1/products/:id` | ✅ | h.DeleteProduct | — |
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

## 4. Pricing (`/api/v1/prices`, `/api/v1/skus/:id/prices`, `/api/v1/competitor-prices`, `/api/v1/pricing-strategies`, `/api/v1/pricing-recommendations`)

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
| GET | `/api/v1/competitor-prices` | ✅ | h.ListCompetitorPrices | — |
| GET | `/api/v1/competitor-prices/:id` | ✅ | h.GetCompetitorPrice | — |
| POST | `/api/v1/competitor-prices` | ✅ | h.CreateCompetitorPrice | — |
| DELETE | `/api/v1/competitor-prices/:id` | ✅ | h.DeleteCompetitorPrice | — |
| GET | `/api/v1/pricing-strategies` | ✅ | h.ListPricingStrategies | — |
| GET | `/api/v1/pricing-strategies/:id` | ✅ | h.GetPricingStrategy | — |
| POST | `/api/v1/pricing-strategies` | ✅ | h.SavePricingStrategy | — |
| PUT | `/api/v1/pricing-strategies/:id` | ✅ | h.UpdatePricingStrategy | — |
| DELETE | `/api/v1/pricing-strategies/:id` | ✅ | h.DeletePricingStrategy | — |
| GET | `/api/v1/pricing-recommendations` | ✅ | h.ListRecommendations | — |
| POST | `/api/v1/pricing-recommendations/generate` | ✅ | h.GenerateRecommendation | — |
| POST | `/api/v1/pricing-recommendations/:id/apply` | ✅ | h.ApplyRecommendation | — |

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
| GET | `/api/v1/inventory/sku/:sku_id/warehouses` | ✅ | h.ListInventoryBySku | — |
| GET | `/api/v1/inventory/locations` | ✅ | h.ListLocations | — |
| GET | `/api/v1/inventory/transfers` | ✅ | h.ListTransfers | — |
| POST | `/api/v1/inventory/sync-cross-platform/:productId` | ✅ | h.SyncCrossPlatform | — |
| GET | `/api/v1/inventory/oversell-report` | ✅ | h.OversellReport | — |

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
| POST | `/api/v1/listing` | ✅ | h.Create (alias) | `(main)/listings/create/page.tsx` |
| POST | `/api/v1/listing/products/:product_id/publish/:platform_id` | ✅ | h.PublishProduct | — |
| GET | `/api/v1/listing/products/:product_id/listings` | ✅ | h.ListByProduct | — |
| POST | `/api/v1/listing/listing-tasks/from-decisions` | ✅ | h.CreateTasksFromDecisions | — |
| POST | `/api/v1/listing/listing-tasks/:task_id/recheck` | ✅ | h.RecheckTask | — |
| POST | `/api/v1/listing/listing-tasks/:task_id/cancel` | ✅ | h.CancelTask | — |
| POST | `/api/v1/listing/listing-tasks/:task_id/publish` | ✅ | h.PublishTask | — |

**Note:** `POST /v1/listing` is registered as an alias of `POST /v1/listings` for frontend compatibility.

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
| GET | `/api/v1/listing-task/stats` | ✅ | h.ListStats | `listing-tasks/workbench/page.tsx` |
| POST | `/api/v1/listing-task/retry-all` | ✅ | h.RetryAll | `listing-tasks/workbench/page.tsx` |
| POST | `/api/v1/listing-task/:task_id/execute` | ✅ | h.Execute | `listing-tasks/[id]/page.tsx` |
| POST | `/api/v1/listing-task/:task_id/retry-failed` | ✅ | h.RetryFailed | `listing-tasks/[id]/page.tsx` |
| POST | `/api/v1/listing-task/:task_id/items/:item_id/retry` | ✅ | h.RetryItem | — |

**Return types** (from `domain/listingtask/model.go`):
- `TaskResponse` — `{ id, status, platform_id, product_id, total_items, completed, failed }`
- `TaskItemResponse` — `{ id, task_id, sku_id, platform_id, status, error }`

---

## 9. Supplier (`/api/v1/suppliers`, `/api/v1/product-suppliers`, `/api/v1/products/:id/supplier-comparison`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/suppliers` | ✅ | h.List | — |
| GET | `/api/v1/suppliers/scoreboard` | ✅ | h.ListScoreboard | — |
| GET | `/api/v1/suppliers/:id` | ✅ | h.Get | — |
| GET | `/api/v1/suppliers/:id/score` | ✅ | h.GetScore | — |
| POST | `/api/v1/suppliers/:id/recalculate` | ✅ | h.RecalculateScore | — |
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

## 11. Platform Integrations (`/api/v1/platform-integrations`, `/api/v1/platform-webhooks`)

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
| GET | `/api/v1/platform-integrations/:id/ozon-products` | ✅ | h.ListOzonProducts | — |
| GET | `/api/v1/platform-webhooks/config` | ✅ | h.GetConfig | — |
| POST | `/api/v1/platform-webhooks/test-event` | ✅ | h.TestEvent | — |

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
| POST | `/api/v1/shipping/bill-batches/import` | ✅ | h.ImportBill | — |
| DELETE | `/api/v1/shipping/bill-batches/:id` | ✅ | h.DeleteBillBatch | — |
| GET | `/api/v1/shipping/bill-batches/:id/items` | ✅ | h.ListBillItems | — |
| POST | `/api/v1/shipping/bill-batches/:id/reconcile` | ✅ | h.ReconcileBillBatch | — |
| GET | `/api/v1/shipping/bill-batches/:id/anomalies` | ✅ | h.ListBillAnomalies | — |
| POST | `/api/v1/shipping/quote-unified` | ✅ | h.QuoteUnified | — |
| POST | `/api/v1/shipping/snapshots` | ✅ | h.CreateSnapshot | — |
| GET | `/api/v1/shipping/snapshots` | ✅ | h.ListSnapshots | — |
| GET | `/api/v1/shipping/snapshots/:orderId` | ✅ | h.GetSnapshot | — |
| PUT | `/api/v1/shipping/bill-items/:id/review` | ✅ | h.ReviewBillItem | — |
| GET | `/api/v1/shipping/rules/:id/versions` | ✅ | h.ListRuleVersions | — |
| POST | `/api/v1/shipping/tracking` | ✅ | h.CreateTracking | — |
| GET | `/api/v1/shipping/tracking` | ✅ | h.ListTracking | — |
| GET | `/api/v1/shipping/tracking/:orderId` | ✅ | h.GetTracking | — |
| PUT | `/api/v1/shipping/tracking/:id/event` | ✅ | h.UpdateTrackingEvent | — |
| PUT | `/api/v1/shipping/tracking/:id/exception` | ✅ | h.MarkTrackingException | — |
| GET | `/api/v1/shipping/carrier-performance` | ✅ | h.GetCarrierPerformance | — |
| GET | `/api/v1/shipping/carriers` | ✅ | h.ListCarriers | — |
| POST | `/api/v1/shipping/carriers/:code/quote` | ✅ | h.CarrierQuote | — |

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

## 18. AI / AgentOS

### AI (`/api/v1/ai`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/ai/chat` | ✅ | h.Chat | `(main)/ai/page.tsx`, `components/copilot/CopilotPanel.tsx` |
| POST | `/api/v1/ai/run` | ✅ | h.RunAgent | `(main)/ai/page.tsx` |
| GET | `/api/v1/ai/traces` | ✅ | h.ListTraces | `(main)/ai/page.tsx` |
| GET | `/api/v1/ai/actions` | ✅ | h.ListActions | `(main)/ai/page.tsx`, `(main)/actions/page.tsx` |
| GET | `/api/v1/ai/agents` | ✅ | h.Roster | `(main)/ai/page.tsx` |
| GET | `/api/v1/ai/agents/specs` | ✅ | h.AgentSpecs | — |
| POST | `/api/v1/ai/actions` | ✅ | h.CreateAction | — |
| GET | `/api/v1/ai/traces/:trace_id` | ✅ | h.GetTrace | `agents/[id]/trace/[traceId]/page.tsx` |
| GET | `/api/v1/ai/actions/:id` | ✅ | h.GetAction | `actions/[id]/page.tsx` |
| POST | `/api/v1/ai/actions/:id/approve` | ✅ | h.ApproveAction | `ai/page.tsx`, `agentos/page.tsx`, `actions/page.tsx` |
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
| POST | `/api/v1/agents/:id/actions` | ✅ | h.ExecuteAction | — |

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
| GET | `/api/v1/agentos/work-items` | ✅ | h.WorkItems | `(main)/agentos/page.tsx` |
| GET | `/api/v1/agentos/work-items/:id` | ✅ | h.WorkItemDetail | — |
| GET | `/api/v1/agentos/agent-timeline` | ✅ | h.AgentTimeline | — |
| GET | `/api/v1/agentos/failures` | ✅ | h.FailedRuns | — |
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
| GET | `/api/v1/rbac/users/:id/roles` | ✅ | h.GetUserRoles | — |
| POST | `/api/v1/rbac/users/:id/roles` | ✅ | h.AssignUserRoles | — |

**Note:** `GET /api/v1/rbac/current/permissions` is defined in `RegisterPublicRoutes` which is not called by `router.go`. It is available through the admin `rbac.manage`-protected group.

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

## 24. Settings (`/api/v1/settings`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/settings/llm` | ✅ | h.GetLLM | `(main)/settings/llm/page.tsx` |
| PUT | `/api/v1/settings/llm` | ✅ | h.UpdateLLM | — |

---

## 25. Candidate Products (`/api/v1/candidates`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/candidates` | ✅ | h.List | — |
| GET | `/api/v1/candidates/count` | ✅ | h.Count | — |
| GET | `/api/v1/candidates/dedup` | ✅ | h.Dedup | — |
| GET | `/api/v1/candidates/collect-leads` | ✅ | h.ListCollectLeads | — |
| GET | `/api/v1/candidates/collect-leads/:id` | ✅ | h.GetCollectLead | — |
| POST | `/api/v1/candidates` | ✅ | h.Create | — |
| GET | `/api/v1/candidates/:id` | ✅ | h.Get | — |
| PUT | `/api/v1/candidates/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/candidates/:id` | ✅ | h.Delete | — |
| PUT | `/api/v1/candidates/:id/fields` | ✅ | h.FillFields | — |
| POST | `/api/v1/candidates/:id/skip-field` | ✅ | h.SkipField | — |
| POST | `/api/v1/candidates/:id/rescrape` | ✅ | h.Rescrape | — |
| POST | `/api/v1/candidates/seed` | ✅ | h.Seed | — |

---

## 26. Completeness (`/api/v1/completeness`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/completeness/checks` | ✅ | h.ListChecks | — |
| POST | `/api/v1/completeness/check/:productId` | ✅ | h.Check | — |

---

## 27. Compliance (`/api/v1/compliance`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/compliance/check` | ✅ | h.Check | — |
| POST | `/api/v1/compliance/scan` | ✅ | h.Scan | — |
| GET | `/api/v1/compliance/results` | ✅ | h.ListResults | — |
| GET | `/api/v1/compliance/results/:id` | ✅ | h.GetResult | — |
| PUT | `/api/v1/compliance/results/:id/suppress` | ✅ | h.SuppressResult | — |

---

## 28. Profit (`/api/v1/profit`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/profit/summaries` | ✅ | h.ListSummaries | — |
| GET | `/api/v1/profit/summary/:productId` | ✅ | h.Summary | — |

---

## 29. Evaluation Loop (`/api/v1/loop`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/loop/recommendations` | ✅ | h.GetRecommendations | — |
| POST | `/api/v1/loop/evaluate/:productId` | ✅ | h.Evaluate | — |

---

## 30. Mock Data (`/api/v1/mock`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/mock/seed` | ✅ | h.Seed | — |
| GET | `/api/v1/mock/orders` | ✅ | h.ListOrders | — |
| GET | `/api/v1/mock/settlements` | ✅ | h.ListSettlements | — |
| GET | `/api/v1/mock/sync-statuses` | ✅ | h.SyncStatuses | — |

---

## 31. Owner Cockpit (`/api/v1/owner`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/owner/risk-summary` | ✅ | h.RiskSummary | — |
| GET | `/api/v1/owner/suggestions` | ✅ | h.Suggestions | — |
| GET | `/api/v1/owner/platform-sync` | ✅ | h.PlatformSyncStatus | — |
| POST | `/api/v1/owner/suggestions/:id/feedback` | ✅ | h.Feedback | — |
| GET | `/api/v1/owner/agent-activity` | ✅ | h.AgentActivity | — |
| GET | `/api/v1/owner/pipeline-chain` | ✅ | h.PipelineChain | — |

---

## 32. Agent Learning (`/api/v1/agent-learning`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/agent-learning/accuracy` | ✅ | h.GetAllAccuracy | — |
| GET | `/api/v1/agent-learning/accuracy/:agentId` | ✅ | h.GetAccuracyByAgent | — |
| GET | `/api/v1/agent-learning/evaluations` | ✅ | h.ListEvaluations | — |
| POST | `/api/v1/agent-learning/evaluate` | ✅ | h.EvaluateDecision | — |
| POST | `/api/v1/agent-learning/recalculate` | ✅ | h.RecalculateAccuracy | — |

---

## 33. Approval (`/api/v1/approval`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/approval` | ✅ | h.ListApprovals | — |
| GET | `/api/v1/approval/:id` | ✅ | h.GetApproval | — |
| POST | `/api/v1/approval` | ✅ | h.CreateApproval | — |
| PUT | `/api/v1/approval/:id/review` | ✅ | h.ReviewApproval | — |
| GET | `/api/v1/approval/my` | ✅ | h.MyPending | — |
| GET | `/api/v1/approval/stats` | ✅ | h.ApprovalStats | — |

---

## 34. Landed Cost (`/api/v1/landed-cost`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/landed-cost/calculate` | ✅ | h.Calculate | — |
| GET | `/api/v1/landed-cost/:productId` | ✅ | h.GetLandedCost | — |
| GET | `/api/v1/landed-cost/:productId/compare` | ✅ | h.CompareAcrossPlatforms | — |

---

## 35. Orchestration (`/api/v1/orchestration`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/orchestration/products/:id/pipeline` | ✅ | h.GetPipelineStatus | — |
| POST | `/api/v1/orchestration/products/:id/pipeline/start` | ✅ | h.StartPipeline | — |
| POST | `/api/v1/orchestration/products/:id/pipeline/step/:step/retry` | ✅ | h.RetryStep | — |
| GET | `/api/v1/orchestration/pipeline/config` | ✅ | h.ListConfigs | — |
| POST | `/api/v1/orchestration/pipeline/config` | ✅ | h.CreateConfig | — |

---

## 36. Workflow (`/api/v1/workflow`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/workflow/defs` | ✅ | h.ListDefs | — |
| GET | `/api/v1/workflow/defs/:id` | ✅ | h.GetDef | — |
| POST | `/api/v1/workflow/defs` | ✅ | h.CreateDef | — |
| PUT | `/api/v1/workflow/defs/:id` | ✅ | h.UpdateDef | — |
| DELETE | `/api/v1/workflow/defs/:id` | ✅ | h.DeleteDef | — |
| POST | `/api/v1/workflow/defs/:defId/start` | ✅ | h.StartRun | — |
| POST | `/api/v1/workflow/runs/:id/pause` | ✅ | h.PauseRun | — |
| POST | `/api/v1/workflow/runs/:id/resume` | ✅ | h.ResumeRun | — |
| GET | `/api/v1/workflow/runs` | ✅ | h.ListRuns | — |
| GET | `/api/v1/workflow/runs/:id` | ✅ | h.GetRun | — |
| POST | `/api/v1/workflow/runs/:id/advance` | ✅ | h.AdvanceStep | — |

---

## 37. Product Hub (`/api/v1/product-hub`)
Product Hub is registered under the product.read permission group. It includes Master CRUD, Variants, Supplier Offers, Sample Requests, Cost Versions, and sub-resources under `/products/:id` for versions, freshness, relations, and 360 dashboard.

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/product-hub` | ✅ | masterH.List | `(main)/product-hub/page.tsx` |
| GET | `/api/v1/product-hub/:id` | ✅ | masterH.Get | — |
| POST | `/api/v1/product-hub` | ✅ | masterH.Create | — |
| PUT | `/api/v1/product-hub/:id` | ✅ | masterH.Update | — |
| DELETE | `/api/v1/product-hub/:id` | ✅ | masterH.Delete | — |
| POST | `/api/v1/product-hub/:id/transition` | ✅ | masterH.TransitionLifecycle | — |
| GET | `/api/v1/product-hub/:id/hub` | ✅ | hubH.GetHub | — |
| GET | `/api/v1/product-hub/:id/variants` | ✅ | inline (variantSvc) | — |
| POST | `/api/v1/product-hub/variants` | ✅ | inline (variantSvc) | — |
| GET | `/api/v1/product-hub/:id/offers` | ✅ | inline (offerSvc) | — |
| POST | `/api/v1/product-hub/offers` | ✅ | inline (offerSvc) | — |
| GET | `/api/v1/product-hub/:id/samples` | ✅ | inline (sampleSvc) | — |
| POST | `/api/v1/product-hub/samples` | ✅ | inline (sampleSvc) | — |
| GET | `/api/v1/product-hub/:id/costs` | ✅ | inline (costSvc) | — |
| POST | `/api/v1/product-hub/costs` | ✅ | inline (costSvc) | — |
| POST | `/api/v1/product-hub/costs/:costId/confirm` | ✅ | inline (costSvc) | — |

### Products sub-resources (`/api/v1/products/:id/` — product hub add-ons)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/products/:id/versions` | ✅ | h.ListVersions | — |
| GET | `/api/v1/products/:id/versions/:versionId` | ✅ | h.GetVersion | — |
| POST | `/api/v1/products/:id/versions/:versionId/rollback` | ✅ | h.Rollback | — |
| POST | `/api/v1/products/:id/decisions` | ✅ | h.RecordDecision | — |
| GET | `/api/v1/products/:id/freshness` | ✅ | h.GetProductFreshness | — |
| GET | `/api/v1/products/freshness/stale` | ✅ | h.ListStaleProducts | — |
| POST | `/api/v1/products/:id/freshness/verify` | ✅ | h.VerifyDimension | — |
| GET | `/api/v1/products/:id/relations` | ✅ | h.GetRelatedProducts | — |
| POST | `/api/v1/products/:id/discover-relations` | ✅ | h.AutoDiscoverRelations | — |
| GET | `/api/v1/products/360/summary` | ✅ | inline (ProductHub) | — |
| GET | `/api/v1/products/decision` | ✅ | inline (ProductHub) | — |
| POST | `/api/v1/products/relations` | ✅ | h.CreateRelation | — |
| DELETE | `/api/v1/products/relations/:id` | ✅ | h.DeleteRelation | — |

---

## 38. Content (`/api/v1/content`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/content/generate` | ✅ | h.GenerateContent | — |
| POST | `/api/v1/content/validate` | ✅ | h.ValidateContent | — |

---

## 39. Sentiment (`/api/v1/sentiment`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/sentiment/negative` | ✅ | h.ListNegativeSentiment | — |
| GET | `/api/v1/sentiment/:productId` | ✅ | h.GetProductSentiment | — |
| POST | `/api/v1/sentiment/:productId/refresh` | ✅ | h.RefreshSentiment | — |

---

## 40. Image Gen (`/api/v1/imagegen`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/imagegen` | ✅ | h.ListImageGens | — |
| GET | `/api/v1/imagegen/:id` | ✅ | h.GetImageGen | — |
| POST | `/api/v1/imagegen` | ✅ | h.CreateImageGen | — |
| PUT | `/api/v1/imagegen/:id/status` | ✅ | h.UpdateImageGenStatus | — |
| DELETE | `/api/v1/imagegen/:id` | ✅ | h.DeleteImageGen | — |
| GET | `/api/v1/imagegen/canvas` | ✅ | h.ListCanvases | — |
| POST | `/api/v1/imagegen/canvas` | ✅ | h.CreateCanvas | — |
| GET | `/api/v1/imagegen/canvas/:id` | ✅ | h.GetCanvas | — |
| PUT | `/api/v1/imagegen/canvas/:id` | ✅ | h.UpdateCanvas | — |
| DELETE | `/api/v1/imagegen/canvas/:id` | ✅ | h.DeleteCanvas | — |
| GET | `/api/v1/imagegen/templates` | ✅ | h.ListTemplates | — |
| POST | `/api/v1/imagegen/templates` | ✅ | h.CreateTemplate | — |
| GET | `/api/v1/imagegen/templates/:id` | ✅ | h.GetTemplate | — |
| PUT | `/api/v1/imagegen/templates/:id` | ✅ | h.UpdateTemplate | — |
| POST | `/api/v1/imagegen/templates/:id/use` | ✅ | h.UseTemplate | — |
| DELETE | `/api/v1/imagegen/templates/:id` | ✅ | h.DeleteTemplate | — |

---

## 41. Search (`/api/v1/search`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/search` | ✅ | h.Search | `(main)/search/page.tsx` |
| GET | `/api/v1/search/recent` | ✅ | h.Recent | — |

---

## 42. Notification (`/api/v1/notification`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/notification` | ✅ | h.List | — |
| GET | `/api/v1/notification/unread-count` | ✅ | h.UnreadCount | — |
| PUT | `/api/v1/notification/read-all` | ✅ | h.MarkAllRead | — |
| GET | `/api/v1/notification/:id` | ✅ | h.Get | — |
| POST | `/api/v1/notification` | ✅ | h.Create | — |
| PUT | `/api/v1/notification/:id/read` | ✅ | h.MarkAsRead | — |
| DELETE | `/api/v1/notification/:id` | ✅ | h.Delete | — |
| GET | `/api/v1/notification/alert-rules` | ✅ | h.ListAlertRules | — |
| POST | `/api/v1/notification/alert-rules` | ✅ | h.CreateAlertRule | — |
| PUT | `/api/v1/notification/alert-rules/:id` | ✅ | h.UpdateAlertRule | — |
| DELETE | `/api/v1/notification/alert-rules/:id` | ✅ | h.DeleteAlertRule | — |

---

## 43. Operation Log (`/api/v1/operationlog`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/operationlog` | ✅ | h.List | — |
| GET | `/api/v1/operationlog/:id` | ✅ | h.Get | — |
| POST | `/api/v1/operationlog` | ✅ | h.Create | — |

---

## 44. Product Analysis (`/api/v1/product-analysis`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/product-analysis/analyze` | ✅ | h.Analyze | — |
| GET | `/api/v1/product-analysis/analyses` | ✅ | h.ListAnalyses | — |
| GET | `/api/v1/product-analysis/analyses/:id` | ✅ | h.GetAnalysis | — |
| POST | `/api/v1/product-analysis/analyses/:id/feedback` | ✅ | h.RecordFeedback | — |

---

## 45. Aftersales (`/api/v1/aftersales`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/aftersales` | ✅ | h.List | — |
| GET | `/api/v1/aftersales/summary` | ✅ | h.Summary | — |
| GET | `/api/v1/aftersales/:id` | ✅ | h.Get | — |
| POST | `/api/v1/aftersales` | ✅ | h.Create | — |
| PUT | `/api/v1/aftersales/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/aftersales/:id` | ✅ | h.Delete | — |
| POST | `/api/v1/aftersales/:id/auto-decide` | ✅ | h.AutoDecide | — |
| POST | `/api/v1/aftersales/:id/approve` | ✅ | h.Approve | — |
| POST | `/api/v1/aftersales/:id/reject` | ✅ | h.Reject | — |
| POST | `/api/v1/aftersales/:id/receive` | ✅ | h.Receive | — |
| POST | `/api/v1/aftersales/:id/refund` | ✅ | h.Refund | — |

### Disputes (`/api/v1/aftersales/disputes`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/aftersales/disputes` | ✅ | h.ListDisputes | — |
| POST | `/api/v1/aftersales/disputes` | ✅ | h.CreateDispute | — |
| GET | `/api/v1/aftersales/disputes/:id` | ✅ | h.GetDispute | — |
| POST | `/api/v1/aftersales/disputes/:id/evaluate` | ✅ | h.EvaluateDispute | — |
| POST | `/api/v1/aftersales/disputes/:id/auto-decide` | ✅ | h.AutoDecideDispute | — |
| PUT | `/api/v1/aftersales/disputes/:id/status` | ✅ | h.UpdateDisputeStatus | — |

---

## 46. Exceptions (`/api/v1/exceptions`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/exceptions` | ✅ | h.List | — |
| GET | `/api/v1/exceptions/:id` | ✅ | h.Get | — |
| POST | `/api/v1/exceptions` | ✅ | h.Create | — |
| PUT | `/api/v1/exceptions/:id` | ✅ | h.Update | — |
| PUT | `/api/v1/exceptions/:id/resolve` | ✅ | h.Resolve | — |
| PUT | `/api/v1/exceptions/:id/assign` | ✅ | h.Assign | — |
| DELETE | `/api/v1/exceptions/:id` | ✅ | h.Delete | — |

---

## 47. Allocation (`/api/v1/allocation`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/allocation/warehouses` | ✅ | h.ListWarehouses | — |
| POST | `/api/v1/allocation/warehouses` | ✅ | h.CreateWarehouse | — |
| PUT | `/api/v1/allocation/warehouses/:id` | ✅ | h.UpdateWarehouse | — |
| DELETE | `/api/v1/allocation/warehouses/:id` | ✅ | h.DeleteWarehouse | — |
| GET | `/api/v1/allocation/rules` | ✅ | h.ListRules | — |
| POST | `/api/v1/allocation/rules` | ✅ | h.CreateRule | — |
| PUT | `/api/v1/allocation/rules/:id` | ✅ | h.UpdateRule | — |
| DELETE | `/api/v1/allocation/rules/:id` | ✅ | h.DeleteRule | — |
| GET | `/api/v1/allocation/cost/batches` | ✅ | h.ListBatches | — |
| POST | `/api/v1/allocation/cost/batches` | ✅ | h.CreateBatch | — |
| GET | `/api/v1/allocation/cost/batches/:id` | ✅ | h.GetBatch | — |
| POST | `/api/v1/allocation/cost/:batchId/compute` | ✅ | h.ComputeAllocation | — |
| POST | `/api/v1/allocation/auto-allocate/:skuId` | ✅ | h.AutoAllocate | — |

---

## 48. Exchange Rates (`/api/v1/exchange-rates`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/exchange-rates` | ✅ | h.List | — |
| POST | `/api/v1/exchange-rates` | ✅ | h.Create | — |
| DELETE | `/api/v1/exchange-rates/:id` | ✅ | h.Delete | — |
| PUT | `/api/v1/exchange-rates/:from_currency/:to_currency` | ✅ | h.UpdateByPair | — |
| GET | `/api/v1/exchange-rates/:from_currency/:to_currency/latest` | ✅ | h.GetLatest | — |

---

## 49. Import Batch (`/api/v1/importbatch`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/importbatch` | ✅ | h.ListBatches | — |
| GET | `/api/v1/importbatch/:id` | ✅ | h.GetBatch | — |
| POST | `/api/v1/importbatch` | ✅ | h.CreateBatch | — |
| POST | `/api/v1/importbatch/upload` | ✅ | h.Upload | — |
| PUT | `/api/v1/importbatch/:id` | ✅ | h.UpdateBatch | — |
| DELETE | `/api/v1/importbatch/:id` | ✅ | h.DeleteBatch | — |
| GET | `/api/v1/importbatch/:id/rows` | ✅ | h.ListRows | — |

---

## 50. Purchase (`/api/v1/purchase`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/purchase/orders` | ✅ | h.ListOrders | — |
| GET | `/api/v1/purchase/orders/:id` | ✅ | h.GetOrder | — |
| POST | `/api/v1/purchase/orders` | ✅ | h.CreateOrder | — |
| POST | `/api/v1/purchase/orders/:id/approve` | ✅ | h.ApproveOrder | — |
| POST | `/api/v1/purchase/orders/:id/receive` | ✅ | h.ReceiveOrder | — |
| POST | `/api/v1/purchase/orders/:id/cancel` | ✅ | h.CancelOrder | — |
| GET | `/api/v1/purchase/suggestions` | ✅ | h.ListSuggestions | — |
| POST | `/api/v1/purchase/suggestions/generate` | ✅ | h.GenerateSuggestions | — |

---

## 51. Sourcing 1688 (`/api/v1/sourcing1688`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/sourcing1688` | ✅ | h.List | — |
| GET | `/api/v1/sourcing1688/summary` | ✅ | h.Summary | — |
| GET | `/api/v1/sourcing1688/:id` | ✅ | h.Get | — |
| POST | `/api/v1/sourcing1688` | ✅ | h.Create | — |
| PUT | `/api/v1/sourcing1688/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/sourcing1688/:id` | ✅ | h.Delete | — |
| POST | `/api/v1/sourcing1688/:id/import` | ✅ | h.Import | — |
| POST | `/api/v1/sourcing1688/:id/reject` | ✅ | h.Reject | — |

---

## 52. Support (`/api/v1/support`)

### Conversations (`/api/v1/support/conversations`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/support/conversations` | ✅ | h.ListConversations | — |
| GET | `/api/v1/support/conversations/:id` | ✅ | h.GetConversation | — |
| POST | `/api/v1/support/conversations` | ✅ | h.CreateConversation | — |
| PUT | `/api/v1/support/conversations/:id` | ✅ | h.UpdateConversation | — |
| DELETE | `/api/v1/support/conversations/:id` | ✅ | h.DeleteConversation | — |
| POST | `/api/v1/support/conversations/:id/reply` | ✅ | h.SendReply | — |
| POST | `/api/v1/support/conversations/:id/close` | ✅ | h.CloseConversation | — |
| GET | `/api/v1/support/conversations/:id/messages` | ✅ | h.GetMessages | — |

### Templates (`/api/v1/support/templates`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/support/templates` | ✅ | h.ListTemplates | — |
| GET | `/api/v1/support/templates/:id` | ✅ | h.GetTemplate | — |
| POST | `/api/v1/support/templates` | ✅ | h.CreateTemplate | — |
| PUT | `/api/v1/support/templates/:id` | ✅ | h.UpdateTemplate | — |
| DELETE | `/api/v1/support/templates/:id` | ✅ | h.DeleteTemplate | — |

### Blacklist (`/api/v1/support/blacklist`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/support/blacklist` | ✅ | h.ListBlacklist | — |
| POST | `/api/v1/support/blacklist` | ✅ | h.AddBlacklist | — |
| GET | `/api/v1/support/blacklist/check` | ✅ | h.CheckBlacklist | — |
| DELETE | `/api/v1/support/blacklist/:id` | ✅ | h.DeleteBlacklist | — |

---

## 53. Action Policy (`/api/v1/policy`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/policy/rules` | ✅ | h.ListRules | `(main)/settings/policy/page.tsx` |
| GET | `/api/v1/policy/rules/:id` | ✅ | h.GetRule | — |
| POST | `/api/v1/policy/rules` | ✅ | h.CreateRule | — |
| PUT | `/api/v1/policy/rules/:id` | ✅ | h.UpdateRule | — |
| DELETE | `/api/v1/policy/rules/:id` | ✅ | h.DeleteRule | — |
| POST | `/api/v1/policy/rules/:id/toggle` | ✅ | h.HandleToggleRule | — |
| POST | `/api/v1/policy/evaluate` | ✅ | h.Evaluate | — |

---

## 54. Agent Rule (`/api/v1/agent-rules`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/agent-rules` | ✅ | h.ListRules | — |
| GET | `/api/v1/agent-rules/:id` | ✅ | h.GetRule | — |
| POST | `/api/v1/agent-rules` | ✅ | h.CreateRule | — |
| PUT | `/api/v1/agent-rules/:id` | ✅ | h.UpdateRule | — |
| DELETE | `/api/v1/agent-rules/:id` | ✅ | h.DeleteRule | — |
| POST | `/api/v1/agent-rules/:id/toggle` | ✅ | h.ToggleRule | — |
| POST | `/api/v1/agent-rules/evaluate` | ✅ | h.EvaluateRules | — |

---

## 55. Sourcing (`/api/v1/sourcing`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/sourcing/fetch` | ✅ | h.Fetch | — |
| GET | `/api/v1/sourcing/recommendations` | ✅ | h.ListRecommendations | — |
| GET | `/api/v1/sourcing/market-trends` | ✅ | h.MarketTrends | — |
| GET | `/api/v1/sourcing/keyword-trends` | ✅ | h.KeywordTrends | — |

---

## 56. Tariff (`/api/v1/tariff`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/tariff/decide` | ✅ | h.Decide | — |
| GET | `/api/v1/tariff` | ✅ | h.List | — |
| POST | `/api/v1/tariff` | ✅ | h.Create | — |
| GET | `/api/v1/tariff/:id` | ✅ | h.Get | — |
| PUT | `/api/v1/tariff/:id` | ✅ | h.Update | — |
| DELETE | `/api/v1/tariff/:id` | ✅ | h.Delete | — |

---

## 57. Logistics (`/api/v1/logistics`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/logistics/quote` | ✅ | h.GetQuotes | — |

---

## 58. Consolidation (`/api/v1/consolidation`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/consolidation/groups` | ✅ | h.CreateGroup | — |
| GET | `/api/v1/consolidation/groups` | ✅ | h.ListGroups | — |
| GET | `/api/v1/consolidation/groups/:groupId` | ✅ | h.GetGroup | — |
| POST | `/api/v1/consolidation/groups/:groupId/items` | ✅ | h.AddItem | — |
| GET | `/api/v1/consolidation/groups/:groupId/items` | ✅ | h.GetGroupItems | — |
| DELETE | `/api/v1/consolidation/groups/:groupId/items/:itemId` | ✅ | h.RemoveItem | — |
| POST | `/api/v1/consolidation/groups/:groupId/negotiate` | ✅ | h.NegotiateGroup | — |

---

## 59. Supply Chain (`/api/v1/supplychain/flows`, `/api/v1/supplychain/tracking`)

### Flows

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/supplychain/flows` | ✅ | h.List | — |
| GET | `/api/v1/supplychain/flows/:id` | ✅ | h.Get | — |
| GET | `/api/v1/supplychain/flows/:id/events` | ✅ | h.GetEvents | — |
| POST | `/api/v1/supplychain/flows` | ✅ | h.Create | — |
| PUT | `/api/v1/supplychain/flows/:id` | ✅ | h.Update | — |

### Tracking

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/supplychain/tracking` | ✅ | th.List | — |
| GET | `/api/v1/supplychain/tracking/:id` | ✅ | th.Get | — |
| GET | `/api/v1/supplychain/tracking/flow/:flowId` | ✅ | th.GetByFlow | — |
| POST | `/api/v1/supplychain/tracking` | ✅ | th.Create | — |
| PUT | `/api/v1/supplychain/tracking/:id/status` | ✅ | th.UpdateStatus | — |
| POST | `/api/v1/supplychain/tracking/:id/sync` | ✅ | th.SyncFromCarrier | — |

---

## 60. Feedback (`/api/v1/feedback`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| POST | `/api/v1/feedback/submissions` | ✅ | h.CreateSubmission | — |
| GET | `/api/v1/feedback/projects` | ✅ | h.ListProjects | — |
| GET | `/api/v1/feedback/projects/:id` | ✅ | h.GetProject | — |
| GET | `/api/v1/feedback/submissions/:id` | ✅ | h.GetSubmission | — |
| GET | `/api/v1/feedback/projects/:id/categories` | ✅ | h.ListCategories | — |
| POST | `/api/v1/feedback/categories` | ✅ | h.CreateCategory | — |
| PUT | `/api/v1/feedback/categories/:id` | ✅ | h.UpdateCategory | — |
| DELETE | `/api/v1/feedback/categories/:id` | ✅ | h.DeleteCategory | — |
| GET | `/api/v1/feedback/projects/:id/tags` | ✅ | h.ListTags | — |
| POST | `/api/v1/feedback/tags` | ✅ | h.CreateTag | — |
| DELETE | `/api/v1/feedback/tags/:id` | ✅ | h.DeleteTag | — |
| POST | `/api/v1/feedback/projects` | ✅ | h.CreateProject | — |
| PUT | `/api/v1/feedback/projects/:id` | ✅ | h.UpdateProject | — |
| DELETE | `/api/v1/feedback/projects/:id` | ✅ | h.DeleteProject | — |
| GET | `/api/v1/feedback/projects/:id/submissions` | ✅ | h.ListSubmissions | — |
| PUT | `/api/v1/feedback/submissions/:id` | ✅ | h.UpdateSubmission | — |
| PUT | `/api/v1/feedback/submissions/:id/status` | ✅ | h.UpdateSubmissionStatus | — |
| DELETE | `/api/v1/feedback/submissions/:id` | ✅ | h.DeleteSubmission | — |
| GET | `/api/v1/feedback/mine` | ✅ | h.ListMySubmissions | — |
| POST | `/api/v1/feedback/submissions/:id/vote` | ✅ | h.Vote | — |
| GET | `/api/v1/feedback/submissions/:id/comments` | ✅ | h.ListComments | — |
| POST | `/api/v1/feedback/submissions/:id/comments` | ✅ | h.AddComment | — |
| DELETE | `/api/v1/feedback/comments/:id` | ✅ | h.DeleteComment | — |
| POST | `/api/v1/feedback/submissions/:id/tags/:tagId` | ✅ | h.AddTag | — |
| DELETE | `/api/v1/feedback/submissions/:id/tags/:tagId` | ✅ | h.RemoveTag | — |
| GET | `/api/v1/feedback/projects/:id/stats` | ✅ | h.GetDashboardStats | — |
| GET | `/api/v1/feedback/projects/:id/analytics` | ✅ | h.GetAnalytics | — |
| POST | `/api/v1/feedback/migrate` | ✅ | h.Migrate | — |
| GET | `/api/v1/feedback/pending-for-agent` | ✅ | h.ListSubmissionsForAgent | — |
| GET | `/api/v1/feedback/assigned-to-me` | ✅ | h.ListSubmissionsForAgent | — |

**Note:** AI classifier and AgentOS triage are behind optional params passed as `nil` in `router.go`. These features are not currently wired.

---

## 61. Cost Dashboard (`/api/v1/cost`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/cost/dashboard` | ✅ | h.Dashboard | — |

---

## 62. Metabolism (`/api/v1/metabolism`)

| Method | Path | Status | Handler | Frontend Ref |
|--------|------|--------|---------|-------------|
| GET | `/api/v1/metabolism` | ✅ | h.ListLogs | — |
| GET | `/api/v1/metabolism/:id` | ✅ | h.GetLog | — |
| POST | `/api/v1/metabolism/dry-run` | ✅ | h.DryRun | — |
| POST | `/api/v1/metabolism/execute` | ✅ | h.ExecuteEntities | — |
| GET | `/api/v1/metabolism/excretion-result` | ✅ | h.GetExcretionResult | — |

**Note:** Metabolism service is instantiated with `nil, nil` for `ScoringAdapter` and `SemanticScorer`. Scoring and semantic analysis are not wired.

---

## 63. Other Modules

| Module | Prefix | Routes | Status |
|--------|--------|--------|--------|
| Webhooks (public) | `/api/v1/webhooks/:platform` | POST (receive) | ✅ |
| Webhooks (admin) | `/api/v1/platform-webhooks` | GET config, POST test-event | ✅ |

---

## 64. Known Gaps

| Module | Issue |
|--------|-------|
| **Metabolism (M1)** | `NewService` called with `nil, nil` for `ScoringAdapter` and `SemanticScorer`. Scoring and entity semantic analysis are not wired. Scheduler tick `M1` is registered and runs, but scoring logic is a no-op. |
| **Feedback** | AI classifier (`classifyFn`) and AgentOS triage (`actionCreator`) passed as `nil`. These features are not currently wired. |
| **A9 Scheduler** | `scheduler.tick.A9` subscriber is a no-op (`return nil`). A9 is API-driven only. |
| **RBAC Public Routes** | `RegisterPublicRoutes` (for `GET /rbac/current/permissions`) is defined but not called in `router.go`. The route may be inaccessible without the `rbac.manage` permission. |

---

## 65. Scheduler-Driven Agents (No HTTP Endpoints)

These agents run as event bus subscriptions on cron ticks, not as REST API calls.

| Agent | Decision Point | Interval | Description |
|-------|---------------|----------|-------------|
| G0 | system_health | 5 min | 协调仲裁健康检查 |
| A4 | auto_reply | 5 min | 客服待处理消息检查 |
| A5 | stock_alert | 15 min | 库存检查 |
| G3 | discount_risk_check | 30 min | 折扣风控扫描 |
| A6 | profit_watch | 1 hr | 利润看护 |
| A3 | acos_analysis | 1 hr | 广告分析 |
| G2 | warehouse_routing | 1 hr | 仓储报关 |
| trustscore | recalculate | 1 hr | 信任分重算 |
| entropy | defend | 6 hr | 熵防御周期 |
| ozon_sync | sync_orders | 15 min | Ozon 订单同步 |
| A8 | sourcing_scan | 1 hr | 选品扫描 |
| A9 | batch_operations | 2 hr | 批量运维扫描 (no-op) |
| M1 | excretion_scoring | 1 hr | 代谢排泄评分 |
| agentos | sla_escalation | 15 min | SLA过期升级待审批动作 |
| orch | supply_chain_heartbeat | 15 min | 供应链编排心跳 |

---

## 66. Summary Statistics

| Category | Count |
|----------|-------|
| **Total registered routes** | ~470 |
| **Modules with routes** | 71 |
| **Modules with full CRUD (handler + service)** | 63 of 71 |
| **Modules with stub/unproven wiring** | 2 (metabolism, feedback) |
| **Config-gated modules** | 0 |
| **Scheduler-driven agents (no HTTP)** | 15 |
| **Frontend-referenced endpoints** | ~80+ |
| **Frontend calls with no backend route** | 0 |

---

## 67. Key Architecture Notes

1. **Products and SKUs share a single module** (`internal/domain/sku/`). The `sku` package handles both `/products` and `/skus` routes. There is no separate `domain/products/` directory.
2. **Listings split across two prefixes**: CRUD under `/listings`, publish chain under `/listing`. The singular `/listing` prefix also includes `POST /v1/listing` as an alias of `POST /v1/listings` for frontend compatibility.
3. **Listing tasks split across two prefixes**: CRUD under `/listing-tasks`, operational endpoints (stats, retry, execute) under `/listing-task`.
4. **Product Hub** adds sub-resources under `/products/:id/` (versions, freshness, relations, decisions, 360 summary) which are separate from the core SKU module.
5. **Scheduler-driven agents** (15 total) have no HTTP endpoints. They run as event bus subscriptions on cron ticks.
6. **Pricing module** has been expanded with competitor prices, pricing strategies, and pricing recommendations (20 routes total).
7. **Shipping module** has the most routes (37) covering providers, channels, zones, rules, bill batches, tracking, carrier performance, and carrier API.
8. **Feedback module** has 27 routes (most of any single module) with public submissions and authenticated admin/agent endpoints.
9. **Webhook routes** are public (no auth) because external platforms need to call them. Admin webhook management routes are JWT-protected.
