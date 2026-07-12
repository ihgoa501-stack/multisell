# Module: `aftersales`

Package: `backend-go/internal/domain/aftersales/`

**Base mount prefix:** `/api/v1`

## API Routes

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

## Models

### `AfterSalesOrder`
**DB table:** `after_sales_order`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OrderID` | `int64` | `order_id` | `order_id` | NOT NULL |
| `ItemID` | `*int64` | `item_id,omitempty` | `item_id` |  |
| `SkuID` | `*int64` | `sku_id,omitempty` | `sku_id` |  |
| `ReturnQuantity` | `int` | `return_quantity` | `return_quantity` | default:0 |
| `Reason` | `string` | `reason` | `reason` |  |
| `Status` | `string` | `status` | `status` | default:pending |
| `RefundAmount` | `float64` | `refund_amount` | `refund_amount` | default:0 |
| `InspectionResult` | `string` | `inspection_result` | `inspection_result` |  |
| `RejectionReason` | `string` | `rejection_reason` | `rejection_reason` |  |
| `CreatedBy` | `string` | `created_by` | `created_by` |  |
| `ApprovedBy` | `string` | `approved_by` | `approved_by` |  |
| `ApprovedAt` | `*time.Time` | `approved_at,omitempty` | `approved_at` |  |
| `RejectedBy` | `string` | `rejected_by` | `rejected_by` |  |
| `RejectedAt` | `*time.Time` | `rejected_at,omitempty` | `rejected_at` |  |
| `ReceivedBy` | `string` | `received_by` | `received_by` |  |
| `ReceivedAt` | `*time.Time` | `received_at,omitempty` | `received_at` |  |
| `RefundedBy` | `string` | `refunded_by` | `refunded_by` |  |
| `RefundedAt` | `*time.Time` | `refunded_at,omitempty` | `refunded_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `CreateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderID` | `int64` | `order_id` | `—` |  |
| `ItemID` | `*int64` | `item_id` | `—` |  |
| `SkuID` | `*int64` | `sku_id` | `—` |  |
| `ReturnQuantity` | `*int` | `return_quantity` | `—` |  |
| `Reason` | `string` | `reason` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `RefundAmount` | `*float64` | `refund_amount` | `—` |  |
| `CreatedBy` | `string` | `created_by` | `—` |  |

### `UpdateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ReturnQuantity` | `*int` | `return_quantity` | `—` |  |
| `Reason` | `*string` | `reason` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |
| `RefundAmount` | `*float64` | `refund_amount` | `—` |  |
| `InspectionResult` | `*string` | `inspection_result` | `—` |  |
| `RejectionReason` | `*string` | `rejection_reason` | `—` |  |

### `ListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Search` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |
| `OrderID` | `*int64` | `` | `—` |  |

### `ApproveInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ApprovedBy` | `string` | `approved_by` | `—` |  |
| `InspectionResult` | `string` | `inspection_result` | `—` |  |

### `RejectInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `RejectedBy` | `string` | `rejected_by` | `—` |  |
| `RejectionReason` | `string` | `rejection_reason` | `—` |  |

### `ReceiveInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ReceivedBy` | `string` | `received_by` | `—` |  |

### `RefundInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `RefundedBy` | `string` | `refunded_by` | `—` |  |
| `RefundAmount` | `float64` | `refund_amount` | `—` |  |

### `Summary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Total` | `int64` | `total` | `—` |  |
| `ByStatus` | `map[string]int64` | `by_status` | `—` |  |
| `TotalRefunded` | `float64` | `total_refunded` | `—` |  |

### `DisputeCase`
**DB table:** `dispute_case`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OrderID` | `*int64` | `order_id,omitempty` | `order_id` |  |
| `TransactionID` | `string` | `transaction_id` | `transaction_id` | NOT NULL |
| `Platform` | `string` | `platform` | `platform` | NOT NULL |
| `ClaimType` | `string` | `claim_type` | `claim_type` | NOT NULL |
| `Amount` | `float64` | `amount` | `amount` | default:0 |
| `Status` | `string` | `status` | `status` | default:pending |
| `Evidence` | `string` | `evidence,omitempty` | `evidence` |  |
| `DecisionScore` | `float64` | `decision_score` | `decision_score` | default:0 |
| `AiReason` | `string` | `ai_reason,omitempty` | `ai_reason` |  |
| `DecisionSource` | `string` | `decision_source` | `decision_source` | default:rule |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `CreateDisputeInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderID` | `*int64` | `order_id` | `—` |  |
| `TransactionID` | `string` | `transaction_id` | `—` |  |
| `Platform` | `string` | `platform` | `—` |  |
| `ClaimType` | `string` | `claim_type` | `—` |  |
| `Amount` | `float64` | `amount` | `—` |  |
| `Evidence` | `string` | `evidence` | `—` |  |

### `DisputeListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Platform` | `string` | `` | `—` |  |
| `ClaimType` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |

### `EvaluateDisputeResult`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Dispute` | `*DisputeCase` | `dispute` | `—` |  |
| `Score` | `float64` | `score` | `—` |  |
| `Decision` | `string` | `decision` | `—` |  |
| `RuleBreakdown` | `[]RuleBreakdownItem` | `rule_breakdown` | `—` |  |

### `RuleBreakdownItem`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Rule` | `string` | `rule` | `—` |  |
| `Score` | `float64` | `score` | `—` |  |
| `Reason` | `string` | `reason` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
