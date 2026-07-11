# Module: `experiment`

Package: `backend-go/internal/domain/experiment/`

**Base mount prefix:** `/api/v1`

## API Routes

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

## Models

### `ExperimentCase`
**DB table:** `experiment_case`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `ExperimentID` | `string` | `experiment_id` | `—` | NOT NULL |
| `Name` | `string` | `name` | `—` | NOT NULL |
| `Stage` | `string` | `stage` | `—` | NOT NULL |
| `Status` | `string` | `status` | `—` | default:active |
| `FinalProfitStatus` | `string` | `final_profit_status` | `—` | default:pending |
| `FinalRevenue` | `float64` | `final_revenue` | `—` |  |
| `FinalTotalCost` | `float64` | `final_total_cost` | `—` |  |
| `FinalProfitAmount` | `float64` | `final_profit_amount` | `—` |  |
| `ProfitCurrency` | `string` | `profit_currency` | `—` |  |
| `CashRecoveryStatus` | `string` | `cash_recovery_status` | `—` | default:pending |
| `CashRecoveredAmount` | `float64` | `cash_recovered_amount` | `—` |  |
| `CashCurrency` | `string` | `cash_currency` | `—` |  |
| `CashRecoveredAt` | `*time.Time` | `cash_recovered_at` | `—` |  |
| `FinalDecision` | `string` | `final_decision` | `—` |  |
| `OwnerID` | `int64` | `owner_id` | `—` |  |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `—` |  |

### `GateDecision`
**DB table:** `experiment_gate_decision`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `ExperimentID` | `string` | `experiment_id` | `—` | NOT NULL |
| `Stage` | `string` | `stage` | `—` | NOT NULL |
| `GateCode` | `string` | `gate_code` | `—` | NOT NULL |
| `Result` | `string` | `result` | `—` | NOT NULL |
| `Reason` | `string` | `reason` | `—` |  |
| `EvidenceIDs` | `string` | `-` | `—` |  |
| `DecidedBy` | `int64` | `decided_by` | `—` |  |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

### `EvidenceRecord`
**DB table:** `experiment_evidence`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `ExperimentID` | `string` | `experiment_id` | `—` | NOT NULL |
| `Stage` | `string` | `stage` | `—` | NOT NULL |
| `EvidenceKind` | `string` | `evidence_kind` | `—` | NOT NULL, default:support |
| `TruthStatus` | `string` | `truth_status` | `—` | NOT NULL |
| `Title` | `string` | `title` | `—` | NOT NULL |
| `SourceURI` | `string` | `source_uri` | `—` |  |
| `ObservedAt` | `*time.Time` | `observed_at` | `—` |  |
| `ExpiresAt` | `*time.Time` | `expires_at` | `—` |  |
| `VerifiedBy` | `int64` | `verified_by` | `—` |  |
| `VerifiedAt` | `*time.Time` | `verified_at` | `—` |  |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

### `ObjectLink`
**DB table:** `experiment_object_link`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `ExperimentID` | `string` | `experiment_id` | `—` | NOT NULL |
| `ObjectType` | `string` | `object_type` | `—` | NOT NULL |
| `ObjectID` | `string` | `object_id` | `—` | NOT NULL |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

### `GateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Stage` | `string` | `stage` | `—` |  |
| `GateCode` | `string` | `gate_code` | `—` |  |
| `Result` | `string` | `result` | `—` |  |
| `Reason` | `string` | `reason` | `—` |  |
| `EvidenceIDs` | `[]int64` | `evidence_ids` | `—` |  |
| `DecidedBy` | `int64` | `decided_by` | `—` |  |

### `Detail`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Case` | `ExperimentCase` | `case` | `—` |  |
| `Gates` | `[]GateDecision` | `gates` | `—` |  |
| `Evidence` | `[]EvidenceRecord` | `evidence` | `—` |  |
| `ObjectLinks` | `[]ObjectLink` | `object_links` | `—` |  |

### `OwnerSummary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ExperimentID` | `string` | `experiment_id` | `—` |  |
| `Stage` | `string` | `stage` | `—` |  |
| `PassedGates` | `int64` | `passed_gates` | `—` |  |
| `Blockers` | `[]string` | `blockers` | `—` |  |
| `FinalProfitStatus` | `string` | `final_profit_status` | `—` |  |
| `FinalRevenue` | `float64` | `final_revenue` | `—` |  |
| `FinalTotalCost` | `float64` | `final_total_cost` | `—` |  |
| `FinalProfitAmount` | `float64` | `final_profit_amount` | `—` |  |
| `ProfitCurrency` | `string` | `profit_currency` | `—` |  |
| `CashRecoveryStatus` | `string` | `cash_recovery_status` | `—` |  |
| `CashRecoveredAmount` | `float64` | `cash_recovered_amount` | `—` |  |
| `CashCurrency` | `string` | `cash_currency` | `—` |  |
| `CashRecoveredAt` | `*time.Time` | `cash_recovered_at` | `—` |  |
| `FinalDecision` | `string` | `final_decision` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
