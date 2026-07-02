package realtime

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// PluginDriver defines callbacks for routing extension WebSocket results
// back to the sourcing agent plugin.
type PluginDriver interface {
	// OnFetchProductResult is called when a fetch_product_result message
	// is received from the extension client.
	OnFetchProductResult(userID int64, data json.RawMessage) error
}

// ExtensionHandler handles WebSocket connections for the extension.
// Auth is done via JWT token in the URL query parameter (?token=...),
// matching the main /ws handler pattern.
type ExtensionHandler struct {
	hub          *Hub
	logger       *zap.Logger
	jwtSecret    string
	pluginDriver PluginDriver
}

// NewExtensionHandler creates a new ExtensionHandler.
func NewExtensionHandler(hub *Hub, logger *zap.Logger, jwtSecret string) *ExtensionHandler {
	return &ExtensionHandler{
		hub:       hub,
		logger:    logger,
		jwtSecret: jwtSecret,
	}
}

// WithPluginDriver sets the plugin driver for routing results back.
func (h *ExtensionHandler) WithPluginDriver(driver PluginDriver) *ExtensionHandler {
	h.pluginDriver = driver
	return h
}

// ServeWS upgrades HTTP connections to WebSocket, validating JWT from URL
// query parameter (matching the /ws handler pattern).
func (h *ExtensionHandler) ServeWS(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		h.logger.Warn("extension ws upgrade rejected: missing token")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	userID, valid := h.validateExtensionToken(tokenStr)
	if !valid {
		h.logger.Warn("extension ws upgrade rejected: invalid token")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("extension websocket upgrade failed", zap.Error(err))
		return
	}

	client := &Client{
		Hub:    h.hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}

	h.hub.register <- client
	h.logger.Info("extension websocket client authenticated",
		zap.Int64("user_id", *userID),
		zap.Int("total", h.hub.ClientCount()),
	)

	go client.writePump()
	go h.extensionReadPump(client)
}

// extensionReadPump reads messages from the extension WebSocket connection.
// Only fetch_product_result, list_page_result, and ping messages are handled.
func (h *ExtensionHandler) extensionReadPump(client *Client) {
	defer func() {
		h.hub.unregister <- client
		client.Conn.Close()
	}()

	for {
		_, msgBytes, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}
		var incoming struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msgBytes, &incoming); err != nil {
			client.writeJSON(map[string]string{"type": "error", "data": "invalid JSON"})
			continue
		}
		switch incoming.Type {
		case "fetch_product_result":
			h.handleFetchProductResult(client, incoming.Data)
		case "ping":
			client.writeJSON(map[string]string{"type": "pong"})
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
