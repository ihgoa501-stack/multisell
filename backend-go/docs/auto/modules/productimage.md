# Module: `productimage`

Package: `backend-go/internal/domain/productimage/`

**Base mount prefix:** `/api/v1`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `POST` | `/api/v1/product-images/assets` | `h.UploadAsset` |
| `GET` | `/api/v1/product-images/budget-policies` | `budget.ListPolicies` |
| `POST` | `/api/v1/product-images/budget-policies` | `budget.CreatePolicy` |
| `GET` | `/api/v1/product-images/budget-reservations` | `budget.ListReservations` |
| `POST` | `/api/v1/product-images/budget-reservations/:reservation_id/cancel` | `budget.Cancel` |
| `POST` | `/api/v1/product-images/budget-reservations/:reservation_id/charges` | `budget.Reconcile` |
| `POST` | `/api/v1/product-images/budget-reservations/:reservation_id/no-charge-reconciliations` | `budget.ReconcileNoCharge` |
| `GET` | `/api/v1/product-images/capabilities` | `h.Capabilities` |
| `POST` | `/api/v1/product-images/image-sets` | `imageSetHandler.Create` |
| `GET` | `/api/v1/product-images/image-sets/:set_id` | `imageSetHandler.Get` |
| `POST` | `/api/v1/product-images/image-sets/:set_id/decisions` | `release.DecideSet` |
| `POST` | `/api/v1/product-images/image-sets/:set_id/freeze` | `imageSetHandler.Freeze` |
| `GET` | `/api/v1/product-images/manual-imports` | `h.ListManualImports` |
| `POST` | `/api/v1/product-images/manual-imports` | `h.CreateManualImport` |
| `POST` | `/api/v1/product-images/mcp` | `mcpHandler.ServeHTTP` |
| `GET` | `/api/v1/product-images/publish-attempts/:attempt_id` | `publish.Get` |
| `POST` | `/api/v1/product-images/publish-attempts/:attempt_id/reconcile` | `publish.Reconcile` |
| `GET` | `/api/v1/product-images/recipes/:recipe_key/summary` | `governance.RecipeSummary` |
| `POST` | `/api/v1/product-images/release-attestations` | `release.Issue` |
| `GET` | `/api/v1/product-images/release-attestations/:attestation_id` | `release.Get` |
| `POST` | `/api/v1/product-images/release-attestations/:attestation_id/publish-attempts` | `publish.Execute` |
| `GET` | `/api/v1/product-images/rights-grants` | `governance.ListRights` |
| `POST` | `/api/v1/product-images/rights-grants` | `governance.CreateRights` |
| `POST` | `/api/v1/product-images/rights-grants/:grant_id/revocations` | `governance.RevokeRights` |
| `POST` | `/api/v1/product-images/rule-snapshots` | `release.CreateRule` |
| `GET` | `/api/v1/product-images/tasks` | `h.ListTasks` |
| `POST` | `/api/v1/product-images/tasks` | `h.CreateTask` |
| `GET` | `/api/v1/product-images/tasks/:id` | `h.GetTask` |
| `GET` | `/api/v1/product-images/tasks/:id/attempts` | `h.Attempts` |
| `GET` | `/api/v1/product-images/tasks/:id/costs` | `governance.ListCosts` |
| `POST` | `/api/v1/product-images/tasks/:id/costs` | `governance.CreateCost` |
| `POST` | `/api/v1/product-images/tasks/:id/execution-approvals` | `h.ApproveExecution` |
| `POST` | `/api/v1/product-images/tasks/:id/executions` | `h.Execute` |
| `POST` | `/api/v1/product-images/tasks/:id/feedback` | `governance.CreateFeedback` |
| `GET` | `/api/v1/product-images/tasks/:id/output/content` | `h.OutputContent` |
| `GET` | `/api/v1/product-images/tasks/:id/reviews` | `governance.ListReviews` |
| `POST` | `/api/v1/product-images/tasks/:id/reviews` | `governance.CreateReview` |

## Models

