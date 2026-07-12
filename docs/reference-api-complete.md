# 凌镜完整 API 参考

> 生成日期：2026-07-12
>
> 事实等级：`implemented`（路由已在当前代码中注册）。这不等于 `manually_verified`、`external_observed` 或生产可用。
>
> 生成依据：以 `httpx.NewRouter(...).Engine.Routes()` 导出的 Gin 运行时路由表为准，而不是旧文档或前端调用。

本文覆盖当前后端注册的全部 HTTP API。在 Prism 关闭、Metrics 关闭的基线配置下，运行时共发现 **687** 条路由，其中 **683** 条位于 `/api/v1`。此外有 1 条 Prism 条件路由和 1 条 Metrics 条件路由。每条记录包含 HTTP 方法、完整路径、访问门槛和实际处理器。请求体及响应字段仍应以对应 `model.go`、`handler.go` 和测试为准；本清单不虚构代码未声明的字段。

## 调用基础

- 默认服务地址：`http://localhost:8080`（可由 `SERVER_PORT` 覆盖）。
- 受保护接口使用 `Authorization: Bearer <access_token>`。
- `POST /api/v1/auth/login`、`register`、`refresh` 和 `GET /api/v1/health` 无需 JWT；`GET /api/v1/auth/me` **需要 JWT**。
- `POST /api/v1/webhooks/:platform` 不用 JWT，但代码要求平台适配器支持并通过 webhook 签名校验。
- 受保护写操作还经过审批中间件；只有路由目录中命中的高风险生产写操作才会被要求提供有效审批。
- 多数业务接口返回统一包裹：`{"code":0,"message":"ok","data":...}`；少数基础设施接口直接返回 JSON，须以处理器实现为准。

### 登录示例

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"<owner-username>","password":"<password>"}'
```

仓库没有可依赖的默认 Owner 密码。请使用已有有效 Owner 账户或批准的凭据重置流程。

### 访问门槛说明

| 标记 | 含义 |
|---|---|
| Public | 无 JWT |
| Webhook signature | 外部平台调用；必须通过签名校验 |
| JWT | 任意已认证用户；业务层仍可能执行额外校验 |
| JWT + `*.read` / `*.manage` | JWT 后还需对应 RBAC 权限 |

## `/api/v1` 完整路由

### `aftersales`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/aftersales` | JWT | `domain/aftersales.(*Handler).List` |
| `POST` | `/api/v1/aftersales` | JWT | `domain/aftersales.(*Handler).Create` |
| `DELETE` | `/api/v1/aftersales/:id` | JWT | `domain/aftersales.(*Handler).Delete` |
| `GET` | `/api/v1/aftersales/:id` | JWT | `domain/aftersales.(*Handler).Get` |
| `PUT` | `/api/v1/aftersales/:id` | JWT | `domain/aftersales.(*Handler).Update` |
| `POST` | `/api/v1/aftersales/:id/approve` | JWT | `domain/aftersales.(*Handler).Approve` |
| `POST` | `/api/v1/aftersales/:id/auto-decide` | JWT | `domain/aftersales.(*Handler).AutoDecide` |
| `POST` | `/api/v1/aftersales/:id/receive` | JWT | `domain/aftersales.(*Handler).Receive` |
| `POST` | `/api/v1/aftersales/:id/refund` | JWT | `domain/aftersales.(*Handler).Refund` |
| `POST` | `/api/v1/aftersales/:id/reject` | JWT | `domain/aftersales.(*Handler).Reject` |
| `GET` | `/api/v1/aftersales/disputes` | JWT | `domain/aftersales.(*Handler).ListDisputes` |
| `POST` | `/api/v1/aftersales/disputes` | JWT | `domain/aftersales.(*Handler).CreateDispute` |
| `GET` | `/api/v1/aftersales/disputes/:id` | JWT | `domain/aftersales.(*Handler).GetDispute` |
| `POST` | `/api/v1/aftersales/disputes/:id/auto-decide` | JWT | `domain/aftersales.(*Handler).AutoDecideDispute` |
| `POST` | `/api/v1/aftersales/disputes/:id/evaluate` | JWT | `domain/aftersales.(*Handler).EvaluateDispute` |
| `PUT` | `/api/v1/aftersales/disputes/:id/status` | JWT | `domain/aftersales.(*Handler).UpdateDisputeStatus` |
| `GET` | `/api/v1/aftersales/summary` | JWT | `domain/aftersales.(*Handler).Summary` |

### `agent-learning`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/agent-learning/accuracy` | JWT | `domain/agentlearning.(*Handler).GetAllAccuracy` |
| `GET` | `/api/v1/agent-learning/accuracy/:agentId` | JWT | `domain/agentlearning.(*Handler).GetAccuracyByAgent` |
| `POST` | `/api/v1/agent-learning/evaluate` | JWT | `domain/agentlearning.(*Handler).EvaluateDecision` |
| `GET` | `/api/v1/agent-learning/evaluations` | JWT | `domain/agentlearning.(*Handler).ListEvaluations` |
| `POST` | `/api/v1/agent-learning/recalculate` | JWT | `domain/agentlearning.(*Handler).RecalculateAccuracy` |

### `agent-rules`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/agent-rules` | JWT | `domain/agentrule.(*Handler).ListRules` |
| `POST` | `/api/v1/agent-rules` | JWT | `domain/agentrule.(*Handler).CreateRule` |
| `DELETE` | `/api/v1/agent-rules/:id` | JWT | `domain/agentrule.(*Handler).DeleteRule` |
| `GET` | `/api/v1/agent-rules/:id` | JWT | `domain/agentrule.(*Handler).GetRule` |
| `PUT` | `/api/v1/agent-rules/:id` | JWT | `domain/agentrule.(*Handler).UpdateRule` |
| `POST` | `/api/v1/agent-rules/:id/toggle` | JWT | `domain/agentrule.(*Handler).ToggleRule` |
| `POST` | `/api/v1/agent-rules/evaluate` | JWT | `domain/agentrule.(*Handler).EvaluateRules` |

### `agentos`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/agentos` | JWT | `agentos.(*Handler).Overview` |
| `GET` | `/api/v1/agentos/agent-metrics` | JWT | `agentos.(*Handler).AgentMetrics` |
| `GET` | `/api/v1/agentos/agent-timeline` | JWT | `agentos.(*Handler).AgentTimeline` |
| `GET` | `/api/v1/agentos/audit-replay/:correlation_id` | JWT | `agentos.(*Handler).AuditReplay` |
| `GET` | `/api/v1/agentos/autonomy` | JWT | `agentos.(*Handler).Autonomy` |
| `GET` | `/api/v1/agentos/external-health` | JWT | `agentos.(*Handler).ExternalHealth` |
| `GET` | `/api/v1/agentos/failures` | JWT | `agentos.(*Handler).FailedRuns` |
| `GET` | `/api/v1/agentos/intercepted-actions` | JWT | `agentos.(*Handler).InterceptedActions` |
| `GET` | `/api/v1/agentos/status` | JWT | `agentos.(*Handler).Status` |
| `GET` | `/api/v1/agentos/traffic-summary` | JWT | `agentos.(*Handler).TrafficSummary` |
| `GET` | `/api/v1/agentos/work-items` | JWT | `agentos.(*Handler).WorkItems` |
| `GET` | `/api/v1/agentos/work-items/:id` | JWT | `agentos.(*Handler).WorkItemDetail` |

### `agents`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/agents` | JWT | `agent.(*Handler).ListAgents` |
| `POST` | `/api/v1/agents` | JWT | `agent.(*Handler).CreateAgent` |
| `GET` | `/api/v1/agents/:id` | JWT | `agent.(*Handler).GetAgent` |
| `POST` | `/api/v1/agents/:id/actions` | JWT | `agent.(*Handler).ExecuteAction` |
| `GET` | `/api/v1/agents/entropy` | JWT | `agent.(*Handler).Entropy` |
| `GET` | `/api/v1/agents/evolution` | JWT | `agent.(*Handler).Evolution` |
| `GET` | `/api/v1/agents/rules` | JWT | `domain/personalrule.(*Handler).ListRules` |
| `POST` | `/api/v1/agents/rules` | JWT | `domain/personalrule.(*Handler).CreateRule` |
| `DELETE` | `/api/v1/agents/rules/:id` | JWT | `domain/personalrule.(*Handler).DeleteRule` |
| `GET` | `/api/v1/agents/rules/:id` | JWT | `domain/personalrule.(*Handler).GetRule` |
| `PUT` | `/api/v1/agents/rules/:id` | JWT | `domain/personalrule.(*Handler).UpdateRule` |
| `POST` | `/api/v1/agents/rules/apply` | JWT | `domain/personalrule.(*Handler).ApplyRules` |

### `ai`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/ai/actions` | JWT | `ai.(*Handler).ListActions` |
| `POST` | `/api/v1/ai/actions` | JWT | `ai.(*Handler).CreateAction` |
| `GET` | `/api/v1/ai/actions/:id` | JWT | `ai.(*Handler).GetAction` |
| `POST` | `/api/v1/ai/actions/:id/approve` | JWT + ai.action | `ai.(*Handler).ApproveAction` |
| `POST` | `/api/v1/ai/actions/:id/execute` | JWT + ai.action | `ai.(*Handler).ExecuteAction` |
| `POST` | `/api/v1/ai/actions/:id/reject` | JWT + ai.action | `ai.(*Handler).RejectAction` |
| `POST` | `/api/v1/ai/actions/:id/review` | JWT | `ai.(*Handler).ReviewAction` |
| `GET` | `/api/v1/ai/agents` | JWT | `ai.(*Handler).Roster` |
| `GET` | `/api/v1/ai/agents/specs` | JWT | `ai.(*Handler).AgentSpecs` |
| `POST` | `/api/v1/ai/chat` | JWT | `ai.(*Handler).Chat` |
| `POST` | `/api/v1/ai/moa` | JWT | `ai.RegisterRoutes.func1` |
| `POST` | `/api/v1/ai/run` | JWT | `ai.(*Handler).RunAgent` |
| `GET` | `/api/v1/ai/traces` | JWT | `ai.(*Handler).ListTraces` |
| `GET` | `/api/v1/ai/traces/:trace_id` | JWT | `ai.(*Handler).GetTrace` |

### `aios`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/aios/health` | JWT | `aios/setup.RegisterAIOSRoutes.func1` |
| `GET` | `/api/v1/aios/ipc/health` | JWT | `aios/setup.RegisterAIOSRoutes.func4` |
| `GET` | `/api/v1/aios/runtime/agents` | JWT | `aios/setup.RegisterAIOSRoutes.func2` |
| `GET` | `/api/v1/aios/scheduler/retry-queue` | JWT | `httpx.NewRouter.func20` |
| `GET` | `/api/v1/aios/scheduler/tasks` | JWT | `httpx.NewRouter.func19` |
| `GET` | `/api/v1/aios/tools` | JWT | `aios/setup.RegisterAIOSRoutes.func3` |

