package listing

import "context"

// Sku is the listing view of a SKU (only fields listing reads).
type Sku struct {
	ID        int64
	ProductID int64
}

// PreListingDecision is the listing view of a pre-listing decision (only fields listing reads).
type PreListingDecision struct {
	ID          int64
	SkuID       int64
	PlatformID  *int64
	CountryCode string
}

// Candidate is the listing view of a candidate product.
type Candidate struct {
	ID                 int64
	Title              string
	PurchasePrice      float64
	PackageWeightKg    float64
	HSCode             string
	OriginCountry      string
	TargetSalePrice    float64
	PlatformID         *int64
	DestinationCountry string
}

// ProfitSummary is the listing view of a profit summary.
type ProfitSummary struct {
	TotalCost       float64
	TargetRevenue   float64
	EstimatedProfit float64
	ProfitMargin    float64
	Status          string
}

// SKUProvider provides SKU lookups without importing the sku package.
type SKUProvider interface {
	GetByIDs(ctx context.Context, ids []int64) ([]Sku, error)
}

// DecisionReader provides read access to pre-listing decisions without importing the decision package.
type DecisionReader interface {
	GetByIDs(ctx context.Context, ids []int64) ([]PreListingDecision, error)
}

// CandidateReader provides candidate product lookups without importing the candidate package.
type CandidateReader interface {
	GetByID(ctx context.Context, id uint) (*Candidate, error)
}

// ProfitReader provides profit summary lookups without importing the profit package.
type ProfitReader interface {
	GetByProductID(productID int64) (*ProfitSummary, error)
}
