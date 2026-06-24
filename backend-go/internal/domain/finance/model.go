package finance

import (
	"time"
)

// FinanceAccount maps to "finance_account".
type FinanceAccount struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;not null" json:"name"`
	AccountType string    `gorm:"column:account_type;not null" json:"account_type"` // platform/payment/bank/cash
	PlatformID  *int64    `gorm:"column:platform_id" json:"platform_id,omitempty"`
	Currency    string    `gorm:"column:currency;default:CNY" json:"currency"`
	Balance     float64   `gorm:"column:balance;default:0" json:"balance"`
	Status      string    `gorm:"column:status;default:active" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (FinanceAccount) TableName() string { return "finance_account" }

// FinanceTransaction maps to "finance_transaction".
type FinanceTransaction struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AccountID       int64      `gorm:"column:account_id;not null;index" json:"account_id"`
	TransactionType string     `gorm:"column:transaction_type;not null" json:"transaction_type"` // revenue/cost/fee/refund/transfer
	Amount          float64    `gorm:"column:amount;not null" json:"amount"`
	Currency        string     `gorm:"column:currency;default:CNY" json:"currency"`
	OrderID         *int64     `gorm:"column:order_id" json:"order_id,omitempty"`
	SettlementID    *int64     `gorm:"column:settlement_id" json:"settlement_id,omitempty"`
	PlatformID      *int64     `gorm:"column:platform_id" json:"platform_id,omitempty"`
	Description     string     `gorm:"column:description" json:"description"`
	TransactionDate *time.Time `gorm:"column:transaction_date" json:"transaction_date,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (FinanceTransaction) TableName() string { return "finance_transaction" }

// FinanceLedgerEntry maps to "finance_ledger_entry".
type FinanceLedgerEntry struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID    *int64    `gorm:"column:order_id;index" json:"order_id,omitempty"`
	EntryType  string    `gorm:"column:entry_type;not null" json:"entry_type"` // revenue/product_cost/shipping_cost/platform_fee/payment_fee/refund/adjustment/other_fee
	Amount     float64   `gorm:"column:amount;not null" json:"amount"`
	Currency   string    `gorm:"column:currency;default:CNY" json:"currency"`
	CostLayer  string    `gorm:"column:cost_layer;default:estimated" json:"cost_layer"` // estimated/snapshot/actual/allocated
	SourceType string    `gorm:"column:source_type" json:"source_type"`
	SourceID   *int64    `gorm:"column:source_id" json:"source_id,omitempty"`
	Description string   `gorm:"column:description" json:"description"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (FinanceLedgerEntry) TableName() string { return "finance_ledger_entry" }

// ---------- Input / DTO structs ----------

// CreateAccountInput is the payload for POST /finance/accounts.
type CreateAccountInput struct {
	Name        string  `json:"name" binding:"required"`
	AccountType string  `json:"account_type" binding:"required"`
	PlatformID  *int64  `json:"platform_id"`
	Currency    string  `json:"currency"`
	Balance     *float64 `json:"balance"`
	Status      string  `json:"status"`
}

// UpdateAccountInput allows partial updates.
type UpdateAccountInput struct {
	Name        *string  `json:"name"`
	AccountType *string  `json:"account_type"`
	PlatformID  *int64  `json:"platform_id"`
	Currency    *string `json:"currency"`
	Balance     *float64 `json:"balance"`
	Status      *string `json:"status"`
}

// AccountListFilter captures query parameters.
type AccountListFilter struct {
	Search      string
	AccountType string
	Status      string
}

// CreateTransactionInput is the payload for POST /finance/transactions.
type CreateTransactionInput struct {
	AccountID       int64      `json:"account_id" binding:"required"`
	TransactionType string     `json:"transaction_type" binding:"required"`
	Amount          float64    `json:"amount" binding:"required"`
	Currency        string     `json:"currency"`
	OrderID         *int64     `json:"order_id"`
	SettlementID    *int64     `json:"settlement_id"`
	PlatformID      *int64     `json:"platform_id"`
	Description     string     `json:"description"`
	TransactionDate *time.Time `json:"transaction_date"`
}

// TransactionListFilter captures query parameters.
type TransactionListFilter struct {
	AccountID       *int64
	TransactionType string
	OrderID         *int64
}

// LedgerListFilter captures query parameters.
type LedgerListFilter struct {
	OrderID   *int64
	EntryType string
	CostLayer string
}

// FinanceSummary is the aggregation payload for GET /finance/summary.
type FinanceSummary struct {
	TotalBalance      float64            `json:"total_balance"`
	BalanceByType     map[string]float64 `json:"balance_by_type"`
	LedgerByEntryType map[string]float64 `json:"ledger_by_entry_type"`
	MonthRevenue      float64            `json:"month_revenue"`
	MonthCost         float64            `json:"month_cost"`
}

// ProfitSummary is the aggregation payload for GET /finance/profit-summary.
type ProfitSummary struct {
	TotalRevenue float64            `json:"total_revenue"`
	TotalCost    float64            `json:"total_cost"`
	TotalProfit  float64            `json:"total_profit"`
	ProfitMargin float64            `json:"profit_margin"`
	ByPlatform   []ProfitByPlatform `json:"by_platform"`
	ByMonth      []ProfitByMonth    `json:"by_month"`
}

// ProfitByPlatform is one row in ProfitSummary.ByPlatform.
type ProfitByPlatform struct {
	PlatformID *int64  `json:"platform_id,omitempty"`
	Revenue    float64 `json:"revenue"`
	Cost       float64 `json:"cost"`
	Profit     float64 `json:"profit"`
}

// ProfitByMonth is one row in ProfitSummary.ByMonth.
type ProfitByMonth struct {
	Month   string  `json:"month"`
	Revenue float64 `json:"revenue"`
	Cost    float64 `json:"cost"`
	Profit  float64 `json:"profit"`
}

// ProfitSummaryFilter captures query parameters for profit-summary.
type ProfitSummaryFilter struct {
	From       string
	To         string
	PlatformID *int64
}

// OrderProfit is the per-order profit summary for GET /finance/orders/:order_id/profit.
type OrderProfit struct {
	OrderID      int64   `json:"order_id"`
	Revenue      float64 `json:"revenue"`
	ProductCost  float64 `json:"product_cost"`
	ShippingCost float64 `json:"shipping_cost"`
	PlatformFee  float64 `json:"platform_fee"`
	PaymentFee   float64 `json:"payment_fee"`
	OtherFee     float64 `json:"other_fee"`
	Profit       float64 `json:"profit"`
	Margin       float64 `json:"margin"`
}

// MockDataInput is the payload for POST /finance/mock.
type MockDataInput struct {
	Count int `json:"count"`
}
