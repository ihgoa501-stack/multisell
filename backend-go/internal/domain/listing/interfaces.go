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

// SKUProvider provides SKU lookups without importing the sku package.
type SKUProvider interface {
	GetByIDs(ctx context.Context, ids []int64) ([]Sku, error)
	// TODO: wire via dependency injection
}

// DecisionReader provides read access to pre-listing decisions without importing the decision package.
type DecisionReader interface {
	GetByIDs(ctx context.Context, ids []int64) ([]PreListingDecision, error)
	// TODO: wire via dependency injection
}
