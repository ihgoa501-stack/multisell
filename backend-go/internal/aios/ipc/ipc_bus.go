package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"go.uber.org/zap"
)

// Handler processes an IPC message and optionally returns a response.
// For request-type messages (Request, Delegate, Gather, Consensus),
// a non-nil response with the same SessionID is routed back to the caller.
type Handler func(ctx context.Context, msg *Message) (*Message, error)

// IPC is the inter-agent communication bus.
// It uses EventBus as the underlying transport layer: Send publishes to
// "ipc.agent.{target}" topics, and a wildcard subscriber dispatches
// incoming messages to locally-registered handlers.
// Reply channels (sync.Map of sessionID -> chan *Message) handle the
// request/response pattern independently of the transport.
type IPC struct {
	handlers map[string]Handler
	pending  sync.Map // sessionID -> chan *Message
	bus      *eventbus.Bus
	logger   *zap.Logger
	mu       sync.RWMutex
	subID    string // EventBus subscription ID
}

// New creates a new IPC bus with EventBus as the transport layer.
// It subscribes to "ipc.agent.*" on the given bus to receive inter-agent messages.
func New(logger *zap.Logger, bus *eventbus.Bus) *IPC {
	ipc := &IPC{
		handlers: make(map[string]Handler),
		bus:      bus,
		logger:   logger,
	}

	// Subscribe to all IPC agent messages on EventBus.
	subID := bus.Subscribe("ipc.agent.*", ipc.handleEventBusEvent)
	ipc.subID = subID

	return ipc
}

// RegisterHandler registers a handler for the given topic (typically an agent ID or squad name).
func (ipc *IPC) RegisterHandler(topic string, handler Handler) {
	ipc.mu.Lock()
	defer ipc.mu.Unlock()
	ipc.handlers[topic] = handler
	ipc.logger.Info("ipc handler registered", zap.String("topic", topic))
}

// getHandler returns the handler for a topic, under read lock.
func (ipc *IPC) getHandler(topic string) (Handler, bool) {
	ipc.mu.RLock()
	defer ipc.mu.RUnlock()
	h, ok := ipc.handlers[topic]
	return h, ok
}

// isRequestType returns true if the message type expects a response.
func isRequestType(t MsgType) bool {
	return t == MsgTypeRequest || t == MsgTypeDelegate ||
		t == MsgTypeGather || t == MsgTypeConsensus
}

// Send delivers a message to the target agent by publishing to
// the EventBus topic "ipc.agent.{msg.To}". The event payload is the
// serialized Message -- a wildcard subscriber handleEventBusEvent
// receives it, dispatches to the registered handler, and routes any
// response back through the pending channel.
func (ipc *IPC) Send(ctx context.Context, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("ipc: cannot send nil message")
	}

	// Pre-check: verify a handler is registered for this target.
	if _, ok := ipc.getHandler(msg.To); !ok {
		return fmt.Errorf("ipc: no handler registered for topic %q", msg.To)
	}

	// Serialize the Message into the EventBus payload via JSON round-trip.
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("ipc: failed to serialize message: %w", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("ipc: failed to serialize message payload: %w", err)
	}

	// Publish to EventBus.
	_, err = ipc.bus.Publish(ctx, "ipc.agent."+msg.To, "ipc", payload)
	if err != nil {
		return fmt.Errorf("ipc: event bus publish error: %w", err)
	}

	return nil
}

// handleEventBusEvent is the EventBus subscriber for "ipc.agent.*" topics.
// It deserializes the event payload into an IPC Message, looks up the
// registered handler for the target agent, invokes it, and routes any
// response back through the pending channel.
func (ipc *IPC) handleEventBusEvent(ctx context.Context, evt eventbus.Event) error {
	// Deserialize EventBus payload into IPC Message.
	data, err := json.Marshal(evt.Payload)
	if err != nil {
		ipc.logger.Warn("ipc: failed to marshal event payload", zap.Error(err))
		return nil
	}
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		ipc.logger.Warn("ipc: failed to unmarshal IPC message from event payload", zap.Error(err))
		return nil
	}

	handler, ok := ipc.getHandler(msg.To)
	if !ok {
		ipc.logger.Warn("ipc: no handler registered for agent",
			zap.String("agent_id", msg.To))
		return nil
	}

	resp, err := handler(ctx, &msg)
	if err != nil {
		ipc.logger.Error("ipc handler error",
			zap.String("agent_id", msg.To),
			zap.Error(err),
		)
		return nil
	}

	// Route the response back to the caller's pending channel.
	if resp != nil && isRequestType(msg.Type) {
		sessionID := resp.SessionID
		if sessionID == "" {
			sessionID = msg.SessionID
		}
		if ch, loaded := ipc.pending.Load(sessionID); loaded {
			select {
			case ch.(chan *Message) <- resp:
			default:
				ipc.logger.Warn("ipc: response dropped, channel full",
					zap.String("session_id", sessionID),
				)
			}
		}
	}

	return nil
}

