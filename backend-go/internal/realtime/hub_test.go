package realtime

import (
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestHub creates a Hub with a test logger and starts its run loop.
func newTestHub(t *testing.T) *Hub {
	t.Helper()
	logger := dbtest.NewLogger(t)
	hub := NewHub(logger)
	go hub.Run()
	return hub
}

// newTestClient creates a Client without a real WebSocket connection.
// The Hub never accesses client.Conn directly, so a nil connection is safe
// for unit tests of Hub operations.
func newTestClient(hub *Hub) *Client {
	return &Client{
		Hub:  hub,
		Conn: nil,
		Send: make(chan []byte, 256),
	}
}

// mustReceive reads from a client's Send channel with a short timeout,
// failing the test if no message arrives.
func mustReceive(t *testing.T, client *Client) []byte {
	t.Helper()
	select {
	case msg := <-client.Send:
		return msg
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message on client.Send")
		return nil
	}
}

// ---------------------------------------------------------------------------
// Hub creation basics
// ---------------------------------------------------------------------------

func TestNewHub_InitialState(t *testing.T) {
	hub := newTestHub(t)

	if got := hub.ClientCount(); got != 0 {
		t.Errorf("expected ClientCount=0 for new hub, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// Client registration and unregistration
// ---------------------------------------------------------------------------

func TestRegisterClient(t *testing.T) {
	hub := newTestHub(t)

	client := newTestClient(hub)
	hub.register <- client

	// Allow Hub.Run goroutine to process the registration.
	time.Sleep(10 * time.Millisecond)

	if got := hub.ClientCount(); got != 1 {
		t.Errorf("expected ClientCount=1 after register, got %d", got)
	}
}

func TestRegisterAndUnregisterClient(t *testing.T) {
	hub := newTestHub(t)

	client := newTestClient(hub)
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	if got := hub.ClientCount(); got != 1 {
		t.Fatalf("expected 1 client, got %d", got)
	}

	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	if got := hub.ClientCount(); got != 0 {
		t.Errorf("expected ClientCount=0 after unregister, got %d", got)
	}

	// Verify client.Send channel is closed after unregister.
	select {
	case _, ok := <-client.Send:
		if ok {
			t.Error("expected client.Send to be closed after unregister")
		}
	default:
		t.Error("client.Send is still open (receive blocked)")
	}
}

func TestClientCountTracking(t *testing.T) {
	hub := newTestHub(t)

	// Register 5 clients one by one, checking count each time.
	clients := make([]*Client, 5)
	for i := range clients {
		clients[i] = newTestClient(hub)
		hub.register <- clients[i]
	}
	time.Sleep(10 * time.Millisecond)

	if got := hub.ClientCount(); got != 5 {
		t.Fatalf("expected ClientCount=5 after 5 registers, got %d", got)
	}

	// Unregister 2 clients.
	hub.unregister <- clients[0]
	hub.unregister <- clients[2]
	time.Sleep(10 * time.Millisecond)

	if got := hub.ClientCount(); got != 3 {
		t.Errorf("expected ClientCount=3 after 2 unregisters, got %d", got)
	}

	// Register another client.
	newClient := newTestClient(hub)
	hub.register <- newClient
	time.Sleep(10 * time.Millisecond)

	if got := hub.ClientCount(); got != 4 {
		t.Errorf("expected ClientCount=4 after re-register, got %d", got)
	}
}

func TestUnregisterNonexistentClient(t *testing.T) {
	hub := newTestHub(t)

	// Unregister a client that was never registered should not panic.
	client := newTestClient(hub)
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	// Count should remain 0.
	if got := hub.ClientCount(); got != 0 {
		t.Errorf("expected ClientCount=0, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// Broadcast
// ---------------------------------------------------------------------------

func TestBroadcast_SendsToAllClients(t *testing.T) {
	hub := newTestHub(t)

	const numClients = 3
	clients := make([]*Client, numClients)
	for i := range clients {
		clients[i] = newTestClient(hub)
		hub.register <- clients[i]
	}
	time.Sleep(10 * time.Millisecond)

	msg := []byte("hello from broadcast")
	hub.Broadcast(msg)

	for i := range clients {
		received := mustReceive(t, clients[i])
		if string(received) != string(msg) {
			t.Errorf("client[%d] received wrong message: got %q, want %q", i, string(received), string(msg))
		}
	}
}

func TestBroadcastAndWait_DeliversSynchronously(t *testing.T) {
	hub := newTestHub(t)

	clients := make([]*Client, 3)
	for i := range clients {
		clients[i] = newTestClient(hub)
		hub.register <- clients[i]
	}
	time.Sleep(10 * time.Millisecond)

	msg := []byte("sync broadcast")
	hub.BroadcastAndWait(msg)

	for i := range clients {
		received := mustReceive(t, clients[i])
		if string(received) != string(msg) {
			t.Errorf("client[%d] received wrong message: got %q, want %q", i, string(received), string(msg))
		}
	}
}

func TestBroadcast_WithNoClients(t *testing.T) {
	hub := newTestHub(t)

	// Should not panic or block.
	hub.Broadcast([]byte("nobody home"))
	hub.BroadcastAndWait([]byte("still nobody"))
}

func TestBroadcast_ClientRemovedOnFullBuffer(t *testing.T) {
	hub := newTestHub(t)

	// Create a client with a tiny send buffer (capacity 1).
	client := newTestClient(hub)
	client.Send = make(chan []byte, 1)
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Fill the buffer so the next broadcast finds it full.
	client.Send <- []byte("fill")

	// BroadcastAndWait should detect the full buffer, close Send, and remove client.
	hub.BroadcastAndWait([]byte("overflow"))

	// Client must have been removed from the hub.
	if got := hub.ClientCount(); got != 0 {
		t.Errorf("expected 0 clients after removing full-buffer client, got %d", got)
	}

	// Send channel must be closed but still contains the buffered "fill".
	// The first receive drains that remaining value.
	<-client.Send
	// The second receive proves the channel is closed.
	if _, ok := <-client.Send; ok {
		t.Error("expected Send channel to be closed after buffer overflow")
	}
}

// ---------------------------------------------------------------------------
// SendToUser
// ---------------------------------------------------------------------------

func TestSendToUser_DeliversToMatchingClient(t *testing.T) {
	hub := newTestHub(t)

	userID1 := int64(100)
	userID2 := int64(200)

	client1 := newTestClient(hub)
	client1.UserID = &userID1
	hub.register <- client1

	client2 := newTestClient(hub)
	client2.UserID = &userID2
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	msg := []byte("private message for user 100")
	err := hub.SendToUser(userID1, msg)
	if err != nil {
		t.Fatalf("SendToUser returned error: %v", err)
	}

	// Verify client1 received the message.
	received := mustReceive(t, client1)
	if string(received) != string(msg) {
		t.Errorf("client1 received wrong message: got %q, want %q", string(received), string(msg))
	}

	// Verify client2 did NOT receive it.
	select {
	case msg2 := <-client2.Send:
		t.Errorf("client2 should not have received a message, got %q", string(msg2))
	default:
		// Expected.
	}
}

func TestSendToUser_ReturnsErrNoClient(t *testing.T) {
	hub := newTestHub(t)

	err := hub.SendToUser(999, []byte("nobody"))
	if err != ErrNoClient {
		t.Errorf("expected ErrNoClient, got %v", err)
	}
}

func TestSendToUser_ReturnsErrBufferFull(t *testing.T) {
	hub := newTestHub(t)

	userID := int64(42)
	client := newTestClient(hub)
	client.UserID = &userID
	client.Send = make(chan []byte, 1) // tiny buffer
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Fill the buffer.
	client.Send <- []byte("fill")

	err := hub.SendToUser(userID, []byte("overflow"))
	if err != ErrBufferFull {
		t.Errorf("expected ErrBufferFull, got %v", err)
	}
}

func TestSendToUser_SkipsClientsWithNilUserID(t *testing.T) {
	hub := newTestHub(t)

	// Client with nil UserID — should be skipped by SendToUser.
	client := newTestClient(hub)
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	err := hub.SendToUser(1, []byte("msg"))
	if err != ErrNoClient {
		t.Errorf("expected ErrNoClient when only nil-UserID clients exist, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Concurrent safety
// ---------------------------------------------------------------------------

func TestConcurrentRegisterAndUnregister(t *testing.T) {
	hub := newTestHub(t)

	n := 50
	clients := make([]*Client, n)
	var wg sync.WaitGroup

	// Concurrently register all clients.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			client := newTestClient(hub)
			clients[i] = client
			hub.register <- client
		}(i)
	}
	wg.Wait()

	time.Sleep(50 * time.Millisecond)

	if got := hub.ClientCount(); got != n {
		t.Fatalf("expected ClientCount=%d after concurrent register, got %d", n, got)
	}

	// Concurrently unregister half of them.
	for i := 0; i < n/2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			hub.unregister <- clients[i]
		}(i)
	}
	wg.Wait()

	time.Sleep(50 * time.Millisecond)

	if got := hub.ClientCount(); got != n/2 {
		t.Errorf("expected ClientCount=%d after concurrent unregister, got %d", n/2, got)
	}
}

func TestConcurrentBroadcastAndClientCount(t *testing.T) {
	hub := newTestHub(t)

	clients := make([]*Client, 10)
	for i := range clients {
		clients[i] = newTestClient(hub)
		hub.register <- clients[i]
	}
	time.Sleep(10 * time.Millisecond)

	var wg sync.WaitGroup

	// Concurrent broadcasts and ClientCount reads — must not race.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.Broadcast([]byte("concurrent"))
			hub.ClientCount()
		}()
	}
	wg.Wait()
}

func TestConcurrentSendToUser(t *testing.T) {
	hub := newTestHub(t)

	n := 20
	clients := make([]*Client, n)
	for i := 0; i < n; i++ {
		uid := int64(i + 1)
		clients[i] = newTestClient(hub)
		clients[i].UserID = &uid
		hub.register <- clients[i]
	}
	time.Sleep(10 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()
			err := hub.SendToUser(uid, []byte("ping"))
			if err != nil && err != ErrBufferFull {
				t.Errorf("SendToUser(%d) unexpected error: %v", uid, err)
			}
		}(int64(i + 1))
	}
	wg.Wait()
}
