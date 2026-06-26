package drivers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// mockExtensionHandler — test double for ExtensionHandler interface
// ---------------------------------------------------------------------------

type mockExtensionHandler struct {
	isOnlineFn    func(userID int64) bool
	sendRequestFn func(ctx context.Context, userID int64, req *ExtensionRequest) (*ExtensionResponse, error)
}

func (m *mockExtensionHandler) IsOnline(userID int64) bool {
	if m.isOnlineFn != nil {
		return m.isOnlineFn(userID)
	}
	return true
}

func (m *mockExtensionHandler) SendRequest(ctx context.Context, userID int64, req *ExtensionRequest) (*ExtensionResponse, error) {
	if m.sendRequestFn != nil {
		return m.sendRequestFn(ctx, userID, req)
	}
	return nil, errors.New("not implemented")
}

// Helper: build a valid fetch_product_result response payload.
func okResultPayload(data interface{}) json.RawMessage {
	raw, _ := json.Marshal(FetchProductResult{
		Status: "ok",
		Data:   mustMarshal(data),
	})
	return raw
}

// Helper: build a valid fetch_product_error response payload.
func errResultPayload(code, message string) json.RawMessage {
	raw, _ := json.Marshal(FetchProductError{
		Code:    code,
		Message: message,
	})
	return raw
}

func mustMarshal(v interface{}) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPluginDriver_FetchPage_Success(t *testing.T) {
	logger := zap.NewNop()
	handler := &mockExtensionHandler{
		sendRequestFn: func(_ context.Context, _ int64, req *ExtensionRequest) (*ExtensionResponse, error) {
			if req.Type != CommandFetchProduct {
				t.Fatalf("expected command type %q, got %q", CommandFetchProduct, req.Type)
			}
			return &ExtensionResponse{
				Type:    ResultFetchProduct,
				ID:      req.ID,
				Payload: okResultPayload(toolbridge.PageData{Title: "1688 Item", Price: 15.50, Currency: "CNY"}),
			}, nil
		},
	}

	driver := NewPluginDriver(handler, logger)
	driver.MarkOnline() // simulate ping received

	result, err := driver.FetchPage(context.Background(), "https://detail.1688.com/offer/123.html")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Title != "1688 Item" {
		t.Fatalf("expected title '1688 Item', got: %q", result.Title)
	}
	if result.Price != 15.50 {
		t.Fatalf("expected price 15.50, got: %f", result.Price)
	}
	if result.Currency != "CNY" {
		t.Fatalf("expected currency 'CNY', got: %q", result.Currency)
	}
	if result.SourceURL != "https://detail.1688.com/offer/123.html" {
		t.Fatalf("expected source URL to be preserved, got: %q", result.SourceURL)
	}
	if result.Driver != "plugin" {
		t.Fatalf("expected driver 'plugin', got: %q", result.Driver)
	}
	if result.CollectedAt.IsZero() {
		t.Fatal("expected CollectedAt to be set")
	}
}

func TestPluginDriver_FetchPage_ExtensionOffline(t *testing.T) {
	logger := zap.NewNop()
	handler := &mockExtensionHandler{}
	driver := NewPluginDriver(handler, logger)
	// Do NOT call MarkOnline() — extension is offline.

	_, err := driver.FetchPage(context.Background(), "https://detail.1688.com/offer/123.html")
	if err == nil {
		t.Fatal("expected error when extension is offline, got nil")
	}
	if err.Error() != "extension is offline" {
		t.Fatalf("expected 'extension is offline', got: %q", err.Error())
	}
}

