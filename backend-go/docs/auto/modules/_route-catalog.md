# Route Catalog (auto-generated)

> Total modules: 65 | Total routes: 589

## Overview

| Module | Routes | Permission | Package |
|--------|--------|------------|--------|
| [`shipping`](shipping.md) | 39 | `shipping.read` | `domain/shipping/` |
| [`producthub`](producthub.md) | 30 | `product.read` | `domain/producthub/` |
| [`price`](price.md) | 20 | `finance.read` | `domain/price/` |
| [`workflow`](workflow.md) | 19 | `—` | `domain/workflow/` |
| [`sku`](sku.md) | 18 | `product.read` | `domain/sku/` |
| [`integrations`](integrations.md) | 18 | `—` | `domain/integrations/` |
| [`finance`](finance.md) | 18 | `finance.read` | `domain/finance/` |
| [`aftersales`](aftersales.md) | 17 | `—` | `domain/aftersales/` |
| [`inventory`](inventory.md) | 17 | `inventory.read` | `domain/inventory/` |
| [`listingtask`](listingtask.md) | 17 | `listing.read` | `domain/listingtask/` |
| [`support`](support.md) | 17 | `—` | `domain/support/` |
| [`listing`](listing.md) | 16 | `listing.read` | `domain/listing/` |
| [`imagegen`](imagegen.md) | 16 | `—` | `domain/imagegen/` |
| [`supplier`](supplier.md) | 16 | `—` | `domain/supplier/` |
| [`candidate`](candidate.md) | 14 | `—` | `domain/candidate/` |
| [`allocation`](allocation.md) | 13 | `—` | `domain/allocation/` |
| [`settlement`](settlement.md) | 11 | `settlement.read` | `domain/settlement/` |
| [`notification`](notification.md) | 11 | `—` | `domain/notification/` |
| [`supplychain`](supplychain.md) | 11 | `—` | `domain/supplychain/` |
| [`exceptions`](exceptions.md) | 10 | `—` | `domain/exceptions/` |
| [`platform`](platform.md) | 10 | `—` | `domain/platform/` |
| [`demandcase`](demandcase.md) | 9 | `—` | `domain/demandcase/` |
| [`experiment`](experiment.md) | 9 | `—` | `domain/experiment/` |
| [`orderimport`](orderimport.md) | 8 | `order.read` | `domain/orderimport/` |
| [`competitor`](competitor.md) | 8 | `—` | `domain/competitor/` |
| [`sourcing1688`](sourcing1688.md) | 8 | `—` | `domain/sourcing1688/` |
| [`purchase`](purchase.md) | 8 | `—` | `domain/purchase/` |
| [`decision`](decision.md) | 8 | `—` | `domain/decision/` |
| [`trustscore`](trustscore.md) | 7 | `—` | `domain/trustscore/` |
| [`importbatch`](importbatch.md) | 7 | `—` | `domain/importbatch/` |
| [`agentrule`](agentrule.md) | 7 | `—` | `domain/agentrule/` |
| [`report`](report.md) | 7 | `report.read` | `domain/report/` |
| [`actionpolicy`](actionpolicy.md) | 7 | `—` | `domain/actionpolicy/` |
| [`owner`](owner.md) | 7 | `—` | `domain/owner/` |
| [`order`](order.md) | 7 | `order.read` | `domain/order/` |
| [`consolidation`](consolidation.md) | 7 | `—` | `domain/consolidation/` |
| [`dashboard`](dashboard.md) | 6 | `—` | `domain/dashboard/` |
| [`platformfee`](platformfee.md) | 6 | `—` | `domain/platformfee/` |
| [`category`](category.md) | 6 | `—` | `domain/category/` |
| [`tariff`](tariff.md) | 6 | `—` | `domain/tariff/` |
| [`approval`](approval.md) | 6 | `—` | `domain/approval/` |
| [`personalrule`](personalrule.md) | 6 | `—` | `domain/personalrule/` |
| [`entropy`](entropy.md) | 5 | `—` | `domain/entropy/` |
| [`orchestration`](orchestration.md) | 5 | `—` | `domain/orchestration/` |
| [`compliance`](compliance.md) | 5 | `—` | `domain/compliance/` |
| [`productanalysis`](productanalysis.md) | 5 | `—` | `domain/productanalysis/` |
| [`brand`](brand.md) | 5 | `—` | `domain/brand/` |
| [`metabolism`](metabolism.md) | 5 | `—` | `domain/metabolism/` |
| [`sourcing`](sourcing.md) | 5 | `—` | `domain/sourcing/` |
| [`exchangerate`](exchangerate.md) | 5 | `—` | `domain/exchangerate/` |
| [`agentlearning`](agentlearning.md) | 5 | `—` | `domain/agentlearning/` |
| [`mock`](mock.md) | 4 | `—` | `domain/mock/` |
| [`evolution`](evolution.md) | 4 | `—` | `domain/evolution/` |
| [`loop`](loop.md) | 3 | `—` | `domain/loop/` |
| [`sentiment`](sentiment.md) | 3 | `—` | `domain/sentiment/` |
| [`profit`](profit.md) | 3 | `—` | `domain/profit/` |
| [`operationlog`](operationlog.md) | 3 | `—` | `domain/operationlog/` |
| [`landedcost`](landedcost.md) | 3 | `—` | `domain/landedcost/` |
| [`completeness`](completeness.md) | 3 | `—` | `domain/completeness/` |
| [`settings`](settings.md) | 2 | `—` | `domain/settings/` |
| [`search`](search.md) | 2 | `—` | `domain/search/` |
| [`reliability`](reliability.md) | 2 | `—` | `domain/reliability/` |
| [`content`](content.md) | 2 | `—` | `domain/content/` |
| [`logistics`](logistics.md) | 1 | `—` | `domain/logistics/` |
| [`cost`](cost.md) | 1 | `—` | `domain/cost/` |

## actionpolicy

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/policy/evaluate` | `h.Evaluate` |
| `GET` | `/api/v1/policy/rules` | `h.ListRules` |
| `POST` | `/api/v1/policy/rules` | `h.CreateRule` |
| `DELETE` | `/api/v1/policy/rules/:id` | `h.DeleteRule` |
| `GET` | `/api/v1/policy/rules/:id` | `h.GetRule` |
| `PUT` | `/api/v1/policy/rules/:id` | `h.UpdateRule` |
| `POST` | `/api/v1/policy/rules/:id/toggle` | `h.HandleToggleRule` |

## aftersales

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/aftersales` | `h.List` |
| `POST` | `/api/v1/aftersales` | `h.Create` |
| `DELETE` | `/api/v1/aftersales/:id` | `h.Delete` |
| `GET` | `/api/v1/aftersales/:id` | `h.Get` |
| `PUT` | `/api/v1/aftersales/:id` | `h.Update` |
| `POST` | `/api/v1/aftersales/:id/approve` | `h.Approve` |
| `POST` | `/api/v1/aftersales/:id/auto-decide` | `h.AutoDecide` |
| `POST` | `/api/v1/aftersales/:id/receive` | `h.Receive` |
| `POST` | `/api/v1/aftersales/:id/refund` | `h.Refund` |
| `POST` | `/api/v1/aftersales/:id/reject` | `h.Reject` |
| `GET` | `/api/v1/aftersales/disputes` | `h.ListDisputes` |
| `POST` | `/api/v1/aftersales/disputes` | `h.CreateDispute` |
| `GET` | `/api/v1/aftersales/disputes/:id` | `h.GetDispute` |
| `POST` | `/api/v1/aftersales/disputes/:id/auto-decide` | `h.AutoDecideDispute` |
| `POST` | `/api/v1/aftersales/disputes/:id/evaluate` | `h.EvaluateDispute` |
| `PUT` | `/api/v1/aftersales/disputes/:id/status` | `h.UpdateDisputeStatus` |
| `GET` | `/api/v1/aftersales/summary` | `h.Summary` |

