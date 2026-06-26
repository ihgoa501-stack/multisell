package realtime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// ExtensionHandler manages WebSocket connections from Chrome Extensions.
// It shares the Hub with regular WS clients but maintains its own extension
// connection tracking, ping health monitoring, and pending request registry.
type ExtensionHandler struct {
	hub       *Hub
	logger    *zap.Logger
	jwtSecret string

	// Extension client tracking: client → last ping time.
	extensions map[*Client]time.Time
	extMu      sync.RWMutex

	// Pending request registry: req_uuid → result channel.
	pendingReqs map[string]chan []byte
	pendingMu   sync.RWMutex

	// Configurable timeouts for testability.
	pingTimeout time.Duration // default 45s
	cleanupTick time.Duration // default 15s

	closeCh chan struct{}
}

// NewExtensionHandler creates a new ExtensionHandler.
// jwtSecret is used to validate the token query param on upgrade.
func NewExtensionHandler(hub *Hub, logger *zap.Logger, jwtSecret string) *ExtensionHandler {
	h := &ExtensionHandler{
		hub:         hub,
		logger:      logger,
		jwtSecret:   jwtSecret,
		extensions:  make(map[*Client]time.Time),
		pendingReqs: make(map[string]chan []byte),
		pingTimeout: 45 * time.Second,
		cleanupTick: 15 * time.Second,
		closeCh:     make(chan struct{}),
	}
	go h.cleanupLoop()
	return h
}

// cleanupLoop periodically checks for extension clients that have not sent a
// ping within the timeout period and marks them offline.
func (h *ExtensionHandler) cleanupLoop() {
	ticker := time.NewTicker(h.cleanupTick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			h.cleanupOfflineClients()
		case <-h.closeCh:
			return
		}
	}
}

// cleanupOfflineClients removes extension clients whose last ping exceeds the
// pingTimeout threshold.
func (h *ExtensionHandler) cleanupOfflineClients() {
	h.extMu.Lock()
	defer h.extMu.Unlock()
	now := time.Now()
	for client, lastPing := range h.extensions {
		if now.Sub(lastPing) > h.pingTimeout {
			uid := int64(0)
			if client.UserID != nil {
				uid = *client.UserID
			}
			h.logger.Warn("extension client timed out",
				zap.Int64("user_id", uid),
				zap.Duration("idle", now.Sub(lastPing)))
			h.hub.unregister <- client
			delete(h.extensions, client)
		}
	}
}

// ServeWS upgrades an HTTP connection to WebSocket for Chrome Extensions.
// Auth is validated via JWT token in the URL query parameter: ?token={jwt}
func (h *ExtensionHandler) ServeWS(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		h.logger.Warn("extension ws rejected: missing token")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var userID *int64
	if h.jwtSecret != "" {
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(h.jwtSecret), nil
		})
		if err != nil || !token.Valid {
			h.logger.Warn("extension ws rejected: invalid token",
				zap.Error(err),
				zap.String("ip", c.ClientIP()),
			)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if uid, exists := claims["user_id"]; exists {
				switch v := uid.(type) {
				case float64:
					n := int64(v)
					userID = &n
				}
			}
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("extension ws upgrade failed", zap.Error(err))
		return
	}

	client := &Client{
		Hub:    h.hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}

	h.hub.register <- client

	h.extMu.Lock()
	h.extensions[client] = time.Now()
	h.extMu.Unlock()

	uidLog := int64(0)
	if userID != nil {
		uidLog = *userID
	}
	h.logger.Info("extension client connected",
		zap.Int64("user_id", uidLog),
		zap.Int("total_extensions", len(h.extensions)),
	)

	go client.writePump()
	go h.extensionReadPump(client)
}

