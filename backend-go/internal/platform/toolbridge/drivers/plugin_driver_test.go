package drivers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
)

// mockExtService implements ExtensionService for testing.
type mockExtService struct {
	mu        sync.Mutex
	callbacks map[int64][]func([]byte)
	sendErr   error
	sentMsgs  [][]byte
	sentUsers []int64
}

func newMockExtService() *mockExtService {
	return &mockExtService{
		callbacks: make(map[int64][]func([]byte)),
	}
}

func (m *mockExtService) SendToUser(userID int64, msg []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMsgs = append(m.sentMsgs, msg)
	m.sentUsers = append(m.sentUsers, userID)
	return m.sendErr
}

func TestPluginDriverRoutesListRequestToOwnerFromContext(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 10*time.Millisecond)
	ctx := toolbridge.WithOwnerUserID(context.Background(), 42)

	_, _ = driver.FetchListPage(ctx, "https://www.ozon.ru/search/?text=storage")

	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.sentUsers) != 1 || svc.sentUsers[0] != 42 {
		t.Fatalf("sent users = %v, want [42]", svc.sentUsers)
	}
}

func TestPluginDriverRoutesDetailRequestToOwnerFromContext(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 10*time.Millisecond)
	ctx := toolbridge.WithOwnerUserID(context.Background(), 42)

	_, _ = driver.FetchPage(ctx, "https://www.ozon.ru/product/1")

	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.sentUsers) != 1 || svc.sentUsers[0] != 42 {
		t.Fatalf("sent users = %v, want [42]", svc.sentUsers)
	}
}

func (m *mockExtService) RegisterCallback(userID int64, callback func([]byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks[userID] = append(m.callbacks[userID], callback)
}

// invokeCallback simulates an extension response by calling all registered
// callbacks for the given userID.
func (m *mockExtService) invokeCallback(userID int64, data []byte) {
	m.mu.Lock()
	callbacks := make([]func([]byte), len(m.callbacks[userID]))
	copy(callbacks, m.callbacks[userID])
	m.mu.Unlock()
	for _, cb := range callbacks {
		cb(data)
	}
}

// TestPluginDriverConstructorRegistersCallback verifies that NewPluginDriver
// registers at least one callback for userID 0.
func TestPluginDriverConstructorRegistersCallback(t *testing.T) {
	svc := newMockExtService()
	_ = NewPluginDriver(svc, 10*time.Second)

	svc.mu.Lock()
	cbs := svc.callbacks[0]
	svc.mu.Unlock()

	if len(cbs) == 0 {
		t.Fatal("expected at least one callback registered for userID 0")
	}
}

// TestPluginDriverFetchPageSendsRequest verifies that FetchPage sends a
// properly-formed JSON message via SendToUser.
func TestPluginDriverFetchPageSendsRequest(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 10*time.Second)

	// Run FetchPage in a goroutine; it will block waiting for a response.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ctx = toolbridge.WithOwnerUserID(ctx, 42)

	go func() {
		// Wait for the message to be sent, then send back a dummy response quickly.
		for i := 0; i < 50; i++ {
			svc.mu.Lock()
			sentCount := len(svc.sentMsgs)
			svc.mu.Unlock()
			if sentCount > 0 {
				break
			}
			time.Sleep(2 * time.Millisecond)
		}

		svc.mu.Lock()
		if len(svc.sentMsgs) == 0 {
			svc.mu.Unlock()
			return
		}
		msg := svc.sentMsgs[len(svc.sentMsgs)-1]
		svc.mu.Unlock()

		var req struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal(msg, &req); err != nil || req.ID == "" {
			return
		}

		// Send a response via the registered callback.
		resp, _ := json.Marshal(map[string]interface{}{
			"type": "fetch_product_result",
			"id":   req.ID,
			"data": &toolbridge.PageData{
				SourceURL:    "http://example.com/product",
				Title:        "Extension Product",
				PriceCNY:     150.0,
				MOQ:          1,
				SupplierName: "Extension Supplier",
				CollectedAt:  time.Now(),
			},
		})
		svc.invokeCallback(0, resp)
	}()

	data, err := driver.FetchPage(ctx, "http://example.com/product")
	if err != nil {
		t.Fatalf("FetchPage returned error: %v", err)
	}
	if data == nil {
		t.Fatal("FetchPage returned nil data")
	}
	if data.Title != "Extension Product" {
		t.Errorf("expected Title='Extension Product', got %q", data.Title)
	}
	if data.Driver != "plugin" {
		t.Errorf("expected Driver='plugin', got %q", data.Driver)
	}
}

