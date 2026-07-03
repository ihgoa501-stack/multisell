package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// makeOriginCheck creates a CheckOrigin function from a CORS allowed-origins string.
// If origins is empty or "*", all origins are allowed (dev mode).
func makeOriginCheck(allowedOrigins string) func(r *http.Request) bool {
	origins := parseAllowedOrigins(allowedOrigins)
	if len(origins) == 0 {
		return func(r *http.Request) bool { return true }
	}
	if origins[0] == "*" {
		return func(r *http.Request) bool { return true }
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // non-browser client
		}
		for _, o := range origins {
			if o == origin {
				return true
			}
		}
		return false
	}
}

// parseAllowedOrigins splits a comma-separated origin string into a slice.
// Returns nil if input is empty or "*".
func parseAllowedOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// AIChatChunk is one streaming chunk from the AI.
type AIChatChunk struct {
	TraceID string `json:"trace_id"`
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// AIChatFunc processes an AI chat message via the orchestrator and streams
// response chunks back through the returned channel.
type AIChatFunc func(ctx context.Context, message string, userID *int64) (<-chan AIChatChunk, error)

// Handler handles WebSocket upgrade requests.
type Handler struct {
	hub           *Hub
	logger        *zap.Logger
	jwtSecret     string
	aiChatHandler AIChatFunc
	upgrader      websocket.Upgrader
}

// NewHandler creates a new WebSocket handler.
// jwtSecret is used to validate token query param on upgrade.
// allowedOrigins is the CORS allowed-origins config; "*" or empty allows all.
func NewHandler(hub *Hub, logger *zap.Logger, jwtSecret string, allowedOrigins string) *Handler {
	return &Handler{
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

// WithAIChat sets the AI chat handler for streaming responses over WebSocket.
func (h *Handler) WithAIChat(handler AIChatFunc) *Handler {
	h.aiChatHandler = handler
	return h
}

// ServeWS upgrades HTTP connections to WebSocket after validating JWT token.
func (h *Handler) ServeWS(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		// Also check Authorization header (Bearer token)
		auth := c.GetHeader("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if tokenStr == "" {
		h.logger.Warn("websocket upgrade rejected: missing token")
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
			h.logger.Warn("websocket upgrade rejected: invalid token",
				zap.Error(err),
				zap.String("ip", c.ClientIP()),
			)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Store user identity from token for downstream use.
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

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	client := &Client{
		Hub:        h.hub,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		UserID:     userID,
		aiChatFunc: h.aiChatHandler,
	}

	h.hub.register <- client
	h.logger.Info("websocket client connected",
		zap.String("ip", c.ClientIP()),
		zap.Int("total", h.hub.ClientCount()),
	)

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()
	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		var incoming struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(msgBytes, &incoming); err != nil {
			c.writeJSON(map[string]string{"type": "error", "data": "invalid JSON"})
			continue
		}
		switch incoming.Type {
		case "ai:chat":
			go c.handleAIChat(incoming.Message)
		default:
			c.writeJSON(map[string]string{"type": "pong"})
		}
	}
}

func (c *Client) handleAIChat(message string) {
	if c.aiChatFunc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	chunkChan, err := c.aiChatFunc(ctx, message, c.UserID)
	if err != nil {
		c.writeJSON(map[string]interface{}{
			"type": "error",
			"data": map[string]string{"message": err.Error()},
		})
		return
	}
	for chunk := range chunkChan {
		payload, _ := json.Marshal(map[string]interface{}{
			"type": "ai:stream",
			"data": chunk,
		})
		select {
		case c.Send <- payload:
		default:
			return
		}
	}
}

func (c *Client) writeJSON(v interface{}) {
	payload, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.Send <- payload:
	default:
		c.Hub.logger.Warn("client send buffer full, dropping message")
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}
