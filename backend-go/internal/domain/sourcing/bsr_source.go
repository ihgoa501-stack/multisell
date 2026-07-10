package sourcing

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

//go:embed testdata/amazon_bsr.csv
var bsrCSVData string

// AmazonBSRSource implements MarketTrendSource by providing mock Amazon
// Best Sellers Rank data loaded from an embedded CSV file.
//
// The CSV columns are: category,rank,product_title,price_range,review_count,avg_rating
//
// To replace with a real API, add a client field (e.g. *http.Client) and an
// endpoint URL, then call the real API in FetchTrends before falling back
// to the embedded CSV.
type AmazonBSRSource struct {
	// CSVData is the embedded CSV string. Exported for test injection.
	CSVData string
}

// NewAmazonBSRSource creates an AmazonBSRSource backed by the default
// embedded CSV data.
func NewAmazonBSRSource() *AmazonBSRSource {
	return &AmazonBSRSource{CSVData: bsrCSVData}
}

// Name returns "amazon_bsr".
func (s *AmazonBSRSource) Name() string {
	return "amazon_bsr"
}

// FetchTrends returns mock BSR data for the given product category.
// The query parameter is a category name (e.g. "家居"). Returns only items
// matching the category. Case-insensitive matching.
// FetchTrends returns mock BSR data for the given product category.
// The query parameter is a category name (e.g. "家居"). Returns only items
// matching the category. An empty query returns all items (market overview).
// Case-insensitive matching.
func (s *AmazonBSRSource) FetchTrends(_ context.Context, query string) ([]MarketTrendItem, error) {
	if s == nil {
		return nil, fmt.Errorf("amazon_bsr: nil source")
	}

	data := s.CSVData
	if data == "" {
		return nil, fmt.Errorf("amazon_bsr: no CSV data loaded")
	}

	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("amazon_bsr: CSV data has no data rows")
	}

	var results []MarketTrendItem
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		item, err := parseBSRLine(line)
		if err != nil {
			continue
		}
		item.Source = "amazon_bsr"
		if query == "" || strings.EqualFold(strings.TrimSpace(item.Category), strings.TrimSpace(query)) {
			results = append(results, item)
		}
	}

	return results, nil
}


func parseBSRLine(line string) (MarketTrendItem, error) {
	parts := splitCSV(line)
	if len(parts) < 6 {
		return MarketTrendItem{}, fmt.Errorf("amazon_bsr: expected 6 fields, got %d", len(parts))
	}

	rank, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return MarketTrendItem{}, fmt.Errorf("amazon_bsr: invalid rank %q: %w", parts[1], err)
	}

	reviewCount, err := strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		return MarketTrendItem{}, fmt.Errorf("amazon_bsr: invalid review_count %q: %w", parts[4], err)
	}

	avgRating, err := strconv.ParseFloat(strings.TrimSpace(parts[5]), 64)
	if err != nil {
		return MarketTrendItem{}, fmt.Errorf("amazon_bsr: invalid avg_rating %q: %w", parts[5], err)
	}

	return MarketTrendItem{
		Category:     strings.TrimSpace(parts[0]),
		Rank:         rank,
		ProductTitle: strings.TrimSpace(parts[2]),
		PriceRange:   strings.TrimSpace(parts[3]),
		ReviewCount:  reviewCount,
		AvgRating:    avgRating,
	}, nil
}

// splitCSV splits a CSV line into fields, handling quoted values.
// This is a simple implementation sufficient for the MVP mock data format
// (no multiline quoted fields).
func splitCSV(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, r := range line {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case r == ',' && !inQuotes:
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	fields = append(fields, current.String())
	return fields
}
