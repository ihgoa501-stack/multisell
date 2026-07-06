package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/realtime"
)

// ── NewStreamer ─────────────────────────────────────────────────────────────

func TestNewStreamer(t *testing.T) {
	s := NewStreamer(nil, nil)
	if s == nil {
		t.Fatal("NewStreamer returned nil")
	}
}

// ── chunkText ───────────────────────────────────────────────────────────────

func TestChunkText_Short(t *testing.T) {
	chunks := chunkText("hello", 10)
	if len(chunks) != 1 || chunks[0] != "hello" {
		t.Fatalf("chunkText(hello, 10) = %v, want [hello]", chunks)
	}
}

func TestChunkText_Long(t *testing.T) {
	chunks := chunkText("abcdefghij", 3)
	want := []string{"abc", "def", "ghi", "j"}
	if len(chunks) != len(want) {
		t.Fatalf("len = %d, want %d: %v", len(chunks), len(want), chunks)
	}
	for i := range want {
		if chunks[i] != want[i] {
			t.Fatalf("chunks[%d] = %q, want %q", i, chunks[i], want[i])
		}
	}
}

func TestChunkText_Exact(t *testing.T) {
	chunks := chunkText("hello", 5)
	if len(chunks) != 1 || chunks[0] != "hello" {
		t.Fatalf("chunkText(hello, 5) = %v, want [hello]", chunks)
	}
}

func TestChunkText_Empty(t *testing.T) {
	chunks := chunkText("", 5)
	if len(chunks) != 1 || chunks[0] != "" {
		t.Fatalf("chunkText(\"\", 5) = %v, want [\"\"]", chunks)
	}
}

func TestChunkText_SingleCharacter(t *testing.T) {
	chunks := chunkText("a", 1)
	if len(chunks) != 1 || chunks[0] != "a" {
		t.Fatalf("chunkText(a, 1) = %v, want [a]", chunks)
	}
}

// ── SSEEvent JSON ───────────────────────────────────────────────────────────

func TestSSEEvent_JSON(t *testing.T) {
	ts := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	ev := &SSEEvent{
		Event:     "token",
		TraceID:   "trc_123",
		AgentID:   "A6",
		Seq:       1,
		Data:      map[string]string{"text": "hello"},
		Timestamp: ts,
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["event"] != "token" {
		t.Errorf("event = %v, want token", got["event"])
	}
	if got["trace_id"] != "trc_123" {
		t.Errorf("trace_id = %v, want trc_123", got["trace_id"])
	}
	if got["agent_id"] != "A6" {
		t.Errorf("agent_id = %v, want A6", got["agent_id"])
	}
	if got["seq"] != float64(1) {
		t.Errorf("seq = %v, want 1", got["seq"])
	}
	if _, ok := got["data"]; !ok {
		t.Error("data field missing")
	}
	if _, ok := got["ts"]; !ok {
		t.Error("ts field missing")
	}
}

// ── Streamer methods ────────────────────────────────────────────────────────

func TestStreamer_StartSSE(t *testing.T) {
	hub := realtime.NewHub(testLogger())
	s := NewStreamer(hub, testLogger())
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	s.StartSSE(c)

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", w.Header().Get("Cache-Control"))
	}
	if w.Header().Get("Connection") != "keep-alive" {
		t.Errorf("Connection = %q, want keep-alive", w.Header().Get("Connection"))
	}
	if w.Header().Get("X-Accel-Buffering") != "no" {
		t.Errorf("X-Accel-Buffering = %q, want no", w.Header().Get("X-Accel-Buffering"))
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestStreamer_WriteSSE(t *testing.T) {
	hub := realtime.NewHub(testLogger())
	s := NewStreamer(hub, testLogger())
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ev := &SSEEvent{
		Event: "test_event",
		Data:  map[string]string{"key": "val"},
	}
	if err := s.WriteSSE(c, ev); err != nil {
		t.Fatalf("WriteSSE: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: test_event") {
		t.Errorf("body missing event line: %s", body)
	}
	if !strings.Contains(body, `"key":"val"`) || !strings.Contains(body, `"event":"test_event"`) {
		t.Errorf("body missing JSON data: %s", body)
	}
}

func TestStreamer_BroadcastAIMessage_NilHub(t *testing.T) {
	s := NewStreamer(nil, testLogger())
	// Should not panic with nil hub
	s.BroadcastAIMessage("trc_1", "hello", false)
	s.BroadcastAIMessage("trc_1", "", true)
}

func TestStreamer_BroadcastEvent_NilHub(t *testing.T) {
	s := NewStreamer(nil, testLogger())
	// Should not panic with nil hub
	s.BroadcastEvent(&SSEEvent{Event: "test"})
}

func TestStreamer_BroadcastAIMessage_WithHub(t *testing.T) {
	hub := realtime.NewHub(testLogger())
	s := NewStreamer(hub, testLogger())
	// Should not panic with real hub (no clients connected)
	s.BroadcastAIMessage("trc_1", "hello", false)
	s.BroadcastAIMessage("trc_1", "", true)
}

func TestStreamer_BroadcastEvent_WithHub(t *testing.T) {
	hub := realtime.NewHub(testLogger())
	s := NewStreamer(hub, testLogger())
	// Should not panic with real hub
	s.BroadcastEvent(&SSEEvent{Event: "test"})
}
