package listing

import (
	"context"

	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"gorm.io/gorm"
)

// candidateReaderAdapter implements CandidateReader by querying the candidate_product table via GORM.
type candidateReaderAdapter struct {
	db *gorm.DB
}

// NewCandidateReader creates a new CandidateReader backed by a *gorm.DB.
func NewCandidateReader(db *gorm.DB) CandidateReader {
	return &candidateReaderAdapter{db: db}
}

// GetByID fetches a candidate product by its ID and returns it as a listing.Candidate.
func (a *candidateReaderAdapter) GetByID(ctx context.Context, id uint) (*Candidate, error) {
	var c candidate.CandidateProduct
	if err := a.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &Candidate{
		ID:                 c.ID,
		Title:              c.Title,
		PurchasePrice:      c.PurchasePrice,
		PackageWeightKg:    c.PackageWeightKg,
		HSCode:             c.HSCode,
		OriginCountry:      c.OriginCountry,
		TargetSalePrice:    c.TargetSalePrice,
		PlatformID:         c.TargetPlatformID,
		DestinationCountry: c.DestinationCountry,
	}, nil
}
