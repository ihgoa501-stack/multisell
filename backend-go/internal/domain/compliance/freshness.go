package compliance

import "gorm.io/gorm"

// FreshnessWriter updates the data_freshness table for the compliance dimension.
//
// ponytail: direct SQL for simplicity. Extract to a shared freshness service
// if more consumers appear.
type FreshnessWriter struct {
	db *gorm.DB
}

// NewFreshnessWriter creates a new FreshnessWriter.
func NewFreshnessWriter(db *gorm.DB) *FreshnessWriter {
	return &FreshnessWriter{db: db}
}

// RecordVerification upserts a freshness record for the compliance dimension.
func (w *FreshnessWriter) RecordVerification(productID int64, status string) error {
	return w.db.Exec(`
		INSERT INTO data_freshness (product_id, dimension, status, last_verified_at, created_at, updated_at)
		VALUES (?, 'compliance', ?, NOW(), NOW(), NOW())
		ON CONFLICT (product_id, dimension) DO UPDATE SET
			status = EXCLUDED.status,
			last_verified_at = NOW(),
			updated_at = NOW()
	`, productID, status).Error
}