### `Asset`
**DB table:** `product_image_assets`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `BlobID` | `string` | `blob_id` | `—` | NOT NULL |
| `Filename` | `string` | `filename` | `—` | NOT NULL |
| `ContentType` | `string` | `content_type` | `—` | NOT NULL |
| `SizeBytes` | `int64` | `size_bytes` | `—` | NOT NULL |
| `SHA256` | `string` | `sha256` | `—` | NOT NULL |
| `Truth` | `string` | `truth` | `—` | NOT NULL, default:unknown |
| `SourceKind` | `string` | `source_kind` | `—` | NOT NULL, default:upload |
| `ParentAssetID` | `*int64` | `parent_asset_id,omitempty` | `—` |  |
| `ParentAssetSHA` | `string` | `parent_asset_sha256,omitempty` | `—` |  |
| `ChannelRestriction` | `string` | `channel_restriction` | `—` | NOT NULL, default:* |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

### `Task`
**DB table:** `product_image_tasks`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `AssetID` | `int64` | `asset_id` | `—` | NOT NULL |
| `SKUID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `RecipeKey` | `string` | `recipe_key` | `—` | NOT NULL |
| `RecipeVersion` | `int` | `recipe_version` | `—` | NOT NULL, default:1 |
| `RecipeManifest` | `json.RawMessage` | `recipe_manifest` | `—` |  |
| `RecipeHash` | `string` | `recipe_hash` | `—` | NOT NULL |
| `ParentTaskID` | `*int64` | `parent_task_id,omitempty` | `—` |  |
| `CandidateRound` | `int` | `candidate_round` | `—` | NOT NULL, default:1 |
| `ImageServiceJobID` | `string` | `image_service_job_id,omitempty` | `—` |  |
| `OutputBlobID` | `string` | `output_blob_id,omitempty` | `—` |  |
| `OutputURL` | `string` | `output_url,omitempty` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` | NOT NULL |
| `ManifestHash` | `string` | `manifest_hash` | `—` | NOT NULL |
| `Operation` | `string` | `operation` | `—` | NOT NULL |
| `Processor` | `string` | `processor` | `—` | NOT NULL, default:deterministic |
| `Purpose` | `string` | `purpose` | `—` | NOT NULL |
| `Channel` | `string` | `channel` | `—` | NOT NULL |
| `Region` | `string` | `region` | `—` | NOT NULL |
| `ProviderEnvironment` | `string` | `provider_environment` | `—` | NOT NULL, default:'' |
| `MaxCost` | `string` | `max_cost,omitempty` | `—` | NOT NULL, default:'' |
| `Currency` | `string` | `currency,omitempty` | `—` | NOT NULL, default:'' |
| `Sandbox` | `bool` | `sandbox` | `—` | NOT NULL, default:false |
| `Watermarked` | `bool` | `watermarked` | `—` | NOT NULL, default:false |
| `NonPublishable` | `bool` | `non_publishable` | `—` | NOT NULL, default:false |
| `Version` | `int64` | `version` | `—` | NOT NULL, default:1 |
| `Width` | `int` | `width` | `—` | NOT NULL |
| `Height` | `int` | `height` | `—` | NOT NULL |
| `Format` | `string` | `format` | `—` | NOT NULL |
| `Status` | `string` | `status` | `—` | NOT NULL |
| `ErrorCode` | `string` | `error_code,omitempty` | `—` |  |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `—` |  |