### `allocation`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/allocation/auto-allocate/:skuId` | JWT | `domain/allocation.(*Handler).AutoAllocate` |
| `POST` | `/api/v1/allocation/cost/:batchId/compute` | JWT | `domain/allocation.(*Handler).ComputeAllocation` |
| `GET` | `/api/v1/allocation/cost/batches` | JWT | `domain/allocation.(*Handler).ListBatches` |
| `POST` | `/api/v1/allocation/cost/batches` | JWT | `domain/allocation.(*Handler).CreateBatch` |
| `GET` | `/api/v1/allocation/cost/batches/:id` | JWT | `domain/allocation.(*Handler).GetBatch` |
| `GET` | `/api/v1/allocation/rules` | JWT | `domain/allocation.(*Handler).ListRules` |
| `POST` | `/api/v1/allocation/rules` | JWT | `domain/allocation.(*Handler).CreateRule` |
| `DELETE` | `/api/v1/allocation/rules/:id` | JWT | `domain/allocation.(*Handler).DeleteRule` |
| `PUT` | `/api/v1/allocation/rules/:id` | JWT | `domain/allocation.(*Handler).UpdateRule` |
| `GET` | `/api/v1/allocation/warehouses` | JWT | `domain/allocation.(*Handler).ListWarehouses` |
| `POST` | `/api/v1/allocation/warehouses` | JWT | `domain/allocation.(*Handler).CreateWarehouse` |
| `DELETE` | `/api/v1/allocation/warehouses/:id` | JWT | `domain/allocation.(*Handler).DeleteWarehouse` |
| `PUT` | `/api/v1/allocation/warehouses/:id` | JWT | `domain/allocation.(*Handler).UpdateWarehouse` |

### `approval`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/approval` | JWT | `domain/approval.(*Handler).ListApprovals` |
| `POST` | `/api/v1/approval` | JWT | `domain/approval.(*Handler).CreateApproval` |
| `GET` | `/api/v1/approval/:id` | JWT | `domain/approval.(*Handler).GetApproval` |
| `PUT` | `/api/v1/approval/:id/review` | JWT | `domain/approval.(*Handler).ReviewApproval` |
| `GET` | `/api/v1/approval/my` | JWT | `domain/approval.(*Handler).MyPending` |
| `GET` | `/api/v1/approval/stats` | JWT | `domain/approval.(*Handler).ApprovalStats` |

### `auth`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/auth/login` | Public | `auth.(*Handler).Login` |
| `GET` | `/api/v1/auth/me` | JWT | `auth.(*Handler).CurrentUser` |
| `POST` | `/api/v1/auth/refresh` | Public | `auth.(*Handler).Refresh` |
| `POST` | `/api/v1/auth/register` | Public | `auth.(*Handler).Register` |

### `brands`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/brands` | JWT | `domain/brand.(*Handler).List` |
| `POST` | `/api/v1/brands` | JWT | `domain/brand.(*Handler).Create` |
| `DELETE` | `/api/v1/brands/:id` | JWT | `domain/brand.(*Handler).Delete` |
| `GET` | `/api/v1/brands/:id` | JWT | `domain/brand.(*Handler).Get` |
| `PUT` | `/api/v1/brands/:id` | JWT | `domain/brand.(*Handler).Update` |

### `candidates`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/candidates` | JWT | `domain/candidate.(*Handler).List` |
| `POST` | `/api/v1/candidates` | JWT | `domain/candidate.(*Handler).Create` |
| `DELETE` | `/api/v1/candidates/:id` | JWT | `domain/candidate.(*Handler).Delete` |
| `GET` | `/api/v1/candidates/:id` | JWT | `domain/candidate.(*Handler).Get` |
| `PUT` | `/api/v1/candidates/:id` | JWT | `domain/candidate.(*Handler).Update` |
| `POST` | `/api/v1/candidates/:id/completeness` | JWT | `domain/completeness.(*Handler).CheckEnhanced` |
| `PUT` | `/api/v1/candidates/:id/fields` | JWT | `domain/candidate.(*Handler).FillFields` |
| `POST` | `/api/v1/candidates/:id/rescrape` | JWT | `domain/candidate.(*Handler).Rescrape` |
| `POST` | `/api/v1/candidates/:id/skip-field` | JWT | `domain/candidate.(*Handler).SkipField` |
| `GET` | `/api/v1/candidates/collect-leads` | JWT | `domain/candidate.(*Handler).ListCollectLeads` |
| `GET` | `/api/v1/candidates/collect-leads/:id` | JWT | `domain/candidate.(*Handler).GetCollectLead` |
| `GET` | `/api/v1/candidates/collection-evidence/:id` | JWT | `domain/candidate.(*Handler).GetCollectionEvidence` |
| `GET` | `/api/v1/candidates/count` | JWT | `domain/candidate.(*Handler).Count` |
| `GET` | `/api/v1/candidates/dedup` | JWT | `domain/candidate.(*Handler).Dedup` |
| `POST` | `/api/v1/candidates/seed` | JWT | `domain/candidate.(*Handler).Seed` |

### `categories`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/categories` | JWT | `domain/category.(*Handler).List` |
| `POST` | `/api/v1/categories` | JWT | `domain/category.(*Handler).Create` |
| `DELETE` | `/api/v1/categories/:id` | JWT | `domain/category.(*Handler).Delete` |
| `GET` | `/api/v1/categories/:id` | JWT | `domain/category.(*Handler).Get` |
| `PUT` | `/api/v1/categories/:id` | JWT | `domain/category.(*Handler).Update` |
| `GET` | `/api/v1/categories/tree` | JWT | `domain/category.(*Handler).Tree` |

### `competitor-prices`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/competitor-prices` | JWT + finance.read | `domain/price.(*Handler).ListCompetitorPrices` |
| `POST` | `/api/v1/competitor-prices` | JWT + finance.read | `domain/price.(*Handler).CreateCompetitorPrice` |
| `DELETE` | `/api/v1/competitor-prices/:id` | JWT + finance.read | `domain/price.(*Handler).DeleteCompetitorPrice` |
| `GET` | `/api/v1/competitor-prices/:id` | JWT + finance.read | `domain/price.(*Handler).GetCompetitorPrice` |

### `competitors`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/competitors` | JWT | `domain/competitor.(*Handler).List` |
| `POST` | `/api/v1/competitors` | JWT | `domain/competitor.(*Handler).Create` |
| `DELETE` | `/api/v1/competitors/:id` | JWT | `domain/competitor.(*Handler).Delete` |
| `GET` | `/api/v1/competitors/:id` | JWT | `domain/competitor.(*Handler).Get` |
| `PUT` | `/api/v1/competitors/:id` | JWT | `domain/competitor.(*Handler).Update` |
| `GET` | `/api/v1/competitors/:id/prices` | JWT | `domain/competitor.(*Handler).ListPrices` |
| `POST` | `/api/v1/competitors/:id/prices` | JWT | `domain/competitor.(*Handler).RecordPrice` |
| `GET` | `/api/v1/competitors/:id/trend` | JWT | `domain/competitor.(*Handler).GetPriceTrend` |

### `completeness`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/completeness/check/:productId` | JWT | `domain/completeness.(*Handler).Check` |
| `GET` | `/api/v1/completeness/checks` | JWT | `domain/completeness.(*Handler).ListChecks` |

### `compliance`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/compliance/check` | JWT | `domain/compliance.(*Handler).Check` |
| `GET` | `/api/v1/compliance/results` | JWT | `domain/compliance.(*Handler).ListResults` |
| `GET` | `/api/v1/compliance/results/:id` | JWT | `domain/compliance.(*Handler).GetResult` |
| `PUT` | `/api/v1/compliance/results/:id/suppress` | JWT | `domain/compliance.(*Handler).SuppressResult` |
| `POST` | `/api/v1/compliance/scan` | JWT | `domain/compliance.(*Handler).Scan` |

### `consolidation`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/consolidation/groups` | JWT | `domain/consolidation.(*Handler).ListGroups` |
| `POST` | `/api/v1/consolidation/groups` | JWT | `domain/consolidation.(*Handler).CreateGroup` |
| `GET` | `/api/v1/consolidation/groups/:groupId` | JWT | `domain/consolidation.(*Handler).GetGroup` |
| `GET` | `/api/v1/consolidation/groups/:groupId/items` | JWT | `domain/consolidation.(*Handler).GetGroupItems` |
| `POST` | `/api/v1/consolidation/groups/:groupId/items` | JWT | `domain/consolidation.(*Handler).AddItem` |
| `DELETE` | `/api/v1/consolidation/groups/:groupId/items/:itemId` | JWT | `domain/consolidation.(*Handler).RemoveItem` |
| `POST` | `/api/v1/consolidation/groups/:groupId/negotiate` | JWT | `domain/consolidation.(*Handler).NegotiateGroup` |

### `content`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/content/generate` | JWT | `domain/content.(*Handler).GenerateContent` |
| `POST` | `/api/v1/content/validate` | JWT | `domain/content.(*Handler).ValidateContent` |

### `cost`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/cost/dashboard` | JWT | `domain/cost.(*Handler).Dashboard` |

### `dashboard`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/dashboard/brief` | JWT | `domain/dashboard.(*Handler).DailyBrief` |
| `GET` | `/api/v1/dashboard/exceptions` | JWT | `domain/dashboard.(*Handler).Exceptions` |
| `GET` | `/api/v1/dashboard/inventory` | JWT | `domain/dashboard.(*Handler).Inventory` |
| `GET` | `/api/v1/dashboard/orders` | JWT | `domain/dashboard.(*Handler).Orders` |
| `GET` | `/api/v1/dashboard/overview` | JWT | `domain/dashboard.(*Handler).Overview` |
| `GET` | `/api/v1/dashboard/rejection-reasons` | JWT | `domain/dashboard.(*Handler).RejectionReasons` |

### `decision`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/decision` | JWT | `domain/decision.(*Handler).List` |
| `POST` | `/api/v1/decision` | JWT | `domain/decision.(*Handler).Create` |
| `DELETE` | `/api/v1/decision/:id` | JWT | `domain/decision.(*Handler).Delete` |
| `GET` | `/api/v1/decision/:id` | JWT | `domain/decision.(*Handler).Get` |
| `PUT` | `/api/v1/decision/:id` | JWT | `domain/decision.(*Handler).Update` |
| `POST` | `/api/v1/decision/:id/approve` | JWT | `domain/decision.(*Handler).Approve` |
| `POST` | `/api/v1/decision/:id/reject` | JWT | `domain/decision.(*Handler).Reject` |
| `GET` | `/api/v1/decision/summary` | JWT | `domain/decision.(*Handler).Summary` |

### `demand-cases`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/demand-cases` | JWT | `domain/demandcase.(*Handler).List` |
| `POST` | `/api/v1/demand-cases` | JWT | `domain/demandcase.(*Handler).Create` |
| `GET` | `/api/v1/demand-cases/:id` | JWT | `domain/demandcase.(*Handler).Get` |
| `GET` | `/api/v1/demand-cases/:id/decision-card` | JWT | `domain/demandcase.(*Handler).DecisionCard` |
| `POST` | `/api/v1/demand-cases/:id/evaluate` | JWT | `domain/demandcase.(*Handler).Evaluate` |
| `POST` | `/api/v1/demand-cases/:id/evidence` | JWT | `domain/demandcase.(*Handler).AddEvidence` |
| `POST` | `/api/v1/demand-cases/:id/falsifications` | JWT | `domain/demandcase.(*Handler).AddFalsification` |
| `POST` | `/api/v1/demand-cases/research/first-public-batch` | JWT | `domain/demandcase.(*Handler).RunFirstBatch` |
| `POST` | `/api/v1/demand-cases/research/import` | JWT | `domain/demandcase.(*Handler).ImportResearch` |

