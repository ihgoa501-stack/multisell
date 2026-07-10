package foreigntrade

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// CreateRFQ inserts a new RFQ record.
func (s *Service) CreateRFQ(ctx context.Context, record *RFQRecord) error {
	return s.db.WithContext(ctx).Create(record).Error
}

// GetRFQByRFQID retrieves an RFQ by its RFQID string.
func (s *Service) GetRFQByRFQID(ctx context.Context, rfqID string) (*RFQRecord, error) {
	var record RFQRecord
	if err := s.db.WithContext(ctx).Where("rfq_id = ?", rfqID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// GenerateAndSubmitQuotation generates a quotation for an RFQ, saves it locally, and submits it using the RFQAdapter.
func (s *Service) GenerateAndSubmitQuotation(ctx context.Context, adapter RFQAdapter, rfqID string, quotePrice float64, currency string, validityDays int32) (*QuotationResult, error) {
	var rfq RFQRecord
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("rfq_id = ?", rfqID).First(&rfq).Error; err != nil {
			return err
		}
		if rfq.Status == "expired" || rfq.Status == "cancelled" {
			return fmt.Errorf("cannot quote for RFQ with status %s", rfq.Status)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	quote := &Quotation{
		RFQRecordID:  rfq.ID,
		QuotePrice:   quotePrice,
		Currency:     currency,
		ValidityDays: validityDays,
		Status:       "submitted",
	}

	// Submit via adapter
	result, err := adapter.SubmitQuotation(ctx, rfqID, quote)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return result, nil
	}

	// If successful, save quotation and update RFQ status in a transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(quote).Error; err != nil {
			return err
		}
		if err := tx.Model(&rfq).Update("status", "quoted").Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