## agentlearning

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/agent-learning/accuracy` | `h.GetAllAccuracy` |
| `GET` | `/api/v1/agent-learning/accuracy/:agentId` | `h.GetAccuracyByAgent` |
| `POST` | `/api/v1/agent-learning/evaluate` | `h.EvaluateDecision` |
| `GET` | `/api/v1/agent-learning/evaluations` | `h.ListEvaluations` |
| `POST` | `/api/v1/agent-learning/recalculate` | `h.RecalculateAccuracy` |

## agentrule

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/agent-rules` | `h.ListRules` |
| `POST` | `/api/v1/agent-rules` | `h.CreateRule` |
| `DELETE` | `/api/v1/agent-rules/:id` | `h.DeleteRule` |
| `GET` | `/api/v1/agent-rules/:id` | `h.GetRule` |
| `PUT` | `/api/v1/agent-rules/:id` | `h.UpdateRule` |
| `POST` | `/api/v1/agent-rules/:id/toggle` | `h.ToggleRule` |
| `POST` | `/api/v1/agent-rules/evaluate` | `h.EvaluateRules` |

## allocation

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/allocation/auto-allocate/:skuId` | `h.AutoAllocate` |
| `POST` | `/api/v1/allocation/cost/:batchId/compute` | `h.ComputeAllocation` |
| `GET` | `/api/v1/allocation/cost/batches` | `h.ListBatches` |
| `POST` | `/api/v1/allocation/cost/batches` | `h.CreateBatch` |
| `GET` | `/api/v1/allocation/cost/batches/:id` | `h.GetBatch` |
| `GET` | `/api/v1/allocation/rules` | `h.ListRules` |
| `POST` | `/api/v1/allocation/rules` | `h.CreateRule` |
| `DELETE` | `/api/v1/allocation/rules/:id` | `h.DeleteRule` |
| `PUT` | `/api/v1/allocation/rules/:id` | `h.UpdateRule` |
| `GET` | `/api/v1/allocation/warehouses` | `h.ListWarehouses` |
| `POST` | `/api/v1/allocation/warehouses` | `h.CreateWarehouse` |
| `DELETE` | `/api/v1/allocation/warehouses/:id` | `h.DeleteWarehouse` |
| `PUT` | `/api/v1/allocation/warehouses/:id` | `h.UpdateWarehouse` |

## approval

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/approval` | `h.ListApprovals` |
| `POST` | `/api/v1/approval` | `h.CreateApproval` |
| `GET` | `/api/v1/approval/:id` | `h.GetApproval` |
| `PUT` | `/api/v1/approval/:id/review` | `h.ReviewApproval` |
| `GET` | `/api/v1/approval/my` | `h.MyPending` |
| `GET` | `/api/v1/approval/stats` | `h.ApprovalStats` |

## brand

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/brands` | `h.List` |
| `POST` | `/api/v1/brands` | `h.Create` |
| `DELETE` | `/api/v1/brands/:id` | `h.Delete` |
| `GET` | `/api/v1/brands/:id` | `h.Get` |
| `PUT` | `/api/v1/brands/:id` | `h.Update` |

## candidate

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/candidates` | `h.List` |
| `POST` | `/api/v1/candidates` | `h.Create` |
| `DELETE` | `/api/v1/candidates/:id` | `h.Delete` |
| `GET` | `/api/v1/candidates/:id` | `h.Get` |
| `PUT` | `/api/v1/candidates/:id` | `h.Update` |
| `PUT` | `/api/v1/candidates/:id/fields` | `h.FillFields` |
| `POST` | `/api/v1/candidates/:id/rescrape` | `h.Rescrape` |
| `POST` | `/api/v1/candidates/:id/skip-field` | `h.SkipField` |
| `GET` | `/api/v1/candidates/collect-leads` | `h.ListCollectLeads` |
| `GET` | `/api/v1/candidates/collect-leads/:id` | `h.GetCollectLead` |
| `GET` | `/api/v1/candidates/collection-evidence/:id` | `h.GetCollectionEvidence` |
| `GET` | `/api/v1/candidates/count` | `h.Count` |
| `GET` | `/api/v1/candidates/dedup` | `h.Dedup` |
| `POST` | `/api/v1/candidates/seed` | `h.Seed` |

## category

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/categories` | `h.List` |
| `POST` | `/api/v1/categories` | `h.Create` |
| `DELETE` | `/api/v1/categories/:id` | `h.Delete` |
| `GET` | `/api/v1/categories/:id` | `h.Get` |
| `PUT` | `/api/v1/categories/:id` | `h.Update` |
| `GET` | `/api/v1/categories/tree` | `h.Tree` |

## competitor

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/competitors` | `h.List` |
| `POST` | `/api/v1/competitors` | `h.Create` |
| `DELETE` | `/api/v1/competitors/:id` | `h.Delete` |
| `GET` | `/api/v1/competitors/:id` | `h.Get` |
| `PUT` | `/api/v1/competitors/:id` | `h.Update` |
| `GET` | `/api/v1/competitors/:id/prices` | `h.ListPrices` |
| `POST` | `/api/v1/competitors/:id/prices` | `h.RecordPrice` |
| `GET` | `/api/v1/competitors/:id/trend` | `h.GetPriceTrend` |

## completeness

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/candidates/:id/completeness` | `h.CheckEnhanced` |
| `POST` | `/api/v1/completeness/check/:productId` | `h.Check` |
| `GET` | `/api/v1/completeness/checks` | `h.ListChecks` |

## compliance

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/compliance/check` | `h.Check` |
| `GET` | `/api/v1/compliance/results` | `h.ListResults` |
| `GET` | `/api/v1/compliance/results/:id` | `h.GetResult` |
| `PUT` | `/api/v1/compliance/results/:id/suppress` | `h.SuppressResult` |
| `POST` | `/api/v1/compliance/scan` | `h.Scan` |

