package exchangerate

import (
	"errors"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides exchange-rate business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new exchange-rate service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated exchange rates with optional filters.
func (s *Service) List(p *common.Pagination, f *ListFilter) ([]ExchangeRate, int64, error) {
	q := s.db.Model(&ExchangeRate{})
	if f != nil {
		if f.FromCurrency != "" {
			q = q.Where("UPPER(from_currency) = ?", strings.ToUpper(f.FromCurrency))
		}
		if f.ToCurrency != "" {
			q = q.Where("UPPER(to_currency) = ?", strings.ToUpper(f.ToCurrency))
		}
		if f.EffectiveDate != "" {
			q = q.Where("DATE(effective_date) = ?", f.EffectiveDate)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []ExchangeRate
	if err := q.Order("effective_date DESC, id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Create inserts a new exchange rate.
func (s *Service) Create(in *CreateInput) (*ExchangeRate, error) {
	effDate, err := time.Parse("2006-01-02", in.EffectiveDate)
	if err != nil {
		return nil, errors.New("effective_date must be YYYY-MM-DD")
	}
	source := in.Source
	if source == "" {
		source = "manual"
	}
	er := ExchangeRate{
		FromCurrency:  strings.ToUpper(in.FromCurrency),
		ToCurrency:    strings.ToUpper(in.ToCurrency),
		Rate:          in.Rate,
		EffectiveDate: effDate,
		Source:        source,
	}
	if err := s.db.Create(&er).Error; err != nil {
		return nil, err
	}
	return &er, nil
}

// UpdateByPair updates the most recent rate for a from→to currency pair.
func (s *Service) UpdateByPair(from, to string, in *UpdateInput) (*ExchangeRate, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	var er ExchangeRate
	if err := s.db.Where("from_currency = ? AND to_currency = ?", from, to).
		Order("effective_date DESC").
		First(&er).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("exchange rate not found for " + from + "/" + to)
		}
		return nil, err
	}
	updates := map[string]interface{}{"rate": in.Rate}
	if in.EffectiveDate != "" {
		effDate, err := time.Parse("2006-01-02", in.EffectiveDate)
		if err != nil {
			return nil, errors.New("effective_date must be YYYY-MM-DD")
		}
		updates["effective_date"] = effDate
	}
	if err := s.db.Model(&er).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&er, er.ID).Error; err != nil {
		return nil, err
	}
	return &er, nil
}

// Delete removes an exchange rate by ID.
func (s *Service) Delete(id int64) error {
	return s.db.Delete(&ExchangeRate{}, id).Error
}

// GetLatest returns the most recent rate for a currency pair, or an error.
func (s *Service) GetLatest(from, to string) (*ExchangeRate, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	var er ExchangeRate
	if err := s.db.Where("from_currency = ? AND to_currency = ?", from, to).
		Order("effective_date DESC").
		First(&er).Error; err != nil {
		return nil, err
	}
	return &er, nil
}
