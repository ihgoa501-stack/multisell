# Module: `support`

Package: `backend-go/internal/domain/support/`

**Base mount prefix:** `/api/v1`

## API Routes

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

## Models

### `CustomerConversation`
**DB table:** `customer_conversations`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OrderID` | `*int64` | `order_id,omitempty` | `order_id` |  |
| `Platform` | `string` | `platform` | `platform` | NOT NULL |
| `CustomerName` | `string` | `customer_name` | `customer_name` | NOT NULL |
| `CustomerEmail` | `string` | `customer_email` | `customer_email` | NOT NULL |
| `Subject` | `string` | `subject` | `subject` | NOT NULL |
| `Status` | `string` | `status` | `status` | NOT NULL, default:open |
| `Priority` | `string` | `priority` | `priority` | NOT NULL, default:medium |
| `AssignedTo` | `*string` | `assigned_to,omitempty` | `assigned_to` |  |
| `LastMessageAt` | `*time.Time` | `last_message_at,omitempty` | `last_message_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |
| `Messages` | `[]ChatMessage` | `messages,omitempty` | `—` |  |

### `ChatMessage`
**DB table:** `chat_messages`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `ConversationID` | `int64` | `conversation_id` | `conversation_id` | NOT NULL |
| `SenderType` | `string` | `sender_type` | `sender_type` | NOT NULL |
| `Content` | `string` | `content` | `content` | NOT NULL |
| `AutoReplied` | `bool` | `auto_replied` | `auto_replied` | default:false |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `AutoReplyTemplate`
**DB table:** `auto_reply_templates`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `Name` | `string` | `name` | `name` | NOT NULL |
| `Category` | `string` | `category` | `category` | NOT NULL |
| `Content` | `string` | `content` | `content` | NOT NULL |
| `Variables` | `json.RawMessage` | `variables,omitempty` | `variables` |  |
| `Platform` | `string` | `platform` | `platform` |  |
| `Enabled` | `bool` | `enabled` | `enabled` | default:true |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `BlacklistEntry`
**DB table:** `blacklist_entries`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `CustomerEmail` | `string` | `customer_email` | `customer_email` | NOT NULL |
| `CustomerName` | `string` | `customer_name` | `customer_name` |  |
| `Reason` | `string` | `reason` | `reason` |  |
| `AddedBy` | `string` | `added_by` | `added_by` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `CreateConversationInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderID` | `*int64` | `order_id` | `—` |  |
| `Platform` | `string` | `platform` | `—` |  |
| `CustomerName` | `string` | `customer_name` | `—` |  |
| `CustomerEmail` | `string` | `customer_email` | `—` |  |
| `Subject` | `string` | `subject` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `Priority` | `string` | `priority` | `—` |  |
| `AssignedTo` | `*string` | `assigned_to` | `—` |  |

### `UpdateConversationInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderID` | `*int64` | `order_id` | `—` |  |
| `Platform` | `*string` | `platform` | `—` |  |
| `CustomerName` | `*string` | `customer_name` | `—` |  |
| `CustomerEmail` | `*string` | `customer_email` | `—` |  |
| `Subject` | `*string` | `subject` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |
| `Priority` | `*string` | `priority` | `—` |  |
| `AssignedTo` | `*string` | `assigned_to` | `—` |  |

### `SendReplyInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Content` | `string` | `content` | `—` |  |
| `IsAuto` | `bool` | `is_auto` | `—` |  |

### `CreateTemplateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Name` | `string` | `name` | `—` |  |
| `Category` | `string` | `category` | `—` |  |
| `Content` | `string` | `content` | `—` |  |
| `Variables` | `json.RawMessage` | `variables` | `—` |  |
| `Platform` | `string` | `platform` | `—` |  |
| `Enabled` | `*bool` | `enabled` | `—` |  |

### `UpdateTemplateInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Name` | `*string` | `name` | `—` |  |
| `Category` | `*string` | `category` | `—` |  |
| `Content` | `*string` | `content` | `—` |  |
| `Variables` | `*json.RawMessage` | `variables` | `—` |  |
| `Platform` | `*string` | `platform` | `—` |  |
| `Enabled` | `*bool` | `enabled` | `—` |  |

### `CreateBlacklistInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `CustomerEmail` | `string` | `customer_email` | `—` |  |
| `CustomerName` | `string` | `customer_name` | `—` |  |
| `Reason` | `string` | `reason` | `—` |  |
| `AddedBy` | `string` | `added_by` | `—` |  |

### `ConversationFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Status` | `string` | `status` | `—` |  |
| `Priority` | `string` | `priority` | `—` |  |
| `Platform` | `string` | `platform` | `—` |  |
| `Search` | `string` | `search` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