// TestPluginDriverTimeout verifies that FetchPage returns a timeout error
// when no response arrives within the request timeout.
func TestPluginDriverTimeout(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 10*time.Millisecond) // very short timeout

	ctx := toolbridge.WithOwnerUserID(context.Background(), 42)
	_, err := driver.FetchPage(ctx, "http://example.com/product")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestPluginDriverFetchPageReturnsActionableExtensionError(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, time.Second)
	go func() {
		for i := 0; i < 100; i++ {
			svc.mu.Lock()
			if len(svc.sentMsgs) > 0 {
				msg := append([]byte(nil), svc.sentMsgs[len(svc.sentMsgs)-1]...)
				svc.mu.Unlock()
				var req struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(msg, &req)
				svc.invokeCallback(0, []byte(`{"type":"fetch_product_error","id":"`+req.ID+`","payload":{"code":"TAB_NOT_FOUND","message":"Open the exact 1688 product page first"}}`))
				return
			}
			svc.mu.Unlock()
			time.Sleep(time.Millisecond)
		}
	}()

	ctx := toolbridge.WithOwnerUserID(context.Background(), 42)
	_, err := driver.FetchPage(ctx, "https://detail.1688.com/offer/1.html")
	if err == nil || !strings.Contains(err.Error(), "TAB_NOT_FOUND") {
		t.Fatalf("error = %v, want actionable TAB_NOT_FOUND", err)
	}
}

// TestPluginDriverContextCancellation verifies that FetchPage respects
// context cancellation.
func TestPluginDriverContextCancellation(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 30*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = toolbridge.WithOwnerUserID(ctx, 42)
	cancel() // cancel immediately

	_, err := driver.FetchPage(ctx, "http://example.com/product")
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestPluginDriverSendError verifies that FetchPage propagates SendToUser errors.
func TestPluginDriverSendError(t *testing.T) {
	svc := newMockExtService()
	svc.sendErr = errors.New("connection lost")
	driver := NewPluginDriver(svc, 10*time.Second)

	ctx := toolbridge.WithOwnerUserID(context.Background(), 42)
	_, err := driver.FetchPage(ctx, "http://example.com/product")
	if err == nil {
		t.Fatal("expected SendToUser error, got nil")
	}
}

// TestPluginDriverHandleResponseRouting verifies that HandleResponse routes
// the response to the correct pending request by request ID.
func TestPluginDriverHandleResponseRouting(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 10*time.Second)

	// Create a result channel and manually inject it into the pending map.
	resultCh := make(chan pageResponse, 1)
	driver.mu.Lock()
	driver.pending["test_req_1"] = resultCh
	driver.mu.Unlock()

	// Call HandleResponse with a matching fetch_product_result.
	resp, _ := json.Marshal(map[string]interface{}{
		"type": "fetch_product_result",
		"id":   "test_req_1",
		"data": &toolbridge.PageData{
			Title: "Routed Response",
		},
	})
	driver.HandleResponse(resp)

	select {
	case result := <-resultCh:
		if result.data == nil {
			t.Fatal("expected non-nil data on the pending channel")
		}
		if result.data.Title != "Routed Response" {
			t.Errorf("expected Title='Routed Response', got %q", result.data.Title)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for response to be routed")
	}
}

func TestPluginDriverHandleResponseAcceptsExtensionPayloadEnvelope(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 10*time.Second)

	resultCh := make(chan pageResponse, 1)
	driver.mu.Lock()
	driver.pending["protocol_req_1"] = resultCh
	driver.mu.Unlock()

	resp := []byte(`{"type":"fetch_product_result","id":"protocol_req_1","payload":{"status":"ok","data":{"source_url":"https://detail.1688.com/offer/1.html","title":"协议商品","price_1688":88,"min_order_qty":3,"supplier_id_1688":"supplier-1","package_weight_kg":0.5}}}`)
	driver.HandleResponse(resp)

	select {
	case result := <-resultCh:
		if result.data == nil {
			t.Fatal("expected non-nil protocol payload data")
		}
		if result.data.Title != "协议商品" || result.data.PriceCNY != 88 || result.data.MOQ != 3 || result.data.SupplierBusinessID != "supplier-1" || result.data.WeightKg == nil || *result.data.WeightKg != 0.5 {
			t.Fatalf("unexpected data: %+v", result.data)
		}
		wantRaw := `{"source_url":"https://detail.1688.com/offer/1.html","title":"协议商品","price_1688":88,"min_order_qty":3,"supplier_id_1688":"supplier-1","package_weight_kg":0.5}`
		if string(result.data.RawResponse) != wantRaw {
			t.Fatalf("raw response changed: %s", result.data.RawResponse)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for protocol payload response")
	}
}

func TestPluginDriverFetchListPageReturnsDiscoveredItems(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 10*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = toolbridge.WithOwnerUserID(ctx, 42)

	go func() {
		for i := 0; i < 100; i++ {
			svc.mu.Lock()
			if len(svc.sentMsgs) > 0 {
				msg := append([]byte(nil), svc.sentMsgs[len(svc.sentMsgs)-1]...)
				svc.mu.Unlock()
				var req struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				}
				if json.Unmarshal(msg, &req) != nil || req.Type != "fetch_list_page" || req.ID == "" {
					return
				}
				resp := []byte(`{"type":"list_page_result","id":"` + req.ID + `","payload":{"status":"ok","data":{"page_url":"https://www.ozon.ru/search/?text=storage","collected_at":"2026-07-11T00:00:00Z","items":[{"title":"收纳架","price_range":"1299","detail_url":"https://www.ozon.ru/product/1","image_url":"https://cdn.example/1.jpg","raw_text":"收纳架 1 299 ₽","raw_html":"<div data-index=\"1\">收纳架 1 299 ₽</div>"}]}}}`)
				svc.invokeCallback(0, resp)
				return
			}
			svc.mu.Unlock()
			time.Sleep(2 * time.Millisecond)
		}
	}()

	data, err := driver.FetchListPage(ctx, "https://www.ozon.ru/search/?text=storage")
	if err != nil {
		t.Fatalf("FetchListPage() error = %v", err)
	}
	if data.Driver != "plugin" || len(data.Items) != 1 {
		t.Fatalf("unexpected list data: %+v", data)
	}
	if len(data.RawData) == 0 {
		t.Fatal("raw list evidence must be preserved")
	}
	if data.Items[0].Title != "收纳架" || data.Items[0].DetailURL == "" {
		t.Fatalf("unexpected item: %+v", data.Items[0])
	}
	if data.Items[0].RawText == "" || data.Items[0].RawHTML == "" {
		t.Fatalf("raw product-card evidence missing: %+v", data.Items[0])
	}
}

