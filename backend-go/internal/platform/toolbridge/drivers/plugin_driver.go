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
	pendingLists   map[string]chan listPageResponse
}

type listPageResponse struct {
	data *toolbridge.ListPageData
	err  error
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
		pendingLists:   make(map[string]chan listPageResponse),
	}
	// Attempt to register the response callback. If the service ignores
	// RegisterCallback (T1 not merged), this is a no-op and HandleResponse
	// can be called directly from the realtime layer.
	svc.RegisterCallback(0, d.HandleResponse)
	return d
}

// FetchListPage asks the browser extension to discover all visible product
// cards on a marketplace or supplier search page.
func (d *PluginDriver) FetchListPage(ctx context.Context, url string) (*toolbridge.ListPageData, error) {
	ownerUserID, ok := toolbridge.OwnerUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("plugin driver: authenticated owner user ID required")
	}
	reqID := fmt.Sprintf("list_%s", uuid.New().String())
	resultCh := make(chan listPageResponse, 1)

	d.mu.Lock()
	d.pendingLists[reqID] = resultCh
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.pendingLists, reqID)
		d.mu.Unlock()
	}()

	msg, err := json.Marshal(map[string]interface{}{
		"type":    "fetch_list_page",
		"id":      reqID,
		"payload": map[string]string{"url": url},
	})
	if err != nil {
		return nil, fmt.Errorf("plugin driver: marshal list request: %w", err)
	}
	if err := d.extensionSvc.SendToUser(ownerUserID, msg); err != nil {
		return nil, fmt.Errorf("plugin driver: send list request: %w", err)
	}

	select {
	case result := <-resultCh:
		return result.data, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(d.requestTimeout):
		return nil, errors.New("plugin driver: list request timeout")
	}
}

// FetchPage sends a fetch request to the user's extension and waits for the
// response. It generates a unique request ID, creates a pending channel, and
// sends the request via the ExtensionService. The response is received through
// the registered callback and routed to the pending channel.
func (d *PluginDriver) FetchPage(ctx context.Context, url string) (*toolbridge.PageData, error) {
	ownerUserID, ok := toolbridge.OwnerUserIDFromContext(ctx)
	if !ok {
		return nil, errors.New("plugin driver: authenticated owner user ID required")
	}
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

	if err := d.extensionSvc.SendToUser(ownerUserID, msg); err != nil {
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

// HasPending checks whether a request with the given ID still has a pending
// channel in the driver's map. The realtime layer calls this to decide whether
// to route a fetch_product_result through the pending-request handler (plugin)
// or the auto-collect handler (candidate creation).
func (d *PluginDriver) HasPending(requestID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.pending[requestID]
	return ok
}

// HasPendingList reports whether requestID belongs to an active list-page
// collection request.
func (d *PluginDriver) HasPendingList(requestID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.pendingLists[requestID]
	return ok
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

// Category returns ToolCategoryRead — this driver only fetches page data.
func (d *PluginDriver) Category() toolbridge.ToolCategory { return toolbridge.ToolCategoryRead }

// Execute is not supported for page-fetching drivers.
func (d *PluginDriver) Execute(_ map[string]interface{}) (*toolbridge.ToolResult, error) {
	return nil, errors.New("plugin driver: Execute not supported, use FetchPage instead")
}

// HandleResponse processes an extension response and routes it to the
// correct pending request channel. This is intended to be called by the
// realtime layer when a browser extension sends back a fetch result.
func (d *PluginDriver) HandleResponse(data []byte) {
	var resp struct {
		Type    string               `json:"type"`
		ID      string               `json:"id"`
		Data    *toolbridge.PageData `json:"data"`
		Payload *struct {
			Status string               `json:"status"`
			Data   *toolbridge.PageData `json:"data"`
		} `json:"payload"`
	}
	var listResp struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Payload *struct {
			Status string                   `json:"status"`
			Data   *toolbridge.ListPageData `json:"data"`
			Error  *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(data, &listResp); err == nil && listResp.Type == "list_page_result" && listResp.ID != "" {
		if listResp.Payload == nil {
			return
		}
		d.mu.RLock()
		ch, ok := d.pendingLists[listResp.ID]
		d.mu.RUnlock()
		if !ok {
			return
		}
		if listResp.Payload.Status != "ok" {
			code, message := "COLLECTION_FAILED", "browser extension could not collect list page"
			if listResp.Payload.Error != nil {
				if listResp.Payload.Error.Code != "" {
					code = listResp.Payload.Error.Code
				}
				if listResp.Payload.Error.Message != "" {
					message = listResp.Payload.Error.Message
				}
			}
			select {
			case ch <- listPageResponse{err: fmt.Errorf("%s: %s", code, message)}:
			default:
			}
			return
		}
		if listResp.Payload.Data == nil {
			return
		}
		if len(listResp.Payload.Data.RawData) == 0 {
			listResp.Payload.Data.RawData, _ = json.Marshal(listResp.Payload.Data)
		}
		listResp.Payload.Data.Driver = "plugin"
		select {
		case ch <- listPageResponse{data: listResp.Payload.Data}:
		default:
		}
		return
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return
	}
	if resp.Type != "fetch_product_result" || resp.ID == "" {
		return
	}
	result := resp.Data
	if result == nil && resp.Payload != nil && resp.Payload.Status == "ok" {
		result = resp.Payload.Data
	}
	if result == nil {
		return
	}
	result.Driver = "plugin"

	d.mu.RLock()
	ch, ok := d.pending[resp.ID]
	d.mu.RUnlock()
	if !ok {
		return
	}

	select {
	case ch <- result:
	default:
	}
}
