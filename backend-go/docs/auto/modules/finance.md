# Module: `finance`

Package: `backend-go/internal/domain/finance/`

**Base mount prefix:** `/api/v1`
**Required permission:** `finance.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/finance/accounts` | `h.ListAccounts` |
| `POST` | `/api/v1/finance/accounts` | `h.CreateAccount` |
| `DELETE` | `/api/v1/finance/accounts/:id` | `h.DeleteAccount` |
| `GET` | `/api/v1/finance/accounts/:id` | `h.GetAccount` |
| `PUT` | `/api/v1/finance/accounts/:id` | `h.UpdateAccount` |
| `GET` | `/api/v1/finance/cash-receipts` | `h.ListCashReceipts` |
| `POST` | `/api/v1/finance/cash-receipts` | `h.CreateCashReceipt` |
| `GET` | `/api/v1/finance/cash-reconciliations` | `h.ListCashReconciliations` |
| `POST` | `/api/v1/finance/cash-reconciliations` | `h.CreateCashReconciliation` |
| `GET` | `/api/v1/finance/ledger` | `h.ListLedger` |
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

## Models

### `FinanceAccount`
**DB table:** `finance_account`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OwnerID` | `int64` | `owner_id` | `owner_id` | NOT NULL |
| `Name` | `string` | `name` | `name` | NOT NULL |
| `AccountType` | `string` | `account_type` | `account_type` | NOT NULL |
| `PlatformID` | `*int64` | `platform_id,omitempty` | `platform_id` |  |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `Balance` | `float64` | `balance` | `balance` | default:0 |
| `Status` | `string` | `status` | `status` | default:active |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `FinanceTransaction`
**DB table:** `finance_transaction`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `AccountID` | `int64` | `account_id` | `account_id` | NOT NULL |
| `TransactionType` | `string` | `transaction_type` | `transaction_type` | NOT NULL |
| `Amount` | `float64` | `amount` | `amount` | NOT NULL |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `OrderID` | `*int64` | `order_id,omitempty` | `order_id` |  |
| `SettlementID` | `*int64` | `settlement_id,omitempty` | `settlement_id` |  |
| `PlatformID` | `*int64` | `platform_id,omitempty` | `platform_id` |  |
| `Description` | `string` | `description` | `description` |  |
| `TransactionDate` | `*time.Time` | `transaction_date,omitempty` | `transaction_date` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `FinanceLedgerEntry`
**DB table:** `finance_ledger_entry`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OrderID` | `*int64` | `order_id,omitempty` | `order_id` |  |
| `EntryType` | `string` | `entry_type` | `entry_type` | NOT NULL |
| `Amount` | `float64` | `amount` | `amount` | NOT NULL |
| `Currency` | `string` | `currency` | `currency` | default:CNY |
| `CostLayer` | `string` | `cost_layer` | `cost_layer` | default:estimated |
| `SourceType` | `string` | `source_type` | `source_type` |  |
| `SourceID` | `*int64` | `source_id,omitempty` | `source_id` |  |
| `Description` | `string` | `description` | `description` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `CreateAccountInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OwnerID` | `int64` | `-` | `—` |  |
| `Name` | `string` | `name` | `—` |  |
| `AccountType` | `string` | `account_type` | `—` |  |
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |
| `Balance` | `*float64` | `balance` | `—` |  |
| `Status` | `string` | `status` | `—` |  |

### `UpdateAccountInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Name` | `*string` | `name` | `—` |  |
| `AccountType` | `*string` | `account_type` | `—` |  |
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `Currency` | `*string` | `currency` | `—` |  |
| `Balance` | `*float64` | `balance` | `—` |  |
| `Status` | `*string` | `status` | `—` |  |

### `AccountListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Search` | `string` | `` | `—` |  |
| `AccountType` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |

### `CreateTransactionInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `AccountID` | `int64` | `account_id` | `—` |  |
| `TransactionType` | `string` | `transaction_type` | `—` |  |
| `Amount` | `float64` | `amount` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |
| `OrderID` | `*int64` | `order_id` | `—` |  |
| `SettlementID` | `*int64` | `settlement_id` | `—` |  |
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `Description` | `string` | `description` | `—` |  |
| `TransactionDate` | `*time.Time` | `transaction_date` | `—` |  |