## consolidation

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/consolidation/groups` | `h.ListGroups` |
| `POST` | `/api/v1/consolidation/groups` | `h.CreateGroup` |
| `GET` | `/api/v1/consolidation/groups/:groupId` | `h.GetGroup` |
| `GET` | `/api/v1/consolidation/groups/:groupId/items` | `h.GetGroupItems` |
| `POST` | `/api/v1/consolidation/groups/:groupId/items` | `h.AddItem` |
| `DELETE` | `/api/v1/consolidation/groups/:groupId/items/:itemId` | `h.RemoveItem` |
| `POST` | `/api/v1/consolidation/groups/:groupId/negotiate` | `h.NegotiateGroup` |

## content

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/content/generate` | `h.GenerateContent` |
| `POST` | `/api/v1/content/validate` | `h.ValidateContent` |

## cost

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/cost/dashboard` | `h.Dashboard` |

## dashboard

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/dashboard/brief` | `h.DailyBrief` |
| `GET` | `/api/v1/dashboard/exceptions` | `h.Exceptions` |
| `GET` | `/api/v1/dashboard/inventory` | `h.Inventory` |
| `GET` | `/api/v1/dashboard/orders` | `h.Orders` |
| `GET` | `/api/v1/dashboard/overview` | `h.Overview` |
| `GET` | `/api/v1/dashboard/rejection-reasons` | `h.RejectionReasons` |

## decision

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/decision` | `h.List` |
| `POST` | `/api/v1/decision` | `h.Create` |
| `DELETE` | `/api/v1/decision/:id` | `h.Delete` |
| `GET` | `/api/v1/decision/:id` | `h.Get` |
| `PUT` | `/api/v1/decision/:id` | `h.Update` |
| `POST` | `/api/v1/decision/:id/approve` | `h.Approve` |
| `POST` | `/api/v1/decision/:id/reject` | `h.Reject` |
| `GET` | `/api/v1/decision/summary` | `h.Summary` |

## demandcase

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/demand-cases` | `h.List` |
| `POST` | `/api/v1/demand-cases` | `h.Create` |
| `GET` | `/api/v1/demand-cases/:id` | `h.Get` |
| `GET` | `/api/v1/demand-cases/:id/decision-card` | `h.DecisionCard` |
| `POST` | `/api/v1/demand-cases/:id/evaluate` | `h.Evaluate` |
| `POST` | `/api/v1/demand-cases/:id/evidence` | `h.AddEvidence` |
| `POST` | `/api/v1/demand-cases/:id/falsifications` | `h.AddFalsification` |
| `POST` | `/api/v1/demand-cases/research/first-public-batch` | `h.RunFirstBatch` |
| `POST` | `/api/v1/demand-cases/research/import` | `h.ImportResearch` |

## entropy

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/entropy` | `h.GetSummary` |
| `GET` | `/api/v1/entropy/changelog` | `h.GetChangeLog` |
| `POST` | `/api/v1/entropy/defense` | `h.RunDefenses` |
| `GET` | `/api/v1/entropy/health` | `h.GetHealthScores` |
| `GET` | `/api/v1/entropy/spc` | `h.GetSpcStatus` |

## evolution

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/evolution/nudges` | `h.ListNudges` |
| `POST` | `/api/v1/evolution/nudges/:id/accept` | `h.AcceptNudge` |
| `POST` | `/api/v1/evolution/nudges/:id/dismiss` | `h.DismissNudge` |
| `POST` | `/api/v1/evolution/nudges/evaluate` | `h.EvaluateNudges` |

## exceptions

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/exceptions` | `h.List` |
| `POST` | `/api/v1/exceptions` | `h.Create` |
| `DELETE` | `/api/v1/exceptions/:id` | `h.Delete` |
| `GET` | `/api/v1/exceptions/:id` | `h.Get` |
| `PUT` | `/api/v1/exceptions/:id` | `h.Update` |
| `PUT` | `/api/v1/exceptions/:id/assign` | `h.Assign` |
| `POST` | `/api/v1/exceptions/:id/resolve` | `h.OwnerResolve` |
| `PUT` | `/api/v1/exceptions/:id/resolve` | `h.Resolve` |
| `POST` | `/api/v1/exceptions/:id/suggest` | `h.Suggest` |
| `POST` | `/api/v1/exceptions/auto-detect` | `h.AutoDetect` |

## exchangerate

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/exchange-rates` | `h.List` |
| `POST` | `/api/v1/exchange-rates` | `h.Create` |
| `PUT` | `/api/v1/exchange-rates/:from_currency/:to_currency` | `h.UpdateByPair` |
| `GET` | `/api/v1/exchange-rates/:from_currency/:to_currency/latest` | `h.GetLatest` |
| `DELETE` | `/api/v1/exchange-rates/:id` | `h.Delete` |

## experiment

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/experiments` | `h.List` |
| `POST` | `/api/v1/experiments` | `h.Create` |
| `GET` | `/api/v1/experiments/:experimentId` | `h.Get` |
| `PUT` | `/api/v1/experiments/:experimentId` | `h.Update` |
| `POST` | `/api/v1/experiments/:experimentId/evidence` | `h.AddEvidence` |
| `POST` | `/api/v1/experiments/:experimentId/evidence/:evidenceId/verify` | `h.VerifyEvidence` |
| `POST` | `/api/v1/experiments/:experimentId/gates/evaluate` | `h.EvaluateGate` |
| `POST` | `/api/v1/experiments/:experimentId/links` | `h.AddObjectLink` |
| `GET` | `/api/v1/experiments/:experimentId/owner-summary` | `h.OwnerSummary` |

## finance

**Permission:** `finance.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/finance/accounts` | `h.ListAccounts` |
| `POST` | `/api/v1/finance/accounts` | `h.CreateAccount` |
| `DELETE` | `/api/v1/finance/accounts/:id` | `h.DeleteAccount` |
| `GET` | `/api/v1/finance/accounts/:id` | `h.GetAccount` |
| `PUT` | `/api/v1/finance/accounts/:id` | `h.UpdateAccount` |
| `GET` | `/api/v1/finance/ledger` | `h.ListLedger` |
| `POST` | `/api/v1/finance/mock` | `h.Mock` |
| `GET` | `/api/v1/finance/orders/:order_id/ledger` | `h.ListOrderLedger` |
| `POST` | `/api/v1/finance/orders/:order_id/ledger/rebuild` | `h.RebuildOrderLedger` |
| `GET` | `/api/v1/finance/orders/:order_id/profit` | `h.OrderProfit` |
| `GET` | `/api/v1/finance/profit-summary` | `h.ProfitSummary` |
| `POST` | `/api/v1/finance/profit/batch-calculate` | `h.BatchCalculateProfit` |
| `POST` | `/api/v1/finance/profit/calculate` | `h.CalculateProfit` |
| `GET` | `/api/v1/finance/profit/ranking` | `h.GetSKUProfitRanking` |
| `GET` | `/api/v1/finance/profit/summary` | `h.GetProfitSummary` |
| `GET` | `/api/v1/finance/summary` | `h.Summary` |
| `GET` | `/api/v1/finance/transactions` | `h.ListTransactions` |
| `POST` | `/api/v1/finance/transactions` | `h.CreateTransaction` |

## imagegen

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/image-gen` | `h.ListImageGens` |
| `POST` | `/api/v1/image-gen` | `h.CreateImageGen` |
| `DELETE` | `/api/v1/image-gen/:id` | `h.DeleteImageGen` |
| `GET` | `/api/v1/image-gen/:id` | `h.GetImageGen` |
| `PUT` | `/api/v1/image-gen/:id/status` | `h.UpdateImageGenStatus` |
| `GET` | `/api/v1/image-gen/canvas` | `h.ListCanvases` |
| `POST` | `/api/v1/image-gen/canvas` | `h.CreateCanvas` |
| `DELETE` | `/api/v1/image-gen/canvas/:id` | `h.DeleteCanvas` |
| `GET` | `/api/v1/image-gen/canvas/:id` | `h.GetCanvas` |
| `PUT` | `/api/v1/image-gen/canvas/:id` | `h.UpdateCanvas` |
| `GET` | `/api/v1/image-gen/templates` | `h.ListTemplates` |
| `POST` | `/api/v1/image-gen/templates` | `h.CreateTemplate` |
| `DELETE` | `/api/v1/image-gen/templates/:id` | `h.DeleteTemplate` |
| `GET` | `/api/v1/image-gen/templates/:id` | `h.GetTemplate` |
| `PUT` | `/api/v1/image-gen/templates/:id` | `h.UpdateTemplate` |
| `POST` | `/api/v1/image-gen/templates/:id/use` | `h.UseTemplate` |