### `Review`
**DB table:** `product_image_reviews`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `TaskID` | `int64` | `task_id` | `—` | NOT NULL |
| `Decision` | `string` | `decision` | `—` | NOT NULL |
| `Outcome` | `string` | `outcome,omitempty` | `—` |  |
| `ReasonCodes` | `json.RawMessage` | `reason_codes,omitempty` | `—` |  |
| `ErrorRegions` | `json.RawMessage` | `error_regions,omitempty` | `—` |  |
| `ReworkInstruction` | `string` | `rework_instruction,omitempty` | `—` |  |
| `ReviewSeconds` | `int` | `review_seconds,omitempty` | `—` |  |
| `Truth` | `string` | `truth` | `—` | NOT NULL, default:unknown |
| `Notes` | `string` | `notes,omitempty` | `—` |  |
| `AssetSHA` | `string` | `asset_sha256,omitempty` | `—` |  |
| `Purpose` | `string` | `purpose,omitempty` | `—` |  |
| `Channel` | `string` | `channel,omitempty` | `—` |  |
| `ProductAuthenticity` | `string` | `product_authenticity,omitempty` | `—` |  |
| `RightsStatus` | `string` | `rights_status,omitempty` | `—` |  |
| `ChannelRules` | `string` | `channel_rules,omitempty` | `—` |  |
| `ClaimsScene` | `string` | `claims_scene,omitempty` | `—` |  |
| `TechnicalVisual` | `string` | `technical_visual,omitempty` | `—` |  |
| `EvidenceSHA` | `string` | `evidence_sha256,omitempty` | `—` |  |
| `EvidenceTruth` | `string` | `evidence_truth,omitempty` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key,omitempty` | `—` |  |
| `RequestHash` | `string` | `request_hash,omitempty` | `—` |  |
| `ExpectedTaskVersion` | `int64` | `expected_task_version,omitempty` | `—` |  |
| `VerifiedAt` | `*time.Time` | `verified_at,omitempty` | `—` |  |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

### `RightsGrant`
**DB table:** `product_image_rights_grants`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `AssetID` | `*int64` | `asset_id,omitempty` | `—` |  |
| `AssetSHA` | `string` | `asset_sha256` | `—` | NOT NULL |
| `CanCopy` | `bool` | `can_copy` | `—` |  |
| `CanModify` | `bool` | `can_modify` | `—` |  |
| `CanThirdPartyAI` | `bool` | `can_third_party_ai` | `—` |  |
| `CanCrossBorder` | `bool` | `can_cross_border` | `—` |  |
| `CanCommercialPublish` | `bool` | `can_commercial_publish` | `—` |  |
| `CanPlatformSublicense` | `bool` | `can_platform_sublicense` | `—` |  |
| `TrademarkCleared` | `bool` | `trademark_cleared` | `—` |  |
| `LikenessCleared` | `bool` | `likeness_cleared` | `—` |  |
| `Purpose` | `string` | `purpose` | `—` | NOT NULL |
| `Jurisdiction` | `string` | `jurisdiction` | `—` | NOT NULL |
| `Channel` | `string` | `channel` | `—` | NOT NULL |
| `Provider` | `string` | `provider` | `—` | NOT NULL |
| `Region` | `string` | `region` | `—` | NOT NULL |
| `Grantor` | `string` | `grantor` | `—` | NOT NULL |
| `RightsChain` | `string` | `rights_chain` | `—` | NOT NULL |
| `EvidenceSHA` | `string` | `evidence_sha256` | `—` | NOT NULL |
| `OwnerVerified` | `bool` | `owner_verified` | `—` | NOT NULL |
| `ValidFrom` | `time.Time` | `valid_from` | `—` |  |
| `ValidUntil` | `*time.Time` | `valid_until,omitempty` | `—` |  |
| `RevokedAt` | `*time.Time` | `revoked_at,omitempty` | `—` |  |
| `RevocationReason` | `string` | `revocation_reason,omitempty` | `—` | default:null |
| `RevocationIdempotencyKey` | `string` | `revocation_idempotency_key,omitempty` | `—` |  |
| `RevocationRequestHash` | `string` | `revocation_request_hash,omitempty` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` | NOT NULL |
| `RequestHash` | `string` | `request_hash` | `—` | NOT NULL |
| `Version` | `int64` | `version` | `—` | NOT NULL, default:1 |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `—` |  |

### `CostEntry`
**DB table:** `product_image_cost_entries`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `TaskID` | `int64` | `task_id` | `—` | NOT NULL |
| `Kind` | `string` | `kind` | `—` | NOT NULL |
| `Category` | `string` | `category` | `—` | NOT NULL |
| `Provider` | `string` | `provider` | `—` | NOT NULL |
| `Amount` | `string` | `amount` | `—` | NOT NULL |
| `Currency` | `string` | `currency` | `—` | NOT NULL |
| `ExchangeRate` | `string` | `exchange_rate` | `—` | NOT NULL |
| `ExchangeRateSource` | `string` | `exchange_rate_source` | `—` | NOT NULL |
| `ObservedAt` | `time.Time` | `observed_at` | `—` |  |
| `BillingStatus` | `string` | `billing_status` | `—` | NOT NULL |
| `EvidenceSHA` | `string` | `evidence_sha256,omitempty` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` | NOT NULL |
| `RequestHash` | `string` | `request_hash` | `—` | NOT NULL |
| `ExpectedTaskVersion` | `int64` | `expected_task_version` | `—` |  |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `—` |  |

### `CreateTaskInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `AssetID` | `int64` | `asset_id` | `—` |  |
| `SKUID` | `int64` | `sku_id` | `—` |  |
| `RecipeKey` | `string` | `recipe_key` | `—` |  |
| `RecipeVersion` | `int` | `recipe_version` | `—` |  |
| `Recipe` | `RecipeManifest` | `recipe` | `—` |  |
| `ParentTaskID` | `*int64` | `parent_task_id,omitempty` | `—` |  |
| `CandidateRound` | `int` | `candidate_round` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` |  |
| `Operation` | `string` | `operation` | `—` |  |
| `Processor` | `string` | `processor` | `—` |  |
| `Purpose` | `string` | `purpose` | `—` |  |
| `Channel` | `string` | `channel` | `—` |  |
| `Region` | `string` | `region` | `—` |  |
| `Width` | `int` | `width` | `—` |  |
| `Height` | `int` | `height` | `—` |  |
| `Format` | `string` | `format` | `—` |  |
| `MaxCost` | `string` | `max_cost,omitempty` | `—` |  |
| `Currency` | `string` | `currency,omitempty` | `—` |  |

### `RecipeManifest`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ReferenceAssetIDs` | `[]int64` | `reference_asset_ids,omitempty` | `—` |  |
| `MaskAssetID` | `*int64` | `mask_asset_id,omitempty` | `—` |  |
| `SceneStructure` | `string` | `scene_structure` | `—` |  |
| `Prompt` | `string` | `prompt,omitempty` | `—` |  |
| `NegativePrompt` | `string` | `negative_prompt,omitempty` | `—` |  |
| `Model` | `string` | `model` | `—` |  |
| `ModelVersion` | `string` | `model_version` | `—` |  |
| `Parameters` | `json.RawMessage` | `parameters` | `—` |  |
| `MustNotChange` | `[]string` | `must_not_change` | `—` |  |