### `entropy`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/entropy` | JWT | `domain/entropy.(*Handler).GetSummary` |
| `GET` | `/api/v1/entropy/changelog` | JWT | `domain/entropy.(*Handler).GetChangeLog` |
| `POST` | `/api/v1/entropy/defense` | JWT | `domain/entropy.(*Handler).RunDefenses` |
| `GET` | `/api/v1/entropy/health` | JWT | `domain/entropy.(*Handler).GetHealthScores` |
| `GET` | `/api/v1/entropy/spc` | JWT | `domain/entropy.(*Handler).GetSpcStatus` |

### `evolution`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/evolution/nudges` | JWT | `domain/evolution.(*Handler).ListNudges` |
| `POST` | `/api/v1/evolution/nudges/:id/accept` | JWT | `domain/evolution.(*Handler).AcceptNudge` |
| `POST` | `/api/v1/evolution/nudges/:id/dismiss` | JWT | `domain/evolution.(*Handler).DismissNudge` |
| `POST` | `/api/v1/evolution/nudges/evaluate` | JWT | `domain/evolution.(*Handler).EvaluateNudges` |

### `exceptions`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/exceptions` | JWT | `domain/exceptions.(*Handler).List` |
| `POST` | `/api/v1/exceptions` | JWT | `domain/exceptions.(*Handler).Create` |
| `DELETE` | `/api/v1/exceptions/:id` | JWT | `domain/exceptions.(*Handler).Delete` |
| `GET` | `/api/v1/exceptions/:id` | JWT | `domain/exceptions.(*Handler).Get` |
| `PUT` | `/api/v1/exceptions/:id` | JWT | `domain/exceptions.(*Handler).Update` |
| `PUT` | `/api/v1/exceptions/:id/assign` | JWT | `domain/exceptions.(*Handler).Assign` |
| `POST` | `/api/v1/exceptions/:id/resolve` | JWT | `domain/exceptions.(*Handler).OwnerResolve` |
| `PUT` | `/api/v1/exceptions/:id/resolve` | JWT | `domain/exceptions.(*Handler).Resolve` |
| `POST` | `/api/v1/exceptions/:id/suggest` | JWT | `domain/exceptions.(*Handler).Suggest` |
| `POST` | `/api/v1/exceptions/auto-detect` | JWT | `domain/exceptions.(*Handler).AutoDetect` |

### `exchange-rates`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/exchange-rates` | JWT | `domain/exchangerate.(*Handler).List` |
| `POST` | `/api/v1/exchange-rates` | JWT | `domain/exchangerate.(*Handler).Create` |
| `PUT` | `/api/v1/exchange-rates/:from_currency/:to_currency` | JWT | `domain/exchangerate.(*Handler).UpdateByPair` |
| `GET` | `/api/v1/exchange-rates/:from_currency/:to_currency/latest` | JWT | `domain/exchangerate.(*Handler).GetLatest` |
| `DELETE` | `/api/v1/exchange-rates/:id` | JWT | `domain/exchangerate.(*Handler).Delete` |

### `experiments`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/experiments` | JWT | `domain/experiment.(*Handler).List` |
| `POST` | `/api/v1/experiments` | JWT | `domain/experiment.(*Handler).Create` |
| `GET` | `/api/v1/experiments/:experimentId` | JWT | `domain/experiment.(*Handler).Get` |
| `PUT` | `/api/v1/experiments/:experimentId` | JWT | `domain/experiment.(*Handler).Update` |
| `POST` | `/api/v1/experiments/:experimentId/evidence` | JWT | `domain/experiment.(*Handler).AddEvidence` |
| `POST` | `/api/v1/experiments/:experimentId/evidence/:evidenceId/verify` | JWT | `domain/experiment.(*Handler).VerifyEvidence` |
| `POST` | `/api/v1/experiments/:experimentId/gates/evaluate` | JWT | `domain/experiment.(*Handler).EvaluateGate` |
| `POST` | `/api/v1/experiments/:experimentId/links` | JWT | `domain/experiment.(*Handler).AddObjectLink` |
| `GET` | `/api/v1/experiments/:experimentId/owner-summary` | JWT | `domain/experiment.(*Handler).OwnerSummary` |

### `feedback`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/feedback/assigned-to-me` | JWT | `feedback.(*Handler).ListSubmissionsForAgent` |
| `POST` | `/api/v1/feedback/categories` | JWT | `feedback.(*Handler).CreateCategory` |
| `DELETE` | `/api/v1/feedback/categories/:id` | JWT | `feedback.(*Handler).DeleteCategory` |
| `PUT` | `/api/v1/feedback/categories/:id` | JWT | `feedback.(*Handler).UpdateCategory` |
| `DELETE` | `/api/v1/feedback/comments/:id` | JWT | `feedback.(*Handler).DeleteComment` |
| `POST` | `/api/v1/feedback/migrate` | JWT | `feedback.(*Handler).Migrate` |
| `GET` | `/api/v1/feedback/mine` | JWT | `feedback.(*Handler).ListMySubmissions` |
| `GET` | `/api/v1/feedback/pending-for-agent` | JWT | `feedback.(*Handler).ListSubmissionsForAgent` |
| `GET` | `/api/v1/feedback/projects` | JWT | `feedback.(*Handler).ListProjects` |
| `POST` | `/api/v1/feedback/projects` | JWT | `feedback.(*Handler).CreateProject` |
| `DELETE` | `/api/v1/feedback/projects/:id` | JWT | `feedback.(*Handler).DeleteProject` |
| `GET` | `/api/v1/feedback/projects/:id` | JWT | `feedback.(*Handler).GetProject` |
| `PUT` | `/api/v1/feedback/projects/:id` | JWT | `feedback.(*Handler).UpdateProject` |
| `GET` | `/api/v1/feedback/projects/:id/analytics` | JWT | `feedback.(*Handler).GetAnalytics` |
| `GET` | `/api/v1/feedback/projects/:id/categories` | JWT | `feedback.(*Handler).ListCategories` |
| `GET` | `/api/v1/feedback/projects/:id/stats` | JWT | `feedback.(*Handler).GetDashboardStats` |
| `GET` | `/api/v1/feedback/projects/:id/submissions` | JWT | `feedback.(*Handler).ListSubmissions` |
| `GET` | `/api/v1/feedback/projects/:id/tags` | JWT | `feedback.(*Handler).ListTags` |
| `POST` | `/api/v1/feedback/submissions` | JWT | `feedback.(*Handler).CreateSubmission` |
| `DELETE` | `/api/v1/feedback/submissions/:id` | JWT | `feedback.(*Handler).DeleteSubmission` |
| `GET` | `/api/v1/feedback/submissions/:id` | JWT | `feedback.(*Handler).GetSubmission` |
| `PUT` | `/api/v1/feedback/submissions/:id` | JWT | `feedback.(*Handler).UpdateSubmission` |
| `GET` | `/api/v1/feedback/submissions/:id/comments` | JWT | `feedback.(*Handler).ListComments` |
| `POST` | `/api/v1/feedback/submissions/:id/comments` | JWT | `feedback.(*Handler).AddComment` |
| `PUT` | `/api/v1/feedback/submissions/:id/status` | JWT | `feedback.(*Handler).UpdateSubmissionStatus` |
| `DELETE` | `/api/v1/feedback/submissions/:id/tags/:tagId` | JWT | `feedback.(*Handler).RemoveTag` |
| `POST` | `/api/v1/feedback/submissions/:id/tags/:tagId` | JWT | `feedback.(*Handler).AddTag` |
| `POST` | `/api/v1/feedback/submissions/:id/vote` | JWT | `feedback.(*Handler).Vote` |
| `POST` | `/api/v1/feedback/tags` | JWT | `feedback.(*Handler).CreateTag` |
| `DELETE` | `/api/v1/feedback/tags/:id` | JWT | `feedback.(*Handler).DeleteTag` |

### `finance`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/finance/accounts` | JWT + finance.read | `domain/finance.(*Handler).ListAccounts` |
| `POST` | `/api/v1/finance/accounts` | JWT + finance.read | `domain/finance.(*Handler).CreateAccount` |
| `DELETE` | `/api/v1/finance/accounts/:id` | JWT + finance.read | `domain/finance.(*Handler).DeleteAccount` |
| `GET` | `/api/v1/finance/accounts/:id` | JWT + finance.read | `domain/finance.(*Handler).GetAccount` |
| `PUT` | `/api/v1/finance/accounts/:id` | JWT + finance.read | `domain/finance.(*Handler).UpdateAccount` |
| `GET` | `/api/v1/finance/ledger` | JWT + finance.read | `domain/finance.(*Handler).ListLedger` |
| `POST` | `/api/v1/finance/mock` | JWT + finance.read | `domain/finance.(*Handler).Mock` |
| `GET` | `/api/v1/finance/orders/:order_id/ledger` | JWT + finance.read | `domain/finance.(*Handler).ListOrderLedger` |
| `POST` | `/api/v1/finance/orders/:order_id/ledger/rebuild` | JWT + finance.read | `domain/finance.(*Handler).RebuildOrderLedger` |
| `GET` | `/api/v1/finance/orders/:order_id/profit` | JWT + finance.read | `domain/finance.(*Handler).OrderProfit` |
| `GET` | `/api/v1/finance/profit-summary` | JWT + finance.read | `domain/finance.(*Handler).ProfitSummary` |
| `POST` | `/api/v1/finance/profit/batch-calculate` | JWT + finance.read | `domain/finance.(*Handler).BatchCalculateProfit` |
| `POST` | `/api/v1/finance/profit/calculate` | JWT + finance.read | `domain/finance.(*Handler).CalculateProfit` |
| `GET` | `/api/v1/finance/profit/ranking` | JWT + finance.read | `domain/finance.(*Handler).GetSKUProfitRanking` |
| `GET` | `/api/v1/finance/profit/summary` | JWT + finance.read | `domain/finance.(*Handler).GetProfitSummary` |
| `GET` | `/api/v1/finance/summary` | JWT + finance.read | `domain/finance.(*Handler).Summary` |
| `GET` | `/api/v1/finance/transactions` | JWT + finance.read | `domain/finance.(*Handler).ListTransactions` |
| `POST` | `/api/v1/finance/transactions` | JWT + finance.read | `domain/finance.(*Handler).CreateTransaction` |

### `health`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/health` | Public | `httpx.NewRouter.func18` |