## importbatch

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/import-batch` | `h.ListBatches` |
| `POST` | `/api/v1/import-batch` | `h.CreateBatch` |
| `DELETE` | `/api/v1/import-batch/:id` | `h.DeleteBatch` |
| `GET` | `/api/v1/import-batch/:id` | `h.GetBatch` |
| `PUT` | `/api/v1/import-batch/:id` | `h.UpdateBatch` |
| `GET` | `/api/v1/import-batch/:id/rows` | `h.ListRows` |
| `POST` | `/api/v1/import-batch/upload` | `h.Upload` |

## integrations

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/platform-integrations` | `h.List` |
| `POST` | `/api/v1/platform-integrations` | `h.Create` |
| `DELETE` | `/api/v1/platform-integrations/:id` | `h.Delete` |
| `GET` | `/api/v1/platform-integrations/:id` | `h.Get` |
| `PUT` | `/api/v1/platform-integrations/:id` | `h.Update` |
| `GET` | `/api/v1/platform-integrations/:id/attributes` | `h.ListAttributes` |
| `POST` | `/api/v1/platform-integrations/:id/attributes` | `h.CreateAttribute` |
| `GET` | `/api/v1/platform-integrations/:id/categories` | `h.ListCategories` |
| `POST` | `/api/v1/platform-integrations/:id/categories` | `h.CreateCategory` |
| `GET` | `/api/v1/platform-integrations/:id/mode` | `h.GetMode` |
| `PUT` | `/api/v1/platform-integrations/:id/mode` | `h.UpdateMode` |
| `GET` | `/api/v1/platform-integrations/:id/ozon-products` | `h.ListOzonProducts` |
| `POST` | `/api/v1/platform-integrations/:id/sync` | `h.Sync` |
| `POST` | `/api/v1/platform-integrations/:id/test` | `h.TestConnection` |
| `POST` | `/api/v1/platform-integrations/mock/seed` | `` |
| `POST` | `/api/v1/platform-integrations/publish-to-ozon` | `h.PublishToOzon` |
| `POST` | `/api/v1/platform-integrations/write-back` | `h.WriteBack` |
| `POST` | `/api/v1/platform-integrations/write-back/:ref-id/retry` | `h.RetryWriteBack` |

## inventory

**Permission:** `inventory.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/inventory` | `h.List` |
| `GET` | `/api/v1/inventory/:id` | `h.Get` |
| `PUT` | `/api/v1/inventory/:id` | `h.Update` |
| `POST` | `/api/v1/inventory/:id/lock` | `h.Lock` |
| `POST` | `/api/v1/inventory/:id/unlock` | `h.Unlock` |
| `GET` | `/api/v1/inventory/allocate/:sku_id` | `h.AllocateStock` |
| `POST` | `/api/v1/inventory/dead-stock/analyze` | `h.IdentifyDeadStock` |
| `GET` | `/api/v1/inventory/dead-stock/logs` | `h.ListDeadStockLogs` |
| `GET` | `/api/v1/inventory/locations` | `h.ListLocations` |
| `GET` | `/api/v1/inventory/logs` | `h.ListLogs` |
| `GET` | `/api/v1/inventory/oversell-report` | `h.OversellReport` |
| `GET` | `/api/v1/inventory/safety-config/:sku_id` | `h.GetSafetyConfig` |
| `PUT` | `/api/v1/inventory/safety-config/:sku_id` | `h.UpsertSafetyConfig` |
| `GET` | `/api/v1/inventory/safety-configs` | `h.ListSafetyConfigs` |
| `GET` | `/api/v1/inventory/sku/:sku_id/warehouses` | `h.ListInventoryBySku` |
| `POST` | `/api/v1/inventory/sync-cross-platform/:productId` | `h.SyncCrossPlatform` |
| `GET` | `/api/v1/inventory/transfers` | `h.ListTransfers` |

## landedcost

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/landed-cost/:productId` | `h.GetLandedCost` |
| `GET` | `/api/v1/landed-cost/:productId/compare` | `h.CompareAcrossPlatforms` |
| `POST` | `/api/v1/landed-cost/calculate` | `h.Calculate` |

## listing

**Permission:** `listing.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/listing` | `h.Create` |
| `POST` | `/api/v1/listing/listing-tasks/:task_id/cancel` | `h.CancelTask` |
| `POST` | `/api/v1/listing/listing-tasks/:task_id/publish` | `h.PublishTask` |
| `POST` | `/api/v1/listing/listing-tasks/:task_id/recheck` | `h.RecheckTask` |
| `POST` | `/api/v1/listing/listing-tasks/from-decisions` | `h.CreateTasksFromDecisions` |
| `GET` | `/api/v1/listing/products/:product_id/listings` | `h.ListByProduct` |
| `GET` | `/api/v1/listing/products/:product_id/platform-comparison` | `h.GetPlatformComparison` |
| `POST` | `/api/v1/listing/products/:product_id/publish/:platform_id` | `h.PublishProduct` |
| `GET` | `/api/v1/listings` | `h.List` |
| `POST` | `/api/v1/listings` | `h.Create` |
| `DELETE` | `/api/v1/listings/:id` | `h.Delete` |
| `GET` | `/api/v1/listings/:id` | `h.Get` |
| `PUT` | `/api/v1/listings/:id` | `h.Update` |
| `POST` | `/api/v1/listings/:id/publish` | `h.Publish` |
| `POST` | `/api/v1/listings/:id/sync` | `h.Sync` |
| `POST` | `/api/v1/listings/suggest` | `h.Suggest` |

## listingtask

**Permission:** `listing.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/listing-task/:task_id/execute` | `h.Execute` |
| `POST` | `/api/v1/listing-task/:task_id/feedback` | `h.Feedback` |
| `POST` | `/api/v1/listing-task/:task_id/items/:item_id/retry` | `h.RetryItem` |
| `POST` | `/api/v1/listing-task/:task_id/retry-failed` | `h.RetryFailed` |
| `POST` | `/api/v1/listing-task/retry-all` | `h.RetryAll` |
| `GET` | `/api/v1/listing-task/stats` | `h.ListStats` |
| `GET` | `/api/v1/listing-tasks` | `h.List` |
| `POST` | `/api/v1/listing-tasks` | `h.Create` |
| `DELETE` | `/api/v1/listing-tasks/:id` | `h.Delete` |
| `GET` | `/api/v1/listing-tasks/:id` | `h.Get` |
| `PUT` | `/api/v1/listing-tasks/:id` | `h.Update` |
| `GET` | `/api/v1/listing-tasks/:id/items` | `h.ListItems` |
| `POST` | `/api/v1/listing-tasks/:id/items` | `h.CreateItem` |
| `DELETE` | `/api/v1/listing-tasks/:id/items/:item_id` | `h.DeleteItem` |
| `PUT` | `/api/v1/listing-tasks/:id/items/:item_id` | `h.UpdateItem` |
| `GET` | `/api/v1/listing-tasks/:id/review` | `h.Review` |
| `POST` | `/api/v1/listing-tasks/from-suggestion` | `h.CreateFromSuggestion` |

## logistics

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/logistics/quote` | `h.GetQuotes` |

