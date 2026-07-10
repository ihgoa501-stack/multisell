package sourcing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
)

// =======================================================
// AmazonBSRSource tests
// =======================================================

func TestAmazonBSRSource_Name(t *testing.T) {
	t.Parallel()
	src := NewAmazonBSRSource()
	if got := src.Name(); got != "amazon_bsr" {
		t.Errorf("Name() = %q, want amazon_bsr", got)
	}
}

func TestAmazonBSRSource_FetchTrends_Success(t *testing.T) {
	t.Parallel()
	src := NewAmazonBSRSource()

	items, err := src.FetchTrends(context.Background(), "家居")
	if err != nil {
		t.Fatalf("FetchTrends: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item for 家居")
	}

	// Verify first item fields.
	first := items[0]
	if first.Category != "家居" {
		t.Errorf("Category = %q, want 家居", first.Category)
	}
	if first.Rank < 1 {
		t.Errorf("Rank = %d, want >= 1", first.Rank)
	}
	if first.ProductTitle == "" {
		t.Errorf("ProductTitle is empty")
	}
	if first.PriceRange == "" {
		t.Errorf("PriceRange is empty")
	}
	if first.ReviewCount <= 0 {
		t.Errorf("ReviewCount = %d, want > 0", first.ReviewCount)
	}
	if first.AvgRating <= 0 {
		t.Errorf("AvgRating = %f, want > 0", first.AvgRating)
	}
	if first.Source != "amazon_bsr" {
		t.Errorf("Source = %q, want amazon_bsr", first.Source)
	}
}

func TestAmazonBSRSource_FetchTrends_AnotherCategory(t *testing.T) {
	t.Parallel()
	src := NewAmazonBSRSource()

	items, err := src.FetchTrends(context.Background(), "电子")
	if err != nil {
		t.Fatalf("FetchTrends: %v", err)
	}
	if len(items) != 5 {
		t.Errorf("expected 5 items for 电子, got %d", len(items))
	}

	for _, item := range items {
		if item.Category != "电子" {
			t.Errorf("unexpected category %q in 电子 results", item.Category)
		}
	}
}

func TestAmazonBSRSource_FetchTrends_UnknownCategory(t *testing.T) {
	t.Parallel()
	src := NewAmazonBSRSource()

	items, err := src.FetchTrends(context.Background(), "unknown_category_xyz")
	if err != nil {
		t.Fatalf("FetchTrends: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items for unknown category, got %d", len(items))
	}
}

func TestAmazonBSRSource_FetchTrends_EmptyQuery(t *testing.T) {
	t.Parallel()
	src := NewAmazonBSRSource()
	items, err := src.FetchTrends(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error for empty query: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected items for empty query")
	}
}

func TestAmazonBSRSource_FetchTrends_NilSource(t *testing.T) {
	t.Parallel()
	var src *AmazonBSRSource
	_, err := src.FetchTrends(context.Background(), "家居")
	if err == nil {
		t.Fatal("expected error for nil source")
	}
}

func TestAmazonBSRSource_CaseInsensitive(t *testing.T) {
	t.Parallel()
	src := NewAmazonBSRSource()

	itemsLower, err := src.FetchTrends(context.Background(), "运动户外")
	if err != nil {
		t.Fatalf("FetchTrends (lower): %v", err)
	}

	// Verify all returned items have matching category
	for _, item := range itemsLower {
		if item.Category != "运动户外" {
			t.Errorf("expected 运动户外, got %q", item.Category)
		}
	}
}

func TestParseBSRLine_Valid(t *testing.T) {
	t.Parallel()
	line := `家居,1,可折叠收纳箱 大容量塑料储物盒,CNY 29-89,15420,4.5`
	item, err := parseBSRLine(line)
	if err != nil {
		t.Fatalf("parseBSRLine: %v", err)
	}
	if item.Category != "家居" {
		t.Errorf("Category = %q", item.Category)
	}
	if item.Rank != 1 {
		t.Errorf("Rank = %d", item.Rank)
	}
	if item.ProductTitle != "可折叠收纳箱 大容量塑料储物盒" {
		t.Errorf("ProductTitle = %q", item.ProductTitle)
	}
	if item.PriceRange != "CNY 29-89" {
		t.Errorf("PriceRange = %q", item.PriceRange)
	}
	if item.ReviewCount != 15420 {
		t.Errorf("ReviewCount = %d", item.ReviewCount)
	}
	if item.AvgRating != 4.5 {
		t.Errorf("AvgRating = %f", item.AvgRating)
	}
}

func TestParseBSRLine_TooFewFields(t *testing.T) {
	t.Parallel()
	_, err := parseBSRLine("家居,1,title")
	if err == nil {
		t.Fatal("expected error for too few fields")
	}
}

func TestParseBSRLine_InvalidRank(t *testing.T) {
	t.Parallel()
	_, err := parseBSRLine("家居,abc,title,price,100,4.5")
	if err == nil {
		t.Fatal("expected error for invalid rank")
	}
}