// Request sends a request message and waits for a response within the given timeout.
func (ipc *IPC) Request(ctx context.Context, to string, payload map[string]interface{}, timeout time.Duration) (*Message, error) {
	sessionID := uuid.New().String()

	msg := &Message{
		ID:        uuid.New().String(),
		Type:      MsgTypeRequest,
		From:      "system",
		To:        to,
		SessionID: sessionID,
		Payload:   payload,
		Priority:  0,
		TimeoutMs: int(timeout.Milliseconds()),
		CreatedAt: time.Now(),
	}

	ch := make(chan *Message, 1)
	ipc.pending.Store(sessionID, ch)
	defer ipc.pending.Delete(sessionID)

	if err := ipc.Send(ctx, msg); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("ipc: request to %q timed out after %v", to, timeout)
	}
}

// Broadcast sends payload to all specified squads/topics and collects responses.
// It uses an internal request/reply pattern per squad with a 10-second timeout.
func (ipc *IPC) Broadcast(ctx context.Context, squads []string, payload map[string]interface{}) []*Message {
	var (
		results []*Message
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	for _, squad := range squads {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()

			sessionID := uuid.New().String()
			msg := &Message{
				ID:        uuid.New().String(),
				Type:      MsgTypeRequest,
				From:      "system",
				To:        s,
				SessionID: sessionID,
				Payload:   payload,
				CreatedAt: time.Now(),
			}

			ch := make(chan *Message, 1)
			ipc.pending.Store(sessionID, ch)
			defer ipc.pending.Delete(sessionID)

			if err := ipc.Send(ctx, msg); err != nil {
				ipc.logger.Warn("broadcast send failed",
					zap.String("squad", s),
					zap.Error(err),
				)
				return
			}

			select {
			case resp := <-ch:
				mu.Lock()
				results = append(results, resp)
				mu.Unlock()
			case <-time.After(10 * time.Second):
				ipc.logger.Warn("broadcast timeout",
					zap.String("squad", s),
				)
			case <-ctx.Done():
				ipc.logger.Warn("broadcast context done",
					zap.String("squad", s),
					zap.Error(ctx.Err()),
				)
			}
		}(squad)
	}

	wg.Wait()
	return results
}

// Delegate sends a task to a specific agent and waits for the result.
// Returns the response message and a boolean indicating success.
// Uses an internal 30-second timeout.
func (ipc *IPC) Delegate(ctx context.Context, to string, task TaskDef) (*Message, bool) {
	sessionID := uuid.New().String()

	payload := map[string]interface{}{
		"task_id":          task.ID,
		"task_description": task.Description,
		"decision_point":   task.DecisionPoint,
	}

	// Merge task payload into message payload.
	for k, v := range task.Payload {
		payload[k] = v
	}

	msg := &Message{
		ID:        uuid.New().String(),
		Type:      MsgTypeDelegate,
		From:      "system",
		To:        to,
		SessionID: sessionID,
		Payload:   payload,
		TimeoutMs: 30000,
		CreatedAt: time.Now(),
	}

	ch := make(chan *Message, 1)
	ipc.pending.Store(sessionID, ch)
	defer ipc.pending.Delete(sessionID)

	if err := ipc.Send(ctx, msg); err != nil {
		ipc.logger.Warn("delegate send failed",
			zap.String("to", to),
			zap.String("task_id", task.ID),
			zap.Error(err),
		)
		return nil, false
	}

	select {
	case resp := <-ch:
		return resp, true
	case <-time.After(30 * time.Second):
		ipc.logger.Warn("delegate timeout",
			zap.String("to", to),
			zap.String("task_id", task.ID),
		)
		return nil, false
	case <-ctx.Done():
		ipc.logger.Warn("delegate context done",
			zap.String("to", to),
			zap.String("task_id", task.ID),
			zap.Error(ctx.Err()),
		)
		return nil, false
	}
}

// Gather sends a task to multiple agents in parallel and collects all results.
// Returns an error only if all targets fail.
func (ipc *IPC) Gather(ctx context.Context, targets []string, task TaskDef) ([]*Message, error) {
	type gatherResult struct {
		msg *Message
		err error
	}

	resultCh := make(chan gatherResult, len(targets))

	for _, target := range targets {
		go func(t string) {
			resp, ok := ipc.Delegate(ctx, t, task)
			if !ok {
				resultCh <- gatherResult{err: fmt.Errorf("gather: delegate to %q failed", t)}
				return
			}
			resultCh <- gatherResult{msg: resp}
		}(target)
	}

	var messages []*Message
	for i := 0; i < len(targets); i++ {
		r := <-resultCh
		if r.err != nil {
			ipc.logger.Warn("gather: target failed", zap.Error(r.err))
			continue
		}
		messages = append(messages, r.msg)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("ipc: gather failed -- all %d targets returned errors", len(targets))
	}

	return messages, nil
}

// Consensus sends a question to multiple agents, collects their responses,
// and computes an aggregated consensus result.
func (ipc *IPC) Consensus(ctx context.Context, question string, agents []string) (*ConsensusResult, error) {
	task := TaskDef{
		ID:            uuid.New().String(),
		Description:   question,
		DecisionPoint: "consensus",
		Payload: map[string]interface{}{
			"question": question,
			"purpose":  "consensus",
		},
	}

	results, err := ipc.Gather(ctx, agents, task)
	if err != nil {
		return nil, fmt.Errorf("ipc: consensus gather failed: %w", err)
	}

	return computeConsensus(results), nil
}