func TestPluginDriver_FetchPage_Timeout(t *testing.T) {
	logger := zap.NewNop()
	handler := &mockExtensionHandler{
		sendRequestFn: func(ctx context.Context, _ int64, _ *ExtensionRequest) (*ExtensionResponse, error) {
			// Block until context is cancelled (timeout).
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	driver := NewPluginDriver(handler, logger, WithFetchTimeout(5*time.Millisecond))
	driver.MarkOnline()

	_, err := driver.FetchPage(context.Background(), "https://detail.1688.com/offer/123.html")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestPluginDriver_FetchPage_ExtensionError(t *testing.T) {
	logger := zap.NewNop()
	handler := &mockExtensionHandler{
		sendRequestFn: func(_ context.Context, _ int64, req *ExtensionRequest) (*ExtensionResponse, error) {
			return &ExtensionResponse{
				Type:    ErrorFetchProduct,
				ID:      req.ID,
				Payload: errResultPayload("PAGE_NOT_FOUND", "offer does not exist"),
			}, nil
		},
	}

	driver := NewPluginDriver(handler, logger)
	driver.MarkOnline()

	_, err := driver.FetchPage(context.Background(), "https://detail.1688.com/offer/missing.html")
	if err == nil {
		t.Fatal("expected error from extension, got nil")
	}
	if err.Error() != "plugindriver: extension error [PAGE_NOT_FOUND]: offer does not exist" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

func TestPluginDriver_FetchPage_ExtensionStatusNotOK(t *testing.T) {
	logger := zap.NewNop()
	handler := &mockExtensionHandler{
		sendRequestFn: func(_ context.Context, _ int64, req *ExtensionRequest) (*ExtensionResponse, error) {
			return &ExtensionResponse{
				Type: ResultFetchProduct,
				ID:   req.ID,
				Payload: func() json.RawMessage {
					raw, _ := json.Marshal(FetchProductResult{Status: "error", Data: nil})
					return raw
				}(),
			}, nil
		},
	}

	driver := NewPluginDriver(handler, logger)
	driver.MarkOnline()

	_, err := driver.FetchPage(context.Background(), "https://detail.1688.com/offer/fail.html")
	if err == nil {
		t.Fatal("expected error when status is not 'ok', got nil")
	}
}

func TestPluginDriver_FetchPage_SendError(t *testing.T) {
	logger := zap.NewNop()
	handler := &mockExtensionHandler{
		sendRequestFn: func(_ context.Context, _ int64, _ *ExtensionRequest) (*ExtensionResponse, error) {
			return nil, errors.New("websocket connection lost")
		},
	}

	driver := NewPluginDriver(handler, logger)
	driver.MarkOnline()

	_, err := driver.FetchPage(context.Background(), "https://detail.1688.com/offer/123.html")
	if err == nil {
		t.Fatal("expected error from send failure, got nil")
	}
}

func TestPluginDriver_FetchPage_ResponseParsing(t *testing.T) {
	logger := zap.NewNop()

	// Test with full PageData including all optional fields.
	priceMin := 10.0
	priceMax := 20.0
	supplierScore := 85
	packageWeight := 0.5
	freight := 8.0

	fullData := toolbridge.PageData{
		Title:        "Premium Widget",
		Price:        15.0,
		PriceMin:     &priceMin,
		PriceMax:     &priceMax,
		Currency:     "CNY",
		MOQ:          10,
		Images:       []string{"https://img.example.com/1.jpg", "https://img.example.com/2.jpg"},
		SpecVariants: []toolbridge.SpecVariant{{Spec: "color:red", Price: 15.0, Stock: 100, ImageURL: "https://img.example.com/red.jpg"}},
		SupplierName: "Guangzhou Trading Co",
		SupplierID:   "cn123456",
		SupplierScore: &supplierScore,
		Description:  "High quality widget for export",
		Attributes:   map[string]string{"material": "stainless steel", "weight": "500g"},
		PackageWeight: &packageWeight,
		FreightCNY:   &freight,
	}

	handler := &mockExtensionHandler{
		sendRequestFn: func(_ context.Context, _ int64, req *ExtensionRequest) (*ExtensionResponse, error) {
			return &ExtensionResponse{
				Type:    ResultFetchProduct,
				ID:      req.ID,
				Payload: okResultPayload(fullData),
			}, nil
		},
	}

	driver := NewPluginDriver(handler, logger)
	driver.MarkOnline()

	result, err := driver.FetchPage(context.Background(), "https://detail.1688.com/offer/full.html")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify fields.
	if result.Title != "Premium Widget" {
		t.Fatalf("title: expected 'Premium Widget', got: %q", result.Title)
	}
	if result.Price != 15.0 {
		t.Fatalf("price: expected 15.0, got: %f", result.Price)
	}
	if result.PriceMin == nil || *result.PriceMin != 10.0 {
		t.Fatal("PriceMin: expected 10.0, got nil or different")
	}
	if result.PriceMax == nil || *result.PriceMax != 20.0 {
		t.Fatal("PriceMax: expected 20.0, got nil or different")
	}
	if result.MOQ != 10 {
		t.Fatalf("MOQ: expected 10, got: %d", result.MOQ)
	}
	if len(result.Images) != 2 {
		t.Fatalf("Images: expected 2, got: %d", len(result.Images))
	}
	if len(result.SpecVariants) != 1 {
		t.Fatalf("SpecVariants: expected 1, got: %d", len(result.SpecVariants))
	}
	if result.SpecVariants[0].Spec != "color:red" {
		t.Fatalf("variant spec: expected 'color:red', got: %q", result.SpecVariants[0].Spec)
	}
	if result.SupplierName != "Guangzhou Trading Co" {
		t.Fatalf("SupplierName: expected 'Guangzhou Trading Co', got: %q", result.SupplierName)
	}
	if result.SupplierScore == nil || *result.SupplierScore != 85 {
		t.Fatal("SupplierScore: expected 85, got nil or different")
	}
	if result.PackageWeight == nil || *result.PackageWeight != 0.5 {
		t.Fatal("PackageWeight: expected 0.5, got nil or different")
	}
	if len(result.Attributes) != 2 {
		t.Fatalf("Attributes: expected 2, got: %d", len(result.Attributes))
	}
	if result.Driver != "plugin" {
		t.Fatalf("Driver: expected 'plugin', got: %q", result.Driver)
	}
}

func TestPluginDriver_MarkOnline_MarkOffline(t *testing.T) {
	logger := zap.NewNop()
	handler := &mockExtensionHandler{
		sendRequestFn: func(_ context.Context, _ int64, _ *ExtensionRequest) (*ExtensionResponse, error) {
			return &ExtensionResponse{
				Type:    ResultFetchProduct,
				Payload: okResultPayload(toolbridge.PageData{Title: "Test"}),
			}, nil
		},
	}

	driver := NewPluginDriver(handler, logger)

	// Initially offline.
	status, _ := driver.Health()
	if status != "offline" {
		t.Fatalf("expected initial status 'offline', got: %q", status)
	}

	// Mark online.
	driver.MarkOnline()
	status, _ = driver.Health()
	if status != "online" {
		t.Fatalf("expected status 'online' after MarkOnline, got: %q", status)
	}

	// Should succeed.
	result, err := driver.FetchPage(context.Background(), "https://test.com")
	if err != nil {
		t.Fatalf("expected success when online, got: %v", err)
	}
	if result.Title != "Test" {
		t.Fatalf("expected title 'Test', got: %q", result.Title)
	}

	// Mark offline.
	driver.MarkOffline()
	status, _ = driver.Health()
	if status != "offline" {
		t.Fatalf("expected status 'offline' after MarkOffline, got: %q", status)
	}

	// Should fail.
	_, err = driver.FetchPage(context.Background(), "https://test.com")
	if err == nil {
		t.Fatal("expected error when offline, got nil")
	}
}

func TestPluginDriver_FetchPage_NilImagesNormalized(t *testing.T) {
	logger := zap.NewNop()

	// Simulate extension returning data with nil Images.
	raw := mustMarshal(toolbridge.PageData{
		Title: "No Images",
		Price: 5.0,
	})

	handler := &mockExtensionHandler{
		sendRequestFn: func(_ context.Context, _ int64, req *ExtensionRequest) (*ExtensionResponse, error) {
			return &ExtensionResponse{
				Type:    ResultFetchProduct,
				ID:      req.ID,
				Payload: okResultPayload(raw),
			}, nil
		},
	}

	driver := NewPluginDriver(handler, logger)
	driver.MarkOnline()

	result, err := driver.FetchPage(context.Background(), "https://detail.1688.com/offer/noimg.html")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Images == nil {
		t.Fatal("expected Images to be non-nil (empty slice), got nil")
	}
	if len(result.Images) != 0 {
		t.Fatalf("expected empty Images, got: %d", len(result.Images))
	}
}

func TestPluginDriver_DefaultCurrency(t *testing.T) {
	logger := zap.NewNop()
	handler := &mockExtensionHandler{
		sendRequestFn: func(_ context.Context, _ int64, req *ExtensionRequest) (*ExtensionResponse, error) {
			return &ExtensionResponse{
				Type:    ResultFetchProduct,
				ID:      req.ID,
				Payload: okResultPayload(toolbridge.PageData{Title: "No Currency", Price: 10.0, Currency: ""}),
			}, nil
		},
	}

	driver := NewPluginDriver(handler, logger)
	driver.MarkOnline()

	result, err := driver.FetchPage(context.Background(), "https://test.com")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Currency != "CNY" {
		t.Fatalf("expected default currency 'CNY', got: %q", result.Currency)
	}
}

func TestPluginDriver_HasTimeoutOption(t *testing.T) {
	logger := zap.NewNop()
	handler := &mockExtensionHandler{}

	driver := NewPluginDriver(handler, logger, WithFetchTimeout(10*time.Second))
	if driver.fetchTimeout != 10*time.Second {
		t.Fatalf("expected timeout 10s, got: %v", driver.fetchTimeout)
	}
}