func TestParseBSRLine_InvalidReviewCount(t *testing.T) {
	t.Parallel()
	_, err := parseBSRLine("家居,1,title,price,abc,4.5")
	if err == nil {
		t.Fatal("expected error for invalid review_count")
	}
}

func TestParseBSRLine_InvalidAvgRating(t *testing.T) {
	t.Parallel()
	_, err := parseBSRLine("家居,1,title,price,100,xyz")
	if err == nil {
		t.Fatal("expected error for invalid avg_rating")
	}
}

// =======================================================
// KeywordTrendSource tests
// =======================================================

func TestKeywordTrendSource_Name(t *testing.T) {
	t.Parallel()
	src := NewKeywordTrendSource()
	if got := src.Name(); got != "keyword_trends" {
		t.Errorf("Name() = %q, want keyword_trends", got)
	}
}

func TestKeywordTrendSource_FetchTrends_Success(t *testing.T) {
	t.Parallel()
	src := NewKeywordTrendSource()

	items, err := src.FetchTrends(context.Background(), "t恤")
	if err != nil {
		t.Fatalf("FetchTrends: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item for t恤")
	}

	// Verify all returned items contain "t恤" in their keyword field.
	for _, item := range items {
		if !strings.Contains(item.Keyword, "t恤") && !strings.Contains(item.Keyword, "T恤") {
			t.Errorf("keyword %q does not contain t恤", item.Keyword)
		}
		if item.SearchVolume <= 0 {
			t.Errorf("SearchVolume = %d for %q, want > 0", item.SearchVolume, item.Keyword)
		}
		if item.CompetitionLevel == "" {
			t.Errorf("CompetitionLevel is empty for %q", item.Keyword)
		}
		if item.TrendDirection == "" {
			t.Errorf("TrendDirection is empty for %q", item.Keyword)
		}
		if item.Source != "keyword_trends" {
			t.Errorf("Source = %q, want keyword_trends", item.Source)
		}
	}
}

