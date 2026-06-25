package ai

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/realtime"
)

// NewAIChatHandler creates a realtime.AIChatFunc that routes messages through
// the orchestrator and streams responses chunk-by-chunk over WebSocket.
func NewAIChatHandler(orch *Orchestrator) realtime.AIChatFunc {
	return func(ctx context.Context, message string, userID *int64) (<-chan realtime.AIChatChunk, error) {
		result, err := orch.Chat(message, userID)
		if err != nil {
			return nil, err
		}

		answer := ""
		if s, ok := result.Output["recommendation"].(string); ok {
			answer = s
		}

		ch := make(chan realtime.AIChatChunk, 256)
		go func() {
			defer close(ch)
			chunks := chunkText(answer, 24)
			for i, chunk := range chunks {
				select {
				case ch <- realtime.AIChatChunk{
					TraceID: result.TraceID,
					Content: chunk,
					Done:    i == len(chunks)-1,
				}:
				case <-ctx.Done():
					return
				}
				time.Sleep(15 * time.Millisecond)
			}
		}()
		return ch, nil
	}
}
