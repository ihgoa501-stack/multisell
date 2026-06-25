package realtime

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins in dev
	},
}

// Handler handles WebSocket upgrade requests.
type Handler struct {
	hub       *Hub
	logger    *zap.Logger
	jwtSecret string
}

// NewHandler creates a new WebSocket handler.
// jwtSecret is used to validate token query param on upgrade.
func NewHandler(hub *Hub, logger *zap.Logger, jwtSecret string) *Handler {
	return &Handler{hub: hub, logger: logger, jwtSecret: jwtSecret}
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
			if userID, exists := claims["user_id"]; exists {
				c.Set("user_id", userID)
			}
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	client := &Client{
		Hub:  h.hub,
		Conn: conn,
		Send: make(chan []byte, 256),
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
		mt, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		if mt == websocket.TextMessage {
			c.Send <- []byte(`{"type":"pong"}`)
		}
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