func TestPluginDriverReportsPendingListRequest(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 20*time.Millisecond)
	driver.mu.Lock()
	driver.pendingLists["list-1"] = make(chan listPageResponse, 1)
	driver.mu.Unlock()

	if !driver.HasPendingList("list-1") {
		t.Fatal("expected list-1 to be reported as pending")
	}
	if driver.HasPendingList("missing") {
		t.Fatal("missing request must not be reported as pending")
	}
}

func TestPluginDriverFetchListPageReturnsActionableExtensionError(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, time.Second)

	go func() {
		for i := 0; i < 100; i++ {
			svc.mu.Lock()
			if len(svc.sentMsgs) > 0 {
				msg := append([]byte(nil), svc.sentMsgs[len(svc.sentMsgs)-1]...)
				svc.mu.Unlock()
				var req struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(msg, &req)
				resp := []byte(`{"type":"list_page_result","id":"` + req.ID + `","payload":{"status":"error","error":{"code":"CAPTCHA_REQUIRED","message":"Ozon requested verification"},"data":{"page_url":"https://www.ozon.ru/search/","items":[]}}}`)
				svc.invokeCallback(0, resp)
				return
			}
			svc.mu.Unlock()
			time.Sleep(time.Millisecond)
		}
	}()

	ctx := toolbridge.WithOwnerUserID(context.Background(), 42)
	_, err := driver.FetchListPage(ctx, "https://www.ozon.ru/search/")
	if err == nil || !strings.Contains(err.Error(), "CAPTCHA_REQUIRED") {
		t.Fatalf("error = %v, want actionable CAPTCHA_REQUIRED", err)
	}
}

// TestPluginDriverHealth verifies that Health sends a health_check message
// and returns availability based on SendToUser success.
func TestPluginDriverHealth(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 10*time.Second)

	avail, latency, err := driver.Health()
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if !avail {
		t.Error("expected available=true when SendToUser succeeds")
	}
	if latency <= 0 {
		t.Error("expected positive latency")
	}
}

// TestPluginDriverHealthFailure verifies that Health returns unavailable
// when SendToUser fails.
func TestPluginDriverHealthFailure(t *testing.T) {
	svc := newMockExtService()
	svc.sendErr = errors.New("service unavailable")
	driver := NewPluginDriver(svc, 10*time.Second)

	avail, _, err := driver.Health()
	if err == nil {
		t.Fatal("expected Health to return error when SendToUser fails")
	}
	if avail {
		t.Error("expected available=false when SendToUser fails")
	}
}

// TestPluginDriverHandleResponseIgnoresNonMatchingType verifies that
// HandleResponse ignores messages that are not fetch_product_result.
func TestPluginDriverHandleResponseIgnoresNonMatchingType(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 10*time.Second)

	resultCh := make(chan pageResponse, 1)
	driver.mu.Lock()
	driver.pending["test_ignore"] = resultCh
	driver.mu.Unlock()

	// Send a non-matching type and confirm the channel is untouched.
	resp, _ := json.Marshal(map[string]interface{}{
		"type": "other_type",
		"id":   "test_ignore",
	})
	driver.HandleResponse(resp)

	// Channel should still be buffered with no message.
	select {
	case <-resultCh:
		t.Fatal("expected no data on channel for non-matching type")
	default:
		// OK - no message delivered.
	}
}
