// Package drivers contains concrete ToolDriver implementations.
package drivers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Extension message protocol types (shared with the Chrome Extension)
// ---------------------------------------------------------------------------

const (
	// CommandFetchProduct is sent from backend to extension to request a page fetch.
	CommandFetchProduct = "fetch_product"

	// ResultFetchProduct is sent from extension to backend with the fetched data.
	ResultFetchProduct = "fetch_product_result"

	// ErrorFetchProduct is sent from extension to backend on failure.
	ErrorFetchProduct = "fetch_product_error"

	// DefaultFetchTimeout is how long PluginDriver waits for an extension response.
	DefaultFetchTimeout = 30 * time.Second

	// ExtensionPingInterval is the expected ping interval from the extension (15s).
	// The driver marks the extension offline after missing 3 consecutive pings.
	ExtensionPingInterval = 15 * time.Second
)

// ExtensionRequest is the message sent from the backend to the Chrome Extension.
type ExtensionRequest struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

// FetchProductPayload is the payload for a fetch_product command.
type FetchProductPayload struct {
	URL string `json:"url"`
}

// ExtensionResponse is the message received from the Chrome Extension.
type ExtensionResponse struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

// FetchProductResult is the payload of a successful fetch_product_result message.
type FetchProductResult struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// FetchProductError is the payload of a fetch_product_error message.
type FetchProductError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// Local interface (no dependency on realtime package)
// ---------------------------------------------------------------------------

// ExtensionHandler abstracts the WebSocket handler that communicates with the
// Chrome Extension. Implementations live in the realtime package; this local
// interface keeps PluginDriver decoupled from concrete imports.
type ExtensionHandler interface {
	// IsOnline reports whether the Chrome Extension for the given user has an
	// active WebSocket connection and has sent a recent ping.
	IsOnline(userID int64) bool

	// SendRequest sends a request message to the user's extension and waits for
	// the matching response. The response is returned or an error on timeout.
	//
	// Implementations should use a pending-request registry with a channel per
	// request ID and enforce the context deadline.
	SendRequest(ctx context.Context, userID int64, req *ExtensionRequest) (*ExtensionResponse, error)
}

// ---------------------------------------------------------------------------
// PluginDriver
// ---------------------------------------------------------------------------

// PluginDriver implements ToolDriver via a WebSocket bridge to a Chrome
// Extension running in the user's browser. It is the preferred driver because
// the extension operates under the user's real 1688 login session, avoiding
// anti-crawling countermeasures.
type PluginDriver struct {
	name             string
	extHandler       ExtensionHandler
	logger           *zap.Logger
	fetchTimeout     time.Duration
	lastPingTime     time.Time
	online           bool
	mu               sync.RWMutex
	userID           int64
}

// PluginDriverOption configures a PluginDriver.
type PluginDriverOption func(*PluginDriver)

// WithFetchTimeout sets the per-request timeout for extension fetch commands.
func WithFetchTimeout(timeout time.Duration) PluginDriverOption {
	return func(d *PluginDriver) {
		d.fetchTimeout = timeout
	}
}

// WithUserID sets the default user ID for extension communication.
func WithUserID(userID int64) PluginDriverOption {
	return func(d *PluginDriver) {
		d.userID = userID
	}
}

// NewPluginDriver creates a new PluginDriver.
//
// extHandler is the interface to the realtime ExtensionHandler. Use
// PluginDriverOption functions to customize timeouts and user identification.
func NewPluginDriver(extHandler ExtensionHandler, logger *zap.Logger, opts ...PluginDriverOption) *PluginDriver {
	d := &PluginDriver{
		name:         "plugin",
		extHandler:   extHandler,
		logger:       logger,
		fetchTimeout: DefaultFetchTimeout,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// FetchPage sends a fetch_product command to the Chrome Extension and returns
// the parsed PageData. It returns an error if the extension is offline, the
// request times out, or parsing fails.
func (d *PluginDriver) FetchPage(ctx context.Context, url string) (*toolbridge.PageData, error) {
	if !d.isOnline() {
		d.logger.Warn("plugindriver: extension offline, cannot fetch",
			zap.String("url", url))
		return nil, errors.New("extension is offline")
	}

	// Build the fetch request.
	payload, err := json.Marshal(FetchProductPayload{URL: url})
	if err != nil {
		return nil, fmt.Errorf("plugindriver: marshal payload: %w", err)
	}

	req := &ExtensionRequest{
		Type:    CommandFetchProduct,
		ID:      fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Payload: payload,
	}

	// Create a context with timeout for the extension round-trip.
	fetchCtx, cancel := context.WithTimeout(ctx, d.fetchTimeout)
	defer cancel()

	d.logger.Debug("plugindriver: sending fetch request to extension",
		zap.String("req_id", req.ID),
		zap.String("url", url))

	resp, err := d.extHandler.SendRequest(fetchCtx, d.userID, req)
	if err != nil {
		return nil, fmt.Errorf("plugindriver: send request: %w", err)
	}

	// Parse the response.
	switch resp.Type {
	case ResultFetchProduct:
		var result FetchProductResult
		if err := json.Unmarshal(resp.Payload, &result); err != nil {
			return nil, fmt.Errorf("plugindriver: parse result payload: %w", err)
		}
		if result.Status != "ok" {
			return nil, fmt.Errorf("plugindriver: extension returned status %q", result.Status)
		}
		if len(result.Data) == 0 {
			return nil, errors.New("plugindriver: extension returned empty data")
		}
		return parsePageData(result.Data, d.name, url)

	case ErrorFetchProduct:
		var errPayload FetchProductError
		if err := json.Unmarshal(resp.Payload, &errPayload); err != nil {
			return nil, fmt.Errorf("plugindriver: parse error payload: %w", err)
		}
		return nil, fmt.Errorf("plugindriver: extension error [%s]: %s",
			errPayload.Code, errPayload.Message)

	default:
		return nil, fmt.Errorf("plugindriver: unexpected response type: %s", resp.Type)
	}
}

// Health reports whether the extension is online based on the last ping.
func (d *PluginDriver) Health() (string, time.Duration) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.online {
		return "online", time.Since(d.lastPingTime)
	}
	return "offline", 0
}

// Name returns "plugin".
func (d *PluginDriver) Name() string {
	return d.name
}

// MarkOnline updates the driver's online status. Called by the extension
// handler when a ping is received.
func (d *PluginDriver) MarkOnline() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.online = true
	d.lastPingTime = time.Now()
}

// MarkOffline updates the driver's online status. Called when the extension
// disconnects or the ping interval expires.
func (d *PluginDriver) MarkOffline() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.online = false
}

// isOnline is a thread-safe check for the extension connection state.
func (d *PluginDriver) isOnline() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.online
}

// parsePageData converts raw JSON from the extension into a PageData struct.
func parsePageData(raw json.RawMessage, driverName, sourceURL string) (*toolbridge.PageData, error) {
	// First try to deserialize directly into a PageData (extension returns
	// structured JSON matching our schema).
	var pd toolbridge.PageData
	if err := json.Unmarshal(raw, &pd); err != nil {
		return nil, fmt.Errorf("plugindriver: unmarshal PageData: %w", err)
	}

	// Set meta fields that the driver controls.
	pd.SourceURL = sourceURL
	pd.CollectedAt = time.Now()
	pd.Driver = driverName

	// Default currency if not set.
	if pd.Currency == "" {
		pd.Currency = "CNY"
	}

	// Normalize image URLs (ensure array is non-nil).
	if pd.Images == nil {
		pd.Images = []string{}
	}

	return &pd, nil
}
