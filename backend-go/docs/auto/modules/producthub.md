# Module: `producthub`

Package: `backend-go/internal/domain/producthub/`

**Base mount prefix:** `/api/v1`
**Required permission:** `product.read`

## API Routes

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

## Models

### `ProductVersion`
**DB table:** `product_version`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ProductID` | `int64` | `product_id` | `product_id` | NOT NULL |
| `VersionData` | `json.RawMessage` | `version_data` | `version_data` |  |
| `Snapshot` | `json.RawMessage` | `snapshot` | `snapshot` |  |
| `AgentID` | `string` | `agent_id` | `agent_id` |  |
| `Reason` | `string` | `reason` | `reason` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `ProductRelation`
**DB table:** `product_relation`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SourceID` | `int64` | `source_id` | `source_id` | NOT NULL |
| `TargetID` | `int64` | `target_id` | `target_id` | NOT NULL |
| `RelationType` | `string` | `relation_type` | `relation_type` | NOT NULL |
| `Weight` | `float64` | `weight` | `weight` | default:0 |
| `AutoDiscovered` | `bool` | `auto_discovered` | `auto_discovered` | default:true |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `VersionListResponse`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Items` | `[]ProductVersion` | `items` | `—` |  |
| `Total` | `int64` | `total` | `—` |  |
| `Page` | `int` | `page` | `—` |  |
| `Size` | `int` | `size` | `—` |  |

### `DecisionRecordInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ProductID` | `int64` | `product_id` | `—` |  |
| `AgentID` | `string` | `agent_id` | `—` |  |
| `Action` | `string` | `action` | `—` |  |
| `Reasoning` | `string` | `reasoning` | `—` |  |
| `Confidence` | `float64` | `confidence` | `—` |  |

### `DecisionRecord`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `AgentID` | `string` | `agent_id` | `—` |  |
| `Action` | `string` | `action` | `—` |  |
| `Reasoning` | `string` | `reasoning` | `—` |  |
| `Confidence` | `float64` | `confidence` | `—` |  |
| `CreatedAt` | `string` | `created_at` | `—` |  |

### `RelationRequest`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SourceID` | `int64` | `source_id` | `—` |  |
| `TargetID` | `int64` | `target_id` | `—` |  |
| `RelationType` | `string` | `relation_type` | `—` |  |
| `Weight` | `float64` | `weight` | `—` |  |

### `RelatedProductResponse`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` |  |
| `Name` | `string` | `name` | `—` |  |
| `MainImage` | `string` | `main_image` | `—` |  |
| `RelationType` | `string` | `relation_type` | `—` |  |
| `Weight` | `float64` | `weight` | `—` |  |
| `AutoDiscovered` | `bool` | `auto_discovered` | `—` |  |

### `RelationListResponse`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SourceID` | `int64` | `source_id` | `—` |  |
| `Groups` | `[]RelationGroup` | `groups` | `—` |  |

### `RelationGroup`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `RelationType` | `string` | `relation_type` | `—` |  |
| `Label` | `string` | `label` | `—` |  |
| `Items` | `[]RelatedProductResponse` | `items` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
