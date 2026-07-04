package compliance

import (
	"context"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ScannedProduct represents a product row loaded for compliance checking.
type ScannedProduct struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Country  string `json:"country"`
	Platform string `json:"platform"`
}

// ScanResult summarizes a compliance scan batch.
type ScanResult struct {
	TotalProducts   int           `json:"total_products"`
	ScannedProducts int           `json:"scanned_products"`
	IssuesFound     int           `json:"issues_found"`
	Duration        time.Duration `json:"duration"`
	Errors          []string      `json:"errors,omitempty"`
}

// Scanner performs batch compliance scanning on all products.
// It pages through the product table in batches; real compliance rules will
// be added as the Compliance Risk Engine matures.
type Scanner struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewScanner creates a new Scanner.
func NewScanner(db *gorm.DB, logger *zap.Logger) *Scanner {
	return &Scanner{db: db, logger: logger}
}

// productQueryRow is the raw DB result row for product scanning queries.
// The product table does not have target_country or target_platform columns,
// so those fields are not queried here.
type productQueryRow struct {
	ID         int64
	Name       string
	CategoryID int64
}

// ScanPaginated iterates over all products in pages and applies compliance
// checks to each. Returns a summary of the batch.
//
// Full compliance checking (target platform, target country) requires joining
// with platform and listing tables — that will be added in a future iteration.
func (s *Scanner) ScanPaginated(ctx context.Context) (*ScanResult, error) {
	const pageSize = 100
	start := time.Now()

	var total int64
	if err := s.db.Raw("SELECT COUNT(*) FROM product").Scan(&total).Error; err != nil {
		return nil, fmt.Errorf("count products: %w", err)
	}

	result := &ScanResult{
		TotalProducts: int(total),
	}

	offset := 0
	for {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(start)
			return result, ctx.Err()
		default:
		}

		var rows []productQueryRow
		if err := s.db.Raw(
			"SELECT id, name, category_id FROM product ORDER BY id LIMIT ? OFFSET ?",
			pageSize, offset,
		).Scan(&rows).Error; err != nil {
			result.Errors = append(result.Errors, err.Error())
			break
		}

		if len(rows) == 0 {
			break
		}

		// ponytail: compliance rules not yet wired — requires platform/listing join
		result.ScannedProducts += len(rows)
		offset += pageSize
	}

	result.Duration = time.Since(start)

	if len(result.Errors) > 0 {
		s.logger.Warn("compliance scan completed with errors",
			zap.Int("total", result.TotalProducts),
			zap.Int("scanned", result.ScannedProducts),
			zap.Int("issues", result.IssuesFound),
			zap.Duration("duration", result.Duration),
			zap.Strings("errors", result.Errors),
		)
	} else {
		s.logger.Info("compliance scan completed",
			zap.Int("total", result.TotalProducts),
			zap.Int("scanned", result.ScannedProducts),
			zap.Int("issues", result.IssuesFound),
			zap.Duration("duration", result.Duration),
		)
	}

	return result, nil
}

// HandleTick returns an eventbus.Handler that runs a compliance scan on each
// scheduler tick. The ScanResult is logged but otherwise discarded; future
// iterations will route issues to the compliance dashboard.
func HandleTick(db *gorm.DB, logger *zap.Logger) eventbus.Handler {
	return func(ctx context.Context, evt eventbus.Event) error {
		scanner := NewScanner(db, logger)
		_, err := scanner.ScanPaginated(ctx)
		if err != nil {
			logger.Error("compliance scan tick failed", zap.Error(err))
		}
		return err
	}
}
