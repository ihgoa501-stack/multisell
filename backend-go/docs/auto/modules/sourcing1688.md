# Module: `sourcing1688`

Package: `backend-go/internal/domain/sourcing1688/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/sourcing-1688` | `h.List` |
| `GET` | `/api/v1/sourcing-1688/:id` | `h.Get` |
| `GET` | `/api/v1/sourcing-1688/:id/acceptance-report` | `h.AcceptanceReport` |
| `POST` | `/api/v1/sourcing-1688/:id/approvals/:approvalId/decision` | `h.DecideDraftApproval` |
| `POST` | `/api/v1/sourcing-1688/:id/capture-failed` | `h.CaptureFailed` |
| `GET` | `/api/v1/sourcing-1688/:id/collection-quality` | `h.CollectionQuality` |
| `POST` | `/api/v1/sourcing-1688/:id/convert-to-draft` | `h.ConvertToDraft` |
| `GET` | `/api/v1/sourcing-1688/:id/cost-versions` | `h.ListSourcingCostVersions` |
| `POST` | `/api/v1/sourcing-1688/:id/cost-versions` | `h.CreateSourcingCostVersion` |
| `GET` | `/api/v1/sourcing-1688/:id/draft` | `h.Draft` |
| `PUT` | `/api/v1/sourcing-1688/:id/draft` | `h.UpdateDraft` |
| `GET` | `/api/v1/sourcing-1688/:id/identity-history` | `h.IdentityHistory` |
| `GET` | `/api/v1/sourcing-1688/:id/lifecycle` | `h.Lifecycle` |
| `POST` | `/api/v1/sourcing-1688/:id/private-archive` | `h.ArchivePrivateCollection` |
| `POST` | `/api/v1/sourcing-1688/:id/private-restore` | `h.RestorePrivateCollection` |
| `PATCH` | `/api/v1/sourcing-1688/:id/private-workcopy` | `h.UpdatePrivateWorkcopy` |
| `GET` | `/api/v1/sourcing-1688/:id/publish-requests` | `h.ListPublishRequests` |
| `POST` | `/api/v1/sourcing-1688/:id/publish-requests` | `h.RequestPublish` |
| `POST` | `/api/v1/sourcing-1688/:id/publish-requests/:attemptId/decision` | `h.DecidePublish` |
| `POST` | `/api/v1/sourcing-1688/:id/publish-requests/:attemptId/execute` | `h.ExecutePublish` |
| `POST` | `/api/v1/sourcing-1688/:id/publish-requests/:attemptId/reconcile` | `h.ReconcilePublish` |
| `POST` | `/api/v1/sourcing-1688/:id/review` | `h.Review` |
| `POST` | `/api/v1/sourcing-1688/:id/review-decision` | `h.ReviewDecision` |
| `GET` | `/api/v1/sourcing-1688/:id/samples` | `h.ListSourcingSamples` |
| `POST` | `/api/v1/sourcing-1688/:id/samples` | `h.CreateSourcingSample` |
| `POST` | `/api/v1/sourcing-1688/:id/samples/:sampleId/transitions` | `h.TransitionSourcingSample` |
| `GET` | `/api/v1/sourcing-1688/:id/snapshot` | `h.Snapshot` |
| `POST` | `/api/v1/sourcing-1688/:id/submit-draft-approval` | `h.SubmitDraftApproval` |
| `GET` | `/api/v1/sourcing-1688/:id/task-links` | `h.ListPrivateTaskLinks` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links` | `h.LinkPrivateTask` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/approvals/:approvalId/decision` | `h.DecideTaskDraftApproval` |
| `GET` | `/api/v1/sourcing-1688/:id/task-links/:linkId/compliance-evidence` | `h.ListComplianceEvidence` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/compliance-evidence` | `h.CreateComplianceEvidence` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/compliance-evidence/:evidenceId/review` | `h.ReviewComplianceEvidence` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/compliance-evidence/:evidenceId/revoke` | `h.RevokeComplianceEvidence` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/convert-to-draft` | `h.ConvertTaskToDraft` |
| `GET` | `/api/v1/sourcing-1688/:id/task-links/:linkId/draft` | `h.TaskDraft` |
| `PUT` | `/api/v1/sourcing-1688/:id/task-links/:linkId/draft` | `h.UpdateTaskDraft` |
| `GET` | `/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets` | `h.ListMaterialAssets` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets` | `h.CreateMaterialAsset` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/archive` | `h.ArchiveMaterialAsset` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/mark-used` | `h.MarkMaterialUsed` |
| `PATCH` | `/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/order` | `h.ReorderMaterialAsset` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/renditions` | `h.AttachMaterialRendition` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/rights-evidence` | `h.AddMaterialRights` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/rights-evidence/:evidenceId/review` | `h.ReviewMaterialRights` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/material-assets/:assetId/rights-evidence/:evidenceId/revoke` | `h.RevokeMaterialRights` |
| `GET` | `/api/v1/sourcing-1688/:id/task-links/:linkId/publish-requests` | `h.ListTaskPublishRequests` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/publish-requests` | `h.RequestTaskPublish` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/publish-requests/:attemptId/decision` | `h.DecideTaskPublish` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/publish-requests/:attemptId/execute` | `h.ExecuteTaskPublish` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/publish-requests/:attemptId/reconcile` | `h.ReconcileTaskPublish` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/publish-requests/:attemptId/terminal-observations` | `h.ObserveTaskPublishTerminal` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/sample-waiver` | `h.WaiveSourcingSample` |
| `GET` | `/api/v1/sourcing-1688/:id/task-links/:linkId/sku-mappings` | `h.ListCanonicalSKUMappings` |
| `GET` | `/api/v1/sourcing-1688/:id/task-links/:linkId/sku-workspace` | `h.SKUWorkspace` |
| `POST` | `/api/v1/sourcing-1688/:id/task-links/:linkId/submit-draft-approval` | `h.SubmitTaskDraftApproval` |
| `GET` | `/api/v1/sourcing-1688/:id/watch` | `h.GetWatch` |
| `PUT` | `/api/v1/sourcing-1688/:id/watch` | `h.SetWatch` |
| `GET` | `/api/v1/sourcing-1688/:id/watch/alerts` | `h.ListWatchAlerts` |
| `GET` | `/api/v1/sourcing-1688/:id/watch/refresh-runs` | `h.ListWatchRuns` |
| `POST` | `/api/v1/sourcing-1688/:id/watch/refresh-runs` | `h.CreateWatchRun` |
| `GET` | `/api/v1/sourcing-1688/:id/watch/refresh-runs/:runId` | `h.GetWatchRun` |
| `POST` | `/api/v1/sourcing-1688/:id/watch/refresh-runs/:runId/evaluate` | `h.EvaluateWatchRun` |
| `POST` | `/api/v1/sourcing-1688/capture` | `h.Capture` |
| `GET` | `/api/v1/sourcing-1688/capture-failures` | `h.ListCaptureFailures` |
| `POST` | `/api/v1/sourcing-1688/capture-failures` | `h.RecordCaptureFailure` |
| `POST` | `/api/v1/sourcing-1688/duplicates/:id/resolve` | `h.ResolveDuplicate` |
| `GET` | `/api/v1/sourcing-1688/eligible-tasks` | `h.ListEligibleTasks` |
| `POST` | `/api/v1/sourcing-1688/fetch` | `fetchHandler.Fetch` |
| `POST` | `/api/v1/sourcing-1688/private-collections` | `h.CollectPrivate` |
| `POST` | `/api/v1/sourcing-1688/private-collections` | `h.CollectPrivate` |
| `GET` | `/api/v1/sourcing-1688/private-collections/failures` | `h.ListPrivateCaptureFailures` |
| `POST` | `/api/v1/sourcing-1688/private-collections/failures` | `h.RecordPrivateCaptureFailure` |
| `GET` | `/api/v1/sourcing-1688/private-collections/requests/:requestId` | `h.GetPrivateCollectionRequest` |
| `GET` | `/api/v1/sourcing-1688/private-collections/requests/:requestId` | `h.GetPrivateCollectionRequest` |
| `POST` | `/api/v1/sourcing-1688/processed-images` | `h.ProcessImage` |
| `GET` | `/api/v1/sourcing-1688/processed-images/:id/content` | `h.ProcessedImageContent` |
| `GET` | `/api/v1/sourcing-1688/summary` | `h.Summary` |

## Models

### `Sourcing1688Product`
**DB table:** `sourcing_1688_product`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OwnerID` | `int64` | `owner_id` | `owner_id` |  |
| `SourceURL` | `string` | `source_url` | `source_url` | NOT NULL |
| `SourceOfferID` | `string` | `source_offer_id,omitempty` | `source_offer_id` |  |
| `SourceProductFingerprint` | `string` | `source_product_fingerprint,omitempty` | `source_product_fingerprint` |  |
| `SupplierBusinessID` | `string` | `supplier_business_id,omitempty` | `supplier_business_id` |  |
| `Title` | `*string` | `title,omitempty` | `title` |  |
| `Price` | `*float64` | `price,omitempty` | `price` |  |
| `MOQ` | `int` | `moq` | `moq` | default:1 |
| `SupplierName` | `string` | `supplier_name` | `supplier_name` |  |
| `ShopURL` | `*string` | `shop_url,omitempty` | `shop_url` |  |
| `ShopLocation` | `*string` | `shop_location,omitempty` | `shop_location` |  |
| `Images` | `*json.RawMessage` | `images,omitempty` | `images` |  |
| `Attributes` | `*json.RawMessage` | `attributes,omitempty` | `attributes` |  |
| `SkuVariants` | `*json.RawMessage` | `sku_variants,omitempty` | `sku_variants` |  |
| `Description` | `*string` | `description,omitempty` | `description` |  |
| `PackageLengthCm` | `*float64` | `package_length_cm,omitempty` | `package_length_cm` |  |
| `PackageWidthCm` | `*float64` | `package_width_cm,omitempty` | `package_width_cm` |  |
| `PackageHeightCm` | `*float64` | `package_height_cm,omitempty` | `package_height_cm` |  |
| `PackageWeightKg` | `*float64` | `package_weight_kg,omitempty` | `package_weight_kg` |  |
| `RawData` | `*json.RawMessage` | `raw_data,omitempty` | `raw_data` |  |
| `Status` | `string` | `status` | `status` | default:collected |
| `ProductID` | `*int64` | `product_id,omitempty` | `product_id` |  |
| `SupplierID` | `*int64` | `supplier_id,omitempty` | `supplier_id` |  |
| `CollectedBy` | `*string` | `collected_by,omitempty` | `collected_by` |  |
| `ImportedBy` | `*string` | `imported_by,omitempty` | `imported_by` |  |
| `ImportedAt` | `*time.Time` | `imported_at,omitempty` | `imported_at` |  |
| `DemandCaseID` | `*int64` | `demand_case_id,omitempty` | `demand_case_id` |  |
| `ExperimentID` | `*string` | `experiment_id,omitempty` | `experiment_id` |  |
| `SnapshotID` | `*int64` | `snapshot_id,omitempty` | `snapshot_id` |  |
| `ReviewedBy` | `*int64` | `reviewed_by,omitempty` | `reviewed_by` |  |
| `ReviewedAt` | `*time.Time` | `reviewed_at,omitempty` | `reviewed_at` |  |
| `ReviewNotes` | `string` | `review_notes,omitempty` | `review_notes` |  |
| `LifecycleStatus` | `string` | `lifecycle_status` | `lifecycle_status` | default:pending_review |
| `LifecycleActorID` | `*int64` | `lifecycle_actor_id,omitempty` | `lifecycle_actor_id` |  |
| `LifecycleReason` | `string` | `lifecycle_reason,omitempty` | `lifecycle_reason` |  |
| `LifecycleUpdatedAt` | `*time.Time` | `lifecycle_updated_at,omitempty` | `lifecycle_updated_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `PrivateCollectionListItem`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `FieldStatuses` | `map[string]string` | `field_statuses` | `—` |  |
| `ObservationCount` | `int64` | `observation_count` | `—` |  |
| `TaskLinkCount` | `int64` | `task_link_count` | `—` |  |

### `Sourcing1688Snapshot`
**DB table:** `sourcing_1688_snapshot`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `SourcingProductID` | `int64` | `sourcing_product_id` | `sourcing_product_id` | NOT NULL |
| `SourceURL` | `string` | `source_url` | `source_url` | NOT NULL |
| `CollectedAt` | `time.Time` | `collected_at` | `collected_at` | NOT NULL |
| `CollectedBy` | `int64` | `collected_by` | `collected_by` | NOT NULL |
| `Driver` | `string` | `driver` | `driver` | NOT NULL |
| `ParserVersion` | `string` | `parser_version` | `parser_version` | NOT NULL |
| `ExtensionVersion` | `string` | `extension_version` | `extension_version` | NOT NULL, default:'' |
| `SchemaVersion` | `string` | `schema_version` | `schema_version` | NOT NULL, default:'' |
| `CaptureMode` | `string` | `capture_mode` | `capture_mode` | NOT NULL, default:legacy_unknown |
| `CollectionRequestID` | `string` | `collection_request_id` | `collection_request_id` | NOT NULL, default:'' |
| `RawPayload` | `json.RawMessage` | `raw_payload` | `raw_payload` | NOT NULL |
| `RawSHA256` | `string` | `raw_sha256` | `raw_sha256` | NOT NULL |
| `StructuredDataSHA256` | `string` | `structured_data_sha256` | `structured_data_sha256` | NOT NULL, default:'' |
| `RequestEnvelopeSHA256` | `string` | `request_envelope_sha256` | `request_envelope_sha256` | NOT NULL, default:'' |
| `ObservedTitle` | `*string` | `observed_title,omitempty` | `observed_title` |  |
| `ObservedPrice` | `*float64` | `observed_price,omitempty` | `observed_price` |  |
| `ObservedMOQ` | `int` | `observed_moq` | `observed_moq` |  |
| `ObservedSupplier` | `string` | `observed_supplier,omitempty` | `observed_supplier` |  |
| `ObservedSupplierBusinessID` | `string` | `observed_supplier_business_id,omitempty` | `observed_supplier_business_id` |  |
| `ProductFingerprint` | `string` | `product_fingerprint,omitempty` | `product_fingerprint` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `CaptureInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `DemandCaseID` | `int64` | `demand_case_id` | `—` |  |
| `ExperimentID` | `string` | `experiment_id` | `—` |  |
| `SourceURL` | `string` | `source_url` | `—` |  |
| `CollectedAt` | `time.Time` | `collected_at` | `—` |  |
| `CollectedBy` | `int64` | `collected_by` | `—` |  |
| `Driver` | `string` | `driver` | `—` |  |
| `ParserVersion` | `string` | `parser_version` | `—` |  |
| `RawPayload` | `json.RawMessage` | `raw_payload` | `—` |  |
| `Title` | `*string` | `title` | `—` |  |
| `Price` | `*float64` | `price` | `—` |  |
| `MOQ` | `*int` | `moq` | `—` |  |
| `SupplierName` | `string` | `supplier_name` | `—` |  |
| `SupplierBusinessID` | `string` | `supplier_business_id` | `—` |  |
| `Images` | `json.RawMessage` | `images` | `—` |  |
| `SkuVariants` | `json.RawMessage` | `sku_variants` | `—` |  |
| `CaptureMode` | `string` | `-` | `—` |  |
| `CollectionRequestID` | `string` | `-` | `—` |  |