### `TransactionListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `AccountID` | `*int64` | `` | `—` |  |
| `TransactionType` | `string` | `` | `—` |  |
| `OrderID` | `*int64` | `` | `—` |  |

### `LedgerListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderID` | `*int64` | `` | `—` |  |
| `EntryType` | `string` | `` | `—` |  |
| `CostLayer` | `string` | `` | `—` |  |

### `FinanceSummary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TotalBalance` | `float64` | `total_balance` | `—` |  |
| `BalanceByType` | `map[string]float64` | `balance_by_type` | `—` |  |
| `LedgerByEntryType` | `map[string]float64` | `ledger_by_entry_type` | `—` |  |
| `MonthRevenue` | `float64` | `month_revenue` | `—` |  |
| `MonthCost` | `float64` | `month_cost` | `—` |  |

### `ProfitSummary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TotalRevenue` | `float64` | `total_revenue` | `—` |  |
| `TotalCost` | `float64` | `total_cost` | `—` |  |
| `TotalProfit` | `float64` | `total_profit` | `—` |  |
| `ProfitMargin` | `float64` | `profit_margin` | `—` |  |
| `ByPlatform` | `[]ProfitByPlatform` | `by_platform` | `—` |  |
| `ByMonth` | `[]ProfitByMonth` | `by_month` | `—` |  |

### `ProfitByPlatform`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `PlatformID` | `*int64` | `platform_id,omitempty` | `—` |  |
| `Revenue` | `float64` | `revenue` | `—` |  |
| `Cost` | `float64` | `cost` | `—` |  |
| `Profit` | `float64` | `profit` | `—` |  |

### `ProfitByMonth`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Month` | `string` | `month` | `—` |  |
| `Revenue` | `float64` | `revenue` | `—` |  |
| `Cost` | `float64` | `cost` | `—` |  |
| `Profit` | `float64` | `profit` | `—` |  |

### `ProfitSummaryFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `From` | `string` | `` | `—` |  |
| `To` | `string` | `` | `—` |  |
| `PlatformID` | `*int64` | `` | `—` |  |

### `OrderProfit`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderID` | `int64` | `order_id` | `—` |  |
| `Revenue` | `float64` | `revenue` | `—` |  |
| `ProductCost` | `float64` | `product_cost` | `—` |  |
| `ShippingCost` | `float64` | `shipping_cost` | `—` |  |
| `PlatformFee` | `float64` | `platform_fee` | `—` |  |
| `PaymentFee` | `float64` | `payment_fee` | `—` |  |
| `OtherFee` | `float64` | `other_fee` | `—` |  |
| `Profit` | `float64` | `profit` | `—` |  |
| `Margin` | `float64` | `margin` | `—` |  |

### `MockDataInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Count` | `int` | `count` | `—` |  |

### `TransferRequest`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SourceAccount` | `string` | `source_account` | `—` |  |
| `TargetAccount` | `string` | `target_account` | `—` |  |
| `AmountCents` | `int64` | `amount_cents` | `—` |  |
| `Currency` | `string` | `currency` | `—` |  |
| `Description` | `string` | `description` | `—` |  |

### `TransferResponse`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `TransactionID` | `string` | `transaction_id` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `TransferredAt` | `time.Time` | `transferred_at` | `—` |  |
| `ErrorMessage` | `string` | `error_message,omitempty` | `—` |  |

### `LedgerEntry`
**DB table:** `ledger_entry`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `TransactionID` | `string` | `transaction_id` | `transaction_id` | NOT NULL |
| `AccountID` | `int64` | `account_id` | `account_id` | NOT NULL |
| `Description` | `string` | `description` | `description` |  |
| `DebitCents` | `int64` | `debit_cents` | `debit_cents` | NOT NULL, default:0 |
| `CreditCents` | `int64` | `credit_cents` | `credit_cents` | NOT NULL, default:0 |
| `PriceCents` | `int64` | `price_cents` | `price_cents` | NOT NULL, default:0 |
| `Currency` | `string` | `currency` | `currency` | NOT NULL, default:CNY |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