### `CandidateFeedbackInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Outcome` | `string` | `outcome` | `—` |  |
| `ReasonCodes` | `[]string` | `reason_codes,omitempty` | `—` |  |
| `ErrorRegions` | `json.RawMessage` | `error_regions,omitempty` | `—` |  |
| `ReworkInstruction` | `string` | `rework_instruction,omitempty` | `—` |  |
| `ReviewSeconds` | `int` | `review_seconds` | `—` |  |
| `Notes` | `string` | `notes,omitempty` | `—` |  |
| `AssetSHA` | `string` | `asset_sha256` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` |  |
| `ExpectedVersion` | `int64` | `expected_version` | `—` |  |

### `RecipeSummary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `RecipeKey` | `string` | `recipe_key` | `—` |  |
| `SKUID` | `int64` | `sku_id` | `—` |  |
| `Purpose` | `string` | `purpose` | `—` |  |
| `Channel` | `string` | `channel` | `—` |  |
| `LatestRecipeVersion` | `int` | `latest_recipe_version` | `—` |  |
| `Candidates` | `int64` | `candidates` | `—` |  |
| `Selected` | `int64` | `selected` | `—` |  |
| `Rejected` | `int64` | `rejected` | `—` |  |
| `ReworkRequested` | `int64` | `rework_requested` | `—` |  |
| `AcceptanceRate` | `float64` | `acceptance_rate` | `—` |  |
| `ReviewSeconds` | `int64` | `review_seconds` | `—` |  |
| `ProductionSeconds` | `int64` | `production_seconds` | `—` |  |
| `ReworkRounds` | `int` | `rework_rounds` | `—` |  |
| `ActualCost` | `string` | `actual_cost` | `—` |  |
| `Currency` | `string` | `currency,omitempty` | `—` |  |

### `ExecutionInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `IdempotencyKey` | `string` | `idempotency_key` | `—` |  |

### `ExecutionApproval`
**DB table:** `product_image_execution_approvals`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `ExecutionID` | `string` | `execution_id` | `—` | NOT NULL |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `TaskID` | `int64` | `task_id` | `—` | NOT NULL |
| `TaskVersion` | `int64` | `task_version` | `—` | NOT NULL |
| `ManifestHash` | `string` | `manifest_hash` | `—` | NOT NULL |
| `Operation` | `string` | `operation` | `—` | NOT NULL |
| `Processor` | `string` | `processor` | `—` | NOT NULL |
| `MaxCost` | `string` | `max_cost` | `—` | NOT NULL |
| `Currency` | `string` | `currency` | `—` | NOT NULL |
| `Nonce` | `string` | `-` | `—` | NOT NULL |
| `ApprovedAt` | `time.Time` | `approved_at` | `—` |  |
| `ExpiresAt` | `time.Time` | `expires_at` | `—` |  |
| `ConsumedAt` | `*time.Time` | `consumed_at,omitempty` | `—` |  |
| `BudgetReservationID` | `int64` | `budget_reservation_id,omitempty` | `—` |  |

