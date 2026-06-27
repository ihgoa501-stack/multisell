// Package drivers implements ToolDriver backends for the toolbridge.
package drivers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
)

// ExtensionService is the interface PluginDriver needs from the realtime layer.
// It defines how the driver sends messages to browser extensions and receives
// callbacks. This abstraction allows unit tests to mock the realtime layer
// without importing the realtime package.
type ExtensionService interface {
	// SendToUser sends a message to the browser extension for the given user.
	// userID=0 means broadcast to all connected extensions.
	SendToUser(userID int64, msg []byte) error
	// RegisterCallback registers a callback to be invoked when the extension
	// sends a response. Multiple callbacks for the same userID are appended.
	RegisterCallback(userID int64, callback func([]byte))
}

// PluginDriver communicates through the WebSocket ExtensionHandler.
// It sends fetch requests to the user's browser extension and waits for the
// response through a registered callback mechanism.
type PluginDriver struct {
	extensionSvc   ExtensionService
	requestTimeout time.Duration
	mu             sync.RWMutex
	pending        map[string]chan *toolbridge.PageData
}

// NewPluginDriver creates a PluginDriver.
//
// The constructor registers a response callback on the ExtensionService so
// that incoming extension responses are automatically routed to the correct
// pending request. If the ExtensionService implementation does not handle
// RegisterCallback (e.g., T1 is not merged), the caller can wire
// HandleResponse directly from the realtime layer in a later task (T5).
func NewPluginDriver(svc ExtensionService, timeout time.Duration) *PluginDriver {
	d := &PluginDriver{
		extensionSvc:   svc,
		requestTimeout: timeout,
		pending:        make(map[string]chan *toolbridge.PageData),
	}
	// Attempt to register the response callback. If the service ignores
	// RegisterCallback (T1 not merged), this is a no-op and HandleResponse
	// can be called directly from the realtime layer.
	svc.RegisterCallback(0, d.HandleResponse)
	return d
}

// FetchPage sends a fetch request to the user's extension and waits for the
// response. It generates a unique request ID, creates a pending channel, and
// sends the request via the ExtensionService. The response is received through
// the registered callback and routed to the pending channel.
func (d *PluginDriver) FetchPage(ctx context.Context, url string) (*toolbridge.PageData, error) {
	reqID := fmt.Sprintf("req_%s", uuid.New().String())
	resultCh := make(chan *toolbridge.PageData, 1)

	d.mu.Lock()
	d.pending[reqID] = resultCh
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.pending, reqID)
		d.mu.Unlock()
	}()

	msg, err := json.Marshal(map[string]interface{}{
		"type":    "fetch_product",
		"id":      reqID,
		"payload": map[string]string{"url": url},
	})
	if err != nil {
		return nil, fmt.Errorf("plugin driver: marshal request: %w", err)
	}

	// Send to the extension. In real usage, userID comes from context.
	if err := d.extensionSvc.SendToUser(0, msg); err != nil {
		return nil, fmt.Errorf("plugin driver: send: %w", err)
	}

	select {
	case result := <-resultCh:
		if result != nil {
			result.Driver = "plugin"
		}
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(d.requestTimeout):
		return nil, errors.New("plugin driver: request timeout")
	}
}

// Health checks if the extension service is available by sending a
// health-check probe.
func (d *PluginDriver) Health() (available bool, latency time.Duration, err error) {
	start := time.Now()
	msg, err := json.Marshal(map[string]string{
		"type": "health_check",
	})
	if err != nil {
		return false, 0, fmt.Errorf("plugin driver: health marshal: %w", err)
	}
	if err := d.extensionSvc.SendToUser(0, msg); err != nil {
		return false, time.Since(start), fmt.Errorf("plugin driver: health send: %w", err)
	}
	return true, time.Since(start), nil
}

// HandleResponse processes an extension response and routes it to the
// correct pending request channel. This is intended to be called by the
// realtime layer when a browser extension sends back a fetch result.
func (d *PluginDriver) HandleResponse(data []byte) {
	var resp struct {
		Type string                `json:"type"`
		ID   string                `json:"id"`
		Data *toolbridge.PageData  `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return
	}
	if resp.Type != "fetch_product_result" || resp.ID == "" {
		return
	}

	d.mu.RLock()
	ch, ok := d.pending[resp.ID]
	d.mu.RUnlock()
	if !ok {
		return
	}

	select {
	case ch <- resp.Data:
	default:
	}
}