### `image-gen`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/image-gen` | JWT | `domain/imagegen.(*Handler).ListImageGens` |
| `POST` | `/api/v1/image-gen` | JWT | `domain/imagegen.(*Handler).CreateImageGen` |
| `DELETE` | `/api/v1/image-gen/:id` | JWT | `domain/imagegen.(*Handler).DeleteImageGen` |
| `GET` | `/api/v1/image-gen/:id` | JWT | `domain/imagegen.(*Handler).GetImageGen` |
| `PUT` | `/api/v1/image-gen/:id/status` | JWT | `domain/imagegen.(*Handler).UpdateImageGenStatus` |
| `GET` | `/api/v1/image-gen/canvas` | JWT | `domain/imagegen.(*Handler).ListCanvases` |
| `POST` | `/api/v1/image-gen/canvas` | JWT | `domain/imagegen.(*Handler).CreateCanvas` |
| `DELETE` | `/api/v1/image-gen/canvas/:id` | JWT | `domain/imagegen.(*Handler).DeleteCanvas` |
| `GET` | `/api/v1/image-gen/canvas/:id` | JWT | `domain/imagegen.(*Handler).GetCanvas` |
| `PUT` | `/api/v1/image-gen/canvas/:id` | JWT | `domain/imagegen.(*Handler).UpdateCanvas` |
| `GET` | `/api/v1/image-gen/templates` | JWT | `domain/imagegen.(*Handler).ListTemplates` |
| `POST` | `/api/v1/image-gen/templates` | JWT | `domain/imagegen.(*Handler).CreateTemplate` |
| `DELETE` | `/api/v1/image-gen/templates/:id` | JWT | `domain/imagegen.(*Handler).DeleteTemplate` |
| `GET` | `/api/v1/image-gen/templates/:id` | JWT | `domain/imagegen.(*Handler).GetTemplate` |
| `PUT` | `/api/v1/image-gen/templates/:id` | JWT | `domain/imagegen.(*Handler).UpdateTemplate` |
| `POST` | `/api/v1/image-gen/templates/:id/use` | JWT | `domain/imagegen.(*Handler).UseTemplate` |

### `import-batch`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/import-batch` | JWT | `domain/importbatch.(*Handler).ListBatches` |
| `POST` | `/api/v1/import-batch` | JWT | `domain/importbatch.(*Handler).CreateBatch` |
| `DELETE` | `/api/v1/import-batch/:id` | JWT | `domain/importbatch.(*Handler).DeleteBatch` |
| `GET` | `/api/v1/import-batch/:id` | JWT | `domain/importbatch.(*Handler).GetBatch` |
| `PUT` | `/api/v1/import-batch/:id` | JWT | `domain/importbatch.(*Handler).UpdateBatch` |
| `GET` | `/api/v1/import-batch/:id/rows` | JWT | `domain/importbatch.(*Handler).ListRows` |
| `POST` | `/api/v1/import-batch/upload` | JWT | `domain/importbatch.(*Handler).Upload` |

### `inventory`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/inventory` | JWT + inventory.read | `domain/inventory.(*Handler).List` |
| `GET` | `/api/v1/inventory/:id` | JWT + inventory.read | `domain/inventory.(*Handler).Get` |
| `PUT` | `/api/v1/inventory/:id` | JWT + inventory.read | `domain/inventory.(*Handler).Update` |
| `POST` | `/api/v1/inventory/:id/lock` | JWT + inventory.read | `domain/inventory.(*Handler).Lock` |
| `POST` | `/api/v1/inventory/:id/unlock` | JWT + inventory.read | `domain/inventory.(*Handler).Unlock` |
| `GET` | `/api/v1/inventory/allocate/:sku_id` | JWT + inventory.read | `domain/inventory.(*Handler).AllocateStock` |
| `POST` | `/api/v1/inventory/dead-stock/analyze` | JWT + inventory.read | `domain/inventory.(*Handler).IdentifyDeadStock` |
| `GET` | `/api/v1/inventory/dead-stock/logs` | JWT + inventory.read | `domain/inventory.(*Handler).ListDeadStockLogs` |
| `GET` | `/api/v1/inventory/locations` | JWT + inventory.read | `domain/inventory.(*Handler).ListLocations` |
| `GET` | `/api/v1/inventory/logs` | JWT + inventory.read | `domain/inventory.(*Handler).ListLogs` |
| `GET` | `/api/v1/inventory/oversell-report` | JWT + inventory.read | `domain/inventory.(*Handler).OversellReport` |
| `GET` | `/api/v1/inventory/safety-config/:sku_id` | JWT + inventory.read | `domain/inventory.(*Handler).GetSafetyConfig` |
| `PUT` | `/api/v1/inventory/safety-config/:sku_id` | JWT + inventory.read | `domain/inventory.(*Handler).UpsertSafetyConfig` |
| `GET` | `/api/v1/inventory/safety-configs` | JWT + inventory.read | `domain/inventory.(*Handler).ListSafetyConfigs` |
| `GET` | `/api/v1/inventory/sku/:sku_id/warehouses` | JWT + inventory.read | `domain/inventory.(*Handler).ListInventoryBySku` |
| `POST` | `/api/v1/inventory/sync-cross-platform/:productId` | JWT + inventory.read | `domain/inventory.(*Handler).SyncCrossPlatform` |
| `GET` | `/api/v1/inventory/transfers` | JWT + inventory.read | `domain/inventory.(*Handler).ListTransfers` |

### `kill-switch`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/kill-switch/activate` | JWT + admin.killswitch | `platform/killswitch.(*Handler).Activate` |
| `POST` | `/api/v1/kill-switch/deactivate` | JWT + admin.killswitch | `platform/killswitch.(*Handler).Deactivate` |
| `GET` | `/api/v1/kill-switch/status` | JWT + admin.killswitch | `platform/killswitch.(*Handler).GetStatus` |

### `landed-cost`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/landed-cost/:productId` | JWT | `domain/landedcost.(*Handler).GetLandedCost` |
| `GET` | `/api/v1/landed-cost/:productId/compare` | JWT | `domain/landedcost.(*Handler).CompareAcrossPlatforms` |
| `POST` | `/api/v1/landed-cost/calculate` | JWT | `domain/landedcost.(*Handler).Calculate` |

### `listing`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/listing` | JWT + listing.read | `domain/listing.(*Handler).Create` |

### `listing-task`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/listing-task/:task_id/execute` | JWT + listing.read | `domain/listingtask.(*Handler).Execute` |
| `POST` | `/api/v1/listing-task/:task_id/feedback` | JWT + listing.read | `domain/listingtask.(*Handler).Feedback` |
| `POST` | `/api/v1/listing-task/:task_id/items/:item_id/retry` | JWT + listing.read | `domain/listingtask.(*Handler).RetryItem` |
| `POST` | `/api/v1/listing-task/:task_id/retry-failed` | JWT + listing.read | `domain/listingtask.(*Handler).RetryFailed` |
| `POST` | `/api/v1/listing-task/retry-all` | JWT + listing.read | `domain/listingtask.(*Handler).RetryAll` |
| `GET` | `/api/v1/listing-task/stats` | JWT + listing.read | `domain/listingtask.(*Handler).ListStats` |

### `listing-tasks`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/listing-tasks` | JWT + listing.read | `domain/listingtask.(*Handler).List` |
| `POST` | `/api/v1/listing-tasks` | JWT + listing.read | `domain/listingtask.(*Handler).Create` |
| `DELETE` | `/api/v1/listing-tasks/:id` | JWT + listing.read | `domain/listingtask.(*Handler).Delete` |
| `GET` | `/api/v1/listing-tasks/:id` | JWT + listing.read | `domain/listingtask.(*Handler).Get` |
| `PUT` | `/api/v1/listing-tasks/:id` | JWT + listing.read | `domain/listingtask.(*Handler).Update` |
| `GET` | `/api/v1/listing-tasks/:id/items` | JWT + listing.read | `domain/listingtask.(*Handler).ListItems` |
| `POST` | `/api/v1/listing-tasks/:id/items` | JWT + listing.read | `domain/listingtask.(*Handler).CreateItem` |
| `DELETE` | `/api/v1/listing-tasks/:id/items/:item_id` | JWT + listing.read | `domain/listingtask.(*Handler).DeleteItem` |
| `PUT` | `/api/v1/listing-tasks/:id/items/:item_id` | JWT + listing.read | `domain/listingtask.(*Handler).UpdateItem` |
| `GET` | `/api/v1/listing-tasks/:id/review` | JWT + listing.read | `domain/listingtask.(*Handler).Review` |
| `POST` | `/api/v1/listing-tasks/from-suggestion` | JWT + listing.read | `domain/listingtask.(*Handler).CreateFromSuggestion` |

### `listing`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/listing/listing-tasks/:task_id/cancel` | JWT + listing.read | `domain/listing.(*Handler).CancelTask` |
| `POST` | `/api/v1/listing/listing-tasks/:task_id/publish` | JWT + listing.read | `domain/listing.(*Handler).PublishTask` |
| `POST` | `/api/v1/listing/listing-tasks/:task_id/recheck` | JWT + listing.read | `domain/listing.(*Handler).RecheckTask` |
| `POST` | `/api/v1/listing/listing-tasks/from-decisions` | JWT + listing.read | `domain/listing.(*Handler).CreateTasksFromDecisions` |
| `GET` | `/api/v1/listing/products/:product_id/listings` | JWT + listing.read | `domain/listing.(*Handler).ListByProduct` |
| `GET` | `/api/v1/listing/products/:product_id/platform-comparison` | JWT + listing.read | `domain/listing.(*Handler).GetPlatformComparison` |
| `POST` | `/api/v1/listing/products/:product_id/publish/:platform_id` | JWT + listing.read | `domain/listing.(*Handler).PublishProduct` |

### `listings`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/listings` | JWT + listing.read | `domain/listing.(*Handler).List` |
| `POST` | `/api/v1/listings` | JWT + listing.read | `domain/listing.(*Handler).Create` |
| `DELETE` | `/api/v1/listings/:id` | JWT + listing.read | `domain/listing.(*Handler).Delete` |
| `GET` | `/api/v1/listings/:id` | JWT + listing.read | `domain/listing.(*Handler).Get` |
| `PUT` | `/api/v1/listings/:id` | JWT + listing.read | `domain/listing.(*Handler).Update` |
| `POST` | `/api/v1/listings/:id/publish` | JWT + listing.read | `domain/listing.(*Handler).Publish` |
| `POST` | `/api/v1/listings/:id/sync` | JWT + listing.read | `domain/listing.(*Handler).Sync` |
| `POST` | `/api/v1/listings/suggest` | JWT + listing.read | `domain/listing.(*Handler).Suggest` |

### `logistics`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/logistics/quote` | JWT | `domain/logistics.(*Handler).GetQuotes` |

### `loop`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/loop/batch-evaluate` | JWT | `domain/loop.(*Handler).BatchEvaluate` |
| `POST` | `/api/v1/loop/evaluate/:productId` | JWT | `domain/loop.(*Handler).Evaluate` |
| `GET` | `/api/v1/loop/recommendations` | JWT | `domain/loop.(*Handler).GetRecommendations` |

### `metabolism`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/metabolism` | JWT | `domain/metabolism.(*Handler).ListLogs` |
| `GET` | `/api/v1/metabolism/:id` | JWT | `domain/metabolism.(*Handler).GetLog` |
| `POST` | `/api/v1/metabolism/dry-run` | JWT | `domain/metabolism.(*Handler).DryRun` |
| `GET` | `/api/v1/metabolism/excretion-result` | JWT | `domain/metabolism.(*Handler).GetExcretionResult` |
| `POST` | `/api/v1/metabolism/execute` | JWT | `domain/metabolism.(*Handler).ExecuteEntities` |