### `ExecutionRightsSnapshot`
**DB table:** `product_image_execution_rights_snapshots`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `ApprovalID` | `int64` | `approval_id` | `—` | NOT NULL |
| `ApprovalExecutionID` | `string` | `approval_execution_id` | `—` | NOT NULL |
| `TaskID` | `int64` | `task_id` | `—` | NOT NULL |
| `TaskVersion` | `int64` | `task_version` | `—` | NOT NULL |
| `ManifestHash` | `string` | `manifest_hash` | `—` | NOT NULL |
| `Provider` | `string` | `provider` | `—` | NOT NULL |
| `GrantID` | `int64` | `grant_id` | `—` | NOT NULL |
| `GrantVersion` | `int64` | `grant_version` | `—` | NOT NULL |
| `AssetSHA` | `string` | `asset_sha256` | `—` | NOT NULL |
| `EvidenceSHA` | `string` | `evidence_sha256` | `—` | NOT NULL |
| `GrantRequestHash` | `string` | `grant_request_sha256` | `—` | NOT NULL |
| `CanCopy` | `bool` | `can_copy` | `—` | NOT NULL |
| `CanModify` | `bool` | `can_modify` | `—` | NOT NULL |
| `CanThirdPartyAI` | `bool` | `can_third_party_ai` | `—` | NOT NULL |
| `CanCrossBorder` | `bool` | `can_cross_border` | `—` | NOT NULL |
| `ValidFrom` | `time.Time` | `valid_from` | `—` |  |
| `ValidUntil` | `*time.Time` | `valid_until,omitempty` | `—` |  |
| `ClaimedAt` | `time.Time` | `claimed_at` | `—` |  |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

### `ApprovalInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Processor` | `string` | `processor` | `—` |  |
| `MaxCost` | `string` | `max_cost` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |
| `ExpectedVersion` | `int64` | `expected_version` | `—` |  |

### `BudgetPolicy`
**DB table:** `product_image_budget_policies`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `Currency` | `string` | `currency` | `—` | NOT NULL |
| `PeriodStart` | `time.Time` | `period_start` | `—` |  |
| `PeriodEnd` | `time.Time` | `period_end` | `—` |  |
| `TotalAmount` | `string` | `total_amount` | `—` | NOT NULL |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` | NOT NULL |
| `RequestHash` | `string` | `request_hash` | `—` | NOT NULL |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

### `BudgetReservation`
**DB table:** `product_image_budget_reservations`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `PolicyID` | `int64` | `policy_id` | `—` | NOT NULL |
| `ApprovalID` | `int64` | `approval_id` | `—` | NOT NULL |
| `TaskID` | `int64` | `task_id` | `—` | NOT NULL |
| `TaskVersion` | `int64` | `task_version` | `—` | NOT NULL |
| `ManifestHash` | `string` | `manifest_hash` | `—` | NOT NULL |
| `Provider` | `string` | `provider` | `—` | NOT NULL |
| `Currency` | `string` | `currency` | `—` | NOT NULL |
| `ReservedAmount` | `string` | `reserved_amount` | `—` | NOT NULL |
| `State` | `string` | `state` | `—` | NOT NULL |
| `ClaimedAt` | `*time.Time` | `claimed_at,omitempty` | `—` |  |
| `ReleasedAt` | `*time.Time` | `released_at,omitempty` | `—` |  |
| `ReleaseReason` | `string` | `release_reason,omitempty` | `—` |  |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `—` |  |

### `BudgetCharge`
**DB table:** `product_image_budget_charges`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` | PK |
| `OwnerID` | `int64` | `owner_id` | `—` | NOT NULL |
| `ReservationID` | `int64` | `reservation_id` | `—` | NOT NULL |
| `Amount` | `string` | `amount` | `—` | NOT NULL |
| `DeltaAmount` | `string` | `delta_amount` | `—` | NOT NULL |
| `Currency` | `string` | `currency` | `—` | NOT NULL |
| `Kind` | `string` | `kind` | `—` | NOT NULL |
| `OverBudget` | `bool` | `over_budget` | `—` |  |
| `EvidenceSHA` | `string` | `evidence_sha256` | `—` | NOT NULL |
| `ObservedAt` | `time.Time` | `observed_at` | `—` |  |
| `IdempotencyKey` | `string` | `idempotency_key` | `—` | NOT NULL |
| `RequestHash` | `string` | `request_hash` | `—` | NOT NULL |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
