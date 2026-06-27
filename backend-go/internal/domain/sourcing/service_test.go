package sourcing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
)

// mockToolBridge implements ToolBridge for testing.
type mockToolBridge struct {
	pageData *toolbridge.PageData
	err      error
}

func (m *mockToolBridge) Route(_ context.Context, _ string) (*toolbridge.PageData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.pageData, nil
}

// newToolbridgePageData is a test helper that creates a toolbridge.PageData
// from the common test fields used across sourcing tests.
func newToolbridgePageData(sourceURL, title string, price float64, images []string, supplierName string) *toolbridge.PageData {
	tb := &toolbridge.PageData{
		SourceURL:    sourceURL,
		Title:        title,
		PriceCNY:        price,
		Images:       images,
		SupplierName: supplierName,
		MOQ:          1,
		CollectedAt:  time.Now(),
	}
	if len(images) > 0 {
		tb.Images = images
	}
	return tb
}

// mockEventPublisher implements EventPublisher for testing.
type mockEventPublisher struct {
	lastTopic   string
	lastSource  string
	lastPayload map[string]interface{}
	err         error
}

func (m *mockEventPublisher) Publish(_ context.Context, topic, source string, payload map[string]interface{}) (string, error) {
	m.lastTopic = topic
	m.lastSource = source
	m.lastPayload = payload
	return "test-event-id", m.err
}

func newTestService(t *testing.T, bridge ToolBridge, events EventPublisher) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &sourcing1688.Sourcing1688Product{})
	return NewService(db, dbtest.NewLogger(t), bridge, events)
}

func TestFetchProduct_Success(t *testing.T) {
	t.Parallel()
	bridge := &mockToolBridge{
		pageData: newToolbridgePageData(
			"https://detail.1688.com/offer/test.html",
			"Test Product with Long Title for Scoring",
			99.50,
			[]string{"img1.jpg", "img2.jpg", "img3.jpg"},
			"Test Supplier",
		),
	}
	svc := newTestService(t, bridge, nil)

	data, err := svc.FetchProduct(context.Background(), "https://detail.1688.com/offer/test.html")
	if err != nil {
		t.Fatalf("FetchProduct: %v", err)
	}
	if data.Title != "Test Product with Long Title for Scoring" {
		t.Errorf("Title = %q", data.Title)
	}
	if data.Price != 99.50 {
		t.Errorf("Price = %v", data.Price)
	}
}

func TestFetchProduct_NoBridge(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	_, err := svc.FetchProduct(context.Background(), "https://detail.1688.com/offer/test.html")
	if err == nil {
		t.Fatal("expected error when ToolBridge not configured")
	}
}

func TestFetchProduct_BridgeError(t *testing.T) {
	t.Parallel()
	bridge := &mockToolBridge{err: errors.New("connection refused")}
	svc := newTestService(t, bridge, nil)

	_, err := svc.FetchProduct(context.Background(), "https://detail.1688.com/offer/test.html")
	if err == nil {
		t.Fatal("expected error from bridge")
	}
}

func TestAnalyzePage_FullScore(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	data := &PageData{
		Title:        "A very long product title that exceeds fifty characters easily for testing purposes",
		Price:        100.0,
		PriceMin:     floatPtr(90.0),
		Images:       []string{"a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg"},
		SupplierName: "Test Supplier",
	}

	score, reason := svc.AnalyzePage(data)
	if score < 8 {
		t.Errorf("expected high score, got %d (reason: %s)", score, reason)
	}
}

func TestAnalyzePage_MinimalData(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	data := &PageData{
		Title: "Short",
		Price: 0,
	}

	score, _ := svc.AnalyzePage(data)
	if score < 1 || score > 3 {
		t.Errorf("expected low score (1-3) for minimal data, got %d", score)
	}
}

func TestAnalyzePage_NilData(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	score, reason := svc.AnalyzePage(nil)
	if score != 0 {
		t.Errorf("expected score 0 for nil data, got %d", score)
	}
	if reason != "no data" {
		t.Errorf("expected reason 'no data', got %s", reason)
	}
}

func TestSaveRecommendation_Success(t *testing.T) {
	t.Parallel()
	bridge := &mockToolBridge{
		pageData: newToolbridgePageData(
			"https://detail.1688.com/offer/save-test.html",
			"Save Test Product",
			50.0,
			[]string{"img1.jpg"},
			"Save Supplier",
		),
	}
	publisher := &mockEventPublisher{}
	svc := newTestService(t, bridge, publisher)

	data, _ := svc.FetchProduct(context.Background(), "https://detail.1688.com/offer/save-test.html")
	score, reason := svc.AnalyzePage(data)

	product, err := svc.SaveRecommendation(context.Background(), data, score, reason)
	if err != nil {
		t.Fatalf("SaveRecommendation: %v", err)
	}
	if product.ID == 0 {
		t.Fatal("expected non-zero product ID")
	}
	if product.SourceURL != "https://detail.1688.com/offer/save-test.html" {
		t.Errorf("SourceURL = %q", product.SourceURL)
	}

	// Verify event was published
	if publisher.lastTopic != "sourcing.recommend" {
		t.Errorf("expected topic sourcing.recommend, got %s", publisher.lastTopic)
	}
	if publisher.lastSource != "A8" {
		t.Errorf("expected source A8, got %s", publisher.lastSource)
	}
}

func TestSaveRecommendation_NoPublisher(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	data := &PageData{
		SourceURL:    "https://detail.1688.com/offer/no-pub.html",
		Title:        "No Publisher Test",
		Price:        30.0,
		Images:       []string{"img.jpg"},
		SupplierName: "Supplier",
	}

	product, err := svc.SaveRecommendation(context.Background(), data, 7, "ok")
	if err != nil {
		t.Fatalf("SaveRecommendation: %v", err)
	}
	if product.ID == 0 {
		t.Fatal("expected non-zero product ID")
	}
}

func TestSaveRecommendation_NilData(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	_, err := svc.SaveRecommendation(context.Background(), nil, 0, "no data")
	if err == nil {
		t.Fatal("expected error for nil page data")
	}
}

func TestListRecommendations_Empty(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	items, total, err := svc.ListRecommendations(1, 20)
	if err != nil {
		t.Fatalf("ListRecommendations: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestListRecommendations_WithItems(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, nil, nil)

	// Save two recommendations
	data1 := &PageData{
		SourceURL:    "https://detail.1688.com/offer/list1.html",
		Title:        "List Product 1",
		Price:        10.0,
		Images:       []string{"img.jpg"},
		SupplierName: "Supplier 1",
	}
	data2 := &PageData{
		SourceURL:    "https://detail.1688.com/offer/list2.html",
		Title:        "List Product 2",
		Price:        20.0,
		Images:       []string{"img.jpg", "img2.jpg"},
		SupplierName: "Supplier 2",
	}

	svc.SaveRecommendation(context.Background(), data1, 8, "high_quality")
	svc.SaveRecommendation(context.Background(), data2, 5, "medium")

	items, total, err := svc.ListRecommendations(1, 20)
	if err != nil {
		t.Fatalf("ListRecommendations: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
