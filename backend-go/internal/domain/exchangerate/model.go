package exchangerate

import "time"

// ExchangeRate maps to the "exchange_rate" table.
type ExchangeRate struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FromCurrency   string    `gorm:"column:from_currency;size:3;not null;index:idx_from_to_date" json:"from_currency"`
	ToCurrency     string    `gorm:"column:to_currency;size:3;not null;index:idx_from_to_date" json:"to_currency"`
	Rate           float64   `gorm:"column:rate;type:numeric(14,6);not null" json:"rate"`
	EffectiveDate  time.Time `gorm:"column:effective_date;not null;index:idx_from_to_date" json:"effective_date"`
	Source         string    `gorm:"column:source;size:20;default:manual" json:"source"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ExchangeRate) TableName() string { return "exchange_rate" }

// CreateInput is the payload for POST /exchange-rates.
type CreateInput struct {
	FromCurrency  string  `json:"from_currency" binding:"required,len=3"`
	ToCurrency    string  `json:"to_currency" binding:"required,len=3"`
	Rate          float64 `json:"rate" binding:"required,gt=0"`
	EffectiveDate string  `json:"effective_date" binding:"required"` // YYYY-MM-DD
	Source        string  `json:"source"`
}

// UpdateInput is the payload for PUT /exchange-rates/:from/:to.
type UpdateInput struct {
	Rate          float64 `json:"rate" binding:"required,gt=0"`
	EffectiveDate string  `json:"effective_date"` // optional, YYYY-MM-DD
}

// ListFilter captures query parameters.
type ListFilter struct {
	FromCurrency  string
	ToCurrency    string
	EffectiveDate string
}