## loop

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/loop/batch-evaluate` | `h.BatchEvaluate` |
| `POST` | `/api/v1/loop/evaluate/:productId` | `h.Evaluate` |
| `GET` | `/api/v1/loop/recommendations` | `h.GetRecommendations` |

## metabolism

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/metabolism` | `h.ListLogs` |
| `GET` | `/api/v1/metabolism/:id` | `h.GetLog` |
| `POST` | `/api/v1/metabolism/dry-run` | `h.DryRun` |
| `GET` | `/api/v1/metabolism/excretion-result` | `h.GetExcretionResult` |
| `POST` | `/api/v1/metabolism/execute` | `h.ExecuteEntities` |

## mock

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/mock/orders` | `h.ListOrders` |
| `POST` | `/api/v1/mock/seed` | `h.Seed` |
| `GET` | `/api/v1/mock/settlements` | `h.ListSettlements` |
| `GET` | `/api/v1/mock/sync-statuses` | `h.SyncStatuses` |

## notification

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/notification` | `handler.List` |
| `POST` | `/api/v1/notification` | `handler.Create` |
| `DELETE` | `/api/v1/notification/:id` | `handler.Delete` |
| `GET` | `/api/v1/notification/:id` | `handler.Get` |
| `PUT` | `/api/v1/notification/:id/read` | `handler.MarkAsRead` |
| `GET` | `/api/v1/notification/alert-rules` | `handler.ListAlertRules` |
| `POST` | `/api/v1/notification/alert-rules` | `handler.CreateAlertRule` |
| `DELETE` | `/api/v1/notification/alert-rules/:id` | `handler.DeleteAlertRule` |
| `PUT` | `/api/v1/notification/alert-rules/:id` | `handler.UpdateAlertRule` |
| `PUT` | `/api/v1/notification/read-all` | `handler.MarkAllRead` |
| `GET` | `/api/v1/notification/unread-count` | `handler.UnreadCount` |

## operationlog

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/operation-log` | `h.List` |
| `POST` | `/api/v1/operation-log` | `h.Create` |
| `GET` | `/api/v1/operation-log/:id` | `h.Get` |

## orchestration

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/orchestration/pipeline/config` | `h.ListConfigs` |
| `POST` | `/api/v1/orchestration/pipeline/config` | `h.CreateConfig` |
| `GET` | `/api/v1/orchestration/products/:id/pipeline` | `h.GetPipelineStatus` |
| `POST` | `/api/v1/orchestration/products/:id/pipeline/start` | `h.StartPipeline` |
| `POST` | `/api/v1/orchestration/products/:id/pipeline/step/:step/retry` | `h.RetryStep` |

## order

**Permission:** `order.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/order` | `h.List` |
| `POST` | `/api/v1/order` | `h.Create` |
| `DELETE` | `/api/v1/order/:id` | `h.Delete` |
| `GET` | `/api/v1/order/:id` | `h.Get` |
| `PUT` | `/api/v1/order/:id` | `h.Update` |
| `POST` | `/api/v1/order/:id/status` | `h.UpdateStatus` |
| `GET` | `/api/v1/order/summary` | `h.Summary` |

## orderimport

**Permission:** `order.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/order-import` | `h.List` |
| `POST` | `/api/v1/order-import` | `h.Create` |
| `DELETE` | `/api/v1/order-import/:id` | `h.Delete` |
| `GET` | `/api/v1/order-import/:id` | `h.Get` |
| `PUT` | `/api/v1/order-import/:id` | `h.Update` |
| `POST` | `/api/v1/order-import/:id/complete` | `h.Complete` |
| `POST` | `/api/v1/order-import/:id/process` | `h.Process` |
| `GET` | `/api/v1/order-import/summary` | `h.Summary` |

## owner

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/owner/agent-activity` | `h.AgentActivity` |
| `GET` | `/api/v1/owner/decision-queue` | `h.GetDecisionQueue` |
| `GET` | `/api/v1/owner/pipeline-chain` | `h.PipelineChain` |
| `GET` | `/api/v1/owner/platform-sync` | `h.PlatformSyncStatus` |
| `GET` | `/api/v1/owner/risk-summary` | `h.RiskSummary` |
| `GET` | `/api/v1/owner/suggestions` | `h.Suggestions` |
| `POST` | `/api/v1/owner/suggestions/:id/feedback` | `h.Feedback` |

## personalrule

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/agents/rules` | `h.ListRules` |
| `POST` | `/api/v1/agents/rules` | `h.CreateRule` |
| `DELETE` | `/api/v1/agents/rules/:id` | `h.DeleteRule` |
| `GET` | `/api/v1/agents/rules/:id` | `h.GetRule` |
| `PUT` | `/api/v1/agents/rules/:id` | `h.UpdateRule` |
| `POST` | `/api/v1/agents/rules/apply` | `h.ApplyRules` |

## platform

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/platforms` | `h.ListPlatforms` |
| `POST` | `/api/v1/platforms` | `h.CreatePlatform` |
| `DELETE` | `/api/v1/platforms/:id` | `h.DeletePlatform` |
| `GET` | `/api/v1/platforms/:id` | `h.GetPlatform` |
| `PUT` | `/api/v1/platforms/:id` | `h.UpdatePlatform` |
| `GET` | `/api/v1/stores` | `h.ListStores` |
| `POST` | `/api/v1/stores` | `h.CreateStore` |
| `DELETE` | `/api/v1/stores/:id` | `h.DeleteStore` |
| `GET` | `/api/v1/stores/:id` | `h.GetStore` |
| `PUT` | `/api/v1/stores/:id` | `h.UpdateStore` |

## platformfee

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/platform-fee` | `h.List` |
| `POST` | `/api/v1/platform-fee` | `h.Create` |
| `DELETE` | `/api/v1/platform-fee/:id` | `h.Delete` |
| `GET` | `/api/v1/platform-fee/:id` | `h.Get` |
| `PUT` | `/api/v1/platform-fee/:id` | `h.Update` |
| `POST` | `/api/v1/platform-fee/calculate` | `h.Calculate` |