### `mock`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/mock/orders` | JWT | `domain/mock.(*Handler).ListOrders` |
| `POST` | `/api/v1/mock/seed` | JWT | `domain/mock.(*Handler).Seed` |
| `GET` | `/api/v1/mock/settlements` | JWT | `domain/mock.(*Handler).ListSettlements` |
| `GET` | `/api/v1/mock/sync-statuses` | JWT | `domain/mock.(*Handler).SyncStatuses` |

### `notification`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/notification` | JWT | `domain/notification.(*Handler).List` |
| `POST` | `/api/v1/notification` | JWT | `domain/notification.(*Handler).Create` |
| `DELETE` | `/api/v1/notification/:id` | JWT | `domain/notification.(*Handler).Delete` |
| `GET` | `/api/v1/notification/:id` | JWT | `domain/notification.(*Handler).Get` |
| `PUT` | `/api/v1/notification/:id/read` | JWT | `domain/notification.(*Handler).MarkAsRead` |
| `GET` | `/api/v1/notification/alert-rules` | JWT | `domain/notification.(*Handler).ListAlertRules` |
| `POST` | `/api/v1/notification/alert-rules` | JWT | `domain/notification.(*Handler).CreateAlertRule` |
| `DELETE` | `/api/v1/notification/alert-rules/:id` | JWT | `domain/notification.(*Handler).DeleteAlertRule` |
| `PUT` | `/api/v1/notification/alert-rules/:id` | JWT | `domain/notification.(*Handler).UpdateAlertRule` |
| `PUT` | `/api/v1/notification/read-all` | JWT | `domain/notification.(*Handler).MarkAllRead` |
| `GET` | `/api/v1/notification/unread-count` | JWT | `domain/notification.(*Handler).UnreadCount` |

### `operation-log`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/operation-log` | JWT | `domain/operationlog.(*Handler).List` |
| `POST` | `/api/v1/operation-log` | JWT | `domain/operationlog.(*Handler).Create` |
| `GET` | `/api/v1/operation-log/:id` | JWT | `domain/operationlog.(*Handler).Get` |

### `orchestration`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/orchestration/pipeline/config` | JWT | `domain/orchestration.(*Handler).ListConfigs` |
| `POST` | `/api/v1/orchestration/pipeline/config` | JWT | `domain/orchestration.(*Handler).CreateConfig` |
| `GET` | `/api/v1/orchestration/products/:id/pipeline` | JWT | `domain/orchestration.(*Handler).GetPipelineStatus` |
| `POST` | `/api/v1/orchestration/products/:id/pipeline/start` | JWT | `domain/orchestration.(*Handler).StartPipeline` |
| `POST` | `/api/v1/orchestration/products/:id/pipeline/step/:step/retry` | JWT | `domain/orchestration.(*Handler).RetryStep` |

### `order`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/order` | JWT + order.read | `domain/order.(*Handler).List` |
| `POST` | `/api/v1/order` | JWT + order.read | `domain/order.(*Handler).Create` |

### `order-import`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/order-import` | JWT + order.read | `domain/orderimport.(*Handler).List` |
| `POST` | `/api/v1/order-import` | JWT + order.read | `domain/orderimport.(*Handler).Create` |
| `DELETE` | `/api/v1/order-import/:id` | JWT + order.read | `domain/orderimport.(*Handler).Delete` |
| `GET` | `/api/v1/order-import/:id` | JWT + order.read | `domain/orderimport.(*Handler).Get` |
| `PUT` | `/api/v1/order-import/:id` | JWT + order.read | `domain/orderimport.(*Handler).Update` |
| `POST` | `/api/v1/order-import/:id/complete` | JWT + order.read | `domain/orderimport.(*Handler).Complete` |
| `POST` | `/api/v1/order-import/:id/process` | JWT + order.read | `domain/orderimport.(*Handler).Process` |
| `GET` | `/api/v1/order-import/summary` | JWT + order.read | `domain/orderimport.(*Handler).Summary` |

### `order`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `DELETE` | `/api/v1/order/:id` | JWT + order.read | `domain/order.(*Handler).Delete` |
| `GET` | `/api/v1/order/:id` | JWT + order.read | `domain/order.(*Handler).Get` |
| `PUT` | `/api/v1/order/:id` | JWT + order.read | `domain/order.(*Handler).Update` |
| `POST` | `/api/v1/order/:id/status` | JWT + order.read | `domain/order.(*Handler).UpdateStatus` |
| `GET` | `/api/v1/order/summary` | JWT + order.read | `domain/order.(*Handler).Summary` |

### `owner`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/owner/agent-activity` | JWT | `domain/owner.(*Handler).AgentActivity` |
| `GET` | `/api/v1/owner/decision-queue` | JWT | `domain/owner.(*Handler).GetDecisionQueue` |
| `GET` | `/api/v1/owner/pipeline-chain` | JWT | `domain/owner.(*Handler).PipelineChain` |
| `GET` | `/api/v1/owner/platform-sync` | JWT | `domain/owner.(*Handler).PlatformSyncStatus` |
| `GET` | `/api/v1/owner/risk-summary` | JWT | `domain/owner.(*Handler).RiskSummary` |
| `GET` | `/api/v1/owner/suggestions` | JWT | `domain/owner.(*Handler).Suggestions` |
| `POST` | `/api/v1/owner/suggestions/:id/feedback` | JWT | `domain/owner.(*Handler).Feedback` |

### `platform-fee`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/platform-fee` | JWT | `domain/platformfee.(*Handler).List` |
| `POST` | `/api/v1/platform-fee` | JWT | `domain/platformfee.(*Handler).Create` |
| `DELETE` | `/api/v1/platform-fee/:id` | JWT | `domain/platformfee.(*Handler).Delete` |
| `GET` | `/api/v1/platform-fee/:id` | JWT | `domain/platformfee.(*Handler).Get` |
| `PUT` | `/api/v1/platform-fee/:id` | JWT | `domain/platformfee.(*Handler).Update` |
| `POST` | `/api/v1/platform-fee/calculate` | JWT | `domain/platformfee.(*Handler).Calculate` |

### `platform-integrations`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/platform-integrations` | JWT | `domain/integrations.(*Handler).List` |
| `POST` | `/api/v1/platform-integrations` | JWT | `domain/integrations.(*Handler).Create` |
| `DELETE` | `/api/v1/platform-integrations/:id` | JWT | `domain/integrations.(*Handler).Delete` |
| `GET` | `/api/v1/platform-integrations/:id` | JWT | `domain/integrations.(*Handler).Get` |
| `PUT` | `/api/v1/platform-integrations/:id` | JWT | `domain/integrations.(*Handler).Update` |
| `GET` | `/api/v1/platform-integrations/:id/attributes` | JWT | `domain/integrations.(*Handler).ListAttributes` |
| `POST` | `/api/v1/platform-integrations/:id/attributes` | JWT | `domain/integrations.(*Handler).CreateAttribute` |
| `GET` | `/api/v1/platform-integrations/:id/categories` | JWT | `domain/integrations.(*Handler).ListCategories` |
| `POST` | `/api/v1/platform-integrations/:id/categories` | JWT | `domain/integrations.(*Handler).CreateCategory` |
| `GET` | `/api/v1/platform-integrations/:id/mode` | JWT | `domain/integrations.(*Handler).GetMode` |
| `PUT` | `/api/v1/platform-integrations/:id/mode` | JWT | `domain/integrations.(*Handler).UpdateMode` |
| `GET` | `/api/v1/platform-integrations/:id/ozon-products` | JWT | `domain/integrations.(*Handler).ListOzonProducts` |
| `POST` | `/api/v1/platform-integrations/:id/sync` | JWT | `domain/integrations.(*Handler).Sync` |
| `POST` | `/api/v1/platform-integrations/:id/test` | JWT | `domain/integrations.(*Handler).TestConnection` |
| `POST` | `/api/v1/platform-integrations/mock/seed` | JWT | `domain/integrations.RegisterRoutes.func1` |
| `POST` | `/api/v1/platform-integrations/publish-to-ozon` | JWT | `domain/integrations.(*Handler).PublishToOzon` |
| `POST` | `/api/v1/platform-integrations/write-back` | JWT | `domain/integrations.(*Handler).WriteBack` |
| `POST` | `/api/v1/platform-integrations/write-back/:ref-id/retry` | JWT | `domain/integrations.(*Handler).RetryWriteBack` |

### `platform-webhooks`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/platform-webhooks/config` | JWT | `domain/integrations.(*webhookHandler).GetConfig` |
| `POST` | `/api/v1/platform-webhooks/test-event` | JWT | `domain/integrations.(*webhookHandler).TestEvent` |

### `platforms`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/platforms` | JWT | `domain/platform.(*Handler).ListPlatforms` |
| `POST` | `/api/v1/platforms` | JWT | `domain/platform.(*Handler).CreatePlatform` |
| `DELETE` | `/api/v1/platforms/:id` | JWT | `domain/platform.(*Handler).DeletePlatform` |
| `GET` | `/api/v1/platforms/:id` | JWT | `domain/platform.(*Handler).GetPlatform` |
| `PUT` | `/api/v1/platforms/:id` | JWT | `domain/platform.(*Handler).UpdatePlatform` |

### `policy`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/policy/evaluate` | JWT | `domain/actionpolicy.(*Handler).Evaluate` |
| `GET` | `/api/v1/policy/rules` | JWT | `domain/actionpolicy.(*Handler).ListRules` |
| `POST` | `/api/v1/policy/rules` | JWT | `domain/actionpolicy.(*Handler).CreateRule` |
| `DELETE` | `/api/v1/policy/rules/:id` | JWT | `domain/actionpolicy.(*Handler).DeleteRule` |
| `GET` | `/api/v1/policy/rules/:id` | JWT | `domain/actionpolicy.(*Handler).GetRule` |
| `PUT` | `/api/v1/policy/rules/:id` | JWT | `domain/actionpolicy.(*Handler).UpdateRule` |
| `POST` | `/api/v1/policy/rules/:id/toggle` | JWT | `domain/actionpolicy.(*Handler).HandleToggleRule` |

### `prices`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/prices` | JWT + finance.read | `domain/price.(*Handler).ListPrices` |
| `POST` | `/api/v1/prices` | JWT + finance.read | `domain/price.(*Handler).SetPrice` |
| `DELETE` | `/api/v1/prices/:id` | JWT + finance.read | `domain/price.(*Handler).DeletePrice` |
| `GET` | `/api/v1/prices/:id` | JWT + finance.read | `domain/price.(*Handler).GetPrice` |
| `PUT` | `/api/v1/prices/:id` | JWT + finance.read | `domain/price.(*Handler).UpdatePrice` |

### `pricing-recommendations`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/pricing-recommendations` | JWT + finance.read | `domain/price.(*Handler).ListRecommendations` |
| `POST` | `/api/v1/pricing-recommendations/:id/apply` | JWT + finance.read | `domain/price.(*Handler).ApplyRecommendation` |
| `POST` | `/api/v1/pricing-recommendations/generate` | JWT + finance.read | `domain/price.(*Handler).GenerateRecommendation` |

