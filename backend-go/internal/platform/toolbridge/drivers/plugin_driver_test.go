package drivers

import (
	"context"
	"encoding/json"
	"errors"
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
	return m.sendErr
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

	ctx := context.Background()
	_, err := driver.FetchPage(ctx, "http://example.com/product")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// TestPluginDriverContextCancellation verifies that FetchPage respects
// context cancellation.
func TestPluginDriverContextCancellation(t *testing.T) {
	svc := newMockExtService()
	driver := NewPluginDriver(svc, 30*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
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

	ctx := context.Background()
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
	resultCh := make(chan *toolbridge.PageData, 1)
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
	case data := <-resultCh:
		if data == nil {
			t.Fatal("expected non-nil data on the pending channel")
		}
		if data.Title != "Routed Response" {
			t.Errorf("expected Title='Routed Response', got %q", data.Title)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for response to be routed")
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

	resultCh := make(chan *toolbridge.PageData, 1)
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
