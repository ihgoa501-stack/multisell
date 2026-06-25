package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/realtime"
	"go.uber.org/zap"
)

// Streamer handles SSE streaming and WebSocket broadcast of AI events.
type Streamer struct {
	hub    *realtime.Hub
	logger *zap.Logger
}

// NewStreamer creates a new Streamer.
func NewStreamer(hub *realtime.Hub, logger *zap.Logger) *Streamer {
	return &Streamer{hub: hub, logger: logger}
}

// SSEEvent is the wire format for Server-Sent Events.
type SSEEvent struct {
	Event     string      `json:"event"`
	TraceID   string      `json:"trace_id,omitempty"`
	AgentID   string      `json:"agent_id,omitempty"`
	Seq       int         `json:"seq,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"ts"`
}

// WriteSSE writes one SSE event to the response stream.
func (s *Streamer) WriteSSE(c *gin.Context, ev *SSEEvent) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Event, payload)
	if err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

// StartSSE prepares the response for Server-Sent Events streaming.
func (s *Streamer) StartSSE(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()
}

// BroadcastAIMessage sends an ai:stream message to all WebSocket clients
// in the format: { type: "ai:stream", data: { trace_id, content, done } }.
func (s *Streamer) BroadcastAIMessage(traceID, content string, done bool) {
	if s.hub == nil {
		return
	}
	msg := map[string]interface{}{
		"type": "ai:stream",
		"data": map[string]interface{}{
			"trace_id": traceID,
			"content":  content,
			"done":     done,
		},
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		s.logger.Warn("broadcast ai message marshal failed", zap.Error(err))
		return
	}
	if done {
		s.hub.BroadcastAndWait(payload)
	} else {
		s.hub.Broadcast(payload)
	}
}

// BroadcastEvent pushes an event to all WebSocket clients (realtime ticker).
func (s *Streamer) BroadcastEvent(ev *SSEEvent) {
	if s.hub == nil {
		return
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		s.logger.Warn("broadcast marshal failed", zap.Error(err))
		return
	}
	s.hub.BroadcastAndWait(payload)
}

// StreamChat simulates a streaming chat response by emitting token-like
// chunks. In production this would proxy to an LLM provider.
func (s *Streamer) StreamChat(c *gin.Context, traceID, agentID, answer string) {
	s.StartSSE(c)
	chunks := chunkText(answer, 24)
	for i, ch := range chunks {
		ev := &SSEEvent{
			Event:     "token",
			TraceID:   traceID,
			AgentID:   agentID,
			Seq:       i + 1,
			Data:      map[string]string{"text": ch},
			Timestamp: time.Now(),
		}
		if err := s.WriteSSE(c, ev); err != nil {
			if err != io.EOF {
				s.logger.Debug("sse write ended", zap.Error(err))
			}
			return
		}
		s.BroadcastEvent(ev)
		s.BroadcastAIMessage(traceID, ch, false)
		time.Sleep(15 * time.Millisecond)
	}
	s.BroadcastAIMessage(traceID, "", true)
	done := &SSEEvent{
		Event:     "done",
		TraceID:   traceID,
		AgentID:   agentID,
		Data:      map[string]string{"status": "completed"},
		Timestamp: time.Now(),
	}
	_ = s.WriteSSE(c, done)
	s.BroadcastEvent(done)
}

// chunkText splits a string into chunks of up to n runes.
func chunkText(s string, n int) []string {
	if n <= 0 || len(s) <= n {
		return []string{s}
	}
	runes := []rune(s)
	chunks := make([]string, 0, len(runes)/n+1)
	for i := 0; i < len(runes); i += n {
		end := i + n
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}