### `pricing-strategies`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/pricing-strategies` | JWT + finance.read | `domain/price.(*Handler).ListPricingStrategies` |
| `POST` | `/api/v1/pricing-strategies` | JWT + finance.read | `domain/price.(*Handler).SavePricingStrategy` |
| `DELETE` | `/api/v1/pricing-strategies/:id` | JWT + finance.read | `domain/price.(*Handler).DeletePricingStrategy` |
| `GET` | `/api/v1/pricing-strategies/:id` | JWT + finance.read | `domain/price.(*Handler).GetPricingStrategy` |
| `PUT` | `/api/v1/pricing-strategies/:id` | JWT + finance.read | `domain/price.(*Handler).UpdatePricingStrategy` |

### `product-analysis`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/product-analysis/analyses` | JWT | `domain/productanalysis.(*Handler).ListAnalyses` |
| `GET` | `/api/v1/product-analysis/analyses/:id` | JWT | `domain/productanalysis.(*Handler).GetAnalysis` |
| `POST` | `/api/v1/product-analysis/analyses/:id/feedback` | JWT | `domain/productanalysis.(*Handler).RecordFeedback` |
| `POST` | `/api/v1/product-analysis/analyze` | JWT | `domain/productanalysis.(*Handler).Analyze` |

### `product-hub`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/product-hub` | JWT + product.read | `domain/producthub.(*MasterHandler).List` |
| `POST` | `/api/v1/product-hub` | JWT + product.read | `domain/producthub.(*MasterHandler).Create` |
| `DELETE` | `/api/v1/product-hub/:id` | JWT + product.read | `domain/producthub.(*MasterHandler).Delete` |
| `GET` | `/api/v1/product-hub/:id` | JWT + product.read | `domain/producthub.(*MasterHandler).Get` |
| `PUT` | `/api/v1/product-hub/:id` | JWT + product.read | `domain/producthub.(*MasterHandler).Update` |
| `GET` | `/api/v1/product-hub/:id/costs` | JWT + product.read | `domain/producthub.(*Handler).ListCosts` |
| `GET` | `/api/v1/product-hub/:id/evidence` | JWT + product.read | `domain/producthub.(*Handler).GetEvidence` |
| `GET` | `/api/v1/product-hub/:id/hub` | JWT + product.read | `domain/producthub.(*HubHandler).GetHub` |
| `GET` | `/api/v1/product-hub/:id/offers` | JWT + product.read | `domain/producthub.(*Handler).ListOffers` |
| `GET` | `/api/v1/product-hub/:id/samples` | JWT + product.read | `domain/producthub.(*Handler).ListSamples` |
| `POST` | `/api/v1/product-hub/:id/transition` | JWT + product.read | `domain/producthub.(*MasterHandler).TransitionLifecycle` |
| `GET` | `/api/v1/product-hub/:id/variants` | JWT + product.read | `domain/producthub.(*Handler).ListVariants` |
| `POST` | `/api/v1/product-hub/costs` | JWT + product.read | `domain/producthub.(*Handler).CreateCost` |
| `POST` | `/api/v1/product-hub/costs/:costId/confirm` | JWT + product.read | `domain/producthub.(*Handler).ConfirmCost` |
| `POST` | `/api/v1/product-hub/offers` | JWT + product.read | `domain/producthub.(*Handler).CreateOffer` |
| `POST` | `/api/v1/product-hub/samples` | JWT + product.read | `domain/producthub.(*Handler).CreateSample` |
| `POST` | `/api/v1/product-hub/variants` | JWT + product.read | `domain/producthub.(*Handler).CreateVariant` |

### `product-master`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/product-master` | JWT + product.read | `domain/sku.(*Handler).ListProducts` |
| `POST` | `/api/v1/product-master` | JWT + product.read | `domain/sku.(*Handler).CreateProduct` |
| `DELETE` | `/api/v1/product-master/:id` | JWT + product.read | `domain/sku.(*Handler).DeleteProduct` |
| `GET` | `/api/v1/product-master/:id` | JWT + product.read | `domain/sku.(*Handler).GetProduct` |
| `PUT` | `/api/v1/product-master/:id` | JWT + product.read | `domain/sku.(*Handler).UpdateProduct` |
| `GET` | `/api/v1/product-master/:id/skus` | JWT + product.read | `domain/sku.(*Handler).ListSkusByProduct` |
| `GET` | `/api/v1/product-master/:id/specs` | JWT + product.read | `domain/sku.(*Handler).ListSpecs` |
| `POST` | `/api/v1/product-master/:id/specs` | JWT + product.read | `domain/sku.(*Handler).CreateSpec` |
| `DELETE` | `/api/v1/product-master/:id/specs/:spec_id` | JWT + product.read | `domain/sku.(*Handler).DeleteSpec` |
| `PUT` | `/api/v1/product-master/:id/specs/:spec_id` | JWT + product.read | `domain/sku.(*Handler).UpdateSpec` |
| `POST` | `/api/v1/product-master/:id/specs/:spec_id/values` | JWT + product.read | `domain/sku.(*Handler).CreateSpecValue` |

### `product-suppliers`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/product-suppliers` | JWT | `domain/supplier.(*Handler).ListProductSuppliers` |
| `POST` | `/api/v1/product-suppliers` | JWT | `domain/supplier.(*Handler).CreateProductSupplier` |
| `DELETE` | `/api/v1/product-suppliers/:id` | JWT | `domain/supplier.(*Handler).DeleteProductSupplier` |
| `PUT` | `/api/v1/product-suppliers/:id` | JWT | `domain/supplier.(*Handler).UpdateProductSupplier` |
| `GET` | `/api/v1/product-suppliers/comparison` | JWT | `domain/supplier.(*Handler).GetSupplierComparison` |

### `products`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/products/360/summary` | JWT + product.read | `domain/producthub.(*Handler).GetProductSummary` |
| `POST` | `/api/v1/products/:id/decisions` | JWT + product.read | `domain/producthub.(*Handler).RecordDecision` |
| `POST` | `/api/v1/products/:id/discover-relations` | JWT + product.read | `domain/producthub.(*Handler).AutoDiscoverRelations` |
| `GET` | `/api/v1/products/:id/freshness` | JWT + product.read | `domain/producthub.(*Handler).GetProductFreshness` |
| `POST` | `/api/v1/products/:id/freshness/verify` | JWT + product.read | `domain/producthub.(*Handler).VerifyDimension` |
| `GET` | `/api/v1/products/:id/relations` | JWT + product.read | `domain/producthub.(*Handler).GetRelatedProducts` |
| `GET` | `/api/v1/products/:id/versions` | JWT + product.read | `domain/producthub.(*Handler).ListVersions` |
| `GET` | `/api/v1/products/:id/versions/:versionId` | JWT + product.read | `domain/producthub.(*Handler).GetVersion` |
| `POST` | `/api/v1/products/:id/versions/:versionId/rollback` | JWT + product.read | `domain/producthub.(*Handler).Rollback` |
| `GET` | `/api/v1/products/decision` | JWT + product.read | `domain/producthub.(*Handler).ListRecentDecisions` |
| `GET` | `/api/v1/products/freshness/stale` | JWT + product.read | `domain/producthub.(*Handler).ListStaleProducts` |
| `POST` | `/api/v1/products/relations` | JWT + product.read | `domain/producthub.(*Handler).CreateRelation` |
| `DELETE` | `/api/v1/products/relations/:id` | JWT + product.read | `domain/producthub.(*Handler).DeleteRelation` |

### `profit`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/profit/evidence-card/:productId` | JWT | `domain/profit.(*EvidenceHandler).GetEvidenceCard` |
| `POST` | `/api/v1/profit/order/:orderId/calculate` | JWT | `domain/profit.(*Handler).CalculateOrderProfit` |
| `GET` | `/api/v1/profit/summaries` | JWT | `domain/profit.(*Handler).ListSummaries` |
| `GET` | `/api/v1/profit/summary/:productId` | JWT | `domain/profit.(*Handler).Summary` |

### `purchase`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/purchase/orders` | JWT | `domain/purchase.(*Handler).ListOrders` |
| `POST` | `/api/v1/purchase/orders` | JWT | `domain/purchase.(*Handler).CreateOrder` |
| `GET` | `/api/v1/purchase/orders/:id` | JWT | `domain/purchase.(*Handler).GetOrder` |
| `POST` | `/api/v1/purchase/orders/:id/approve` | JWT | `domain/purchase.(*Handler).ApproveOrder` |
| `POST` | `/api/v1/purchase/orders/:id/cancel` | JWT | `domain/purchase.(*Handler).CancelOrder` |
| `POST` | `/api/v1/purchase/orders/:id/receive` | JWT | `domain/purchase.(*Handler).ReceiveOrder` |
| `GET` | `/api/v1/purchase/suggestions` | JWT | `domain/purchase.(*Handler).ListSuggestions` |
| `POST` | `/api/v1/purchase/suggestions/generate` | JWT | `domain/purchase.(*Handler).GenerateSuggestions` |

### `rbac`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/rbac/current/permissions` | JWT | `rbac.(*Handler).GetCurrentUserPermissions` |
| `GET` | `/api/v1/rbac/permissions` | JWT + rbac.manage | `rbac.(*Handler).ListPermissions` |
| `POST` | `/api/v1/rbac/permissions` | JWT + rbac.manage | `rbac.(*Handler).CreatePermission` |
| `DELETE` | `/api/v1/rbac/permissions/:id` | JWT + rbac.manage | `rbac.(*Handler).DeletePermission` |
| `GET` | `/api/v1/rbac/permissions/:id` | JWT + rbac.manage | `rbac.(*Handler).GetPermission` |
| `PUT` | `/api/v1/rbac/permissions/:id` | JWT + rbac.manage | `rbac.(*Handler).UpdatePermission` |
| `GET` | `/api/v1/rbac/roles` | JWT + rbac.manage | `rbac.(*Handler).ListRoles` |
| `POST` | `/api/v1/rbac/roles` | JWT + rbac.manage | `rbac.(*Handler).CreateRole` |
| `DELETE` | `/api/v1/rbac/roles/:id` | JWT + rbac.manage | `rbac.(*Handler).DeleteRole` |
| `GET` | `/api/v1/rbac/roles/:id` | JWT + rbac.manage | `rbac.(*Handler).GetRole` |
| `PUT` | `/api/v1/rbac/roles/:id` | JWT + rbac.manage | `rbac.(*Handler).UpdateRole` |
| `GET` | `/api/v1/rbac/roles/:id/permissions` | JWT + rbac.manage | `rbac.(*Handler).GetRolePermissions` |
| `POST` | `/api/v1/rbac/roles/:id/permissions` | JWT + rbac.manage | `rbac.(*Handler).AssignRolePermissions` |
| `GET` | `/api/v1/rbac/users/:id/roles` | JWT + rbac.manage | `rbac.(*Handler).GetUserRoles` |
| `POST` | `/api/v1/rbac/users/:id/roles` | JWT + rbac.manage | `rbac.(*Handler).AssignUserRoles` |

### `reliability`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/reliability/budget` | JWT | `domain/reliability.(*Handler).GetBudget` |
| `PUT` | `/api/v1/reliability/budget` | JWT | `domain/reliability.(*Handler).SetBudget` |

