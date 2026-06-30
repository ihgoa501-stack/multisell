package sourcing

import "context"

// MarketTrendItem represents a single data point from an external market
// trend data source. Different sources populate different fields:
//
// Amazon BSR sources populate: Category, Rank, ProductTitle, PriceRange,
// ReviewCount, AvgRating.
//
// Keyword trend sources populate: Keyword, SearchVolume, CompetitionLevel,
// TrendDirection.
type MarketTrendItem struct {
	Source string `json:"source"`

	// Amazon BSR fields
	Category     string  `json:"category,omitempty"`
	Rank         int     `json:"rank,omitempty"`
	ProductTitle string  `json:"product_title,omitempty"`
	PriceRange   string  `json:"price_range,omitempty"`
	ReviewCount  int     `json:"review_count,omitempty"`
	AvgRating    float64 `json:"avg_rating,omitempty"`

	// Keyword trend fields
	Keyword          string `json:"keyword,omitempty"`
	SearchVolume     int    `json:"search_volume,omitempty"`
	CompetitionLevel string `json:"competition_level,omitempty"`
	TrendDirection   string `json:"trend_direction,omitempty"`
}

// MarketTrendSource defines the interface for external market trend data
// providers. Each implementation loads data from its own backend (mock CSV,
// real API, etc.).
//
// Implementations must be safe for concurrent use.
type MarketTrendSource interface {
	// Name returns the human-readable display name of this data source.
	Name() string

	// FetchTrends retrieves market trend data matching the query string.
	// For BSR sources the query is a category name; for keyword sources
	// the query is a search keyword. Returns an error if the source is
	// unavailable or the query is empty.
	FetchTrends(ctx context.Context, query string) ([]MarketTrendItem, error)
}
