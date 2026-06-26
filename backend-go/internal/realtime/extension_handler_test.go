package realtime

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func TestExtensionHandler_Auth(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)
	go hub.Run()

	handler := NewExtensionHandler(hub, logger, "test-secret")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/extension", handler.ServeWS)

	srv := httptest.NewServer(router)
	defer srv.Close()

	wsURL := "ws://" + srv.Listener.Addr().String() + "/ws/extension"

	// Test missing token.
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected error for missing token")
	}

	// Test invalid token.
	_, _, err = websocket.DefaultDialer.Dial(wsURL+"?token=invalid", nil)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}

	// Test valid token.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(42),
		"exp":     float64(time.Now().Add(time.Hour).Unix()),
	})
	tokenStr, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal("sign token:", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?token="+tokenStr, nil)
	if err != nil {
		t.Fatal("expected success for valid token:", err)
	}
	defer conn.Close()

	if !handler.IsOnline(42) {
		t.Fatal("expected extension to be online after connect")
	}

	// Verify that a different user is not marked online.
	if handler.IsOnline(99) {
		t.Fatal("expected user 99 to be offline")
	}
}

func TestExtensionHandler_Auth_JWTSecretEmpty(t *testing.T) {
	// When jwtSecret is empty, skip JWT validation.
	logger := zap.NewNop()
	hub := NewHub(logger)
	go hub.Run()

	handler := NewExtensionHandler(hub, logger, "")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/extension", handler.ServeWS)

	srv := httptest.NewServer(router)
	defer srv.Close()

	wsURL := "ws://" + srv.Listener.Addr().String() + "/ws/extension"

	// With empty secret, any token should pass.
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?token=anything", nil)
	if err != nil {
		t.Fatal("expected success when jwtSecret is empty:", err)
	}
	defer conn.Close()
}

func TestExtensionHandler_PingPong(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)
	go hub.Run()

	handler := NewExtensionHandler(hub, logger, "test-secret")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/extension", handler.ServeWS)

	srv := httptest.NewServer(router)
	defer srv.Close()

	token := createTestToken(t, "test-secret", 42)
	conn := dialExtension(t, srv, token)
	defer conn.Close()

	// Send ping.
	if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
		t.Fatal("write ping:", err)
	}

	// Expect pong.
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatal("read response:", err)
	}
	var resp struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatal("unmarshal response:", err)
	}
	if resp.Type != "pong" {
		t.Fatalf("expected pong, got %s", resp.Type)
	}
}

func TestExtensionHandler_PingPong_StaysOnline(t *testing.T) {
	// Sending pings should prevent the client from being marked offline.
	logger := zap.NewNop()
	hub := NewHub(logger)
	go hub.Run()

	handler := NewExtensionHandler(hub, logger, "test-secret")
	handler.cleanupTick = 50 * time.Millisecond
	handler.pingTimeout = 150 * time.Millisecond

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/extension", handler.ServeWS)

	srv := httptest.NewServer(router)
	defer srv.Close()

	token := createTestToken(t, "test-secret", 42)
	conn := dialExtension(t, srv, token)
	defer conn.Close()

	// Send pings frequently to stay alive.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(30 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = conn.WriteJSON(map[string]string{"type": "ping"})
				// Read pong non-blocking.
				conn.SetReadDeadline(time.Now().Add(5 * time.Millisecond))
				_, _, _ = conn.ReadMessage()
				conn.SetReadDeadline(time.Time{})
			case <-stop:
				return
			}
		}
	}()

	// Wait for 3 cleanup cycles.
	time.Sleep(300 * time.Millisecond)

	if !handler.IsOnline(42) {
		t.Fatal("expected extension to stay online with pings")
	}
}

func TestExtensionHandler_OfflineDetection(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)
	go hub.Run()

	handler := NewExtensionHandler(hub, logger, "test-secret")
	handler.cleanupTick = 30 * time.Millisecond
	handler.pingTimeout = 60 * time.Millisecond

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/extension", handler.ServeWS)

	srv := httptest.NewServer(router)
	defer srv.Close()

	token := createTestToken(t, "test-secret", 42)
	conn := dialExtension(t, srv, token)
	// Do NOT defer conn.Close() — the hub will close it on unregister.
	// We close explicitly after the test.
	defer conn.Close()

	// Verify initially online.
	if !handler.IsOnline(42) {
		t.Fatal("expected extension online initially")
	}

	// Wait for cleanup to fire (timeout 60ms, tick 30ms → should trigger within ~120ms).
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !handler.IsOnline(42) {
			return // success — client was removed
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("expected extension offline after ping timeout")
}

