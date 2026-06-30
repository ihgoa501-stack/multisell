package sourcing

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

//go:embed testdata/keyword_trends.csv
var keywordCSVData string

// KeywordTrendSource implements MarketTrendSource by providing mock keyword
// search volume and competition data loaded from an embedded CSV file.
//
// The CSV columns are: keyword,search_volume,competition_level,trend_direction
//
// To replace with a real API, add a client field (e.g. *http.Client) and an
// API endpoint URL, then call the real API in FetchTrends before falling back
// to the embedded CSV.
type KeywordTrendSource struct {
	// CSVData is the embedded CSV string. Exported for test injection.
	CSVData string
}

// NewKeywordTrendSource creates a KeywordTrendSource backed by the default
// embedded CSV data.
func NewKeywordTrendSource() *KeywordTrendSource {
	return &KeywordTrendSource{CSVData: keywordCSVData}
}

// Name returns "keyword_trends".
func (s *KeywordTrendSource) Name() string {
	return "keyword_trends"
}

// FetchTrends returns mock keyword trend data matching the given keyword.
// The query parameter is a search keyword (e.g. "t恤 男"). Returns only items
// where the keyword field contains the query (substring match, case-insensitive).
func (s *KeywordTrendSource) FetchTrends(_ context.Context, query string) ([]MarketTrendItem, error) {
	if s == nil {
		return nil, fmt.Errorf("keyword_trends: nil source")
	}
	if query == "" {
		return nil, fmt.Errorf("keyword_trends: query (keyword) is required")
	}

	data := s.CSVData
	if data == "" {
		return nil, fmt.Errorf("keyword_trends: no CSV data loaded")
	}

	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("keyword_trends: CSV data has no data rows")
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	var results []MarketTrendItem

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		item, err := parseKeywordLine(line)
		if err != nil {
			continue // skip malformed rows gracefully
		}
		if strings.Contains(strings.ToLower(item.Keyword), queryLower) {
			item.Source = "keyword_trends"
			results = append(results, item)
		}
	}

	return results, nil
}

// parseKeywordLine parses a single CSV line into a MarketTrendItem.
// Expected format: keyword,search_volume,competition_level,trend_direction
func parseKeywordLine(line string) (MarketTrendItem, error) {
	parts := splitCSV(line)
	if len(parts) < 4 {
		return MarketTrendItem{}, fmt.Errorf("keyword_trends: expected 4 fields, got %d", len(parts))
	}

	searchVolume, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return MarketTrendItem{}, fmt.Errorf("keyword_trends: invalid search_volume %q: %w", parts[1], err)
	}

	return MarketTrendItem{
		Keyword:          strings.TrimSpace(parts[0]),
		SearchVolume:     searchVolume,
		CompetitionLevel: strings.TrimSpace(parts[2]),
		TrendDirection:   strings.TrimSpace(parts[3]),
	}, nil
}
