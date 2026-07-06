package realtime

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Event types for real-time action status push.
const (
	EventActionStatusChanged = "agent.action.status_changed"
)

// ActionStatusChangePayload is WS event payload for status changes.
type ActionStatusChangePayload struct {
	ActionID      int64  `json:"action_id"`
	ActionType    string `json:"action_type"`
	AgentID       string `json:"agent_id"`
	RiskLevel     string `json:"risk_level"`
	OldStatus     string `json:"old_status"`
	NewStatus     string `json:"new_status"`
	CorrelationID string `json:"correlation_id"`
	Timestamp     string `json:"timestamp"`
}

var (
	// ErrNoClient is returned when no client is connected for the given user ID.
	ErrNoClient = errors.New("no client found for user")
	// ErrBufferFull is returned when the client's send buffer is full.
	ErrBufferFull = errors.New("client send buffer full")
)

// Client represents a WebSocket client.
type Client struct {
	Hub     *Hub
	Conn    *websocket.Conn
	Send    chan []byte
	UserID  *int64
	aiChatFunc AIChatFunc
}

// Hub maintains a set of active clients and broadcasts messages.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewHub creates a new Hub.
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			count := len(h.clients)
			h.mu.Unlock()
			h.logger.Debug("client connected", zap.Int("total", count))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			count := len(h.clients)
			h.mu.Unlock()
			h.logger.Debug("client disconnected", zap.Int("total", count))

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// Broadcast sends a message to all connected clients. Non-blocking; if the
// broadcast channel is full the message is dropped (callers should use the
// synchronous BroadcastAndWait variant for critical events).
func (h *Hub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		h.logger.Warn("broadcast channel full, message dropped")
	}
}

// BroadcastAndWait sends a message synchronously by invoking the inner loop's
// delivery logic directly. Safe to call from any goroutine.
func (h *Hub) BroadcastAndWait(message []byte) {
	h.mu.Lock()
	for client := range h.clients {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(h.clients, client)
		}
	}
	h.mu.Unlock()
}

// BroadcastActionStatusChange sends a status change event to all WS clients.
func (h *Hub) BroadcastActionStatusChange(payload ActionStatusChangePayload) {
	msg, _ := json.Marshal(map[string]interface{}{
		"type":    EventActionStatusChanged,
		"payload": payload,
	})
	h.Broadcast(msg)
}

// ClientCount returns the current number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SendToUser sends a message to a specific user by userID.
// Returns ErrNoClient if no client is connected for that user.
// Returns ErrBufferFull if the client's send buffer is full.
func (h *Hub) SendToUser(userID int64, msg []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.UserID != nil && *client.UserID == userID {
			select {
			case client.Send <- msg:
				return nil
			default:
				return ErrBufferFull
			}
		}
	}
	return ErrNoClient
}