### `report`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/report/daily` | JWT + report.read | `domain/report.(*Handler).DailyReport` |
| `GET` | `/api/v1/report/inventory` | JWT + report.read | `domain/report.(*Handler).Inventory` |
| `GET` | `/api/v1/report/platform-fee` | JWT + report.read | `domain/report.(*Handler).PlatformFee` |
| `GET` | `/api/v1/report/profit` | JWT + report.read | `domain/report.(*Handler).Profit` |
| `GET` | `/api/v1/report/sales` | JWT + report.read | `domain/report.(*Handler).Sales` |
| `GET` | `/api/v1/report/settlement` | JWT + report.read | `domain/report.(*Handler).Settlement` |
| `GET` | `/api/v1/report/weekly` | JWT + report.read | `domain/report.(*Handler).WeeklyReport` |

### `search`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/search` | JWT | `domain/search.(*Handler).Search` |
| `GET` | `/api/v1/search/recent` | JWT | `domain/search.(*Handler).Recent` |

### `sentiment`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/sentiment/:productId` | JWT | `domain/sentiment.(*Handler).GetProductSentiment` |
| `POST` | `/api/v1/sentiment/:productId/refresh` | JWT | `domain/sentiment.(*Handler).RefreshSentiment` |
| `GET` | `/api/v1/sentiment/negative` | JWT | `domain/sentiment.(*Handler).ListNegativeSentiment` |

### `settings`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/settings/llm` | JWT | `domain/settings.(*Handler).GetLLM` |
| `PUT` | `/api/v1/settings/llm` | JWT | `domain/settings.(*Handler).UpdateLLM` |

### `settlement`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/settlement` | JWT + settlement.read | `domain/settlement.(*Handler).List` |
| `POST` | `/api/v1/settlement` | JWT + settlement.read | `domain/settlement.(*Handler).Create` |
| `DELETE` | `/api/v1/settlement/:id` | JWT + settlement.read | `domain/settlement.(*Handler).Delete` |
| `GET` | `/api/v1/settlement/:id` | JWT + settlement.read | `domain/settlement.(*Handler).Get` |
| `PUT` | `/api/v1/settlement/:id` | JWT + settlement.read | `domain/settlement.(*Handler).Update` |
| `GET` | `/api/v1/settlement/:id/items` | JWT + settlement.read | `domain/settlement.(*Handler).ListItems` |
| `POST` | `/api/v1/settlement/:id/items` | JWT + settlement.read | `domain/settlement.(*Handler).AddItem` |
| `POST` | `/api/v1/settlement/:id/reconcile` | JWT + settlement.read | `domain/settlement.(*Handler).Reconcile` |
| `PUT` | `/api/v1/settlement/items/:item_id/reconciliation` | JWT + settlement.read | `domain/settlement.(*Handler).UpdateItemReconciliation` |
| `POST` | `/api/v1/settlement/recalculate` | JWT + settlement.read | `domain/settlement.(*Handler).RecalculateAll` |
| `GET` | `/api/v1/settlement/summary` | JWT + settlement.read | `domain/settlement.(*Handler).Summary` |

### `shipping`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/shipping/bill-batches` | JWT + shipping.read | `domain/shipping.(*Handler).ListBillBatches` |
| `POST` | `/api/v1/shipping/bill-batches` | JWT + shipping.read | `domain/shipping.(*Handler).CreateBillBatch` |
| `DELETE` | `/api/v1/shipping/bill-batches/:id` | JWT + shipping.read | `domain/shipping.(*Handler).DeleteBillBatch` |
| `GET` | `/api/v1/shipping/bill-batches/:id` | JWT + shipping.read | `domain/shipping.(*Handler).GetBillBatch` |
| `GET` | `/api/v1/shipping/bill-batches/:id/anomalies` | JWT + shipping.read | `domain/shipping.(*Handler).ListBillAnomalies` |
| `GET` | `/api/v1/shipping/bill-batches/:id/items` | JWT + shipping.read | `domain/shipping.(*Handler).ListBillItems` |
| `POST` | `/api/v1/shipping/bill-batches/:id/reconcile` | JWT + shipping.read | `domain/shipping.(*Handler).ReconcileBillBatch` |
| `POST` | `/api/v1/shipping/bill-batches/import` | JWT + shipping.read | `domain/shipping.(*Handler).ImportBill` |
| `PUT` | `/api/v1/shipping/bill-items/:id/review` | JWT + shipping.read | `domain/shipping.(*Handler).ReviewBillItem` |
| `GET` | `/api/v1/shipping/carrier-performance` | JWT + shipping.read | `domain/shipping.(*Handler).GetCarrierPerformance` |
| `GET` | `/api/v1/shipping/carriers` | JWT + shipping.read | `domain/shipping.(*Handler).ListCarriers` |
| `POST` | `/api/v1/shipping/carriers/:code/quote` | JWT + shipping.read | `domain/shipping.(*Handler).CarrierQuote` |
| `GET` | `/api/v1/shipping/channels` | JWT + shipping.read | `domain/shipping.(*Handler).ListChannels` |
| `POST` | `/api/v1/shipping/channels` | JWT + shipping.read | `domain/shipping.(*Handler).CreateChannel` |
| `DELETE` | `/api/v1/shipping/channels/:id` | JWT + shipping.read | `domain/shipping.(*Handler).DeleteChannel` |
| `GET` | `/api/v1/shipping/channels/:id` | JWT + shipping.read | `domain/shipping.(*Handler).GetChannel` |
| `PUT` | `/api/v1/shipping/channels/:id` | JWT + shipping.read | `domain/shipping.(*Handler).UpdateChannel` |
| `GET` | `/api/v1/shipping/providers` | JWT + shipping.read | `domain/shipping.(*Handler).ListProviders` |
| `POST` | `/api/v1/shipping/providers` | JWT + shipping.read | `domain/shipping.(*Handler).CreateProvider` |
| `DELETE` | `/api/v1/shipping/providers/:id` | JWT + shipping.read | `domain/shipping.(*Handler).DeleteProvider` |
| `GET` | `/api/v1/shipping/providers/:id` | JWT + shipping.read | `domain/shipping.(*Handler).GetProvider` |
| `PUT` | `/api/v1/shipping/providers/:id` | JWT + shipping.read | `domain/shipping.(*Handler).UpdateProvider` |
| `POST` | `/api/v1/shipping/quote` | JWT + shipping.read | `domain/shipping.(*Handler).Quote` |
| `POST` | `/api/v1/shipping/quote-unified` | JWT + shipping.read | `domain/shipping.(*Handler).QuoteUnified` |
| `GET` | `/api/v1/shipping/rules` | JWT + shipping.read | `domain/shipping.(*Handler).ListRules` |
| `POST` | `/api/v1/shipping/rules` | JWT + shipping.read | `domain/shipping.(*Handler).CreateRule` |
| `DELETE` | `/api/v1/shipping/rules/:id` | JWT + shipping.read | `domain/shipping.(*Handler).DeleteRule` |
| `GET` | `/api/v1/shipping/rules/:id/versions` | JWT + shipping.read | `domain/shipping.(*Handler).ListRuleVersions` |
| `GET` | `/api/v1/shipping/snapshots` | JWT + shipping.read | `domain/shipping.(*Handler).ListSnapshots` |
| `POST` | `/api/v1/shipping/snapshots` | JWT + shipping.read | `domain/shipping.(*Handler).CreateSnapshot` |
| `GET` | `/api/v1/shipping/snapshots/:orderId` | JWT + shipping.read | `domain/shipping.(*Handler).GetSnapshot` |
| `GET` | `/api/v1/shipping/tracking` | JWT + shipping.read | `domain/shipping.(*Handler).ListTracking` |
| `POST` | `/api/v1/shipping/tracking` | JWT + shipping.read | `domain/shipping.(*Handler).CreateTracking` |
| `PUT` | `/api/v1/shipping/tracking/:id/event` | JWT + shipping.read | `domain/shipping.(*Handler).UpdateTrackingEvent` |
| `PUT` | `/api/v1/shipping/tracking/:id/exception` | JWT + shipping.read | `domain/shipping.(*Handler).MarkTrackingException` |
| `GET` | `/api/v1/shipping/tracking/:orderId` | JWT + shipping.read | `domain/shipping.(*Handler).GetTracking` |
| `GET` | `/api/v1/shipping/zones` | JWT + shipping.read | `domain/shipping.(*Handler).ListZones` |
| `POST` | `/api/v1/shipping/zones` | JWT + shipping.read | `domain/shipping.(*Handler).CreateZone` |
| `DELETE` | `/api/v1/shipping/zones/:id` | JWT + shipping.read | `domain/shipping.(*Handler).DeleteZone` |

### `skus`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/skus` | JWT + product.read | `domain/sku.(*Handler).ListSkus` |
| `POST` | `/api/v1/skus` | JWT + product.read | `domain/sku.(*Handler).CreateSku` |
| `DELETE` | `/api/v1/skus/:id` | JWT + product.read | `domain/sku.(*Handler).DeleteSku` |
| `GET` | `/api/v1/skus/:id` | JWT + product.read | `domain/sku.(*Handler).GetSku` |
| `PUT` | `/api/v1/skus/:id` | JWT + product.read | `domain/sku.(*Handler).UpdateSku` |
| `GET` | `/api/v1/skus/:id/current-price` | JWT + product.read | `domain/price.(*Handler).GetCurrentPrice` |
| `GET` | `/api/v1/skus/:id/price-history` | JWT + product.read | `domain/price.(*Handler).PriceHistory` |
| `GET` | `/api/v1/skus/:id/prices` | JWT + product.read | `domain/price.(*Handler).ListPricesBySKU` |

### `sourcing-1688`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/sourcing-1688` | JWT | `domain/sourcing1688.(*Handler).List` |
| `POST` | `/api/v1/sourcing-1688` | JWT | `domain/sourcing1688.(*Handler).Create` |
| `DELETE` | `/api/v1/sourcing-1688/:id` | JWT | `domain/sourcing1688.(*Handler).Delete` |
| `GET` | `/api/v1/sourcing-1688/:id` | JWT | `domain/sourcing1688.(*Handler).Get` |
| `PUT` | `/api/v1/sourcing-1688/:id` | JWT | `domain/sourcing1688.(*Handler).Update` |
| `POST` | `/api/v1/sourcing-1688/:id/import` | JWT | `domain/sourcing1688.(*Handler).Import` |
| `POST` | `/api/v1/sourcing-1688/:id/reject` | JWT | `domain/sourcing1688.(*Handler).Reject` |
| `GET` | `/api/v1/sourcing-1688/summary` | JWT | `domain/sourcing1688.(*Handler).Summary` |

### `sourcing`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/sourcing/fetch` | JWT | `domain/sourcing.(*Handler).Fetch` |
| `GET` | `/api/v1/sourcing/keyword-trends` | JWT | `domain/sourcing.(*Handler).KeywordTrends` |
| `GET` | `/api/v1/sourcing/market-overview` | JWT | `domain/sourcing.(*Handler).MarketOverview` |
| `GET` | `/api/v1/sourcing/market-trends` | JWT | `domain/sourcing.(*Handler).MarketTrends` |
| `GET` | `/api/v1/sourcing/recommendations` | JWT | `domain/sourcing.(*Handler).ListRecommendations` |

### `spec-values`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `DELETE` | `/api/v1/spec-values/:id` | JWT + product.read | `domain/sku.(*Handler).DeleteSpecValue` |
| `PUT` | `/api/v1/spec-values/:id` | JWT + product.read | `domain/sku.(*Handler).UpdateSpecValue` |