## price

**Permission:** `finance.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/competitor-prices` | `h.ListCompetitorPrices` |
| `POST` | `/api/v1/competitor-prices` | `h.CreateCompetitorPrice` |
| `DELETE` | `/api/v1/competitor-prices/:id` | `h.DeleteCompetitorPrice` |
| `GET` | `/api/v1/competitor-prices/:id` | `h.GetCompetitorPrice` |
| `GET` | `/api/v1/prices` | `h.ListPrices` |
| `POST` | `/api/v1/prices` | `h.SetPrice` |
| `DELETE` | `/api/v1/prices/:id` | `h.DeletePrice` |
| `GET` | `/api/v1/prices/:id` | `h.GetPrice` |
| `PUT` | `/api/v1/prices/:id` | `h.UpdatePrice` |
| `GET` | `/api/v1/pricing-recommendations` | `h.ListRecommendations` |
| `POST` | `/api/v1/pricing-recommendations/:id/apply` | `h.ApplyRecommendation` |
| `POST` | `/api/v1/pricing-recommendations/generate` | `h.GenerateRecommendation` |
| `GET` | `/api/v1/pricing-strategies` | `h.ListPricingStrategies` |
| `POST` | `/api/v1/pricing-strategies` | `h.SavePricingStrategy` |
| `DELETE` | `/api/v1/pricing-strategies/:id` | `h.DeletePricingStrategy` |
| `GET` | `/api/v1/pricing-strategies/:id` | `h.GetPricingStrategy` |
| `PUT` | `/api/v1/pricing-strategies/:id` | `h.UpdatePricingStrategy` |
| `GET` | `/api/v1/skus/:id/current-price` | `h.GetCurrentPrice` |
| `GET` | `/api/v1/skus/:id/price-history` | `h.PriceHistory` |
| `GET` | `/api/v1/skus/:id/prices` | `h.ListPricesBySKU` |

## productanalysis

**Prefix:** ``

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/analyses` | `h.ListAnalyses` |
| `GET` | `/analyses/:id` | `h.GetAnalysis` |
| `POST` | `/analyses/:id/feedback` | `h.RecordFeedback` |
| `POST` | `/analyze` | `h.Analyze` |
| `POST` | `/trigger-prism` | `h.TriggerPrism` |

## producthub

**Permission:** `product.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/product-hub` | `masterH.List` |
| `POST` | `/api/v1/product-hub` | `masterH.Create` |
| `DELETE` | `/api/v1/product-hub/:id` | `masterH.Delete` |
| `GET` | `/api/v1/product-hub/:id` | `masterH.Get` |
| `PUT` | `/api/v1/product-hub/:id` | `masterH.Update` |
| `GET` | `/api/v1/product-hub/:id/costs` | `h.ListCosts` |
| `GET` | `/api/v1/product-hub/:id/evidence` | `h.GetEvidence` |
| `GET` | `/api/v1/product-hub/:id/hub` | `hubH.GetHub` |
| `GET` | `/api/v1/product-hub/:id/offers` | `h.ListOffers` |
| `GET` | `/api/v1/product-hub/:id/samples` | `h.ListSamples` |
| `POST` | `/api/v1/product-hub/:id/transition` | `masterH.TransitionLifecycle` |
| `GET` | `/api/v1/product-hub/:id/variants` | `h.ListVariants` |
| `POST` | `/api/v1/product-hub/costs` | `h.CreateCost` |
| `POST` | `/api/v1/product-hub/costs/:costId/confirm` | `h.ConfirmCost` |
| `POST` | `/api/v1/product-hub/offers` | `h.CreateOffer` |
| `POST` | `/api/v1/product-hub/samples` | `h.CreateSample` |
| `POST` | `/api/v1/product-hub/variants` | `h.CreateVariant` |
| `GET` | `/api/v1/products/360/summary` | `h.GetProductSummary` |
| `POST` | `/api/v1/products/:id/decisions` | `h.RecordDecision` |
| `POST` | `/api/v1/products/:id/discover-relations` | `h.AutoDiscoverRelations` |
| `GET` | `/api/v1/products/:id/freshness` | `h.GetProductFreshness` |
| `POST` | `/api/v1/products/:id/freshness/verify` | `h.VerifyDimension` |
| `GET` | `/api/v1/products/:id/relations` | `h.GetRelatedProducts` |
| `GET` | `/api/v1/products/:id/versions` | `h.ListVersions` |
| `GET` | `/api/v1/products/:id/versions/:versionId` | `h.GetVersion` |
| `POST` | `/api/v1/products/:id/versions/:versionId/rollback` | `h.Rollback` |
| `GET` | `/api/v1/products/decision` | `h.ListRecentDecisions` |
| `GET` | `/api/v1/products/freshness/stale` | `h.ListStaleProducts` |
| `POST` | `/api/v1/products/relations` | `h.CreateRelation` |
| `DELETE` | `/api/v1/products/relations/:id` | `h.DeleteRelation` |

## profit

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/profit/order/:orderId/calculate` | `h.CalculateOrderProfit` |
| `GET` | `/api/v1/profit/summaries` | `h.ListSummaries` |
| `GET` | `/api/v1/profit/summary/:productId` | `h.Summary` |

## purchase

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/purchase/orders` | `h.ListOrders` |
| `POST` | `/api/v1/purchase/orders` | `h.CreateOrder` |
| `GET` | `/api/v1/purchase/orders/:id` | `h.GetOrder` |
| `POST` | `/api/v1/purchase/orders/:id/approve` | `h.ApproveOrder` |
| `POST` | `/api/v1/purchase/orders/:id/cancel` | `h.CancelOrder` |
| `POST` | `/api/v1/purchase/orders/:id/receive` | `h.ReceiveOrder` |
| `GET` | `/api/v1/purchase/suggestions` | `h.ListSuggestions` |
| `POST` | `/api/v1/purchase/suggestions/generate` | `h.GenerateSuggestions` |

## reliability

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/reliability/budget` | `h.GetBudget` |
| `PUT` | `/api/v1/reliability/budget` | `h.SetBudget` |

## report

**Permission:** `report.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/report/daily` | `h.DailyReport` |
| `GET` | `/api/v1/report/inventory` | `h.Inventory` |
| `GET` | `/api/v1/report/platform-fee` | `h.PlatformFee` |
| `GET` | `/api/v1/report/profit` | `h.Profit` |
| `GET` | `/api/v1/report/sales` | `h.Sales` |
| `GET` | `/api/v1/report/settlement` | `h.Settlement` |
| `GET` | `/api/v1/report/weekly` | `h.WeeklyReport` |

## search

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/search` | `h.Search` |
| `GET` | `/api/v1/search/recent` | `h.Recent` |

## sentiment

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/sentiment/:productId` | `h.GetProductSentiment` |
| `POST` | `/api/v1/sentiment/:productId/refresh` | `h.RefreshSentiment` |
| `GET` | `/api/v1/sentiment/negative` | `h.ListNegativeSentiment` |