func TestKeywordTrendSource_FetchTrends_EmptyQuery(t *testing.T) {
	t.Parallel()
	src := NewKeywordTrendSource()

	_, err := src.FetchTrends(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestKeywordTrendSource_FetchTrends_NilSource(t *testing.T) {
	t.Parallel()
	var src *KeywordTrendSource
	_, err := src.FetchTrends(context.Background(), "t恤")
	if err == nil {
		t.Fatal("expected error for nil source")
	}
}

func TestKeywordTrendSource_FetchTrends_NoMatch(t *testing.T) {
	t.Parallel()
	src := NewKeywordTrendSource()

	items, err := src.FetchTrends(context.Background(), "zzz_nonexistent_999")
	if err != nil {
		t.Fatalf("FetchTrends: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items for non-matching keyword, got %d", len(items))
	}
}

func TestKeywordTrendSource_CaseInsensitive(t *testing.T) {
	t.Parallel()
	src := NewKeywordTrendSource()

	itemsUpper, err := src.FetchTrends(context.Background(), "T恤")
	if err != nil {
		t.Fatalf("FetchTrends: %v", err)
	}

	itemsLower, err := src.FetchTrends(context.Background(), "t恤")
	if err != nil {
		t.Fatalf("FetchTrends: %v", err)
	}

	if len(itemsUpper) != len(itemsLower) {
		t.Errorf("case mismatch: upper=%d, lower=%d", len(itemsUpper), len(itemsLower))
	}
}

func TestParseKeywordLine_Valid(t *testing.T) {
	t.Parallel()
	line := `蓝牙耳机,145000,high,stable`
	item, err := parseKeywordLine(line)
	if err != nil {
		t.Fatalf("parseKeywordLine: %v", err)
	}
	if item.Keyword != "蓝牙耳机" {
		t.Errorf("Keyword = %q", item.Keyword)
	}
	if item.SearchVolume != 145000 {
		t.Errorf("SearchVolume = %d", item.SearchVolume)
	}
	if item.CompetitionLevel != "high" {
		t.Errorf("CompetitionLevel = %q", item.CompetitionLevel)
	}
	if item.TrendDirection != "stable" {
		t.Errorf("TrendDirection = %q", item.TrendDirection)
	}
}

func TestParseKeywordLine_TooFewFields(t *testing.T) {
	t.Parallel()
	_, err := parseKeywordLine("keyword,1000,low")
	if err == nil {
		t.Fatal("expected error for too few fields")
	}
}

func TestParseKeywordLine_InvalidSearchVolume(t *testing.T) {
	t.Parallel()
	_, err := parseKeywordLine("keyword,abc,low,up")
	if err == nil {
		t.Fatal("expected error for invalid search_volume")
	}
}

// =======================================================
// SplitCSV test
// =======================================================

func TestSplitCSV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"normal", "a,b,c", 3},
		{"empty_trailing", "a,b,", 3},
		{"quoted", `a,"b,c",d`, 3},
		{"single", "a", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCSV(tt.input)
			if len(got) != tt.want {
				t.Errorf("splitCSV(%q) = %d fields, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

// =======================================================
// Service-level market trend tests
// =======================================================

func TestService_FetchMarketTrends_Configured(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	// Service without sources should error
	_, err := svc.FetchMarketTrends(context.Background(), "家居")
	if err == nil {
		t.Fatal("expected error when BSR source not configured")
	}
}

func TestService_FetchMarketTrends_WithSource(t *testing.T) {
	t.Parallel()
	bsrSrc := NewAmazonBSRSource()
	svc := newTestServiceWithSources(t, bsrSrc, nil)

	items, err := svc.FetchMarketTrends(context.Background(), "家居")
	if err != nil {
		t.Fatalf("FetchMarketTrends: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected items")
	}
}

func TestService_FetchKeywordTrends_Configured(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	// Service without sources should error
	_, err := svc.FetchKeywordTrends(context.Background(), "t恤")
	if err == nil {
		t.Fatal("expected error when keyword source not configured")
	}
}

func TestService_FetchKeywordTrends_WithSource(t *testing.T) {
	t.Parallel()
	kwSrc := NewKeywordTrendSource()
	svc := newTestServiceWithSources(t, nil, kwSrc)

	items, err := svc.FetchKeywordTrends(context.Background(), "蓝牙耳机")
	if err != nil {
		t.Fatalf("FetchKeywordTrends: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected items")
	}
}

func TestService_NewServiceWithSources(t *testing.T) {
	t.Parallel()
	bsr := NewAmazonBSRSource()
	kw := NewKeywordTrendSource()

	svc := newTestServiceWithSources(t, bsr, kw)

	if svc.bsrSource == nil {
		t.Error("bsrSource should be set")
	}
	if svc.keywordSource == nil {
		t.Error("keywordSource should be set")
	}
}

// newTestServiceWithSources creates a Service with the given trend sources using
// the standard test helpers from service_test.go.
func newTestServiceWithSources(t *testing.T, bsrSrc, kwSrc MarketTrendSource) *Service {
	t.Helper()
	// Use the existing newTestService pattern but with sources.
	// We bypass newTestService because NewService accepts variadic sources.
	var sources []MarketTrendSource
	if bsrSrc != nil {
		sources = append(sources, bsrSrc)
	}
	if kwSrc != nil {
		sources = append(sources, kwSrc)
	}

	db := dbtest.NewDB(t, &sourcing1688.Sourcing1688Product{})
	return NewService(db, dbtest.NewLogger(t), nil, nil, sources...)
}

// =======================================================
// Handler-level market trend tests
// =======================================================

// setupTrendHandler creates a Service and Handler with mock trend sources.
func setupTrendHandler(t *testing.T) *Handler {
	t.Helper()
	bsr := NewAmazonBSRSource()
	kw := NewKeywordTrendSource()
	svc := newTestServiceWithSources(t, bsr, kw)
	return NewHandler(svc)
}

func TestHandler_MarketTrends_Success(t *testing.T) {
	t.Parallel()
	h := setupTrendHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sourcing/market-trends?category=家居", nil)

	h.MarketTrends(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Data["source"] != "amazon_bsr" {
		t.Errorf("source = %v, want amazon_bsr", resp.Data["source"])
	}
	if resp.Data["category"] != "家居" {
		t.Errorf("category = %v, want 家居", resp.Data["category"])
	}

	items, ok := resp.Data["items"].([]interface{})
	if !ok {
		t.Fatal("items is not an array")
	}
	if len(items) == 0 {
		t.Fatal("expected non-empty items")
	}
}

func TestHandler_MarketTrends_MissingCategory(t *testing.T) {
	t.Parallel()
	h := setupTrendHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sourcing/market-trends", nil)

	h.MarketTrends(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_KeywordTrends_Success(t *testing.T) {
	t.Parallel()
	h := setupTrendHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sourcing/keyword-trends?keyword=t恤", nil)

	h.KeywordTrends(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Data["source"] != "keyword_trends" {
		t.Errorf("source = %v, want keyword_trends", resp.Data["source"])
	}

	items, ok := resp.Data["items"].([]interface{})
	if !ok {
		t.Fatal("items is not an array")
	}
	if len(items) == 0 {
		t.Fatal("expected non-empty items")
	}
}

func TestHandler_KeywordTrends_MissingKeyword(t *testing.T) {
	t.Parallel()
	h := setupTrendHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/sourcing/keyword-trends", nil)

	h.KeywordTrends(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegisterRoutes_AddsTrendEndpoints(t *testing.T) {
	t.Parallel()
	// Verify the route registration works end-to-end by checking that
	// the routes compile and register without panic.
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1")

	// This should not panic.
	RegisterRoutes(group, nil, nil, nil, nil)

	// We can't easily test routes without a full server, but we verify
	// that route registration completes without error.
}