### `stores`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/stores` | JWT | `domain/platform.(*Handler).ListStores` |
| `POST` | `/api/v1/stores` | JWT | `domain/platform.(*Handler).CreateStore` |
| `DELETE` | `/api/v1/stores/:id` | JWT | `domain/platform.(*Handler).DeleteStore` |
| `GET` | `/api/v1/stores/:id` | JWT | `domain/platform.(*Handler).GetStore` |
| `PUT` | `/api/v1/stores/:id` | JWT | `domain/platform.(*Handler).UpdateStore` |

### `suppliers`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/suppliers` | JWT | `domain/supplier.(*Handler).List` |
| `POST` | `/api/v1/suppliers` | JWT | `domain/supplier.(*Handler).Create` |
| `DELETE` | `/api/v1/suppliers/:id` | JWT | `domain/supplier.(*Handler).Delete` |
| `GET` | `/api/v1/suppliers/:id` | JWT | `domain/supplier.(*Handler).Get` |
| `PUT` | `/api/v1/suppliers/:id` | JWT | `domain/supplier.(*Handler).Update` |
| `PUT` | `/api/v1/suppliers/:id/kpi-score` | JWT | `domain/supplier.(*Handler).UpdateScoreManual` |
| `POST` | `/api/v1/suppliers/:id/recalculate` | JWT | `domain/supplier.(*Handler).RecalculateScore` |
| `GET` | `/api/v1/suppliers/:id/score` | JWT | `domain/supplier.(*Handler).GetScore` |
| `GET` | `/api/v1/suppliers/:id/score-history` | JWT | `domain/supplier.(*Handler).GetScoreHistory` |
| `POST` | `/api/v1/suppliers/:id/score-snapshot` | JWT | `domain/supplier.(*Handler).RecordScoreSnapshot` |
| `GET` | `/api/v1/suppliers/scoreboard` | JWT | `domain/supplier.(*Handler).ListScoreboard` |

### `supply-chain`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/supply-chain/flows` | JWT | `domain/supplychain.(*Handler).List` |
| `POST` | `/api/v1/supply-chain/flows` | JWT | `domain/supplychain.(*Handler).Create` |
| `GET` | `/api/v1/supply-chain/flows/:id` | JWT | `domain/supplychain.(*Handler).Get` |
| `PUT` | `/api/v1/supply-chain/flows/:id` | JWT | `domain/supplychain.(*Handler).Update` |
| `GET` | `/api/v1/supply-chain/flows/:id/events` | JWT | `domain/supplychain.(*Handler).GetEvents` |
| `GET` | `/api/v1/supply-chain/tracking` | JWT | `domain/supplychain.(*TrackingHandler).List` |
| `POST` | `/api/v1/supply-chain/tracking` | JWT | `domain/supplychain.(*TrackingHandler).Create` |
| `GET` | `/api/v1/supply-chain/tracking/:id` | JWT | `domain/supplychain.(*TrackingHandler).Get` |
| `PUT` | `/api/v1/supply-chain/tracking/:id/status` | JWT | `domain/supplychain.(*TrackingHandler).UpdateStatus` |
| `POST` | `/api/v1/supply-chain/tracking/:id/sync` | JWT | `domain/supplychain.(*TrackingHandler).SyncFromCarrier` |
| `GET` | `/api/v1/supply-chain/tracking/flow/:flowId` | JWT | `domain/supplychain.(*TrackingHandler).GetByFlow` |

### `support`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/support/blacklist` | JWT | `domain/support.(*Handler).ListBlacklist` |
| `POST` | `/api/v1/support/blacklist` | JWT | `domain/support.(*Handler).AddBlacklist` |
| `DELETE` | `/api/v1/support/blacklist/:id` | JWT | `domain/support.(*Handler).DeleteBlacklist` |
| `GET` | `/api/v1/support/blacklist/check` | JWT | `domain/support.(*Handler).CheckBlacklist` |
| `GET` | `/api/v1/support/conversations` | JWT | `domain/support.(*Handler).ListConversations` |
| `POST` | `/api/v1/support/conversations` | JWT | `domain/support.(*Handler).CreateConversation` |
| `DELETE` | `/api/v1/support/conversations/:id` | JWT | `domain/support.(*Handler).DeleteConversation` |
| `GET` | `/api/v1/support/conversations/:id` | JWT | `domain/support.(*Handler).GetConversation` |
| `PUT` | `/api/v1/support/conversations/:id` | JWT | `domain/support.(*Handler).UpdateConversation` |
| `POST` | `/api/v1/support/conversations/:id/close` | JWT | `domain/support.(*Handler).CloseConversation` |
| `GET` | `/api/v1/support/conversations/:id/messages` | JWT | `domain/support.(*Handler).GetMessages` |
| `POST` | `/api/v1/support/conversations/:id/reply` | JWT | `domain/support.(*Handler).SendReply` |
| `GET` | `/api/v1/support/templates` | JWT | `domain/support.(*Handler).ListTemplates` |
| `POST` | `/api/v1/support/templates` | JWT | `domain/support.(*Handler).CreateTemplate` |
| `DELETE` | `/api/v1/support/templates/:id` | JWT | `domain/support.(*Handler).DeleteTemplate` |
| `GET` | `/api/v1/support/templates/:id` | JWT | `domain/support.(*Handler).GetTemplate` |
| `PUT` | `/api/v1/support/templates/:id` | JWT | `domain/support.(*Handler).UpdateTemplate` |

### `tariff`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/tariff` | JWT | `domain/tariff.(*Handler).List` |
| `POST` | `/api/v1/tariff` | JWT | `domain/tariff.(*Handler).Create` |
| `DELETE` | `/api/v1/tariff/:id` | JWT | `domain/tariff.(*Handler).Delete` |
| `GET` | `/api/v1/tariff/:id` | JWT | `domain/tariff.(*Handler).Get` |
| `PUT` | `/api/v1/tariff/:id` | JWT | `domain/tariff.(*Handler).Update` |
| `POST` | `/api/v1/tariff/decide` | JWT | `domain/tariff.(*Handler).Decide` |

### `trust-scores`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/trust-scores` | JWT | `domain/trustscore.(*Handler).List` |
| `GET` | `/api/v1/trust-scores/:agent_id` | JWT | `domain/trustscore.(*Handler).GetByAgent` |
| `PUT` | `/api/v1/trust-scores/:agent_id/level` | JWT | `domain/trustscore.(*Handler).UpdateLevel` |
| `POST` | `/api/v1/trust-scores/auto-upgrade` | JWT | `domain/trustscore.(*Handler).AutoUpgrade` |
| `POST` | `/api/v1/trust-scores/eligible` | JWT | `domain/trustscore.(*Handler).Eligible` |
| `POST` | `/api/v1/trust-scores/recalculate` | JWT | `domain/trustscore.(*Handler).Recalculate` |
| `GET` | `/api/v1/trust-scores/summary` | JWT | `domain/trustscore.(*Handler).Summary` |

### `webhooks`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `POST` | `/api/v1/webhooks/:platform` | Webhook signature | `domain/integrations.(*webhookHandler).ReceiveWebhook` |

### `workflow`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/workflow/defs` | JWT | `domain/workflow.(*Handler).ListDefs` |
| `POST` | `/api/v1/workflow/defs` | JWT | `domain/workflow.(*Handler).CreateDef` |
| `POST` | `/api/v1/workflow/defs/:defId/start` | JWT | `domain/workflow.(*Handler).StartRun` |
| `DELETE` | `/api/v1/workflow/defs/:id` | JWT | `domain/workflow.(*Handler).DeleteDef` |
| `GET` | `/api/v1/workflow/defs/:id` | JWT | `domain/workflow.(*Handler).GetDef` |
| `PUT` | `/api/v1/workflow/defs/:id` | JWT | `domain/workflow.(*Handler).UpdateDef` |
| `GET` | `/api/v1/workflow/monitor` | JWT | `domain/workflow.(*Handler).GetMonitor` |
| `GET` | `/api/v1/workflow/monitor/stats` | JWT | `domain/workflow.(*Handler).GetMonitorStats` |
| `GET` | `/api/v1/workflow/runs` | JWT | `domain/workflow.(*Handler).ListRuns` |
| `GET` | `/api/v1/workflow/runs/:id` | JWT | `domain/workflow.(*Handler).GetRun` |
| `POST` | `/api/v1/workflow/runs/:id/advance` | JWT | `domain/workflow.(*Handler).AdvanceStep` |
| `POST` | `/api/v1/workflow/runs/:id/pause` | JWT | `domain/workflow.(*Handler).PauseRun` |
| `POST` | `/api/v1/workflow/runs/:id/resume` | JWT | `domain/workflow.(*Handler).ResumeRun` |
| `POST` | `/api/v1/workflow/runs/:id/retry` | JWT | `domain/workflow.(*Handler).RetryRun` |

### `workflows`

| 方法 | 路径 | 访问 | 处理器 |
|---|---|---|---|
| `GET` | `/api/v1/workflows` | JWT | `domain/workflow.(*Handler).ListWorkflows` |
| `POST` | `/api/v1/workflows` | JWT | `domain/workflow.(*Handler).CreateWorkflow` |
| `GET` | `/api/v1/workflows/:id` | JWT | `domain/workflow.(*Handler).GetWorkflow` |
| `POST` | `/api/v1/workflows/runs/:id/approve` | JWT | `domain/workflow.(*Handler).ApproveStep` |
| `POST` | `/api/v1/workflows/runs/:id/reject` | JWT | `domain/workflow.(*Handler).RejectStep` |

## `/api/v1` 之外的已注册入口

| 方法 | 路径 | 访问/用途 |
|---|---|---|
| `GET` | `/api/health` | 公共服务健康检查 |
| `GET` | `/metrics` | Prometheus 指标；仅在 `metrics.enabled` 开启时注册 |
| `GET` | `/swagger/*any` | Swagger UI |
| `GET` | `/ws` | JWT 通过 WebSocket 处理器校验；AI 流式与实时更新 |
| `GET` | `/ws/extension` | JWT 通过扩展 WebSocket 处理器校验；浏览器扩展采集通道 |

## 条件注册路由

| 方法 | 路径 | 注册条件 | 访问 | 处理器 |
|---|---|---|---|---|
| `GET` | `/metrics` | `metrics.enabled=true` | 由部署网络边界保护；路由本身未挂 JWT | `middleware.MetricsHandler` |

## 已知边界与验证方式

- **actual：** 在本文所述基线配置下，当前代码构建出的 Gin 路由数量为 687，`/api/v1` 为 683。
- **implemented：** 表中路由及处理器已注册。
- **unknown：** 未逐条执行 683 个接口，未连接生产数据库、真实平台账号或生产服务器。
- **重要限制：** handler 存在不证明其不是 mock、stub、空成功或确定性回退；参见 [项目真相审计](research/project-truth-audit-2026-07-11.md)。
- 重新核验路由时，应在 `backend-go` 内构建 `httpx.NewRouter`，调用 Gin 的 `Engine.Routes()`，然后比较方法与路径集合。

## 相关文档

- [API 快速参考](reference-api-quick.md)
- [模块目录](reference-module-catalog.md)
- [权限与审计](PERMISSIONS_AND_AUDIT.md)
- [Owner 自用经营方向](SELF_USE_OPERATING_DIRECTION.md)