## settings

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/settings/llm` | `h.GetLLM` |
| `PUT` | `/api/v1/settings/llm` | `h.UpdateLLM` |

## settlement

**Permission:** `settlement.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/settlement` | `h.List` |
| `POST` | `/api/v1/settlement` | `h.Create` |
| `DELETE` | `/api/v1/settlement/:id` | `h.Delete` |
| `GET` | `/api/v1/settlement/:id` | `h.Get` |
| `PUT` | `/api/v1/settlement/:id` | `h.Update` |
| `GET` | `/api/v1/settlement/:id/items` | `h.ListItems` |
| `POST` | `/api/v1/settlement/:id/items` | `h.AddItem` |
| `POST` | `/api/v1/settlement/:id/reconcile` | `h.Reconcile` |
| `PUT` | `/api/v1/settlement/items/:item_id/reconciliation` | `h.UpdateItemReconciliation` |
| `POST` | `/api/v1/settlement/recalculate` | `h.RecalculateAll` |
| `GET` | `/api/v1/settlement/summary` | `h.Summary` |

## shipping

**Permission:** `shipping.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/shipping/bill-batches` | `h.ListBillBatches` |
| `POST` | `/api/v1/shipping/bill-batches` | `h.CreateBillBatch` |
| `DELETE` | `/api/v1/shipping/bill-batches/:id` | `h.DeleteBillBatch` |
| `GET` | `/api/v1/shipping/bill-batches/:id` | `h.GetBillBatch` |
| `GET` | `/api/v1/shipping/bill-batches/:id/anomalies` | `h.ListBillAnomalies` |
| `GET` | `/api/v1/shipping/bill-batches/:id/items` | `h.ListBillItems` |
| `POST` | `/api/v1/shipping/bill-batches/:id/reconcile` | `h.ReconcileBillBatch` |
| `POST` | `/api/v1/shipping/bill-batches/import` | `h.ImportBill` |
| `PUT` | `/api/v1/shipping/bill-items/:id/review` | `h.ReviewBillItem` |
| `GET` | `/api/v1/shipping/carrier-performance` | `h.GetCarrierPerformance` |
| `GET` | `/api/v1/shipping/carriers` | `h.ListCarriers` |
| `POST` | `/api/v1/shipping/carriers/:code/quote` | `h.CarrierQuote` |
| `GET` | `/api/v1/shipping/channels` | `h.ListChannels` |
| `POST` | `/api/v1/shipping/channels` | `h.CreateChannel` |
| `DELETE` | `/api/v1/shipping/channels/:id` | `h.DeleteChannel` |
| `GET` | `/api/v1/shipping/channels/:id` | `h.GetChannel` |
| `PUT` | `/api/v1/shipping/channels/:id` | `h.UpdateChannel` |
| `GET` | `/api/v1/shipping/providers` | `h.ListProviders` |
| `POST` | `/api/v1/shipping/providers` | `h.CreateProvider` |
| `DELETE` | `/api/v1/shipping/providers/:id` | `h.DeleteProvider` |
| `GET` | `/api/v1/shipping/providers/:id` | `h.GetProvider` |
| `PUT` | `/api/v1/shipping/providers/:id` | `h.UpdateProvider` |
| `POST` | `/api/v1/shipping/quote` | `h.Quote` |
| `POST` | `/api/v1/shipping/quote-unified` | `h.QuoteUnified` |
| `GET` | `/api/v1/shipping/rules` | `h.ListRules` |
| `POST` | `/api/v1/shipping/rules` | `h.CreateRule` |
| `DELETE` | `/api/v1/shipping/rules/:id` | `h.DeleteRule` |
| `GET` | `/api/v1/shipping/rules/:id/versions` | `h.ListRuleVersions` |
| `GET` | `/api/v1/shipping/snapshots` | `h.ListSnapshots` |
| `POST` | `/api/v1/shipping/snapshots` | `h.CreateSnapshot` |
| `GET` | `/api/v1/shipping/snapshots/:orderId` | `h.GetSnapshot` |
| `GET` | `/api/v1/shipping/tracking` | `h.ListTracking` |
| `POST` | `/api/v1/shipping/tracking` | `h.CreateTracking` |
| `PUT` | `/api/v1/shipping/tracking/:id/event` | `h.UpdateTrackingEvent` |
| `PUT` | `/api/v1/shipping/tracking/:id/exception` | `h.MarkTrackingException` |
| `GET` | `/api/v1/shipping/tracking/:orderId` | `h.GetTracking` |
| `GET` | `/api/v1/shipping/zones` | `h.ListZones` |
| `POST` | `/api/v1/shipping/zones` | `h.CreateZone` |
| `DELETE` | `/api/v1/shipping/zones/:id` | `h.DeleteZone` |

## sku

**Permission:** `product.read`

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/product-master` | `h.ListProducts` |
| `POST` | `/api/v1/product-master` | `h.CreateProduct` |
| `DELETE` | `/api/v1/product-master/:id` | `h.DeleteProduct` |
| `GET` | `/api/v1/product-master/:id` | `h.GetProduct` |
| `PUT` | `/api/v1/product-master/:id` | `h.UpdateProduct` |
| `GET` | `/api/v1/product-master/:id/skus` | `h.ListSkusByProduct` |
| `GET` | `/api/v1/product-master/:id/specs` | `h.ListSpecs` |
| `POST` | `/api/v1/product-master/:id/specs` | `h.CreateSpec` |
| `DELETE` | `/api/v1/product-master/:id/specs/:spec_id` | `h.DeleteSpec` |
| `PUT` | `/api/v1/product-master/:id/specs/:spec_id` | `h.UpdateSpec` |
| `POST` | `/api/v1/product-master/:id/specs/:spec_id/values` | `h.CreateSpecValue` |
| `GET` | `/api/v1/skus` | `h.ListSkus` |
| `POST` | `/api/v1/skus` | `h.CreateSku` |
| `DELETE` | `/api/v1/skus/:id` | `h.DeleteSku` |
| `GET` | `/api/v1/skus/:id` | `h.GetSku` |
| `PUT` | `/api/v1/skus/:id` | `h.UpdateSku` |
| `DELETE` | `/api/v1/spec-values/:id` | `h.DeleteSpecValue` |
| `PUT` | `/api/v1/spec-values/:id` | `h.UpdateSpecValue` |

## sourcing

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/sourcing/fetch` | `h.Fetch` |
| `GET` | `/api/v1/sourcing/keyword-trends` | `h.KeywordTrends` |
| `GET` | `/api/v1/sourcing/market-overview` | `h.MarketOverview` |
| `GET` | `/api/v1/sourcing/market-trends` | `h.MarketTrends` |
| `GET` | `/api/v1/sourcing/recommendations` | `h.ListRecommendations` |

## sourcing1688

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/sourcing-1688` | `h.List` |
| `POST` | `/api/v1/sourcing-1688` | `h.Create` |
| `DELETE` | `/api/v1/sourcing-1688/:id` | `h.Delete` |
| `GET` | `/api/v1/sourcing-1688/:id` | `h.Get` |
| `PUT` | `/api/v1/sourcing-1688/:id` | `h.Update` |
| `POST` | `/api/v1/sourcing-1688/:id/import` | `h.Import` |
| `POST` | `/api/v1/sourcing-1688/:id/reject` | `h.Reject` |
| `GET` | `/api/v1/sourcing-1688/summary` | `h.Summary` |