### `ReviewInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ReviewedBy` | `int64` | `reviewed_by` | `—` |  |
| `Notes` | `string` | `notes` | `—` |  |

### `CreateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SourceURL` | `string` | `source_url` | `—` |  |
| `Title` | `*string` | `title` | `—` |  |
| `Price` | `*float64` | `price` | `—` |  |
| `MOQ` | `*int` | `moq` | `—` |  |
| `SupplierName` | `string` | `supplier_name` | `—` |  |
| `ShopURL` | `*string` | `shop_url` | `—` |  |
| `ShopLocation` | `*string` | `shop_location` | `—` |  |
| `Description` | `*string` | `description` | `—` |  |
| `ProductID` | `*int64` | `product_id` | `—` |  |
| `SupplierID` | `*int64` | `supplier_id` | `—` |  |
| `CollectedBy` | `*string` | `collected_by` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `RawData` | `*json.RawMessage` | `raw_data` | `—` |  |

### `UpdateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SourceURL` | `*string` | `source_url` | `—` |  |
| `Title` | `*string` | `title` | `—` |  |
| `Price` | `*float64` | `price` | `—` |  |
| `MOQ` | `*int` | `moq` | `—` |  |
| `SupplierName` | `*string` | `supplier_name` | `—` |  |
| `ShopURL` | `*string` | `shop_url` | `—` |  |
| `ShopLocation` | `*string` | `shop_location` | `—` |  |
| `Description` | `*string` | `description` | `—` |  |
| `ProductID` | `*int64` | `product_id` | `—` |  |
| `SupplierID` | `*int64` | `supplier_id` | `—` |  |
| `CollectedBy` | `*string` | `collected_by` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |
| `RawData` | `*json.RawMessage` | `raw_data` | `—` |  |

### `ListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Search` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |
| `LifecycleStatus` | `string` | `` | `—` |  |
| `ProductID` | `*int64` | `` | `—` |  |

### `ImportInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ImportedBy` | `string` | `imported_by` | `—` |  |

### `RejectInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `RejectedBy` | `string` | `rejected_by` | `—` |  |
| `Reason` | `string` | `reason` | `—` |  |

### `Summary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Total` | `int64` | `total` | `—` |  |
| `ByStatus` | `map[string]int64` | `by_status` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