// extensionReadPump reads messages from the extension WebSocket connection.
// Supported message types: ping, fetch_product_result, fetch_product_error.
func (h *ExtensionHandler) extensionReadPump(client *Client) {
	defer func() {
		h.extMu.Lock()
		delete(h.extensions, client)
		h.extMu.Unlock()
		h.hub.unregister <- client
		client.Conn.Close()
	}()

	for {
		_, msgBytes, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Type    string          `json:"type"`
			ID      string          `json:"id,omitempty"`
			Payload json.RawMessage `json:"payload,omitempty"`
		}
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			h.sendJSON(client, map[string]string{"type": "error", "data": "invalid JSON"})
			continue
		}

		switch msg.Type {
		case "ping":
			h.extMu.Lock()
			h.extensions[client] = time.Now()
			h.extMu.Unlock()
			h.sendJSON(client, map[string]string{"type": "pong"})

		case "fetch_product_result":
			if msg.ID != "" {
				h.pendingMu.RLock()
				ch, ok := h.pendingReqs[msg.ID]
				h.pendingMu.RUnlock()
				if ok {
					select {
					case ch <- msgBytes:
					default:
					}
				}
			}

		case "fetch_product_error":
			if msg.ID != "" {
				h.pendingMu.RLock()
				ch, ok := h.pendingReqs[msg.ID]
				h.pendingMu.RUnlock()
				if ok {
					select {
					case ch <- msgBytes:
					default:
					}
				}
			}

		default:
			h.sendJSON(client, map[string]string{
				"type": "error",
				"data": "unknown message type",
			})
		}
	}
}

// SendRequest sends a fetch_product request to the extension belonging to the
// given user and waits for a response. It blocks until the extension responds,
// the context is cancelled, or a 30-second timeout elapses.
func (h *ExtensionHandler) SendRequest(ctx context.Context, userID int64, url string) ([]byte, error) {
	// Find an extension client for this user.
	h.extMu.RLock()
	var target *Client
	for client := range h.extensions {
		if client.UserID != nil && *client.UserID == userID {
			target = client
			break
		}
	}
	h.extMu.RUnlock()

	if target == nil {
		return nil, fmt.Errorf("sourcing: no extension online for user %d", userID)
	}

	reqID := newRequestID()
	ch := make(chan []byte, 1)

	h.pendingMu.Lock()
	h.pendingReqs[reqID] = ch
	h.pendingMu.Unlock()

	defer func() {
		h.pendingMu.Lock()
		delete(h.pendingReqs, reqID)
		h.pendingMu.Unlock()
	}()

	// Send fetch_product request to the extension.
	msg := map[string]interface{}{
		"type": "fetch_product",
		"id":   reqID,
		"payload": map[string]string{
			"url": url,
		},
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("sourcing: marshal request failed: %w", err)
	}

	select {
	case target.Send <- payload:
	default:
		return nil, fmt.Errorf("sourcing: extension send buffer full for user %d", userID)
	}

	// Wait for response with timeout.
	select {
	case result := <-ch:
		return result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("sourcing: extension request %s timed out", reqID)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsOnline returns true if the given user has at least one extension client
// that has sent a ping within the timeout period.
func (h *ExtensionHandler) IsOnline(userID int64) bool {
	h.extMu.RLock()
	defer h.extMu.RUnlock()
	for client := range h.extensions {
		if client.UserID != nil && *client.UserID == userID {
			return true
		}
	}
	return false
}

// PendingRequestCount returns the number of pending fetch_product requests.
func (h *ExtensionHandler) PendingRequestCount() int {
	h.pendingMu.RLock()
	defer h.pendingMu.RUnlock()
	return len(h.pendingReqs)
}

// Close stops the cleanup goroutine and releases resources.
func (h *ExtensionHandler) Close() {
	select {
	case <-h.closeCh:
		// Already closed.
	default:
		close(h.closeCh)
	}
}

// sendJSON marshals v as JSON and sends it to the client's send channel.
func (h *ExtensionHandler) sendJSON(client *Client, v interface{}) {
	payload, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case client.Send <- payload:
	default:
		h.logger.Warn("extension client send buffer full, dropping message")
	}
}

// newRequestID generates a unique request ID for the pending registry.
func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("req_%x", b)
}