## supplier

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/product-suppliers` | `h.ListProductSuppliers` |
| `POST` | `/api/v1/product-suppliers` | `h.CreateProductSupplier` |
| `DELETE` | `/api/v1/product-suppliers/:id` | `h.DeleteProductSupplier` |
| `PUT` | `/api/v1/product-suppliers/:id` | `h.UpdateProductSupplier` |
| `GET` | `/api/v1/product-suppliers/comparison` | `h.GetSupplierComparison` |
| `GET` | `/api/v1/suppliers` | `h.List` |
| `POST` | `/api/v1/suppliers` | `h.Create` |
| `DELETE` | `/api/v1/suppliers/:id` | `h.Delete` |
| `GET` | `/api/v1/suppliers/:id` | `h.Get` |
| `PUT` | `/api/v1/suppliers/:id` | `h.Update` |
| `PUT` | `/api/v1/suppliers/:id/kpi-score` | `h.UpdateScoreManual` |
| `POST` | `/api/v1/suppliers/:id/recalculate` | `h.RecalculateScore` |
| `GET` | `/api/v1/suppliers/:id/score` | `h.GetScore` |
| `GET` | `/api/v1/suppliers/:id/score-history` | `h.GetScoreHistory` |
| `POST` | `/api/v1/suppliers/:id/score-snapshot` | `h.RecordScoreSnapshot` |
| `GET` | `/api/v1/suppliers/scoreboard` | `h.ListScoreboard` |

## supplychain

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/supply-chain/flows` | `h.List` |
| `POST` | `/api/v1/supply-chain/flows` | `h.Create` |
| `GET` | `/api/v1/supply-chain/flows/:id` | `h.Get` |
| `PUT` | `/api/v1/supply-chain/flows/:id` | `h.Update` |
| `GET` | `/api/v1/supply-chain/flows/:id/events` | `h.GetEvents` |
| `GET` | `/api/v1/supply-chain/tracking` | `th.List` |
| `POST` | `/api/v1/supply-chain/tracking` | `th.Create` |
| `GET` | `/api/v1/supply-chain/tracking/:id` | `th.Get` |
| `PUT` | `/api/v1/supply-chain/tracking/:id/status` | `th.UpdateStatus` |
| `POST` | `/api/v1/supply-chain/tracking/:id/sync` | `th.SyncFromCarrier` |
| `GET` | `/api/v1/supply-chain/tracking/flow/:flowId` | `th.GetByFlow` |

## support

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/support/blacklist` | `h.ListBlacklist` |
| `POST` | `/api/v1/support/blacklist` | `h.AddBlacklist` |
| `DELETE` | `/api/v1/support/blacklist/:id` | `h.DeleteBlacklist` |
| `GET` | `/api/v1/support/blacklist/check` | `h.CheckBlacklist` |
| `GET` | `/api/v1/support/conversations` | `h.ListConversations` |
| `POST` | `/api/v1/support/conversations` | `h.CreateConversation` |
| `DELETE` | `/api/v1/support/conversations/:id` | `h.DeleteConversation` |
| `GET` | `/api/v1/support/conversations/:id` | `h.GetConversation` |
| `PUT` | `/api/v1/support/conversations/:id` | `h.UpdateConversation` |
| `POST` | `/api/v1/support/conversations/:id/close` | `h.CloseConversation` |
| `GET` | `/api/v1/support/conversations/:id/messages` | `h.GetMessages` |
| `POST` | `/api/v1/support/conversations/:id/reply` | `h.SendReply` |
| `GET` | `/api/v1/support/templates` | `h.ListTemplates` |
| `POST` | `/api/v1/support/templates` | `h.CreateTemplate` |
| `DELETE` | `/api/v1/support/templates/:id` | `h.DeleteTemplate` |
| `GET` | `/api/v1/support/templates/:id` | `h.GetTemplate` |
| `PUT` | `/api/v1/support/templates/:id` | `h.UpdateTemplate` |

## tariff

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/tariff` | `h.List` |
| `POST` | `/api/v1/tariff` | `h.Create` |
| `DELETE` | `/api/v1/tariff/:id` | `h.Delete` |
| `GET` | `/api/v1/tariff/:id` | `h.Get` |
| `PUT` | `/api/v1/tariff/:id` | `h.Update` |
| `POST` | `/api/v1/tariff/decide` | `h.Decide` |

## trustscore

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/trust-scores` | `h.List` |
| `GET` | `/api/v1/trust-scores/:agent_id` | `h.GetByAgent` |
| `PUT` | `/api/v1/trust-scores/:agent_id/level` | `h.UpdateLevel` |
| `POST` | `/api/v1/trust-scores/auto-upgrade` | `h.AutoUpgrade` |
| `POST` | `/api/v1/trust-scores/eligible` | `h.Eligible` |
| `POST` | `/api/v1/trust-scores/recalculate` | `h.Recalculate` |
| `GET` | `/api/v1/trust-scores/summary` | `h.Summary` |

## workflow

**Prefix:** `/api/v1`

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/workflow/defs` | `h.ListDefs` |
| `POST` | `/api/v1/workflow/defs` | `h.CreateDef` |
| `POST` | `/api/v1/workflow/defs/:defId/start` | `h.StartRun` |
| `DELETE` | `/api/v1/workflow/defs/:id` | `h.DeleteDef` |
| `GET` | `/api/v1/workflow/defs/:id` | `h.GetDef` |
| `PUT` | `/api/v1/workflow/defs/:id` | `h.UpdateDef` |
| `GET` | `/api/v1/workflow/monitor` | `h.GetMonitor` |
| `GET` | `/api/v1/workflow/monitor/stats` | `h.GetMonitorStats` |
| `GET` | `/api/v1/workflow/runs` | `h.ListRuns` |
| `GET` | `/api/v1/workflow/runs/:id` | `h.GetRun` |
| `POST` | `/api/v1/workflow/runs/:id/advance` | `h.AdvanceStep` |
| `POST` | `/api/v1/workflow/runs/:id/pause` | `h.PauseRun` |
| `POST` | `/api/v1/workflow/runs/:id/resume` | `h.ResumeRun` |
| `POST` | `/api/v1/workflow/runs/:id/retry` | `h.RetryRun` |
| `GET` | `/api/v1/workflows` | `h.ListWorkflows` |
| `POST` | `/api/v1/workflows` | `h.CreateWorkflow` |
| `GET` | `/api/v1/workflows/:id` | `h.GetWorkflow` |
| `POST` | `/api/v1/workflows/runs/:id/approve` | `h.ApproveStep` |
| `POST` | `/api/v1/workflows/runs/:id/reject` | `h.RejectStep` |
