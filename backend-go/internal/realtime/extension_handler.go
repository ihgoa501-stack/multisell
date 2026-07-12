package realtime

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type extensionIncomingMessage struct {
	Type    string
	ID      string
	Payload json.RawMessage
}

// decodeExtensionMessage normalizes the browser extension protocol. Current
// clients send business data under "payload"; "data" remains accepted for
// backwards compatibility with early development clients.
func decodeExtensionMessage(raw []byte) (*extensionIncomingMessage, error) {
	var envelope struct {
		Type    string          `json:"type"`
		ID      string          `json:"id"`
		Payload json.RawMessage `json:"payload"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.Type == "" {
		return nil, fmt.Errorf("extension message type required")
	}
	payload := envelope.Payload
	if len(payload) == 0 || string(payload) == "null" {
		payload = envelope.Data
	}
	if envelope.Type != "ping" && (len(payload) == 0 || string(payload) == "null") {
		return nil, fmt.Errorf("extension message payload required")
	}
	return &extensionIncomingMessage{Type: envelope.Type, ID: envelope.ID, Payload: payload}, nil
}

// PluginDriver defines callbacks for routing extension WebSocket results
// back to the sourcing agent plugin.
type PluginDriver interface {
	// OnFetchProductResult is called when a fetch_product_result message
	// is received from the extension client.
	OnFetchProductResult(userID int64, data json.RawMessage) error
	// HasPending checks if there is a pending request with the given ID.
	HasPending(requestID string) bool
	OnListPageResult(userID int64, data json.RawMessage) error
	HasPendingList(requestID string) bool
}

// ExtensionHandler handles WebSocket connections for the A8 Sourcing Agent extension.
// Unlike the main WebSocket handler, auth is performed via the first message
// rather than a URL query parameter.
type ExtensionHandler struct {
	hub                *Hub
	logger             *zap.Logger
	jwtSecret          string
	pluginDriver       PluginDriver
	autoCollectHandler func(userID int64, payload json.RawMessage) error
	listCollectHandler func(userID int64, payload json.RawMessage) error
	upgrader           websocket.Upgrader
}

// NewExtensionHandler creates a new ExtensionHandler.
// allowedOrigins is the CORS allowed-origins config; "*" or empty allows all.
func NewExtensionHandler(hub *Hub, logger *zap.Logger, jwtSecret string, allowedOrigins string) *ExtensionHandler {
	return &ExtensionHandler{
		hub:       hub,
		logger:    logger,
		jwtSecret: jwtSecret,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     makeOriginCheck(allowedOrigins),
		},
	}
}

// WithPluginDriver sets the plugin driver for routing results back.
func (h *ExtensionHandler) WithPluginDriver(driver PluginDriver) *ExtensionHandler {
	h.pluginDriver = driver
	return h
}

// OnAutoCollect sets a handler for auto-collect (push-style) fetch_product_result
// messages that have no pending plugin request. When the extension pushes a result
// and there is no matching pending request, this handler is called instead.
func (h *ExtensionHandler) OnAutoCollect(hook func(userID int64, payload json.RawMessage) error) *ExtensionHandler {
	h.autoCollectHandler = hook
	return h
}

// OnListCollect sets a handler for list_page_result messages from the extension.
func (h *ExtensionHandler) OnListCollect(hook func(userID int64, payload json.RawMessage) error) *ExtensionHandler {
	h.listCollectHandler = hook
	return h
}

// ServeWS upgrades HTTP connections to WebSocket. Unlike the main WebSocket
// handler, auth is deferred to the first WebSocket message.
func (h *ExtensionHandler) ServeWS(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("extension websocket upgrade failed", zap.Error(err))
		return
	}

	client := &Client{
		Hub:  h.hub,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	h.hub.register <- client
	h.logger.Info("extension websocket client connected",
		zap.String("ip", c.ClientIP()),
		zap.Int("total", h.hub.ClientCount()),
	)

	go client.writePump()
	go h.extensionReadPump(client)
}

// extensionReadPump reads messages from the extension WebSocket connection.
// The first message must be an auth message with a valid JWT token.
// Only fetch_product_result and ping messages are handled.
func (h *ExtensionHandler) extensionReadPump(client *Client) {
	defer func() {
		h.hub.unregister <- client
		client.Conn.Close()
	}()

	// First message must be auth.
	_, msgBytes, err := client.Conn.ReadMessage()
	if err != nil {
		return
	}
	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(msgBytes, &authMsg); err != nil || authMsg.Type != "auth" || authMsg.Token == "" {
		h.logger.Warn("extension ws auth rejected: first message must be auth with token")
		client.writeJSON(map[string]string{"type": "error", "data": "first message must be auth with token"})
		return
	}

	userID, valid := h.validateExtensionToken(authMsg.Token)
	if !valid {
		h.logger.Warn("extension ws auth rejected: invalid token")
		client.writeJSON(map[string]string{"type": "error", "data": "invalid token"})
		return
	}
	client.UserID = userID
	client.writeJSON(map[string]string{"type": "auth", "data": "ok"})
	h.logger.Info("extension ws client authenticated", zap.Int64("user_id", *userID))

	// Process subsequent messages.
	for {
		_, msgBytes, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}
		incoming, err := decodeExtensionMessage(msgBytes)
		if err != nil {
			client.writeJSON(map[string]string{"type": "error", "data": "invalid JSON"})
			continue
		}
		switch incoming.Type {
		case "fetch_product_result", "fetch_product_error":
			if incoming.ID == "" {
				client.writeJSON(map[string]string{"type": "error", "data": "invalid fetch product response"})
				continue
			}
			if h.pluginDriver != nil && h.pluginDriver.HasPending(incoming.ID) {
				h.handleFetchProductResult(client, msgBytes)
			} else if incoming.Type == "fetch_product_result" && h.autoCollectHandler != nil && client.UserID != nil {
				if err := h.autoCollectHandler(*client.UserID, msgBytes); err != nil {
					h.logger.Error("auto-collect handler failed", zap.Int64("user_id", *client.UserID), zap.Error(err))
					client.writeJSON(map[string]string{"type": "error", "data": "auto-collect failed: " + err.Error()})
				}
			} else {
				h.logger.Warn("fetch product response dropped: no pending request or auto-collect handler", zap.String("request_id", incoming.ID), zap.String("message_type", incoming.Type))
			}
		case "ping":
			client.writeJSON(map[string]string{"type": "pong"})
		case "list_page_result":
			if h.pluginDriver != nil && incoming.ID != "" && h.pluginDriver.HasPendingList(incoming.ID) {
				if err := h.pluginDriver.OnListPageResult(*client.UserID, msgBytes); err != nil {
					h.logger.Error("extension ws: list plugin driver error", zap.Error(err))
					client.writeJSON(map[string]string{"type": "error", "data": "list plugin error: " + err.Error()})
				} else {
					client.writeJSON(map[string]string{"type": "list_page_result", "data": "ack"})
				}
			} else if h.listCollectHandler != nil && client.UserID != nil {
				if err := h.listCollectHandler(*client.UserID, incoming.Payload); err != nil {
					h.logger.Error("list-collect handler failed", zap.Int64("user_id", *client.UserID), zap.Error(err))
					client.writeJSON(map[string]string{"type": "error", "data": "list-collect failed: " + err.Error()})
				} else {
					client.writeJSON(map[string]string{"type": "list_page_result", "data": "ack"})
				}
			} else {
				h.logger.Warn("list_page_result dropped: no handler")
			}
		default:
			client.writeJSON(map[string]string{"type": "error", "data": "unknown message type: " + incoming.Type})
		}
	}
}

// validateExtensionToken parses and validates a JWT token, returning the user ID.
func (h *ExtensionHandler) validateExtensionToken(tokenStr string) (*int64, bool) {
	if h.jwtSecret == "" {
		return nil, false
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, false
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if uid, exists := claims["user_id"]; exists {
			switch v := uid.(type) {
			case float64:
				n := int64(v)
				return &n, true
			}
		}
	}
	return nil, false
}

// handleFetchProductResult routes a fetch_product_result to the plugin driver.
func (h *ExtensionHandler) handleFetchProductResult(client *Client, data json.RawMessage) {
	if h.pluginDriver == nil {
		h.logger.Warn("extension ws: no plugin driver registered for fetch_product_result")
		client.writeJSON(map[string]string{"type": "error", "data": "no plugin driver available"})
		return
	}
	if client.UserID == nil {
		return
	}
	if err := h.pluginDriver.OnFetchProductResult(*client.UserID, data); err != nil {
		h.logger.Error("extension ws: plugin driver error",
			zap.Int64("user_id", *client.UserID),
			zap.Error(err),
		)
		client.writeJSON(map[string]string{"type": "error", "data": "plugin error: " + err.Error()})
		return
	}
	client.writeJSON(map[string]string{"type": "fetch_product_result", "data": "ack"})
}