func TestExtensionHandler_PendingRequest(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)
	go hub.Run()

	handler := NewExtensionHandler(hub, logger, "test-secret")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/extension", handler.ServeWS)

	srv := httptest.NewServer(router)
	defer srv.Close()

	token := createTestToken(t, "test-secret", 42)
	conn := dialExtension(t, srv, token)
	defer conn.Close()

	// Register a pending request manually.
	reqID := "test_req_123"
	resultCh := make(chan []byte, 1)
	handler.pendingMu.Lock()
	handler.pendingReqs[reqID] = resultCh
	handler.pendingMu.Unlock()

	// Send fetch_product_result from the extension side.
	resultPayload, _ := json.Marshal(map[string]interface{}{
		"type": "fetch_product_result",
		"id":   reqID,
		"payload": map[string]interface{}{
			"status": "ok",
			"data":   map[string]string{"title": "Test Product"},
		},
	})
	if err := conn.WriteMessage(websocket.TextMessage, resultPayload); err != nil {
		t.Fatal("write result:", err)
	}

	// Wait for response on the pending channel.
	select {
	case result := <-resultCh:
		var msg struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal(result, &msg); err != nil {
			t.Fatal("unmarshal result:", err)
		}
		if msg.Type != "fetch_product_result" {
			t.Fatalf("expected type fetch_product_result, got %s", msg.Type)
		}
		if msg.ID != reqID {
			t.Fatalf("expected id %s, got %s", reqID, msg.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for pending request result")
	}

	// Verify handler tracks pending count.
	if handler.PendingRequestCount() != 1 {
		t.Fatalf("expected 1 pending request (not yet consumed by SendRequest)")
	}
}

func TestExtensionHandler_PendingRequest_UnknownID(t *testing.T) {
	// Sending a fetch_product_result with an unknown ID should not panic.
	logger := zap.NewNop()
	hub := NewHub(logger)
	go hub.Run()

	handler := NewExtensionHandler(hub, logger, "test-secret")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/extension", handler.ServeWS)

	srv := httptest.NewServer(router)
	defer srv.Close()

	token := createTestToken(t, "test-secret", 42)
	conn := dialExtension(t, srv, token)
	defer conn.Close()

	resultPayload, _ := json.Marshal(map[string]interface{}{
		"type": "fetch_product_result",
		"id":   "nonexistent",
	})
	if err := conn.WriteMessage(websocket.TextMessage, resultPayload); err != nil {
		t.Fatal("write result:", err)
	}

	// No response expected — just verify no panic.
	time.Sleep(100 * time.Millisecond)
}

func TestExtensionHandler_OnlineCheck(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)
	go hub.Run()

	handler := NewExtensionHandler(hub, logger, "test-secret")

	// No clients connected.
	if handler.IsOnline(1) {
		t.Fatal("expected false when no clients")
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/extension", handler.ServeWS)

	srv := httptest.NewServer(router)
	defer srv.Close()

	// Connect user 42.
	token42 := createTestToken(t, "test-secret", 42)
	conn42 := dialExtension(t, srv, token42)
	defer conn42.Close()

	// Connect user 99.
	token99 := createTestToken(t, "test-secret", 99)
	conn99 := dialExtension(t, srv, token99)
	defer conn99.Close()

	if !handler.IsOnline(42) {
		t.Fatal("expected user 42 online")
	}
	if !handler.IsOnline(99) {
		t.Fatal("expected user 99 online")
	}
	if handler.IsOnline(999) {
		t.Fatal("expected user 999 offline")
	}
}

// --- test helpers ---

func createTestToken(t *testing.T, secret string, userID int64) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(userID),
		"exp":     float64(time.Now().Add(time.Hour).Unix()),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal("sign token:", err)
	}
	return tokenStr
}

func dialExtension(t *testing.T, srv *httptest.Server, token string) *websocket.Conn {
	t.Helper()
	wsURL := "ws://" + srv.Listener.Addr().String() + "/ws/extension?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal("dial extension:", err)
	}
	return conn
}
