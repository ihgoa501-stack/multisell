# Module: `businessdecision`

Package: `backend-go/internal/domain/businessdecision/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/business-decisions` | `h.List` |
| `POST` | `/api/v1/business-decisions` | `h.Create` |
| `GET` | `/api/v1/business-decisions/:id` | `h.Get` |
| `POST` | `/api/v1/business-decisions/:id/ai-recommendations` | `h.Recommend` |
| `POST` | `/api/v1/business-decisions/:id/owner-decisions` | `h.Decide` |
| `GET` | `/api/v1/business-decisions/fact-options` | `h.FactOptions` |

## Models

### `Case`
**DB table:** `business_decision_case`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `Question` | `string` | `question` | `—` | NOT NULL |
| `Target` | `string` | `target` | `—` | NOT NULL |
| `ObjectType` | `string` | `object_type` | `—` | NOT NULL |
| `ObjectID` | `int64` | `object_id` | `—` | NOT NULL |
| `TruthStatus` | `string` | `truth_status` | `—` | NOT NULL |
| `UnknownsJSON` | `string` | `-` | `—` | NOT NULL |
| `ManifestSHA256` | `string` | `manifest_sha256` | `—` | NOT NULL |
| `IdempotencyKey` | `string` | `-` | `—` | NOT NULL |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |
| `Unknowns` | `[]string` | `unknowns` | `—` |  |

### `FactSnapshot`
**DB table:** `business_decision_fact_snapshot`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `DecisionCaseID` | `int64` | `decision_case_id` | `—` | NOT NULL |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `ObjectType` | `string` | `object_type` | `—` | NOT NULL |
| `ObjectID` | `int64` | `object_id` | `—` | NOT NULL |
| `TruthStatus` | `string` | `truth_status` | `—` | NOT NULL |
| `SourceTable` | `string` | `source_table` | `—` | NOT NULL |
| `SourceObservedAt` | `time.Time` | `source_observed_at` | `—` | NOT NULL |
| `PayloadJSON` | `string` | `payload_json` | `—` | NOT NULL |
| `PayloadSHA256` | `string` | `payload_sha256` | `—` | NOT NULL |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

### `AIRecommendation`
**DB table:** `business_ai_recommendation`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `DecisionCaseID` | `int64` | `decision_case_id` | `—` | NOT NULL |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `Recommendation` | `string` | `recommendation` | `—` | NOT NULL |
| `Rationale` | `string` | `rationale` | `—` | NOT NULL |
| `TruthStatus` | `string` | `truth_status` | `—` | NOT NULL |
| `UnknownsJSON` | `string` | `-` | `—` | NOT NULL |
| `ManifestSHA256` | `string` | `manifest_sha256` | `—` | NOT NULL |
| `IdempotencyKey` | `string` | `-` | `—` | NOT NULL |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |
| `Unknowns` | `[]string` | `unknowns` | `—` |  |

### `OwnerDecision`
**DB table:** `business_owner_decision`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `DecisionCaseID` | `int64` | `decision_case_id` | `—` | NOT NULL |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `RecommendationID` | `*int64` | `recommendation_id,omitempty` | `—` |  |
| `Decision` | `string` | `decision` | `—` | NOT NULL |
| `CapabilityID` | `string` | `capability_id,omitempty` | `—` | NOT NULL |
| `CommandType` | `string` | `command_type,omitempty` | `—` | NOT NULL |
| `TargetType` | `string` | `target_type,omitempty` | `—` | NOT NULL |
| `TargetID` | `string` | `target_id,omitempty` | `—` | NOT NULL |
| `InputSHA256` | `string` | `input_sha256,omitempty` | `—` | NOT NULL |
| `InputPayload` | `json.RawMessage` | `input_payload,omitempty` | `—` | NOT NULL |
| `Reason` | `string` | `reason` | `—` | NOT NULL |
| `ManifestSHA256` | `string` | `manifest_sha256` | `—` | NOT NULL |
| `IdempotencyKey` | `string` | `-` | `—` | NOT NULL |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

### `CreateCaseInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Question` | `string` | `question` | `—` |  |
| `Target` | `string` | `target` | `—` |  |
| `ObjectType` | `string` | `object_type` | `—` |  |
| `ObjectID` | `int64` | `object_id` | `—` |  |
| `Unknowns` | `[]string` | `unknowns` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` |  |

### `RecommendInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Recommendation` | `string` | `recommendation` | `—` |  |
| `Rationale` | `string` | `rationale` | `—` |  |
| `TruthStatus` | `string` | `truth_status` | `—` |  |
| `Unknowns` | `[]string` | `unknowns` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` |  |

### `DecideInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `RecommendationID` | `*int64` | `recommendation_id` | `—` |  |
| `Decision` | `string` | `decision` | `—` |  |
| `CapabilityID` | `string` | `capability_id` | `—` |  |
| `CommandType` | `string` | `command_type` | `—` |  |
| `TargetType` | `string` | `target_type` | `—` |  |
| `TargetID` | `string` | `target_id` | `—` |  |
| `InputSHA256` | `string` | `input_sha256` | `—` |  |
| `InputPayload` | `json.RawMessage` | `input_payload` | `—` |  |
| `Reason` | `string` | `reason` | `—` |  |
| `ManifestSHA256` | `string` | `manifest_sha256` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` |  |

### `Detail`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Case` | `Case` | `case` | `—` |  |
| `Snapshot` | `FactSnapshot` | `fact_snapshot` | `—` |  |
| `Recommendations` | `[]AIRecommendation` | `ai_recommendations` | `—` |  |
| `Decisions` | `[]OwnerDecision` | `owner_decisions` | `—` |  |

### `ListItem`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `LatestDecision` | `*OwnerDecision` | `latest_owner_decision,omitempty` | `—` |  |

### `FactOption`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ObjectType` | `string` | `object_type` | `—` |  |
| `ObjectID` | `int64` | `object_id` | `—` |  |
| `Label` | `string` | `label` | `—` |  |
| `TruthStatus` | `string` | `truth_status` | `—` |  |
| `ObservedAt` | `time.Time` | `observed_at` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
