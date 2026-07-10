package foreigntrade

import (
	"context"
	"time"
)

// RFQRecord represents a Request for Quotation (RFQ) in the database.
type RFQRecord struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RFQID       string    `gorm:"column:rfq_id;uniqueIndex;not null" json:"rfq_id"`
	BuyerName   string    `gorm:"column:buyer_name" json:"buyer_name"`
	ProductName string    `gorm:"column:product_name;not null" json:"product_name"`
	Quantity    int32     `gorm:"column:quantity;not null" json:"quantity"`
	TargetPrice float64   `gorm:"column:target_price" json:"target_price"`
	Currency    string    `gorm:"column:currency;default:USD" json:"currency"`
	Status      string    `gorm:"column:status;default:pending" json:"status"` // pending, quoted, expired, cancelled
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RFQRecord) TableName() string { return "rfq_record" }

// Quotation represents a submitted quotation for an RFQ.
type Quotation struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RFQRecordID  int64     `gorm:"column:rfq_record_id;index;not null" json:"rfq_record_id"`
	QuotePrice   float64   `gorm:"column:quote_price;not null" json:"quote_price"`
	Currency     string    `gorm:"column:currency;default:USD;not null" json:"currency"`
	ValidityDays int32     `gorm:"column:validity_days;default:30" json:"validity_days"`
	Status       string    `gorm:"column:status;default:submitted" json:"status"` // submitted, accepted, rejected
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Quotation) TableName() string { return "rfq_quotation" }

// RFQAdapter defines the contract for foreign trade RFQ operations.
type RFQAdapter interface {
	SubmitQuotation(ctx context.Context, rfqID string, quote *Quotation) (*QuotationResult, error)
}

// QuotationResult is the response from RFQAdapter.SubmitQuotation.
type QuotationResult struct {
	ExternalQuoteID string    `json:"external_quote_id"`
	Success         bool      `json:"success"`
	Message         string    `json:"message,omitempty"`
	SubmittedAt     time.Time `json:"submitted_at"`
}
