# Module: `integrations`

Package: `backend-go/internal/domain/integrations/`

**Base mount prefix:** `/api/v1`

## API Routes

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
| `POST` | `/api/v1/platform-integrations/:id/order-events` | `h.IngestOrderEvent` |
| `GET` | `/api/v1/platform-integrations/:id/ozon-products` | `h.ListOzonProducts` |
| `POST` | `/api/v1/platform-integrations/:id/sync` | `h.Sync` |
| `POST` | `/api/v1/platform-integrations/:id/test` | `h.TestConnection` |
| `POST` | `/api/v1/platform-integrations/mock/seed` | `` |
| `GET` | `/api/v1/platform-integrations/owner-fact-options` | `h.OwnerFactOptions` |

## Models

### `PlatformIntegrationAccount`
**DB table:** `platform_integration_account`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `PlatformID` | `int64` | `platform_id` | `platform_id` | NOT NULL |
| `StoreName` | `string` | `store_name` | `store_name` |  |
| `AccountID` | `string` | `account_id` | `account_id` |  |
| `AccessToken` | `string` | `-` | `access_token` |  |
| `RefreshToken` | `string` | `-` | `refresh_token` |  |
| `TokenExpiresAt` | `*time.Time` | `token_expires_at,omitempty` | `token_expires_at` |  |
| `Status` | `string` | `status` | `status` | default:active |
| `LastSyncAt` | `*time.Time` | `last_sync_at,omitempty` | `last_sync_at` |  |
| `SyncStatus` | `string` | `sync_status` | `sync_status` | default:idle |
| `LastError` | `string` | `last_error` | `last_error` |  |
| `ExecutionMode` | `int8` | `execution_mode` | `execution_mode` | default:0 |
| `Config` | `json.RawMessage` | `config,omitempty` | `config` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |
| `PlatformName` | `string` | `platform_name,omitempty` | `—` |  |

### `PlatformCategoryMapping`
**DB table:** `platform_category_mapping`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `AccountID` | `int64` | `account_id` | `account_id` | NOT NULL |
| `LocalCategoryID` | `int64` | `local_category_id` | `local_category_id` | NOT NULL |
| `PlatformCategoryID` | `string` | `platform_category_id` | `platform_category_id` |  |
| `PlatformCategoryName` | `string` | `platform_category_name` | `platform_category_name` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `PlatformAttributeMapping`
**DB table:** `platform_attribute_mapping`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `AccountID` | `int64` | `account_id` | `account_id` | NOT NULL |
| `LocalAttrName` | `string` | `local_attr_name` | `local_attr_name` | NOT NULL |
| `PlatformAttrID` | `string` | `platform_attr_id` | `platform_attr_id` |  |
| `PlatformAttrName` | `string` | `platform_attr_name` | `platform_attr_name` |  |
| `Required` | `bool` | `required` | `required` | default:false |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `CreateAccountInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `int64` | `platform_id` | `—` |  |
| `StoreName` | `string` | `store_name` | `—` |  |
| `AccountID` | `string` | `account_id` | `—` |  |
| `AccessToken` | `string` | `access_token` | `—` |  |
| `RefreshToken` | `string` | `refresh_token` | `—` |  |
| `TokenExpiresAt` | `*time.Time` | `token_expires_at` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Config` | `json.RawMessage` | `config` | `—` |  |

### `UpdateAccountInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `StoreName` | `*string` | `store_name` | `—` |  |
| `AccountID` | `*string` | `account_id` | `—` |  |
| `AccessToken` | `*string` | `access_token` | `—` |  |
| `RefreshToken` | `*string` | `refresh_token` | `—` |  |
| `TokenExpiresAt` | `*time.Time` | `token_expires_at` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |
| `SyncStatus` | `*string` | `sync_status` | `—` |  |
| `LastError` | `*string` | `last_error` | `—` |  |
| `Config` | `*json.RawMessage` | `config` | `—` |  |

### `AccountListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Search` | `string` | `` | `—` |  |
| `PlatformID` | `*int64` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |

### `CreateCategoryMappingInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `LocalCategoryID` | `int64` | `local_category_id` | `—` |  |
| `PlatformCategoryID` | `string` | `platform_category_id` | `—` |  |
| `PlatformCategoryName` | `string` | `platform_category_name` | `—` |  |

### `CreateAttributeMappingInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `LocalAttrName` | `string` | `local_attr_name` | `—` |  |
| `PlatformAttrID` | `string` | `platform_attr_id` | `—` |  |
| `PlatformAttrName` | `string` | `platform_attr_name` | `—` |  |
| `Required` | `bool` | `required` | `—` |  |

### `TestConnectionResult`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Success` | `bool` | `success` | `—` |  |
| `Message` | `string` | `message` | `—` |  |

### `WriteBackRecord`
**DB table:** `write_back_record`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `uint` | `` | `—` | PK |
| `ReferenceID` | `string` | `` | `—` |  |
| `AccountID` | `int64` | `` | `—` | NOT NULL |
| `Action` | `string` | `` | `—` | NOT NULL |
| `Payload` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` | default:pending |
| `Result` | `string` | `` | `—` |  |
| `Error` | `string` | `` | `—` |  |
| `RetryCount` | `int` | `` | `—` | default:0 |
| `CreatedAt` | `time.Time` | `` | `—` |  |
| `UpdatedAt` | `time.Time` | `` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
